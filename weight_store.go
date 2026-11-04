package main

import (
	"errors"
	"os"
	"path/filepath"
)

// The executable and its portable catalog reside in one folder.
func storageRoot() string {
	if value := os.Getenv("EBALA_HOME"); value != "" {
		return value
	}
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}
func writeAtomic(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".weight-catalog-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(content); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	backup := path + ".bak"
	os.Remove(backup)
	if err = os.Rename(path, backup); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err = os.Rename(name, path); err != nil {
		os.Rename(backup, path)
		return err
	}
	os.Remove(backup)
	return nil
}
