package main

import (
	"encoding/xml"
	"fmt"
	"html"
	"path/filepath"
	"strings"
	"testing"
)

// These tests are prepared for the final test run. The owner asked not to
// compile or run Go during source-only intermediate stages.
func TestExcelPhase4CommonPlacementReusesIndependentFormulas(t *testing.T) {
	placement, err := CalculateCranePlacements([]float64{6000}, []CraneTrain{{Name: "Кран А", WheelXMM: []float64{0}, ForcesKN: []float64{100}}})
	if err != nil {
		t.Fatal(err)
	}
	report := TempReport{Version: 1, CraneGroups: []CraneReportGroup{{
		Title: "Набор", Cranes: []TempSelection{{Group: "Краны", Title: "Кран А", Wheels: []TempWheel{{PKN: 100, XMM: 0, XKnown: true}}}}, Placements: []CraneReportPlacement{{Title: "Балка 6 м", Result: placement}},
	}}}
	path := filepath.Join(t.TempDir(), "combined.xlsx")
	if err := exportTempExcel(report, path); err != nil {
		t.Fatal(err)
	}
	files := excel051Parts(t, path)
	wb := files["xl/workbook.xml"]
	for _, name := range []string{"Р01 Вход", "Р01 Балка", "Р01 Колёса", "Р01 Итоги"} {
		if !strings.Contains(wb, fmt.Sprintf(`name="%s"`, name)) {
			t.Errorf("missing sheet %s", name)
		}
	}
	main := strings.ReplaceAll(files["xl/worksheets/sheet1.xml"], "&#39;", "'")
	if !strings.Contains(main, "'Р01 Итоги'!C5") || !strings.Contains(main, "'Р01 Итоги'!E5") {
		t.Fatal("general ledger still uses Go efforts or sections, not calculation references")
	}
	all := ""
	for name, data := range files {
		if strings.HasSuffix(name, ".xml") && strings.HasPrefix(name, "xl/worksheets/") {
			// Namespace and duplicate-attribute errors must be caught here,
			// not by an Excel recovery dialog on the user's computer.
			if err := xml.Unmarshal([]byte(data), new(interface{})); err != nil {
				t.Errorf("malformed sheet %s: %v", name, err)
			}
		}
		if strings.HasPrefix(name, "xl/worksheets/snow") {
			all += html.UnescapeString(data)
		}
	}
	for _, fragment := range []string{"'Р01 Вход'!", "'Р01 Балка'!", "'Р01 Колёса'!", "SUMIF(", "D6*E6"} {
		if !strings.Contains(all, fragment) {
			t.Errorf("missing independent formula %q", fragment)
		}
	}
	if strings.Contains(all, "<f>'Исходные данные'!") || strings.Contains(all, "<f>'Балка'!") {
		t.Fatal("unrenamed standalone sheet reference in common Excel")
	}
}

func TestExcelPhase4PlacementSheetNamesUnique(t *testing.T) {
	r, err := CalculateCranePlacements([]float64{6000}, []CraneTrain{{Name: "Кран", WheelXMM: []float64{0}, ForcesKN: []float64{75}}})
	if err != nil {
		t.Fatal(err)
	}
	report := TempReport{Version: 1, CraneGroups: []CraneReportGroup{{Title: "Набор", Cranes: []TempSelection{{Group: "Краны", Title: "Кран", Wheels: []TempWheel{{PKN: 75, XMM: 0, XKnown: true}}}}, Placements: []CraneReportPlacement{
		{Title: "Первая", Result: r}, {Title: "Вторая", Result: r},
	}}}}
	path := filepath.Join(t.TempDir(), "two.xlsx")
	if err := exportTempExcel(report, path); err != nil {
		t.Fatal(err)
	}
	wb := excel051Parts(t, path)["xl/workbook.xml"]
	for _, name := range []string{"Р01 Вход", "Р01 Итоги", "Р02 Вход", "Р02 Итоги"} {
		if strings.Count(wb, fmt.Sprintf(`name="%s"`, name)) != 1 {
			t.Errorf("nonunique sheet %s", name)
		}
	}
}
