package main

import (
	"os"
	"testing"
)

func TestSeismicSearchByRegionOrPlace(t *testing.T) {
	old := seismicCatalog
	defer func() { seismicCatalog = old }()
	seismicCatalog = []SeismicRecord{
		{Region: "Ростовская область", Place: "Азов", A: "6", B: "6", C: "7"},
		{Region: "Ростовская область", Place: "Ростов-на-Дону", A: "6", B: "6", C: "7"},
		{Region: "Краснодарский край", Place: "Сочи", A: "8", B: "9", C: "9"},
	}
	rows, _ := seismicMatches("Краснодарский край", "Ростов")
	if len(rows) != 2 {
		t.Fatalf("region query must find all Rostov cities, got %+v", rows)
	}
	rows, present := seismicMatches("Ростовская область", "")
	if !present || len(rows) != 2 {
		t.Fatalf("empty query should show just current region: %+v", rows)
	}
	rows, _ = seismicMatches("Ростовская область", "Сочи")
	if len(rows) != 1 || rows[0].Place != "Сочи" {
		t.Fatalf("search must include other regions: %+v", rows)
	}
	rows, present = seismicMatches("Несуществующий регион", "")
	if present || len(rows) != 3 {
		t.Fatalf("missing region must show all rows: %+v", rows)
	}
}

func TestRealSeismicTableRegionLookup(t *testing.T) {
	b, err := os.ReadFile("Maps/Seismic/settlements.tsv")
	if err != nil {
		t.Fatal(err)
	}
	old := seismicCatalog
	defer func() { seismicCatalog = old }()
	seismicCatalog = parseSeismicTSV(string(b))
	rostov, _ := seismicMatches("Краснодарский край", "Ростов")
	if len(rostov) < 45 {
		t.Fatalf("Ростов search should show whole region, got %d", len(rostov))
	}
	krasnodar, ok := seismicMatches("Краснодарский край", "")
	if !ok || len(krasnodar) < 50 {
		t.Fatalf("Krasnodar region absent: %d", len(krasnodar))
	}
}
