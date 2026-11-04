package main

import (
	"fmt"
	"strings"
)

// ExportWeightExcel is an independent, editable surface-load takeoff for the new module.
func ExportWeightExcel(w WeightCalc, path string) error {
	if err := w.Validate(); err != nil {
		return err
	}
	norm, design, eq := w.Totals()
	var a strings.Builder
	title := "СОБСТВЕННЫЙ ВЕС"
	if strings.TrimSpace(w.Name) != "" {
		title += ": " + w.Name
	}
	a.WriteString(xrow(1, cellText("A1", title, 1)))
	a.WriteString(xrow(2, cellText("A2", "Наименование", 2), cellText("B2", "Способ расчёта", 2), cellText("C2", "t, мм", 2), cellText("D2", "ρ, кг/м³", 2), cellText("E2", "gн, кгс/м²", 2), cellText("F2", "γf", 2), cellText("G2", "gр, кгс/м²", 2), cellText("H2", "gн, кПа", 2), cellText("I2", "gр, кПа", 2)))
	for i, r := range w.Rows {
		n := i + 3
		cs := []string{cellText(fmt.Sprintf("A%d", n), r.Name, 0), cellText(fmt.Sprintf("B%d", n), map[bool]string{true: "По толщине", false: "Готовая нагрузка"}[r.Kind == "layer"], 0)}
		if r.Kind == "layer" {
			cs = append(cs, cellNum(fmt.Sprintf("C%d", n), r.Thickness, 5), cellNum(fmt.Sprintf("D%d", n), r.Density, 5), cellFormula(fmt.Sprintf("E%d", n), fmt.Sprintf("C%d*D%d/1000", n, n), r.NormKgf(), 5))
		} else {
			cs = append(cs, cellNum(fmt.Sprintf("E%d", n), r.GNorm, 5))
		}
		cs = append(cs, cellNum(fmt.Sprintf("F%d", n), r.Gamma, 3), cellFormula(fmt.Sprintf("G%d", n), fmt.Sprintf("E%d*F%d", n, n), r.NormKgf()*r.Gamma, 5), cellFormula(fmt.Sprintf("H%d", n), fmt.Sprintf("E%d*0.00980665", n), kgfM2ToKpa(r.NormKgf()), 3), cellFormula(fmt.Sprintf("I%d", n), fmt.Sprintf("G%d*0.00980665", n), kgfM2ToKpa(r.NormKgf()*r.Gamma), 3))
		a.WriteString(xrow(n, cs...))
	}
	t := len(w.Rows) + 3
	var fd, fg, fh, fi string
	if len(w.Rows) > 0 {
		fd = fmt.Sprintf("SUM(E3:E%d)", t-1)
		fg = fmt.Sprintf("SUM(G3:G%d)", t-1)
		fh = fmt.Sprintf("SUM(H3:H%d)", t-1)
		fi = fmt.Sprintf("SUM(I3:I%d)", t-1)
	} else {
		fd = "0"
		fg = "0"
		fh = "0"
		fi = "0"
	}
	a.WriteString(xrow(t, cellText(fmt.Sprintf("A%d", t), "ИТОГО", 2), cellFormula(fmt.Sprintf("E%d", t), fd, norm, 6), cellFormula(fmt.Sprintf("G%d", t), fg, design, 6), cellFormula(fmt.Sprintf("H%d", t), fh, kgfM2ToKpa(norm), 4), cellFormula(fmt.Sprintf("I%d", t), fi, kgfM2ToKpa(design), 4)))
	a.WriteString(xrow(t+2, cellText(fmt.Sprintf("A%d", t+2), "Общий коэффициент γf = Σgр / Σgн", 2), cellFormula(fmt.Sprintf("E%d", t+2), fmt.Sprintf(`IF(E%d=0,"",G%d/E%d)`, t, t, t), eq, 4)))
	a.WriteString(xrow(t+3, cellText(fmt.Sprintf("A%d", t+3), "gр = gн × γf,экв;  gн = gр / γf,экв", 0)))
	return packageZip(path, map[string]string{
		"[Content_Types].xml":        `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/></Types>`,
		"_rels/.rels":                `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Собственный вес" sheetId="1" r:id="rId1"/></sheets><calcPr calcMode="auto" fullCalcOnLoad="1"/></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`,
		"xl/styles.xml":              `<?xml version="1.0" encoding="UTF-8"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><numFmts count="2"><numFmt numFmtId="164" formatCode="0.0000"/><numFmt numFmtId="165" formatCode="0.0"/></numFmts><fonts count="2"><font><sz val="10"/><name val="Arial"/></font><font><b/><sz val="10"/><name val="Arial"/></font></fonts><fills count="3"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill><fill><patternFill patternType="solid"><fgColor rgb="FFE8EEF5"/></patternFill></fill></fills><borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders><cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs><cellXfs count="7"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="2" borderId="0" xfId="0"/><xf numFmtId="164" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/><xf numFmtId="164" fontId="1" fillId="2" borderId="0" xfId="0" applyNumberFormat="1"/><xf numFmtId="165" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/><xf numFmtId="165" fontId="1" fillId="2" borderId="0" xfId="0" applyNumberFormat="1"/></cellXfs></styleSheet>`,
		"xl/worksheets/sheet1.xml":   sheetXML(a.String(), []float64{48, 23, 15, 18, 20, 12, 20, 19, 19}),
	})
}

func ExportWeightWord(w WeightCalc, path string) error {
	if err := w.Validate(); err != nil {
		return err
	}
	gn, gr, eq := w.Totals()
	var b strings.Builder
	b.WriteString(pBold("РАСЧЁТ СОБСТВЕННОГО ВЕСА"))
	if strings.TrimSpace(w.Name) != "" {
		b.WriteString(pText("Конструкция: " + w.Name))
	}
	b.WriteString(`<w:tbl><w:tblPr><w:tblBorders>`)
	for _, e := range []string{"top", "left", "bottom", "right", "insideH", "insideV"} {
		fmt.Fprintf(&b, `<w:%s w:val="single" w:sz="6" w:color="000000"/>`, e)
	}
	b.WriteString(`</w:tblBorders></w:tblPr><w:tblGrid>`)
	widths := []int{3600, 1100, 1600, 1450, 750, 1650}
	for _, x := range widths {
		fmt.Fprintf(&b, `<w:gridCol w:w="%d"/>`, x)
	}
	b.WriteString(`</w:tblGrid>`)
	add := func(a ...string) {
		b.WriteString(`<w:tr><w:trPr><w:cantSplit/></w:trPr>`)
		for i, v := range a {
			b.WriteString(tableCell(v, widths[i]))
		}
		b.WriteString(`</w:tr>`)
	}
	add("Наименование", "t, мм", "ρ, кг/м³", "gн, кгс/м²", "γf", "gр, кгс/м²")
	for _, r := range w.Rows {
		thick, den := "—", "—"
		if r.Kind == "layer" {
			thick = decimalInput(r.Thickness, 2)
			den = decimalInput(r.Density, 2)
		}
		add(r.Name, thick, den, decimalInput(r.NormKgf(), 1), decimalInput(r.Gamma, 4), decimalInput(r.NormKgf()*r.Gamma, 1))
	}
	add("ИТОГО", "", "", decimalInput(gn, 1), "", decimalInput(gr, 1))
	b.WriteString(`</w:tbl>`)
	b.WriteString(pText("Нормативная нагрузка: " + decimalInput(gn, 1) + " кгс/м² = " + decimalInput(kgfM2ToKpa(gn), 4) + " кПа."))
	b.WriteString(pText("Расчётная нагрузка: " + decimalInput(gr, 1) + " кгс/м² = " + decimalInput(kgfM2ToKpa(gr), 4) + " кПа."))
	if gn > 0 {
		b.WriteString(pBold("Общий коэффициент надёжности по нагрузке γf,экв = " + decimalInput(eq, 3)))
	} else {
		b.WriteString(pText("Общий коэффициент не определяется при нулевой нормативной нагрузке."))
	}
	// The standalone Word table contains all needed input and output data.
	// Detailed formula audit remains in Excel.
	doc := `<?xml version="1.0" encoding="UTF-8"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` + b.String() + `<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="850" w:right="700" w:bottom="850" w:left="700"/></w:sectPr></w:body></w:document>`
	return packageZip(path, map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/></Types>`,
		"_rels/.rels":         `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`,
		"word/document.xml":   doc,
		"word/styles.xml":     `<?xml version="1.0" encoding="UTF-8"?><w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="20"/></w:rPr></w:style></w:styles>`,
	})
}
