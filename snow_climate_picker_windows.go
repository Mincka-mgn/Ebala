//go:build windows

package main

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

const (
	climatePickSearch  = 15101
	climatePickList    = 15102
	climatePickAccept  = 15103
	climatePickCancel  = 15104
	climatePickCounter = 15105
)

var (
	climatePickerHwnd       uintptr
	climatePickerRegistered bool
	climatePickerProc       uintptr
	climatePickerControls   = map[int]uintptr{}
	climatePickerRows       []SnowClimateRecord
	climatePickerRegion     string
	climatePickerOrigin     string
)

func climatePickerControl(class, caption string, style uintptr, x, y, w, h, id int) uintptr {
	v, _, _ := pCreate.Call(0, uintptr(unsafe.Pointer(ptr(class))), uintptr(unsafe.Pointer(ptr(caption))), style,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h), climatePickerHwnd, uintptr(id), 0, 0)
	if v != 0 && font != 0 {
		pSend.Call(v, WM_SETFONT, font, 1)
	}
	if id != 0 {
		climatePickerControls[id] = v
	}
	return v
}
func climatePickerGet(id int) string {
	h := climatePickerControls[id]
	if h == 0 {
		return ""
	}
	n, _, _ := pGetTextLen.Call(h)
	b := make([]uint16, n+2)
	pGetText.Call(h, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	return syscall.UTF16ToString(b)
}
func climatePickerSet(id int, text string) {
	if h := climatePickerControls[id]; h != 0 {
		pSetText.Call(h, uintptr(unsafe.Pointer(ptr(text))))
	}
}
func climatePickerUpdate() {
	h := climatePickerControls[climatePickList]
	if h == 0 {
		return
	}
	// Region is a preferred starting position, never a hard filter.
	// Search must always cover the entire normative catalogue.
	climatePickerRows = snowClimateFiltered("", climatePickerGet(climatePickSearch))
	pSend.Call(h, LVM_DELETEALLITEMS, 0, 0)
	for i, r := range climatePickerRows {
		item := lvItem{mask: 1, iItem: int32(i), text: ptr(r.Region)}
		pSend.Call(h, LVM_INSERTITEMW, 0, uintptr(unsafe.Pointer(&item)))
		values := []string{r.Locality, decimalInput(r.January, 1), decimalInput(r.Wind, 1)}
		for j, v := range values {
			sub := lvItem{iSubItem: int32(j + 1), text: ptr(v)}
			pSend.Call(h, LVM_SETITEMTEXTW, uintptr(i), uintptr(unsafe.Pointer(&sub)))
		}
	}
	// On first opening, scroll to the nearest row of the region selected
	// in Site, while leaving every other region available for selection.
	if strings.TrimSpace(climatePickerGet(climatePickSearch)) == "" && climatePickerRegion != "" {
		for i, rec := range climatePickerRows {
			if canonicalRegionMatch(climatePickerRegion, rec.Region) {
				state := lvItem{state: 2, stateMask: 2} // LVIS_SELECTED
				pSend.Call(h, LVM_SETITEMSTATE, uintptr(i), uintptr(unsafe.Pointer(&state)))
				pSend.Call(h, 0x1013, uintptr(i), 0) // LVM_ENSUREVISIBLE
				break
			}
		}
	}
	climatePickerSet(climatePickCounter, fmt.Sprintf("Найдено пунктов: %d. Для выбора дважды щёлкните строку или нажмите «Принять».", len(climatePickerRows)))
}
func climatePickerInitTable() {
	initComctl.Call()
	// IMPORTANT: do not use LVS_OWNERDRAWFIXED (0x0400) here.
	// Unlike the main window's custom tables, this dialog has no WM_DRAWITEM
	// handler. The flag produced selectable but invisible rows in Ebala 0.28.
	h := climatePickerControl("SysListView32", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|LVS_REPORT|LVS_SINGLESEL|LVS_SHOWSELALWAYS, 22, 123, 750, 357, climatePickList)
	pSend.Call(h, LVM_SETEXTENDEDLISTVIEWSTYLE, uintptr(lvsExFullRowSelect|lvsExGridLines), uintptr(lvsExFullRowSelect|lvsExGridLines))
	columns := []struct {
		label string
		width int
	}{
		{"Регион", 246}, {"Населённый пункт", 211}, {"T января, °C", 120}, {"Ветер (t ≤ 8 °C), м/с", 151},
	}
	for i, c := range columns {
		col := lvColumn{mask: 0x1 | 0x2 | 0x4, cx: int32(c.width), text: ptr(c.label), subItem: int32(i)}
		pSend.Call(h, LVM_INSERTCOLUMNW, uintptr(i), uintptr(unsafe.Pointer(&col)))
	}
}
func climatePickerClose() {
	if climatePickerHwnd == 0 {
		return
	}
	h := climatePickerHwnd
	climatePickerHwnd = 0
	climatePickerControls = map[int]uintptr{}
	pEnable.Call(hwnd, 1)
	pDestroy.Call(h)
	pSetFocus.Call(hwnd)
}
func climatePickerChoose() {
	h := climatePickerControls[climatePickList]
	idx := int(int32(pSendWindow(h, LVM_GETNEXTITEM, ^uintptr(0), LVNI_SELECTED)))
	climatePickerChooseAt(idx)
}
func climatePickerChooseAt(idx int) {
	if idx < 0 || idx >= len(climatePickerRows) {
		return
	}
	rec := climatePickerRows[idx]
	origin := climatePickerOrigin
	climatePickerClose()
	if origin == "site" {
		// Preserve all other unsaved site values, including snow/wind zones and seismic inputs.
		s, err := readSiteManual()
		if err != nil {
			s = draftSite
		}
		s.SetSnowClimate(rec)
		draftSite = s
		project.SetSite(s)
		set(662, decimalInput(s.ClimateJanuary, 1))
		set(663, decimalInput(s.ClimateWind, 1))
		previewSite(s)
		set(627, siteClimateSummary(s))
		projectHomeOverview()
	} else {
		s, err := snowRead()
		if err != nil {
			s = snowDraft
		}
		s.JanuaryTemp = rec.January
		s.JanuarySet = true
		s.WinterWind = rec.Wind
		s.WinterWindSet = true
		s.ClimateOrigin = "sp131"
		s.ClimateReferencePlace = rec.Locality
		s.ClimateReferenceRegion = rec.Region
		snowDraft = s
		set(snowJanuary, decimalInput(rec.January, 1))
		set(snowWinterWind, decimalInput(rec.Wind, 1))
		project.Site.SetSnowClimate(rec)
		snowCalculate(false)
	}
}

// Use a distinct helper so an absent control never accidentally queries the main window.
func pSendWindow(h uintptr, msg uint32, wp, lp uintptr) uintptr {
	if h == 0 {
		return 0
	}
	v, _, _ := pSend.Call(h, uintptr(msg), wp, lp)
	return v
}
func climatePickerWndProc(w, m, wp, lp uintptr) uintptr {
	switch m {
	case WM_COMMAND:
		id, code := int(wp&0xffff), int((wp>>16)&0xffff)
		switch {
		case id == climatePickSearch && code == EN_CHANGE:
			climatePickerUpdate()
			return 0
		case id == climatePickAccept && code == 0:
			climatePickerChoose()
			return 0
		case id == climatePickCancel && code == 0:
			climatePickerClose()
			return 0
		}
	case WM_NOTIFY:
		if lp != 0 {
			hdr := (*tempNotifyHeader)(unsafe.Pointer(lp))
			if int(hdr.id) == climatePickList && hdr.code == -3 { // NM_DBLCLK
				act := (*tempListActivate)(unsafe.Pointer(lp))
				if act.Item >= 0 { // Ignore double clicks in the empty part of the table.
					climatePickerChooseAt(int(act.Item))
				}
				return 0
			}
		}
	case WM_CLOSE:
		climatePickerClose()
		return 0
	}
	v, _, _ := pDefWndProc.Call(w, m, wp, lp)
	return v
}
func openSnowClimatePicker(origin string) {
	if climatePickerHwnd != 0 {
		return
	}
	if snowClimateError != nil || len(snowClimateCatalog) == 0 {
		message("Не удалось прочитать базу СП 131.13330.2025: "+fmt.Sprint(snowClimateError), true)
		return
	}
	climatePickerOrigin = origin
	region := strings.TrimSpace(project.Site.Region)
	if origin == "site" {
		region = strings.TrimSpace(get(631))
	}
	climatePickerRegion = region
	if !climatePickerRegistered {
		module, _, _ := pGetModule.Call(0)
		climatePickerProc = syscall.NewCallback(climatePickerWndProc)
		wc := wndClassEx{Size: uint32(unsafe.Sizeof(wndClassEx{})), WndProc: climatePickerProc, Instance: module, Background: 6, ClassName: ptr("EbalaSnowClimatePickerV029")}
		ok, _, _ := pRegister.Call(uintptr(unsafe.Pointer(&wc)))
		if ok == 0 {
			message("Не удалось зарегистрировать окно выбора климата", true)
			return
		}
		climatePickerRegistered = true
	}
	module, _, _ := pGetModule.Call(0)
	climatePickerHwnd, _, _ = pCreate.Call(0, uintptr(unsafe.Pointer(ptr("EbalaSnowClimatePickerV029"))), uintptr(unsafe.Pointer(ptr("Температура января и ветер по СП 131.13330.2025"))),
		WS_POPUP_CATALOG|WS_CAPTION_CATALOG|WS_SYSMENU_CATALOG|WS_VISIBLE,
		230, 120, 810, 610, hwnd, 0, module, 0)
	if climatePickerHwnd == 0 {
		message("Не удалось открыть справочник", true)
		return
	}
	climatePickerControls = map[int]uintptr{}
	title := "Все регионы России. Сортировка: сначала регион, затем населённый пункт."
	if climatePickerRegion != "" {
		title = "Переход к региону: " + climatePickerRegion + ". Все остальные регионы также доступны."
	}
	climatePickerControl("STATIC", title, WS_CHILD|WS_VISIBLE, 22, 17, 755, 24, 0)
	climatePickerControl("STATIC", "Поиск по региону или населённому пункту:", WS_CHILD|WS_VISIBLE, 22, 47, 755, 23, 0)
	climatePickerControl("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|ES_AUTOHSCROLL, 22, 75, 750, 30, climatePickSearch)
	climatePickerInitTable()
	climatePickerControl("STATIC", "", WS_CHILD|WS_VISIBLE, 22, 488, 740, 24, climatePickCounter)
	climatePickerControl("BUTTON", "Принять", WS_CHILD|WS_VISIBLE, 22, 520, 365, 40, climatePickAccept)
	climatePickerControl("BUTTON", "Отмена", WS_CHILD|WS_VISIBLE, 406, 520, 366, 40, climatePickCancel)
	climatePickerUpdate()
	pEnable.Call(hwnd, 0)
	pShow.Call(climatePickerHwnd, 1)
	pSetFocus.Call(climatePickerControls[climatePickSearch])
}
