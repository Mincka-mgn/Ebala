package main

import (
	"fmt"
	"strings"
)

// Every exported alternative has a fixed number of Excel rows, but the abscissa
// formulas are tied to the *editable* input values. For profiles with narrow
// discontinuities we split the rows into segments whose endpoints are exactly
// the normative boundaries: the last row of one zone and the first of the next
// share the same Excel-computed x. This avoids missing a snow pocket merely
// because a uniform grid happened to skip it.
const snowExcelGridSamples = 128
const snowExcelGridMultiSamples = 12 * 32

func snowExcelSampleCount(s SnowCalc) int {
	switch s.Shape {
	case "multi_gable", "multi_vault", "shed":
		return snowExcelGridMultiSamples
	default:
		return snowExcelGridSamples
	}
}

func snowExcelSampleDomain(s SnowCalc) (string, string) {
	src := "'Исходные данные'!"
	cf := "'Коэффициенты'!"
	if s.RightRoof != nil && s.RightRoof.Shape != "none" {
		return "0", "(" + src + "$B$11+" + src + "$B$41)"
	}
	switch s.Shape {
	case "step", "wall":
		return "(-" + src + "$B$18)", src + "$B$19"
	case "multi_gable", "multi_vault", "shed":
		return "0", "(" + src + "$B$11*" + src + "$B$17)"
	default:
		return "0", cf + "$B$47"
	}
}

// The editable nodes are ordered along the roof. Clamping is only for the
// partition of the displayed calculation grid; it does not modify input values
// or replace the actual coefficient formula at that point.
func snowExcelSampleNodes(s SnowCalc) []string {
	src := "'Исходные данные'!"
	cf := "'Коэффициенты'!"
	start, end := snowExcelSampleDomain(s)
	if s.RightRoof != nil && s.RightRoof.Shape != "none" {
		left := src + "$B$11"
		nodes := []string{"0"}
		if s.Shape == "skylight" {
			b := "(" + left + "-" + src + "$B$16)/2"
			b1 := "MIN(" + b + "," + src + "$B$15)"
			nodes = append(nodes, "("+b+"-"+b1+")", b, "("+b+"+"+src+"$B$16)", "("+b+"+"+src+"$B$16+"+b1+")")
		} else {
			drift := "'Перепад Б.8'!$B$23"
			active := "'Перепад Б.8'!$B$25"
			width := "IF(" + active + "=\"Да\",MIN(" + drift + "," + left + ")," + left + "/2)"
			nodes = append(nodes, "("+left+"-"+width+")")
		}
		nodes = append(nodes, left)
		if s.RightRoof.Shape == "skylight" {
			b := "(" + src + "$B$41-" + src + "$B$46)/2"
			b1 := "MIN(" + b + "," + src + "$B$47)"
			nodes = append(nodes, "("+left+"+"+b+"-"+b1+")", "("+left+"+"+b+")", "("+left+"+"+b+"+"+src+"$B$46)", "("+left+"+"+b+"+"+src+"$B$46+"+b1+")")
		} else {
			drift := "'Перепад Б.8'!$B$23"
			active := "'Перепад Б.8'!$B$25"
			width := "IF(" + active + "=\"Да\",MIN(" + drift + "," + src + "$B$41)," + src + "$B$41/2)"
			nodes = append(nodes, "("+left+"+"+width+")")
		}
		return append(nodes, end)
	}
	switch s.Shape {
	case "skylight":
		b := cf + "$B$24"
		a := cf + "$B$23"
		b1 := cf + "$B$25"
		return []string{"0", "(" + b + "-" + b1 + ")", b, "(" + b + "+" + a + ")", "(" + b + "+" + a + "+" + b1 + ")", end}
	case "step", "wall":
		return []string{start, "0", "MIN(" + cf + "$B$36," + end + ")", end}
	case "vault":
		l := src + "$B$11"
		f := src + "$B$14"
		radius := "(" + l + "^2/(8*" + f + ")+" + f + "/2)"
		x60 := "MAX(0," + l + "/2-" + radius + "*SIN(RADIANS(60)))"
		x30 := "MAX(0," + l + "/2-" + radius + "*SIN(RADIANS(30)))"
		return []string{"0", x60, x30, "(" + l + "/2)", "(" + l + "-" + x30 + ")", "(" + l + "-" + x60 + ")", end}
	default:
		return []string{start, end}
	}
}

// The boundary table is intentionally independent of the variant catalogue.
// Both the detailed calculation sheet and its charts refer to these nodes,
// so the same geometric break cannot acquire two different X coordinates.
func snowExcelZoneBoundaryGuide(s SnowCalc) string {
	nodes := snowExcelSampleNodes(s)
	var b strings.Builder
	b.WriteString(tempExcelRow(1, "На этом листе вычислены точные координаты границ снеговых зон. Эти значения используются формулами всех вариантов и обновляются при изменении исходной геометрии."))
	b.WriteString(tempExcelRow(2, "Граница", "x, м", "Геометрический смысл"))
	descriptions := []string{}
	if s.RightRoof == nil || s.RightRoof.Shape == "none" {
		switch s.Shape {
		case "vault":
			descriptions = []string{"левая опора", "касательная 60° слева", "касательная 30° слева", "середина свода", "касательная 30° справа", "касательная 60° справа", "правая опора"}
		case "skylight":
			descriptions = []string{"левая граница кровли", "начало левой зоны b₁", "левая стенка фонаря", "правая стенка фонаря", "конец правой зоны b₁", "правая граница кровли"}
		case "step", "wall":
			descriptions = []string{"край верхнего покрытия", "линия перепада", "окончание местного снегозаноса", "край нижнего покрытия"}
		}
	}
	for i, node := range nodes {
		meaning := fmt.Sprintf("Граница расчётного участка № %d", i)
		if i < len(descriptions) {
			meaning = descriptions[i]
		}
		b.WriteString(tempExcelRow(i+3, fmt.Sprintf("Узел %d", i), "="+node, meaning))
	}
	return b.String()
}

// Distribute all rows over every segment, including exact endpoints. A formula
// in either side of a boundary remains live after the user changes geometry.
func snowExcelSampleInterval(s SnowCalc, index, count int) (string, string) {
	// Exactly 32 intervals are available for each of 12 possible bays. Changing
	// n in the source workbook does not shift the sampling points of the other
	// bays; rows past n remain truly empty, not a zero-width or zero-load bay.
	if s.Shape == "multi_gable" || s.Shape == "multi_vault" {
		bay := index / 32
		sub := index % 32
		src := "'Исходные данные'!"
		coord := func(k int) string {
			return fmt.Sprintf(`=IF(AND(%s$B$17>=%d,%s$B$17<=12),(%d+%d/32)*%s$B$11,"")`, src, bay+1, src, bay, k, src)
		}
		return coord(sub), coord(sub + 1)
	}
	nodes := snowExcelSampleNodes(s)
	segments := len(nodes) - 1
	// All nodes are computed once in the dedicated Excel worksheet.
	for i := range nodes {
		nodes[i] = fmt.Sprintf("'Границы зон'!$B$%d", i+3)
	}
	if segments <= 0 || count < segments {
		return "=0", "=0"
	}
	position := func(n int) string {
		if n <= 0 {
			return "=" + nodes[0]
		}
		if n >= count {
			return "=" + nodes[segments]
		}
		// Split the fixed capacity as evenly as possible. At each segment end
		// the two adjacent rows refer to the *same* exact boundary formula.
		first := 0
		for k := 0; k < segments; k++ {
			length := count / segments
			if k < count%segments {
				length++
			}
			if n <= first+length {
				local := n - first
				if local == 0 {
					return "=" + nodes[k]
				}
				if local == length {
					return "=" + nodes[k+1]
				}
				return fmt.Sprintf("=(%s)+((%s)-(%s))*%d/%d", nodes[k], nodes[k+1], nodes[k], local, length)
			}
			first += length
		}
		return "=" + nodes[segments]
	}
	return position(index), position(index + 1)
}
