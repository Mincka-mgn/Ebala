package main

import (
	"archive/zip"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWind062ZeroRoofHasNoRoofCasesInAnyOutput(t *testing.T) {
	windTestData(t)
	in := testRoofCaseBase(0)
	in.PulseMode = "mean"
	c, err := CalculateWindVariants(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Variants) != 2 {
		t.Fatalf("expected two wind directions, got %d", len(c.Variants))
	}
	for _, v := range c.Variants {
		for _, z := range v.Zones {
			if z.Surface == "roof" {
				t.Fatalf("h=0 created roof case %s/%s", v.ID, z.Code)
			}
		}
	}
	dir := t.TempDir()
	report := TempReport{Version: 1, Winds: []WindCalculation{c}, WindTitles: []string{"Здание без подъёма кровли"}}
	doc := filepath.Join(dir, "Проверка.docx")
	xls := filepath.Join(dir, "Проверка.xlsx")
	if err = exportTempWord(report, doc); err != nil {
		t.Fatal(err)
	}
	if err = exportTempExcel(report, xls); err != nil {
		t.Fatal(err)
	}
	r, err := zip.OpenReader(doc)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			stream, _ := f.Open()
			raw, _ := io.ReadAll(stream)
			stream.Close()
			if strings.Contains(string(raw), "Результаты для кровли") || strings.Contains(string(raw), "зона F") || strings.Contains(string(raw), "Разбивка двускатной кровли") {
				t.Fatal("h=0 roof escaped into DOCX")
			}
		}
	}
	r2, err := zip.OpenReader(xls)
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Close()
	sheets := 0
	for _, f := range r2.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/") && strings.HasSuffix(f.Name, ".xml") {
			sheets++
		}
	}
	if sheets < 5 {
		t.Fatalf("Excel missing calculated wall sheets: %d", sheets)
	}
}

func TestWind062CustomerWordIsCleanAndGrouped(t *testing.T) {
	windTestData(t)
	c, err := CalculateWindVariants(testRoofCaseBase(20))
	if err != nil {
		t.Fatal(err)
	}
	report := TempReport{Version: 1, Winds: []WindCalculation{c}, WindTitles: []string{"Промышленный корпус"}}
	path := filepath.Join(t.TempDir(), "Ветровая нагрузка.docx")
	if err := exportTempWord(report, path); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var xml string
	figures := 0
	for _, f := range reader.File {
		if strings.HasPrefix(f.Name, "word/media/wind_") {
			figures++
		}
		if f.Name == "word/document.xml" {
			rd, _ := f.Open()
			bytes, _ := io.ReadAll(rd)
			rd.Close()
			xml = string(bytes)
		}
	}
	for _, bad := range []string{"Идентификаторы применённых формул", "для воспроизводимости", "полный массив точек", "независимом XLSX", "По формуле (11.1):", "Расчётное значение: wрасч =", "Коэффициент k(zₑ) ="} {
		if strings.Contains(strings.ToLower(xml), strings.ToLower(bad)) {
			t.Fatalf("customer DOCX contains developer text: %q", bad)
		}
	}
	for _, want := range []string{"Ветер вдоль конька", "Ветер поперёк конька", "Итого нормативная", "Итого расчётная", "wₘ =", "Знаковый вариант 2", "Размеры участков кровли", `<w:jc w:val="center"/>`} {
		if !strings.Contains(xml, want) {
			t.Errorf("missing customer-facing item %q", want)
		}
	}
	if figures != 4 {
		t.Errorf("expected two explanatory pictures per direction, got %d", figures)
	}
}

func TestWind062DiagramAlwaysLinearScale(t *testing.T) {
	pts := []float64{0, 2, 5, 105}
	axis := windAxis(pts, 100, 900)
	if axis.px(0) != 100 || axis.px(105) != 1000 {
		t.Fatal("axis extrema corrupted")
	}
	for i, x := range pts {
		if math.Abs(axis.cuts[i]-x) > 1e-12 {
			t.Fatal("axis modified original coordinates")
		}
	}
	for _, x := range []float64{2, 5, 40, 75} {
		want := 100 + int(math.Round(900*x/105))
		if axis.px(x) != want {
			t.Fatalf("conditional break remains at %.2f: %d vs %d", x, axis.px(x), want)
		}
	}
}

func TestWind062NumbersRussianAndNoFalsePrecision(t *testing.T) {
	for _, v := range []struct {
		x float64
		n int
		s string
	}{
		{.920000, 3, "0,92"}, {2, 3, "2,0"}, {1.92449, 3, "1,924"}, {-0.685637, 3, "−0,686"}, {1215.002, 3, "1 215,002"}, {12.04, 1, "12,0"},
	} {
		if a := windWordFmt(v.x, v.n); a != v.s {
			t.Errorf("%v -> %q expected %q", v.x, a, v.s)
		}
	}
}

var _ = os.ErrNotExist
