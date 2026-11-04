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

func Test048ParapetApprovedFigure(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("Resources", "SnowRoofs", "snow_B13_parapet.png"))
	if err != nil {
		t.Fatal(err)
	}
	im, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if im.Bounds().Dx() != 506 || im.Bounds().Dy() != 214 {
		t.Fatalf("approved sketch size %v", im.Bounds())
	}
}
func Test048ParapetExtraAlternativeSmoothPolyline(t *testing.T) {
	s := sampleSnow("parapet")
	s.Ground = 2
	s.H = 2
	s.Span = 18
	out, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range out.Cases {
		if !strings.Contains(c.Title, "Дополнительная проверка") {
			continue
		}
		found = true
		pts, ok := snowParapetProfile(out, c)
		if !ok || len(pts) < 2 || len(pts) > 3 {
			t.Fatalf("unexpected alternative profile %v ok=%v", pts, ok)
		}
		if math.Abs(pts[0].Mu-c.Intervals[0].Mu) > 1e-8 {
			t.Fatalf("peak changed %v", pts)
		}
		if math.Abs(pts[len(pts)-1].Mu-c.Intervals[len(c.Intervals)-1].Mu) > 1e-8 {
			t.Fatalf("background changed %v", pts)
		}
		if _, err := snowImage(out, c); err != nil {
			t.Fatal(err)
		}
	}
	if !found {
		t.Fatal("no parapet B.8 alternative")
	}
}
func Test048DomeSteepEdgeIsZeroAndNotRejected(t *testing.T) {
	s := sampleSnow("dome")
	s.Span = 18
	s.Rise = 6 // maximum local slope ~67.4°, only the edges should have μ=0
	out, err := SnowCompute(s)
	if err != nil {
		t.Fatalf("steep dome should be calculated, not blocked: %v", err)
	}
	if len(out.Cases) < 2 {
		t.Fatalf("missing dome alternatives")
	}
	for _, c := range out.Cases {
		if c.Intervals[0].Mu > 1e-8 && strings.Contains(c.Title, "Симметрич") {
			t.Fatalf("steep edge μ must be 0; %v", c.Intervals[0])
		}
	}
}
