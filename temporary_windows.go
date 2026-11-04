//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unsafe"
)

const (
	tempHubSP20          = 12001
	tempHubSP35          = 12002
	tempHubCrane         = 12003
	tempProject          = 12004
	tempHome             = 12008
	tempHubBack          = 12009
	tempReportWord       = 12007
	tempReportExcel      = 12010
	tempDel              = 12006
	tempChosen           = 12050
	tempCraneChosen      = 12051
	tempSp20Chosen       = 12052
	tempSp35Chosen       = 12053
	tempCraneLocalChosen = 12054
	tempSP20Category     = 12101
	tempSP20Search       = 12102
	tempSP20List         = 12103
	tempSP20Detail       = 12104
	tempSP20Add          = 12105
	tempSP20Coef         = 12106
	tempSP20Back         = 12107
	tempSP20Count        = 12108
	tempSP20Mode         = 12109
	tempSP20Area         = 12110
	tempSP20Floors       = 12111
	tempSP20Remove       = 12112
	tempSP20CustomName   = 12113
	tempSP20CustomLoad   = 12114
	tempSP20CustomGamma  = 12115
	tempSP20CustomInfo   = 12116
	tempSP20AreaHelp     = 12117
	tempSP20NewRow       = 12118
	tempReportWordLocal  = 12119
	tempReportExcelLocal = 12120
	tempTransportChosen  = 12121
	tempCraneDeleteRow   = 12122
	tempRowEdit          = 12123
	tempSP35Category     = 12200
	tempSP35Search       = 12201
	tempSP35List         = 12202
	tempSP35Detail       = 12203
	tempSP35Add          = 12204
	tempSP35Back         = 12205
	tempSP35Remove       = 12206
	tempSP35Image        = 12210
	tempSP35Gamma        = 12211
	tempSP35Length       = 12212
	tempSP20Preview      = 12213
	tempCraneCount       = 12300
	tempCraneFirst       = 12310
	tempCraneSecond      = 12320
	tempCraneApply       = 12330
	tempCraneBack        = 12331
	tempCranePick1       = 12332
	tempCranePick2       = 12333
	tempCraneRemove      = 12334
	tempCraneAdd         = 12335
	tempCraneDelete      = 12336
	tempCraneDuplicate   = 12337
	tempCraneUnit        = 12338
	tempCraneCatalog     = 12400
	tempCraneCap         = 12401
	tempCraneSpan        = 12402
	tempCraneHeight      = 12403
	tempCraneList        = 12404
	tempCraneDetail      = 12405
	tempCraneUse         = 12406
	tempCraneCancel      = 12407
)

var tempCatalogData TempCatalog
var sp20Guide SP20Guide
var tempCatalogError string
var tempSections = TempReport{Version: 1, Sections: []TempSelection{}}
var sp20Filtered []int
var sp35Filtered []int
var tempLastTransportRecord = -1
var craneFiltered []int
var craneEditTarget = 1
var cranePickAppend bool
var cranePicked = map[int]*TempSelection{}
var secondCraneControls []uintptr
var tempSummaryRows = map[int][]int{}
var sp20SortOrder = 0
var craneEditingCount = 1
var craneActiveRow = -1
var craneChangingRow bool
var tempTransportContext = false
var craneUnitTon = false
var sp35Categories = []string{"Все виды транспорта", "Автомобильные АК и НК", "Железнодорожные СК", "Метрополитен и трамваи", "Промышленный транспорт АБ"}

func tempLoad() {
	var e error
	tempCatalogData, e = loadTempCatalog(filepath.Join(storageRoot(), "Catalog", "temporary_loads.json"))
	if e != nil {
		tempCatalogError = e.Error()
	} else {
		// The crane catalogue is replaced in full by the current GOST/handbook database.
		// Legacy series and overlapping crane records are not selectable.
		base := tempCatalogData.Records[:0]
		for _, record := range tempCatalogData.Records {
			if tempCraneCategory(record) == 0 {
				base = append(base, record)
			}
		}
		tempCatalogData.Records = base
		newCranes, loadErr := loadTempCatalog(filepath.Join(storageRoot(), "Catalog", "cranes_gost_handbook.json"))
		if loadErr != nil {
			tempCatalogError = loadErr.Error()
		} else {
			tempCatalogData.Records = append(tempCatalogData.Records, newCranes.Records...)
		}
	}
	sp20Guide, e = loadSP20Guide(filepath.Join(storageRoot(), "Catalog", "sp20_guide.json"))
	if e != nil {
		tempCatalogError = e.Error()
	}
}
func tempPageEntered() {
	if tempCatalogError != "" {
		message(tempCatalogError, true)
	}
	tempRefreshChosen()
}
func tempComboIndex(id int) int {
	if controls[id] == 0 {
		return -1
	}
	return int(send(id, 0x147, 0, 0))
} // CB_GETCURSEL
func tempList(id int, items []string) {
	send(id, LB_RESETCONTENT, 0, 0)
	for _, s := range items {
		pSend.Call(controls[id], LB_ADDSTRING, 0, uintptr(unsafe.Pointer(ptr(s))))
	}
}
func tempListIndex(id int) int {
	v := send(id, LB_GETCURSEL, 0, 0)
	if int32(v) == -1 {
		return -1
	}
	return int(v)
}
func tempSummary(id int, cranes bool) {
	var data [][]string
	var mapping []int
	for i, s := range tempSections.Sections {
		if tempIsCrane(s) != cranes || (!cranes && s.Group != "СП 20 · временные") || (cranes && id == tempCraneChosen && len(s.Wheels) == 0) {
			continue
		}
		mapping = append(mapping, i)
		if cranes {
			if id == tempCraneLocalChosen {
				data = append(data, []string{fmt.Sprint(len(mapping)), s.Title, tempCraneKind(s), func() string {
					if tempCraneWheelsCount(s) == 0 {
						return ""
					}
					return fmt.Sprint(tempCraneWheelsCount(s))
				}(), tempCraneForcesText(s, tempComboIndex(tempCraneUnit) == 1), tempCraneStepsText(s), "Удалить"})
			} else {
				data = append(data, []string{fmt.Sprint(len(mapping)), s.Title, tempCraneKind(s), tempForcesBoth(s), tempSteps(s), "Удалить"})
			}
		} else {
			n, d := "—", "—"
			if s.Group == "СП 20 · временные" {
				if v, ok := s.Params["is_defined"].(bool); !ok || v {
					n = tempUnitPair(tempNumber(s, "normative_reduced"), tempNumberText(s.Params["unit"]))
					d = tempUnitPair(tempNumber(s, "design_reduced"), tempNumberText(s.Params["unit"]))
				} else {
					n = "По заданию"
					d = "По заданию"
				}
			}
			if s.Group == "СП 35 · транспорт" {
				n, d = tempTransportSummary(s)
			}
			data = append(data, []string{fmt.Sprint(len(mapping)), s.Title, n, func() string {
				if s.Group == "СП 20 · временные" {
					return tempRU(tempNumber(s, "gamma_f"), 2)
				}
				return "—"
			}(), d, "Удалить"})
		}
	}
	tempSummaryRows[id] = mapping
	tempRows(id, data)
}
func tempRefreshChosen() {
	if !project.ReportManaged {
		hasCalculation := false
		for _, sel := range tempSections.Sections {
			if !tempIsCrane(sel) || len(sel.Wheels) > 0 {
				hasCalculation = true
				break
			}
		}
		if hasCalculation {
			project.initReportBlocks(tempSections.Sections)
		}
	}
	if project.ReportManaged {
		if editingReportBlockID > 0 && editingTemporaryIndex >= 0 && editingTemporaryIndex < len(tempSections.Sections) {
			for _, b := range project.SavedBlocks {
				if b.ID == editingReportBlockID && b.Kind == "temporary" {
					snap := cloneTempSelection(tempSections.Sections[editingTemporaryIndex])
					b.Temporary = &snap
					project.updateBlock(b.ID, b)
					break
				}
			}
		}
		project.ensureTempSelections(tempSections.Sections)
		if len(currentCraneSelections(tempSections.Sections)) > 0 {
			syncCurrentCraneSet()
		}
		for i := range project.SavedBlocks {
			b := &project.SavedBlocks[i]
			if b.Kind != "temporary" || b.Temporary == nil {
				continue
			}
			for _, selection := range tempSections.Sections {
				if tempKey(selection) == tempKey(*b.Temporary) {
					snap := cloneTempSelection(selection)
					b.Temporary = &snap
					break
				}
			}
		}
	}
	projectHomeOverview()
	for _, id := range []int{tempChosen, tempSp20Chosen, tempSp35Chosen, tempTransportChosen, tempCraneChosen, tempCraneLocalChosen} {
		tempUpdateSummary(id)
	}
}
func removeSavedTempSelection(s TempSelection) {
	for _, b := range append([]SavedReportBlock(nil), project.SavedBlocks...) {
		if b.Kind == "temporary" && b.Temporary != nil && tempKey(*b.Temporary) == tempKey(s) {
			project.removeBlock(b.ID)
			return
		}
	}
}
func tempDelete(id int) {
	row := tempSelected(id)
	mapping := tempSummaryRows[id]
	if row < 0 || row >= len(mapping) {
		return
	}
	n := mapping[row]
	removeSavedTempSelection(tempSections.Sections[n])
	tempSections.Sections = append(tempSections.Sections[:n], tempSections.Sections[n+1:]...)
	tempRefreshChosen()
}
func sp20Update() {
	if controls[tempSP20List] == 0 {
		return
	}
	cat := ""
	ci := tempComboIndex(tempSP20Category)
	if ci > 0 && ci <= len(sp20Guide.Categories) {
		cat = sp20Guide.Categories[ci-1]
	}
	query := strings.ToLower(strings.TrimSpace(get(tempSP20Search)))
	sp20Filtered = nil
	for i, r := range sp20Guide.Records {
		if cat != "" && r.Category != cat {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(r.Title), query) {
			continue
		}
		sp20Filtered = append(sp20Filtered, i)
	}
	rows := [][]string{}
	for _, idx := range sp20Filtered {
		r := sp20Guide.Records[idx]
		rows = append(rows, []string{r.Title, func() string {
			if r.Editable && r.Norm == 0 {
				return "По заданию"
			}
			return tempRU(r.Norm*func() float64 {
				if r.Unit == "кПа" {
					return kpaToKgfM2(1)
				}
				return kgfPerKN
			}(), 1) + func() string {
				if r.Unit == "кПа" {
					return " кгс/м²"
				}
				if r.Unit == "кН/м" {
					return " кгс/м"
				}
				return " кгс"
			}()
		}(), func() string {
			if r.Editable && r.Norm == 0 {
				return ""
			}
			return tempRU(r.Norm, 2) + " " + r.Unit
		}()})
	}
	tempRows(tempSP20List, rows)
	if len(rows) > 0 {
		tempSelectFirst(tempSP20List)
	}
	set(tempSP20Count, fmt.Sprintf("Найдено: %d", len(rows)))
	if len(rows) > 0 {
		sp20Show(0)
	} else {
		set(tempSP20Detail, "")
	}
}
func sp20Show(n int) {
	if n < 0 || n >= len(sp20Filtered) {
		return
	}
	r := sp20Guide.Records[sp20Filtered[n]]
	msg := r.Title + "\r\n\r\nОфициальная формулировка:\r\n" + r.Official + "\r\n\r\n" + r.Source
	if !r.Editable || r.Norm > 0 {
		msg += "\r\n\r\nНормативная: " + tempUnitPair(r.Norm, r.Unit)
	}
	if r.Notes != "" {
		msg += "\r\n\r\n" + r.Notes
	}
	if r.Minimum {
		msg += "\r\nНе менее указанного значения."
	}
	set(tempSP20Detail, msg)
	sp20UpdateControls(r)
	sp20UpdatePreview()
}
func sp20UpdateControls(r SP20Row) {
	applicable := !r.Editable && tempReductionClass(r) != 0 && r.Type == "Равномерная"
	for _, id := range []int{tempSP20Mode, tempSP20Area, tempSP20Floors} {
		if controls[id] != 0 {
			pEnable.Call(controls[id], boolFlag(applicable))
		}
	}
	if !applicable {
		send(tempSP20Mode, 0x14E, 0, 0)
	}
	if applicable {
		sp20ModeControls()
	}
	/* Custom loads are edited directly in the summary table. */

}
func boolFlag(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}
func sp20ModeControls() {
	n := tempComboIndex(tempSP20Mode)
	if controls[tempSP20Area] != 0 {
		pEnable.Call(controls[tempSP20Area], boolFlag(n > 0))
	}
	if controls[tempSP20Floors] != 0 {
		pEnable.Call(controls[tempSP20Floors], boolFlag(n == 2))
	}
	sp20UpdatePreview()
}
func sp35Kind(title string) int {
	t := strings.ToLower(title)
	switch {
	case strings.Contains(t, "метрополит"):
		return 3
	case strings.Contains(t, "трамва"):
		return 3
	case strings.Contains(t, "железнодорож") || strings.Contains(t, "с14") || strings.Contains(t, "с11"):
		return 2
	case strings.Contains(t, "промышленн") || strings.HasPrefix(t, "аб-"):
		return 4
	default:
		return 1
	}
}
func sp35Update() {
	if controls[tempSP35List] == 0 {
		return
	}
	ci := tempComboIndex(tempSP35Category)
	query := strings.ToLower(strings.TrimSpace(get(tempSP35Search)))
	sp35Filtered = nil
	rows := [][]string{}
	used := map[string]bool{}
	for i, e := range tempCatalogData.Records {
		if e.Group != "СП 35 · транспорт" || strings.HasPrefix(e.Title, "АБ-") {
			continue
		}
		if ci > 0 && sp35Kind(e.Title) != ci {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(e.Title), query) {
			continue
		}
		key := strings.ToLower(e.Title)
		if used[key] {
			continue
		}
		used[key] = true
		sp35Filtered = append(sp35Filtered, i)
	}
	sort.SliceStable(sp35Filtered, func(i, j int) bool {
		return tempCatalogData.Records[sp35Filtered[i]].Title < tempCatalogData.Records[sp35Filtered[j]].Title
	})
	for _, idx := range sp35Filtered {
		r := tempCatalogData.Records[idx]
		_, norm, _, _ := tempTransportFields(tempSectionFromEntry(r))
		rows = append(rows, []string{r.Title, norm})
	}
	tempRows(tempSP35List, rows)
	if len(rows) > 0 {
		tempSelectFirst(tempSP35List)
	}
	if len(rows) > 0 {
		sp35Show(0)
	} else {
		set(tempSP35Detail, "")
	}
}
func tempSP35Short(e TempEntry) string {
	p := e.Params
	if v, ok := p["axle_load_kN"]; ok {
		return fmt.Sprintf("На ось: %.2f кН", v)
	}
	if v, ok := p["distributed_kN_per_m"]; ok {
		return fmt.Sprintf("%.2f кН/м", v)
	}
	if v, ok := p["rear_axle_kN"]; ok {
		return fmt.Sprintf("Задняя ось %.2f кН", v)
	}
	return "Нагрузочная схема"
}
func sp35Show(n int) {
	if n < 0 || n >= len(sp35Filtered) {
		return
	}
	if sp35Filtered[n] != tempLastTransportRecord {
		tempLastTransportRecord = sp35Filtered[n]
		// Справочник не требует дополнительных вводимых параметров.
	}
	e := tempCatalogData.Records[sp35Filtered[n]]
	details, norm, gamma, calc := tempTransportFields(tempSectionFromEntry(e))
	_ = details
	_ = norm
	_ = gamma
	_ = calc
	s := tempSectionFromEntry(e)
	set(tempSP35Detail, transportDetailedText(s, false))
	tempShowTransportImage(transportKind(s))
}
func tempScope() (TempAreaMode, float64, int, error) {
	mode := TempAreaMode(tempComboIndex(tempSP20Mode))
	if mode < 0 || mode > 2 {
		mode = TempFull
	}
	if mode == TempFull {
		return mode, 0, 1, nil
	}
	area, e := parseNumber(strings.TrimSpace(get(tempSP20Area)))
	if e != nil || area <= 0 {
		return mode, 0, 0, fmt.Errorf("Укажите грузовую площадь, м²")
	}
	floors := 1
	if mode == TempSeveralFloors {
		floors, e = strconv.Atoi(strings.TrimSpace(get(tempSP20Floors)))
		if e != nil || floors < 2 {
			return mode, 0, 0, fmt.Errorf("Укажите число перекрытий не менее 2")
		}
	}
	return mode, area, floors, nil
}
func tempManualFromUI(i int) (TempSelection, error) {
	base := tempCraneFirst
	if i == 2 {
		base = tempCraneSecond
	}
	num, e := strconv.Atoi(strings.TrimSpace(get(base + 1)))
	if e != nil {
		return TempSelection{}, fmt.Errorf("укажите число колёс")
	}
	forces := get(base + 2)
	if tempComboIndex(tempCraneUnit) == 1 {
		f, e := parseWheelVector(forces, num, "Нагрузки на колёса, тс", true)
		if e != nil {
			return TempSelection{}, e
		}
		values := make([]string, len(f))
		for j, v := range f {
			values[j] = strconv.FormatFloat(v*9.80665, 'f', -1, 64)
		}
		forces = strings.Join(values, "; ")
	}
	manual, e := tempManualSteps(get(base), num, forces, get(base+3))
	if e != nil {
		return manual, e
	}
	if picked, ok := cranePicked[i]; ok && picked != nil && manual.Title == picked.Title && tempSameCraneWheels(manual, *picked) {
		return *picked, nil
	}
	return manual, nil
}
func tempSameCraneWheels(a, b TempSelection) bool {
	if len(a.Wheels) != len(b.Wheels) {
		return false
	}
	for j := range a.Wheels {
		if abs(a.Wheels[j].PKN-b.Wheels[j].PKN) > 0.05 {
			return false
		}
		if a.Wheels[j].XKnown != b.Wheels[j].XKnown {
			return false
		}
		if a.Wheels[j].XKnown && abs(a.Wheels[j].XMM-b.Wheels[j].XMM) > 0.001 {
			return false
		}
	}
	return true
}
func tempToggleSecond() {
	visible := craneEditingCount == 2
	for _, h := range secondCraneControls {
		if visible && activePage == "temp_cranes" {
			pShow.Call(h, 1)
		} else {
			pShow.Call(h, 0)
		}
	}
}
func tempCraneCategory(e TempEntry) int {
	switch e.Group {
	case "Краны · ГОСТ":
		return 1
	case "Краны · серии":
		return 2
	case "Краны · справочник":
		return 3
	case "Краны · многоколёсные":
		return 4
	}
	return 0
}
func tempParamFloat(e TempEntry, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := e.Params[k]; ok {
			if x, ok := v.(float64); ok {
				return x
			}
		}
	}
	return 0
}
func tempCraneClass(e TempEntry) string {
	v := tempNumberText(e.Params["crane_mode"])
	if v == "<nil>" {
		v = tempNumberText(e.Params["duty_group"])
	}
	if v == "<nil>" || v == "" {
		return "Класс работы: не указан"
	}
	switch v {
	case "3К":
		return "Класс работы: лёгкий (3К)"
	case "5К":
		return "Класс работы: средний (5К)"
	case "6К":
		return "Класс работы: тяжёлый (6К)"
	case "7К":
		return "Класс работы: весьма тяжёлый (7К)"
	}
	return "Класс работы: " + v
}
func tempFilterCrane() {
	if controls[tempCraneList] == 0 {
		return
	}
	category := tempComboIndex(tempCraneCatalog)
	cap := strings.TrimSpace(get(tempCraneCap))
	span := strings.TrimSpace(get(tempCraneSpan))
	height := strings.TrimSpace(get(tempCraneHeight))
	craneFiltered = nil
	rows := [][]string{}
	for i, e := range tempCatalogData.Records {
		cat := tempCraneCategory(e)
		if cat == 0 {
			continue
		}
		if category > 0 {
			source := tempNumberText(e.Params["source_standard"])
			allowed := []string{"", "ГОСТ 22045-89", "ГОСТ 25711-83", "ГОСТ 7890-93", "Кузнецов, Металлические конструкции, т.2 (1998)"}
			if category >= len(allowed) || source != allowed[category] {
				continue
			}
		}
		capacity := tempParamFloat(e, "capacity_t", "crane_capacity_t")
		craneSpan := tempParamFloat(e, "crane_span_m", "span_m")
		if cap != "" {
			v, err := parseNumber(cap)
			if err == nil && (capacity == 0 || abs(capacity-v) > 0.01) {
				continue
			}
		}
		if span != "" {
			v, err := parseNumber(span)
			if err == nil && (craneSpan == 0 || abs(craneSpan-v) > 0.01) {
				continue
			}
		}
		if height != "" {
			v, err := parseNumber(height)
			if err == nil {
				h := tempParamFloat(e, "lifting_height_m", "lift_height_m")
				if h == 0 || abs(h-v) > 0.01 {
					continue
				}
			}
		}
		craneFiltered = append(craneFiltered, i)
		rows = append(rows, []string{tempRU(capacity, 1), func() string {
			if craneSpan > 0 {
				return tempRU(craneSpan, 1)
			}
			return "—"
		}(), func() string {
			h := tempParamFloat(e, "lifting_height_m", "lift_height_m")
			if h > 0 {
				return tempRU(h, 1)
			}
			return "—"
		}(), tempCraneClass(e)})
	}
	sort.SliceStable(craneFiltered, func(i, j int) bool {
		a, b := tempCatalogData.Records[craneFiltered[i]], tempCatalogData.Records[craneFiltered[j]]
		for _, keys := range [][]string{{"capacity_t", "crane_capacity_t"}, {"crane_span_m", "span_m"}, {"lifting_height_m", "lift_height_m"}} {
			aa, bb := tempParamFloat(a, keys...), tempParamFloat(b, keys...)
			if aa != bb {
				return aa < bb
			}
		}
		return tempCraneClass(a) < tempCraneClass(b)
	})
	rows = rows[:0]
	for _, idx := range craneFiltered {
		e := tempCatalogData.Records[idx]
		cap, span, h := tempParamFloat(e, "capacity_t", "crane_capacity_t"), tempParamFloat(e, "crane_span_m", "span_m"), tempParamFloat(e, "lifting_height_m", "lift_height_m")
		r := []string{tempRU(cap, 1), "—", "—", tempCraneClass(e)}
		if span > 0 {
			r[1] = tempRU(span, 1)
		}
		if h > 0 {
			r[2] = tempRU(h, 1)
		}
		rows = append(rows, r)
	}
	tempRows(tempCraneList, rows)
	if len(rows) > 0 {
		tempSelectFirst(tempCraneList)
	}
	if len(rows) > 0 {
		tempCraneShow(0)
	} else {
		set(tempCraneDetail, "")
	}
}
func tempCraneShow(n int) {
	if n < 0 || n >= len(craneFiltered) {
		return
	}
	e := tempCatalogData.Records[craneFiltered[n]]
	s := tempSectionFromEntry(e)
	msg := e.Title + "\r\nТип: " + tempCraneKind(s) + "\r\n" + tempCraneClass(e) + "\r\n" + tempReadableSource(e.Source) + "\r\n"
	if base := tempParamFloat(e, "wheelbase_mm"); base > 0 {
		msg += "База ходовых колёс: " + tempRU(base, 0) + " мм\r\n"
	}
	if envelope := tempParamFloat(e, "overall_length_mm"); envelope > 0 {
		msg += "Продольный габарит крана: " + tempRU(envelope, 0) + " мм\r\n"
	}
	if configuration, ok := e.Params["configuration"].(string); ok && configuration != "" {
		msg += "Исполнение: " + configuration + "\r\n"
	}
	if len(s.Wheels) > 0 {
		msg += fmt.Sprintf("\r\nКолёс на одной нитке: %d\r\nНагрузки, кН: %s", len(s.Wheels), tempForces(s))
		if steps := tempSteps(s); steps != "" {
			msg += "\r\nРасстояния, мм: " + steps
		}
	} else {
		msg += "\r\nКолёсная схема в источнике не задана."
	}
	set(tempCraneDetail, msg)
}

type tempListActivate struct {
	Header  tempNotifyHeader
	Item    int32
	SubItem int32
}

func tempTableNotify(lp uintptr) bool {
	if lp != 0 {
		hdr := (*tempNotifyHeader)(unsafe.Pointer(lp))
		if int(hdr.id) == reportCompositionList {
			z := (*tempListActivate)(unsafe.Pointer(lp))
			if hdr.code == -3 {
				return compositionClick(int(z.Item), int(z.SubItem), true)
			}
			if hdr.code == -2 {
				return compositionClick(int(z.Item), int(z.SubItem), false)
			}
			return false
		}
	}
	if lp == 0 {
		return false
	}
	h := (*tempNotifyHeader)(unsafe.Pointer(lp))
	id := int(h.id)
	n := tempSelected(id)
	if cranePlaceNotify(id, int(h.code)) {
		return true
	}
	if h.code == -3 { // NM_DBLCLK
		z := (*tempListActivate)(unsafe.Pointer(lp))
		row := int(z.Item)
		if row < 0 {
			row = n
		}
		if tempSummaryControlID(id) && z.SubItem > 0 && z.SubItem < int32(tempColumnCount[id]-1) {
			tempStartRowEditing(id, row, int(z.SubItem))
			return true
		}
		return tempSummaryDblClick(id, row)
	}
	if h.code == -2 && tempSummaryControlID(id) { // NM_CLICK
		z := (*tempListActivate)(unsafe.Pointer(lp))
		row := int(z.Item)
		if row >= 0 && int(z.SubItem) == tempColumnCount[id]-1 {
			tempDeleteAt(id, row)
			return true
		}
	}
	if h.code != -101 && h.code != -2 {
		return false
	}
	switch id {
	case tempSP20List:
		sp20Show(n)
		return true
	case tempSP35List:
		sp35Show(n)
		return true
	case tempCraneList:
		tempCraneShow(n)
		return true
	case tempCraneLocalChosen:
		if n >= 0 && !craneChangingRow {
			tempCraneSelectRow(n)
		}
		return true
	}
	return false
}
func tempNotify(id, code int) bool {
	if id == tempCraneUnit && code == 1 {
		tempRefreshChosen()
		return true
	}
	if tempEditNotify(id, code) {
		return true
	}
	switch id {
	case tempSP20Category:
		if code == CBN_SELCHANGE {
			sp20Update()
			return true
		}
	case tempSP20Mode:
		if code == CBN_SELCHANGE {
			sp20ModeControls()
			return true
		}
	case tempSP20Area, tempSP20Floors:
		if code == EN_CHANGE {
			sp20UpdatePreview()
			return true
		}
	case tempSP20Search:
		if code == EN_CHANGE {
			sp20Update()
			return true
		}
	case tempSP35Category:
		if code == CBN_SELCHANGE {
			sp35Update()
			return true
		}
	case tempSP35Search:
		if code == EN_CHANGE {
			sp35Update()
			return true
		}
	case tempCraneUnit:
		if code == CBN_SELCHANGE {
			tons := tempComboIndex(tempCraneUnit) == 1
			if tons != craneUnitTon {
				for _, base := range []int{tempCraneFirst, tempCraneSecond} {
					raw := strings.TrimSpace(get(base + 2))
					if raw == "" {
						continue
					}
					num, err := strconv.Atoi(strings.TrimSpace(get(base + 1)))
					if err != nil || num < 1 {
						continue
					}
					v, err := parseWheelVector(raw, num, "Нагрузки", true)
					if err != nil {
						continue
					}
					a := make([]string, len(v))
					for j, x := range v {
						if tons {
							x /= 9.80665
						} else {
							x *= 9.80665
						}
						a[j] = tempRU(x, 4)
					}
					set(base+2, strings.Join(a, "; "))
				}
				craneUnitTon = tons
			}
			return true
		}
	case tempCraneCatalog:
		if code == CBN_SELCHANGE {
			tempFilterCrane()
			return true
		}
	case tempCraneCap, tempCraneSpan, tempCraneHeight:
		if code == EN_CHANGE {
			tempFilterCrane()
			return true
		}

	}
	return false
}
func tempAction(id int) bool {
	if craneUi043Action(id) {
		return true
	}
	if cranePlaceAction(id) {
		return true
	}
	if tempRunContext(id) {
		return true
	}
	switch id {
	case tempHubSP20:
		switchPage("temp_sp20")
		sp20Update()
		tempRefreshChosen()
		return true
	case tempHubSP35:
		switchPage("temp_sp35")
		sp35Update()
		tempRefreshChosen()
		return true
	case tempHubCrane:
		if tempCraneDraftCount() == 0 {
			tempSections.Sections = append(tempSections.Sections, tempEmptyCrane())
		}
		switchPage("temp_cranes")
		tempRefreshChosen()
		tempCraneSelectRow(0)
		return true
	case 399:
		if activePage == "temporary" {
			switchPage("home")
		} else if activePage == "temp_crane_place" {
			switchPage("temp_cranes")
		} else if activePage == "temp_crane_pick" {
			switchPage("temp_cranes")
			tempToggleSecond()
		} else if strings.HasPrefix(activePage, "temp_") {
			switchPage("home")
		} else {
			return false
		}
		return true
	case tempHome:
		switchPage("home")
		return true
	case tempHubBack, tempSP20Back, tempSP35Back, tempCraneBack:
		switchPage("home")
		return true
	case tempSP20Coef:
		message("СП 20.13330.2016\n\nРавномерные нагрузки менее 2,0 кПа: γf = 1,3\nОт 2,0 кПа: γf = 1,2\nСосредоточенные нагрузки: 1,2\nНагрузки на поручни: 1,2\nТранспорт на перекрытиях: 1,2\n\nОборудование стационарное: 1,05\nЕго изоляция: 1,2\nЖидкости: 1,0\nСуспензии, шламы, сыпучие: 1,1\nСкладируемые материалы: 1,2\n\nТаблица 8.2; пункты 8.2.7, 8.3.5, 8.4.5.", false)
		return true
	case tempSP20NewRow:
		tempAddManualRow()
		return true
	case tempSP20Add:
		n := tempSelected(tempSP20List)
		if n < 0 || n >= len(sp20Filtered) {
			message("Выберите строку нагрузки", true)
			return true
		}
		r := sp20Guide.Records[sp20Filtered[n]]
		if r.Editable {
			result, e := sp20CustomSection(r, r.Title, "", tempRU(r.Gamma, 2))
			if e != nil {
				message(e.Error(), true)
				return true
			}
			result.Params["custom_line"] = true
			tempSections.Sections = append(tempSections.Sections, result)
			project.unsuppressTemporary(tempKey(result))
			tempRefreshChosen()
			return true
		}
		mode, area, floors, e := tempScope()
		if e != nil {
			message(e.Error(), true)
			return true
		}
		s, e := tempSP20Section(r, mode, area, floors)
		if e != nil {
			message(e.Error(), true)
			return true
		}
		tempSections.Sections = append(tempSections.Sections, s)
		project.unsuppressTemporary(tempKey(s))
		tempRefreshChosen()
		return true
	case tempSP35Add:
		n := tempSelected(tempSP35List)
		if n < 0 || n >= len(sp35Filtered) {
			message("Выберите нагрузку", true)
			return true
		}
		section := tempSectionFromEntry(tempCatalogData.Records[sp35Filtered[n]])
		tempSections.Sections = append(tempSections.Sections, section)
		tempRefreshChosen()
		return true
	case tempDel:
		tempDelete(tempChosen)
		return true
	case tempSP20Remove:
		tempDelete(tempSp20Chosen)
		return true
	case tempSP35Remove:
		tempDelete(tempSp35Chosen)
		return true
	case tempCraneRemove:
		if activePage == "temporary" {
			tempDelete(tempCraneChosen)
		} else {
			tempDelete(tempCraneLocalChosen)
		}
		return true
	case tempCraneApply:
		if craneActiveRow < 0 {
			message("Сначала выберите кран в таблице", true)
			return true
		}
		draft, e := tempManualFromUI(1)
		if e != nil {
			message(e.Error(), true)
			return true
		}
		row := tempCranePhysical(craneActiveRow)
		if row < 0 {
			return true
		}
		tempSections.Sections[row] = draft
		tempRefreshChosen()
		tempSetListRow(tempCraneLocalChosen, craneActiveRow)
		return true
	case tempCraneAdd:
		tempSections.Sections = append(tempSections.Sections, tempEmptyCrane())
		tempRefreshChosen()
		tempCraneSelectRow(tempCraneDraftCount() - 1)
		return true
	case tempCraneDuplicate:
		row := tempCranePhysical(craneActiveRow)
		if row < 0 {
			return true
		}
		current := tempSections.Sections[row]
		current.Params = copyTempParams(current.Params)
		current.Wheels = append([]TempWheel(nil), current.Wheels...)
		current.Title += " (копия)"
		tempSections.Sections = append(tempSections.Sections, current)
		tempRefreshChosen()
		tempCraneSelectRow(tempCraneDraftCount() - 1)
		return true
	case tempCraneDelete:
		row := tempCranePhysical(craneActiveRow)
		if row < 0 {
			return true
		}
		tempSections.Sections = append(tempSections.Sections[:row], tempSections.Sections[row+1:]...)
		if tempCraneDraftCount() == 0 {
			tempSections.Sections = append(tempSections.Sections, tempEmptyCrane())
		}
		tempRefreshChosen()
		index := craneActiveRow
		if index >= tempCraneDraftCount() {
			index = tempCraneDraftCount() - 1
		}
		tempCraneSelectRow(index)
		return true
	case tempCranePick1, tempCranePick2:
		craneEditTarget = 1
		row := tempCranePhysical(craneActiveRow)
		cranePickAppend = row >= 0 && len(tempSections.Sections[row].Wheels) > 0
		switchPage("temp_crane_pick")
		tempFilterCrane()
		return true
	case tempCraneCancel:
		switchPage("temp_cranes")
		return true
	case tempCraneUse:
		n := tempSelected(tempCraneList)
		if n < 0 || n >= len(craneFiltered) {
			message("Выберите кран", true)
			return true
		}
		e := tempCatalogData.Records[craneFiltered[n]]
		s := tempSectionFromEntry(e)
		if len(s.Wheels) == 0 {
			count := int(tempParamFloat(e, "wheel_count_one_rail", "wheel_count"))
			p := tempParamFloat(e, "p_wheel_kN")
			if count > 0 && p > 0 {
				forces := make([]string, count)
				for j := range forces {
					forces[j] = fmt.Sprint(p)
				}
				generated, err := tempManualSteps(e.Title, count, strings.Join(forces, "; "), "")
				if err == nil {
					s = generated
					s.Source = e.Source + " (максимум принят для всех колёс)"
				}
			}
		}
		if craneActiveRow < 0 || cranePickAppend {
			tempSections.Sections = append(tempSections.Sections, s)
			craneActiveRow = tempCraneDraftCount() - 1
		} else {
			row := tempCranePhysical(craneActiveRow)
			if row >= 0 {
				tempSections.Sections[row] = s
			} else {
				tempSections.Sections = append(tempSections.Sections, s)
			}
		}
		cranePickAppend = false
		tempRefreshChosen()
		switchPage("temp_cranes")
		tempCraneSelectRow(craneActiveRow)
		return true

	case tempReportWord, tempReportExcel, tempReportWordLocal, tempReportExcelLocal:
		local := id == tempReportWordLocal || id == tempReportExcelLocal
		report := tempSections
		if !local {
			compositionEntered()
			selection := project.selectedReport()
			report.SiteLines = projectSiteLines(project.Site, project)
			report.Snows = selection.Snows
			report.Ices = selection.Ices
			report.WeightRows = selection.WeightRows
			report.WeightBlocks = selection.WeightBlocks
			report.SnowTitles = selection.SnowTitles
			report.IceTitles = selection.IceTitles
			report.CraneGroups = selection.CraneGroups
			report.Sections = selection.Sections
		} else {
			report.SiteLines = nil
			report.Snow = nil
			report.Ice = nil
			report.WeightRows = nil
			report.WeightBlocks = nil
		}
		report.Project = strings.TrimSpace(get(tempProject))
		if local {
			report.Sections = nil
		}
		fileBase := "Общая ведомость нагрузок"
		if local {
			switch activePage {
			case "temp_sp20":
				fileBase = "Временные нагрузки"
			case "temp_sp35":
				fileBase = "Транспортные нагрузки"
			case "temp_cranes":
				fileBase = "Мостовые краны"
			}
		}
		availableSections := tempSections.Sections
		if !local {
			availableSections = nil
		} // use the saved report composition
		for _, section := range availableSections {
			if !tempReportInclude(section) || (tempIsCrane(section) && len(section.Wheels) == 0) {
				continue
			}
			if local {
				switch activePage {
				case "temp_sp20":
					if section.Group != "СП 20 · временные" {
						continue
					}
				case "temp_sp35":
					if !tempTransport(section) {
						continue
					}
				case "temp_cranes":
					if !tempIsCrane(section) {
						continue
					}
				}
			}
			report.Sections = append(report.Sections, section)
		}
		if len(report.Sections)+len(report.WeightRows)+len(report.SiteLines)+len(report.Snows)+len(report.Ices) == 0 && report.Snow == nil && report.Ice == nil {
			message("Ведомость пуста. Добавьте хотя бы одну нагрузку.", true)
			return true
		}
		ext := "docx"
		name := fileBase + ".docx"
		pattern := "Word"
		if id == tempReportExcel || id == tempReportExcelLocal {
			ext = "xlsx"
			name = fileBase + ".xlsx"
			pattern = "Excel"
		}
		path := chooseSave(ext, pattern, name, false)
		if path == "" {
			return true
		}
		saveAndNotify(path, func(file string) error {
			if id == tempReportWord || id == tempReportWordLocal {
				return exportTempWord(report, file)
			}
			return exportTempExcel(report, file)
		})
		return true
	}
	return false
}

func tempBottomTable(id int, cranes bool, y int) {
	if cranes {
		tempTable(id, 25, y, 1085, 123, []string{"№", "Кран", "Тип крана", "Нагрузки на колёса, кН (тс)", "Расстояния, мм", "Удалить"}, []int{42, 235, 210, 310, 210, 77})
	} else {
		tempTable(id, 25, y, 1085, 123, []string{"№", "Нагрузка", "Нормативная", "γf", "Расчётная", "Удалить"}, []int{42, 353, 242, 78, 290, 77})
	}
}
func tempBuildPages() {
	buildingPage = "temporary"
	navigation("Состав общего отчёта", "Двойной щелчок — переименовать; «Править» — открыть расчёт")
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 25, 106, 1085, 27, tempGeneralPlace)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 25, 139, 1085, 27, tempGeneralClimate)
	heading("РАСЧЁТНЫЕ СХЕМЫ, ВКЛЮЧЁННЫЕ В ОБЩИЙ ОТЧЁТ", 25, 177, 1000, 30)
	tempTable(reportCompositionList, 25, 216, 1085, 550, []string{"№", "Раздел", "Расчётная схема", "Править", "Удалить"}, []int{45, 245, 615, 100, 80})
	button(tempHome, "На главную", 25, 792, 200, 40)
	button(tempReportWord, "Общий отчёт Word", 635, 792, 215, 40)
	button(tempReportExcel, "Общий расчёт Excel", 865, 792, 245, 40)
	buildingPage = "temp_sp20"
	navigation("СП 20.13330.2016 · Временные нагрузки", "Выберите помещение слева. Нормативное значение видно в таблице.")
	label("Категория", 25, 102, 170, 22)
	combo(tempSP20Category, append([]string{"Все категории"}, sp20Guide.Categories...), 25, 127, 530)
	send(tempSP20Category, 0x14E, 0, 0)
	label("Поиск", 570, 102, 240, 22)
	edit(tempSP20Search, "", 570, 127, 540, 29)
	label("ПОМЕЩЕНИЕ И НОРМАТИВНАЯ НАГРУЗКА", 25, 169, 637, 23)
	label("ОБОСНОВАНИЕ И ПОЯСНЕНИЯ", 680, 169, 430, 23)
	tempTable(tempSP20List, 25, 195, 640, 270, []string{"Помещение / конструкция", "Нагрузка (тех. ед.)", "Нагрузка (СИ)"}, []int{345, 150, 142})
	create("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_READONLY, 680, 195, 430, 270, tempSP20Detail)
	label("Учет площади:", 25, 480, 133, 24)
	combo(tempSP20Mode, []string{"Полная нагрузка", "Одно перекрытие, п. 6.7", "Несколько перекрытий, п. 6.8"}, 153, 478, 300)
	send(tempSP20Mode, 0x14E, 0, 0)
	label("Площадь A, м²", 471, 480, 120, 24)
	edit(tempSP20Area, "", 588, 478, 118, 28)
	label("Число перекрытий", 722, 480, 155, 24)
	edit(tempSP20Floors, "", 881, 478, 80, 28)
	button(tempSP20Coef, "γf · Подсказка", 973, 478, 137, 31)

	label("В ведомость:", 470, 512, 112, 23)
	label("", 580, 512, 520, 23)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 582, 512, 514, 23, tempSP20Preview)
	button(tempSP20Add, "Добавить выбранную нагрузку", 804, 541, 306, 36)
	button(tempSP20NewRow, "+ Добавить свою строку", 25, 541, 288, 36)
	heading("ВЫБРАННЫЕ ВРЕМЕННЫЕ НАГРУЗКИ", 25, 590, 650, 30)
	tempBottomTable(tempSp20Chosen, false, 630)
	button(tempSaveNew, "Сохранить в ведомость и новый расчёт", 25, 758, 373, 30)
	button(tempResetCurrent, "Сбросить нагрузки", 410, 758, 185, 30)
	tempLocalWordButtons["temp_sp20"] = button(tempReportWordLocal, "Отчёт Word", 635, 792, 217, 38)
	tempLocalExcelButtons["temp_sp20"] = button(tempReportExcelLocal, "Отчёт Excel", 865, 792, 245, 38)
	buildingPage = "temp_sp35"
	navigation("СП 35 · Транспортные нагрузки", "Выберите вид транспорта и нагрузочную схему")
	label("Вид транспорта", 25, 102, 240, 22)
	combo(tempSP35Category, sp35Categories, 25, 128, 530)
	send(tempSP35Category, 0x14E, 0, 0)
	label("Поиск", 570, 102, 150, 22)
	edit(tempSP35Search, "", 570, 128, 540, 28)
	label("ТРАНСПОРТНАЯ СХЕМА", 25, 169, 637, 23)
	label("ХАРАКТЕРИСТИКИ И СХЕМА НАГРУЗКИ", 680, 169, 430, 23)
	tempTable(tempSP35List, 25, 195, 640, 343, []string{"Транспортная схема", "Нормативная нагрузка"}, []int{327, 310})
	create("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_READONLY, 680, 195, 430, 165, tempSP35Detail)
	create("STATIC", "", WS_CHILD|WS_VISIBLE|0xE, 680, 368, 420, 168, tempSP35Image)
	button(tempSP35Add, "Добавить выбранную нагрузку", 810, 545, 300, 34)
	heading("ВЫБРАННЫЕ ТРАНСПОРТНЫЕ НАГРУЗКИ", 25, 588, 750, 30)
	tempTransportTable(tempSp35Chosen, 620)
	button(tempSaveNew, "Сохранить в ведомость и новый расчёт", 25, 758, 373, 30)
	button(tempResetCurrent, "Сбросить нагрузки", 410, 758, 185, 30)
	tempLocalWordButtons["temp_sp35"] = button(tempReportWordLocal, "Отчёт Word", 635, 790, 215, 42)
	tempLocalExcelButtons["temp_sp35"] = button(tempReportExcelLocal, "Отчёт Excel", 865, 790, 245, 42)
	buildingPage = "temp_cranes"
	navigation("Мостовые краны", "Двойной щелчок по ячейке для редактирования. Нагрузки и расстояния через пробел или ;")
	heading("КРАНЫ", 25, 97, 1085, 29)
	tempTable(tempCraneLocalChosen, 25, 133, 1085, 350, []string{"№", "Кран", "Тип крана", "Колёс", "Нагрузки на колёса, кН (тс)", "Межколёсные расстояния, мм", "Удалить"}, []int{42, 165, 195, 80, 285, 240, 77})
	button(tempCraneAdd, "+ Добавить кран", 25, 508, 215, 39)
	button(tempCraneDuplicate, "Дублировать", 258, 508, 194, 39)
	button(tempCraneDelete, "Удалить кран", 470, 508, 194, 39)
	button(tempCranePick1, "Подобрать из справочника", 775, 508, 335, 39)
	button(cranePlaceOpen, "Разместить краны", 775, 564, 335, 40)
	label("Единицы ввода колёсных нагрузок", 25, 565, 330, 25)
	combo(tempCraneUnit, []string{"кН", "тс"}, 355, 558, 180)
	send(tempCraneUnit, 0x14E, 0, 0)
	label("Число колёс задаётся для одной нитки пути; разделитель значений — пробел либо точка с запятой.", 25, 632, 1080, 27)
	button(craneSetSaveNew, "Сохранить в ведомость и новый расчёт", 25, 680, 405, 39)
	button(craneSetResetCurrent, "Сбросить нагрузки", 440, 680, 200, 39)
	tempLocalWordButtons["temp_cranes"] = button(tempReportWordLocal, "Отчёт Word", 690, 744, 195, 38)
	tempLocalExcelButtons["temp_cranes"] = button(tempReportExcelLocal, "Отчёт Excel", 901, 744, 209, 38)
	buildingPage = "temp_crane_pick"
	navigation("Подбор мостового крана", "Выберите источник и характеристики типового крана")
	label("Источник", 25, 105, 150, 21)
	combo(tempCraneCatalog, []string{"Все источники", "ГОСТ 22045-89 · опорные 1–5 т", "ГОСТ 25711-83 · опорные 5–50 т", "ГОСТ 7890-93 · подвесные", "Справочник проектировщика"}, 25, 130, 355)
	send(tempCraneCatalog, 0x14E, 0, 0)
	label("Грузоподъёмность, т", 399, 105, 215, 21)
	edit(tempCraneCap, "", 399, 130, 205, 29)
	label("Пролёт крана, м", 615, 105, 200, 21)
	edit(tempCraneSpan, "", 615, 130, 200, 29)
	label("Высота подъёма, м", 832, 105, 278, 21)
	edit(tempCraneHeight, "", 832, 130, 278, 29)
	tempTable(tempCraneList, 25, 179, 740, 480, []string{"Г/п, т", "Пролёт, м", "Высота, м", "Класс работы"}, []int{115, 125, 120, 370})
	create("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_READONLY, 782, 179, 328, 480, tempCraneDetail)
	button(tempCraneCancel, "Назад к кранам", 25, 712, 247, 46)
	button(cranePlaceOpen+20, "Разместить краны", 549, 712, 260, 46)
	button(tempCraneUse, "Выбрать кран", 837, 712, 273, 46)
	cranePlaceBuild()
	buildingPage = "home"
}
func tempCraneBlock(i, x, y int) {
	base := tempCraneFirst
	pick := tempCranePick1
	if i == 2 {
		base = tempCraneSecond
		pick = tempCranePick2
	}
	heading(fmt.Sprintf("КРАН %d", i), x, y, 180, 29)
	button(pick, "Подобрать из справочника", x+760, y-1, 325, 32)
	label("Наименование крана", x, y+35, 200, 20)
	edit(base, fmt.Sprintf("Кран %d", i), x, y+57, 460, 27)
	label("Колёс на одну нитку", x+477, y+35, 195, 20)
	edit(base+1, "", x+477, y+57, 137, 27)
	label("Нагрузки на колёса (через пробел или ;)", x, y+92, 510, 20)
	edit(base+2, "", x, y+114, 520, 27)
	label("Расстояния между соседними колёсами, мм (через пробел или ;)", x+535, y+92, 555, 20)
	edit(base+3, "", x+535, y+114, 550, 27)
}
