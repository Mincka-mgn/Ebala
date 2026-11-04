package main

import (
	"html"
	"strings"
	"testing"
)

// These tests belong to phase 2 and are deliberately NOT executed until
// the owner authorizes a compilation/test run at the final stage.
func TestIceExcelDynamicHeightGridContract(t *testing.T) {
	const input = "Гололёд - исходные"
	grid := html.UnescapeString(iceExcelHeightGrid(input))
	for _, want := range []string{"На этом листе", "'Гололёд - исходные'!$B$12", "'Гололёд - исходные'!$B$18", "'Гололёд - исходные'!$B$19", "SMALL($A$4:$B$305", "Гололёд"} {
		if !strings.Contains(grid, want) {
			t.Errorf("в расчётной сетке высот не найдено %q", want)
		}
	}
	if count := strings.Count(grid, "<row "); count != 3+iceExcelHeightCapacity {
		t.Errorf("неверное число строк высотной сетки: %d", count)
	}
}

func TestConeRoundPolarDoesNotCopyGoMu(t *testing.T) {
	s := SnowCalc{Shape: "cone", Span: 18, Angle: 25, BetaStep: 15}
	c := SnowCase{Title: "2. Пространственный занос", Polar: []SnowPolarPoint{{Radius: 123456, Beta: 90, Mu: 777777}}}
	body := html.UnescapeString(snowExcelRoundPolar(s, c, 1, "Исходные данные", "Коэффициенты"))
	for _, want := range []string{"На этом листе", "'Исходные данные'!$B$33", "'Исходные данные'!$B$11", "SIN(RADIANS(", "'Коэффициенты'!B19", "'Коэффициенты'!$B$4"} {
		if !strings.Contains(body, want) {
			t.Errorf("не найдена формульная зависимость %q", want)
		}
	}
	for _, prohibited := range []string{"777777", "123456"} {
		if strings.Contains(body, prohibited) {
			t.Errorf("из ядра перенесено готовое значение %s", prohibited)
		}
	}
}

func TestDomeRoundPolarGeometryReferences(t *testing.T) {
	s := SnowCalc{Shape: "dome", Span: 18, Rise: 12, BetaStep: 15}
	c := SnowCase{Title: "2. Пространственный занос"}
	body := html.UnescapeString(snowExcelRoundPolar(s, c, 1, "Исходные данные", "Коэффициенты"))
	for _, want := range []string{"'Коэффициенты'!$B$47/2", "'Исходные данные'!$B$14", "ASIN(MIN(1,"} {
		if !strings.Contains(body, want) {
			t.Errorf("геометрия высокого купола не пересчитывается: %q", want)
		}
	}
}

func TestIceCalculationIndependentGridAndTables(t *testing.T) {
	var s IceLoadCalc
	text := html.UnescapeString(iceExcelCalcData(s))
	for _, want := range []string{"'Гололёд - сетка'!D4", "'Гололёд табл.12.2'!$E$10", "'Гололёд табл.12.3'!$H$14", "'Гололёд табл.12.4'!$G$12", "'Ветер табл.11.2'!$H$18"} {
		if !strings.Contains(text, want) {
			t.Errorf("гололёд: нет обязательной Excel-зависимости %q", want)
		}
	}
}
