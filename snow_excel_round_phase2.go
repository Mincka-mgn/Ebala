package main

import (
	"fmt"
	"strings"
)

// A self-contained polar calculation for cones and spherical domes.
// No radius, angle, zone or mu from SnowCase.Polar is transmitted to Excel.
// The radial/azimuth sampling is calculated from user inputs and the
// normalized user-selected beta step, with capacity for steps >= 1 degree.
// The client can inspect all applied equations in columns H-K.
func snowExcelRoundPolar(s SnowCalc, c SnowCase, index int, input, coef string) string {
	var b strings.Builder
	add := func(n int, cells ...string) { b.WriteString(tempExcelRow(n, cells...)) }
	src := "'" + input + "'!"
	cf := "'" + coef + "'!"
	add(1, fmt.Sprintf("На этом листе независимо рассчитываются пространственные μ, снеговые нагрузки и зонирование круглого покрытия для варианта %d. Радиусы и углы формируются формулами Excel.", index+1))
	add(2, "Sg, кПа", "="+src+"B3", "ce", "="+cf+"B19", "ct", "="+cf+"B21", "γf", "="+src+"B4")
	add(3, "β, град", "Радиус z, м", "Локальный α, град", "μ(z, β)", "Sн, кПа", "Sр, кПа", "Вариант", "Базовый μ₁(α)", "Cr и геометрические параметры", "Нормативная зависимость")
	partialLo, partialHi, partial := snowPartialSectorBounds(c)
	for j := 0; j <= 360; j++ {
		for i := 0; i <= 24; i++ {
			row := 4 + j*25 + i
			beta := fmt.Sprintf("A%d", row)
			z := fmt.Sprintf("B%d", row)
			alpha := fmt.Sprintf("C%d", row)
			base := fmt.Sprintf("H%d", row)
			radial := fmt.Sprintf("I%d", row)
			bt := fmt.Sprintf("=IF(AND(%s$B$33>0,%d<=ROUNDUP(360/%s$B$33,0)),MIN(360,%d*%s$B$33),\"\")", src, j, src, j, src)
			rad := ""
			if s.Shape == "dome" {
				rad = fmt.Sprintf("=IF(%s=\"\",\"\",%s$B$47/2*%d/24)", beta, cf, i)
			} else {
				rad = fmt.Sprintf("=IF(%s=\"\",\"\",%s$B$11/2*%d/24)", beta, src, i)
			}
			ang := ""
			if s.Shape == "dome" {
				rr := fmt.Sprintf("(%s$B$11^2/(8*%s$B$14)+%s$B$14/2)", src, src, src)
				ang = fmt.Sprintf("=IF(%s=\"\",\"\",DEGREES(ASIN(MIN(1,%s/%s))))", beta, z, rr)
			} else {
				ang = "=IF(" + beta + "=\"\",\"\"," + cf + "$B$4)"
			}
			u := fmt.Sprintf("=IF(%s=\"\",\"\",IF(%s<=30,1,IF(%s>=60,0,(60-%s)/30)))", beta, alpha, alpha, alpha)
			rCoeff := ""
			if s.Shape == "cone" {
				rCoeff = fmt.Sprintf("=IF(%s=\"\",\"\",IF(OR(%s$B$4<=7,%s$B$4>=60),0,IF(%s$B$4<=30,2.15*POWER(RADIANS(%s$B$4),1/3),1.7*30/%s$B$4)) * %s/(%s$B$11/2))", beta, cf, cf, cf, cf, cf, z, src)
			} else if strings.HasPrefix(c.Title, "3.") {
				rCoeff = fmt.Sprintf("=IF(%s=\"\",\"\",MAX(0,3*SQRT(2*%s$B$14/%s$B$11)*SIN(RADIANS(3*%s))))", beta, src, src, alpha)
			} else {
				rr := fmt.Sprintf("(%s$B$11^2/(8*%s$B$14)+%s$B$14/2)", src, src, src)
				cr := fmt.Sprintf("(2.55-EXP(0.8-14*%s$B$14/%s$B$11))", src, src)
				r30 := "(" + rr + "*0.5)"
				v := fmt.Sprintf("IF(%s<=30,%s*(%s/%s)^2,IF(%s<=45,%s+(%s-30)/15*(1.5-%s),IF(%s<60,1.5*(60-%s)/15,0)))", alpha, cr, z, r30, alpha, cr, alpha, cr, alpha, alpha)
				rCoeff = fmt.Sprintf("=IF(%s=\"\",\"\",MAX(0,%s))", beta, v)
			}
			mu := ""
			desc := ""
			switch {
			case partial:
				mu = fmt.Sprintf("=IF(%s=\"\",\"\",IF(AND(MOD(%s,360)>=%g,MOD(%s,360)<%g),%s,0))", beta, beta, partialLo, beta, partialHi, base)
				desc = "Частичное секторное загружение по п. 10.4; μ₁ внутри сектора"
			case strings.HasPrefix(c.Title, "1."):
				mu = "=IF(" + beta + "=\"\",\"\"," + base + ")"
				desc = "Равномерная схема: μ₁ по таблице Б.3"
			default:
				mu = fmt.Sprintf("=IF(%s=\"\",\"\",MAX(0,%s*SIN(RADIANS(%s))))", beta, radial, beta)
				desc = "Пространственная схема: Б.10–Б.13, множитель sin β"
			}
			variantTitle, normDescription := "", ""
			if j == 0 && i == 0 {
				variantTitle, normDescription = c.Title, desc
			}
			add(row, bt, rad, ang, mu,
				fmt.Sprintf("=IF(%s=\"\",\"\",D%d*$B$2*$D$2*$F$2)", beta, row),
				fmt.Sprintf("=IF(%s=\"\",\"\",E%d*$H$2)", beta, row),
				variantTitle, u, rCoeff, normDescription)
		}
	}
	return b.String()
}
