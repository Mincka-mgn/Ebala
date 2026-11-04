package main

import (
	"os"
	"regexp"
	"testing"
)

func TestSnowClimatePickerUsesNativeListViewRendering(t *testing.T) {
	b, err := os.ReadFile("snow_climate_picker_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	// Rows are drawn by the native control. LVS_OWNERDRAWFIXED requires a
	// WM_DRAWITEM handler, absent in the popup window's window procedure.
	re := regexp.MustCompile(`(?m)climatePickerControl\("SysListView32",[^\n]*\)`)
	create := re.FindString(string(b))
	if create == "" {
		t.Fatal("climate picker SysListView32 was not found")
	}
	if regexp.MustCompile(`\|\s*0x0400\s*,`).MatchString(create) || regexp.MustCompile(`LVS_OWNERDRAWFIXED`).MatchString(create) {
		t.Fatal("climate picker must not set LVS_OWNERDRAWFIXED")
	}
}
