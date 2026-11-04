package main

import (
	"archive/zip"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func Test013IndustryAndOrder(t *testing.T) {
	g, e := loadSP20Guide("Catalog/sp20_guide.json")
	if e != nil {
		t.Fatal(e)
	}
	if g.Records[0].Title != "Квартиры, гостиницы, общежития, спальни, палаты, террасы" {
		t.Fatal("logical order")
	}
	var industrial SP20Row
	for _, r := range g.Records {
		if r.ID == "IND-1" {
			industrial = r
			break
		}
	}
	if !industrial.Editable || industrial.Gamma != 1 {
		t.Fatal("industrial liquid gamma")
	}
	s, e := sp20CustomSection(industrial, "Жидкость в технологической емкости", "350,5", "1,0")
	if e != nil || abs(tempNumber(s, "normative_reduced")-kgfM2ToKpa(350.5)) > 0.00001 {
		t.Fatal(s, e)
	}
	blank, e := sp20CustomSection(industrial, "Неизвестная жидкость", "", "1,0")
	if e != nil {
		t.Fatal(e)
	}
	if d, ok := blank.Params["is_defined"].(bool); !ok || d {
		t.Fatal("blank should not be zero loading")
	}
	out := filepath.Join(t.TempDir(), "blank.xlsx")
	if e := exportTempExcel(TempReport{Version: 1, Sections: []TempSelection{blank}}, out); e != nil {
		t.Fatal(e)
	}
	z, e := zip.OpenReader(out)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	for _, f := range z.File {
		if f.Name == "xl/worksheets/sheet1.xml" {
			r, _ := f.Open()
			b, _ := io.ReadAll(r)
			r.Close()
			if !strings.Contains(string(b), "По заданию") {
				t.Fatal("blank value lost")
			}
		}
	}
}
func Test013CraneUnits(t *testing.T) {
	s, e := tempManualSteps("Кран", 4, "98,0665; 98,0665; 98,0665; 98,0665", "1000 2000 1000")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(tempForcesBoth(s), "тс") {
		t.Fatal("tf missing")
	}
	if tempSteps(s) != "1000; 2000; 1000" {
		t.Fatal("steps wrong")
	}
}
