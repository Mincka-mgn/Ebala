package main

import (
	"math"
	"strings"
	"testing"
)

func Test038IceNormativeSection12(t *testing.T) {
	s := defaultIceLoad()
	s.Region = "II"
	s.BaseMM = 5
	s.HeightStart = 10
	s.HeightEnd = 10
	out, e := IceCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(out.Points) != 1 {
		t.Fatalf("points=%d", len(out.Points))
	}
	p := out.Points[0]
	expected := math.Pi * 5 * (10 + 5) * .9 * 9.81 * .001
	if math.Abs(p.Norm-expected) > 1e-12 || math.Abs(p.Design-1.8*expected) > 1e-12 {
		t.Errorf("12.1: %#v vs %g", p, expected)
	}
	s.Shape = "surface_small"
	s.HeightStart = 10
	s.HeightEnd = 10
	out, e = IceCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if math.Abs(out.Points[0].Norm-5*.6*.9*9.81) > 1e-12 {
		t.Errorf("12.2: %#v", out.Points[0])
	}
	s.Shape = "round"
	s.HeightStart = 5
	s.HeightEnd = 100
	s.HeightStep = 25
	out, e = IceCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	km := map[float64]float64{5: .8, 10: 1, 20: 1.2, 30: 1.4, 50: 1.6, 70: 1.8, 100: 2}
	for _, p := range out.Points {
		if wanted, ok := km[p.HeightM]; ok && math.Abs(p.K-wanted) > 1e-12 {
			t.Errorf("h=%v k=%v want %v", p.HeightM, p.K, wanted)
		}
	}
}
func Test038IceLimitations(t *testing.T) {
	cases := []struct {
		region, category string
		fromStudy        bool
		base, h, diam    float64
		study            string
		msg              string
	}{
		{"I", "ordinary", false, 3, 10, 10, "", "только нижнюю"},
		{"V", "ordinary", false, 20, 10, 10, "", "только нижнюю"},
		{"II", "ordinary", false, 4, 10, 10, "", "ниже минимального"},
		{"II", "ordinary", false, 5, 4, 10, "", "Высоты"},
		{"II", "ordinary", false, 5, 401, 10, "", "Высоты"},
		{"II", "ordinary", false, 5, 10, 71, "", "диаметров"},
		{"II", "special", false, 5, 300, 10, "", "обследований"},
	}
	for _, c := range cases {
		s := defaultIceLoad()
		s.Region = c.region
		s.HighZone = c.category
		s.BaseMM = c.base
		s.BaseFromStudy = c.fromStudy
		s.StudyReference = c.study
		s.HeightStart = c.h
		s.HeightEnd = c.h
		s.DiameterMM = c.diam
		_, e := IceCompute(s)
		if e == nil || !strings.Contains(e.Error(), c.msg) {
			t.Errorf("%+v expected %q got %v", c, c.msg, e)
		}
	}
}
func Test038IceHighInterpAndHorizontal(t *testing.T) {
	s := defaultIceLoad()
	s.Region = "III"
	s.BaseMM = 10
	s.HeightStart = 200
	s.HeightEnd = 300
	s.HeightStep = 50
	o, e := IceCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if math.Abs(o.Points[0].BaseMM-35) > 1e-12 || math.Abs(o.Points[0].K-1) > 1e-12 {
		t.Errorf("200m %#v", o.Points[0])
	}
	if math.Abs(o.Points[1].BaseMM-35) > 1e-12 || math.Abs(o.Points[1].K-(1+.5*(45./35-1))) > 1e-12 {
		t.Errorf("250m %#v", o.Points[1])
	}
	if math.Abs(o.Points[2].BaseMM-45) > 1e-12 {
		t.Errorf("300m %#v", o.Points[2])
	}
	s.Horizontal = true
	o, e = IceCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if math.Abs(o.Points[0].BaseMM-31.5) > 1e-12 {
		t.Errorf("horizontal 200m %#v", o.Points[0])
	}
}
