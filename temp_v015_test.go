package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTransportV015(t *testing.T) {
	c, e := loadTempCatalog(filepath.Join("Catalog", "temporary_loads.json"))
	if e != nil {
		t.Fatal(e)
	}
	accepted := map[string]int{}
	for _, e := range c.Records {
		if e.Group != "СП 35 · транспорт" {
			continue
		}
		s := tempSectionFromEntry(e)
		kind := transportKind(s)
		if kind == "" {
			continue
		}
		accepted[kind]++
		d := transportDetail(s)
		if len(d.Rows) < 4 {
			t.Fatalf("empty card: %s", s.Title)
		}
		if kind != "SK" && kind != "AB" && len(d.Figures) == 0 {
			t.Fatalf("figure missing for %s", s.Title)
		}
	}
	for _, k := range []string{"AK", "NK", "AB", "SK", "Metro", "Tram"} {
		if accepted[k] < 1 {
			t.Fatalf("no %s", k)
		}
	}
	s := TempSelection{Group: "СП 35 · транспорт", Title: "Поезд метрополитена", Params: map[string]interface{}{"loaded_wagon_kN": 588.0, "empty_wagon_kN": 294.0, "load_length_m": 100.0}}
	if !strings.Contains(transportDetailedText(s, false), "1,170") {
		t.Fatal("wrong metro gamma")
	}
	if g := transportGammaSK(100); g != 1.125 {
		t.Fatal(g)
	}
	if g := transportGammaSK(200); g != 1.1 {
		t.Fatal(g)
	}
}
func TestTransportReportsV015(t *testing.T) {
	c, e := loadTempCatalog(filepath.Join("Catalog", "temporary_loads.json"))
	if e != nil {
		t.Fatal(e)
	}
	var r TempReport
	r.Version = 1
	seen := map[string]bool{}
	for _, e := range c.Records {
		if e.Group == "СП 35 · транспорт" {
			s := tempSectionFromEntry(e)
			k := transportKind(s)
			if !seen[k] && k != "" {
				seen[k] = true
				r.Sections = append(r.Sections, s)
			}
		}
	}
	dir := t.TempDir()
	doc := filepath.Join(dir, "transport.docx")
	xls := filepath.Join(dir, "transport.xlsx")
	if e := exportTempWord(r, doc); e != nil {
		t.Fatal(e)
	}
	if e := exportTempExcel(r, xls); e != nil {
		t.Fatal(e)
	}
	for _, p := range []string{doc, xls} {
		z, e := zip.OpenReader(p)
		if e != nil {
			t.Fatal(e)
		}
		figures := 0
		for _, f := range z.File {
			if strings.Contains(f.Name, "/media/") {
				figures++
			}
		}
		z.Close()
		if p == doc && figures < 4 {
			t.Fatalf("expected four Word figures in %s, got %d", p, figures)
		}
		if p == xls {
			parts := excel051Parts(t, p)
			if !strings.Contains(parts["xl/workbook.xml"], "Транспорт - исходные") || !strings.Contains(parts["xl/workbook.xml"], "Транспорт - расчёт") {
				t.Fatal("Excel missing independent transport formula sheets")
			}
		}
		if s, _ := os.Stat(p); s.Size() < 3000 {
			t.Fatal("undersized", p)
		}
	}
}
