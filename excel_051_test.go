package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func excel051Parts(t *testing.T, path string) map[string]string {
	t.Helper()
	z, e := zip.OpenReader(path)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	entries := map[string]string{}
	for _, f := range z.File {
		if !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		rd, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		b, e := io.ReadAll(rd)
		if e != nil {
			t.Fatal(e)
		}
		rd.Close()
		entries[f.Name] = string(b)
	}
	return entries
}
func Test051SnowIndependentTables(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "gable"
	s.Angle = 25
	s.JanuarySet = true
	s.JanuaryTemp = -15
	s.WinterWindSet = true
	s.WinterWind = 5
	s, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "snow.xlsx")
	if e = ExportSnowExcel(s, path); e != nil {
		t.Fatal(e)
	}
	z := excel051Parts(t, path)
	for _, p := range []string{"Таблица 10.2", "Таблица 11.2", "Исходные данные", "Коэффициенты"} {
		if !strings.Contains(z["xl/workbook.xml"], p) {
			t.Fatalf("missing %s", p)
		}
	}
	if !strings.Contains(strings.Join(mapValues051(z), " "), "INDEX($B$4:$D$14") {
		t.Fatal("table 11.2 does not interpolate")
	}
	if !strings.Contains(strings.Join(mapValues051(z), " "), "Таблица 11.2") {
		t.Fatal("snow coefficient does not reference normative table")
	}
}
func Test051IceIndependentTables(t *testing.T) {
	ice, e := IceCompute(IceLoadCalc{Region: "III", BaseMM: 10, Shape: "round", DiameterMM: 20, HeightStart: 10, HeightEnd: 70, HeightStep: 10, HighZone: "ordinary"})
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "ice.xlsx")
	if e = exportTempExcel(TempReport{Version: 1, Ice: &ice}, path); e != nil {
		t.Fatal(e)
	}
	z := excel051Parts(t, path)
	wb := z["xl/workbook.xml"]
	for _, name := range []string{"Гололёд табл.12.2", "Гололёд табл.12.3", "Гололёд табл.12.4", "Гололёд - исходные"} {
		if !strings.Contains(wb, name) {
			t.Fatalf("missing %s", name)
		}
	}
	if strings.Index(wb, "Гололёд - исходные") > strings.Index(wb, "Гололёд табл.12.2") {
		t.Fatal("input sheet not first")
	}
	all := strings.Join(mapValues051(z), " ")
	if !strings.Contains(all, "Гололёд - параметры") {
		t.Fatal("formula lacks g/gamma reference")
	}
	if regexp.MustCompile(`(?i)<t(?:\s[^>]*)?>\s*0/1\s*</t>`).MatchString(all) {
		t.Fatal("old binary unit label")
	}
}
func Test051StandaloneWeightSeparated(t *testing.T) {
	var w WeightCalc
	w.Name = "Покрытие"
	w.Rows = []WeightRow{{Name: "Стяжка", Kind: "layer", Thickness: 60, Density: 1800, Gamma: 1.3}, {Name: "Гидроизоляция", Kind: "direct", GNorm: 10, Gamma: 1.2}}
	path := filepath.Join(t.TempDir(), "weight.xlsx")
	if e := ExportWeightExcel(w, path); e != nil {
		t.Fatal(e)
	}
	z := excel051Parts(t, path)
	wb := z["xl/workbook.xml"]
	if !strings.Contains(wb, "Исходные данные") || !strings.Contains(wb, "Собственный вес") {
		t.Fatalf("missing weight sheets %s", wb)
	}
	if strings.Index(wb, "Исходные данные") > strings.Index(wb, "Собственный вес") {
		t.Fatal("weight input not first")
	}
	all := strings.Join(mapValues051(z), " ")
	if !strings.Contains(all, "&#39;Исходные данные&#39;!") && !strings.Contains(all, "'Исходные данные'!") {
		t.Fatal("no source references")
	}
}
func Test051GeneralWeightAndTemporaryFormulas(t *testing.T) {
	item := TempSelection{Group: "СП 20 · временные", Title: "Нагрузка", Source: "таблица 8.3", Params: map[string]interface{}{"normative": 4.0, "gamma_f": 1.2, "normative_reduced": 4.0, "design_reduced": 4.8, "unit": "кПа", "calc_scope": "Полное значение"}}
	r := TempReport{Version: 1, Sections: []TempSelection{item}, WeightBlocks: []ProjectWeightBlock{{Title: "Перекрытие", Rows: []ProjectWeightLine{{Name: "Плита", Norm: 2.5, Gamma: 1.1, Design: 2.75}}}}}
	path := filepath.Join(t.TempDir(), "general.xlsx")
	if e := exportTempExcel(r, path); e != nil {
		t.Fatal(e)
	}
	z := excel051Parts(t, path)
	wb := z["xl/workbook.xml"]
	for _, name := range []string{"Временные - исходные", "Временные - расчёт", "Собственный вес - исходные", "Собственный вес - расчёт"} {
		if !strings.Contains(wb, name) {
			t.Fatal("missing " + name)
		}
	}
	all := strings.Join(mapValues051(z), " ")
	if !regexp.MustCompile(`(?s)Собственный вес - расчёт.*Временные - расчёт`).MatchString(all) {
		t.Log("references are on distinct sheet parts")
	}
	if !strings.Contains(all, "qр,сниж") {
		t.Fatal("no reduced-load calculation")
	}
}
func mapValues051(m map[string]string) []string {
	v := make([]string, 0, len(m))
	for key, item := range m {
		v = append(v, fmt.Sprintf("%s:%s", key, item))
	}
	return v
}

var _ = os.ErrNotExist
