package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV014ReportOnlySelected(t *testing.T) {
	dir := t.TempDir()
	a, e := tempManualSteps("Кран", 4, "100;110;90;80", "2000; 3000; 2000")
	if e != nil {
		t.Fatal(e)
	}
	report := TempReport{Version: 1, Sections: []TempSelection{a, {Group: "Краны · ручной ввод", Title: "Кран пустой", Wheels: nil}}}
	dst := filepath.Join(dir, "crane.docx")
	if e := exportTempWord(report, dst); e != nil {
		t.Fatal(e)
	}
	z, e := zip.OpenReader(dst)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	for _, f := range z.File {
		if f.Name != "word/document.xml" {
			continue
		}
		h, _ := f.Open()
		b, _ := io.ReadAll(h)
		h.Close()
		if !strings.Contains(string(b), "Колёсная схема") || strings.Contains(string(b), "Кран пустой") || strings.Contains(string(b), "Транспортные нагрузки (СП") {
			t.Fatal("extraneous sections")
		}
	}
	if _, e := os.Stat(dst); e != nil {
		t.Fatal(e)
	}
}
func TestV014ReadableTransport(t *testing.T) {
	s := TempSelection{Group: "СП 35 · транспорт", Title: "АБ-51", Params: map[string]interface{}{"front_axle_kN": 167.0, "rear_axle_kN": 333.0, "axle_spacing_m": 3.5}}
	d, n, g, c := tempTransportFields(s)
	if !strings.Contains(d, "3,5 м") || !strings.Contains(n, "167 кН") || !strings.Contains(n, "333 кН") || !strings.Contains(g, "1,10") {
		t.Fatalf("fields %s %s %s %s", d, n, g, c)
	}
	s.Params["gamma_selected"] = 1.1
	_, _, _, c = tempTransportFields(s)
	if !strings.Contains(c, "183,7") {
		t.Fatal(c)
	}
}
