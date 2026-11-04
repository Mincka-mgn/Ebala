package main

import "testing"

func TestCranePicture(t *testing.T) {
	c := CranePlacementCase{Name: "М+", Points: []CranePoint{{XMM: 1700, PKN: 170, Crane: "Кран 1"}, {XMM: 2700, PKN: 170, Crane: "Кран 1"}, {XMM: 7900, PKN: 120, Crane: "Кран 2"}}}
	if err := cranePictureForCase([]float64{6000, 6000}, c, "/tmp/ebala_crane_preview.bmp"); err != nil {
		t.Fatal(err)
	}
}
