//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var tempLoadImage = u32.NewProc("LoadImageW")
var tempDeleteBitmap = g32.NewProc("DeleteObject")
var tempActiveBitmap uintptr
var tempViewImage = ""

func tempShowTransportImage(kind string) {
	if controls[tempSP35Image] == 0 {
		return
	}
	if kind == "SK" {
		kind = ""
	} // железнодорожная схема на предоставленном рисунке отсутствует
	if kind != "AK" && kind != "NK" && kind != "AB" && kind != "Metro" && kind != "Tram" {
		kind = ""
	}
	if kind == "" {
		pSend.Call(controls[tempSP35Image], 0x172, 0, 0)
		if tempActiveBitmap != 0 {
			tempDeleteBitmap.Call(tempActiveBitmap)
			tempActiveBitmap = 0
		}
		return
	}
	path := filepath.Join(storageRoot(), "Resources", "Transport", kind+".bmp")
	if _, err := os.Stat(path); err != nil {
		return
	}
	b, _, _ := tempLoadImage.Call(0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))), 0, 0, 0, 0x10)
	if b == 0 {
		return
	}
	pSend.Call(controls[tempSP35Image], 0x172, 0, b)
	if tempActiveBitmap != 0 {
		tempDeleteBitmap.Call(tempActiveBitmap)
	}
	tempActiveBitmap = b
	tempViewImage = kind
}
func tempTransportOptionControls(s TempSelection) {
	for _, p := range []struct {
		id  int
		yes bool
	}{{tempSP35Length, transportNeedsLength(s)}, {tempSP35Gamma, transportNeedsABGamma(s)}} {
		if controls[p.id] != 0 {
			pEnable.Call(controls[p.id], boolFlag(p.yes))
		}
	}
}
func tempApplyTransportOptions(s *TempSelection) {
	if s.Params == nil {
		s.Params = map[string]interface{}{}
	}
	if transportNeedsLength(*s) {
		if n, e := parseNumber(strings.TrimSpace(get(tempSP35Length))); e == nil && n > 0 {
			s.Params["load_length_m"] = n
		}
	}
	if transportNeedsABGamma(*s) {
		if n, e := parseNumber(strings.TrimSpace(get(tempSP35Gamma))); e == nil && transportValidateGamma(n) {
			s.Params["gamma_selected"] = n
		}
	}
}
func sp20UpdatePreview() {
	if controls[tempSP20Preview] == 0 {
		return
	}
	n := tempSelected(tempSP20List)
	if n < 0 || n >= len(sp20Filtered) {
		set(tempSP20Preview, "")
		return
	}
	row := sp20Guide.Records[sp20Filtered[n]]
	if row.Editable && row.Norm == 0 {
		set(tempSP20Preview, "По заданию")
		return
	}
	scope, a, floors, e := tempScope()
	if e != nil {
		set(tempSP20Preview, "Введите площадь и число перекрытий")
		return
	}
	s, e := tempSP20Section(row, scope, a, floors)
	if e != nil {
		set(tempSP20Preview, "Проверьте исходные данные")
		return
	}
	set(tempSP20Preview, tempUnitPair(tempNumber(s, "normative_reduced"), row.Unit)+"  × γf = "+tempUnitPair(tempNumber(s, "design_reduced"), row.Unit))
}
