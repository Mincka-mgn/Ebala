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
	if strings.HasPrefix(name, "Секторная сетка") {
		// The six sectors occupy 22 columns. The default 14-column layout
		// truncates the normative/design load headings and first-row guide.
		start := strings.Index(sheet, "<cols>")
		end := strings.Index(sheet, "</cols>")
		if start >= 0 && end > start {
			sheet = sheet[:start] + `<cols><col min="1" max="1" width="16" customWidth="1"/><col min="2" max="2" width="21" customWidth="1"/><col min="3" max="4" width="24" customWidth="1"/><col min="5" max="22" width="23" customWidth="1"/></cols>` + sheet[end+len("</cols>"):]
		}
		sheet = strings.Replace(sheet, `<row r="1">`, `<row r="1" ht="52" customHeight="1">`, 1)
		sheet = strings.Replace(sheet, `</sheetData>`, `</sheetData><mergeCells count="1"><mergeCell ref="A1:V1"/></mergeCells>`, 1)
	}
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
	sheetId, err := snowExcelSheet(entries, name, tempExcelRow(1, "На этом листе показана иллюстративная геометрическая схема покрытия; численные нагрузки и эпюры рассчитываются на других листах.")+tempExcelRow(2, "Размеры — в метрах, интенсивности нагрузок — в кПа"), true)
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
	add := func(row int, v ...string) { b.WriteString(tempExcelInputRow(row, v...)) }
	add(1, "На этом листе приведены исходные данные расчёта. Столбец A — показатель, B — значение, C — единица измерения, D — нормативное основание.")
	add(2, "Тип покрытия", s.ReportName(), "", "Выбрано пользователем")
	add(3, "Вес снега Sg", xnum(s.Ground), "кПа", "Снеговой район; СП 20.13330.2016, таблица 10.1")
	add(4, "γf — надёжность", xnum(s.Gamma), "", "СП 20.13330.2016, пункт 10.12")
	add(5, "cе — вычисляется на листе коэффициентов", "", "", "СП 20.13330.2016, пп. 10.6–10.9")
	add(6, "ct — вычисляется на листе коэффициентов", "", "", "СП 20.13330.2016, пункт 10.10")
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
		add(12, "Поперечный габарит B", "", "м", "Для круглого покрытия не задаётся отдельно; D берётся из B11")
	} else {
		add(12, "B — размер вдоль", xnum(s.Width), "м", "Задано пользователем")
	}
	angleInput := xnum(s.Angle)
	if s.SlopeMode == "height" {
		angleInput = "" // Angle is derived from user-entered rise; it is not an input.
	}
	add(13, "α — уклон конуса"+func() string {
		if s.Shape != "sphere_cone" {
			return " / кровли"
		}
		return ""
	}(), angleInput, "град", "Исходный угол при способе задания «Угол»")
	if s.Shape == "sphere_cone" {
		// Derived cone height is deliberately NOT placed among user inputs.
		// It belongs to the independent geometry worksheet (B21).
		add(14, "h конуса = r × tg α₂", "", "м", "Вычисляется на листе «Геометрия Б.15а»")
	} else {
		riseInput := xnum(s.Rise)
		if s.SlopeMode != "height" && (s.Shape == "mono" || s.Shape == "gable" || s.Shape == "multi_gable" || s.Shape == "skylight" || s.Shape == "cone" || s.Shape == "step" || s.Shape == "wall") {
			riseInput = "" // In angle mode, rise is calculated, not accepted.
		}
		add(14, "h/f — подъём", riseInput, "м", "Исходный подъём при способе задания «Подъём»; для сводов и куполов геометрический размер")
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
	add(25, "Пониженная нормативная", "", "", "Определяется по температуре января на листе коэффициентов")
	add(26, "Применимость сноса снега", "", "", "Вычисляется по условиям п. 10.7 на листе коэффициентов")
	if s.RightRoof != nil && s.RightRoof.Shape != "none" {
		r := s.RightRoof
		add(40, "Второй пролёт — тип", snowShapeTitle(r.Shape), "", "Задано пользователем")
		add(41, "Второй пролёт — L2", xnum(r.Span), "м", "Задано пользователем")
		add(42, "Второй пролёт — B2", xnum(r.Width), "м", "Задано пользователем")
		add(43, "Второй пролёт — H2 конька", xnum(r.RoofHeight), "м", "Задано пользователем")
		rightAngle := xnum(r.Angle)
		if r.SlopeMode == "height" {
			rightAngle = ""
		}
		add(44, "Второй пролёт — α2 (ввод углом)", rightAngle, "град", "Исходный угол, если выбран ввод углом")
		add(45, "Второй пролёт — f/h2 (ввод подъёмом)", xnum(r.Rise), "м", "Исходная высота/подъём второго покрытия")
		add(46, "Второй пролёт — a2 фонаря", xnum(r.SkylightWidth), "м", "Задано пользователем")
		add(47, "Второй пролёт — hf2 фонаря", xnum(r.LanternHeight), "м", "Задано пользователем")
		add(48, "Второй пролёт — неутеплённое покрытие с тепловыделением", snowExcelYes(r.WarmRoof), "", "Принятый пользователем признак для ct по пункту 10.10")
		rightMode := "Угол"
		if r.SlopeMode == "height" {
			rightMode = "Подъём"
		}
		add(49, "Второй пролёт — способ задания уклона", rightMode, "", "Исходный способ задания уклона, выбираемый пользователем")
		add(51, "Второй пролёт — Z кромки (при H2=0)", xnum(r.EdgeLevel), "м", "Принятая кромка второго покрытия, если отдельно задана")
	}
	add(35, "Оборудование у конька (Б.1, вариант 3)", snowExcelYes(s.RidgeEquipment), "", "Принятый пользователем признак")
	add(31, "Защищённое от прямого воздействия ветра", snowExcelYes(!s.Exposed), "", "СП 20.13330.2016, пункт 10.6; при защите ce=1")
	add(27, "Форма кровли", "", "", "Уклон и применение п. 10.7 определяются на листе коэффициентов")
	if s.Shape != "sphere_cone" {
		add(28, "α1 — верхняя кровля", xnum(upper.resolvedAngle()), "град", "Из параметров верхнего покрытия")
		add(29, "α2 — нижняя кровля", xnum(lower.resolvedAngle()), "град", "Из параметров нижнего покрытия")
	} else {
		add(32, "Считать строго по формуле зоны 5", snowExcelYes(s.SphereStrictFormula), "", "Да: по α₁; Нет: по α₂")
		add(33, "Шаг угла β", xnum(s.BetaStep), "град", "Задано пользователем")
		add(34, "α₁ у стенки", "", "град", "Вычисляется на листе «Геометрия Б.15а», не передаётся из ядра")
	}
	if s.Shape == "cone" || s.Shape == "dome" {
		add(33, "Шаг угла β", xnum(s.BetaStep), "град", "Задано пользователем")
	}
	mode := "Угол"
	if s.SlopeMode == "height" {
		mode = "Подъём"
	}
	add(37, "Способ задания уклона", mode, "", "Принято пользователем в программе")
	if s.Shape == "sphere_cone" && s.SlopeMode == "height" {
		add(38, "h конуса — исходная высота", xnum(s.Rise), "м", "Исходный параметр конуса при задании уклона подъёмом")
	}
	add(30, "Примечание", "Все производные коэффициенты вычисляются на расчётных листах Excel.")
	return b.String()
}
func snowExcelCoefData(s SnowCalc) string {
	var b strings.Builder
	add := func(n int, a ...string) { b.WriteString(tempExcelRow(n, a...)) }
	src := "'Исходные данные'!"
	add(1, "На этом листе вычисляются коэффициенты ce, ct, γf, μ и геометрические параметры по формулам Excel. Производные результаты расчётного ядра сюда не передаются.")
	add(2, "Sg, кПа", "="+src+"B3", "Вес снега на земле, таблица 10.1 СП 20.13330.2016")
	add(3, "γf", "="+src+"B4", "Пункт 10.12 СП 20.13330.2016")
	slopeRun := src + "$B$11/2"
	switch s.Shape {
	case "mono":
		slopeRun = src + "$B$11"
	case "skylight":
		slopeRun = "(" + src + "$B$11-" + src + "$B$16)/2"
	case "sphere_cone":
		slopeRun = src + "$B$16"
	case "step", "wall":
		slopeRun = src + "$B$19"
		if s.LowerShape == "gable" {
			slopeRun += "/2"
		}
	}
	riseCell := src + "$B$14"
	if s.Shape == "sphere_cone" {
		riseCell = src + "$B$38"
	}
	add(4, "α, град", "=IF("+src+"$B$37=\"Подъём\",IFERROR(DEGREES(ATAN("+riseCell+"/("+slopeRun+"))),\"\"),"+src+"$B$13)", "Геометрия: угол вычисляется по введённому подъёму либо берётся из исходного угла")
	add(5, "L, м", "="+src+"B11", "Геометрия")
	add(6, "B, м", "="+src+"B12", "Геометрия")
	add(7, "H конька, м", "="+src+"B10", "Геометрия")
	add(8, "T января, °C", "="+src+"B8", "СП 131.13330.2025 / ручной ввод")
	add(9, "V зимы, м/с", "="+src+"B9", "СП 131.13330.2025 / ручной ввод")
	add(10, "Тип местности A", "=IF("+src+"B7=\"А\",1,0)", "1 = А; 0 = В/С")
	// Признаки формы и применимости рассчитываются из исходных геометрических
	// параметров. Выбранная схема покрытия фиксирует структуру расчёта, но не
	// результаты интерполяции и не значения коэффициентов.
	shapeEligible := "0"
	angleEligible := "0"
	switch s.Shape {
	case "flat":
		shapeEligible, angleEligible = "1", "1"
	case "mono", "gable", "multi_gable":
		shapeEligible, angleEligible = "1", "ABS(B4)<=10"
	case "vault", "multi_vault":
		shapeEligible, angleEligible = "1", "IFERROR(2*"+src+"B14/"+src+"B11<=0.1,FALSE)"
	}
	add(11, "Условия применения ce", "=IF(AND("+src+"B31=\"Нет\","+src+"B23=\"Да\",B8<=-5,"+src+"B24=\"Да\",B9>3,B7>0,OR("+src+"B7=\"А\","+src+"B7=\"В\"),"+shapeEligible+","+angleEligible+"),1,0)", "Пункт 10.7 СП 20.13330.2016")
	length := "B5"
	if s.Shape == "multi_gable" || s.Shape == "multi_vault" {
		length = "(B5*" + src + "B17)"
	}
	add(12, "Характерный размер lc, м", "=IFERROR(MIN(100,2*MIN("+length+",B6)-MIN("+length+",B6)^2/MAX("+length+",B6)),0)", "Формула (10.3), суммарная ширина многопролётного покрытия")
	add(13, "H1 по таблице 11.2", "='Таблица 11.2'!D18", "м, нижняя высотная отметка")
	add(14, "H2 по таблице 11.2", "='Таблица 11.2'!E18", "м, верхняя высотная отметка")
	add(15, "k1 по таблице 11.2", "='Таблица 11.2'!F18", "Нижний табличный коэффициент")
	add(16, "k2 по таблице 11.2", "='Таблица 11.2'!G18", "Верхний табличный коэффициент")
	add(17, "k(H), интерполяция", "='Таблица 11.2'!H18", "Линейная интерполяция на листе таблицы 11.2")
	add(18, "kv, коэффициент ветра", "='Таблица 10.2'!B15", "Выбор по температуре, скорости ветра и местности из таблицы 10.2")
	add(19, "ce, коэффициент сноса", "=B45", "Результат формулы Excel")
	gradFormula := "=ABS(TAN(RADIANS(B4)))"
	if s.Shape == "flat" || s.Shape == "wall" {
		gradFormula = "=0"
	} else if s.Shape == "vault" || s.Shape == "multi_vault" || s.Shape == "dome" {
		gradFormula = "=2*" + src + "B14/" + src + "B11"
	}
	add(20, "i — уклон кровли", gradFormula, "По вводимому уклону/подъёму, а не из результата Go")
	add(21, "ct, термический", "=B46", "Результат формулы Excel")
	generalCe := "IF(B11=0,1,MAX(0.5,MIN(1,(B18-0.4*SQRT(B17))*(0.8+0.002*B12))))"
	ceCheck := "=IF(OR(" + src + "B31=\"Да\"," + src + "B23<>\"Да\",B8> -5),1," + generalCe + ")"
	if s.Shape == "cone" || s.Shape == "dome" || s.Shape == "sphere_cone" {
		roundAllowed := "FALSE"
		if s.Shape == "cone" {
			roundAllowed = "ABS(B4)<=7"
		} else if s.Shape == "dome" {
			roundAllowed = "IFERROR(" + src + "B14/" + src + "B11<=0.05,FALSE)"
		}
		ceCheck = "=IF(AND(" + src + "B31=\"Нет\"," + src + "B23=\"Да\",B8<=-5," + roundAllowed + "),IF(B5<=60,0.85,IF(B5>=100,1,0.85+0.00375*(B5-60))),1)"
	}
	add(45, "ce по п. 10.7/10.8", ceCheck, "Формула и все пороговые условия находятся в ячейке B45")
	ctCheck := "=IF(AND(" + src + "B22=\"Да\",B20>0.03),0.8,1)"
	if s.RightRoof != nil {
		ctCheck = "=IF(AND(" + src + "B22=\"Да\"," + src + "B48=\"Да\",B20>0.03,B56/" + src + "B41>0.03),0.8,1)"
	}
	add(46, "ct по п. 10.10", ctCheck, "Термический коэффициент по пользовательским исходным данным")
	if s.Shape == "skylight" {
		// Wide-slope optional B.8 wall drift, recomputed from the editable B.3 inputs.
		add(48, "h для местного заноса, м", "=MIN(8,'Исходные данные'!B15)", "Б.8: эффективная высота")
		add(49, "m2 для плоского ската", "=IF(B24<21,MAX(0.1,0.5*SQRT(B24/21)),0.4)", "Б.8: коэффициент нижней кровли")
		add(50, "Максимум без ограничения", "=1+B49*MIN(B24,3*B6)/B48", "Б.8: m1=0, m2=m(B)")
		add(51, "Верхний предел μ", "=MIN(2*B48/(B2*B21),IF(B24>48,4+2*MIN(1,(B24-48)/24),4))", "Б.8: ограничение μmax, ce локального заноса=1")
		add(52, "μ снегового мешка", "=MAX(1,MIN(B50,B51))", "Б.8: максимальная ордината")
		add(53, "Ширина зоны заноса, м", "=MIN(IF(B52>2*B48/(B2*B21),IF(2*B48/(B2*B21)-1+2*B49>0,2*B48*(B52-1+2*B49)/(2*B48/(B2*B21)-1+2*B49),2*B48),2*B48),MIN(5*B48,16))", "Б.8: ширина со всеми ограничениями")
		add(54, "μ вне мешка", "=MAX(0,IF(AND(B53<B24,B52>2*B48/(B2*B21)),MAX(0.2,(B24-0.5*B52*B53)/(B24-0.5*B53)),1-2*B49))", "Б.8: ордината вне заноса")
	}
	// Deep spherical caps overhang the support ring; the real span is 2R.
	if s.Shape == "dome" {
		add(47, "Фактическая горизонтальная проекция, м", "=IF('Исходные данные'!B14>'Исходные данные'!B11/2,('Исходные данные'!B11^2/(4*'Исходные данные'!B14)+'Исходные данные'!B14),'Исходные данные'!B11)", "Купол с f>D/2: проекция 2R")
	} else {
		add(47, "Фактический размер по X, м", "="+src+"B11", "Ширина покрытия")
	}

	add(22, "μ базовое", "=IF(ABS(B4)<=30,1,IF(ABS(B4)>=60,0,(60-ABS(B4))/30))", "Коэффициент формы для плоских и скатных покрытий")
	add(23, "a, ширина фонаря", "="+src+"B16", "Схема фонаря")
	add(24, "b=(L−a)/2", "=(B5-B23)/2", "Ширина ската до фонаря")
	add(25, "b1=min(b;h1)", "=MIN(B24,"+src+"B15)", "Вспомогательная зона фонаря")
	add(26, "μ1 фонаря", "0.8", "Фиксированный нормативный коэффициент Б.3 СП 20.13330.2016, а не рассчитанное число ядра")
	add(27, "μ2 фонаря", "=MIN(4,MIN(3*"+src+"B15/B2,1+0.1*B23/B24))", "Б.3, вариант 1")
	add(28, "μ3 фонаря", "=MIN(4,MIN(3*"+src+"B15/B2,1+0.5*B23/B25))", "Б.3, вариант 2")
	// For a roof step, x>=0 refers to the lower roof; the original computed values
	// remain in D cells for cross-checking, while Excel reproduces all intermediate quantities.
	upper, lower := s.stepRoofs()
	d := lower.Span
	_ = upper
	_ = d
	add(29, "h перепада (ограничение 8 м)", "=MIN(8,"+src+"B15)", "Схема Б.8")
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
		steep = "IF(OR(" + src + "B29>20,B43/" + src + "B19>0.125),0.3,0.4)"
	}
	add(31, "m2 нижней кровли", "=IF("+src+"B19<21,MAX(0.1,0.5*SQRT("+src+"B19/21)*"+k2+"*"+k3+"),"+steep+")", "По типу и уклону нижнего покрытия; формулы Б.8")
	add(32, "L нижней кровли", "="+src+"B19", "Размер ниже перепада")
	add(33, "L переноса", "=MIN(B32,3*B6)", "Для нижней кровли без парапета")
	add(34, "μ мешка без ограничения", "=1+(B30*"+src+"B18+B31*B33)/B29", "Формулы Б.5–Б.6")
	add(35, "μ мешка, ограниченный", "=MAX(1,MIN(B34,MIN(2*B29/(B2*B19*B21),IF(MAX("+src+"B18,"+src+"B19)>48,4+2*MIN(1,MAX(0,(MAX("+src+"B18,"+src+"B19)-48)/24)),4))))", "Б.8: ограничение μmax зависит от размеров покрытия")
	add(36, "b зоны заноса", "=MIN(2*B29,16)", "Длина зоны заноса у перепада, м")
	add(40, "μ края нижней зоны", "=MAX(0,1-2*B31)", "Начальное значение снеговой эпюры по Б.8")
	add(37, "Таблица 10.2", "=B18", "Значение kv берётся из отдельной нормативной таблицы через ячейку B18")
	add(41, "Формула коэффициента сноса", "ce = (kv − 0,4√k)·(0,8+0,002lc)", "Пункт 10.7; функция с ограничением 0,5…1,0")
	add(42, "Формула термического коэффициента", "ct = 0,8 при условиях пункта 10.10; иначе 1", "По исходным данным")
	add(43, "h — расчётный подъём, м", "=IF("+src+"$B$37=\"Подъём\","+riseCell+",("+slopeRun+")*TAN(RADIANS(B4)))", "При вводе угла подъём вычисляется Excel; при вводе подъёма берётся исходное значение")
	if s.RightRoof != nil && s.RightRoof.Shape != "none" {
		run := src + "$B$41"
		if s.RightRoof.Shape == "gable" || s.RightRoof.Shape == "skylight" {
			run = "(" + src + "$B$41" + func() string {
				if s.RightRoof.Shape == "skylight" {
					return "-" + src + "$B$46"
				}
				return ""
			}() + ")/2"
		}
		add(55, "α₂ второго покрытия, град", "=IF("+src+"$B$49=\"Подъём\",DEGREES(ATAN("+src+"$B$45/("+run+"))),"+src+"$B$44)", "Независимый угол правой крыши; вычисляется по введённой высоте либо углу")
		rightRiseFormula := "=IF(" + src + "$B$49=\"Подъём\"," + src + "$B$45,(" + run + ")*TAN(RADIANS(B55)))"
		if s.RightRoof.Shape == "vault" {
			rightRiseFormula = "=" + src + "$B$45"
		}
		add(56, "f₂ — подъём второй крыши", rightRiseFormula, "Геометрия правой крыши; для кругового свода заданный подъём сохраняется")
	}
	return b.String()
}
func snowExcelMuFormula(s SnowCalc, c SnowCase, v SnowInterval, row int) (string, string) {
	cx := "'Коэффициенты'!"
	src := "'Исходные данные'!"
	mid := fmt.Sprintf("((A%d+B%d)/2)", row, row)
	if s.RightRoof != nil && s.RightRoof.Shape != "none" {
		return snowExcelPairedMu(s, c, row)
	}
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
			z := fmt.Sprintf("ABS(%s-IF(%s>%s/2,%s,%s/2))", mid, f, d, radius, d)
			angle = fmt.Sprintf("DEGREES(ASIN(MIN(1,%s/%s)))", z, radius)
		} else if s.Shape == "sphere_cone" {
			geom := "'Геометрия Б.15а'!"
			d := geom + "$B$3"
			r := fmt.Sprintf("ABS(%s-%s/2)", mid, d)
			a := fmt.Sprintf("ABS(DEGREES(ASIN(MAX(-1,MIN(1,(%s-%s$B$11)/ABS(%s$B$10))))))", r, geom, geom)
			angle = fmt.Sprintf("IF(%s<=%s$B$5,%s$B$6,%s)", r, geom, geom, a)
			// The roof may have an underground abutment; suppress snow there.
			h := fmt.Sprintf("IF(%s<=%s$B$5,%s$B$2-%s*TAN(RADIANS(%s$B$6)),IF(%s$B$10>=0,%s$B$12+SQRT(%s$B$10^2-(%s-%s$B$11)^2),%s$B$12-SQRT(%s$B$10^2-(%s-%s$B$11)^2)))", r, geom, geom, r, geom, geom, geom, geom, r, geom, geom, geom, r, geom)
			mu := fmt.Sprintf("IF(%s<0,0,IF(%s<=30,1,IF(%s>=60,0,(60-%s)/30)))", h, angle, angle, angle)
			return fmt.Sprintf("=IF(%s,%s,0)", gate, mu), "Пункт 10.4: половина/четверть круга; μ₁ по местной касательной в расчётной точке"
		}
		mu := fmt.Sprintf("IF(%s<=30,1,IF(%s>=60,0,(60-%s)/30))", angle, angle, angle)
		return fmt.Sprintf("=IF(%s,%s,0)", gate, mu), "Пункт 10.4: половина/четверть круга, частичное загружение по таблице Б.3"
	}
	switch s.Shape {
	case "flat", "mono", "gable":
		factor := "1"
		if strings.Contains(c.Title, "0,9/1,1") {
			factor = fmt.Sprintf("IF(%s<%s$B$11/2,0.9,1.1)", mid, src)
		} else if strings.Contains(c.Title, "Неравномерная") && s.Shape == "gable" {
			if strings.Contains(c.Title, "зеркальная") {
				factor = fmt.Sprintf("IF(%s<%s$B$11/2,1.25,0.75)", mid, src)
			} else {
				factor = fmt.Sprintf("IF(%s<%s$B$11/2,0.75,1.25)", mid, src)
			}
		} else if strings.Contains(c.Title, "левого ската") {
			factor = fmt.Sprintf("IF(%s<%s$B$11/2,1,0)", mid, src)
		} else if strings.Contains(c.Title, "правого ската") {
			factor = fmt.Sprintf("IF(%s>=%s$B$11/2,1,0)", mid, src)
		} else if strings.Contains(c.Title, "Накопление у конька") {
			return fmt.Sprintf("=IF(%s<%s$B$11/2,0.6,IF(%s<3*%s$B$11/4,1.4,0.6))", mid, src, mid, src), "Б.1 вариант 3: коэффициент по текущему участку"
		}
		return "=" + cx + "$B$22*(" + factor + ")", "Б.1: μ(α) с динамическим выбором ветвей и коэффициента варианта"
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
		return "=IF(A" + strconv.Itoa(row) + "<2*" + src + "$B$15,MIN(3,2*" + src + "$B$15/" + src + "$B$3),1)", "Б.13(в): коэффициент по высоте парапета"
	case "skylight":
		// B.3: partial left/right fields remain editable after export.
		b := cx + "$B$24"
		a := cx + "$B$23"
		if strings.Contains(c.Title, "Частичное загружение у левого ската") {
			return fmt.Sprintf("=IF(%s<%s,%s$B$27,0)", mid, b, cx), "Пункт 10.4: левый скат у фонаря, μ₂ по Б.3"
		}
		if strings.Contains(c.Title, "Частичное загружение у правого ската") {
			return fmt.Sprintf("=IF(%s>=%s+%s,%s$B$27,0)", mid, b, a, cx), "Пункт 10.4: правый скат у фонаря, μ₂ по Б.3"
		}
		// The optional wide-slope drift is a B.8 distribution at the lantern wall;
		// it depends on editable skylight height and slope width, not exported Go μ.
		if strings.Contains(c.Title, "Дополнительное местное загружение у") {
			x := fmt.Sprintf("IF(%s<%s,%s-%s,%s-(%s+%s))", mid, b, b, mid, mid, b, a)
			width := b
			peak := cx + "$B$52"
			baseline := cx + "$B$54"
			drift := fmt.Sprintf("IF(%s<%s$B$53,%s+(%s-%s)*%s/%s$B$53,%s)", x, cx, peak, baseline, peak, x, cx, baseline)
			if strings.Contains(c.Title, "левого") {
				return fmt.Sprintf("=IF(%s<%s,%s,1)", mid, width, drift), "Б.3 примечание 3, Б.8: местный занос у левого фонаря"
			}
			return fmt.Sprintf("=IF(%s>=%s+%s,%s,1)", mid, width, a, drift), "Б.3 примечание 3, Б.8: местный занос у правого фонаря"
		}
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
			return fmt.Sprintf("=IF(OR(%s$B$4<=7,%s$B$4>=60),0,MAX(0,IF(%s$B$4<=30,2.15*RADIANS(%s$B$4)^(1/3),1.7*30/%s$B$4)*(%s-%s$B$5/2)/(%s$B$5/2)))", cx, cx, cx, cx, cx, mid, cx, cx), "Б.12: меридиональный разрез β=90°; ветви 7°, 30°, 60° вычисляет Excel"
		}
		return "=" + cx + "$B$22", "Б.12, равномерная схема"
	case "dome":
		// The column x measures the full diameter; the spatial drift is plotted
		// in its beta=90 meridian and the polar grid is calculated separately.
		d := src + "$B$11"
		f := src + "$B$14"
		rad := fmt.Sprintf("(%s^2/(8*%s)+%s/2)", d, f, f)
		z := fmt.Sprintf("ABS(%s-IF(%s>%s/2,%s,%s/2))", mid, f, d, rad, d)
		alpha := fmt.Sprintf("DEGREES(ASIN(MIN(1,%s/%s)))", z, rad)
		if strings.HasPrefix(c.Title, "1.") {
			return fmt.Sprintf("=IF(%s<=30,1,IF(%s>=60,0,(60-%s)/30))", alpha, alpha, alpha), "Б.11, таблица Б.2: μ1 по местному углу сферического покрытия"
		}
		betaSin := "1" // Meridional beta = 90 degrees in the plotted section.
		_ = betaSin
		if strings.HasPrefix(c.Title, "3.") {
			return fmt.Sprintf("=IF(%s<=%s$B$47/2,0,MAX(0,3*SQRT(2*%s/%s)*SIN(RADIANS(3*%s))))", mid, cx, f, d, alpha), "Б.11, формула (Б.11); меридиональное сечение β=90°"
		}
		cr := fmt.Sprintf("(2.55-EXP(0.8-14*%s/%s))", f, d)
		r1 := fmt.Sprintf("(%s*SIN(RADIANS(30)))", rad)
		mu := fmt.Sprintf("IF(%s<=%s,%s*(%s/%s)^2,IF(%s<=45,%s+(%s-30)/15*(1.5-%s),IF(%s<60,1.5*(60-%s)/15,0)))", z, r1, cr, z, r1, alpha, cr, alpha, cr, alpha, alpha)
		return fmt.Sprintf("=IF(%s<=%s$B$47/2,0,%s)", mid, cx, mu), "Б.11, формула (Б.10); за α=30° линейная интерполяция до α=45° и 60°; β=90°"
	case "sphere_cone":
		g := "'Геометрия Б.15а'!"
		radius := fmt.Sprintf("ABS(%s-%s$B$3/2)", mid, g)
		angle := fmt.Sprintf("IF(%s<=%s$B$5,%s$B$6,ABS(DEGREES(ASIN(MAX(-1,MIN(1,(%s-%s$B$11)/ABS(%s$B$10)))))))", radius, g, g, radius, g, g)
		height := fmt.Sprintf("IF(%s<=%s$B$5,%s$B$2-%s*TAN(RADIANS(%s$B$6)),IF(%s$B$10>=0,%s$B$12+SQRT(%s$B$10^2-(%s-%s$B$11)^2),%s$B$12-SQRT(%s$B$10^2-(%s-%s$B$11)^2)))", radius, g, g, radius, g, g, g, g, radius, g, g, g, radius, g)
		if strings.HasPrefix(c.Title, "1.") {
			return fmt.Sprintf("=IF(%s<0,0,IF(%s<=30,1,IF(%s>=60,0,(60-%s)/30)))", height, angle, angle, angle), "Б.15а: базовая схема, μ₁ по местной касательной, высота проверяется по геометрии"
		}
		return fmt.Sprintf("=IF(AND(%s<%s$B$3/2,%s<%s$B$5,%s>=0),%s$B$18,0)", mid, g, radius, g, height, g), "Б.15а: сечение по зонам 6–1; у левого полудиаметра зона 5, остальные нулевые по sinβ"
	case "multi_gable":
		// Only a variant identity and fixed angular sampling are chosen by Go.
		// Every variable coefficient is recomputed from the source Excel cells.
		u := fmt.Sprintf("MOD(%s,%s$B$11)/%s$B$11", mid, src, src)
		bay := fmt.Sprintf("INT(%s/%s$B$11)", mid, src)
		side := fmt.Sprintf("IF(%s<0.5,0,1)", u)
		baseMu := cx + "$B$22"
		if strings.Contains(c.Title, "Равномерная") {
			return "=1", "Б.5: равномерная по пролётам, μ=1"
		}
		if strings.HasPrefix(c.Title, "2. Неравномерная") {
			return fmt.Sprintf("=IF(%s$B$17=2,IF(AND(%s>%s$B$11/2,%s<3*%s$B$11/2),1.4,0.6),IF(OR(%s<0.25,%s>0.75),1.4,0.6))", src, mid, src, mid, src, u, u), "Б.5: μ=1,4/0,6; проверяются положение внутри пролёта и их количество"
		}
		if strings.Contains(c.Title, "Вариант по рисунку") {
			return fmt.Sprintf("=2*ABS(2*%s-1)", u), "Б.5: линейная треугольная эпюра по каждому пролёту"
		}
		if strings.Contains(c.Title, "Проверочная неравномерная") {
			return fmt.Sprintf("=IF(MOD(%s,2)=0,0.9,1.1)", bay), "Б.5: чередование нормативных множителей 0,9/1,1"
		}
		if strings.Contains(c.Title, "один скат в каждом пролёте") {
			var scheme int
			fmt.Sscanf(c.Title, "Частичная нагрузка: один скат в каждом пролёте, схема %d", &scheme)
			loaded := "0"
			switch scheme {
			case 2:
				loaded = "1"
			case 3:
				loaded = "MOD(" + bay + ",2)"
			case 4:
				loaded = "1-MOD(" + bay + ",2)"
			}
			return fmt.Sprintf("=IF(%s=%s,%s,0)", side, loaded, baseMu), "П. 10.4: один нагруженный скат каждого пролёта"
		}
		if strings.Contains(c.Title, "два смежных ската через пролёт") {
			var scheme int
			fmt.Sscanf(c.Title, "Частичная нагрузка: два смежных ската через пролёт, схема %d", &scheme)
			offset := scheme - 1
			// The right half of one bay and left half of its neighbour form a valley pair.
			return fmt.Sprintf("=IF(OR(AND(%s=1,%s<%s$B$17-1,MOD(%s-%d,2)=0),AND(%s=0,%s>0,MOD(%s-%d-1,2)=0)),%s,0)", side, bay, src, bay, offset, side, bay, bay, offset, baseMu), "П. 10.4: два смежных ската у ендов с чередованием"
		}
		if strings.Contains(c.Title, "Частичная нагрузка: пролёт") {
			var bnum, snum int
			fmt.Sscanf(c.Title, "Частичная нагрузка: пролёт %d, скат %d", &bnum, &snum)
			return fmt.Sprintf("=IF(AND(%s=%d,%s=%d),%s,0)", bay, bnum-1, side, snum-1, baseMu), "П. 10.4: выбранный пользовательским вариантом скат, μ из уклона"
		}
		return "=1", "Б.5: базовый коэффициент для неописанного варианта"
	case "multi_vault":
		if strings.Contains(c.Title, "Равномерная") {
			return "=1", "Б.5: μ=1"
		}
		if strings.Contains(c.Title, "Частичная нагрузка") {
			var first, second int
			fmt.Sscanf(c.Title, "Частичная нагрузка: смежные своды %d и %d", &first, &second)
			bay := fmt.Sprintf("INT(%s/%s$B$11)", mid, src)
			side := fmt.Sprintf("IF(MOD(%s,%s$B$11)<%s$B$11/2,0,1)", mid, src, src)
			return fmt.Sprintf("=IF(OR(AND(%s=%d,%s=1),AND(%s=%d,%s=0)),1,0)", bay, first-1, side, bay, second-1, side), "П. 10.4: две половины смежных сводов"
		}
		return fmt.Sprintf("=2*ABS(2*MOD(%s,%s$B$11)/%s$B$11-1)", mid, src, src), "Б.5: треугольная эпюра в ендовах"
	case "vault":
		if strings.HasPrefix(c.Title, "Частичное загружение") {
			// One half is strictly unloaded. On the loaded half calculate
			// μ1 from circular-arch geometry, exactly as the Go core does.
			left := !strings.Contains(c.Title, "зеркальное")
			// The half-cover selection is evaluated by Excel using the interval midpoint.
			R := fmt.Sprintf("(%s$B$11^2/(8*%s$B$14)+%s$B$14/2)", src, src, src)
			dx := fmt.Sprintf("ABS(%s-%s$B$11/2)", mid, src)
			mu1 := fmt.Sprintf("MAX(0,COS(1.5*ATAN2(SQRT(%s^2-%s^2),%s)))", R, dx, dx)
			if left {
				return fmt.Sprintf("=IF(%s<%s$B$11/2,%s,0)", mid, src, mu1), "П. 10.4; Б.2: μ1 только на левой половине"
			}
			return fmt.Sprintf("=IF(%s>%s$B$11/2,%s,0)", mid, src, mu1), "П. 10.4; Б.2: μ1 только на правой половине"
		}
		if strings.Contains(c.Title, "Симметричная") {
			x := mid
			R := fmt.Sprintf("(%s$B$11^2/(8*%s$B$14)+%s$B$14/2)", src, src, src)
			dx := fmt.Sprintf("ABS(%s-%s$B$11/2)", x, src)
			return fmt.Sprintf("=MAX(0,COS(1.5*ATAN2(SQRT(%s^2-%s^2),%s)))", R, dx, dx), "Б.2: μ1 = cos(1,5α), α из геометрии кругового свода"
		}
		// B.2: control sections are calculated from the live radius and rise,
		// rather than captured from Go at export time.
		L := src + "$B$11"
		f := src + "$B$14"
		R := fmt.Sprintf("(%s^2/(8*%s)+%s/2)", L, f, f)
		x := fmt.Sprintf("MIN(%s,%s-%s)", mid, L, mid)
		edge := fmt.Sprintf("DEGREES(ATAN2(%s-%s,%s/2))", R, f, L)
		qEdge := fmt.Sprintf("IF(%s>=60,0,2*SIN(RADIANS(3*%s)))", edge, edge)
		x60 := fmt.Sprintf("(%s/2-%s*SIN(RADIANS(60)))", L, R)
		x30 := fmt.Sprintf("(%s/2-%s*SIN(RADIANS(30)))", L, R)
		// For edge>60: 0 -> x60 (0), x60 -> x30 (2), x30 -> L/2 (0).
		// For 30<edge<=60: 0 -> x30 (2), x30 -> L/2 (0).
		// Otherwise: 0 -> L/2 (0).
		lo := fmt.Sprintf("(%s+(0-%s)*%s/(%s))", qEdge, qEdge, x, x60)
		middle := fmt.Sprintf("(2*(%s-%s)/(%s-%s))", x, x60, x30, x60)
		leftMain := fmt.Sprintf("(2*(%s/2-%s)/(%s/2-%s))", L, x, L, x30)
		fromEdgeTo30 := fmt.Sprintf("(%s+(2-%s)*%s/%s)", qEdge, qEdge, x, x30)
		base := fmt.Sprintf("IF(%s>60,IF(%s<=%s,%s,IF(%s<=%s,%s,%s)),IF(%s>30,IF(%s<=%s,%s,%s),%s*(1-2*%s/%s)))", edge, x, x60, lo, x, x30, middle, leftMain, edge, x, x30, fromEdgeTo30, leftMain, qEdge, x, L)
		leftBig := strings.Contains(c.Title, "2. Несимметричная")
		if leftBig {
			return fmt.Sprintf("=(%s)*IF(%s<=%s/2,1,0.5)", base, mid, L), "Б.2: динамические координаты контрольных сечений 30°/60°, линейная интерполяция"
		}
		return fmt.Sprintf("=(%s)*IF(%s<=%s/2,0.5,1)", base, mid, L), "Б.2: динамическая линейная интерполяция, зеркальный вариант"
	case "step", "wall":

		if strings.Contains(c.Title, "перепада") || strings.Contains(c.Title, "высокой стены") {
			// The user sees both the independent input and exact applied μ in this formula.
			return fmt.Sprintf("=IF(%s<0,1,IF(A%d<%s$B$36,%s$B$35+(%s$B$40-%s$B$35)*A%d/%s$B$36,%s$B$40))", mid, row, cx, cx, cx, cx, row, cx, cx), "Б.8: линейная эпюра заноса с ограничением μmax"
		}
		return "=1", "Б.8: базовый вариант нижней кровли"
	}
	return "=NA()", "Не реализована независимая формула этой схемы: требуется дополнение генератора, не подставлять μ из ядра"
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

// snowExcelIntervalX recalculates important segment boundaries from editable
// geometry instead of freezing the Go-computed segment edges.
func snowExcelIntervalX(s SnowCalc, c SnowCase, x float64) string {
	src := "'Исходные данные'!"
	approx := func(a, b float64) bool { return math.Abs(a-b) < 1e-6*math.Max(1, math.Abs(b)) }
	if s.Shape == "skylight" {
		b := (s.Span - s.SkylightWidth) / 2
		a := s.SkylightWidth
		b1 := math.Min(b, s.H)
		switch {
		case approx(x, 0):
			return "=0"
		case approx(x, s.Span):
			return "=" + src + "$B$11"
		case approx(x, b):
			return "='Коэффициенты'!$B$24"
		case approx(x, b+a):
			return "='Коэффициенты'!$B$24+'Коэффициенты'!$B$23"
		case approx(x, b-b1):
			return "='Коэффициенты'!$B$24-'Коэффициенты'!$B$25"
		case approx(x, b+a+b1):
			return "='Коэффициенты'!$B$24+'Коэффициенты'!$B$23+'Коэффициенты'!$B$25"
		}
	}
	if s.Shape == "step" || s.Shape == "wall" {
		up, down := s.stepRoofs()
		if x < 0 && up.Span > 0 {
			return fmt.Sprintf("=%s$B$18*%s", src, xnum(x/up.Span))
		}
		if x >= 0 && down.Span > 0 {
			return fmt.Sprintf("=%s$B$19*%s", src, xnum(x/down.Span))
		}
	}
	span := s.Span
	if s.Shape == "dome" && s.Rise > s.Span/2 {
		span = s.Span*s.Span/(4*s.Rise) + s.Rise
	}
	return fmt.Sprintf("='Коэффициенты'!$B$47*%s", xnum(x/span))
}

func snowExcelVariantData(s SnowCalc, c SnowCase, index int) string {
	return snowExcelVariantDataWithStatus(s, c, index, `="Да"`)
}

func snowExcelVariantDataWithStatus(s SnowCalc, c SnowCase, index int, applicability string) string {
	var b strings.Builder
	add := func(n int, v ...string) { b.WriteString(tempExcelRow(n, v...)) }
	add(1, fmt.Sprintf("На этом листе рассчитаны коэффициенты μ и снеговые нагрузки варианта %d «%s». Все координаты строятся по редактируемой геометрии; отдельные нормативные формулы приведены в столбце G.", index+1, c.Title))
	add(2, "Нормативное основание", c.Source)
	// c.Note sometimes embeds Go-computed geometry and μ in its text. Reusing
	// that annotation would make the worksheet stale after an Excel edit.
	add(3, "Применяется к текущим исходным данным", applicability, "Да — рассчитывается; Нет — нагрузка отсутствует. После изменения исходных данных состояние пересчитывается автоматически.")
	add(4, "Начало x, м", "Конец x, м", "μ(x)", "Sн, кПа", "Sр, кПа", "Sпониж, кПа", "Формула и нормативное основание", "Точка x, м")
	samples := snowExcelSampleCount(s)
	for j := 0; j < samples; j++ {
		row := j + 5
		x1, x2 := snowExcelSampleInterval(s, j, samples)
		f, desc := snowExcelMuFormula(s, c, SnowInterval{}, row)
		// An inactive variant has BLANK mu and loads, never a fabricated zero.
		f = fmt.Sprintf(`=IF(AND($B$3="Да",ISNUMBER(A%d),ISNUMBER(B%d)),%s,"")`, row, row, strings.TrimPrefix(f, "="))
		// A missing January temperature must remain missing rather than
		// getting frozen to the original Go-side eligibility decision.
		reduced := fmt.Sprintf(`=IF(AND($B$3="Да",'Исходные данные'!$B$23="Да",'Исходные данные'!$B$8<=-5),C%d*'Коэффициенты'!$B$2*0.5,"")`, row)
		add(row, x1, x2, f,
			fmt.Sprintf(`=IF(C%d="","",C%d*'Коэффициенты'!$B$2*'Коэффициенты'!$B$19*'Коэффициенты'!$B$21)`, row, row),
			fmt.Sprintf(`=IF(D%d="","",D%d*'Коэффициенты'!$B$3)`, row, row),
			reduced, desc, fmt.Sprintf("=(A%d+B%d)/2", row, row))
	}
	tail := samples + 7
	if _, _, sector := snowPartialSectorBounds(c); sector && (s.Shape == "cone" || s.Shape == "dome" || s.Shape == "sphere_cone") {
		add(tail, "Пространственная сетка", "Секторная сетка", "Шесть отдельных секторных загружений рассчитываются в общей полярной таблице; каждый сектор является самостоятельным вариантом")
	}
	if len(c.Polar) > 0 && s.Shape == "sphere_cone" && !snowIsPartialSector(c) {
		add(tail, "Пространственная сетка", fmt.Sprintf("Сетка Б.15а вар.%d", index+1), "Полный расчёт в полярных координатах вынесен на отдельный формульный лист")
	}
	if len(c.Polar) > 0 && (s.Shape == "cone" || s.Shape == "dome") && !snowIsPartialSector(c) {
		add(tail, "Пространственная сетка", fmt.Sprintf("Сетка круглой кровли %d", index+1), "Полярная сетка пересчитывается по исходным диаметру, подъёму и шагу β")
	}
	return b.String()
}
func snowExcelGridStyle(entries map[string]string) {
	st := entries["xl/styles.xml"]
	// Multiple calculation modules may share one workbook. Styling is a
	// workbook operation, not a per-module operation: repeated additions
	// formerly left fonts and xf counts inconsistent with their children.
	if strings.Contains(st, `<borders count="2"><border/><border><left style="thin"`) {
		return
	}
	st = strings.Replace(st, `<fonts count="1">`, `<fonts count="2">`, 1)
	st = strings.Replace(st, `</fonts>`, `<font><b/><sz val="11"/><name val="Arial"/></font></fonts>`, 1)
	st = strings.Replace(st, `<borders count="1"><border/></borders>`, `<borders count="2"><border/><border><left style="thin"><color indexed="64"/></left><right style="thin"><color indexed="64"/></right><top style="thin"><color indexed="64"/></top><bottom style="thin"><color indexed="64"/></bottom></border></borders>`, 1)
	st = strings.Replace(st, `<cellXfs count="1">`, `<cellXfs count="3">`, 1)
	st = strings.Replace(st, `</cellXfs>`, `<xf numFmtId="0" fontId="0" fillId="0" borderId="1" xfId="0" applyBorder="1"/><xf numFmtId="0" fontId="1" fillId="0" borderId="1" xfId="0" applyBorder="1" applyFont="1"/></cellXfs>`, 1)
	entries["xl/styles.xml"] = st
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
		body = strings.ReplaceAll(body, "'Таблица 10.2'!", "'Таблица 10.2"+suffix+"'!")
		body = strings.ReplaceAll(body, "'Таблица 11.2'!", "'Таблица 11.2"+suffix+"'!")
		body = strings.ReplaceAll(body, "'Геометрия Б.15а'!", "'Геометрия Б.15а"+suffix+"'!")
		body = strings.ReplaceAll(body, "'Границы зон'!", "'Границы зон"+suffix+"'!")
		body = strings.ReplaceAll(body, "'Перепад Б.8'!", "'Перепад Б.8"+suffix+"'!")
		body = strings.ReplaceAll(body, "'Секторная сетка'!", "'Секторная сетка"+suffix+"'!")
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
	if e := snowExcelNormTables(entries, "Исходные данные"+suffix, "Коэффициенты"+suffix, suffix); e != nil {
		return e
	}
	if _, e := snowExcelSheet(entries, "Коэффициенты"+suffix, rename(snowExcelCoefData(s)), false); e != nil {
		return e
	}
	if _, e := snowExcelSheet(entries, "Границы зон"+suffix, rename(snowExcelZoneBoundaryGuide(s)), false); e != nil {
		return e
	}
	if s.RightRoof != nil && s.RightRoof.Shape != "none" {
		if _, e := snowExcelSheet(entries, "Перепад Б.8"+suffix, rename(snowExcelPairedStepData(s)), false); e != nil {
			return e
		}
	}
	if _, e := snowExcelSheet(entries, "Формулы снега"+suffix, snowExcelFormulaGuide(s), false); e != nil {
		return e
	}
	if s.Shape == "sphere_cone" {
		if _, e := snowExcelSheet(entries, "Геометрия Б.15а"+suffix, snowExcelSphereGeometry(s, "Исходные данные"+suffix), false); e != nil {
			return e
		}
	}
	plan := snowExcelVariants(s)
	if s.Shape == "cone" || s.Shape == "dome" || s.Shape == "sphere_cone" {
		if _, e := snowExcelSheet(entries, "Секторная сетка"+suffix, rename(snowExcelSectorMatrix(s)), false); e != nil {
			return e
		}
	}
	if _, e := snowExcelSheet(entries, "Перечень загружений"+suffix, snowExcelVariantCatalogue(plan, suffix, snowExcelSampleCount(s)), false); e != nil {
		return e
	}
	for i, item := range plan {
		c := item.Case
		// Check the original expression BEFORE wrapping it in an applicability
		// condition. Otherwise IF(...,NA(),"") would escape the old string check.
		muFormula, _ := snowExcelMuFormula(s, c, SnowInterval{}, 5)
		if strings.Contains(muFormula, "NA()") {
			return fmt.Errorf("невозможно сформировать проверочный Excel для схемы %q, загружение %q: независимая формула μ не реализована", s.Shape, c.Title)
		}
		variant := rename(snowExcelVariantDataWithStatus(s, c, i, item.Applicability))
		if _, e := snowExcelSheet(entries, fmt.Sprintf("Вариант %d%s", i+1, suffix), variant, false); e != nil {
			return e
		}
		if s.Shape == "sphere_cone" && len(c.Polar) > 0 && !snowIsPartialSector(c) {
			if _, e := snowExcelSheet(entries, fmt.Sprintf("Сетка Б.15а вар.%d%s", i+1, suffix), snowExcelSpherePolar(s, c, i, "Геометрия Б.15а"+suffix, "Исходные данные"+suffix, "Коэффициенты"+suffix), false); e != nil {
				return e
			}
		}
		if (s.Shape == "cone" || s.Shape == "dome") && len(c.Polar) > 0 && !snowIsPartialSector(c) {
			if _, e := snowExcelSheet(entries, fmt.Sprintf("Сетка круглой кровли %d%s", i+1, suffix), snowExcelRoundPolar(s, c, i, "Исходные данные"+suffix, "Коэффициенты"+suffix), false); e != nil {
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
	for i := range plan {
		// Native Excel scatter chart reads the same formula cells as the load table.
		// Static geometry illustration above is intentionally unchanged.
		if err := snowExcelLiveChart(entries, fmt.Sprintf("Вариант %d%s", i+1, suffix), fmt.Sprintf("График %d%s", i+1, suffix), snowExcelSampleCount(s)); err != nil {
			return err
		}
	}

	return nil
}
