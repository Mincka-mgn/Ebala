package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestV09BuiltInFivePackages(t *testing.T) {
	got, err := readTemplates(filepath.Join("Catalog", "constructions_builtin.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"Полы с линолеумом", "Полы с плиткой", "Плоская холодная кровля", "Плоская утеплённая кровля", "Сэндвич-кровля (послойная)"}
	if len(got) != len(expected) {
		t.Fatalf("got %d builtin packages, expected 5", len(got))
	}
	for i, tmpl := range got {
		if tmpl.Name != expected[i] {
			t.Fatalf("name #%d: %q", i, tmpl.Name)
		}
		if len(tmpl.Rows) < 2 {
			t.Errorf("package %q has no layers", tmpl.Name)
		}
		if e := (WeightCalc{Name: tmpl.Name, Rows: tmpl.Rows}).Validate(); e != nil {
			t.Errorf("%s: %v", tmpl.Name, e)
		}
		_, design, _ := (WeightCalc{Rows: tmpl.Rows}).Totals()
		if design <= 0 {
			t.Errorf("package %q has no weight", tmpl.Name)
		}
	}
	if got[0].Rows[0].Thickness != 40 || got[1].Rows[1].Thickness != 10 || got[1].Rows[2].Thickness != 10 {
		t.Fatal("wrong floor-layer thicknesses")
	}
	if len(got[2].Rows) != 5 {
		t.Fatal("cold roofing must have screed, mastic and 3 membrane layers")
	}
	if got[3].Rows[1].Thickness != 100 {
		t.Fatal("insulation should start at 100 mm")
	}
	if got[4].Rows[3].Thickness != 50 || got[4].Rows[1].Name == got[4].Rows[4].Name {
		t.Fatal("sandwich missing 50 mm insulation or two sheets")
	}
}

func TestV09UserTemplatePersistenceAndIsolation(t *testing.T) {
	t.Setenv("EBALA_HOME", t.TempDir())
	if entries, e := readTemplates(templateUserPath(), true); e != nil || len(entries) != 0 {
		t.Fatalf("new directory: %v %v", entries, e)
	}
	src := []WeightRow{{Name: "Непонятная хуетень", Kind: "direct", Gamma: 1.3, GNorm: 25}}
	entries, overwritten, e := saveNamedConstruction(nil, "Мой пирог", src)
	if e != nil || overwritten {
		t.Fatalf("save: %v %v", e, overwritten)
	}
	src[0].GNorm = 100
	if entries[0].Rows[0].GNorm != 25 {
		t.Fatal("template must not share mutable list with calculation")
	}
	if e = saveUserTemplates(entries); e != nil {
		t.Fatal(e)
	}
	onDisk, e := readTemplates(templateUserPath(), true)
	if e != nil || onDisk[0].Rows[0].GNorm != 25 {
		t.Fatalf("roundtrip: %v %+v", e, onDisk)
	}
	entries, replaced, e := saveNamedConstruction(onDisk, "Мой пирог", []WeightRow{{Name: "new", Kind: "direct", Gamma: 1.1, GNorm: 77}})
	if e != nil || !replaced || len(entries) != 1 || entries[0].Rows[0].GNorm != 77 {
		t.Fatalf("replace: %v %+v", e, entries)
	}
	b, e := os.ReadFile(templateUserPath())
	if e != nil {
		t.Fatal(e)
	}
	var f weightTemplateFile
	if e = json.Unmarshal(b, &f); e != nil || f.SchemaVersion != 1 {
		t.Fatalf("invalid portable file: %v", e)
	}
}

func TestV09OldWeightDoesNotRequireTemplates(t *testing.T) {
	rows := []WeightRow{{Name: "старое покрытие", Kind: "layer", Thickness: 200, Density: 2500, Gamma: 1.1}}
	b, e := json.Marshal(WeightCalc{Name: "", Rows: rows})
	if e != nil {
		t.Fatal(e)
	}
	var w WeightCalc
	if e = json.Unmarshal(b, &w); e != nil {
		t.Fatal(e)
	}
	if w.Rows[0].NormKgf() != 500 {
		t.Fatalf("old project changed: %+v", w)
	}
}
