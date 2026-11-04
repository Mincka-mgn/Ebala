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

func sample049() SnowCalc {
	s := defaultSnowCalc()
	s.Shape = "sphere_cone"
	s.Span = 18
	s.RoofHeight = 12
	s.H = 6
	s.SkylightWidth = 6
	s.Angle = 18
	s.BetaStep = 15
	s.Width = 18
	return s
}
func Test049GeometryUserDimensions(t *testing.T) {
	s := sample049()
	g, e := makeSphereConeGeometry(s)
	if e != nil {
		t.Fatal(e)
	}
	if math.Abs(g.EdgeHeight-(s.RoofHeight-s.H)) > 1e-7 {
		t.Fatalf("f is no longer roof rise: %v", g)
	}
	if math.Abs(g.sideAngle(s.SkylightWidth)-s.Angle) > 1e-7 {
		t.Fatalf("wrong cone/sphere tangent: %v", g)
	}
	if math.Abs(g.heightAtRadius(s.Span/2)-(s.RoofHeight-s.H)) > 1e-6 {
		t.Fatalf("sphere does not reach wall: %v", g)
	}
	angle := sphereConeTangentAngle(s)
	f := sphereConeRiseForAngle1(s, angle)
	if math.Abs(f-s.H) > 1e-9 {
		t.Fatalf("alpha1 edit would not reconstruct f: %v vs %v", f, s.H)
	}
	// The abutment can be located underground; this is not an input error.
	s.RoofHeight = 5
	s.H = 8
	_, d, changed := sphereConeExpandDiameter(&s)
	if changed && d <= 0 {
		t.Fatal("wrong diameter correction")
	}
	gg, e := makeSphereConeGeometry(s)
	if e != nil {
		t.Fatal(e)
	}
	if gg.EdgeHeight >= 0 {
		t.Fatalf("underground wall was incorrectly clamped: %g", gg.EdgeHeight)
	}
	if mu, _ := gg.spatialMu(gg.D/2, 0); mu != 0 {
		t.Fatalf("snow on buried edge: %g", mu)
	}
}
func Test049AutoDiameterAndBetaStep(t *testing.T) {
	s := sample049()
	s.H = 12
	s.RoofHeight = 8
	old, neu, corrected := sphereConeExpandDiameter(&s)
	if !corrected || neu <= old {
		t.Fatalf("no automatic D adjustment: old=%g new=%g corrected=%v", old, neu, corrected)
	}
	if a1 := sphereConeTangentAngle(s); a1 > 90.00001 {
		t.Fatalf("alpha1>90 after D correction: %g", a1)
	}
	for _, step := range []float64{5, 7, 10, 15, 30} {
		b := sphereConeBetas(step)
		if b[0] != 0 || b[len(b)-1] != 180 {
			t.Fatalf("missing beta end values for step %g", step)
		}
		x := sample049()
		x.BetaStep = step
		v, e := SnowCompute(x)
		if e != nil {
			t.Fatal(e)
		}
		if len(v.Cases) < 2 {
			t.Fatalf("variants=%d for beta step=%g", len(v.Cases), step)
		}
		if len(v.Cases[1].Cross) != 120 {
			t.Fatalf("missing transverse diagram at step %g", step)
		}
		if n := len(v.Cases[0].Polar); n != len(v.Cases[1].Polar) || n%len(b) != 0 {
			t.Fatalf("wrong polar grid size for step %g: %d", step, n)
		}
	}
}
func Test049ApprovedImagesAreExactlySupplied(t *testing.T) {
	for _, item := range []struct{ shape, file, hash string }{
		{"sphere_cone", "snow_B15a_approved_geometry.png", "302ae61ec5dd87e9f639d6a67cef637ad3da16a66b9e1d8c87d5ff6046ac9343"},
		{"multi_gable", "snow_multi_gable_approved.png", "bef77c0ff0aba9499b1445d549b924a7f54e54ce9764d99772126513501185a7"},
	} {
		b, e := os.ReadFile(filepath.Join("Resources", "SnowRoofs", item.file))
		if e != nil {
			t.Fatal(e)
		}
		hash := sha256.Sum256(b)
		if hex.EncodeToString(hash[:]) != item.hash {
			t.Fatalf("changed approved image %s", item.file)
		}
		s := defaultSnowCalc()
		s.Shape = item.shape
		got, e := snowRoofImage(s)
		if e != nil {
			t.Fatal(e)
		}
		gotHash := sha256.Sum256(got)
		if gotHash != hash {
			t.Fatalf("%s uses another image", item.shape)
		}
	}
}
func zipFileText(t *testing.T, path, name string) string {
	t.Helper()
	z, e := zip.OpenReader(path)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	for _, f := range z.File {
		if f.Name == name {
			r, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			b, e := io.ReadAll(r)
			r.Close()
			if e != nil {
				t.Fatal(e)
			}
			return string(b)
		}
	}
	t.Fatalf("missing %s in %s", name, path)
	return ""
}
func Test049WordExcelCompletePolars(t *testing.T) {
	s := sample049()
	s.BetaStep = 30
	v, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if len(v.Cases[0].Polar) < 100 || len(v.Cases[1].Polar) < 100 {
		t.Fatalf("incomplete grids: %d %d", len(v.Cases[0].Polar), len(v.Cases[1].Polar))
	}
	dir := t.TempDir()
	doc := filepath.Join(dir, "snow.docx")
	xls := filepath.Join(dir, "snow.xlsx")
	if e := ExportSnowWord(v, doc); e != nil {
		t.Fatal(e)
	}
	if e := ExportSnowExcel(v, xls); e != nil {
		t.Fatal(e)
	}
	xml := zipFileText(t, doc, "word/document.xml")
	if !strings.Contains(xml, "Сечение по зонам 6–1") || !strings.Contains(xml, "Сечение по зонам 2–4–4–2") {
		t.Fatal("Word omitted diagram section")
	}
	if strings.Count(xml, "<w:tr>") < len(v.Cases[0].Polar)+len(v.Cases[1].Polar) {
		t.Fatalf("Word tables were shortened: rows=%d required=%d", strings.Count(xml, "<w:tr>"), len(v.Cases[0].Polar)+len(v.Cases[1].Polar))
	}
	workbook := zipFileText(t, xls, "xl/workbook.xml")
	if !strings.Contains(workbook, "Сетка Б.15а вар.1") || !strings.Contains(workbook, "Сетка Б.15а вар.2") {
		t.Fatal("Excel missing variant data sheet")
	}
}
