package main

import (
	"strings"
	"testing"
)

func TestSnow045B2MirrorOnlyInCalculation(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "vault"
	s.Span = 20
	s.Rise = 3
	s.Ground = 2
	s, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	full, shown := len(s.Cases), len(snowPresentations(s))
	if shown >= full {
		t.Fatalf("mirrors should be removed from presentation, got %d/%d", shown, full)
	}
	for _, v := range snowPresentations(s) {
		if strings.Contains(v.Title, "зеркальн") {
			t.Fatal(v.Title)
		}
	}
}
func TestSnow045B3SimultaneousZones(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "skylight"
	s.Span = 24
	s.Width = 30
	s.H = 2
	s.SkylightWidth = 6
	s.Ground = 2
	s, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	views := snowPresentations(s)
	if len(views) < 2 {
		t.Fatalf("missing variants: %v", views)
	}
	for i := 0; i < 2; i++ {
		if len(views[i].Cases) != 3 || strings.Join(views[i].Zones, "") != "ACB" {
			t.Fatalf("B3 variant %d should have simultaneous A/C/B: %+v", i+1, views[i])
		}
		if views[i].Cases[2].Intervals[0].Mu != 1 {
			t.Fatal("end zone B must have mu 1")
		}
	}
}
