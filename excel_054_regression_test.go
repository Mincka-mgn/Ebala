package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression: the 0.53 exporter escaped apostrophes before index substitution.
func TestExcel054EscapedMultiLoadReferences(t *testing.T) {
	tests := []struct{ name, source, expected string }{
		{"snow inputs", "<f>&#39;Исходные данные&#39;!B3</f>", "<f>&#39;Исходные данные 14&#39;!B3</f>"},
		{"snow coefficients", "<f>&#39;Коэффициенты&#39;!$B$22</f>", "<f>&#39;Коэффициенты 14&#39;!$B$22</f>"},
		{"sphere geometry", "<f>&#39;Геометрия Б.15а&#39;!$B$5</f>", "<f>&#39;Геометрия Б.15а 14&#39;!$B$5</f>"},
		{"ice grid", "<f>&#39;Гололёд - сетка&#39;!D4</f>", "<f>&#39;Гололёд - сетка 14&#39;!D4</f>"},
		{"plain quotes", "<f>'Границы зон'!B3</f>", "<f>'Границы зон 14'!B3</f>"},
	}
	names := []string{"Исходные данные", "Коэффициенты", "Геометрия Б.15а", "Границы зон", "Гололёд - сетка"}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := excelRenameOwnSheetRefs(tc.source, " 14", names...)
			if got != tc.expected {
				t.Fatalf("got %s; want %s", got, tc.expected)
			}
		})
	}
	input := "<f>&#39;Исходные данные 2&#39;!B3+'Исходные данные 7'!B4+'Временные - исходные'!B3</f>"
	if output := excelRenameOwnSheetRefs(input, " 14", names...); output != input {
		t.Fatalf("переписаны чужие ссылки: %s", output)
	}
	if output := excelRenameOwnSheetRefs("<f>&#39;Коэффициенты&#39;!B3</f>", "", names...); !strings.Contains(output, "&#39;Коэффициенты&#39;!") {
		t.Fatal("отдельный экспорт не должен менять ссылки")
	}
}

// Build real combined books, not just standalone snow/ice books.
// Independently inspect *every* formula with Verification/Phase13/check_workbook_isolation.py.
func TestExcel054MultiLoadFixtures(t *testing.T) {
	dir := os.Getenv("EBALA_QA_DIR")
	if dir == "" {
		t.Skip("создание больших контрольных книг включается EBALA_QA_DIR")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	snow := func(shape string, angle float64, sg float64) SnowCalc {
		t.Helper()
		s := defaultSnowCalc()
		s.Shape, s.Angle, s.Ground = shape, angle, sg
		s.Span, s.Count, s.Rise, s.SkylightWidth, s.H = 18, 3, 5, 6, 2.4
		c, err := SnowCompute(s)
		if err != nil {
			t.Fatalf("SnowCompute(%s): %v", shape, err)
		}
		return c
	}
	ice, err := IceCompute(IceLoadCalc{Region: "III", BaseMM: 10, Shape: "round", DiameterMM: 20, HeightStart: 10, HeightEnd: 70, HeightStep: 10, HighZone: "ordinary"})
	if err != nil {
		t.Fatal(err)
	}
	placement, err := CalculateCranePlacements([]float64{6000}, []CraneTrain{{Name: "Кран А", WheelXMM: []float64{0}, ForcesKN: []float64{100}}})
	if err != nil {
		t.Fatal(err)
	}
	base := TempReport{
		Version:      1,
		WeightBlocks: []ProjectWeightBlock{{Title: "Покрытие", Rows: []ProjectWeightLine{{Name: "Плита", Norm: 2.5, Gamma: 1.1, Design: 2.75}}}},
		Sections: []TempSelection{
			{Group: "СП 20 · временные", Title: "Нагрузка", Source: "таблица 8.3", Params: map[string]interface{}{"normative": 4.0, "gamma_f": 1.2, "normative_reduced": 4.0, "design_reduced": 4.8, "unit": "кПа", "calc_scope": "Полное значение"}},
			{Group: "СП 35 · транспорт", Title: "АК-14", Source: "СП 35", Params: map[string]interface{}{"K": 14.0}},
		},
		CraneGroups: []CraneReportGroup{{Title: "Краны", Cranes: []TempSelection{{Group: "Краны", Title: "Кран А", Wheels: []TempWheel{{PKN: 100, XMM: 0, XKnown: true}}}}, Placements: []CraneReportPlacement{{Title: "Балка 6 м", Result: placement}}}},
		Ices:        []IceLoadCalc{ice}, IceTitles: []string{"Гололёд 1"},
	}
	base.Snows = []SnowCalc{snow("gable", 20, 3.5903), snow("sphere_cone", 18, 4)}
	base.SnowTitles = []string{"Двускатная", "Сфера с конусом"}
	if err := exportTempExcel(base, filepath.Join(dir, "multi_2snow_1ice_weight_crane_transport.xlsx")); err != nil {
		t.Fatal(err)
	}

	stress := TempReport{Version: 1, Ices: []IceLoadCalc{ice}, IceTitles: []string{"Гололёд 1"}}
	kinds := []string{"gable", "multi_gable", "dome", "skylight", "gable", "cone", "gable", "multi_gable", "dome", "skylight", "gable", "cone", "gable", "sphere_cone"}
	for i, kind := range kinds {
		sg := 3.0 + float64(i)/10
		stress.Snows = append(stress.Snows, snow(kind, 20, sg))
		stress.SnowTitles = append(stress.SnowTitles, kind)
	}
	if err := exportTempExcel(stress, filepath.Join(dir, "stress_14snow_1ice.xlsx")); err != nil {
		t.Fatal(err)
	}
}
