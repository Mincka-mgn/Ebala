package main

import (
	"bytes"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test038PrintedSnowLoadsAndColors(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "flat"
	s.Span = 18
	s.Ground = 2
	s.JanuarySet = true
	s.JanuaryTemp = -10
	out, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if !out.ReducedApplicable() {
		t.Fatal("expected reduced load in this example")
	}
	label := snowLoadValue(out, 1)
	if !strings.Contains(label, ",00") || !strings.Contains(label, " (") || !strings.Contains(label, "/") || !strings.HasSuffix(label, ")") {
		t.Fatalf("unreadable printed notation: %s", label)
	}
	b, e := snowImage(out, out.Cases[0])
	if e != nil {
		t.Fatal(e)
	}
	im, e := png.Decode(bytes.NewReader(b))
	if e != nil {
		t.Fatal(e)
	}
	red, green, black := 0, 0, 0
	for y := 0; y < 220; y++ {
		for x := 0; x < im.Bounds().Dx(); x++ {
			c := color.RGBAModel.Convert(im.At(x, y)).(color.RGBA)
			if c.R > 140 && c.R > c.G*2 && c.R > c.B {
				red++
			}
			if c.G > 80 && c.G > c.R*2 && c.G > c.B {
				green++
			}
			if c.R < 85 && c.G < 85 && c.B < 85 {
				black++
			}
		}
	}
	if red < 30 || green < 30 || black < 30 {
		t.Fatalf("printed color contributions missing: black %d red %d green %d", black, red, green)
	}
	if dir := os.Getenv("EBALA_QA_OUT"); dir != "" {
		if e = os.WriteFile(filepath.Join(dir, "snow_038_colored_diagram.png"), b, 0644); e != nil {
			t.Fatal(e)
		}
	}
}
