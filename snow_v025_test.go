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

func TestSnowV025ExposureAndThermal(t *testing.T) {
	s := sampleSnow("gable")
	s.Angle = 8
	s.RoofHeight = 10
	s.Width = 24
	s.Span = 18
	s.Terrain = "А"
	s.Exposed = true
	s.JanuarySet = true
	s.JanuaryTemp = -10
	s.WinterWindSet = true
	s.WinterWind = 5
	out, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	// LC=22.5, k(A,10m)=1, kv(A,-10C,5m/s)=1.3
	want := (1.3 - .4) * (.8 + .002*22.5)
	if math.Abs(out.CE-want) > 1e-9 {
		t.Fatalf("ce=%f expected=%f", out.CE, want)
	}
	s.JanuaryTemp = -4
	out, e = SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if out.CE != 1 || out.ReducedApplicable() {
		t.Fatal("warm January: reduction is prohibited")
	}
	s.JanuarySet = false
	out, e = SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if out.CE != 1 || out.ReducedApplicable() {
		t.Fatal("unknown temperature must not reduce")
	}
	s.JanuarySet = true
	s.JanuaryTemp = -10
	s.Terrain = "С"
	out, e = SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if out.CE != 1 {
		t.Fatal("urban C: reduction is prohibited")
	}
	s.Terrain = "А"
	s.WarmRoof = true
	s.Angle = 10
	out, e = SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if out.CT != .8 {
		t.Fatal("thermal roof should have ct=.8")
	}
	s.Shape = "flat"
	out, e = SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if out.CT != 1 {
		t.Fatal("flat roof must not have thermal reduction")
	}
	s.Shape = "step"
	s.H = 3
	out, e = SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if out.CE != 1 {
		t.Fatal("roof step: do not apply global wind reduction")
	}
}
func TestSnowV025LanternAndExports(t *testing.T) {
	s := sampleSnow("skylight")
	s.SkylightWidth = 6
	s.H = 2
	s.Span = 18
	s.JanuarySet = true
	s.JanuaryTemp = -10
	s.RoofHeight = 8
	s, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(s.Cases) != 6 {
		t.Fatalf("expected 4 B.3 plus 2 partial variants, got %d", len(s.Cases))
	}
	if math.Abs(s.MaxMu(s.Cases[2])-2.5) > 1e-9 {
		t.Fatalf("B.3 lantern coefficient %.3f", s.MaxMu(s.Cases[2]))
	}
	dir := t.TempDir()
	excel := filepath.Join(dir, "snow.xlsx")
	word := filepath.Join(dir, "snow.docx")
	if e = ExportSnowExcel(s, excel); e != nil {
		t.Fatal(e)
	}
	if e = ExportSnowWord(s, word); e != nil {
		t.Fatal(e)
	}
	for _, file := range []string{excel, word} {
		z, e := zip.OpenReader(file)
		if e != nil {
			t.Fatal(e)
		}
		n := 0
		for _, f := range z.File {
			if strings.Contains(f.Name, "/snow_") && strings.HasSuffix(f.Name, ".png") {
				n++
			}
		}
		z.Close()
		expected := len(s.Cases)+1
        if file == word { expected += 3 } // two joint-zone B sections and the approved plan
        if n != expected {
            t.Fatalf("%s images %d expected %d", file, n,expected)
        }
	}
	z, e := zip.OpenReader(excel)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	for _, f := range z.File {
		if strings.HasSuffix(f.Name, "sheet1.xml") {
			r, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			b, e := io.ReadAll(r)
			r.Close()
			if e != nil {
				t.Fatal(e)
			}
			text := string(b)
			// The ct and gamma cells are the SECOND metadata row, despite the extra new metadata row.
			if !strings.Contains(text, "$B$4") || !strings.Contains(text, "$D$4") {
				t.Fatal("wrong formula references after metadata row inserted")
			}
		}
	}
	_ = os.Stdout
}
