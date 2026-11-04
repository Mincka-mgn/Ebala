package main

import (
	"fmt"
	"strings"
)

func tempTableStart(cols []int) string {
	var b strings.Builder
	b.WriteString(`<w:tbl><w:tblPr><w:tblBorders>`)
	for _, e := range []string{"top", "left", "bottom", "right", "insideH", "insideV"} {
		fmt.Fprintf(&b, `<w:%s w:val="single" w:sz="6" w:color="000000"/>`, e)
	}
	b.WriteString(`</w:tblBorders></w:tblPr><w:tblGrid>`)
	for _, n := range cols {
		fmt.Fprintf(&b, `<w:gridCol w:w="%d"/>`, n)
	}
	b.WriteString(`</w:tblGrid>`)
	return b.String()
}
func tempTableRow(values []string, widths []int) string {
	var b strings.Builder
	b.WriteString(`<w:tr><w:trPr><w:cantSplit/></w:trPr>`)
	for i, v := range values {
		b.WriteString(tableCell(v, widths[i]))
	}
	b.WriteString(`</w:tr>`)
	return b.String()
}
func tempTableHeaderRow(values []string, widths []int) string {
	row := tempTableRow(values, widths)
	return strings.Replace(row, `<w:trPr><w:cantSplit/></w:trPr>`, `<w:trPr><w:cantSplit/><w:tblHeader w:val="true"/></w:trPr>`, 1)
}

func exportTempWord(r TempReport, path string) error {
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
	var b strings.Builder
	imageCount := 0
	var rel strings.Builder
	entries := map[string]string{}
	b.WriteString(pBold(func() string {
		if len(r.SiteLines)+len(r.WeightRows) > 0 {
			return "ОБЩАЯ ВЕДОМОСТЬ НАГРУЗОК"
		}
		if r.Snow != nil && r.Ice == nil && len(regular)+len(transport)+len(cranes) == 0 {
			return "РАСЧЁТ СНЕГОВЫХ НАГРУЗОК"
		}
		if r.Ice != nil && r.Snow == nil && len(regular)+len(transport)+len(cranes) == 0 {
			return "РАСЧЁТ ГОЛОЛЁДНЫХ НАГРУЗОК"
		}
		return "ВРЕМЕННЫЕ И ТРАНСПОРТНЫЕ НАГРУЗКИ"
	}()))
	if r.Project != "" {
		b.WriteString(pText("Объект: " + r.Project))
	}
	if len(r.SiteLines) > 0 {
		b.WriteString(pBold("Характеристики местности"))
		for _, line := range r.SiteLines {
			b.WriteString(pText(line))
		}
	}
	if len(r.WeightRows) > 0 || len(r.WeightBlocks) > 0 {
		b.WriteString(pBold("Постоянные нагрузки. Собственный вес"))
		widths := []int{430, 4400, 2200, 1000, 2200}
		for _, group := range weightReportBlocks(r) {
			b.WriteString(pBold(group.Title))
			b.WriteString(tempTableStart(widths))
			b.WriteString(tempTableHeaderRow([]string{"№", "Наименование", "Нормативная", "γf", "Расчётная"}, widths))
			norm, design := 0.0, 0.0
			for i, x := range group.Rows {
				norm += x.Norm
				design += x.Design
				b.WriteString(tempTableRow([]string{fmt.Sprint(i + 1), x.Name, tempUnitPair(x.Norm, "кПа"), tempRU(x.Gamma, 3), tempUnitPair(x.Design, "кПа")}, widths))
			}
			b.WriteString(tempTableRow([]string{"", "ИТОГО", tempUnitPair(norm, "кПа"), "", tempUnitPair(design, "кПа")}, widths))
			b.WriteString(`</w:tbl>`)
		}
	}
	if r.Snow != nil {
		section, err := snowWordAppend(*r.Snow, &imageCount, &rel, entries)
		if err != nil {
			return err
		}
		b.WriteString(section)
	}
	if r.Ice != nil {
		section, err := iceWordAppend(*r.Ice, &imageCount, &rel, entries)
		if err != nil {
			return err
		}
		b.WriteString(section)
	}
	if len(r.Snows) > 0 {
		for i, snow := range r.Snows {
			if i < len(r.SnowTitles) {
				b.WriteString(pBold("Расчёт: " + r.SnowTitles[i]))
			}
			section, err := snowWordAppend(snow, &imageCount, &rel, entries)
			if err != nil {
				return err
			}
			b.WriteString(section)
		}
	}
	if len(r.Ices) > 0 {
		for i, ice := range r.Ices {
			if i < len(r.IceTitles) {
				b.WriteString(pBold("Расчёт: " + r.IceTitles[i]))
			}
			section, err := iceWordAppend(ice, &imageCount, &rel, entries)
			if err != nil {
				return err
			}
			b.WriteString(section)
		}
	}
	if len(regular) > 0 {
		b.WriteString(pBold("Временные нагрузки (СП 20.13330.2016)"))
		width := []int{380, 2900, 2030, 900, 2030, 1600}
		b.WriteString(tempTableStart(width))
		b.WriteString(tempTableHeaderRow([]string{"№", "Наименование", "Нормативная", "γf", "Расчётная", "Основание"}, width))
		for i, s := range regular {
			unit := tempNumberText(s.Params["unit"])
			norm, calc := "По заданию", "По заданию"
			if def, ok := s.Params["is_defined"].(bool); !ok || def {
				norm = tempUnitPair(tempNumber(s, "normative_reduced"), unit)
				calc = tempUnitPair(tempNumber(s, "design_reduced"), unit)
			}
			b.WriteString(tempTableRow([]string{fmt.Sprint(i + 1), s.Title, norm, tempRU(tempNumber(s, "gamma_f"), 2), calc, tempReadableSource(s.Source)}, width))
		}
		b.WriteString(`</w:tbl>`)
	}
	if len(transport) > 0 {
		b.WriteString(pBold("Транспортные нагрузки (СП 35.13330.2011)"))
		width := []int{320, 1830, 1850, 2450, 1080, 2450}
		b.WriteString(tempTableStart(width))
		b.WriteString(tempTableHeaderRow([]string{"№", "Наименование", "Общие данные", "Нормативные", "γf", "Расчётные"}, width))
		for i, s := range transport {
			detail, n, g, c := tempTransportFields(s)
			b.WriteString(tempTableRow([]string{fmt.Sprint(i + 1), s.Title, detail, n, g, c}, width))
		}
		b.WriteString(`</w:tbl>`)
		for _, s := range transport {
			b.WriteString(tempWordTransportAppend(s, &imageCount, &rel, entries))
		}
	}
	if len(cranes) > 0 {
		b.WriteString(pBold("Мостовые краны. Колёсная схема"))
		hasDistances := false
		for _, s := range cranes {
			if tempSteps(s) != "" {
				hasDistances = true
			}
		}
		widths := []int{360, 1900, 1750, 800, 3400, 2240}
		names := []string{"№", "Кран", "Тип крана", "Колёс на путь", "Нагрузки колёс, кН (тс)", "Основание"}
		if hasDistances {
			widths = []int{360, 1650, 1500, 650, 2650, 1750, 1840}
			names = []string{"№", "Кран", "Тип крана", "Колёс", "Нагрузки, кН (тс)", "Межколёсные расстояния, мм", "Основание"}
		}
		b.WriteString(tempTableStart(widths))
		b.WriteString(tempTableRow(names, widths))
		for i, s := range cranes {
			row := []string{fmt.Sprint(i + 1), s.Title, tempCraneKind(s), fmt.Sprint(len(s.Wheels)), tempForcesBoth(s), tempReadableSource(s.Source)}
			if hasDistances {
				row = []string{fmt.Sprint(i + 1), s.Title, tempCraneKind(s), fmt.Sprint(len(s.Wheels)), tempForcesBoth(s), tempSteps(s), tempReadableSource(s.Source)}
			}
			b.WriteString(tempTableRow(row, widths))
		}
		b.WriteString(`</w:tbl>`)
	}
	// Parent crane set, followed immediately by its own placement schemes.
	for _, group := range r.CraneGroups {
		b.WriteString(pBold("Мостовые краны. " + group.Title))
		widths := []int{400, 2400, 2300, 1950, 3200}
		b.WriteString(tempTableStart(widths))
		b.WriteString(tempTableHeaderRow([]string{"№", "Кран", "Тип", "Нагрузки на колёса", "Расстояния, мм"}, widths))
		for i, c := range group.Cranes {
			b.WriteString(tempTableRow([]string{fmt.Sprint(i + 1), c.Title, tempCraneKind(c), tempForcesBoth(c), tempSteps(c)}, widths))
		}
		b.WriteString(`</w:tbl>`)
		for _, place := range group.Placements {
			b.WriteString(pBold("Расстановка: " + place.Title))
			b.WriteString(pText("Пролёты балок: " + place.SpansText + " м"))
			widths2 := []int{540, 5100, 2100, 1800}
			b.WriteString(tempTableStart(widths2))
			b.WriteString(tempTableHeaderRow([]string{"№", "Опасное положение", "Значение", "Сечение, мм"}, widths2))
			for i, c := range place.Result.Cases {
				b.WriteString(tempTableRow([]string{fmt.Sprint(i + 1), c.Name, tempRU(c.Value, 3) + " " + c.Units, tempRU(c.SectionMM, 1)}, widths2))
			}
			b.WriteString(`</w:tbl>`)
			for _, c := range place.Result.Cases {
				positions := []string{}
				for _, wheel := range c.Points {
					positions = append(positions, fmt.Sprintf("%s: x=%s мм, P=%s кН", wheel.Crane, tempRU(wheel.XMM, 1), tempRU(wheel.PKN, 2)))
				}
				b.WriteString(pText(c.Name + ": " + strings.Join(positions, "; ")))
				picture, err := craneCasePNG(place.Result.SpansMM, c)
				if err != nil {
					return err
				}
				imageCount++
				id := imageCount
				path := fmt.Sprintf("word/media/crane_%d.png", id)
				entries[path] = string(picture)
				fmt.Fprintf(&rel, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/crane_%d.png"/>`, id, id)
				b.WriteString(tempWordImageInline(id, 1040, 300))
			}
		}
	}
	doc := `<?xml version="1.0" encoding="UTF-8"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` + b.String() + `<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="720" w:right="720" w:bottom="720" w:left="720"/></w:sectPr></w:body></w:document>`
	entries["[Content_Types].xml"] = `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Default Extension="png" ContentType="image/png"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`
	entries["_rels/.rels"] = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`
	entries["word/document.xml"] = doc
	entries["word/_rels/document.xml.rels"] = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + rel.String() + `</Relationships>`
	return packageZip(path, entries)
}
