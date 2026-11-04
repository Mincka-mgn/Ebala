package main

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test043SiteOverridesAllAndMountainUnrestricted(t *testing.T) {
	p := initialProject()
	p.SetSite(Site{Region: "Краснодарский край", Place: "А", SnowZone: "III", SnowGround: 1.5, IceZone: "IV", WindZone: "V"})
	p.Snow = ClimateOverride{Enabled: true, Zone: "VIII", Value: 4}
	p.Ice = ClimateOverride{Enabled: true, Zone: "IV", Value: 15}
	p.WindOverride = true
	p.Wind.W0 = 2
	snow := defaultSnowCalc()
	snow.Region = "VIII"
	snow.Ground = 4
	snow.Enabled = true
	p.addBlock(SavedReportBlock{Kind: "snow", Title: "Покрытие", Snow: &snow})
	ice := defaultIceLoad()
	ice.Enabled = true
	p.addBlock(SavedReportBlock{Kind: "ice", Title: "Трос", Ice: &ice})
	p.SetSite(Site{Region: "Владимирская область", Place: "Площадка", SnowZone: "II", SnowGround: 1, IceZone: "II", WindZone: "I", MountainID: "dagestan", ElevationM: 600})
	if p.Snow.Enabled || p.Ice.Enabled || p.WindOverride {
		t.Fatal("site must replace local climate overrides")
	}
	if p.SavedBlocks[0].Snow.Region != "VIII" || p.SavedBlocks[1].Ice.Region != "IV" {
		t.Fatal("0.44: site change must not modify historical calculations")
	}
	if p.SavedBlocks[0].Title != "Покрытие" {
		t.Fatal("name lost")
	}
	if _, _, _, applied, err := p.Site.ElevationDetails(); err != nil || !applied {
		t.Fatalf("cross-region manual selection must work: %v", err)
	}
}
func Test043IndependentNamesAndEmptyName(t *testing.T) {
	p := initialProject()
	id := p.addBlock(SavedReportBlock{Kind: "ice", Title: "Гололёдная нагрузка"})
	if got := p.suggestedCurrentName("ice", id); got != "Гололёдная нагрузка 1" {
		t.Fatal(got)
	}
	p.renameBlock(id, "")
	if p.SavedBlocks[0].Title != "" {
		t.Fatal("empty user name not preserved")
	}
	p.addBlock(SavedReportBlock{Kind: "ice", Title: "Гололёдная нагрузка"})
	if got := p.suggestedCurrentName("ice", 2); got != "Гололёдная нагрузка 2" {
		t.Fatal(got)
	}
	p.addBlock(SavedReportBlock{Kind: "snow", Title: "Снеговая нагрузка"})
	if got := p.suggestedCurrentName("snow", 3); got != "Снеговая нагрузка 1" {
		t.Fatal(got)
	}
}
func Test043AutoTemporaryTransportAndCraneSet(t *testing.T) {
	p := initialProject()
	a := TempSelection{Group: "СП 20 · временные", Title: "Офис", Source: "manual", Params: map[string]interface{}{"q_kPa": 4.0}}
	b := TempSelection{Group: "СП 35 · транспорт", Title: "АК", Source: "manual", Params: map[string]interface{}{}}
	crane := TempSelection{Group: "Краны · ручной ввод", Title: "К1", Source: "manual", Wheels: []TempWheel{{XMM: 0, PKN: 100, XKnown: true}}}
	p.initReportBlocks([]TempSelection{a, b, crane})
	n := len(p.SavedBlocks)
	p.ensureTempSelections([]TempSelection{a, b, crane})
	if len(p.SavedBlocks) != n {
		t.Fatalf("duplicate blocks %d->%d", n, len(p.SavedBlocks))
	}
	r := p.selectedReport()
	if len(r.Sections) != 2 || len(r.CraneGroups) != 1 || len(r.CraneGroups[0].Cranes) != 1 {
		t.Fatalf("missing automatic report sections %+v", r)
	}
}
func Test043PlacementGroupPersistAndOutput(t *testing.T) {
	p := initialProject()
	cranes := []TempSelection{{Group: "Краны · ручной ввод", Title: "Кран Т", Source: "Задано пользователем", Wheels: []TempWheel{{XMM: 0, PKN: 100, XKnown: true}}}}
	setID := p.putCraneSet(0, cranes)
	for _, span := range []float64{6000, 12000} {
		result, err := CalculateCranePlacements([]float64{span}, []CraneTrain{{Name: "Кран Т", WheelXMM: []float64{0}, ForcesKN: []float64{100}}})
		if err != nil {
			t.Fatal(err)
		}
		id := p.putCranePlacement(0, setID, decimalInput(span/1000, 0), result)
		if !p.renameBlock(id, "Балка "+decimalInput(span/1000, 0)+" м") {
			t.Fatal("rename")
		}
	}
	g := p.craneGroups()
	if len(g) != 1 || len(g[0].Placements) != 2 || g[0].Placements[0].Result.SpansMM[0] != 6000 || g[0].Placements[1].Result.SpansMM[0] != 12000 {
		t.Fatalf("wrong parent/child %+v", g)
	}
	dir := t.TempDir()
	projectFile := filepath.Join(dir, "project.json")
	if err := p.Save(projectFile); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadProject(projectFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.craneGroups()) != 1 || len(loaded.craneGroups()[0].Placements) != 2 {
		t.Fatal("lost crane placements in project")
	}
	r := loaded.selectedReport()
	for _, ext := range []string{"docx", "xlsx"} {
		path := filepath.Join(dir, "report."+ext)
		if ext == "docx" {
			err = exportTempWord(r, path)
		} else {
			err = exportTempExcel(r, path)
		}
		if err != nil {
			t.Fatal(err)
		}
		z, e := zip.OpenReader(path)
		if e != nil {
			t.Fatal(e)
		}
		countPNG := 0
		output := ""
		for _, file := range z.File {
			if strings.HasSuffix(file.Name, ".png") {
				countPNG++
			}
			if strings.HasSuffix(file.Name, "document.xml") || strings.HasSuffix(file.Name, "sheet1.xml") {
				reader, e := file.Open()
				if e != nil {
					t.Fatal(e)
				}
				bytes, e := io.ReadAll(reader)
				reader.Close()
				if e != nil {
					t.Fatal(e)
				}
				if e = xml.Unmarshal(bytes, new(interface{})); e != nil {
					t.Fatalf("bad XML: %v", e)
				}
				output += string(bytes)
			}
		}
		z.Close()
		// The Word contains physical placement drawings. The XLSX is
		// intentionally formula-driven and does not duplicate static PNG
		// charts that become stale when the engineer edits source inputs.
		if ext == "docx" && countPNG < 2 {
			t.Fatalf("%s missing placement drawings (%d)", ext, countPNG)
		}
		if !strings.Contains(output, "Балка 6 м") || !strings.Contains(output, "Балка 12 м") {
			t.Fatalf("%s missing child names", ext)
		}
	}
	if err := os.Remove(projectFile); err != nil {
		t.Fatal(err)
	}
	p.removeBlock(setID)
	if len(p.craneGroups()) != 0 || len(p.SavedBlocks) != 0 {
		t.Fatalf("orphan placement found %+v", p.SavedBlocks)
	}
}
