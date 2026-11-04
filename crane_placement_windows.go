//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const (
	cranePlaceOpen = 12600 + iota
	cranePlaceSpanCount
	cranePlaceLengths
	cranePlaceRun
	cranePlaceResults
	cranePlaceDescription
	cranePlaceImage
	cranePlaceBack
	cranePlaceSave
	cranePlaceTopBack
	cranePlaceTrainInfo
	cranePlaceWord
	cranePlaceExcel
	cranePlaceScrollUp
	cranePlaceScrollDown
)

var cranePlaceResult CranePlacementResult
var cranePlaceBitmap uintptr
var cranePlaceImageFile string
var cranePlaceShowScroll = u32.NewProc("ShowScrollBar")

func cranePlaceBuild() {
	buildingPage = "temp_crane_place"
	navigation("Расстановка мостовых кранов", "Поиск опасных положений колёс на однопролётной и неразрезной балке")
	button(cranePlaceTopBack, "← К кранам", 20, 18, 157, 37)
	label("Длины пролётов, м (через пробел или ;) ", 25, 103, 350, 24)
	edit(cranePlaceLengths, "6", 384, 101, 558, 30)
	button(cranePlaceRun, "Разместить", 959, 100, 151, 32)
	label("Опоры: вертикальные перемещения закреплены, повороты свободны. EI по пролётам одинаковое.", 25, 142, 1050, 26)
	label("КОЛЁСНАЯ ГРУППА, КОТОРУЮ РАЗМЕЩАЕМ", 25, 169, 700, 23)
	create("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|ES_MULTILINE|0x40|ES_READONLY, 25, 193, 950, 45, cranePlaceTrainInfo)
	button(cranePlaceScrollUp, "▲", 984, 193, 57, 23)
	button(cranePlaceScrollDown, "▼", 1046, 193, 64, 23)
	heading("ОПАСНЫЕ ПОЛОЖЕНИЯ", 25, 245, 600, 27)
	tempTable(cranePlaceResults, 25, 274, 1085, 157, []string{"№", "Усилие и сечение", "Значение", "Координата первого колеса, мм"}, []int{45, 530, 170, 315})
	create("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_READONLY, 25, 439, 1085, 82, cranePlaceDescription)
	create("STATIC", "", WS_CHILD|WS_VISIBLE|0xE, 25, 527, 1040, 300, cranePlaceImage)
	button(cranePlaceSaveNew, "Сохранить в ведомость и новый расчёт", 25, 791, 410, 36)
	button(cranePlaceResetCurrent, "Сбросить нагрузки", 445, 791, 195, 36)
	button(cranePlaceBack, "Назад к кранам", 25, 839, 235, 36)
	button(cranePlaceSave, "Сохранить схемы картинками", 275, 839, 258, 36)
	button(cranePlaceWord, "Отчёт Word", 553, 839, 253, 36)
	button(cranePlaceExcel, "Отчёт Excel", 826, 839, 284, 36)
	buildingPage = "home"
}
func cranePlaceTrains() ([]CraneTrain, error) {
	var out []CraneTrain
	for _, s := range tempSections.Sections {
		if !tempIsCrane(s) {
			continue
		}
		if len(s.Wheels) == 0 {
			continue
		}
		coords, loads := []float64{}, []float64{}
		for _, w := range s.Wheels {
			if !w.XKnown {
				return nil, fmt.Errorf("%s: укажите расстояния между колёсами", s.Title)
			}
			coords = append(coords, w.XMM)
			loads = append(loads, w.PKN)
		}
		overall := tempNumber(s, "overall_length_mm")
		out = append(out, CraneTrain{Name: s.Title, WheelXMM: coords, ForcesKN: loads, OverallMM: overall})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("Добавьте кран с нагрузками и расстояниями между колёсами")
	}
	return out, nil
}
func cranePlaceRender(n int) {
	if n < 0 || n >= len(cranePlaceResult.Cases) {
		return
	}
	c := cranePlaceResult.Cases[n]
	lines := []string{c.Name + ": " + tempRU(c.Value, 3) + " " + c.Units, "Положение каждого колеса от левой опоры:"}
	if strings.Contains(c.Name, "М") {
		lines = append(lines, "Расчётное сечение: x="+tempRU(c.SectionMM, 1)+" мм")
	}
	for i, w := range c.Points {
		if w.XMM < 0 || w.XMM > cranePlaceTotal() {
			lines = append(lines, fmt.Sprintf("%s, колесо %d: x=%s мм, P=%s кН (за пределами балки)", w.Crane, i+1, tempRU(w.XMM, 1), tempRU(w.PKN, 2)))
		} else {
			lines = append(lines, fmt.Sprintf("%s, колесо %d: x=%s мм, P=%s кН", w.Crane, i+1, tempRU(w.XMM, 1), tempRU(w.PKN, 2)))
		}
	}
	set(cranePlaceDescription, strings.Join(lines, "\r\n"))
	cranePlaceImageFile = filepath.Join(os.TempDir(), "ebala_crane_position.bmp")
	if err := cranePictureForCase(cranePlaceResult.SpansMM, c, cranePlaceImageFile); err != nil {
		return
	}
	bmp, _, _ := tempLoadImage.Call(0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(cranePlaceImageFile))), 0, 0, 0, 0x10)
	if bmp != 0 {
		pSend.Call(controls[cranePlaceImage], 0x172, 0, bmp)
		if cranePlaceBitmap != 0 {
			tempDeleteBitmap.Call(cranePlaceBitmap)
		}
		cranePlaceBitmap = bmp
	}
}
func cranePlaceTotal() float64 {
	v := 0.
	for _, s := range cranePlaceResult.SpansMM {
		v += s
	}
	return v
}
func cranePlaceTrainDescription(trains []CraneTrain) string {
	var lines []string
	for _, c := range trains {
		parts := []string{}
		for i, p := range c.ForcesKN {
			parts = append(parts, tempRU(p, 2)+" кН")
			if i+1 < len(c.WheelXMM) {
				parts = append(parts, "— "+tempRU(c.WheelXMM[i+1]-c.WheelXMM[i], 0)+" мм —")
			}
		}
		lines = append(lines, c.Name+": "+strings.Join(parts, " "))
	}
	return strings.Join(lines, "\r\n")
}
func cranePlaceAction(id int) bool {
	switch id {
	case cranePlaceScrollUp, cranePlaceScrollDown:
		if controls[cranePlaceTrainInfo] != 0 {
			delta := int32(2)
			if id == cranePlaceScrollUp {
				delta = -2
			}
			pSend.Call(controls[cranePlaceTrainInfo], 0xB6, 0, uintptr(delta)) // EM_LINESCROLL
		}
		return true
	case cranePlaceOpen, cranePlaceOpen + 20:
		if trains, e := cranePlaceTrains(); e == nil {
			set(cranePlaceTrainInfo, cranePlaceTrainDescription(trains))
			// Explicit visible thumb: long multi-wheel trains can otherwise look truncated.
			cranePlaceShowScroll.Call(controls[cranePlaceTrainInfo], 1, 1)
		} else {
			set(cranePlaceTrainInfo, "Заполните нагрузки и расстояния на странице «Мостовые краны»")
		}
		switchPage("temp_crane_place")
		return true
	case cranePlaceWord, cranePlaceExcel:
		if len(cranePlaceResult.Cases) == 0 {
			message("Сначала рассчитайте положения кранов", true)
			return true
		}
		kind, name := "docx", "Расстановка кранов.docx"
		if id == cranePlaceExcel {
			kind, name = "xlsx", "Расстановка кранов.xlsx"
		}
		path := chooseSave(kind, "Отчёт "+kind, name, false)
		if path == "" {
			return true
		}
		saveAndNotify(path, func(file string) error {
			if id == cranePlaceWord {
				return exportCranePlacementWord(cranePlaceResult, file)
			}
			return exportCranePlacementExcel(cranePlaceResult, file)
		})
		return true
	case cranePlaceSave:
		if len(cranePlaceResult.Cases) == 0 {
			message("Сначала рассчитайте положения кранов", true)
			return true
		}
		path := chooseSave("zip", "Архив схем (ZIP)", "Схемы размещения кранов.zip", false)
		if path == "" {
			return true
		}
		parts := map[string]string{}
		var index strings.Builder
		index.WriteString("РАССТАНОВКА КРАНОВ. НОРМАТИВНЫЕ КОЛЁСНЫЕ НАГРУЗКИ\r\n")
		for i, c := range cranePlaceResult.Cases {
			file := fmt.Sprintf("case_%03d.bmp", i+1)
			tmp := filepath.Join(os.TempDir(), fmt.Sprintf("ebala_case_%03d.bmp", i+1))
			if e := cranePictureForCase(cranePlaceResult.SpansMM, c, tmp); e != nil {
				message(e.Error(), true)
				return true
			}
			b, e := os.ReadFile(tmp)
			if e != nil {
				message(e.Error(), true)
				return true
			}
			parts[file] = string(b)
			fmt.Fprintf(&index, "\r\n%03d. %s: %s %s [файл: %s]\r\n", i+1, c.Name, tempRU(c.Value, 3), c.Units, file)
			for j, w := range c.Points {
				fmt.Fprintf(&index, "  Колесо %d (%s): x=%s мм, P=%s кН\r\n", j+1, w.Crane, tempRU(w.XMM, 1), tempRU(w.PKN, 3))
			}
		}
		for _, w := range cranePlaceResult.Warnings {
			index.WriteString("\r\n" + w)
		}
		parts["Результаты.txt"] = index.String()
		if e := packageZip(path, parts); e != nil {
			message(e.Error(), true)
		} else {
			message("Схемы сохранены", false)
		}
		return true
	case cranePlaceBack, cranePlaceTopBack:
		switchPage("temp_cranes")
		return true
	case cranePlaceRun:
		spans, suspicious, err := parseCraneSpanMetersAuto(get(cranePlaceLengths))
		if err != nil {
			message(err.Error(), true)
			return true
		}
		if suspicious {
			message("Проверьте длину пролёта: введено более 100 м. Возможно, значение указано в миллиметрах, а здесь нужны метры. Расчёт будет выполнен с введёнными значениями.", false)
		}
		trains, e := cranePlaceTrains()
		if e != nil {
			message(e.Error(), true)
			return true
		}
		res, e := CalculateCranePlacements(spans, trains)
		if e != nil {
			message(e.Error(), true)
			return true
		}
		cranePlaceResult = res
		syncCurrentCraneSet()
		if craneSetCurrentID > 0 {
			cranePlaceCurrentBlockID = project.putCranePlacement(cranePlaceCurrentBlockID, craneSetCurrentID, get(cranePlaceLengths), res)
		}
		projectHomeOverview()
		set(cranePlaceTrainInfo, cranePlaceTrainDescription(trains))
		// Explicit visible thumb: long multi-wheel trains can otherwise look truncated.
		cranePlaceShowScroll.Call(controls[cranePlaceTrainInfo], 1, 1)
		rows := [][]string{}
		for i, c := range res.Cases {
			rows = append(rows, []string{fmt.Sprint(i + 1), c.Name, tempRU(c.Value, 3) + " " + c.Units, tempRU(c.Points[0].XMM, 1)})
		}
		tempRows(cranePlaceResults, rows)
		if len(rows) > 0 {
			tempSelectFirst(cranePlaceResults)
			cranePlaceRender(0)
		} else {
			set(cranePlaceDescription, "Для заданных параметров опасные положения не найдены")
		}
		if len(res.Warnings) > 0 {
			message(strings.Join(res.Warnings, "\n"), false)
		}
		return true
	}
	return false
}
func cranePlaceNotify(id int, code int) bool {
	if id != cranePlaceResults {
		return false
	}
	if code == -101 || code == -2 {
		n := tempSelected(cranePlaceResults)
		if n >= 0 {
			cranePlaceRender(n)
		}
		return true
	}
	return false
}
