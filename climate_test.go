package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pickCity(t *testing.T, name string) ClimateRecord {
	t.Helper()
	for _, v := range climateRecords() {
		if v.Place == name {
			return v
		}
	}
	t.Fatalf("city missing: %s", name)
	return ClimateRecord{}
}
func TestClimateCitiesAndRegions(t *testing.T) {
	if len(climateRecords()) < 150 {
		t.Fatalf("too few records: %d", len(climateRecords()))
	}
	if got := matchingRecords(climateRecords(), "казань"); len(got) != 1 || got[0].Place != "Казань" {
		t.Fatalf("city search: %#v", got)
	}
	if got := matchingRecords(climateRecords(), "краснодарский край"); len(got) < 3 {
		t.Fatalf("regional search: %#v", got)
	}
	if got := matchingRecords(climateRecords(), "темрюкский район"); len(got) != 0 {
		t.Fatalf("empty district must not appear: %#v", got)
	}
	if got := matchingRecords(climateRecords(), "орел"); len(got) < 1 {
		t.Fatalf("ё/e search")
	}
	if len(matchingRecords(climateRecords(), "несуществующее место")) != 0 {
		t.Fatal("false matches")
	}
}
func TestClimateMappings(t *testing.T) {
	if v, ok := snowForZone("IV"); !ok || v != 2 {
		t.Fatal("snow IV")
	}
	if v, ok := windForZone("VI"); !ok || v != .73 {
		t.Fatal("wind VI")
	}
	if v, ok := windForZone("Ia"); !ok || v != .17 {
		t.Fatal("wind Ia")
	}
	if v, ok := iceForZone("V"); !ok || v != 20 {
		t.Fatal("ice V")
	}
	if _, ok := iceForZone("VII"); ok {
		t.Fatal("wrong ice VII")
	}
}
func TestSitePropagationAndOverrides(t *testing.T) {
	p := initialProject()
	if p.WindOverride {
		t.Fatal("initial project should accept site wind")
	}
	s := climateFromRecord(pickCity(t, "Казань"))
	if s.SnowGround != 2.3 || s.SnowZone != "IV" || s.WindZone != "II" {
		t.Fatalf("bad Kazan: %#v", s)
	}
	p.SetSite(s)
	if p.Wind.W0 != .3 {
		t.Fatalf("wind from Kazan: %f", p.Wind.W0)
	}
	s = climateFromRecord(pickCity(t, "Краснодар"))
	p.SetSite(s)
	if p.Wind.W0 != .73 {
		t.Fatalf("wind from Krasnodar: %f", p.Wind.W0)
	}
	p.WindOverride = true
	p.Wind.W0 = .49
	p.SetSite(climateFromRecord(pickCity(t, "Москва")))
	if p.Wind.W0 != .23 || p.WindOverride {
		t.Fatal("selection of a new site must override local wind settings")
	}
	p.WindOverride = false
	p.SetSite(p.Site)
	if p.Wind.W0 != .23 {
		t.Fatalf("restore site wind: %f", p.Wind.W0)
	}
	if p.Site.SnowGround != 1.45 {
		t.Fatal("Moscow app K wrong")
	}
	p.Snow = ClimateOverride{Enabled: true, Zone: "V", Value: 2.8}
	if zone, sg, ov := p.SnowInputs(); zone != "V" || sg != 2.8 || !ov {
		t.Fatal("manual snow")
	}
	p.SetSite(climateFromRecord(pickCity(t, "Казань")))
	if _, sg, ov := p.SnowInputs(); sg != 2.3 || ov {
		t.Fatal("new site must replace manual snow override")
	}
	p.Snow = ClimateOverride{}
	if _, sg, ov := p.SnowInputs(); sg != 2.3 || ov {
		t.Fatal("snow restore")
	}
	if zone, value, ov := p.IceInputs(); zone != "" || value != 0 || ov {
		t.Fatal("unknown ice must be blank")
	}
	p.Ice = ClimateOverride{Enabled: true, Zone: "III", Value: 10}
	if zone, value, ov := p.IceInputs(); zone != "III" || value != 10 || !ov {
		t.Fatal("ice manual")
	}
}
func TestSaveNewClimateLegacy(t *testing.T) {
	p := initialProject()
	p.SetSite(climateFromRecord(pickCity(t, "Казань")))
	p.Snow = ClimateOverride{Enabled: true, Zone: "III", Value: 1.6}
	fn := filepath.Join(t.TempDir(), "new.json")
	if err := p.Save(fn); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadProject(fn)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Site.Place != "Казань" || loaded.WindOverride || loaded.Snow.Value != 1.6 {
		t.Fatalf("roundtrip: %#v", loaded)
	}
	legacy := `{"format_version":1,"layers":[{"name":"Стяжка","thickness_m":0.05,"unit_weight_kn_m3":20,"gamma_f":1.1}],"wind":{"w0_kpa":0.51,"k_z":1,"c":0.8,"gamma_f":1.4}}`
	if err = os.WriteFile(fn, []byte(legacy), 0644); err != nil {
		t.Fatal(err)
	}
	old, err := LoadProject(fn)
	if err != nil {
		t.Fatal(err)
	}
	if !old.WindOverride || old.Wind.W0 != .51 {
		t.Fatalf("legacy lost wind: %#v", old)
	}
}
func TestClimateExcelWordExport(t *testing.T) {
	p := initialProject()
	p.SetSite(climateFromRecord(pickCity(t, "Казань")))
	x := filepath.Join(t.TempDir(), "result.xlsx")
	w := filepath.Join(t.TempDir(), "result.docx")
	if err := ExportExcel(p, x); err != nil {
		t.Fatal(err)
	}
	if err := ExportWord(p, w); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ file, part, needle string }{
		{x, "xl/worksheets/sheet2.xml", "&#39;Местность&#39;!B7"},
		{x, "xl/worksheets/sheet3.xml", "Казань"},
		{x, "xl/workbook.xml", "Местность"},
		{w, "word/document.xml", "Казань"},
	} {
		z, err := zip.OpenReader(tc.file)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, f := range z.File {
			if f.Name == tc.part {
				r, err := f.Open()
				if err != nil {
					t.Fatal(err)
				}
				b, err := io.ReadAll(r)
				r.Close()
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(b), tc.needle) {
					found = true
				}
				break
			}
		}
		z.Close()
		if !found {
			t.Fatalf("missing %s in %s", tc.needle, tc.part)
		}
	}
}
