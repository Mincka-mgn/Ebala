//go:build windows

package main

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

// Only report selections entered by the user are shown in summary tables.
func tempTransportTable(id, y int) {
	tempTable(id, 25, y, 1085, 123, []string{"№", "Транспортная нагрузка", "Общие данные", "Нормативные нагрузки", "γf", "Расчётные нагрузки", "Удалить"}, []int{37, 201, 173, 239, 102, 252, 77})
}
func tempAddManualRow() {
	row := SP20Row{ID: "CUSTOM", SourceID: "задано пользователем", Category: "Собственная нагрузка", Title: "Новая нагрузка", Official: "Задано пользователем", Type: "Равномерная", Unit: "кПа", Gamma: 1.3, Editable: true, Source: "Данные пользователя"}
	s, e := sp20CustomSection(row, "Новая нагрузка", "", "1,3")
	if e != nil {
		message(e.Error(), true)
		return
	}
	s.Params["custom_line"] = true
	tempSections.Sections = append(tempSections.Sections, s)
	tempRefreshChosen()
	tempStartRowEditing(tempSp20Chosen, len(tempSummaryRows[tempSp20Chosen])-1, 1)
}
func tempNotifySummary(id, code int) bool {
	return false
}
func tempRowsForTransport(id int) {
	var data [][]string
	var mapping []int
	for i, s := range tempSections.Sections {
		if !tempTransport(s) {
			continue
		}
		mapping = append(mapping, i)
		d, n, g, c := tempTransportFields(s)
		data = append(data, []string{fmt.Sprint(len(mapping)), tempSP35NormalTitle(s.Title), d, n, g, c, "Удалить"})
	}
	tempSummaryRows[id] = mapping
	tempRows(id, data)
}
func tempUpdateSummary(id int) {
	if id == tempSp35Chosen || id == tempTransportChosen {
		tempRowsForTransport(id)
		return
	}
	tempSummary(id, id == tempCraneChosen || id == tempCraneLocalChosen)
}
func tempSelectedPhysical(id int, row int) int {
	vals := tempSummaryRows[id]
	if row < 0 || row >= len(vals) {
		return -1
	}
	return vals[row]
}
func tempDeleteAt(id, row int) {
	i := tempSelectedPhysical(id, row)
	if i < 0 {
		return
	}
	removeSavedTempSelection(tempSections.Sections[i])
	tempSections.Sections = append(tempSections.Sections[:i], tempSections.Sections[i+1:]...)
	tempRefreshChosen()
}
func tempDuplicateAt(id, row int) {
	i := tempSelectedPhysical(id, row)
	if i < 0 {
		return
	}
	x := tempSections.Sections[i]
	x.Params = copyTempParams(x.Params)
	x.Wheels = append([]TempWheel(nil), x.Wheels...)
	tempSections.Sections = append(tempSections.Sections[:i+1], append([]TempSelection{x}, tempSections.Sections[i+1:]...)...)
	tempRefreshChosen()
}
func copyTempParams(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := map[string]interface{}{}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
func tempMoveAt(id, row, shift int) {
	i := tempSelectedPhysical(id, row)
	if i < 0 {
		return
	}
	seq := tempSummaryRows[id]
	other := row + shift
	if other < 0 || other >= len(seq) {
		return
	}
	j := seq[other]
	s := tempSections.Sections[i]
	tempSections.Sections = append(tempSections.Sections[:i], tempSections.Sections[i+1:]...)
	if j > i {
		j--
	}
	if shift > 0 {
		j++
	}
	tempSections.Sections = append(tempSections.Sections[:j], append([]TempSelection{s}, tempSections.Sections[j:]...)...)
	tempRefreshChosen()
}

// Minimal native table context menu, independent of input pages.
const (
	tempMenuDuplicate = 12881
	tempMenuDelete    = 12882
	tempMenuUp        = 12883
	tempMenuDown      = 12884
)

var tempContextID int
var tempContextRow int

func tempSummaryControlID(id int) bool {
	switch id {
	case tempChosen, tempSp20Chosen, tempSp35Chosen, tempTransportChosen, tempCraneChosen, tempCraneLocalChosen:
		return true
	}
	return false
}
func tempContextMenu(wp, lp uintptr) bool {
	id := 0
	for _, candidate := range []int{tempChosen, tempSp20Chosen, tempSp35Chosen, tempTransportChosen, tempCraneChosen, tempCraneLocalChosen} {
		if controls[candidate] != 0 && controls[candidate] == wp {
			id = candidate
			break
		}
	}
	if id == 0 {
		return false
	}
	row := tempSelected(id)
	if row < 0 {
		return true
	}
	tempContextID, tempContextRow = id, row
	menu, _, _ := pCreatePopupMenuWeight.Call()
	defer pDestroyMenuWeight.Call(menu)
	for _, opt := range []struct {
		id      int
		caption string
	}{{tempMenuDuplicate, "Дублировать"}, {tempMenuDelete, "Удалить"}, {tempMenuUp, "Выше"}, {tempMenuDown, "Ниже"}} {
		pAppendMenuWeight.Call(menu, 0, uintptr(opt.id), uintptr(unsafe.Pointer(ptr(opt.caption))))
	}
	pos := point{}
	pGetCursorPosWeight.Call(uintptr(unsafe.Pointer(&pos)))
	pTrackPopupMenuWeight.Call(menu, 0, posX(pos), posY(pos), 0, hwnd, 0)
	return true
}
func posX(p point) uintptr { return uintptr(uint32(p.X)) }
func posY(p point) uintptr { return uintptr(uint32(p.Y)) }
func tempRunContext(id int) bool {
	if id < tempMenuDuplicate || id > tempMenuDown {
		return false
	}
	switch id {
	case tempMenuDuplicate:
		tempDuplicateAt(tempContextID, tempContextRow)
	case tempMenuDelete:
		tempDeleteAt(tempContextID, tempContextRow)
	case tempMenuUp:
		tempMoveAt(tempContextID, tempContextRow, -1)
	case tempMenuDown:
		tempMoveAt(tempContextID, tempContextRow, 1)
	}
	return true
}
func tempSummaryDblClick(id int, row int) bool {
	if id == tempSP20List {
		tempAction(tempSP20Add)
		return true
	}
	if id == tempSP35List {
		tempAction(tempSP35Add)
		return true
	}
	if tempSummaryControlID(id) {
		i := tempSelectedPhysical(id, row)
		if i < 0 {
			return true
		}
		s := tempSections.Sections[i]
		if s.Group == "СП 20 · временные" || tempTransport(s) || tempIsCrane(s) {
			tempStartRowEditing(id, row, 1)
		}
		return true
	}
	if id == tempCraneList {
		tempAction(tempCraneUse)
		return true
	}
	return false
}

var tempEditParentID, tempEditRow, tempEditCol int
var tempEditActive bool
var tempEditOriginal string

const tempEditControl = 12179

var tempFocusProc = u32.NewProc("SetFocus")
var tempGetWindowRect = u32.NewProc("GetWindowRect")
var tempScreenToClient = u32.NewProc("ScreenToClient")
var tempOldEditProc uintptr
var tempEditCallback uintptr

func tempEditWndProc(w, m, wp, lp uintptr) uintptr {
	if m == 0x100 { // WM_KEYDOWN
		if wp == 13 {
			tempCommitRowEditing()
			return 0
		} // Enter
		if wp == 27 {
			tempEditActive = false
			pShow.Call(controls[tempEditControl], 0)
			return 0
		} // Escape
	}
	r, _, _ := pCallWindowProcWeight.Call(tempOldEditProc, w, m, wp, lp)
	return r
}

func tempStartRowEditing(id, row, col int) {
	if !tempSummaryControlID(id) {
		return
	}
	i := tempSelectedPhysical(id, row)
	if i < 0 {
		return
	}
	section := tempSections.Sections[i]
	if tempTransport(section) {
		if col != 1 {
			return
		}
	} else if tempIsCrane(section) {
		if col < 1 || col > 5 {
			return
		}
	} else if section.Group == "СП 20 · временные" {
		if col != 1 && col != 2 && col != 3 {
			return
		}
	} else {
		return
	}
	pSend.Call(controls[id], 0x1013, uintptr(row), 0) // LVM_ENSUREVISIBLE: scroll to edited row
	var r tempRect
	r.Top = int32(col) // LVM_GETSUBITEMRECT expects subitem in top
	r.Left = 0         // LVIR_BOUNDS
	pSend.Call(controls[id], 0x1038, uintptr(row), uintptr(unsafe.Pointer(&r)))
	listBox := tempRect{}
	tempGetWindowRect.Call(controls[id], uintptr(unsafe.Pointer(&listBox)))
	pt := point{X: listBox.Left + r.Left + 3, Y: listBox.Top + r.Top + 2}
	tempScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&pt)))
	x, y := int(pt.X), int(pt.Y)
	w, h := int(r.Right-r.Left-8), int(r.Bottom-r.Top-5)
	if w < 70 {
		w = 200
	}
	if h < 22 {
		h = 24
	}
	if controls[tempEditControl] == 0 {
		previousBuildingPage := buildingPage
		buildingPage = activePage
		create("EDIT", "", WS_CHILD|WS_BORDER|ES_AUTOHSCROLL, x, y, w, h, tempEditControl)
		buildingPage = previousBuildingPage
		tempEditCallback = syscall.NewCallback(tempEditWndProc)
		tempOldEditProc, _, _ = pSetWindowLongPtrWeight.Call(controls[tempEditControl], ^uintptr(3), tempEditCallback)
	}
	pSetWindowPosWeight.Call(controls[tempEditControl], 0, uintptr(x), uintptr(y), uintptr(w), uintptr(h), 0)
	tempEditParentID, tempEditRow, tempEditCol = id, row, col
	s := tempSections.Sections[i]
	tempEditOriginal = s.Title
	if tempIsCrane(s) {
		switch col {
		case 2:
			tempEditOriginal = tempCraneKind(s)
		case 3:
			tempEditOriginal = fmt.Sprint(tempCraneWheelsCount(s))
		case 4:
			tempEditOriginal = tempCraneForcesText(s, tempComboIndex(tempCraneUnit) == 1)
		case 5:
			tempEditOriginal = tempCraneStepsText(s)
		}
		if col == 3 && tempCraneWheelsCount(s) == 0 {
			tempEditOriginal = ""
		}
	} else if s.Group == "СП 20 · временные" {
		if col == 2 {
			if def, ok := s.Params["is_defined"].(bool); ok && def {
				tempEditOriginal = tempRU(tempNumber(s, "normative_reduced")*kpaToKgfM2(1), 2)
			} else {
				tempEditOriginal = ""
			}
		}
		if col == 3 {
			tempEditOriginal = tempRU(tempNumber(s, "gamma_f"), 3)
		}
	}
	set(tempEditControl, tempEditOriginal)
	tempEditActive = true
	pShow.Call(controls[tempEditControl], 1)
	tempFocusProc.Call(controls[tempEditControl])
	pSend.Call(controls[tempEditControl], 0xB1, 0, ^uintptr(0)) // EM_SETSEL
}
func tempCommitRowEditing() {
	if !tempEditActive {
		return
	}
	tempEditActive = false
	val := strings.TrimSpace(get(tempEditControl))
	pShow.Call(controls[tempEditControl], 0)
	i := tempSelectedPhysical(tempEditParentID, tempEditRow)
	if i < 0 {
		return
	}
	s := &tempSections.Sections[i]
	if tempIsCrane(*s) {
		if tempEditCol == 2 {
			if s.Params == nil {
				s.Params = map[string]interface{}{}
			}
			s.Params["crane_kind"] = val
		} else {
			mappedColumn := tempEditCol
			if mappedColumn > 2 {
				mappedColumn--
			}
			if err := tempCraneSetCell(s, mappedColumn, val, tempComboIndex(tempCraneUnit) == 1); err != nil {
				message(err.Error(), true)
			}
		}
	} else if tempTransport(*s) {
		if tempEditCol == 1 && val != "" {
			s.Title = val
		}
	} else if s.Group == "СП 20 · временные" {
		switch tempEditCol {
		case 1:
			if val != "" {
				s.Title = val
			}
		case 2:
			if val == "" {
				s.Params["is_defined"] = false
				s.Params["normative_reduced"] = 0.0
				s.Params["design_reduced"] = 0.0
			} else {
				n, e := parseNumber(val)
				if e != nil || n < 0 {
					message("Нагрузка должна быть неотрицательным числом, кгс/м².", true)
					break
				}
				kpa := kgfM2ToKpa(n)
				gamma := tempNumber(*s, "gamma_f")
				if s.Params["custom_line"] == true || gamma <= 0 {
					gamma = 1.3
					if kpa >= 2 {
						gamma = 1.2
					}
				}
				s.Params["normative_reduced"] = kpa
				s.Params["design_reduced"] = kpa * gamma
				s.Params["normative"] = kpa
				s.Params["design"] = kpa * gamma
				s.Params["gamma_f"] = gamma
				s.Params["is_defined"] = true
			}
		case 3:
			gamma, e := parseNumber(val)
			if e != nil || gamma <= 0 {
				message("Коэффициент надёжности должен быть положительным числом.", true)
				break
			}
			s.Params["gamma_f"] = gamma
			s.Params["design_reduced"] = tempNumber(*s, "normative_reduced") * gamma
			s.Params["design"] = tempNumber(*s, "normative") * gamma
		}
	}
	tempRefreshChosen()
}
func tempEditNotify(id, code int) bool {
	if id != tempEditControl {
		return false
	}
	if code == EN_KILLFOCUS {
		tempCommitRowEditing()
		return true
	}
	return true
}

func tempEmptyCrane() TempSelection {
	n := tempNextCraneNumber()
	return TempSelection{Group: "Краны · ручной ввод", Title: fmt.Sprintf("Кран %d", n), Source: "Данные пользователя", Params: map[string]interface{}{"wheel_count": 0, "crane_kind": ""}, Wheels: nil}
}
func tempCraneDraftCount() int {
	n := 0
	for _, s := range tempSections.Sections {
		if tempIsCrane(s) {
			n++
		}
	}
	return n
}
func tempCranePhysical(row int) int {
	if row < 0 {
		return -1
	}
	n := 0
	for i, s := range tempSections.Sections {
		if tempIsCrane(s) {
			if n == row {
				return i
			}
			n++
		}
	}
	return -1
}
func tempSetListRow(id, row int) {
	if controls[id] == 0 || row < 0 {
		return
	}
	item := lvItem{state: 3, stateMask: 3}
	pSend.Call(controls[id], 0x102B, uintptr(row), uintptr(unsafe.Pointer(&item)))
}
func tempCraneSelectRow(row int) {
	if tempCranePhysical(row) < 0 {
		return
	}
	craneChangingRow = true
	craneActiveRow = row
	tempSetListRow(tempCraneLocalChosen, row)
	craneChangingRow = false
}
