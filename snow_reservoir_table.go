package main

import (
	"fmt"
	"math"
)

// Compact client-facing polar table. Detailed original sampled points remain
// in Excel; nonzero values here are selected from the actual calculation grid.
func snowWordReservoirRows(s SnowCalc, c SnowCase) [][]string {
	out := [][]string{}
	if s.Shape == "sphere_cone" && len(c.Polar) > 0 {
		// Full engineering table: NO thinning. User-selected beta step and
		// every radial station appear in DOCX as they do in the workbook.
		for _, p := range c.Polar {
			zone := "—"
			if p.Zone > 0 {
				zone = fmt.Sprint(p.Zone)
			}
			out = append(out, []string{decimalInput(p.Beta, 3), decimalInput(s.Span/2-p.Radius, 3), zone, decimalInput(p.Mu, 5), decimalInput(s.Q(p.Mu), 5), decimalInput(s.QDesign(p.Mu), 5), snowReducedCell(s, p.Mu)})
		}
		return out
	}
	appendRow := func(radius, sector string, mu float64) {
		out = append(out, []string{radius, sector, decimalInput(mu, 4), decimalInput(s.Q(mu), 4), decimalInput(s.QDesign(mu), 4), snowReducedCell(s, mu)})
	}
	if len(c.Polar) == 0 {
		// Axisymmetric loading: no angular dependence; show characteristic radii.
		if len(c.Intervals) == 1 {
			appendRow(fmt.Sprintf("0…%s", decimalInput(s.Span/2, 2)), "0…360", c.Intervals[0].Mu)
			return out
		}
		R := s.Span / 2
		for _, f := range []float64{0, .25, .5, .75, 1} {
			r := R * f
			m := snowMuAt(c, R+r)
			appendRow(decimalInput(r, 3), "0…360", m)
		}
		return out
	}
	rmax := 0.0
	for _, p := range c.Polar {
		if p.Radius > rmax {
			rmax = p.Radius
		}
	}
	for _, f := range []float64{0, .25, .5, .75, 1} {
		r := f * rmax
		subset := []SnowPolarPoint{}
		minDist := math.MaxFloat64
		for _, p := range c.Polar {
			d := math.Abs(p.Radius - r)
			if d < minDist {
				minDist = d
			}
		}
		for _, p := range c.Polar {
			if math.Abs(math.Abs(p.Radius-r)-minDist) < 1e-8 {
				subset = append(subset, p)
			}
		}
		if len(subset) == 0 {
			continue
		}
		r = subset[0].Radius
		allZero := true
		for _, p := range subset {
			if math.Abs(p.Mu) > 1e-9 {
				allZero = false
				break
			}
		}
		if allZero {
			appendRow(decimalInput(r, 3), "0…360", 0)
			continue
		}
		betaSamples := []float64{0, 30, 60, 90, 120, 150, 180}
		if s.Shape == "sphere_cone" {
			betaSamples = []float64{0, 30, 60, 90, 120, 150, 180, 210, 240, 270, 300, 330}
		}
		for _, beta := range betaSamples {
			nearest := subset[0]
			best := math.MaxFloat64
			for _, p := range subset {
				d := math.Abs(p.Beta - beta)
				if d < best {
					best = d
					nearest = p
				}
			}
			label := decimalInput(beta, 0)
			if s.Shape == "sphere_cone" && nearest.Zone > 0 {
				label += fmt.Sprintf(" (зона %d)", nearest.Zone)
			}
			appendRow(decimalInput(r, 3), label, nearest.Mu)
		}
		// On some axisymmetric variants the rear half is zero; B.15a
		// has its own nonzero zones 5/6, so this shortcut is conditional.
		lastZero := true
		for _, p := range subset {
			if p.Beta >= 195 && p.Beta <= 360 && math.Abs(p.Mu) > 1e-8 {
				lastZero = false
			}
		}
		if lastZero {
			appendRow(decimalInput(r, 3), "195…360", 0)
		}
	}
	return out
}
