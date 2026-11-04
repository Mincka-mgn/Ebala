package main

import "fmt"

func snowJanuaryReductionReason(s SnowCalc) string {
	source := "п. 10.11 СП 20.13330.2016"
	if !s.JanuarySet {
		return "Средняя температура января не задана; пониженная нормативная нагрузка не рассчитывается (" + source + ")."
	}
	if s.JanuaryTemp > -5 {
		return fmt.Sprintf("Средняя температура января %s °C выше −5 °C; пониженная нормативная нагрузка не рассчитывается (%s).", decimalInput(s.JanuaryTemp, 2), source)
	}
	return "Пониженная нормативная нагрузка рассчитывается (" + source + ")."
}
