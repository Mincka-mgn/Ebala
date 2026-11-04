//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

const (
	ODS_SELECTED = 0x0001
	ETO_OPAQUE   = 0x0002
	ETO_CLIPPED  = 0x0004
)

type winRect struct{ Left, Top, Right, Bottom int32 }
type measureItemStruct struct {
	CtlType, CtlID, ItemID, ItemWidth, ItemHeight uint32
	ItemData                                      uintptr
}
type drawItemStruct struct {
	CtlType, CtlID, ItemID, ItemAction, ItemState uint32
	HwndItem, HDC                                 uintptr
	ItemRect                                      winRect
	ItemData                                      uintptr
}

var pSetTextColor = g32.NewProc("SetTextColor")
var pSetBkColor = g32.NewProc("SetBkColor")
var pExtTextOut = g32.NewProc("ExtTextOutW")

func drawClimateRow(lp uintptr) {
	d := (*drawItemStruct)(unsafe.Pointer(lp))
	idx := int(d.ItemID)
	if idx < 0 || idx >= len(foundPlaces) {
		return
	}
	special := foundPlaces[idx].SnowK > 0
	missing := foundPlaces[idx].IsCapital && !validClimateRecord(foundPlaces[idx])
	fg, bg := uintptr(0x00202020), uintptr(0x00FFFFFF)
	if special {
		fg = 0x000050A0
		bg = 0x00E4F5FF
	} // pale peach BG, dark red-orange foreground (COLORREF BGR)
	if missing {
		fg = 0x00808080
		bg = 0x00F7F7F7
	}
	if d.ItemState&ODS_SELECTED != 0 {
		fg = 0x00FFFFFF
		bg = 0x00A85316
	}
	pSetBkColor.Call(d.HDC, bg)
	pSetTextColor.Call(d.HDC, fg)
	text := recordTitle(foundPlaces[idx])
	u := syscall.StringToUTF16(text)
	r := d.ItemRect
	pExtTextOut.Call(d.HDC, uintptr(r.Left+6), uintptr(r.Top+2), ETO_OPAQUE|ETO_CLIPPED, uintptr(unsafe.Pointer(&r)), uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), 0)
}
