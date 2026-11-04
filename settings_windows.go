//go:build windows

package main

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	settingsModeLast  = 16001
	settingsModeFixed = 16002
	settingsFolder    = 16003
	settingsBrowse    = 16004
	settingsApply     = 16005
	settingsCancel    = 16006
)

var settingsDraftMode = "last"
var shellBrowse = syscall.NewLazyDLL("shell32.dll").NewProc("SHBrowseForFolderW")
var shellPath = syscall.NewLazyDLL("shell32.dll").NewProc("SHGetPathFromIDListW")
var coTaskFree = syscall.NewLazyDLL("ole32.dll").NewProc("CoTaskMemFree")

type settingsBrowseInfo struct {
	Owner, Root uintptr
	DisplayName *uint16
	Title       *uint16
	Flags       uint32
	Callback    uintptr
	Param       uintptr
	Image       int32
}

func browseFolder() string {
	buf := make([]uint16, 260)
	info := settingsBrowseInfo{Owner: hwnd, DisplayName: &buf[0], Title: ptr("Выберите папку для сохранения файлов"), Flags: 0x0040 | 0x0001}
	id, _, _ := shellBrowse.Call(uintptr(unsafe.Pointer(&info)))
	if id == 0 {
		return ""
	}
	defer coTaskFree.Call(id)
	out := make([]uint16, 1024)
	r, _, _ := shellPath.Call(id, uintptr(unsafe.Pointer(&out[0])))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(out)
}
func buildSettings() {
	buildingPage = "settings"
	navigation("Настройки", "")
	heading("Сохранение файлов", 28, 113, 1000, 36)
	button(settingsModeLast, "Последняя использованная папка", 28, 182, 460, 38)
	button(settingsModeFixed, "Всегда сохранять в указанную папку", 28, 229, 460, 38)
	label("Папка для сохранения", 28, 304, 540, 25)
	edit(settingsFolder, "", 28, 336, 850, 32)
	button(settingsBrowse, "Обзор…", 889, 335, 193, 34)
	button(settingsApply, "Сохранить настройки", 29, 438, 260, 42)
	button(settingsCancel, "Отмена", 302, 438, 180, 42)
}
func settingsEntered() {
	settingsDraftMode = appSettings.SaveMode
	if settingsDraftMode == "" {
		settingsDraftMode = "last"
	}
	set(settingsFolder, appSettings.FixedFolder)
	settingsUpdate()
}
func settingsUpdate() {
	if settingsDraftMode == "fixed" {
		set(settingsModeLast, "○ Последняя использованная папка")
		set(settingsModeFixed, "● Всегда сохранять в указанную папку")
	} else {
		set(settingsModeLast, "● Последняя использованная папка")
		set(settingsModeFixed, "○ Всегда сохранять в указанную папку")
	}
	pEnable.Call(controls[settingsFolder], boolToUintptr(settingsDraftMode == "fixed"))
	pEnable.Call(controls[settingsBrowse], boolToUintptr(settingsDraftMode == "fixed"))
}
func settingsAction(id int) bool {
	switch id {
	case settingsModeLast:
		settingsDraftMode = "last"
		settingsUpdate()
	case settingsModeFixed:
		settingsDraftMode = "fixed"
		settingsUpdate()
	case settingsBrowse:
		if p := browseFolder(); p != "" {
			set(settingsFolder, p)
		}
	case settingsCancel:
		switchPage("home")
	case settingsApply:
		v := get(settingsFolder)
		if settingsDraftMode == "fixed" {
			p, e := os.Stat(v)
			if e != nil || !p.IsDir() {
				message("Укажите существующую папку для сохранения файлов", true)
				return true
			}
		}
		appSettings.SaveMode = settingsDraftMode
		appSettings.FixedFolder = filepath.Clean(v)
		if v == "" {
			appSettings.FixedFolder = ""
		}
		if e := saveSettings(); e != nil {
			message("Не удалось сохранить настройки: "+e.Error(), true)
			return true
		}
		switchPage("home")
	default:
		return false
	}
	return true
}
