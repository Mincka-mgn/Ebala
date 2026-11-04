package main

import (
	"math"
	"path/filepath"
	"testing"
)

func TestAutoCraneSpans022(t *testing.T) {
	for _, input := range []string{"36;36", "36 36", "36; 36"} {
		mm, w, e := parseCraneSpanMetersAuto(input)
		if e != nil || w || len(mm) != 2 || mm[0] != 36000 || mm[1] != 36000 {
			t.Fatalf("%q: %v %v %v", input, mm, w, e)
		}
	}
	if mm, _, e := parseCraneSpanMetersAuto("6,5"); e != nil || len(mm) != 1 || mm[0] != 6500 {
		t.Fatalf("decimal metres: %v %v", mm, e)
	}
	if _, warn, e := parseCraneSpanMetersAuto("120"); e != nil || !warn {
		t.Fatalf("warning: %v %v", warn, e)
	}
}

func TestExcelMultiWheel022(t *testing.T) {
	pts := make([]CranePoint, 32)
	for i := range pts {
		pts[i] = CranePoint{XMM: 1000 + float64(i)*1100, PKN: 650 + float64(i%5)*27, Crane: "Кран 400 т"}
	}
	r := CranePlacementResult{SpansMM: []float64{36000, 36000}}
	c := CranePlacementCase{Name: "M+ в середине пролёта 1", Units: "кН·м", SectionMM: 18000, Points: pts}
	out, e := solveCraneBeam(r.SpansMM, pts, []float64{18})
	if e != nil {
		t.Fatal(e)
	}
	c.Value = out.moments[0]
	if math.IsNaN(c.Value) {
		t.Fatal("nan")
	}
	r.Cases = []CranePlacementCase{c}
	if err := exportCranePlacementExcel(r, filepath.Join(t.TempDir(), "multiwheel.xlsx")); err != nil {
		t.Fatal(err)
	}
}
