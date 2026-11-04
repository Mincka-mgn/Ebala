package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func snow048SphereConeSample() SnowCalc {
	s := defaultSnowCalc()
	s.Shape = "sphere_cone"
	s.Span = 18
	s.Width = 24
	s.RoofHeight = 8
	s.H = 6
	s.SkylightWidth = 6
	s.Angle = math.Atan(2.0/6.0) * 180 / math.Pi
	s.SlopeMode = "angle"
	return s
}
func Test048SphereConeGeometryAndZones(t *testing.T) {
	s := snow048SphereConeSample()
	geo, e := makeSphereConeGeometry(s)
	if e != nil {
		t.Fatal(e)
	}
	if math.Abs(geo.sideAngle(geo.ConeRadius)-s.Angle) > 1e-6 {
		t.Fatalf("wrong tangent angle: %v", geo)
	}
	if geo.EdgeHeight <= 0 {
		t.Fatalf("roof crosses foundation: %v", geo)
	}
	res, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(res.Cases) < 2 {
		t.Fatalf("expected both B.15a alternatives, got %d", len(res.Cases))
	}
	if len(res.Cases[1].Polar) < 300 {
		t.Fatalf("missing spatial distribution: %d", len(res.Cases[1].Polar))
	}
	zones := map[int]bool{}
	for _, p := range res.Cases[1].Polar {
		zones[p.Zone] = true
	}
	for i := 1; i <= 6; i++ {
		if !zones[i] {
			t.Fatalf("B.15a zone %d missing, saw %v", i, zones)
		}
	}
	first := res.Cases[0]
	if first.Intervals[0].Mu != 0 || first.Intervals[len(first.Intervals)-1].Mu != 0 {
		t.Fatalf("spherical edges with >60 deg must carry zero snow")
	}
	bad := s
	bad.H = 2
	if _, err := SnowCompute(bad); err != nil {
		t.Fatalf("conditional nonstandard geometry must not be blocked: %v", err)
	}
}
func Test048ApprovedSphereConePNGUnchanged(t *testing.T) {
	file := filepath.Join("Resources", "SnowRoofs", "snow_B15a_approved_geometry.png")
	b, e := os.ReadFile(file)
	if e != nil {
		t.Fatal(e)
	}
	h := sha256.Sum256(b)
	if hex.EncodeToString(h[:]) != "302ae61ec5dd87e9f639d6a67cef637ad3da16a66b9e1d8c87d5ff6046ac9343" {
		t.Fatal("approved image bytes changed")
	}
	got, e := snowRoofImage(snow048SphereConeSample())
	if e != nil {
		t.Fatal(e)
	}
	gh := sha256.Sum256(got)
	if gh != h {
		t.Fatal("the approved PNG was re-drawn instead of used verbatim")
	}
}
func Test048SphereConeWordExcel(t *testing.T) {
	s, e := SnowCompute(snow048SphereConeSample())
	if e != nil {
		t.Fatal(e)
	}
	dir := t.TempDir()
	for _, v := range []struct {
		ext string
		f   func(SnowCalc, string) error
	}{{"docx", ExportSnowWord}, {"xlsx", ExportSnowExcel}} {
		path := filepath.Join(dir, "sphere_cone."+v.ext)
		if err := v.f(s, path); err != nil {
			t.Fatalf("%s report: %v", v.ext, err)
		}
		z, e := zip.OpenReader(path)
		if e != nil {
			t.Fatalf("%s invalid zip: %v", v.ext, e)
		}
		foundNewFigure := false
		foundB15a := false
		foundPolar := false
		for _, f := range z.File {
			if strings.HasPrefix(f.Name, "word/media/") && f.UncompressedSize64 > 500000 {
				foundNewFigure = true
			}
			if strings.HasSuffix(f.Name, ".xml") {
				r, e := f.Open()
				if e != nil {
					t.Fatal(e)
				}
				data, e := io.ReadAll(r)
				r.Close()
				if e != nil {
					t.Fatal(e)
				}
				if strings.Contains(string(data), "Б.15а") || strings.Contains(string(data), "Сфера с конусом") {
					foundB15a = true
				}
				if strings.Contains(string(data), "пространственные коэффициенты") || strings.Contains(string(data), "Сетка Б.15а") || strings.Contains(string(data), "Пространственный вариант по β") {
					foundPolar = true
				}
			}
		}
		z.Close()
		if v.ext == "docx" && !foundNewFigure {
			t.Fatal("approved PNG missing in DOCX")
		}
		if !foundB15a {
			t.Fatalf("new roof absent from %s", v.ext)
		}
		if v.ext == "xlsx" && !foundPolar {
			t.Fatal("B.15a polar values absent from XLSX")
		}
	}
}
func Test048DomeOnionCap(t *testing.T) {
	s := sampleSnow("dome")
	s.Span = 18
	s.Rise = 14
	out, e := SnowCompute(s)
	if e != nil {
		t.Fatalf("f > D/2 must be accepted: %v", e)
	}
	if len(out.Cases) == 0 || len(out.Cases[0].Intervals) == 0 {
		t.Fatal("missing case")
	}
	first := out.Cases[0].Intervals
	if first[len(first)-1].X2 <= s.Span {
		t.Fatalf("overhanging zero-loaded annulus omitted: %v", first[len(first)-1])
	}
	if first[0].Mu > 0 || first[len(first)-1].Mu > 0 {
		t.Fatal("slope>60 zones must have zero snow")
	}
}
