package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSampleReportSeismic(t *testing.T) {
	if os.Getenv("POVERKA_SAMPLE") != "1" {
		return
	}
	s := Site{Region: "Республика Адыгея", District: "Майкопский район", Place: "Условный хутор", SeismicMap: "А", SeismicRefPlace: "Майкоп", SeismicRefRegion: "Республика Адыгея", SeismicPoints: "7", SnowZone: "II", SnowGround: 1.0}
	if e := ExportSiteWord(s, filepath.Join("/mnt/data", "Поверка_0_12_пример_сейсмика.docx")); e != nil {
		t.Fatal(e)
	}
	if e := ExportSiteExcel(s, filepath.Join("/mnt/data", "Поверка_0_12_пример_сейсмика.xlsx")); e != nil {
		t.Fatal(e)
	}
}
