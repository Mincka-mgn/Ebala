package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSP20Guide(t *testing.T) {
	g, e := loadSP20Guide("Catalog/sp20_guide.json")
	if e != nil {
		t.Fatal(e)
	}
	foundOffice, foundCorridor := false, false
	for _, r := range g.Records {
		if strings.Contains(r.Title, "Офисы, кабинеты") {
			foundOffice = true
			if r.Norm != 2 || r.Design != 2.4 {
				t.Fatalf("office: %+v", r)
			}
		}
		if strings.Contains(strings.ToLower(r.Title), "коридоры") && r.Category == "Офисы и учёба" {
			foundCorridor = true
		}
	}
	if !foundOffice || !foundCorridor {
		t.Fatal("expected office and its corridor grouped")
	}
	dir := t.TempDir()
	sel := sp20Selection(g.Records[0])
	p := filepath.Join(dir, "report.docx")
	if e := exportTempWord(TempReport{Version: 1, Sections: []TempSelection{sel}}, p); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(p)
	if e != nil || len(b) < 1000 {
		t.Fatal("Word export failed")
	}
}
