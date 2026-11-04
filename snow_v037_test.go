package main

import (
	"bytes"
	"image"
	"image/png"
	"math"
	"strings"
	"testing"
)

func TestSnow037CommaBaseline(t *testing.T) {
	atlas := snowNumericAtlas()
	if atlas == nil {
		t.Fatal("glyph atlas missing")
	}
	idx := strings.IndexRune(snowAtlasChars, ',')
	ymin, ymax := snowAtlasCellHeight, 0
	for y := 0; y < snowAtlasCellHeight; y++ {
		for x := idx * snowAtlasCellWidth; x < (idx+1)*snowAtlasCellWidth; x++ {
			_, _, _, a := atlas.At(x, y).RGBA()
			if a != 0 {
				if y < ymin {
					ymin = y
				}
				if y > ymax {
					ymax = y
				}
			}
		}
	}
	if ymin < 30 || ymax < 40 {
		t.Fatalf("comma not at bottom of number cell: %d..%d", ymin, ymax)
	}
}
func TestSnow037EmbeddedUnitsCaption(t *testing.T) {
	for _, kgf := range []bool{false, true} {
		f := newSnowFigure()
		if err := snowDrawUnitCaption(f.img, 85, 429, kgf); err != nil {
			t.Fatal(err)
		}
		dark := 0
		for y := 429; y < 468; y++ {
			for x := 85; x < 750; x++ {
				c := f.img.RGBAAt(x, y)
				if c.R < 160 && c.G < 160 {
					dark++
				}
			}
		}
		if dark < 100 {
			t.Fatalf("unit text invisible for kgf=%v: %d", kgf, dark)
		}
	}
	s := defaultSnowCalc()
	s.Shape = "flat"
	s.Ground = 1
	s.Span = 18
	out, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	img, err := snowImage(out, out.Cases[0])
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(img))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds() != image.Rect(0, 0, 1400, 470) {
		t.Fatalf("unexpected bounds %v", decoded.Bounds())
	}
}
func TestSnow037AutomaticUneven(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "gable"
	s.Angle = 5
	s.Ground = 2
	s.RoofSensitive = false
	s.Span = 18
	s.Width = 24
	out, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Cases) < 2 || !strings.Contains(out.Cases[1].Title, "Проверочная") {
		t.Fatalf("optional check omitted: %+v", out.Cases)
	}
	if math.Abs(out.Cases[1].Intervals[0].Mu-0.9) > 1e-9 || math.Abs(out.Cases[1].Intervals[1].Mu-1.1) > 1e-9 {
		t.Fatal("unexpected uneven factors")
	}
	s.Span = 120
	s.Width = 150
	out, err = SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Cases) < 2 || !strings.Contains(out.Cases[1].Title, "Обязательная") {
		t.Fatalf("mandatory uneven B.1 omitted: %+v", out.Cases)
	}
}
func TestSnow037ConversionRoundTrip(t *testing.T) {
	for _, v := range []float64{0.5, 1, 1.4, 2.5, 6.3} {
		if math.Abs(kgfM2ToKpa(kpaToKgfM2(v))-v) > 1e-11 {
			t.Fatalf("round-trip changed input %g", v)
		}
	}
}
