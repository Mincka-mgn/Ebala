package main

import (
	"bytes"
	"image/color"
	"image/png"
	"os"
	"testing"
)

// Regression: printed labels remain readable in a different working
// directory and preserve the black/red/green numerical notation.
func TestSnow036EmbeddedGlyphAtlasFromForeignCWD(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	s := defaultSnowCalc()
	s.Shape = "flat"
	s.Span = 18
	s.Ground = 1
	s.JanuaryTemp = -10
	s.JanuarySet = true
	out, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	b, err := snowImage(out, out.Cases[0])
	if err != nil || len(b) == 0 {
		t.Fatalf("render failed: %v", err)
	}
	f := newSnowFigure()
	// Distinct colors and three contributions are drawn in a single label.
	f.coloredLoadText(100, 40, 5, out, 1)
	b, err = f.png()
	if err != nil {
		t.Fatal(err)
	}
	im, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	reds, greens := 0, 0
	for y := 40; y < 120; y++ {
		for x := 100; x < 650; x++ {
			c := color.RGBAModel.Convert(im.At(x, y)).(color.RGBA)
			if c.R > 120 && c.R > c.G*2 && c.R > c.B {
				reds++
			}
			if c.G > 90 && c.G > c.R*2 && c.G > c.B {
				greens++
			}
		}
	}
	if reds < 20 || greens < 20 {
		t.Fatalf("colors missing: red=%d green=%d", reds, greens)
	}
	// Verify parentheses by comparing against the glyph positions in the atlas.
	left := 100 + snowCaptionWidth(decimalInput(out.Q(1), 2), 5) + snowCaptionWidth(" ", 5)
	ink := false
	for y := 50; y < 90; y++ {
		for x := left; x < left+snowCaptionWidth("(", 5); x++ {
			c := color.RGBAModel.Convert(im.At(x, y)).(color.RGBA)
			if c != (color.RGBA{255, 255, 255, 255}) {
				ink = true
				break
			}
		}
	}
	if !ink {
		t.Fatal("opening parenthesis is invisible")
	}
}
func TestSnow036ReducedSnowDependsOnJanuary(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "flat"
	s.Span = 18
	s.Ground = 1
	s.JanuarySet = true
	s.JanuaryTemp = 0
	if s.ReducedApplicable() {
		t.Fatal("reduction must not apply for T=0")
	}
	s.JanuaryTemp = -10
	if !s.ReducedApplicable() {
		t.Fatal("reduction missing at T=-10")
	}
}

func TestSnow036PreviewImage(t *testing.T) {
	path := os.Getenv("SNOW_036_PREVIEW")
	if path == "" {
		return
	}
	s := defaultSnowCalc()
	s.Shape = "flat"
	s.Span = 18
	s.Ground = 1
	s.JanuarySet = true
	s.JanuaryTemp = -10
	out, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	b, e := snowImage(out, out.Cases[0])
	if e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(path, b, 0644); e != nil {
		t.Fatal(e)
	}
}
