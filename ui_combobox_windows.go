//go:build windows

package main

import "unsafe"

const (
	CBS_DROPDOWNLIST   = 0x0003
	CB_ADDSTRING       = 0x0143
	CB_GETCURSEL       = 0x0147
	CB_SETCURSEL       = 0x014E
	CB_FINDSTRINGEXACT = 0x0158
	CBN_SELCHANGE      = 1
)

func combo(id int, choices []string, x, y, w int) {
	create("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_VSCROLL|CBS_DROPDOWNLIST, x, y, w, 265, id)
	for _, item := range choices {
		pSend.Call(controls[id], CB_ADDSTRING, 0, uintptr(unsafe.Pointer(ptr(item))))
	}
}
func setZone(id int, zone string) {
	if controls[id] == 0 {
		return
	}
	if id == 636 || id == 206 {
		zone = windDisplayZone(zone)
	}
	idx := send(id, CB_FINDSTRINGEXACT, ^uintptr(0), uintptr(unsafe.Pointer(ptr(zone))))
	if int32(idx) < 0 {
		pSend.Call(controls[id], CB_SETCURSEL, ^uintptr(0), 0)
	} else {
		pSend.Call(controls[id], CB_SETCURSEL, idx, 0)
	}
}
