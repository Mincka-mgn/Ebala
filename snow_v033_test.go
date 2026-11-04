package main

import (
	"math"
	"strings"
	"testing"
)

func TestSnowV033TwoIndependentSkylights(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "skylight"
	s.Ground = 2.5
	s.Span = 18
	s.Width = 25
	s.SkylightWidth = 6
	s.H = 2
	s.RoofHeight = 9
	s.Angle = 10
	s.Rise = (s.Span - s.SkylightWidth) / 2 * math.Tan(10*math.Pi/180)
	s.WarmRoof = false
	s.RightRoof = &SnowRoofPart{Shape: "skylight", Span: 24, Width: 23, SkylightWidth: 8, LanternHeight: 1.5,
		Angle: 5, SlopeMode: "angle", Rise: 16.0 / 2 * math.Tan(5*math.Pi/180), RoofHeight: 9, EdgeLevel: 9 - 16.0/2*math.Tan(5*math.Pi/180)}
	// Keep the two eave levels the same, avoiding an additional step drift case.
	s.RightRoof.RoofHeight = s.RoofHeight - s.Rise + s.RightRoof.Rise
	s.RightRoof.EdgeLevel = s.RoofHeight - s.Rise
	out, err := SnowCompute(s)
	if err != nil {
		t.Fatalf("paired skylights: %v", err)
	}
	if len(out.Cases) < 5 {
		t.Fatalf("Expected paired lantern cases, got %d", len(out.Cases))
	}
	for _, c := range out.Cases {
		if len(c.Intervals) == 0 {
			t.Errorf("empty case %s", c.Title)
			continue
		}
		if !strings.Contains(c.Title, "перепада") { // base and asymmetric cases cover both spans
			m := 0.
			for _, v := range c.Intervals {
				if v.X2 > m {
					m = v.X2
				}
			}
			if math.Abs(m-42) > .0001 {
				t.Fatalf("case %s covers %.2f, expected 42", c.Title, m)
			}
		}
	}
}

func TestSnowV033IncompatibleReservoirPair(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "cone"
	s.Ground = 2.5
	s.RightRoof = &SnowRoofPart{Shape: "gable", Span: 12, RoofHeight: 7}
	_, err := SnowCompute(s)
	// The Windows selector blocks a right roof. Programmatic inputs must not silently pass.
	if err == nil {
		t.Fatal("reservoir pairing should be rejected")
	}
}

func TestSnowV033ExportPairLanterns(t *testing.T) {
	s := defaultSnowCalc()
	s.Ground = 2.5
	s.Shape = "skylight"
	s.Span = 18
	s.Width = 24
	s.RoofHeight = 9
	s.H = 2
	s.SkylightWidth = 6
	s.Angle = 10
	s.Rise = (s.Span - s.SkylightWidth) / 2 * math.Tan(s.Angle*math.Pi/180)
	r := SnowRoofPart{Shape: "skylight", Span: 24, Width: 24, RoofHeight: s.RoofHeight, SkylightWidth: 7, LanternHeight: 1.5, Angle: 12, SlopeMode: "angle"}
	r.Rise = (r.Span - r.SkylightWidth) / 2 * math.Tan(r.Angle*math.Pi/180)
	r.RoofHeight = s.RoofHeight - s.Rise + r.Rise
	r.EdgeLevel = r.RoofHeight - r.Rise
	s.RightRoof = &r
	out, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Cases) < 5 {
		t.Fatalf("not enough cases: %d", len(out.Cases))
	}
	// Main check: neither document writer may panic on two independent lanterns.
	d := t.TempDir()
	if err = ExportSnowWord(out, d+"/two_lanterns.docx"); err != nil {
		t.Fatal(err)
	}
	if err = ExportSnowExcel(out, d+"/two_lanterns.xlsx"); err != nil {
		t.Fatal(err)
	}
}
