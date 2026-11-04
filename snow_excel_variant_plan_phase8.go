package main

import (
	"fmt"
	"strings"
)

// A standard XLSX cannot add worksheets on recalculation. For a roof with a
// finite list of angle-dependent arrangements, export the complete catalogue
// in advance, and let each worksheet decide its own applicability from Excel
// source cells. This catalogue is for XLSX only: Word and saved Go load cases
// retain the actual arrangements of the original calculation.
//
// No mu, load or interpolated coefficient is passed through this plan.
type snowExcelVariant struct {
	Case          SnowCase
	Applicability string // Excel formula returning "Да" or "Нет"
	Reserved      bool
}

func snowExcelVariantCondition(s SnowCalc, c SnowCase) string {
	yes := `="Да"`
	// The programme accepts 2..12 bays. Reserve the complete finite catalogue
	// independently of the count at the original Go export. A sheet for bay k
	// becomes active only when k exists in the edited Excel source data.
	if s.Shape == "multi_gable" || s.Shape == "multi_vault" {
		n := "'Исходные данные'!$B$17"
		valid := fmt.Sprintf("AND(%s>=2,%s<=12,%s=INT(%s))", n, n, n, n)
		angle := "'Коэффициенты'!$B$4"
		cond := "TRUE"
		title := c.Title
		if s.Shape == "multi_gable" {
			switch {
			case strings.HasPrefix(title, "2. Неравномерная"):
				cond = fmt.Sprintf("ABS(%s)>=15", angle)
			case strings.HasPrefix(title, "3. Вариант по рисунку"):
				cond = fmt.Sprintf("ABS(%s)>=30", angle)
			case strings.HasPrefix(title, "Частичная нагрузка: один скат"),
				strings.HasPrefix(title, "Частичная нагрузка: два смежных"):
				cond = fmt.Sprintf("ABS(%s)>20", angle)
			case strings.HasPrefix(title, "Частичная нагрузка: пролёт"):
				var bay, side int
				if _, err := fmt.Sscanf(title, "Частичная нагрузка: пролёт %d, скат %d", &bay, &side); err == nil {
					cond = fmt.Sprintf("AND(ABS(%s)>20,%s>=%d)", angle, n, bay)
				}
			}
			if strings.HasPrefix(title, "Частичная нагрузка: два смежных") && strings.HasSuffix(title, "схема 2") {
				cond = fmt.Sprintf("AND(ABS(%s)>20,%s>=3)", angle, n)
			}
		} else {
			switch {
			case strings.HasPrefix(title, "2. Снег в ендовах"):
				cond = "('Исходные данные'!$B$14/'Исходные данные'!$B$11>0.1)"
			case strings.HasPrefix(title, "Частичная нагрузка: смежные своды"):
				var first, second int
				if _, err := fmt.Sscanf(title, "Частичная нагрузка: смежные своды %d и %d", &first, &second); err == nil {
					cond = fmt.Sprintf("AND(%s>=%d,DEGREES(2*ATAN(2*'Исходные данные'!$B$14/'Исходные данные'!$B$11))>20)", n, second)
				}
			}
		}
		return fmt.Sprintf(`=IF(AND(%s,%s),"Да","Нет")`, valid, cond)
	}
	if s.RightRoof != nil && s.RightRoof.Shape != "none" {
		if strings.HasPrefix(c.Title, "Занос у перепада высот:") {
			return `=IF('Перепад Б.8'!$B$25="Да","Да","Нет")`
		}
		name, a := c.Title, "'Коэффициенты'!$B$4"
		if strings.HasPrefix(name, "Правый пролёт: ") {
			name = strings.TrimPrefix(name, "Правый пролёт: ")
			a = "'Коэффициенты'!$B$55"
		} else if strings.HasPrefix(name, "Левый пролёт: ") {
			name = strings.TrimPrefix(name, "Левый пролёт: ")
		} else {
			return yes
		}
		if strings.HasPrefix(name, "2. Неравномерная") || strings.HasPrefix(name, "3. Неравномерная") {
			return fmt.Sprintf(`=IF(AND(%s>=15,%s<=40),"Да","Нет")`, a, a)
		}
		if strings.HasPrefix(name, "Частичное загружение левого ската") || strings.HasPrefix(name, "Частичное загружение правого ската") {
			return fmt.Sprintf(`=IF(ABS(%s)>20,"Да","Нет")`, a)
		}
		if strings.HasPrefix(name, "Частичное загружение половины свода") || strings.HasPrefix(name, "Частичное загружение (зеркальное)") {
			l, f := "'Исходные данные'!$B$11", "'Исходные данные'!$B$14"
			if strings.HasPrefix(c.Title, "Правый пролёт: ") {
				l, f = "'Исходные данные'!$B$41", "'Исходные данные'!$B$45"
			}
			return fmt.Sprintf(`=IF(OR(%s<=0,%s<=0),"Нет",IF(DEGREES(2*ATAN(MIN(2*%s/%s,%s/(2*%s))))>20,"Да","Нет"))`, l, f, f, l, l, f)
		}
		return yes
	}
	a := "'Коэффициенты'!$B$4"
	src := "'Исходные данные'!"
	title := c.Title
	// Sector sheets already present in Go must cease reporting load when the
	// current roof geometry no longer satisfies the 20° condition. Creating
	// new polar sector sheets remains a separate open task (large grid).
	if _, _, sector := snowPartialSectorBounds(c); sector {
		switch s.Shape {
		case "cone":
			return fmt.Sprintf(`=IF(ABS(%s)>20,"Да","Нет")`, a)
		case "dome":
			return fmt.Sprintf(`=IF(AND(%s$B$11>0,2*DEGREES(ATAN(2*%s$B$14/%s$B$11))>20),"Да","Нет")`, src, src, src)
		case "sphere_cone":
			return `=IF(ABS('Геометрия Б.15а'!$B$7)>20,"Да","Нет")`
		}
	}
	if s.Shape == "cone" && strings.HasPrefix(title, "2. Пространственный занос") {
		return fmt.Sprintf(`=IF(AND(%s>7,%s<60),"Да","Нет")`, a, a)
	}
	switch s.Shape {
	case "gable":
		switch {
		case strings.HasPrefix(title, "2. Неравномерная"), strings.HasPrefix(title, "3. Неравномерная"):
			return fmt.Sprintf(`=IF(AND(%s>=15,%s<=40),"Да","Нет")`, a, a)
		case strings.HasPrefix(title, "Частичное загружение левого ската"), strings.HasPrefix(title, "Частичное загружение правого ската"):
			return fmt.Sprintf(`=IF(ABS(%s)>20,"Да","Нет")`, a)
		case strings.Contains(title, "Накопление у конька"):
			return fmt.Sprintf(`=IF(AND(%s>=10,%s<=30,%s$B$35="Да"),"Да","Нет")`, a, a, src)
		}
	case "vault":
		if strings.HasPrefix(title, "Частичное загружение") {
			l, f := src+"$B$11", src+"$B$14"
			// For a circular arch the support tangent is 2 atan(2f/L).
			return fmt.Sprintf(`=IF(OR(%s<=0,%s<=0),"Нет",IF(DEGREES(2*ATAN(MIN(2*%s/%s,%s/(2*%s))))>20,"Да","Нет"))`, l, f, f, l, l, f)
		}
	case "skylight":
		if strings.HasPrefix(title, "Дополнительное местное загружение у") {
			b := "'Коэффициенты'!$B$24"
			return fmt.Sprintf(`=IF(AND(%s>24,ABS(%s)<15,'Исходные данные'!$B$15>0,'Исходные данные'!$B$15>='Исходные данные'!$B$3*'Коэффициенты'!$B$21/2),"Да","Нет")`, b, a)
		}
	case "parapet":
		if strings.HasPrefix(title, "2. Местная нагрузка") {
			return fmt.Sprintf(`=IF(%s$B$15>%s$B$3/2,"Да","Нет")`, src, src)
		}
	}
	return yes
}

func snowExcelVariants(s SnowCalc) []snowExcelVariant {
	out := make([]snowExcelVariant, 0, len(s.Cases)+8)
	for _, c := range s.Cases {
		out = append(out, snowExcelVariant{Case: c, Applicability: snowExcelVariantCondition(s, c)})
	}
	// Only reserve variants with independently implemented coefficient formulas.
	// Do not fabricate special schemes merely because their name is known.
	reserve := func(title, source string) {
		for _, v := range out {
			if v.Case.Title == title {
				return
			}
		}
		c := SnowCase{Title: title, Source: snowNormSource + ", " + source}
		out = append(out, snowExcelVariant{Case: c, Applicability: snowExcelVariantCondition(s, c), Reserved: true})
	}
	// Canonical title: changing n changes the type of the B.5 0.6/1.4
	// distribution, not its identity as an independent load arrangement.
	if s.Shape == "multi_gable" {
		for i := range out {
			if strings.HasPrefix(out[i].Case.Title, "2. Неравномерная") {
				out[i].Case.Title = "2. Неравномерная по пролётам"
			}
		}
	}
	if s.Shape == "multi_gable" {
		reserve("2. Неравномерная по пролётам", "Б.5, рисунок Б.8")
		reserve("3. Вариант по рисунку Б.9", "Б.5, рисунок Б.9")
		for scheme := 1; scheme <= 4; scheme++ {
			reserve(fmt.Sprintf("Частичная нагрузка: один скат в каждом пролёте, схема %d", scheme), "п. 10.4, Б.5")
		}
		for scheme := 1; scheme <= 2; scheme++ {
			reserve(fmt.Sprintf("Частичная нагрузка: два смежных ската через пролёт, схема %d", scheme), "п. 10.4, Б.5")
		}
		for bay := 1; bay <= 12; bay++ {
			for side := 1; side <= 2; side++ {
				reserve(fmt.Sprintf("Частичная нагрузка: пролёт %d, скат %d", bay, side), "п. 10.4, Б.5")
			}
		}
		reserve("Проверочная неравномерная по пролётам", "Б.5")
		return out
	}
	if s.Shape == "multi_vault" {
		reserve("2. Снег в ендовах", "Б.5, рисунок Б.9")
		for bay := 1; bay < 12; bay++ {
			reserve(fmt.Sprintf("Частичная нагрузка: смежные своды %d и %d", bay, bay+1), "п. 10.4, Б.6")
		}
		return out
	}
	if s.Shape == "cone" || s.Shape == "dome" || s.Shape == "sphere_cone" {
		for _, sector := range [][2]int{{0, 180}, {180, 360}, {0, 90}, {90, 180}, {180, 270}, {270, 360}} {
			reserve(fmt.Sprintf("Частичное загружение: сектор %d–%d°", sector[0], sector[1]), "п. 10.4, Б.10–Б.12")
		}
		return out
	}
	if s.RightRoof != nil && s.RightRoof.Shape != "none" {
		addPart := func(prefix, shape string) {
			switch shape {
			case "gable":
				reserve(prefix+"2. Неравномерная", "Б.1, рисунок Б.1, вариант 2")
				reserve(prefix+"3. Неравномерная (зеркальная)", "Б.1, рисунок Б.1, вариант 2")
				reserve(prefix+"Частичное загружение левого ската", "пункт 10.4, Б.1")
				reserve(prefix+"Частичное загружение правого ската", "пункт 10.4, Б.1")
			case "vault":
				reserve(prefix+"Частичное загружение половины свода", "пункт 10.4, Б.2")
				reserve(prefix+"Частичное загружение (зеркальное)", "пункт 10.4, Б.2")
			}
		}
		addPart("Левый пролёт: ", s.Shape)
		addPart("Правый пролёт: ", s.RightRoof.Shape)
		// A change of neighbouring roof elevations can create a step even
		// when the exported Go case catalogue had no such load arrangement.
		// Its B.8 worksheet is already computed independently in Excel.
		reserve("Занос у перепада высот: 2. У высокой стены / перепада", "Б.8, рисунок Б.11")
		return out
	}
	switch s.Shape {
	case "gable":
		reserve("2. Неравномерная", "Б.1, рисунок Б.1, вариант 2")
		reserve("3. Неравномерная (зеркальная)", "Б.1, рисунок Б.1, вариант 2")
		reserve("Частичное загружение левого ската", "пункт 10.4, Б.1")
		reserve("Частичное загружение правого ската", "пункт 10.4, Б.1")
		reserve("Вариант 3 Б.1. Накопление у конька", "Б.1, рисунок Б.1, вариант 3")
	case "vault":
		reserve("Частичное загружение половины свода", "пункт 10.4, Б.2")
		reserve("Частичное загружение (зеркальное)", "пункт 10.4, Б.2")
	case "skylight":
		reserve("Дополнительное местное загружение у левого ската фонаря", "Б.3, примечание 3; Б.8")
		reserve("Дополнительное местное загружение у правого ската фонаря", "Б.3, примечание 3; Б.8")
	case "parapet":
		reserve("2. Местная нагрузка у парапета", "Б.13, рисунок Б.16")
	}
	return out
}

// The catalogue is deliberately separate from the Go case catalogue. No ready
// Go value is present; the status and maximum loads link to each live sheet.
func snowExcelVariantCatalogue(plan []snowExcelVariant, suffix string, samples int) string {
	var b strings.Builder
	add := func(row int, cells ...string) { b.WriteString(tempExcelRow(row, cells...)) }
	add(1, "На этом листе перечислены предусмотренные варианты снегового загружения. Состояние «Да/Нет» определяется формулами из изменяемых исходных данных; значения нагрузок берутся только с формульных листов. Отдельные варианты не суммируются.")
	add(2, "Вариант", "Нормативный источник", "Расчётный лист", "Применяется", "μmax", "Sн,max, кПа", "Sр,max, кПа")
	for i, v := range plan {
		row := i + 3
		name := fmt.Sprintf("Вариант %d%s", i+1, suffix)
		sheet := fmt.Sprintf("'%s'!", name)
		end := samples + 4
		label := v.Case.Title
		if strings.Contains(label, "неравномерная 0,9/1,1") && suffix == "" {
			label = `=IF(AND('Исходные данные'!$B$11>100,'Исходные данные'!$B$12>100),"Обязательная неравномерная 0,9/1,1","Проверочная неравномерная 0,9/1,1")`
		}
		add(row, label, v.Case.Source, name,
			"="+sheet+"$B$3",
			fmt.Sprintf(`=IF(D%d="Да",MAX(%s$C$5:$C$%d),"")`, row, sheet, end),
			fmt.Sprintf(`=IF(D%d="Да",MAX(%s$D$5:$D$%d),"")`, row, sheet, end),
			fmt.Sprintf(`=IF(D%d="Да",MAX(%s$E$5:$E$%d),"")`, row, sheet, end))
	}
	last := len(plan) + 2
	add(last+2, "Количество применимых вариантов", fmt.Sprintf(`=COUNTIF(D3:D%d,"Да")`, last))
	return b.String()
}
