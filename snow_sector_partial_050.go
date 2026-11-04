package main

import (
	"fmt"
	"math"
	"strings"
)

// Б.10–Б.12 и п. 10.4 СП 20.13330.2016: additional half- and quarter-
// area loads are independent load arrangements, not simultaneous sectors.
// The meridional section displayed in the main plot follows the bisector of
// the loaded sector; the polar grid defines the actual two-dimensional load.
func snowAddSectorPartialCases(s SnowCalc, cases []SnowCase) []SnowCase {
	if len(cases) == 0 {
		return cases
	}
	if s.Shape != "cone" && s.Shape != "dome" && s.Shape != "sphere_cone" {
		return cases
	}
	maxAngle := s.Angle
	if s.Shape == "dome" && s.Span > 0 {
		maxAngle = 2 * math.Atan2(2*s.Rise, s.Span) * 180 / math.Pi
	}
	if s.Shape == "sphere_cone" {
		maxAngle = math.Abs(sphereConeTangentAngle(s))
	}
	if maxAngle <= 20 {
		return cases
	}
	D := s.Span
	if D <= 0 {
		return cases
	}
	base := cases[0]
	type span struct{ start, end int }
	sectors := []span{{0, 180}, {180, 360}, {0, 90}, {90, 180}, {180, 270}, {270, 360}}
	for _, sect := range sectors {
		center := float64(sect.start+sect.end) / 2
		sampled := func(radius, beta float64) float64 {
			beta = math.Mod(beta+360, 360)
			if beta < float64(sect.start) || beta >= float64(sect.end) {
				return 0
			}
			v := snowMuAt(base, D/2+radius)
			if v < 0 {
				return 0
			}
			return v
		}
		intervals := make([]SnowInterval, 0, 80)
		for j := 0; j < 80; j++ {
			x1 := D * float64(j) / 80
			x2 := D * float64(j+1) / 80
			x := (x1 + x2) / 2
			beta := center
			if x < D/2 {
				beta += 180
			}
			intervals = append(intervals, snowMuInterval(x1, x2, sampled(math.Abs(x-D/2), beta)))
		}
		title := fmt.Sprintf("Частичное загружение: сектор %d–%d°", sect.start, sect.end)
		note := fmt.Sprintf("Сектор β от %d до %d град; отдельно проверяемая половина/четверть площади. Эпюра построена по диаметру через биссектрису сектора; полная нагрузка задана полярной сеткой.", sect.start, sect.end)
		c := snowCase(title, snowNormSource+", п. 10.4; приложение Б, Б.10–Б.12", note, intervals...)
		for i := 0; i <= 12; i++ {
			rad := D / 2 * float64(i) / 12
			for j := 0; j <= 36; j++ {
				beta := float64(j) * 10
				c.Polar = append(c.Polar, SnowPolarPoint{Radius: rad, Beta: beta, Mu: sampled(rad, beta)})
			}
		}
		cases = append(cases, c)
	}
	return cases
}

func snowPartialSectorBounds(c SnowCase) (float64, float64, bool) {
	if !strings.HasPrefix(c.Title, "Частичное загружение: сектор ") {
		return 0, 0, false
	}
	var a, b float64
	if _, err := fmt.Sscanf(c.Title, "Частичное загружение: сектор %f–%f°", &a, &b); err != nil {
		return 0, 0, false
	}
	return a, b, true
}
