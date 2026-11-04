package main

import (
	"image/png"
	"math"
	"os"
	"strings"
	"testing"
)

func TestSnow035BothLanternsCZone(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "skylight"
	s.Span = 18
	s.Width = 24
	s.Ground = 1
	s.H = 2
	s.SkylightWidth = 6
	s.RoofHeight = 8
	s.SlopeMode = "height"
	s.Rise = 0.27
	s.RightRoof = &SnowRoofPart{Shape: "skylight", Span: 18, Width: 24, SkylightWidth: 6, LanternHeight: 2, RoofHeight: 8, SlopeMode: "height", Rise: 0.27}
	result, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range result.Cases {
		if !strings.Contains(v.Title, "1C.") || !strings.Contains(v.Title, "Оба фонаря") || !strings.Contains(v.Title, "правый 1C.") || !strings.Contains(v.Title, "левый 1C.") {
			continue
		}
		if math.Abs(snowMuAt(v, 3)-1.1) > 1e-7 || math.Abs(snowMuAt(v, 21)-1.1) > 1e-7 {
			t.Fatalf("both lanterns should use mu2=1.1: %v", v.Intervals)
		}
		found = true
	}
	if !found {
		t.Fatal("missing simultaneous C+C case")
	}
}
func TestSnow035SmoothFigure(t *testing.T) {
	s := defaultSnowCalc()
	s.Ground = 1
	s.Shape = "flat"
	s.Span = 18
	result, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	data, err := snowImage(result, result.Cases[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 5000 {
		t.Fatalf("figure unexpectedly small %d", len(data))
	}
	im, e := png.Decode(strings.NewReader(string(data)))
	if e != nil {
		t.Fatal(e)
	}
	if im.Bounds().Dx() != 1400 {
		t.Fatal("wrong image width")
	}
	if _, err := os.Stat("Resources/SnowRoofs/number_glyphs.png"); err != nil {
		t.Fatal(err)
	}
}
func TestSnow035QAImage(t *testing.T) {
	dest := os.Getenv("SNOW_035_FIG_OUT")
	if dest == "" {
		return
	}
	s := defaultSnowCalc()
	s.Shape = "skylight"
	s.Ground = 1
	s.Span = 18
	s.Width = 24
	s.H = 2
	s.SkylightWidth = 6
	s.RoofHeight = 8
	s.Rise = .27
	s.SlopeMode = "height"
	s.RightRoof = &SnowRoofPart{Shape: "skylight", Span: 18, Width: 24, SkylightWidth: 6, LanternHeight: 2, RoofHeight: 8, Rise: .27, SlopeMode: "height"}
	out, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range out.Cases {
		if strings.Contains(c.Title, "Оба фонаря") && strings.Contains(c.Title, "левый 1C.") && strings.Contains(c.Title, "правый 1C.") {
			b, e := snowImage(out, c)
			if e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(dest, b, 0644); e != nil {
				t.Fatal(e)
			}
			return
		}
	}
	t.Fatal("C+C example not generated")
}
func TestSnow035QAWord(t *testing.T) {
	dest := os.Getenv("SNOW_035_WORD_OUT")
	if dest == "" {
		return
	}
	s := defaultSnowCalc()
	s.Shape = "skylight"
	s.Ground = 1
	s.Span = 18
	s.Width = 24
	s.H = 2
	s.SkylightWidth = 6
	s.RoofHeight = 8
	s.Rise = .27
	s.SlopeMode = "height"
	s.RightRoof = &SnowRoofPart{Shape: "skylight", Span: 18, Width: 24, SkylightWidth: 6, LanternHeight: 2, RoofHeight: 8, Rise: .27, SlopeMode: "height"}
	out, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	if err = ExportSnowWord(out, dest); err != nil {
		t.Fatal(err)
	}
}
