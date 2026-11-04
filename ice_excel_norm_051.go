package main

import (
	"fmt"
	"strings"
)

// Export the actual tabular source values and formula-based selection/interpolation.
// Input cells in ice source hold the user's choices; Go-calculated k, mu1 and w0
// are never copied into the checking workbook.
func iceExcelNormSheets(entries map[string]string, s IceLoadCalc, suffix string) error {
	input := "'Гололёд - исходные" + suffix + "'!"
	calc := "'Гололёд - расчёт" + suffix + "'!"
	addSheet := func(name string, body string) error {
		_, err := snowExcelSheet(entries, name+suffix, body, false)
		return err
	}
	var par strings.Builder
	ap := func(r int, c ...string) { par.WriteString(tempExcelRow(r, c...)) }
	ap(1, "На этом листе приведены нормативные постоянные и коэффициенты надёжности для расчёта гололёда и ветра при гололёде; они не передаются как рассчитанные результаты ядра.")
	ap(2, "СП 20.13330.2016, разделы 11, 12")
	ap(3, "γf гололёда", "1.8", "Пункт 12.5")
	ap(5, "g — ускорение свободного падения", "9.81", "м/с²")
	ap(40, "γf ветра", "1.4", "Пункт 11.4")
	if err := addSheet("Гололёд - параметры", par.String()); err != nil {
		return err
	}

	var hi strings.Builder
	ah := func(r int, c ...string) {
		if r >= 10 && len(c) > 1 {
			for j := 1; j < len(c); j++ {
				if strings.HasPrefix(c[j], "=") {
					c[j] = fmt.Sprintf("=IF(A%d=\"\",\"\",%s)", r, strings.TrimPrefix(c[j], "="))
				}
			}
		}
		hi.WriteString(tempExcelRow(r, c...))
	}
	ah(1, "На этом листе приведена таблица 12.2 СП 20.13330.2016 и построчное определение высотных значений b для каждой расчётной высоты; особые значения берутся исключительно из введённых инженером исследований.")
	ah(2, "СП 20.13330.2016, таблица 12.2; для северной части Европы, горной местности и V района используются исходные данные исследований")
	ah(3, "Высота, м", "I район, Азия, мм", "Остальные районы, мм", "По исследованиям, мм")
	for i, h := range []int{200, 300, 400} {
		azi := []string{"15", "20", "25"}
		ord := []string{"35", "45", "60"}
		ah(i+4, fmt.Sprint(h), azi[i], ord[i], fmt.Sprintf("=IF(ISNUMBER(%sB%d),%sB%d,\"\")", input, 15+i, input, 15+i))
	}
	ah(8, "ИНТЕРПОЛЯЦИЯ: b200, b1 и b2 из строк таблицы 12.2 для каждой расчётной отметки")
	ah(9, "Высота h, м", "Нижняя строка", "Верхняя строка", "Нижняя H, м", "b1, мм", "b2, мм", "b200, мм", "Обоснование")
	selectedCol := fmt.Sprintf("IF(OR(%sB11=\"Особая климатическая местность\",%sB10=\"V\",ISNUMBER(%sB15),ISNUMBER(%sB16),ISNUMBER(%sB17)),3,IF(%sB11=\"I гололёдный район азиатской части\",1,2))", input, input, input, input, input, input)

	// Missing special-study observations are unknown, not zero millimetres.
	// An INDEX of a blank formula can otherwise be coerced to numeric zero.
	studyNode := func(index string) string {
		return fmt.Sprintf("IF(%s=3,IF(ISNUMBER(INDEX($D$4:$D$6,%s)),INDEX($D$4:$D$6,%s),\"\"),INDEX($B$4:$C$6,%s,%s))", selectedCol, index, index, index, selectedCol)
	}
	for i := 0; i < iceExcelHeightCapacity; i++ {
		r := 10 + i
		c := 5 + i
		ah(r,
			fmt.Sprintf("=%sA%d", calc, c),
			fmt.Sprintf("=IF(A%d<300,1,2)", r),
			fmt.Sprintf("=MIN(3,B%d+1)", r),
			fmt.Sprintf("=INDEX($A$4:$A$6,B%d)", r),
			"="+studyNode(fmt.Sprintf("B%d", r)),
			"="+studyNode(fmt.Sprintf("C%d", r)),
			"="+studyNode("1"),
			"СП 20.13330.2016, таблица 12.2, ближайшие узлы")
	}
	if err := addSheet("Гололёд табл.12.2", hi.String()); err != nil {
		return err
	}

	var k strings.Builder
	ak := func(r int, c ...string) {
		if r >= 14 && len(c) > 1 {
			for j := 1; j < len(c); j++ {
				if strings.HasPrefix(c[j], "=") {
					c[j] = fmt.Sprintf("=IF(A%d=\"\",\"\",%s)", r, strings.TrimPrefix(c[j], "="))
				}
			}
		}
		k.WriteString(tempExcelRow(r, c...))
	}
	ak(1, "На этом листе полностью приведена таблица 12.3 СП 20.13330.2016 и отдельная строка линейной интерполяции коэффициента k для каждой расчётной отметки.")
	ak(2, "СП 20.13330.2016, таблица 12.3: коэффициент k до 100 м")
	ak(3, "Высота h, м", "k")
	for i, h := range []int{5, 10, 20, 30, 50, 70, 100} {
		ak(i+4, fmt.Sprint(h), xnum(.8+float64(i)*.2))
	}
	ak(12, "ИНТЕРПОЛЯЦИЯ: отдельная строка для каждой расчётной высоты")
	ak(13, "h исходная", "Нижняя строка", "Верхняя строка", "h1", "h2", "k1", "k2", "k(h)", "Формула")
	for i := 0; i < iceExcelHeightCapacity; i++ {
		r := 14 + i
		c := 5 + i
		ak(r, fmt.Sprintf("=%sA%d", calc, c),
			fmt.Sprintf("=IF(A%d<=$A$4,1,IF(A%d>=$A$10,6,MATCH(A%d,$A$4:$A$10,1)))", r, r, r),
			fmt.Sprintf("=MIN(7,B%d+1)", r),
			fmt.Sprintf("=INDEX($A$4:$A$10,B%d)", r),
			fmt.Sprintf("=INDEX($A$4:$A$10,C%d)", r),
			fmt.Sprintf("=INDEX($B$4:$B$10,B%d)", r),
			fmt.Sprintf("=INDEX($B$4:$B$10,C%d)", r),
			fmt.Sprintf("=IF(E%d=D%d,F%d,F%d+(A%d-D%d)*(G%d-F%d)/(E%d-D%d))", r, r, r, r, r, r, r, r, r, r),
			"k=k1+(h−h1)·(k2−k1)/(h2−h1)")
	}
	if err := addSheet("Гололёд табл.12.3", k.String()); err != nil {
		return err
	}

	var mu strings.Builder
	am := func(r int, c ...string) { mu.WriteString(tempExcelRow(r, c...)) }
	am(1, "На этом листе полностью приведена таблица 12.4 СП 20.13330.2016 и рассчитана линейная интерполяция μ₁ по введённому диаметру провода или троса.")
	am(2, "СП 20.13330.2016, таблица 12.4: зависимость μ₁(d)")
	am(3, "Диаметр d, мм", "μ₁")
	for i, d := range []int{5, 10, 20, 30, 50, 70} {
		am(i+4, fmt.Sprint(d), xnum(1.1-.1*float64(i)))
	}
	am(10, "ИНТЕРПОЛЯЦИЯ по диаметру d")
	am(11, "d исходный", "Нижняя строка", "Верхняя строка", "d1", "d2", "μ1 узла", "μ₁(d)", "Формула")
	am(12, "="+input+"B6",
		"=IF(A12<=$A$4,1,IF(A12>=$A$9,5,MATCH(A12,$A$4:$A$9,1)))",
		"=MIN(6,B12+1)",
		"=INDEX($A$4:$A$9,B12)", "=INDEX($A$4:$A$9,C12)", "=INDEX($B$4:$B$9,B12)",
		"=IF(E12=D12,F12,F12+(A12-D12)*(INDEX($B$4:$B$9,C12)-F12)/(E12-D12))",
		"μ₁=μ1+(d−d1)·(μ2−μ1)/(d2−d1)")
	if err := addSheet("Гололёд табл.12.4", mu.String()); err != nil {
		return err
	}

	{
		var w0 strings.Builder
		aw := func(r int, c ...string) { w0.WriteString(tempExcelRow(r, c...)) }
		aw(1, "На этом листе приведена полная таблица 11.1 СП 20.13330.2016: нормативное ветровое давление w₀ и выбор по введённому ветровому району.")
		aw(2, "СП 20.13330.2016, таблица 11.1")
		aw(3, "Ветровой район", "w₀, кПа")
		for i, info := range [][]string{{"IA", "0.17"}, {"I", "0.23"}, {"II", "0.30"}, {"III", "0.38"}, {"IV", "0.48"}, {"V", "0.60"}, {"VI", "0.73"}, {"VII", "0.85"}} {
			aw(i+4, info...)
		}
		aw(13, "ВЫБОР: нормативное давление по району")
		aw(14, "Ветровой район", "=SUBSTITUTE(SUBSTITUTE(UPPER(TRIM("+input+"B38)),\"А\",\"A\"),\"1A\",\"IA\")")
		aw(15, "w₀, кПа", "=IF("+input+"B45<>\"Да\",\"\",IFERROR(INDEX($B$4:$B$11,MATCH(B14,$A$4:$A$11,0)),\"\"))")
		if err := addSheet("Ветер табл.11.1", w0.String()); err != nil {
			return err
		}

		var wind strings.Builder
		addW := func(r int, c ...string) {
			if r >= 18 && len(c) > 1 {
				for j := 1; j < len(c); j++ {
					if strings.HasPrefix(c[j], "=") {
						c[j] = fmt.Sprintf("=IF(A%d=\"\",\"\",%s)", r, strings.TrimPrefix(c[j], "="))
					}
				}
			}
			wind.WriteString(tempExcelRow(r, c...))
		}
		addW(1, "На этом листе полностью приведена таблица 11.2 СП 20.13330.2016 для трёх типов местности и построчно определён коэффициент k(z) для каждой расчётной высоты при гололёде.")
		addW(2, "СП 20.13330.2016, таблица 11.2: линейная интерполяция")
		addW(3, "h, м", "А", "В", "С")
		hs := []int{5, 10, 20, 40, 60, 80, 100, 150, 200, 250, 300}
		ka := []float64{.75, 1, 1.25, 1.5, 1.7, 1.85, 2, 2.25, 2.45, 2.65, 2.75}
		kb := []float64{.5, .65, .85, 1.1, 1.3, 1.45, 1.6, 1.9, 2.1, 2.3, 2.5}
		kc := []float64{.4, .4, .55, .8, 1, 1.15, 1.25, 1.55, 1.8, 2, 2.2}
		for i, h := range hs {
			addW(i+4, fmt.Sprint(h), xnum(ka[i]), xnum(kb[i]), xnum(kc[i]))
		}
		addW(16, "ИНТЕРПОЛЯЦИЯ: каждая высота — отдельная строка")
		addW(17, "Высота, м", "Нижняя строка", "Верхняя строка", "h1", "h2", "k1", "k2", "k(h)", "Формула")
		col := fmt.Sprintf("IF(OR(%sB44=\"A\",%sB44=\"А\"),1,IF(OR(%sB44=\"B\",%sB44=\"В\"),2,3))", input, input, input, input)
		for i := 0; i < iceExcelHeightCapacity; i++ {
			r := 18 + i
			c := 5 + i
			addW(r, fmt.Sprintf("=%sA%d", calc, c),
				fmt.Sprintf("=IF(A%d<=$A$4,1,IF(A%d>=$A$14,10,MATCH(A%d,$A$4:$A$14,1)))", r, r, r),
				fmt.Sprintf("=MIN(11,B%d+1)", r),
				fmt.Sprintf("=INDEX($A$4:$A$14,B%d)", r), fmt.Sprintf("=INDEX($A$4:$A$14,C%d)", r),
				fmt.Sprintf("=INDEX($B$4:$D$14,B%d,%s)", r, col), fmt.Sprintf("=INDEX($B$4:$D$14,C%d,%s)", r, col),
				fmt.Sprintf("=IF(E%d=D%d,F%d,F%d+(A%d-D%d)*(G%d-F%d)/(E%d-D%d))", r, r, r, r, r, r, r, r, r, r),
				"k=k1+(h−h1)·(k2−k1)/(h2−h1)")
		}
		if err := addSheet("Ветер табл.11.2", wind.String()); err != nil {
			return err
		}
	}
	return nil
}
