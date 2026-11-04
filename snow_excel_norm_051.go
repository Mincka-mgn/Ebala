package main

import (
	"fmt"
	"strings"
)

// The SP 20.13330.2016 tables are exported in their entirety rather than as
// Go-selected brackets. Each interpolated number is a live spreadsheet cell.
func snowExcelNormTables(entries map[string]string, input, coef, suffix string) error {
	q := func(name string) string { return "'" + name + "'!" }
	src := q(input)
	cf := q(coef)
	var wind strings.Builder
	addW := func(row int, c ...string) { wind.WriteString(tempExcelRow(row, c...)) }
	addW(1, "На этом листе полностью воспроизведена таблица 10.2 СП 20.13330.2016; ниже таблицы автоматически выбирается kv для введённых температуры января, зимнего ветра и типа местности.")
	addW(2, "СП 20.13330.2016, таблица 10.2: коэффициент kv для категорий местности А и В")
	addW(3, "Температура января", "А: 3–4 м/с", "А: 4–6 м/с", "А: >6 м/с", "В: 3–4 м/с", "В: 4–6 м/с", "В: >6 м/с")
	addW(4, "−15…−5 °C", "1.4", "1.3", "1.3", "1.4", "1.4", "1.3")
	addW(5, "−25…−15 °C", "1.4", "1.3", "1.2", "1.4", "1.4", "1.3")
	addW(6, "T < −25 °C", "1.3", "1.2", "1.2", "1.4", "1.3", "1.2")
	addW(9, "БЛОК ВЫБОРА: каждое значение берётся из исходных данных и таблицы 10.2")
	addW(10, "T января, °C", "="+src+"B8")
	addW(11, "V зимы, м/с", "="+src+"B9")
	addW(12, "Тип местности", "="+src+"B7")
	addW(13, "Номер строки таблицы", "=IF(B10< -25,3,IF(B10< -15,2,1))")
	addW(14, "Номер столбца таблицы", "=IF(B12=\"А\",0,3)+IF(B11>6,3,IF(B11>4,2,1))")
	addW(15, "kv по таблице 10.2", "=INDEX(B4:G6,B13,B14)")
	if _, err := snowExcelSheet(entries, "Таблица 10.2"+suffix, wind.String(), false); err != nil {
		return err
	}

	var k strings.Builder
	addK := func(row int, c ...string) { k.WriteString(tempExcelRow(row, c...)) }
	addK(1, "На этом листе полностью приведена таблица 11.2 СП 20.13330.2016 для местностей А, В и С; под таблицей рассчитана линейная интерполяция k(H) по высоте из исходных данных.")
	addK(2, "СП 20.13330.2016, таблица 11.2; промежуточные значения по линейной интерполяции")
	addK(3, "Высота H, м", "Местность А", "Местность В", "Местность С")
	hs := []float64{5, 10, 20, 40, 60, 80, 100, 150, 200, 250, 300}
	ka := []float64{.75, 1, 1.25, 1.5, 1.7, 1.85, 2, 2.25, 2.45, 2.65, 2.75}
	kb := []float64{.5, .65, .85, 1.1, 1.3, 1.45, 1.6, 1.9, 2.1, 2.3, 2.5}
	kc := []float64{.4, .4, .55, .8, 1, 1.15, 1.25, 1.55, 1.8, 2, 2.2}
	for i, h := range hs {
		addK(i+4, xnum(h), xnum(ka[i]), xnum(kb[i]), xnum(kc[i]))
	}
	addK(16, "ИНТЕРПОЛЯЦИЯ: отдельная строка для значения k при высоте H")
	addK(17, "H исходная, м", "Нижний индекс", "Верхний индекс", "H1, м", "H2, м", "k1", "k2", "k(H)", "Зависимость")
	addK(18, "="+cf+"B7",
		"=IF(A18<=$A$4,1,IF(A18>=$A$14,10,MATCH(A18,$A$4:$A$14,1)))",
		"=MIN(11,B18+1)",
		"=INDEX($A$4:$A$14,B18)", "=INDEX($A$4:$A$14,C18)",
		fmt.Sprintf("=INDEX($B$4:$D$14,B18,IF(%sB7=\"А\",1,IF(%sB7=\"В\",2,3)))", src, src),
		fmt.Sprintf("=INDEX($B$4:$D$14,C18,IF(%sB7=\"А\",1,IF(%sB7=\"В\",2,3)))", src, src),
		"=IF(E18=D18,F18,F18+(A18-D18)*(G18-F18)/(E18-D18))",
		"k = k1 + (H − H1)·(k2 − k1)/(H2 − H1)")
	if _, err := snowExcelSheet(entries, "Таблица 11.2"+suffix, k.String(), false); err != nil {
		return err
	}
	return nil
}
