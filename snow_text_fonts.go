package main

import (
	"image/color"
	"unicode"
)

// Printed diagram text is drawn with a real OS TrueType font, not a bitmap atlas.
// The Linux renderer is used to inspect generated reports in CI; the shipped
// Windows executable uses native GDI text drawing and does not require Python.
type snowTextCmd struct {
	X, Y, Size int
	Text       string
	Ink        color.RGBA
}

func snowPointSize(size int) int {
	switch {
	case size >= 6:
		return 41
	case size == 5:
		return 38
	case size == 4:
		return 37
	default:
		return 34
	}
}
func snowCaptionWidth(s string, size int) int {
	fontPx := float64(snowPointSize(size))
	w := 0.0
	for _, r := range s {
		switch {
		case r == ' ':
			w += .34 * fontPx
		case r == ',' || r == '.' || r == ':':
			w += .28 * fontPx
		case r == '(' || r == ')' || r == '/' || r == '-' || r == '·':
			w += .4 * fontPx
		case unicode.IsDigit(r):
			w += .64 * fontPx
		case r == 'м' || r == 'н' || r == 'и' || r == 'р':
			w += .60 * fontPx
		default:
			w += .65 * fontPx
		}
	}
	return int(w + 2)
}
