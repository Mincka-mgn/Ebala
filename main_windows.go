//go:build windows

package main

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	u32          = syscall.NewLazyDLL("user32.dll")
	k32          = syscall.NewLazyDLL("kernel32.dll")
	g32          = syscall.NewLazyDLL("gdi32.dll")
	c32          = syscall.NewLazyDLL("comdlg32.dll")
	pRegister    = u32.NewProc("RegisterClassExW")
	pCreate      = u32.NewProc("CreateWindowExW")
	pDefWndProc  = u32.NewProc("DefWindowProcW")
	pDestroy     = u32.NewProc("DestroyWindow")
	pShow        = u32.NewProc("ShowWindow")
	pEnable      = u32.NewProc("EnableWindow")
	pUpdate      = u32.NewProc("UpdateWindow")
	pGetMessage  = u32.NewProc("GetMessageW")
	pTranslate   = u32.NewProc("TranslateMessage")
	pDispatch    = u32.NewProc("DispatchMessageW")
	pPostQuit    = u32.NewProc("PostQuitMessage")
	pPostMessage = u32.NewProc("PostMessageW")
	pSend        = u32.NewProc("SendMessageW")
	pSetText     = u32.NewProc("SetWindowTextW")
	pGetText     = u32.NewProc("GetWindowTextW")
	pGetTextLen  = u32.NewProc("GetWindowTextLengthW")
	pMsgBox      = u32.NewProc("MessageBoxW")
	pGetModule   = k32.NewProc("GetModuleHandleW")
	pCreateFont  = g32.NewProc("CreateFontW")
	pGetSave     = c32.NewProc("GetSaveFileNameW")
	pGetOpen     = c32.NewProc("GetOpenFileNameW")
)

const ebalaVersion = "0.54"
const ebalaTitle = "Ebala " + ebalaVersion

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_CHILD            = 0x40000000
	WS_VISIBLE          = 0x10000000
	WS_BORDER           = 0x00800000
	WS_VSCROLL          = 0x00200000
	ES_AUTOHSCROLL      = 0x80
	ES_MULTILINE        = 0x0004
	ES_READONLY         = 0x0800
	BS_MULTILINE        = 0x2000
	LBS_NOTIFY          = 0x0001
	LBS_OWNERDRAWFIXED  = 0x0010
	LBS_HASSTRINGS      = 0x0040
	WM_DRAWITEM         = 0x002B
	WM_CONTEXTMENU      = 0x007B
	WM_MEASUREITEM      = 0x002C
	WM_COMMAND          = 0x111
	EN_KILLFOCUS        = 0x0200
	WM_APP_ELEVATION    = 0x8001
	WM_APP_SEISMIC      = 0x8002
	WM_DESTROY          = 2
	WM_SETFONT          = 0x30
	LB_ADDSTRING        = 0x180
	LB_RESETCONTENT     = 0x184
	LB_GETCURSEL        = 0x188
	LB_SETCURSEL        = 0x186
	LB_ERR              = 0xffffffff
	LBN_SELCHANGE       = 1
	EN_CHANGE           = 0x300
	MB_ICONERROR        = 0x10
	MB_ICONINFORMATION  = 0x40
	MB_ICONWARNING      = 0x30
	MB_YESNO            = 0x04
	IDYES               = 6
	OFN_OVERWRITEPROMPT = 0x2
	OFN_PATHMUSTEXIST   = 0x800
	OFN_FILEMUSTEXIST   = 0x1000
	OFN_EXPLORER        = 0x80000
)

type wndClassEx struct {
	Size, Style                        uint32
	WndProc                            uintptr
	ClsExtra, WndExtra                 int32
	Instance, Icon, Cursor, Background uintptr
	MenuName, ClassName                *uint16
	IconSm                             uintptr
}
type point struct{ X, Y int32 }
type msg struct {
	Hwnd           uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             point
	Private        uint32
}
type openFileName struct {
	Size                         uint32
	Owner, Instance              uintptr
	Filter, CustomFilter         *uint16
	MaxCustFilter, FilterIndex   uint32
	File                         *uint16
	MaxFile                      uint32
	FileTitle                    *uint16
	MaxFileTitle                 uint32
	InitialDir, Title            *uint16
	Flags                        uint32
	FileOffset, FileExtension    uint16
	DefaultExt                   *uint16
	CustData, Hook, TemplateName uintptr
	Reserved                     uintptr
	ReservedDword, FlagsEx       uint32
}

var hwnd uintptr
var font, headingFont uintptr
var controls = map[int]uintptr{}
var pageControls = map[string][]uintptr{}
var activePage = "home"
var buildingPage = "home"
var changingUnits bool
var siteReturnPage string
var siteBackButton uintptr

func ptr(s string) *uint16 { return syscall.StringToUTF16Ptr(s) }
func message(s string, isError bool) {
	flag, title := uintptr(MB_ICONINFORMATION), "Ebala"
	if isError {
		flag, title = MB_ICONERROR, "Ошибка"
	}
	pMsgBox.Call(hwnd, uintptr(unsafe.Pointer(ptr(s))), uintptr(unsafe.Pointer(ptr(title))), flag)
}

// Only a completely empty form should ask about a blank report.
// A deliberate text in the elevation field also counts as an input.
func askConfirm(text string) bool {
	r, _, _ := pMsgBox.Call(hwnd, uintptr(unsafe.Pointer(ptr(text))), uintptr(unsafe.Pointer(ptr("Подтверждение"))), MB_YESNO|MB_ICONWARNING)
	return r == IDYES
}

func confirmSiteReport(s Site) bool {
	if siteReportHasInput(s) {
		return true
	}
	q := "В характеристиках местности ничего не заполнено.\nСформировать пустой отчёт?"
	r, _, _ := pMsgBox.Call(hwnd, uintptr(unsafe.Pointer(ptr(q))), uintptr(unsafe.Pointer(ptr("Пустые исходные данные"))), MB_ICONWARNING|MB_YESNO)
	return r == IDYES
}

func create(class, title string, style uintptr, x, y, w, h, id int) uintptr {
	if buildingPage != activePage {
		style &^= WS_VISIBLE
	}
	v, _, _ := pCreate.Call(0, uintptr(unsafe.Pointer(ptr(class))), uintptr(unsafe.Pointer(ptr(title))), style,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h), hwnd, uintptr(id), 0, 0)
	if font != 0 && v != 0 {
		pSend.Call(v, WM_SETFONT, font, 1)
	}
	pageControls[buildingPage] = append(pageControls[buildingPage], v)
	if id > 0 {
		controls[id] = v
	}
	return v
}
func label(s string, x, y, w, h int) { create("STATIC", s, WS_CHILD|WS_VISIBLE, x, y, w, h, 0) }
func heading(s string, x, y, w, h int) {
	v := create("STATIC", s, WS_CHILD|WS_VISIBLE, x, y, w, h, 0)
	if headingFont != 0 {
		pSend.Call(v, WM_SETFONT, headingFont, 1)
	}
}
func edit(id int, s string, x, y, w, h int) {
	create("EDIT", s, WS_CHILD|WS_VISIBLE|WS_BORDER|ES_AUTOHSCROLL, x, y, w, h, id)
}
func button(id int, s string, x, y, w, h int) uintptr {
	return create("BUTTON", s, WS_CHILD|WS_VISIBLE, x, y, w, h, id)
}
func menuButton(id int, s string, x, y int, available bool) {
	v := create("BUTTON", s, WS_CHILD|WS_VISIBLE|BS_MULTILINE, x, y, 525, 89, id)
	if !available {
		pEnable.Call(v, 0)
	}
}
func get(id int) string {
	c := controls[id]
	if c == 0 {
		return ""
	}
	n, _, _ := pGetTextLen.Call(c)
	buf := make([]uint16, n+2)
	pGetText.Call(c, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}
func set(id int, s string) {
	if controls[id] != 0 {
		pSetText.Call(controls[id], uintptr(unsafe.Pointer(ptr(s))))
	}
}
func send(id, m int, w, l uintptr) uintptr {
	r, _, _ := pSend.Call(controls[id], uintptr(m), w, l)
	return r
}
func selected() int {
	v := send(100, LB_GETCURSEL, 0, 0)
	if int32(v) == -1 {
		return -1
	}
	return int(v)
}

// Every editable pressure/weight field has its equivalent permanently visible.
// We update only the OTHER edit control, never the one where the caret is.
func convertEdit(source, target int, factor float64, decimals int) {
	if changingUnits {
		return
	}
	value, err := parseNumber(get(source))
	if err != nil {
		return
	} // permit intermediate states, e.g. "1,"
	changingUnits = true
	set(target, decimalInput(value*factor, decimals))
	changingUnits = false
}
func unitChanged(id int) {
	switch id {
	case 102:
		convertEdit(102, 106, 1000, 3) // m -> mm
	case 106:
		convertEdit(106, 102, 0.001, 8) // mm -> m
	case 103:
		convertEdit(103, 105, kgfPerKN, 3) // kN/m3 -> kgf/m3
	case 105:
		convertEdit(105, 103, 1/kgfPerKN, 8) // kgf/m3 -> kN/m3
	case 201:
		if !changingUnits {
			project.WindOverride = true
			project.WindZoneOverride = "Пользовательский"
			setZone(206, "Пользовательский")
			set(210, "РЕЖИМ: пользовательское давление w0 (локальная правка).")
		}
		convertEdit(201, 205, kgfPerKN, 3) // kPa -> kgf/m2
	case 205:
		if !changingUnits {
			project.WindOverride = true
			project.WindZoneOverride = "Пользовательский"
			setZone(206, "Пользовательский")
			set(210, "РЕЖИМ: пользовательское давление w0 (локальная правка).")
		}
		convertEdit(205, 201, 1/kgfPerKN, 8) // kgf/m2 -> kPa
	}
}
func syncWind() error {
	keys := []int{201, 202, 203, 204}
	vals := make([]float64, len(keys))
	for i, k := range keys {
		v, e := parseNumber(get(k))
		if e != nil {
			return fmt.Errorf("Поле ветра №%d: %w", i+1, e)
		}
		vals[i] = v
	}
	w := Wind{vals[0], vals[1], vals[2], vals[3]}
	check := project
	check.Wind = w
	if _, e := check.Calc(); e != nil {
		return e
	}
	project.Wind = w
	return nil
}
func readLayer() (Layer, error) {
	name := strings.TrimSpace(get(101))
	t, e := parseNumber(get(102))
	if e != nil {
		return Layer{}, fmt.Errorf("Толщина: %w", e)
	}
	g, e := parseNumber(get(103))
	if e != nil {
		return Layer{}, fmt.Errorf("Удельный вес: %w", e)
	}
	f, e := parseNumber(get(104))
	if e != nil {
		return Layer{}, fmt.Errorf("Коэффициент надёжности: %w", e)
	}
	l := Layer{name, t, g, f}
	if name == "" || t < 0 || g < 0 || f < 0 {
		return l, fmt.Errorf("Недопустимые параметры слоя")
	}
	return l, nil
}
func showLayer(idx int) {
	if idx < 0 || idx >= len(project.Layers) {
		return
	}
	l := project.Layers[idx]
	changingUnits = true
	set(101, l.Name)
	set(102, xnum(l.Thickness))
	set(106, decimalInput(l.Thickness*1000, 4))
	set(103, xnum(l.UnitWeight))
	set(105, decimalInput(l.UnitWeight*kgfPerKN, 3))
	set(104, xnum(l.GammaF))
	changingUnits = false
}
func refresh() {
	send(100, LB_RESETCONTENT, 0, 0)
	for i, l := range project.Layers {
		row := fmt.Sprintf("%02d | %s | t=%s м | γ=%s кН/м³ | γf=%s | gн=%s | gр=%s кПа", i+1, l.Name,
			displayNumber(l.Thickness), displayNumber(l.UnitWeight), displayNumber(l.GammaF),
			displayNumber(l.Thickness*l.UnitWeight), displayNumber(l.Thickness*l.UnitWeight*l.GammaF))
		pSend.Call(controls[100], LB_ADDSTRING, 0, uintptr(unsafe.Pointer(ptr(row))))
	}
	changingUnits = true
	for i, v := range []float64{project.Wind.W0, project.Wind.KZ, project.Wind.C, project.Wind.GammaF} {
		set(201+i, xnum(v))
	}
	set(205, decimalInput(project.Wind.W0*kgfPerKN, 3))
	changingUnits = false
	if len(project.Layers) > 0 {
		send(100, LB_SETCURSEL, 0, 0)
		showLayer(0)
	}
	refreshClimateUI()
}
func renderResults() {
	r, e := project.Calc()
	if e != nil {
		return
	}
	set(401, fmt.Sprintf("Нормативная: %s кПа  =  %s кгс/м²", displayNumber(r.SWN), decimalInput(r.SWN*kgfPerKN, 2)))
	set(405, fmt.Sprintf("Расчётная:     %s кПа  =  %s кгс/м²", displayNumber(r.SWD), decimalInput(r.SWD*kgfPerKN, 2)))
	set(402, fmt.Sprintf("Нормативная: %s кПа  =  %s кгс/м²", displayNumber(r.WN), decimalInput(r.WN*kgfPerKN, 2)))
	set(406, fmt.Sprintf("Расчётная:     %s кПа  =  %s кгс/м²", displayNumber(r.WD), decimalInput(r.WD*kgfPerKN, 2)))
	projectHomeOverview()
}
func chooseSave(ext, pattern, initial string, open bool) string {
	buf := make([]uint16, 1024)
	copy(buf, syscall.StringToUTF16(initial))
	filter := append(utf16.Encode([]rune(pattern+"\x00"+"*."+ext+"\x00Все файлы\x00*.*\x00")), 0)
	f := openFileName{Size: uint32(unsafe.Sizeof(openFileName{})), Owner: hwnd, Filter: &filter[0], File: &buf[0], MaxFile: uint32(len(buf)), Flags: OFN_EXPLORER | OFN_PATHMUSTEXIST, DefaultExt: ptr(ext)}
	if !open {
		f.InitialDir = ptr(reportInitialDir())
	}
	var r uintptr
	if open {
		f.Flags |= OFN_FILEMUSTEXIST
		r, _, _ = pGetOpen.Call(uintptr(unsafe.Pointer(&f)))
	} else {
		r, _, _ = pGetSave.Call(uintptr(unsafe.Pointer(&f)))
	}
	if r == 0 {
		return ""
	}
	path := syscall.UTF16ToString(buf)
	if !open && !strings.HasSuffix(strings.ToLower(path), "."+ext) {
		path += "." + ext
	}

	return path
}

// Never overwrite an earlier report without explicit file-management action.

func switchPage(next string) {
	if next == activePage {
		return
	}
	if editingReportBlockID > 0 && (activePage == "weight" || activePage == "temp_sp20" || activePage == "temp_sp35" || activePage == "temp_cranes") {
		persistReportBlockEdit()
	}
	if activePage == "site" && next != "site" {
		if s, err := readSiteManual(); err == nil {
			project.SetSite(s)
			draftSite = s
		}
	}
	if tempEditActive {
		tempCommitRowEditing()
	}
	if activePage == "wind" {
		if err := syncWind(); err != nil {
			message(err.Error(), true)
			return
		}
	}
	for _, h := range pageControls[activePage] {
		pShow.Call(h, 0)
	}
	activePage = next
	if next == "home" {
		editingReportBlockID = 0
	}
	for _, h := range pageControls[activePage] {
		pShow.Call(h, 1)
	}
	if next == "site" && siteBackButton != 0 {
		title := "← На главную"
		if siteReturnPage == "snow" {
			title = "← К снегу"
		} else if siteReturnPage == "ice" {
			title = "← К гололёду"
		}
		pSetText.Call(siteBackButton, uintptr(unsafe.Pointer(ptr(title))))
	}
	// An in-cell editor must never appear on the home page after navigation.
	if controls[tempEditControl] != 0 {
		pShow.Call(controls[tempEditControl], 0)
	}
	refreshClimateUI()
	projectHomeOverview()
	if next == "temporary" {
		projectOverviewRefresh()
	}
	pUpdate.Call(hwnd)
}
func runAction(id int) {
	if iceAction(id) {
		return
	}
	if snowAction(id) {
		return
	}
	if settingsAction(id) {
		return
	}
	if aboutActions(id) {
		return
	}
	if weightAction(id) {
		return
	}
	if compositionCommand(id) {
		return
	}
	if tempAction(id) {
		return
	}
	switch id {
	case 501:
		siteReturnPage = ""
		switchPage("site")
	case 506:
		switchPage("temp_sp20")
		sp20Update()
		tempPageEntered()
	case 507:
		switchPage("temp_sp35")
		sp35Update()
		tempPageEntered()
	case 509:
		if tempCraneDraftCount() == 0 {
			tempSections.Sections = append(tempSections.Sections, tempEmptyCrane())
		}
		switchPage("temp_cranes")
		tempPageEntered()
		tempCraneSelectRow(0)
	case 513:
		switchPage("temporary")
		tempPageEntered()
	case 508:
		switchPage("about")
	case 514:
		switchPage("settings")
		settingsEntered()
	case 515:
		q := "Удалить все нагрузки текущего проекта? Параметры местности сохранятся."
		a, _, _ := pMsgBox.Call(hwnd, uintptr(unsafe.Pointer(ptr(q))), uintptr(unsafe.Pointer(ptr("Сбросить нагрузки"))), MB_YESNO|MB_ICONWARNING)
		if a == IDYES {
			site := project.Site
			project = initialProject()
			project.SetSite(site)
			project.Weight = &WeightCalc{Rows: []WeightRow{}}
			tempSections.Sections = nil
			snowDraft = defaultSnowCalc()
			iceDraft = defaultIceLoad()
			snowCurrentBlockID = 0
			iceCurrentBlockID = 0
			weightCurrentBlockID = 0
			craneSetCurrentID = 0
			craneNewSetRequested = false
			cranePlaceCurrentBlockID = 0
			cranePlaceResult = CranePlacementResult{}
			editingReportBlockID = 0
			refresh()
		}
	case 503:
		switchPage("snow")
		snowPageEntered()
	case 505:
		switchPage("ice")
		icePageEntered()
	case 1000:
		switchSeismic("seismic")
	case 1012:
		seismicInputs()
		switchSeismic("site")
	case 1013:
		seismicInputs()
		switchSeismic("seismic_help")
	case 1030:
		switchSeismic("seismic")
	case 1014:
		reloadSeismic()
	case 1033:
		showHelpGroup("А")
	case 1034:
		showHelpGroup("В")
	case 1035:
		showHelpGroup("С")
	case 1011:
		seismicInputs()
		switchSeismic("site")
		applySiteManual()
	case 661:
		openSnowClimatePicker("site")
	case 14038:
		openSnowClimatePicker("snow")
	case 651, 652, 653:
		openMapViewer(id - 651)
	case 642:
		applySiteManual()
	case 647:
		project.SetSite(Site{})
		draftSite = Site{}
		refreshClimateUI()
		projectHomeOverview()
	case 646:
		lookupCurrentElevation()
	case 207:
		useSiteWind()
	case 703:
		useSiteSnow()
	case 704:
		applySnowManual()

	case 502:
		switchPage("weight")
		weightPageEntered()
	case 504:
		switchPage("wind")
	case 399:
		if activePage == "site" && siteReturnPage != "" {
			target := siteReturnPage
			// Preserve selected mountains/height before returning.
			if s, err := readSiteManual(); err == nil {
				project.SetSite(s)
				draftSite = s
			}
			siteReturnPage = ""
			switchPage(target)
			if target == "snow" {
				snowPageEntered()
			}
			if target == "ice" {
				icePageEntered()
			}
		} else if activePage == "snow" && snowCurrentTab == 1 {
			snowSetTab(0)
		} else if activePage == "ice_extra" {
			if iceAdvancedStore() {
				icePreserveDraft = true
				switchPage("ice")
				icePageEntered()
			}
		} else {
			switchPage("home")
		}
	case 111:
		l, e := readLayer()
		if e != nil {
			message(e.Error(), true)
			return
		}
		project.Layers = append(project.Layers, l)
		refresh()
		ix := len(project.Layers) - 1
		send(100, LB_SETCURSEL, uintptr(ix), 0)
		showLayer(ix)
	case 112:
		ix := selected()
		if ix < 0 {
			message("Сначала выберите слой в списке", true)
			return
		}
		l, e := readLayer()
		if e != nil {
			message(e.Error(), true)
			return
		}
		project.Layers[ix] = l
		refresh()
		send(100, LB_SETCURSEL, uintptr(ix), 0)
		showLayer(ix)
	case 113:
		ix := selected()
		if ix < 0 {
			return
		}
		if len(project.Layers) == 1 {
			message("Нельзя удалить последний слой", true)
			return
		}
		project.Layers = append(project.Layers[:ix], project.Layers[ix+1:]...)
		refresh()
		if ix >= len(project.Layers) {
			ix = len(project.Layers) - 1
		}
		send(100, LB_SETCURSEL, uintptr(ix), 0)
		showLayer(ix)
	case 301:
		if err := syncWind(); err != nil {
			message(err.Error(), true)
			return
		}
		renderResults()
	case 302:
		if activePage == "wind" {
			if err := syncWind(); err != nil {
				message(err.Error(), true)
				return
			}
		}
		path := chooseSave("json", "Проект расчёта (JSON)", "расчёт_нагрузок.json", false)
		if path == "" {
			return
		}
		compositionEntered()
		saved, e := writeWithLockFallback(path, project.Save)
		if e != nil {
			message(e.Error(), true)
			return
		}
		reportDidSave(saved)
		message("Проект сохранён:\n"+saved, false)
	case 303:
		path := chooseSave("json", "Проект расчёта (JSON)", "", true)
		if path == "" {
			return
		}
		p, err := LoadProject(path)
		if err != nil {
			message(err.Error(), true)
			return
		}
		project = p
		snowCurrentBlockID = 0
		iceCurrentBlockID = 0
		weightCurrentBlockID = 0
		for _, block := range project.SavedBlocks {
			switch block.Kind {
			case "snow":
				snowCurrentBlockID = block.ID
			case "ice":
				iceCurrentBlockID = block.ID
			case "weight":
				if block.WeightCalc != nil {
					weightCurrentBlockID = block.ID
				}
			}
		}
		tempSections.Sections = nil
		craneSetCurrentID = 0
		craneNewSetRequested = false
		cranePlaceCurrentBlockID = 0
		cranePlaceResult = CranePlacementResult{}
		for _, b := range p.SavedBlocks {
			if b.Temporary != nil {
				tempSections.Sections = append(tempSections.Sections, *b.Temporary)
			}
			if b.Kind == "crane_set" {
				craneSetCurrentID = b.ID
			}
		}
		if craneSetCurrentID > 0 {
			for _, b := range p.SavedBlocks {
				if b.ID == craneSetCurrentID {
					tempSections.Sections = append(tempSections.Sections, b.CraneSet...)
					break
				}
			}
		}
		project.WindOverride = p.WindOverride
		refresh()
		message("Проект открыт", false)
	case 304:
		if activePage == "site" {
			s, err := currentSiteForReport()
			if err != nil {
				message(err.Error(), true)
				return
			}
			if !confirmSiteReport(s) {
				return
			}
			path := chooseSave("xlsx", "Excel XLSX", "характеристики_местности.xlsx", false)
			if path != "" {
				saveAndNotify(path, func(file string) error { return ExportSiteExcel(s, file) })
			}
			return
		}
		if err := syncWind(); err != nil {
			message(err.Error(), true)
			return
		}
		path := chooseSave("xlsx", "Файлы Excel (XLSX)", "нагрузки_расчёт.xlsx", false)
		if path == "" {
			return
		}
		// User exports always use the independent formula-based load ledger.
		// No fallback to the historical demonstration workbook with copied Go results.
		saveAndNotify(path, func(file string) error {
			return exportTempExcel(project.selectedReport(), file)
		})
	case 305:
		if activePage == "site" {
			s, err := currentSiteForReport()
			if err != nil {
				message(err.Error(), true)
				return
			}
			if !confirmSiteReport(s) {
				return
			}
			path := chooseSave("docx", "Word DOCX", "характеристики_местности.docx", false)
			if path != "" {
				saveAndNotify(path, func(file string) error { return ExportSiteWord(s, file) })
			}
			return
		}
		if err := syncWind(); err != nil {
			message(err.Error(), true)
			return
		}
		path := chooseSave("docx", "Документы Word (DOCX)", "нагрузки_отчёт.docx", false)
		if path == "" {
			return
		}
		saveAndNotify(path, func(file string) error { return ExportWord(project, file) })
	}
}
func wndProc(w, m, wp, lp uintptr) uintptr {
	switch m {
	case WM_APP_ELEVATION:
		finishElevationLookup()
		return 0
	case WM_COMMAND:
		id, code := int(wp&0xffff), int((wp>>16)&0xffff)
		if snowNotification(id, code) {
			return 0
		}
		if weightNotification(id, code) {
			return 0
		}
		if tempNotify(id, code) {
			return 0
		}
		if id == 100 && code == LBN_SELCHANGE {
			showLayer(selected())
		} else if id == 1004 && code == LBN_SELCHANGE {
			seismicListPick()
		} else if id == 1031 && code == LBN_SELCHANGE {
			showHelpDetail()
		} else if id == 602 && code == LBN_SELCHANGE {
			selectedPlaceRow()
		} else if id == 1007 && code == CBN_SELCHANGE {
			seismicInputs()
		} else if id == 1001 && code == CBN_SELCHANGE {
			seismicMapChanged()
		} else if code == CBN_SELCHANGE && id == iceKind {
			iceAction(iceKind)
		} else if code == CBN_SELCHANGE && isClimateCombo(id) {
			climateComboChanged(id)
			if activePage == "site" && !changingUnits {
				siteSaveSelectionSilently()
			}
		} else if code == EN_CHANGE {
			if activePage == "site" && !changingUnits && ((id >= 631 && id <= 635) || id == 644 || id == 662 || id == 663) {
				siteSaveSelectionSilently()
			}
			if id == 1003 {
				seismicScheduleSearch()
			} else if id == 1006 || id == 1007 || id == 1019 {
				if activePage == "seismic" && !changingUnits {
					seismicInputs()
				}
			} else if id == 601 {
				searchPlaces()
			} else if id == iceWindDrag && activePage == "ice" && !changingUnits {
				if !iceProgrammaticDrag {
					iceManualDrag = true
				}
			} else if id == 644 && activePage == "site" && !changingUnits {
				s, _ := readSiteManual()
				set(645, mountainStatus(s))
			} else {
				unitChanged(id)
			}
		} else if code == 0 { // BN_CLICKED, other control notifications ignored
			runAction(id)
		}
		return 0
	case 0x0113: // WM_TIMER
		if wp == seismicSearchTimerID {
			seismicRunScheduledSearch()
			return 0
		}
	case 5: // WM_SIZE
		tempResize()
		snowResize()
		return 0
	case WM_NOTIFY:
		if tempTableNotify(lp) {
			return 0
		}
		if weightNotify(lp) {
			return 0
		}
	case WM_MEASUREITEM:
		if wp == 902 {
			m := (*measureItemStruct)(unsafe.Pointer(lp))
			m.ItemHeight = 24
			return 1
		}
		if wp == 602 {
			m := (*measureItemStruct)(unsafe.Pointer(lp))
			m.ItemHeight = 25
			return 1
		}
	case WM_DRAWITEM:
		if snowPaintSphereWarning(wp, lp) {
			return 1
		}
		if snowPaintNormDiagram(wp, lp) {
			return 1
		}
		if snowPaintZonePlan(wp, lp) {
			return 1
		}
		if snowPaintSketch(wp, lp) {
			return 1
		}
		if snowPaintDiagram(wp, lp) {
			return 1
		}
		if tempWrapPaint(lp) {
			return 1
		}
		if wp == 902 {
			drawWeightMaterialRow(lp)
			return 1
		}
		if wp == 602 {
			drawClimateRow(lp)
			return 1
		}
	case WM_CONTEXTMENU:
		if tempContextMenu(wp, lp) {
			return 0
		}
		if weightContextMenu(wp, lp) {
			return 0
		}
	case WM_DESTROY:
		aboutCleanUp()
		pPostQuit.Call(0)
		return 0
	}
	r, _, _ := pDefWndProc.Call(w, m, wp, lp)
	return r
}
func footer() {
	button(304, "Расчёт Excel", 429, 649, 199, 39)
	button(305, "Отчёт Word", 640, 649, 193, 39)
}
func footerSite() {
	button(304, "Местность → Excel", 662, 784, 203, 39)
	button(305, "Местность → Word", 876, 784, 203, 39)
}
func navigation(title, desc string) {
	backLabel := "←  На главную"
	if buildingPage == "ice_extra" {
		backLabel = "← К гололёду"
	}
	h := button(399, backLabel, 20, 18, 157, 37)
	if buildingPage == "site" {
		siteBackButton = h
	}
	heading(title, 198, 15, 880, 37)
	label(desc, 198, 57, 920, 23)
}
func buildHome() {
	buildingPage = "home"
	heading(ebalaTitle, 26, 19, 850, 43)
	label("Помогатор для поверочных расчётов", 28, 61, 900, 28)
	button(508, "?", 1051, 18, 56, 48)
	label("ВЫБЕРИТЕ РАЗДЕЛ", 28, 112, 900, 27)
	menuButton(501, "01   ХАРАКТЕРИСТИКИ МЕСТНОСТИ\r\nПоиск по климату и СП 14.13330.2018", 26, 148, true)
	menuButton(502, "02   СОБСТВЕННЫЙ ВЕС\r\nСправочник материалов и расчёт пирога", 573, 148, true)
	menuButton(503, "03   РАСЧЁТ СНЕГА\r\nПокрытия, перепады и резервуары · СП 20.13330.2016", 26, 247, true)
	// ЗАГЛУШКА: самостоятельный модуль ветровой нагрузки НЕ РЕАЛИЗОВАН.
	// Отключён намеренно до завершения разработки снегового модуля.
	// Не активировать при аудитах/рефакторинге без отдельного ТЗ.
	menuButton(504, "04   РАСЧЁТ ВЕТРА\r\nМодуль в разработке", 573, 247, false)
	menuButton(505, "05   РАСЧЁТ ГОЛОЛЁДА\r\nКруглые и строительные элементы · СП 20.13330.2016", 26, 346, true)
	menuButton(506, "06   ВРЕМЕННЫЕ НАГРУЗКИ\r\nПомещения и конструкции · СП 20.13330.2016", 573, 346, true)
	menuButton(507, "07   ТРАНСПОРТНЫЕ НАГРУЗКИ\r\nАвтомобильные и железнодорожные · СП 35.13330.2011", 26, 445, true)
	menuButton(509, "08   МОСТОВЫЕ КРАНЫ\r\nКолёсные нагрузки и справочник кранов", 573, 445, true)
	button(513, "ОБЩАЯ ВЕДОМОСТЬ НАГРУЗОК", 26, 557, 1072, 49)
	label("В ТЕКУЩЕМ ПРОЕКТЕ УЧТЕНО", 28, 617, 650, 24)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 28, 646, 1070, 25, 403)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 28, 678, 1070, 25, 404)
	button(302, "Сохранить расчёт", 26, 764, 254, 42)
	button(303, "Открыть расчёт", 291, 764, 254, 42)
	button(515, "Сбросить все нагрузки", 556, 764, 286, 42)
	button(514, "Настройки", 854, 764, 244, 42)
}

// Read-only edit boxes let users select and copy addresses using Ctrl+C.
func editReadOnlyLink(id int, value string, x, y, w, h int) {
	create("EDIT", value, WS_CHILD|WS_VISIBLE|ES_READONLY|ES_AUTOHSCROLL, x, y, w, h, id)
}
func buildAbout() {
	buildingPage = "about"
	navigation("О программе", "")
	label(ebalaTitle+" — помогатор для поверочных расчётов", 28, 111, 1060, 33)
	heading("Об авторе", 28, 204, 1000, 32)
	aboutAvatar(30, 250)
	heading("Романов Михаил", 191, 250, 900, 35)
	label("Адрес: Нить Персея-Пегаса, Сверхскопление Ланиакея,", 193, 295, 880, 27)
	label("Галактика Млечный Путь, Рукав Ориона, Солнечная система,", 193, 324, 880, 27)
	label("Третья планета от Солнца, там уже спросите - подскажут.", 193, 353, 880, 27)
	heading("Немного о технологии", 28, 408, 1050, 34)
	label("Программа написана per rectum силами GPT.", 30, 450, 1060, 26)
	label("Экселевские таблицы в выводе программы нужны для проверки формул", 30, 479, 1060, 26)
	label("(никогда не доверяйте GPT и рекламе средств для похудения).", 30, 507, 1060, 26)
	heading("Музыка, донаты и обратная связь", 28, 558, 1050, 32)
	button(510, "Слушайте группу «Башкирские снегири» на всех площадках", 30, 600, 675, 32)
	editReadOnlyLink(15010, "https://band.link/UylkI", 724, 603, 350, 25)
	button(511, "Оставляйте донаты (но только если понравилась моя музыка!)", 30, 640, 675, 32)
	editReadOnlyLink(15011, "https://tips.yandex.ru/guest/payment/3749434", 724, 643, 380, 26)
	button(512, "Отзывы, замечания, поздравления с днём рождения", 30, 680, 675, 32)
	editReadOnlyLink(15012, "mincka@yandex.ru", 724, 683, 350, 26)
	heading("И да пребудет с вами Сила!", 29, 758, 1015, 41)
}
func buildWeight() { buildWeightNative() }

// ЗАГЛУШКА / СТАРЫЙ ПРОТОТИП: buildWind() оставлен в исходниках только
// как историческая заготовка. Полный нормативный расчёт ветра по
// СП 20.13330.2016 (аэродинамические схемы, пульсации и пр.) НЕ РЕАЛИЗОВАН.
// Кнопка входа на главной отключена сознательно. Не считать этот код
// реализованной функцией при аудите и не включать без отдельного задания.
// Это не относится к вспомогательному учёту ветра внутри модуля гололёда.
func buildWind() {
	buildingPage = "wind"
	navigation("Расчёт ветра", "Временная проверочная схема · полный нормативный модуль будет разработан отдельно")
	label("ВНИМАНИЕ: здесь только w = w₀ × k(z) × c. Не учитываются пульсации и специальные аэродинамические схемы СП 20.13330.2016.", 21, 116, 1100, 39)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 28, 151, 1075, 30, 210)
	label("Район ветра (табл. 11.1)", 28, 175, 350, 24)
	combo(206, windOptions, 28, 199, 340)
	label("Базовое давление w₀, кПа", 28, 243, 360, 24)
	label("То же давление, кгс/м²", 568, 243, 360, 24)
	edit(201, "", 28, 272, 340, 33)
	edit(205, "", 568, 272, 340, 33)
	label("Выбор района подставляет w₀ автоматически; изменение давления создаёт локальную правку.", 29, 316, 1050, 25)
	button(207, "Восстановить из местности", 568, 344, 340, 34)
	label("Коэффициент высоты k(z)", 28, 394, 380, 24)
	label("Аэродинамический коэффициент c", 568, 394, 440, 24)
	edit(202, "", 28, 423, 340, 32)
	edit(203, "", 568, 423, 340, 32)
	label("Коэффициент надёжности γf", 28, 471, 435, 24)
	edit(204, "", 28, 500, 340, 32)
	button(301, "Рассчитать", 568, 496, 240, 40)
	label("РЕЗУЛЬТАТ РАСЧЁТА ВЕТРОВОГО ДАВЛЕНИЯ", 21, 547, 1080, 25)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 21, 576, 1075, 24, 402)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 21, 601, 1075, 24, 406)
	label("Предварительный расчёт. Не заменяет полный нормативный ветровой расчёт.", 21, 626, 1080, 20)
	footer()
}
func buildUI() {
	buildHome()
	buildAbout()
	buildSettings()
	buildSite()
	buildSeismic()
	buildSeismicHelp()
	buildWeight()
	buildSnowInputs()
	buildWind()
	buildIceInputs()
	buildIceAdvanced()
	tempLoad()
	tempBuildPages()
	refresh()
	searchPlaces()
}
func main() {
	runtime.LockOSThread()
	h, _, _ := pGetModule.Call(0)
	font, _, _ = pCreateFont.Call(18, 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 0, 0, uintptr(unsafe.Pointer(ptr("Segoe UI"))))
	headingFont, _, _ = pCreateFont.Call(27, 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, 0, 0, uintptr(unsafe.Pointer(ptr("Segoe UI"))))
	cname := ptr("EbalaLoadsWindowV002")
	cb := syscall.NewCallback(wndProc)
	wc := wndClassEx{Size: uint32(unsafe.Sizeof(wndClassEx{})), WndProc: cb, Instance: h, Background: 6, ClassName: cname}
	r, _, err := pRegister.Call(uintptr(unsafe.Pointer(&wc)))
	if r == 0 {
		message(fmt.Sprintf("Не удалось зарегистрировать окно: %v", err), true)
		return
	}
	hwnd, _, err = pCreate.Call(0, uintptr(unsafe.Pointer(cname)), uintptr(unsafe.Pointer(ptr(ebalaTitle+" · Помогатор для поверочных расчётов"))), WS_OVERLAPPEDWINDOW,
		80, 35, 1160, 945, 0, 0, h, 0)
	if hwnd == 0 {
		message(fmt.Sprintf("Не удалось создать окно: %v", err), true)
		return
	}
	loadSettings()
	loadSeismicCatalog()
	project.Weight = &WeightCalc{Rows: []WeightRow{}}
	buildUI()
	pShow.Call(hwnd, 1)
	pUpdate.Call(hwnd)
	var m msg
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslate.Call(uintptr(unsafe.Pointer(&m)))
		pDispatch.Call(uintptr(unsafe.Pointer(&m)))
	}
}
