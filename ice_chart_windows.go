//go:build windows

package main

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"unsafe"
)

var iceGdiDeleteObject = g32.NewProc("DeleteObject")

// Display the same numerical chart that goes into Word and Excel.  The control
// is a native Windows SS_BITMAP, no browser, no external dependencies.
func iceDisplayChart(s IceLoadCalc) {
	raw, err := iceFigure(s)
	if err != nil {
		set(iceChartStatus, err.Error())
		return
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return
	}
	w, h := 1040, 245
	scaled := image.NewRGBA(image.Rect(0, 0, w, h))
	src := img.Bounds()
	sw, sh := src.Dx(), src.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			scaled.Set(x, y, img.At(src.Min.X+x*sw/w, src.Min.Y+y*sh/h))
		}
	}
	bmp, err := bitmap24(scaled)
	if err != nil {
		return
	}
	file := filepath.Join(os.TempDir(), "Ebala_ice_load_chart.bmp")
	if err = os.WriteFile(file, bmp, 0600); err != nil {
		return
	}
	hbitmap, _, _ := pLoadImage.Call(0, uintptr(unsafe.Pointer(ptr(file))), 0, 0, 0, 0x10|0x2000)
	if hbitmap == 0 {
		set(iceChartStatus, "Не удалось отобразить эпюру в окне программы")
		return
	}
	pSend.Call(controls[iceChartImage], 0x0172, 0, hbitmap)
	if iceChartBitmap != 0 {
		iceGdiDeleteObject.Call(iceChartBitmap)
	}
	iceChartBitmap = hbitmap
}
