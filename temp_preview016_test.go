package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuild016Preview(t *testing.T) {
	out := os.Getenv("EBALA_PREVIEW_DIR")
	if out == "" {
		t.Skip("preview output directory is not requested")
	}
	c, e := loadTempCatalog(filepath.Join("Catalog", "temporary_loads.json"))
	if e != nil {
		t.Fatal(e)
	}
	selected := map[string]bool{}
	r := TempReport{Version: 1, Project: "Проверка оформления транспортных нагрузок"}
	for _, entry := range c.Records {
		if entry.Group != "СП 35 · транспорт" {
			continue
		}
		s := tempSectionFromEntry(entry)
		kind := transportKind(s)
		if kind == "" || selected[kind] {
			continue
		}
		selected[kind] = true
		r.Sections = append(r.Sections, s)
	}
	if len(r.Sections) != 6 {
		t.Fatalf("six scheme types expected, got %d", len(r.Sections))
	}
	if e := exportTempWord(r, filepath.Join(out, "preview016.docx")); e != nil {
		t.Fatal(e)
	}
	if e := exportTempExcel(r, filepath.Join(out, "preview016.xlsx")); e != nil {
		t.Fatal(e)
	}
}
