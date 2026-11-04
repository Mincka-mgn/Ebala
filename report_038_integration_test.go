package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test038CombinedIceAndSnowExports(t *testing.T) {
	snow := defaultSnowCalc()
	snow.Shape = "gable"
	snow.Span = 18
	snow.Ground = 2
	snow.Angle = 30
	snow.JanuarySet = true
	snow.JanuaryTemp = -10
	calculatedSnow, e := SnowCompute(snow)
	if e != nil {
		t.Fatal(e)
	}
	ice, e := IceCompute(IceLoadCalc{Region: "III", BaseMM: 10, Shape: "round", DiameterMM: 20, HeightStart: 10, HeightEnd: 100, HeightStep: 20, HighZone: "ordinary"})
	if e != nil {
		t.Fatal(e)
	}
	doc := TempReport{Version: 1, Project: "Контрольный объект", Snow: &calculatedSnow, Ice: &ice}
	dir := t.TempDir()
	if e = exportTempExcel(doc, filepath.Join(dir, "общий_расчёт.xlsx")); e != nil {
		t.Fatal(e)
	}
	if e = exportTempWord(doc, filepath.Join(dir, "общий_расчёт.docx")); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"общий_расчёт.xlsx", "общий_расчёт.docx"} {
		r, e := zip.OpenReader(filepath.Join(dir, name))
		if e != nil {
			t.Fatal(e)
		}
		for _, file := range r.File {
			if _, e = file.Open(); e != nil {
				t.Fatalf("%s/%s: %v", name, file.Name, e)
			}
		}
		r.Close()
	}
	if os.Getenv("EBALA_QA_OUT") != "" {
		for _, name := range []string{"общий_расчёт.xlsx", "общий_расчёт.docx"} {
			content, e := os.ReadFile(filepath.Join(dir, name))
			if e != nil {
				t.Fatal(e)
			}
			e = os.WriteFile(filepath.Join(os.Getenv("EBALA_QA_OUT"), strings.ReplaceAll(name, "общий_расчёт", "ebala_038_combined")), content, 0644)
			if e != nil {
				t.Fatal(e)
			}
		}
	}
}
