package main

import (
	"fmt"
	"image/color"
	"math"
	"sort"
)

// The drawing axis is strictly linear. A schematic never shortens zones or
// inserts a conditional break: printed dimensions always describe the same
// geometry as the wind calculation.
type windDrawingAxis struct {
	cuts        []float64
	start, span int
}

func windAxis(cuts []float64, start, span int) windDrawingAxis {
	return windDrawingAxis{cuts: windSortedCuts(cuts), start: start, span: span}
}
func (a windDrawingAxis) px(x float64) int {
	if len(a.cuts) < 2 {
		return a.start
	}
	begin, end := a.cuts[0], a.cuts[len(a.cuts)-1]
	if end <= begin {
		return a.start
	}
	t := (x - begin) / (end - begin)
	return a.start + int(math.Round(math.Max(0, math.Min(1, t))*float64(a.span)))
}
func windSortedCuts(vals []float64) []float64 {
	out := make([]float64, 0, len(vals))
	for _, x := range vals {
		if math.IsNaN(x) || math.IsInf(x, 0) {
			continue
		}
		has := false
		for _, y := range out {
			if math.Abs(x-y) < 1e-8 {
				has = true
				break
			}
		}
		if !has {
			out = append(out, x)
		}
	}
	sort.Float64s(out)
	return out
}

// Coordinates on the screen: +X to the right, +Y down. Wind flows from
// left to right along the ridge (RIDGE), and from bottom to top across it
// (CROSS). Normative roof X is always measured from the windward eave;
// therefore CROSS roof/wall Y coordinates MUST be reversed for display.
type windDrawingRect struct {
	code           string
	x0, x1, y0, y1 float64
}

func windDrawingRects(v WindVariant, roof bool, alt string) ([]windDrawingRect, []float64, []float64) {
	out := []windDrawingRect{}
	xEnd, yEnd := v.AlongM, v.AcrossM
	if v.ID == "CROSS" {
		xEnd, yEnd = v.AcrossM, v.AlongM
	}
	xs, ys := []float64{0, xEnd}, []float64{0, yEnd}
	for _, z := range v.Zones {
		if roof {
			if z.Surface != "roof" || z.Alternative != alt {
				continue
			}
			for _, r := range z.Rectangles {
				rr := windDrawingRect{code: z.Code, x0: r.X0, x1: r.X1, y0: r.Y0, y1: r.Y1}
				if v.ID == "CROSS" {
					rr.x0, rr.x1 = r.Y0, r.Y1
					rr.y0, rr.y1 = v.AlongM-r.X1, v.AlongM-r.X0
				}
				out = append(out, rr)
				xs = append(xs, rr.x0, rr.x1)
				ys = append(ys, rr.y0, rr.y1)
			}
		} else if z.Surface != "roof" && z.Code != "D" && z.Code != "E" {
			// Side-wall zones are LINE SEGMENTS on the plan boundaries,
			// not filled rectangles across the plan of the building.
			rr := windDrawingRect{code: z.Code}
			if v.ID == "RIDGE" {
				rr.x0, rr.x1 = z.From, z.To
				rr.y0, rr.y1 = 0, yEnd
				xs = append(xs, z.From, z.To)
			}
			if v.ID == "CROSS" {
				rr.x0, rr.x1 = 0, xEnd
				rr.y0, rr.y1 = v.AlongM-z.To, v.AlongM-z.From
				ys = append(ys, rr.y0, rr.y1)
			}
			out = append(out, rr)
		}
	}
	return out, windSortedCuts(xs), windSortedCuts(ys)
}

func windReportPlanTitle(v WindVariant) string { return "План здания. " + v.Name }

func windReportDiagram(v WindVariant, roof bool, alt string) ([]byte, error) {
	if v.ID == "W" {
		return windReportWallDiagram(v)
	}
	f := newSnowFigure()
	black := color.RGBA{17, 27, 38, 255}
	dim := color.RGBA{100, 107, 113, 255}
	// Larger vertical drawing height makes every roof strip legible.
	x0, y0, xlen, ylen := 195, 157, 945, 205
	boxes, xCuts, yCuts := windDrawingRects(v, roof, alt)
	ax, ay := windAxis(xCuts, x0, xlen), windAxis(yCuts, y0, ylen)
	f.line(x0, y0, x0+xlen, y0, black)
	f.line(x0+xlen, y0, x0+xlen, y0+ylen, black)
	f.line(x0+xlen, y0+ylen, x0, y0+ylen, black)
	f.line(x0, y0+ylen, x0, y0, black)
	if roof {
		for _, r := range boxes {
			x1, x2 := ax.px(r.x0), ax.px(r.x1)
			y1, y2 := ay.px(r.y0), ay.px(r.y1)
			for _, e := range [][4]int{{x1, y1, x2, y1}, {x2, y1, x2, y2}, {x2, y2, x1, y2}, {x1, y2, x1, y1}} {
				f.line(e[0], e[1], e[2], e[3], dim)
			}
			// Short zones can use labels just outside the patch, never erased.
			if x2-x1 > 25 && y2-y1 > 24 {
				lx := (x1+x2)/2 - snowCaptionWidth(r.code, 4)/2
				ly := (y1+y2)/2 - 14
				f.text(lx, ly, 4, r.code, black)
			}
		}
	} else {
		for _, r := range boxes {
			a, b := ax.px(r.x0), ax.px(r.x1)
			c, d := ay.px(r.y0), ay.px(r.y1)
			if v.ID == "RIDGE" {
				// Both long side-wall lines carry segments A/B/C.
				f.line(a, y0, b, y0, black)
				f.line(a, y0+ylen, b, y0+ylen, black)
				f.line(a, y0-6, a, y0+6, dim)
				f.line(b, y0-6, b, y0+6, dim)
				f.line(a, y0+ylen-6, a, y0+ylen+6, dim)
				f.line(b, y0+ylen-6, b, y0+ylen+6, dim)
				if b-a > 22 {
					f.text((a+b)/2-11, y0+10, 4, r.code, black)
					f.text((a+b)/2-11, y0+ylen-49, 4, r.code, black)
				}
			} else {
				f.line(x0, c, x0, d, black)
				f.line(x0+xlen, c, x0+xlen, d, black)
				for _, yy := range []int{c, d} {
					f.line(x0-6, yy, x0+6, yy, dim)
					f.line(x0+xlen-6, yy, x0+xlen+6, yy, dim)
				}
				if d-c > 22 {
					f.text(x0+13, (c+d)/2-15, 4, r.code, black)
					f.text(x0+xlen-39, (c+d)/2-15, 4, r.code, black)
				}
			}
		}
	}
	// End walls are explicitly identified on the wall plan, without converting
	// the side-wall zones A/B/C into plan areas. In CROSS the windward face
	// (D) is at the bottom of the plan; in RIDGE it is at the left.
	if !roof {
		if v.ID == "RIDGE" {
			f.text(x0+15, y0+55, 4, "D", black)
			f.text(x0+xlen-38, y0+55, 4, "E", black)
		} else {
			f.text(x0+xlen-110, y0+12, 4, "E", black)
			f.text(x0+xlen-110, y0+ylen-52, 4, "D", black)
		}
	}
	// Structural reference: an unmistakable solid ridge across the WHOLE plan.
	hasRidge := roof
	for _, z := range v.Zones {
		if z.Surface == "roof" {
			hasRidge = true
			break
		}
	}
	if hasRidge {
		for off := -1; off <= 1; off++ {
			f.line(x0, y0+ylen/2+off, x0+xlen, y0+ylen/2+off, black)
		}
		// Place the caption near the right-hand end, away from central zone letters.
		labelX := x0 + xlen - 145
		if !roof {
			labelX = x0 + xlen/2 - 55
		}
		f.text(labelX, y0+ylen/2-51, 4, "Конёк", black)
	}
	// The word and arrow are one visual group in each orientation.
	if v.ID == "RIDGE" {
		cy := y0 + ylen/2
		f.text(13, cy-38, 4, "Ветер", black)
		// Keep the arrow away from the end-wall letters and contour.
		f.line(74, cy, 113, cy, black)
		f.line(97, cy-9, 113, cy, black)
		f.line(97, cy+9, 113, cy, black)
	} else {
		cx := x0 + xlen/2
		tip, tail := y0+ylen+43, y0+ylen+84
		f.line(cx, tail, cx, tip, black)
		f.line(cx-10, tip+15, cx, tip, black)
		f.line(cx+10, tip+15, cx, tip, black)
		f.text(cx+23, tail-38, 4, "Ветер", black)
	}
	// Fully enumerated dimension chains. Every nonzero interval gets its
	// actual value even if the zone is too narrow to hold an in-place label.
	for k := 0; k < len(xCuts)-1; k++ {
		left, right := ax.px(xCuts[k]), ax.px(xCuts[k+1])
		y := y0 - 17
		f.line(left, y, right, y, dim)
		f.line(left, y-7, left, y+7, dim)
		f.line(right, y-7, right, y+7, dim)
		label := decimalInput(xCuts[k+1]-xCuts[k], 2) + " м"
		tx := (left+right)/2 - snowCaptionWidth(label, 4)/2
		ty := y - 62
		if right-left < 105 {
			// Move narrow labels above and tether to the actual strip.
			ty = y - 105 - 23*(k%2)
			tx = left
			f.line((left+right)/2, y-2, (left+right)/2, ty+43, dim)
		}
		f.text(tx, ty, 4, label, black)
	}
	// Numeric vertical dimensions on the right, including the narrow strips.
	// Distribute captions slightly if rows are closer than the text height;
	// a leader keeps each caption tied to the correct dimension segment.
	lastY := y0 - 25
	for k := 0; k < len(yCuts)-1; k++ {
		top, bottom := ay.px(yCuts[k]), ay.px(yCuts[k+1])
		x := x0 + xlen + 35
		f.line(x, top, x, bottom, dim)
		f.line(x-7, top, x+7, top, dim)
		f.line(x-7, bottom, x+7, bottom, dim)
		label := decimalInput(yCuts[k+1]-yCuts[k], 2) + " м"
		mid := (top + bottom) / 2
		labelY := mid - 18
		if labelY < lastY+29 {
			labelY = lastY + 29
		}
		if labelY > 407 {
			labelY = 407
		}
		f.line(x+2, mid, x+15, labelY+17, dim)
		f.text(x+23, labelY, 4, label, black)
		lastY = labelY
	}
	return f.png()
}
func windReportWallDiagram(v WindVariant) ([]byte, error) {
	f := newSnowFigure()
	black := snowInk
	dim := snowDim
	x0, x1, y := 160, 1200, 250
	f.text(100, 38, 6, "Отдельно стоящая стена. Разбивка на зоны", black)
	f.line(x0, y, x1, y, black)
	ax := windAxis(func() []float64 {
		a := []float64{0, v.AlongM}
		for _, z := range v.Zones {
			a = append(a, z.From, z.To)
		}
		return a
	}(), x0, x1-x0)
	for k, z := range v.Zones {
		from, to := ax.px(z.From), ax.px(z.To)
		f.line(from, y-30, from, y+30, dim)
		f.line(to, y-30, to, y+30, dim)
		f.text((from+to)/2-15, y-80, 5, z.Code, black)
		f.line(from, y+42, to, y+42, dim)
		f.line(from, y+36, from, y+50, dim)
		f.line(to, y+36, to, y+50, dim)
		label := decimalInput(z.To-z.From, 2) + " м"
		if to-from > 115 {
			f.text((from+to)/2-44, y+57, 4, label, black)
		} else {
			f.line((from+to)/2, y+42, (from+to)/2, y+98+25*(k%2), dim)
			f.text(from, y+102+25*(k%2), 4, label, black)
		}
	}
	// The wind must be normal to the free-standing wall, not along its length.
	// The arrow is below the wall and points upwards towards it.
	f.line(82, 395, 82, 323, black)
	f.line(70, 340, 82, 323, black)
	f.line(94, 340, 82, 323, black)
	f.text(102, 352, 4, "Ветер", black)
	return f.png()
}

var _ = fmt.Sprintf
