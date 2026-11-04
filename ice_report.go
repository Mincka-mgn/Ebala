package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"
)

func iceFigure(s IceLoadCalc) ([]byte, error) {
	if len(s.Points) == 0 {
		return nil, fmt.Errorf("Нет результатов для высотной эпюры")
	}
	f := newSnowFigure()
	black := color.RGBA{22, 28, 34, 255}
	red := color.RGBA{192, 28, 43, 255}
	blue := color.RGBA{36, 91, 194, 255}
	green := color.RGBA{15, 121, 71, 255}
	grid := color.RGBA{216, 223, 230, 255}
	left, right, top, bottom := 165, 1320, 123, 355
	h0, h1 := s.Points[0].HeightM, s.Points[len(s.Points)-1].HeightM
	if h1 <= h0 {
		h0 -= 0.5
		h1 += 0.5
	}
	vmax := 0.0
	for _, pt := range s.Points {
		vmax = math.Max(vmax, math.Max(pt.Norm, pt.Design))
		if s.Wind.Enabled {
			vmax = math.Max(vmax, math.Max(pt.WindDesign, pt.WindLeading))
		}
	}
	if vmax <= 0 {
		vmax = 1
	}
	vmax = iceNiceMax(vmax)
	X := func(x float64) int { return left + int(float64(right-left)*x/vmax) }
	Y := func(h float64) int { return bottom - int(float64(bottom-top)*(h-h0)/(h1-h0)) }
	f.text(24, 9, 4, "Гололёд и ветер. Распределение по высоте", black)
	f.line(25, 92, 60, 92, black)
	f.text(70, 70, 3, "Гололёд норм.", black)
	f.line(346, 92, 381, 92, red)
	f.text(391, 70, 3, "Гололёд расч.", red)
	if s.Wind.Enabled {
		f.line(682, 92, 717, 92, blue)
		f.text(726, 70, 3, "Ветер + лёд", blue)
		f.line(1013, 92, 1048, 92, green)
		f.text(1057, 70, 3, "Ветер ведущий", green)
	}
	for j := 0; j <= 5; j++ {
		x := left + (right-left)*j/5
		f.line(x, top, x, bottom, grid)
		a := decimalInput(vmax*float64(j)/5, 0)
		f.text(x-snowCaptionWidth(a, 3)/2, 370, 3, a, black)
	}
	for j := 0; j <= 4; j++ {
		h := h0 + (h1-h0)*float64(j)/4
		y := Y(h)
		f.line(left, y, right, y, grid)
		a := decimalInput(h, 1)
		f.text(left-18-snowCaptionWidth(a, 3), y-17, 3, a, black)
	}
	f.line(left, top, left, bottom, black)
	f.line(left, bottom, right, bottom, black)
	stroke := func(x1, y1, x2, y2 int, c color.RGBA) {
		for d := -1; d <= 1; d++ {
			f.line(x1+d, y1, x2+d, y2, c)
		}
	}
	for i := 1; i < len(s.Points); i++ {
		a, b := s.Points[i-1], s.Points[i]
		stroke(X(a.Norm), Y(a.HeightM), X(b.Norm), Y(b.HeightM), black)
		stroke(X(a.Design), Y(a.HeightM), X(b.Design), Y(b.HeightM), red)
		if s.Wind.Enabled {
			stroke(X(a.WindDesign), Y(a.HeightM), X(b.WindDesign), Y(b.HeightM), blue)
			stroke(X(a.WindLeading), Y(a.HeightM), X(b.WindLeading), Y(b.HeightM), green)
		}
	}
	f.text(20, 129, 3, "h, м", black)
	f.text(750, 424, 3, "Интенсивность нагрузки, "+s.NormUnit(), black)

	return f.png()
}

func iceWordAppend(s IceLoadCalc, imageCount *int, rel *strings.Builder, entries map[string]string) (string, error) {
	if len(s.Points) == 0 {
		var e error
		s, e = IceCompute(s)
		if e != nil {
			return "", e
		}
	}
	var b strings.Builder
	const sp = "СП 20.13330.2016"
	b.WriteString(snowWordHeading("Гололёдная нагрузка и совместное действие с ветром"))
	b.WriteString(snowWordParagraph("Нормативное основание: " + sp + ", разделы 11, 12; сочетания по пунктам 6.4 и 12.3."))
	inputs := [][]string{
		{"Гололёдный район", s.Region, sp + ", таблица 12.1; карта 3 приложения Е"},
		{"Тип элемента", s.Description(), sp + ", пункт 12.2"},
		{"Исходная нормативная толщина стенки гололёда b, мм", decimalInput(s.BaseMM, 2), sp + ", таблица 12.1 / данные наблюдений"},
		{"Высота элемента над землёй h, м", decimalInput(s.HeightStart, 1) + "…" + decimalInput(s.HeightEnd, 1), "Высота над поверхностью земли"},
		{"Плотность льда ρ, кг/м³", decimalInput(s.density()*1000, 0), sp + ", пункт 12.2"},
		{"Коэффициент надёжности для гололёда γf", "1,80", sp + ", пункт 12.5"},
		{"Тип высотной территории", iceHeightZoneName(s.HighZone), sp + ", таблица 12.2"},
	}
	if s.IsRound() {
		inputs = append(inputs, []string{"Диаметр без гололёда d, мм", decimalInput(s.DiameterMM, 2), "Наружный диаметр без гололёда"})
	}
	if s.Shape == "round_wire" || s.Horizontal {
		inputs = append(inputs, []string{"Горизонтальный подвешенный круглый элемент", "Да", sp + ", указание после таблицы 12.4: b на высотах ≥200 м уменьшается на 10%"})
	}
	if s.Shape == "surface_study" {
		inputs = append(inputs, []string{"Коэффициент обледенения поверхности μ₂", decimalInput(s.Mu2, 3), sp + ", пункт 12.2; значение задано пользователем"})
	}
	if s.StudyReference != "" {
		inputs = append(inputs, []string{"Источник специальных данных", s.StudyReference, "Материалы наблюдений/исследований, предоставленные пользователем"})
	}
	if s.Wind.Enabled {
		w0, _ := windForZone(s.Wind.Zone)
		psi := "0,60"
		if s.IsFlexible() {
			psi = "0,25"
		}
		inputs = append(inputs,
			[]string{"Ветровой район и нормативное давление w₀", s.Wind.Zone + " (" + decimalInput(w0, 2) + " кПа)", sp + ", таблица 11.1"},
			[]string{"Тип местности по ветру", iceTerrainDescription(s.Wind.Terrain), sp + ", пункт 11.1.6, таблица 11.2"},
			[]string{"Коэффициент лобового аэродинамического сопротивления Cx", decimalInput(s.Wind.Drag, 3), sp + ", пункт 11.1.7, приложение В"},
			[]string{"Коэффициент лобового аэродинамического сопротивления обледеневшего элемента Cx", decimalInput(func() float64 {
				if s.Wind.DragIced > 0 {
					return s.Wind.DragIced
				}
				return s.Wind.Drag
			}(), 3), sp + ", пункт 11.1.7, приложение В"},
			[]string{"Коэффициент надёжности для ветра γf", "1,40", sp + ", пункт 11.4"},
			[]string{"Коэффициент сочетания ветра при ведущем гололёде ψ", psi, sp + ", пункт 12.3б"},
		)
		if s.Wind.PulseVerified {
			inputs = append(inputs, []string{"Отношение пульсационной и средней составляющих ветра wp/wm", decimalInput(s.Wind.PulseRatio, 3), sp + ", пункт 11.1.8: " + s.Wind.PulseSource})
		}
	}
	b.WriteString(snowWordTable("Исходные данные", []string{"Характеристика", "Значение", "Нормативное основание"}, inputs, []int{3250, 2100, 4150}))
	b.WriteString(snowWordHeading("Принятые обозначения"))
	for _, definition := range [][2]string{
		{"h", "высота расположения элемента над поверхностью земли, м."},
		{"b", "исходная нормативная толщина стенки гололёда, мм; в высотных расчётах используется соответствующее значение из таблицы 12.2 СП 20.13330.2016."},
		{"k", "коэффициент изменения толщины стенки гололёда в зависимости от высоты, безразмерный."},
		{"μ₁", "коэффициент, учитывающий влияние диаметра провода или троса на толщину гололёда, безразмерный."},
		{"μ₂", "коэффициент, учитывающий отношение покрытой гололёдом площади поверхности к полной площади поверхности элемента, безразмерный."},
		{"d", "диаметр круглого элемента до обледенения, мм."},
		{"bэф", "толщина стенки гололёда с учётом высоты и применимых поправок, мм."},
		{"ρ", "плотность гололёда, кг/м³ (в формулах учитывается соответствующее преобразование единиц)."},
		{"g", "ускорение свободного падения, м/с²."},
		{"iн", "нормативная вертикальная интенсивность гололёдной нагрузки, " + s.NormUnit() + "."},
		{"iр", "расчётная вертикальная интенсивность гололёдной нагрузки, " + s.NormUnit() + "."},
		{"γf", "коэффициент надёжности по нагрузке."},
	} {
		b.WriteString(iceWordDefinition(definition[0], definition[1]))
	}
	if s.Wind.Enabled {
		for _, definition := range [][2]string{
			{"w₀", "нормативное ветровое давление для выбранного района, кПа."},
			{"kвет(h)", "коэффициент изменения ветрового давления с высотой, безразмерный."},
			{"Cx", "коэффициент лобового аэродинамического сопротивления элемента, безразмерный."},
			{"ψ", "коэффициент сочетания сопутствующей ветровой нагрузки при ведущем гололёде, безразмерный."},
			{"dл", "расчётный диаметр круглого элемента с гололёдом, мм."},
			{"wн", "нормативная горизонтальная ветровая нагрузка на обледеневший элемент, " + s.NormUnit() + "."},
			{"wр, Б", "расчётная горизонтальная ветровая нагрузка при ведущем гололёде, " + s.NormUnit() + "."},
			{"wр, А", "расчётная горизонтальная ветровая нагрузка при ведущем ветре, без гололёда, " + s.NormUnit() + "."},
		} {
			b.WriteString(iceWordDefinition(definition[0], definition[1]))
		}
	}
	b.WriteString(snowWordParagraph("По " + sp + ", формулы (12.1), (12.2): для круглого элемента i = π·b·k·μ₁·(d + b·k·μ₁)·ρ·g·10⁻³, Н/м; для остальных i′ = b·k·μ₂·ρ·g, Па. Расчётная гололёдная нагрузка равна 1,8 нормативной. Толщина bэф учитывает k и при необходимости μ₁; для подвешенного горизонтального круглого элемента поправка 0,9 к b применяется на высотах 200 м и более."))
	if s.Wind.Enabled {
		b.WriteString(snowWordParagraph("По " + sp + ", пункты 11.1.1–11.1.3 и 12.3: средняя ветровая нагрузка wₘ = w₀·kвет(h)·c. Для круглого элемента используется обдуваемый диаметр dл = d + 2bэф; удельная ветровая нагрузка в Н/м равна wₘ·dл/1000 (при wₘ в Па и dл в мм). Случай А: ветер ведущий, гололёд не учитывается; случай Б: гололёд ведущий, сопутствующий ветер умножается на ψ=0,25 для гибких элементов или 0,6 для остальных. Коэффициент надёжности для ветра 1,4 (п. 11.4)."))
	}
	rows := [][]string{}
	for _, pt := range s.Points {
		rows = append(rows, []string{
			decimalInput(pt.HeightM, 1), decimalInput(pt.EffectiveMM, 2),
			decimalInput(pt.Norm, 3), decimalInput(pt.Design, 3),
		})
	}
	b.WriteString(snowWordTable("Результаты расчёта гололёдной нагрузки по высоте", []string{
		"Высота h, м", "Толщина льда bэф, мм", "Нормативная гололёдная нагрузка, " + s.NormUnit(), "Расчётная гололёдная нагрузка, " + s.NormUnit(),
	}, rows, []int{1600, 2400, 2750, 2750}))
	if s.Wind.Enabled {
		windRows := [][]string{}
		for _, pt := range s.Points {
			windRows = append(windRows, []string{
				decimalInput(pt.HeightM, 1), decimalInput(pt.WindNorm, 3),
				decimalInput(pt.WindDesign, 3), decimalInput(pt.WindLeading, 3),
			})
		}
		b.WriteString(snowWordTable("Горизонтальная ветровая нагрузка по высоте", []string{
			"Высота h, м", "Нормативная ветровая нагрузка, " + s.NormUnit(),
			"Расчётный ветер при гололёде, " + s.NormUnit(), "Расчётный ветер без гололёда, " + s.NormUnit(),
		}, windRows, []int{1700, 2650, 2700, 2450}))
		b.WriteString(snowWordParagraph("Ветровые нагрузки действуют горизонтально, гололёдные — вертикально; их интенсивности не суммируются как однонаправленные силы. Случаи А и Б определены по пункту 12.3 СП 20.13330.2016."))
	}
	pic, e := iceFigure(s)
	if e != nil {
		return "", e
	}
	*imageCount++
	id := *imageCount
	entries[fmt.Sprintf("word/media/ice_%d.png", id)] = string(pic)
	fmt.Fprintf(rel, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/ice_%d.png"/>`, id, id)
	b.WriteString(strings.Replace(snowWordHeading("Высотные эпюры нагрузок"), "<w:pPr>", "<w:pPr><w:pageBreakBefore/>", 1))
	b.WriteString(tempWordImageInline(id, 1400, 470))

	return b.String(), nil
}

func ExportIceWord(s IceLoadCalc, path string) error {
	if len(s.Points) == 0 {
		var e error
		s, e = IceCompute(s)
		if e != nil {
			return e
		}
	}
	return exportTempWord(TempReport{Version: 1, Ice: &s}, path)
}
func ExportIceExcel(s IceLoadCalc, path string) error {
	if len(s.Points) == 0 {
		var e error
		s, e = IceCompute(s)
		if e != nil {
			return e
		}
	}
	return exportTempExcel(TempReport{Version: 1, Ice: &s}, path)
}

func iceTerrainDescription(k string) string {
	switch k {
	case "A":
		return "А — открытые побережья, степи, тундра и другие открытые территории"
	case "B":
		return "В — городские территории, леса и местности с препятствиями выше 10 м"
	case "C":
		return "С — районы плотной городской застройки с зданиями выше 25 м"
	}
	return k
}
func iceWordDefinition(symbol, explanation string) string {
	return `<w:p><w:pPr><w:spacing w:before="35" w:after="115"/></w:pPr><w:r><w:rPr><w:b/><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="20"/></w:rPr><w:t>` + tempXML(symbol) + `</w:t></w:r><w:r><w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="20"/></w:rPr><w:t xml:space="preserve"> — ` + tempXML(explanation) + `</w:t></w:r></w:p>`
}

// Round scale divisions use 1/2/5 * 10^n, not arbitrary output values.
func iceNiceMax(value float64) float64 {
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 5
	}
	minStep := value / 5
	power := math.Pow(10, math.Floor(math.Log10(minStep)))
	for _, multiplier := range []float64{1, 2, 5, 10} {
		step := multiplier * power
		if step*5 >= value*(1+1e-10) {
			return step * 5
		}
	}
	return 50 * power
}
