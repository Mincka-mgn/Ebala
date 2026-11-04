package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readZipText(t *testing.T, path, name string) string {
	t.Helper()
	z, e := zip.OpenReader(path)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	for _, f := range z.File {
		if f.Name == name {
			r, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			defer r.Close()
			b, e := io.ReadAll(r)
			if e != nil {
				t.Fatal(e)
			}
			return string(b)
		}
	}
	t.Fatalf("missing %s in %s", name, path)
	return ""
}
func TestProjectSummaryAndExports017(t *testing.T) {
	p := Project{Version: 1, Site: Site{Region: "Тестовая область", Place: "Контрольный город", SnowZone: "II", SnowGround: 1.0}, Weight: &WeightCalc{Rows: []WeightRow{{Name: "Монолитная плита", Kind: "direct", GNorm: 300, Gamma: 1.1}}}}
	wr := projectWeightLines(p)
	if len(wr) != 1 || wr[0].Norm <= 0 || wr[0].Design <= wr[0].Norm {
		t.Fatalf("weight rows: %+v", wr)
	}
	lines := projectSiteLines(p.Site, p)
	if len(lines) == 0 {
		t.Fatal("missing site details")
	}
	r := TempReport{Version: 1, Project: "Контрольный объект", SiteLines: lines, WeightRows: wr}
	dir := t.TempDir()
	doc := filepath.Join(dir, "report.docx")
	sheet := filepath.Join(dir, "report.xlsx")
	if e := exportTempWord(r, doc); e != nil {
		t.Fatal(e)
	}
	if e := exportTempExcel(r, sheet); e != nil {
		t.Fatal(e)
	}
	for _, v := range []struct{ path, name string }{{doc, "word/document.xml"}, {sheet, "xl/worksheets/sheet1.xml"}} {
		s := readZipText(t, v.path, v.name)
		for _, value := range []string{"Тестовая область", "Монолитная плита"} {
			if !strings.Contains(s, value) {
				t.Errorf("%s missing %s", v.path, value)
			}
		}
	}
	if _, e := os.Stat(doc); e != nil {
		t.Fatal(e)
	}
}
func TestEditableCraneCells017(t *testing.T) {
	s := TempSelection{Group: "Краны · ручной ввод", Title: "Кран 1", Params: map[string]interface{}{}}
	for _, step := range []struct {
		col   int
		value string
		tons  bool
	}{{2, "4", false}, {3, "10; 11; 12; 13", false}, {4, "1000; 2000; 1000", false}} {
		if e := tempCraneSetCell(&s, step.col, step.value, step.tons); e != nil {
			t.Fatal(e)
		}
	}
	if len(s.Wheels) != 4 || s.Wheels[2].XMM != 3000 || s.Wheels[1].PKN != 11 {
		t.Fatalf("wrong crane %+v", s.Wheels)
	}
	if e := tempCraneSetCell(&s, 3, "1; 2; 3; 4", true); e != nil {
		t.Fatal(e)
	}
	if len(s.Wheels) != 4 || abs(s.Wheels[0].PKN-9.80665) > 0.01 {
		t.Fatalf("unit conversion %+v", s.Wheels)
	}
	if e := tempCraneSetCell(&s, 2, "8", false); e != nil {
		t.Fatal(e)
	}
	if len(s.Wheels) != 0 || tempCraneWheelsCount(s) != 8 {
		t.Fatal("incomplete wheel system must stay visible but not reportable")
	}
}
