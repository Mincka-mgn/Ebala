package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test040MountainSnowFeedsProject(t *testing.T) {
	s := Site{Region: "Краснодарский край", SnowZone: "VII", MountainID: "krasnodar_adler", ElevationM: 1200, ElevationKnown: true}
	base, high, m, on, err := s.ElevationDetails()
	if err != nil || !on || m.ID != "krasnodar_adler" {
		t.Fatalf("mountain E1: %v on=%v %#v", err, on, m)
	}
	if math.Abs(high-base*(1+.001*m.Kh*(1200-m.H0))) > 1e-10 {
		t.Fatalf("mountain E1 %.8f %.8f", base, high)
	}
	p := Project{Site: s}
	zone, ground, override := p.SnowInputs()
	if zone != "VII" || override || math.Abs(ground-high) > 1e-10 {
		t.Fatalf("SnowInputs: %s %g %v, expected %.3f", zone, ground, override, high)
	}
	s.MountainID = ""
	p.SetSite(s)
	_, noMountain, _ := p.SnowInputs()
	if noMountain == high {
		t.Fatal("mountain correction remained after deselection")
	}
}

func Test040ExcelNativeChartAndFormulaReferences(t *testing.T) {
	for _, shape := range []string{"round_wire", "surface_small"} {
		s := defaultIceLoad()
		s.Region = "III"
		s.BaseMM = 10
		s.Shape = shape
		s.DiameterMM = 70
		s.HeightStart = 5
		s.HeightEnd = 300
		s.HeightStep = 25
		s.Wind = IceWindInput{Enabled: true, Zone: "III", Terrain: "A", Drag: 1.2}
		if shape == "surface_small" {
			s.DiameterMM = 0
		}
		out, err := IceCompute(s)
		if err != nil {
			t.Fatal(err)
		}
		tmp := filepath.Join(t.TempDir(), shape+".xlsx")
		if err := ExportIceExcel(out, tmp); err != nil {
			t.Fatal(err)
		}
		z, err := zip.OpenReader(tmp)
		if err != nil {
			t.Fatal(err)
		}
		var chart, inputs, calc string
		for _, f := range z.File {
			r, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			b := make([]byte, f.UncompressedSize64)
			n, _ := r.Read(b)
			r.Close()
			if n != len(b) { // zip reader can return partial reads
				r, e = f.Open()
				if e != nil {
					t.Fatal(e)
				}
				data := make([]byte, 0, int(f.UncompressedSize64))
				buf := make([]byte, 4096)
				for {
					k, x := r.Read(buf)
					data = append(data, buf[:k]...)
					if x != nil {
						break
					}
				}
				r.Close()
				b = data
			}
			if strings.HasSuffix(f.Name, ".xml") || strings.HasSuffix(f.Name, ".rels") {
				var target interface{}
				if e := xml.Unmarshal(b, &target); e != nil { // XML input is structurally checked below by strict decoder instead
					dec := xml.NewDecoder(strings.NewReader(string(b)))
					for {
						_, z := dec.Token()
						if z != nil {
							if z.Error() != "EOF" {
								t.Fatalf("bad XML in %s: %s", f.Name, z)
							}
							break
						}
					}
				}
			}
			if strings.HasPrefix(f.Name, "xl/charts/chart") {
				chart = string(b)
			}
			if strings.Contains(f.Name, "snow2.xml") {
				inputs = string(b)
			}
			if strings.Contains(f.Name, "snow3.xml") {
				calc = string(b)
			}
		}
		z.Close()
		if !strings.Contains(chart, "<c:scatterChart>") || !strings.Contains(chart, "Гололёд - расчёт") {
			t.Fatal("No editable XY chart linked to calculation cells")
		}
		if strings.Contains(inputs, "мм / μ1") || !strings.Contains(inputs, "Диаметр провода или троса") {
			t.Fatal("Unexplained diameter table")
		}
		if !strings.Contains(calc, "Гололёд - исходные") || !strings.Contains(calc, "Гололёд нормативный") {
			t.Fatal("Calculation formula references or headings missing")
		}
		if shape == "round_wire" {
			if !strings.Contains(inputs, "IF(B6&lt;=A20") {
				t.Fatal("Mu1 interpolation should guard lower boundary")
			}
			for _, p := range out.Points {
				if p.Mu1 != .6 {
					t.Fatalf("d=70 should be 0.6, got %g", p.Mu1)
				}
			}
		}
	}
}

func Test040WindOptionalAndManualDrag(t *testing.T) {
	s := defaultIceLoad()
	s.Region = "III"
	s.BaseMM = 10
	s.Shape = "round_wire"
	s.DiameterMM = 20
	s.HeightStart = 10
	s.HeightEnd = 10
	noWind, e := IceCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if noWind.Points[0].WindNorm != 0 {
		t.Fatal("wind added without requested zone")
	}
	s.Wind = IceWindInput{Enabled: true, Zone: "III", Terrain: "A", Drag: 1.2}
	withWind, e := IceCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	s.Wind.Drag = 1.5
	manual, e := IceCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if math.Abs(manual.Points[0].WindNorm/withWind.Points[0].WindNorm-1.25) > 1e-10 {
		t.Fatal("Manual Cx did not propagate")
	}
	if math.Abs(manual.Points[0].Norm-noWind.Points[0].Norm) > 1e-10 {
		t.Fatal("Changing Cx changed ice weight")
	}
}

func Test040ArtifactsForOffice(t *testing.T) {
	dir := os.Getenv("EBALA040_QA_DIR")
	if dir == "" {
		t.Skip("QA export not requested")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, d := range []float64{5, 20, 70} {
		s := defaultIceLoad()
		s.Region = "III"
		s.BaseMM = 10
		s.Shape = "round_wire"
		s.DiameterMM = d
		s.HeightStart = 5
		s.HeightEnd = 300
		s.HeightStep = 5
		s.Wind = IceWindInput{Enabled: true, Zone: "II", Terrain: "B", Drag: 1.2, PulseVerified: true, PulseRatio: .32, PulseSource: "Контрольный независимый расчёт"}
		out, err := IceCompute(s)
		if err != nil {
			t.Fatal(err)
		}
		name := fmt.Sprintf("ice_d%.0f", d)
		if err = ExportIceExcel(out, filepath.Join(dir, name+".xlsx")); err != nil {
			t.Fatal(err)
		}
		if err = ExportIceWord(out, filepath.Join(dir, name+".docx")); err != nil {
			t.Fatal(err)
		}
		var csv strings.Builder
		csv.WriteString("h;norm;design;wind_norm;wind_design;wind_leading\n")
		for _, p := range out.Points {
			fmt.Fprintf(&csv, "%.9f;%.9f;%.9f;%.9f;%.9f;%.9f\n", p.HeightM, p.Norm, p.Design, p.WindNorm, p.WindDesign, p.WindLeading)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".csv"), []byte(csv.String()), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func Test040CombinedSnowIceExport(t *testing.T) {
	snow := defaultSnowCalc()
	snow.Shape = "gable"
	snow.Region = "III"
	snow.Ground = 1.5
	s, err := SnowCompute(snow)
	if err != nil {
		t.Fatal(err)
	}
	ice := defaultIceLoad()
	ice.Region = "III"
	ice.BaseMM = 10
	ice.Shape = "round_wire"
	ice.HeightStart = 5
	ice.HeightEnd = 80
	ice.Wind = IceWindInput{Enabled: true, Zone: "III", Terrain: "A", Drag: 1.2}
	i, err := IceCompute(ice)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "combined.xlsx")
	if err := exportTempExcel(TempReport{Version: 1, Snow: &s, Ice: &i}, path); err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	seen := map[string]bool{}
	for _, f := range z.File {
		if seen[f.Name] {
			t.Fatalf("duplicate zip path %s", f.Name)
		}
		seen[f.Name] = true
	}
	chartCount := 0
	drawCount := 0
	for n := range seen {
		if strings.HasPrefix(n, "xl/charts/chart") {
			chartCount++
		}
		if strings.HasPrefix(n, "xl/drawings/drawing") && strings.HasSuffix(n, ".xml") {
			drawCount++
		}
	}
	if chartCount != 1 || drawCount < 2 {
		t.Fatalf("combined chart/drawings: chart %d, drawing %d", chartCount, drawCount)
	}
	if !seen["xl/workbook.xml"] || !seen["xl/_rels/workbook.xml.rels"] {
		t.Fatal("missing common workbook")
	}
	if dir := os.Getenv("EBALA040_QA_DIR"); dir != "" {
		b, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(filepath.Join(dir, "combined.xlsx"), b, 0644); e != nil {
			t.Fatal(e)
		}
	}
}
