//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var savedDialogWnd uintptr
var savedDialogDone, savedDialogOpen bool
var savedDialogClass = ptr("EbalaReportSavedDialog042")
var savedDialogCallback = syscall.NewCallback(savedDialogProc)

func savedDialogProc(w, m, wp, lp uintptr) uintptr {
	switch m {
	case WM_COMMAND:
		if (wp>>16)&0xffff == 0 {
			switch wp & 0xffff {
			case 1:
				savedDialogOpen = false
				savedDialogDone = true
				pDestroy.Call(w)
				return 0
			case 2:
				savedDialogOpen = true
				savedDialogDone = true
				pDestroy.Call(w)
				return 0
			}
		}
	case nameWMClose:
		savedDialogDone = true
		pDestroy.Call(w)
		return 0
	}
	r, _, _ := pDefWndProc.Call(w, m, wp, lp)
	return r
}
func reportSaved(path string) {
	reportDidSave(path)
	inst, _, _ := pGetModule.Call(0)
	wc := wndClassEx{Size: uint32(unsafe.Sizeof(wndClassEx{})), WndProc: savedDialogCallback, Instance: inst, Background: 6, ClassName: savedDialogClass}
	pRegister.Call(uintptr(unsafe.Pointer(&wc)))
	savedDialogDone = false
	savedDialogOpen = false
	savedDialogWnd, _, _ = pCreate.Call(0, uintptr(unsafe.Pointer(savedDialogClass)), uintptr(unsafe.Pointer(ptr("Файл сохранён"))),
		WS_VISIBLE|nameWSCaption|nameWSSystemMenu|nameWSDialogFrame, 230, 200, 690, 235, hwnd, 0, inst, 0)
	if savedDialogWnd == 0 {
		message("Файл сохранён:\n"+path, false)
		return
	}
	add := func(class, title string, style uintptr, x, y, ww, hh, id int) {
		control, _, _ := pCreate.Call(0, uintptr(unsafe.Pointer(ptr(class))), uintptr(unsafe.Pointer(ptr(title))), style,
			uintptr(x), uintptr(y), uintptr(ww), uintptr(hh), savedDialogWnd, uintptr(id), 0, 0)
		if font != 0 && control != 0 {
			pSend.Call(control, WM_SETFONT, font, 1)
		}
	}
	add("STATIC", "Отчёт успешно сохранён:", WS_CHILD|WS_VISIBLE, 20, 15, 640, 26, 0)
	add("EDIT", path, WS_CHILD|WS_VISIBLE|WS_BORDER|ES_READONLY|ES_AUTOHSCROLL, 20, 49, 640, 29, 0)
	add("BUTTON", "ОК", WS_CHILD|WS_VISIBLE, 306, 111, 125, 38, 1)
	add("BUTTON", "Открыть сохранённый файл", WS_CHILD|WS_VISIBLE, 440, 111, 220, 38, 2)
	pEnable.Call(hwnd, 0)
	pShow.Call(savedDialogWnd, 1)
	var m msg
	for !savedDialogDone {
		result, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(result) <= 0 {
			savedDialogDone = true
			break
		}
		pTranslate.Call(uintptr(unsafe.Pointer(&m)))
		pDispatch.Call(uintptr(unsafe.Pointer(&m)))
	}
	pEnable.Call(hwnd, 1)
	u32.NewProc("SetForegroundWindow").Call(hwnd)
	if savedDialogOpen {
		ret, _, _ := shellExecute.Call(hwnd, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr(path))), 0, 0, 1)
		if ret <= 32 {
			message("Не удалось открыть файл:\n"+path, true)
		}
	}
}
func saveAndNotify(path string, write func(string) error) {
	if e := write(path); e != nil {
		message("Не удалось сохранить файл:\n"+e.Error()+"\nПроверьте, не открыт ли отчёт в Word или Excel.", true)
		return
	}
	reportSaved(path)
}
