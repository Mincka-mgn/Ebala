package main

import (
	"os"
	"path/filepath"
	"strings"
)

// Editable auxiliary catalog from EVRAZ Steel Box. It is NOT the normative
// zoning map. Exact settlements only: do not inherit values from a region.
func climateCatalogPath() (string, error) {
	// Allows integration tests to exercise the very same external catalog as the app.
	if override := strings.TrimSpace(os.Getenv("EBALA_CLIMATE_CATALOG_PATH")); override != "" {
		return override, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "Maps", "Climate", "settlements.tsv"), nil
}

func parseExternalClimateCatalog(input string) []ClimateRecord {
	var rows []ClimateRecord
	seen := make(map[string]bool)
	for _, raw := range strings.Split(strings.TrimPrefix(input, "\ufeff"), "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		cols := strings.Split(raw, "|")
		if len(cols) != 4 {
			continue
		}
		region := strings.TrimSpace(cols[0])
		city := strings.TrimSpace(cols[1])
		snow := romanZone(cols[2])
		wind := romanZone(cols[3])
		if city == "" || region == "" || (snow == "" && wind == "") {
			continue
		}
		if c := canonicalSeismicRegion(region); c != "" {
			region = c
		}
		key := normalizeSearch(region) + "|" + normalizeSearch(city)
		if seen[key] {
			continue
		}
		seen[key] = true
		rows = append(rows, ClimateRecord{Region: region, Place: city, SnowZone: snow, WindZone: wind})
	}
	return rows
}

func loadExternalClimateCatalog() []ClimateRecord {
	p, e := climateCatalogPath()
	if e != nil {
		return nil
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return nil
	}
	rows := parseExternalClimateCatalog(string(b))
	if len(rows) < 30 {
		return nil
	} // fail safely on an accidental incomplete/invalid file
	return rows
}
