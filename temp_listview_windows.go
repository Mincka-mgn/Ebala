//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

const (
	lvsReport                   = 0x1
	lvsShowSelAlways            = 0x8
	lvmFirst                    = 0x1000
	lvmInsertColumnW            = 0x1061
	lvmInsertItemW              = 0x104D
	lvmSetItemTextW             = 0x1074
	lvmDeleteAllItems           = 0x1009
	lvmGetNextItem              = 0x100C
	lvniSelected                = 2
	lvmSetExtendedListViewStyle = 0x1036
	lvsExFullRowSelect          = 0x20
	lvsExGridLines              = 1
)

type lvColumn struct {
	mask         uint32
	fmt          int32
	cx           int32
	text         *uint16
	textMax      int32
	subItem      int32
	image        int32
	order        int32
	minWidth     int32
	defaultWidth int32
	idealWidth   int32
}
type lvItem struct {
	mask      uint32
	iItem     int32
	iSubItem  int32
	state     uint32
	stateMask uint32
	text      *uint16
	textMax   int32
	image     int32
	lParam    uintptr
	indent    int32
	groupID   int32
	columns   uint32
	puColumns uintptr
}
type tempNotifyHeader struct {
	hwnd uintptr
	id   uintptr
	code int32
}

var initComctl = syscall.NewLazyDLL("comctl32.dll").NewProc("InitCommonControls")

func tempTable(id, x, y, w, h int, columns []string, widths []int) {
	initComctl.Call()
	v := create("SysListView32", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|lvsReport|lvsShowSelAlways|0x0400, x, y, w, h, id)
	tempColumnCount[id] = len(columns)
	rowHeight := 54
	if id == reportCompositionList {
		rowHeight = 31
	}
	im, _, _ := tempImageListCreate.Call(1, uintptr(rowHeight), 0x20, 1, 1)
	pSend.Call(v, 0x1003, 1, im) // LVSIL_SMALL: 54px rows, enough for wrapped text
	pSend.Call(v, lvmSetExtendedListViewStyle, uintptr(lvsExFullRowSelect|lvsExGridLines), uintptr(lvsExFullRowSelect|lvsExGridLines))
	for i, s := range columns {
		c := lvColumn{mask: 0x1 | 0x2 | 0x4, cx: int32(widths[i]), text: ptr(s), subItem: int32(i)}
		pSend.Call(v, lvmInsertColumnW, uintptr(i), uintptr(unsafe.Pointer(&c)))
	}
}
func tempRows(id int, rows [][]string) {
	if controls[id] == 0 {
		return
	}
	tempRowsCache[id] = rows
	send(id, lvmDeleteAllItems, 0, 0)
	for i, values := range rows {
		if len(values) == 0 {
			continue
		}
		s := ptr(values[0])
		v := lvItem{mask: 1, iItem: int32(i), text: s}
		send(id, lvmInsertItemW, 0, uintptr(unsafe.Pointer(&v)))
		for j := 1; j < len(values); j++ {
			value := ptr(values[j])
			sub := lvItem{iSubItem: int32(j), text: value}
			send(id, lvmSetItemTextW, uintptr(i), uintptr(unsafe.Pointer(&sub)))
		}
	}
}
func tempSelectFirst(id int) {
	if controls[id] == 0 {
		return
	}
	v := lvItem{state: 3, stateMask: 3}
	send(id, 0x102B, 0, uintptr(unsafe.Pointer(&v)))
}
func tempSelected(id int) int {
	if controls[id] == 0 {
		return -1
	}
	v := send(id, lvmGetNextItem, ^uintptr(0), lvniSelected)
	if int32(v) < 0 {
		return -1
	}
	return int(v)
}

// Table cells are painted with line wrapping. Column headers retain native
// resizing and reordering behavior; no fixed-width text separators are used.
type tempRect struct{ Left, Top, Right, Bottom int32 }
type tempDrawItem struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HwndItem   uintptr
	HDC        uintptr
	ItemRect   tempRect
	ItemData   uintptr
}

var tempRowsCache = map[int][][]string{}
var tempColumnCount = map[int]int{}
var tempImageListCreate = syscall.NewLazyDLL("comctl32.dll").NewProc("ImageList_Create")
var tempDrawText = u32.NewProc("DrawTextW")
var tempBrush = u32.NewProc("GetSysColorBrush")
var tempFillRect = u32.NewProc("FillRect")
var tempSetTextColor = g32.NewProc("SetTextColor")
var tempSetBkMode = g32.NewProc("SetBkMode")
var tempSelectObject = g32.NewProc("SelectObject")
var tempDrawLine = g32.NewProc("MoveToEx")
var tempLineTo = g32.NewProc("LineTo")
var tempCreateSolidBrush = g32.NewProc("CreateSolidBrush")
var tempRoundRect = g32.NewProc("RoundRect")
var tempDeleteObject = g32.NewProc("DeleteObject")

func tempWrapPaint(lp uintptr) bool {
	if lp == 0 {
		return false
	}
	ds := (*tempDrawItem)(unsafe.Pointer(lp))
	id := int(ds.CtlID)
	data, ok := tempRowsCache[id]
	if !ok {
		return false
	}
	index := int(ds.ItemID)
	if index < 0 || index >= len(data) {
		return true
	}
	selected := ds.ItemState&1 != 0
	bg := uintptr(5)
	fg := uintptr(0x000000)
	if selected {
		bg = 13
		fg = 0xFFFFFF
	}
	brush, _, _ := tempBrush.Call(bg)
	rect := ds.ItemRect
	tempFillRect.Call(ds.HDC, uintptr(unsafe.Pointer(&rect)), brush)
	tempSetTextColor.Call(ds.HDC, fg)
	tempSetBkMode.Call(ds.HDC, 1)
	if font != 0 {
		old, _, _ := tempSelectObject.Call(ds.HDC, font)
		defer tempSelectObject.Call(ds.HDC, old)
	}
	left := rect.Left
	values := data[index]
	for j := 0; j < len(values); j++ {
		cw, _, _ := pSend.Call(ds.HwndItem, 0x101D, uintptr(j), 0) // LVM_GETCOLUMNWIDTH
		width := int32(cw)
		if width <= 0 {
			continue
		}
		r := tempRect{Left: left + 5, Top: rect.Top + 4, Right: left + width - 6, Bottom: rect.Bottom - 3}
		if r.Right > rect.Right {
			r.Right = rect.Right - 4
		}
		if r.Right > r.Left {
			if ((tempSummaryControlID(id) || id == 903) && j == len(values)-1) || (id == reportCompositionList && (j == 3 || j == 4) && compositionIDAt(index) > 0) {
				// Neutral gray control area, visually separated from load cells.
				gray, _, _ := tempCreateSolidBrush.Call(0x00E8E8E8)
				bgRect := tempRect{Left: left, Top: rect.Top, Right: left + width, Bottom: rect.Bottom}
				if bgRect.Right > rect.Right {
					bgRect.Right = rect.Right
				}
				tempFillRect.Call(ds.HDC, uintptr(unsafe.Pointer(&bgRect)), gray)
				tempDeleteObject.Call(gray)
				// The delete affordance is drawn as a red, visible button,
				// not as an ordinary character in a data column.
				cx := (r.Left + r.Right) / 2
				cy := (r.Top + r.Bottom) / 2
				color := uintptr(0x0000D7) // red delete everywhere
				if id == reportCompositionList && j == 3 {
					color = 0xAA6633
				} // blue edit
				fill, _, _ := tempCreateSolidBrush.Call(color)
				oldBrush, _, _ := tempSelectObject.Call(ds.HDC, fill)
				if id == reportCompositionList && j == 3 {
					tempRoundRect.Call(ds.HDC, uintptr(cx-38), uintptr(cy-12), uintptr(cx+38), uintptr(cy+12), 9, 9)
				} else {
					tempRoundRect.Call(ds.HDC, uintptr(cx-16), uintptr(cy-12), uintptr(cx+16), uintptr(cy+12), 9, 9)
				}
				tempSelectObject.Call(ds.HDC, oldBrush)
				tempDeleteObject.Call(fill)
				tempSetTextColor.Call(ds.HDC, 0xFFFFFF)
				letter := ptr("×")
				cr := tempRect{Left: cx - 15, Top: cy - 13, Right: cx + 15, Bottom: cy + 13}
				if id == reportCompositionList && j == 3 {
					letter = ptr("Править")
					cr.Left = cx - 37
					cr.Right = cx + 37
				}
				tempDrawText.Call(ds.HDC, uintptr(unsafe.Pointer(letter)), ^uintptr(0), uintptr(unsafe.Pointer(&cr)), 0x1|0x4|0x20|0x800)
				tempSetTextColor.Call(ds.HDC, fg)
			} else {
				p := ptr(values[j])
				tempDrawText.Call(ds.HDC, uintptr(unsafe.Pointer(p)), ^uintptr(0), uintptr(unsafe.Pointer(&r)), 0x10|0x800)
			}
		}
		left += width
		if left >= rect.Right {
			break
		}
	}
	return true
}
