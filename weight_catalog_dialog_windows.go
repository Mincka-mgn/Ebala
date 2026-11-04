//go:build windows

package main

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

const (
	WM_CLOSE_CATALOG   = 0x0010
	WS_POPUP_CATALOG   = 0x80000000
	WS_CAPTION_CATALOG = 0x00C00000
	WS_SYSMENU_CATALOG = 0x00080000
)

var (
	catalogEditorClassRegistered bool
	catalogEditorCallback        uintptr
	catalogEditorWindow          uintptr
	catalogEditorInputs          = map[int]uintptr{}
)

func weightCatalogDialogProc(w, m, wp, lp uintptr) uintptr {
	switch m {
	case WM_COMMAND:
		if wp&0xffff == 1105 {
			weightSaveCatalogEdit()
			return 0
		}
		if wp&0xffff == 1106 {
			weightHideCatalogEdit()
			return 0
		}
		if wp&0xffff == 1107 {
			weightShowGammaReference()
			return 0
		}
	case WM_CLOSE_CATALOG:
		weightHideCatalogEdit()
		return 0
	}
	v, _, _ := pDefWndProc.Call(w, m, wp, lp)
	return v
}
func weightCatalogControl(class, title string, style uintptr, x, y, width, height, id int) uintptr {
	v, _, _ := pCreate.Call(0, uintptr(unsafe.Pointer(ptr(class))), uintptr(unsafe.Pointer(ptr(title))), style,
		uintptr(x), uintptr(y), uintptr(width), uintptr(height), catalogEditorWindow, uintptr(id), 0, 0)
	if v != 0 && font != 0 {
		pSend.Call(v, WM_SETFONT, font, 1)
	}
	if id > 0 {
		catalogEditorInputs[id] = v
	}
	return v
}
func weightCatalogSet(id int, s string) {
	if h := catalogEditorInputs[id]; h != 0 {
		pSetText.Call(h, uintptr(unsafe.Pointer(ptr(s))))
	}
}
func weightCatalogGet(id int) string {
	h := catalogEditorInputs[id]
	if h == 0 {
		return ""
	}
	n, _, _ := pGetTextLen.Call(h)
	buf := make([]uint16, n+2)
	pGetText.Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}
func weightHideCatalogEdit() {
	weightEditingCatalog = -1
	if catalogEditorWindow == 0 {
		return
	}
	old := catalogEditorWindow
	catalogEditorWindow = 0
	catalogEditorInputs = map[int]uintptr{}
	pEnable.Call(hwnd, 1)
	pDestroy.Call(old)
	pSetFocus.Call(hwnd)
}
func weightStartCatalogEdit(m material) {
	if catalogEditorWindow != 0 {
		return
	}
	weightEditingCatalog = -1
	for i, x := range catalogUser.Items {
		if x.ID == m.ID && x.CategoryID == m.CategoryID {
			weightEditingCatalog = i
			break
		}
	}
	if weightEditingCatalog < 0 {
		return
	}
	if !catalogEditorClassRegistered {
		h, _, _ := pGetModule.Call(0)
		catalogEditorCallback = syscall.NewCallback(weightCatalogDialogProc)
		wc := wndClassEx{Size: uint32(unsafe.Sizeof(wndClassEx{})), WndProc: catalogEditorCallback, Instance: h, Background: 6, ClassName: ptr("EbalaCatalogEditorV008")}
		ok, _, _ := pRegister.Call(uintptr(unsafe.Pointer(&wc)))
		if ok == 0 {
			weightEditingCatalog = -1
			message("Не удалось открыть редактор личного материала.", true)
			return
		}
		catalogEditorClassRegistered = true
	}
	h, _, _ := pGetModule.Call(0)
	catalogEditorWindow, _, _ = pCreate.Call(0, uintptr(unsafe.Pointer(ptr("EbalaCatalogEditorV008"))), uintptr(unsafe.Pointer(ptr("Изменить личный материал"))),
		WS_POPUP_CATALOG|WS_CAPTION_CATALOG|WS_SYSMENU_CATALOG|WS_VISIBLE,
		260, 150, 670, 480, hwnd, 0, h, 0)
	if catalogEditorWindow == 0 {
		weightEditingCatalog = -1
		message("Не удалось создать окно редактора.", true)
		return
	}
	catalogEditorInputs = map[int]uintptr{}
	weightCatalogControl("STATIC", "Название материала", WS_CHILD|WS_VISIBLE, 24, 20, 580, 23, 0)
	weightCatalogControl("EDIT", m.Name, WS_CHILD|WS_VISIBLE|WS_BORDER|ES_AUTOHSCROLL, 24, 45, 605, 28, 1101)
	primaryLabel := "Плотность, кг/м³"
	if m.Kind != "layer" {
		primaryLabel = "Нормативный вес, кгс/м²"
	}
	weightCatalogControl("STATIC", primaryLabel, WS_CHILD|WS_VISIBLE, 24, 94, 230, 23, 0)
	weightCatalogControl("STATIC", "Толщина, мм", WS_CHILD|WS_VISIBLE, 271, 94, 195, 23, 0)
	weightCatalogControl("STATIC", "Коэффициент γf", WS_CHILD|WS_VISIBLE, 495, 94, 140, 23, 0)
	weightCatalogControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|ES_AUTOHSCROLL, 24, 120, 220, 28, 1102)
	weightCatalogControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|ES_AUTOHSCROLL, 271, 120, 195, 28, 1103)
	weightCatalogControl("EDIT", decimalInput(m.Gamma, 4), WS_CHILD|WS_VISIBLE|WS_BORDER|ES_AUTOHSCROLL, 495, 120, 134, 28, 1104)
	weightCatalogControl("BUTTON", "Подсказка по γf (СП 20.13330.2016)", WS_CHILD|WS_VISIBLE, 24, 167, 300, 32, 1107)
	weightCatalogControl("STATIC", "Металл: 1,05; тяжёлый бетон / ЖБ / кладка / дерево: 1,10", WS_CHILD|WS_VISIBLE, 24, 218, 620, 23, 0)
	weightCatalogControl("STATIC", "Лёгкие бетоны, изоляция, стяжка, отделка: 1,20 завод / 1,30 площадка", WS_CHILD|WS_VISIBLE, 24, 242, 630, 25, 0)
	weightCatalogControl("STATIC", "Выбор коэффициента проверяйте по фактическому исполнению.", WS_CHILD|WS_VISIBLE, 24, 275, 630, 23, 0)
	weightCatalogControl("STATIC", "Изменения сохраняются только в личном справочнике.", WS_CHILD|WS_VISIBLE, 24, 314, 590, 25, 0)
	weightCatalogControl("BUTTON", "Сохранить", WS_CHILD|WS_VISIBLE, 24, 369, 282, 38, 1105)
	weightCatalogControl("BUTTON", "Отмена", WS_CHILD|WS_VISIBLE, 325, 369, 304, 38, 1106)
	if m.Kind == "layer" {
		if m.Max != nil {
			weightCatalogSet(1102, decimalInput(*m.Max, 1))
		}
		weightCatalogSet(1103, decimalInput(m.Thickness, 1))
	} else {
		weightCatalogSet(1102, decimalInput(m.GNorm, 1))
		pEnable.Call(catalogEditorInputs[1103], 0)
	}
	pEnable.Call(hwnd, 0)
	pShow.Call(catalogEditorWindow, 1)
	pSetFocus.Call(catalogEditorInputs[1101])
}
func weightSaveCatalogEdit() {
	i := weightEditingCatalog
	if i < 0 || i >= len(catalogUser.Items) {
		return
	}
	m := catalogUser.Items[i]
	name := strings.TrimSpace(weightCatalogGet(1101))
	if name == "" {
		pMsgBox.Call(catalogEditorWindow, uintptr(unsafe.Pointer(ptr("Введите название материала."))), uintptr(unsafe.Pointer(ptr("Ошибка"))), MB_ICONERROR)
		return
	}
	value, e := parseNumber(weightCatalogGet(1102))
	if e != nil || value < 0 {
		pMsgBox.Call(catalogEditorWindow, uintptr(unsafe.Pointer(ptr("Проверьте плотность или вес."))), uintptr(unsafe.Pointer(ptr("Ошибка"))), MB_ICONERROR)
		return
	}
	gamma, e := parseNumber(weightCatalogGet(1104))
	if e != nil || gamma <= 0 {
		pMsgBox.Call(catalogEditorWindow, uintptr(unsafe.Pointer(ptr("Проверьте коэффициент надёжности."))), uintptr(unsafe.Pointer(ptr("Ошибка"))), MB_ICONERROR)
		return
	}
	m.Name, m.Gamma = name, gamma
	if m.Kind == "layer" {
		t, e := parseNumber(weightCatalogGet(1103))
		if e != nil || t < 0 {
			pMsgBox.Call(catalogEditorWindow, uintptr(unsafe.Pointer(ptr("Проверьте толщину."))), uintptr(unsafe.Pointer(ptr("Ошибка"))), MB_ICONERROR)
			return
		}
		m.Min, m.Max, m.Thickness = &value, &value, t
	} else {
		m.GNorm = value
	}
	catalogUser.Items[i] = m
	if e := weightSaveCustom(); e != nil {
		text := fmt.Sprintf("Не удалось сохранить материал: %v", e)
		pMsgBox.Call(catalogEditorWindow, uintptr(unsafe.Pointer(ptr(text))), uintptr(unsafe.Pointer(ptr("Ошибка"))), MB_ICONERROR)
		return
	}
	weightHideCatalogEdit()
	weightFillMaterials()
}
