package main

import (
	"archive/zip"
	"io"
	"math"
	"path/filepath"
	"strings"
	"testing"
)

func auditedSnow() SnowCalc {
	s := defaultSnowCalc()
	s.Shape = "gable"
	s.Ground = 2
	s.Span = 18
	s.Width = 24
	s.Angle = 5
	s.Rise = 18 / 2 * math.Tan(5*math.Pi/180)
	s.RoofHeight = 10
	s.Terrain = "А"
	s.JanuarySet = true
	s.JanuaryTemp = -10
	s.WinterWindSet = true
	s.WinterWind = 5
	s.Exposed = true
	return s
}

func TestAudit034A01ExcelAndGoPaired(t *testing.T) {
	s := auditedSnow()
	s.WarmRoof = true
	s.RightRoof = &SnowRoofPart{Shape: "gable", Span: 18, Width: 24, Angle: 5, Rise: s.Rise, RoofHeight: 10, WarmRoof: false}
	out, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	if out.CE != 1 || out.CT != 1 || math.Abs(out.Q(1)-2) > 1e-10 {
		t.Fatalf("A01 core mismatch ce=%v ct=%v q=%v", out.CE, out.CT, out.Q(1))
	}
	fn := filepath.Join(t.TempDir(), "two_roofs.xlsx")
	if err = ExportSnowExcel(out, fn); err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(fn)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	found := false
	for _, f := range z.File {
		if !strings.Contains(f.Name, "worksheets/snow") {
			continue
		}
		rc, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		content, e := io.ReadAll(rc)
		rc.Close()
		if e != nil {
			t.Fatal(e)
		}
		x := string(content)
		if strings.Contains(x, "КОЭФФИЦИЕНТЫ") {
			found = true
			if !strings.Contains(x, "&#39;Исходные данные&#39;!B5") || !strings.Contains(x, "&#39;Исходные данные&#39;!B6") {
				t.Fatalf("A01: Excel ce/ct not bound to actual core: %s", f.Name)
			}
		}
	}
	if !found {
		t.Fatal("coefficient sheet not found")
	}
}

func TestAudit034A02SpatialConeNoReduction(t *testing.T) {
	s := auditedSnow()
	s.Shape = "cone"
	s.Span = 30
	s.Width = 30
	s.Angle = 25
	s.Rise = 15 * math.Tan(25*math.Pi/180)
	out, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if out.CE != 1 {
		t.Fatalf("A02 spatial cone ce=%v", out.CE)
	}
	if len(out.Cases) < 2 || len(out.Cases[1].Polar) == 0 {
		t.Fatalf("A02 spatial cone cases=%d", len(out.Cases))
	}
}

func TestAudit034A03LargeRoofAutomaticNonuniform(t *testing.T) {
	s := auditedSnow()
	s.Span = 120
	s.Width = 150
	s.Angle = 5
	s.Rise = 60 * math.Tan(5*math.Pi/180)
	s.RoofSensitive = false
	out, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(out.Cases) < 2 {
		t.Fatalf("A03: >100m in both axes must have nonuniform variants: %d", len(out.Cases))
	}
}

func TestAudit034A04PartialOnePitch(t *testing.T) {
	s := auditedSnow()
	s.Angle = 35
	s.Rise = 9 * math.Tan(35*math.Pi/180)
	out, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	a, b := false, false
	for _, c := range out.Cases {
		if strings.Contains(c.Title, "Частичное загружение левого") {
			a = true
		}
		if strings.Contains(c.Title, "Частичное загружение правого") {
			b = true
		}
	}
	if !a || !b {
		t.Fatalf("A04 partial cases absent")
	}
}

func TestAudit034A05ProtectedRoofNoWindReduction(t *testing.T) {
	s := auditedSnow()
	s.Exposed = false
	s.Angle = 5
	out, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if out.CE != 1 {
		t.Fatalf("A05 ce=%v, expected 1", out.CE)
	}
}

func TestAudit034A07Screed(t *testing.T) {
	p := initialProject()
	if len(p.Layers) == 0 || p.Layers[0].GammaF != 1.3 {
		t.Fatalf("A07 screed gamma: %+v", p.Layers)
	}
}

func TestAudit034A09ReductionByIDOnly(t *testing.T) {
	cases := map[string]int{"T-00012-1": 1, "T-00013-6": 1, "T-00015-9": 2, "T-00030-10": 2, "T-00030-20": 2, "T-00030-26": 2}
	for id, want := range cases {
		got := tempReductionClass(SP20Row{ID: id, Title: "коридоры вокзалов, залы театров, этажи", Source: "СП 20.13330.2016, таблица 8.3"})
		if got != want {
			t.Fatalf("A09 id %s got %d want %d", id, got, want)
		}
	}
}

func TestAudit034NegativeMonoRoof(t *testing.T) {
	s := auditedSnow()
	s.Shape = "mono"
	s.Angle = -10
	s.Rise = 0
	s.SlopeMode = "angle"
	out, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if out.Angle >= 0 || out.Rise >= 0 || len(out.Cases) == 0 {
		t.Fatal("negative mono direction lost", out.Angle, out.Rise)
	}
}

func TestAudit034LanternAtStepNoSilentOmission(t *testing.T) {
	s := auditedSnow()
	s.Shape = "skylight"
	s.SkylightWidth = 6
	s.H = 2
	s.Rise = (18 - 6) / 2 * math.Tan(5*math.Pi/180)
	s.RightRoof = &SnowRoofPart{Shape: "skylight", Span: 18, Width: 24, SkylightWidth: 6, LanternHeight: 2, Angle: 5, Rise: s.Rise, RoofHeight: 7}
	out, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	yes, note := false, false
	for _, c := range out.Cases {
		if strings.Contains(c.Title, "Занос у перепада") {
			yes = true
			if strings.Contains(c.Note, "ПРЕДВАРИТЕЛЬНО") {
				note = true
			}
		}
	}
	if !yes || !note {
		t.Fatalf("step with lantern must have computed preliminary variant and caveat")
	}
}

func TestAudit034A06AsymmetricCraneDirection(t *testing.T) {
	r, err := CalculateCranePlacements([]float64{6000, 12000}, []CraneTrain{{Name: "Несимметричный", WheelXMM: []float64{0, 2000}, ForcesKN: []float64{20, 80}}})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range r.Cases {
		if c.Name == "Q справа от опоры 1 (+)" {
			found = true
			if c.Value < 90 {
				t.Fatalf("A06 lost reversed wheel orientation: Q=%.6f, want at least 90 kN", c.Value)
			}
		}
	}
	if !found {
		t.Fatal("A06 missing shear case")
	}
}
