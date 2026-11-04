package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// A new numbered name is used only when Windows reports a sharing/lock violation.
// Ordinary existing files are overwritten by the chosen exporter.
func fileInUse(err error) bool {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno == 32 || errno == 33 // ERROR_SHARING_VIOLATION, ERROR_LOCK_VIOLATION
	}
	return false
}
func numberedFileName(path string, n int) string {
	ext := filepath.Ext(path)
	return fmt.Sprintf("%s (%d)%s", strings.TrimSuffix(path, ext), n, ext)
}
func writeWithLockFallback(path string, write func(string) error) (string, error) {
	if path == "" {
		return "", fmt.Errorf("не выбрано имя файла")
	}
	if e := write(path); e == nil {
		return path, nil
	} else if !fileInUse(e) {
		return "", e
	}
	for i := 1; i <= 9999; i++ {
		next := numberedFileName(path, i)
		if _, e := os.Stat(next); e == nil {
			continue
		} else if !os.IsNotExist(e) {
			return "", e
		}
		if e := write(next); e == nil {
			return next, nil
		} else if !fileInUse(e) {
			return "", e
		}
	}
	return "", fmt.Errorf("не удалось найти свободное имя для %s", path)
}
