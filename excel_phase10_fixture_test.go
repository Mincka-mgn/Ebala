package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExcelPhase10Fixture(t *testing.T) {
	dir := os.Getenv("EBALA_QA_DIR")
	if dir == "" {
		t.Skip("контрольные XLSX создаются только при явном задании EBALA_QA_DIR")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, q := range []struct {
		kind  string
		slope float64
	}{{"gable", 20}, {"multi_gable", 10}, {"dome", 30}, {"sphere_cone", 18}, {"skylight", 0}} {
		s := defaultSnowCalc()
		s.Shape = q.kind
		s.Angle = q.slope
		s.Count = 3
		s.Span = 18
		s.Rise = 5
		s.SkylightWidth = 6
		s.H = 2.4
		if s.Shape == "sphere_cone" {
			s.Span = 18
		}
		if c, e := SnowCompute(s); e == nil {
			if e := ExportSnowExcel(c, filepath.Join(dir, q.kind+".xlsx")); e != nil {
				t.Fatalf("экспорт %s: %v", q.kind, e)
			}
		} else {
			t.Fatalf("расчёт %s: %v", q.kind, e)
		}
	}
	ice, e := IceCompute(IceLoadCalc{Region: "III", BaseMM: 10, Shape: "round", DiameterMM: 20, HeightStart: 10, HeightEnd: 70, HeightStep: 10, HighZone: "ordinary"})
	if e != nil {
		t.Fatal(e)
	}
	if e := exportTempExcel(TempReport{Version: 1, Ice: &ice}, filepath.Join(dir, "ice.xlsx")); e != nil {
		t.Fatal(e)
	}

	weight := WeightCalc{Name: "Покрытие", Rows: []WeightRow{{Name: "Стяжка", Kind: "layer", Thickness: 60, Density: 1800, Gamma: 1.3}, {Name: "Гидроизоляция", Kind: "direct", GNorm: 10, Gamma: 1.2}}}
	if err := ExportWeightExcel(weight, filepath.Join(dir, "weight.xlsx")); err != nil {
		t.Fatal(err)
	}
	general := TempReport{Version: 1, Sections: []TempSelection{{Group: "СП 20 · временные", Title: "Нагрузка", Source: "таблица 8.3", Params: map[string]interface{}{"normative": 4.0, "gamma_f": 1.2, "normative_reduced": 4.0, "design_reduced": 4.8, "unit": "кПа", "calc_scope": "Полное значение"}}}, WeightBlocks: []ProjectWeightBlock{{Title: "Перекрытие", Rows: []ProjectWeightLine{{Name: "Плита", Norm: 2.5, Gamma: 1.1, Design: 2.75}}}}}
	if err := exportTempExcel(general, filepath.Join(dir, "general.xlsx")); err != nil {
		t.Fatal(err)
	}
	transport := TempReport{Version: 1, Sections: []TempSelection{{Group: "СП 35 · транспорт", Title: "АК-14", Source: "СП 35", Params: map[string]interface{}{"K": 14.0}}, {Group: "СП 35 · транспорт", Title: "НК-14", Source: "СП 35", Params: map[string]interface{}{"K": 14.0}}}}
	if err := exportTempExcel(transport, filepath.Join(dir, "transport.xlsx")); err != nil {
		t.Fatal(err)
	}

	placement, err := CalculateCranePlacements([]float64{6000}, []CraneTrain{{Name: "Кран А", WheelXMM: []float64{0}, ForcesKN: []float64{100}}})
	if err != nil {
		t.Fatal(err)
	}
	report := TempReport{Version: 1, CraneGroups: []CraneReportGroup{{Title: "Набор", Cranes: []TempSelection{{Group: "Краны", Title: "Кран А", Wheels: []TempWheel{{PKN: 100, XMM: 0, XKnown: true}}}}, Placements: []CraneReportPlacement{{Title: "Балка 6 м", Result: placement}}}}}
	if err := exportTempExcel(report, filepath.Join(dir, "crane.xlsx")); err != nil {
		t.Fatal(err)
	}

}
