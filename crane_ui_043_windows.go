//go:build windows

package main

import (
	"fmt"
)

var craneSetCurrentID int
var craneNewSetRequested bool
var cranePlaceCurrentBlockID int

const (
	tempSaveNew            = 12751
	craneSetSaveNew        = 12752
	cranePlaceSaveNew      = 12753
	tempResetCurrent       = 12754
	craneSetResetCurrent   = 12755
	cranePlaceResetCurrent = 12756
)

func filterNonCraneSections(sections []TempSelection) []TempSelection {
	out := make([]TempSelection, 0, len(sections))
	for _, s := range sections {
		if !tempIsCrane(s) {
			out = append(out, s)
		}
	}
	return out
}
func syncCurrentCraneSet() {
	cranes := currentCraneSelections(tempSections.Sections)
	if len(cranes) == 0 {
		return
	}
	wasManaged := project.ReportManaged
	project.initReportBlocks(tempSections.Sections)
	if craneSetCurrentID == 0 && !craneNewSetRequested {
		for _, b := range project.SavedBlocks {
			if b.Kind != "crane_set" || len(b.CraneSet) != len(cranes) {
				continue
			}
			same := true
			for j, cr := range cranes {
				if tempKey(cr) != tempKey(b.CraneSet[j]) || len(cr.Wheels) != len(b.CraneSet[j].Wheels) {
					same = false
					break
				}
				for k, w := range cr.Wheels {
					if w != b.CraneSet[j].Wheels[k] {
						same = false
						break
					}
				}
			}
			if same {
				craneSetCurrentID = b.ID
				break
			}
		}
	}
	_ = wasManaged
	craneSetCurrentID = project.putCraneSet(craneSetCurrentID, cranes)
	craneNewSetRequested = false
	if craneSetCurrentID > 0 {
		project.recalculateLinkedCranePlacements(craneSetCurrentID)
	}
}
func craneSaveSetNew() {
	syncCurrentCraneSet()
	if craneSetCurrentID == 0 {
		message("Сначала задайте краны", true)
		return
	}
	name, ok := askReportBlockName(project.suggestedCurrentName("crane_set", craneSetCurrentID))
	if !ok {
		return
	}
	project.renameBlock(craneSetCurrentID, name)
	tempSections.Sections = append(filterNonCraneSections(tempSections.Sections), tempEmptyCrane())
	craneSetCurrentID = 0
	craneNewSetRequested = true
	cranePlaceCurrentBlockID = 0
	cranePlaceResult = CranePlacementResult{}
	tempRefreshChosen()
	tempCraneSelectRow(0)
}
func cranePlacePopulateResult() {
	rows := [][]string{}
	for i, c := range cranePlaceResult.Cases {
		x := ""
		if len(c.Points) > 0 {
			x = tempRU(c.Points[0].XMM, 1)
		}
		rows = append(rows, []string{fmt.Sprint(i + 1), c.Name, tempRU(c.Value, 3) + " " + c.Units, x})
	}
	tempRows(cranePlaceResults, rows)
	if len(rows) > 0 {
		tempSelectFirst(cranePlaceResults)
		cranePlaceRender(0)
	} else {
		set(cranePlaceDescription, "")
	}
}
func craneSavePlacementNew() {
	if len(cranePlaceResult.Cases) == 0 {
		message("Сначала рассчитайте расстановку", true)
		return
	}
	syncCurrentCraneSet()
	if craneSetCurrentID == 0 {
		message("Для расстановки необходим набор кранов", true)
		return
	}
	cranePlaceCurrentBlockID = project.putCranePlacement(cranePlaceCurrentBlockID, craneSetCurrentID, get(cranePlaceLengths), cranePlaceResult)
	if cranePlaceCurrentBlockID == 0 {
		message("Не удалось сохранить расстановку", true)
		return
	}
	name, ok := askReportBlockName(project.suggestedCurrentName("crane_placement", cranePlaceCurrentBlockID))
	if !ok {
		return
	}
	project.renameBlock(cranePlaceCurrentBlockID, name)
	cranePlaceCurrentBlockID = 0
	cranePlaceResult = CranePlacementResult{}
	set(cranePlaceLengths, "6")
	cranePlacePopulateResult()
	projectHomeOverview()
}
func tempSaveSelectedAndNew() {
	isTransport := activePage == "temp_sp35"
	listID := tempSp20Chosen
	kind := "temporary"
	if isTransport {
		listID = tempSp35Chosen
		kind = "transport"
	}
	n := tempSelectedPhysical(listID, tempSelected(listID))
	if n < 0 {
		message("Выберите рассчитанную нагрузку в нижней таблице", true)
		return
	}
	project.initReportBlocks(tempSections.Sections)
	project.ensureTempSelections(tempSections.Sections)
	selected := tempSections.Sections[n]
	target := 0
	for _, b := range project.SavedBlocks {
		if b.Kind == "temporary" && b.Temporary != nil && tempKey(*b.Temporary) == tempKey(selected) {
			target = b.ID
			break
		}
	}
	if target == 0 {
		message("Не удалось найти расчёт в ведомости", true)
		return
	}
	name, ok := askReportBlockName(project.suggestedCurrentName(kind, target))
	if !ok {
		return
	}
	project.renameBlock(target, name)
	list := tempSP20List
	if isTransport {
		list = tempSP35List
	}
	if controls[list] != 0 {
		send(list, LB_SETCURSEL, ^uintptr(0), 0)
	}
	tempRefreshChosen()
}
func craneUi043Action(id int) bool {
	switch id {
	case tempSaveNew:
		if activePage == "temp_sp20" || activePage == "temp_sp35" {
			tempSaveSelectedAndNew()
			return true
		}
	case craneSetSaveNew:
		craneSaveSetNew()
		return true
	case cranePlaceSaveNew:
		craneSavePlacementNew()
		return true
	case tempResetCurrent:
		listID := tempSp20Chosen
		if activePage == "temp_sp35" {
			listID = tempSp35Chosen
		}
		index := tempSelectedPhysical(listID, tempSelected(listID))
		if index < 0 {
			message("Выберите нагрузку для сброса", true)
			return true
		}
		if !askConfirm("Удалить выбранную нагрузку из проекта?") {
			return true
		}
		if index < len(tempSections.Sections) {
			key := tempKey(tempSections.Sections[index])
			for _, b := range append([]SavedReportBlock(nil), project.SavedBlocks...) {
				if b.Kind == "temporary" && b.Temporary != nil && tempKey(*b.Temporary) == key {
					project.removeBlock(b.ID)
					break
				}
			}
			tempSections.Sections = append(tempSections.Sections[:index], tempSections.Sections[index+1:]...)
		}
		tempRefreshChosen()
		return true
	case craneSetResetCurrent:
		if craneSetCurrentID > 0 {
			if !askConfirm("Удалить текущий набор кранов и связанные расстановки?") {
				return true
			}
			project.removeBlock(craneSetCurrentID)
		}
		tempSections.Sections = append(filterNonCraneSections(tempSections.Sections), tempEmptyCrane())
		craneSetCurrentID = 0
		craneNewSetRequested = true
		cranePlaceCurrentBlockID = 0
		cranePlaceResult = CranePlacementResult{}
		tempRefreshChosen()
		tempCraneSelectRow(0)
		return true
	case cranePlaceResetCurrent:
		if cranePlaceCurrentBlockID > 0 {
			if !askConfirm("Удалить текущую расстановку из проекта?") {
				return true
			}
			project.removeBlock(cranePlaceCurrentBlockID)
		}
		cranePlaceCurrentBlockID = 0
		cranePlaceResult = CranePlacementResult{}
		set(cranePlaceLengths, "6")
		cranePlacePopulateResult()
		projectHomeOverview()
		return true
	}
	return false
}
