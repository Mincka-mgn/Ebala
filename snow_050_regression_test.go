package main

import (
	"archive/zip"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func snow050Sample() SnowCalc {
	s := sample049()
	s.Ground = 4
	s.Gamma = 1.4
	s.H = 6
	s.Angle = 18
	return s
}
func Test050Zone5ModeAndAngularDistribution(t *testing.T) {
	s := snow050Sample()
	g, e := makeSphereConeGeometry(s)
	if e != nil {
		t.Fatal(e)
	}
	if g.Angle1 <= 30 {
		t.Fatalf("test geometry does not reach zone 6: %.4f", g.Angle1)
	}
	r := g.ConeRadius / 2
	m, z := g.spatialMu(r, 135)
	if z != 5 || m <= 0 {
		t.Fatalf("default interpretation zone 5: z=%d μ=%g", z, m)
	}
	strict := s
	strict.SphereStrictFormula = true
	gs, e := makeSphereConeGeometry(strict)
	if e != nil {
		t.Fatal(e)
	}
	m, z = gs.spatialMu(r, 135)
	if z != 5 || m != 0 {
		t.Fatalf("strict interpretation must return zero: z=%d μ=%g", z, m)
	}
	for _, it := range []struct {
		beta  float64
		outer bool
		zone  int
	}{
		{0, true, 1}, {45, true, 1}, {90, true, 2}, {135, true, 6},
		{45, false, 3}, {90, false, 4}, {135, false, 5},
	} {
		radius := r
		if it.outer {
			radius = (g.ConeRadius + g.D/2) / 2
		}
		_, zone := g.spatialMu(radius, it.beta)
		if zone != it.zone {
			t.Errorf("r=%g β=%g: zone=%d want=%d", radius, it.beta, zone, it.zone)
		}
	}
	if mu, _ := g.spatialMu((g.ConeRadius+g.D/2)/2, 0); math.Abs(mu) > 1e-8 {
		t.Errorf("zone1 sin(0)≠0: %g", mu)
	}
	if mu, _ := g.spatialMu((g.ConeRadius+g.D/2)/2, 180); math.Abs(mu) > 1e-8 {
		t.Errorf("zone6 sin(180)≠0: %g", mu)
	}
}
func Test050PartialSectors(t *testing.T) {
	s, e := SnowCompute(snow050Sample())
	if e != nil {
		t.Fatal(e)
	}
	if len(s.Cases) != 8 {
		t.Fatalf("expected 2 primary + 6 partial, got %d", len(s.Cases))
	}
	partial := s.Cases[2:]
	for i, c := range partial {
		low, high, ok := snowPartialSectorBounds(c)
		if !ok || high-low != 180 && high-low != 90 {
			t.Fatalf("bad partial title %q", c.Title)
		}
		if len(c.Polar) != 13*37 {
			t.Fatalf("case %d: polar=%d", i, len(c.Polar))
		}
		good := false
		for _, p := range c.Polar {
			beta := math.Mod(p.Beta, 360)
			if beta < low || beta >= high {
				if p.Mu != 0 {
					t.Fatalf("outside sector has snow: %s %+v", c.Title, p)
				}
			} else if p.Mu > 0 {
				good = true
			}
		}
		if !good {
			t.Errorf("no positive snow in %s", c.Title)
		}
	}
	s.Angle = 18
	s.H = 3
	s.RoofHeight = 6
	// When max roof slope remains below 20 degrees, no partial cases.
	// Use a shallow cone instead of relying on the spherical join geometry.
	q := sampleSnow("cone")
	q.Angle = 10
	v, e := SnowCompute(q)
	if e != nil {
		t.Fatal(e)
	}
	if len(v.Cases) != 2 {
		t.Fatalf("partial sectors added to shallow cone: %d", len(v.Cases))
	}
}
func Test050SheetFormulasAndOrder(t *testing.T) {
	s, e := SnowCompute(snow050Sample())
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "snow050.xlsx")
	if e := ExportSnowExcel(s, path); e != nil {
		t.Fatal(e)
	}
	z, e := zip.OpenReader(path)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	parts := map[string]string{}
	for _, f := range z.File {
		if strings.HasSuffix(f.Name, ".xml") && strings.HasPrefix(f.Name, "xl/") {
			r, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			b, e := io.ReadAll(r)
			r.Close()
			if e != nil {
				t.Fatal(e)
			}
			parts[f.Name] = string(b)
		}
	}
	wb := parts["xl/workbook.xml"]
	first := strings.Index(wb, "name=\"Исходные данные\"")
	last := strings.Index(wb, "name=\"Нагрузки\"")
	if first < 0 || last < first {
		t.Fatalf("sheet order: %s", wb[:min(1200, len(wb))])
	}
	fcount := 0
	seenGrid := false
	for name, xml := range parts {
		if !strings.Contains(name, "worksheets/") {
			continue
		}
		if strings.Contains(name, "snow") {
			if !strings.Contains(xml, "На этом листе") {
				t.Errorf("no A1 introduction in %s", name)
			}
			if strings.Contains(xml, "$I$2") {
				t.Errorf("misaddressed gamma factor in %s", name)
			}
			fcount += strings.Count(xml, "<f>")
			if strings.Contains(xml, "μ по формулам Excel") && strings.Contains(xml, "Геометрия Б.15а") && !strings.Contains(xml, "μ Go") {
				seenGrid = true
			}
		}
	}
	if fcount < 600 {
		t.Fatalf("too few Excel formulas: %d", fcount)
	}
	if !seenGrid {
		t.Fatal("no independent B15a calculation sheet")
	}
}
func Test050QAExport(t *testing.T) {
	dir := os.Getenv("EBALA_QA_OUT")
	if dir == "" {
		t.Skip("export fixtures on demand")
	}
	if e := os.MkdirAll(dir, 0755); e != nil {
		t.Fatal(e)
	}
	for _, item := range []struct {
		name string
		calc SnowCalc
	}{
		{"sphere_alpha2", snow050Sample()},
		{"sphere_alpha1_strict", func() SnowCalc { s := snow050Sample(); s.SphereStrictFormula = true; return s }()},
		{"dome", func() SnowCalc { s := sampleSnow("dome"); s.Span = 24; s.Rise = 3.5; return s }()},
		{"cone", func() SnowCalc { s := sampleSnow("cone"); s.Angle = 25; return s }()},
	} {
		c, e := SnowCompute(item.calc)
		if e != nil {
			t.Fatalf("%s: %v", item.name, e)
		}
		for ext, fn := range map[string]func(SnowCalc, string) error{"docx": ExportSnowWord, "xlsx": ExportSnowExcel} {
			name := filepath.Join(dir, fmt.Sprintf("%s.%s", item.name, ext))
			if e := fn(c, name); e != nil {
				t.Fatalf("%s: %v", name, e)
			}
		}
	}
}
