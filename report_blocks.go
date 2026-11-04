package main

import (
	"fmt"
	"strings"
)

func savedGroup(b SavedReportBlock) string {
	switch b.Kind {
	case "weight":
		return "Собственный вес"
	case "snow":
		return "Снеговые нагрузки"
	case "ice":
		return "Гололёдные нагрузки"
	case "crane_set", "crane_placement":
		return "Мостовые краны"
	case "temporary":
		if b.Temporary != nil {
			if tempIsCrane(*b.Temporary) {
				return "Мостовые краны"
			}
			if tempTransport(*b.Temporary) {
				return "Транспортные нагрузки"
			}
		}
		return "Временные нагрузки"
	}
	return "Другие нагрузки"
}
func (p *Project) nextBlockID() int {
	n := 1
	for _, b := range p.SavedBlocks {
		if b.ID >= n {
			n = b.ID + 1
		}
	}
	return n
}
func (p *Project) addBlock(b SavedReportBlock) int {
	p.ReportManaged = true
	b.ID = p.nextBlockID()
	p.SavedBlocks = append(p.SavedBlocks, b)
	return b.ID
}
func (p *Project) updateBlock(id int, b SavedReportBlock) bool {
	for i := range p.SavedBlocks {
		if p.SavedBlocks[i].ID == id {
			b.ID = id
			p.SavedBlocks[i] = b
			return true
		}
	}
	return false
}
func (p *Project) removeBlock(id int) bool {
	for i, b := range p.SavedBlocks {
		if b.ID == id {
			if b.Kind == "crane_set" {
				filtered := p.SavedBlocks[:0]
				for _, child := range p.SavedBlocks {
					if child.ID != id && !(child.Kind == "crane_placement" && child.ParentID == id) {
						filtered = append(filtered, child)
					}
				}
				p.SavedBlocks = filtered
				return true
			}
			if b.Kind == "temporary" && b.Temporary != nil {
				p.SuppressedSelections = append(p.SuppressedSelections, tempKey(*b.Temporary))
			}
			p.SavedBlocks = append(p.SavedBlocks[:i], p.SavedBlocks[i+1:]...)
			return true
		}
	}
	return false
}
func (p *Project) moveBlock(id, delta int) bool {
	for i, b := range p.SavedBlocks {
		if b.ID == id {
			j := i + delta
			for j >= 0 && j < len(p.SavedBlocks) && savedGroup(p.SavedBlocks[j]) != savedGroup(b) {
				j += delta
			}
			if j < 0 || j >= len(p.SavedBlocks) {
				return false
			}
			p.SavedBlocks[i], p.SavedBlocks[j] = p.SavedBlocks[j], p.SavedBlocks[i]
			return true
		}
	}
	return false
}

// Convert the old single-current-snow / single-current-ice project once.
func (p *Project) initReportBlocks(sections []TempSelection) {
	if p.ReportManaged {
		return
	}
	if p.Weight != nil && len(p.Weight.Rows) > 0 {
		w := *p.Weight
		title := w.Name
		if strings.TrimSpace(title) == "" || title == "Покрытие" {
			title = "Собственный вес"
		}
		p.addBlock(SavedReportBlock{Kind: "weight", Title: title, WeightCalc: &w})
	} else if p.Weight == nil && len(p.Layers) > 0 {
		w := p.CurrentWeight()
		if len(w.Rows) > 0 {
			p.addBlock(SavedReportBlock{Kind: "weight", Title: "Собственный вес", WeightCalc: &w})
		}
	}
	if p.SnowCalc != nil && p.SnowCalc.Enabled {
		v := *p.SnowCalc
		p.addBlock(SavedReportBlock{Kind: "snow", Title: v.ReportName(), Snow: &v})
	}
	if p.IceLoad != nil && p.IceLoad.Enabled {
		v := *p.IceLoad
		p.addBlock(SavedReportBlock{Kind: "ice", Title: v.Description(), Ice: &v})
	}
	var cranes []TempSelection
	for _, s := range sections {
		if tempIsCrane(s) {
			if len(s.Wheels) > 0 {
				cranes = append(cranes, cloneTempSelection(s))
			}
			continue
		}
		v := s
		p.addBlock(SavedReportBlock{Kind: "temporary", Title: v.Title, Temporary: &v})
	}
	if len(cranes) > 0 {
		p.putCraneSet(0, cranes)
	}
	p.ReportManaged = true
}

// Explicit user actions keep a stable block ID. An ordinary repeated calculation
// updates the same block rather than adding indistinguishable copies.
func (p *Project) putSnowCalc(id int, s SnowCalc) int {
	v := s
	v.Enabled = true
	b := SavedReportBlock{Kind: "snow", Title: "Снеговая нагрузка", Snow: &v}
	if id > 0 {
		for _, old := range p.SavedBlocks {
			if old.ID == id && old.Kind == "snow" {
				b.Title = old.Title
				break
			}
		}
		if p.updateBlock(id, b) {
			p.SnowCalc = &v
			return id
		}
	}
	id = p.addBlock(b)
	p.SnowCalc = &v
	return id
}
func (p *Project) putIceCalc(id int, s IceLoadCalc) int {
	v := s
	v.Enabled = true
	b := SavedReportBlock{Kind: "ice", Title: "Гололёдная нагрузка", Ice: &v}
	if id > 0 {
		for _, old := range p.SavedBlocks {
			if old.ID == id && old.Kind == "ice" {
				b.Title = old.Title
				break
			}
		}
		if p.updateBlock(id, b) {
			p.IceLoad = &v
			return id
		}
	}
	id = p.addBlock(b)
	p.IceLoad = &v
	return id
}
func (p *Project) putWeightCalc(id int, w WeightCalc) int {
	v := w
	v.Rows = append([]WeightRow(nil), w.Rows...) // prevent mutations via shared slice storage
	title := w.Name
	if strings.TrimSpace(title) == "" {
		title = "Собственный вес"
	}
	b := SavedReportBlock{Kind: "weight", Title: title, WeightCalc: &v}
	if id > 0 {
		for _, old := range p.SavedBlocks {
			if old.ID == id && old.Kind == "weight" {
				// A title changed in the construction editor takes precedence.
				// Empty names are allowed when the user explicitly chose them.
				if w.Name == old.WeightCalc.Name {
					b.Title = old.Title
				}
				break
			}
		}
		if p.updateBlock(id, b) {
			return id
		}
	}
	return p.addBlock(b)
}
func (p *Project) renameBlock(id int, title string) bool {
	for i := range p.SavedBlocks {
		if p.SavedBlocks[i].ID == id {
			p.SavedBlocks[i].Title = title
			if p.SavedBlocks[i].Kind == "weight" && p.SavedBlocks[i].WeightCalc != nil {
				p.SavedBlocks[i].WeightCalc.Name = title
			}
			return true
		}
	}
	return false
}
func (p *Project) ensureTempSelections(sections []TempSelection) {
	if !p.ReportManaged {
		return
	}
	// Snapshot each current crane set once. A stored group retains a stable ID.
	cranes := currentCraneSelections(sections)
	if len(cranes) > 0 {
		existing := 0
		for _, b := range p.SavedBlocks {
			if b.Kind == "crane_set" {
				existing = b.ID
				break
			}
		}
		// The active set is tracked by the Windows UI. On non-Windows the
		// first existing set serves as the default, without duplication.
		if existing == 0 {
			p.putCraneSet(0, cranes)
		}
	}
	count := map[string]int{}
	suppressed := map[string]bool{}
	for _, key := range p.SuppressedSelections {
		suppressed[key] = true
	}
	for _, b := range p.SavedBlocks {
		if b.Kind == "temporary" && b.Temporary != nil {
			count[tempKey(*b.Temporary)]++
		}
	}
	for _, s := range sections {
		if tempIsCrane(s) {
			continue
		}
		key := tempKey(s)
		if suppressed[key] {
			continue
		}
		if count[key] > 0 {
			count[key]--
			continue
		}
		v := cloneTempSelection(s)
		p.addBlock(SavedReportBlock{Kind: "temporary", Title: s.Title, Temporary: &v})
	}
}
func (p *Project) suggestedBlockName(kind string) string {
	base := map[string]string{"weight": "Собственный вес", "snow": "Снеговая нагрузка", "ice": "Гололёдная нагрузка", "temporary": "Временная нагрузка", "transport": "Транспортная нагрузка", "crane_set": "Мостовые краны", "crane_placement": "Расстановка кранов"}[kind]
	if base == "" {
		base = "Нагрузка"
	}
	count := 0
	for _, b := range p.SavedBlocks {
		match := b.Kind == kind
		if kind == "transport" {
			match = b.Kind == "temporary" && b.Temporary != nil && tempTransport(*b.Temporary)
		}
		if kind == "temporary" {
			match = b.Kind == "temporary" && b.Temporary != nil && !tempTransport(*b.Temporary) && !tempIsCrane(*b.Temporary)
		}
		if match {
			count++
		}
	}
	return fmt.Sprintf("%s %d", base, count+1)
}
func (p Project) suggestedCurrentName(kind string, current int) string {
	base := map[string]string{"weight": "Собственный вес", "snow": "Снеговая нагрузка", "ice": "Гололёдная нагрузка", "temporary": "Временная нагрузка", "transport": "Транспортная нагрузка", "crane_set": "Мостовые краны", "crane_placement": "Расстановка кранов"}[kind]
	if base == "" {
		base = "Нагрузка"
	}
	count := 0
	for _, b := range p.SavedBlocks {
		match := b.Kind == kind
		if kind == "transport" {
			match = b.Kind == "temporary" && b.Temporary != nil && tempTransport(*b.Temporary)
		}
		if kind == "temporary" {
			match = b.Kind == "temporary" && b.Temporary != nil && !tempTransport(*b.Temporary) && !tempIsCrane(*b.Temporary)
		}
		if match {
			count++
		}
	}
	if count == 0 {
		count = 1
	}
	return fmt.Sprintf("%s %d", base, count)
}
func tempKey(t TempSelection) string { return fmt.Sprint(t.Group, "|", t.Title, "|", t.Source) }
func (p Project) selectedReport() TempReport {
	r := TempReport{Version: 1}
	for _, b := range p.SavedBlocks {
		switch b.Kind {
		case "weight":
			if b.WeightCalc != nil {
				group := ProjectWeightBlock{Title: b.Title}
				for _, row := range b.WeightCalc.Rows {
					item := ProjectWeightLine{Name: row.Name, Norm: kgfM2ToKpa(row.NormKgf()), Gamma: row.Gamma, Design: kgfM2ToKpa(row.NormKgf() * row.Gamma)}
					group.Rows = append(group.Rows, item)
					r.WeightRows = append(r.WeightRows, ProjectWeightLine{Name: b.Title + " / " + row.Name, Norm: item.Norm, Gamma: item.Gamma, Design: item.Design})
				}
				if len(group.Rows) > 0 {
					r.WeightBlocks = append(r.WeightBlocks, group)
				}
			} else if b.Weight != nil {
				r.WeightRows = append(r.WeightRows, *b.Weight)
			}
		case "snow":
			if b.Snow != nil {
				r.SnowTitles = append(r.SnowTitles, b.Title)
				r.Snows = append(r.Snows, *b.Snow)
			}
		case "ice":
			if b.Ice != nil {
				r.IceTitles = append(r.IceTitles, b.Title)
				r.Ices = append(r.Ices, *b.Ice)
			}
		case "temporary":
			if b.Temporary != nil {
				v := cloneTempSelection(*b.Temporary)
				v.Title = b.Title
				r.Sections = append(r.Sections, v)
			}
		}
	}
	r.CraneGroups = p.craneGroups()
	return r
}

func (p *Project) unsuppressTemporary(key string) {
	filtered := p.SuppressedSelections[:0]
	for _, k := range p.SuppressedSelections {
		if k != key {
			filtered = append(filtered, k)
		}
	}
	p.SuppressedSelections = filtered
}
