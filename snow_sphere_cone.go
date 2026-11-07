package main

import (
	"fmt"
	"math"
	"sort"
)

// Б.15а. H is the overall ridge level, f is the roof rise above the wall.
// r is the HORIZONTAL cone radius, not the inclined generator. The spherical
// shoulder is a circular arc tangent to the conical slope at x=r; the angle
// alpha1 at the outer edge is derived, not fixed at 90 degrees.
type sphereConeGeometry struct {
	D, H, Rise, ConeRadius, Angle                      float64
	SphereRadius, CenterX, CenterY, EdgeHeight, Angle1 float64
	LinearArc                                          bool
	StrictZone5Formula                                 bool
}

func sphereConeTangentAngle(s SnowCalc) float64 {
	r, w := s.SkylightWidth, s.Span/2-s.SkylightWidth
	if w <= 0 {
		return math.NaN()
	}
	a := s.Angle * math.Pi / 180
	delta := s.H - r*math.Tan(a)
	return (2*math.Atan2(delta, w) - a) * 180 / math.Pi
}

// When the spherical tangent would turn beyond 90 degrees, increase D only.
// Do NOT reject a negative H-f: an underground abutment is meaningful.
func sphereConeExpandDiameter(s *SnowCalc) (old, newValue float64, adjusted bool) {
	old, newValue = s.Span, s.Span
	a := s.Angle * math.Pi / 180
	if s.SkylightWidth <= 0 || s.Angle <= 0 || s.Angle >= 90 || s.Span <= 0 {
		return
	}
	delta := s.H - s.SkylightWidth*math.Tan(a)
	// D must be larger than 2r even when the shoulder's rise is zero or
	// negative. Earlier the UI returned before reaching this test, while
	// SnowCompute could still silently enlarge D on Calculate.
	minD := 2*s.SkylightWidth + .001
	if delta > 0 {
		wmin := delta / math.Tan((a+math.Pi/2)/2)
		minD = math.Max(minD, 2*(s.SkylightWidth+wmin))
	}
	if minD > s.Span+1e-8 {
		// Allow a tiny margin so floating-point rounding never produces >90°.
		s.Span = math.Ceil((minD+0.0001)*1000) / 1000
		s.Width = s.Span
		newValue = s.Span
		adjusted = true
	}
	return
}
func sphereConeRiseForAngle1(s SnowCalc, alpha1 float64) float64 {
	a := (s.Angle + alpha1) * math.Pi / 360
	return s.SkylightWidth*math.Tan(s.Angle*math.Pi/180) + (s.Span/2-s.SkylightWidth)*math.Tan(a)
}
func makeSphereConeGeometry(s SnowCalc) (sphereConeGeometry, error) {
	g := sphereConeGeometry{D: s.Span, H: s.RoofHeight, Rise: s.H, ConeRadius: s.SkylightWidth, Angle: s.Angle, StrictZone5Formula: s.SphereStrictFormula}
	e := g.D / 2
	if g.D <= 0 || g.H <= 0 || g.ConeRadius <= 0 || e <= g.ConeRadius || g.Angle <= 0 || g.Angle >= 90 || g.Rise <= 0 {
		return g, fmt.Errorf("Для Б.15а необходимы D>2r>0, H>0, f>0 и 0°<α<90°")
	}
	g.Angle1 = sphereConeTangentAngle(s)
	if !num(g.Angle1) {
		return g, fmt.Errorf("Не удалось определить касательную сферической части")
	}
	if g.Angle1 > 90.00001 {
		return g, fmt.Errorf("Касательная у края больше 90°; увеличьте D до %s м", decimalInput(2*(g.ConeRadius+(g.Rise-g.ConeRadius*math.Tan(g.Angle*math.Pi/180))/math.Tan((g.Angle+90)*math.Pi/360)), 3))
	}
	a := g.Angle * math.Pi / 180
	dx, dh := e-g.ConeRadius, g.Rise-g.ConeRadius*math.Tan(a)
	denominator := 2 * (dh*math.Cos(a) - dx*math.Sin(a))
	g.EdgeHeight = g.H - g.Rise
	if math.Abs(denominator) < 1e-9 {
		g.LinearArc = true // equal slopes: limiting infinite-radius case
		return g, nil
	}
	g.SphereRadius = (dx*dx + dh*dh) / denominator // signed: concave case allowed as conditional geometry
	g.CenterX = g.ConeRadius - g.SphereRadius*math.Sin(a)
	g.CenterY = g.H - g.ConeRadius*math.Tan(a) - g.SphereRadius*math.Cos(a)
	if !num(g.SphereRadius) || math.Abs(g.SphereRadius) < 1e-9 {
		return g, fmt.Errorf("Невозможно построить сферическое сопряжение по указанным размерам")
	}
	// Arc coordinate must be single-valued on the roof projection.
	if math.Abs(e-g.CenterX) > math.Abs(g.SphereRadius)+1e-6 {
		return g, fmt.Errorf("Сферическая часть выходит за пределы дуги окружности")
	}
	return g, nil
}
func (g sphereConeGeometry) heightAtRadius(radius float64) float64 {
	if radius <= g.ConeRadius {
		return g.H - radius*math.Tan(g.Angle*math.Pi/180)
	}
	if g.LinearArc {
		return g.H - radius*math.Tan(g.Angle*math.Pi/180)
	}
	dx := radius - g.CenterX
	inside := g.SphereRadius*g.SphereRadius - dx*dx
	if inside < 0 {
		inside = 0
	}
	if g.SphereRadius > 0 {
		return g.CenterY + math.Sqrt(inside)
	}
	return g.CenterY - math.Sqrt(inside)
}

// The coefficient Cr1 in B.10 refers to the rise of a hypothetical FULL
// spherical dome, not to the height of the combined sphere-and-cone roof.
// A spherical cap having the same diameter and the same peripheral tangent
// alpha1 has rise f_sphere=(D/2)*tan(alpha1/2).
func (g sphereConeGeometry) sphereOnlyRise() float64 {
	return (g.D / 2) * math.Tan(g.Angle1*math.Pi/360)
}

func (g sphereConeGeometry) domeCriticalRadii() []float64 {
	rs := []float64{g.ConeRadius, g.D / 2}
	if g.LinearArc || g.SphereRadius == 0 {
		return rs
	}
	// B.10 is quadratic until alpha=30 degrees, then piecewise linear
	// in the local slope until alpha=60 degrees. On each interval extrema
	// can only occur at an end or one of these critical slopes.
	for _, angle := range []float64{30, 45, 60} {
		dr := math.Abs(g.SphereRadius) * math.Sin(angle*math.Pi/180)
		for _, radius := range []float64{g.CenterX - dr, g.CenterX + dr} {
			if radius > g.ConeRadius && radius < g.D/2 {
				rs = append(rs, radius)
			}
		}
	}
	r30 := g.CenterX + math.Abs(g.SphereRadius)*0.5
	if r30 > g.ConeRadius && r30 < g.D/2 {
		rs = append(rs, r30)
	}
	// For a buried shoulder the visible snow-covered part ends where the
	// roof crosses ground level. The maximum may occur at that boundary.
	a, b := g.ConeRadius, g.D/2
	ha, hb := g.heightAtRadius(a), g.heightAtRadius(b)
	if (ha < 0) != (hb < 0) {
		for i := 0; i < 80; i++ {
			mid := (a + b) / 2
			if (g.heightAtRadius(mid) < 0) == (ha < 0) {
				a = mid
			} else {
				b = mid
			}
		}
		rs = append(rs, (a+b)/2)
	}
	return rs
}
func (g sphereConeGeometry) maxDomeMu2() float64 {
	max := 0.0
	for _, radius := range g.domeCriticalRadii() {
		if g.heightAtRadius(radius) >= 0 {
			if mu := g.domeMu2(radius); mu > max {
				max = mu
			}
		}
	}
	return max
}
func (g sphereConeGeometry) zone5Mu() float64 {
	zone5Angle := g.Angle // user-approved geometric interpretation (alpha2)
	if g.StrictZone5Formula {
		zone5Angle = g.Angle1 // literal wording in B.12(b)
	}
	if zone5Angle > 7 && zone5Angle <= 15 {
		return .8 * snowRoofMu(g.Angle)
	}
	if zone5Angle > 15 && zone5Angle <= 30 {
		return .5 * g.domeMu2(g.ConeRadius)
	}
	return 0
}
func (g sphereConeGeometry) spatialMaxMu() float64 {
	// Zone 1 peaks at beta=90°, cone zones at their perpendicular
	// azimuth; zones 2 and 6 cannot exceed half the full dome peak.
	// Zone 5 is piecewise constant. All critical dome slopes are checked
	// analytically, without dependence on the presentation grid step.
	maxConeRadius := g.ConeRadius
	if tan := math.Tan(g.Angle * math.Pi / 180); tan > 0 {
		maxConeRadius = math.Min(maxConeRadius, g.H/tan)
	}
	return math.Max(g.maxDomeMu2(), math.Max(g.coneMu2(math.Max(0, maxConeRadius)), g.zone5Mu()))
}

func (g sphereConeGeometry) sideAngle(radius float64) float64 {
	if radius <= g.ConeRadius {
		return g.Angle
	}
	if g.LinearArc {
		return g.Angle
	}
	v := (radius - g.CenterX) / math.Abs(g.SphereRadius)
	if v < -1 {
		v = -1
	}
	if v > 1 {
		v = 1
	}
	// Signed curvature allows a sphere shoulder which becomes concave.
	if g.SphereRadius < 0 {
		return -math.Asin(v) * 180 / math.Pi
	}
	return math.Asin(v) * 180 / math.Pi
}
func (g sphereConeGeometry) domeMu2(radius float64) float64 {
	alpha := math.Abs(g.sideAngle(radius))
	cr := 2.55 - math.Exp(.8-14*g.sphereOnlyRise()/g.D)
	r30 := g.CenterX + math.Abs(g.SphereRadius)*.5
	if r30 <= 0 || g.LinearArc {
		r30 = g.ConeRadius
	}
	if alpha <= 30 {
		return math.Max(0, cr*math.Pow(radius/r30, 2))
	}
	if alpha <= 45 {
		return math.Max(0, cr+(1.5-cr)*(alpha-30)/15)
	}
	if alpha < 60 {
		return 1.5 * (60 - alpha) / 15
	}
	return 0
}
func (g sphereConeGeometry) coneMu2(radius float64) float64 {
	if g.Angle <= 7 || g.Angle >= 60 {
		return 0
	}
	var c float64
	if g.Angle <= 30 {
		c = 2.15 * math.Cbrt(g.Angle*math.Pi/180)
	} else {
		c = 1.7 * 30 / g.Angle
	}
	return math.Max(0, c*radius/g.ConeRadius)
}

// Beta is the SINGLE normative polar azimuth of figure B.15a: beta=0°
// points straight down in plan, 90° right (middle of zone 1), 180° up,
// 270° left (middle of zones 5/6). The zones do NOT reset beta.
// The roof sector boundaries are at 30, 150, 210 and 330 degrees.
func (g sphereConeGeometry) spatialMu(radius, beta float64) (float64, int) {
	b := math.Mod(beta, 360)
	if b < 0 {
		b += 360
	}
	right := b > 30 && b < 150
	left := b > 210 && b < 330
	outer := radius >= g.ConeRadius
	if g.heightAtRadius(radius) < 0 {
		if outer {
			if right {
				return 0, 1
			}
			if left {
				return 0, 6
			}
			return 0, 2
		}
		if right {
			return 0, 3
		}
		if left {
			return 0, 5
		}
		return 0, 4
	}
	sine := math.Sin(b * math.Pi / 180)
	if outer {
		dome := g.domeMu2(radius)
		if right {
			return dome * sine, 1
		}
		if left {
			// B.10 at 270° has a negative sine. Agreed engineering
			// interpretation of B.15a zone 6: disregard that sign.
			return .5 * dome * math.Abs(sine), 6
		}
		// Zone 2 uses the ENTIRE radial epure B.10 at beta=90°, halved.
		// At beta=30° and 150°, sin(beta)=0.5, so zones 1/2 match.
		return .5 * dome, 2
	}
	cone := g.coneMu2(radius)
	if right {
		return cone * sine, 3
	}
	if left {
		return g.zone5Mu(), 5
	}
	// Zone 4 belongs to both half-planes; its B.12/B.13 ordinate is a
	// magnitude and cannot be negative on the left of the 180° meridian.
	return cone * math.Abs(sine), 4
}
func sphereConeBetas(step float64) []float64 {
	if !num(step) || step <= 0 {
		step = 15
	}
	list := []float64{}
	for b := 0.; b < 360.-1e-8; b += step {
		list = append(list, b)
	}
	return append(list, 360)
}
func sphereConeRadii(g sphereConeGeometry) []float64 {
	radii := []float64{}
	for i := 0; i <= 24; i++ {
		radii = append(radii, g.D/2*float64(i)/24)
	}
	radii = append(radii, g.ConeRadius)
	sort.Float64s(radii)
	unique := radii[:0]
	for _, r := range radii {
		if len(unique) == 0 || r-unique[len(unique)-1] > 1e-7 {
			unique = append(unique, r)
		}
	}
	return unique
}
func snowSphereConeCases(s SnowCalc) ([]SnowCase, error) {
	g, err := makeSphereConeGeometry(s)
	if err != nil {
		return nil, err
	}
	const N = 120
	fillLine := func(betaRight, betaLeft float64, fn func(float64, float64) float64) []SnowInterval {
		result := make([]SnowInterval, 0, N)
		for i := 0; i < N; i++ {
			x1 := g.D * float64(i) / N
			x2 := g.D * float64(i+1) / N
			mid := (x1 + x2) / 2
			radius := math.Abs(mid - g.D/2)
			beta := betaRight
			if mid < g.D/2 {
				beta = betaLeft
			}
			result = append(result, snowMuInterval(x1, x2, fn(radius, beta)))
		}
		return result
	}
	ordinary := func(radius, beta float64) float64 {
		if g.heightAtRadius(radius) < 0 {
			return 0
		}
		return snowRoofMu(math.Abs(g.sideAngle(radius)))
	}
	all := fillLine(0, 180, ordinary)
	note := fmt.Sprintf("Сопряжение конуса и сферы: α1=%s°; уровень стенки H−f=%s м. При местном уклоне ≥60° μ1=0.", decimalInput(g.Angle1, 2), decimalInput(g.EdgeHeight, 3))
	if g.EdgeHeight < 0 {
		/* This remains a calculation choice; do not insert GUI warnings into customer reports. */
	}
	if g.Angle1 <= g.Angle {
		/* Surface geometry is user-controlled; normative limits are indicated in GUI only. */
	}
	first := snowCase("1. По уклонам сферической и конической частей", "СП 20.13330.2016, Б.12(б), рисунок Б.15а, таблица Б.3", note, all...)
	betaSamples := sphereConeBetas(s.BetaStep)
	radialSamples := sphereConeRadii(g)
	for _, beta := range betaSamples {
		for _, radius := range radialSamples {
			first.Polar = append(first.Polar, SnowPolarPoint{Radius: radius, Beta: beta, Mu: ordinary(radius, beta)})
		}
	}
	// Normative shallow-dome exception: the second scheme is not required.
	if g.Angle1 <= 10 && g.Angle <= 7 {
		return []SnowCase{first}, nil
	}
	line := fillLine(90, 270, func(r, b float64) float64 { m, _ := g.spatialMu(r, b); return m })
	second := snowCase("2. Пространственное загружение, зоны 1–6", "СП 20.13330.2016, Б.12(б), рисунок Б.15а, формулы (Б.10), (Б.12), (Б.13)",
		"Все шесть зон действуют совместно. β отсчитывается от направления вниз в плане. Сечения представлены по зонам 6–1 и 2–4–4–2; таблицы Word и Excel содержат полярную сетку.", line...)
	second.Cross = fillLine(180, 0, func(r, b float64) float64 { m, _ := g.spatialMu(r, b); return m })
	second.PeakMu = g.spatialMaxMu()
	for _, beta := range betaSamples {
		for _, radius := range radialSamples {
			mu, zone := g.spatialMu(radius, beta)
			second.Polar = append(second.Polar, SnowPolarPoint{Radius: radius, Beta: beta, Mu: mu, Zone: zone})
		}
	}
	return []SnowCase{first, second}, nil
}
