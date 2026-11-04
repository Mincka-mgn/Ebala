package main

import (
	"os"
	"testing"
)

func TestEvrazExternalClimateCatalog(t *testing.T) {
	b, e := os.ReadFile("Maps/Climate/settlements.tsv")
	if e != nil {
		t.Fatal(e)
	}
	rr := parseExternalClimateCatalog(string(b))
	if len(rr) < 450 {
		t.Fatalf("incomplete EVRAZ catalog: %d", len(rr))
	}
	seen := make(map[string]bool)
	for _, r := range rr {
		if r.IsCapital || r.IsRegion {
			t.Fatal("unexpected capital/region marker")
		}
		if r.SnowZone == "" || r.WindZone == "" {
			t.Errorf("missing climate zone: %v", r)
		}
		key := normalizeSearch(r.Region) + "|" + normalizeSearch(r.Place)
		if seen[key] {
			t.Errorf("duplicate: %s", key)
		}
		seen[key] = true
	}
	for _, tc := range []struct{ city, region, snow, wind string }{
		{"Майкоп", "Республика Адыгея", "II", "IV"},
		{"Красноярск", "Красноярский край", "III", "III"},
		{"Тихорецк", "Краснодарский край", "II", "IV"},
		{"Ростов-на-Дону", "Ростовская область", "II", "III"},
	} {
		found := false
		for _, r := range rr {
			if r.Place == tc.city && r.Region == tc.region {
				found = true
				if r.SnowZone != tc.snow || r.WindZone != tc.wind {
					t.Errorf("%s: %v", tc.city, r)
				}
			}
		}
		if !found {
			t.Errorf("missing %s", tc.city)
		}
	}
}

func TestAltitudeNoWikiInReport(t *testing.T) {
	s := Site{Region: "Республика Адыгея", Place: "Майкоп", ElevationRaw: "220", ElevationSource: "Википедия"}
	rows, e := siteReportRows(s)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, r := range rows {
		if r.Parameter == "Абсолютная отметка h, м" {
			found = true
			if r.Value != "220" || r.Source != "" {
				t.Fatalf("source or value: %+v", r)
			}
		}
	}
	if !found {
		t.Fatal("altitude lost")
	}
}

// The old tests relied on embedded cities and could not catch a bad live file.
func TestEffectiveAppClimateCatalogAndKPriority(t *testing.T) {
	t.Setenv("EBALA_CLIMATE_CATALOG_PATH", "Maps/Climate/settlements.tsv")
	recs := climateRecords()
	if len(recs) < 540 {
		t.Fatalf("external catalog not merged: %d", len(recs))
	}
	for _, tc := range []struct {
		city, region, snow, wind string
		k                        float64
	}{
		{"Краснодар", "Краснодарский край", "II", "IV", 1.1},
		{"Тихорецк", "Краснодарский край", "II", "IV", 0},
		{"Казань", "Республика Татарстан", "V", "II", 2.3},
	} {
		matches := matchingRecords(recs, tc.city)
		var found bool
		for _, r := range matches {
			if r.Place == tc.city && r.Region == tc.region {
				found = true
				if r.WindZone != tc.wind || r.SnowZone != tc.snow || r.SnowK != tc.k {
					t.Errorf("%s actual %v", tc.city, r)
				}
				if tc.k > 0 {
					if s := climateFromRecord(r); s.SnowGround != tc.k || !specialSnowSite(s) {
						t.Errorf("K must override EVRAZ for %s: %v", tc.city, s)
					}
				}
			}
		}
		if !found {
			t.Errorf("missing from actual combined catalog: %s", tc.city)
		}
	}
}
