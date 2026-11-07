package main

import (
	"archive/zip"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestWind060RoofNormativeCoefficientBranches(t *testing.T) {
	windTestData(t)
	db, e := getWindDB()
	if e != nil {
		t.Fatal(e)
	}
	cases := []struct {
		table, zone string
		beta        float64
		alternative int
		want        float64
	}{
		{"В.3а", "F", 0, 1, -1.8},
		{"В.3а", "F", 15, 1, -.9},
		{"В.3а", "F", 15, 2, .2},
		{"В.3а", "H", 30, 1, -.2},
		{"В.3а", "H", 30, 2, .4},
		{"В.3а", "F", 22.5, 1, -.7},
		{"В.3а", "F", 22.5, 2, .45},
		{"В.3а", "H", 37.5, 1, .5}, // from positive 30° to single 45°
		{"В.3а", "I", 45, 1, -.2},
		{"В.3а", "J", 60, 1, -.3},
		{"В.3б", "G", 15, 1, -1.3},
		{"В.3б", "G", 22.5, 1, -1.35},
		{"В.3б", "I", 75, 1, -.5},
	}
	for _, tc := range cases {
		got, e := db.roofCoefficient(tc.table, tc.zone, tc.beta, tc.alternative)
		if e != nil {
			t.Fatalf("%+v: %v", tc, e)
		}
		if math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("%s %s beta %.1f alt %d: %.9f != %.9f", tc.table, tc.zone, tc.beta, tc.alternative, got, tc.want)
		}
	}
	if _, e := db.roofCoefficient("В.3а", "F", 15, 0); e == nil {
		t.Fatal("missing sign branch silently accepted")
	}
	if _, e := db.roofCoefficient("В.3а", "F", 76, 1); e == nil {
		t.Fatal("unsupported extrapolation silently accepted")
	}
}

func testRoofCaseBase(beta float64) WindInput {
	in := windDefaultInput("III")
	in.Geometry = "building"
	in.Height = 20
	in.Depth = 30 // B, ridge length
	in.Rho = 12   // L, span
	in.RoofRise = in.Rho / 2 * math.Tan(beta*math.Pi/180)
	in.PulseMode = "nonresonant"
	return in
}
func TestWind060RoofDirectionsAndIndependentSignAlternatives(t *testing.T) {
	windTestData(t)
	c, e := CalculateWindVariants(testRoofCaseBase(20))
	if e != nil {
		t.Fatal(e)
	}
	if len(c.Variants) != 2 || c.Variants[0].ID != "RIDGE" || c.Variants[1].ID != "CROSS" {
		t.Fatalf("directions %+v", c.Variants)
	}
	var counts = map[string]int{}
	var roofByAlt = map[string]map[string]WindZoneCalculation{}
	for _, v := range c.Variants {
		for _, z := range v.Zones {
			key := v.ID + ":" + z.Alternative
			if z.Surface == "roof" {
				counts[key]++
				if roofByAlt[key] == nil {
					roofByAlt[key] = map[string]WindZoneCalculation{}
				}
				roofByAlt[key][z.Code] = z
				if len(z.Rectangles) == 0 {
					t.Errorf("no roof footprint %s/%s", v.ID, z.Code)
				}
				if z.Calculation.Input.RoofSpan != 12 || z.Calculation.Input.RoofLength != 30 {
					t.Error("source dimensions lost")
				}
				if math.Abs(z.Calculation.Input.RoofPitchDeg-20) > 1e-8 {
					t.Error("wrong pitch")
				}
				if z.Calculation.Result.DataSHA256 == "" {
					t.Error("no DB checksum")
				}
				if z.Calculation.Result.NormPa != z.Calculation.Result.MeanPa+z.Calculation.Result.PulsePa {
					t.Error("bad load sum")
				}
				if math.Abs(z.Calculation.Result.MeanPa-z.Calculation.Result.W0Pa*z.Calculation.Result.K*z.Calculation.Result.C) > 1e-8 {
					t.Error("bad mean load")
				}
			}
		}
	}
	if counts["RIDGE:1"] != 4 || counts["CROSS:1"] != 5 || counts["CROSS:2"] != 5 {
		t.Fatalf("wrong case zones: %v", counts)
	}
	for _, zone := range []string{"F", "G", "H", "I", "J"} {
		a, b := roofByAlt["CROSS:1"][zone], roofByAlt["CROSS:2"][zone]
		if (zone == "F" || zone == "G" || zone == "H") && math.Abs(a.Calculation.Result.C-b.Calculation.Result.C) < 1e-8 {
			t.Error("alternatives merged: " + zone)
		}
		if (zone == "I" || zone == "J") && math.Abs(a.Calculation.Result.C-b.Calculation.Result.C) > 1e-8 {
			t.Error("common zones change between alternatives: " + zone)
		}
	}
	b, e := json.Marshal(c)
	if e != nil {
		t.Fatal(e)
	}
	var decoded WindCalculation
	if e := json.Unmarshal(b, &decoded); e != nil {
		t.Fatal(e)
	}
	if len(decoded.Variants) != 2 || len(decoded.Variants[1].Zones) != len(c.Variants[1].Zones) {
		t.Fatal("roof zone loss at project save/load")
	}
}
func TestWind060RoofZoneFootprintsPartitionWholePlan(t *testing.T) {
	windTestData(t)
	for _, ratio := range []struct{ B, L, H, beta float64 }{
		{30, 12, 20, 20}, {9, 40, 40, 45}, {50, 8, 12, 5}, {5, 80, 100, 25},
	} {
		in := testRoofCaseBase(ratio.beta)
		in.Depth, in.Rho, in.Height = ratio.B, ratio.L, ratio.H
		in.RoofRise = ratio.L / 2 * math.Tan(ratio.beta*math.Pi/180)
		c, e := CalculateWindVariants(in)
		if e != nil {
			t.Fatal(e)
		}
		for _, v := range c.Variants {
			alt := []string{"1"}
			if v.ID == "CROSS" && ratio.beta >= 15 && ratio.beta <= 30 {
				alt = []string{"1", "2"}
			}
			for _, a := range alt {
				for xi := 0; xi < 40; xi++ {
					for yi := 0; yi < 40; yi++ {
						x := v.AlongM * (float64(xi) + 0.5) / 40
						y := v.AcrossM * (float64(yi) + 0.5) / 40
						seen := 0
						for _, z := range v.Zones {
							if z.Surface != "roof" || z.Alternative != a {
								continue
							}
							for _, r := range z.Rectangles {
								if x > r.X0 && x < r.X1 && y > r.Y0 && y < r.Y1 {
									seen++
								}
							}
						}
						if seen != 1 {
							t.Fatalf("plan under-/over-covered: B %.1f L %.1f H %.1f beta %.1f, %s case %s at (%.4f,%.4f): count=%d", ratio.B, ratio.L, ratio.H, ratio.beta, v.ID, a, x, y, seen)
						}
					}
				}
			}
		}
	}
}
func TestWind060RoofXLSXFormulaIndependence(t *testing.T) {
	windTestData(t)
	c, e := CalculateWindVariants(testRoofCaseBase(20))
	if e != nil {
		t.Fatal(e)
	}
	entries := map[string]string{
		"xl/workbook.xml":            `<workbook><sheets></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships></Relationships>`,
		"[Content_Types].xml":        `<Types></Types>`,
	}
	if e := windExcelAdd(entries, []WindCalculation{c}, []string{"Проверочная двускатная крыша"}); e != nil {
		t.Fatal(e)
	}
	for _, sheet := range []string{"Ветер В.3а 1", "Ветер В.3а 2", "Ветер В.3б"} {
		if !strings.Contains(entries["xl/workbook.xml"], sheet) {
			t.Fatalf("missing normative sheet %s", sheet)
		}
	}
	all := strings.Join(mapValues(entries), "\n")
	if !strings.Contains(all, "DEGREES(ATAN(2*") {
		t.Fatal("pitch is not calculated by Excel")
	}
	for _, part := range []string{"Ветер В.3а 1", "Ветер В.3а 2", "Ветер В.3б", "B46", "B47", "MATCH(", "INDEX("} {
		if !strings.Contains(all, part) {
			t.Fatalf("missing formula dependency %s", part)
		}
	}
	// No precomputed Go c in coefficient cells of roof calculation sheets.
	for _, content := range entries {
		if strings.Contains(content, "Аэродинамический коэффициент c") && strings.Contains(content, "В.3") {
			if !strings.Contains(content, "<f>IF(") {
				t.Fatal("coefficient is cached instead of formula")
			}
		}
	}
	// Structural check with an actual workbook emitted by the Ebala writer.
	dir := t.TempDir()
	path := filepath.Join(dir, "roof.xlsx")
	if e := exportTempExcel(TempReport{Version: 1, Winds: []WindCalculation{c}, WindTitles: []string{"Тест"}}, path); e != nil {
		t.Fatal(e)
	}
	z, e := zip.OpenReader(path)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	if len(z.File) < 12 {
		t.Fatal("suspiciously small Excel package")
	}
	for _, f := range z.File {
		if strings.HasSuffix(f.Name, ".xml") && f.UncompressedSize64 == 0 {
			t.Fatalf("empty XML %s", f.Name)
		}
	}
	_ = os.Remove(path)
}
func mapKeys(m map[string]string) []string {
	xs := make([]string, 0, len(m))
	for k := range m {
		xs = append(xs, k)
	}
	return xs
}
func mapValues(m map[string]string) []string {
	xs := make([]string, 0, len(m))
	for _, v := range m {
		xs = append(xs, v)
	}
	return xs
}

var _ = regexp.MustCompile
var _ = strconv.Atoi

func TestWind060RoofWordRetainsSourcesAndBranches(t *testing.T) {
	windTestData(t)
	c, e := CalculateWindVariants(testRoofCaseBase(20))
	if e != nil {
		t.Fatal(e)
	}
	counter := 0
	rel := &strings.Builder{}
	entries := map[string]string{}
	section, err := windWordGroup(c, "Здание", &counter, rel, entries)
	if err != nil {
		t.Fatal(err)
	}
	for _, term := range []string{"В.3а", "Знаковый вариант 2", "Зона F", "Зона G", "Отметка, м"} {
		if !strings.Contains(section, term) {
			t.Errorf("DOCX lacks %q", term)
		}
	}
	if len(entries) < 2 {
		t.Fatal("Word must contain wall and roof schemes")
	}

}

// Optional acceptance artifacts. This test stays skipped in normal CI;
// EBALA_WIND_QA_OUT=/path GO111MODULE=off go test -run TestWind060QAExport
// produces independently inspectable DOCX/XLSX for the two-direction roof
// and the complete sign alternatives.
func TestWind060QAExport(t *testing.T) {
	folder := os.Getenv("EBALA_WIND_QA_OUT")
	if folder == "" {
		t.Skip("Set EBALA_WIND_QA_OUT for export samples")
	}
	windTestData(t)
	if e := os.MkdirAll(folder, 0755); e != nil {
		t.Fatal(e)
	}
	calc, e := CalculateWindVariants(testRoofCaseBase(20))
	if e != nil {
		t.Fatal(e)
	}
	rpt := TempReport{Version: 1, Winds: []WindCalculation{calc}, WindTitles: []string{"Проверочная двускатная кровля"}}
	if e := exportTempExcel(rpt, filepath.Join(folder, "Wind_Roof_20deg.xlsx")); e != nil {
		t.Fatal(e)
	}
	if e := exportTempWord(rpt, filepath.Join(folder, "Wind_Roof_20deg.docx")); e != nil {
		t.Fatal(e)
	}
}
