package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWind059DefaultGrid(t *testing.T) {
	cases := []struct {
		H    float64
		want []float64
	}{
		{3, []float64{0, 3}},
		{5, []float64{0, 5}},
		{10, []float64{0, 5, 7.5, 10}},
		{11, []float64{0, 5, 7.5, 10, 11}},
	}
	for _, v := range cases {
		xs, e := WindProfileHeights(v.H, 0)
		if e != nil {
			t.Fatal(e)
		}
		if len(xs) != len(v.want) {
			t.Fatalf("H=%g: %v", v.H, xs)
		}
		for i, x := range xs {
			windNear(t, "default height", x, v.want[i])
		}
	}
}
func TestWind059CustomGridMandatoryPoints(t *testing.T) {
	cases := []struct {
		H, step float64
		want    []float64
	}{
		{10, 3, []float64{0, 3, 5, 8, 10}},
		{4, 3, []float64{0, 3, 4}},
		{6, 1.2, []float64{0, 1.2, 2.4, 3.6, 4.8, 5, 6}},
	}
	for _, v := range cases {
		xs, e := WindProfileHeights(v.H, v.step)
		if e != nil {
			t.Fatal(e)
		}
		if len(xs) != len(v.want) {
			t.Fatalf("%g/%g: %v", v.H, v.step, xs)
		}
		for i, x := range xs {
			if math.Abs(x-v.want[i]) > 1e-8 {
				t.Fatalf("%g/%g: %v", v.H, v.step, xs)
			}
			if i > 0 && x-xs[i-1] > v.step+1e-9 {
				t.Fatalf("interval too large: %v", xs)
			}
		}
	}
	if _, e := WindProfileHeights(100, .000001); e == nil {
		t.Fatal("large grid must be rejected")
	}
}
func TestWind059BuildingBothWindDirectionsAndAllApplicableZones(t *testing.T) {
	windTestData(t)
	in := windDefaultInput("III")
	in.Height = 20
	in.Depth = 30
	in.Rho = 10
	in.RoofRise = 4
	in.ProfileStep = 3
	c, e := CalculateWindVariants(in)
	if e != nil {
		t.Fatal(e)
	}
	if len(c.Variants) != 2 {
		t.Fatalf("variants %d", len(c.Variants))
	}
	if c.Variants[0].Name == c.Variants[1].Name {
		t.Fatal("same wind direction")
	}
	v := c.Variants[0]
	windNear(t, "e=min(b,2H)", v.E, 10)
	if len(v.Zones) < 5 {
		t.Fatalf("missing building wall zones: %d", len(v.Zones))
	}
	var found map[string]WindZoneCalculation = map[string]WindZoneCalculation{}
	for _, z := range v.Zones {
		found[z.Code] = z
		if len(z.Calculation.Profile) < 2 {
			t.Fatal("missing profile")
		}
		for _, p := range z.Calculation.Profile {
			cc := z.Calculation.Input
			cc.Z = p.Z
			calc, e := CalculateWind(cc)
			if e != nil {
				t.Fatal(e)
			}
			windNear(t, "mean core", p.Result.MeanPa, calc.Result.MeanPa)
		}
	}
	windNear(t, "A width", found["A"].To-found["A"].From, 2)
	windNear(t, "B width", found["B"].To-found["B"].From, 8)
	windNear(t, "C width", found["C"].To-found["C"].From, 20)
	// Вторая ориентация использует иные поперечный и продольный размеры.
	if c.Variants[1].AlongM != 10 || c.Variants[1].AcrossM != 30 {
		t.Fatal("rotated dimensions")
	}
	if len(c.Variants[1].Zones) < 4 {
		t.Fatalf("small face must include its wall zones, zones=%d", len(c.Variants[1].Zones))
	}
	for _, zone := range c.Variants[1].Zones {
		if zone.Surface != "roof" && zone.Code == "C" {
			t.Fatal("must not fabricate empty wall C")
		}
		if zone.Calculation.Input.Depth != 30 {
			t.Fatal("wrong transverse width in z_e")
		}
	}
	for _, zone := range c.Variants[0].Zones {
		if zone.Calculation.Input.Depth != 10 {
			t.Fatal("wrong transverse width in z_e")
		}
	}
	// The saved object must round-trip with all variants and their independent results.
	raw, e := json.Marshal(c)
	if e != nil {
		t.Fatal(e)
	}
	var saved WindCalculation
	if e = json.Unmarshal(raw, &saved); e != nil {
		t.Fatal(e)
	}
	if len(saved.Variants) != 2 || len(saved.Variants[0].Zones) < 5 {
		t.Fatal("variant loss in JSON")
	}
}
func baseAlongMinus(a, b float64) float64 { return a - b }

func TestWind059StandaloneWallZonesBoundedByPhysicalLength(t *testing.T) {
	windTestData(t)
	in := windDefaultInput("III")
	in.Geometry = "free_wall"
	in.Depth = 0
	in.Height = 10
	in.Rho = 90
	in.PulseMode = "mean"
	c, e := CalculateWindVariants(in)
	if e != nil {
		t.Fatal(e)
	}
	if len(c.Variants) != 1 || len(c.Variants[0].Zones) != 4 {
		t.Fatalf("wall zones: %+v", c.Variants)
	}
	zones := c.Variants[0].Zones
	for i, x := range []float64{3, 23, 63, 90} {
		windNear(t, "wall zone end", zones[i].To, x)
	}
	for _, z := range zones {
		windNear(t, "wall effective height", z.Calculation.Result.Ze, 10)
	}
}
func TestWind059DiscontinuityAtBuildingUpperRegion(t *testing.T) {
	windTestData(t)
	in := windDefaultInput("III")
	in.Height = 100
	in.Depth = 30
	in.Rho = 15
	c, e := CalculateWindProfile(in)
	if e != nil {
		t.Fatal(e)
	}
	var below, above *WindProfilePoint
	for j := range c.Profile {
		p := &c.Profile[j]
		if p.Z == 70 {
			above = p
		}
		if p.Z < 70 && p.Z > 69.999999 {
			below = p
		}
	}
	if below == nil || above == nil {
		t.Fatal("missing jump border and left limit")
	}
	if math.Abs(above.Result.NormPa-below.Result.NormPa) < 1e-6 {
		t.Fatal("unjustified smoothing of a real regime transition")
	}
}
func TestWind059AllZonesExportIndependentExcelAndWord(t *testing.T) {
	windTestData(t)
	in := windDefaultInput("VII")
	in.Height = 15
	in.Depth = 20
	in.Rho = 10
	in.ProfileStep = 3
	c, e := CalculateWindVariants(in)
	if e != nil {
		t.Fatal(e)
	}
	detailed, names := windExpandReports([]WindCalculation{c}, []string{"Промышленный корпус"})
	expected := 0
	for _, v := range c.Variants {
		expected += len(v.Zones)
	}
	if len(detailed) != expected || len(names) != expected || expected < 8 {
		t.Fatalf("wrong report expansion: %d %d", len(detailed), expected)
	}
	dir := t.TempDir()
	xlsx := filepath.Join(dir, "wind059.xlsx")
	if e = exportTempExcel(TempReport{Version: 1, Winds: []WindCalculation{c}}, xlsx); e != nil {
		t.Fatal(e)
	}
	z, e := zip.OpenReader(xlsx)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	sheets := 0
	for _, f := range z.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/") && strings.HasSuffix(f.Name, ".xml") {
			sheets++
		}
		if !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		b, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
		if bytes.Contains(b, []byte("#REF!")) {
			t.Fatalf("invalid ref %s", f.Name)
		}
		decoder := xml.NewDecoder(bytes.NewReader(b))
		for {
			_, e = decoder.Token()
			if e != nil {
				if e != io.EOF {
					t.Fatalf("broken XML %s: %v", f.Name, e)
				}
				break
			}
		}
	}
	if sheets < expected*3+1 {
		t.Fatalf("some zones omitted from formula sheets: %d vs %d", sheets, expected)
	}
	docx := filepath.Join(dir, "wind059.docx")
	if e = exportTempWord(TempReport{Version: 1, Winds: []WindCalculation{c}, WindTitles: []string{"Промышленный корпус"}}, docx); e != nil {
		t.Fatal(e)
	}
	doc, e := zip.OpenReader(docx)
	if e != nil {
		t.Fatal(e)
	}
	defer doc.Close()
	found := 0
	for _, f := range doc.File {
		if f.Name != "word/document.xml" {
			continue
		}
		r, _ := f.Open()
		b, _ := io.ReadAll(r)
		r.Close()
		// One combined five-column table per saved normative zone, including
		// both sign alternatives. Individual engineering fields must NOT be
		// duplicated into many separately titled quasi-calculations.
		found = bytes.Count(b, []byte("Отметка, м"))
		for _, v := range c.Variants {
			if !bytes.Contains(b, []byte(v.Name)) {
				t.Fatalf("missing wind direction %q", v.Name)
			}
		}
	}
	if found != expected {
		t.Fatalf("Word dropped zone tables %d/%d", found, expected)
	}
	if st, e := os.Stat(docx); e != nil || st.Size() < 500 {
		t.Fatal("Word empty")
	}
}
