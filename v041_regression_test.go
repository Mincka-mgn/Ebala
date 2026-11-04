package main

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test041ExcelRowOrderAndNoDuplicates(t *testing.T) {
	unsorted := tempExcelRow(1, "header") + tempExcelRow(9, "9") + tempExcelRow(3, "3")
	ordered, err := orderExcelSheetRows(unsorted)
	if err != nil || strings.Index(ordered, `r="3"`) > strings.Index(ordered, `r="9"`) {
		t.Fatalf("order: %v %s", err, ordered)
	}
	if _, err = orderExcelSheetRows(tempExcelRow(2, "a") + tempExcelRow(2, "b")); err == nil {
		t.Fatal("duplicate row accepted")
	}
}
func Test041BlocksCanRepeatMoveDeleteAndPersist(t *testing.T) {
	p := initialProject()
	a := p.addBlock(SavedReportBlock{Kind: "snow", Title: "Двускатная крыша"})
	b := p.addBlock(SavedReportBlock{Kind: "snow", Title: "Свод"})
	c := p.addBlock(SavedReportBlock{Kind: "ice", Title: "Трос"})
	if !p.moveBlock(b, -1) || p.SavedBlocks[0].ID != b {
		t.Fatal("cannot reorder snow blocks")
	}
	if p.moveBlock(c, -1) {
		t.Fatal("moved across groups")
	}
	if !p.removeBlock(a) || len(p.SavedBlocks) != 2 {
		t.Fatal("cannot remove")
	}
	f := filepath.Join(t.TempDir(), "project.json")
	if err := p.Save(f); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadProject(f)
	if err != nil || len(loaded.SavedBlocks) != 2 || loaded.SavedBlocks[0].Title != "Свод" {
		t.Fatalf("project did not persist: %v %+v", err, loaded.SavedBlocks)
	}
}
func Test041MultipleSnowAndIceExports(t *testing.T) {
	if os.Getenv("EBALA_QA_OUT") == "" {
		t.Skip("run via EBALA_QA_OUT")
	}
	snow := defaultSnowCalc()
	snow.Shape = "gable"
	snow.Span = 18
	snow.Ground = 2
	snow.Angle = 30
	snow.JanuarySet = true
	snow.JanuaryTemp = -10
	x, err := SnowCompute(snow)
	if err != nil {
		t.Fatal(err)
	}
	ice, err := IceCompute(IceLoadCalc{Region: "III", BaseMM: 10, Shape: "round", DiameterMM: 20, HeightStart: 10, HeightEnd: 70, HeightStep: 10, HighZone: "ordinary"})
	if err != nil {
		t.Fatal(err)
	}
	r := TempReport{Version: 1, Snows: []SnowCalc{x, x}, Ices: []IceLoadCalc{ice, ice}}
	dir := os.Getenv("EBALA_QA_OUT")
	if err = os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"docx", "xlsx"} {
		path := filepath.Join(dir, "Ebala_041_multi."+kind)
		if kind == "docx" {
			err = exportTempWord(r, path)
		} else {
			err = exportTempExcel(r, path)
		}
		if err != nil {
			t.Fatal(err)
		}
		z, e := zip.OpenReader(path)
		if e != nil {
			t.Fatal(e)
		}
		for _, f := range z.File {
			if !(strings.HasSuffix(f.Name, ".xml") || strings.HasSuffix(f.Name, ".rels")) {
				continue
			}
			rc, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			b, e := io.ReadAll(rc)
			rc.Close()
			if e != nil {
				t.Fatal(e)
			}
			var data interface{}
			if e = xml.Unmarshal(b, &data); e != nil {
				t.Fatalf("invalid XML in %s: %v", f.Name, e)
			}
		}
		z.Close()
	}
}

func Test041Mu2DoesNotRequireNarrativeSource(t *testing.T) {
	s := IceLoadCalc{Region: "III", BaseMM: 10, Shape: "surface_study", Mu2: 0.6, HeightStart: 5, HeightEnd: 50, HeightStep: 5, HighZone: "ordinary"}
	if _, err := IceCompute(s); err != nil {
		t.Fatalf("user-specified mu2 is a valid calculation input without a separate source description: %v", err)
	}
}
