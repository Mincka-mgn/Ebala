package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type excelPhase11GoldCase struct {
	File   string             `json:"file"`
	Shape  string             `json:"shape"`
	Span   float64            `json:"span"`
	Angle  float64            `json:"angle"`
	Ground float64            `json:"ground"`
	Gamma  float64            `json:"gamma"`
	Cases  []excelPhase11Case `json:"cases"`
}
type excelPhase11Case struct {
	Name      string  `json:"name"`
	PeakMu    float64 `json:"peak_mu"`
	Intervals int     `json:"intervals"`
}

func TestExcelPhase11Fixtures(t *testing.T) {
	dir := os.Getenv("EBALA_QA_DIR")
	if dir == "" {
		t.Skip("контрольные книги только при EBALA_QA_DIR")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	result := []excelPhase11GoldCase{}
	for _, test := range []struct {
		file, shape string
		angle       float64
	}{
		{"gable_20.xlsx", "gable", 20},
		{"gable_40.xlsx", "gable", 40},
		{"skylight_0.xlsx", "skylight", 0},
	} {
		s := defaultSnowCalc()
		s.Shape = test.shape
		s.Angle = test.angle
		s.Span = 18
		s.SkylightWidth = 6
		calc, err := SnowCompute(s)
		if err != nil {
			t.Fatal(err)
		}
		if err := ExportSnowExcel(calc, filepath.Join(dir, test.file)); err != nil {
			t.Fatal(err)
		}
		ref := excelPhase11GoldCase{File: test.file, Shape: test.shape, Span: s.Span, Angle: s.Angle, Ground: s.Ground, Gamma: s.Gamma}
		for _, c := range calc.Cases {
			peak := 0.0
			for _, v := range c.Intervals {
				if v.Mu > peak {
					peak = v.Mu
				}
			}
			ref.Cases = append(ref.Cases, excelPhase11Case{Name: c.Title, PeakMu: peak, Intervals: len(c.Intervals)})
		}
		result = append(result, ref)
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go_snow_reference.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	single, err := CalculateCranePlacements([]float64{6000}, []CraneTrain{{Name: "Кран А", WheelXMM: []float64{0}, ForcesKN: []float64{100}}})
	if err != nil {
		t.Fatal(err)
	}
	singleReport := TempReport{Version: 1, CraneGroups: []CraneReportGroup{{Title: "Один пролёт", Cranes: []TempSelection{{Group: "Краны", Title: "Кран А", Wheels: []TempWheel{{PKN: 100, XMM: 0, XKnown: true}}}}, Placements: []CraneReportPlacement{{Title: "Балка 6 м", Result: single}}}}}
	if err := exportTempExcel(singleReport, filepath.Join(dir, "crane_1span.xlsx")); err != nil {
		t.Fatal(err)
	}

	placement, err := CalculateCranePlacements([]float64{6000, 6000}, []CraneTrain{{Name: "Кран А", WheelXMM: []float64{0}, ForcesKN: []float64{100}}})
	if err != nil {
		t.Fatal(err)
	}
	general := TempReport{Version: 1, CraneGroups: []CraneReportGroup{{Title: "Два пролёта", Cranes: []TempSelection{{Group: "Краны", Title: "Кран А", Wheels: []TempWheel{{PKN: 100, XMM: 0, XKnown: true}}}}, Placements: []CraneReportPlacement{{Title: "Балка 6+6 м", Result: placement}}}}}
	if err := exportTempExcel(general, filepath.Join(dir, "crane_2span.xlsx")); err != nil {
		t.Fatal(err)
	}
	type craneRef struct {
		Name  string  `json:"name"`
		Value float64 `json:"value"`
		Units string  `json:"units"`
	}
	craneValues := []craneRef{}
	for _, c := range placement.Cases {
		craneValues = append(craneValues, craneRef{c.Name, c.Value, c.Units})
	}
	craneData, err := json.MarshalIndent(craneValues, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go_crane_reference.json"), craneData, 0644); err != nil {
		t.Fatal(err)
	}
}
