//go:build windows

package main

import (
	"fmt"
	"image"
	"image/color"
	"syscall"
	"unsafe"
)

type snowBIH struct {
	Size                         uint32
	Width, Height                int32
	Planes, BitCount             uint16
	Compression, SizeImage       uint32
	XPelsPerMeter, YPelsPerMeter int32
	ClrUsed, ClrImportant        uint32
}

// Render all caption runs in one 32-bit top-down DIB using the Windows GDI
// TrueType font rasterizer. No font bitmaps or font files are distributed.
func snowRenderPlatformText(dst *image.RGBA, texts []snowTextCmd) error {
	if len(texts) == 0 {
		return nil
	}
	dll := syscall.NewLazyDLL("gdi32.dll")
	dcProc := dll.NewProc("CreateCompatibleDC")
	dibProc := dll.NewProc("CreateDIBSection")
	sel := dll.NewProc("SelectObject")
	bk := dll.NewProc("SetBkMode")
	tc := dll.NewProc("SetTextColor")
	out := dll.NewProc("TextOutW")
	fontProc := dll.NewProc("CreateFontW")
	delObj := dll.NewProc("DeleteObject")
	delDC := dll.NewProc("DeleteDC")
	dc, _, _ := dcProc.Call(0)
	if dc == 0 {
		return fmt.Errorf("GDI: CreateCompatibleDC failed")
	}
	defer delDC.Call(dc)
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	head := snowBIH{Size: 40, Width: int32(w), Height: -int32(h), Planes: 1, BitCount: 32}
	var addr uintptr
	bmp, _, _ := dibProc.Call(dc, uintptr(unsafe.Pointer(&head)), 0, uintptr(unsafe.Pointer(&addr)), 0, 0)
	if bmp == 0 || addr == 0 {
		return fmt.Errorf("GDI: CreateDIBSection failed")
	}
	defer delObj.Call(bmp)
	old, _, _ := sel.Call(dc, bmp)
	defer sel.Call(dc, old)
	px := unsafe.Slice((*byte)(unsafe.Pointer(addr)), w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := dst.RGBAAt(x, y)
			i := (y*w + x) * 4
			px[i], px[i+1], px[i+2], px[i+3] = c.B, c.G, c.R, 255
		}
	}
	bk.Call(dc, 1)
	name, _ := syscall.UTF16PtrFromString("Segoe UI")
	fonts := map[int]uintptr{}
	defer func() {
		for _, f := range fonts {
			delObj.Call(f)
		}
	}()
	for _, t := range texts {
		if t.Text == "" {
			continue
		}
		sz := snowPointSize(t.Size)
		font := fonts[sz]
		if font == 0 {
			font, _, _ = fontProc.Call(uintptr(int32(-sz)), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 4, 0, uintptr(unsafe.Pointer(name)))
			if font == 0 {
				return fmt.Errorf("GDI: CreateFontW failed")
			}
			fonts[sz] = font
		}
		sel.Call(dc, font)
		col := uintptr(t.Ink.R) | uintptr(t.Ink.G)<<8 | uintptr(t.Ink.B)<<16
		tc.Call(dc, col)
		str, err := syscall.UTF16FromString(t.Text)
		if err != nil {
			return err
		}
		out.Call(dc, uintptr(t.X), uintptr(t.Y), uintptr(unsafe.Pointer(&str[0])), uintptr(len(str)-1))
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			dst.SetRGBA(x, y, color.RGBA{px[i+2], px[i+1], px[i], 255})
		}
	}
	return nil
}
