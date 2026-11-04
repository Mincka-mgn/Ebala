package main

import (
	"fmt"
	"strings"
)

// The six independent half-/quarter-roof load cases share one polar grid.
// This avoids six copies of the same (computationally expensive) geometry.
// Each sector's mu, normative and design loads are nonetheless REAL Excel
// formulas linked exclusively to accepted input cells and computed geometry.
// Fixed 1-degree angular / 12-interval radial sampling also ensures that
// changing the originally selected beta step cannot remove an entire sector.
func snowIsPartialSector(c SnowCase) bool {
	_, _, ok := snowPartialSectorBounds(c)
	return ok
}

func snowExcelSectorMatrix(s SnowCalc) string {
	var b strings.Builder
	add := func(row int, cells ...string) { b.WriteString(tempExcelRow(row, cells...)) }
	src := "'Исходные данные'!"
	cf := "'Коэффициенты'!"
	geom := "'Геометрия Б.15а'!"
	sectors := [][2]int{{0, 180}, {180, 360}, {0, 90}, {90, 180}, {180, 270}, {270, 360}}
	add(1, "На этом листе приведена пространственная сетка шести самостоятельных частичных загружений круглого покрытия по пункту 10.4 СП 20.13330.2016. Шаг β равен 1°, радиус разделён на 12 интервалов; значения и итоговые нагрузки вычисляет Excel.")
	add(2, "Секторы", "0–180°", "180–360°", "0–90°", "90–180°", "180–270°", "270–360°")
	columns := []string{"β, град", "Радиус z, м", "Местный угол α, град", "μ₁ (основная схема)"}
	for _, p := range sectors {
		columns = append(columns, fmt.Sprintf("μ %d–%d°", p[0], p[1]))
	}
	for _, p := range sectors {
		columns = append(columns, fmt.Sprintf("Sн %d–%d°, кПа", p[0], p[1]))
	}
	for _, p := range sectors {
		columns = append(columns, fmt.Sprintf("Sр %d–%d°, кПа", p[0], p[1]))
	}
	add(3, columns...)
	for az := 0; az <= 360; az++ {
		for ir := 0; ir <= 12; ir++ {
			row := 4 + az*13 + ir
			beta := fmt.Sprintf("A%d", row)
			radius := fmt.Sprintf("B%d", row)
			angle := fmt.Sprintf("C%d", row)
			base := fmt.Sprintf("D%d", row)
			radialMax := src + "$B$11/2"
			if s.Shape == "dome" {
				radialMax = cf + "$B$47/2"
			}
			r := fmt.Sprintf("=%s*%d/12", radialMax, ir)
			var local, mu string
			switch s.Shape {
			case "cone":
				local = "=" + cf + "$B$4"
			case "dome":
				R := "(" + src + "$B$11^2/(8*" + src + "$B$14)+" + src + "$B$14/2)"
				local = fmt.Sprintf("=DEGREES(ASIN(MIN(1,%s/%s)))", radius, R)
			case "sphere_cone":
				local = fmt.Sprintf("=IF(OR(%s$B$22=\"Да\",%s<=%s$B$5),%s$B$6,ABS(DEGREES(ASIN(MAX(-1,MIN(1,(%s-%s$B$11)/ABS(%s$B$10)))))))", geom, radius, geom, geom, radius, geom, geom)
			default:
				local = "=0"
			}
			mu = fmt.Sprintf("=IF(%s<=30,1,IF(%s>=60,0,(60-%s)/30))", angle, angle, angle)
			if s.Shape == "sphere_cone" {
				h := "IF(OR(" + geom + "$B$22=\"Да\"," + radius + "<=" + geom + "$B$5)," +
					geom + "$B$2-" + radius + "*TAN(RADIANS(" + geom + "$B$6)),IF(" +
					geom + "$B$10>=0," + geom + "$B$12+SQRT(" + geom + "$B$10^2-(" + radius + "-" + geom + "$B$11)^2)," +
					geom + "$B$12-SQRT(" + geom + "$B$10^2-(" + radius + "-" + geom + "$B$11)^2)))"
				mu = fmt.Sprintf("=IF(%s<0,0,IF(%s<=30,1,IF(%s>=60,0,(60-%s)/30)))", h, angle, angle, angle)
			}
			eligibility := "ABS(" + cf + "$B$4)>20"
			if s.Shape == "dome" {
				eligibility = "DEGREES(2*ATAN(2*" + src + "$B$14/" + src + "$B$11))>20"
			} else if s.Shape == "sphere_cone" {
				eligibility = "ABS(" + geom + "$B$7)>20"
			}
			cells := []string{fmt.Sprintf("%d", az), r, local, mu}
			for _, bounds := range sectors {
				cells = append(cells, fmt.Sprintf(`=IF(%s,IF(AND(MOD(%s,360)>=%d,MOD(%s,360)<%d),%s,0),"")`, eligibility, beta, bounds[0], beta, bounds[1], base))
			}
			for i := 0; i < len(sectors); i++ {
				col := string(rune('E' + i))
				cells = append(cells, fmt.Sprintf(`=IF(%s%d="","",%s%d*%s$B$3*%s$B$19*%s$B$21)`, col, row, col, row, src, cf, cf))
			}
			for i := 0; i < len(sectors); i++ {
				col := string(rune('K' + i))
				cells = append(cells, fmt.Sprintf(`=IF(%s%d="","",%s%d*%s$B$4)`, col, row, col, row, src))
			}
			add(row, cells...)
		}
	}
	return b.String()
}
