package main

import (
	"os"
	"path/filepath"
	"testing"
)

func Test049QAExport(t *testing.T) {
	folder := os.Getenv("EBALA_QA_OUT")
	if folder == "" {
		t.Skip("set EBALA_QA_OUT to export 0.49 QA artifacts")
	}
	if err := os.MkdirAll(folder, 0755); err != nil {
		t.Fatal(err)
	}
	cases := map[string]SnowCalc{}
	s := sample049()
	cases["sphere_cone_15deg"] = s
	s.BetaStep = 7
	cases["sphere_cone_7deg"] = s
	s = sample049()
	s.RoofHeight = 5
	s.H = 9
	cases["sphere_cone_underground"] = s
	m := defaultSnowCalc()
	m.Shape = "multi_gable"
	m.Count = 2
	m.Angle = 18
	m.SlopeMode = "angle"
	cases["multi_gable"] = m
	for name, raw := range cases {
		computed, err := SnowCompute(raw)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := ExportSnowWord(computed, filepath.Join(folder, name+".docx")); err != nil {
			t.Fatalf("DOCX %s: %v", name, err)
		}
		if err := ExportSnowExcel(computed, filepath.Join(folder, name+".xlsx")); err != nil {
			t.Fatalf("XLSX %s: %v", name, err)
		}
	}
}
