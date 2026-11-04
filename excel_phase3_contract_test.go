package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestExcelPhase3ReductionsUseTypedInputAndExactCatalogueClass(t *testing.T) {
	row := SP20Row{
		ID: "T-00012-1", SourceID: "T-00012", Source: "СП 20.13330.2016, таблица 8.3",
		Norm: 2, Gamma: 1.2, Design: 2.4,
	}
	selection, err := tempSP20Section(row, TempSeveralFloors, 36, 3)
	if err != nil {
		t.Fatal(err)
	}
	area, floors, class, mode := excel051ReductionSource(selection)
	if area != 36 || floors != 3 || class != 1 || mode != "Несколько перекрытий" {
		t.Fatalf("typed input differs from accepted choice: A=%g floors=%d class=%d mode=%q", area, floors, class, mode)
	}
	if c := excel051LegacyReductionClass("T-00012"); c != 1 {
		t.Fatalf("unambiguous legacy catalogue id lost: %d", c)
	}
	if c := excel051LegacyReductionClass("T-000123"); c != 0 {
		t.Fatalf("unrelated row received spurious reduction class: %d", c)
	}
}

func TestExcelPhase3CraneWheelSheetReferencesAcceptedInputs(t *testing.T) {
	r := TempReport{Version: 1, CraneGroups: []CraneReportGroup{{Title: "Краны", Cranes: []TempSelection{{
		Group: "Краны", Title: "=НЕ_ФОРМУЛА", Wheels: []TempWheel{
			{XMM: 0, PKN: 100, XKnown: true}, {XMM: 2000, PKN: 120, XKnown: true},
		},
	}}}}}
	out := filepath.Join(t.TempDir(), "cranes.xlsx")
	if err := exportTempExcel(r, out); err != nil {
		t.Fatal(err)
	}
	files := excel051Parts(t, out)
	workbook := files["xl/workbook.xml"]
	if !strings.Contains(workbook, "Краны - паспорт") || !strings.Contains(workbook, "Краны - исходные") || !strings.Contains(workbook, "Краны - расчёт") {
		t.Fatal("missing independent accepted wheel data sheets")
	}
	var source, calc string
	for key, xml := range files {
		if strings.HasPrefix(key, "xl/worksheets/snow") {
			if strings.Contains(xml, "Положение x, мм") {
				source = xml
			}
			if strings.Contains(xml, "Шаг до предыдущего") {
				calc = xml
			}
		}
	}
	if source == "" || calc == "" || !strings.Contains(calc, "E3/9.80665") || !strings.Contains(calc, "D4-D3") {
		t.Fatal("wheel loads or steps are not recomputed by Excel formulas")
	}
	if !strings.Contains(source, "=НЕ_ФОРМУЛА") || strings.Contains(source, "<f>=НЕ_ФОРМУЛА") {
		t.Fatal("user-entered crane name is executed as a formula")
	}
}

func TestExcelPhase3SiteUsesActualAcceptedInputs(t *testing.T) {
	site := Site{Region: "=ТЕКСТ", SnowZone: "III", ClimateJanuary: -10, ClimateJanuarySet: true}
	r := TempReport{Version: 1, Site: &site, SiteLines: []string{"Снеговой район III"}}
	path := filepath.Join(t.TempDir(), "site.xlsx")
	if err := exportTempExcel(r, path); err != nil {
		t.Fatal(err)
	}
	parts := excel051Parts(t, path)
	if !strings.Contains(parts["xl/workbook.xml"], "Местность - исходные") {
		t.Fatal("missing accepted site data sheet")
	}
	if !strings.Contains(parts["xl/workbook.xml"], `name="Исходные данные"`) {
		t.Fatal("general workbook must begin with an input sheet")
	}
}

func TestExcelPhase3LiteralLabelsNotConvertedToFormulas(t *testing.T) {
	for _, label := range []string{"=SUM(1,2)", "=HYPERLINK(\"https://example.invalid\",\"x\")", "125"} {
		c := tempExcelCell(3, 0, tempExcelLiteral(label))
		if strings.Contains(c, "<f>") || !strings.Contains(c, `t="inlineStr"`) {
			t.Fatalf("literal label was converted into a formula or number: %s", c)
		}
	}
}

func TestExcelPhase3CranePassportKeepsSelectedParameters(t *testing.T) {
	crane := TempSelection{
		Group: "Краны", Title: "=КРАН-7", Source: "Паспорт крана",
		Params: map[string]interface{}{
			"crane_span_m": 24.0,
			"capacity_t":   32.0,
			"wheelbase_mm": 4800.0,
		},
		Wheels: []TempWheel{{PKN: 105, XKnown: true, XMM: 0}},
	}
	out := filepath.Join(t.TempDir(), "crane-passport.xlsx")
	if err := exportTempExcel(TempReport{Version: 1, Sections: []TempSelection{crane}}, out); err != nil {
		t.Fatal(err)
	}
	parts := excel051Parts(t, out)
	if !strings.Contains(parts["xl/workbook.xml"], "Краны - паспорт") {
		t.Fatal("missing accepted crane passport sheet")
	}
	var passport string
	for name, body := range parts {
		if strings.HasPrefix(name, "xl/worksheets/") && strings.Contains(body, "Принятое значение") && strings.Contains(body, "4800") {
			passport = body
			break
		}
	}
	if passport == "" || !strings.Contains(passport, "32") || !strings.Contains(passport, "24") {
		t.Fatal("accepted crane passport dimensions or capacity were not preserved")
	}
	if strings.Contains(passport, "<f>КРАН-7") {
		t.Fatal("literal crane title was interpreted as an Excel formula")
	}
}
