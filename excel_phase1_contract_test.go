package main

import (
	"strings"
	"testing"
)

// A label is an input, never an instruction to the spreadsheet engine.
func TestExcelPhase1SourceCellsAreNotFormulas(t *testing.T) {
	for _, raw := range []string{"=1+2", "=HYPERLINK(\"https://example.invalid\",\"test\")", "@SUM(A1:A2)", "Обычная строка"} {
		cell := tempExcelInputCell(3, 0, raw)
		if strings.Contains(cell, "<f>") || !strings.Contains(cell, `t="inlineStr"`) {
			t.Fatalf("user text %q became an executable cell: %s", raw, cell)
		}
	}
	numeric := tempExcelInputCell(3, 1, "-2.5")
	if strings.Contains(numeric, "<f>") || !strings.Contains(numeric, "<v>-2.5</v>") {
		t.Fatalf("source number is not a numeric value: %s", numeric)
	}
}

func TestExcelPhase1DerivedConeHeightLivesOnGeometrySheet(t *testing.T) {
	calc := defaultSnowCalc()
	calc.Shape = "sphere_cone"
	input := snowExcelInputData(calc)
	start := strings.Index(input, `<row r="14">`)
	if start < 0 {
		t.Fatal("missing cone height placeholder")
	}
	rest := input[start:]
	end := strings.Index(rest, "</row>")
	if end < 0 {
		t.Fatal("missing input row ending")
	}
	if strings.Contains(rest[:end], "<f>") || strings.Contains(rest[:end], `<c r="B14"><v>`) {
		t.Fatal("derived cone height is still stored on the input sheet")
	}
	geom := snowExcelSphereGeometry(calc, "Исходные данные")
	if !strings.Contains(geom, "B5*TAN(RADIANS(B6))") {
		t.Fatal("missing cone height formula on the geometry sheet")
	}
}

func TestExcelPhase1IceInputsDoNotExecuteNames(t *testing.T) {
	calc := IceLoadCalc{}
	source := iceExcelInputData(calc)
	if strings.Contains(source, "<f>") {
		t.Fatal("an accepted ice input was encoded as a formula")
	}
}

func TestExcelPhase1FormulaHasNoKernelCache(t *testing.T) {
	result := cellFormula("E9", "B9*C9", 999.123, 3)
	if strings.Contains(result, "999.123") || !strings.Contains(result, "<f>B9*C9</f><v></v>") {
		t.Fatalf("formula contains precomputed Go value: %s", result)
	}
}
