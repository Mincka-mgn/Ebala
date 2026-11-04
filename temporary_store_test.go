package main

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemporaryCatalogAndReports(t *testing.T) {
	c, e := loadTempCatalog("Catalog/temporary_loads.json")
	if e != nil {
		t.Fatal(e)
	}
	if len(c.Records) < 70 {
		t.Fatalf("expected full catalog, got %d", len(c.Records))
	}
	groups := map[string]bool{}
	for _, r := range c.Records {
		groups[r.Group] = true
	}
	for _, g := range []string{"СП 20 · временные", "СП 35 · транспорт"} {
		if !groups[g] {
			t.Fatalf("missing group %s", g)
		}
	}
	a, e := manualCrane("Кран №1", 4, "120; 130; 130; 120", "0; 2000; 7000; 9000")
	if e != nil {
		t.Fatal(e)
	}
	if len(a.Wheels) != 4 || a.Wheels[2].XMM != 7000 || !a.Wheels[2].XKnown {
		t.Fatalf("wheel parsing: %+v", a.Wheels)
	}
	missing, e := manualCrane("без координат", 2, "10 20", "")
	if e != nil || missing.Wheels[0].XKnown {
		t.Fatalf("unspecified geometry must stay unspecified: %+v %v", missing, e)
	}
	if _, e = manualCrane("invalid", 4, "1 2 3 4", "0 200 100 300"); e == nil {
		t.Fatal("coordinate order must be validated")
	}
	if _, e = manualCrane("invalid", 4, "1 2 3", "0 200 300 400"); e == nil {
		t.Fatal("wheel count must be validated")
	}
	dir := t.TempDir()
	report := TempReport{Version: 1, Project: "Проверка", Sections: []TempSelection{tempSectionFromEntry(c.Records[0]), a}}
	p := filepath.Join(dir, "test.json")
	if e = saveTempReport(p, report); e != nil {
		t.Fatal(e)
	}
	got, e := readTempReport(p)
	if e != nil || len(got.Sections) != 2 {
		t.Fatalf("roundtrip: %+v %v", got, e)
	}
	doc := filepath.Join(dir, "test.docx")
	if e = exportTempWord(got, doc); e != nil {
		t.Fatal(e)
	}
	z, e := zip.OpenReader(doc)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	found := false
	for _, f := range z.File {
		if f.Name != "word/document.xml" {
			continue
		}
		r, _ := f.Open()
		buf, _ := io.ReadAll(r)
		r.Close()
		found = true
		dec := xml.NewDecoder(strings.NewReader(string(buf)))
		for {
			_, e := dec.Token()
			if e == io.EOF {
				break
			}
			if e != nil {
				t.Fatalf("Word XML error: %v", e)
			}
		}
		if !strings.Contains(string(buf), "Колёсная схема") {
			t.Fatal("missing wheels")
		}
		if strings.Contains(string(buf), "ИТОГО") {
			t.Fatal("unexpected summation")
		}
	}
	if !found {
		t.Fatal("missing document")
	}
	if _, e = os.Stat(doc); e != nil {
		t.Fatal(e)
	}
}
