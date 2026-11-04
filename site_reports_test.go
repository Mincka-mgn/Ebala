package main

import (
	"archive/zip"
	"encoding/xml"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMountainKrasnodar(t *testing.T) {
	s := climateFromRecord(pickCity(t, "Краснодар"))
	if s.EffectiveSnow() != 1.1 {
		t.Fatal("ordinary Krasnodar altered")
	}
	s.MountainID = "krasnodar_other"
	s.ElevationM = 800
	base, v, m, on, e := s.ElevationDetails()
	if e != nil || !on || m.Zone != "II" || base != 1 || math.Abs(v-2.5) > 1e-10 {
		t.Fatalf("incorrect elevation: %.3f %.3f %#v %v", base, v, m, e)
	}
	if s.EffectiveSnow() != 2.5 {
		t.Fatal("effective snow")
	}
	s.MountainID = "krasnodar_adler"
	s.ElevationM = 600
	base, v, m, on, e = s.ElevationDetails()
	if e != nil || !on || m.Zone != "VII" || base != 3.5 || math.Abs(v-4.2525) > 1e-10 {
		t.Fatalf("Adler: %.4f %.4f %v", base, v, e)
	}
	s.MountainID = ""
	if s.ElevationSummary() != "" {
		t.Fatal("non-mountain should be silent")
	}
}
func TestMountainUdokan(t *testing.T) {
	s := Site{Region: "Забайкальский край", SnowZone: "II", SnowGround: 1, MountainID: "zab_udokan", ElevationM: 800}
	if s.EffectiveSnow() != 1 {
		t.Fatalf("must use h0=1000: %.2f", s.EffectiveSnow())
	}
	s.ElevationM = 1500
	if math.Abs(s.EffectiveSnow()-1.9) > 1e-10 {
		t.Fatalf("Udokan 1500: %.5f", s.EffectiveSnow())
	}
}
func TestAreaNotAssignedToEntireRegion(t *testing.T) {
	for _, city := range []string{"Краснодар", "Казань", "Норильск"} {
		s := climateFromRecord(pickCity(t, city))
		if s.MountainID != "" {
			t.Fatalf("automatic mountain assigned for %s", city)
		}
	}
	s := Site{Region: "Москва", SnowGround: 2, MountainID: "krasnodar_other", ElevationM: 600}
	if _, value, _, applied, e := s.ElevationDetails(); e != nil || !applied || value <= 1 {
		t.Fatalf("the engineer must be able to select any mountain area manually: value %v, err %v", value, e)
	}
}
func TestSiteReportsSeparate(t *testing.T) {
	for _, s := range []Site{climateFromRecord(pickCity(t, "Казань")), {Region: "Краснодарский край", Place: "Горная площадка", SnowZone: "II", SnowGround: 1, WindZone: "VI", MountainID: "krasnodar_other", ElevationM: 800}} {
		for _, ext := range []string{"xlsx", "docx"} {
			dst := filepath.Join(t.TempDir(), "site."+ext)
			var e error
			if ext == "xlsx" {
				e = ExportSiteExcel(s, dst)
			} else {
				e = ExportSiteWord(s, dst)
			}
			if e != nil {
				t.Fatal(e)
			}
			z, e := zip.OpenReader(dst)
			if e != nil {
				t.Fatal(e)
			}
			for _, f := range z.File {
				if strings.HasSuffix(f.Name, ".xml") {
					c, e := f.Open()
					if e != nil {
						t.Fatal(e)
					}
					dec := xml.NewDecoder(c)
					for {
						if _, e = dec.Token(); e != nil {
							break
						}
					}
					c.Close()
					if e != nil && e.Error() != "EOF" {
						t.Fatalf("bad %s xml: %v", f.Name, e)
					}
				}
			}
			z.Close()
			data, _ := os.ReadFile(dst)
			if len(data) < 1000 {
				t.Fatal("too small")
			}
			if ext == "xlsx" {
				z, _ := zip.OpenReader(dst)
				for _, f := range z.File {
					if f.Name == "xl/worksheets/sheet1.xml" {
						c, _ := f.Open()
						raw, _ := os.ReadFile(dst)
						_ = raw
						buf := make([]byte, f.UncompressedSize64)
						n, _ := c.Read(buf)
						_ = n
						c.Close()
					}
				}
				z.Close()
			}
		}
	}
}

func TestDecimalNoTrailingZeroLoss(t *testing.T) {
	if decimalInput(500, 0) != "500" {
		t.Fatalf("lost zeros: %q", decimalInput(500, 0))
	}
	if decimalInput(1000, 0) != "1000" {
		t.Fatalf("lost zeros: %q", decimalInput(1000, 0))
	}
	if decimalInput(2.500, 3) != "2,5" {
		t.Fatalf("bad decimal trimming: %q", decimalInput(2.500, 3))
	}
}
