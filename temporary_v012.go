package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type TempAreaMode int

const (
	TempFull TempAreaMode = iota
	TempOneFloor
	TempSeveralFloors
)

// The 6.7/6.8 reductions are permitted only for certain entries of SP20 table 8.3.
// Classes are mapped by immutable catalogue row IDs, never by user-facing titles.
// SP 20 table 8.3, area reductions §6.7–6.8.
var sp20ReductionIDs = map[string]int{
	// Table 8.3 pos. 1, 2 and 12a: φ1 (A1=9 m²)
	"T-00012-1": 1, "T-00029-2": 1, "T-00013-6": 1, "T-00029-8": 1,
	// Table 8.3 pos. 4, 11 and 12b: φ2 (A2=36 m²)
	"T-00015-9": 2, "T-00016-11": 2, "T-00017-12": 2, "T-00018-13": 2,
	"T-00030-10": 2, "T-00030-15": 2, "T-00028-25": 2,
	"T-00030-20": 2, "T-00030-26": 2,
}

func tempReductionClass(r SP20Row) int {
	if !strings.Contains(r.Source, "8.3") {
		return 0
	}
	return sp20ReductionIDs[r.ID]
}
func tempAreaFactor(r SP20Row, mode TempAreaMode, area float64, floors int) (float64, error) {
	if mode == TempFull || tempReductionClass(r) == 0 {
		return 1, nil
	}
	if area <= 0 {
		return 0, fmt.Errorf("Укажите грузовую площадь, м²")
	}
	class := tempReductionClass(r)
	base, min := 9.0, 0.4
	if class == 2 {
		base, min = 36, 0.5
	}
	phi := 1.0
	if area > base {
		phi = min + (1-min)*math.Sqrt(base/area)
	}
	if mode == TempSeveralFloors {
		if floors < 2 {
			return 0, fmt.Errorf("Число перекрытий должно быть не менее 2")
		}
		phi = min + (phi-min)/math.Sqrt(float64(floors))
	}
	return phi, nil
}
func tempSP20Section(r SP20Row, mode TempAreaMode, area float64, floors int) (TempSelection, error) {
	factor, e := tempAreaFactor(r, mode, area, floors)
	if e != nil {
		return TempSelection{}, e
	}
	s := sp20Selection(r)
	s.Params["area_factor"] = factor
	s.Params["normative_reduced"] = r.Norm * factor
	s.Params["design_reduced"] = r.Design * factor
	scope := "Полное значение"
	if mode != TempFull && tempReductionClass(r) != 0 {
		if mode == TempOneFloor {
			scope = fmt.Sprintf("Одно перекрытие, A=%g м², φ=%.3f; СП 20.13330.2016, п. 6.7", area, factor)
		} else {
			scope = fmt.Sprintf("%d перекрытий, A=%g м², φ=%.3f; СП 20.13330.2016, п. 6.8", floors, area, factor)
		}
	}
	s.Params["calc_scope"] = scope
	return s, nil
}
func tempRU(value float64, precision int) string {
	return strings.ReplaceAll(strconv.FormatFloat(value, 'f', precision, 64), ".", ",")
}
func tempUnitPair(value float64, unit string) string {
	switch unit {
	case "кПа":
		return tempRU(value*kpaToKgfM2(1), 1) + " кгс/м² (" + tempRU(value, 3) + " кПа)"
	case "кН/м":
		return tempRU(value*kgfPerKN, 1) + " кгс/м (" + tempRU(value, 3) + " кН/м)"
	case "кН":
		return tempRU(value*kgfPerKN, 1) + " кгс (" + tempRU(value, 3) + " кН)"
	default:
		return fmt.Sprintf("%.3f %s", value, unit)
	}
}
func tempNumber(s TempSelection, key string) float64 {
	if s.Params == nil {
		return 0
	}
	switch v := s.Params[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(strings.ReplaceAll(v, ",", "."), 64)
		return f
	}
	return 0
}
func tempIsCrane(s TempSelection) bool { return strings.Contains(strings.ToLower(s.Group), "кран") }
func tempSp20Display(s TempSelection) string {
	if s.Group == "СП 20 · временные" {
		unit := tempNumberText(s.Params["unit"])
		return tempUnitPair(tempNumber(s, "normative_reduced"), unit)
	}
	return "По схеме"
}
func tempForces(s TempSelection) string {
	items := make([]string, 0, len(s.Wheels))
	for _, w := range s.Wheels {
		items = append(items, strconv.FormatFloat(w.PKN, 'f', -1, 64))
	}
	return strings.Join(items, "; ")
}
func tempSteps(s TempSelection) string {
	if len(s.Wheels) < 2 {
		return ""
	}
	vals := make([]string, 0, len(s.Wheels)-1)
	for i := 1; i < len(s.Wheels); i++ {
		if !s.Wheels[i].XKnown || !s.Wheels[i-1].XKnown {
			return ""
		}
		vals = append(vals, fmt.Sprintf("%g", s.Wheels[i].XMM-s.Wheels[i-1].XMM))
	}
	return strings.Join(vals, "; ")
}

// Step positions are distances between consecutive wheels, count-1 values.
func tempManualSteps(title string, count int, forces, steps string) (TempSelection, error) {
	if strings.TrimSpace(title) == "" {
		title = "Мостовой кран"
	}
	f, e := parseWheelVector(forces, count, "Нагрузки на колёса", true)
	if e != nil {
		return TempSelection{}, e
	}
	var distances []float64
	if strings.TrimSpace(steps) != "" {
		if count < 2 {
			return TempSelection{}, fmt.Errorf("для одного колеса расстояния не требуются")
		}
		distances, e = parseWheelVector(steps, count-1, "Расстояния между соседними колёсами, мм", true)
		if e != nil {
			return TempSelection{}, e
		}
	}
	w := make([]TempWheel, count)
	x := 0.0
	for i := range w {
		if i > 0 && len(distances) > 0 {
			x += distances[i-1]
		}
		w[i] = TempWheel{XMM: x, PKN: f[i], XKnown: len(distances) > 0}
	}
	return TempSelection{Group: "Краны · ручной ввод", Title: strings.TrimSpace(title), Source: "Данные пользователя", Params: map[string]interface{}{"wheel_count": count}, Wheels: w}, nil
}

func tempForcesUnit(s TempSelection, tons bool) string {
	parts := make([]string, len(s.Wheels))
	for i, w := range s.Wheels {
		v := w.PKN
		if tons {
			v /= 9.80665
		}
		parts[i] = tempRU(v, 2)
	}
	return strings.Join(parts, "; ")
}
func tempForcesBoth(s TempSelection) string {
	if len(s.Wheels) == 0 {
		return "Не задано"
	}
	return tempForcesUnit(s, false) + " кН (" + tempForcesUnit(s, true) + " тс)"
}
