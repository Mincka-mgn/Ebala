package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateCraneSample023(t *testing.T) {
	if os.Getenv("EBALA_SAMPLE_OUT") == "" {
		return
	}
	r, e := CalculateCranePlacements([]float64{12000, 9000}, []CraneTrain{{Name: "Кран контрольный", WheelXMM: []float64{0, 4400}, ForcesKN: []float64{60, 60}, OverallMM: 5000}})
	if e != nil {
		t.Fatal(e)
	}
	p := os.Getenv("EBALA_SAMPLE_OUT")
	if e = exportCranePlacementExcel(r, p); e != nil {
		t.Fatal(e)
	}
	t.Log("created ", filepath.Base(p), "cases", len(r.Cases))
}
