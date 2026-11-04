package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// Main-window XLSX export must use the same accepted locality inputs as the
// temporary-loads window. Previously selectedReport did not carry Site at all.
func TestExcelPhase3MainWindowRetainsLocalitySnapshot(t *testing.T) {
	p := Project{Site: Site{Region: "=Ставрополь", SnowZone: "IV", ClimateJanuary: -12, ClimateJanuarySet: true},
		SavedBlocks: []SavedReportBlock{{ID: 1, Kind: "weight", Title: "Постоянная нагрузка", Weight: &ProjectWeightLine{Name: "Слой", Norm: 1.2, Gamma: 1.1, Design: 1.32}}}}
	report := p.selectedReport()
	if report.Site == nil || report.Site.SnowZone != "IV" || report.Site.ClimateJanuary != -12 {
		t.Fatal("main-window report dropped the accepted locality inputs")
	}
	// Changing project site after creating the report must not modify its
	// accepted source snapshot.
	p.Site.SnowZone = "I"
	if report.Site.SnowZone != "IV" {
		t.Fatal("locality source is not a stable snapshot")
	}
	file := filepath.Join(t.TempDir(), "site-main.xlsx")
	if err := exportTempExcel(report, file); err != nil {
		t.Fatal(err)
	}
	parts := excel051Parts(t, file)
	if !strings.Contains(parts["xl/workbook.xml"], `name="Местность - исходные"`) {
		t.Fatal("main export omitted accepted locality input worksheet")
	}
	if !strings.Contains(parts["xl/workbook.xml"], `name="Исходные данные"`) {
		t.Fatal("main export omitted first input worksheet")
	}
	var found bool
	for name, body := range parts {
		if strings.HasPrefix(name, "xl/worksheets/") && strings.Contains(body, "Ставрополь") {
			found = true
			if strings.Contains(body, "<f>Ставрополь") {
				t.Fatal("literal locality text became an Excel formula")
			}
		}
	}
	if !found {
		t.Fatal("locality value not found in Excel source cells")
	}
}

// An accepted locality is not itself a complete load report. The separate
// locality export handles that use case and general report stays nonempty.
func TestExcelPhase3EmptyLoadReportStillRejected(t *testing.T) {
	p := Project{Site: Site{Region: "Республика Адыгея"}}
	err := exportTempExcel(p.selectedReport(), filepath.Join(t.TempDir(), "empty.xlsx"))
	if err == nil || !strings.Contains(err.Error(), "Ведомость пуста") {
		t.Fatalf("an empty project was exported as a complete load report: %v", err)
	}
}
