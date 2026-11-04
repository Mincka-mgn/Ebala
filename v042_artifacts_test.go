package main

import (
	"os"
	"path/filepath"
	"testing"
)

func Test042ReleaseSamples(t *testing.T) {
	dir := os.Getenv("E042_QA_OUT")
	if dir == "" {
		t.Skip("release QA only")
	}
	if e := os.MkdirAll(dir, 0755); e != nil {
		t.Fatal(e)
	}
	s := defaultSnowCalc()
	s.Shape = "cone"
	s.Angle = 25
	s.Span = 20
	s.Width = 20
	s.RoofHeight = 12
	s.Ground = 2
	snow, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if e = ExportSnowWord(snow, filepath.Join(dir, "Снег_резервуар.docx")); e != nil {
		t.Fatal(e)
	}
	if e = ExportSnowExcel(snow, filepath.Join(dir, "Снег_резервуар.xlsx")); e != nil {
		t.Fatal(e)
	}
	ice, e := IceCompute(defaultIceLoad())
	if e != nil {
		t.Fatal(e)
	}
	p := initialProject()
	p.Weight = &WeightCalc{Rows: []WeightRow{{Name: "Стяжка", Kind: "layer", Thickness: 50, Density: 2000, Gamma: 1.3}}}
	p.initReportBlocks(nil)
	p.putSnowCalc(0, snow)
	p.putIceCalc(0, ice)
	r := p.selectedReport()
	r.SiteLines = projectSiteLines(p.Site, p)
	if e = exportTempWord(r, filepath.Join(dir, "Общий_отчёт.docx")); e != nil {
		t.Fatal(e)
	}
	if e = exportTempExcel(r, filepath.Join(dir, "Общий_расчёт.xlsx")); e != nil {
		t.Fatal(e)
	}
}
