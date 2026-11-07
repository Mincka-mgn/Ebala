package main

import (
	"math"
	"strings"
	"testing"
)

func TestWind060StepDefaultAndMandatoryHeights(t *testing.T) {
	x := windDefaultInput("")
	if x.ProfileStep != 2.5 {
		t.Fatalf("default step=%v", x.ProfileStep)
	}
	heights, err := WindProfileHeights(11, x.ProfileStep)
	if err != nil {
		t.Fatal(err)
	}
	want := []float64{0, 2.5, 5, 7.5, 10, 11}
	if len(heights) != len(want) {
		t.Fatalf("got %v want %v", heights, want)
	}
	for j, z := range want {
		if math.Abs(heights[j]-z) > 1e-9 {
			t.Fatalf("got %v want %v", heights, want)
		}
	}
	if _, err = WindProfileHeights(12, -1); err == nil {
		t.Fatal("negative step accepted")
	}
}

func TestWind060TwoRidgeDirectionsAndTransverseSize(t *testing.T) {
	windTestData(t)
	in := windDefaultInput("VII")
	in.Height = 60
	in.Depth = 50 // B: length along ridge
	in.Rho = 12   // L: width across ridge
	in.ProfileStep = 2.5
	calc, err := CalculateWindVariants(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(calc.Variants) != 2 {
		t.Fatalf("expected 2 directions, got %d", len(calc.Variants))
	}
	if !strings.Contains(calc.Variants[0].Name, "вдоль конька") {
		t.Fatal(calc.Variants[0].Name)
	}
	if !strings.Contains(calc.Variants[1].Name, "поперёк конька") {
		t.Fatal(calc.Variants[1].Name)
	}
	for vi, v := range calc.Variants {
		width := in.Rho
		if vi == 1 {
			width = in.Depth
		}
		if v.AcrossM != width {
			t.Fatalf("variant %d: transverse %g != %g", vi, v.AcrossM, width)
		}
		for _, zone := range v.Zones {
			if zone.Calculation.Input.Depth != width {
				t.Fatalf("zone %s: wrong transverse in core %g != %g", zone.Code, zone.Calculation.Input.Depth, width)
			}
		}
	}
	// СП 20.13330.2016, 11.1.5, H>2L: z_e=L in lower zone;
	// this is NOT a smooth profile increasing immediately above five metres.
	found := false
	for _, p := range calc.Variants[0].Zones[0].Calculation.Profile {
		if math.Abs(p.Z-5) < 1e-9 {
			found = true
			if math.Abs(p.Result.Ze-12) > 1e-8 {
				t.Fatalf("incorrect lower z_e: %g", p.Result.Ze)
			}
		}
	}
	if !found {
		t.Fatal("mandatory z=5 is absent")
	}
}

func TestWind060ManualPressureDoesNotUseRegion(t *testing.T) {
	windTestData(t)
	in := windDefaultInput("VII")
	in.ManualW0 = true
	in.W0KPa = 1.11
	in.AeroZone = "D"
	c, err := CalculateWind(in)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(c.Result.W0Pa-1110) > 1e-9 {
		t.Fatalf("manual w0 overwritten: %g", c.Result.W0Pa)
	}
	if len(c.Result.Warnings) == 0 {
		t.Fatal("manual source should be visible to engineer")
	}
}
