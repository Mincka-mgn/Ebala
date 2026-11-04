package main

import (
	"os"
	"testing"
)

func TestExternalSeismicCatalog(t *testing.T) {
	b, e := os.ReadFile("Maps/Seismic/settlements.tsv")
	if e != nil {
		t.Fatal(e)
	}
	rr := parseSeismicTSV(string(b))
	if len(rr) < 2500 {
		t.Fatalf("Too few records: %d", len(rr))
	}
	for _, x := range []struct{ region, place, a, b, c string }{
		{"Республика Адыгея", "Майкоп", "7", "8", "9"},
		{"Краснодарский край", "Краснодар", "7", "8", "9"},
		{"Краснодарский край", "Тихорецк", "6", "6", "7"},
		{"Краснодарский край", "Туапсе", "8", "9", "9"},
		{"Камчатский край", "Петропавловск-Камчатский", "9", "10", "10"},
		{"Донецкая Народная Республика", "Донецк", "-", "6", "7"},
	} {
		var found bool
		for _, r := range rr {
			if canonicalRegionMatch(r.Region, x.region) && normalizeGeo(r.Place) == normalizeGeo(x.place) {
				found = true
				if r.A != x.a || r.B != x.b || r.C != x.c {
					t.Errorf("%s: %+v", x.place, r)
				}
			}
		}
		if !found {
			t.Errorf("not found %s %s", x.region, x.place)
		}
	}
	if len(seismicReviews) > 0 {
		t.Fatal("clean DOCX source should not carry OCR review flags")
	}
}
