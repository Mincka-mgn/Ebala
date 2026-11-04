package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Source: SP 131.13330.2025, Tables 7.1 (January) and 5.1
// (mean wind over the period with mean daily temperature <=8°C).
// Database is supplied as a separate readable file and embedded to avoid
// dependency on the executable's working directory.
//
//go:embed Databases/SP131_2025_snow_climate.json
var snowClimateJSON string

const snowClimateSource = "СП 131.13330.2025, таблицы 7.1 (температура января) и 5.1 (ветер холодного периода)"

type SnowClimateRecord struct {
	Region   string
	Locality string
	January  float64
	Wind     float64
}

type snowClimateJSONRow struct {
	Region   string  `json:"region"`
	Locality string  `json:"locality"`
	January  float64 `json:"t_january_c"`
	Wind     float64 `json:"v_cold_period_m_s"`
}

func parseSnowClimateDB(data string) ([]SnowClimateRecord, error) {
	var arr []snowClimateJSONRow
	if err := json.Unmarshal([]byte(data), &arr); err != nil {
		return nil, err
	}
	var out []SnowClimateRecord
	seen := map[string]bool{}
	for _, row := range arr {
		if row.Region == "" || row.Locality == "" || row.Wind < 0 {
			return nil, fmt.Errorf("некорректная запись в справочнике СП 131")
		}
		k := normalizeSearch(row.Region) + "|" + normalizeSearch(row.Locality)
		if seen[k] {
			return nil, fmt.Errorf("повтор населённого пункта: %s", k)
		}
		seen[k] = true
		out = append(out, SnowClimateRecord{row.Region, row.Locality, row.January, row.Wind})
	}
	return out, nil
}
func loadSnowClimateDB() ([]SnowClimateRecord, error) {
	// Files alongside the executable can be updated without recompiling Ebala.
	exe, e := os.Executable()
	if e == nil {
		path := filepath.Join(filepath.Dir(exe), "Databases", "SP131_2025_snow_climate.json")
		if content, e := os.ReadFile(path); e == nil {
			if out, e := parseSnowClimateDB(string(content)); e == nil && len(out) >= 100 {
				return out, nil
			}
		}
	}
	return parseSnowClimateDB(snowClimateJSON)
}

var snowClimateCatalog, snowClimateError = loadSnowClimateDB()

func snowClimateFiltered(region, q string) []SnowClimateRecord {
	query := normalizeSearch(q)
	matches := []SnowClimateRecord{}
	for _, r := range snowClimateCatalog {
		if region != "" && !canonicalRegionMatch(region, r.Region) {
			continue
		}
		if query != "" && !strings.Contains(normalizeSearch(r.Locality+" "+r.Region), query) {
			continue
		}
		matches = append(matches, r)
	}
	sort.SliceStable(matches, func(i, j int) bool {
		regionI, regionJ := normalizeSearch(matches[i].Region), normalizeSearch(matches[j].Region)
		if regionI != regionJ {
			return regionI < regionJ
		}
		return normalizeSearch(matches[i].Locality) < normalizeSearch(matches[j].Locality)
	})
	return matches
}
func (s *Site) SetSnowClimate(r SnowClimateRecord) {
	s.ClimateJanuary = r.January
	s.ClimateWind = r.Wind
	s.ClimateJanuarySet = true
	s.ClimateWindSet = true
	s.ClimateReferencePlace = r.Locality
	s.ClimateReferenceRegion = r.Region
	s.ClimateOrigin = "sp131"
}
func (s *Site) SetSnowClimateManual(january, wind float64, hasJanuary, hasWind bool) {
	s.ClimateJanuary, s.ClimateWind = january, wind
	s.ClimateJanuarySet, s.ClimateWindSet = hasJanuary, hasWind
	s.ClimateReferencePlace = ""
	s.ClimateReferenceRegion = ""
	s.ClimateOrigin = "manual"
}
func snowClimateReference(s Site) string {
	if s.ClimateOrigin != "sp131" || s.ClimateReferencePlace == "" {
		return "Задано пользователем"
	}
	place := s.ClimateReferencePlace
	if s.ClimateReferenceRegion != "" {
		place += " (" + s.ClimateReferenceRegion + ")"
	}
	return snowClimateSource + "; принято по населённому пункту " + place
}
func (s *SnowCalc) InheritClimate(site Site) {
	if site.ClimateJanuarySet {
		s.JanuaryTemp = site.ClimateJanuary
		s.JanuarySet = true
	} else {
		s.JanuarySet = false
		s.JanuaryTemp = 0
	}
	if site.ClimateWindSet {
		s.WinterWind = site.ClimateWind
		s.WinterWindSet = true
	} else {
		s.WinterWindSet = false
		s.WinterWind = 0
	}
	s.ClimateOrigin = site.ClimateOrigin
	s.ClimateReferencePlace = site.ClimateReferencePlace
	s.ClimateReferenceRegion = site.ClimateReferenceRegion
}
