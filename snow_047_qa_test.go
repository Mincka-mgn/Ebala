package main

import (
	"os"
	"path/filepath"
	"testing"
)

func Test047QAExport(t *testing.T) {
	dir := os.Getenv("EBALA_QA_OUT")
	if dir == "" {
		t.Skip("set EBALA_QA_OUT to write user-facing verification files")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"parapet", "multi_vault", "cone", "dome"} {
		s := sampleSnow(kind)
		s.Ground = 2
		s.Span = 18
		s.Width = 24
		s.RoofHeight = 12
		s.H = 2
		s.Count = 2
		s.Rise = 2
		res, err := SnowCompute(s)
		if err != nil {
			t.Fatalf("%s calculation: %v", kind, err)
		}
		doc := filepath.Join(dir, kind+".docx")
		xl := filepath.Join(dir, kind+".xlsx")
		if err := ExportSnowWord(res, doc); err != nil {
			t.Fatalf("%s docx: %v", kind, err)
		}
		if err := ExportSnowExcel(res, xl); err != nil {
			t.Fatalf("%s xlsx: %v", kind, err)
		}
	}
}
