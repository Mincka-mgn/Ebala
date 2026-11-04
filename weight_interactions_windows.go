//go:build windows

package main

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

const (
	LVM_GETSUBITEMRECT       = LVM_FIRST + 56
	LB_ITEMFROMPOINT         = 0x01A9
	WM_KEYDOWN_WEIGHT        = 0x0100
	VK_RETURN_WEIGHT         = 13
	VK_ESCAPE_WEIGHT         = 27
	SWP_SHOWWINDOW_WEIGHT    = 0x0040
	LVIR_BOUNDS_WEIGHT       = 0
	LVM_HITTEST_WEIGHT       = LVM_FIRST + 18
	WM_MOUSEMOVE_WEIGHT      = 0x0200
	WM_LBUTTONUP_WEIGHT      = 0x0202
	WM_CAPTURECHANGED_WEIGHT = 0x0215
	VK_DELETE_WEIGHT         = 46
	MK_LBUTTON_WEIGHT        = 1
)

var (
	pClientToScreenWeight   = u32.NewProc("ClientToScreen")
	pScreenToClientWeight   = u32.NewProc("ScreenToClient")
	pSetWindowPosWeight     = u32.NewProc("SetWindowPos")
	pShowWindowWeight       = u32.NewProc("ShowWindow")
	pSetWindowLongPtrWeight = u32.NewProc("SetWindowLongPtrW")
	pCallWindowProcWeight   = u32.NewProc("CallWindowProcW")
	pCreatePopupMenuWeight  = u32.NewProc("CreatePopupMenu")
	pAppendMenuWeight       = u32.NewProc("AppendMenuW")
	pTrackPopupMenuWeight   = u32.NewProc("TrackPopupMenu")
	pDestroyMenuWeight      = u32.NewProc("DestroyMenu")
	pGetCursorPosWeight     = u32.NewProc("GetCursorPos")
	pSetCaptureWeight       = u32.NewProc("SetCapture")
	pReleaseCaptureWeight   = u32.NewProc("ReleaseCapture")
)

type weightItemActivate struct {
	Header notificationHeader
	Item   int32
	Sub    int32
}

var (
	weightEditingCell    bool
	weightEndingCell     bool
	weightCellRow        = -1
	weightCellCol        = -1
	weightOldEditProc    uintptr
	weightEditCallback   uintptr
	weightEditingCatalog = -1
	weightOldListProc    uintptr
	weightListCallback   uintptr
	weightDragging       bool
	weightDragFrom       = -1
	weightDragTarget     = -1
)

func weightInitCellSubclass() {
	if controls[940] == 0 || weightOldEditProc != 0 {
		return
	}
	weightEditCallback = syscall.NewCallback(weightCellWndProc)
	weightOldEditProc, _, _ = pSetWindowLongPtrWeight.Call(controls[940], ^uintptr(3), weightEditCallback)
}
func weightCellWndProc(w, msg, wp, lp uintptr) uintptr {
	if msg == WM_KEYDOWN_WEIGHT {
		if wp == VK_RETURN_WEIGHT {
			weightCommitCellEdit(true)
			return 0
		}
		if wp == VK_ESCAPE_WEIGHT {
			weightHideCellEdit()
			pSetFocus.Call(controls[903])
			return 0
		}
	}
	r, _, _ := pCallWindowProcWeight.Call(weightOldEditProc, w, msg, wp, lp)
	return r
}
func weightHideCellEdit() {
	weightEditingCell = false
	weightCellRow, weightCellCol = -1, -1
	if controls[940] != 0 {
		pShowWindowWeight.Call(controls[940], 0)
	}
	if controls[941] != 0 {
		pShowWindowWeight.Call(controls[941], 0)
	}
}
func weightCellDisplay(r WeightRow, col int) string {
	switch col {
	case 0:
		return r.Name
	case 1:
		if r.Kind == "layer" {
			return "По толщине"
		}
		return "Готовая нагрузка"
	case 2:
		if r.Kind == "layer" {
			return decimalInput(r.Thickness, 1)
		}
	case 3:
		if r.Kind == "layer" {
			return decimalInput(r.Density, 1)
		}
	case 4:
		return decimalInput(r.NormKgf(), 1)
	case 5:
		return decimalInput(r.Gamma, 4)
	case 6:
		return decimalInput(r.NormKgf()*r.Gamma, 1)
	}
	return ""
}
func weightStartCellEdit(row, col int) {
	if weightEndingCell {
		return
	}
	if row < 0 || row >= len(weightCurrent().Rows) || col < 0 || col > 6 {
		return
	}
	if weightCurrent().Rows[row].Kind != "layer" && (col == 2 || col == 3) {
		return
	}
	if weightEditingCell {
		weightCommitCellEdit(false)
	}
	r := winRect{Left: LVIR_BOUNDS_WEIGHT, Top: int32(col)}
	if col == 0 {
		r.Left = 2 /* LVIR_LABEL: first column only */
	}
	ok, _, _ := pSend.Call(controls[903], LVM_GETSUBITEMRECT, uintptr(row), uintptr(unsafe.Pointer(&r)))
	if ok == 0 || r.Right <= r.Left || r.Bottom <= r.Top {
		return
	}
	// List-view coordinates -> main-window client coordinates.
	top := point{X: r.Left, Y: r.Top}
	pClientToScreenWeight.Call(controls[903], uintptr(unsafe.Pointer(&top)))
	pScreenToClientWeight.Call(hwnd, uintptr(unsafe.Pointer(&top)))
	width := int(r.Right - r.Left)
	height := int(r.Bottom - r.Top)
	if width < 24 || height < 16 {
		return
	}
	weightEditingCell, weightCellRow, weightCellCol = true, row, col
	currentWeightIndex = row
	if col == 1 {
		which := uintptr(0)
		if weightCurrent().Rows[row].Kind != "layer" {
			which = 1
		}
		pSend.Call(controls[941], 0x014E /* CB_SETCURSEL */, which, 0)
		pSetWindowPosWeight.Call(controls[941], 0, uintptr(top.X+1), uintptr(top.Y+1), uintptr(width-2), 240, SWP_SHOWWINDOW_WEIGHT)
		pSetFocus.Call(controls[941])
		pSend.Call(controls[941], 0x014F /* CB_SHOWDROPDOWN */, 1, 0)
		return
	}
	set(940, weightCellDisplay(weightCurrent().Rows[row], col))
	pSetWindowPosWeight.Call(controls[940], 0, uintptr(top.X+1), uintptr(top.Y+1), uintptr(width-2), uintptr(height-2), SWP_SHOWWINDOW_WEIGHT)
	pSetFocus.Call(controls[940])
	pSend.Call(controls[940], 0x00B1 /* EM_SETSEL */, 0, ^uintptr(0))
}

func weightCommitKindSelection() {
	if !weightEditingCell || weightCellCol != 1 || weightEndingCell {
		return
	}
	row := weightCellRow
	idx, _, _ := pSend.Call(controls[941], 0x0147 /* CB_GETCURSEL */, 0, 0)
	if row < 0 || row >= len(weightCurrent().Rows) || idx > 1 {
		return
	}
	w := weightCurrent()
	kind := "layer"
	if idx == 1 {
		kind = "direct"
	}
	w.Rows[row] = SwitchWeightKind(w.Rows[row], kind)
	weightHideCellEdit()
	weightRefreshSingleRow(row)
	pSetFocus.Call(controls[903])
}
func weightCommitCellEdit(moveFocus bool) {
	if !weightEditingCell || weightEndingCell {
		return
	}
	weightEndingCell = true
	row, col := weightCellRow, weightCellCol
	newText := strings.TrimSpace(get(940))
	weightHideCellEdit()
	if row >= 0 && row < len(weightCurrent().Rows) {
		r := weightCurrent().Rows[row]
		if col == 0 {
			if newText != "" {
				r.Name = newText
			} else {
				message("Название строки не может быть пустым.", true)
			}
		} else if col == 1 {
			// Mode is handled by the inline dropdown.
		} else {
			v, e := parseNumber(newText)
			if e != nil {
				message("Введите число в выбранной ячейке.", true)
			} else {
				field := []string{"name", "kind", "thickness", "density", "norm", "gamma", "design"}[col]
				edited, err := EditWeightField(r, field, v)
				if err != nil {
					message(err.Error(), true)
				} else {
					r = edited
				}
			}
		}
		weightCurrent().Rows[row] = r
	}
	weightEndingCell = false
	weightRefreshSingleRow(row)
	if moveFocus {
		pSetFocus.Call(controls[903])
	}
}
func isPersonalWeightMaterial(m material) bool {
	for _, x := range catalogUser.Items {
		if x.ID == m.ID && x.CategoryID == m.CategoryID {
			return true
		}
	}
	return false
}
func drawWeightMaterialRow(lp uintptr) {
	d := (*drawItemStruct)(unsafe.Pointer(lp))
	idx := int(d.ItemID)
	if idx < 0 || idx >= len(shownMaterials) {
		return
	}
	personal := isPersonalWeightMaterial(shownMaterials[idx])
	fg, bg := uintptr(0x00202020), uintptr(0x00FFFFFF)
	if personal {
		fg, bg = 0x00915E11, 0x00E8F5FF
	} // personal = dark blue on pale cream
	if d.ItemState&ODS_SELECTED != 0 {
		fg, bg = 0x00FFFFFF, 0x00A85316
	}
	pSetBkColor.Call(d.HDC, bg)
	pSetTextColor.Call(d.HDC, fg)
	// Use the exact same text as the underlying item; no checkbox widgets.
	s := shownMaterials[idx].Name
	m := shownMaterials[idx]
	if m.Kind == "direct" {
		s += fmt.Sprintf("    •    %.1f кгс/м²", m.GNorm)
	} else if m.Max != nil {
		if m.Min != nil {
			if *m.Min == *m.Max {
				s += fmt.Sprintf("    •    %.0f кг/м³", *m.Max)
			} else {
				s += fmt.Sprintf("    •    %.0f–%.0f кг/м³", *m.Min, *m.Max)
			}
		} else {
			s += fmt.Sprintf("    •    до %.0f кг/м³", *m.Max)
		}
	}
	u := syscall.StringToUTF16(s)
	r := d.ItemRect
	pExtTextOut.Call(d.HDC, uintptr(r.Left+7), uintptr(r.Top+3), ETO_OPAQUE|ETO_CLIPPED, uintptr(unsafe.Pointer(&r)), uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), 0)
}

// ListView hit testing takes client coordinates, while context menus take screen coordinates.
type weightHitTest struct {
	Pt    point
	Flags uint32
	Item  int32
	Sub   int32
	Group int32
}

func weightRowAt(x, y int32) int {
	hit := weightHitTest{Pt: point{X: x, Y: y}, Item: -1}
	pSend.Call(controls[903], LVM_HITTEST_WEIGHT, 0, uintptr(unsafe.Pointer(&hit)))
	return int(hit.Item)
}
func weightInitListSubclass() {
	if controls[903] == 0 || weightOldListProc != 0 {
		return
	}
	weightListCallback = syscall.NewCallback(weightListWndProc)
	weightOldListProc, _, _ = pSetWindowLongPtrWeight.Call(controls[903], ^uintptr(3), weightListCallback)
}
func weightBeginDrag(row int) {
	if row < 0 || row >= len(weightCurrent().Rows) || weightEditingCell {
		return
	}
	weightDragging = true
	weightDragFrom, weightDragTarget = row, row
	currentWeightIndex = row
	pSetCaptureWeight.Call(controls[903])
}
func weightListWndProc(w, m, wp, lp uintptr) uintptr {
	switch m {
	case WM_KEYDOWN_WEIGHT:
		if wp == VK_DELETE_WEIGHT && activePage == "weight" {
			if ix := weightSelectedRow(); ix >= 0 {
				currentWeightIndex = ix
				weightRemove()
			}
			return 0
		}
	case WM_MOUSEMOVE_WEIGHT:
		if weightDragging {
			if wp&MK_LBUTTON_WEIGHT == 0 {
				weightCancelDrag()
				return 0
			}
			x := int32(int16(lp & 0xffff))
			y := int32(int16((lp >> 16) & 0xffff))
			target := weightRowAt(x, y)
			if target < 0 {
				if y < 0 {
					target = 0
				} else {
					target = len(weightCurrent().Rows) - 1
				}
			}
			if target >= 0 && target < len(weightCurrent().Rows) && weightDragTarget != target {
				weightDragTarget = target
				stateOff := listItem{State: 0, StateMask: LVIS_SELECTED}
				pSend.Call(controls[903], LVM_SETITEMSTATE, ^uintptr(0), uintptr(unsafe.Pointer(&stateOff)))
				weightSelectRow(target)
			}
			return 0
		}
	case WM_CAPTURECHANGED_WEIGHT:
		weightDragging = false
		weightDragFrom, weightDragTarget = -1, -1
	case WM_LBUTTONUP_WEIGHT:
		if weightDragging {
			from, to := weightDragFrom, weightDragTarget
			x := int32(int16(lp & 0xffff))
			y := int32(int16((lp >> 16) & 0xffff))
			if found := weightRowAt(x, y); found >= 0 {
				to = found
			}
			weightCancelDrag()
			weightDropRow(from, to)
			return 0
		}
	}
	r, _, _ := pCallWindowProcWeight.Call(weightOldListProc, w, m, wp, lp)
	return r
}
func weightCancelDrag() {
	weightDragging = false
	weightDragFrom, weightDragTarget = -1, -1
	pReleaseCaptureWeight.Call()
}
func weightContextMenu(control, packed uintptr) bool {
	if activePage != "weight" {
		return false
	}
	if control != controls[902] && control != controls[903] {
		return false
	}
	pt := point{}
	keyboard := packed == ^uintptr(0)
	if keyboard {
		pGetCursorPosWeight.Call(uintptr(unsafe.Pointer(&pt)))
	} else {
		pt.X = int32(int16(packed & 0xffff))
		pt.Y = int32(int16((packed >> 16) & 0xffff))
	}
	if control == controls[903] {
		row := weightSelectedRow()
		if !keyboard {
			inside := pt
			pScreenToClientWeight.Call(controls[903], uintptr(unsafe.Pointer(&inside)))
			row = weightRowAt(inside.X, inside.Y)
		}
		if row < 0 || row >= len(weightCurrent().Rows) {
			return true
		}
		currentWeightIndex = row
		weightSelectRow(row)
		menu, _, _ := pCreatePopupMenuWeight.Call()
		if menu == 0 {
			return true
		}
		defer pDestroyMenuWeight.Call(menu)
		pAppendMenuWeight.Call(menu, 0, 1, uintptr(unsafe.Pointer(ptr("Удалить строку"))))
		pAppendMenuWeight.Call(menu, 0, 2, uintptr(unsafe.Pointer(ptr("Дублировать строку"))))
		pAppendMenuWeight.Call(menu, 0, 3, uintptr(unsafe.Pointer(ptr("Переместить вверх"))))
		pAppendMenuWeight.Call(menu, 0, 4, uintptr(unsafe.Pointer(ptr("Переместить вниз"))))
		action, _, _ := pTrackPopupMenuWeight.Call(menu, 0x100|0x2, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
		switch action {
		case 1:
			weightRemove()
		case 2:
			weightDuplicate()
		case 3:
			weightMove(-1)
		case 4:
			weightMove(1)
		}
		return true
	}
	inside := pt
	pScreenToClientWeight.Call(controls[902], uintptr(unsafe.Pointer(&inside)))
	result := send(902, LB_ITEMFROMPOINT, 0, uintptr(uint16(inside.X))|(uintptr(uint16(inside.Y))<<16))
	if result>>16 != 0 {
		return true
	}
	idx := int(result & 0xffff)
	if idx < 0 || idx >= len(shownMaterials) {
		return true
	}
	send(902, LB_SETCURSEL, uintptr(idx), 0)
	m := shownMaterials[idx]
	if !isPersonalWeightMaterial(m) {
		return true
	}
	menu, _, _ := pCreatePopupMenuWeight.Call()
	if menu == 0 {
		return true
	}
	defer pDestroyMenuWeight.Call(menu)
	pAppendMenuWeight.Call(menu, 0, 1, uintptr(unsafe.Pointer(ptr("Изменить мою запись"))))
	pAppendMenuWeight.Call(menu, 0, 2, uintptr(unsafe.Pointer(ptr("Удалить мою запись"))))
	action, _, _ := pTrackPopupMenuWeight.Call(menu, 0x100|0x2, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
	switch action {
	case 1:
		weightStartCatalogEdit(m)
	case 2:
		weightDeletePersonalMaterial(m)
	}
	return true
}
func weightDeletePersonalMaterial(m material) {
	prompt := "Удалить свою запись «" + m.Name + "» из справочника?"
	yes, _, _ := pMsgBox.Call(hwnd, uintptr(unsafe.Pointer(ptr(prompt))), uintptr(unsafe.Pointer(ptr("Удаление записи"))), MB_YESNO|MB_ICONWARNING)
	if yes != IDYES {
		return
	}
	for i, x := range catalogUser.Items {
		if x.ID == m.ID && x.CategoryID == m.CategoryID {
			catalogUser.Items = append(catalogUser.Items[:i], catalogUser.Items[i+1:]...)
			if e := weightSaveCustom(); e != nil {
				message(e.Error(), true)
				return
			}
			weightFillMaterials()
			return
		}
	}
}
