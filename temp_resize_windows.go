//go:build windows

package main

import "unsafe"

var tempMoveWindow = u32.NewProc("MoveWindow")
var tempLocalWordButtons = map[string]uintptr{}
var tempLocalExcelButtons = map[string]uintptr{}

func tempMoveHandle(handle uintptr, x, y, w, h int) {
	if handle != 0 {
		tempMoveWindow.Call(handle, uintptr(x), uintptr(y), uintptr(w), uintptr(h), 1)
	}
}

var tempGetClientRect = u32.NewProc("GetClientRect")

func tempMove(id, x, y, w, h int) {
	if controls[id] == 0 {
		return
	}
	if w < 40 {
		w = 40
	}
	if h < 28 {
		h = 28
	}
	tempMoveWindow.Call(controls[id], uintptr(x), uintptr(y), uintptr(w), uintptr(h), 1)
}
func tempMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func tempResize() {
	if hwnd == 0 {
		return
	}
	var r tempRect
	tempGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	w, h := int(r.Right), int(r.Bottom)
	if w < 900 || h < 620 {
		return
	}
	wide := w - 50
	weightH := tempMax(220, h-650)
	weightButtonsY := 500 + weightH + 8
	tempMove(903, 20, 500, w-40, weightH)
	tempMove(911, 20, weightButtonsY, 186, 31)
	tempMove(912, 216, weightButtonsY, 185, 31)
	tempMove(913, 415, weightButtonsY, 50, 31)
	tempMove(914, 471, weightButtonsY, 50, 31)
	tempMove(915, 535, weightButtonsY, tempMax(220, w-765), 31)
	tempMove(928, 20, weightButtonsY+35, w-40, 25)
	tempMove(929, 20, weightButtonsY+58, w-40, 23)
	tempMove(955, 20, weightButtonsY+82, 218, 36)
	tempMove(956, 249, weightButtonsY+82, 398, 36)
	tempMove(921, 658, weightButtonsY+82, 219, 36)
	tempMove(922, 888, weightButtonsY+82, 220, 36)
	tempMove(tempSP20Detail, 680, 195, tempMax(320, w-705), 270)
	tempMove(tempSP35Detail, 680, 195, tempMax(320, w-705), 165)
	tempMove(tempSP20Preview, 582, 512, tempMax(300, w-615), 23)
	tempMove(tempSp20Chosen, 25, 630, wide, tempMax(120, h-715))
	tempMove(tempSp35Chosen, 25, 620, wide, tempMax(120, h-715))
	for _, page := range []string{"temp_sp20", "temp_sp35", "temp_cranes"} {
		tempMoveHandle(tempLocalWordButtons[page], w-505, h-64, 215, 42)
		tempMoveHandle(tempLocalExcelButtons[page], w-270, h-64, 245, 42)
	}
	// Keep summary tables under their fixed headers, regardless of window size.
	tempMove(reportCompositionList, 25, 216, wide, tempMax(310, h-360))
	tempMove(tempChosen, 25, 322, wide, 123)
	tempMove(tempTransportChosen, 25, 479, wide, 123)
	tempMove(tempCraneChosen, 25, 639, wide, tempMax(100, h-775))
	// The crane table must never overlap the actions beneath it.
	// Do not expand its height past the controls' fixed layout.
	tempMove(tempCraneLocalChosen, 25, 133, wide, 350)
	tempMove(tempCraneAdd, 25, 508, 215, 39)
	tempMove(tempCraneDuplicate, 258, 508, 194, 39)
	tempMove(tempCraneDelete, 470, 508, 194, 39)
	tempMove(tempCranePick1, w-360, 508, 335, 39)
	tempMove(cranePlaceOpen, w-360, 564, 335, 40)
	tempMove(tempCraneUnit, 355, 558, 180, 220)
	tempMove(tempReportWord, w-505, h-65, 215, 40)
	tempMove(tempReportExcel, w-270, h-65, 245, 40)
	tempMove(tempHome, 25, h-65, 200, 40)
}
