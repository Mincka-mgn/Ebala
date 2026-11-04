package main

import (
	"bytes"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test047ParapetContinuousRenderingMath(t *testing.T) {
	if got := snowParapetPeak(2, 2); got != 2 {
		t.Fatalf("peak=%g", got)
	}
	for _, p := range []struct{ x, want float64 }{{0, 2}, {1, 1.75}, {2, 1.5}, {4, 1}, {9, 1}} {
		if got := snowParapetMuAt(2, 2, p.x); math.Abs(got-p.want) > 1e-10 {
			t.Fatalf("x=%g got %g want %g", p.x, got, p.want)
		}
	}
	s := sampleSnow("parapet")
	s.Ground = 2
	s.H = 2
	s.Span = 18
	out, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	var local *SnowCase
	for i := range out.Cases {
		if snowParapetLocalCase(out, out.Cases[i]) {
			local = &out.Cases[i]
			break
		}
	}
	if local == nil {
		t.Fatal("missing B.13 local case")
	}
	img, err := snowImage(out, *local)
	if err != nil {
		t.Fatal(err)
	}
	_, err = png.Decode(bytes.NewReader(img))
	if err != nil {
		t.Fatal(err)
	}
}
func Test047PositiveSnowStartCoordinates(t *testing.T) {
	c := snowCase("area", "", "", snowMuInterval(0, 7, 0), snowMuInterval(7, 10, .5), snowMuInterval(10, 18, 1.3))
	starts := snowEpureStartCoordinates(c)
	if len(starts) != 1 || starts[0] != 7 {
		t.Fatalf("onset=%v", starts)
	}
}
func Test047MultiVaultUserDrawing(t *testing.T) {
	s := sampleSnow("multi_vault")
	s.Count = 2
	data, err := snowRoofImage(s)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() < 900 {
		t.Fatalf("preview too small %v", img.Bounds())
	}
	file := filepath.Join("Resources", "SnowRoofs", "snow_multi_vault.svg")
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(b), " Q") < 2 {
		t.Fatal("drawing must contain two vaulted bays")
	}
}
func Test047StepRetainedInCoreButHiddenInChoice(t *testing.T) {
	s := sampleSnow("step")
	if _, err := SnowCompute(s); err != nil {
		t.Fatalf("existing step math should not be deleted: %v", err)
	}
}
