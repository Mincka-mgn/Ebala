package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
)

type cranePicture struct {
	w, h   int
	pixels []byte
}

func newCranePicture(w, h int) *cranePicture {
	p := &cranePicture{w: w, h: h, pixels: make([]byte, w*h*3)}
	for i := range p.pixels {
		p.pixels[i] = 255
	}
	return p
}
func (p *cranePicture) dot(x, y int, r, g, b byte) {
	if x < 0 || x >= p.w || y < 0 || y >= p.h {
		return
	}
	i := (y*p.w + x) * 3
	p.pixels[i], p.pixels[i+1], p.pixels[i+2] = b, g, r
}
func (p *cranePicture) line(x0, y0, x1, y1 int, r, g, b byte) {
	dx, dy := int(math.Abs(float64(x1-x0))), int(math.Abs(float64(y1-y0)))
	sx, sy := -1, -1
	if x0 < x1 {
		sx = 1
	}
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy
	for {
		p.dot(x0, y0, r, g, b)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}
func (p *cranePicture) string(x, y int, s string, r, g, b byte) {
	for _, ch := range strings.ToUpper(s) {
		glyph, ok := craneGlyphs[ch]
		if !ok {
			glyph = craneGlyphs[' ']
		}
		for j, row := range glyph {
			for i := 0; i < 5; i++ {
				if row&(1<<(4-i)) != 0 {
					p.dot(x+i, y+j, r, g, b)
				}
			}
		}
		x += 6
	}
}

var craneGlyphs = map[rune][7]byte{
	'0': {14, 17, 19, 21, 25, 17, 14}, '1': {4, 12, 4, 4, 4, 4, 14}, '2': {14, 17, 1, 2, 4, 8, 31}, '3': {30, 1, 1, 14, 1, 1, 30},
	'4': {2, 6, 10, 18, 31, 2, 2}, '5': {31, 16, 30, 1, 1, 17, 14}, '6': {6, 8, 16, 30, 17, 17, 14}, '7': {31, 1, 2, 4, 8, 8, 8},
	'8': {14, 17, 17, 14, 17, 17, 14}, '9': {14, 17, 17, 15, 1, 2, 12}, 'P': {30, 17, 17, 30, 16, 16, 16}, 'K': {17, 18, 20, 24, 20, 18, 17},
	'N': {17, 25, 21, 19, 17, 17, 17}, 'M': {17, 27, 21, 21, 17, 17, 17}, 'L': {16, 16, 16, 16, 16, 16, 31},
	'.': {0, 0, 0, 0, 0, 6, 6}, ',': {0, 0, 0, 0, 6, 6, 4}, '-': {0, 0, 0, 31, 0, 0, 0}, '+': {0, 4, 4, 31, 4, 4, 0},
	'=': {0, 0, 31, 0, 31, 0, 0}, ' ': {0, 0, 0, 0, 0, 0, 0}, '(': {2, 4, 8, 8, 8, 4, 2}, ')': {8, 4, 2, 2, 2, 4, 8},
	':': {0, 4, 4, 0, 4, 4, 0}, '/': {1, 2, 2, 4, 8, 8, 16}, 'X': {17, 17, 10, 4, 10, 17, 17},
}

func (p *cranePicture) saveBMP(path string) error {
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	stride := (p.w*3 + 3) &^ 3
	size := 54 + stride*p.h
	head := make([]byte, 54)
	head[0], head[1] = 'B', 'M'
	binary.LittleEndian.PutUint32(head[2:], uint32(size))
	binary.LittleEndian.PutUint32(head[10:], 54)
	binary.LittleEndian.PutUint32(head[14:], 40)
	binary.LittleEndian.PutUint32(head[18:], uint32(p.w))
	binary.LittleEndian.PutUint32(head[22:], uint32(p.h))
	binary.LittleEndian.PutUint16(head[26:], 1)
	binary.LittleEndian.PutUint16(head[28:], 24)
	binary.LittleEndian.PutUint32(head[34:], uint32(stride*p.h))
	if _, e = f.Write(head); e != nil {
		return e
	}
	pad := make([]byte, stride-p.w*3)
	for y := p.h - 1; y >= 0; y-- {
		if _, e = f.Write(p.pixels[y*p.w*3 : (y+1)*p.w*3]); e != nil {
			return e
		}
		if _, e = f.Write(pad); e != nil {
			return e
		}
	}
	return nil
}

// Arrows are scaled to normative wheel loads, with a minimum visible length.
func cranePictureForCase(spans []float64, c CranePlacementCase, path string) error {
	const w, h = 1040, 300
	const x0, x1 = 66, 974
	p := newCranePicture(w, h)
	total := 0.0
	for _, span := range spans {
		total += span
	}
	if total <= 0 {
		return fmt.Errorf("не задана длина балки")
	}
	minX, maxX := 0.0, total
	for _, pt := range c.Points {
		if pt.XMM < minX {
			minX = pt.XMM
		}
		if pt.XMM > maxX {
			maxX = pt.XMM
		}
	}
	if maxX-minX < 1 {
		maxX = minX + 1
	}
	xx := func(x float64) int { return x0 + int((x-minX)/(maxX-minX)*float64(x1-x0)+0.5) }
	beamY := 169
	beamLeft, beamRight := xx(0), xx(total)
	p.line(beamLeft, beamY, beamRight, beamY, 25, 43, 61)
	p.line(beamLeft, beamY+1, beamRight, beamY+1, 25, 43, 61)
	pos := 0.0
	for i := 0; i <= len(spans); i++ {
		x := xx(pos)
		p.line(x, beamY+5, x-9, beamY+25, 61, 80, 96)
		p.line(x, beamY+5, x+9, beamY+25, 61, 80, 96)
		p.line(x-12, beamY+26, x+12, beamY+26, 61, 80, 96)
		p.string(x-3, beamY+34, fmt.Sprint(i+1), 35, 55, 75)
		if i < len(spans) {
			start := pos
			pos += spans[i]
			mx := xx((start + pos) / 2)
			lab := fmt.Sprintf("L=%g MM", spans[i])
			p.string(mx-len(lab)*3, 282, lab, 55, 65, 78)
		}
	}
	// Above: contiguous chain dimensions support-to-wheel-to-wheel-to-support.
	chain := []float64{0, total}
	for _, pt := range c.Points {
		chain = append(chain, pt.XMM)
	}
	sort.Float64s(chain)
	uniq := chain[:0]
	for _, v := range chain {
		if len(uniq) == 0 || math.Abs(v-uniq[len(uniq)-1]) > 0.01 {
			uniq = append(uniq, v)
		}
	}
	p.line(xx(uniq[0]), 80, xx(uniq[len(uniq)-1]), 80, 93, 102, 113)
	for i, v := range uniq {
		x := xx(v)
		p.line(x, 74, x, 87, 93, 102, 113)
		if i > 0 {
			prev := uniq[i-1]
			label := fmt.Sprintf("%g", math.Round((v-prev)*10)/10)
			xm := (xx(prev)+x)/2 - len(label)*3
			// Alternate compact neighbouring dimensions to reduce collisions.
			textY := 64
			if x-xx(prev) < len(label)*6+6 && i%2 == 0 {
				textY = 88
			}
			p.string(xm, textY, label, 69, 78, 90)
		}
	}
	// Scale arrow lengths by wheel load while keeping small forces clearly visible.
	maxForce := 0.0
	for _, pt := range c.Points {
		if pt.PKN > maxForce {
			maxForce = pt.PKN
		}
	}
	for _, pt := range c.Points {
		x := xx(pt.XMM)
		arrowHeight := 28.0
		if maxForce > 0 {
			arrowHeight += 42 * pt.PKN / maxForce
		}
		startY := beamY - 4 - int(math.Round(arrowHeight))
		p.line(x, startY, x, beamY-5, 18, 94, 174)
		p.line(x-5, beamY-13, x, beamY-4, 18, 94, 174)
		p.line(x+5, beamY-13, x, beamY-4, 18, 94, 174)
	}
	// Text labels with multiple lanes; values have an explicit shared unit.
	p.string(7, 8, "P, KN", 20, 67, 122)
	lastRight := [5]int{-500, -500, -500, -500, -500}
	for _, pt := range c.Points {
		x := xx(pt.XMM)
		label := fmt.Sprintf("%g", math.Round(pt.PKN*100)/100)
		left := x - len(label)*3
		if left < 0 {
			left = 0
		}
		if left+len(label)*6 > w {
			left = w - len(label)*6
		}
		row := 0
		for row < len(lastRight)-1 && left < lastRight[row]+6 {
			row++
		}
		lastRight[row] = left + len(label)*6
		p.string(left, 8+row*11, label, 20, 67, 122)
	}
	// Below: one independent ordinate from the left support for EACH wheel.
	lastBottom := [4]int{-500, -500, -500, -500}
	for _, pt := range c.Points {
		x := xx(pt.XMM)
		p.line(x, beamY+3, x, 238, 180, 186, 194)
		p.line(x-3, 235, x+3, 241, 116, 122, 133)
		p.line(x-3, 241, x+3, 235, 116, 122, 133)
		label := fmt.Sprintf("%g", math.Round(pt.XMM*10)/10)
		left := x - len(label)*3
		if left < 0 {
			left = 0
		}
		if left+len(label)*6 > w {
			left = w - len(label)*6
		}
		row := 0
		for row < len(lastBottom)-1 && left < lastBottom[row]+5 {
			row++
		}
		lastBottom[row] = left + len(label)*6
		p.string(left, 247+row*11, label, 60, 67, 79)
	}
	p.string(7, 272, "X, MM", 60, 67, 79)
	// End-support stations are always shown, including if all wheels are inside.
	p.string(beamLeft-3, 215, "0", 60, 67, 79)
	end := fmt.Sprintf("%g MM", total)
	p.string(beamRight-len(end)*6, 215, end, 60, 67, 79)
	return p.saveBMP(path)
}
