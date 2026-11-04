package main

import (
	"archive/zip"
	"io"
	"math"
	"path/filepath"
	"strings"
	"testing"
)

func exampleWeight() WeightCalc {
	return WeightCalc{Name: "Покрытие (проверка)", Rows: []WeightRow{
		{Name: "Стяжка", Kind: "layer", Thickness: 40, Density: 1800, Gamma: 1.3},
		{Name: "Плита", Kind: "direct", GNorm: 300, Gamma: 1.1},
		{Name: "Непонятная хуетень", Kind: "direct", GNorm: 35, Gamma: 1.2},
	}}
}
func TestV04WeightEffectiveGamma(t *testing.T) {
	w := exampleWeight()
	n, d, g := w.Totals()
	// 40*1800/1000=72, so total normative=72+300+35=407
	if n != 407 {
		t.Errorf("norm %.12g", n)
	}
	if math.Abs(d-(72*1.3+300*1.1+35*1.2)) > 1e-9 {
		t.Errorf("design %g", d)
	}
	if math.Abs(g-d/n) > 1e-12 {
		t.Errorf("effective gamma %g", g)
	}
	p := initialProject()
	p.Weight = &w
	r, e := p.Calc()
	if e != nil {
		t.Fatal(e)
	}
	if math.Abs(r.SWN-n*0.00980665) > 1e-12 || math.Abs(r.SWD-d*0.00980665) > 1e-12 {
		t.Fatalf("unit mismatch %+v", r)
	}
	if math.Abs(r.SWD/r.SWN-g) > 1e-12 {
		t.Errorf("effective gamma mismatch")
	}
}
func TestV04ZeroLoadAndValidation(t *testing.T) {
	w := WeightCalc{Name: "Пустая", Rows: []WeightRow{}}
	n, d, g := w.Totals()
	if n != 0 || d != 0 || g != 0 {
		t.Fatal("unexpected zero summary")
	}
	if e := w.Validate(); e != nil {
		t.Fatal(e)
	}
	w.Rows = []WeightRow{{Name: "Некорректно", Kind: "direct", GNorm: 5, Gamma: 0}}
	if w.Validate() == nil {
		t.Error("zero gamma accepted")
	}
}
func zipEntry(t *testing.T, file, entry string) string {
	t.Helper()
	z, e := zip.OpenReader(file)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	for _, f := range z.File {
		if f.Name == entry {
			r, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			defer r.Close()
			b, e := io.ReadAll(r)
			if e != nil {
				t.Fatal(e)
			}
			return string(b)
		}
	}
	t.Fatalf("missing %s", entry)
	return ""
}
func TestV04EditableReports(t *testing.T) {
	w := exampleWeight()
	x := filepath.Join(t.TempDir(), "weight.xlsx")
	if e := ExportWeightExcel(w, x); e != nil {
		t.Fatal(e)
	}
	sx := zipEntry(t, x, "xl/worksheets/sheet1.xml")
	for _, v := range []string{"C3*D3/1000", "E4*F4", "IF(E6=0,&#34;&#34;,G6/E6)", "Непонятная хуетень"} {
		if !strings.Contains(sx, v) {
			t.Errorf("missing XLSX element %q", v)
		}
	}
	wd := filepath.Join(t.TempDir(), "weight.docx")
	if e := ExportWeightWord(w, wd); e != nil {
		t.Fatal(e)
	}
	ds := zipEntry(t, wd, "word/document.xml")
	for _, v := range []string{"Непонятная хуетень", "коэффициент надёжности", "407"} {
		if !strings.Contains(ds, v) {
			t.Errorf("missing DOCX %q", v)
		}
	}
	p := initialProject()
	p.Weight = &w
	fullx := filepath.Join(t.TempDir(), "full.xlsx")
	if e := ExportExcel(p, fullx); e != nil {
		t.Fatal(e)
	}
	full := zipEntry(t, fullx, "xl/worksheets/sheet1.xml")
	if !strings.Contains(full, "Непонятная хуетень") || !strings.Contains(full, "IF(E6=0,&#34;&#34;,G6/E6)") {
		t.Fatal("native project export didn't use new table")
	}
	fullw := filepath.Join(t.TempDir(), "full.docx")
	if e := ExportWord(p, fullw); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(zipEntry(t, fullw, "word/document.xml"), "Непонятная хуетень") {
		t.Fatal("native Word report missing direct row")
	}
}
func TestV04ProjectBackwardCompatibility(t *testing.T) {
	before := initialProject()
	file := filepath.Join(t.TempDir(), "old.json")
	if e := before.Save(file); e != nil {
		t.Fatal(e)
	}
	after, e := LoadProject(file)
	if e != nil {
		t.Fatal(e)
	}
	if after.Weight != nil {
		t.Error("old file unexpectedly contains new weight")
	}
	w := after.CurrentWeight()
	if len(w.Rows) != len(after.Layers) {
		t.Fatal("legacy conversion failed")
	}
	after.Weight = &w
	file2 := filepath.Join(t.TempDir(), "new.json")
	if e := after.Save(file2); e != nil {
		t.Fatal(e)
	}
	saved, e := LoadProject(file2)
	if e != nil {
		t.Fatal(e)
	}
	if saved.Weight == nil || len(saved.Weight.Rows) != len(w.Rows) {
		t.Fatal("new project lost its rows")
	}
}
