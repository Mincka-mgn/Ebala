package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func Test038IceArtifactFixtures(t *testing.T) {
	dir := os.Getenv("EBALA_QA_OUT")
	if dir == "" {
		t.Skip("set EBALA_QA_OUT")
	}
	if e := os.MkdirAll(dir, 0755); e != nil {
		t.Fatal(e)
	}
	for _, cfg := range []struct {
		name string
		s    IceLoadCalc
	}{
		{"ice_round", IceLoadCalc{Region: "II", BaseMM: 5, Shape: "round", DiameterMM: 10, HeightStart: 5, HeightEnd: 100, HeightStep: 10, HighZone: "ordinary"}},
		{"ice_surface", IceLoadCalc{Region: "III", BaseMM: 10, Shape: "surface_small", Mu2: .6, HeightStart: 10, HeightEnd: 80, HeightStep: 10, HighZone: "ordinary"}},
		{"ice_high", IceLoadCalc{Region: "III", BaseMM: 10, Shape: "round", DiameterMM: 20, HeightStart: 100, HeightEnd: 300, HeightStep: 25, HighZone: "ordinary", Horizontal: true}},
	} {
		out, e := IceCompute(cfg.s)
		if e != nil {
			t.Fatal(e)
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		if e = os.WriteFile(filepath.Join(dir, cfg.name+".json"), b, 0644); e != nil {
			t.Fatal(e)
		}
		if e = ExportIceWord(out, filepath.Join(dir, cfg.name+".docx")); e != nil {
			t.Fatal(e)
		}
		if e = ExportIceExcel(out, filepath.Join(dir, cfg.name+".xlsx")); e != nil {
			t.Fatal(e)
		}
	}
}
