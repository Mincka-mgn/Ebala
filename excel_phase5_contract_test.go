package main

import (
	"html"
	"strings"
	"testing"
)

// Regression contracts prepared at phase 5, intentionally not executed until
// the owner authorizes the final compiled test run.
func TestPhase5MissingIceStudyCannotBecomeZeroOrNA(t *testing.T) {
	entries := map[string]string{
		"xl/workbook.xml":            `<workbook><sheets></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships></Relationships>`,
		"[Content_Types].xml":        `<Types></Types>`,
	}
	if err := iceExcelNormSheets(entries, IceLoadCalc{}, ""); err != nil {
		t.Fatal(err)
	}
	foundTable := false
	for name, body := range entries {
		if strings.Contains(body, "NA()") {
			t.Fatalf("ice study introduced Excel NA() in %s", name)
		}
		if strings.Contains(body, "ISNUMBER(INDEX($D$4:$D$6") {
			foundTable = true
		}
	}
	if !foundTable {
		t.Fatal("special-study table lost explicit missing-observation guards")
	}
}

func TestPhase5IceDerivedValuesPropagateBlankInputs(t *testing.T) {
	body := html.UnescapeString(iceExcelCalcData(IceLoadCalc{}))
	for _, part := range []string{
		`B5=""`, `C5=""`, `D5=""`, `E5=""`, `F5=""`,
		`G5=""`, `'Гололёд табл.12.2'!$G$10`,
	} {
		if !strings.Contains(body, part) {
			t.Errorf("ice formula is missing a guard/reference: %s", part)
		}
	}
	if strings.Contains(body, "NA()") || strings.Contains(body, "1/0") {
		t.Fatal("ice calculation contains an error-producing placeholder")
	}
	if strings.Contains(body, `IF('Гололёд - исходные'!$B$46>0,`) {
		t.Fatal("accepted zero for iced drag is still silently overridden")
	}
}
