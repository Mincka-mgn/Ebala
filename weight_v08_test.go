package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestV08MoveWeightRowAndCopy(t *testing.T) {
	rows := []WeightRow{
		{Name: "A", GNorm: 1, Kind: "direct"},
		{Name: "B", GNorm: 2, Kind: "direct"},
		{Name: "C", GNorm: 3, Kind: "direct"},
		{Name: "D", GNorm: 4, Kind: "direct"},
	}
	if !MoveWeightRow(rows, 0, 3) {
		t.Fatal("failed to move down")
	}
	for i, name := range []string{"B", "C", "D", "A"} {
		if rows[i].Name != name {
			t.Fatalf("moved down at %d: %+v", i, rows)
		}
	}
	if !MoveWeightRow(rows, 3, 1) {
		t.Fatal("failed to move up")
	}
	for i, name := range []string{"B", "A", "C", "D"} {
		if rows[i].Name != name {
			t.Fatalf("moved up at %d: %+v", i, rows)
		}
	}
	if MoveWeightRow(rows, 0, 99) || MoveWeightRow(rows, -1, 0) {
		t.Fatal("invalid reorder accepted")
	}
	rows = DuplicateWeightRow(rows, 1)
	if len(rows) != 5 || rows[2].Name != "A" || rows[2].NormKgf() != 1 {
		t.Fatalf("bad duplicate: %#v", rows)
	}
	rows[2].Name = "A (edited)"
	if rows[1].Name != "A" {
		t.Fatal("duplicate must be independent")
	}
}

func TestV08CatalogOrderAndRenames(t *testing.T) {
	b, e := os.ReadFile("Catalog/materials.json")
	if e != nil {
		t.Fatal(e)
	}
	var c struct {
		Categories []struct {
			ID    string
			Group string
			Name  string
		}
		Items []struct {
			Name      string
			Thickness float64 `json:"default_thickness_mm"`
		}
	}
	if e = json.Unmarshal(b, &c); e != nil {
		t.Fatal(e)
	}
	expected := []string{"Бетоны и растворы", "Покрытия и кровли", "Кладка, стены и перегородки", "Отделка и древесина", "Теплоизоляция и засыпки", "Паро- и гидроизоляция", "Другое"}
	groups := []string{}
	for _, cat := range c.Categories {
		if len(groups) == 0 || groups[len(groups)-1] != cat.Group {
			groups = append(groups, cat.Group)
		}
		if cat.ID == "c30" && cat.Group != "Кладка, стены и перегородки" {
			t.Error("supporting frames not moved to wall group")
		}
	}
	if strings.Join(groups, "|") != strings.Join(expected, "|") {
		t.Fatalf("unexpected group order: %v", groups)
	}
	foundPlate, foundPaver := false, false
	for _, m := range c.Items {
		if m.Name == "Плита железобетонная (сплошная)" {
			foundPlate = true
			if m.Thickness != 200 {
				t.Fatalf("default slab thickness %v", m.Thickness)
			}
		}
		if m.Name == "Плитка тротуарная" {
			foundPaver = true
		}
		if strings.Contains(m.Name, "Калифорния") {
			t.Fatalf("obsolete name: %s", m.Name)
		}
	}
	if !foundPlate || !foundPaver {
		t.Fatalf("missing slab/paver %v %v", foundPlate, foundPaver)
	}
}
