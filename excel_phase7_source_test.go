package main

import (
	"strings"
	"testing"
)

// Source-only checkpoint: tests are prepared, not executed until the final
// integration phase as requested by the project owner.
func TestPhase7ZoneBoundaryGridUsesExactSkylightSections(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "skylight"
	nodes := snowExcelSampleNodes(s)
	if len(nodes) != 6 {
		t.Fatalf("expected 6 skylight nodes, got %d", len(nodes))
	}
	for _, token := range []string{"$B$24", "$B$25", "$B$23"} {
		guide := snowExcelZoneBoundaryGuide(s)
		if !strings.Contains(guide, token) {
			t.Fatalf("zone guide misses editable coefficient %s", token)
		}
	}
	n := snowExcelSampleCount(s)
	_, end := snowExcelSampleInterval(s, n-1, n)
	if end != "='Границы зон'!$B$8" {
		t.Fatalf("unexpected last boundary: %s", end)
	}
}
func TestPhase7VaultBreakpointsAtThirtyAndSixtyDegrees(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "vault"
	nodes := snowExcelSampleNodes(s)
	if len(nodes) != 7 {
		t.Fatalf("expected 7 arch nodes, got %d", len(nodes))
	}
	guide := snowExcelZoneBoundaryGuide(s)
	for _, token := range []string{"RADIANS(30)", "RADIANS(60)", "касательная 30°", "касательная 60°"} {
		if !strings.Contains(guide, token) {
			t.Fatalf("vault boundary guide misses %s", token)
		}
	}
}
func TestPhase7PairedRoofUsesTwoIndependentFormulas(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "gable"
	s.RightRoof = &SnowRoofPart{Shape: "flat", Span: 24, RoofHeight: 8}
	base, _ := snowExcelPairedMu(s, SnowCase{Title: "1. Основное загружение обоих пролётов"}, 5)
	for _, token := range []string{"'Исходные данные'!$B$11", "'Коэффициенты'!$B$4", "'Коэффициенты'!$B$55"} {
		if !strings.Contains(base, token) {
			t.Fatalf("combined formula misses %s", token)
		}
	}
	// The right flat-roof μ is constant, so its span belongs to the
	// calculation grid, not necessarily the μ expression itself.
	_, gridEnd := snowExcelSampleDomain(s)
	if !strings.Contains(gridEnd, "'Исходные данные'!$B$41") {
		t.Fatal("paired-roof sampling grid misses editable right-roof span")
	}
	uneven, _ := snowExcelPairedMu(s, SnowCase{Title: "Левый пролёт: 2. Неравномерная"}, 5)
	if !strings.Contains(uneven, "0.75") || !strings.Contains(uneven, "1.25") {
		t.Fatal("left gable alternative missing in combined case")
	}
}
func TestPhase7PairedSkylightCZonesAreCalculatedSeparately(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "skylight"
	s.RightRoof = &SnowRoofPart{Shape: "skylight", Span: 36, SkylightWidth: 8, LanternHeight: 2}
	combined, _ := snowExcelPairedMu(s, SnowCase{Title: "Оба фонаря, зона C: левый 1C. По бокам фонаря (зона C) + правый 1C. По бокам фонаря (зона C)"}, 5)
	for _, token := range []string{"'Исходные данные'!$B$16", "'Исходные данные'!$B$46", "'Исходные данные'!$B$15", "'Исходные данные'!$B$47"} {
		if !strings.Contains(combined, token) {
			t.Fatalf("two-skylight case misses input %s", token)
		}
	}
}
func TestPhase7PairedStepUsesIndependentB8Parameters(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "gable"
	s.RightRoof = &SnowRoofPart{Shape: "flat", Span: 24}
	f, _ := snowExcelPairedMu(s, SnowCase{Title: "Занос у перепада высот: У высокой стены / перепада"}, 5)
	if f == "=NA()" || !strings.Contains(f, "'Перепад Б.8'!$B$25") {
		t.Fatal("paired B.8 must use separately computed snowdrift parameters")
	}
	for _, token := range []string{"m₂ нижней кровли", "dпер — длина переноса", "μ мешка", "Применимость отдельного заноса"} {
		if !strings.Contains(snowExcelPairedStepData(s), token) {
			t.Fatalf("missing B.8 check line %s", token)
		}
	}
}
func TestPhase7RightRoofAngleFromEditableRise(t *testing.T) {
	s := defaultSnowCalc()
	s.RightRoof = &SnowRoofPart{Shape: "gable", Span: 24, Rise: 5, SlopeMode: "height"}
	inp := snowExcelInputData(s)
	cf := snowExcelCoefData(s)
	if !strings.Contains(inp, "Второй пролёт — способ задания уклона") ||
		!strings.Contains(cf, "$B$49") || !strings.Contains(cf, "$B$45") ||
		!strings.Contains(cf, "DEGREES(ATAN(") {
		t.Fatal("second roof input mode is not linked to an independent Excel formula")
	}
}
