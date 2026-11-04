package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test018ReportPickedCraneHasKindAndDistances(t *testing.T) {
	c, e := loadTempCatalog("Catalog/cranes_gost_handbook.json")
	if e != nil {
		t.Fatal(e)
	}
	var chosen *TempEntry
	for i := range c.Records {
		r := &c.Records[i]
		if strings.Contains(tempNumberText(r.Params["crane_kind"]), "двухпролётный") && len(r.Wheels) == 2 {
			chosen = r
			break
		}
	}
	if chosen == nil {
		t.Fatal("two-span crane not in catalogue")
	}
	s := tempSectionFromEntry(*chosen)
	if len(s.Wheels) < 2 || tempCraneKind(s) == "" || s.Wheels[1].XMM <= 0 {
		t.Fatal("crane geometry missing")
	}
	r := TempReport{Version: 1, Sections: []TempSelection{s}}
	dir := t.TempDir()
	doc := filepath.Join(dir, "test.docx")
	exc := filepath.Join(dir, "test.xlsx")
	if e := exportTempWord(r, doc); e != nil {
		t.Fatal(e)
	}
	if e := exportTempExcel(r, exc); e != nil {
		t.Fatal(e)
	}
	for _, p := range []string{doc, exc} {
		z, e := zip.OpenReader(p)
		if e != nil {
			t.Fatal(e)
		}
		foundKind := false
		foundTitle := false
		for _, f := range z.File {
			if !strings.HasSuffix(f.Name, ".xml") {
				continue
			}
			rd, e := f.Open()
			if e != nil {
				continue
			}
			data, _ := io.ReadAll(rd)
			rd.Close()
			if strings.Contains(string(data), "двухпролётный") {
				foundKind = true
			}
			if strings.Contains(string(data), "Межколёсные") || strings.Contains(string(data), "Расстояния") || strings.Contains(string(data), "Шаг до предыдущего") {
				foundTitle = true
			}
		}
		z.Close()
		if !foundKind {
			t.Fatalf("crane type missing in %s", p)
		}
		if !foundTitle {
			t.Fatalf("crane geometry column missing in %s", p)
		}
		if stat, e := os.Stat(p); e != nil || stat.Size() < 500 {
			t.Fatalf("report too small: %s, %v", p, e)
		}
	}
}
