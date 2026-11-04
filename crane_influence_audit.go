package main

import (
	"fmt"
	"math"
	"strings"
)

// CraneInfluenceAudit independently reproduces the unit-load influence ordinate
// by Clapeyron's three-moment equation. EI is uniform as in solveCraneBeam.
// All distances below are metres; unit forces are 1 kN.
type CraneInfluenceAudit struct {
	Span                                        int // 1-based wheel span, or 0 when the wheel is outside the beam
	A, B, L                                     float64
	RHS, Forward, Supports                      []float64 // right sides, Thomas sweep, M at internal supports
	CPrime                                      []float64 // sweep coefficient, depends on span lengths only
	TargetSpan                                  int       // 1-based span of the requested M or Q
	TargetType                                  string    // M, QR (right of support), QL (left of support)
	SectionM, Z, M0, ML, MR, RLeft, RRight, Eta float64
}

func craneTargetSpan(spans []float64, c CranePlacementCase) (int, string, error) {
	if c.Units == "кН·м" {
		x := c.SectionMM / 1000
		start := 0.0
		for i, Lmm := range spans {
			L := Lmm / 1000
			if x <= start+L+1e-8 && x >= start-1e-8 {
				return i + 1, "M", nil
			}
			start += L
		}
		return 0, "", fmt.Errorf("сечение момента вне балки: %.1f мм", c.SectionMM)
	}
	if c.Units == "кН" {
		const marker = "опоры "
		idx := strings.Index(c.Name, marker)
		if idx < 0 {
			return 0, "", fmt.Errorf("нет номера опоры в %q", c.Name)
		}
		var sup int
		if _, err := fmt.Sscanf(c.Name[idx+len(marker):], "%d", &sup); err != nil {
			return 0, "", err
		}
		if strings.Contains(c.Name, "справа от опоры") && sup >= 1 && sup <= len(spans) {
			return sup, "QR", nil
		}
		if strings.Contains(c.Name, "слева от опоры") && sup >= 2 && sup <= len(spans)+1 {
			return sup - 1, "QL", nil
		}
	}
	return 0, "", fmt.Errorf("не распознана величина %q (%q)", c.Units, c.Name)
}

func craneInfluenceUnitAudit(spansMM []float64, c CranePlacementCase, xMM float64) (CraneInfluenceAudit, error) {
	var z CraneInfluenceAudit
	n := len(spansMM)
	if n < 1 {
		return z, fmt.Errorf("балка без пролётов")
	}
	z.SectionM = c.SectionMM / 1000
	target, kind, err := craneTargetSpan(spansMM, c)
	if err != nil {
		return z, err
	}
	z.TargetSpan, z.TargetType = target, kind
	m := n - 1
	z.RHS = make([]float64, m)
	z.Forward = make([]float64, m)
	z.Supports = make([]float64, m)
	z.CPrime = make([]float64, m)
	x := xMM / 1000
	total := 0.0
	for _, l := range spansMM {
		total += l / 1000
	}
	if x >= 0 && x <= total {
		pos := 0.0
		for i, lmm := range spansMM {
			l := lmm / 1000
			if x <= pos+l+1e-9 {
				z.Span = i + 1
				z.A = math.Max(0, math.Min(l, x-pos))
				z.B = l - z.A
				z.L = l
				break
			}
			pos += l
		}
	}
	for j := 1; j <= m; j++ {
		if z.Span == j {
			z.RHS[j-1] = -z.A * z.B * (z.L + z.A) / z.L
		} else if z.Span == j+1 {
			z.RHS[j-1] = -z.A * z.B * (2*z.L - z.A) / z.L
		}
	}
	// Coefficients of the three-moment tridiagonal system, solved by Thomas sweep.
	for j := 1; j <= m; j++ {
		left := spansMM[j-1] / 1000
		right := spansMM[j] / 1000
		lower := 0.0
		if j > 1 {
			lower = left
		}
		upper := 0.0
		if j < m {
			upper = right
		}
		denom := 2 * (left + right)
		if j > 1 {
			denom -= lower * z.CPrime[j-2]
		}
		if denom <= 0 {
			return z, fmt.Errorf("неверная матрица опорных моментов")
		}
		z.CPrime[j-1] = upper / denom
		rhs := z.RHS[j-1]
		if j > 1 {
			rhs -= lower * z.Forward[j-2]
		}
		z.Forward[j-1] = rhs / denom
	}
	for j := m; j >= 1; j-- {
		z.Supports[j-1] = z.Forward[j-1]
		if j < m {
			z.Supports[j-1] -= z.CPrime[j-1] * z.Supports[j]
		}
	}
	if target > 1 {
		z.ML = z.Supports[target-2]
	}
	if target <= m {
		z.MR = z.Supports[target-1]
	}
	start := 0.0
	for i := 0; i < target-1; i++ {
		start += spansMM[i] / 1000
	}
	length := spansMM[target-1] / 1000
	z.Z = z.SectionM - start
	if kind == "M" && z.Span == target {
		if z.Z <= z.A {
			z.M0 = z.B * z.Z / z.L
		} else {
			z.M0 = z.A * (z.L - z.Z) / z.L
		}
	}
	z.RLeft = (z.MR - z.ML) / length
	z.RRight = (z.ML - z.MR) / length
	if z.Span == target {
		z.RLeft += z.B / z.L
		z.RRight += z.A / z.L
	}
	switch kind {
	case "M":
		z.Eta = z.ML*(1-z.Z/length) + z.MR*z.Z/length + z.M0
	case "QR":
		z.Eta = z.RLeft
	case "QL":
		z.Eta = -z.RRight
	}
	return z, nil
}
