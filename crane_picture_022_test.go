package main

import (
	"path/filepath"
	"testing"
)

func TestCranePictureMulti022(t *testing.T) {
	c := CranePlacementCase{Name: "M+"}
	for i := 0; i < 32; i++ {
		c.Points = append(c.Points, CranePoint{XMM: 3100 + float64(i)*950, PKN: 660 + float64(i%7)*36, Crane: "Кран 500 т"})
	}
	if e := cranePictureForCase([]float64{36000, 36000}, c, filepath.Join(t.TempDir(), "multi.bmp")); e != nil {
		t.Fatal(e)
	}
}
