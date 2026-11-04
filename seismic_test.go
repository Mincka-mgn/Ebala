package main

import (
	"strings"
	"testing"
)

func TestSeismicTSVRoundTrip(t *testing.T) {
	src := []SeismicRecord{{"Республика Адыгея", "Майкоп", "7", "8", "9"}, {"Республика Адыгея", "Энем", "8", "8", "9"}}
	got := parseSeismicTSV(seismicTSV(src))
	if len(got) != 2 {
		t.Fatalf("got %d", len(got))
	}
	a, ok := seismicAutoMatchTest(got, "Республика Адыгея", "Майкоп")
	if !ok || a.Value("В") != "8" {
		t.Fatalf("got %+v %v", a, ok)
	}
}
func seismicAutoMatchTest(rows []SeismicRecord, region, place string) (SeismicRecord, bool) {
	for _, r := range rows {
		if canonicalRegionMatch(r.Region, region) && normalizeGeo(r.Place) == normalizeGeo(place) {
			return r, true
		}
	}
	return SeismicRecord{}, false
}
func TestParseSeismicHTMLThreeColumns(t *testing.T) {
	sample := `<h2>Приложение А</h2><table><tr><td colspan="12">Республика Адыгея</td></tr><tr><td>Майкоп</td><td>7</td><td>8</td><td>9</td><td>Энем</td><td>8</td><td>8</td><td>9</td><td>Адыгейск</td><td>8</td><td>8</td><td>9</td></tr><tr><td colspan="12">Краснодарский край</td></tr><tr><td>Условный город</td><td>-</td><td>6</td><td>7</td></tr></table>`
	rows := parseSeismicHTML(sample)
	if len(rows) != 4 {
		t.Fatalf("rows=%d: %+v", len(rows), rows)
	}
	if rows[0].Region == "" {
		t.Fatal("missing region")
	}
}
func TestSeismicNoAutoZero(t *testing.T) {
	ss := seismicSeed()
	found, _ := seismicAutoMatchTest(ss, "Краснодарский край", "Кемерово")
	if found.Place != "" {
		t.Fatal("invented regional value")
	}
}
func TestSeismicInSiteReport(t *testing.T) {
	s := Site{Region: "Республика Адыгея", Place: "Неизвестное село", SeismicMap: "В", SeismicRefPlace: "Майкоп", SeismicRefRegion: "Республика Адыгея", SeismicPoints: "8"}
	rows, e := siteReportRows(s)
	if e != nil {
		t.Fatal(e)
	}
	var got string
	for _, r := range rows {
		if strings.Contains(r.Parameter, "Сейсмичность") {
			got = r.Zone + " / " + r.Value + " / " + r.Source
		}
	}
	if !strings.Contains(got, "Карта В / 8 баллов") || !strings.Contains(got, "Майкоп") {
		t.Fatalf("report: %s", got)
	}
}
