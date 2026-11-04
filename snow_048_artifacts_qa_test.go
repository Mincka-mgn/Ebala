package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Set EBALA_QA_OUT to produce independently inspectable reports for this release.
func Test048QAExport(t *testing.T) {
	folder := os.Getenv("EBALA_QA_OUT")
	if folder == "" {
		t.Skip("set EBALA_QA_OUT to export verification files")
	}
	if e := os.MkdirAll(folder, 0755); e != nil {
		t.Fatal(e)
	}
	cases := map[string]SnowCalc{
		"sphere_cone": snow048SphereConeSample(),
		"lantern":     sampleSnow("skylight"),
		"parapet":     sampleSnow("parapet"),
		"dome_steep":  sampleSnow("dome"),
	}
	d := cases["dome_steep"]
	d.Span, d.Rise = 18, 14
	cases["dome_steep"] = d
	for name, raw := range cases {
		computed, e := SnowCompute(raw)
		if e != nil {
			t.Fatalf("%s compute: %v", name, e)
		}
		if e = ExportSnowWord(computed, filepath.Join(folder, name+".docx")); e != nil {
			t.Fatalf("%s DOCX: %v", name, e)
		}
		if e = ExportSnowExcel(computed, filepath.Join(folder, name+".xlsx")); e != nil {
			t.Fatalf("%s XLSX: %v", name, e)
		}
	}
}
