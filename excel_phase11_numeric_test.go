package main

import (
	"strings"
	"testing"
)

// A coordinate of zero is a valid boundary, not an empty row. Some spreadsheet
// engines treat the comparison 0<>"" as FALSE; ISNUMBER does not have this
// ambiguity and works for both calculated zero and other numeric coordinates.
func TestExcelPhase11OriginIsNumericPoint(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "gable"
	s.Angle = 20
	c, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Cases) == 0 {
		t.Fatal("нет расчётных загружений")
	}
	sheet := snowExcelVariantData(c, c.Cases[0], 0)
	if !strings.Contains(sheet, `ISNUMBER(A5),ISNUMBER(B5)`) {
		t.Fatal("первую точку x=0 ошибочно проверяют на равенство пустой строке")
	}
}

func TestExcelPhase11CraneSingleSpanDirectReferences(t *testing.T) {
	result, err := CalculateCranePlacements([]float64{6000}, []CraneTrain{{Name: "Кран А", WheelXMM: []float64{0}, ForcesKN: []float64{100}}})
	if err != nil {
		t.Fatal(err)
	}
	parts, err := cranePlacementAuditSheets(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(parts.Detail, "INDEX(&#39;Балка&#39;!$B$5:$B$5") || strings.Contains(parts.Detail, "MATCH(MAX(") {
		t.Fatal("однопролётная балка не должна иметь поиска по одноклеточному диапазону")
	}
}

func TestExcelPhase11CraneNoFmtFragments(t *testing.T) {
	for _, spans := range [][]float64{{6000}, {6000, 6000}} {
		result, err := CalculateCranePlacements(spans, []CraneTrain{{Name: "Кран А", WheelXMM: []float64{0}, ForcesKN: []float64{100}}})
		if err != nil {
			t.Fatal(err)
		}
		p, err := cranePlacementAuditSheets(result)
		if err != nil {
			t.Fatal(err)
		}
		for _, sheet := range []string{p.Inputs, p.Beam, p.Detail, p.Summary} {
			if strings.Contains(sheet, "%!") {
				t.Fatal("некорректное форматирование Go в формуле XLSX")
			}
		}
	}
}
