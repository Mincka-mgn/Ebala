//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var nameDialogWnd, nameDialogEdit uintptr
var nameDialogDone, nameDialogAccepted bool
var nameDialogValue string
var nameDialogClass = ptr("EbalaNameDialog042")
var nameDialogCallback = syscall.NewCallback(nameDialogProc)

const (
	nameWMClose       = 0x10
	nameWSCaption     = 0x00C00000
	nameWSSystemMenu  = 0x00080000
	nameWSDialogFrame = 0x00400000
)

func nameDialogProc(wnd, m, wp, lp uintptr) uintptr {
	switch m {
	case WM_COMMAND:
		if (wp>>16)&0xffff == 0 {
			switch wp & 0xffff {
			case 1:
				n, _, _ := pGetTextLen.Call(nameDialogEdit)
				b := make([]uint16, n+2)
				pGetText.Call(nameDialogEdit, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
				nameDialogValue = syscall.UTF16ToString(b)
				nameDialogAccepted = true
				nameDialogDone = true
				pDestroy.Call(wnd)
				return 0
			case 2:
				nameDialogDone = true
				pDestroy.Call(wnd)
				return 0
			}
		}
	case nameWMClose:
		nameDialogDone = true
		pDestroy.Call(wnd)
		return 0
	}
	r, _, _ := pDefWndProc.Call(wnd, m, wp, lp)
	return r
}
func nameDialogControl(class, title string, style uintptr, x, y, w, h, id int) uintptr {
	c, _, _ := pCreate.Call(0, uintptr(unsafe.Pointer(ptr(class))), uintptr(unsafe.Pointer(ptr(title))), style,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h), nameDialogWnd, uintptr(id), 0, 0)
	if font != 0 && c != 0 {
		pSend.Call(c, WM_SETFONT, font, 1)
	}
	return c
}

// Empty user-supplied names are deliberately allowed.
func askReportBlockName(suggest string) (string, bool) {
	var instance, _, _ = pGetModule.Call(0)
	wc := wndClassEx{Size: uint32(unsafe.Sizeof(wndClassEx{})), WndProc: nameDialogCallback, Instance: instance, Background: 6, ClassName: nameDialogClass}
	pRegister.Call(uintptr(unsafe.Pointer(&wc)))
	nameDialogDone = false
	nameDialogAccepted = false
	nameDialogValue = ""
	nameDialogWnd, _, _ = pCreate.Call(0, uintptr(unsafe.Pointer(nameDialogClass)), uintptr(unsafe.Pointer(ptr("Название нагрузки"))),
		WS_VISIBLE|nameWSCaption|nameWSSystemMenu|nameWSDialogFrame, 270, 220, 560, 218, hwnd, 0, instance, 0)
	if nameDialogWnd == 0 {
		return "", false
	}
	nameDialogControl("STATIC", "Под каким именем сохранить текущую нагрузку?", WS_CHILD|WS_VISIBLE, 20, 16, 510, 27, 0)
	nameDialogEdit = nameDialogControl("EDIT", suggest, WS_CHILD|WS_VISIBLE|WS_BORDER|ES_AUTOHSCROLL, 20, 53, 510, 31, 10)
	nameDialogControl("BUTTON", "Отмена", WS_CHILD|WS_VISIBLE, 283, 110, 116, 36, 2)
	nameDialogControl("BUTTON", "ОК", WS_CHILD|WS_VISIBLE, 412, 110, 118, 36, 1)
	pEnable.Call(hwnd, 0)
	pShow.Call(nameDialogWnd, 1)
	u32.NewProc("SetFocus").Call(nameDialogEdit)
	pSend.Call(nameDialogEdit, 0x00B1, 0, ^uintptr(0)) // EM_SETSEL select suggested text
	var m msg
	for !nameDialogDone {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			nameDialogDone = true
			break
		}
		pTranslate.Call(uintptr(unsafe.Pointer(&m)))
		pDispatch.Call(uintptr(unsafe.Pointer(&m)))
	}
	pEnable.Call(hwnd, 1)
	u32.NewProc("SetForegroundWindow").Call(hwnd)
	return nameDialogValue, nameDialogAccepted
}
