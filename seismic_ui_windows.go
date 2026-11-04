//go:build windows

package main

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

const seismicSearchTimerID = 44003

var seismicSetTimer = u32.NewProc("SetTimer")
var seismicKillTimer = u32.NewProc("KillTimer")

func seismicScheduleSearch() {
	seismicKillTimer.Call(hwnd, seismicSearchTimerID)
	seismicSetTimer.Call(hwnd, seismicSearchTimerID, 400, 0)
}
func seismicRunScheduledSearch() {
	seismicKillTimer.Call(hwnd, seismicSearchTimerID)
	if activePage == "seismic" {
		searchSeismicRows()
	}
}

var seismicSearchOldProc uintptr
var seismicSearchCallback uintptr

func seismicSearchEditProc(wnd, msg, wp, lp uintptr) uintptr {
	if msg == 0x0100 && wp == 13 { // Enter immediately commits the current query.
		seismicRunScheduledSearch()
		return 0
	}
	ret, _, _ := pCallWindowProcWeight.Call(seismicSearchOldProc, wnd, msg, wp, lp)
	return ret
}

func seismicAttachSearchEdit() {
	if controls[1003] == 0 || seismicSearchOldProc != 0 {
		return
	}
	seismicSearchCallback = syscall.NewCallback(seismicSearchEditProc)
	seismicSearchOldProc, _, _ = pSetWindowLongPtrWeight.Call(controls[1003], ^uintptr(3), seismicSearchCallback)
}

var seismicRows []SeismicRecord
var seismicSelected = -1
var seismicIntensityOptions = []string{"5 и менее", "6", "7", "8", "9", "10"}
var helpGroup = "В"
var helpRows []SeismicHelpItem

func switchSeismic(target string) {
	if activePage == "site" {
		if s, e := readSiteManual(); e == nil {
			draftSite = s
		}
	}
	if activePage == target {
		return
	}
	for _, h := range pageControls[activePage] {
		pShow.Call(h, 0)
	}
	activePage = target
	for _, h := range pageControls[activePage] {
		pShow.Call(h, 1)
	}
	if target == "seismic" {
		fillSeismicUI()
		searchSeismicRows()
		if draftSite.SeismicRefPlace == "" {
			seismicAutoPick()
		}
	}
	if target == "seismic_help" {
		showHelpGroup(helpGroup)
	}
	if target == "site" {
		fillSiteInputs(draftSite)
		previewSite(draftSite)
	}
	pUpdate.Call(hwnd)
}
func seismicInputs() {
	draftSite.SeismicMap = get(1001)
	draftSite.SeismicRefPlace = strings.TrimSpace(get(1006))
	draftSite.SeismicRefRegion = strings.TrimSpace(get(1019))
	draftSite.SeismicPoints = strings.TrimSpace(get(1007))
	updateSeismicSummary()
}
func updateSeismicSummary() {
	set(1020, "Карта: "+seismicValueOrDash(get(1001))+"     Балльность: "+seismicValueOrDash(get(1007)))
}
func seismicValueOrDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}
func fillSeismicUI() {
	set(1009, "Исходная местность: "+formatSitePlace(draftSite.Region, draftSite.District, draftSite.Place))
	changingUnits = true
	m := draftSite.SeismicMap
	if m == "" {
		m = "А"
	}
	setZone(1001, m)
	setZone(1007, seismicDisplayPoints(draftSite.SeismicPoints))
	set(1006, draftSite.SeismicRefPlace)
	set(1019, draftSite.SeismicRefRegion)
	changingUnits = false
	set(1005, seismicCatalogStatus)
	updateSeismicSummary()
}
func searchSeismicRows() {
	region := draftSite.Region
	term := get(1003)
	// A single filtered scan is sufficient; do not run a full second scan
	// of the large catalog after every keystroke.
	allRows, regionPresent := seismicMatches(region, term)
	total := len(allRows)
	seismicRows = allRows
	if len(seismicRows) > 250 {
		seismicRows = seismicRows[:250]
	}
	send(1004, LB_RESETCONTENT, 0, 0)
	for _, r := range seismicRows {
		title := r.Place + "  |  А: " + seismicDisplayPoints(r.A) + "  В: " + seismicDisplayPoints(r.B) + "  С: " + seismicDisplayPoints(r.C)
		if seismicReviews[normalizeGeo(r.Region)+"|"+normalizeGeo(r.Place)] != "" {
			title = "⚠ " + title
		}
		if !regionPresent || strings.TrimSpace(term) != "" {
			title = r.Region + " · " + title
		}
		pSend.Call(controls[1004], LB_ADDSTRING, 0, uintptr(unsafe.Pointer(ptr(title))))
	}
	seismicSelected = -1
	status := ""
	if regionPresent {
		status = fmt.Sprintf("%s · %d населённых пунктов по фильтру", region, total)
	}
	if !regionPresent {
		status = "Регион в справочнике отсутствует. Найдите населённый пункт в общем списке."
	} else if strings.TrimSpace(term) != "" {
		status = fmt.Sprintf("Поиск по названию региона или населённого пункта: %d записей", total)
	}
	if len(seismicCatalog) < 300 {
		status += " ВНИМАНИЕ: загружена только тестовая часть перечня."
	}
	if total > len(seismicRows) {
		status += fmt.Sprintf(" · показаны первые %d, уточните запрос", len(seismicRows))
	}
	set(1008, status)
}
func seismicAutoPick() {
	if r, ok := seismicAutoMatch(draftSite.Region, draftSite.Place); ok {
		draftSite.SeismicRefPlace = r.Place
		draftSite.SeismicRefRegion = r.Region
		draftSite.SeismicMap = get(1001)
		draftSite.SeismicPoints = seismicDisplayPoints(r.Value(draftSite.SeismicMap))
		set(1006, r.Place)
		set(1019, r.Region)
		setZone(1007, draftSite.SeismicPoints)
		updateSeismicSummary()
		if v := seismicReviews[normalizeGeo(r.Region)+"|"+normalizeGeo(r.Place)]; v != "" {
			set(1018, "Точное совпадение с названием, но данные распознаны из скана: СВЕРЬТЕ балльность с СП 14. Метка "+v)
		} else {
			set(1018, "")
		}
	} else {
		set(1018, "Точного совпадения нет. Выберите населённый пункт-аналог в списке.")
	}
}
func seismicListPick() {
	ix := int(int32(send(1004, LB_GETCURSEL, 0, 0)))
	if ix < 0 || ix >= len(seismicRows) {
		return
	}
	r := seismicRows[ix]
	seismicSelected = ix
	set(1006, r.Place)
	set(1019, r.Region)
	setZone(1007, seismicDisplayPoints(r.Value(get(1001))))
	seismicInputs()
	if v := seismicReviews[normalizeGeo(r.Region)+"|"+normalizeGeo(r.Place)]; v != "" {
		set(1018, "ВНИМАНИЕ: строка из скана, требует сверки с приложением А СП 14. Признак: "+v)
	} else {
		set(1018, "")
	}
}
func seismicMapChanged() {
	if changingUnits {
		return
	}
	seismicInputs()
	if r, ok := seismicAutoMatch(get(1019), get(1006)); ok {
		setZone(1007, seismicDisplayPoints(r.Value(get(1001))))
	}
	seismicInputs()
}
func reloadSeismic() {
	loadSeismicCatalog()
	set(1005, seismicCatalogStatus)
	searchSeismicRows()
	if draftSite.SeismicRefPlace == "" {
		seismicAutoPick()
	}
}
func buildSeismic() {
	buildingPage = "seismic"
	button(1012, "← К местности", 20, 18, 160, 37)
	heading("Сейсмичность · ОСР-2015", 198, 15, 880, 37)
	label("СП 14.13330.2018 с изм. № 2–4, приложение А", 198, 57, 920, 23)
	label("Карта ОСР", 22, 99, 172, 26)
	label("Балльность", 209, 99, 250, 26)
	combo(1001, []string{"А", "В", "С"}, 22, 130, 162)
	combo(1007, seismicIntensityOptions, 210, 130, 190)
	button(1013, "Какую карту выбрать?", 422, 124, 285, 38)
	button(1014, "Обновить справочник", 730, 124, 369, 38)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 22, 174, 1080, 54, 1005)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 22, 231, 1080, 25, 1009)
	label("ПОИСК ПО СП 14: название населённого пункта ИЛИ региона", 22, 270, 1050, 24)
	edit(1003, "", 22, 299, 1080, 32)
	seismicAttachSearchEdit()
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 22, 338, 1080, 44, 1008)
	create("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|LBS_NOTIFY, 22, 381, 1080, 225, 1004)
	label("ВЫБРАНО ДЛЯ РАСЧЁТА И ОТЧЁТА", 22, 615, 1080, 26)
	label("Населённый пункт", 22, 643, 420, 22)
	edit(1006, "", 22, 669, 490, 30)
	label("Регион", 531, 643, 420, 22)
	edit(1019, "", 531, 669, 571, 30)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 22, 705, 1080, 27, 1020)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 22, 738, 1080, 37, 1018)
	button(1011, "Сохранить параметры", 22, 787, 430, 40)
}
func showHelpGroup(m string) {
	helpGroup = m
	set(1042, "КАТЕГОРИИ ОБЪЕКТОВ ДЛЯ КАРТЫ "+m)
	helpRows = seismicHelpItems(m)
	send(1031, LB_RESETCONTENT, 0, 0)
	for _, r := range helpRows {
		t := r.Title
		// The selection itself opens details on the right; no [Подробнее] label.
		pSend.Call(controls[1031], LB_ADDSTRING, 0, uintptr(unsafe.Pointer(ptr(t))))
	}
	set(1041, "Выберите категорию слева.")
	if len(helpRows) > 0 {
		send(1031, LB_SETCURSEL, 0, 0)
		showHelpDetail()
	}
}
func showHelpDetail() {
	ix := int(int32(send(1031, LB_GETCURSEL, 0, 0)))
	if ix < 0 || ix >= len(helpRows) {
		return
	}
	h := helpRows[ix]
	description := h.Title + "\r\n\r\n" + h.Description
	if h.Detail != "" {
		description += "\r\n\r\nУТОЧНЕНИЯ:\r\n" + h.Detail
	}
	description += "\r\n\r\nНОРМАТИВНОЕ ОСНОВАНИЕ:\r\n" + h.Norm
	set(1041, description)
}
func buildSeismicHelp() {
	buildingPage = "seismic_help"
	button(1030, "← К сейсмике", 20, 18, 170, 38)
	heading("Какую карту ОСР-2015 выбрать?", 205, 15, 880, 40)
	label("СП 14.13330.2018 с изменениями № 2, 3, 4. Для обследования существующих зданий.", 23, 60, 1090, 26)
	heading("СНАЧАЛА СМОТРИТЕ ТЕХНИЧЕСКОЕ ЗАДАНИЕ!", 23, 98, 1090, 39)
	label("Если карта указана в ТЗ, используйте её. Если нет — ориентируйтесь на назначение и ответственность объекта.", 23, 144, 1090, 31)
	button(1033, "Карта А · обычные объекты", 23, 196, 343, 42)
	button(1034, "Карта В · ответственные", 381, 196, 344, 42)
	button(1035, "Карта С · особо ответственные", 740, 196, 362, 42)
	label("А — обычные объекты. В — ответственные объекты. С — особо ответственные объекты.", 23, 248, 1080, 35)
	create("STATIC", "КАТЕГОРИИ ОБЪЕКТОВ ДЛЯ КАРТЫ А", WS_CHILD|WS_VISIBLE, 23, 297, 458, 23, 1042)
	label("ПОЯСНЕНИЕ И КОНКРЕТНЫЕ УСЛОВИЯ", 488, 297, 615, 23)
	create("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|LBS_NOTIFY, 23, 331, 457, 408, 1031)
	create("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_READONLY, 488, 331, 614, 408, 1041)
	label("Основание: СП 14.13330.2018 с изменением № 4, пункт 4.3 и таблица 4.2.", 23, 755, 1080, 42)
}
