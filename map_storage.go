package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var mapFolders = []string{"Snow", "Wind", "Ice"}
var mapExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".bmp"}

// A manually copied map can use either Latin transliteration (2g.jpg) or its
// Cyrillic name from SP 20 (2г.jpg). The Import action creates Latin filenames.
var localizedMapNames = map[string]string{
	"1a": "1а", "1b": "1б", "1v": "1в", "2a": "2а", "2b": "2б", "2v": "2в", "2g": "2г",
	"2d": "2д", "2e": "2е", "2zh": "2ж", "3a": "3а", "3b": "3б", "3v": "3в",
	"3g": "3г", "3d": "3д", "3e": "3е", "3zh": "3ж",
}

// Next to the executable by default. A user-selected absolute path can override it.
// Settings are kept outside the executable folder so a read-only installation works.
func defaultMapRoot() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "Maps"), nil
}
func mapSettingsFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Poverka_Nagruzki", "maps_location.txt"), nil
}
func configuredMapRoot() (string, error) {
	cfg, err := mapSettingsFile()
	if err == nil {
		raw, e := os.ReadFile(cfg)
		if e == nil {
			p := strings.TrimSpace(strings.TrimPrefix(string(raw), "\ufeff"))
			if p != "" && filepath.IsAbs(p) {
				return filepath.Clean(p), nil
			}
		}
	}
	return defaultMapRoot()
}
func initMapsDirectory(root string) error {
	if !filepath.IsAbs(root) {
		return fmt.Errorf("путь к картам должен быть абсолютным: %s", root)
	}
	for _, folder := range mapFolders {
		if err := os.MkdirAll(filepath.Join(root, folder), 0755); err != nil {
			return err
		}
	}
	return nil
}
func saveMapRoot(root string) error {
	p, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	p = filepath.Clean(p)
	if err = initMapsDirectory(p); err != nil {
		return err
	}
	cfg, err := mapSettingsFile()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(cfg), 0755); err != nil {
		return err
	}
	return os.WriteFile(cfg, []byte(p+"\n"), 0644)
}
func resetMapRoot() (string, error) {
	p, err := defaultMapRoot()
	if err != nil {
		return "", err
	}
	if err = initMapsDirectory(p); err != nil {
		return "", err
	}
	cfg, err := mapSettingsFile()
	if err == nil {
		if e := os.Remove(cfg); e != nil && !os.IsNotExist(e) {
			return "", e
		}
	}
	return p, nil
}
func localMapPath(root string, family int, key, extension string) (string, error) {
	if family < 0 || family >= len(mapFolders) {
		return "", errors.New("неверный тип карты")
	}
	if key == "" || strings.ContainsAny(key, "/\\.") {
		return "", errors.New("неверный номер карты")
	}
	if !strings.HasPrefix(extension, ".") {
		return "", errors.New("неверное расширение")
	}
	return filepath.Join(root, mapFolders[family], key+extension), nil
}

// Read every visit, so replacing local files is reflected immediately on the next opening.
// If multiple variants exist, newest wins. This also allows a user to drop in a newer PNG
// alongside an older downloaded JPEG without using a special import workflow.
func findLocalMap(root string, family int, key string) (string, error) {
	type hit struct {
		path string
		time int64
	}
	var hits []hit
	names := []string{key}
	if localized := localizedMapNames[key]; localized != "" {
		names = append(names, localized)
	}
	for _, name := range names {
		for _, ext := range mapExtensions {
			p, err := localMapPath(root, family, name, ext)
			if err != nil {
				return "", err
			}
			st, e := os.Stat(p)
			if e == nil && st.Mode().IsRegular() && st.Size() > 0 {
				hits = append(hits, hit{p, st.ModTime().UnixNano()})
			}
		}
	}
	if len(hits) == 0 {
		return "", os.ErrNotExist
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].time > hits[j].time })
	return hits[0].path, nil
}
func installLocalMap(root string, family int, key, ext string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("файл пуст")
	}
	p, err := localMapPath(root, family, key, ext)
	if err != nil {
		return "", err
	}
	if err = initMapsDirectory(root); err != nil {
		return "", err
	}
	// Never alter other maps. A replacement of this map is atomic on the same volume.
	temp, err := os.CreateTemp(filepath.Dir(p), ".map-*.tmp")
	if err != nil {
		return "", err
	}
	tmp := temp.Name()
	defer os.Remove(tmp)
	if _, err = io.Copy(temp, bytes.NewReader(data)); err != nil {
		temp.Close()
		return "", err
	}
	if err = temp.Close(); err != nil {
		return "", err
	}
	// Windows os.Rename does not replace an existing destination atomically.
	// Safely remove only the exact destination chosen by the operator.
	if err = os.Remove(p); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err = os.Rename(tmp, p); err != nil {
		return "", err
	}
	// Remove other extensions for this ONE key after successfully installing its replacement.
	for _, other := range mapExtensions {
		if other == ext {
			continue
		}
		op, _ := localMapPath(root, family, key, other)
		os.Remove(op)
	}
	return p, nil
}
