package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sameWeight(a, b float64) bool { return math.Abs(a-b) < 1e-8 }
func TestV05BiDirectionalEdits(t *testing.T) {
	basic := WeightRow{Name: "Засыпка", Kind: "layer", Thickness: 40, Density: 1800, Gamma: 1.3}
	// Before any manual normative edit, design input holds gamma and adjusts density/normative.
	r, e := EditWeightField(basic, "design", 130)
	if e != nil || !sameWeight(r.Gamma, 1.3) || !sameWeight(r.NormKgf(), 100) || r.NormEdited {
		t.Fatalf("normal branch: %+v %v", r, e)
	}
	// After manual normative change the same design input holds norm and changes gamma.
	r, e = EditWeightField(basic, "norm", 80)
	if e != nil || !r.NormEdited || !sameWeight(r.NormKgf(), 80) {
		t.Fatalf("norm branch: %+v %v", r, e)
	}
	r, e = EditWeightField(r, "design", 112)
	if e != nil || !sameWeight(r.Gamma, 1.4) || !sameWeight(r.NormKgf(), 80) {
		t.Fatalf("gamma branch: %+v %v", r, e)
	}
	r, e = EditWeightField(r, "gamma", 1.2)
	if e != nil || !sameWeight(r.NormKgf()*r.Gamma, 96) {
		t.Fatalf("gamma direct edit: %+v %v", r, e)
	}
	free := WeightRow{Name: "Непонятная хуетень", Kind: "direct", GNorm: 0, Gamma: 1.2}
	free, e = EditWeightField(free, "design", 42)
	if e != nil || !sameWeight(free.GNorm, 35) || !sameWeight(free.Gamma, 1.2) {
		t.Fatalf("direct branch: %+v %v", free, e)
	}
	free, e = EditWeightField(free, "norm", 40)
	if e != nil || !free.NormEdited {
		t.Fatal(e)
	}
	free, e = EditWeightField(free, "design", 52)
	if e != nil || !sameWeight(free.Gamma, 1.3) || !sameWeight(free.GNorm, 40) {
		t.Fatalf("direct changed gamma: %+v %v", free, e)
	}
}
func TestV05EditStatePersists(t *testing.T) {
	p := initialProject()
	p.Weight = &WeightCalc{Name: "Кровля", Rows: []WeightRow{{Name: "Стяжка", Kind: "direct", GNorm: 80, Gamma: 1.2, NormEdited: true}}}
	path := filepath.Join(t.TempDir(), "calc.ebala")
	if e := p.Save(path); e != nil {
		t.Fatal(e)
	}
	q, e := LoadProject(path)
	if e != nil {
		t.Fatal(e)
	}
	if q.Weight == nil || !q.Weight.Rows[0].NormEdited {
		t.Fatalf("edit precedence lost: %+v", q.Weight)
	}
}
func TestV05WordWeightPrecision(t *testing.T) {
	w := WeightCalc{Name: "Кровля", Rows: []WeightRow{{Name: "Плита", Kind: "direct", GNorm: 296.123, Gamma: 1.1}}}
	path := filepath.Join(t.TempDir(), "w.docx")
	if e := ExportWeightWord(w, path); e != nil {
		t.Fatal(e)
	}
	x := zipEntry(t, path, "word/document.xml")
	if !strings.Contains(x, "296,1") || !strings.Contains(x, "325,7") || strings.Contains(x, "296,123") {
		t.Fatalf("wrong decimal precision: %s", x)
	}
	ex := filepath.Join(t.TempDir(), "w.xlsx")
	if e := ExportWeightExcel(w, ex); e != nil {
		t.Fatal(e)
	}
	sx := zipEntry(t, ex, "xl/styles.xml")
	if !strings.Contains(sx, "0.0") || !strings.Contains(sx, "0.0000") {
		t.Fatal("load vs gamma styles missing")
	}
	if _, e := os.Stat(ex); e != nil {
		t.Fatal(e)
	}
}
