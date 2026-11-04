package main

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
)

func TestV07SwitchWeightMethodPreservesLoads(t *testing.T) {
	r := WeightRow{Name: "Утеплитель", Kind: "layer", Thickness: 70, Density: 400, Gamma: 1.3, NormEdited: true}
	norm, design := r.NormKgf(), r.NormKgf()*r.Gamma
	direct := SwitchWeightKind(r, "direct")
	if direct.Kind != "direct" || math.Abs(direct.NormKgf()-norm) > 1e-9 || math.Abs(direct.NormKgf()*direct.Gamma-design) > 1e-9 || !direct.NormEdited {
		t.Fatalf("layer->direct: %#v", direct)
	}
	back := SwitchWeightKind(direct, "layer")
	if back.Kind != "layer" || math.Abs(back.NormKgf()-norm) > 1e-9 || math.Abs(back.NormKgf()*back.Gamma-design) > 1e-9 {
		t.Fatalf("direct->layer: %#v", back)
	}
	x := SwitchWeightKind(WeightRow{Name: "Вес плиты", Kind: "direct", GNorm: 296, Gamma: 1.1}, "layer")
	if x.Thickness <= 0 || x.Density <= 0 || math.Abs(x.NormKgf()-296) > 1e-9 {
		t.Fatalf("default thickness: %#v", x)
	}
}
func TestV07CategorizedCatalogCompleteAndDeduplicated(t *testing.T) {
	bytes, err := os.ReadFile("Catalog/materials.json")
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Categories []struct{ ID, Group, Name string }
		Items      []struct {
			ID, Name, Kind string
			CategoryID     string   `json:"category_id"`
			CategoryIDs    []string `json:"category_ids"`
			DensityMin     *float64 `json:"density_min"`
			DensityMax     *float64 `json:"density_max"`
			Thickness      float64  `json:"default_thickness_mm"`
			GNorm          float64  `json:"g_norm"`
		}
	}
	if e := json.Unmarshal(bytes, &c); e != nil {
		t.Fatal(e)
	}
	valid := map[string]bool{}
	for _, cat := range c.Categories {
		valid[cat.ID] = true
	}
	names := map[string]bool{}
	shared := 0
	for _, v := range c.Items {
		if !valid[v.CategoryID] {
			t.Errorf("no primary category %s: %q", v.CategoryID, v.Name)
		}
		if v.Kind != "layer" && v.Kind != "direct" {
			t.Errorf("unknown method: %q", v.Name)
		}
		if v.Kind == "layer" && v.DensityMax == nil {
			t.Errorf("missing density: %q", v.Name)
		}
		if v.Kind == "direct" && v.GNorm < 0 {
			t.Errorf("negative weight: %q", v.Name)
		}
		k := strings.ToLower(v.Name) + "/" + v.Kind
		if names[k] {
			t.Errorf("duplicate row: %q", v.Name)
		}
		names[k] = true
		for _, id := range v.CategoryIDs {
			if !valid[id] || id == v.CategoryID {
				t.Errorf("incorrect secondary category %q for %q", id, v.Name)
			}
		}
		if len(v.CategoryIDs) > 0 {
			shared++
		}
	}
	if shared < 60 {
		t.Fatalf("too few multi-category materials: %d", shared)
	}
	find := func(name string) (string, []string) {
		for _, v := range c.Items {
			if v.Name == name {
				return v.CategoryID, v.CategoryIDs
			}
		}
		return "", nil
	}
	id, ext := find("Асфальтобетон")
	if id != "c22" || !contains(ext, "c27") {
		t.Errorf("asphalt categories %q %q", id, ext)
	}
	id, _ = find("Минераловатные плиты на битумном связующем")
	if id != "c2" {
		t.Errorf("bitumen wool category: %s", id)
	}
	id, _ = find("Гипсовая сухая штукатурка (лист)")
	if id != "c28" {
		t.Errorf("plaster category: %s", id)
	}
	if len(c.Items) > 200 {
		t.Errorf("excess repeated materials: %d", len(c.Items))
	}
}
func contains(s []string, value string) bool {
	for _, v := range s {
		if v == value {
			return true
		}
	}
	return false
}
