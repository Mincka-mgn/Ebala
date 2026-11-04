package main

import (
	"fmt"
	"strings"
)

// Independent B.8 snowdrift for two adjacent roofs. A single list of
// intermediate Excel cells, rather than a nested string copied into each
// sample, is needed both for verification and acceptable XLSX file size.
// All calculated coefficients depend on editable input cells.
func snowExcelPairedStepData(s SnowCalc) string {
	src := "'Исходные данные'!"
	cf := "'Коэффициенты'!"
	var b strings.Builder
	add := func(row int, cells ...string) { b.WriteString(tempExcelRow(row, cells...)) }
	add(1, "На этом листе рассчитаны параметры снегового мешка Б.8 для перепада высот между соседними покрытиями. Изменение размеров и отметок пересчитывает высоту перепада, сторону нижней кровли, μ и ширину заноса.")
	add(2, "Параметр", "Значение / формула Excel", "Нормативная зависимость")
	leftRise := cf + "$B$43"
	if s.Shape == "vault" {
		leftRise = src + "$B$14"
	}
	leftEdge := src + "$B$10"
	if s.Shape == "mono" {
		leftEdge = "IF(" + cf + "$B$4>=0," + leftEdge + "," + leftEdge + "-ABS(" + leftRise + "))"
	} else if s.Shape != "flat" {
		leftEdge = "(" + leftEdge + "-ABS(" + leftRise + "))"
	}
	add(3, "Zл у сопряжения, м", "="+leftEdge, "Геометрия левой кровли, у правой кромки")
	rightRise := cf + "$B$56"
	if s.RightRoof != nil && s.RightRoof.Shape == "vault" {
		rightRise = src + "$B$45"
	}
	rightEdge := src + "$B$43"
	switch s.RightRoof.Shape {
	case "mono":
		rightEdge = "IF(" + cf + "$B$55<0," + rightEdge + "," + rightEdge + "-ABS(" + rightRise + "))"
	case "flat":
	default:
		rightEdge = "(" + rightEdge + "-ABS(" + rightRise + "))"
	}
	rightEdge = "IF(" + src + "$B$43=0,IF(" + src + "$B$51=0,B3," + src + "$B$51)," + rightEdge + ")"
	add(4, "Zп у сопряжения, м", "="+rightEdge, "Геометрия правой кровли, у левой кромки")
	add(5, "Разность отметок ΔZ, м", "=ABS(B3-B4)", "Модуль разности отметок в месте сопряжения")
	add(6, "Высокая сторона слева", "=B3>B4", "Булев выбор стороны высокого покрытия")
	add(7, "u — длина верхней кровли, м", "=IF(B6,"+src+"$B$11,"+src+"$B$41)", "Б.8: снегосборная длина верхней крыши")
	add(8, "d — длина нижней кровли, м", "=IF(B6,"+src+"$B$41,"+src+"$B$11)", "Б.8: длина нижней крыши")
	add(9, "α верхнего покрытия, °", "=IF(B6,"+cf+"$B$4,"+cf+"$B$55)", "Коэффициент m₁ зависит от уклона верхнего покрытия")
	add(10, "α нижнего покрытия, °", "=IF(B6,"+cf+"$B$55,"+cf+"$B$4)", "Коэффициент m₂ зависит от уклона нижнего покрытия")
	add(11, "h — эффективный перепад, м", "=MIN(8,B5)", "Б.8: h не более 8 м")
	add(12, "m₁ верхнего покрытия", "=IF(B9>20,0.3,0.4)", "Б.8, формула Б.5")
	m2 := func(shape, span, rise, angle string) string {
		k2 := "(1-MIN(35," + angle + ")/35)"
		k3 := "MAX(0.3,1-" + angle + "/30)"
		if shape == "flat" || shape == "vault" {
			k2, k3 = "1", "1"
		}
		steep := "(" + angle + ">20)"
		if shape == "vault" {
			steep = "OR(" + angle + ">20," + rise + "/" + span + ">1/8)"
		}
		return "IF(" + span + "<21,MAX(0.1,0.5*SQRT(" + span + "/21)*" + k2 + "*" + k3 + "),IF(" + steep + ",0.3,0.4))"
	}
	add(13, "m₂ для правого покрытия", "="+m2(s.RightRoof.Shape, src+"$B$41", rightRise, cf+"$B$55"), "Б.8, учёт типа и уклона правой кровли")
	add(14, "m₂ для левого покрытия", "="+m2(s.Shape, src+"$B$11", leftRise, cf+"$B$4"), "Б.8, учёт типа и уклона левой кровли")
	add(15, "m₂ нижней кровли", "=IF(B6,B13,B14)", "Выбор расчётного коэффициента нижней кровли")
	add(16, "B нижней кровли, м", "=IF(B6,IF("+src+"$B$42>0,"+src+"$B$42,"+src+"$B$12),"+src+"$B$12)", "Параметр транспортной длины при наличии второго покрытия")
	add(17, "dпер — длина переноса, м", "=MIN(B8,3*B16)", "Б.8: длина переноса не больше 3B")
	add(18, "Sосн = Sg·ct, кПа", "="+cf+"$B$2*"+cf+"$B$21", "Для местного заноса у перепада ce=1")
	add(19, "μ без ограничения", "=IF(B11=0,\"\",1+(B12*B7+B15*B17)/B11)", "Б.8, формулы Б.5–Б.6")
	add(20, "Предел μmax", "=IF(OR(B11=0,B18=0),\"\",MIN(2*B11/B18,IF(MAX(B7,B8)>48,4+2*MIN(1,MAX(0,(MAX(B7,B8)-48)/24)),4)))", "Б.8: предел μmax по нагрузке и длине крыши")
	add(21, "μ мешка", "=IF(OR(B19=\"\",B20=\"\"),\"\",MAX(1,MIN(B19,B20)))", "Б.8, итоговая ордината после ограничений")
	add(22, "Знаменатель для b", "=IF(OR(B11=0,B18=0),\"\",2*B11/B18-1+2*B15)", "Б.8: альтернативная ширина местного заноса")
	trial := "MIN(IF(AND(B21>2*B11/B18,B22>0),2*B11*(B21-1+2*B15)/B22,MIN(2*B11,16)),5*B11,16)"
	add(23, "b — ширина зоны снегозаноса, м", "=IF(B21=\"\",\"\",IF("+trial+">0,"+trial+",MIN(B8,1)))", "Б.8: условие b≤min(5h;16 м), корректная область")
	add(24, "μ вне мешка", "=IF(B23=\"\",\"\",MAX(0,IF(AND(B23<B8,B21>2*B11/B18),MAX(0.2,(B8-0.5*B21*B23)/(B8-0.5*B23)),1-2*B15)))", "Б.8: ордината за участком местного заноса")
	add(25, "Применимость отдельного заноса", "=IF(OR(B11=0,B18=0),\"Нет\",IF(B5>=B18/2,\"Да\",\"Нет\"))", "Б.8, примечание 3: порог от полного перепада ΔZ")
	return b.String()
}

func snowExcelPairedStepMu(s SnowCalc, row int) (string, string) {
	src := "'Исходные данные'!"
	cf := "'Коэффициенты'!"
	step := "'Перепад Б.8'!"
	x := fmt.Sprintf("((A%d+B%d)/2)", row, row)
	left, _, lok := snowExcelPartMu(s.Shape, "", x, src+"$B$11", cf+"$B$4", src+"$B$14", src+"$B$16", src+"$B$15", src+"$B$3")
	rightLocal := "(" + x + "-" + src + "$B$11)"
	right, _, rok := snowExcelPartMu(s.RightRoof.Shape, "", rightLocal, src+"$B$41", cf+"$B$55", func() string {
		if s.RightRoof.Shape == "vault" {
			return src + "$B$45"
		}
		return cf + "$B$56"
	}(), src+"$B$46", src+"$B$47", src+"$B$3")
	if !lok || !rok {
		return "=NA()", "Не рассчитана одна из основных форм покрытия для заноса Б.8"
	}
	// Position measured from the shared wall outwards along the lower roof.
	distance := fmt.Sprintf("IF(%s$B$6,%s-%s$B$11,%s$B$11-%s)", step, x, src, src, x)
	drift := fmt.Sprintf("IF(%s<%s$B$23,%s$B$21+(%s$B$24-%s$B$21)*%s/%s$B$23,%s$B$24)", distance, step, step, step, step, distance, step, step)
	mu := fmt.Sprintf("IF(%s$B$6,IF(%s<%s$B$11,%s,%s),IF(%s<%s$B$11,%s,%s))", step, x, src, left, drift, x, src, drift, right)
	return fmt.Sprintf("=IF(%s$B$25=\"Да\",%s,\"\")", step, mu), "Б.8: локальный занос только на нижней кровле; параметры из отдельного расчётного листа «Перепад Б.8»"
}
