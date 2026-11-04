package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMapsRootCreatesThreeFolders(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Maps")
	if e := initMapsDirectory(root); e != nil {
		t.Fatal(e)
	}
	for _, sub := range []string{"Snow", "Wind", "Ice"} {
		if s, e := os.Stat(filepath.Join(root, sub)); e != nil || !s.IsDir() {
			t.Fatalf("missing %s: %v", sub, e)
		}
	}
}
func TestMapFilesSavedAndReloaded(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Maps")
	got, e := installLocalMap(root, 1, "2a", ".jpg", []byte("old image"))
	if e != nil {
		t.Fatal(e)
	}
	if want := filepath.Join(root, "Wind", "2a.jpg"); got != want {
		t.Fatalf("%s != %s", got, want)
	}
	if read, e := findLocalMap(root, 1, "2a"); e != nil || read != got {
		t.Fatalf("find map: %s %v", read, e)
	}
	time.Sleep(2 * time.Millisecond)
	newer, e := installLocalMap(root, 1, "2a", ".png", []byte("new image"))
	if e != nil {
		t.Fatal(e)
	}
	if read, e := findLocalMap(root, 1, "2a"); e != nil || read != newer {
		t.Fatalf("did not reread new map: %s %v", read, e)
	}
	if _, e := os.Stat(got); !os.IsNotExist(e) {
		t.Fatalf("obsolete extension was not removed: %v", e)
	}
	if data, e := os.ReadFile(newer); e != nil || string(data) != "new image" {
		t.Fatalf("replacement content: %s %v", data, e)
	}
}
func TestMissingMapAndInvalidPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Maps")
	if _, e := findLocalMap(root, 2, "3"); !os.IsNotExist(e) {
		t.Fatalf("want not-exist, got %v", e)
	}
	for _, key := range []string{"../secret", "1/2", ""} {
		if _, e := localMapPath(root, 1, key, ".jpg"); e == nil {
			t.Fatalf("allowed %q", key)
		}
	}
	if _, e := localMapPath(root, 3, "3", ".jpg"); e == nil {
		t.Fatal("bad family accepted")
	}
}
func TestMapRootSettingPersists(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	root := filepath.Join(t.TempDir(), "Отдельная папка с картами")
	if e := saveMapRoot(root); e != nil {
		t.Fatal(e)
	}
	chosen, e := configuredMapRoot()
	if e != nil || chosen != root {
		t.Fatalf("got %s err %v", chosen, e)
	}
	cfg, e := mapSettingsFile()
	if e != nil {
		t.Fatal(e)
	}
	text, e := os.ReadFile(cfg)
	if e != nil || !strings.Contains(string(text), "Отдельная") {
		t.Fatalf("settings are missing: %v", e)
	}
	def, e := resetMapRoot()
	if e != nil {
		t.Fatal(e)
	}
	chosen, e = configuredMapRoot()
	if e != nil || chosen != def {
		t.Fatalf("reset did not restore default: %s vs %s %v", chosen, def, e)
	}
}
