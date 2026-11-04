package main

import (
	"archive/zip"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func near039(t *testing.T, label string, a, b float64) {
	t.Helper()
	if math.Abs(a-b) > 1e-9*math.Max(1, math.Abs(b)) {
		t.Errorf("%s got %.12g expected %.12g", label, a, b)
	}
}
func test039Inputs(shape string, h float64) IceLoadCalc {
	s := defaultIceLoad()
	s.Region = "III"
	s.BaseMM = 10
	s.Shape = shape
	s.DiameterMM = 20
	s.HeightStart = h
	s.HeightEnd = h
	s.Wind = IceWindInput{Enabled: true, Zone: "II", Terrain: "B", Drag: 1.2}
	return s
}
func Test039WindIceIndependentTwoCases(t *testing.T) {
	for _, shape := range []string{"round_wire", "round", "surface_small"} {
		s := test039Inputs(shape, 20)
		if shape == "surface_small" {
			s.DiameterMM = 0
		}
		out, e := IceCompute(s)
		if e != nil {
			t.Fatal(e)
		}
		p := out.Points[0]
		thick := 12. * .9 // b*k*mu1 for d20
		expectedIce := math.Pi * thick * (20 + thick) * .9 * 9.81 * .001
		if shape == "surface_small" {
			expectedIce = 10 * 1.2 * .6 * .9 * 9.81
		}
		near039(t, "ice norm", p.Norm, expectedIce)
		near039(t, "ice gamma", p.Design, expectedIce*1.8)
		k := .85
		w0 := 300.
		c := 1.2
		area := 1.
		bare := 1.
		if shape != "surface_small" {
			area = (20 + 2*thick) / 1000
			bare = .020
		}
		wm := w0 * k * c * area
		near039(t, "wind mean", p.WindMean, wm)
		psi := .6
		if shape == "round_wire" {
			psi = .25
		}
		near039(t, "ice leading wind", p.WindDesign, wm*1.4*psi)
		near039(t, "wind leading wind", p.WindLeading, w0*k*c*bare*1.4)
		if p.WindLeading < 0 || p.WindDesign < 0 {
			t.Fatalf("negative loads")
		}
	}
}
func Test039WindTablesAndHeightBands(t *testing.T) {
	s := test039Inputs("round_wire", 5)
	for _, tc := range []struct{ h, ki, kw float64 }{{5, .8, .5}, {10, 1, .65}, {20, 1.2, .85}, {100, 2, 1.6}, {150, 0, 1.9}, {200, 1, 2.1}, {250, 1 + .5*(45./35-1), 2.3}, {300, 1, 2.5}} {
		s.HeightStart = tc.h
		s.HeightEnd = tc.h
		out, e := IceCompute(s)
		if e != nil {
			t.Fatalf("h=%g: %v", tc.h, e)
		}
		p := out.Points[0]
		near039(t, "wind k", p.WindK, tc.kw)
		if tc.ki > 0 {
			near039(t, "ice k", p.K, tc.ki)
		}
		if tc.h == 200 {
			near039(t, "horizontal b at 200", p.BaseMM, 31.5)
		}
		if tc.h == 150 {
			near039(t, "100..200 clamp", p.K, math.Min(math.Exp(.007*tc.h), 35/10.))
		}
	}
}
func Test039WindLimitAndVerifiedPulsation(t *testing.T) {
	s := test039Inputs("round", 301)
	if _, e := IceCompute(s); e == nil || !strings.Contains(e.Error(), "300") {
		t.Fatalf("higher than 300 m wind must be blocked %v", e)
	}
	s.HeightStart = 30
	s.HeightEnd = 30
	a, e := IceCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	s.Wind.PulseVerified = true
	s.Wind.PulseRatio = .45
	s.Wind.PulseSource = "Проверка пульсаций по п. 11.1.8, расчёт 1"
	b, e := IceCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	near039(t, "verified external pulsation", b.Points[0].WindNorm, a.Points[0].WindNorm*1.45)
	s.Wind.PulseSource = ""
	if _, e := IceCompute(s); e == nil {
		t.Fatal("Missing verified pulse reference must fail")
	}
}
func Test039SpecialHighValuesNeedEvidence(t *testing.T) {
	s := test039Inputs("round", 200)
	s.HighZone = "special"
	s.HighValuesMM = []float64{40, 55, 70}
	if _, e := IceCompute(s); e == nil {
		t.Fatal("special high loads without study must fail")
	}
	s.StudyReference = "Акт метеоизысканий 2026"
	o, e := IceCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	near039(t, "verified study override", o.Points[0].BaseMM, 40)
}
func Test039WordExcelAndDiagramIntegration(t *testing.T) {
	s := test039Inputs("round_wire", 10)
	s.HeightEnd = 60
	s.HeightStep = 10
	s.Wind.PulseVerified = true
	s.Wind.PulseRatio = .32
	s.Wind.PulseSource = "Ветровой расчёт №123"
	s.Wind.DragSource = "Приложение В, проектный расчёт №123"
	out, e := IceCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	plot, e := iceFigure(out)
	if e != nil || len(plot) < 1000 {
		t.Fatalf("diagram %v", e)
	}
	for _, needle := range []string{"$B$41", "$B$42", "$B$43", "Гололёд табл.12.3", "Ветер табл.11.1", "J5"} {
		if !strings.Contains(iceExcelCalcData(out), needle) {
			t.Errorf("missing excel formula %q", needle)
		}
	}
	dir := t.TempDir()
	doc := filepath.Join(dir, "Гололёд.docx")
	book := filepath.Join(dir, "Гололёд.xlsx")
	if e = ExportIceWord(out, doc); e != nil {
		t.Fatal(e)
	}
	if e = ExportIceExcel(out, book); e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{doc, book} {
		z, e := zip.OpenReader(path)
		if e != nil {
			t.Fatal(e)
		}
		files := make(map[string]bool)
		for _, f := range z.File {
			files[f.Name] = true
		}
		if !files["[Content_Types].xml"] {
			t.Errorf("missing OOXML content type in %s", path)
		}
		if len(files) < 5 {
			t.Errorf("too few file parts %s: %d", path, len(files))
		}
		z.Close()
	}
	if dest := os.Getenv("EBALA_QA_OUT"); dest != "" {
		os.MkdirAll(dest, 0755)
		for _, x := range []struct{ from, to string }{{doc, "ice_wind.docx"}, {book, "ice_wind.xlsx"}} {
			data, _ := os.ReadFile(x.from)
			os.WriteFile(filepath.Join(dest, x.to), data, 0644)
		}
		v, _ := json.MarshalIndent(out, "", "  ")
		os.WriteFile(filepath.Join(dest, "ice_wind.json"), v, 0644)
		os.WriteFile(filepath.Join(dest, "ice_wind.png"), plot, 0644)
	}
}

func Test039SpecialDensityAndCoatedAerodynamics(t *testing.T) {
	s := test039Inputs("round_wire", 50)
	s.DensityKgM3 = 850
	s.StudyReference = "Акт лабораторного исследования плотности льда, испытание №5"
	s.Wind.DragIced = 1.6
	out, e := IceCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	p := out.Points[0]
	original := test039Inputs("round_wire", 50)
	base, e := IceCompute(original)
	if e != nil {
		t.Fatal(e)
	}
	near039(t, "special documented density", p.Norm, base.Points[0].Norm*850/900)
	near039(t, "special coated c", p.WindNorm, base.Points[0].WindNorm*1.6/1.2)
	if !strings.Contains(iceExcelCalcData(out), "$B$46") {
		t.Fatal("coated drag coefficient is not in Excel formulas")
	}
	s.StudyReference = ""
	if _, e := IceCompute(s); e == nil {
		t.Fatal("undocumented modified density was accepted")
	}
}
