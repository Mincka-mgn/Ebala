//go:build windows

package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

// Maps in appendix E are raster illustrations, NOT a georeferenced cadastral layer.
// Incomplete direct image sources fall back to the complete SP20 PDF in a browser.
const completeMapPDF = "https://fkr-spb.ru/upload/iblock/264/f0xpu4t46rz5z4xxlcu2r091qxnj0i6j.pdf"
const snowMapsPage = "https://lsk-lskos.ru/sr"
const windMapsPage = "https://lsk-lskos.ru/vr"

type mapSpec struct {
	key, button, title string
	page               string
	urls               []string
}

func remoteMap(folder, name string) []string {
	// Try original resolution first; published thumbnail is a verified fallback.
	base := "https://lsk-lskos.ru/1/" + folder + "/" + folder + "-m/"
	return []string{base + name + ".jpg", base + name + "-m.jpg"}
}

var mapCatalog = [][]mapSpec{
	{
		{"1", "Общая 1", "Снег · карта 1 · Россия", snowMapsPage, remoteMap("sr", "Карта_1_РФ")},
		{"1a", "1а · Сахалин", "Снег · карта 1а · Сахалин", snowMapsPage, remoteMap("sr", "Карта_1а_Сахалин")},
		{"1b", "1б · Крым", "Снег · карта 1б · Крым", snowMapsPage, remoteMap("sr", "Карта_1б_Крым")},
		{"1v", "1в · новые регионы", "Снег · карта 1в · дополнение", completeMapPDF, nil},
	},
	{
		{"2", "Общая 2", "Ветер · карта 2 · Россия", windMapsPage, remoteMap("vr", "Карта_2_РФ")},
		{"2a", "2а · Приморье", "Ветер · карта 2а · Приморье и Сахалин", windMapsPage, remoteMap("vr", "Карта_2а_Сахалин")},
		{"2b", "2б · Камчатка", "Ветер · карта 2б · Камчатка", windMapsPage, remoteMap("vr", "Карта_2б_Камчатка")},
		{"2v", "2в · Кольский", "Ветер · карта 2в · Кольский полуостров", windMapsPage, remoteMap("vr", "Карта_2в_Кольский_полуостров")},
		{"2g", "2г · Кавказ", "Ветер · карта 2г · Кавказ", windMapsPage, remoteMap("vr", "Карта_2г_Кавказ")},
		{"2d", "2д · Калининград", "Ветер · карта 2д · Калининградская область", windMapsPage, remoteMap("vr", "Карта_2д_Калининград")},
		{"2e", "2е · Крым", "Ветер · карта 2е · Республика Крым", windMapsPage, remoteMap("vr", "Карта_2_Крым")},
		{"2zh", "2ж · новые регионы", "Ветер · карта 2ж · дополнение", completeMapPDF, nil},
	},
	{
		{"3", "Общая 3", "Гололёд · карта 3 · Россия", completeMapPDF, nil},
		{"3a", "3а · Кавказ", "Гололёд · карта 3а · Горный Кавказ", completeMapPDF, nil},
		{"3b", "3б · Крым", "Гололёд · карта 3б · Крым", completeMapPDF, nil},
		{"3v", "3в · Приморье", "Гололёд · карта 3в · Приморье и Сахалин", completeMapPDF, nil},
		{"3g", "3г · 200 м", "Гололёд · карта 3г · север ЕТР, высота 200 м", completeMapPDF, nil},
		{"3d", "3д · 300 м", "Гололёд · карта 3д · север ЕТР, высота 300 м", completeMapPDF, nil},
		{"3e", "3е · 400 м", "Гололёд · карта 3е · север ЕТР, высота 400 м", completeMapPDF, nil},
		{"3zh", "3ж · новые регионы", "Гололёд · карта 3ж · дополнение", completeMapPDF, nil},
	},
}

var (
	mapHwnd                                                     uintptr
	mapFamily, mapIndex                                         int
	mapBmp                                                      uintptr
	mapW, mapH                                                  int32
	mapZoom                                                     = 1.0
	mapFocusX, mapFocusY                                        = .5, .5
	mapPanX, mapPanY                                            float64
	mapDrag                                                     bool
	dragLastX, dragLastY                                        int32
	mapTabs                                                     []uintptr
	mapInfo, mapFolderInfo, mapImport, mapSource                uintptr
	mapSelectFolder, mapOpenFolder, mapDefaultFolder, mapReload uintptr
	mapStatus                                                   = "Откройте карту из файла или нажмите «Источник»."
	pLoadImage                                                  = u32.NewProc("LoadImageW")
	pInvalidateRect                                             = u32.NewProc("InvalidateRect")
	pBeginPaint                                                 = u32.NewProc("BeginPaint")
	pEndPaint                                                   = u32.NewProc("EndPaint")
	pGetClientRect                                              = u32.NewProc("GetClientRect")
	pGetObject                                                  = g32.NewProc("GetObjectW")
	pCreateCompatibleDC                                         = g32.NewProc("CreateCompatibleDC")
	pSelectObject                                               = g32.NewProc("SelectObject")
	pStretchBlt                                                 = g32.NewProc("StretchBlt")
	pDeleteDC                                                   = g32.NewProc("DeleteDC")
	pDeleteObject                                               = g32.NewProc("DeleteObject")
	pSetStretchBltMode                                          = g32.NewProc("SetStretchBltMode")
	pSetCapture                                                 = u32.NewProc("SetCapture")
	pReleaseCapture                                             = u32.NewProc("ReleaseCapture")
	pShellExecute                                               = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")
	pBrowseForFolder                                            = syscall.NewLazyDLL("shell32.dll").NewProc("SHBrowseForFolderW")
	pGetPathFromIDList                                          = syscall.NewLazyDLL("shell32.dll").NewProc("SHGetPathFromIDListW")
	pCoTaskMemFree                                              = syscall.NewLazyDLL("ole32.dll").NewProc("CoTaskMemFree")
	pCoInitializeEx                                             = syscall.NewLazyDLL("ole32.dll").NewProc("CoInitializeEx")
	pCoUninitialize                                             = syscall.NewLazyDLL("ole32.dll").NewProc("CoUninitialize")
)

const (
	WM_PAINT            = 0xF
	WM_SIZE             = 0x5
	WM_MOUSEWHEEL       = 0x20A
	WM_LBUTTONDOWN      = 0x201
	WM_LBUTTONUP        = 0x202
	WM_MOUSEMOVE        = 0x200
	WM_KEYDOWN          = 0x100
	WM_CLOSE            = 0x10
	IMAGE_BITMAP        = 0
	LR_LOADFROMFILE     = 0x0010
	LR_CREATEDIBSECTION = 0x2000
	SRCCOPY             = 0x00CC0020
	HALFTONE            = 4
	mapTabFirst         = 1800
	mapImportID         = 1890
	mapSourceID         = 1891
	mapSelectFolderID   = 1892
	mapOpenFolderID     = 1893
	mapDefaultFolderID  = 1894
	mapReloadID         = 1895
	mapTop              = 135
)

type winBitmap struct {
	Type, Width, Height, WidthBytes int32
	Planes, BitsPixel               uint16
	Bits                            uintptr
}
type winPaint struct {
	HDC                uintptr
	Erase              int32
	RC                 winRect
	Restore, IncUpdate int32
	Reserved           [32]byte
}
type browseInfo struct {
	Owner, Root        uintptr
	DisplayName, Title *uint16
	Flags              uint32
	Callback, Param    uintptr
	Image              int32
}

func browseMapFolder() (string, error) {
	// SHBrowseForFolder is available in all supported Windows versions.
	hr, _, _ := pCoInitializeEx.Call(0, 2) // COINIT_APARTMENTTHREADED
	if hr == 0 || hr == 1 {
		defer pCoUninitialize.Call()
	}
	var display [260]uint16
	bi := browseInfo{Owner: mapHwnd, DisplayName: &display[0], Title: ptr("Выберите папку для нормативных карт"), Flags: 0x1 | 0x40}
	pidl, _, _ := pBrowseForFolder.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return "", nil
	}
	defer pCoTaskMemFree.Call(pidl)
	var path [32768]uint16
	r, _, e := pGetPathFromIDList.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if r == 0 {
		return "", fmt.Errorf("не удалось получить путь к папке: %v", e)
	}
	return syscall.UTF16ToString(path[:]), nil
}
func rasterExt(data []byte) string {
	if len(data) > 2 && string(data[:2]) == "BM" {
		return ".bmp"
	}
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return ".jpg"
	}
	if len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n" {
		return ".png"
	}
	if len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a") {
		return ".gif"
	}
	return ""
}
func mapBMPForViewer(source string) (string, error) {
	raw, e := os.ReadFile(source)
	if e != nil {
		return "", e
	}
	if len(raw) > 30<<20 {
		return "", errors.New("файл карты слишком велик (более 30 МБ)")
	}
	converted, e := toBitmap(raw)
	if e != nil {
		return "", e
	}
	if rasterExt(raw) == ".bmp" {
		return source, nil
	}
	dir := filepath.Join(os.TempDir(), "Poverka_Nagruzki_MapRenderer")
	if e = os.MkdirAll(dir, 0755); e != nil {
		return "", e
	}
	target := filepath.Join(dir, fmt.Sprintf("map_%d_%s.bmp", mapFamily, currentMap().key))
	if e = os.WriteFile(target, converted, 0644); e != nil {
		return "", e
	}
	return target, nil
}
func loadMapSource(path string) error {
	bmp, e := mapBMPForViewer(path)
	if e != nil {
		return e
	}
	return loadBitmapIntoViewer(bmp)
}
func bitmap24(img image.Image) ([]byte, error) {
	rect := img.Bounds()
	w, h := rect.Dx(), rect.Dy()
	if w < 100 || h < 100 || w > 12000 || h > 12000 || int64(w)*int64(h) > 45_000_000 {
		return nil, errors.New("недопустимые размеры изображения")
	}
	stride := (w*3 + 3) &^ 3
	size := 54 + stride*h
	b := make([]byte, size)
	b[0], b[1] = 'B', 'M'
	binary.LittleEndian.PutUint32(b[2:], uint32(size))
	binary.LittleEndian.PutUint32(b[10:], 54)
	binary.LittleEndian.PutUint32(b[14:], 40)
	binary.LittleEndian.PutUint32(b[18:], uint32(w))
	binary.LittleEndian.PutUint32(b[22:], uint32(h))
	binary.LittleEndian.PutUint16(b[26:], 1)
	binary.LittleEndian.PutUint16(b[28:], 24)
	binary.LittleEndian.PutUint32(b[34:], uint32(stride*h))
	for y := 0; y < h; y++ {
		pos := 54 + (h-1-y)*stride
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(rect.Min.X+x, rect.Min.Y+y).RGBA()
			b[pos], b[pos+1], b[pos+2] = byte(bl>>8), byte(g>>8), byte(r>>8)
			pos += 3
		}
	}
	return b, nil
}
func toBitmap(data []byte) ([]byte, error) {
	// Content-based check rather than trusting a .jpg extension or HTTP header.
	trimmed := bytes.ToLower(bytes.TrimSpace(data))
	if len(trimmed) == 0 {
		return nil, errors.New("источник вернул пустой файл")
	}
	if bytes.HasPrefix(trimmed, []byte("<!doctype")) || bytes.HasPrefix(trimmed, []byte("<html")) || bytes.HasPrefix(trimmed, []byte("<?xml")) {
		return nil, errors.New("источник прислал HTML/XML вместо изображения")
	}
	if len(data) > 2 && string(data[:2]) == "BM" {
		return data, nil
	}
	cfg, _, cfgErr := image.DecodeConfig(bytes.NewReader(data))
	if cfgErr != nil {
		return nil, fmt.Errorf("формат карты не распознан: %v", cfgErr)
	}
	if cfg.Width < 100 || cfg.Height < 100 || cfg.Width > 12000 || cfg.Height > 12000 || int64(cfg.Width)*int64(cfg.Height) > 45_000_000 {
		return nil, errors.New("недопустимый размер карты")
	}
	im, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("файл не является поддерживаемым JPG/PNG/GIF/BMP: %v", err)
	}
	if format == "" {
		return nil, errors.New("формат изображения не определён")
	}
	return bitmap24(im)
}
func currentMap() mapSpec { return mapCatalog[mapFamily][mapIndex] }

// Карты пользователь добавляет локально. Сетевые загрузки отключены полностью.
func fetchMap(spec mapSpec) (string, error) {
	root, err := configuredMapRoot()
	if err != nil {
		return "", err
	}
	if err = initMapsDirectory(root); err != nil {
		return "", fmt.Errorf("папка карт недоступна: %w", err)
	}
	if local, e := findLocalMap(root, mapFamily, spec.key); e == nil {
		return local, nil
	}
	return "", fmt.Errorf("карта %s отсутствует в %s; добавьте JPG/PNG через папку Maps или кнопку «Из файла…»", spec.key, root)
}
func loadBitmapIntoViewer(path string) error {
	h, _, err := pLoadImage.Call(0, uintptr(unsafe.Pointer(ptr(path))), IMAGE_BITMAP, 0, 0, LR_LOADFROMFILE|LR_CREATEDIBSECTION)
	if h == 0 {
		return fmt.Errorf("Windows не открыла BMP: %v", err)
	}
	var bmp winBitmap
	if r, _, _ := pGetObject.Call(h, uintptr(unsafe.Sizeof(bmp)), uintptr(unsafe.Pointer(&bmp))); r == 0 {
		pDeleteObject.Call(h)
		return errors.New("не удалось определить размер изображения")
	}
	if mapBmp != 0 {
		pDeleteObject.Call(mapBmp)
	}
	mapBmp = h
	mapW, mapH = bmp.Width, bmp.Height
	return nil
}
func roughRegionFocus(region string) (float64, float64, float64) {
	s := strings.ToLower(region)
	switch {
	case strings.Contains(s, "краснодар"), strings.Contains(s, "адыге"), strings.Contains(s, "ростов"), strings.Contains(s, "ставропол"), strings.Contains(s, "крым"):
		return .20, .73, 2.7
	case strings.Contains(s, "моск"), strings.Contains(s, "тул"), strings.Contains(s, "калуг"), strings.Contains(s, "рязан"), strings.Contains(s, "владимир"), strings.Contains(s, "твер"):
		return .23, .56, 2.5
	case strings.Contains(s, "санкт"), strings.Contains(s, "ленинград"), strings.Contains(s, "псков"), strings.Contains(s, "новгород"):
		return .19, .41, 2.6
	case strings.Contains(s, "татар"), strings.Contains(s, "башкор"), strings.Contains(s, "самар"), strings.Contains(s, "перм"):
		return .31, .56, 2.6
	case strings.Contains(s, "урал"), strings.Contains(s, "челябин"), strings.Contains(s, "свердлов"), strings.Contains(s, "оренбург"), strings.Contains(s, "курган"):
		return .38, .58, 2.5
	case strings.Contains(s, "тюмен"), strings.Contains(s, "омск"), strings.Contains(s, "новосибир"), strings.Contains(s, "томск"):
		return .48, .57, 2.2
	case strings.Contains(s, "краснояр"), strings.Contains(s, "кемеров"), strings.Contains(s, "хакас"):
		return .60, .58, 2.0
	case strings.Contains(s, "иркут"), strings.Contains(s, "бурят"), strings.Contains(s, "забайкал"):
		return .68, .67, 2.0
	case strings.Contains(s, "хабаров"), strings.Contains(s, "примор"), strings.Contains(s, "амурск"), strings.Contains(s, "сахалин"):
		return .86, .73, 2.4
	case strings.Contains(s, "якут"), strings.Contains(s, "магадан"), strings.Contains(s, "чукот"):
		return .78, .34, 2.0
	}
	return .5, .5, 1
}
func mapSiteForView() Site {
	if activePage == "site" && selectedPlace >= 0 && selectedPlace < len(foundPlaces) {
		return draftSite
	}
	return project.Site
}
func initialMapIndex(family int) int {
	region := strings.ToLower(mapSiteForView().Region)
	for _, tc := range []struct {
		text  string
		index [3]int
	}{
		{"крым", [3]int{2, 6, 2}}, {"сахалин", [3]int{1, 1, 3}},
		{"примор", [3]int{0, 1, 3}}, {"камчат", [3]int{0, 2, 0}},
		{"мурман", [3]int{0, 3, 0}}, {"краснодар", [3]int{0, 4, 1}},
		{"адыге", [3]int{0, 4, 1}}, {"калининград", [3]int{0, 5, 0}},
		{"чеч", [3]int{0, 4, 1}}, {"кабардино", [3]int{0, 4, 1}},
	} {
		if strings.Contains(region, tc.text) {
			return tc.index[family]
		}
	}
	return 0
}
func viewerTitle() string {
	s := mapSiteForView()
	place := s.Place
	if place == "" {
		place = s.District
	}
	if place == "" {
		place = s.Region
	}
	if place == "" {
		place = "местность не выбрана"
	}
	return "Ebala · " + currentMap().title + " · " + place + " · карты СП 20.13330.2016, приложение Е"
}
func addMapButton(id int, text string, x, y, w, h int) uintptr {
	b, _, _ := pCreate.Call(0, uintptr(unsafe.Pointer(ptr("BUTTON"))), uintptr(unsafe.Pointer(ptr(text))), WS_CHILD|WS_VISIBLE, uintptr(x), uintptr(y), uintptr(w), uintptr(h), mapHwnd, uintptr(id), 0, 0)
	if font != 0 {
		pSend.Call(b, WM_SETFONT, font, 1)
	}
	return b
}
func mapFolderStatus() string {
	root, e := configuredMapRoot()
	if e != nil {
		return "Папка карт: ошибка: " + e.Error()
	}
	return "Папка карт: " + root + "  (Snow / Wind / Ice)"
}
func refreshMapControls() {
	for _, b := range mapTabs {
		pDestroy.Call(b)
	}
	mapTabs = nil
	specs := mapCatalog[mapFamily]
	x := 10
	for i, s := range specs {
		title := s.button
		if i == mapIndex {
			title = "● " + title
		}
		width := 115
		if mapFamily == 0 {
			width = 155
		}
		if mapFamily == 2 {
			width = 128
		}
		b := addMapButton(mapTabFirst+i, title, x, 8, width-5, 34)
		mapTabs = append(mapTabs, b)
		x += width
	}
	for _, b := range []uintptr{mapImport, mapSource, mapSelectFolder, mapOpenFolder, mapDefaultFolder, mapReload, mapInfo, mapFolderInfo} {
		if b != 0 {
			pDestroy.Call(b)
		}
	}
	mapImport = addMapButton(mapImportID, "Из файла…", 10, 50, 135, 30)
	mapSelectFolder = addMapButton(mapSelectFolderID, "Выбрать папку…", 152, 50, 156, 30)
	mapOpenFolder = addMapButton(mapOpenFolderID, "Открыть папку", 316, 50, 157, 30)
	mapDefaultFolder = addMapButton(mapDefaultFolderID, "По умолчанию", 481, 50, 154, 30)
	mapReload = addMapButton(mapReloadID, "Перечитать карту", 643, 50, 157, 30)
	mapSource = addMapButton(mapSourceID, "Источник ↗", 809, 50, 141, 30)
	mapFolderInfo, _, _ = pCreate.Call(0, uintptr(unsafe.Pointer(ptr("STATIC"))), uintptr(unsafe.Pointer(ptr(mapFolderStatus()))), WS_CHILD|WS_VISIBLE, 10, 89, 1370, 20, mapHwnd, 0, 0, 0)
	mapInfo, _, _ = pCreate.Call(0, uintptr(unsafe.Pointer(ptr("STATIC"))), uintptr(unsafe.Pointer(ptr(mapStatus))), WS_CHILD|WS_VISIBLE, 10, 111, 1370, 20, mapHwnd, 0, 0, 0)
	if font != 0 {
		pSend.Call(mapInfo, WM_SETFONT, font, 1)
		pSend.Call(mapFolderInfo, WM_SETFONT, font, 1)
	}
	pSetText.Call(mapHwnd, uintptr(unsafe.Pointer(ptr(viewerTitle()))))
}
func changeMap(index int) {
	if index < 0 || index >= len(mapCatalog[mapFamily]) {
		return
	}
	mapIndex = index
	if mapBmp != 0 {
		pDeleteObject.Call(mapBmp)
		mapBmp = 0
	}
	path, err := fetchMap(currentMap())
	if err != nil {
		mapStatus = "Нет карты локально: " + err.Error() + ". Выберите «Из файла…» или «Источник»"
	} else if e := loadMapSource(path); e != nil {
		mapStatus = "Ошибка чтения " + path + ": " + e.Error()
	} else {
		mapStatus = fmt.Sprintf("Открыта %s (%d × %d пикс.). Колесо — масштаб, ЛКМ — перемещение", path, mapW, mapH)
	}
	if index == 0 {
		mapFocusX, mapFocusY, mapZoom = roughRegionFocus(mapSiteForView().Region)
	} else {
		mapFocusX, mapFocusY, mapZoom = .5, .5, 1
	}
	mapPanX, mapPanY = 0, 0
	refreshMapControls()
	pInvalidateRect.Call(mapHwnd, 0, 1)
}
func openMapViewer(family int) {
	if family < 0 || family >= len(mapCatalog) {
		return
	}
	if mapHwnd == 0 {
		h, _, _ := pGetModule.Call(0)
		cname := ptr("PoverkaAppendixEViewerV06")
		wc := wndClassEx{Size: uint32(unsafe.Sizeof(wndClassEx{})), WndProc: syscall.NewCallback(mapWndProc), Instance: h, Background: 6, ClassName: cname}
		pRegister.Call(uintptr(unsafe.Pointer(&wc)))
		mapHwnd, _, _ = pCreate.Call(0, uintptr(unsafe.Pointer(cname)), uintptr(unsafe.Pointer(ptr("Карты приложения Е"))), WS_OVERLAPPEDWINDOW, 110, 40, 1420, 900, 0, 0, h, 0)
		if mapHwnd == 0 {
			message("Не удалось создать окно карт", true)
			return
		}
	}
	mapFamily = family
	changeMap(initialMapIndex(family))
	pShow.Call(mapHwnd, 1)
	pUpdate.Call(mapHwnd)
}
func openMapSource() {
	page := currentMap().page
	if page == "" {
		page = completeMapPDF
	}
	pShellExecute.Call(mapHwnd, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr(page))), 0, 0, 1)
}
func importMapFromFile() {
	buf := make([]uint16, 1024)
	filter := append(utf16.Encode([]rune("Изображения JPG, PNG, GIF, BMP\x00*.jpg;*.jpeg;*.png;*.gif;*.bmp\x00Все файлы\x00*.*\x00")), 0)
	f := openFileName{Size: uint32(unsafe.Sizeof(openFileName{})), Owner: mapHwnd, Filter: &filter[0], File: &buf[0], MaxFile: uint32(len(buf)), Flags: OFN_EXPLORER | OFN_PATHMUSTEXIST | OFN_FILEMUSTEXIST}
	r, _, _ := pGetOpen.Call(uintptr(unsafe.Pointer(&f)))
	if r == 0 {
		return
	}
	path := syscall.UTF16ToString(buf)
	raw, e := os.ReadFile(path)
	if e != nil {
		message(e.Error(), true)
		return
	}
	if len(raw) > 30<<20 {
		message("Файл карты больше 30 МБ", true)
		return
	}
	if _, e = toBitmap(raw); e != nil {
		message(e.Error(), true)
		return
	}
	ext := rasterExt(raw)
	if ext == "" {
		message("Неизвестный формат изображения", true)
		return
	}
	root, e := configuredMapRoot()
	if e != nil {
		message(e.Error(), true)
		return
	}
	_, e = installLocalMap(root, mapFamily, currentMap().key, ext, raw)
	if e != nil {
		message("Не удалось сохранить карту: "+e.Error(), true)
		return
	}
	changeMap(mapIndex)
}
func chooseMapFolder() {
	p, e := browseMapFolder()
	if e != nil {
		message(e.Error(), true)
		return
	}
	if p == "" {
		return
	}
	if e = saveMapRoot(p); e != nil {
		message("Не удалось выбрать папку карт: "+e.Error(), true)
		return
	}
	changeMap(mapIndex)
}
func showMapFolder() {
	p, e := configuredMapRoot()
	if e != nil {
		message(e.Error(), true)
		return
	}
	if e = initMapsDirectory(p); e != nil {
		message(e.Error(), true)
		return
	}
	pShellExecute.Call(mapHwnd, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr(p))), 0, 0, 1)
}
func resetMapFolder() {
	_, e := resetMapRoot()
	if e != nil {
		message(e.Error(), true)
		return
	}
	changeMap(mapIndex)
}
func mapWndProc(w, m, wp, lp uintptr) uintptr {
	switch m {
	case WM_COMMAND:
		id := int(wp & 0xffff)
		if id >= mapTabFirst && id < mapTabFirst+len(mapCatalog[mapFamily]) {
			changeMap(id - mapTabFirst)
		}
		if id == mapImportID {
			importMapFromFile()
		}
		if id == mapSourceID {
			openMapSource()
		}
		if id == mapSelectFolderID {
			chooseMapFolder()
		}
		if id == mapOpenFolderID {
			showMapFolder()
		}
		if id == mapDefaultFolderID {
			resetMapFolder()
		}
		if id == mapReloadID {
			changeMap(mapIndex)
		}
		return 0
	case WM_PAINT:
		var ps winPaint
		dc, _, _ := pBeginPaint.Call(w, uintptr(unsafe.Pointer(&ps)))
		if mapBmp != 0 && mapW > 0 && mapH > 0 {
			var r winRect
			pGetClientRect.Call(w, uintptr(unsafe.Pointer(&r)))
			cw, ch := float64(r.Right-r.Left), float64(r.Bottom-r.Top-mapTop)
			if cw > 0 && ch > 0 {
				scale := math.Min(cw/float64(mapW), ch/float64(mapH)) * mapZoom
				dw, dh := float64(mapW)*scale, float64(mapH)*scale
				x := cw*.5 - mapFocusX*dw + mapPanX
				y := mapTop + ch*.5 - mapFocusY*dh + mapPanY
				md, _, _ := pCreateCompatibleDC.Call(dc)
				old, _, _ := pSelectObject.Call(md, mapBmp)
				pSetStretchBltMode.Call(dc, HALFTONE)
				pStretchBlt.Call(dc, uintptr(int32(x)), uintptr(int32(y)), uintptr(int32(dw)), uintptr(int32(dh)), md, 0, 0, uintptr(mapW), uintptr(mapH), SRCCOPY)
				pSelectObject.Call(md, old)
				pDeleteDC.Call(md)
			}
		}
		pEndPaint.Call(w, uintptr(unsafe.Pointer(&ps)))
		return 0
	case WM_MOUSEWHEEL:
		delta := int16((wp >> 16) & 0xffff)
		if delta > 0 {
			mapZoom *= 1.2
		} else {
			mapZoom /= 1.2
		}
		mapZoom = math.Max(.7, math.Min(12, mapZoom))
		pInvalidateRect.Call(w, 0, 1)
		return 0
	case WM_LBUTTONDOWN:
		if int16((lp>>16)&0xffff) < mapTop {
			return 0
		}
		mapDrag = true
		dragLastX = int32(int16(lp & 0xffff))
		dragLastY = int32(int16((lp >> 16) & 0xffff))
		pSetCapture.Call(w)
		return 0
	case WM_MOUSEMOVE:
		if mapDrag {
			x := int32(int16(lp & 0xffff))
			y := int32(int16((lp >> 16) & 0xffff))
			mapPanX += float64(x - dragLastX)
			mapPanY += float64(y - dragLastY)
			dragLastX, dragLastY = x, y
			pInvalidateRect.Call(w, 0, 1)
		}
		return 0
	case WM_LBUTTONUP:
		mapDrag = false
		pReleaseCapture.Call()
		return 0
	case WM_SIZE:
		pInvalidateRect.Call(w, 0, 1)
		return 0
	case WM_CLOSE:
		pShow.Call(w, 0)
		return 0
	}
	ret, _, _ := pDefWndProc.Call(w, m, wp, lp)
	return ret
}
