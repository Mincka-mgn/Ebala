package main

import (
	"math"
	"strings"
	"testing"
)

// External cross-check: compare intended Go snow intervals with Excel cell formulas.
func TestAudit037ExcelVaultPartial(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "vault"
	s.Span = 20
	s.Rise = 5
	s.Ground = 2
	out, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	for _, c := range out.Cases {
		if !strings.HasPrefix(c.Title, "Частичное загружение половины свода") {
			continue
		}
		for _, v := range c.Intervals {
			if v.X1 >= s.Span/2 && v.Mu == 0 {
				f, _ := snowExcelMuFormula(out, c, v, 15)
				// Excel must compute the unloaded half dynamically from editable span and coordinate, not copy a Go-derived constant.
				if !strings.HasPrefix(f, "=IF(") || !strings.Contains(f, ",0)") || !strings.Contains(f, "Исходные данные") {
					t.Errorf("vault partial: Go mu=0 at x=[%.3f,%.3f], Excel generated %s", v.X1, v.X2, f)
				}
				return // The audit fixture forgot to return on a successful match.
			}
		}
		t.Fatal("could not locate unloaded vault segment")
	}
	t.Fatal("vault partial case missing")
}
func TestAudit037ExcelMultivaultPartial(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "multi_vault"
	s.Span = 20
	s.Count = 2
	s.Rise = 5
	s.Ground = 2
	out, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	for _, c := range out.Cases {
		if !strings.Contains(c.Title, "Частичная нагрузка") {
			continue
		}
		for _, v := range c.Intervals {
			if v.Mu == 0 && v.X1 == 0 {
				f, _ := snowExcelMuFormula(out, c, v, 15)
				if !strings.HasPrefix(f, "=IF(") || !strings.Contains(f, ",0)") || !strings.Contains(f, "Исходные данные") {
					t.Errorf("multivault partial: Go mu=0 at [%.2f,%.2f], Excel generated %s", v.X1, v.X2, f)
				}
				return // The audit fixture forgot to return on a successful match.
			}
		}
		t.Fatal("could not locate unloaded multi-vault segment")
	}
	t.Fatal("multi-vault partial case missing")
}
func TestAudit037ExcelMonoNegativeSlope(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "mono"
	s.Angle = -45
	s.Ground = 2
	out, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if math.Abs(out.Cases[0].Intervals[0].Mu-0.5) > 1e-10 {
		t.Fatalf("expected Go mu=0.5 got %v", out.Cases[0].Intervals[0].Mu)
	}
	coefs := snowExcelCoefData(out)
	if strings.Contains(coefs, "IF(B4&lt;=30,1,IF(B4&gt;=60,0,(60-B4)/30))") {
		t.Errorf("mono negative slope: Go mu=0.5, Excel B22 evaluated with -45° gives 1.0 (Sg=2: Go 1 kPa, Excel 2 kPa)")
	}
}

// Normative SP 20 p. 10.4: multi-span gables need at least the alternative
// "one slope in EVERY span", not merely one slope in one selected span.
func TestAudit037MultiGableAllBayPartial(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "multi_gable"
	s.Span = 20
	s.Count = 3
	s.Angle = 35
	s.Ground = 2
	out, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	maxLoaded := 0
	for _, c := range out.Cases {
		if !strings.Contains(c.Title, "Частич") {
			continue
		}
		n := 0
		for _, v := range c.Intervals {
			if v.Mu > 0 {
				n++
			}
		}
		if n > maxLoaded {
			maxLoaded = n
		}
	}
	if maxLoaded < s.Count {
		t.Errorf("СП 20.13330.2016 п.10.4: 3-span 35deg gable requires one loaded slope per each bay (3), generated partial variants contain at most %d loaded slopes", maxLoaded)
	}
}
