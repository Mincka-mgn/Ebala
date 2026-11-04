package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSnow031SVGAndExcelLinks(t *testing.T) {
	for _, shape := range []string{"mono", "gable", "vault", "skylight", "step", "cone", "dome"} {
		s := defaultSnowCalc()
		s.Shape = shape
		_, e := snowRoofImage(s)
		if e != nil {
			t.Fatalf("%s: %v", shape, e)
		}
	}
	s := sampleSnow("skylight")
	s.JanuarySet = true
	s.JanuaryTemp = -12
	calc, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(t.TempDir(), "x.xlsx")
	if e := ExportSnowExcel(calc, file); e != nil {
		t.Fatal(e)
	}
	z, e := zip.OpenReader(file)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	for _, f := range z.File {
		if strings.HasSuffix(f.Name, ".xml") || strings.HasSuffix(f.Name, ".rels") {
			r, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			decoder := xml.NewDecoder(r)
			for {
				_, e := decoder.Token()
				if e == io.EOF {
					break
				}
				if e != nil {
					t.Fatalf("invalid %s: %v", f.Name, e)
				}
			}
			r.Close()
		}
	}
}
func TestSnow031ExamplesToDisk(t *testing.T) {
	if os.Getenv("EBALA_QA_EXPORT") != "1" {
		return
	}
	for _, shape := range []string{"gable", "skylight", "step", "cone"} {
		s := sampleSnow(shape)
		if shape == "cone" {
			s.Angle = 20
		}
		if shape == "step" {
			s.UpperRoof = SnowRoofPart{Shape: "gable", Span: 18, Angle: 15, EdgeLevel: 10}
			s.LowerRoof = SnowRoofPart{Shape: "mono", Span: 14, Angle: 6, EdgeLevel: 6}
			s.H = 4
			s.Upper = 18
			s.Lower = 14
		}
		s.JanuarySet = true
		s.JanuaryTemp = -13
		s.WinterWindSet = true
		s.WinterWind = 4.3
		s.RoofHeight = 8
		s.Gamma = 1.4
		out, e := SnowCompute(s)
		if e != nil {
			t.Fatal(e)
		}
		dir := "/mnt/data/ebala031work/qa"
		os.MkdirAll(dir, 0755)
		for _, ext := range []string{"xlsx", "docx"} {
			path := filepath.Join(dir, fmt.Sprintf("%s.%s", shape, ext))
			if ext == "xlsx" {
				e = ExportSnowExcel(out, path)
			} else {
				e = ExportSnowWord(out, path)
			}
			if e != nil {
				t.Fatal(e)
			}
		}
	}
}
