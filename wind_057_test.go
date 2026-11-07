package main

import (
	"archive/zip"
	"encoding/xml"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func windTestData(t *testing.T) {
	t.Helper()
	root, e := filepath.Abs("WindData")
	if e != nil {
		t.Fatal(e)
	}
	t.Setenv("EBALA_WIND_DATA", root)
}
func windNear(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > math.Max(1e-9, math.Abs(want)*1e-9) {
		t.Fatalf("%s: %.12g != %.12g", name, got, want)
	}
}
func windSample(t *testing.T) WindCalculation {
	t.Helper()
	windTestData(t)
	in := windDefaultInput("III")
	// Legacy control example explicitly uses the nonresonant branch; the
	// application default changed to mean-only in Ebala 0.61.
	in.PulseMode = "nonresonant"
	in.Terrain = "B"
	in.Height = 20
	in.Depth = 30
	in.Z = 20
	in.AeroTable = "В.2"
	in.AeroZone = "D"
	in.Rho = 10
	in.Chi = 20
	c, e := CalculateWind(in)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func TestWind057ControlExampleWallBasicB(t *testing.T) {
	c := windSample(t)
	r := c.Result
	windNear(t, "w0", r.W0Pa, 380)
	windNear(t, "ze", r.Ze, 20)
	windNear(t, "k", r.K, .85)
	windNear(t, "c", r.C, .8)
	windNear(t, "zeta", r.Zeta, .92)
	windNear(t, "nu", r.Nu, .81)
	windNear(t, "wm", r.MeanPa, 258.4)
	windNear(t, "wp", r.PulsePa, 192.55968)
	windNear(t, "wn", r.NormPa, 450.95968)
	windNear(t, "wd", r.DesignPa, 631.343552)
	for _, id := range []string{"11.2", "11.5", "11.1", "11.4.gamma", "11.2.lookup_k", "B.1.2.side_wall"} {
		if !strings.Contains(strings.Join(r.FormulaIDs, ";"), id) {
			t.Fatalf("нет формулы %s", id)
		}
	}
	if r.DataSHA256 == "" {
		t.Fatal("missing SHA")
	}
}
func TestWind057EquivalentHeightBranches(t *testing.T) {
	windTestData(t)
	base := windDefaultInput("III")
	base.Height = 60
	base.Depth = 20
	base.Z = 35
	base.PulseMode = "mean"
	c, e := CalculateWind(base)
	if e != nil {
		t.Fatal(e)
	}
	windNear(t, "high middle", c.Result.Ze, 35)
	base.Z = 4
	c, e = CalculateWind(base)
	if e != nil {
		t.Fatal(e)
	}
	windNear(t, "high lower", c.Result.Ze, 20)
	base.Z = 50
	c, e = CalculateWind(base)
	if e != nil {
		t.Fatal(e)
	}
	windNear(t, "high upper", c.Result.Ze, 60)
	base.Height = 25
	base.Depth = 20
	base.Z = 3
	c, e = CalculateWind(base)
	if e != nil {
		t.Fatal(e)
	}
	windNear(t, "mid lower", c.Result.Ze, 20)
	base.Z = 9
	c, e = CalculateWind(base)
	if e != nil {
		t.Fatal(e)
	}
	windNear(t, "mid upper", c.Result.Ze, 25)
	base.Height = 15
	base.Depth = 20
	base.Z = 4
	c, e = CalculateWind(base)
	if e != nil {
		t.Fatal(e)
	}
	windNear(t, "low", c.Result.Ze, 15)
	base.Geometry = "tower"
	base.Height = 60
	base.Z = 13
	c, e = CalculateWind(base)
	if e != nil {
		t.Fatal(e)
	}
	windNear(t, "tower", c.Result.Ze, 13)
}
func TestWind057AeroReferenceAndUserOverride(t *testing.T) {
	windTestData(t)
	b := windDefaultInput("III")
	b.PulseMode = "mean"
	for _, tcase := range []struct {
		table, zone string
		want        float64
	}{{"В.1", "A", 2.1}, {"В.1", "D", 1.2}, {"В.2", "A", -1}, {"В.2", "B", -.8}, {"В.2", "D", .8}, {"В.2", "E", -.5}} {
		b.AeroTable = tcase.table
		b.AeroZone = tcase.zone
		c, e := CalculateWind(b)
		if e != nil {
			t.Fatal(e)
		}
		windNear(t, tcase.table+":"+tcase.zone, c.Result.C, tcase.want)
	}
	b.AeroTable = "manual"
	b.ManualC = -.73
	b.ManualW0 = true
	b.W0KPa = .51
	c, e := CalculateWind(b)
	if e != nil {
		t.Fatal(e)
	}
	windNear(t, "manual c", c.Result.C, -.73)
	windNear(t, "manual w0", c.Result.W0Pa, 510)
	if !strings.Contains(strings.Join(c.Result.Warnings, ";"), "вручную") {
		t.Fatal("manual source must warn")
	}
}
func TestWind057ExpressionXiFromJSON(t *testing.T) {
	windTestData(t)
	db, e := getWindDB()
	if e != nil {
		t.Fatal(e)
	}
	xi, e := db.calcExpr("11.1.xi_davenport_approx", map[string]float64{"Tg1": .2, "delta_damp": .18})
	if e != nil {
		t.Fatal(e)
	}
	windNear(t, "xi", xi, 2.6329047496374844)
	b := windDefaultInput("III")
	b.PulseMode = "dynamic"
	b.Frequency = 1.2
	b.Delta = .18
	c, e := CalculateWind(b)
	if e != nil {
		t.Fatal(e)
	}
	if c.Result.Xi <= 1 || c.Result.PulsePa == 0 {
		t.Fatalf("bad xi and pulse %+v", c.Result)
	}
	if c.Result.FLim != 0 {
		t.Fatal("non-tabulated delta must not fabricate Tg_lim")
	}
	if !strings.Contains(strings.Join(c.Result.Warnings, ";"), "табл. 11.5") {
		t.Fatal("no warning for non-tabulated delta")
	}
}
func TestWind057ManifestRejectsModifiedJSON(t *testing.T) {
	windTestData(t)
	src, e := filepath.Abs("WindData")
	if e != nil {
		t.Fatal(e)
	}
	dir := t.TempDir()
	for _, p := range []string{"MANIFEST_SHA256.json", "formulas.json", "tables_section11.json", "SP20_wind_all_in_one.json"} {
		data, e := os.ReadFile(filepath.Join(src, p))
		if e != nil {
			t.Fatal(e)
		}
		if p == "tables_section11.json" {
			data = append(data, ' ')
		}
		if e = os.WriteFile(filepath.Join(dir, p), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := loadWindDB(dir); e == nil || !strings.Contains(e.Error(), "контрольная сумма") {
		t.Fatalf("ожидалась ошибка контрольной суммы: %v", e)
	}
}
func TestWind057ProjectStableNamedBlocks(t *testing.T) {
	c := windSample(t)
	p := initialProject()
	id := p.putWindCalc(0, c)
	if id <= 0 || len(p.SavedBlocks) != 1 {
		t.Fatalf("no block %v", p.SavedBlocks)
	}
	c.Input.Height = 40
	r, e := CalculateWind(c.Input)
	if e != nil {
		t.Fatal(e)
	}
	id2 := p.putWindCalc(id, r)
	if id != id2 || len(p.SavedBlocks) != 1 {
		t.Fatal("overwrite created duplicate")
	}
	file := filepath.Join(t.TempDir(), "load.json")
	if e := p.Save(file); e != nil {
		t.Fatal(e)
	}
	q, e := LoadProject(file)
	if e != nil {
		t.Fatal(e)
	}
	if len(q.SavedBlocks) != 1 || q.SavedBlocks[0].WindCalc == nil {
		t.Fatal("lost wind snapshot")
	}
	windNear(t, "reopened", q.SavedBlocks[0].WindCalc.Result.K, r.Result.K)
}
func TestWind057WordAndIndependentExcel(t *testing.T) {
	c := windSample(t)
	dir := t.TempDir()
	if specified := os.Getenv("EBALA_WIND_QA_OUT"); specified != "" {
		dir = specified
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	report := TempReport{Version: 1, Winds: []WindCalculation{c}, WindTitles: []string{"Контрольный фасад"}}
	for _, tc := range []struct {
		name string
		f    func(TempReport, string) error
	}{{"wind.docx", exportTempWord}, {"wind.xlsx", exportTempExcel}} {
		p := filepath.Join(dir, tc.name)
		if e := tc.f(report, p); e != nil {
			t.Fatalf("%s: %v", tc.name, e)
		}
		z, e := zip.OpenReader(p)
		if e != nil {
			t.Fatal(e)
		}
		sheets := 0
		hasForm := false
		hasNorm := false
		for _, f := range z.File {
			if !strings.HasSuffix(f.Name, ".xml") {
				continue
			}
			r, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			data := make([]byte, f.UncompressedSize64)
			_, _ = r.Read(data)
			r.Close()
			dec := xml.NewDecoder(strings.NewReader(string(data)))
			for {
				if _, e := dec.Token(); e != nil {
					if e.Error() != "EOF" {
						t.Fatalf("%s invalid XML %s: %v", tc.name, f.Name, e)
					}
					break
				}
			}
			if strings.HasPrefix(f.Name, "xl/worksheets/") {
				sheets++
			}
			if strings.Contains(string(data), "Ветер Исходные 1") && strings.Contains(string(data), "Ветер Расчёт 1") {
				hasNorm = true
			}
			if strings.Contains(string(data), "VLOOKUP") && strings.Contains(string(data), "INDEX") && strings.Contains(string(data), "MATCH") {
				hasForm = true
			}
			if strings.Contains(string(data), "#REF!") {
				t.Fatalf("%s #REF!", tc.name)
			}
			if strings.Contains(f.Name, "document.xml") && !strings.Contains(string(data), "0,451") {
				t.Fatal("Word lacks core result")
			}
		}
		z.Close()
		if tc.name == "wind.xlsx" && (!hasForm || !hasNorm || sheets < 9) {
			t.Fatalf("no formula/sources, sheets=%d hasForm=%v hasNorm=%v", sheets, hasForm, hasNorm)
		}
	}
}
