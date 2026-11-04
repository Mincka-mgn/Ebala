package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"sync"
)

// Pre-rasterized antialiased numerical lettering. This is only an image atlas,
// not a font file or a runtime font dependency. It works without installed fonts.
const snowAtlasChars = "0123456789,./()- "
const snowAtlasCellWidth = 32
const snowAtlasCellHeight = 58

// The atlas is embedded in the EXE to make Word figures independent of the
// current directory and of resource installation location.
//
//go:embed Resources/SnowRoofs/number_glyphs.png
var snowAtlasBytes []byte

var snowAtlasOnce sync.Once
var snowAtlas image.Image

func snowNumericAtlas() image.Image {
	snowAtlasOnce.Do(func() {
		img, err := png.Decode(bytes.NewReader(snowAtlasBytes))
		if err == nil && img.Bounds().Dx() == len(snowAtlasChars)*snowAtlasCellWidth {
			snowAtlas = img
		}
	})
	return snowAtlas
}
func snowSmoothAdvance(r rune) int {
	switch r {
	case ',', '.':
		return 14
	case '/', '(':
		return 21
	case ')':
		return 19
	case ' ':
		return 13
	case '-':
		return 24
	default:
		return 28
	}
}
func snowSmoothTextWidth(s string) int {
	w := 0
	for _, r := range s {
		w += snowSmoothAdvance(r)
	}
	return w
}
func snowDrawSmoothText(dst *image.RGBA, x, y int, s string, col color.RGBA) bool {
	atlas := snowNumericAtlas()
	if atlas == nil {
		return false
	}
	for _, r := range s {
		found := false
		for _, c := range snowAtlasChars {
			if r == c {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	for _, r := range s {
		idx := -1
		for i, c := range snowAtlasChars {
			if c == r {
				idx = i
				break
			}
		}
		if idx < 0 {
			continue
		}
		for dy := 0; dy < snowAtlasCellHeight; dy++ {
			for dx := 0; dx < snowAtlasCellWidth; dx++ {
				px, py := x+dx, y+dy
				if !image.Pt(px, py).In(dst.Bounds()) {
					continue
				}
				_, _, _, a16 := atlas.At(idx*snowAtlasCellWidth+dx, dy).RGBA()
				a := uint32(a16 >> 8)
				if a == 0 {
					continue
				}
				old := dst.RGBAAt(px, py)
				inv := 255 - a
				dst.SetRGBA(px, py, color.RGBA{
					R: uint8((uint32(col.R)*a + uint32(old.R)*inv) / 255),
					G: uint8((uint32(col.G)*a + uint32(old.G)*inv) / 255),
					B: uint8((uint32(col.B)*a + uint32(old.B)*inv) / 255), A: 255,
				})
			}
		}
		x += snowSmoothAdvance(r)
	}
	return true
}

// Unit captions are pre-rasterized and embedded like numerical glyphs.
// Word/Excel use kPa irrespective of the units selected for the GUI.
//
//go:embed Resources/SnowRoofs/diagram_units_kpa.png
var snowUnitsKpaPNG []byte

//go:embed Resources/SnowRoofs/diagram_units_kgf.png
var snowUnitsKgfPNG []byte

func snowDrawUnitCaption(dst *image.RGBA, x, y int, kgf bool) error {
	data := snowUnitsKpaPNG
	if kgf {
		data = snowUnitsKgfPNG
	}
	im, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	r := image.Rect(x, y, x+im.Bounds().Dx(), y+im.Bounds().Dy()).Intersect(dst.Bounds())
	draw.Draw(dst, r, im, image.Point{X: r.Min.X - x, Y: r.Min.Y - y}, draw.Over)
	return nil
}
