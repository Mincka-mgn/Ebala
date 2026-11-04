package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AppSettings struct {
	SaveMode    string `json:"save_mode"` // last or fixed
	FixedFolder string `json:"fixed_folder,omitempty"`
	LastFolder  string `json:"last_folder,omitempty"`
}

var appSettings = AppSettings{SaveMode: "last"}

func settingsPath() string {
	dir, e := os.UserConfigDir()
	if e != nil || dir == "" {
		dir = "."
	}
	return filepath.Join(dir, "Ebala", "settings.json")
}
func loadSettings() {
	data, e := os.ReadFile(settingsPath())
	if e != nil {
		return
	}
	var s AppSettings
	if json.Unmarshal(data, &s) == nil && (s.SaveMode == "last" || s.SaveMode == "fixed") {
		appSettings = s
	}
}
func saveSettings() error {
	path := settingsPath()
	if e := os.MkdirAll(filepath.Dir(path), 0755); e != nil {
		return e
	}
	data, e := json.MarshalIndent(appSettings, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(path, data, 0644)
}
func reportInitialDir() string {
	if appSettings.SaveMode == "fixed" && appSettings.FixedFolder != "" {
		return appSettings.FixedFolder
	}
	if appSettings.LastFolder != "" {
		return appSettings.LastFolder
	}
	p, e := os.UserHomeDir()
	if e != nil {
		return ""
	}
	doc := filepath.Join(p, "Documents")
	if info, e := os.Stat(doc); e == nil && info.IsDir() {
		return doc
	}
	return p
}
func reportDidSave(path string) {
	if path == "" {
		return
	}
	appSettings.LastFolder = filepath.Dir(path)
	_ = saveSettings()
}
