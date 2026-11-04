package main

import (
	"archive/zip"
	"io"
	"math"
	"path/filepath"
	"strings"
	"testing"
)

func TestSnowV030IndependentRoofSections(t *testing.T) {
	s := sampleSnow("step")
	s.UpperRoof = SnowRoofPart{Shape: "gable", Span: 20, Angle: 20, EdgeLevel: 10}
	s.LowerRoof = SnowRoofPart{Shape: "mono", Span: 14, Rise: 1.5, SlopeMode: "height", EdgeLevel: 6}
	s.Upper = 20
	s.Lower = 14
	s.H = 4
	s.Span = 34
	s.Angle = 6
	s.Rise = 1.5
	x, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(x.Cases) < 4 {
		t.Fatalf("upper and lower alternatives omitted: %d", len(x.Cases))
	}
	if math.Abs(x.Cases[0].Intervals[0].X1+20) > 1e-6 {
		t.Fatal("upper roof negative axis missing")
	}
	found := false
	for _, c := range x.Cases {
		if strings.Contains(c.Title, "Основное верхнее") {
			found = true
		}
	}
	if !found {
		t.Fatal("missing independent upper variants")
	}
}
func TestSnowV030LanternRunAndConeSpatial(t *testing.T) {
	s := sampleSnow("skylight")
	s.Span = 18
	s.SkylightWidth = 6
	s.Rise = 1
	s.SlopeMode = "height"
	s.Angle = 0
	x, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	want := math.Atan2(1, 6) * 180 / math.Pi
	if math.Abs(x.Angle-want) > 1e-6 {
		t.Fatalf("Lantern slope: %.4f != %.4f", x.Angle, want)
	}
	c := sampleSnow("cone")
	c.Angle = 25
	x, e = SnowCompute(c)
	if e != nil {
		t.Fatal(e)
	}
	if len(x.Cases) < 2 || len(x.Cases[1].Polar) < 200 {
		t.Fatal("polar sector not generated")
	}
	if x.Cases[1].Polar[0].Mu != 0 {
		t.Fatal("apex must have no drift")
	}
}
func TestSnowV030WorkbookSeparateSheets(t *testing.T) {
	s := sampleSnow("skylight")
	s.JanuarySet = true
	s.JanuaryTemp = -12
	s, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	dest := filepath.Join(t.TempDir(), "snow.xlsx")
	if e = ExportSnowExcel(s, dest); e != nil {
		t.Fatal(e)
	}
	z, e := zip.OpenReader(dest)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	imgs := 0
	mainHasDrawing := false
	other := 0
	audit := false
	for _, f := range z.File {
		if strings.HasSuffix(f.Name, ".png") {
			imgs++
		}
		if f.Name == "xl/worksheets/sheet1.xml" {
			r, _ := f.Open()
			data, _ := io.ReadAll(r)
			r.Close()
			mainHasDrawing = strings.Contains(string(data), `<drawing `)
		}
		if strings.HasPrefix(f.Name, "xl/worksheets/") && strings.HasSuffix(f.Name, ".xml") && !strings.Contains(f.Name, "/_rels/") {
			other++
		}
		if f.Name == "xl/workbook.xml" {
			r, _ := f.Open()
			data, _ := io.ReadAll(r)
			r.Close()
			audit = strings.Contains(string(data), "Коэффициенты")
		}
	}
	if imgs != len(s.Cases)+1 || mainHasDrawing || other < len(s.Cases)+3 || !audit {
		t.Fatalf("images %d, drawing in main %v, sheets %d, audit %v", imgs, mainHasDrawing, other, audit)
	}
}
