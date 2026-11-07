package main

import (
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func tempXML(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(s))
	return b.String()
}

// A literal marker distinguishes a user-entered name (possibly beginning
// with '=') from the actual formula expressions written by our generator.
// The marker is removed before it reaches OOXML; no XML control bytes leak.
func tempExcelLiteral(value string) string { return "\x00" + value }

func tempExcelCell(row, col int, value string) string {
	if strings.HasPrefix(value, "\x00") {
		// A name remains a string even when it looks numeric or starts '='.
		return fmt.Sprintf(`<c r="%c%d" t="inlineStr"><is><t>%s</t></is></c>`, rune(65+col), row, tempXML(strings.TrimPrefix(value, "\x00")))
	}
	if strings.HasPrefix(value, "=") {
		return fmt.Sprintf(`<c r="%c%d"><f>%s</f><v></v></c>`, rune(65+col), row, tempXML(strings.TrimPrefix(value, "=")))
	}
	if n, e := strconv.ParseFloat(strings.TrimSpace(value), 64); e == nil && !math.IsNaN(n) && !math.IsInf(n, 0) {
		return fmt.Sprintf(`<c r="%c%d"><v>%s</v></c>`, rune(65+col), row, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return fmt.Sprintf(`<c r="%c%d" t="str"><v>%s</v></c>`, rune(65+col), row, tempXML(value))
}

// tempExcelInputRow writes a user/accepted SOURCE row. Unlike tempExcelRow,
// user-entered strings are never interpreted as Excel formulas. This keeps
// input sheets independent of precomputed Go values and prevents formula
// injection via a user-defined layer, load or crane name beginning with "=".
// Numeric inputs remain numeric cells so Excel formulas can reference them.
func tempExcelInputCell(row, col int, value string) string {
	address := fmt.Sprintf("%c%d", rune(65+col), row)
	if number, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && !math.IsNaN(number) && !math.IsInf(number, 0) {
		return fmt.Sprintf(`<c r="%s"><v>%s</v></c>`, address, strconv.FormatFloat(number, 'f', -1, 64))
	}
	return fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, address, tempXML(value))
}

func tempExcelInputRow(row int, cells ...string) string {
	if row == 1 && len(cells) > 0 && !strings.HasPrefix(cells[0], "На этом листе") {
		cells[0] = "На этом листе приведены исходные данные: " + cells[0]
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<row r="%d">`, row))
	for col, value := range cells {
		b.WriteString(tempExcelInputCell(row, col, value))
	}
	b.WriteString(`</row>`)
	return b.String()
}

func tempExcelRow(row int, cells ...string) string {
	if row == 1 && len(cells) > 0 && !strings.HasPrefix(cells[0], "На этом листе") {
		cells[0] = "На этом листе приведены данные раздела: " + cells[0]
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<row r="%d">`, row))
	for i, c := range cells {
		b.WriteString(tempExcelCell(row, i, c))
	}
	b.WriteString(`</row>`)
	return b.String()
}
func exportTempExcel(r TempReport, path string) error {
	if e := validateTempReport(r); e != nil {
		return e
	}
	var regular, transport, cranes []TempSelection
	for _, s := range r.Sections {
		if !tempReportInclude(s) || (tempIsCrane(s) && len(s.Wheels) == 0) {
			continue
		}
		if tempIsCrane(s) {
			cranes = append(cranes, s)
		} else if tempTransport(s) {
			transport = append(transport, s)
		} else {
			regular = append(regular, s)
		}
	}
	if len(regular)+len(transport)+len(cranes)+len(r.WeightRows)+len(r.WeightBlocks)+len(r.SiteLines)+len(r.Snows)+len(r.Ices)+len(r.Winds)+len(r.CraneGroups) == 0 && r.Snow == nil && r.Ice == nil {
		return fmt.Errorf("Ведомость пуста")
	}
	var w strings.Builder
	imageCount := 0
	var imageRels, drawing strings.Builder
	imageEntries := map[string]string{}
	rowNum := 0
	add := func(vals ...string) { rowNum++; w.WriteString(tempExcelRow(rowNum, vals...)) }
	// Final results are intentionally placed AFTER source, normative and
	// formula worksheets. This worksheet is a navigation/summary view, not
	// a second place to store Go-computed load values.
	add("На этом листе представлены итоговые нормативные и расчётные нагрузки. Для ветровых зон приведены ссылки на формульные расчёты: их названия и исходные параметры перечислены в оглавлении «Исходные данные».")
	if len(r.SiteLines) > 0 {
		add("ХАРАКТЕРИСТИКИ МЕСТНОСТИ")
		for _, line := range r.SiteLines {
			add(tempExcelLiteral(line))
		}
	}
	if len(r.WeightRows) > 0 || len(r.WeightBlocks) > 0 {
		add("ПОСТОЯННЫЕ НАГРУЗКИ. СОБСТВЕННЫЙ ВЕС")
		ix := 3
		for _, group := range weightReportBlocks(r) {
			add(tempExcelLiteral(group.Title))
			add("№", "Наименование", "Нормативная, кПа", "γf", "Расчётная, кПа")
			begin := ix
			for i, line := range group.Rows {
				add(fmt.Sprint(i+1), tempExcelLiteral(line.Name), fmt.Sprintf("='Собственный вес - расчёт'!C%d", ix), fmt.Sprintf("='Собственный вес - расчёт'!D%d", ix), fmt.Sprintf("='Собственный вес - расчёт'!E%d", ix))
				ix++
			}
			if ix > begin {
				add("", "ИТОГО", fmt.Sprintf("=SUM('Собственный вес - расчёт'!C%d:C%d)", begin, ix-1), "", fmt.Sprintf("=SUM('Собственный вес - расчёт'!E%d:E%d)", begin, ix-1))
			}
		}
	}
	if r.Snow != nil {
		if err := snowExcelAppend(*r.Snow, add, &rowNum, &imageCount, &imageRels, &drawing, imageEntries, ""); err != nil {
			return err
		}
	}
	if r.Ice != nil {
		iceExcelAppend(*r.Ice, add)
	}
	for idx, ice := range r.Ices {
		if idx < len(r.IceTitles) {
			add("Гололёдная нагрузка", tempExcelLiteral(r.IceTitles[idx]))
		}
		iceExcelAppendIndexed(ice, idx+1, add)
	}
	for i, snow := range r.Snows {
		if i < len(r.SnowTitles) {
			add("Снеговая нагрузка", tempExcelLiteral(r.SnowTitles[i]))
		}
		if err := snowExcelAppend(snow, add, &rowNum, &imageCount, &imageRels, &drawing, imageEntries, fmt.Sprintf(" %d", i+1)); err != nil {
			return err
		}
	}
	windDetailed, windDetailedNames := windExpandReports(r.Winds, r.WindTitles)
	for j, title := range windDetailedNames {
		i := j + 1
		src := fmt.Sprintf("'Ветер Исходные %d'!B21", i)
		norm := fmt.Sprintf("'Ветер Расчёт %d'!B36", i)
		design := fmt.Sprintf("'Ветер Расчёт %d'!B37", i)
		f := fmt.Sprintf("IF(%s=\"кгс/м²\",%.12g,0.001)", src, kgfPerKN/1000)
		add("Ветровая нагрузка, в выбранной единице", tempExcelLiteral(title), "="+norm+"*"+f, "="+design+"*"+f)
	}
	_ = windDetailed
	if len(regular) > 0 {
		add("Временные нагрузки (СП 20.13330.2016)")
		add("№", "Наименование", "Нормативная в кгс", "Нормативная в кПа/кН", "γf", "Расчётная в кгс", "Расчётная в кПа/кН", "Основание")
		for i, sel := range regular {
			row := i + 3
			undefined := false
			if v, ok := sel.Params["is_defined"].(bool); ok && !v {
				undefined = true
			}
			if undefined {
				add(fmt.Sprint(i+1), tempExcelLiteral(sel.Title), "По заданию", "", "='Временные - исходные'!C"+strconv.Itoa(row), "По заданию", "", tempExcelLiteral(tempReadableSource(sel.Source)))
				continue
			}
			c := "'Временные - расчёт'!"
			add(fmt.Sprint(i+1), tempExcelLiteral(sel.Title), fmt.Sprintf("=%sG%d", c, row), fmt.Sprintf("=%sD%d", c, row), fmt.Sprintf("=%sE%d", c, row), fmt.Sprintf("=%sH%d", c, row), fmt.Sprintf("=%sF%d", c, row), tempExcelLiteral(tempReadableSource(sel.Source)))
		}
	}
	if len(transport) > 0 {
		add("Транспортные нагрузки (СП 35.13330.2011)")
		add("№", "Транспортная нагрузка", "Общие данные", "Нормативные нагрузки", "γf", "Расчётные нагрузки", "Основание")
		for i, s := range transport {
			r := i + 3
			calc := "'Транспорт - расчёт'!"
			add(fmt.Sprint(i+1), tempExcelLiteral(s.Title), fmt.Sprintf("=%sB%d", calc, r), fmt.Sprintf("=%sC%d", calc, r), fmt.Sprintf("=%sE%d", calc, r), fmt.Sprintf("=%sF%d", calc, r), tempExcelLiteral(tempReadableSource(s.Source)))
		}
	}
	if len(transport) > 0 {
		add("Подробные формулы транспортных схем", "См. листы «Транспорт - исходные» и «Транспорт - расчёт»")
	}
	craneWheelRow := 3
	if len(cranes) > 0 {
		add("Мостовые краны")
		add("№ крана", "Кран", "Тип", "№ колеса", "Pн, кН", "Pн, тс", "x, мм", "Шаг, мм")
		for i, sel := range cranes {
			for j := range sel.Wheels {
				add(fmt.Sprint(i+1), tempExcelLiteral(sel.Title), tempCraneKind(sel), fmt.Sprint(j+1),
					fmt.Sprintf("='Краны - расчёт'!E%d", craneWheelRow),
					fmt.Sprintf("='Краны - расчёт'!F%d", craneWheelRow),
					fmt.Sprintf("='Краны - расчёт'!D%d", craneWheelRow),
					fmt.Sprintf("='Краны - расчёт'!G%d", craneWheelRow))
				craneWheelRow++
			}
		}
	}
	placementIndex := 0
	for _, group := range r.CraneGroups {
		add("МОСТОВЫЕ КРАНЫ", tempExcelLiteral(group.Title))
		add("№ крана", "Кран", "Тип", "№ колеса", "Pн, кН", "Pн, тс", "x, мм", "Шаг, мм")
		for i, c := range group.Cranes {
			for j := range c.Wheels {
				add(fmt.Sprint(i+1), tempExcelLiteral(c.Title), tempCraneKind(c), fmt.Sprint(j+1),
					fmt.Sprintf("='Краны - расчёт'!E%d", craneWheelRow),
					fmt.Sprintf("='Краны - расчёт'!F%d", craneWheelRow),
					fmt.Sprintf("='Краны - расчёт'!D%d", craneWheelRow),
					fmt.Sprintf("='Краны - расчёт'!G%d", craneWheelRow))
				craneWheelRow++
			}
		}
		for _, place := range group.Placements {
			placementIndex++
			add("РАССТАНОВКА КРАНОВ", tempExcelLiteral(place.Title), "Пролёты балки", tempExcelLiteral(place.SpansText))
			add("№", "Опасное положение", "Усилие из формул, кН или кН·м", "Единицы", "Сечение, мм")
			// Numerical optimization supplies the locations, not the effort.
			// The combined workbook repeats the independently calculated
			// influence ordinates and sums using the same four worksheets as
			// the standalone verification file.
			pointRow := 9 + len(place.Result.SpansMM)
			for i, c := range place.Result.Cases {
				caseRow := i + 5
				add(fmt.Sprint(i+1), tempExcelLiteral(c.Name),
					fmt.Sprintf("='%s'!C%d", excelPhase4PlacementSheetName(placementIndex, "Итоги"), caseRow),
					c.Units,
					fmt.Sprintf("='%s'!E%d", excelPhase4PlacementSheetName(placementIndex, "Итоги"), caseRow))
				add("", "Координата колеса (найдена численно), мм", "Принятая Pн, кН", "Кран")
				for _, wheel := range c.Points {
					add("",
						fmt.Sprintf("='%s'!C%d*1000", excelPhase4PlacementSheetName(placementIndex, "Вход"), pointRow),
						fmt.Sprintf("='%s'!D%d", excelPhase4PlacementSheetName(placementIndex, "Вход"), pointRow),
						tempExcelLiteral(wheel.Crane))
					pointRow++
				}
			}
		}
	}

	sheet := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheetFormatPr defaultRowHeight="19"/><cols><col min="1" max="1" width="24" customWidth="1"/><col min="2" max="2" width="58" customWidth="1"/><col min="3" max="7" width="28" customWidth="1"/><col min="8" max="8" width="82" customWidth="1"/></cols><sheetData>` + w.String() + `</sheetData>` + func() string {
		if imageCount > 0 {
			return `<drawing r:id="rId1"/>`
		}
		return ""
	}() + `</worksheet>`
	entries := map[string]string{
		"[Content_Types].xml":        `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/></Types>`,
		"_rels/.rels":                `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Итоговые нагрузки" sheetId="1" r:id="rId1"/></sheets><calcPr calcMode="auto" fullCalcOnLoad="1" forceFullCalc="1"/></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`,
		"xl/worksheets/sheet1.xml":   sheet,
		"xl/styles.xml":              `<?xml version="1.0" encoding="UTF-8"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="1"><font><sz val="11"/><name val="Arial"/></font></fonts><fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills><borders count="1"><border/></borders><cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs><cellXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/></cellXfs><cellStyles count="1"><cellStyle name="Normal" xfId="0"/></cellStyles></styleSheet>`,
	}
	for name, data := range imageEntries {
		entries[name] = data
	}
	if imageCount > 0 {
		entries["[Content_Types].xml"] = strings.Replace(entries["[Content_Types].xml"], `</Types>`, `<Default Extension="png" ContentType="image/png"/><Override PartName="/xl/drawings/drawing1.xml" ContentType="application/vnd.openxmlformats-officedocument.drawing+xml"/></Types>`, 1)
		entries["xl/worksheets/_rels/sheet1.xml.rels"] = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/drawing" Target="../drawings/drawing1.xml"/></Relationships>`
		entries["xl/drawings/drawing1.xml"] = `<?xml version="1.0" encoding="UTF-8"?><xdr:wsDr xmlns:xdr="http://schemas.openxmlformats.org/drawingml/2006/spreadsheetDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` + drawing.String() + `</xdr:wsDr>`
		entries["xl/drawings/_rels/drawing1.xml.rels"] = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + imageRels.String() + `</Relationships>`
	}
	// Every combined report has one entry point for accepted data. Each
	// module owns its actual source rows on clearly named following sheets.
	snowExcelGridStyle(entries)
	if e := excelPhase3InputIndex(entries, r, regular, transport, cranes); e != nil {
		return e
	}
	if e := excelPhase3SiteSheets(entries, r.Site); e != nil {
		return e
	}
	if len(weightReportBlocks(r)) > 0 || len(regular) > 0 || len(transport) > 0 {
		if e := excel051LedgerSheets(entries, r, regular, transport); e != nil {
			return e
		}
	}
	if e := excelPhase3CraneSheets(entries, r, cranes); e != nil {
		return e
	}
	if e := excelPhase4CranePlacementSheets(entries, r); e != nil {
		return e
	}
	if r.Snow != nil {
		if e := snowExcelV031(entries, *r.Snow); e != nil {
			return e
		}
	}
	if r.Ice != nil {
		if r.Snow == nil {
			snowExcelGridStyle(entries)
		}
		if e := iceExcelV038(entries, *r.Ice); e != nil {
			return e
		}
	}
	for i, snow := range r.Snows {
		if e := snowExcelV031(entries, snow, i+1); e != nil {
			return e
		}
	}
	if len(r.Ices) > 0 && r.Snow == nil && len(r.Snows) == 0 {
		snowExcelGridStyle(entries)
	}
	for i, ice := range r.Ices {
		if e := iceExcelV038(entries, ice, i+1); e != nil {
			return e
		}
	}
	if err := windExcelAdd(entries, r.Winds, r.WindTitles); err != nil {
		return err
	}
	snowExcelReorderSheets(entries)
	if err := wind064PolishWorkbook(entries); err != nil {
		return err
	}
	return packageZip(path, entries)
}
