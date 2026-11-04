package main

import (
	"fmt"
	"sort"
	"strings"
)

// A crane set contains the engineer's accepted passport wheel forces and positions.
// These values are NOT results of a placement optimization. The worksheet does
// not import Go-calculated reactions, total forces or influence ordinates.
// The latter require the independent beam calculation (the separate placement
// workbook) and are deliberately not guessed in this takeoff.
type excelCraneSource struct {
	Group, Crane string
	Wheel        TempWheel
	WheelNumber  int
	CraneNumber  int
}

func excelPhase3CraneSources(r TempReport, selections []TempSelection) []excelCraneSource {
	var out []excelCraneSource
	appendCrane := func(group string, number int, c TempSelection) {
		for k, wheel := range c.Wheels {
			out = append(out, excelCraneSource{Group: group, Crane: c.Title, Wheel: wheel, WheelNumber: k + 1, CraneNumber: number})
		}
	}
	for i, c := range selections {
		appendCrane("Самостоятельные краны", i+1, c)
	}
	for _, group := range r.CraneGroups {
		for i, c := range group.Cranes {
			appendCrane(group.Title, i+1, c)
		}
	}
	return out
}

// The general export also needs the original accepted crane passport inputs.
// Wheels alone do not describe the selected crane. These rows are source
// values, not calculated reactions or results of beam optimization.
func excelPhase3CranePassport(entries map[string]string, r TempReport, selections []TempSelection) error {
	var out strings.Builder
	out.WriteString(tempExcelInputRow(1, "На этом листе приведены принятые паспортные параметры мостовых кранов. Колёсные нагрузки и координаты находятся на листе «Краны - исходные»; результаты расчёта сюда не копируются."))
	out.WriteString(tempExcelInputRow(2, "Группа", "Кран", "Параметр", "Принятое значение", "Источник"))
	row := 3
	appendCrane := func(group string, c TempSelection) {
		// A named crane without optional passport keys still has an accepted
		// source record; the navigation sheet must never point to a missing tab.
		out.WriteString(tempExcelInputRow(row, group, c.Title, "Наименование крана", c.Title, c.Source))
		row++
		keys := make([]string, 0, len(c.Params))
		for key := range c.Params {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value := tempNumberText(c.Params[key])
			if v, ok := c.Params[key].(bool); ok {
				if v {
					value = "Да"
				} else {
					value = "Нет"
				}
			}
			description := tempParamNames[key]
			if description == "" {
				description = key
			}
			out.WriteString(tempExcelInputRow(row, group, c.Title, description, value, c.Source))
			row++
		}
	}
	for _, c := range selections {
		appendCrane("Самостоятельные краны", c)
	}
	for _, group := range r.CraneGroups {
		for _, c := range group.Cranes {
			appendCrane(group.Title, c)
		}
	}
	if row == 3 {
		return nil
	}
	_, err := snowExcelSheet(entries, "Краны - паспорт", out.String(), false)
	return err
}

// Excel rows begin at 3. A single point of truth for references from the
// general ledger to the independent crane source / computation sheets.
func excelPhase3CraneSheets(entries map[string]string, r TempReport, selections []TempSelection) error {
	sources := excelPhase3CraneSources(r, selections)
	if len(sources) == 0 {
		return nil
	}
	var input, calc strings.Builder
	input.WriteString(tempExcelInputRow(1, "На этом листе перечислены исходные колёсные нагрузки и координаты, принятые инженером. Для каждого колеса свои значения P и x; отсутствующая координата остаётся пустой."))
	input.WriteString(tempExcelInputRow(2, "Группа", "Кран", "№ колеса", "Положение x, мм", "Нагрузка Pн, кН", "Координата задана", "№ крана"))
	calc.WriteString(tempExcelRow(1, "На этом листе каждый вес колеса пересчитывается из принятых нагрузок в килоньютонах, а расстояния между соседними колёсами находятся вычитанием их координат. Опасные положения балки не являются входом этого расчёта."))
	calc.WriteString(tempExcelRow(2, "Группа", "Кран", "№ колеса", "x, мм", "Pн, кН", "Pн, тс", "Шаг до предыдущего, мм", "Формула"))
	for i, w := range sources {
		row := 3 + i
		x := ""
		if w.Wheel.XKnown {
			x = xnum(w.Wheel.XMM)
		}
		known := "Нет"
		if w.Wheel.XKnown {
			known = "Да"
		}
		input.WriteString(tempExcelInputRow(row, w.Group, w.Crane, fmt.Sprint(w.WheelNumber), x, xnum(w.Wheel.PKN), known, fmt.Sprint(w.CraneNumber)))
		refs := fmt.Sprintf("'Краны - исходные'!")
		step := `=""`
		if w.WheelNumber > 1 && i > 0 {
			step = fmt.Sprintf("=IF(AND(%sF%d=\"Да\",%sF%d=\"Да\",%sB%d=%sB%d,%sA%d=%sA%d),D%d-D%d,\"\")",
				refs, row, refs, row-1, refs, row, refs, row-1, refs, row, refs, row-1, row, row-1)
		}
		calc.WriteString(tempExcelRow(row,
			fmt.Sprintf("=%sA%d", refs, row),
			fmt.Sprintf("=%sB%d", refs, row),
			fmt.Sprintf("=%sC%d", refs, row),
			fmt.Sprintf("=IF(%sF%d=\"Да\",%sD%d,\"\")", refs, row, refs, row),
			fmt.Sprintf("=%sE%d", refs, row),
			fmt.Sprintf("=E%d/9.80665", row),
			step,
			"Pтс=PкН/9,80665; aᵢ=xᵢ−xᵢ₋₁"))
	}
	if err := excelPhase3CranePassport(entries, r, selections); err != nil {
		return err
	}
	if _, err := snowExcelSheet(entries, "Краны - исходные", input.String(), false); err != nil {
		return err
	}
	_, err := snowExcelSheet(entries, "Краны - расчёт", calc.String(), false)
	return err
}
