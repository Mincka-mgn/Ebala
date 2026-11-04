package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCranePlacementReports021(t *testing.T) {
	r, err := CalculateCranePlacements([]float64{12000, 12000}, []CraneTrain{{Name: "Кран 1", WheelXMM: []float64{0, 4400}, ForcesKN: []float64{60, 60}, OverallMM: 4400}})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Cases) == 0 {
		t.Fatal("no cases")
	}
	dir := t.TempDir()
	wd := filepath.Join(dir, "c.docx")
	xl := filepath.Join(dir, "c.xlsx")
	if err := exportCranePlacementWord(r, wd); err != nil {
		t.Fatal("word:", err)
	}
	if err := exportCranePlacementExcel(r, xl); err != nil {
		t.Fatal("excel:", err)
	}
	for _, p := range []string{wd, xl} {
		s, err := os.Stat(p)
		if err != nil || s.Size() < 2000 {
			t.Fatalf("empty %s: %v", p, err)
		}
		z, err := zip.OpenReader(p)
		if err != nil {
			t.Fatal(err)
		}
		formula := false
		drawing := false
		for _, f := range z.File {
			if strings.HasPrefix(f.Name, "word/media/position_") {
				drawing = true
			}
			if strings.Contains(f.Name, "sheet2") {
				rd, err := f.Open()
				if err != nil {
					t.Fatal(err)
				}
				buf := make([]byte, 2000000)
				n, _ := rd.Read(buf)
				rd.Close()
				formula = strings.Contains(string(buf[:n]), "<f>D")
			}
		}
		z.Close()
		if strings.HasSuffix(p, "xlsx") && !formula {
			t.Fatal("no formulas")
		}
		if strings.HasSuffix(p, "docx") && !drawing {
			t.Fatal("no diagrams")
		}
	}

	t.Log(dir)
}
