package main

import (
	"math"
	"strings"
)

// The normative B.13 drift is a straight segment from the parapet peak to
// the ordinary snow cover, not a stepped histogram. This is only the visual
// profile; conservative discrete intervals remain available for audit.
func snowParapetLocalCase(s SnowCalc, c SnowCase) bool {
	return s.Shape == "parapet" && strings.Contains(c.Title, "Местная нагрузка у парапета") && s.H > 0 && s.Ground > 0
}
func snowParapetPeak(ground, height float64) float64 {
	if ground <= 0 || height <= 0 {
		return 1
	}
	return math.Min(3, 2*height/ground)
}
func snowParapetMuAt(ground, height, x float64) float64 {
	if height <= 0 {
		return 1
	}
	if x <= 0 {
		return snowParapetPeak(ground, height)
	}
	if x >= 2*height {
		return 1
	}
	return snowParapetPeak(ground, height) + (1-snowParapetPeak(ground, height))*x/(2*height)
}

// Only physical discontinuities and the onset of non-zero loading are marked
// on the dimension line. The number is computed from interval X-coordinates.
func snowEpureStartCoordinates(c SnowCase) []float64 {
	if len(c.Intervals) < 2 {
		return nil
	}
	maxX := c.Intervals[len(c.Intervals)-1].X2
	minX := c.Intervals[0].X1
	out := make([]float64, 0, 4)
	for i := 1; i < len(c.Intervals); i++ {
		a, b := c.Intervals[i-1], c.Intervals[i]
		if math.Abs(a.Mu) < 1e-9 && b.Mu > 1e-9 && b.X1 > minX+1e-8 && b.X1 < maxX-1e-8 {
			if len(out) == 0 || math.Abs(out[len(out)-1]-b.X1) > maxX*0.02 {
				out = append(out, b.X1)
			}
		}
	}
	return out
}

// snowParapetProfile reconstructs the *display* polyline of both normative
// parapet alternatives. The numeric conservative interval table is preserved
// for the calculation/audit. It must not be mistaken for a stepped epure.
type snowProfilePoint struct{ X, Mu float64 }

func snowParapetProfile(s SnowCalc, c SnowCase) ([]snowProfilePoint, bool) {
	if s.Shape != "parapet" || len(c.Intervals) == 0 {
		return nil, false
	}
	minX := c.Intervals[0].X1
	maxX := c.Intervals[len(c.Intervals)-1].X2
	if snowParapetLocalCase(s, c) {
		cut := math.Min(maxX, minX+2*s.H)
		points := []snowProfilePoint{{minX, snowParapetMuAt(s.Ground, s.H, 0)}, {cut, snowParapetMuAt(s.Ground, s.H, cut-minX)}}
		if cut < maxX {
			points = append(points, snowProfilePoint{maxX, 1})
		}
		return points, true
	}
	if !strings.Contains(c.Title, "Дополнительная проверка снегозаноса") {
		return nil, false
	}
	// This B.8 alternative has a linear descent from the local maximum to
	// the far-field background and then a constant tail. Its 30 samples were
	// generated at each segment's LEFT endpoint in snowCases(). Therefore the
	// true slope is obtained from the first two samples, not by connecting
	// their horizontal step edges.
	peak := c.Intervals[0].Mu
	background := c.Intervals[len(c.Intervals)-1].Mu
	points := []snowProfilePoint{{minX, peak}}
	if len(c.Intervals) > 1 {
		step := c.Intervals[1].X1 - minX
		if step > 0 {
			slope := (c.Intervals[1].Mu - peak) / step
			if slope < -1e-12 && peak > background+1e-10 {
				cut := minX + (background-peak)/slope
				if cut > minX && cut < maxX {
					points = append(points, snowProfilePoint{cut, background})
				}
			}
		}
	}
	points = append(points, snowProfilePoint{maxX, background})
	return points, true
}

func snowProfileMuAt(points []snowProfilePoint, x float64) float64 {
	if len(points) == 0 {
		return 0
	}
	if x <= points[0].X {
		return points[0].Mu
	}
	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		if x <= b.X {
			if b.X <= a.X {
				return b.Mu
			}
			return a.Mu + (b.Mu-a.Mu)*(x-a.X)/(b.X-a.X)
		}
	}
	return points[len(points)-1].Mu
}
