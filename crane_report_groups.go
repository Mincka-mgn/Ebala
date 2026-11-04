package main

import "fmt"

// A crane set owns any number of placement calculations. In the common report
// each placement follows its own parent set, not a global placements section.
type CraneReportGroup struct {
	ID         int                    `json:"id"`
	Title      string                 `json:"title"`
	Cranes     []TempSelection        `json:"cranes"`
	Placements []CraneReportPlacement `json:"placements,omitempty"`
}
type CraneReportPlacement struct {
	Title     string               `json:"title"`
	SpansText string               `json:"spans_text"`
	Result    CranePlacementResult `json:"result"`
}

func currentCraneSelections(sections []TempSelection) []TempSelection {
	var out []TempSelection
	for _, s := range sections {
		if tempIsCrane(s) && len(s.Wheels) > 0 {
			out = append(out, cloneTempSelection(s))
		}
	}
	return out
}
func cloneTempSelection(s TempSelection) TempSelection {
	t := s
	if s.Params != nil {
		t.Params = make(map[string]interface{}, len(s.Params))
		for k, v := range s.Params {
			t.Params[k] = v
		}
	}
	t.Wheels = append([]TempWheel(nil), s.Wheels...)
	return t
}
func (p *Project) putCraneSet(id int, cranes []TempSelection) int {
	if len(cranes) == 0 {
		return id
	}
	snapshot := make([]TempSelection, len(cranes))
	for i, v := range cranes {
		snapshot[i] = cloneTempSelection(v)
	}
	b := SavedReportBlock{Kind: "crane_set", Title: "Мостовые краны", CraneSet: snapshot}
	if id > 0 {
		for _, old := range p.SavedBlocks {
			if old.ID == id && old.Kind == "crane_set" {
				b.Title = old.Title
				if p.updateBlock(id, b) {
					return id
				}
			}
		}
	}
	return p.addBlock(b)
}
func (p *Project) putCranePlacement(id, parent int, spans string, r CranePlacementResult) int {
	if parent <= 0 || len(r.Cases) == 0 {
		return 0
	}
	snap := r
	b := SavedReportBlock{Kind: "crane_placement", Title: "Расстановка кранов", ParentID: parent, SpansText: spans, Placement: &snap}
	if id > 0 {
		for _, old := range p.SavedBlocks {
			if old.ID == id && old.Kind == "crane_placement" && old.ParentID == parent {
				b.Title = old.Title
				if p.updateBlock(id, b) {
					return id
				}
			}
		}
	}
	return p.addBlock(b)
}
func (p Project) craneGroups() []CraneReportGroup {
	var groups []CraneReportGroup
	for _, b := range p.SavedBlocks {
		if b.Kind != "crane_set" {
			continue
		}
		g := CraneReportGroup{ID: b.ID, Title: b.Title, Cranes: b.CraneSet}
		for _, child := range p.SavedBlocks {
			if child.Kind == "crane_placement" && child.ParentID == b.ID && child.Placement != nil {
				g.Placements = append(g.Placements, CraneReportPlacement{Title: child.Title, SpansText: child.SpansText, Result: *child.Placement})
			}
		}
		groups = append(groups, g)
	}
	return groups
}

func craneTrainsFromSelections(cranes []TempSelection) ([]CraneTrain, error) {
	trains := []CraneTrain{}
	for _, s := range cranes {
		if !tempIsCrane(s) || len(s.Wheels) == 0 {
			continue
		}
		t := CraneTrain{Name: s.Title, OverallMM: tempNumber(s, "overall_length_mm")}
		for _, wheel := range s.Wheels {
			if !wheel.XKnown {
				return nil, fmt.Errorf("%s: задайте координаты колёс", s.Title)
			}
			t.WheelXMM = append(t.WheelXMM, wheel.XMM)
			t.ForcesKN = append(t.ForcesKN, wheel.PKN)
		}
		trains = append(trains, t)
	}
	if len(trains) == 0 {
		return nil, fmt.Errorf("Набор кранов пуст")
	}
	return trains, nil
}
func (p *Project) recalculateLinkedCranePlacements(parentID int) {
	var cranes []TempSelection
	for _, b := range p.SavedBlocks {
		if b.ID == parentID && b.Kind == "crane_set" {
			cranes = b.CraneSet
			break
		}
	}
	trains, err := craneTrainsFromSelections(cranes)
	for i := range p.SavedBlocks {
		child := &p.SavedBlocks[i]
		if child.Kind != "crane_placement" || child.ParentID != parentID || child.Placement == nil {
			continue
		}
		if err != nil {
			child.Placement.Cases = nil
			continue
		}
		res, e := CalculateCranePlacements(child.Placement.SpansMM, trains)
		if e != nil {
			child.Placement.Cases = nil
			continue
		}
		child.Placement = &res
	}
}
