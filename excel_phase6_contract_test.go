package main

import (
	"html"
	"strings"
	"testing"
)

// Prepared for the final test run. The intermediate source-only stages are
// deliberately not compiled on the user's instruction.
func TestPhase6SnowVariantGridUsesEditableDimensions(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "multi_gable"
	s.Count = 3
	s.Span = 18
	c := SnowCase{Title: "1. Равномерная по пролётам", Intervals: []SnowInterval{{X1: 0, X2: 54, Mu: 1}}}
	x1, x2 := snowExcelSampleInterval(s, 0, snowExcelSampleCount(s))
	if !strings.Contains(x1, "'Исходные данные'!$B$17") || !strings.Contains(x1, "'Исходные данные'!$B$11") || !strings.Contains(x2, "/32") {
		t.Fatalf("multi-bay grid must depend on the edited count and span with 32 samples per bay: %s, %s", x1, x2)
	}
	if !strings.Contains(html.UnescapeString(snowExcelZoneBoundaryGuide(s)), "'Исходные данные'!$B$11*'Исходные данные'!$B$17") {
		t.Fatal("end boundary must depend on live span and bay count")
	}
	sheet := html.UnescapeString(snowExcelVariantData(s, c, 0))
	for _, token := range []string{
		"'Исходные данные'!$B$11", "'Исходные данные'!$B$17",
		"'Исходные данные'!$B$23", "'Исходные данные'!$B$8",
		"(A5+B5)/2",
	} {
		if !strings.Contains(sheet, token) {
			t.Fatalf("independent snow sheet misses %s", token)
		}
	}
	if strings.Contains(sheet, "54</v>") {
		t.Fatal("old Go-generated endpoint leaked into the worksheet")
	}
}

func TestPhase6SnowGridDomains(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "step"
	a, b := snowExcelSampleDomain(s)
	if !strings.Contains(a, "$B$18") || !strings.Contains(b, "$B$19") {
		t.Fatalf("step roof must use independently editable upper/lower spans: %q %q", a, b)
	}
	s.Shape = "dome"
	_, b = snowExcelSampleDomain(s)
	if b != "'Коэффициенты'!$B$47" {
		t.Fatalf("deep dome must use calculated projection: %q", b)
	}
}

func TestPhase6AngleFromHeightIsNotGoConstant(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "gable"
	s.SlopeMode = "height"
	src := snowExcelInputData(s)
	cf := snowExcelCoefData(s)
	if !strings.Contains(src, "Способ задания уклона") || !strings.Contains(src, "Подъём") || !strings.Contains(cf, "DEGREES(ATAN(") {
		t.Fatal("derived slope not linked to editable rise")
	}
	if !strings.Contains(cf, "$B$37") || !strings.Contains(cf, "$B$14") {
		t.Fatal("rise/angle mode is not an Excel dependency")
	}
}
