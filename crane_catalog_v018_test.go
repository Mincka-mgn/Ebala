package main

import (
	"strings"
	"testing"
)

func TestCraneCatalogue018(t *testing.T) {
	c, err := loadTempCatalog("Catalog/cranes_gost_handbook.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Records) != 1760 {
		t.Fatalf("cranes: got %d, want 1760", len(c.Records))
	}
	bySource := map[string]int{}
	twoSpan := 0
	for _, r := range c.Records {
		if strings.Contains(r.Group, "серии") {
			t.Fatalf("series in new catalog: %s", r.Title)
		}
		src, ok := r.Params["source_standard"].(string)
		if !ok || src == "" {
			t.Fatalf("no source: %s", r.ID)
		}
		bySource[src]++
		kind, ok := r.Params["crane_kind"].(string)
		if !ok || kind == "" {
			t.Fatalf("no type: %s", r.ID)
		}
		if len(r.Wheels) == 0 {
			t.Fatalf("no wheels: %s", r.ID)
		}
		for i, w := range r.Wheels {
			if w.PKN <= 0 || !w.XKnown {
				t.Fatalf("invalid force/position %s wheel %d", r.ID, i)
			}
			if i > 0 && w.XMM <= r.Wheels[i-1].XMM {
				t.Fatalf("invalid geometry: %s", r.ID)
			}
		}
		if strings.Contains(kind, "двухпролётный") {
			twoSpan++
		}
	}
	if len(bySource) != 4 || twoSpan == 0 {
		t.Fatalf("sources=%v two span=%d", bySource, twoSpan)
	}
}
