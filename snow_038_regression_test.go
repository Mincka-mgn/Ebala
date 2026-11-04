package main

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

// Cross-check all half-arches, boundaries and independent adjacent-vault variants.
func Test038SnowPartialSegmentsAndExcel(t *testing.T) {
	for _, sh := range []string{"vault", "multi_vault"} {
		s := defaultSnowCalc()
		s.Shape = sh
		s.Span = 20
		s.Rise = 5
		s.Ground = 2
		s.Count = 4
		got, err := SnowCompute(s)
		if err != nil {
			t.Fatal(err)
		}
		examined := 0
		for _, c := range got.Cases {
			if !strings.HasPrefix(c.Title, "Частич") {
				continue
			}
			examined++
			for j, v := range c.Intervals {
				f, _ := snowExcelMuFormula(got, c, v, j+5)
				if v.Mu == 0 && f != "=0" {
					t.Errorf("%s / %s, interval %v: unload got %q", sh, c.Title, v, f)
				}
				if v.Mu > 0 && (f == "=0" || strings.Contains(f, "2*ABS(2*MOD")) {
					t.Errorf("%s / %s, interval %v: nonzero got %q", sh, c.Title, v, f)
				}
				if sh == "vault" && v.Mu > 0 {
					mid := (v.X1 + v.X2) / 2
					want := math.Max(0, math.Cos(1.5*snowVaultAngle(s.Span, s.Rise, mid)*math.Pi/180))
					if math.Abs(want-v.Mu) > 1e-12 {
						t.Errorf("vault interval %v: got %g want %g", v, v.Mu, want)
					}
				}
			}
		}
		if examined < 2 {
			t.Errorf("%s: insufficient partial cases %d", sh, examined)
		}
	}
}

func Test038MultiGableNormativePatterns(t *testing.T) {
	for _, n := range []int{2, 3, 4, 12} {
		for _, angle := range []float64{20, 20.01, 35, 60, 70} {
			s := defaultSnowCalc()
			s.Shape = "multi_gable"
			s.Count = n
			s.Span = 20
			s.Angle = angle
			s.Ground = 2
			got, err := SnowCompute(s)
			if err != nil {
				t.Fatal(err)
			}
			all := 0
			pairs := 0
			mu := snowRoofMu(angle)
			for _, c := range got.Cases {
				if !strings.HasPrefix(c.Title, "Частичная") {
					continue
				}
				if len(c.Intervals) != 2*n {
					t.Fatalf("n=%d angle=%.2f intervals %d", n, angle, len(c.Intervals))
				}
				if strings.Contains(c.Title, "один скат в каждом") {
					all++
					for j := 0; j < n; j++ {
						v := c.Intervals[2*j].Mu
						w := c.Intervals[2*j+1].Mu
						if mu > 0 && math.Abs(v+w-mu) > 1e-12 {
							t.Errorf("n=%d alpha=%g span=%d: wrong one slope (%v,%v)", n, angle, j, v, w)
						}
					}
				}
				if strings.Contains(c.Title, "два смежных ската") {
					pairs++
				}
				for i, v := range c.Intervals {
					f, _ := snowExcelMuFormula(got, c, v, i+5)
					if v.Mu == 0 && f != "=0" {
						t.Errorf("n=%d %s unload: %s", n, c.Title, f)
					}
				}
			}
			if angle <= 20 {
				if all != 0 || pairs != 0 {
					t.Errorf("angle=%g must not create partial", angle)
				}
			} else if all != 4 || pairs != minInt(2, n-1) {
				t.Errorf("%s: all=%d pairs=%d", fmt.Sprintf("n=%d angle=%g", n, angle), all, pairs)
			}
		}
	}
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Test038MonoSignNormalization(t *testing.T) {
	for _, a := range []float64{0, 20, 30, 45, 60} {
		for _, sign := range []float64{-1, 1} {
			s := defaultSnowCalc()
			s.Shape = "mono"
			s.Angle = a * sign
			s.Ground = 2
			got, e := SnowCompute(s)
			if e != nil {
				t.Fatal(e)
			}
			expected := snowRoofMu(a)
			if len(got.Cases) == 0 {
				t.Fatal("no variants")
			}
			if got.Cases[0].Intervals[0].Mu != expected {
				t.Errorf("%g: %g != %g", sign*a, got.Cases[0].Intervals[0].Mu, expected)
			}
			formula := snowExcelCoefData(got)
			if !strings.Contains(formula, `IF(ABS(B4)&lt;=30,1`) {
				t.Errorf("missing ABS in excel coefficient formula")
			}
		}
	}
}
