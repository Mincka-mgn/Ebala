package main

import (
	"fmt"
	"strings"
)

// A paired roof is NOT a roof of the left-hand shape with a longer span.
// Each roof must use its own editable dimensions and its own B.1/B.2/B.3
// coefficient formula. A combined case always uses the ordinary base case on
// the unchanged roof, exactly as snowPairedCases in snow.go does.
func snowExcelPairedMu(s SnowCalc, c SnowCase, row int) (string, string) {
	if s.RightRoof == nil || s.RightRoof.Shape == "none" {
		return "=NA()", "Нет второго пролёта"
	}
	src := "'Исходные данные'!"
	cf := "'Коэффициенты'!"
	x := fmt.Sprintf("((A%d+B%d)/2)", row, row)
	leftSpan := src + "$B$11"
	rightSpan := src + "$B$41"
	rightX := "(" + x + "-" + leftSpan + ")"
	leftName, rightName := "", ""
	title := c.Title
	if strings.HasPrefix(title, "Занос у перепада высот:") {
		return snowExcelPairedStepMu(s, row)
	}
	switch {
	case strings.HasPrefix(title, "1. Основное загружение обоих пролётов"):
		// Both components use their first normative arrangement.
	case strings.HasPrefix(title, "Левый пролёт: "):
		leftName = strings.TrimPrefix(title, "Левый пролёт: ")
	case strings.HasPrefix(title, "Правый пролёт: "):
		rightName = strings.TrimPrefix(title, "Правый пролёт: ")
	case strings.HasPrefix(title, "Оба фонаря, зона "):
		split := strings.SplitN(title, "левый ", 2)
		if len(split) != 2 {
			return "=NA()", "Не определена схема обоих фонарей"
		}
		parts := strings.SplitN(split[1], " + правый ", 2)
		if len(parts) != 2 {
			return "=NA()", "Не определена схема обоих фонарей"
		}
		leftName, rightName = parts[0], parts[1]
	default:
		// These cases include additional cross-wall snowdrift. The roof
		// combination requires its own B.8 geometry, not reuse of B.1's μ.
		return "=NA()", "Для совмещённого заноса у перепада высот требуется отдельная формула Б.8"
	}
	left, ld, lok := snowExcelPartMu(s.Shape, leftName, x, leftSpan, cf+"$B$4", src+"$B$14", src+"$B$16", src+"$B$15", src+"$B$3")
	right, rd, rok := snowExcelPartMu(s.RightRoof.Shape, rightName, rightX, rightSpan, cf+"$B$55", func() string {
		if s.RightRoof.Shape == "vault" {
			return src + "$B$45"
		}
		return cf + "$B$56"
	}(), src+"$B$46", src+"$B$47", src+"$B$3")
	if !lok || !rok {
		return "=NA()", fmt.Sprintf("Не реализована формула совмещённой кровли: слева %s (%s), справа %s (%s)", s.Shape, ld, s.RightRoof.Shape, rd)
	}
	return fmt.Sprintf("=IF(%s<%s,%s,%s)", x, leftSpan, left, right), "Два покрытия: слева «" + ld + "», справа «" + rd + "»; зависимость от собственных геометрических параметров"
}

// snowExcelPartMu returns an Excel expression WITHOUT the leading '='.
// Neither the shape coefficients nor the interval positions come from Go.
func snowExcelPartMu(shape, title, x, span, angle, rise, skylightWidth, lanternHeight, sg string) (string, string, bool) {
	base := fmt.Sprintf("IF(ABS(%s)<=30,1,IF(ABS(%s)>=60,0,(60-ABS(%s))/30))", angle, angle, angle)
	center := span + "/2"
	switch shape {
	case "flat", "mono", "gable":
		factor := "1"
		description := "Б.1, равномерное загружение"
		switch {
		case title == "" || strings.HasPrefix(title, "1. Равномерная"):
		case strings.Contains(title, "неравномерная 0,9/1,1"):
			factor = fmt.Sprintf("IF(%s<%s,0.9,1.1)", x, center)
			description = "Б.1, контрольные 0,9/1,1"
		case strings.HasPrefix(title, "2. Неравномерная") && shape == "gable":
			factor = fmt.Sprintf("IF(%s<%s,0.75,1.25)", x, center)
			description = "Б.1, неравномерная 0,75/1,25"
		case strings.HasPrefix(title, "3. Неравномерная") && shape == "gable":
			factor = fmt.Sprintf("IF(%s<%s,1.25,0.75)", x, center)
			description = "Б.1, зеркальная неравномерная"
		case strings.Contains(title, "левого ската") && shape == "gable":
			factor = fmt.Sprintf("IF(%s<%s,1,0)", x, center)
			description = "10.4, загружение левого ската"
		case strings.Contains(title, "правого ската") && shape == "gable":
			factor = fmt.Sprintf("IF(%s>=%s,1,0)", x, center)
			description = "10.4, загружение правого ската"
		default:
			return "", "неподдерживаемый вариант Б.1: " + title, false
		}
		return "(" + base + ")*(" + factor + ")", description, true
	case "vault":
		r := fmt.Sprintf("(%s^2/(8*%s)+%s/2)", span, rise, rise)
		dx := fmt.Sprintf("ABS(%s-%s/2)", x, span)
		angleAt := fmt.Sprintf("DEGREES(ATAN2(SQRT(MAX(0,%s^2-%s^2)),%s))", r, dx, dx)
		uniform := fmt.Sprintf("MAX(0,COS(1.5*RADIANS(%s)))", angleAt)
		switch {
		case title == "" || strings.HasPrefix(title, "1. Симметричная"):
			return uniform, "Б.2, μ₁ = cos(1,5α)", true
		case strings.HasPrefix(title, "Частичное загружение"):
			if strings.Contains(title, "зеркальное") {
				return fmt.Sprintf("IF(%s>%s/2,%s,0)", x, span, uniform), "10.4, правая половина свода", true
			}
			return fmt.Sprintf("IF(%s<%s/2,%s,0)", x, span, uniform), "10.4, левая половина свода", true
		case strings.HasPrefix(title, "2. Несимметричная"), strings.HasPrefix(title, "3. Несимметричная"):
			// Piecewise linear interpolation on the x30 and x60 control
			// sections is separate from the smooth μ₁ formula.
			edge := fmt.Sprintf("DEGREES(ATAN2(SQRT(MAX(0,%s^2-(%s/2)^2)),%s/2))", r, span, span)
			xe := fmt.Sprintf("MIN(%s,%s-%s)", x, span, x)
			qEdge := fmt.Sprintf("IF(%s>=60,0,2*SIN(RADIANS(3*%s)))", edge, edge)
			x60 := fmt.Sprintf("(%s/2-%s*SIN(RADIANS(60)))", span, r)
			x30 := fmt.Sprintf("(%s/2-%s*SIN(RADIANS(30)))", span, r)
			lo := fmt.Sprintf("(%s+(0-%s)*%s/%s)", qEdge, qEdge, xe, x60)
			middle := fmt.Sprintf("(2*(%s-%s)/(%s-%s))", xe, x60, x30, x60)
			from30 := fmt.Sprintf("(2*(%s/2-%s)/(%s/2-%s))", span, xe, span, x30)
			fromEdge := fmt.Sprintf("(%s+(2-%s)*%s/%s)", qEdge, qEdge, xe, x30)
			mu := fmt.Sprintf("IF(%s>60,IF(%s<=%s,%s,IF(%s<=%s,%s,%s)),IF(%s>30,IF(%s<=%s,%s,%s),%s*(1-2*%s/%s)))", edge, xe, x60, lo, xe, x30, middle, from30, edge, xe, x30, fromEdge, from30, qEdge, xe, span)
			leftGreater := strings.HasPrefix(title, "2.")
			if leftGreater {
				return fmt.Sprintf("(%s)*IF(%s<=%s/2,1,0.5)", mu, x, span), "Б.2, несимметричная, контрольные 30°/60°", true
			}
			return fmt.Sprintf("(%s)*IF(%s<=%s/2,0.5,1)", mu, x, span), "Б.2, зеркальная несимметричная", true
		default:
			return "", "неподдерживаемый вариант свода: " + title, false
		}
	case "skylight":
		b := "(" + span + "-" + skylightWidth + ")/2"
		b1 := "MIN(" + b + "," + lanternHeight + ")"
		mu2 := fmt.Sprintf("MIN(4,MIN(3*%s/%s,1+0.1*%s/%s))", lanternHeight, sg, skylightWidth, b)
		mu3 := fmt.Sprintf("MIN(4,MIN(3*%s/%s,1+0.5*%s/%s))", lanternHeight, sg, skylightWidth, b1)
		switch {
		case title == "" || strings.HasPrefix(title, "1A"):
			return "1", "Б.3, равномерная зона A", true
		case strings.HasPrefix(title, "1C"):
			return fmt.Sprintf("IF(%s<%s,%s,IF(%s<%s+%s,0.8,%s))", x, b, mu2, x, b, skylightWidth, mu2), "Б.3, боковые зоны C", true
		case strings.HasPrefix(title, "2A"):
			return fmt.Sprintf("IF(AND(%s>=%s-%s,%s<=%s+%s+%s),%s,1)", x, b, b1, x, b, skylightWidth, b1, mu3), "Б.3, центральное накопление A", true
		case strings.HasPrefix(title, "2C"):
			left := fmt.Sprintf("AND(%s>=%s-%s,%s<%s)", x, b, b1, x, b)
			center := fmt.Sprintf("AND(%s>=%s,%s<%s+%s)", x, b, x, b, skylightWidth)
			right := fmt.Sprintf("AND(%s>=%s+%s,%s<%s+%s+%s)", x, b, skylightWidth, x, b, skylightWidth, b1)
			return fmt.Sprintf("IF(%s,%s,IF(%s,0,IF(%s,%s,1)))", left, mu3, center, right, mu3), "Б.3, боковые участки 2C", true
		case strings.Contains(title, "Частичное загружение у левого ската"):
			return fmt.Sprintf("IF(%s<%s,%s,0)", x, b, mu2), "10.4, левый скат фонаря", true
		case strings.Contains(title, "Частичное загружение у правого ската"):
			return fmt.Sprintf("IF(%s>=%s+%s,%s,0)", x, b, skylightWidth, mu2), "10.4, правый скат фонаря", true
		default:
			return "", "неподдерживаемый вариант фонаря: " + title, false
		}
	}
	return "", "неподдерживаемый тип соседней кровли: " + shape, false
}
