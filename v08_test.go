package main

import (
	"archive/zip"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func zipPart(t *testing.T, path, part string) string {
	t.Helper()
	z, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	for _, f := range z.File {
		if f.Name == part {
			rc, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			b, e := io.ReadAll(rc)
			rc.Close()
			if e != nil {
				t.Fatal(e)
			}
			return string(b)
		}
	}
	t.Fatalf("missing part %s", part)
	return ""
}
func TestRegionOnlyRecordsWithoutCapital(t *testing.T) {
	all := climateRecords()
	regionCount := 0
	for _, v := range all {
		if v.IsRegion {
			regionCount++
			if v.Place != "" || v.District != "" || v.SnowZone != "" || v.WindZone != "" || v.IceZone != "" {
				t.Fatalf("region not empty: %+v", v)
			}
		}
	}
	if regionCount != 89 {
		t.Fatalf("expected 89 independent region rows, got %d", regionCount)
	}
	rs := matchingRecords(all, "красноярский край")
	if len(rs) < 2 || !rs[0].IsRegion || rs[0].Region != "Красноярский край" || recordTitle(rs[0]) != "Регион: Красноярский край" {
		t.Fatalf("region not first: %+v", rs)
	}
	s := climateFromRecord(rs[0])
	if s.Place != "" || s.District != "" || s.SnowGround != 0 {
		t.Fatalf("region populated: %+v", s)
	}
}
func TestShortReportsOmitUnassigned(t *testing.T) {
	s := Site{Region: "Красноярский край", Place: "Дефановка", SnowZone: "II", SnowGround: 1}
	rows, e := siteReportRows(s)
	if e != nil || len(rows) != 1 || !strings.Contains(rows[0].Parameter, "снег") {
		t.Fatalf("rows=%+v err=%v", rows, e)
	}
	for _, h := range reportHeader(s) {
		if strings.Contains(h, "район") || strings.Contains(h, "не задан") {
			t.Fatalf("empty district printed: %q", h)
		}
	}
	for _, ext := range []string{"xlsx", "docx"} {
		path := filepath.Join(t.TempDir(), "site."+ext)
		if ext == "xlsx" {
			e = ExportSiteExcel(s, path)
		} else {
			e = ExportSiteWord(s, path)
		}
		if e != nil {
			t.Fatal(e)
		}
		part := "word/document.xml"
		if ext == "xlsx" {
			part = "xl/worksheets/sheet1.xml"
		}
		body := zipPart(t, path, part)
		for _, word := range []string{"Гололёд", "не установлен", "не задан", "Муниципальный район:"} {
			if strings.Contains(body, word) {
				t.Fatalf("unexpected %q in %s", word, ext)
			}
		}
		if !strings.Contains(body, "Дефановка") || !strings.Contains(body, "кПа") {
			t.Fatalf("missing values in %s", ext)
		}
	}
}
func TestSiteReportFromSelectedCityDiffersFromSaved(t *testing.T) {
	saved := climateFromRecord(pickCity(t, "Краснодар"))
	selected := climateFromRecord(pickCity(t, "Казань"))
	if saved.Place == selected.Place {
		t.Fatal("bad setup")
	}
	fn := filepath.Join(t.TempDir(), "site.docx")
	if e := ExportSiteWord(selected, fn); e != nil {
		t.Fatal(e)
	}
	doc := zipPart(t, fn, "word/document.xml")
	if !strings.Contains(doc, "Казань") || strings.Contains(doc, "Краснодар") || !strings.Contains(doc, "приложение К") {
		t.Fatal("wrong report site")
	}
}
func TestRegionWithoutWindDoesNotFreezeFutureSite(t *testing.T) {
	p := initialProject()
	p.SetSite(Site{Region: "Красноярский край"})
	if p.WindOverride || p.Wind.W0 != 0 {
		t.Fatalf("unassigned wind state %+v", p)
	}
	p.SetSite(climateFromRecord(pickCity(t, "Казань")))
	if p.WindOverride || p.Wind.W0 != .3 {
		t.Fatal("wind not adopted from next populated site")
	}
}
