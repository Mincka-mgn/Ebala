package main

import (
	"math"
	"os"
	"strings"
	"testing"
)

// Independent normative control: S0 is the snow-on-ground Sg in annex B.
// Changing ct cannot change the drift eligibility h>=S0/2.
func Test056B8ApplicabilityIndependentOfCT(t *testing.T) {
	for _, h := range []float64{0.9, 1.1} {
		for _, warm := range []bool{false, true} {
			s := defaultSnowCalc()
			s.Shape = "mono"
			s.Span, s.Width, s.Angle = 12, 12, 5
			s.RoofHeight, s.Ground = 12, 2
			s.WarmRoof = warm
			rise := s.Span * math.Tan(s.Angle*math.Pi/180)
			right := SnowRoofPart{
				Shape: "mono", Span: 12, Width: 12, Angle: 5, Rise: rise,
				RoofHeight: s.RoofHeight + rise - h, WarmRoof: warm,
			}
			s.RightRoof = &right
			got, err := SnowCompute(s)
			if err != nil {
				t.Fatalf("h=%g warm=%v: %v", h, warm, err)
			}
			found := false
			for _, c := range got.Cases {
				if strings.Contains(c.Title, "Занос у перепада") || strings.Contains(c.Title, "У высокой стены / перепада") {
					found = true
				}
			}
			want := h >= 1.0
			if found != want {
				t.Fatalf("h=%g warm=%v ct=%g drift=%v want=%v", h, warm, got.CT, found, want)
			}
			if warm && math.Abs(got.CT-0.8) > 1e-8 {
				t.Fatalf("ct not reduced: %g", got.CT)
			}
		}
	}
}

func Test056B15aAzimuthAndSeam(t *testing.T) {
	s := sample049()
	g, err := makeSphereConeGeometry(s)
	if err != nil {
		t.Fatal(err)
	}
	r := (g.ConeRadius + g.D/2) / 2
	mu90, zone := g.spatialMu(r, 90)
	if zone != 1 || mu90 <= 0 {
		t.Fatalf("beta90: zone=%d mu=%g", zone, mu90)
	}
	for _, b := range []float64{0, 30, 150, 180, 360} {
		got, _ := g.spatialMu(r, b)
		if math.Abs(got-0.5*mu90) > 1e-10 {
			t.Fatalf("beta%g = %g; expected %g", b, got, 0.5*mu90)
		}
	}
	mu270, zone := g.spatialMu(r, 270)
	if zone != 6 || math.Abs(mu270-0.5*mu90) > 1e-10 {
		t.Fatalf("zone6 beta270: mu=%g zone=%d want=%g", mu270, zone, 0.5*mu90)
	}
	for _, b := range []float64{30 - 1e-6, 30 + 1e-6, 150 - 1e-6, 150 + 1e-6} {
		got, _ := g.spatialMu(r, b)
		if math.Abs(got-0.5*mu90) > 1e-6 {
			t.Fatalf("non-seamless zone1/2 beta%g: %g", b, got)
		}
	}
	mz5, z5 := g.spatialMu(g.ConeRadius/2, 270)
	if z5 != 5 || mz5 <= 0 {
		t.Fatalf("zone5 beta270: %g / %d", mz5, z5)
	}
	if _, z4 := g.spatialMu(g.ConeRadius/2, 180); z4 != 4 {
		t.Fatalf("zone4 beta180: %d", z4)
	}
}

func Test056B15aCr1UsesVirtualSphereRise(t *testing.T) {
	g, err := makeSphereConeGeometry(sample049())
	if err != nil {
		t.Fatal(err)
	}
	fs := (g.D / 2) * math.Tan(g.Angle1*math.Pi/360)
	if math.Abs(g.sphereOnlyRise()-fs) > 1e-12 {
		t.Fatal("wrong virtual sphere rise")
	}
	if math.Abs(fs-g.Rise) < 0.1 {
		t.Fatalf("control geometry not discriminating: fs=%g f=%g", fs, g.Rise)
	}
	expected := 2.55 - math.Exp(0.8-14*fs/g.D)
	// Choose alpha=30° point so domeMu2 equals Cr1.
	r30 := g.CenterX + math.Abs(g.SphereRadius)*0.5
	if r30 >= g.ConeRadius && r30 <= g.D/2 && math.Abs(g.sideAngle(r30)-30) < 1e-6 {
		mu := g.domeMu2(r30)
		if math.Abs(mu-expected) > 1e-9 {
			t.Fatalf("Cr1=%g expected=%g", mu, expected)
		}
	}
	// The formula is expressed in the independent Excel geometry as well.
	excel := snowExcelSphereGeometry(sample049(), "Исходные данные")
	if !strings.Contains(excel, "14*B23/B3") || !strings.Contains(excel, "B3/2*TAN(RADIANS(B7/2))") {
		t.Fatal("Excel still computes Cr1 from the overall rise")
	}
}

func Test056B15aMaxIndependentOfPolarGridStep(t *testing.T) {
	ref := -1.0
	for _, step := range []float64{5, 7, 15, 30, 90} {
		s := sample049()
		s.BetaStep = step
		got, err := SnowCompute(s)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Cases) < 2 {
			t.Fatal("missing spatial case")
		}
		c := got.Cases[1]
		max := got.MaxMu(c)
		if ref < 0 {
			ref = max
		} else if math.Abs(max-ref) > 1e-10 {
			t.Fatalf("beta step=%g: max=%g, expected=%g", step, max, ref)
		}
		for _, p := range c.Polar {
			if p.Mu > max+1e-10 {
				t.Fatalf("grid exceeds analytic max at beta=%g r=%g: %g>%g", p.Beta, p.Radius, p.Mu, max)
			}
		}
		v := snowPresentations(got)
		if math.Abs(v[1].MaxMu()-max) > 1e-10 {
			t.Fatalf("presentation shows %g, kernel peak %g", v[1].MaxMu(), max)
		}
	}
}

func Test056HiddenB15aSelectorAndVisibleParapet(t *testing.T) {
	data, err := os.ReadFile("snow_ui_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, line := range strings.Split(source, "\n") {
		if strings.HasPrefix(line, "var snowKinds =") || strings.HasPrefix(line, "var snowKindKeys =") {
			if strings.Contains(line, "sphere_cone") || strings.Contains(line, "Сфера с конусом") {
				t.Fatalf("hidden B15a still selectable: %s", line)
			}
			if !strings.Contains(line, "Покрытие с парапетом") && !strings.Contains(line, "parapet") {
				t.Fatalf("parapet vanished: %s", line)
			}
		}
	}
}

func Test056B8ExcelUsesS0Ground(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "mono"
	s.RightRoof = &SnowRoofPart{Shape: "mono", Span: 12, Width: 12}
	sheet := snowExcelPairedStepData(s)
	if !strings.Contains(sheet, "S₀ = Sg") || strings.Contains(sheet, "$B$2*'Коэффициенты'!$B$21") {
		t.Fatal("B.8 worksheet still uses the thermal factor in S0")
	}
}

func Test056DiameterCorrectedBeforeCompute(t *testing.T) {
	for _, input := range []struct{ d, r, f, alpha float64 }{
		{20, 8, 10, 20}, {10, 8, 10, 20}, {10, 8, 2, 20},
	} {
		s := defaultSnowCalc()
		s.Shape = "sphere_cone"
		s.Span, s.Width, s.SkylightWidth, s.H, s.Angle = input.d, input.d, input.r, input.f, input.alpha
		s.RoofHeight = 15
		old, updated, changed := sphereConeExpandDiameter(&s)
		if !changed || updated <= 2*input.r || old != input.d {
			t.Fatalf("D=%g,r=%g,f=%g: old=%g new=%g changed=%v", input.d, input.r, input.f, old, updated, changed)
		}
		_, again, changedAgain := sphereConeExpandDiameter(&s)
		if changedAgain || again != updated {
			t.Fatalf("non-idempotent geometry correction: %g -> %g", updated, again)
		}
		t.Logf("D=%.3f r=%.3f f=%.3f alpha=%.1f: pre-edit diameter -> %.3f", input.d, input.r, input.f, input.alpha, updated)
	}
}
