//go:build windows

package main

import (
	"fmt"
	"unsafe"
)

const (
	reportCompositionList = 14901
	reportBlockMoveUp     = 14902
	reportBlockMoveDown   = 14903
	reportBlockRemove     = 14904
	reportBlockHelp       = 14905
	reportBlockRename     = 14906
)

var reportCompositionIDs []int
var editingReportBlockID int
var editingWeightIndex = -1
var editingTemporaryIndex = -1

func compositionEntered() {
	project.initReportBlocks(tempSections.Sections)
	project.ensureTempSelections(tempSections.Sections)
	compositionRefresh()
}
func compositionRefresh() {
	groups := []string{"Собственный вес", "Временные нагрузки", "Снеговые нагрузки", "Ветровые нагрузки", "Гололёдные нагрузки", "Транспортные нагрузки", "Мостовые краны"}
	rows := [][]string{}
	reportCompositionIDs = nil
	add := func(b SavedReportBlock, serial int, indent string) {
		rows = append(rows, []string{fmt.Sprint(serial), "", indent + b.Title, "Править", "×"})
		reportCompositionIDs = append(reportCompositionIDs, b.ID)
	}
	for _, group := range groups {
		count := 0
		for _, b := range project.SavedBlocks {
			if savedGroup(b) == group && b.Kind != "crane_placement" {
				count++
			}
		}
		rows = append(rows, []string{"", fmt.Sprintf("%s · %d", group, count), "", "", ""})
		reportCompositionIDs = append(reportCompositionIDs, 0)
		serial := 0
		for _, b := range project.SavedBlocks {
			if savedGroup(b) != group || b.Kind == "crane_placement" {
				continue
			}
			serial++
			add(b, serial, "")
			if b.Kind == "crane_set" {
				for _, child := range project.SavedBlocks {
					if child.Kind == "crane_placement" && child.ParentID == b.ID {
						serial++
						add(child, serial, "    ↳ ")
					}
				}
			}
		}
	}
	tempRows(reportCompositionList, rows)
	projectHomeOverview()
}
func compositionIDAt(row int) int {
	if row >= 0 && row < len(reportCompositionIDs) {
		return reportCompositionIDs[row]
	}
	return 0
}
func compositionSelectedID() int { return compositionIDAt(tempSelected(reportCompositionList)) }
func compositionCommand(id int) bool {
	switch id {
	case reportBlockMoveUp, reportBlockMoveDown, reportBlockRemove:
		blockID := compositionSelectedID()
		if blockID == 0 {
			return true
		}
		if id == reportBlockRemove {
			project.removeBlock(blockID)
		} else if id == reportBlockMoveUp {
			project.moveBlock(blockID, -1)
		} else {
			project.moveBlock(blockID, 1)
		}
		compositionRefresh()
		compositionKeepSelection(blockID)
		return true
	case reportBlockRename:
		blockID := compositionSelectedID()
		if blockID == 0 {
			return true
		}
		for _, b := range project.SavedBlocks {
			if b.ID == blockID {
				name, ok := askReportBlockName(b.Title)
				if ok {
					project.renameBlock(blockID, name)
					compositionRefresh()
					compositionKeepSelection(blockID)
				}
				break
			}
		}
		return true
	case reportBlockHelp:
		message("Дважды щёлкните по схеме, чтобы открыть расчёт. Стрелки меняют порядок схем внутри раздела. × исключает схему из отчёта.", false)
		return true
	}
	return false
}
func compositionClick(row, sub int, double bool) bool {
	blockID := compositionIDAt(row)
	if blockID == 0 {
		return true
	}
	if double {
		for _, b := range project.SavedBlocks {
			if b.ID == blockID {
				name, ok := askReportBlockName(b.Title)
				if ok {
					project.renameBlock(blockID, name)
					compositionRefresh()
					compositionKeepSelection(blockID)
				}
				break
			}
		}
		return true
	}
	if sub == 4 {
		project.removeBlock(blockID)
		compositionRefresh()
		return true
	}
	if sub != 3 {
		return false
	}
	for _, b := range project.SavedBlocks {
		if b.ID != blockID {
			continue
		}
		editingReportBlockID = blockID
		editingWeightIndex = -1
		editingTemporaryIndex = -1
		switch b.Kind {
		case "wind":
			if b.WindCalc != nil {
				windCurrentBlockID = blockID
				c := *b.WindCalc
				windDraft = c.Input
				windUnit = "kpa"
				if c.Input.DisplayUnit == "kgf" {
					windUnit = "kgf"
				}
				windCurrentResult = &c
				switchPage("wind")
				windSetForm()
			}
		case "snow":
			if b.Snow != nil {
				snowCurrentBlockID = blockID
				v := *b.Snow
				snowDraft = v
				project.SnowCalc = &v
				switchPage("snow")
				snowPageEntered()
			}
		case "ice":
			if b.Ice != nil {
				iceCurrentBlockID = blockID
				v := *b.Ice
				iceDraft = v
				project.IceLoad = &v
				switchPage("ice")
				icePageEntered()
			}
		case "weight":
			weightCurrentBlockID = blockID
			if b.WeightCalc != nil {
				w := *b.WeightCalc
				project.Weight = &w
			}
			switchPage("weight")
			weightPageEntered()
			if b.Weight != nil {
				for i, row := range weightCurrent().Rows {
					if row.Name == b.Weight.Name {
						editingWeightIndex = i
						weightRenderRows(i)
						break
					}
				}
			}
		case "crane_set":
			tempSections.Sections = append(filterNonCraneSections(tempSections.Sections), b.CraneSet...)
			craneSetCurrentID = blockID
			cranePlaceCurrentBlockID = 0
			switchPage("temp_cranes")
			tempRefreshChosen()
			tempCraneSelectRow(0)
		case "crane_placement":
			for _, parent := range project.SavedBlocks {
				if parent.ID == b.ParentID && parent.Kind == "crane_set" {
					tempSections.Sections = append(filterNonCraneSections(tempSections.Sections), parent.CraneSet...)
					craneSetCurrentID = parent.ID
					break
				}
			}
			if b.Placement != nil {
				cranePlaceResult = *b.Placement
				cranePlaceCurrentBlockID = blockID
				switchPage("temp_crane_place")
				set(cranePlaceLengths, b.SpansText)
				cranePlacePopulateResult()
			}
		case "temporary":
			if b.Temporary != nil {
				editingTemporaryIndex = -1
				for i, s := range tempSections.Sections {
					if tempKey(s) == tempKey(*b.Temporary) {
						editingTemporaryIndex = i
						break
					}
				}
				if editingTemporaryIndex < 0 {
					tempSections.Sections = append(tempSections.Sections, *b.Temporary)
					editingTemporaryIndex = len(tempSections.Sections) - 1
				}
				switch savedGroup(b) {
				case "Мостовые краны":
					switchPage("temp_cranes")
				case "Транспортные нагрузки":
					switchPage("temp_sp35")
				default:
					switchPage("temp_sp20")
				}
				tempPageEntered()
				var listID int
				switch savedGroup(b) {
				case "Мостовые краны":
					listID = tempCraneChosen
				case "Транспортные нагрузки":
					listID = tempSp35Chosen
				default:
					listID = tempSp20Chosen
				}
				if listID != 0 && controls[listID] != 0 {
					for row, phys := range tempSummaryRows[listID] {
						if phys == editingTemporaryIndex {
							tempFocusReportRow(listID, row)
							break
						}
					}
				}
			}
		}
		return true
	}
	return true
}

// Focus the selected calculation when opening it from the report composition.
func tempFocusReportRow(id, row int) {
	h := controls[id]
	if h == 0 || row < 0 {
		return
	}
	off := listItem{State: 0, StateMask: LVIS_SELECTED}
	pSend.Call(h, LVM_SETITEMSTATE, ^uintptr(0), uintptr(unsafe.Pointer(&off)))
	state := listItem{State: LVIS_SELECTED, StateMask: LVIS_SELECTED}
	pSend.Call(h, LVM_SETITEMSTATE, uintptr(row), uintptr(unsafe.Pointer(&state)))
	pSend.Call(h, 0x1013, uintptr(row), 0)
}

// Persist edits to a saved block when the user navigates away from its module.
func persistReportBlockEdit() {
	if editingReportBlockID <= 0 {
		return
	}
	for _, b := range project.SavedBlocks {
		if b.ID != editingReportBlockID {
			continue
		}
		if b.Kind == "weight" && editingWeightIndex >= 0 && project.Weight != nil && editingWeightIndex < len(project.Weight.Rows) {
			row := project.Weight.Rows[editingWeightIndex]
			v := ProjectWeightLine{row.Name, kgfM2ToKpa(row.NormKgf()), row.Gamma, kgfM2ToKpa(row.NormKgf() * row.Gamma)}
			b.Weight = &v
			b.Title = v.Name
			project.updateBlock(b.ID, b)
		}
		if b.Kind == "temporary" && editingTemporaryIndex >= 0 && editingTemporaryIndex < len(tempSections.Sections) {
			v := tempSections.Sections[editingTemporaryIndex]
			b.Temporary = &v
			project.updateBlock(b.ID, b)
		}
		return
	}
}

func compositionKeepSelection(id int) {
	for row, v := range reportCompositionIDs {
		if v == id {
			tempFocusReportRow(reportCompositionList, row)
			return
		}
	}
}
