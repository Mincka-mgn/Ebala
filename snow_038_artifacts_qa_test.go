package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Optional integration fixture: generate real Word and Excel files from the core
// without changing the application's normal output paths.
func Test038ArtifactFixtures(t *testing.T) {
	dir := os.Getenv("EBALA_QA_OUT")
	if dir == "" {
		t.Skip("set EBALA_QA_OUT to produce external review fixtures")
	}
	if e := os.MkdirAll(dir, 0755); e != nil {
		t.Fatal(e)
	}
	fixtures := []struct {
		name, shape string
		angle, rise float64
		count       int
	}{
		{"vault_partial", "vault", 0, 5, 2},
		{"multi_vault_partial", "multi_vault", 0, 5, 2},
		{"mono_negative", "mono", -45, 0, 2},
		{"multi_gable_partial", "multi_gable", 35, 0, 3},
	}
	for _, f := range fixtures {
		s := defaultSnowCalc()
		s.Shape = f.shape
		s.Span = 20
		s.Width = 24
		s.Angle = f.angle
		s.Rise = f.rise
		s.Ground = 2
		s.Count = f.count
		out, e := SnowCompute(s)
		if e != nil {
			t.Fatal(e)
		}
		ref, _ := json.MarshalIndent(out, "", "  ")
		if e = os.WriteFile(filepath.Join(dir, f.name+".json"), ref, 0644); e != nil {
			t.Fatal(e)
		}
		if e = ExportSnowExcel(out, filepath.Join(dir, f.name+".xlsx")); e != nil {
			t.Fatalf("%s Excel: %v", f.name, e)
		}
		if e = ExportSnowWord(out, filepath.Join(dir, f.name+".docx")); e != nil {
			t.Fatalf("%s Word: %v", f.name, e)
		}
	}
}
