package main

import (
	"strings"
	"testing"
)

// The independent crane-sheet generator writes OOXML entities before its
// sheets are inserted into the combined workbook. Renaming only literal
// apostrophes leaves formulas referring to unrelated existing worksheets.
func TestExcelPhase10CraneEscapedReferences(t *testing.T) {
	source := `<f>&#39;Исходные данные&#39;!C10+INDEX(&#39;Балка&#39;!$B$5:$B$5,1)+SUMIF(&#39;Колёса&#39;!$A$6:$A$9,1,&#39;Колёса&#39;!$F$6:$F$9)</f>`
	got := excelPhase4PlacementRename(source, 1)
	for _, old := range []string{"&#39;Исходные данные&#39;!", "&#39;Балка&#39;!", "&#39;Колёса&#39;!"} {
		if strings.Contains(got, old) {
			t.Fatalf("old worksheet reference remains: %s", old)
		}
	}
	for _, want := range []string{"&#39;Р01 Вход&#39;!", "&#39;Р01 Балка&#39;!", "&#39;Р01 Колёса&#39;!"} {
		if !strings.Contains(got, want) {
			t.Fatalf("renamed reference missing: %s", want)
		}
	}
}
