package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCrane020TwoWheelsMidspanAndGlobal(t *testing.T) {
	L := []float64{12000}
	for _, xs := range [][]float64{{2200, 6600}, {2280, 6680}, {3800, 8200}} {
		s, e := solveCraneBeam(L, []CranePoint{{XMM: xs[0], PKN: 60}, {XMM: xs[1], PKN: 60}}, []float64{6})
		if e != nil {
			t.Fatal(e)
		}
		if math.Abs(s.moments[0]-228) > 1e-6 {
			t.Fatalf("x=%v midpoint moment=%g want 228", xs, s.moments[0])
		}
	}
	xs := []float64{4900, 9300}
	s, e := solveCraneBeam(L, []CranePoint{{XMM: xs[0], PKN: 60}, {XMM: xs[1], PKN: 60}}, []float64{4.9})
	if e != nil || math.Abs(s.moments[0]-240.1) > 1e-6 {
		t.Fatalf("global analytic x=%v, moments=%v, err %v", xs, s.moments, e)
	}
	r, e := CalculateCranePlacements(L, []CraneTrain{{Name: "Test", WheelXMM: []float64{0, 4400}, ForcesKN: []float64{60, 60}}})
	if e != nil {
		t.Fatal(e)
	}
	foundMid, foundMax := false, false
	for _, c := range r.Cases {
		if c.Name == "М+ в середине пролёта 1" {
			foundMid = true
			if math.Abs(c.Value-228) > 1e-4 {
				t.Fatalf("mid %g", c.Value)
			}
		}
		if strings.Contains(c.Name, "М+ наибольший в пролёте 1") {
			foundMax = true
			if math.Abs(c.Value-240.1) > 0.02 {
				t.Fatalf("max %g, x=%v section=%g", c.Value, c.Points, c.SectionMM)
			}
		}
	}
	if !foundMid || !foundMax {
		t.Fatalf("missing cases %+v", r.Cases)
	}
}
func TestCrane020DiagramOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "crane.bmp")
	c := CranePlacementCase{Points: []CranePoint{{XMM: 2280, PKN: 60}, {XMM: 6680, PKN: 60}}}
	if e := cranePictureForCase([]float64{12000}, c, path); e != nil {
		t.Fatal(e)
	}
	if s, e := os.Stat(path); e != nil || s.Size() < 1000 {
		t.Fatalf("picture missing %v", e)
	}
}
func TestCrane020MetresAndWarning(t *testing.T) {
	spans, w, e := parseCraneSpanMeters("12,5;6", 2)
	if e != nil || w || len(spans) != 2 || spans[0] != 12500 || spans[1] != 6000 {
		t.Fatalf("got %v %v %v", spans, w, e)
	}
	spans, w, e = parseCraneSpanMeters("12000", 1)
	if e != nil || !w || spans[0] != 12000000 {
		t.Fatalf("metres warning %v %v %v", spans, w, e)
	}
}
