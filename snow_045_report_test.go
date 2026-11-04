package main

import (
	"archive/zip"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func readDocXML(t *testing.T, path string) string {
	t.Helper()
	a, e := zip.OpenReader(path)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	for _, f := range a.File {
		if f.Name == "word/document.xml" {
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
	t.Fatal("no document.xml")
	return ""
}
func TestSnow045B3WordGroups(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "skylight"
	s.Span = 20
	s.Width = 22
	s.SkylightWidth = 6
	s.H = 1.5
	s.Ground = 2
	s, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "out.docx")
	if e = ExportSnowWord(s, path); e != nil {
		t.Fatal(e)
	}
	xml := readDocXML(t, path)
	if !strings.Contains(xml, "Совместное загружение зон A, B, C") {
		t.Fatal("missing joint case")
	}
	if !strings.Contains(xml, "Зона B") {
		t.Fatal("missing end zone")
	}
	if !strings.Contains(xml, "План зон A, B, C") {
		t.Fatal("missing explanatory plan")
	}
	if strings.Contains(xml, "3.3. 1C.") {
		t.Fatal("separate zone duplicated as case")
	}
}
func TestSnow045B2WordOmitsMirroredImage(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "vault"
	s.Span = 20
	s.Rise = 3
	s.Ground = 2
	s, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "out.docx")
	if e = ExportSnowWord(s, path); e != nil {
		t.Fatal(e)
	}
	xml := readDocXML(t, path)
	if strings.Contains(xml, "Несимметричная (зеркальная)") {
		t.Fatal("mirror shown as variant")
	}
	if !strings.Contains(xml, "может применяться также в зеркальном отображении") {
		t.Fatal("missing note")
	}
}
