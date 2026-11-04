package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func Test010UserTextPreservedButNoHeightUplift(t *testing.T) {
	s := Site{Region: "Краснодарский край", Place: "Дефановка", SnowZone: "II", SnowGround: 1, MountainID: "krasnodar_other"}
	for _, raw := range []string{"", "Сиськи", "-120", "было 700 м", "10000"} {
		s.SetElevationText(raw)
		if s.ValidElevationForCalc() {
			t.Fatalf("accepted %q", raw)
		}
		if s.ElevationM != 0 || s.EffectiveSnow() != 1 {
			t.Fatalf("wrong calculation for %q: %#v", raw, s)
		}
		if raw != "" && s.ElevationWarning() == "" {
			t.Fatalf("missing warning for %q", raw)
		}
		rows, err := siteReportRows(s)
		if err != nil {
			t.Fatalf("rows failed for %q: %v", raw, err)
		}
		hasRaw := false
		for _, r := range rows {
			if r.Parameter == "Абсолютная отметка h, м" && r.Value == raw {
				hasRaw = true
			}
		}
		if hasRaw != (raw != "") {
			t.Fatalf("raw not faithfully reported: %q %#v", raw, rows)
		}
		doc := filepath.Join(t.TempDir(), "site.docx")
		xls := filepath.Join(t.TempDir(), "site.xlsx")
		if err := ExportSiteWord(s, doc); err != nil {
			t.Fatal(err)
		}
		if err := ExportSiteExcel(s, xls); err != nil {
			t.Fatal(err)
		}
		if raw != "" && (!strings.Contains(zipPart(t, doc, "word/document.xml"), raw) || !strings.Contains(zipPart(t, xls, "xl/worksheets/sheet1.xml"), raw)) {
			t.Fatalf("raw lost in file %q", raw)
		}
		s.MountainID = ""
		if err := ExportSiteWord(s, doc); err != nil {
			t.Fatal(err)
		}
		if raw != "" && !strings.Contains(zipPart(t, doc, "word/document.xml"), raw) {
			t.Fatalf("ordinary report must retain typed elevation: %q", raw)
		}
		s.MountainID = "krasnodar_other"
	}
	s.SetElevationText("800")
	if !s.ValidElevationForCalc() || s.EffectiveSnow() != 2.5 {
		t.Fatalf("valid h not processed %+v, sg=%.3f", s, s.EffectiveSnow())
	}
	s.SetElevationText("69")
	if s.EffectiveSnow() != 1 {
		t.Fatal("below threshold increased snow")
	}
	if len(s.ElevationText()) != 2 {
		t.Fatal("raw changed")
	}
}

func Test010WikipediaDirectFallbackForMaykopTikhoretsk(t *testing.T) {
	h := map[string]struct{ region, height string }{
		"Майкоп":   {"Республика Адыгея", "220"},
		"Тихорецк": {"Краснодарский край", "80"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("action") {
		case "wbsearchentities":
			fmt.Fprintf(w, `{"search":[{"id":"Q5","label":%q,"description":"город в России"}]}`, r.URL.Query().Get("search"))
		case "parse":
			city := r.URL.Query().Get("page")
			v, ok := h[city]
			if !ok {
				fmt.Fprint(w, `{"error":{"code":"missingtitle"}}`)
				return
			}
			fmt.Fprintf(w, `{"parse":{"wikitext":%q}}`, "{{НП России\n|Регион = "+v.region+"\n|Высота центра НП = "+v.height+" м\n}}")
		default:
			fmt.Fprint(w, `{"error":{"code":"not-found"}}`)
		}
	}))
	defer server.Close()
	for city, v := range h {
		res, err := fetchWikiElevation(context.Background(), server.Client(), server.URL, city, v.region, "")
		if err != nil {
			t.Fatalf("lookup %s: %v", city, err)
		}
		if res.Source == "" || !strings.Contains(res.Source, "Википедия") || (city == "Майкоп" && res.Height != 220) || (city == "Тихорецк" && res.Height != 80) {
			t.Fatalf("lookup %s: %+v", city, res)
		}
	}
	if _, err := fetchWikiElevation(context.Background(), server.Client(), server.URL, "Майкоп", "Красноярский край", ""); err != errWikiNotFound {
		t.Fatalf("matched wrong region %v", err)
	}
}
