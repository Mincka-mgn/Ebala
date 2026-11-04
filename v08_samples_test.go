package main

import (
	"os"
	"testing"
)

func TestGenerateV08Samples(t *testing.T) {
	dir := "/mnt/data/poverka_v08_samples"
	if e := os.MkdirAll(dir, 0755); e != nil {
		t.Fatal(e)
	}
	for _, c := range []struct {
		f string
		s Site
	}{
		{"Kazan", climateFromRecord(pickCity(t, "Казань"))},
		{"Krasnoyarsk_region", Site{Region: "Красноярский край", Place: "Дефановка", SnowZone: "II", SnowGround: 1}},
		{"Region_only", Site{Region: "Красноярский край"}},
	} {
		if e := ExportSiteWord(c.s, dir+"/"+c.f+".docx"); e != nil {
			t.Fatal(e)
		}
		if e := ExportSiteExcel(c.s, dir+"/"+c.f+".xlsx"); e != nil {
			t.Fatal(e)
		}
	}
}
