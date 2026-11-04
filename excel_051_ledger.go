package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// The general report uses independent source/working sheets. Its legacy
// weight snapshot retains accepted normative loads, not the original layers;
// it cannot reconstruct thicknesses that were not persisted in TempReport.
func excel051LedgerSheets(entries map[string]string, r TempReport, regular, transport []TempSelection) error {
	if groups := weightReportBlocks(r); len(groups) > 0 {
		var input, work strings.Builder
		input.WriteString(tempExcelInputRow(1, "На этом листе приведены исходные данные собственного веса. Для слоёв толщина и плотность берутся из сохранённого расчёта. Для непосредственно введённого gн указывается этот ввод."))
		input.WriteString(tempExcelInputRow(2, "Конструкция", "Слой", "Способ ввода", "Толщина, мм", "Плотность, кг/м³", "Введено gн, кгс/м²", "γf"))
		work.WriteString(tempExcelRow(1, "На этом листе нормативная нагрузка собственного веса вычисляется по толщине и плотности либо по непосредственно введённому значению, после чего умножается на γf."))
		work.WriteString(tempExcelRow(2, "Конструкция", "Слой", "Нормативная, кПа", "γf", "Расчётная, кПа", "Формула"))
		row := 3
		for gi, group := range groups {
			var source []WeightRow
			if gi < len(r.WeightInputs) && r.WeightInputs[gi].Title == group.Title {
				source = r.WeightInputs[gi].Rows
			}
			for li, line := range group.Rows {
				kind := "Прямой ввод"
				thick, dens, direct := "", "", xnum(kpaToKgfM2(line.Norm))
				if li < len(source) && source[li].Name == line.Name {
					raw := source[li]
					if raw.Kind == "layer" {
						kind = "Слой"
						thick, dens, direct = xnum(raw.Thickness), xnum(raw.Density), ""
					} else {
						direct = xnum(raw.GNorm)
					}
				}
				// For old report snapshots without geometry, the previously
				// accepted gн is the only known input; never invent thickness.
				input.WriteString(tempExcelInputRow(row, group.Title, line.Name, kind, thick, dens, direct, xnum(line.Gamma)))
				work.WriteString(tempExcelRow(row,
					fmt.Sprintf("='Собственный вес - исходные'!A%d", row),
					fmt.Sprintf("='Собственный вес - исходные'!B%d", row),
					fmt.Sprintf("=IF('Собственный вес - исходные'!C%d=\"Слой\",'Собственный вес - исходные'!D%d*'Собственный вес - исходные'!E%d/1000,'Собственный вес - исходные'!F%d)*9.80665/1000", row, row, row, row),
					fmt.Sprintf("='Собственный вес - исходные'!G%d", row),
					fmt.Sprintf("=C%d*D%d", row, row),
					"gн=t·ρ·9,80665/10⁶; gр=gн·γf"))
				row++
			}
		}
		if _, err := snowExcelSheet(entries, "Собственный вес - исходные", input.String(), false); err != nil {
			return err
		}
		if _, err := snowExcelSheet(entries, "Собственный вес - расчёт", work.String(), false); err != nil {
			return err
		}
	}
	if len(regular) > 0 {
		var input, work strings.Builder
		input.WriteString(tempExcelInputRow(1, "На этом листе приведены исходные параметры временных нагрузок: нормативное значение, γf, грузовая площадь и количество перекрытий. При изменении данных расчётные листы пересчитываются."))
		input.WriteString(tempExcelInputRow(2, "Название", "qн, кПа/кН", "γf", "A, м²", "Число перекрытий", "Класс снижения", "Схема", "Ед. изм.", "СП"))
		work.WriteString(tempExcelRow(1, "На этом листе рассчитаны временные нагрузки по СП 20.13330.2016 с формульным определением коэффициентов снижения по площади и количеству перекрытий."))
		work.WriteString(tempExcelRow(2, "Название", "qн", "φ", "qн,сниж", "γf", "qр,сниж", "qн в кгс", "qр в кгс", "Пояснение"))
		for i, sel := range regular {
			row := i + 3
			name := sel.Title
			if name == "" {
				name = fmt.Sprintf("Временная нагрузка %d", i+1)
			}
			area, floors, class, mode := excel051ReductionSource(sel)
			ns := xnum(tempNumber(sel, "normative"))
			// A previously computed reduced load is not a valid input here.
			// If the original normative number was never saved, leave it blank.
			if _, exists := sel.Params["normative"]; !exists {
				ns = ""
			}
			if is, ok := sel.Params["is_defined"].(bool); ok && !is {
				ns = ""
			}
			unit := tempNumberText(sel.Params["unit"])
			input.WriteString(tempExcelInputRow(row, name, ns, xnum(tempNumber(sel, "gamma_f")), xnum(area), fmt.Sprint(floors), fmt.Sprint(class), mode, unit, tempReadableSource(sel.Source)))
			src := "'Временные - исходные'!"
			a := fmt.Sprintf("%sD%d", src, row)
			cl := fmt.Sprintf("%sF%d", src, row)
			mn := fmt.Sprintf("IF(%s=1,0.4,0.5)", cl)
			base := fmt.Sprintf("IF(%s=1,9,36)", cl)
			// phi1/phi2 by SP20; below A1/A2 the coefficient is unity.
			one := fmt.Sprintf("IF(%s<=%s,1,%s+(1-%s)*SQRT(%s/%s))", a, base, mn, mn, base, a)
			modeCell := fmt.Sprintf("%sG%d", src, row)
			f := fmt.Sprintf("IF(OR(%s=\"Полное\",%s=0),1,IF(%s=\"Несколько перекрытий\",%s+(%s-%s)/SQRT(%sE%d),%s))", modeCell, cl, modeCell, mn, one, mn, src, row, one)
			factor := "(1/0.00980665)" // Unit-sensitive caption is retained on source sheet.
			work.WriteString(tempExcelRow(row, fmt.Sprintf("=%sA%d", src, row), fmt.Sprintf("=%sB%d", src, row), "="+f,
				fmt.Sprintf("=IF(B%d=\"\",\"\",B%d*C%d)", row, row, row), fmt.Sprintf("=%sC%d", src, row), fmt.Sprintf("=IF(D%d=\"\",\"\",D%d*E%d)", row, row, row),
				fmt.Sprintf("=IF(D%d=\"\",\"\",D%d*(%s))", row, row, factor), fmt.Sprintf("=IF(F%d=\"\",\"\",F%d*(%s))", row, row, factor),
				"φ по СП 20.13330.2016, п. 8.2.4 и 8.2.5; класс выбирается по строке таблицы 8.3"))
		}
		if _, err := snowExcelSheet(entries, "Временные - исходные", input.String(), false); err != nil {
			return err
		}
		if _, err := snowExcelSheet(entries, "Временные - расчёт", work.String(), false); err != nil {
			return err
		}
	}
	if len(transport) > 0 {
		var input, calc strings.Builder
		input.WriteString(tempExcelInputRow(1, "На этом листе указаны принятые исходные параметры транспортных нагрузок. Значения K, осевых нагрузок и длины загружения можно изменять; все производные числа пересчитываются на следующем листе."))
		input.WriteString(tempExcelInputRow(2, "Схема", "Тип", "K", "Выбран γf", "L загруж., м", "Передняя ось, кН", "Задняя ось, кН", "Гружёный вагон, кН", "Порожний вагон, кН", "qн справ., кН/м", "Источник"))
		calc.WriteString(tempExcelRow(1, "На этом листе вычисляются нормативные и расчётные параметры транспортных схем по формулам СП 35.13330.2011. Для железных дорог справочная qн не заменяет расчёт по линии влияния."))
		calc.WriteString(tempExcelRow(2, "Схема", "Тип", "Pн первой компоненты, кН", "Вторая компонента: qн, кН/м или Pн задней оси, кН", "γf первой компоненты", "Pр первой компоненты, кН", "qр, кН/м или Pр задней оси, кН", "Условная сумма при длине L, кН", "Нормативная зависимость и единицы"))
		for i, sel := range transport {
			row := i + 3
			kind := transportKind(sel)
			// Zero is a legitimate user-entered number. Apply a fallback
			// only to legacy entries where K was not stored at all.
			k := tempNumber(sel, "K")
			if _, supplied := sel.Params["K"]; !supplied {
				k = 14
			}
			front, rear := tempNumber(sel, "front_axle_kN"), tempNumber(sel, "rear_axle_kN")
			_, frontKnown := sel.Params["front_axle_kN"]
			_, rearKnown := sel.Params["rear_axle_kN"]
			if kind == "AB" && !frontKnown && !rearKnown {
				switch {
				case strings.Contains(sel.Title, "151"):
					front, rear = 490, 990
				case strings.Contains(sel.Title, "74"):
					front, rear = 235, 490
				default:
					front, rear = 167, 333
				}
			}
			gammaInput := ""
			if g := transportGammaAB(sel); g > 0 {
				gammaInput = xnum(g)
			}
			qInput := ""
			if _, defined := sel.Params["distributed_kN_per_m"]; defined {
				qInput = xnum(tempNumber(sel, "distributed_kN_per_m"))
			}
			input.WriteString(tempExcelInputRow(row, sel.Title, kind, xnum(k), gammaInput, xnum(transportLength(sel)), xnum(front), xnum(rear), xnum(tempNumber(sel, "loaded_wagon_kN")), xnum(tempNumber(sel, "empty_wagon_kN")), qInput, tempReadableSource(sel.Source)))
			sh := "'Транспорт - исходные'!"
			col := func(c string) string { return fmt.Sprintf("%s%s%d", sh, c, row) }
			var axle, q, gamma, axled, qd, total, formula string
			switch kind {
			case "AK":
				axle = "=10*" + col("C")
				q = "=" + col("C")
				gamma = "=1.5"
				axled = fmt.Sprintf("=C%d*E%d", row, row)
				qd = fmt.Sprintf("=D%d*1.25", row)
				total = fmt.Sprintf("=2*F%d+G%d*%s", row, row, col("E"))
				formula = "АК: 2·Pр,оси+qр·L; γf=1,50 для оси, 1,25 для распределённой части"
			case "NK":
				axle = "=18*" + col("C")
				q = "=0"
				gamma = "=1.10"
				axled = fmt.Sprintf("=C%d*E%d", row, row)
				qd = "=0"
				total = fmt.Sprintf("=4*F%d", row)
				formula = "NK: Pоси=18K, 4 оси, γf=1,10"
			case "AB":
				axle = "=" + col("F")
				q = "=" + col("G")
				gamma = "=" + col("D")
				axled = fmt.Sprintf(`=IF(E%d="","",C%d*E%d)`, row, row, row)
				qd = fmt.Sprintf(`=IF(E%d="","",D%d*E%d)`, row, row, row)
				total = fmt.Sprintf(`=IF(E%d="","",F%d+G%d)`, row, row, row)
				formula = "АБ: C=передняя ось, D=задняя ось (обе кН), γf по п. 6.23е"
			case "SK":
				axle = "=0"
				q = fmt.Sprintf(`=IF(%s="","",%s)`, col("J"), col("J"))
				gamma = fmt.Sprintf("=IF(%s<=0,\"\",IF(%s<50,1.30-0.15*%s/50,IF(%s<150,1.15-0.05*(%s-50)/100,1.10)))", col("E"), col("E"), col("E"), col("E"), col("E"))
				axled = "=0"
				qd = fmt.Sprintf("=IF(E%d=\"\",\"\",D%d*E%d)", row, row, row)
				total = ""
				formula = "СК: коэффициент γf(L); q — только исходная справочная интенсивность. Усилие требует линии влияния"
			case "Metro", "Tram":
				axle = "=" + col("H") + "/4"
				q = "=" + col("I") + "/4"
				gamma = fmt.Sprintf("=IF(%s<=0,\"\",MAX(1.10,1.3*(1-%s/1000)))", col("E"), col("E"))
				axled = fmt.Sprintf("=IF(E%d=\"\",\"\",C%d*E%d)", row, row, row)
				qd = fmt.Sprintf("=IF(E%d=\"\",\"\",D%d*E%d)", row, row, row)
				total = fmt.Sprintf("=IF(E%d=\"\",\"\",4*F%d)", row, row)
				formula = "Метро/трамвай: Pос= Pвагона/4; γf=max(1,10;1,30(1−L/1000))"
			default:
				// An unsupported scheme has no valid output, not a zero load.
				axle, q, gamma, axled, qd, total = "", "", "", "", "", ""
				formula = "Расчётная схема не реализована в Excel"
			}
			calc.WriteString(tempExcelRow(row, "="+col("A"), "="+col("B"), axle, q, gamma, axled, qd, total, formula))
		}
		if _, err := snowExcelSheet(entries, "Транспорт - исходные", input.String(), false); err != nil {
			return err
		}
		if _, err := snowExcelSheet(entries, "Транспорт - расчёт", calc.String(), false); err != nil {
			return err
		}
	}
	return nil
}

var excel051AreaRX = regexp.MustCompile(`A=([0-9]+(?:[.,][0-9]+)?)`)
var excel051FloorRX = regexp.MustCompile(`^(\d+) перекрытий`)

func excel051ReductionSource(s TempSelection) (area float64, floors int, class int, mode string) {
	floors, mode = 1, "Полное"
	// New calculations store typed inputs; the display string is NOT the
	// authoritative source for an engineering calculation.
	if _, ok := s.Params["reduction_mode"]; ok {
		area = tempNumber(s, "reduction_area_m2")
		floors = int(tempNumber(s, "reduction_floors"))
		class = int(tempNumber(s, "reduction_class"))
		switch int(tempNumber(s, "reduction_mode")) {
		case int(TempOneFloor):
			mode = "Одно перекрытие"
		case int(TempSeveralFloors):
			mode = "Несколько перекрытий"
		}
		if floors < 1 {
			floors = 1
		}
		return
	}
	// For old saved blocks recover only the LEGACY geometry. The reduction
	// class is always looked up from the exact documented catalogue row ID.
	scope := tempNumberText(s.Params["calc_scope"])
	if strings.HasPrefix(scope, "Одно перекрытие") {
		mode = "Одно перекрытие"
	}
	if strings.Contains(scope, " перекрытий") {
		mode = "Несколько перекрытий"
	}
	if m := excel051AreaRX.FindStringSubmatch(scope); len(m) > 1 {
		area, _ = strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
	}
	if m := excel051FloorRX.FindStringSubmatch(scope); len(m) > 1 {
		floors, _ = strconv.Atoi(m[1])
	}
	if mode != "Полное" {
		// Legacy blocks stored the catalogue SourceID rather than the
		// unique row ID. Infer the class only when all exact rows sharing
		// that SourceID agree; otherwise do not guess a reduction class.
		class = excel051LegacyReductionClass(tempNumberText(s.Params["sp20_row"]))
	}
	return
}

// Unlike a StartsWith("T-00012") test, this cannot accidentally assign
// area reductions to unrelated normative positions. Ambiguity -> no guess.
func excel051LegacyReductionClass(sourceID string) int {
	if n := sp20ReductionIDs[sourceID]; n != 0 {
		return n
	}
	class := 0
	for id, candidate := range sp20ReductionIDs {
		if !strings.HasPrefix(id, sourceID+"-") {
			continue
		}
		if class != 0 && class != candidate {
			return 0
		}
		class = candidate
	}
	return class
}
