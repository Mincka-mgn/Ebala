package main

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Word is a customer-facing calculation section. It intentionally prints
// no intermediate numeric substitutions, Go identifiers, developer notes,
// debugging comments, or references to the independent Excel workbook.
func windWordFmt(v float64, decimals int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "—"
	}
	limit := math.Pow10(decimals)
	if math.Abs(v) < .5/limit {
		v = 0
	}
	s := decimalInput(v, decimals)
	if !strings.Contains(s, ",") {
		s += ",0"
	}
	parts := strings.SplitN(s, ",", 2)
	neg := ""
	if strings.HasPrefix(parts[0], "-") {
		neg = "−"
		parts[0] = strings.TrimPrefix(parts[0], "-")
	}
	digits := parts[0]
	for k := len(digits) - 3; k > 0; k -= 3 {
		digits = digits[:k] + " " + digits[k:]
	}
	return neg + digits + "," + parts[1]
}
func windWordHeader(values []string, widths []int) string {
	var b strings.Builder
	b.WriteString(`<w:tr><w:trPr><w:tblHeader w:val="true"/><w:cantSplit/></w:trPr>`)
	for i, value := range values {
		fmt.Fprintf(&b, `<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/></w:tcPr><w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="20"/></w:rPr><w:t>%s</w:t></w:r></w:p></w:tc>`, widths[i], tempXML(value))
	}
	b.WriteString(`</w:tr>`)
	return b.String()
}
func windWordMethod(i WindInput) string {
	var b strings.Builder
	b.WriteString(snowWordHeading("Методика определения ветровой нагрузки"))
	b.WriteString(snowWordParagraph("Расчёт выполнен по разделу 11 и приложению В СП 20.13330.2016. По таблице 11.1 определяют нормативное базовое ветровое давление w₀; по п. 11.1.5 назначают эквивалентную высоту zₑ; по таблице 11.2 определяют коэффициент изменения давления по высоте k(zₑ). Аэродинамические коэффициенты c принимают для соответствующих поверхностей по приложению В."))
	b.WriteString(pMath("wₘ = w₀ · k(zₑ) · c  (11.2)"))
	if i.PulseMode != "mean" {
		b.WriteString(snowWordParagraph("Пульсационную составляющую определяют по формуле (11.5) СП 20.13330.2016: коэффициент пульсации давления ζ(zₑ) — по таблице 11.4, коэффициент пространственной корреляции ν — по таблице 11.6."))
		b.WriteString(pMath("wₚ = wₘ · ζ(zₑ) · ν  (11.5)"))
	}
	b.WriteString(pMath("w = wₘ + wₚ  (11.1)"))
	b.WriteString(pMath("wрасч = γf · w  (п. 11.4)"))
	b.WriteString(snowWordParagraph("Ниже приведены схемы разбивки поверхностей на зоны и результаты определения ветрового давления по высотным отметкам. Знаки нагрузок сохранены: положительное значение — давление, отрицательное — разрежение."))
	return b.String()
}
func windWordZoneTitle(z WindZoneCalculation) string {
	title := fmt.Sprintf("Зона %s. %s", z.Code, z.Name)
	if z.Surface == "roof" && len(z.Rectangles) > 0 {
		a := z.Rectangles[0]
		if z.Code == "F" {
			title = "Зона F — наветренные углы"
		}
		if len(z.Rectangles) > 1 {
			title += fmt.Sprintf(", %d участка; ширина одного участка %s м; протяжённость одного участка %s м", len(z.Rectangles), windWordFmt(a.Y1-a.Y0, 2), windWordFmt(a.X1-a.X0, 2))
		} else {
			title += fmt.Sprintf("; ширина %s м; протяжённость %s м", windWordFmt(a.Y1-a.Y0, 2), windWordFmt(a.X1-a.X0, 2))
		}
	} else if z.Code == "D" || z.Code == "E" {
		title += "; ширина поперёк ветра " + windWordFmt(z.To-z.From, 2) + " м"
	} else {
		title += "; протяжённость вдоль ветра " + windWordFmt(z.To-z.From, 2) + " м"
	}
	return title
}
func windWordTable(c WindCalculation, title string) string {
	i := c.Input
	unit := "кПа"
	factor := 1. / 1000.
	precision := 3
	if i.DisplayUnit == "kgf" {
		unit = "кгс/м²"
		factor = kgfPerKN / 1000.
		precision = 1
	}
	widths := []int{1360, 2080, 2070, 2290, 2310}
	var b strings.Builder
	b.WriteString(snowWordHeading(title))
	b.WriteString(tempTableStart(widths))
	b.WriteString(windWordHeader([]string{"Отметка, м", "Средняя, " + unit, "Пульсационная, " + unit, "Итого нормативная, " + unit, "Итого расчётная, " + unit}, widths))
	values := c.Profile
	if len(values) == 0 {
		values = []WindProfilePoint{{Z: i.Z, Result: c.Result}}
	}
	for idx, p := range values {
		zLabel := windWordFmt(p.Z, 2)
		// Formula 11.1.5 may yield a genuine jump at a height boundary.
		// Two one-sided ordinates must not be printed as two identical 8,0 m
		// rows with different pressures; label their physical side explicitly.
		if idx+1 < len(values) && math.Abs(values[idx+1].Z-p.Z) < .005 && math.Abs(values[idx+1].Result.NormPa-p.Result.NormPa) > .00001 {
			zLabel += " (снизу)"
		} else if idx > 0 && math.Abs(values[idx-1].Z-p.Z) < .005 && math.Abs(values[idx-1].Result.NormPa-p.Result.NormPa) > .00001 {
			zLabel += " (сверху)"
		}
		b.WriteString(tempTableRow([]string{
			zLabel,
			windWordFmt(p.Result.MeanPa*factor, precision),
			windWordFmt(p.Result.PulsePa*factor, precision),
			windWordFmt(p.Result.NormPa*factor, precision),
			windWordFmt(p.Result.DesignPa*factor, precision),
		}, widths))
	}
	b.WriteString(`</w:tbl>`)
	return b.String()
}

// Kept for individually exported legacy calculation snapshots and tests;
// multi-zone project reports use windWordGroup instead.
func windWordAppend(c WindCalculation, title string) string {
	return windWordMethod(c.Input) + windWordTable(c, title)
}

// Describes all portions of the current physical surface, not just the
// highlighted area. This is a geometry caption, not a list of coefficients.
func windWordDimensions(v WindVariant, roof bool, alt string) string {
	parts := []string{}
	for _, z := range v.Zones {
		if roof {
			if z.Surface != "roof" || z.Alternative != alt || len(z.Rectangles) == 0 {
				continue
			}
			r := z.Rectangles[0]
			t := z.Code + ": ширина " + windWordFmt(r.Y1-r.Y0, 2) + " м, протяжённость " + windWordFmt(r.X1-r.X0, 2) + " м"
			if len(z.Rectangles) > 1 {
				t += fmt.Sprintf("; %d участка", len(z.Rectangles))
			}
			parts = append(parts, t)
		} else {
			if z.Surface == "roof" || z.Code == "D" || z.Code == "E" {
				continue
			}
			parts = append(parts, z.Code+": протяжённость "+windWordFmt(z.To-z.From, 2)+" м")
		}
	}
	if len(parts) == 0 {
		return ""
	}
	if roof {
		return "Размеры участков кровли: " + strings.Join(parts, "; ") + "."
	}
	return "Участки боковых стен вдоль направления ветра: " + strings.Join(parts, "; ") + "."
}

func windWordAddFigure(v WindVariant, roof bool, alt string, imageNum *int, rel *strings.Builder, entries map[string]string) (string, error) {
	png, err := windReportDiagram(v, roof, alt)
	if err != nil {
		return "", err
	}
	*imageNum++
	id := *imageNum
	path := fmt.Sprintf("word/media/wind_%d.png", id)
	entries[path] = string(png)
	fmt.Fprintf(rel, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/wind_%d.png"/>`, id, id)
	tag := tempWordImageInline(id, 1400, 470)
	tag = strings.ReplaceAll(tag, "Схема транспортной нагрузки", "Схема ветровых зон")
	return tag, nil
}
func windWordGroup(c WindCalculation, title string, imageNum *int, rel *strings.Builder, entries map[string]string) (string, error) {
	if len(c.Variants) == 0 {
		return windWordAppend(c, title), nil
	}
	var b strings.Builder
	i := c.Input
	b.WriteString(snowWordHeading(title))
	source := "ветровой район " + i.Region
	if i.ManualW0 {
		source = "введено вручную"
	}
	unit := "кПа"
	w0 := c.Result.W0Pa / 1000.
	prec := 3
	if i.DisplayUnit == "kgf" {
		unit = "кгс/м²"
		w0 = c.Result.W0Pa * kgfPerKN / 1000.
		prec = 1
	}
	b.WriteString(snowWordParagraph("Исходные данные: базовое ветровое давление w₀ = " + windWordFmt(w0, prec) + " " + unit + " (" + source + "); тип местности " + i.Terrain + "; коэффициент надёжности γf = " + windWordFmt(i.Gamma, 3) + "."))
	if i.Geometry == "building" {
		b.WriteString(snowWordParagraph("Геометрия здания: длина вдоль конька B = " + windWordFmt(i.Depth, 2) + " м; поперечный размер L = " + windWordFmt(i.Rho, 2) + " м; высота конька H = " + windWordFmt(i.Height, 2) + " м; подъём кровли h = " + windWordFmt(i.RoofRise, 2) + " м."))
	} else {
		b.WriteString(snowWordParagraph("Отдельно стоящая стена: длина L = " + windWordFmt(i.Rho, 2) + " м; высота H = " + windWordFmt(i.Height, 2) + " м."))
	}
	b.WriteString(windWordMethod(i))
	for _, v := range c.Variants {
		b.WriteString(snowWordHeading(v.Name))
		b.WriteString(snowWordParagraph("План здания. " + v.Name + ". Разбивка боковых стен на расчётные зоны:"))
		photo, err := windWordAddFigure(v, false, "", imageNum, rel, entries)
		if err != nil {
			return "", err
		}
		b.WriteString(photo)
		if dims := windWordDimensions(v, false, ""); dims != "" {
			b.WriteString(snowWordParagraph(dims))
		}
		hasRoof := false
		alt := ""
		for _, z := range v.Zones {
			if z.Surface == "roof" {
				hasRoof = true
				alt = z.Alternative
				break
			}
		}
		if hasRoof {
			b.WriteString(snowWordParagraph("План здания. " + v.Name + ". Разбивка двускатной кровли на расчётные зоны:"))
			photo, err = windWordAddFigure(v, true, alt, imageNum, rel, entries)
			if err != nil {
				return "", err
			}
			b.WriteString(photo)
			if dims := windWordDimensions(v, true, alt); dims != "" {
				b.WriteString(snowWordParagraph(dims))
			}
		}
		b.WriteString(snowWordHeading("Результаты для стен"))
		for _, z := range v.Zones {
			if z.Surface != "roof" {
				b.WriteString(windWordTable(z.Calculation, windWordZoneTitle(z)))
			}
		}
		if hasRoof {
			b.WriteString(snowWordHeading("Результаты для кровли"))
			previousAlt := ""
			for _, z := range v.Zones {
				if z.Surface != "roof" {
					continue
				}
				if z.Alternative != previousAlt {
					previousAlt = z.Alternative
					if v.ID == "CROSS" {
						b.WriteString(snowWordHeading("Знаковый вариант " + previousAlt + " по таблице В.3а СП 20.13330.2016"))
					}
				}
				b.WriteString(windWordTable(z.Calculation, windWordZoneTitle(z)))
			}
		}
	}
	return b.String(), nil
}

var windExprNameRx = regexp.MustCompile(`[A-Za-z_][A-Za-z_0-9]*`)

func windExcelExpr(db *windDB, id string, vars map[string]string) (string, error) {
	f, ok := db.Formulas[id]
	if !ok {
		return "", fmt.Errorf("Нет формулы %s", id)
	}
	errText := ""
	expr := windExprNameRx.ReplaceAllStringFunc(f.Expression, func(id string) string {
		if id == "sqrt" {
			return "SQRT"
		}
		if id == "pi" {
			return "PI()"
		}
		val, ok := vars[id]
		if !ok {
			errText = id
			return id
		}
		return "(" + val + ")"
	})
	if errText != "" {
		return "", fmt.Errorf("Выражение %s: неизвестная переменная %s", id, errText)
	}
	return "=" + strings.ReplaceAll(expr, "**", "^"), nil
}
func windExcelTables(entries map[string]string, db *windDB) error {
	keys := []string{"11.1", "11.2", "11.3", "11.4", "11.5"}
	for _, key := range keys {
		table := db.Tables[key]
		var rows strings.Builder
		rows.WriteString(tempExcelRow(1, "На этом листе приведены нормативные данные СП 20.13330.2016, таблица "+key))
		rows.WriteString(tempExcelRow(2, table.Columns...))
		for k, row := range table.Rows {
			vals := make([]string, len(row))
			for i, x := range row {
				vals[i] = fmt.Sprint(x)
			}
			rows.WriteString(tempExcelRow(k+3, vals...))
		}
		if _, e := snowExcelSheet(entries, "Ветер "+key, rows.String(), false); e != nil {
			return e
		}
	}
	{
		t := db.Tables["11.6"]
		var rows strings.Builder
		rows.WriteString(tempExcelRow(1, "На этом листе приведены данные СП 20.13330.2016, таблица 11.6: коэффициент пространственной корреляции ν"))
		rows.WriteString(tempExcelRow(2, "Оси: ρ, χ, м"))
		headers := []string{"ρ / χ"}
		for _, v := range t.Arguments["chi_m"] {
			headers = append(headers, strconv.FormatFloat(v, 'f', -1, 64))
		}
		rows.WriteString(tempExcelRow(3, headers...))
		for i, r := range t.Values {
			v := []string{strconv.FormatFloat(t.Arguments["rho_m"][i], 'f', -1, 64)}
			for _, x := range r {
				v = append(v, strconv.FormatFloat(x, 'f', -1, 64))
			}
			rows.WriteString(tempExcelRow(i+4, v...))
		}
		if _, e := snowExcelSheet(entries, "Ветер 11.6", rows.String(), false); e != nil {
			return e
		}
	}
	for _, id := range []string{"В.1", "В.2"} {
		rows := []windRefRow{}
		for _, r := range db.Refs {
			if r.TableID == id && r.Condition == "" && r.ResultValue != "" {
				rows = append(rows, r)
			}
		}
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].SchemeID < rows[j].SchemeID })
		var out strings.Builder
		out.WriteString(tempExcelRow(1, "На этом листе приведены аэродинамические коэффициенты СП 20.13330.2016, таблица "+id))
		out.WriteString(tempExcelRow(2, "Зона", "Аэродинамический коэффициент"))
		for k, row := range rows {
			out.WriteString(tempExcelRow(3+k, row.SchemeID, row.ResultValue))
		}
		if _, e := snowExcelSheet(entries, "Ветер "+id, out.String(), false); e != nil {
			return e
		}
	}
	return windExcelRoofTables(entries, db)
}

// Export the exact normative source data for roof tables, including sign
// alternatives. Every cell comes from the checked JSON, not Go constants.
func windExcelRoofTables(entries map[string]string, db *windDB) error {
	low, high, ridge, _, _, err := db.roofTableAxisGroups()
	if err != nil {
		return err
	}
	for _, spec := range []struct {
		name, source string
		branch       int
		angles       []float64
		zones        []string
	}{
		{"Ветер В.3а 1", "В.3а", 1, low, []string{"F", "G", "H", "I", "J"}},
		{"Ветер В.3а 2", "В.3а", 2, high, []string{"F", "G", "H", "I", "J"}},
		{"Ветер В.3б", "В.3б", 1, ridge, []string{"F", "G", "H", "I"}},
	} {
		var out strings.Builder
		out.WriteString(tempExcelRow(1, "На этом листе приведены коэффициенты СП 20.13330.2016, таблица "+spec.source+", знаковая ветвь "+strconv.Itoa(spec.branch)))
		out.WriteString(tempExcelRow(2, "Источник", spec.source))
		out.WriteString(tempExcelRow(3, append([]string{"β, град"}, spec.zones...)...))
		for idx, beta := range spec.angles {
			row := []string{xnum(beta)}
			for _, zone := range spec.zones {
				c, e := db.roofCoefficient(spec.source, zone, beta, spec.branch)
				if e != nil {
					return fmt.Errorf("Таблица Excel %s: %w", spec.name, e)
				}
				row = append(row, xnum(c))
			}
			out.WriteString(tempExcelRow(idx+4, row...))
		}
		if _, e := snowExcelSheet(entries, spec.name, out.String(), false); e != nil {
			return e
		}
	}
	return nil
}

// Linear interpolation INSIDE one normative table/alternative. The angle and
// both c endpoints remain visible and editable in the normative XLSX sheet.
func windExcelRoofInterp(sheet string, last int, beta, zone string) string {
	axis := "'" + sheet + "'!$A$4:$A$" + strconv.Itoa(last)
	headers := "'" + sheet + "'!$B$3:$F$3"
	vals := "'" + sheet + "'!$B$4:$F$" + strconv.Itoa(last)
	if sheet == "Ветер В.3б" {
		headers = "'" + sheet + "'!$B$3:$E$3"
		vals = "'" + sheet + "'!$B$4:$E$" + strconv.Itoa(last)
	}
	col := "MATCH(" + zone + "," + headers + ",0)"
	i := "IF(" + beta + "<=INDEX(" + axis + ",1),1,IF(" + beta + ">=INDEX(" + axis + ",ROWS(" + axis + ")),ROWS(" + axis + ")-1,MATCH(" + beta + "," + axis + ",1)))"
	low := "INDEX(" + axis + "," + i + ")"
	high := "INDEX(" + axis + ",(" + i + ")+1)"
	loVal := "INDEX(" + vals + "," + i + "," + col + ")"
	hiVal := "INDEX(" + vals + ",(" + i + ")+1," + col + ")"
	return "(" + loVal + "+(" + beta + "-" + low + ")/(" + high + "-" + low + ")*(" + hiVal + "-" + loVal + "))"
}

// Derive ρ/χ from editable overall geometry and the normative V.4 geometry.
// These are Excel calculations: do NOT forward calculated Go zone dimensions.
func windExcelRoofCorrelation(in WindInput, ref func(int) string) (string, string, string) {
	B, L, H := ref(25), ref(24), ref(8)
	e := "MIN(" + L + ",2*" + H + ")"
	if in.RoofFlow == "CROSS" {
		e = "MIN(" + B + ",2*" + H + ")"
	}
	var rho, chi string
	if in.RoofFlow == "CROSS" {
		a := "MIN(0.1*B45," + L + "/2)"
		chi = a
		if in.AeroZone == "H" || in.AeroZone == "I" {
			chi = "MAX(0," + L + "/2-" + a + ")"
		}
		switch in.AeroZone {
		case "F":
			rho = "MIN(B45/4," + B + "/2)"
		case "G":
			rho = "MAX(0," + B + "-2*MIN(B45/4," + B + "/2))"
		default:
			rho = B
		}
	} else {
		a := "MIN(B45/10," + B + ")"
		b := "MIN(B45/2," + B + ")"
		switch in.AeroZone {
		case "F", "G":
			chi = a
		case "H":
			chi = "MAX(0," + b + "-" + a + ")"
		default:
			chi = "MAX(0," + B + "-" + b + ")"
		}
		switch in.AeroZone {
		case "F":
			rho = "MIN(B45/4," + L + "/2)"
		case "G":
			rho = "MAX(0," + L + "/2-MIN(B45/4," + L + "/2))"
		default:
			rho = L + "/2"
		}
	}
	return "=" + e, "=" + rho, "=" + chi
}

// Geometry is calculated AGAIN inside the workbook from accepted B, L, H.
// The normative patch topology (zone code) is the selected worksheet branch;
// widths and extents are never copied from calculated Go rectangles.
func wind064ZoneGeometry(i WindInput, ref func(int) string) (width, extent, count string) {
	L, B, H := ref(24), ref(25), ref(8)
	if i.Geometry == "free_wall" {
		a := "MIN(" + L + ",0.3*" + H + ")"
		b := "MIN(" + L + ",2.3*" + H + ")"
		c := "MIN(" + L + ",6.3*" + H + ")"
		switch i.AeroZone {
		case "A":
			width = "=" + a
		case "B":
			width = "=MAX(0," + b + "-" + a + ")"
		case "C":
			width = "=MAX(0," + c + "-" + b + ")"
		case "D":
			width = "=MAX(0," + L + "-" + c + ")"
		}
		return width, "", "=1"
	}
	if i.AeroTable != "В.3а" && i.AeroTable != "В.3б" {
		cross := "IF(" + ref(27) + "=\"Ветер поперёк конька\"," + B + "," + L + ")"
		along := "IF(" + ref(27) + "=\"Ветер поперёк конька\"," + L + "," + B + ")"
		e := "MIN(" + cross + ",2*" + H + ")"
		a := "MIN(" + along + "," + e + "/5)"
		b := "MIN(" + along + "," + e + ")"
		switch i.AeroZone {
		case "D", "E":
			return "=" + cross, "", "=1"
		case "A":
			extent = "=" + a
		case "B":
			extent = "=MAX(0," + b + "-" + a + ")"
		case "C":
			extent = "=MAX(0," + along + "-" + b + ")"
		}
		return "", extent, "=2"
	}
	// B45 contains the same independently determined e as used by Fig V.4.
	e, f, a := "B45", "", ""
	if i.RoofFlow == "CROSS" {
		f = "MIN(" + e + "/4," + B + "/2)"
		a = "MIN(" + e + "/10," + L + "/2)"
		switch i.AeroZone {
		case "F":
			width, extent, count = "="+f, "="+a, "=2"
		case "G":
			width, extent, count = "=MAX(0,"+B+"-2*"+f+")", "="+a, "=1"
		case "H":
			width, extent, count = "="+B, "=MAX(0,"+L+"/2-"+a+")", "=1"
		case "J":
			width, extent, count = "="+B, "=MAX(0,MIN("+L+","+L+"/2+"+e+"/10)-"+L+"/2)", "=1"
		case "I":
			width, extent, count = "="+B, "=MAX(0,"+L+"-MIN("+L+","+L+"/2+"+e+"/10))", "=1"
		}
	} else {
		f = "MIN(" + e + "/4," + L + "/2)"
		a = "MIN(" + e + "/10," + B + ")"
		b := "MIN(" + e + "/2," + B + ")"
		switch i.AeroZone {
		case "F":
			width, extent, count = "="+f, "="+a, "=2"
		case "G":
			width, extent, count = "=MAX(0,"+L+"/2-"+f+")", "="+a, "=2"
		case "H":
			width, extent, count = "="+L+"/2", "=MAX(0,"+b+"-"+a+")", "=2"
		case "I":
			width, extent, count = "="+L+"/2", "=MAX(0,"+B+"-"+b+")", "=2"
		}
	}
	return width, extent, count
}

func windExcelSingle(entries map[string]string, db *windDB, c WindCalculation, title string, index int) error {
	i := c.Input
	src := fmt.Sprintf("Ветер Исходные %d", index)
	dst := fmt.Sprintf("Ветер Расчёт %d", index)
	A := func(r int) string { return fmt.Sprintf("'%s'!B%d", src, r) }
	var b strings.Builder
	b.WriteString(tempExcelInputRow(1, "На этом листе вводятся исходные данные расчёта ветра. Изменения влияют на коэффициенты и нагрузки на расчётных листах."))
	b.WriteString(tempExcelInputRow(2, "Название", title))
	// Input sheet rows have stable addresses because the independent Excel
	// formulas reference them. A technical parameter that does not apply is
	// OMITTED (not printed as a zero or a development flag).
	direction := "Ветер вдоль конька"
	if i.RoofFlow == "CROSS" {
		direction = "Ветер поперёк конька"
	}
	if i.Geometry == "free_wall" {
		direction = "Ветер поперёк стенки"
	}
	geometry := "Здание с двускатной крышей"
	if i.Geometry == "free_wall" {
		geometry = "Отдельно стоящая стена"
	}
	pressureSource := "По ветровому району"
	if i.ManualW0 {
		pressureSource = "Заданное значение"
	}
	methodK := "По таблице 11.2"
	if i.KMode == "formula" {
		methodK = "По формуле (11.4)"
	}
	pulseName := "Не учитывать пульсацию"
	if i.PulseMode != "mean" {
		pulseName = "Упрощённый учёт пульсации"
	}
	unit := "кПа"
	if i.DisplayUnit == "kgf" {
		unit = "кгс/м²"
	}
	structName := ""
	if i.PulseMode != "mean" {
		switch i.StructuralType {
		case "steel":
			structName = "Стальное сооружение"
		case "mixed":
			structName = "Смешанное без сплошных ограждений или стеклянные конструкции"
		default:
			structName = "Железобетонное / каменное / стальной каркас со сплошными ограждениями"
		}
	}
	fields := map[int][]string{
		3:  {"Ветровой район", i.Region},
		4:  {"Источник базового давления", pressureSource},
		5:  {"Значение ветрового давления w₀, кПа", xnum(i.W0KPa)},
		6:  {"Тип местности", i.Terrain},
		7:  {"Тип сооружения", geometry},
		8:  {"Высота сооружения H, м", xnum(i.Height)},
		11: {"Источник аэродинамического коэффициента", i.AeroTable},
		12: {"Расчётная зона", i.AeroZone},
		14: {"Определение коэффициента изменения давления k", methodK},
		15: {"Учёт пульсации", pulseName},
		20: {"Коэффициент надёжности по нагрузке γf", xnum(i.Gamma)},
		21: {"Единица вывода результатов", unit},
		23: {"Шаг эпюры, м", xnum(i.ProfileStep)},
		24: {"Ширина здания L, м", xnum(i.RoofSpan)},
		25: {"Длина здания B, м", xnum(i.RoofLength)},
		27: {"Направление ветра", direction},
	}
	if i.Geometry == "free_wall" {
		fields[24] = []string{"Длина отдельно стоящей стены L, м", xnum(i.RoofSpan)}
		delete(fields, 25)
	} else {
		fields[9] = []string{"Поперечный размер здания относительно ветра, м", ""}
		fields[22] = []string{"Подъём двускатной крыши h, м", xnum(i.RoofRise)}
	}
	if i.AeroTable == "manual" {
		fields[11] = []string{"Источник аэродинамического коэффициента", "Заданное значение"}
		fields[13] = []string{"Аэродинамический коэффициент c", xnum(i.ManualC)}
	}
	if i.AeroTable == "В.3а" && i.RoofBranch != "" {
		fields[26] = []string{"Знаковый вариант кровельной нагрузки", i.RoofBranch}
	}
	if i.PulseMode != "mean" {
		if i.AeroTable != "В.3а" && i.AeroTable != "В.3б" {
			fields[16] = []string{"Размер зоны ρ для коэффициента корреляции, м", xnum(i.Rho)}
			fields[17] = []string{"Размер зоны χ для коэффициента корреляции, м", xnum(i.Chi)}
		}
		fields[19] = []string{"Логарифмический декремент колебаний δ (п. 11.1.10)", xnum(i.Delta)}
		fields[28] = []string{"Конструктивная система", structName}
	}
	for row := 3; row <= 28; row++ {
		if cell, found := fields[row]; found {
			if row == 9 { // cross-wind dimension derives from actual editable B/L
				b.WriteString(tempExcelRow(row, cell[0], "=IF(B27=\"Ветер поперёк конька\",B25,B24)"))
			} else {
				b.WriteString(tempExcelInputRow(row, cell...))
			}
		}
	}
	if _, e := snowExcelSheet(entries, src, b.String(), false); e != nil {
		return e
	}
	var rows strings.Builder
	add := func(r int, label, formula string) { rows.WriteString(tempExcelRow(r, label, formula)) }
	add(1, "На этом листе выполнен независимый расчёт по формулам и таблицам СП 20.13330.2016; исходные данные редактируются отдельно.", "")
	add(2, "Параметр", "Формула или результат")
	w0Multiplier := strings.TrimPrefix(db.Formulas["11.1.lookup_w0"].Expression, "lookup(11.1,wind_region).w0_kPa*")
	if _, e := strconv.ParseFloat(w0Multiplier, 64); e != nil {
		return fmt.Errorf("Не удалось разобрать нормативную конверсию w0: %w", e)
	}
	add(3, "w₀, кПа", fmt.Sprintf("=IF(%s=\"Заданное значение\",%s,VLOOKUP(%s,'Ветер 11.1'!$A$3:$B$10,2,FALSE))", A(4), A(5), A(3)))
	zeFormula := windExcelZeExpr(A(7), A(8), A(9), A(8))
	if i.AeroTable == "В.3а" || i.AeroTable == "В.3б" {
		zeFormula = "=" + A(8)
	}
	add(4, "Эквивалентная высота zₑ, м", zeFormula)
	kTable := "'Ветер 11.2'!$A$3:$A$13"
	kData := "'Ветер 11.2'!$A$3:$D$13"
	col := fmt.Sprintf("MATCH(%s,'Ветер 11.2'!$A$2:$D$2,0)", A(6))
	add(5, "Тип местности: номер столбца", "="+col)
	add(6, "Номер нижнего узла высоты", fmt.Sprintf("=IF(B4<=INDEX(%s,1),1,IF(B4>=INDEX(%s,ROWS(%s)),ROWS(%s)-1,MATCH(B4,%s,1)))", kTable, kTable, kTable, kTable, kTable))
	add(7, "Нижняя высота, м", fmt.Sprintf("=INDEX(%s,B6)", kTable))
	add(8, "Верхняя высота, м", fmt.Sprintf("=INDEX(%s,B6+1)", kTable))
	add(9, "k нижнего узла", fmt.Sprintf("=INDEX(%s,B6,B5)", kData))
	add(10, "k верхнего узла", fmt.Sprintf("=INDEX(%s,B6+1,B5)", kData))
	add(11, "k по табл.11.2", fmt.Sprintf("=IF(B4<=INDEX(%s,1),B9,B9+(B4-B7)/(B8-B7)*(B10-B9))", kTable))
	add(12, "α по табл.11.3", fmt.Sprintf("=INDEX('Ветер 11.3'!$B$3:$D$5,1,B5-1)"))
	add(13, "k₁₀ по табл.11.3", fmt.Sprintf("=INDEX('Ветер 11.3'!$B$3:$D$5,2,B5-1)"))
	kFormula, e := windExcelExpr(db, "11.4", map[string]string{"k10": "B13", "z_e": "B4", "alpha_terrain": "B12"})
	if e != nil {
		return e
	}
	add(14, "k по формуле (11.4)", kFormula)
	add(15, "Принятый коэффициент k", fmt.Sprintf("=IF(%s=\"По формуле (11.4)\",B14,B11)", A(14)))
	low, high, ridge, forkMin, forkMax, err := db.roofTableAxisGroups()
	if err != nil {
		return err
	}
	roof1 := windExcelRoofInterp("Ветер В.3а 1", len(low)+3, "B44", A(12))
	roof2 := windExcelRoofInterp("Ветер В.3а 2", len(high)+3, "B44", A(12))
	roofB := windExcelRoofInterp("Ветер В.3б", len(ridge)+3, "B44", A(12))
	axisRow := func(angle float64) (int, error) {
		for ix, b := range low {
			if b == angle {
				return 4 + ix, nil
			}
		}
		return 0, fmt.Errorf("Ветер В.3а: нет угла %.3f в основной ветви", angle)
	}
	rmin, err := axisRow(forkMin)
	if err != nil {
		return err
	}
	rmax, err := axisRow(forkMax)
	if err != nil {
		return err
	}
	t15 := fmt.Sprintf("'Ветер В.3а 1'!$A$%d", rmin)
	t30 := fmt.Sprintf("'Ветер В.3а 1'!$A$%d", rmax)
	roofA := fmt.Sprintf("IF(B44<%s,%s,IF(B44<=%s,IF(%s=2,%s,%s),%s))", t15, roof1, t30, A(26), roof2, roof1, roof2)
	add(16, "Аэродинамический коэффициент c", fmt.Sprintf("=IF(%s=\"Заданное значение\",%s,IF(%s=\"В.1\",VLOOKUP(%s,'Ветер В.1'!$A$3:$B$6,2,FALSE),IF(%s=\"В.2\",VLOOKUP(%s,'Ветер В.2'!$A$3:$B$7,2,FALSE),IF(%s=\"В.3а\",%s,%s))))", A(11), A(13), A(11), A(12), A(11), A(12), A(11), roofA, roofB))
	mean, e := windExcelExpr(db, "11.2", map[string]string{"w0": "B3*" + w0Multiplier, "k": "B15", "c_aerodynamic": "B16"})
	if e != nil {
		return e
	}
	add(17, "Средняя составляющая wₘ, Па", mean)
	// Only display and evaluate pulsation coefficients if the selected mode
	// actually includes formula (11.5). δ and f₁ are NOT used by (11.5).
	if i.PulseMode != "mean" {
		zData := "'Ветер 11.4'!$A$3:$D$13"
		add(18, "Коэффициент пульсации ветрового давления ζ", fmt.Sprintf("=IF(B4<=INDEX(%s,1,1),INDEX(%s,1,B5),INDEX(%s,B6,B5)+(B4-B7)/(B8-B7)*(INDEX(%s,B6+1,B5)-INDEX(%s,B6,B5)))", zData, zData, zData, zData, zData))
		add(19, "Корреляция пульсаций: таблица 11.6", "Билинейная интерполяция по размерам зоны")
		rhoRef, chiRef := A(16), A(17)
		if i.AeroTable == "В.3а" || i.AeroTable == "В.3б" {
			rhoRef, chiRef = "B46", "B47"
		}
		xAxis := "'Ветер 11.6'!$A$4:$A$10"
		yAxis := "'Ветер 11.6'!$B$3:$H$3"
		xy := "'Ветер 11.6'!$B$4:$H$10"
		add(21, "Нижний индекс ρ", fmt.Sprintf("=IF(%s<=INDEX(%s,1),1,IF(%s>=INDEX(%s,ROWS(%s)),ROWS(%s)-1,MATCH(%s,%s,1)))", rhoRef, xAxis, rhoRef, xAxis, xAxis, xAxis, rhoRef, xAxis))
		add(22, "Нижний индекс χ", fmt.Sprintf("=IF(%s<=INDEX(%s,1,1),1,IF(%s>=INDEX(%s,1,COLUMNS(%s)),COLUMNS(%s)-1,MATCH(%s,%s,1)))", chiRef, yAxis, chiRef, yAxis, yAxis, yAxis, chiRef, yAxis))
		add(23, "ρниж, м", fmt.Sprintf("=INDEX(%s,B21)", xAxis))
		add(24, "ρверх, м", fmt.Sprintf("=INDEX(%s,B21+1)", xAxis))
		add(25, "χниж, м", fmt.Sprintf("=INDEX(%s,1,B22)", yAxis))
		add(26, "χверх, м", fmt.Sprintf("=INDEX(%s,1,B22+1)", yAxis))
		add(27, "ν в нижней точке (ρ,χ)", fmt.Sprintf("=INDEX(%s,B21,B22)", xy))
		add(28, "ν при ρниж и χверх", fmt.Sprintf("=INDEX(%s,B21,B22+1)", xy))
		add(29, "ν при ρверх и χниж", fmt.Sprintf("=INDEX(%s,B21+1,B22)", xy))
		add(30, "ν в верхней точке (ρ,χ)", fmt.Sprintf("=INDEX(%s,B21+1,B22+1)", xy))
		add(31, "Коэффициент пространственной корреляции ν", fmt.Sprintf("=B27+(%s-B23)/(B24-B23)*(B29-B27)+(%s-B25)/(B26-B25)*((B28-B27)+(%s-B23)/(B24-B23)*(B30-B29-B28+B27))", rhoRef, chiRef, rhoRef))
	}
	pulseFormula := "=0" // without pulsation there is no ζ or ν calculation
	if i.PulseMode != "mean" {
		pulse5, e := windExcelExpr(db, "11.5", map[string]string{"w_m": "B17", "zeta": "B18", "v_corr": "B31"})
		if e != nil {
			return e
		}
		pulseFormula = pulse5
	}
	add(35, "Пульсационная составляющая wₚ, Па", pulseFormula)
	norm, e := windExcelExpr(db, "11.1", map[string]string{"w_m": "B17", "w_g": "B35"})
	if e != nil {
		return e
	}
	add(36, "Итого нормативная нагрузка w, Па", norm)
	add(37, "Итого расчётная нагрузка, Па", fmt.Sprintf("=B36*%s", A(20)))
	if i.AeroTable == "В.3а" || i.AeroTable == "В.3б" {
		add(44, "Уклон β двускатной крыши, град", fmt.Sprintf("=IF(%s>0,DEGREES(ATAN(2*%s/%s)),0)", A(24), A(22), A(24)))
	}
	eExpr, rhoExpr, chiExpr := windExcelRoofCorrelation(i, A)
	if i.AeroTable == "В.3а" || i.AeroTable == "В.3б" {
		add(45, "Характерный размер e для разбивки кровли по рисунку В.4, м", eExpr)
		if i.PulseMode != "mean" {
			add(46, "Размер ρ для пространственной корреляции пульсаций, м", rhoExpr)
			add(47, "Размер χ для пространственной корреляции пульсаций, м", chiExpr)
		}
	}
	widthExpr, extentExpr, countExpr := wind064ZoneGeometry(i, A)
	if widthExpr != "" {
		add(48, "Ширина зоны поперёк ветра, м", widthExpr)
	}
	if extentExpr != "" {
		add(49, "Протяжённость зоны вдоль ветра, м", extentExpr)
	}
	add(50, "Количество участков данной зоны", countExpr)
	if i.AeroTable == "В.3а" || i.AeroTable == "В.3б" {
		add(51, "Геометрия зон кровли", "Рисунок В.4 СП 20.13330.2016: ширина поперёк ветра, протяжённость вдоль ветра")
	} else {
		add(51, "Геометрия стеновых зон", "Линейные участки вдоль стены; поперечный размер стены задаётся её геометрией")
	}
	if _, e := snowExcelSheet(entries, dst, rows.String(), false); e != nil {
		return e
	}
	return nil
}
func boolNumber(b bool) int {
	if b {
		return 1
	}
	return 0
}
func windExcelAdd(entries map[string]string, calculations []WindCalculation, titles []string) error {
	calculations, titles = windExpandReports(calculations, titles)
	if len(calculations) == 0 {
		return nil
	}
	db, e := getWindDB()
	if e != nil {
		return e
	}
	for j, c := range calculations {
		if c.Result.DataSHA256 != db.SHA {
			return fmt.Errorf("Ветер №%d: сохранённый расчёт выполнен по другой версии базы; при экспорте независимого Excel требуется совпадение набора нормативных данных", j+1)
		}
	}
	if err := windExcelTables(entries, db); err != nil {
		return err
	}
	for j, c := range calculations {
		title := fmt.Sprintf("Ветровая нагрузка %d", j+1)
		if j < len(titles) {
			title = titles[j]
		}
		if e := windExcelSingle(entries, db, c, title, j+1); e != nil {
			return e
		}
		if len(c.Profile) > 0 {
			if e := windExcelEpure(entries, db, c, j+1); e != nil {
				return e
			}
		}
	}
	return nil
}

// For Excel geometry follows the same approved 11.1.5 and В.1.1 branches.
func windExcelZeExpr(geometry, h, d, z string) string {
	low := fmt.Sprintf("IF(%s<=%s,%s", h, d, h)
	mid := fmt.Sprintf("IF(%s<=2*%s,IF(%s>=%s-%s,%s,%s)", h, d, z, h, d, h, d)
	high := fmt.Sprintf("IF(%s>=%s-%s,%s,IF(%s<=%s,%s,%s))", z, h, d, h, z, d, d, z)
	building := low + "," + mid + "," + high + "))"
	return "=IF(" + geometry + "=\"Отдельно стоящая стена\"," + h + ",IF(" + geometry + "=\"tower\"," + z + "," + building + "))"
}

func lenRoofIntervals(in WindInput) int {
	zs, err := WindRoofProfileHeights(in.Height, in.RoofRise, in.ProfileStep)
	if err != nil || len(zs) < 2 {
		return 1
	}
	return len(zs) - 1
}

// Coordinates are derived from the editable source dimensions and user step.
// Structural row count is fixed at export (as in the other Ebala Excel
// reports), but changing H or the step changes the ordinate formulas.
func windExcelProfileHeight(in WindInput, z float64, ref func(int) string) string {
	H, d, step := ref(8), ref(9), ref(23)
	if in.AeroTable == "В.3а" || in.AeroTable == "В.3б" {
		if math.Abs(z-in.Height) < 1e-8 {
			return "=" + H
		}
		low := math.Max(0, in.Height-in.RoofRise)
		if math.Abs(z-low) < 1e-8 {
			return "=MAX(0," + H + "-" + ref(22) + ")"
		}
		span := in.Height - low
		if span <= 0 {
			return "=" + H
		}
		// Number of intervals recalculates from the independent Excel input
		// H, h, and step. The number of sheet rows is fixed at export.
		n := int(math.Round((z - low) / span * float64(lenRoofIntervals(in))))
		bottom := "MAX(0," + H + "-" + ref(22) + ")"
		actualHeight := "(" + H + "-" + bottom + ")"
		count := "MAX(5,ROUNDUP(" + actualHeight + "/" + step + ",0))"
		return fmt.Sprintf("=MIN(%s,%s+%s*%d/(%s))", H, bottom, actualHeight, n, count)
	}
	clamp := func(s string) string { return "=MIN(" + H + "," + s + ")" }
	if z == 0 {
		return "=0"
	}
	if z == in.Height {
		return "=" + H
	}
	if math.Abs(z-5) < 1e-10 {
		return clamp("5")
	}
	if in.Geometry == "building" {
		upper := in.Height - in.Depth
		if upper > 0 && upper < in.Height {
			if z == upper {
				return clamp(H + "-" + d)
			}
			if z == math.Nextafter(upper, 0) {
				return "=MAX(0," + H + "-" + d + "-0.00000001)"
			}
		}
		if in.Depth > 0 && in.Depth < in.Height && z == in.Depth {
			return clamp(d)
		}
	}
	if in.ProfileStep == 0 {
		if z > 5 {
			n := math.Round((z - 5) / 2.5)
			if math.Abs(z-(5+n*2.5)) < 1e-8 {
				return clamp(fmt.Sprintf("5+2.5*%.0f", n))
			}
		}
	} else {
		base := 0.0
		if z > 5 {
			base = 5
		}
		n := math.Round((z - base) / in.ProfileStep)
		if math.Abs(z-(base+n*in.ProfileStep)) < 1e-8 {
			if base == 0 {
				return clamp(fmt.Sprintf("MIN(5,%s*%.0f)", step, n))
			}
			return clamp(fmt.Sprintf("5+%s*%.0f", step, n))
		}
	}
	// A final deterministic fallback for any additional geometry/normative
	// grid point: keep it height-linked, never a pressure value from Go.
	return clamp(fmt.Sprintf("%.17g", z))
}

// A full, independently recalculating profile. Each elevation has its own
// table interpolation, normative load and design load formulas.
func windExcelEpure(entries map[string]string, db *windDB, c WindCalculation, index int) error {
	src := fmt.Sprintf("Ветер Исходные %d", index)
	calc := fmt.Sprintf("Ветер Расчёт %d", index)
	srcRef := func(row int) string { return fmt.Sprintf("'%s'!$B$%d", src, row) }
	calcRef := func(row int) string { return fmt.Sprintf("'%s'!$B$%d", calc, row) }
	var out strings.Builder
	out.WriteString(tempExcelRow(1, "На этом листе приведена эпюра ветровой нагрузки: все ординаты рассчитываются формулами Excel по исходным и нормативным данным."))
	out.WriteString(tempExcelRow(2, "z, м", "zₑ, м", "k(zₑ)", "ζ(zₑ)", "wₘ, Па", "wₚ, Па", "wн, Па", "wр, Па", "wн, выбранная единица", "wр, выбранная единица"))
	kData := "'Ветер 11.2'!$A$3:$D$13"
	kAxis := "'Ветер 11.2'!$A$3:$A$13"
	zData := "'Ветер 11.4'!$A$3:$D$13"
	zAxis := "'Ветер 11.4'!$A$3:$A$13"
	col := "MATCH(" + srcRef(6) + ",'Ветер 11.2'!$A$2:$D$2,0)"
	for j, point := range c.Profile {
		r := j + 3
		a := fmt.Sprintf("A%d", r)
		b := fmt.Sprintf("B%d", r)
		k := fmt.Sprintf("C%d", r)
		zeta := fmt.Sprintf("D%d", r)
		mean := fmt.Sprintf("E%d", r)
		pulse := fmt.Sprintf("F%d", r)
		norm := fmt.Sprintf("G%d", r)
		hExpr := windExcelProfileHeight(c.Input, point.Z, srcRef)
		ze := windExcelZeExpr(srcRef(7), srcRef(8), srcRef(9), a)
		if c.Input.AeroTable == "В.3а" || c.Input.AeroTable == "В.3б" {
			ze = "=" + calcRef(4)
		}
		interp := func(axis, data string) string {
			idx := "IF(" + b + "<=INDEX(" + axis + ",1),1,IF(" + b + ">=INDEX(" + axis + ",ROWS(" + axis + ")),ROWS(" + axis + ")-1,MATCH(" + b + "," + axis + ",1)))"
			lo := "INDEX(" + axis + "," + idx + ")"
			hi := "INDEX(" + axis + ",(" + idx + ")+1)"
			v0 := "INDEX(" + data + "," + idx + "," + col + ")"
			v1 := "INDEX(" + data + ",(" + idx + ")+1," + col + ")"
			return "=IF(" + b + "<=INDEX(" + axis + ",1)," + v0 + "," + v0 + "+(" + b + "-" + lo + ")/(" + hi + "-" + lo + ")*(" + v1 + "-" + v0 + "))"
		}
		// Interpolation strings use source column names. No intermediate value
		// from the Go profile is written into normative-load Excel cells.
		kv := interp(kAxis, kData)
		zv := ""
		if c.Input.PulseMode != "mean" {
			zv = interp(zAxis, zData)
		}
		m, e := windExcelExpr(db, "11.2", map[string]string{"w0": "(" + calcRef(3) + "*1000)", "k": k, "c_aerodynamic": calcRef(16)})
		if e != nil {
			return e
		}
		pp, e := windExcelExpr(db, "11.5", map[string]string{"w_m": mean, "zeta": zeta, "v_corr": calcRef(31)})
		if e != nil {
			return e
		}
		if c.Input.PulseMode == "mean" {
			pp = "=0"
		}
		nn, e := windExcelExpr(db, "11.1", map[string]string{"w_m": mean, "w_g": pulse})
		if e != nil {
			return e
		}
		design := "=" + norm + "*" + srcRef(20)
		normOut := fmt.Sprintf("=IF(%s=\"кгс/м²\",G%d*%.12g,G%d/1000)", srcRef(21), r, kgfPerKN/1000, r)
		designOut := fmt.Sprintf("=IF(%s=\"кгс/м²\",H%d*%.12g,H%d/1000)", srcRef(21), r, kgfPerKN/1000, r)
		out.WriteString(tempExcelRow(r, hExpr, ze, kv, zv, m, pp, nn, design, normOut, designOut))
	}
	_, e := snowExcelSheet(entries, fmt.Sprintf("Ветер Эпюра %d", index), out.String(), false)
	return e
}
