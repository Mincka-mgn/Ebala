package main

import (
	"archive/zip"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Regression for 0.55: B.15a is a circle and must not receive an
// independent rectangular B or the rectangular lc computation (10.3).
func Test055SphereConeDoesNotInventWidthOrLc(t *testing.T) {
	s := snow048SphereConeSample()
	s.Width = 12345 // deliberately unrelated to the geometry of a circular roof
	input := snowExcelInputData(s)
	coef := snowExcelCoefData(s)
	guide := snowExcelFormulaGuide(s)
	if !strings.Contains(input, "D — диаметр покрытия") || !strings.Contains(coef, "D — диаметр, м") {
		t.Fatal("diameter must be the only horizontal dimension of B.15a")
	}
	cell := func(body, address string) bool {
		return regexp.MustCompile(`<c[^>]* r="` + address + `"`).MatchString(body)
	}
	for _, tc := range []struct{ label, data string }{{"input", input}, {"coef", coef}} {
		if cell(tc.data, "B12") || cell(tc.data, "A12") || (tc.label == "coef" && (cell(tc.data, "B6") || cell(tc.data, "A6"))) {
			t.Fatalf("%s contains a made-up B or lc cell", tc.label)
		}
		if strings.Contains(tc.data, "12345") {
			t.Fatalf("%s copied the rectangular width to B.15a", tc.label)
		}
	}
	if !strings.Contains(coef, "ce для Б.15а") || !strings.Contains(coef, "ct по п. 10.10") || !strings.Contains(coef, "<f>B45</f>") || !strings.Contains(coef, "<f>B46</f>") {
		t.Fatal("the existing snow load coefficient links must remain live")
	}
	if strings.Contains(guide, "lc = min(100;") || !strings.Contains(guide, "диаметр круглого покрытия") {
		t.Fatal("the formula guide must not instruct a rectangular calculation for B.15a")
	}
	gable := s
	gable.Shape = "gable"
	if !cell(snowExcelInputData(gable), "B12") || !cell(snowExcelCoefData(gable), "B6") || !cell(snowExcelCoefData(gable), "B12") {
		t.Fatal("noncircular roofs must retain their independent width and lc")
	}
}

// The previously approved engineering interpretation of zone 5 uses the
// cone tangent at the sphere/cone junction, not the sphere tangent at the wall.
func Test055SphereConeZone5AndOtherLoadsUnchangedByWidth(t *testing.T) {
	a := snow048SphereConeSample()
	a.Angle = 18
	a.Width = 0
	b := a
	b.Width = 1000
	ga, err := makeSphereConeGeometry(a)
	if err != nil {
		t.Fatal(err)
	}
	gb, err := makeSphereConeGeometry(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, radius := range []float64{0, 3, a.SkylightWidth, a.SkylightWidth + 0.01, a.Span / 2} {
		for _, beta := range []float64{0, 30, 60, 90, 120, 150, 180} {
			ma, za := ga.spatialMu(radius, beta)
			mb, zb := gb.spatialMu(radius, beta)
			if za != zb || math.Abs(ma-mb) > 1e-12 {
				t.Fatalf("B changed physical load at r=%g beta=%g: (%g/%d) vs (%g/%d)", radius, beta, ma, za, mb, zb)
			}
		}
	}
	if ga.StrictZone5Formula || gb.StrictZone5Formula {
		t.Fatal("B.15a approved default interpretation has changed")
	}
	// The entire public calculation must work without a rectangular B input.
	ca, err := SnowCompute(a)
	if err != nil {
		t.Fatal(err)
	}
	cb, err := SnowCompute(b)
	if err != nil {
		t.Fatal(err)
	}
	if ca.Width != ca.Span || cb.Width != cb.Span || len(ca.Cases) != len(cb.Cases) {
		t.Fatal("a circular plan must use D internally without requiring B")
	}
	for i := range ca.Cases {
		if math.Abs(ca.MaxMu(ca.Cases[i])-cb.MaxMu(cb.Cases[i])) > 1e-12 {
			t.Fatal("a fictitious B changed the full computed B.15a load")
		}
	}
	mu, zone := ga.spatialMu(a.SkylightWidth/2, 270)
	if zone != 5 || mu <= 0 {
		t.Fatalf("zone 5 must retain the approved load at cone slope 18°, got zone=%d mu=%g", zone, mu)
	}
}

func Test055B15aStandaloneWorkbookNoFakeRectangularCells(t *testing.T) {
	s := snow048SphereConeSample()
	c, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sphere_cone_055.xlsx")
	if err := ExportSnowExcel(c, path); err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var hasDiameter, hasZone5, hasCe bool
	for _, f := range z.File {
		if !strings.HasPrefix(f.Name, "xl/worksheets/") || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			t.Fatal(err)
		}
		st := string(content)
		hasDiameter = hasDiameter || strings.Contains(st, "D — диаметр покрытия")
		hasZone5 = hasZone5 || strings.Contains(st, "Угол для зоны 5")
		hasCe = hasCe || strings.Contains(st, "ce для Б.15а")
	}
	if !hasDiameter || !hasZone5 || !hasCe {
		t.Fatalf("incomplete B.15a workbook, diameter=%v, zone5=%v, ce=%v", hasDiameter, hasZone5, hasCe)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
