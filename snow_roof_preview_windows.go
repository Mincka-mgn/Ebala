package main

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"os"
	"path/filepath"
	"unsafe"
)

// Windows GDI draws the same author-supplied, static roof scheme that is used
// in Word. The SVG originals and white-background PNG previews are shipped
// in Resources/SnowRoofs. No dynamic roof-outline drawing is involved.
func snowPaintRoofAsset(dc uintptr, left, top, width, height int, shape string, reversed bool) bool {
	s := defaultSnowCalc()
	s.Shape = shape
	if reversed && shape == "mono" {
		s.Angle = -10
	}
	data, err := snowRoofImage(s)
	if err != nil {
		return false
	}
	return snowPaintPNG(dc, left, top, width, height, data)
}

// snowPaintNormativeAsset displays the original full normative figure from
// the supplied SP scan, rather than a substitute plan drawn by Ebala.
func snowPaintNormativeAsset(dc uintptr, left, top, width, height int, shape string) bool {
	name := ""
	switch shape {
	case "dome":
		name = "snow_B14_full.png"
	case "cone":
		name = "snow_B15_full.png"
	case "sphere_cone":
		name = "snow_B15a_full.png"
	}
	if name == "" {
		return false
	}
	for _, path := range []string{filepath.Join(storageRoot(), "Resources", "SnowRoofs", name), filepath.Join("Resources", "SnowRoofs", name)} {
		if b, e := os.ReadFile(path); e == nil {
			return snowPaintPNG(dc, left, top, width, height, b)
		}
	}
	return false
}

func snowPaintPlanAsset(dc uintptr, left, top, width, height int) bool {
	for _, p := range []string{filepath.Join(storageRoot(), "Resources", "SnowRoofs", "snow_B3_lantern_zones_preview.png"), filepath.Join("Resources", "SnowRoofs", "snow_B3_lantern_zones_preview.png")} {
		if b, e := os.ReadFile(p); e == nil {
			return snowPaintPNG(dc, left, top, width, height, b)
		}
	}
	return false
}
func snowPaintPNG(dc uintptr, left, top, width, height int, data []byte) bool {
	im, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return false
	}
	w, h := im.Bounds().Dx(), im.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return false
	}
	bits := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := im.At(x+im.Bounds().Min.X, y+im.Bounds().Min.Y).RGBA()
			// GDI's 32-bit DIB does not perform PNG alpha composition. Composite
			// explicitly against white so transparency never becomes black.
			white := uint32(0xffff) - a
			j := (y*w + x) * 4
			bits[j] = byte((b + white) / 257)
			bits[j+1] = byte((g + white) / 257)
			bits[j+2] = byte((r + white) / 257)
			bits[j+3] = 0
		}
	}
	// BITMAPINFOHEADER, 32-bit top-down DIB: 40 bytes.
	info := make([]byte, 40)
	binary.LittleEndian.PutUint32(info[0:], 40)
	binary.LittleEndian.PutUint32(info[4:], uint32(w))
	binary.LittleEndian.PutUint32(info[8:], uint32(int32(-h)))
	binary.LittleEndian.PutUint16(info[12:], 1)
	binary.LittleEndian.PutUint16(info[14:], 32)
	dstW := width - 10
	dstH := int(float64(dstW) * float64(h) / float64(w))
	if dstH > height-8 {
		dstH = height - 8
		dstW = int(float64(dstH) * float64(w) / float64(h))
	}
	x := left + (width-dstW)/2
	y := top + (height-dstH)/2
	if dstW <= 0 || dstH <= 0 {
		return false
	}
	ret, _, _ := g32.NewProc("StretchDIBits").Call(dc, uintptr(x), uintptr(y), uintptr(dstW), uintptr(dstH), 0, 0, uintptr(w), uintptr(h), uintptr(unsafe.Pointer(&bits[0])), uintptr(unsafe.Pointer(&info[0])), 0, 0x00CC0020)
	return ret != 0
}

// The two-lantern plan retains the B.3 A/B/C zone legend for each
// independently configured roof. Do not show the one-lantern picture
// as if it described both roofs.
func snowPaintLanternPlan(dc uintptr, left, top, width, height int, two bool) bool {
	name := "snow_B3_lantern_zones_preview.png"
	if two {
		name = "snow_B3_two_lantern_zones.png"
	}
	for _, p := range []string{filepath.Join(storageRoot(), "Resources", "SnowRoofs", name), filepath.Join("Resources", "SnowRoofs", name)} {
		if b, e := os.ReadFile(p); e == nil {
			return snowPaintPNG(dc, left, top, width, height, b)
		}
	}
	return false
}
