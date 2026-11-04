package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTemp012GeneratedSamples(t *testing.T) {
	g, e := loadSP20Guide("Catalog/sp20_guide.json")
	if e != nil {
		t.Fatal(e)
	}
	var r SP20Row
	for _, x := range g.Records {
		if x.SourceID == "T-00013" {
			r = x
			break
		}
	}
	if r.Title == "" {
		t.Fatal("no office row")
	}
	a, e := tempSP20Section(r, TempOneFloor, 36, 1)
	if e != nil {
		t.Fatal(e)
	}
	if tempNumber(a, "normative_reduced") >= r.Norm {
		t.Fatal("no reduction")
	}
	c, e := tempManualSteps("Кран контрольный", 4, "80 90 90 80", "800 2400 800")
	if e != nil {
		t.Fatal(e)
	}
	if tempSteps(c) != "800; 2400; 800" {
		t.Fatalf("steps: %s", tempSteps(c))
	}
	report := TempReport{Version: 1, Project: "Проверочный объект", Sections: []TempSelection{a, c}}
	base := filepath.Join(os.TempDir(), "ebala012_demo")
	if e = exportTempWord(report, base+".docx"); e != nil {
		t.Fatal(e)
	}
	if e = exportTempExcel(report, base+".xlsx"); e != nil {
		t.Fatal(e)
	}
}
