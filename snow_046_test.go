package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test046ShedNormativeVariantsAndHeight(t *testing.T) {
	s := sampleSnow("shed")
	s.Count = 3
	s.Span = 12
	s.Width = 40
	s.RoofHeight = 9
	s.JanuarySet = true
	s.JanuaryTemp = -17
	s.WinterWindSet = true
	s.WinterWind = 5
	first, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(first.Cases) != 2 || len(first.Cases[1].Intervals) != 6 {
		t.Fatalf("Б.7: %d cases, %d parts", len(first.Cases), len(first.Cases[1].Intervals))
	}
	for i, a := range first.Cases[1].Intervals {
		want := 1.4
		if i%2 == 1 {
			want = .6
		}
		if math.Abs(a.Mu-want) > 1e-12 {
			t.Fatalf("interval %d: got %.3g need %.3g", i, a.Mu, want)
		}
	}
	if math.Abs(first.CE-1) > 1e-12 {
		t.Fatalf("Б.4 should not misuse ordinary wind reduction: ce=%g", first.CE)
	}
	s.RoofHeight = 18
	second, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if math.Abs(first.Q(1.4)-second.Q(1.4)) > 1e-12 {
		t.Fatal("ordinary B.4 load changed just because ridge is taller")
	}
}
func Test046ParapetGeometryAndDrift(t *testing.T) {
	s := sampleSnow("parapet")
	s.Ground = 2
	s.H = 2
	s.Span = 10
	s.Width = 18
	s.Angle = 0
	got, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(got.Cases) != 3 {
		t.Fatalf("expected basic, B.13, and B.8 check; got %d", len(got.Cases))
	}
	if math.Abs(got.MaxMu(got.Cases[1])-2) > 1e-12 {
		t.Fatalf("μmax at parapet: got %g", got.MaxMu(got.Cases[1]))
	}
	if got.Cases[1].Intervals[len(got.Cases[1].Intervals)-1].Mu != 1 {
		t.Fatal("far field must have μ=1")
	}
	s.H = .8
	got, e = SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(got.Cases) != 1 {
		t.Fatalf("local drift shouldn't appear for h<=S0/2: %d", len(got.Cases))
	}
}
func Test046ApprovedSnowAssets(t *testing.T) {
	for _, f := range []string{"snow_B13_parapet.png", "snow_B14_full.png", "snow_B15_full.png", "snow_B3_lantern_zones_preview.png", "snow_B_shed_roof_H_final_preview.png"} {
		b, e := os.ReadFile(filepath.Join("Resources", "SnowRoofs", f))
		if e != nil || len(b) == 0 {
			t.Fatalf("%s: %v", f, e)
		}
	}
}
func Test046ShedExcelUsesFormula(t *testing.T) {
	s := sampleSnow("shed")
	s.Count = 2
	s.Span = 12
	c, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	formula, _ := snowExcelMuFormula(c, c.Cases[1], c.Cases[1].Intervals[0], 5)
	if !strings.Contains(formula, "MOD(") {
		t.Fatalf("not formula driven: %s", formula)
	}
}

func Test046ExportShedAndParapet(t *testing.T) {
	for _, kind := range []string{"shed", "parapet"} {
		s := sampleSnow(kind)
		s.Span = 20
		s.Width = 18
		s.RoofHeight = 12
		s.H = 2
		s.Count = 3
		result, e := SnowCompute(s)
		if e != nil {
			t.Fatal(kind, e)
		}
		dir := t.TempDir()
		doc := filepath.Join(dir, "report.docx")
		xl := filepath.Join(dir, "report.xlsx")
		if e := ExportSnowWord(result, doc); e != nil {
			t.Fatalf("%s Word: %v", kind, e)
		}
		if e := ExportSnowExcel(result, xl); e != nil {
			t.Fatalf("%s Excel: %v", kind, e)
		}
		for _, path := range []string{doc, xl} {
			f, e := os.Stat(path)
			if e != nil || f.Size() < 1000 {
				t.Fatalf("%s output %v", path, e)
			}
		}
	}
}

func Test046WideSkylightDrift(t *testing.T) {
	s := sampleSnow("skylight")
	s.Span = 68
	s.SkylightWidth = 6
	s.Width = 20
	s.H = 2
	s.Angle = 0
	s.Rise = 0
	result, e := SnowCompute(s)
	if e != nil {
		t.Fatalf("wide flat skylight must calculate B.8 variant, not block: %v", e)
	}
	n := 0
	for _, c := range result.Cases {
		if strings.Contains(c.Title, "Дополнительное местное") {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("expected two additional spillover directions, got %d of %d", n, len(result.Cases))
	}
}
