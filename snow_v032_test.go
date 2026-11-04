package main

import (
	"math"
	"strings"
	"testing"
)

func TestSnow032IndependentRoofs(t *testing.T) {
	s := sampleSnow("gable")
	s.Angle = 10
	s.Span = 18
	s.Rise = 0
	s.RightRoof = &SnowRoofPart{Shape: "vault", Span: 22, Rise: 2.8, SlopeMode: "height", EdgeLevel: 8}
	r, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Cases) < 2 {
		t.Fatalf("expected independent cases: %d", len(r.Cases))
	}
	if _, hi, _ := snowCaseBounds(r.Cases[0]); math.Abs(hi-40) > 0.01 {
		t.Fatalf("width of combined roof: %.2f", hi)
	}
	s.RightRoof.EdgeLevel = 4
	r, e = SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, c := range r.Cases {
		if strings.Contains(c.Title, "перепада") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing stepped roof drift")
	}
}
func TestSnow032DomeSpatial(t *testing.T) {
	s := sampleSnow("dome")
	s.Span = 24
	s.Rise = 3.5
	r, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Cases) < 2 || len(r.Cases[1].Polar) == 0 {
		t.Fatalf("dome 2nd: %d", len(r.Cases))
	}
	s.DomeObstacles = true
	r, e = SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Cases) < 3 || len(r.Cases[2].Polar) == 0 {
		t.Fatalf("dome third: %d", len(r.Cases))
	}
	for _, c := range r.Cases {
		for _, p := range c.Polar {
			if p.Mu < 0 || math.IsNaN(p.Mu) {
				t.Fatal("invalid μ")
			}
		}
	}
}
