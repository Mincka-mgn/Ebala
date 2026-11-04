package main

import (
	"fmt"
	"strings"
)

type climateReportRow struct{ Parameter, Zone, Value, Source string }

func siteReportRows(s Site) ([]climateReportRow, error) {
	base, sg, m, mountainOn, err := s.ElevationDetails()
	if err != nil {
		// Incorrect mountain selection cannot block a literal report.
		// The application warns the user; the report keeps their input.
		base, sg, mountainOn = s.SnowGround, s.SnowGround, false
	}
	rows := make([]climateReportRow, 0, 8)
	if s.SnowZone == snowOptionMountain {
		rows = append(rows, climateReportRow{"Снеговое районирование", snowOptionMountain, "", "СП 20.13330.2016, карта 1 приложения Е; Sg определяется отдельно"})
	}
	if sg > 0 {
		snowZone := s.SnowZone
		snowNote := "СП 20.13330.2016, п. 10.2, табл. 10.1; карта 1 приложения Е"
		if specialSnowSite(s) {
			snowZone = snowOptionK
			snowNote = "СП 20.13330.2016, приложение К"
		} else if strings.Contains(strings.ToLower(s.SnowNote), "вручную") || strings.Contains(strings.ToLower(s.SnowNote), "пользоват") {
			snowNote = "Принято пользователем"
		}
		if mountainOn {
			snowZone = m.Zone
			snowNote = "СП 20.13330.2016, таблица Е.1, примечание 1 к карте 1"
			if s.ElevationM <= m.H0 {
				snowNote += " (без повышения)"
			}
		}
		rows = append(rows, climateReportRow{"Вес снегового покрова Sg", snowZone, decimalInput(sg, 3) + " кПа", snowNote})
	}
	if w, ok := windForZone(s.WindZone); ok {
		rows = append(rows, climateReportRow{"Базовое давление ветра w₀", windDisplayZone(s.WindZone), decimalInput(w, 3) + " кПа", "СП 20.13330.2016, п. 11.1.4, табл. 11.1; карта 2 приложения Е"})
	}
	if b, ok := iceForZone(s.IceZone); ok {
		rows = append(rows, climateReportRow{"Толщина стенки гололёда b", s.IceZone, decimalInput(b, 2) + " мм", "СП 20.13330.2016, п. 12.2, табл. 12.1; карта 3 приложения Е"})
	}
	if s.ClimateJanuarySet {
		rows = append(rows, climateReportRow{"Среднемесячная температура января", "", decimalInput(s.ClimateJanuary, 1) + " °C", snowClimateReference(s)})
	}
	if s.ClimateWindSet {
		rows = append(rows, climateReportRow{"Средняя скорость ветра холодного периода (t ≤ 8 °C)", "", decimalInput(s.ClimateWind, 1) + " м/с", snowClimateReference(s)})
	}
	if strings.TrimSpace(s.SeismicMap) != "" && strings.TrimSpace(s.SeismicPoints) != "" && strings.TrimSpace(s.SeismicPoints) != "-" {
		ref := "Принято пользователем"
		if strings.TrimSpace(s.SeismicRefPlace) != "" {
			ref = "СП 14.13330.2018, п. 4.3, приложение А; по населённому пункту " + s.SeismicRefPlace
			if s.SeismicRefRegion != "" && !canonicalRegionMatch(s.SeismicRefRegion, s.Region) {
				ref += " (" + s.SeismicRefRegion + ")"
			}
		}
		rows = append(rows, climateReportRow{"Сейсмичность ОСР-2015", "Карта " + s.SeismicMap, seismicReportIntensity(s.SeismicPoints), ref})
	}
	if s.MountainID != "" {
		if !mountainOn {
			m, _ = mountainByID(s.MountainID)
		}
		caption := m.Caption
		if caption == "" {
			caption = s.MountainID
		}
		rows = append(rows, climateReportRow{"Горная территория", "", caption, "СП 20.13330.2016, таблица Е.1"})
	}
	// A manually entered elevation is part of the user's original data.
	// It appears in the report regardless of the mountain correction switch.
	if raw := s.ElevationText(); strings.TrimSpace(raw) != "" {
		rows = append(rows, climateReportRow{"Абсолютная отметка h, м", "", raw, ""})
	}
	if s.MountainID != "" && mountainOn && sg > 0 {
		rows = append(rows, climateReportRow{"Начальная отметка h₀", "", decimalInput(m.H0, 0) + " м", "Таблица Е.1"})
		rows = append(rows, climateReportRow{"Высотный коэффициент kₕ", "", decimalInput(m.Kh, 2), "Таблица Е.1"})
		rows = append(rows, climateReportRow{"Исходное Sg", "", decimalInput(base, 3) + " кПа", "Таблица 10.1; снеговой район таблицы Е.1"})
	}

	return rows, nil
}

// Used by the Windows confirmation dialog. Reports themselves never
// include this validation or any diagnostic warning.
func siteReportHasInput(s Site) bool {
	return strings.TrimSpace(s.Region) != "" || strings.TrimSpace(s.District) != "" ||
		strings.TrimSpace(s.Place) != "" || strings.TrimSpace(s.SnowZone) != "" ||
		strings.TrimSpace(s.WindZone) != "" || strings.TrimSpace(s.IceZone) != "" ||
		s.SnowGround != 0 || strings.TrimSpace(s.MountainID) != "" ||
		strings.TrimSpace(s.ElevationText()) != "" || strings.TrimSpace(s.SeismicPoints) != "" || s.ClimateJanuarySet || s.ClimateWindSet
}
func reportHeader(s Site) []string {
	var lines []string
	for _, item := range []struct{ name, value string }{
		{"Субъект РФ", s.Region}, {"Муниципальный район", s.District}, {"Населённый пункт", s.Place},
	} {
		if v := strings.TrimSpace(item.value); v != "" {
			lines = append(lines, item.name+": "+v)
		}
	}
	return lines
}

// Самостоятельный Excel: в нём нет листов собственного веса и ветрового расчёта.
func ExportSiteExcel(s Site, path string) error {
	rows, err := siteReportRows(s)
	if err != nil {
		return err
	}
	var x strings.Builder
	x.WriteString(xrow(1, cellText("A1", "ХАРАКТЕРИСТИКИ МЕСТНОСТИ", 1)))
	for i, v := range reportHeader(s) {
		idx := 3 + i
		x.WriteString(xrow(idx, cellText(fmt.Sprintf("A%d", idx), v, 0)))
	}
	x.WriteString(xrow(7, cellText("A7", "Характеристика", 2), cellText("B7", "Район", 2), cellText("C7", "Значение", 2), cellText("D7", "Основание", 2)))
	for i, r := range rows {
		n := i + 8
		cells := []string{cellText(fmt.Sprintf("A%d", n), r.Parameter, 0), cellText(fmt.Sprintf("B%d", n), r.Zone, 0), cellText(fmt.Sprintf("C%d", n), r.Value, 0), cellText(fmt.Sprintf("D%d", n), r.Source, 0)}
		if r.Parameter == "Вес снегового покрова Sg" {
			_, v, _, on, e := s.ElevationDetails()
			if e == nil && on {
				cells[2] = cellFormula(fmt.Sprintf("C%d", n), "B60", v, 3)
			}
		}
		x.WriteString(xrow(n, cells...))
	}
	if s.MountainID != "" {
		base, sg, m, on, e := s.ElevationDetails()
		if e == nil && on {
			// Only a numerically valid input is used in Excel formulas.
			x.WriteString(xrow(55, cellText("A55", "ПРОВЕРЯЕМАЯ ФОРМУЛА Е.1", 1)))
			x.WriteString(xrow(56, cellText("A56", "Отметка h, м", 0), cellNum("B56", s.ElevationM, 3)))
			x.WriteString(xrow(57, cellText("A57", "Начальная отметка h₀, м", 0), cellNum("B57", m.H0, 3)))
			x.WriteString(xrow(58, cellText("A58", "Коэффициент kₕ", 0), cellNum("B58", m.Kh, 3)))
			x.WriteString(xrow(59, cellText("A59", "Исходное Sg, кПа", 0), cellNum("B59", base, 3)))
			x.WriteString(xrow(60, cellText("A60", "Sg(h), кПа", 0), cellFormula("B60", "IF(B56<=B57,B59,B59*(1+0.001*B58*(B56-B57)))", sg, 3)))
			// Значение в таблице выше уже привязано к формуле.
		}
	}
	entries := map[string]string{
		"[Content_Types].xml":        `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/></Types>`,
		"_rels/.rels":                `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Местность" sheetId="1" r:id="rId1"/></sheets><calcPr calcId="191029" fullCalcOnLoad="1" forceFullCalc="1" calcMode="auto"/></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`,
		"xl/styles.xml":              `<?xml version="1.0" encoding="UTF-8"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><numFmts count="1"><numFmt numFmtId="164" formatCode="0.0000"/></numFmts><fonts count="2"><font><sz val="10"/><name val="Arial"/></font><font><b/><sz val="10"/><name val="Arial"/></font></fonts><fills count="3"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill><fill><patternFill patternType="solid"><fgColor rgb="FFE8EEF5"/><bgColor indexed="64"/></patternFill></fill></fills><borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders><cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs><cellXfs count="5"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="2" borderId="0" xfId="0"/><xf numFmtId="164" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/><xf numFmtId="164" fontId="1" fillId="2" borderId="0" xfId="0" applyNumberFormat="1"/></cellXfs><cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles></styleSheet>`,
		"xl/worksheets/sheet1.xml":   sheetXML(x.String(), []float64{38, 18, 24, 84}),
	}
	return packageZip(path, entries)
}

func ExportSiteWord(s Site, path string) error {
	rows, err := siteReportRows(s)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(pBold("Характеристики местности"))
	for _, h := range reportHeader(s) {
		b.WriteString(pText(h))
	}
	b.WriteString(`<w:tbl><w:tblPr><w:tblBorders>`)
	for _, side := range []string{"top", "left", "bottom", "right", "insideH", "insideV"} {
		b.WriteString(fmt.Sprintf(`<w:%s w:val="single" w:sz="6" w:color="000000"/>`, side))
	}
	b.WriteString(`</w:tblBorders></w:tblPr><w:tblGrid>`)
	widths := []int{2650, 1050, 1550, 3890}
	for _, w := range widths {
		b.WriteString(fmt.Sprintf(`<w:gridCol w:w="%d"/>`, w))
	}
	b.WriteString(`</w:tblGrid>`)
	mkrow := func(parts ...string) {
		b.WriteString("<w:tr>")
		for i, p := range parts {
			b.WriteString(tableCell(p, widths[i]))
		}
		b.WriteString("</w:tr>")
	}
	mkrow("Параметр", "Район", "Значение", "Нормативное основание")
	for _, r := range rows {
		// Full report table: climatic values, plus the user's elevation
		// even when the height correction is not selected.
		mkrow(r.Parameter, r.Zone, r.Value, r.Source)
	}
	b.WriteString(`</w:tbl>`)
	if s.MountainID != "" {
		base, v, m, on, _ := s.ElevationDetails()
		if on && s.ElevationM > m.H0 {
			b.WriteString(pText("Уточнение веса снегового покрова по высоте (СП 20.13330.2016, таблица Е.1):"))
			b.WriteString(pMath("Sg(h) = Sg × [1 + 0,001 × kh × (h − h0)]"))
			b.WriteString(pText(fmt.Sprintf("Sg(h) = %s × [1 + 0,001 × %s × (%s − %s)] = %s кПа.", decimalInput(base, 3), decimalInput(m.Kh, 2), decimalInput(s.ElevationM, 2), decimalInput(m.H0, 0), decimalInput(v, 3))))
		}
	}

	doc := `<?xml version="1.0" encoding="UTF-8"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math"><w:body>` + b.String() + `<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1134" w:right="963" w:bottom="1134" w:left="1134"/></w:sectPr></w:body></w:document>`
	entries := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/></Types>`,
		"_rels/.rels":         `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`,
		"word/document.xml":   doc,
		"word/styles.xml":     `<?xml version="1.0" encoding="UTF-8"?><w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="20"/></w:rPr></w:style></w:styles>`,
	}
	return packageZip(path, entries)
}
