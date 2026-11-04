package main

import (
	"strings"
	"testing"
)

func TestSnowClimateDatabase(t *testing.T) {
	if snowClimateError != nil {
		t.Fatal(snowClimateError)
	}
	if n := len(snowClimateCatalog); n != 551 {
		t.Fatalf("expected 551 places, got %d", n)
	}
	regs := map[string]bool{}
	for _, x := range snowClimateCatalog {
		regs[x.Region] = true
	}
	if n := len(regs); n != 86 {
		t.Fatalf("expected 86 regions, got %d", n)
	}
	krasnodar := snowClimateFiltered("Краснодарский край", "")
	if len(krasnodar) != 8 {
		t.Fatalf("Краснодарский край: wanted 8 places, got %d", len(krasnodar))
	}
	rows := snowClimateFiltered("Краснодарский край", "Новоросс")
	if len(rows) != 1 || rows[0].Locality != "Новороссийск" || rows[0].January != 3.8 || rows[0].Wind != 5.1 {
		t.Fatalf("unexpected Novorossiysk: %#v", rows)
	}
	if len(snowClimateFiltered("", "Краснодар")) < 1 {
		t.Fatal("global search must find city without region")
	}
	if len(snowClimateFiltered("Краснодарский край", "Казань")) != 0 {
		t.Fatal("region filter not applied")
	}
}
func TestSnowClimateSavingAndReports(t *testing.T) {
	r := snowClimateFiltered("Республика Татарстан (Татарстан)", "Казань")
	if len(r) != 1 {
		t.Fatalf("Kazan missing: %d", len(r))
	}
	s := Site{Region: "Республика Татарстан (Татарстан)", SnowZone: "III"}
	s.SetSnowClimate(r[0])
	if !s.ClimateJanuarySet || !s.ClimateWindSet {
		t.Fatal("flags")
	}
	if s.ClimateJanuary != -10.8 || s.ClimateWind != 2.8 {
		t.Fatalf("data %#v", s)
	}
	rows, err := siteReportRows(s)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, row := range rows {
		if strings.Contains(row.Parameter, "января") || strings.Contains(row.Parameter, "холодного периода") {
			found++
			if !strings.Contains(row.Source, "Казань") || !strings.Contains(row.Source, "СП 131.13330.2025") {
				t.Fatal("source lost: ", row)
			}
		}
	}
	if found != 2 {
		t.Fatalf("expected temperature and wind rows, got %d", found)
	}
	s.SetSnowClimateManual(0, 3.1, true, true)
	if s.ClimateOrigin != "manual" || s.ClimateReferencePlace != "" {
		t.Fatal("manual input must clear SP source")
	}
	if !strings.Contains(snowClimateReference(s), "пользователем") {
		t.Fatal("manual source")
	}
	snow := defaultSnowCalc()
	snow.InheritClimate(s)
	if !snow.JanuarySet || snow.JanuaryTemp != 0 || !snow.WinterWindSet || snow.WinterWind != 3.1 {
		t.Fatalf("inherit: %#v", snow)
	}
}

func TestSnowClimateSortedByRegionAndLocality(t *testing.T) {
	rows := snowClimateFiltered("", "")
	if len(rows) != 551 {
		t.Fatalf("unexpected length: %d", len(rows))
	}
	for i := 1; i < len(rows); i++ {
		prev, cur := rows[i-1], rows[i]
		rp, rc := normalizeSearch(prev.Region), normalizeSearch(cur.Region)
		if rp > rc || (rp == rc && normalizeSearch(prev.Locality) > normalizeSearch(cur.Locality)) {
			t.Fatalf("wrong order at %d: %s/%s before %s/%s", i, prev.Region, prev.Locality, cur.Region, cur.Locality)
		}
	}
	subset := snowClimateFiltered("Краснодарский край", "")
	for i := 1; i < len(subset); i++ {
		if normalizeSearch(subset[i-1].Locality) > normalizeSearch(subset[i].Locality) {
			t.Fatalf("wrong regional order: %s, %s", subset[i-1].Locality, subset[i].Locality)
		}
	}
	found := snowClimateFiltered("", "Краснодарский край")
	if len(found) != 8 {
		t.Fatalf("region text search failed: got %d", len(found))
	}
}
