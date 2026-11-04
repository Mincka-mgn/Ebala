package main

import (
	"math"
	"testing"
)

func TestCraneSimpleBeamPoint(t *testing.T) {
	s, e := solveCraneBeam([]float64{10000}, []CranePoint{{XMM: 5000, PKN: 100}}, []float64{5})
	if e != nil {
		t.Fatal(e)
	}
	if math.Abs(s.moments[0]-250) > 1e-7 || math.Abs(s.reactions[0]-50) > 1e-7 || math.Abs(s.reactions[1]-50) > 1e-7 {
		t.Fatalf("got %+v", s)
	}
}
func TestCraneContinuousEquilibrium(t *testing.T) {
	s, e := solveCraneBeam([]float64{6000, 6000}, []CranePoint{{XMM: 2000, PKN: 60}, {XMM: 8000, PKN: 90}}, []float64{3, 9})
	if e != nil {
		t.Fatal(e)
	}
	sum := 0.
	for _, v := range s.reactions {
		sum += v
	}
	if math.Abs(sum-150) > 1e-7 {
		t.Fatalf("reactions %+v", s)
	}
}
func TestCraneExtremum(t *testing.T) {
	r, e := CalculateCranePlacements([]float64{10000}, []CraneTrain{{Name: "Кран 1", WheelXMM: []float64{0}, ForcesKN: []float64{100}}})
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, c := range r.Cases {
		if c.Name == "М+ в середине пролёта 1" {
			found = true
			if math.Abs(c.Value-250) > 1e-3 || math.Abs(c.Points[0].XMM-5000) > 1e-2 {
				t.Fatalf("bad maximum %+v", c)
			}
		}
	}
	if !found {
		t.Fatal("no moment")
	}
}
func TestCraneTwoSpanSupportMoment(t *testing.T) {
	s, e := solveCraneBeam([]float64{10000, 10000}, []CranePoint{{XMM: 5000, PKN: 100}}, []float64{10})
	if e != nil {
		t.Fatal(e)
	}
	if math.Abs(s.moments[0]+93.75) > 1e-7 {
		t.Fatalf("support moment got %v want -93.75", s.moments[0])
	}
}
