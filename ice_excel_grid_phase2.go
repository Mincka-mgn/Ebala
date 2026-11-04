package main

import (
	"fmt"
	"strings"
)

// iceExcelHeightGrid contains only spreadsheet formulas and literal SP control
// heights. It does not accept (or copy) the heights calculated by Go.
// The output capacity covers the kernel's limit of 300 intermediate step
// levels, both ends and ten mandatory table levels. Duplicate heights
// are suppressed without changing Excel row addresses.
func iceExcelHeightGrid(input string) string {
	var b strings.Builder
	add := func(row int, values ...string) { b.WriteString(tempExcelRow(row, values...)) }
	src := "'" + input + "'!"
	add(1, "На этом листе высоты гололёдного расчёта формируются в Excel из нижней/верхней отметок, шага и нормативных контрольных отметок 5–400 м; готовые узлы Go не используются.")
	add(2, "А — точки выбранного шага; В — отметки таблиц СП; С — объединённый ряд по возрастанию; D — уникальные рабочие высоты. Пустые ячейки не создают нагрузку.")
	add(3, "Высота по шагу, м", "Контрольная отметка СП, м", "Объединённый ряд, м", "Высота для расчёта, м")
	marks := []int{5, 10, 20, 30, 50, 70, 100, 200, 300, 400}
	for i := 0; i < iceExcelHeightCapacity; i++ {
		row := 4 + i
		stepHeight := ""
		switch {
		case i == 0:
			stepHeight = fmt.Sprintf("=IF(AND(%s$B$12>0,%s$B$19>=%s$B$18),%s$B$18,\"\")", src, src, src, src)
		case i <= 300:
			stepHeight = fmt.Sprintf("=IF(AND(%s$B$12>0,%s$B$18+%d*%s$B$12<%s$B$19),%s$B$18+%d*%s$B$12,\"\")", src, src, i, src, src, src, i, src)
		case i == 301:
			stepHeight = fmt.Sprintf("=IF(%s$B$19>=%s$B$18,%s$B$19,\"\")", src, src, src)
		}
		marked := ""
		if i < len(marks) {
			marked = fmt.Sprintf("=IF(AND(%d>%s$B$18,%d<%s$B$19),%d,\"\")", marks[i], src, marks[i], src, marks[i])
		}
		ordered := fmt.Sprintf("=IF(OR(%s$B$12<=0,%s$B$19<%s$B$18),\"\",IFERROR(SMALL($A$4:$B$305,ROWS($C$4:C%d)),\"\"))", src, src, src, row)
		unique := fmt.Sprintf("=IF(C%d=\"\",\"\",C%d)", row, row)
		if i != 0 {
			unique = fmt.Sprintf("=IF(OR(C%d=\"\",C%d=C%d),\"\",C%d)", row, row, row-1, row)
		}
		add(row, stepHeight, marked, ordered, unique)
	}
	return b.String()
}
