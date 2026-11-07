package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// WindZoneSpan describes the full zone length along the wind. Multiple
// disconnected patches at identical x-intervals share a dimension line.
// It derives from saved normative rectangles/zone boundaries, not new constants.
type WindZoneSpan struct {
	From, To float64
	Codes    []string
}

func windZoneSpans(v *WindVariant, z *WindZoneCalculation) []WindZoneSpan {
	if v == nil || z == nil {
		return nil
	}
	list := []WindZoneSpan{}
	for _, q := range v.Zones {
		if z.Surface == "roof" {
			if q.Surface != "roof" || q.Alternative != z.Alternative {
				continue
			}
		} else {
			if q.Surface == "roof" {
				continue
			}
			// A–C are the side-wall subdivision; D/E are separate end faces.
			if v.ID != "W" && (z.Code == "D" || z.Code == "E") {
				if q.Code != z.Code {
					continue
				}
			} else if v.ID != "W" && (q.Code == "D" || q.Code == "E") {
				continue
			}
		}
		if q.To-q.From <= 1e-9 {
			continue
		}
		merged := false
		for i := range list {
			if math.Abs(list[i].From-q.From) < 1e-8 && math.Abs(list[i].To-q.To) < 1e-8 {
				list[i].Codes = append(list[i].Codes, q.Code)
				merged = true
				break
			}
		}
		if !merged {
			list = append(list, WindZoneSpan{From: q.From, To: q.To, Codes: []string{q.Code}})
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].From == list[j].From {
			return list[i].To < list[j].To
		}
		return list[i].From < list[j].From
	})
	return list
}

// Dimension vocabulary: width is transverse to the wind, lengthwise extent
// is along the wind. Normative roof rectangles store X along wind and Y
// transverse, regardless of their orientation on the plan drawing.
func windZoneLengthsText(v *WindVariant, z *WindZoneCalculation) string {
	if v == nil || z == nil {
		return ""
	}
	parts := []string{}
	for _, q := range v.Zones {
		if z.Surface == "roof" {
			if q.Surface != "roof" || q.Alternative != z.Alternative || len(q.Rectangles) == 0 {
				continue
			}
			r := q.Rectangles[0]
			part := fmt.Sprintf("%s: ширина %s м, протяжённость %s м", q.Code, decimalInput(r.Y1-r.Y0, 2), decimalInput(r.X1-r.X0, 2))
			if len(q.Rectangles) > 1 {
				part += fmt.Sprintf(" (%d участка)", len(q.Rectangles))
			}
			parts = append(parts, part)
		} else if q.Surface != "roof" {
			if z.Code == "D" || z.Code == "E" {
				if q.Code != z.Code {
					continue
				}
				parts = append(parts, fmt.Sprintf("%s: ширина поперёк ветра %s м", q.Code, decimalInput(q.To-q.From, 2)))
			} else if q.Code != "D" && q.Code != "E" {
				parts = append(parts, fmt.Sprintf("%s: протяжённость вдоль ветра %s м", q.Code, decimalInput(q.To-q.From, 2)))
			}
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "Размеры зон: " + strings.Join(parts, "; ")
}

// Roof ordinates are restricted to the physical roof, with at least five
// intervals, and no interval greater than the engineer's selected step.
func WindRoofProfileHeights(height, rise, maxStep float64) ([]float64, error) {
	if math.IsNaN(height) || math.IsInf(height, 0) || height <= 0 || math.IsNaN(rise) || math.IsInf(rise, 0) || rise < 0 {
		return nil, fmt.Errorf("Некорректные высоты кровли H или h")
	}
	if math.IsNaN(maxStep) || math.IsInf(maxStep, 0) || maxStep <= 0 {
		return nil, fmt.Errorf("Укажите положительный шаг эпюры")
	}
	low := math.Max(0, height-rise)
	delta := height - low
	if delta <= 1e-12 {
		return []float64{height}, nil
	}
	intervalCount := math.Ceil(delta / math.Min(maxStep, delta/5))
	if intervalCount > 20000 {
		return nil, fmt.Errorf("Слишком много точек эпюры кровли; увеличьте шаг")
	}
	intervals := int(intervalCount)
	if intervals > 20000 {
		return nil, fmt.Errorf("Слишком много точек эпюры кровли; увеличьте шаг")
	}
	zs := make([]float64, intervals+1)
	for i := range zs {
		zs[i] = low + delta*float64(i)/float64(intervals)
	}
	zs[len(zs)-1] = height // the lower-side ordinate is the only rooftop value
	return zs, nil
}
