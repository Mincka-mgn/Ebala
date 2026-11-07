package main

import (
	"archive/zip"
	"bytes"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func wind063Variant(t *testing.T, height float64, id string) WindVariant {
	t.Helper()
	windTestData(t)
	in := testRoofCaseBase(20)
	in.RoofRise = height
	c, e := CalculateWindVariants(in)
	if e != nil {
		t.Fatal(e)
	}
	for _, v := range c.Variants {
		if v.ID == id {
			return v
		}
	}
	t.Fatal("direction absent: " + id)
	return WindVariant{}
}

func TestWind063WindwardRoofZonesBelowRidgeAcrossWind(t *testing.T) {
	v := wind063Variant(t, 2.18, "CROSS")
	for _, z := range v.Zones {
		if z.Surface != "roof" {
			continue
		}
		if z.Alternative != "1" {
			continue
		}
		for _, r := range z.Rectangles {
			width, extent := r.Y1-r.Y0, r.X1-r.X0
			if width <= 0 || extent <= 0 {
				t.Fatalf("bad normative patch: %s %+v", z.Code, r)
			}
		}
	}
	boxes, xc, yc := windDrawingRects(v, true, "1")
	if len(xc) < 3 || len(yc) < 3 {
		t.Fatalf("missing zone dimensions: %v %v", xc, yc)
	}
	low := 0
	for _, r := range boxes {
		if r.code == "F" || r.code == "G" {
			if r.y0 < v.AlongM/2 {
				t.Fatalf("windward %s wrongly ABOVE ridge: %v", r.code, r)
			}
			low++
		}
	}
	if low != 3 {
		t.Fatalf("expected two F and one G patch, got %d", low)
	}
	if math.Abs(xc[len(xc)-1]-v.AcrossM) > 1e-10 || math.Abs(yc[len(yc)-1]-v.AlongM) > 1e-10 {
		t.Fatal("wrong plan axes")
	}
}
func TestWind063WallSegmentsRemainLinesAndExcludeDE(t *testing.T) {
	for _, id := range []string{"RIDGE", "CROSS"} {
		v := wind063Variant(t, 2.18, id)
		boxes, _, _ := windDrawingRects(v, false, "")
		if len(boxes) == 0 {
			t.Fatal(id + " has no side-wall segments")
		}
		for _, b := range boxes {
			if b.code == "D" || b.code == "E" {
				t.Fatalf("end face on side-wall diagram %s: %+v", id, b)
			}
		}
		if id == "CROSS" {
			for _, b := range boxes {
				if b.code == "A" && b.y1 > v.AlongM/2+1e-9 {
					continue
				}
				if b.code == "A" {
					t.Fatalf("crosswind side-wall A must be at windward lower edge: %+v", b)
				}
			}
		}
	}
}
func TestWind063LongZoneNoBreakAndAllDimensions(t *testing.T) {
	axis := windAxis([]float64{0, 1.2, 6, 30, 3000}, 100, 900)
	for _, p := range []float64{0, 1.2, 6, 30, 3000} {
		want := 100 + int(math.Round(900*p/3000))
		if axis.px(p) != want {
			t.Fatalf("nonlinear drawing p=%v got=%d want=%d", p, axis.px(p), want)
		}
	}
	v := wind063Variant(t, 2.18, "RIDGE")
	for _, z := range v.Zones {
		if z.Surface == "roof" {
			desc := windZoneLengthsText(&v, &z)
			if !strings.Contains(desc, "ширина ") || !strings.Contains(desc, "протяжённость ") {
				t.Fatalf("roof dimensions missing: %s", desc)
			}
			if !strings.Contains(windWordZoneTitle(z), "ширина ") {
				t.Fatal("Word title missing width")
			}
			break
		}
	}
}
func TestWind063DocxDiagramsNoBreakBothAxes(t *testing.T) {
	windTestData(t)
	c, e := CalculateWindVariants(testRoofCaseBase(20))
	if e != nil {
		t.Fatal(e)
	}
	out := filepath.Join(t.TempDir(), "Ветровая нагрузка.docx")
	if e = exportTempWord(TempReport{Version: 1, Winds: []WindCalculation{c}}, out); e != nil {
		t.Fatal(e)
	}
	zr, e := zip.OpenReader(out)
	if e != nil {
		t.Fatal(e)
	}
	defer zr.Close()
	count := 0
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			r, _ := f.Open()
			raw, _ := io.ReadAll(r)
			r.Close()
			txt := string(raw)
			for _, q := range []string{"План здания. Ветер вдоль конька", "План здания. Ветер поперёк конька", "ширина", "протяжённость"} {
				if !strings.Contains(txt, q) {
					t.Errorf("missing Word text %q", q)
				}
			}
			if strings.Contains(txt, "Условный разрыв") {
				t.Error("old caption in Word")
			}
		}
		if strings.HasPrefix(f.Name, "word/media/wind_") {
			r, _ := f.Open()
			raw, _ := io.ReadAll(r)
			r.Close()
			pic, e := png.Decode(bytes.NewReader(raw))
			if e != nil {
				t.Fatal(e)
			}
			if pic.Bounds().Dx() != 1400 || pic.Bounds().Dy() != 470 {
				t.Fatalf("wrong figure size %s", f.Name)
			}
			count++
		}
	}
	if count != 4 {
		t.Fatalf("expected four Word diagrams, got %d", count)
	}
	_ = os.ErrNotExist
}
func TestWind063NoRidgeWhenRoofRiseZero(t *testing.T) {
	v := wind063Variant(t, 0, "RIDGE")
	for _, z := range v.Zones {
		if z.Surface == "roof" {
			t.Fatal("unwanted roof at h=0")
		}
	}
	raw, e := windReportDiagram(v, false, "")
	if e != nil {
		t.Fatal(e)
	}
	img, e := png.Decode(bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	// Middle of the plan has no zone-boundary line when the ridge is absent.
	r, g, b, _ := img.At(195+945/2, 157+205/2).RGBA()
	if r < 50000 || g < 50000 || b < 50000 {
		t.Errorf("ridge shown for h=0")
	}
}
