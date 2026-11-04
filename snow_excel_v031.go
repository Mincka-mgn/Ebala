package main

import (
	"fmt"
	_ "image/png"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Independent relationship IDs are essential: rId2 belongs to styles in the
// base workbook and must never be reused for new worksheets.
var snowRelRx = regexp.MustCompile(`Id="rId([0-9]+)"`)
var snowSheetRx = regexp.MustCompile(`sheetId="([0-9]+)"`)

func snowMaxNumeric(matches [][]string) int {
	m := 0
	for _, a := range matches {
		n, _ := strconv.Atoi(a[1])
		if n > m {
			m = n
		}
	}
	return m
}

// A spreadsheet's sheetData rows MUST be in ascending numerical order.
// Excel repairs badly ordered rows by removing records from the worksheet.
var orderedSheetRow = regexp.MustCompile(`<row r="([0-9]+)">.*?</row>`)

func orderExcelSheetRows(body string) (string, error) {
	matches := orderedSheetRow.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return body, nil
	}
	rows := make(map[int]string, len(matches))
	ids := make([]int, 0, len(matches))
	for _, m := range matches {
		id, e := strconv.Atoi(m[1])
		if e != nil || id < 1 {
			return "", fmt.Errorf("неверный номер строки Excel: %s", m[1])
		}
		if _, ok := rows[id]; ok {
			return "", fmt.Errorf("повтор строки Excel: %d", id)
		}
		rows[id] = m[0]
		ids = append(ids, id)
	}
	// No unrecognized markup can silently get discarded during normalization.
	if strings.TrimSpace(orderedSheetRow.ReplaceAllString(body, "")) != "" {
		return "", fmt.Errorf("неожиданный XML между строками листа %s", "Excel")
	}
	sort.Ints(ids)
	var out strings.Builder
	for _, id := range ids {
		out.WriteString(rows[id])
	}
	return out.String(), nil
}
func snowExcelSheet(entries map[string]string, name, body string, images bool) (int, error) {
	var orderingErr error
	body, orderingErr = orderExcelSheetRows(body)
	if orderingErr != nil {
		return 0, fmt.Errorf("лист %s: %w", name, orderingErr)
	}
	// Use a dedicated bordered style without changing the other report worksheets.
	body = strings.ReplaceAll(body, `<c r=`, `<c s="1" r=`)
	body = strings.Replace(body, `<row r="1">`, `<row r="1" ht="38" customHeight="1">`, 1)
	if i := strings.Index(body, `<row r="1" ht="38" customHeight="1">`); i >= 0 {
		end := strings.Index(body[i:], `</row>`)
		if end > 0 {
			head := body[i : i+end]
			body = body[:i] + strings.ReplaceAll(head, `<c s="1" r=`, `<c s="2" r=`) + body[i+end:]
		}
	}

	wb := entries["xl/workbook.xml"]
	rels := entries["xl/_rels/workbook.xml.rels"]
	types := entries["[Content_Types].xml"]
	sheetId := snowMaxNumeric(snowSheetRx.FindAllStringSubmatch(wb, -1)) + 1
	relId := snowMaxNumeric(snowRelRx.FindAllStringSubmatch(rels, -1)) + 1
	part := fmt.Sprintf("xl/worksheets/snow%d.xml", sheetId)
	sheet := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheetViews><sheetView workbookViewId="0" showGridLines="1"/></sheetViews><sheetFormatPr defaultRowHeight="28"/><cols><col min="1" max="1" width="58" customWidth="1"/><col min="2" max="2" width="26" customWidth="1"/><col min="3" max="3" width="28" customWidth="1"/><col min="4" max="4" width="72" customWidth="1"/><col min="5" max="6" width="28" customWidth="1"/><col min="7" max="7" width="65" customWidth="1"/><col min="8" max="8" width="25" customWidth="1"/><col min="9" max="9" width="22" customWidth="1"/><col min="10" max="13" width="30" customWidth="1"/><col min="14" max="14" width="78" customWidth="1"/></cols><sheetData>` + body + `</sheetData>`
	if images {
		sheet += `<drawing r:id="rId1"/>`
	}
	sheet += `</worksheet>`
	entries[part] = sheet
	wb = strings.Replace(wb, `</sheets>`, fmt.Sprintf(`<sheet name="%s" sheetId="%d" r:id="rId%d"/></sheets>`, tempXML(name), sheetId, relId), 1)
	rels = strings.Replace(rels, `</Relationships>`, fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/snow%d.xml"/></Relationships>`, relId, sheetId), 1)
	types = strings.Replace(types, `</Types>`, fmt.Sprintf(`<Override PartName="/%s" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`, part), 1)
	entries["xl/workbook.xml"], entries["xl/_rels/workbook.xml.rels"], entries["[Content_Types].xml"] = wb, rels, types
	return sheetId, nil
}
func snowExcelFigure(entries map[string]string, name string, pic []byte) error {
	sheetId, err := snowExcelSheet(entries, name, tempExcelRow(1, name)+tempExcelRow(2, "Размеры — в метрах, интенсивности нагрузок — в кПа"), true)
	if err != nil {
		return err
	}
	drawingId := 1
	for {
		if _, ok := entries[fmt.Sprintf("xl/drawings/drawing%d.xml", drawingId)]; !ok {
			break
		}
		drawingId++
	}
	drawing := fmt.Sprintf("drawing%d.xml", drawingId)
	media := fmt.Sprintf("snow_image_%d.png", sheetId)
	entries["xl/media/"+media] = string(pic)
	entries[fmt.Sprintf("xl/worksheets/_rels/snow%d.xml.rels", sheetId)] = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/drawing" Target="../drawings/` + drawing + `"/></Relationships>`
	entries["xl/drawings/"+drawing] = `<?xml version="1.0" encoding="UTF-8"?><xdr:wsDr xmlns:xdr="http://schemas.openxmlformats.org/drawingml/2006/spreadsheetDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><xdr:twoCellAnchor editAs="oneCell"><xdr:from><xdr:col>0</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>3</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:from><xdr:to><xdr:col>7</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>35</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:to><xdr:pic><xdr:nvPicPr><xdr:cNvPr id="1" name="Схема снеговой нагрузки"/><xdr:cNvPicPr/></xdr:nvPicPr><xdr:blipFill><a:blip r:embed="rId1"/><a:stretch><a:fillRect/></a:stretch></xdr:blipFill><xdr:spPr><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></xdr:spPr></xdr:pic><xdr:clientData/></xdr:twoCellAnchor></xdr:wsDr>`
	entries["xl/drawings/_rels/"+drawing+".rels"] = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/` + media + `"/></Relationships>`
	typ := entries["[Content_Types].xml"]
	if !strings.Contains(typ, `Default Extension="png"`) {
		typ = strings.Replace(typ, `</Types>`, `<Default Extension="png" ContentType="image/png"/></Types>`, 1)
	}
	typ = strings.Replace(typ, `</Types>`, fmt.Sprintf(`<Override PartName="/xl/drawings/%s" ContentType="application/vnd.openxmlformats-officedocument.drawing+xml"/></Types>`, drawing), 1)
	entries["[Content_Types].xml"] = typ
	return nil
}
func snowExcelYes(v bool) string {
	if v {
		return "Да"
	}
	return "Нет"
}
func snowExcelInputData(s SnowCalc) string {
	var b strings.Builder
	add := func(row int, v ...string) { b.WriteString(tempExcelRow(row, v...)) }
	add(1, "На этом листе приведены исходные данные расчёта. Столбец A — показатель, B — значение, C — единица измерения, D — нормативное основание.")
	add(2, "Тип покрытия", s.ReportName(), "", "Выбрано пользователем")
	add(3, "Вес снега Sg", xnum(s.Ground), "кПа", "Снеговой район; СП 20.13330.2016, таблица 10.1")
	add(4, "γf — надёжность", xnum(s.Gamma), "", "СП 20.13330.2016, пункт 10.12")
	add(5, "cе в расчётном ядре", xnum(s.CE), "", "СП 20.13330.2016, пп. 10.6–10.9; перепроверить по листу коэффициентов")
	add(6, "ct в расчётном ядре", xnum(s.CT), "", "СП 20.13330.2016, пункт 10.10; перепроверить по листу коэффициентов")
	add(7, "Тип местности", s.Terrain, "", "СП 20.13330.2016, пункт 11.1.6")
	if s.JanuarySet {
		add(8, "Средняя температура января", xnum(s.JanuaryTemp), "°C", snowClimateOrigin(s))
	} else {
		add(8, "Средняя температура января", "не задана", "°C", "Ручной ввод либо СП 131.13330.2025")
	}
	if s.WinterWindSet {
		add(9, "Скорость ветра холодного периода", xnum(s.WinterWind), "м/с", snowClimateOrigin(s))
	} else {
		add(9, "Скорость ветра холодного периода", "не задана", "м/с", "Ручной ввод либо СП 131.13330.2025")
	}
	add(10, "H — полная высота до конька", xnum(s.RoofHeight), "м", "Задано пользователем")
	add(11, "L — размер поперёк", xnum(s.Span), "м", "Задано пользователем")
	if s.Shape == "sphere_cone" {
		add(12, "D — диаметр в плане (повтор)", xnum(s.Span), "м", "Для формул характеристического размера")
	} else {
		add(12, "B — размер вдоль", xnum(s.Width), "м", "Задано пользователем")
	}
	add(13, "α — уклон конуса"+func() string {
		if s.Shape != "sphere_cone" {
			return " / кровли"
		}
		return ""
	}(), xnum(s.Angle), "град", "Из геометрии, заданной пользователем")
	if s.Shape == "sphere_cone" {
		add(14, "h конуса = r × tg α₂", "=B16*TAN(RADIANS(B13))", "м", "Из геометрии конической части")
	} else {
		add(14, "h/f — подъём", xnum(s.Rise), "м", "Из геометрии, заданной пользователем")
	}
	if s.Shape == "sphere_cone" {
		add(15, "f — подъём от стенки до конька", xnum(s.H), "м", "Размер на утверждённом рисунке Б.15а")
		add(16, "r — горизонтальный радиус конической части", xnum(s.SkylightWidth), "м", "Размер на утверждённом рисунке Б.15а")
	} else {
		add(15, "h1 — высота перепада/фонаря", xnum(s.H), "м", "Задано пользователем")
		add(16, "a — ширина фонаря", xnum(s.SkylightWidth), "м", "Задано пользователем")
	}
	upper, lower := s.stepRoofs()
	if s.Shape != "sphere_cone" {
		add(17, "n — число пролётов", fmt.Sprint(s.Count), "", "Задано пользователем")
		add(18, "L1 — верхняя кровля", xnum(upper.Span), "м", "Задано пользователем")
		add(19, "L2 — нижняя кровля", xnum(lower.Span), "м", "Задано пользователем")
		add(20, "Z1 — верхняя кромка", xnum(upper.EdgeLevel), "м", "Задано пользователем")
		add(21, "Z2 — нижняя кромка", xnum(lower.EdgeLevel), "м", "Задано пользователем")
	}
	heat := 0
	if s.WarmRoof {
		heat = 1
	}
	add(22, "Неутеплённая, повышенное тепловыделение", snowExcelYes(heat == 1), "", "Признак задан пользователем")
	j := 0
	if s.JanuarySet {
		j = 1
	}
	add(23, "Температура января задана", snowExcelYes(j == 1), "", "Проверка наличия исходных данных")
	w := 0
	if s.WinterWindSet {
		w = 1
	}
	add(24, "Ветер холодного периода задан", snowExcelYes(w == 1), "", "Проверка наличия исходных данных")
	red := 0
	if s.ReducedApplicable() {
		red = 1
	}
	add(25, "Пониженная нормативная допускается", snowExcelYes(red == 1), "", "Пункт 10.11 СП 20.13330.2016")
	able := 0
	if s.Shape != "step" && s.Shape != "wall" && s.Shape != "skylight" && s.Shape != "cone" && s.Shape != "dome" && s.Shape != "sphere_cone" && s.Shape != "shed" && s.Shape != "parapet" {
		able = 1
	}
	add(26, "Тип кровли допускает снос по 10.7", snowExcelYes(able == 1), "", "Условия СП 20.13330.2016")
	if s.RightRoof != nil && s.RightRoof.Shape != "none" {
		r := s.RightRoof
		add(40, "Второй пролёт — тип", snowShapeTitle(r.Shape), "", "Задано пользователем")
		add(41, "Второй пролёт — L2", xnum(r.Span), "м", "Задано пользователем")
		add(42, "Второй пролёт — B2", xnum(r.Width), "м", "Задано пользователем")
		add(43, "Второй пролёт — H2 конька", xnum(r.RoofHeight), "м", "Задано пользователем")
		add(44, "Второй пролёт — α2", xnum(r.resolvedAngle()), "град", "По геометрии второго покрытия")
		add(45, "Второй пролёт — f/h2", xnum(r.Rise), "м", "По геометрии второго покрытия")
		add(46, "Второй пролёт — a2 фонаря", xnum(r.SkylightWidth), "м", "Задано пользователем")
		add(47, "Второй пролёт — hf2 фонаря", xnum(r.LanternHeight), "м", "Задано пользователем")
	}
	small := 0
	if s.Shape == "flat" || s.Shape == "mono" || s.Shape == "gable" || s.Shape == "multi_gable" {
		if s.Angle <= 10 {
			small = 1
		}
	} else if s.Shape == "vault" || s.Shape == "multi_vault" {
		if s.Span > 0 && s.Rise/s.Span <= .05 {
			small = 1
		}
	}
	add(31, "Защищённое от прямого воздействия ветра", snowExcelYes(!s.Exposed), "", "СП 20.13330.2016, пункт 10.6; при защите ce=1")
	add(27, "Форма допускает снижение по 10.7", snowExcelYes(small == 1), "", "Условия СП 20.13330.2016")
	if s.Shape != "sphere_cone" {
		add(28, "α1 — верхняя кровля", xnum(upper.resolvedAngle()), "град", "Из параметров верхнего покрытия")
		add(29, "α2 — нижняя кровля", xnum(lower.resolvedAngle()), "град", "Из параметров нижнего покрытия")
	} else {
		add(32, "Считать строго по формуле зоны 5", snowExcelYes(s.SphereStrictFormula), "", "Да: по α₁; Нет: по α₂")
		add(33, "Шаг угла β", xnum(s.BetaStep), "град", "Задано пользователем")
		add(34, "α₁ у стенки", "=DEGREES(2*ATAN2(B11/2-B16,B15-B16*TAN(RADIANS(B13)))-RADIANS(B13))", "град", "Касательное сопряжение сферической и конической поверхностей")
	}
	add(30, "Примечание", "Все производные коэффициенты вычисляются на расчётных листах Excel.")
	return b.String()
}
func snowExcelCoefData(s SnowCalc) string {
	var b strings.Builder
	add := func(n int, a ...string) { b.WriteString(tempExcelRow(n, a...)) }
	src := "'Исходные данные'!"
	add(1, "На этом листе рассчитаны КОЭФФИЦИЕНТЫ ce, ct, γf и дополнительные коэффициенты. В столбце B — проверочные формулы Excel, в D — значения ядра, в строках 41–42 — расхождения.")
	add(2, "Sg, кПа", "="+src+"B3", "Вес снега на земле, таблица 10.1 СП 20.13330.2016")
	add(3, "γf", "="+src+"B4", "Пункт 10.12 СП 20.13330.2016")
	add(4, "α, град", "="+src+"B13", "Геометрия")
	add(5, "L, м", "="+src+"B11", "Геометрия")
	add(6, "B, м", "="+src+"B12", "Геометрия")
	add(7, "H конька, м", "="+src+"B10", "Геометрия")
	add(8, "T января, °C", "="+src+"B8", "СП 131.13330.2025 / ручной ввод")
	add(9, "V зимы, м/с", "="+src+"B9", "СП 131.13330.2025 / ручной ввод")
	add(10, "Тип местности A", "=IF("+src+"B7=\"А\",1,0)", "1 = А; 0 = В/С")
	add(11, "Условия применения ce", "=IF(AND("+src+"B23=\"Да\",B8<=-5,"+src+"B24=\"Да\",B9>3,B7>0,"+src+"B26=\"Да\","+src+"B27=\"Да\","+src+"B7<>\"С\"),1,0)", "Условия пункта 10.7 СП 20.13330.2016")
	if s.Shape == "multi_gable" || s.Shape == "multi_vault" {
		add(12, "Характерный размер lc, м", "=MIN(100,2*MIN(B5*"+strconv.Itoa(s.Count)+",B6)-MIN(B5*"+strconv.Itoa(s.Count)+",B6)^2/MAX(B5*"+strconv.Itoa(s.Count)+",B6))", "Формула (10.3); учитывается суммарная ширина всех пролётов")
	} else {
		add(12, "Характерный размер lc, м", "=MIN(100,2*MIN(B5,B6)-MIN(B5,B6)^2/MAX(B5,B6))", "Формула (10.3) СП 20.13330.2016")
	}
	// k from Table 11.2 using source data selected by Go for the two bracket points.
	hs := []float64{5, 10, 20, 40, 60, 80, 100, 150, 200, 250, 300}
	ka := []float64{.75, 1, 1.25, 1.5, 1.7, 1.85, 2, 2.25, 2.45, 2.65, 2.75}
	kb := []float64{.5, .65, .85, 1.1, 1.3, 1.45, 1.6, 1.9, 2.1, 2.3, 2.5}
	lo := 0
	hi := 0
	if s.RoofHeight > 5 {
		for i := 1; i < len(hs); i++ {
			if s.RoofHeight <= hs[i] {
				lo = i - 1
				hi = i
				break
			}
		}
		if s.RoofHeight > 300 {
			lo = len(hs) - 1
			hi = lo
		}
	}
	kt := kb
	if s.Terrain == "А" {
		kt = ka
	}
	add(13, "H1 (табл. 11.2)", xnum(hs[lo]), "м, нижняя табличная отметка", xnum(kt[lo]))
	add(14, "H2 (табл. 11.2)", xnum(hs[hi]), "м, верхняя табличная отметка", xnum(kt[hi]))
	add(15, "k1 (табл. 11.2)", "=D13", "Табличное значение k при H1")
	add(16, "k2 (табл. 11.2)", "=D14", "Табличное значение k при H2")
	add(17, "k(H), интерполяция", "=IF(B14=B13,B15,B15+(B7-B13)*(B16-B15)/(B14-B13))", "Таблица 11.2, линейная интерполяция")
	add(18, "kv, коэффициент ветра", "=IF(B10=1,INDEX(B37:D39,IF(B8< -25,3,IF(B8< -15,2,1)),IF(B9>6,3,IF(B9>4,2,1))),INDEX(E37:G39,IF(B8< -25,3,IF(B8< -15,2,1)),IF(B9>6,3,IF(B9>4,2,1))))", "Таблица 10.2 СП 20.13330.2016")
	add(19, "ce, коэффициент сноса", "=B45", "Независимая проверочная формула Excel, сравнение в строке 41", xnum(s.CE))
	grad := snowRoofGradient(s)
	add(20, "i — уклон кровли", xnum(grad), "Тангенс угла ската, либо 2f/L для свода")
	add(21, "ct, термический", "=B46", "Независимая проверочная формула Excel, сравнение в строке 42", xnum(s.CT))
	ceCheck := "=IF(B11=0,1,MAX(0.5,MIN(1,(B18-0.4*SQRT(B17))*(0.8+0.002*B12))))"
	if s.Shape == "cone" || s.Shape == "dome" || s.Shape == "sphere_cone" {
		ceCheck = "=1"
		if (s.Shape == "cone" && s.Angle <= 7 || s.Shape == "dome" && s.Rise/s.Span <= .05) && s.Exposed && s.JanuarySet && s.JanuaryTemp <= -5 {
			ceCheck = "=IF(B5<=60,0.85,IF(B5>=100,1,0.85+0.00375*(B5-60)))"
		}
	}
	add(45, "Проверочное ce по п. 10.7/10.8", ceCheck, "Сравнить с коэффициентом расчётного ядра")
	ctCheck := "=IF(AND(" + src + "B22=\"Да\",B20>0.03),0.8,1)"
	if s.RightRoof != nil {
		ctCheck = "=1"
	}
	add(46, "Проверочное ct по п. 10.10", ctCheck, "Сравнить с коэффициентом расчётного ядра")
	add(22, "μ базовое", "=IF(ABS(B4)<=30,1,IF(ABS(B4)>=60,0,(60-ABS(B4))/30))", "Коэффициент формы для плоских и скатных покрытий")
	add(23, "a, ширина фонаря", "="+src+"B16", "Схема фонаря")
	add(24, "b=(L−a)/2", "=(B5-B23)/2", "Ширина ската до фонаря")
	add(25, "b1=min(b;h1)", "=MIN(B24,"+src+"B15)", "Вспомогательная зона фонаря")
	add(26, "μ1 фонаря", "=0.8", "Формула Б.3")
	add(27, "μ2 фонаря", "=MIN(4,MIN(3*"+src+"B15/B2,1+0.1*B23/B24))", "Б.3, вариант 1")
	add(28, "μ3 фонаря", "=MIN(4,MIN(3*"+src+"B15/B2,1+0.5*B23/B25))", "Б.3, вариант 2")
	// For a roof step, x>=0 refers to the lower roof; the original computed values
	// remain in D cells for cross-checking, while Excel reproduces all intermediate quantities.
	upper, lower := s.stepRoofs()
	h := math.Min(s.H, 8)
	d := lower.Span
	_ = upper
	_ = d
	add(29, "h перепада (ограничение 8 м)", "=MIN(8,"+src+"B15)", "Схема Б.8", xnum(h))
	add(30, "m1 верхней кровли", func() string {
		if s.Shape == "wall" {
			return "=0"
		}
		return "=IF(" + src + "B28>20,0.3,0.4)"
	}(), "По типу и уклону верхнего покрытия")
	k2 := "(1-MIN(35," + src + "B29)/35)"
	k3 := "MAX(0.3,1-" + src + "B29/30)"
	if lower.Shape == "flat" || lower.Shape == "vault" {
		k2 = "1"
		k3 = "1"
	}
	steep := "IF(" + src + "B29>20,0.3,0.4)"
	if lower.Shape == "vault" {
		steep = "IF(OR(" + src + "B29>20," + src + "B14/" + src + "B19>0.125),0.3,0.4)"
	}
	add(31, "m2 нижней кровли", "=IF("+src+"B19<21,MAX(0.1,0.5*SQRT("+src+"B19/21)*"+k2+"*"+k3+"),"+steep+")", "По типу и уклону нижнего покрытия; формулы Б.8")
	add(32, "L нижней кровли", "="+src+"B19", "Размер ниже перепада")
	add(33, "L переноса", "=MIN(B32,3*B6)", "Для нижней кровли без парапета")
	add(34, "μ мешка без ограничения", "=1+(B30*"+src+"B18+B31*B33)/MAX(0.001,B29)", "Формулы Б.5–Б.6")
	add(35, "μ мешка, ограниченный", "=MAX(1,MIN(B34,MIN(2*B29/MAX(0.001,B2*B19*B21),IF(MAX("+src+"B18,"+src+"B19)>48,4+2*MIN(1,MAX(0,(MAX("+src+"B18,"+src+"B19)-48)/24)),4))))", "Б.8: ограничение μmax зависит от размеров покрытия")
	add(36, "b зоны заноса", "=MIN(2*B29,16)", "Длина зоны заноса у перепада, м")
	add(37, "T: −15…−5; V: 3…4", "1.4", "1.3", "1.3", "1.4", "1.4", "1.3")
	add(38, "T: −25…−15; V: 3…4", "1.4", "1.3", "1.2", "1.4", "1.4", "1.3")
	add(39, "T: менее −25; V: 3…4", "1.3", "1.2", "1.2", "1.4", "1.3", "1.2")
	add(40, "μ края нижней зоны", "=MAX(0,1-2*B31)", "Начальное значение снеговой эпюры по Б.8")
	add(41, "РАСХОЖДЕНИЕ ce", "=B19-"+src+"B5", "Разность независимого Excel-расчёта и значения Go")
	add(42, "РАСХОЖДЕНИЕ ct", "=B21-"+src+"B6", "Разность независимого Excel-расчёта и значения Go")
	return b.String()
}
func snowExcelMuFormula(s SnowCalc, c SnowCase, v SnowInterval, row int) (string, string) {
	cx := "'Коэффициенты'!"
	src := "'Исходные данные'!"
	mid := fmt.Sprintf("((A%d+B%d)/2)", row, row)
	// The numerical factor belongs to the particular *alternative* arrangement.
	// The basic μ is always calculated by Excel, rather than copied from Go.
	if lo, hi, ok := snowPartialSectorBounds(c); ok {
		// A diametral section follows the angular bisector of the chosen sector.
		// The first/second half-diameter have opposite azimuths.
		center := (lo + hi) / 2
		angleAtX := fmt.Sprintf("IF(%s<%s$B$11/2,%g,%g)", mid, src, math.Mod(center+180, 360), center)
		gate := fmt.Sprintf("AND(%s>=%g,%s<%g)", angleAtX, lo, angleAtX, hi)
		angle := cx + "$B$4"
		if s.Shape == "dome" {
			f := src + "$B$14"
			d := src + "$B$11"
			radius := fmt.Sprintf("(%s^2/(8*%s)+%s/2)", d, f, f)
			z := fmt.Sprintf("ABS(%s-%s/2)", mid, d)
			angle = fmt.Sprintf("DEGREES(ASIN(MIN(1,%s/%s)))", z, radius)
		} else if s.Shape == "sphere_cone" {
			geom := "'Геометрия Б.15а'!"
			d := geom + "$B$3"
			r := fmt.Sprintf("ABS(%s-%s/2)", mid, d)
			a := fmt.Sprintf("ABS(DEGREES(ASIN(MAX(-1,MIN(1,(%s-%s$B$11)/ABS(%s$B$10))))))", r, geom, geom)
			angle = fmt.Sprintf("IF(%s<=%s$B$5,%s$B$6,%s)", r, geom, geom, a)
			// The roof may have an underground abutment; suppress snow there.
			h := fmt.Sprintf("IF(%s<=%s$B$5,%s$B$2-%s*TAN(RADIANS(%s$B$6)),IF(%s$B$10>=0,%s$B$12+SQRT(MAX(0,%s$B$10^2-(%s-%s$B$11)^2)),%s$B$12-SQRT(MAX(0,%s$B$10^2-(%s-%s$B$11)^2))))", r, geom, geom, r, geom, geom, geom, geom, r, geom, geom, geom, r, geom)
			mu := fmt.Sprintf("IF(%s<0,0,IF(%s<=30,1,IF(%s>=60,0,(60-%s)/30)))", h, angle, angle, angle)
			return fmt.Sprintf("=IF(%s,%s,0)", gate, mu), "Пункт 10.4: половина/четверть круга; μ₁ по местной касательной в расчётной точке"
		}
		mu := fmt.Sprintf("IF(%s<=30,1,IF(%s>=60,0,(60-%s)/30))", angle, angle, angle)
		return fmt.Sprintf("=IF(%s,%s,0)", gate, mu), "Пункт 10.4: половина/четверть круга, частичное загружение по таблице Б.3"
	}
	switch s.Shape {
	case "flat", "mono", "gable":
		bas := snowRoofMu(s.Angle)
		if bas == 0 {
			return "=0", "Уклон ≥60°, снег не удерживается на скате"
		}
		factor := v.Mu / bas
		return fmt.Sprintf("=%s$B$22*%.9g", cx, factor), "Коэффициент формы по α × множитель соответствующего варианта схемы"
	case "shed":
		if strings.Contains(c.Title, "Равномерное") {
			return "=1", "Б.4, рисунок Б.7, вариант 1"
		}
		return fmt.Sprintf("=IF(MOD(%s,%s$B$11)<%s$B$11/2,1.4,0.6)", mid, src, src), "Б.4, рисунок Б.7: μ=1,4 и μ=0,6 на половинах шедового пролёта"
	case "parapet":
		if strings.Contains(c.Title, "Основная") {
			return "=1", "Б.13: равномерный нормативный вариант"
		}
		if strings.Contains(c.Title, "Местная") {
			h := src + "$B$15"
			sg := src + "$B$3"
			mu := fmt.Sprintf("MIN(3,2*%s/%s)", h, sg)
			return fmt.Sprintf("=IF(A%d<2*%s,%s+(1-%s)*A%d/(2*%s),1)", row, h, mu, mu, row, h), "Б.13: верхняя оценка линейной эпюры на каждом расчётном участке; μmax точно у парапета"
		}
		return "=" + xnum(v.Mu), "Б.13(в): дополнительная схема по Б.8; значения проверены в расчётном ядре"
	case "skylight":
		switch {
		case strings.HasPrefix(c.Title, "1A"):
			return "=1", "Б.3, равномерная схема"
		case strings.HasPrefix(c.Title, "1C"):
			return fmt.Sprintf("=IF(%s<%s$B$24,%s$B$27,IF(%s<%s$B$24+%s$B$23,%s$B$26,%s$B$27))", mid, cx, cx, mid, cx, cx, cx, cx), "Б.3: μ1 и μ2 вычислены на листе коэффициентов"
		case strings.HasPrefix(c.Title, "2A"):
			return fmt.Sprintf("=IF(AND(%s>=%s$B$24-%s$B$25,%s<=%s$B$24+%s$B$23+%s$B$25),%s$B$28,1)", mid, cx, cx, mid, cx, cx, cx, cx), "Б.3: зона b1 у фонаря и μ3"
		case strings.HasPrefix(c.Title, "2C"):
			b := cx + "$B$24"
			b1 := cx + "$B$25"
			a := cx + "$B$23"
			mu3 := cx + "$B$28"
			first := fmt.Sprintf("AND(%s>=%s-%s,%s<%s)", mid, b, b1, mid, b)
			midblock := fmt.Sprintf("AND(%s>=%s,%s<%s+%s)", mid, b, mid, b, a)
			last := fmt.Sprintf("AND(%s>=%s+%s,%s<%s+%s+%s)", mid, b, a, mid, b, a, b1)
			return fmt.Sprintf("=IF(%s,%s,IF(%s,0,IF(%s,%s,1)))", first, mu3, midblock, last, mu3), "Б.3: две локальные зоны b1 и μ3"
		}
	case "cone":
		if len(c.Polar) > 0 {
			return fmt.Sprintf("=MAX(0,IF(%s$B$4<=30,2.15*RADIANS(%s$B$4)^(1/3),1.7*30/%s$B$4)*(%s-%s$B$5/2)/(%s$B$5/2))", cx, cx, cx, mid, cx, cx), "Б.12: меридиональный разрез β=90°, полярная таблица ниже"
		}
		return "=" + cx + "$B$22", "Б.12, равномерная схема"
	case "dome":
		// The column x measures the full diameter; the spatial drift is plotted
		// in its beta=90 meridian and the polar grid is calculated separately.
		d := src + "$B$11"
		f := src + "$B$14"
		rad := fmt.Sprintf("(%s^2/(8*%s)+%s/2)", d, f, f)
		z := fmt.Sprintf("ABS(%s-%s/2)", mid, d)
		alpha := fmt.Sprintf("DEGREES(ASIN(MIN(1,%s/%s)))", z, rad)
		if strings.HasPrefix(c.Title, "1.") {
			return fmt.Sprintf("=IF(%s<=30,1,IF(%s>=60,0,(60-%s)/30))", alpha, alpha, alpha), "Б.11, таблица Б.2: μ1 по местному углу сферического покрытия"
		}
		betaSin := "1" // Meridional beta = 90 degrees in the plotted section.
		_ = betaSin
		if strings.HasPrefix(c.Title, "3.") {
			return fmt.Sprintf("=IF(%s<=%s/2,0,MAX(0,3*SQRT(2*%s/%s)*SIN(RADIANS(3*%s))))", mid, d, f, d, alpha), "Б.11, формула (Б.11); меридиональное сечение β=90°"
		}
		cr := fmt.Sprintf("(2.55-EXP(0.8-14*%s/%s))", f, d)
		r1 := fmt.Sprintf("(%s*SIN(RADIANS(30)))", rad)
		mu := fmt.Sprintf("IF(%s<=%s,%s*(%s/%s)^2,IF(%s<=45,%s+(%s-30)/15*(1.5-%s),IF(%s<60,1.5*(60-%s)/15,0)))", z, r1, cr, z, r1, alpha, cr, alpha, cr, alpha, alpha)
		return fmt.Sprintf("=IF(%s<=%s/2,0,%s)", mid, d, mu), "Б.11, формула (Б.10); за α=30° линейная интерполяция до α=45° и 60°; β=90°"
	case "multi_gable":
		if strings.Contains(c.Title, "Частичная нагрузка") {
			if v.Mu == 0 {
				return "=0", "П. 10.4: скат не загружен"
			}
			return "=" + cx + "$B$22", "П. 10.4: коэффициент ската из угла кровли"
		}
		if strings.Contains(c.Title, "Равномерная") {
			return "=1", "Б.5, равномерная схема"
		}
		if strings.Contains(c.Title, "Неравномерная") || strings.Contains(c.Title, "Вариант по") || strings.Contains(c.Title, "Нагрузки в") {
			// These are coefficient multipliers prescribed by the individual variant,
			// not machine-computed load values. The coefficient itself is an Excel formula.
			return "=" + xnum(v.Mu), "Коэффициент μ по распределению (Б.5); при изменении геометрии повторить формирование вариантов"
		}
	case "multi_vault":
		if strings.Contains(c.Title, "Частичная нагрузка") {
			if v.Mu == 0 {
				return "=0", "П. 10.4: незагруженная половина свода"
			}
			return "=1", "П. 10.4: две нагруженные половины соседних сводов"
		}
		if strings.Contains(c.Title, "Равномерная") {
			return "=1", "Б.5, вариант 1"
		}
		return fmt.Sprintf("=2*ABS(2*MOD(%s,%s$B$11)/%s$B$11-1)", mid, src, src), "Б.5: треугольная эпюра в ендовах"
	case "vault":
		if strings.HasPrefix(c.Title, "Частичное загружение") {
			// One half is strictly unloaded. On the loaded half calculate
			// μ1 from circular-arch geometry, exactly as the Go core does.
			left := !strings.Contains(c.Title, "зеркальное")
			if (left && v.X1 >= s.Span/2) || (!left && v.X2 <= s.Span/2) {
				return "=0", "П. 10.4: разгруженная половина кругового свода"
			}
			R := fmt.Sprintf("(%s$B$11^2/(8*%s$B$14)+%s$B$14/2)", src, src, src)
			dx := fmt.Sprintf("ABS(%s-%s$B$11/2)", mid, src)
			mu1 := fmt.Sprintf("MAX(0,COS(1.5*ATAN2(SQRT(MAX(0.000001,%s^2-%s^2)),%s)))", R, dx, dx)
			if left {
				return fmt.Sprintf("=IF(%s<%s$B$11/2,%s,0)", mid, src, mu1), "П. 10.4; Б.2: μ1 только на левой половине"
			}
			return fmt.Sprintf("=IF(%s>%s$B$11/2,%s,0)", mid, src, mu1), "П. 10.4; Б.2: μ1 только на правой половине"
		}
		if strings.Contains(c.Title, "Симметричная") {
			x := mid
			R := fmt.Sprintf("(%s$B$11^2/(8*%s$B$14)+%s$B$14/2)", src, src, src)
			dx := fmt.Sprintf("ABS(%s-%s$B$11/2)", x, src)
			return fmt.Sprintf("=MAX(0,COS(1.5*ATAN2(SQRT(MAX(0.000001,%s^2-%s^2)),%s)))", R, dx, dx), "Б.2: μ1 = cos(1,5α), α из геометрии кругового свода"
		}
		// Include the piecewise linear interpolation as an explicit Excel expression.
		f, L := s.Rise, s.Span
		R := L*L/(8*f) + f/2
		edge := snowVaultAngle(L, f, 0)
		type cp struct{ x, y float64 }
		edgeQ := 2 * math.Sin(3*edge*math.Pi/180)
		if edge >= 60 {
			edgeQ = 0
		}
		ps := []cp{{0, edgeQ}}
		if edge > 60 {
			ps = append(ps, cp{L/2 - R*math.Sin(math.Pi/3), 0})
		}
		if edge > 30 {
			ps = append(ps, cp{L/2 - R*.5, 2})
		}
		ps = append(ps, cp{L / 2, 0})
		x := fmt.Sprintf("MIN(%s,%s$B$11-%s)", mid, src, mid)
		calc := "0"
		for i := len(ps) - 1; i >= 1; i-- {
			p, q := ps[i-1], ps[i]
			expr := fmt.Sprintf("%.12g+(%.12g-%.12g)*(%s-%.12g)/%.12g", p.y, q.y, p.y, x, p.x, q.x-p.x)
			calc = fmt.Sprintf("IF(%s<=%.12g,%s,%s)", x, q.x, expr, calc)
		}
		leftBig := strings.Contains(c.Title, "2. Несимметричная")
		if leftBig {
			return fmt.Sprintf("=(%s)*IF(%s<=%s$B$11/2,1,0.5)", calc, mid, src), "Б.2: линейная интерполяция по контрольным сечениям и 50% на противоположной половине"
		}
		return fmt.Sprintf("=(%s)*IF(%s<=%s$B$11/2,0.5,1)", calc, mid, src), "Б.2: зеркальное распределение по контрольным сечениям"
	case "step", "wall":
		if v.X2 <= 0 {
			return "=" + xnum(v.Mu), "Верхняя кровля; основное загружение по её самостоятельной схеме"
		}
		if strings.Contains(c.Title, "перепада") || strings.Contains(c.Title, "высокой стены") {
			// The user sees both the independent input and exact applied μ in this formula.
			return fmt.Sprintf("=IF(A%d<%s$B$36,%s$B$35+(%s$B$40-%s$B$35)*A%d/%s$B$36,%s$B$40)", row, cx, cx, cx, cx, row, cx, cx), "Б.8: линейная эпюра заноса с ограничением μmax"
		}
		return "=" + xnum(v.Mu), "Б.8: основная альтернативная схема нижней кровли"
	}
	return "=" + xnum(v.Mu), "Нормативное значение μ по выбранной типовой схеме; проверка основания в Word"
}
func snowStepStartMu(s SnowCalc) float64 {
	// Equivalent base coefficient for the wide lower-roof zone. It is explicitly
	// disclosed in the worksheet and never mistaken for a wind/exposure factor.
	for _, c := range s.Cases {
		if strings.Contains(c.Title, "перепада") || strings.Contains(c.Title, "высокой стены") {
			if len(c.Intervals) > 0 {
				return c.Intervals[len(c.Intervals)-1].Mu
			}
		}
	}
	return 1
}
func snowExcelVariantData(s SnowCalc, c SnowCase, index int) string {
	var b strings.Builder
	add := func(n int, v ...string) { b.WriteString(tempExcelRow(n, v...)) }
	add(1, fmt.Sprintf("На этом листе приведён расчёт коэффициентов μ и снеговых нагрузок для варианта %d «%s». Исходные данные и формулы вынесены на соседние листы.", index+1, c.Title))
	add(2, "Нормативное основание", c.Source)
	add(3, "Описание", c.Note)
	add(4, "Начало x, м", "Конец x, м", "μ(x)", "Sн, кПа", "Sр, кПа", "Sпониж, кПа", "Вывод μ / основание")
	for j, v := range c.Intervals {
		row := j + 5
		f, desc := snowExcelMuFormula(s, c, v, row)
		red := "не определяется"
		if s.ReducedApplicable() {
			red = fmt.Sprintf("=C%d*'Коэффициенты'!$B$2*0.5", row)
		}
		add(row, xnum(v.X1), xnum(v.X2), f, fmt.Sprintf("=C%d*'Коэффициенты'!$B$2*'Коэффициенты'!$B$19*'Коэффициенты'!$B$21", row), fmt.Sprintf("=D%d*'Коэффициенты'!$B$3", row), red, desc)
	}
	if len(c.Polar) > 0 {
		k := len(c.Intervals) + 7
		add(k, "Пространственный вариант по β: радиусы, углы и коэффициенты по приложению Б СП 20.13330.2016")
		add(k+1, "r, м", "β, град", "μ(r,β)", "Sн, кПа", "Sр, кПа", "Зона по СП")
		for i, p := range c.Polar {
			row := k + 2 + i
			mu := fmt.Sprintf("=MAX(0,IF('Коэффициенты'!$B$4<=30,2.15*RADIANS('Коэффициенты'!$B$4)^(1/3),1.7*30/'Коэффициенты'!$B$4)*A%d/('Исходные данные'!$B$11/2)*SIN(RADIANS(B%d)))", row, row)
			if lower, upper, partial := snowPartialSectorBounds(c); partial && s.Shape != "sphere_cone" {
				angle := "'Коэффициенты'!$B$4"
				if s.Shape == "dome" {
					f := "'Исходные данные'!$B$14"
					d := "'Исходные данные'!$B$11"
					radius := fmt.Sprintf("(%s^2/(8*%s)+%s/2)", d, f, f)
					angle = fmt.Sprintf("DEGREES(ASIN(MIN(1,A%d/%s)))", row, radius)
				}
				base := fmt.Sprintf("IF(%s<=30,1,IF(%s>=60,0,(60-%s)/30))", angle, angle, angle)
				mu = fmt.Sprintf("=IF(AND(MOD(B%d,360)>=%g,MOD(B%d,360)<%g),%s,0)", row, lower, row, upper, base)
			} else if s.Shape == "sphere_cone" {
				// Независимая формула на специализированной полярной сетке.
				// Мнимые формулы вида =0.73079 недопустимы.
				mu = fmt.Sprintf("='Сетка Б.15а вар.%d'!E%d", index+1, i+4)
			} else if s.Shape == "dome" {
				d := "'Исходные данные'!$B$11"
				f := "'Исходные данные'!$B$14"
				radius := fmt.Sprintf("(%s^2/(8*%s)+%s/2)", d, f, f)
				r1 := fmt.Sprintf("(%s*0.5)", radius)
				alpha := fmt.Sprintf("DEGREES(ASIN(MIN(1,A%d/%s)))", row, radius)
				if strings.HasPrefix(c.Title, "3.") {
					mu = fmt.Sprintf("=MAX(0,3*SQRT(2*%s/%s)*SIN(RADIANS(3*%s))*SIN(RADIANS(B%d)))", f, d, alpha, row)
				} else {
					cr := fmt.Sprintf("(2.55-EXP(0.8-14*%s/%s))", f, d)
					coef := fmt.Sprintf("IF(A%d<=%s,%s*(A%d/%s)^2,IF(%s<=45,%s+(%s-30)/15*(1.5-%s),IF(%s<60,1.5*(60-%s)/15,0)))", row, r1, cr, row, r1, alpha, cr, alpha, cr, alpha, alpha)
					mu = fmt.Sprintf("=MAX(0,%s*SIN(RADIANS(B%d)))", coef, row)
				}
			}
			note := ""
			if s.Shape == "sphere_cone" {
				note = fmt.Sprintf("Зона %d Б.15а", p.Zone)
			}
			add(row, xnum(p.Radius), xnum(p.Beta), mu, fmt.Sprintf("=C%d*'Коэффициенты'!$B$2*'Коэффициенты'!$B$19*'Коэффициенты'!$B$21", row), fmt.Sprintf("=D%d*'Коэффициенты'!$B$3", row), note)
		}
	}
	return b.String()
}
func snowExcelGridStyle(entries map[string]string) {
	st := entries["xl/styles.xml"]
	st = strings.Replace(st, `<fonts count="1">`, `<fonts count="2">`, 1)
	st = strings.Replace(st, `</fonts>`, `<font><b/><sz val="11"/><name val="Arial"/></font></fonts>`, 1)
	st = strings.Replace(st, `<borders count="1"><border/></borders>`, `<borders count="2"><border/><border><left style="thin"><color indexed="64"/></left><right style="thin"><color indexed="64"/></right><top style="thin"><color indexed="64"/></top><bottom style="thin"><color indexed="64"/></bottom></border></borders>`, 1)
	st = strings.Replace(st, `<cellXfs count="1">`, `<cellXfs count="3">`, 1)
	st = strings.Replace(st, `</cellXfs>`, `<xf numFmtId="0" fontId="0" fillId="0" borderId="1" xfId="0" applyBorder="1"/><xf numFmtId="0" fontId="1" fillId="0" borderId="1" xfId="0" applyBorder="1" applyFont="1"/></cellXfs>`, 1)
	entries["xl/styles.xml"] = st
}

// Full polar record for both B.15a alternatives, without the abbreviated
// graphical sampling. Numerical snow pressures are Excel formula cells.
func snowExcelPolarData(s SnowCalc, c SnowCase, idx int) string {
	var b strings.Builder
	b.WriteString(tempExcelRow(1, fmt.Sprintf("На этом листе представлена полярная сетка Б.15а для варианта %d с формулами нагрузок, номерами зон и контрольными значениями μ.", idx+1)))
	b.WriteString(tempExcelRow(2, "Sg, кПа", xnum(s.Ground), "ce", xnum(s.CE), "ct", xnum(s.CT), "γf", xnum(s.Gamma)))
	b.WriteString(tempExcelRow(3, "β, град", "От края, м", "Радиус, м", "Зона", "μ по СП", "Sн=μ·Sg·ce·ct", "Sр=γf·Sн", "Основание μ"))
	for i, p := range c.Polar {
		row := i + 4
		zone := "—"
		if p.Zone > 0 {
			zone = fmt.Sprint(p.Zone)
		}
		ref := "Таблица Б.3, местный уклон"
		if idx > 0 {
			ref = fmt.Sprintf("Б.12(б), зона %d", p.Zone)
		}
		b.WriteString(tempExcelRow(row, xnum(p.Beta), xnum(s.Span/2-p.Radius), xnum(p.Radius), zone, xnum(p.Mu), fmt.Sprintf("=E%d*$B$2*$D$2*$F$2", row), fmt.Sprintf("=F%d*$H$2", row), ref))
	}
	return b.String()
}

func snowExcelV031(entries map[string]string, s SnowCalc, blockIndex ...int) error {
	suffix := ""
	if len(blockIndex) > 0 && blockIndex[0] > 0 {
		suffix = fmt.Sprintf(" %d", blockIndex[0])
	}
	rename := func(body string) string {
		if suffix == "" {
			return body
		}
		body = strings.ReplaceAll(body, "'Исходные данные'!", "'Исходные данные"+suffix+"'!")
		body = strings.ReplaceAll(body, "'Коэффициенты'!", "'Коэффициенты"+suffix+"'!")
		body = strings.ReplaceAll(body, "'Геометрия Б.15а'!", "'Геометрия Б.15а"+suffix+"'!")
		for i := range s.Cases {
			old := fmt.Sprintf("'Сетка Б.15а вар.%d'!", i+1)
			newName := fmt.Sprintf("'Сетка Б.15а вар.%d%s'!", i+1, suffix)
			body = strings.ReplaceAll(body, old, newName)
		}
		return body
	}

	snowExcelGridStyle(entries)
	if _, e := snowExcelSheet(entries, "Исходные данные"+suffix, rename(snowExcelInputData(s)), false); e != nil {
		return e
	}
	if _, e := snowExcelSheet(entries, "Коэффициенты"+suffix, rename(snowExcelCoefData(s)), false); e != nil {
		return e
	}
	if _, e := snowExcelSheet(entries, "Формулы снега"+suffix, snowExcelFormulaGuide(s), false); e != nil {
		return e
	}
	if s.Shape == "sphere_cone" {
		if _, e := snowExcelSheet(entries, "Геометрия Б.15а"+suffix, snowExcelSphereGeometry(s, "Исходные данные"+suffix), false); e != nil {
			return e
		}
	}
	for i, c := range s.Cases {
		if _, e := snowExcelSheet(entries, fmt.Sprintf("Вариант %d%s", i+1, suffix), rename(snowExcelVariantData(s, c, i)), false); e != nil {
			return e
		}
		if s.Shape == "sphere_cone" && len(c.Polar) > 0 {
			if _, e := snowExcelSheet(entries, fmt.Sprintf("Сетка Б.15а вар.%d%s", i+1, suffix), snowExcelSpherePolar(s, c, i, "Геометрия Б.15а"+suffix, "Исходные данные"+suffix, "Коэффициенты"+suffix), false); e != nil {
				return e
			}
		}
	}
	roof, e := snowRoofImage(s)
	if e != nil {
		return e
	}
	if e = snowExcelFigure(entries, "Схема покрытия"+suffix, roof); e != nil {
		return e
	}
	for i, c := range s.Cases {
		im, e := snowImage(s, c)
		if e != nil {
			return e
		}
		if e = snowExcelFigure(entries, fmt.Sprintf("Эпюра %d%s", i+1, suffix), im); e != nil {
			return e
		}
		if len(c.Cross) > 0 {
			cross := c
			cross.Intervals = c.Cross
			cross.Cross = nil
			cross.Title = "Сечение по зонам 2–4–4–2"
			im2, err := snowImage(s, cross)
			if err != nil {
				return err
			}
			if err = snowExcelFigure(entries, fmt.Sprintf("Сечение 2-4-4-2 %d%s", i+1, suffix), im2); err != nil {
				return err
			}
		}
	}
	return nil
}
