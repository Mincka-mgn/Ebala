package main

import (
	"html"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func Test052MainLoadsReferencesIndependentVariant(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "gable"
	s.Angle = 20
	var err error
	s, err = SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "gable.xlsx")
	if err = ExportSnowExcel(s, p); err != nil {
		t.Fatal(err)
	}
	z := excel051Parts(t, p)
	main := html.UnescapeString(z["xl/worksheets/sheet1.xml"])
	if !strings.Contains(main, "'Перечень загружений'!E3") || !strings.Contains(main, "'Перечень загружений'!F3") {
		t.Fatal("main sheet does not use the live variant catalogue")
	}
	catalogue := html.UnescapeString(strings.Join(mapValues051(z), " "))
	if !strings.Contains(catalogue, "'Вариант 1'!$C$5:") || !strings.Contains(catalogue, "'Вариант 1'!$D$5:") {
		t.Fatal("variant catalogue lacks live μ and load formulas")
	}
	if !strings.Contains(z["xl/workbook.xml"], "График 1") {
		t.Fatal("missing live chart")
	}
	live := false
	for key, data := range z {
		if strings.HasPrefix(key, "xl/charts/") && strings.Contains(html.UnescapeString(data), "'Вариант 1'!$D$5") {
			live = true
		}
	}
	if !live {
		t.Fatal("no OOXML chart referencing formula cells")
	}
}

func Test052SkylightPartialVariantsHaveFormulas(t *testing.T) {
	for _, span := range []float64{30, 72} {
		s := defaultSnowCalc()
		s.Shape = "skylight"
		s.Span = span
		s.SkylightWidth = 6
		s.H = 2.4
		s.Angle = 0
		calc, err := SnowCompute(s)
		if err != nil {
			t.Fatal(err)
		}
		partial := 0
		wide := 0
		for i, c := range calc.Cases {
			if strings.Contains(c.Title, "Частичное загружение у") {
				partial++
			}
			if strings.Contains(c.Title, "Дополнительное местное загружение у") {
				wide++
			}
			body := snowExcelVariantData(calc, c, i)
			if strings.Contains(body, "NA()") {
				t.Fatalf("no formula for variant %s", c.Title)
			}
		}
		if partial != 2 {
			t.Fatalf("span %v: expected two partial cases, got %d", span, partial)
		}
		if span > 60 && wide == 0 {
			t.Fatal("wide-slope local drift not generated")
		}
		path := filepath.Join(t.TempDir(), "lantern.xlsx")
		if err = ExportSnowExcel(calc, path); err != nil {
			t.Fatal(err)
		}
		z := excel051Parts(t, path)
		all := html.UnescapeString(strings.Join(mapValues051(z), " "))
		if strings.Contains(all, "NA()") {
			t.Fatal("unimplemented NA formula in lantern XLSX")
		}
		if span > 60 && !strings.Contains(all, "$B$54") {
			t.Fatal("wide B.8 lantern drift not linked to own coefficients")
		}
	}
}

func Test052HighDomeUsesPhysicalCentre(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "dome"
	s.Span = 18
	s.Rise = 12
	s.RoofHeight = 20
	var err error
	s, err = SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	form, _ := snowExcelMuFormula(s, s.Cases[0], s.Cases[0].Intervals[20], 25)
	if !strings.Contains(form, "'Исходные данные'!$B$14>") {
		t.Fatalf("dome formula not selecting physical centre: %s", form)
	}
	path := filepath.Join(t.TempDir(), "high_dome.xlsx")
	if err = ExportSnowExcel(s, path); err != nil {
		t.Fatal(err)
	}
	z := excel051Parts(t, path)
	all := html.UnescapeString(strings.Join(mapValues051(z), " "))
	if !strings.Contains(all, "Фактическая горизонтальная проекция") {
		t.Fatal("missing formula-driven 2R projection")
	}
	if !strings.Contains(all, "B47") {
		t.Fatal("coordinate grid does not refer to computed projection")
	}
}

func Test052CombinedWeightRetainsLayerInputs(t *testing.T) {
	wc := WeightCalc{Name: "Перекрытие", Rows: []WeightRow{{Name: "Плита", Kind: "layer", Thickness: 60, Density: 1800, Gamma: 1.3}, {Name: "Обмазка", Kind: "direct", GNorm: 13, Gamma: 1.1}}}
	r := TempReport{Version: 1, WeightBlocks: []ProjectWeightBlock{{Title: "Перекрытие", Rows: []ProjectWeightLine{{Name: "Плита", Norm: kgfM2ToKpa(108), Gamma: 1.3}, {Name: "Обмазка", Norm: kgfM2ToKpa(13), Gamma: 1.1}}}}, WeightInputs: []WeightReportInput{{Title: "Перекрытие", Rows: wc.Rows}}}
	path := filepath.Join(t.TempDir(), "weight.xlsx")
	if e := exportTempExcel(r, path); e != nil {
		t.Fatal(e)
	}
	z := excel051Parts(t, path)
	all := html.UnescapeString(strings.Join(mapValues051(z), " "))
	if !strings.Contains(all, "Толщина, мм") || !strings.Contains(all, "1800") || !strings.Contains(all, "D3*'Собственный вес - исходные'!E3") {
		t.Fatal("weight from thickness/density not reconstructible")
	}
}

func Test052TransportFormulaSheets(t *testing.T) {
	r := TempReport{Version: 1, Sections: []TempSelection{
		{Group: "СП 35 · транспорт", Title: "АК-14", Source: "СП 35", Params: map[string]interface{}{"K": 14.}},
		{Group: "СП 35 · транспорт", Title: "НК-14", Source: "СП 35", Params: map[string]interface{}{"K": 14.}},
		{Group: "СП 35 · транспорт", Title: "АБ-51", Source: "СП 35", Params: map[string]interface{}{"gamma_selected": 1.2}},
	}}
	path := filepath.Join(t.TempDir(), "transport.xlsx")
	if e := exportTempExcel(r, path); e != nil {
		t.Fatal(e)
	}
	z := excel051Parts(t, path)
	wb := z["xl/workbook.xml"]
	if !strings.Contains(wb, "Транспорт - исходные") || !strings.Contains(wb, "Транспорт - расчёт") {
		t.Fatal("transport not independent")
	}
	all := html.UnescapeString(strings.Join(mapValues051(z), " "))
	if !strings.Contains(all, "10*'Транспорт - исходные'!C3") || !strings.Contains(all, "18*'Транспорт - исходные'!C4") {
		t.Fatal("AK/NK numeric formula missing")
	}
}

func Test052SphereConePolarGridComputedFromInput(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "sphere_cone"
	s.Span = 18
	s.Width = 18
	s.SkylightWidth = 6
	s.Angle = 18
	s.H = 3.131
	s.RoofHeight = 12
	s.BetaStep = 15
	var err error
	s, err = SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sphere.xlsx")
	if err = ExportSnowExcel(s, path); err != nil {
		t.Fatal(err)
	}
	z := excel051Parts(t, path)
	all := html.UnescapeString(strings.Join(mapValues051(z), " "))
	if !strings.Contains(all, "$B$33") || !strings.Contains(all, "'Геометрия Б.15а'!$B$5") {
		t.Fatal("polar points do not follow entered beta step/cone radius")
	}
	// Sheet 1 must link all μ through variant and no generated formula may use NA() as a placeholder.
	if regexp.MustCompile(`(?m)<f>NA\(\)</f>`).MatchString(all) {
		t.Fatal("unimplemented μ is silently exported")
	}
}
