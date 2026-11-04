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
func tempExcelCell(row, col int, value string) string {
	if strings.HasPrefix(value, "=") {
		return fmt.Sprintf(`<c r="%c%d"><f>%s</f><v></v></c>`, rune(65+col), row, tempXML(strings.TrimPrefix(value, "=")))
	}
	if n, e := strconv.ParseFloat(strings.TrimSpace(value), 64); e == nil && !math.IsNaN(n) && !math.IsInf(n, 0) {
		return fmt.Sprintf(`<c r="%c%d"><v>%s</v></c>`, rune(65+col), row, strconv.FormatFloat(n, 'f', -1, 64))
	}
	return fmt.Sprintf(`<c r="%c%d" t="str"><v>%s</v></c>`, rune(65+col), row, tempXML(value))
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
	if len(regular)+len(transport)+len(cranes)+len(r.WeightRows)+len(r.SiteLines)+len(r.Snows)+len(r.Ices)+len(r.CraneGroups) == 0 && r.Snow == nil && r.Ice == nil {
		return fmt.Errorf("Ведомость пуста")
	}
	var w strings.Builder
	imageCount := 0
	var imageRels, drawing strings.Builder
	imageEntries := map[string]string{}
	rowNum := 0
	add := func(vals ...string) { rowNum++; w.WriteString(tempExcelRow(rowNum, vals...)) }
	add(func() string {
		if len(r.WeightRows)+len(r.SiteLines) > 0 {
			return "ОБЩАЯ ВЕДОМОСТЬ НАГРУЗОК"
		}
		if r.Snow != nil && r.Ice == nil && len(regular)+len(transport)+len(cranes) == 0 {
			return "РАСЧЁТ СНЕГОВЫХ НАГРУЗОК"
		}
		if r.Ice != nil && r.Snow == nil && len(regular)+len(transport)+len(cranes) == 0 {
			return "РАСЧЁТ ГОЛОЛЁДНЫХ НАГРУЗОК"
		}
		return "ВРЕМЕННЫЕ И ТРАНСПОРТНЫЕ НАГРУЗКИ"
	}())
	if r.Project != "" {
		add("Объект", r.Project)
	}
	if len(r.SiteLines) > 0 {
		add("ХАРАКТЕРИСТИКИ МЕСТНОСТИ")
		for _, line := range r.SiteLines {
			add(line)
		}
	}
	if len(r.WeightRows) > 0 || len(r.WeightBlocks) > 0 {
		add("ПОСТОЯННЫЕ НАГРУЗКИ. СОБСТВЕННЫЙ ВЕС")
		for _, group := range weightReportBlocks(r) {
			add(group.Title)
			add("№", "Наименование", "Нормативная, кПа", "γf", "Расчётная, кПа")
			norm, design := 0.0, 0.0
			for i, x := range group.Rows {
				norm += x.Norm
				design += x.Design
				add(fmt.Sprint(i+1), x.Name, tempRU(x.Norm, 4), tempRU(x.Gamma, 3), tempRU(x.Design, 4))
			}
			add("", "ИТОГО", tempRU(norm, 4), "", tempRU(design, 4))
		}
	}
	if r.Snow != nil {
		if err := snowExcelAppend(*r.Snow, add, &rowNum, &imageCount, &imageRels, &drawing, imageEntries); err != nil {
			return err
		}
	}
	if r.Ice != nil {
		iceExcelAppend(*r.Ice, add)
	}
	for idx, ice := range r.Ices {
		if idx < len(r.IceTitles) {
			add("Гололёдная нагрузка", r.IceTitles[idx])
		}
		iceExcelAppendIndexed(ice, idx+1, add)
	}
	for i, snow := range r.Snows {
		if i < len(r.SnowTitles) {
			add("Снеговая нагрузка", r.SnowTitles[i])
		}
		if err := snowExcelAppend(snow, add, &rowNum, &imageCount, &imageRels, &drawing, imageEntries); err != nil {
			return err
		}
	}
	if len(regular) > 0 {
		add("Временные нагрузки (СП 20.13330.2016)")
		add("№", "Наименование", "Нормативная (кгс/м²; кгс/м; кгс)", "Нормативная (кПа; кН/м; кН)", "γf", "Расчётная (кгс/м²; кгс/м; кгс)", "Расчётная (кПа; кН/м; кН)", "Основание")
		for i, s := range regular {
			unit := tempNumberText(s.Params["unit"])
			factor := kgfPerKN
			if unit == "кПа" {
				factor = kpaToKgfM2(1)
			}
			if def, ok := s.Params["is_defined"].(bool); ok && !def {
				add(fmt.Sprint(i+1), s.Title, "По заданию", "", tempRU(tempNumber(s, "gamma_f"), 2), "По заданию", "", tempReadableSource(s.Source))
				continue
			}
			n, d := tempNumber(s, "normative_reduced"), tempNumber(s, "design_reduced")
			technical := "кгс"
			if unit == "кПа" {
				technical = "кгс/м²"
			} else if unit == "кН/м" {
				technical = "кгс/м"
			}
			add(fmt.Sprint(i+1), s.Title, tempRU(n*factor, 2)+" "+technical, tempRU(n, 3)+" "+unit, tempRU(tempNumber(s, "gamma_f"), 2), tempRU(d*factor, 2)+" "+technical, tempRU(d, 3)+" "+unit, tempReadableSource(s.Source))
		}
	}
	if len(transport) > 0 {
		add("Транспортные нагрузки (СП 35.13330.2011)")
		add("№", "Транспортная нагрузка", "Общие данные", "Нормативные нагрузки", "γf", "Расчётные нагрузки", "Основание")
		for i, s := range transport {
			d, n, g, c := tempTransportFields(s)
			add(fmt.Sprint(i+1), s.Title, d, n, g, c, tempReadableSource(s.Source))
		}
	}
	if len(transport) > 0 {
		add("")
		add("ПОДРОБНЫЕ КАРТОЧКИ ТРАНСПОРТНЫХ НАГРУЗОК")
		for _, selected := range transport {
			tempExcelTransportDetail(selected, add, &rowNum, &imageCount, &imageRels, &drawing, imageEntries)
		}
	}
	if len(cranes) > 0 {
		add("Мостовые краны")
		add("№", "Кран", "Тип крана", "Колёс на путь", "Колёсные нагрузки, кН (тс)", "Межколёсные расстояния, мм", "Основание")
		for i, s := range cranes {
			add(fmt.Sprint(i+1), s.Title, tempCraneKind(s), fmt.Sprint(len(s.Wheels)), tempForcesBoth(s), tempSteps(s), tempReadableSource(s.Source))
		}
	}
	for _, group := range r.CraneGroups {
		add("МОСТОВЫЕ КРАНЫ", group.Title)
		add("№", "Кран", "Тип", "Колёсные нагрузки, кН (тс)", "Расстояния между колёсами, мм")
		for i, c := range group.Cranes {
			add(fmt.Sprint(i+1), c.Title, tempCraneKind(c), tempForcesBoth(c), tempSteps(c))
		}
		for _, place := range group.Placements {
			add("РАССТАНОВКА КРАНОВ", place.Title, "Пролёты балок, м", place.SpansText)
			add("№", "Опасное положение", "Значение", "Единицы", "Сечение, мм")
			for i, c := range place.Result.Cases {
				add(fmt.Sprint(i+1), c.Name, tempRU(c.Value, 4), c.Units, tempRU(c.SectionMM, 1))
				add("", "Положения колёс от левой опоры, мм", "Нагрузка, кН", "Кран")
				for _, wheel := range c.Points {
					add("", tempRU(wheel.XMM, 1), tempRU(wheel.PKN, 2), wheel.Crane)
				}
				picture, err := craneCasePNG(place.Result.SpansMM, c)
				if err != nil {
					return err
				}
				imageCount++
				id := imageCount
				imageEntries[fmt.Sprintf("xl/media/crane_%d.png", id)] = string(picture)
				fmt.Fprintf(&imageRels, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/crane_%d.png"/>`, id, id)
				start := rowNum + 1
				fmt.Fprintf(&drawing, `<xdr:twoCellAnchor editAs="oneCell"><xdr:from><xdr:col>1</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>%d</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:from><xdr:to><xdr:col>6</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>%d</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:to><xdr:pic><xdr:nvPicPr><xdr:cNvPr id="%d" name="Расстановка кранов"/><xdr:cNvPicPr/></xdr:nvPicPr><xdr:blipFill><a:blip r:embed="rId%d"/><a:stretch><a:fillRect/></a:stretch></xdr:blipFill><xdr:spPr><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></xdr:spPr></xdr:pic><xdr:clientData/></xdr:twoCellAnchor>`, start, start+15, id, id)
				for j := 0; j < 18; j++ {
					add("")
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
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Нагрузки" sheetId="1" r:id="rId1"/></sheets><calcPr fullCalcOnLoad="1"/></workbook>`,
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
	if r.Snow != nil || len(r.Snows) > 0 {
		snowExcelReorderSheets(entries)
	}
	return packageZip(path, entries)
}
