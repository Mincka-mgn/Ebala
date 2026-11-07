package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWind061DefaultsAndTwoDirections(t *testing.T) {
	windTestData(t)
	in := windDefaultInput("")
	if in.Region != "VII" || in.PulseMode != "mean" || in.ProfileStep != 2.5 {
		t.Fatalf("new defaults: %+v", in)
	}
	in.RoofRise = 2
	c, err := CalculateWindVariants(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Variants) != 2 || c.Variants[c.SelectedVariant].ID != "CROSS" {
		t.Fatalf("default direction %+v", c.Variants)
	}
	for _, v := range c.Variants {
		for _, z := range v.Zones {
			if math.Abs(z.Calculation.Result.PulsePa) > 1e-10 {
				t.Fatalf("mean-only wind: %s/%s", v.ID, z.Code)
			}
		}
	}
}
func TestWind061RoofHeightGrid(t *testing.T) {
	for _, c := range []struct {
		H, h, step float64
		minN       int
	}{{20, 3, 2.5, 6}, {20, 8, 2.5, 6}, {20, 8, .5, 17}, {20, 0, 2.5, 1}} {
		zs, e := WindRoofProfileHeights(c.H, c.h, c.step)
		if e != nil {
			t.Fatal(e)
		}
		if len(zs) < c.minN {
			t.Fatalf("grid H=%g h=%g step=%g, got %v", c.H, c.h, c.step, zs)
		}
		if math.Abs(zs[0]-math.Max(0, c.H-c.h)) > 1e-9 || zs[len(zs)-1] != c.H {
			t.Fatalf("roof boundaries %+v", zs)
		}
		for i := 1; i < len(zs); i++ {
			if zs[i]-zs[i-1] > c.step+1e-9 {
				t.Fatalf("overstep: %v", zs)
			}
		}
	}
	if _, e := WindRoofProfileHeights(20, 9, 0); e == nil {
		t.Fatal("invalid zero step accepted")
	}
	if _, e := WindRoofProfileHeights(20, 9, 1e-12); e == nil {
		t.Fatal("excessive number of ordinates accepted")
	}
}
func TestWind061RoofRealProfilesAndDimensionLabels(t *testing.T) {
	windTestData(t)
	in := testRoofCaseBase(20)
	in.PulseMode = "mean"
	c, e := CalculateWindVariants(in)
	if e != nil {
		t.Fatal(e)
	}
	for _, v := range c.Variants {
		for _, z := range v.Zones {
			text := windZoneLengthsText(&v, &z)
			if !strings.Contains(text, z.Code+":") && z.Surface == "roof" {
				t.Errorf("missing roof zone length in %q", text)
			}
			if z.Surface != "roof" {
				continue
			}
			if len(z.Calculation.Profile) < 6 {
				t.Fatalf("short roof profile %s: %v", z.Code, z.Calculation.Profile)
			}
			got := z.Calculation.Profile
			if math.Abs(got[0].Z-(in.Height-in.RoofRise)) > 1e-8 || got[len(got)-1].Z != in.Height {
				t.Fatalf("roof z range: %v", got)
			}
			for _, p := range got {
				if p.Z > in.Height {
					t.Fatal("profile exceeds ridge")
				}
				if math.Abs(p.Result.DesignPa-z.Calculation.Result.DesignPa) > 1e-8 {
					t.Fatal("unapproved roof load variation")
				}
			}
		}
	}
}
func TestWind061ResourceGeometry(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("Resources", "Wind", "wind_house_side_B_H_h.svg"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), "stroke-dasharray") || strings.Count(string(src), "<rect x=\"180\"") != 1 || strings.Count(string(src), "<rect x=\"310\"") != 1 {
		t.Fatal("side elevation has ridge dash or missing windows")
	}
	fence, err := os.ReadFile(filepath.Join("Resources", "Wind", "wind_fence_H_L.svg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fence), ">L</text>") || !strings.Contains(string(fence), ">H</text>") {
		t.Fatal("fence missing H or L")
	}
}
