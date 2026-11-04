package main

import (
	"archive/zip"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleSnow(shape string) SnowCalc {
	s := defaultSnowCalc()
	s.Shape = shape
	s.Ground = 1.0
	s.Region = "II"
	return s
}
func TestSnowBasicAndVariation(t *testing.T) {
	s, e := SnowCompute(sampleSnow("gable"))
	if e != nil {
		t.Fatal(e)
	}
	if len(s.Cases) != 2 || math.Abs(s.Cases[0].Intervals[0].Mu-1) > 1e-8 || !strings.Contains(s.Cases[1].Title, "Проверочная") {
		t.Fatalf("base and optional uneven loading missing %#v", s.Cases)
	}
	s.Angle = 20
	s.RidgeEquipment = true
	s, e = SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(s.Cases) < 4 {
		t.Errorf("missing snow variation, got %d", len(s.Cases))
	}
	if math.Abs(s.QDesign(1)-1.4) > 1e-10 {
		t.Fatal("design snow should be 1.4 kPa")
	}
	s.Angle = 45
	s.RidgeEquipment = false
	s, e = SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if math.Abs(s.MaxMu(s.Cases[0])-.5) > 1e-8 {
		t.Fatal("slope interpolation failed")
	}
}
func TestSnowStepAndVault(t *testing.T) {
	s := sampleSnow("wall")
	s.H = .25
	ss, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(ss.Cases) != 1 {
		t.Fatalf("drift not applicable at low h: %d", len(ss.Cases))
	}
	s.H = 2
	s.LowerShape = "gable"
	s.Angle = 20
	ss, e = SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(ss.Cases) < 3 || !strings.Contains(ss.Cases[len(ss.Cases)-1].Title, "перепада") {
		t.Fatalf("missing bag: %v", ss.Cases)
	}
	s = sampleSnow("vault")
	ss, e = SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(ss.Cases) != 3 || len(ss.Cases[0].Intervals) != 64 {
		t.Fatalf("vault: %v", ss.Cases)
	}
}
func TestSnowReservoirLimits(t *testing.T) {
	s := sampleSnow("cone")
	s.Angle = 6
	if _, e := SnowCompute(s); e != nil {
		t.Fatal(e)
	}
	s.Angle = 9
	if got, e := SnowCompute(s); e != nil || len(got.Cases) < 2 || len(got.Cases[1].Polar) == 0 {
		t.Fatalf("cone: %#v %v", got.Cases, e)
	}
	s = sampleSnow("dome")
	s.Rise = .5
	s.Span = 20
	if _, e := SnowCompute(s); e != nil {
		t.Fatal(e)
	}
	s.Rise = 2
	if got, e := SnowCompute(s); e != nil || len(got.Cases) < 2 || len(got.Cases[1].Polar) == 0 {
		t.Fatalf("dome spatial variants: %d %v", len(got.Cases), e)
	}
	s.DomeObstacles = true
	if got, e := SnowCompute(s); e != nil || len(got.Cases) < 3 || len(got.Cases[2].Polar) == 0 {
		t.Fatalf("dome obstacle variant: %d %v", len(got.Cases), e)
	}
}
func TestSnowExports(t *testing.T) {
	dir := t.TempDir()
	s := sampleSnow("gable")
	s.Angle = 20
	var e error
	s, e = SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	for _, ext := range []string{"docx", "xlsx"} {
		path := filepath.Join(dir, "snow."+ext)
		if ext == "docx" {
			e = ExportSnowWord(s, path)
		} else {
			e = ExportSnowExcel(s, path)
		}
		if e != nil {
			t.Fatal(e)
		}
		r, e := zip.OpenReader(path)
		if e != nil {
			t.Fatal(e)
		}
		images, charts, found := 0, 0, false
		for _, f := range r.File {
			if strings.HasSuffix(f.Name, ".png") {
				images++
			}
			if strings.HasPrefix(f.Name, "xl/charts/chart") && strings.HasSuffix(f.Name, ".xml") {
				charts++
			}
			if strings.HasSuffix(f.Name, "document.xml") || strings.HasSuffix(f.Name, "sheet1.xml") {
				handle, er := f.Open()
				if er != nil {
					t.Fatal(er)
				}
				b := make([]byte, f.UncompressedSize64)
				_, _ = handle.Read(b)
				handle.Close()
				if strings.Contains(string(b), "Снеговые нагрузки") || strings.Contains(string(b), "СНЕГОВЫЕ НАГРУЗКИ") {
					found = true
				}
			}
		}
		r.Close()
		// Word carries static epures; Excel contains only geometry PNG and formula-driven charts.
		if !found || (ext == "docx" && images != len(s.Cases)+1) || (ext == "xlsx" && (images != 1 || charts != len(snowExcelVariants(s)))) {
			t.Fatalf("%s: images=%d, cases=%d, found=%v", ext, images, len(s.Cases), found)
		}
		fi, _ := os.Stat(path)
		if fi.Size() < 3000 {
			t.Fatalf("report too small: %s", path)
		}
	}
}
