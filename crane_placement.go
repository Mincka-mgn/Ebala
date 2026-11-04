package main

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// CraneTrain is a rigid longitudinal train of wheel loads on one runway rail.
// Wheel positions are millimetres from the first wheel; PKN are normative kN.
type CraneTrain struct {
	Name      string
	WheelXMM  []float64
	ForcesKN  []float64
	OverallMM float64
}
type CranePoint struct {
	XMM, PKN float64
	Crane    string
}
type CranePlacementCase struct {
	Name      string
	Target    string
	Value     float64
	Units     string
	ShiftMM   float64
	SectionMM float64 // section of the governing moment, when applicable
	Points    []CranePoint
	Note      string
}
type CranePlacementResult struct {
	SpansMM  []float64
	Cases    []CranePlacementCase
	Warnings []string
}

type craneTrainGeometry struct {
	points []CranePoint
	width  float64
}

func assembleCraneTrain(trains []CraneTrain, reverse bool) (craneTrainGeometry, error) {
	if len(trains) == 0 {
		return craneTrainGeometry{}, errors.New("Выберите хотя бы один кран")
	}
	ordered := append([]CraneTrain(nil), trains...)
	if reverse {
		for a, b := 0, len(ordered)-1; a < b; a, b = a+1, b-1 {
			ordered[a], ordered[b] = ordered[b], ordered[a]
		}
	}
	g := craneTrainGeometry{}
	var position float64
	for i, c := range ordered {
		if len(c.WheelXMM) == 0 || len(c.WheelXMM) != len(c.ForcesKN) {
			return g, fmt.Errorf("%s: укажите нагрузки на все колёса", c.Name)
		}
		for j, x := range c.WheelXMM {
			if x < 0 || (j > 0 && x <= c.WheelXMM[j-1]) {
				return g, fmt.Errorf("%s: расстояния между колёсами должны быть положительными", c.Name)
			}
			if c.ForcesKN[j] <= 0 {
				return g, fmt.Errorf("%s: нагрузки на колёса должны быть положительными", c.Name)
			}
		}
		wheelLength := c.WheelXMM[len(c.WheelXMM)-1] - c.WheelXMM[0]
		if i > 0 {
			prev := ordered[i-1]
			prevWidth := prev.WheelXMM[len(prev.WheelXMM)-1] - prev.WheelXMM[0]
			// Physical end-to-end clearance from the known overall crane lengths.
			// If an overall dimension is absent, the corresponding extra clearance is zero.
			position += math.Max(0, (prev.OverallMM-prevWidth)/2) + math.Max(0, (c.OverallMM-wheelLength)/2)
		}
		for j, x := range c.WheelXMM {
			g.points = append(g.points, CranePoint{XMM: position + x - c.WheelXMM[0], PKN: c.ForcesKN[j], Crane: c.Name})
		}
		position += wheelLength
	}
	g.width = position
	return g, nil
}

func craneSolveLinear(A [][]float64, b []float64) ([]float64, error) {
	n := len(b)
	m := make([][]float64, n)
	for i := range m {
		m[i] = append(append([]float64(nil), A[i]...), b[i])
	}
	for j := 0; j < n; j++ {
		k := j
		for i := j + 1; i < n; i++ {
			if math.Abs(m[i][j]) > math.Abs(m[k][j]) {
				k = i
			}
		}
		if math.Abs(m[k][j]) < 1e-14 {
			return nil, errors.New("вырожденная расчётная схема")
		}
		m[j], m[k] = m[k], m[j]
		factor := m[j][j]
		for p := j; p <= n; p++ {
			m[j][p] /= factor
		}
		for i := 0; i < n; i++ {
			if i != j {
				f := m[i][j]
				for p := j; p <= n; p++ {
					m[i][p] -= f * m[j][p]
				}
			}
		}
	}
	out := make([]float64, n)
	for i := range out {
		out[i] = m[i][n]
	}
	return out, nil
}

type craneBeamState struct {
	moments    []float64
	shearLeft  []float64
	shearRight []float64
	reactions  []float64
}

// Continuous Euler-Bernoulli beam with uniform EI. All support translations fixed;
// rotations free. Point forces use exact Hermite consistent nodal loads.
func solveCraneBeam(spansMM []float64, points []CranePoint, midSections []float64) (craneBeamState, error) {
	n := len(spansMM) + 1
	nodes := make([]float64, n)
	for i, L := range spansMM {
		if L <= 0 {
			return craneBeamState{}, errors.New("пролёт должен быть положительным")
		}
		nodes[i+1] = nodes[i] + L/1000
	}
	k := make([][]float64, n)
	for i := range k {
		k[i] = make([]float64, n)
	}
	frot := make([]float64, n)
	fvert := make([]float64, n)
	// nodal force contribution of each external point load
	loads := make([][]CranePoint, n-1)
	for _, p := range points {
		x := p.XMM / 1000
		if x < 0 || x > nodes[n-1] || p.PKN <= 0 {
			continue
		}
		i := sort.Search(n-1, func(j int) bool { return nodes[j+1] >= x })
		if i >= n-1 {
			i = n - 2
		}
		loads[i] = append(loads[i], p)
	}
	for i, Lmm := range spansMM {
		L := Lmm / 1000
		k[i][i] += 4 / L
		k[i+1][i+1] += 4 / L
		k[i][i+1] += 2 / L
		k[i+1][i] += 2 / L
		for _, p := range loads[i] {
			t := (p.XMM/1000 - nodes[i]) / L
			n1 := 1 - 3*t*t + 2*t*t*t
			n2 := L * (t - 2*t*t + t*t*t)
			n3 := 3*t*t - 2*t*t*t
			n4 := L * (-t*t + t*t*t)
			fvert[i] -= p.PKN * n1
			fvert[i+1] -= p.PKN * n3
			frot[i] -= p.PKN * n2
			frot[i+1] -= p.PKN * n4
		}
	}
	theta, err := craneSolveLinear(k, frot)
	if err != nil {
		return craneBeamState{}, err
	}
	// Element end reactions including equivalent nodal vertical forces.
	leftV := make([]float64, n-1)
	rightV := make([]float64, n-1)
	leftMoment := make([]float64, n-1)
	reactions := make([]float64, n)
	for i, Lmm := range spansMM {
		L := Lmm / 1000
		t1, t2 := theta[i], theta[i+1]
		// q_v(left)=6/L²*(theta_left - theta_right), q_v(right)=-q_v(left)
		V := 6 / (L * L) * (t1 + t2)
		v1, v2 := V, -V
		ml := 4/L*t1 + 2/L*t2
		for _, p := range loads[i] {
			t := (p.XMM/1000 - nodes[i]) / L
			v1 += p.PKN * (1 - 3*t*t + 2*t*t*t)
			v2 += p.PKN * (3*t*t - 2*t*t*t)
			ml += p.PKN * L * (t - 2*t*t + t*t*t)
		}
		leftV[i], rightV[i], leftMoment[i] = v1, v2, ml
		reactions[i] += v1
		reactions[i+1] += v2
	}
	momentAt := func(x float64) float64 {
		i := sort.Search(n-1, func(j int) bool { return nodes[j+1] >= x })
		if i >= n-1 {
			i = n - 2
		}
		v := -leftMoment[i] + leftV[i]*(x-nodes[i])
		for _, p := range loads[i] {
			xp := p.XMM / 1000
			if xp < x {
				v -= p.PKN * (x - xp)
			}
		}
		return v
	}
	res := craneBeamState{reactions: reactions, moments: make([]float64, len(midSections)), shearLeft: make([]float64, n), shearRight: make([]float64, n)}
	for i, x := range midSections {
		res.moments[i] = momentAt(x)
	}
	for i := 0; i < n; i++ {
		if i > 0 {
			res.shearLeft[i] = -rightV[i-1]
		} // shear just to left of support from left span
		if i < n-1 {
			res.shearRight[i] = leftV[i]
		} // shear just to right of support
	}
	return res, nil
}

// parseCraneSpanMeters converts user-entered metres to internal millimetres.
// A value over 100 m is suspicious but does not block the calculation.
func parseCraneSpanMeters(raw string, count int) ([]float64, bool, error) {
	items := strings.FieldsFunc(raw, func(r rune) bool { return r == ' ' || r == ';' || r == '\n' || r == '\t' })
	if len(items) != count {
		return nil, false, fmt.Errorf("Укажите %d длин пролётов в метрах", count)
	}
	result := make([]float64, count)
	warning := false
	for i, text := range items {
		v, e := strconv.ParseFloat(strings.ReplaceAll(text, ",", "."), 64)
		if e != nil || v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, false, fmt.Errorf("Неверная длина пролёта: %s", text)
		}
		result[i] = v * 1000
		if v > 100 {
			warning = true
		}
	}
	return result, warning, nil
}

// The number of spans is the number of length values entered by the user.
func parseCraneSpanMetersAuto(raw string) ([]float64, bool, error) {
	pieces := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ' ' || r == ';' || r == '\n' || r == '\t'
	})
	if len(pieces) < 1 || len(pieces) > 20 {
		return nil, false, fmt.Errorf("Укажите от 1 до 20 длин пролётов в метрах")
	}
	return parseCraneSpanMeters(raw, len(pieces))
}

// Find extrema over all permissible translations and both train orders.
// Each smooth interval is bounded by a wheel crossing a support; bounded
// golden searches refine the sampled extrema in that interval.
func CalculateCranePlacements(spansMM []float64, trains []CraneTrain) (CranePlacementResult, error) {
	var result CranePlacementResult
	if len(spansMM) == 0 || len(spansMM) > 20 {
		return result, errors.New("Укажите от 1 до 20 пролётов")
	}
	for _, L := range spansMM {
		if L <= 0 || L > 1e9 {
			return result, errors.New("Неверная длина пролёта")
		}
	}
	result.Warnings = append(result.Warnings, "Определены экстремумы от заданных вертикальных колесных сил; коэффициенты динамичности, горизонтальные крановые нагрузки, сочетания и проверки выносливости по разделу 9 СП 20.13330.2016 должны быть учтены отдельно")
	result.SpansMM = append([]float64(nil), spansMM...)
	nodes := make([]float64, len(spansMM)+1)
	mid := make([]float64, len(spansMM))
	for i, L := range spansMM {
		nodes[i+1] = nodes[i] + L
		mid[i] = (nodes[i] + nodes[i+1]) / 2000
	}
	overall := nodes[len(nodes)-1]
	allTrains := []craneTrainGeometry{}
	a, err := assembleCraneTrain(trains, false)
	if err != nil {
		return result, err
	}
	allTrains = append(allTrains, a)
	// Reflect the full wheel train, not merely the order of crane objects.
	// Important for ONE asymmetric crane, and for multiple cranes with
	// different end wheel forces. Both orientations are checked by default.
	mirror := func(g craneTrainGeometry) craneTrainGeometry {
		out := craneTrainGeometry{width: g.width, points: make([]CranePoint, len(g.points))}
		for i, pt := range g.points {
			pt.XMM = g.width - pt.XMM
			out.points[len(g.points)-1-i] = pt
		}
		return out
	}
	allTrains = append(allTrains, mirror(a))
	if len(trains) > 1 {
		b, err := assembleCraneTrain(trains, true)
		if err != nil {
			return result, err
		}
		allTrains = append(allTrains, b, mirror(b))
	}
	for _, c := range trains {
		if c.OverallMM <= 0 {
			result.Warnings = append(result.Warnings, c.Name+": продольный габарит не указан; зазор между кранами принят по крайним колёсам")
		}
	}
	type target struct {
		title string
		idx   int
		field int
		sign  float64
		units string
	}
	// field: 0 mid-M, 1 Q-left, 2 Q-right, 3 global span M
	targets := []target{}
	for i := range spansMM {
		targets = append(targets, target{fmt.Sprintf("М+ в середине пролёта %d", i+1), i, 0, 1, "кН·м"}, target{fmt.Sprintf("М− в середине пролёта %d", i+1), i, 0, -1, "кН·м"})
		targets = append(targets, target{fmt.Sprintf("М+ наибольший в пролёте %d", i+1), i, 3, 1, "кН·м"}, target{fmt.Sprintf("М− наибольший по модулю в пролёте %d", i+1), i, 3, -1, "кН·м"})
	}
	for i := range nodes {
		if i > 0 {
			targets = append(targets, target{fmt.Sprintf("Q слева от опоры %d (+)", i+1), i, 1, 1, "кН"}, target{fmt.Sprintf("Q слева от опоры %d (−)", i+1), i, 1, -1, "кН"})
		}
		if i < len(nodes)-1 {
			targets = append(targets, target{fmt.Sprintf("Q справа от опоры %d (+)", i+1), i, 2, 1, "кН"}, target{fmt.Sprintf("Q справа от опоры %d (−)", i+1), i, 2, -1, "кН"})
		}
	}
	for _, target := range targets {
		best := -math.MaxFloat64
		var bestCase CranePlacementCase
		for _, train := range allTrains {
			// A crane may be partly outside the beam, but at least one wheel must be on it.
			lower, upper := -train.width, overall
			cuts := []float64{lower, upper}
			for _, p := range train.points {
				for _, n := range nodes {
					cut := n - p.XMM
					if cut > lower && cut < upper {
						cuts = append(cuts, cut)
					}
				}
			}
			sort.Float64s(cuts)
			uniq := cuts[:0]
			for _, x := range cuts {
				if len(uniq) == 0 || x-uniq[len(uniq)-1] > 1e-7 {
					uniq = append(uniq, x)
				}
			}
			evaluate := func(shift float64) (float64, float64) {
				pts := make([]CranePoint, len(train.points))
				for i, p := range train.points {
					pts[i] = p
					pts[i].XMM += shift
				}
				sections := mid
				if target.field == 3 {
					// M(x) is piecewise linear between wheels and supports.
					// Therefore its exact spatial extremum for a fixed train
					// is under a wheel or at an end of the chosen span.
					sections = []float64{nodes[target.idx] / 1000, nodes[target.idx+1] / 1000}
					for _, pt := range pts {
						if pt.XMM > nodes[target.idx]+1e-6 && pt.XMM < nodes[target.idx+1]-1e-6 {
							sections = append(sections, pt.XMM/1000)
						}
					}
				}
				state, e := solveCraneBeam(spansMM, pts, sections)
				if e != nil {
					return -math.MaxFloat64, 0
				}
				switch target.field {
				case 0:
					return target.sign * state.moments[target.idx], mid[target.idx] * 1000
				case 1:
					return target.sign * state.shearLeft[target.idx], nodes[target.idx]
				case 2:
					return target.sign * state.shearRight[target.idx], nodes[target.idx]
				case 3:
					bestSpatial, at := -math.MaxFloat64, 0.0
					for k, m := range state.moments {
						if target.sign*m > bestSpatial {
							bestSpatial = target.sign * m
							at = sections[k] * 1000
						}
					}
					return bestSpatial, at
				}
				return 0, 0
			}
			value := func(shift float64) float64 { v, _ := evaluate(shift); return v }
			accept := func(x float64) {
				v, section := evaluate(x)
				if v > best+1e-8 {
					best = v
					bestCase = CranePlacementCase{Name: target.title, Target: target.title, Value: v * target.sign, Units: target.units, ShiftMM: x, SectionMM: section, Points: make([]CranePoint, len(train.points))}
					for i, p := range train.points {
						p.XMM += x
						bestCase.Points[i] = p
					}
				}
			}
			for i := 1; i < len(uniq); i++ {
				l, r := uniq[i-1], uniq[i]
				if r-l < 1e-6 {
					continue
				}
				// Global maxima of piecewise cubic response, sampled then golden refined.
				const samples = 10
				last := l
				lastval := value(l)
				accept(l)
				for j := 1; j <= samples; j++ {
					x := l + (r-l)*float64(j)/samples
					v := value(x)
					accept(x)
					if j >= 2 {
						_ = lastval
					} // endpoints and local interiors refined in adjacent windows
					if j > 0 {
						a := last
						b := x // locate maximum inside each small interval
						const phi = 0.6180339887498949
						p := b - phi*(b-a)
						q := a + phi*(b-a)
						vp, vq := value(p), value(q)
						for z := 0; z < 28; z++ {
							if vp > vq {
								b = q
								q = p
								vq = vp
								p = b - phi*(b-a)
								vp = value(p)
							} else {
								a = p
								p = q
								vp = vq
								q = a + phi*(b-a)
								vq = value(q)
							}
						}
						accept((a + b) / 2)
					}
					last, lastval = x, v
				}
			}
		}
		if best > 1e-8 {
			result.Cases = append(result.Cases, bestCase)
		}
	}
	return result, nil
}
