//go:build windows

package main

import (
	"fmt"
	"math"
	"strings"
)

const (
	iceKind              = 810
	iceDiameter          = 811
	iceHeightFrom        = 812
	iceHeightTo          = 813
	iceHeightStep        = 814
	iceHighCategory      = 815
	iceMuTwo             = 816
	iceStudy             = 817
	iceHoriz             = 818
	iceCalculateButton   = 819
	iceAddButton         = 820
	iceWordButton        = 821
	iceExcelButton       = 822
	iceCalcStatus        = 823
	iceRemoveButton      = 824
	iceWindZone          = 825
	iceWindTerrain       = 826
	iceWindDrag          = 827
	iceWindEnable        = 828
	iceExtraButton       = 829
	iceChartImage        = 830
	iceChartStatus       = 831
	iceExtraBack         = 832
	iceExtraCategory     = 833
	iceExtraStudy        = 834
	iceExtraCSource      = 835
	iceExtraPulse        = 836
	iceExtraPulseSource  = 837
	iceExtraPulseCheck   = 838
	iceExtraHigh200      = 839
	iceExtraHigh300      = 840
	iceExtraHigh400      = 841
	iceExtraDensity      = 842
	iceExtraDragIced     = 843
	iceExtraFlexible     = 844
	iceDiameterLabel     = 845
	iceMuTwoLabel        = 846
	iceWindZoneLabel     = 847
	iceWindTerrainLabel  = 848
	iceWindDragLabel     = 849
	iceDragReset         = 850
	iceSiteButton        = 851
	iceWindHint          = 852
	iceExtraDensityReset = 853
)

var iceDraft = defaultIceLoad()
var iceCurrentBlockID int
var iceChartBitmap uintptr
var iceManualDrag bool
var iceProgrammaticDrag bool
var icePreserveDraft bool
var iceReturnFromSite bool

func iceHighKey(s string) string {
	if strings.HasPrefix(s, "I район") {
		return "asian_I"
	}
	if strings.HasPrefix(s, "Особая") {
		return "special"
	}
	return "ordinary"
}
func iceHighLabel(s string) string {
	switch s {
	case "asian_I":
		return "I район азиатской части"
	case "special":
		return "Особая местность: специальные данные"
	}
	return "Обычная местность (табл. 12.2)"
}
func iceShapeLabel(shape string) string {
	switch shape {
	case "round_wire":
		return "Горизонтальный провод/трос (гибкий)"
	case "round":
		return "Круглый элемент (прочий)"
	case "surface_small":
		return "Остальной элемент: малое сечение"
	case "surface_study":
		return "Остальной элемент: μ₂ (ручной коэффициент)"
	}
	return "Круглый элемент (прочий)"
}
func iceReadPanel() (IceLoadCalc, error) {
	s := iceDraft
	s.Region = romanZone(strings.TrimSpace(get(801)))
	s.Shape = "round"
	switch get(iceKind) {
	case "Горизонтальный провод/трос (гибкий)":
		s.Shape = "round_wire"
	case "Остальной элемент: малое сечение":
		s.Shape = "surface_small"
	case "Остальной элемент: μ₂ (ручной коэффициент)":
		s.Shape = "surface_study"
	}
	val := func(id int) (float64, error) {
		v := strings.TrimSpace(get(id))
		if v == "" {
			return 0, fmt.Errorf("Не заполнено поле «%s»", iceFieldName(id))
		}
		return parseNumber(v)
	}
	var e error
	if s.BaseMM, e = val(802); e != nil {
		return s, e
	}
	if s.IsRound() {
		if s.DiameterMM, e = val(iceDiameter); e != nil {
			return s, e
		}
	}
	if s.HeightStart, e = val(iceHeightFrom); e != nil {
		return s, e
	}
	if s.HeightEnd, e = val(iceHeightTo); e != nil {
		return s, e
	}
	if s.HeightStep, e = val(iceHeightStep); e != nil {
		return s, e
	}
	if s.Shape == "surface_study" {
		if s.Mu2, e = val(iceMuTwo); e != nil {
			return s, e
		}
	}
	// Special values and study references are edited on the advanced page.
	s.Horizontal = s.Shape == "round_wire"
	s.Wind.Enabled = iceWindZoneKey(get(iceWindZone)) != ""
	if iceManualDrag {
		if s.Wind.DragSource == "" || strings.HasPrefix(s.Wind.DragSource, "СП 20.13330.2016") || strings.HasPrefix(s.Wind.DragSource, "Предварительное") {
			s.Wind.DragSource = "Задано пользователем"
		}
	} else {
		s.Wind.DragSource = iceDragBasis(s.Shape)
	}
	s.Wind.Zone = iceWindZoneKey(get(iceWindZone))
	s.Wind.Terrain = iceTerrainKey(get(iceWindTerrain))
	if s.Wind.Terrain == "" {
		s.Wind.Terrain = "A"
	}
	if v := strings.TrimSpace(get(iceWindDrag)); v != "" {
		if s.Wind.Drag, e = parseNumber(v); e != nil {
			return s, e
		}
	} else {
		s.Wind.Drag = iceDefaultDrag(s.Shape)
	}
	base, _ := iceForZone(s.Region)
	s.BaseFromStudy = s.StudyReference != "" && (s.Region == "I" || s.Region == "V" || math.Abs(s.BaseMM-base) > 1e-8)
	return s, nil
}
func icePageEntered() {
	s := defaultIceLoad()
	if iceReturnFromSite {
		s = iceDraft
		zone, b, _ := project.IceInputs()
		if zone != "" {
			s.Region = zone
			s.BaseMM = b
		}
		if project.Site.WindZone != "" {
			s.Wind.Zone = project.Site.WindZone
		}
		iceReturnFromSite = false
	} else if icePreserveDraft {
		s = iceDraft
		icePreserveDraft = false
	} else if project.IceLoad != nil {
		// The stored block is a snapshot. The working form adopts the newly
		// selected local climate, but only Calculate updates that block.
		s = *project.IceLoad
		if zone, b, _ := project.IceInputs(); zone != "" && b > 0 {
			s.Region, s.BaseMM = zone, b
		}
		if project.Site.WindZone != "" {
			s.Wind.Zone = project.Site.WindZone
		}
	} else {
		zone, b, _ := project.IceInputs()
		s.Region, s.BaseMM = zone, b
	}
	// A missing database region is not zero millimetres of ice.
	if s.Region == "" || s.BaseMM <= 0 {
		// Preserve a valid prior manual/stored region; otherwise use the
		// safe initial editable default rather than an invalid zero.
		if s.Region == "" {
			s.Region = "IV"
		}
		if z, ok := iceForZone(s.Region); ok && z > 0 {
			s.BaseMM = z
		} else {
			s.Region, s.BaseMM = "IV", 15
		}
	}
	iceDraft = s
	if project.IceLoad == nil {
		if s.Wind.Zone == "" {
			s.Wind.Zone = project.Site.WindZone
		}
		if s.Wind.Terrain == "" {
			s.Wind.Terrain = "A"
		}
	}
	if s.Region != "" {
		for i, z := range iceOptions {
			if z == s.Region {
				snowSetSel(801, i)
				break
			}
		}
	}
	if s.BaseMM > 0 {
		set(802, decimalInput(s.BaseMM, 2))
	}
	for i, z := range []string{"Круглый элемент (прочий)", "Горизонтальный провод/трос (гибкий)", "Остальной элемент: малое сечение", "Остальной элемент: μ₂ (ручной коэффициент)"} {
		if z == iceShapeLabel(s.Shape) {
			snowSetSel(iceKind, i)
			break
		}
	}
	set(iceDiameter, decimalInput(s.DiameterMM, 2))
	set(iceHeightFrom, decimalInput(s.HeightStart, 2))
	set(iceHeightTo, decimalInput(s.HeightEnd, 2))
	set(iceHeightStep, decimalInput(s.HeightStep, 2))
	snowSetSel(iceWindZone, 0) // no wind is an explicit, selectable state
	for i, z := range windOptions[:len(windOptions)-1] {
		if romanZone(z) == romanZone(s.Wind.Zone) && s.Wind.Zone != "" {
			snowSetSel(iceWindZone, i+1)
			break
		}
	}
	for i, z := range []string{"A", "B", "C"} {
		if z == iceTerrainKey(s.Wind.Terrain) {
			snowSetSel(iceWindTerrain, i)
			break
		}
	}
	if s.Wind.Drag <= 0 {
		s.Wind.Drag = iceDefaultDrag(s.Shape)
	}
	iceManualDrag = s.Wind.DragSource == "Задано пользователем"
	if s.Wind.Drag > 0 {
		iceSetDrag(decimalInput(s.Wind.Drag, 3))
	} else {
		iceSetDrag("")
	}

	icePanelStatus()
	iceUpdateContext()
}
func icePanelStatus() {
	s := iceDraft
	if len(s.Points) == 0 {
		set(iceCalcStatus, "Укажите исходные данные и нажмите «Рассчитать».")
		return
	}
	first, last := s.Points[0], s.Points[len(s.Points)-1]
	set(iceCalcStatus, fmt.Sprintf("Район %s · %d отметок · Гололёд: %s…%s / %s…%s %s", s.Region, len(s.Points), decimalInput(first.Norm, 2), decimalInput(last.Norm, 2), decimalInput(first.Design, 2), decimalInput(last.Design, 2), s.NormUnit()))
	set(iceChartStatus, iceWindCompleteness(s))
}
func iceRecalculate() bool {
	s, e := iceReadPanel()
	if e != nil {
		message(e.Error(), true)
		return false
	}
	s, e = IceCompute(s)
	if e != nil {
		message(e.Error(), true)
		return false
	}
	iceDraft = s
	icePanelStatus()
	iceDisplayChart(s)
	project.initReportBlocks(tempSections.Sections)
	iceCurrentBlockID = project.putIceCalc(iceCurrentBlockID, s)
	projectHomeOverview()
	return true
}
func iceAction(id int) bool {
	switch id {
	case iceKind:
		if !iceManualDrag {
			iceSetDrag(decimalInput(iceDefaultDrag(iceShapeKey(get(iceKind))), 2))
		}
		iceUpdateContext()
		return true
	case iceDragReset:
		iceManualDrag = false
		iceSetDrag(decimalInput(iceDefaultDrag(iceShapeKey(get(iceKind))), 2))
		iceDraft.Wind.DragSource = iceDragBasis(iceShapeKey(get(iceKind)))
		return true
	case iceSiteButton:
		if s, err := iceReadPanel(); err == nil {
			iceDraft = s
		}
		iceReturnFromSite = true
		siteReturnPage = "ice"
		switchPage("site")
		return true
	case iceExtraDensityReset:
		set(iceExtraDensity, "900")
		return true
	case iceExtraButton:
		if s, e := iceReadPanel(); e == nil {
			iceDraft = s
		}
		switchPage("ice_extra")
		iceAdvancedEntered()
		return true
	case iceExtraBack:
		if !iceAdvancedStore() {
			return true
		}
		icePreserveDraft = true
		switchPage("ice")
		icePageEntered()
		iceUpdateContext()
		return true
	case iceCalculateButton:
		iceRecalculate()
		return true
	case iceAddButton:
		if !iceRecalculate() {
			return true
		}
		name, ok := askReportBlockName(project.suggestedCurrentName("ice", iceCurrentBlockID))
		if !ok {
			return true
		}
		project.renameBlock(iceCurrentBlockID, name)
		iceCurrentBlockID = 0
		editingReportBlockID = 0
		project.IceLoad = nil
		iceDraft = defaultIceLoad()
		icePageEntered()
		projectHomeOverview()
		return true
	case iceRemoveButton:
		// Reset is not deletion: the previously calculated block stays stored.
		iceCurrentBlockID = 0
		editingReportBlockID = 0
		project.IceLoad = nil
		iceDraft = defaultIceLoad()
		icePageEntered()
		projectHomeOverview()
		return true

	case iceWordButton, iceExcelButton:
		if !iceRecalculate() {
			return true
		}
		ext, name := "docx", "Гололёдная нагрузка.docx"
		if id == iceExcelButton {
			ext, name = "xlsx", "Гололёдная нагрузка.xlsx"
		}
		path := chooseSave(ext, strings.ToUpper(ext), name, false)
		if path == "" {
			return true
		}
		saveAndNotify(path, func(file string) error {
			if id == iceWordButton {
				return ExportIceWord(iceDraft, file)
			}
			return ExportIceExcel(iceDraft, file)
		})
		return true
	}
	return false
}

func iceAdvancedEntered() {
	s := iceDraft
	for i, z := range []string{"Обычная", "I район азиатской части", "Специальная"} {
		if iceHighKey(z) == s.HighZone {
			snowSetSel(iceExtraCategory, i)
			break
		}
	}
	set(iceExtraStudy, s.StudyReference)
	dragSource := s.Wind.DragSource
	if dragSource == "Значение Cx задано пользователем" || dragSource == "Значение Cx по умолчанию" || dragSource == "Задано пользователем" {
		dragSource = ""
	}
	set(iceExtraCSource, dragSource)
	snowBox(iceExtraFlexible, s.IsFlexible())
	set(iceExtraDensity, decimalInput(s.density()*1000, 0))
	if s.Wind.DragIced > 0 {
		set(iceExtraDragIced, decimalInput(s.Wind.DragIced, 3))
	} else {
		set(iceExtraDragIced, "")
	}
	if s.Wind.PulseVerified {
		set(iceExtraPulse, decimalInput(s.Wind.PulseRatio, 4))
	} else {
		set(iceExtraPulse, "")
	}
	snowBox(iceExtraPulseCheck, s.Wind.PulseVerified)
	set(iceExtraPulseSource, s.Wind.PulseSource)
	for i, id := range []int{iceExtraHigh200, iceExtraHigh300, iceExtraHigh400} {
		if len(s.HighValuesMM) == 3 {
			set(id, decimalInput(s.HighValuesMM[i], 2))
		} else {
			set(id, "")
		}
	}
}
func iceAdvancedStore() bool {
	s := iceDraft
	s.HighZone = iceHighKey(get(iceExtraCategory))
	s.StudyReference = strings.TrimSpace(get(iceExtraStudy))
	s.Wind.DragSource = strings.TrimSpace(get(iceExtraCSource))
	s.Wind.Flexible = snowIsChecked(iceExtraFlexible)
	if v := strings.TrimSpace(get(iceExtraDragIced)); v != "" {
		value, e := parseNumber(v)
		if e != nil {
			message("Проверьте c обледеневшего элемента", true)
			return false
		}
		s.Wind.DragIced = value
	} else {
		s.Wind.DragIced = 0
	}
	if v := strings.TrimSpace(get(iceExtraDensity)); v != "" {
		value, e := parseNumber(v)
		if e != nil {
			message("Проверьте плотность льда", true)
			return false
		}
		s.DensityKgM3 = value
	} else {
		s.DensityKgM3 = 0
	}
	s.Wind.PulseSource = strings.TrimSpace(get(iceExtraPulseSource))
	s.Wind.PulseVerified = snowIsChecked(iceExtraPulseCheck)
	s.Wind.PulseRatio = 0
	if s.Wind.PulseVerified {
		var e error
		s.Wind.PulseRatio, e = parseNumber(get(iceExtraPulse))
		if e != nil {
			message("Некорректное значение wp/wm", true)
			return false
		}
	}
	values := []float64{}
	for _, id := range []int{iceExtraHigh200, iceExtraHigh300, iceExtraHigh400} {
		raw := strings.TrimSpace(get(id))
		if raw == "" {
			values = append(values, 0)
			continue
		}
		v, e := parseNumber(raw)
		if e != nil || v <= 0 {
			message("Проверьте специальные толщины для высот 200/300/400 м", true)
			return false
		}
		values = append(values, v)
	}
	if values[0] > 0 || values[1] > 0 || values[2] > 0 {
		for _, v := range values {
			if v == 0 {
				message("Для специальных высотных данных заполните все три значения b", true)
				return false
			}
		}
		s.HighValuesMM = values
	} else {
		s.HighValuesMM = nil
	}
	iceDraft = s
	return true
}

func iceUpdateContext() {
	if activePage != "ice" {
		return
	}
	shape := get(iceKind)
	isRound := shape == "Круглый элемент (прочий)" || shape == "Горизонтальный провод/трос (гибкий)"
	for _, id := range []int{iceDiameterLabel, iceDiameter} {
		if controls[id] != 0 {
			if isRound {
				pShow.Call(controls[id], 1)
			} else {
				pShow.Call(controls[id], 0)
			}
		}
	}
	isSpecial := shape == "Остальной элемент: μ₂ (ручной коэффициент)"
	for _, id := range []int{iceMuTwoLabel, iceMuTwo} {
		if controls[id] != 0 {
			if isSpecial {
				pShow.Call(controls[id], 1)
			} else {
				pShow.Call(controls[id], 0)
			}
		}
	}
	for _, id := range []int{iceWindZoneLabel, iceWindZone, iceWindTerrainLabel, iceWindTerrain, iceWindDragLabel, iceWindDrag, iceDragReset} {
		if controls[id] != 0 {
			pShow.Call(controls[id], 1)
		}
	}
}

// Normal selection is deliberately editable. The engineer may replace Cx at any time.
func iceShapeKey(label string) string {
	switch label {
	case "Горизонтальный провод/трос (гибкий)":
		return "round_wire"
	case "Остальной элемент: малое сечение":
		return "surface_small"
	case "Остальной элемент: μ₂ (ручной коэффициент)":
		return "surface_study"
	default:
		return "round"
	}
}
func iceDefaultDrag(shape string) float64 {
	// SP20 appendix В.1.12 specifies c_x=1.2 specifically for wires and ropes.
	// For other shapes an illustrative default must not masquerade as a normative one.
	if shape == "round_wire" {
		return 1.2
	}
	return 1.2 // preliminary for manual engineering review, explicitly labeled in the interface
}
func iceDragBasis(shape string) string {
	if shape == "round_wire" {
		return "СП 20.13330.2016, п. В.1.12"
	}
	return "Значение Cx по умолчанию"
}
func iceTerrainKey(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "A"
	}
	switch []rune(strings.ToUpper(label))[0] {
	case 'А', 'A':
		return "A"
	case 'В', 'B':
		return "B"
	case 'С', 'C':
		return "C"
	}
	return label
}

func iceSetDrag(value string) {
	iceProgrammaticDrag = true
	set(iceWindDrag, value)
	iceProgrammaticDrag = false
}

func iceWindZoneKey(label string) string {
	if strings.HasPrefix(label, "—") || strings.TrimSpace(label) == "" {
		return ""
	}
	return romanZone(strings.TrimSpace(label))
}

func iceFieldName(id int) string {
	switch id {
	case 802:
		return "Нормативная толщина гололёда b, мм"
	case iceDiameter:
		return "Диаметр элемента, мм"
	case iceHeightFrom:
		return "Начальная высота, м"
	case iceHeightTo:
		return "Конечная высота, м"
	case iceHeightStep:
		return "Шаг высотных отметок, м"
	case iceMuTwo:
		return "Коэффициент μ₂"
	default:
		return "Исходные данные"
	}
}
