package main

import (
	"fmt"
	"strings"
)

const iceXSrc = "'Гололёд - исходные'!"

func iceExcelAppend(s IceLoadCalc, add func(...string)) {
	add("ГОЛОЛЁДНАЯ НАГРУЗКА · СП 20.13330.2016, раздел 12")
	add("Район", s.Region, "Элемент", s.Description(), "γf", "="+iceXSrc+"$B$3")
	headers := []string{"h, м", "Нормативная гололёдная нагрузка, " + s.NormUnit(), "Расчётная гололёдная нагрузка, " + s.NormUnit()}
	if s.Wind.Enabled {
		headers = append(headers, "Ветер при гололёде, "+s.NormUnit(), "Ветер ведущий, "+s.NormUnit())
	}
	headers = append(headers, "Основание")
	add(headers...)
	for i, p := range s.Points {
		row := i + 5
		values := []string{xnum(p.HeightM), fmt.Sprintf("='Гололёд - расчёт'!G%d", row), fmt.Sprintf("='Гололёд - расчёт'!H%d", row)}
		if s.Wind.Enabled {
			values = append(values, fmt.Sprintf("='Гололёд - расчёт'!N%d", row), fmt.Sprintf("='Гололёд - расчёт'!O%d", row))
		}
		add(append(values, p.Basis)...)
	}
}
func iceExcelAppendIndexed(s IceLoadCalc, index int, add func(...string)) {
	suffix := fmt.Sprintf(" %d", index)
	add("ГОЛОЛЁДНАЯ НАГРУЗКА · " + s.Description())
	add("Район", s.Region, "Тип", s.Description(), "Основание", "СП 20.13330.2016, раздел 12")
	add("Высота h, м", "Нормативная гололёдная нагрузка, "+s.NormUnit(), "Расчётная гололёдная нагрузка, "+s.NormUnit())
	for i, point := range s.Points {
		row := i + 5
		add(xnum(point.HeightM), fmt.Sprintf("='Гололёд - расчёт%s'!G%d", suffix, row), fmt.Sprintf("='Гололёд - расчёт%s'!H%d", suffix, row))
	}
}
func iceExcelInputData(s IceLoadCalc) string {
	var b strings.Builder
	add := func(n int, v ...string) { b.WriteString(tempExcelRow(n, v...)) }
	add(1, "Параметр (расшифровка)", "Значение / коэффициент", "Единицы", "Нормативное основание")
	add(2, "Исходная толщина стенки гололёда b₀", xnum(s.BaseMM), "мм", "Таблица 12.1 СП 20.13330.2016 / специальные данные: "+s.StudyReference)
	add(3, "Коэффициент надёжности γf", xnum(iceGamma), "", "Пункт 12.5 СП 20.13330.2016")
	add(4, "Плотность льда ρ", xnum(s.density()), "г/см³", "Пункт 12.2 СП 20.13330.2016")
	add(5, "Ускорение свободного падения g", xnum(iceGravity), "м/с²", "Пункт 12.2 СП 20.13330.2016")
	add(6, "Диаметр элемента без гололёда d", xnum(s.DiameterMM), "мм", "Диаметр круглого элемента")
	if s.IsRound() {
		add(7, "Коэффициент влияния диаметра μ₁", `=IF(B6<=A20,B20,IF(B6>=A25,B25,INDEX(B20:B25,MATCH(B6,A20:A25,1))+(B6-INDEX(A20:A25,MATCH(B6,A20:A25,1)))*(INDEX(B20:B25,MATCH(B6,A20:A25,1)+1)-INDEX(B20:B25,MATCH(B6,A20:A25,1)))/(INDEX(A20:A25,MATCH(B6,A20:A25,1)+1)-INDEX(A20:A25,MATCH(B6,A20:A25,1)))))`, "", "Таблица 12.4 СП 20.13330.2016: линейная интерполяция")
	} else {
		add(7, "Коэффициент влияния диаметра μ₁", "1", "", "Для остальных элементов не применяется")
	}
	if s.IsRound() {
		add(8, "Коэффициент обледенения поверхности μ₂", "1", "", "Для круглого элемента не применяется")
	} else if s.Shape == "surface_small" {
		add(8, "Коэффициент обледенения поверхности μ₂", "0.6", "", "Пункт 12.2: малые размеры поперечного сечения")
	} else {
		add(8, "Коэффициент обледенения поверхности μ₂", xnum(s.Mu2), "", "Пункт 12.2 СП 20.13330.2016; значение задано пользователем")
	}
	add(9, "Тип элемента", s.Description(), "", "Выбран пользователем")
	add(10, "Гололёдный район", s.Region, "", "Таблица 12.1, карта 3 СП 20.13330.2016")
	add(11, "Высотный район", iceHeightZoneName(s.HighZone), "", "Проверка применимости таблицы 12.2")
	add(12, "Шаг высот", xnum(s.HeightStep), "м", "Задан пользователем")
	if s.Horizontal || s.Shape == "round_wire" {
		add(13, "Подвешенный горизонтальный круглый элемент", "1", "0/1", "Понижение b на 10% по п. 12.2 (таблица 12.2)")
	} else {
		add(13, "Подвешенный горизонтальный круглый элемент", "0", "0/1", "Пункт 12.2 СП 20.13330.2016")
	}
	add(14, "Источник специальных данных", s.StudyReference, "", "Ручной ввод")
	high, e := s.highTable()
	if e == nil {
		for i, v := range high {
			add(15+i, fmt.Sprintf("b%d", 200+i*100), xnum(v), "мм", "Таблица 12.2 СП 20.13330.2016")
		}
	}
	add(19, "Диаметр провода или троса d, мм", "Коэффициент влияния диаметра μ₁", "Нормативное основание")
	for i, d := range []float64{5, 10, 20, 30, 50, 70} {
		add(20+i, xnum(d), xnum(iceMu1(d)), "Таблица 12.4 СП 20.13330.2016; линейная интерполяция")
	}
	windW0 := 0.0
	if s.Wind.Enabled {
		windW0, _ = windForZone(s.Wind.Zone)
	}
	add(39, "Ветер: w₀", xnum(windW0), "кПа", "СП 20.13330.2016, табл. 11.1")
	add(40, "Ветер: γf", xnum(iceWindGamma), "", "Пункт 11.4 СП 20.13330.2016")
	add(41, "Ветер: c", xnum(s.Wind.Drag), "", "Пункт 11.1.7, прил. В: "+s.Wind.DragSource)
	icedC := s.Wind.Drag
	if s.Wind.DragIced > 0 {
		icedC = s.Wind.DragIced
	}
	add(46, "Ветер: c обледен.", xnum(icedC), "", "Пункт 11.1.7, приложение В")
	psi := .6
	if s.IsFlexible() {
		psi = .25
	}
	add(42, "Ветер: ψ", xnum(psi), "", "Пункт 12.3б СП 20.13330.2016")
	pulse := 0.0
	if s.Wind.PulseVerified {
		pulse = s.Wind.PulseRatio
	}
	add(43, "Ветер: wp/wm", xnum(pulse), "", "Пункт 11.1.8: "+s.Wind.PulseSource)
	add(44, "Ветер: местность", s.Wind.Terrain, "", "Пункт 11.1.6 СП 20.13330.2016")
	add(45, "Ветер: ограничение", iceWindCompleteness(s), "", "Пункт 11.1.2 СП 20.13330.2016")
	if s.Wind.Enabled {
		add(49, "Высота для ветра, м", "k(z), безразмерный", "Нормативное основание")
		for i, h := range []float64{5, 10, 20, 40, 60, 80, 100, 150, 200, 250, 300} {
			k, _ := iceWindK(h, s.Wind.Terrain)
			add(50+i, xnum(h), xnum(k), "Таблица 11.2 СП 20.13330.2016")
		}
	}
	add(29, "Высота для гололёда, м", "k, безразмерный", "Нормативное основание")
	for i, h := range []float64{5, 10, 20, 30, 50, 70, 100} {
		add(30+i, xnum(h), xnum(iceHeightK(h)), "Таблица 12.3 СП 20.13330.2016")
	}
	return b.String()
}
func iceExcelCalcData(s IceLoadCalc) string {
	var b strings.Builder
	add := func(n int, v ...string) { b.WriteString(tempExcelRow(n, v...)) }
	add(1, "РАСЧЁТ ГОЛОЛЁДНОЙ НАГРУЗКИ", "СП 20.13330.2016, раздел 12")
	add(2, "Формула 12.1, круглый", "i=π·b·k·μ1·(d+b·k·μ1)·ρ·g·10^-3, Н/м")
	add(3, "Формула 12.2, остальные", "i'=b·k·μ2·ρ·g, Па")
	add(4, "Высота h, м", "Исходное b, мм", "Коэффициент k", "Влияние диаметра μ₁", "Обледенение μ₂", "Толщина льда bэф, мм", "Гололёд нормативный, "+s.NormUnit(), "Гололёд расчётный, "+s.NormUnit(), "Основание СП", "Ветер k(z)", "Обледеневший диаметр, мм", "Ветер средний, "+s.NormUnit(), "Ветер нормативный, "+s.NormUnit(), "Ветер при гололёде, "+s.NormUnit(), "Ветер ведущий, "+s.NormUnit())
	src := iceXSrc
	for i, p := range s.Points {
		row := i + 5
		h := fmt.Sprintf("A%d", row)
		base := src + "$B$2"
		bf := "=" + base
		kf := "=1"
		if p.HeightM <= 100 {
			hs := []float64{5, 10, 20, 30, 50, 70, 100}
			idx := 0
			for j := 1; j < len(hs); j++ {
				if p.HeightM <= hs[j] {
					idx = j - 1
					break
				}
			}
			first := 30 + idx
			last := first + 1
			kf = fmt.Sprintf("=%s$B$%d+(%s-%s$A$%d)*(%s$B$%d-%s$B$%d)/(%s$A$%d-%s$A$%d)", src, first, h, src, first, src, last, src, first, src, last, src, first)
		} else if p.HeightM < 200 {
			kf = fmt.Sprintf("=MIN(EXP(0.007*%s),%s$B$15/%s$B$2)", h, src, src)
		} else {
			if p.HeightM >= 400 {
				bf = "=" + src + "$B$17"
			} else {
				tabrow := 15
				if p.HeightM >= 300 {
					tabrow = 16
				}
				bf = fmt.Sprintf("=%s$B$%d", src, tabrow)
				kf = fmt.Sprintf("=1+0.01*(%s$B$%d/%s$B$%d-1)*(%s-%d)", src, tabrow+1, src, tabrow, h, 200+(tabrow-15)*100)
			}
			if (s.Horizontal || s.Shape == "round_wire") && s.IsRound() {
				bf = "=(" + strings.TrimPrefix(bf, "=") + ")*0.9"
			}
		}
		mu1 := "=" + src + "$B$7"
		mu2 := "=" + src + "$B$8"
		thick := fmt.Sprintf("=B%d*C%d", row, row)
		norm := fmt.Sprintf("=B%d*C%d*E%d*%s$B$4*%s$B$5", row, row, row, src, src)
		if s.IsRound() {
			thick = fmt.Sprintf("=B%d*C%d*D%d", row, row, row)
			norm = fmt.Sprintf("=PI()*F%d*(%s$B$6+F%d)*%s$B$4*%s$B$5*0.001", row, src, row, src, src)
		}
		design := fmt.Sprintf("=G%d*%s$B$3", row, src)
		rowData := []string{xnum(p.HeightM), bf, kf, mu1, mu2, thick, norm, design, p.Basis}
		if s.Wind.Enabled {
			hs := []float64{5, 10, 20, 40, 60, 80, 100, 150, 200, 250, 300}
			j := 0
			for k := 1; k < len(hs); k++ {
				if p.HeightM <= hs[k] {
					j = k - 1
					break
				}
			}
			first := 50 + j
			last := first + 1
			windK := fmt.Sprintf("=%s$B$%d+(%s-%s$A$%d)*(%s$B$%d-%s$B$%d)/(%s$A$%d-%s$A$%d)", src, first, h, src, first, src, last, src, first, src, last, src, first)
			dIced := ""
			if s.IsRound() {
				dIced = fmt.Sprintf("=%s$B$6+2*F%d", src, row)
			}
			icedFactor := "1"
			bareFactor := "1"
			if s.IsRound() {
				icedFactor = fmt.Sprintf("K%d/1000", row)
				bareFactor = src + "$B$6/1000"
			}
			mean := fmt.Sprintf("=1000*%s$B$39*J%d*%s$B$46*(%s)", src, row, src, icedFactor)
			total := fmt.Sprintf("=L%d*(1+%s$B$43)", row, src)
			combined := fmt.Sprintf("=M%d*%s$B$40*%s$B$42", row, src, src)
			leading := fmt.Sprintf("=1000*%s$B$39*J%d*%s$B$41*(%s)*(1+%s$B$43)*%s$B$40", src, row, src, bareFactor, src, src)
			rowData = append(rowData, windK, dIced, mean, total, combined, leading)
		}
		add(row, rowData...)
	}
	return b.String()
}
func iceExcelV038(entries map[string]string, s IceLoadCalc, blockIndex ...int) error {
	suffix := ""
	if len(blockIndex) > 0 && blockIndex[0] > 0 {
		suffix = fmt.Sprintf(" %d", blockIndex[0])
	}
	rename := func(body string) string {
		if suffix == "" {
			return body
		}
		body = strings.ReplaceAll(body, "'Гололёд - исходные'!", "'Гололёд - исходные"+suffix+"'!")
		body = strings.ReplaceAll(body, "'Гололёд - расчёт'!", "'Гололёд - расчёт"+suffix+"'!")
		return body
	}

	id, e := snowExcelSheet(entries, "Гололёд - исходные"+suffix, rename(iceExcelInputData(s)), false)
	if e != nil {
		return e
	}
	iceExcelColumns(entries, id, []int{51, 23, 25, 68})
	id, e = snowExcelSheet(entries, "Гололёд - расчёт"+suffix, rename(iceExcelCalcData(s)), false)
	if e != nil {
		return e
	}
	iceExcelColumns(entries, id, []int{18, 20, 17, 21, 19, 25, 27, 27, 62, 17, 26, 26, 26, 31, 31})
	return iceExcelChart(entries, s, blockIndex...)
}

// Give every checkable parameter its own readable column. The generic snow
// spreadsheet layout only defines seven columns, but ice+wind needs fifteen.
func iceExcelColumns(entries map[string]string, id int, widths []int) {
	key := fmt.Sprintf("xl/worksheets/snow%d.xml", id)
	body := entries[key]
	start := strings.Index(body, "<cols>")
	end := strings.Index(body, "</cols>")
	if start < 0 || end < start {
		return
	}
	var cols strings.Builder
	cols.WriteString("<cols>")
	for i, w := range widths {
		fmt.Fprintf(&cols, `<col min="%d" max="%d" width="%d" customWidth="1"/>`, i+1, i+1, w)
	}
	cols.WriteString("</cols>")
	entries[key] = body[:start] + cols.String() + body[end+len("</cols>"):]
}
