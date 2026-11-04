package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func esc(s string) string   { var b bytes.Buffer; xml.EscapeText(&b, []byte(s)); return b.String() }
func xnum(x float64) string { return strconv.FormatFloat(x, 'f', -1, 64) }
func packageZip(path string, entries map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := zip.NewWriter(f)
	keys := []string{"[Content_Types].xml", "_rels/.rels", "docProps/app.xml", "docProps/core.xml", "xl/workbook.xml", "xl/_rels/workbook.xml.rels", "xl/styles.xml", "xl/worksheets/sheet1.xml", "xl/worksheets/sheet2.xml", "word/document.xml", "word/styles.xml"}
	visited := map[string]bool{}
	for _, k := range keys {
		if s, ok := entries[k]; ok {
			x, e := w.Create(k)
			if e != nil {
				w.Close()
				f.Close()
				return e
			}
			if _, e = x.Write([]byte(s)); e != nil {
				w.Close()
				f.Close()
				return e
			}
			visited[k] = true
		}
	}
	for k, s := range entries {
		if visited[k] {
			continue
		}
		x, e := w.Create(k)
		if e != nil {
			w.Close()
			f.Close()
			return e
		}
		if _, e = x.Write([]byte(s)); e != nil {
			w.Close()
			f.Close()
			return e
		}
	}
	if err = w.Close(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
func cellText(addr, s string, style int) string {
	return fmt.Sprintf(`<c r="%s" s="%d" t="inlineStr"><is><t>%s</t></is></c>`, addr, style, esc(s))
}
func cellNum(addr string, v float64, style int) string {
	return fmt.Sprintf(`<c r="%s" s="%d"><v>%s</v></c>`, addr, style, xnum(v))
}
func cellFormula(addr, formula string, v float64, style int) string {
	return fmt.Sprintf(`<c r="%s" s="%d"><f>%s</f><v>%s</v></c>`, addr, style, esc(formula), xnum(v))
}
func xrow(idx int, cells ...string) string {
	return fmt.Sprintf(`<row r="%d">%s</row>`, idx, strings.Join(cells, ""))
}
func sheetXML(rows string, widths []float64) string {
	var cols strings.Builder
	for i, w := range widths {
		fmt.Fprintf(&cols, `<col min="%d" max="%d" width="%.1f" customWidth="1"/>`, i+1, i+1, w)
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetViews><sheetView showGridLines="0" workbookViewId="0"/></sheetViews><sheetFormatPr defaultRowHeight="17"/><cols>` + cols.String() + `</cols><sheetData>` + rows + `</sheetData></worksheet>`
}
func ExportExcel(p Project, path string) error {
	r, e := p.Calc()
	if e != nil {
		return e
	}
	var a strings.Builder
	a.WriteString(xrow(1, cellText("A1", "СОБСТВЕННЫЙ ВЕС МНОГОСЛОЙНОЙ КОНСТРУКЦИИ", 1)))
	a.WriteString(xrow(2, cellText("A2", "Слой", 2), cellText("B2", "Толщина, м", 2), cellText("C2", "Уд. вес, кН/м³", 2), cellText("D2", "gн, кПа", 2), cellText("E2", "γf", 2), cellText("F2", "gр, кПа", 2), cellText("G2", "gн, кгс/м²", 2), cellText("H2", "gр, кгс/м²", 2), cellText("J2", "1 кПа в кгс/м²", 2)))
	for i, l := range p.Layers {
		j := i + 3
		a.WriteString(xrow(j, cellText(fmt.Sprintf("A%d", j), l.Name, 0), cellNum(fmt.Sprintf("B%d", j), l.Thickness, 3), cellNum(fmt.Sprintf("C%d", j), l.UnitWeight, 3), cellFormula(fmt.Sprintf("D%d", j), fmt.Sprintf("B%d*C%d", j, j), l.Thickness*l.UnitWeight, 3), cellNum(fmt.Sprintf("E%d", j), l.GammaF, 3), cellFormula(fmt.Sprintf("F%d", j), fmt.Sprintf("D%d*E%d", j, j), l.Thickness*l.UnitWeight*l.GammaF, 3), cellFormula(fmt.Sprintf("G%d", j), fmt.Sprintf("D%d*$J$3", j), l.Thickness*l.UnitWeight*kgfPerKN, 3), cellFormula(fmt.Sprintf("H%d", j), fmt.Sprintf("F%d*$J$3", j), l.Thickness*l.UnitWeight*l.GammaF*kgfPerKN, 3), func() string {
			if i == 0 {
				return cellNum("J3", kgfPerKN, 3)
			}
			return ""
		}()))
	}
	last := len(p.Layers) + 2
	total := last + 1
	a.WriteString(xrow(total, cellText(fmt.Sprintf("A%d", total), "ИТОГО", 2), cellFormula(fmt.Sprintf("D%d", total), fmt.Sprintf("SUM(D3:D%d)", last), r.SWN, 4), cellFormula(fmt.Sprintf("F%d", total), fmt.Sprintf("SUM(F3:F%d)", last), r.SWD, 4), cellFormula(fmt.Sprintf("G%d", total), fmt.Sprintf("D%d*$J$3", total), r.SWN*kgfPerKN, 4), cellFormula(fmt.Sprintf("H%d", total), fmt.Sprintf("F%d*$J$3", total), r.SWD*kgfPerKN, 4)))
	a.WriteString(xrow(total+2, cellText(fmt.Sprintf("A%d", total+2), "gн = Σ(tᵢ · γᵢ); gр = Σ(tᵢ · γᵢ · γf,ᵢ)", 0)))
	a.WriteString(xrow(total+3, cellText(fmt.Sprintf("A%d", total+3), "Все коэффициенты задаются пользователем.", 0)))
	// In 0.04 the new weight module writes a richer takeoff with direct-area loads.
	// Other sheets keep the 0.03 layout, links and formulas unchanged.
	if p.Weight != nil {
		a.Reset()
		a.WriteString(xrow(1, cellText("A1", "СОБСТВЕННЫЙ ВЕС: "+p.Weight.Name, 1)))
		a.WriteString(xrow(2, cellText("A2", "Наименование", 2), cellText("B2", "Способ ввода", 2), cellText("C2", "Толщина, мм", 2), cellText("D2", "Плотность, кг/м³", 2), cellText("E2", "gн, кгс/м²", 2), cellText("F2", "γf", 2), cellText("G2", "gр, кгс/м²", 2), cellText("H2", "gн, кПа", 2), cellText("I2", "gр, кПа", 2)))
		for i, l := range p.Weight.Rows {
			j := i + 3
			row := []string{cellText(fmt.Sprintf("A%d", j), l.Name, 0)}
			if l.Kind == "layer" {
				row = append(row, cellText(fmt.Sprintf("B%d", j), "По толщине", 0), cellNum(fmt.Sprintf("C%d", j), l.Thickness, 3), cellNum(fmt.Sprintf("D%d", j), l.Density, 3), cellFormula(fmt.Sprintf("E%d", j), fmt.Sprintf("C%d*D%d/1000", j, j), l.NormKgf(), 3))
			} else {
				row = append(row, cellText(fmt.Sprintf("B%d", j), "Готовая нагрузка", 0), cellNum(fmt.Sprintf("E%d", j), l.GNorm, 3))
			}
			row = append(row, cellNum(fmt.Sprintf("F%d", j), l.Gamma, 3), cellFormula(fmt.Sprintf("G%d", j), fmt.Sprintf("E%d*F%d", j, j), l.NormKgf()*l.Gamma, 3), cellFormula(fmt.Sprintf("H%d", j), fmt.Sprintf("E%d*0.00980665", j), kgfM2ToKpa(l.NormKgf()), 3), cellFormula(fmt.Sprintf("I%d", j), fmt.Sprintf("G%d*0.00980665", j), kgfM2ToKpa(l.NormKgf()*l.Gamma), 3))
			a.WriteString(xrow(j, row...))
		}
		t := len(p.Weight.Rows) + 3
		f := func(col string) string {
			if len(p.Weight.Rows) == 0 {
				return "0"
			}
			return fmt.Sprintf("SUM(%s3:%s%d)", col, col, t-1)
		}
		gn, gr, eq := p.Weight.Totals()
		a.WriteString(xrow(t, cellText(fmt.Sprintf("A%d", t), "ИТОГО", 2), cellFormula(fmt.Sprintf("E%d", t), f("E"), gn, 4), cellFormula(fmt.Sprintf("G%d", t), f("G"), gr, 4), cellFormula(fmt.Sprintf("H%d", t), f("H"), kgfM2ToKpa(gn), 4), cellFormula(fmt.Sprintf("I%d", t), f("I"), kgfM2ToKpa(gr), 4)))
		a.WriteString(xrow(t+2, cellText(fmt.Sprintf("A%d", t+2), "Общий коэффициент γf,экв = Σgр / Σgн", 2), cellFormula(fmt.Sprintf("E%d", t+2), fmt.Sprintf(`IF(E%d=0,"",G%d/E%d)`, t, t, t), eq, 4)))
	} else if r.SWN > 0 {
		a.WriteString(xrow(total+5, cellText(fmt.Sprintf("A%d", total+5), "Общий γf,экв = Σgр / Σgн", 2), cellFormula(fmt.Sprintf("D%d", total+5), fmt.Sprintf("IF(D%d=0,0,F%d/D%d)", total, total, total), r.SWD/r.SWN, 4)))
	}
	var w strings.Builder
	w.WriteString(xrow(1, cellText("A1", "ВЕТЕР: ТОЛЬКО УПРОЩЁННАЯ ОЦЕНКА, НЕ РАСЧЁТ ПО СП", 1)))
	w.WriteString(xrow(2, cellText("A2", "Параметр", 2), cellText("B2", "Значение", 2), cellText("C2", "Единица", 2), cellText("D2", "кгс/м²", 2), cellText("F2", "1 кПа в кгс/м²", 2)))
	for i, x := range []struct {
		label string
		value float64
		unit  string
	}{{"Базовое давление w0", p.Wind.W0, "кПа"}, {"Коэффициент высоты k(z)", p.Wind.KZ, "-"}, {"Аэродинамический коэффициент c", p.Wind.C, "-"}, {"Коэффициент надёжности γf", p.Wind.GammaF, "-"}} {
		j := i + 3
		w.WriteString(xrow(j, cellText(fmt.Sprintf("A%d", j), x.label, 0), func() string {
			if i == 0 && !p.WindOverride {
				if _, ok := windForZone(p.Site.WindZone); ok {
					return cellFormula("B3", "'Местность'!B7", p.Wind.W0, 3)
				}
			}
			return cellNum(fmt.Sprintf("B%d", j), x.value, 3)
		}(), cellText(fmt.Sprintf("C%d", j), x.unit, 0), func() string {
			if i == 0 {
				return cellFormula("D3", "B3*$F$3", p.Wind.W0*kgfPerKN, 3)
			}
			return ""
		}(), func() string {
			if i == 0 {
				return cellNum("F3", kgfPerKN, 3)
			}
			return ""
		}()))
	}
	w.WriteString(xrow(7, cellText("A7", "Нормативное давление", 2), cellFormula("B7", "B3*B4*B5", r.WN, 4), cellText("C7", "кПа", 0), cellFormula("D7", "B7*$F$3", r.WN*kgfPerKN, 4)))
	w.WriteString(xrow(8, cellText("A8", "Расчётное давление", 2), cellFormula("B8", "B7*B6", r.WD, 4), cellText("C8", "кПа", 0), cellFormula("D8", "B8*$F$3", r.WD*kgfPerKN, 4)))
	w.WriteString(xrow(10, cellText("A10", "Не учтены пульсации и другие составляющие ветрового воздействия", 0)))
	w.WriteString(xrow(11, cellText("A11", "Источник давления", 0), cellText("B11", func() string {
		if p.WindOverride {
			return "Локальная правка"
		}
		return "Из местности"
	}(), 0)))
	var clim strings.Builder
	clim.WriteString(xrow(1, cellText("A1", "ХАРАКТЕРИСТИКИ МЕСТНОСТИ · СПРАВОЧНЫЕ ИСХОДНЫЕ ДАННЫЕ", 1)))
	for _, v := range []struct {
		row          int
		title, value string
	}{
		{2, "Субъект РФ", p.Site.Region}, {3, "Район", p.Site.District}, {4, "Населённый пункт", p.Site.Place},
		{5, "Снеговой район", p.Site.SnowZone}, {8, "Ветровой район", p.Site.WindZone},
		{9, "Гололёдный район", p.Site.IceZone}, {11, "Снег: источник", p.Site.SnowNote},
		{12, "Источник районов", p.Site.Reference},
	} {
		clim.WriteString(xrow(v.row, cellText(fmt.Sprintf("A%d", v.row), v.title, 0), cellText(fmt.Sprintf("B%d", v.row), v.value, 0)))
	}
	if p.Site.EffectiveSnow() > 0 {
		clim.WriteString(xrow(6, cellText("A6", "Вес снега земли Sg, кПа", 0), cellNum("B6", p.Site.EffectiveSnow(), 3)))
	}
	if v, ok := windForZone(p.Site.WindZone); ok {
		clim.WriteString(xrow(7, cellText("A7", "Ветровое давление w0, кПа", 0), cellNum("B7", v, 3)))
	}
	if v, ok := iceForZone(p.Site.IceZone); ok {
		clim.WriteString(xrow(10, cellText("A10", "Толщина стенки гололёда, мм (минимум для I/V)", 0), cellNum("B10", v, 3)))
	}
	sz, sg, so := p.SnowInputs()
	clim.WriteString(xrow(14, cellText("A14", "Снег в модуле: район", 0), cellText("B14", sz, 0), cellText("C14", func() string {
		if so {
			return "Локально"
		}
		return "Из местности"
	}(), 0)))
	if sg > 0 {
		clim.WriteString(xrow(15, cellText("A15", "Снег в модуле: Sg, кПа", 0), cellNum("B15", sg, 3)))
	}
	iz, iv, io := p.IceInputs()
	clim.WriteString(xrow(16, cellText("A16", "Гололёд в модуле: район", 0), cellText("B16", iz, 0), cellText("C16", func() string {
		if io {
			return "Локально"
		}
		return "Из местности"
	}(), 0)))
	if iz != "" {
		clim.WriteString(xrow(17, cellText("A17", "Гололёд в модуле: b, мм", 0), cellNum("B17", iv, 3)))
	}
	clim.WriteString(xrow(19, cellText("A19", "Зоны по вторичному справочнику требуют уточнения по карте СП", 0)))
	entries := map[string]string{
		"[Content_Types].xml":        `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/worksheets/sheet2.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/worksheets/sheet3.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/><Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/><Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/></Types>`,
		"_rels/.rels":                `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/></Relationships>`,
		"docProps/app.xml":           `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"><Application>Ebala</Application></Properties>`,
		"docProps/core.xml":          `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Ebala: нагрузки</dc:title></cp:coreProperties>`,
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Собственный вес" sheetId="1" r:id="rId1"/><sheet name="Ветер (упрощённо)" sheetId="2" r:id="rId2"/><sheet name="Местность" sheetId="3" r:id="rId4"/></sheets><calcPr calcId="191029" fullCalcOnLoad="1" forceFullCalc="1" calcMode="auto"/></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet2.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/><Relationship Id="rId4" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet3.xml"/></Relationships>`,
		"xl/styles.xml":              `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><numFmts count="1"><numFmt numFmtId="164" formatCode="0.0000"/></numFmts><fonts count="2"><font><sz val="10"/><name val="Arial"/></font><font><b/><sz val="10"/><name val="Arial"/></font></fonts><fills count="3"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill><fill><patternFill patternType="solid"><fgColor rgb="FFE8EEF5"/><bgColor indexed="64"/></patternFill></fill></fills><borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders><cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs><cellXfs count="5"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="2" borderId="0" xfId="0"/><xf numFmtId="164" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/><xf numFmtId="164" fontId="1" fillId="2" borderId="0" xfId="0" applyNumberFormat="1"/></cellXfs><cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles></styleSheet>`,
		"xl/worksheets/sheet1.xml": sheetXML(a.String(), func() []float64 {
			if p.Weight != nil {
				return []float64{48, 23, 16, 22, 20, 12, 20, 18, 18}
			}
			return []float64{38, 17, 22, 17, 12, 18, 19, 19, 3, 23}
		}()),
		"xl/worksheets/sheet2.xml": sheetXML(w.String(), []float64{70, 22, 18, 20, 3, 23}),
		"xl/worksheets/sheet3.xml": sheetXML(clim.String(), []float64{65, 60, 26}),
	}
	return packageZip(path, entries)
}
func pText(s string) string {
	return `<w:p><w:r><w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="20"/></w:rPr><w:t xml:space="preserve">` + esc(s) + `</w:t></w:r></w:p>`
}
func pBold(s string) string {
	return `<w:p><w:pPr><w:keepNext/></w:pPr><w:r><w:rPr><w:b/><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="20"/></w:rPr><w:t>` + esc(s) + `</w:t></w:r></w:p>`
}
func pMath(s string) string {
	return `<w:p><m:oMath><m:r><m:t>` + esc(s) + `</m:t></m:r></m:oMath></w:p>`
}
func tableCell(s string, width int) string {
	return fmt.Sprintf(`<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/></w:tcPr>`, width) + pText(s) + `</w:tc>`
}
func ExportWord(p Project, path string) error {
	r, e := p.Calc()
	if e != nil {
		return e
	}
	var b strings.Builder
	b.WriteString(pBold("ПОВЕРОЧНЫЙ СБОР НАГРУЗОК"))
	b.WriteString(pText("Демонстрационный расчёт. Районирование требует проверки для конкретной площадки."))
	b.WriteString(pBold("Характеристики местности"))
	b.WriteString(pText("Местность: " + formatSitePlace(p.Site.Region, p.Site.District, p.Site.Place)))
	b.WriteString(pText("Снеговой район: " + p.Site.SnowZone + "; Sg = " + displayNumber(p.Site.EffectiveSnow()) + " кПа" + ". " + p.Site.SnowNote))
	b.WriteString(pText("Ветровой район: " + p.Site.WindZone + "; w0 по району: " + func() string {
		v, ok := windForZone(p.Site.WindZone)
		if !ok {
			return "не установлен"
		}
		return displayNumber(v) + " кПа"
	}() + "."))
	b.WriteString(pText("Гололёдный район: " + p.Site.IceZone + ". " + p.Site.Reference))
	if p.Site.MountainID != "" {
		if summary := p.Site.ElevationSummary(); summary != "" {
			b.WriteString(pText(summary))
		} else if raw := p.Site.ElevationText(); raw != "" {
			b.WriteString(pText("Абсолютная отметка h, м (по исходным данным): " + raw))
		}
	}
	sz, sg, so := p.SnowInputs()
	iz, iv, io := p.IceInputs()
	if so {
		b.WriteString(pText("СНЕГ: локально принято " + sz + ", Sg = " + displayNumber(sg) + " кПа."))
	}
	if io {
		b.WriteString(pText("ГОЛОЛЁД: локально принято " + iz + ", b = " + displayNumber(iv) + " мм."))
	}

	if p.Weight != nil {
		b.WriteString(pBold("1. Собственный вес: " + p.Weight.Name))
		b.WriteString(`<w:tbl><w:tblPr><w:tblBorders>`)
		for _, edge := range []string{"top", "left", "bottom", "right", "insideH", "insideV"} {
			b.WriteString(fmt.Sprintf(`<w:%s w:val="single" w:sz="6" w:color="000000"/>`, edge))
		}
		widths := []int{3600, 1100, 1600, 1450, 750, 1650}
		b.WriteString(`</w:tblBorders></w:tblPr><w:tblGrid>`)
		for _, width := range widths {
			b.WriteString(fmt.Sprintf(`<w:gridCol w:w="%d"/>`, width))
		}
		b.WriteString(`</w:tblGrid>`)
		writeRow := func(v ...string) {
			b.WriteString("<w:tr>")
			for i, s := range v {
				b.WriteString(tableCell(s, widths[i]))
			}
			b.WriteString("</w:tr>")
		}
		writeRow("Наименование", "t, мм", "ρ, кг/м³", "gн, кгс/м²", "γf", "gр, кгс/м²")
		for _, l := range p.Weight.Rows {
			tp, ds := "—", "—"
			if l.Kind == "layer" {
				tp = decimalInput(l.Thickness, 2)
				ds = decimalInput(l.Density, 2)
			}
			writeRow(l.Name, tp, ds, decimalInput(l.NormKgf(), 3), decimalInput(l.Gamma, 3), decimalInput(l.NormKgf()*l.Gamma, 3))
		}
		gn, gr, eq := p.Weight.Totals()
		writeRow("ИТОГО", "", "", decimalInput(gn, 3), "", decimalInput(gr, 3))
		b.WriteString(`</w:tbl>`)
		b.WriteString(pText("Нормативная нагрузка: " + decimalInput(gn, 3) + " кгс/м² = " + displayNumber(r.SWN) + " кПа."))
		b.WriteString(pText("Расчётная нагрузка: " + decimalInput(gr, 3) + " кгс/м² = " + displayNumber(r.SWD) + " кПа."))
		if gn > 0 {
			b.WriteString(pBold("Общий коэффициент надёжности γf,экв = " + decimalInput(eq, 3)))
		}
		// Detailed formulas remain in Excel, not in the client-facing Word takeoff.
	} else {
		b.WriteString(pBold("1. Собственный вес многослойной конструкции"))
		// The takeoff table already contains source geometry, loads and totals.
		b.WriteString(`<w:tbl><w:tblPr><w:tblBorders>`)
		for _, edge := range []string{"top", "left", "bottom", "right", "insideH", "insideV"} {
			b.WriteString(fmt.Sprintf(`<w:%s w:val="single" w:sz="6" w:color="000000"/>`, edge))
		}
		widths := []int{2400, 1100, 1600, 1450, 850, 1450}
		b.WriteString(`</w:tblBorders></w:tblPr><w:tblGrid>`)
		for _, w := range widths {
			b.WriteString(fmt.Sprintf(`<w:gridCol w:w="%d"/>`, w))
		}
		b.WriteString(`</w:tblGrid>`)
		writeRow := func(fields ...string) {
			b.WriteString("<w:tr>")
			for i, s := range fields {
				b.WriteString(tableCell(s, widths[i]))
			}
			b.WriteString("</w:tr>")
		}
		writeRow("Слой", "t, м", "γ, кН/м³", "gн, кПа", "γf", "gр, кПа")
		for _, l := range p.Layers {
			writeRow(l.Name, displayNumber(l.Thickness), displayNumber(l.UnitWeight), displayNumber(l.Thickness*l.UnitWeight), displayNumber(l.GammaF), displayNumber(l.Thickness*l.UnitWeight*l.GammaF))
		}
		writeRow("ИТОГО", "", "", displayNumber(r.SWN), "", displayNumber(r.SWD))
		b.WriteString(`</w:tbl>`)
		b.WriteString(pText("Нормативная нагрузка: " + displayNumber(r.SWN) + " кПа = " + decimalInput(kpaToKgfM2(r.SWN), 2) + " кгс/м²."))
		b.WriteString(pText("Расчётная нагрузка: " + displayNumber(r.SWD) + " кПа = " + decimalInput(kpaToKgfM2(r.SWD), 2) + " кгс/м²."))
		if r.SWN > 0 {
			b.WriteString(pBold("Общий коэффициент надёжности γf,экв = " + decimalInput(r.SWD/r.SWN, 3)))
		}
	}
	b.WriteString(pBold("2. Ветер: упрощённая демонстрационная схема"))
	if p.WindOverride {
		b.WriteString(pText("w0 принято вручную в модуле ветра."))
	} else {
		b.WriteString(pText("w0 перенесено из характеристик местности."))
	}
	b.WriteString(pText("ВНИМАНИЕ: расчёт НЕ является полной методикой определения ветровых воздействий по СП 20.13330.2016. Пульсации и аэродинамические особенности не учитываются."))
	b.WriteString(pMath("wн = w₀ · k(z) · c"))
	b.WriteString(pMath("wр = wн · γf"))
	b.WriteString(pText("wн = " + displayNumber(p.Wind.W0) + " × " + displayNumber(p.Wind.KZ) + " × " + displayNumber(p.Wind.C) + " = " + displayNumber(r.WN) + " кПа = " + decimalInput(kpaToKgfM2(r.WN), 2) + " кгс/м²."))
	b.WriteString(pText("wр = " + displayNumber(r.WN) + " × " + displayNumber(p.Wind.GammaF) + " = " + displayNumber(r.WD) + " кПа = " + decimalInput(kpaToKgfM2(r.WD), 2) + " кгс/м²."))
	b.WriteString(pText("Перевод единиц при стандартном ускорении свободного падения: 1 кПа = 101,97162 кгс/м²."))
	b.WriteString(pBold("3. Нормативная основа и ограничения"))
	b.WriteString(pText("Для будущих нормативных модулей используется СП 20.13330.2016 «Нагрузки и воздействия». В данной тестовой версии нормативные коэффициенты автоматически не назначаются; введённые значения требуют проверки."))
	doc := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math"><w:body>` + b.String() + `<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1134" w:right="963" w:bottom="1134" w:left="1134"/></w:sectPr></w:body></w:document>`
	entries := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/><Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/></Types>`,
		"_rels/.rels":         `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/></Relationships>`,
		"word/document.xml":   doc,
		"word/styles.xml":     `<?xml version="1.0" encoding="UTF-8"?><w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="20"/></w:rPr></w:style></w:styles>`,
		"docProps/core.xml":   `<?xml version="1.0" encoding="UTF-8"?><cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Поверочный сбор нагрузок</dc:title></cp:coreProperties>`,
	}
	return packageZip(path, entries)
}
