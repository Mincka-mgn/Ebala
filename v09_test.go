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

func Test09MountainAndReportRules(t *testing.T) {
	s := climateFromRecord(pickCity(t, "Казань"))
	rows, e := siteReportRows(s)
	if e != nil || rows[0].Zone != snowOptionK || !strings.Contains(rows[0].Source, "приложение К") || strings.Contains(rows[0].Source, "район IV") {
		t.Fatalf("app K rows=%+v err=%v", rows, e)
	}
	s = Site{Region: "Краснодарский край", Place: "Дефановка", SnowZone: "II", SnowGround: 1, ElevationKnown: true, ElevationM: 69, ElevationSource: "Википедия / Wikidata"}
	if s.EffectiveSnow() != 1 {
		t.Fatal("height not selected but applied")
	}
	if len(reportHeader(s)) != 2 {
		t.Fatal("header")
	}
	doc := filepath.Join(t.TempDir(), "height_off.docx")
	if e := ExportSiteWord(s, doc); e != nil {
		t.Fatal(e)
	}
	body := zipPart(t, doc, "word/document.xml")
	if !strings.Contains(body, "69") || !strings.Contains(body, "Абсолютная отметка") {
		t.Fatal("entered elevation must appear even without mountain correction")
	}
	s.MountainID = "krasnodar_other"
	base, value, area, applied, e := s.ElevationDetails()
	if e != nil || !applied || area.H0 != 500 || value != base {
		t.Fatalf("below threshold: %.2f %.2f %+v %v", base, value, area, e)
	}
	if e := ExportSiteWord(s, doc); e != nil {
		t.Fatal(e)
	}
	body = zipPart(t, doc, "word/document.xml")
	if !strings.Contains(body, "69 м") && !strings.Contains(body, "69") {
		t.Fatal("manual mountain selection hidden")
	}
	s.MountainID = ""
	s.SnowZone = snowOptionMountain
	s.SnowGround = 0
	s.SnowNote = "СП 20.13330.2016, карта 1 приложения Е: горные и малоизученные районы"
	rows, e = siteReportRows(s)
	if e != nil || len(rows) != 2 || rows[0].Zone != snowOptionMountain || rows[0].Value != "" {
		t.Fatalf("special area row %+v / %v", rows, e)
	}
	if specialSnowSite(s) {
		t.Fatal("mountain wrongly detected as appendix K")
	}
}
func Test09WikidataFetchAndWikipediaFallback(t *testing.T) {
	serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := r.URL.Query().Get("action")
		w.Header().Set("Content-Type", "application/json")
		if a == "wbsearchentities" {
			fmt.Fprint(w, `{"search":[{"id":"Q5","label":"Дефановка","description":"село в Туапсинском районе Краснодарского края"},{"id":"Q6","label":"Дефановка","description":"село в другой области"}]}`)
			return
		}
		if a == "wbgetentities" {
			fmt.Fprint(w, `{"entities":{"Q5":{"claims":{"P2044":[{"rank":"normal","mainsnak":{"datavalue":{"value":{"amount":"+69","unit":"http://www.wikidata.org/entity/Q11573"}}}}]},"sitelinks":{"ruwiki":{"title":"Дефановка"}}}}}`)
			return
		}
		http.Error(w, "bad", 400)
	}))
	defer serv.Close()
	v, e := fetchWikiElevation(context.Background(), serv.Client(), serv.URL, "Дефановка", "Краснодарский край", "Туапсинский район")
	if e != nil || v.Height != 69 || v.Entity != "Q5" {
		t.Fatalf("v=%+v err=%v", v, e)
	}
	if _, e = fetchWikiElevation(context.Background(), serv.Client(), serv.URL, "Дефановка", "Красноярский край", ""); e == nil {
		t.Fatal("namesake silently matched")
	}
	if h, ok := wikiInfoboxHeight("{{НП России\n|Высота центра НП = 69 м\n}}"); !ok || h != 69 {
		t.Fatal("wiki infobox fallback")
	}
}
