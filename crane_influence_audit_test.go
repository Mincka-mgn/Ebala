package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThreeMomentAuditVsBeamSolver023(t *testing.T) {
	for _, spans := range [][]float64{{12000}, {10000, 10000}, {7000, 12000, 9000}, {4500, 6500, 11500, 8000}, {6000, 7000, 9000, 12000, 4000, 5000}} {
		nodes := []float64{0}
		for _, L := range spans {
			nodes = append(nodes, nodes[len(nodes)-1]+L)
		}
		cases := []CranePlacementCase{}
		for i := 0; i < len(spans); i++ {
			cases = append(cases, CranePlacementCase{Name: fmt.Sprintf("М+ в середине пролёта %d", i+1), SectionMM: (nodes[i] + nodes[i+1]) / 2, Units: "кН·м"})
		}
		for i := 1; i <= len(nodes); i++ {
			if i > 1 {
				cases = append(cases, CranePlacementCase{Name: fmt.Sprintf("Q слева от опоры %d (+)", i), SectionMM: nodes[i-1], Units: "кН"})
			}
			if i <= len(spans) {
				cases = append(cases, CranePlacementCase{Name: fmt.Sprintf("Q справа от опоры %d (+)", i), SectionMM: nodes[i-1], Units: "кН"})
			}
		}
		xs := []float64{-1000, 0, 1, 1900, 3500, nodes[len(nodes)-1] / 3, nodes[len(nodes)-1] / 2, nodes[len(nodes)-1] - 10, nodes[len(nodes)-1], nodes[len(nodes)-1] + 1000}
		for _, x := range nodes {
			xs = append(xs, x)
		}
		for _, c := range cases {
			for _, x := range xs {
				aa, err := craneInfluenceUnitAudit(spans, c, x)
				if err != nil {
					t.Fatal(err)
				}
				eta, err := craneCaseInfluence(spans, c, x)
				if err != nil {
					t.Fatal(err)
				}
				if math.Abs(aa.Eta-eta) > 1e-7*math.Max(1, math.Abs(eta)) {
					t.Fatalf("spans %v case %s x %.2f audit %.10f core %.10f", spans, c.Name, x, aa.Eta, eta)
				}
			}
		}
	}
}
func TestCraneAuditExcel023(t *testing.T) {
	spans := []float64{9000, 12000, 7000}
	pts := []CranePoint{{XMM: 4000, PKN: 60, Crane: "Кран A"}, {XMM: 8500, PKN: 70, Crane: "Кран A"}, {XMM: 13500, PKN: 90, Crane: "Кран Б"}, {XMM: 22300, PKN: 80, Crane: "Кран Б"}}
	state, err := solveCraneBeam(spans, pts, []float64{4500.0 / 1000})
	if err != nil {
		t.Fatal(err)
	}
	r := CranePlacementResult{SpansMM: spans, Cases: []CranePlacementCase{{Name: "М+ в середине пролёта 1", SectionMM: 4500, Units: "кН·м", Value: state.moments[0], Points: pts}, {Name: "Q справа от опоры 2 (+)", SectionMM: 9000, Units: "кН", Value: state.shearRight[1], Points: pts}}}
	file := filepath.Join(t.TempDir(), "audit.xlsx")
	if err := exportCranePlacementExcel(r, file); err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	hasBeam := false
	hasFormulas := false
	hasSum := false
	for _, p := range z.File {
		if strings.HasSuffix(p.Name, ".xml") || strings.HasSuffix(p.Name, ".rels") {
			rd, e := p.Open()
			if e != nil {
				t.Fatal(e)
			}
			data := make([]byte, p.UncompressedSize64)
			n, e := rd.Read(data)
			_ = e
			rd.Close()
			data = data[:n]
			if xml.Unmarshal(data, new(interface{})) != nil { // Validate using stream parser below instead for generic XML.
			}
			d := xml.NewDecoder(strings.NewReader(string(data)))
			for {
				_, e := d.Token()
				if e != nil {
					if e.Error() != "EOF" {
						t.Fatalf("invalid xml %s %v", p.Name, e)
					}
					break
				}
			}
			if p.Name == "xl/worksheets/sheet3.xml" && strings.Contains(string(data), "Lлев") {
				hasBeam = true
			}
			if p.Name == "xl/worksheets/sheet2.xml" && strings.Contains(string(data), "INDEX(") && strings.Contains(string(data), "Балка") {
				hasFormulas = true
			}
			if p.Name == "xl/worksheets/sheet1.xml" && strings.Contains(string(data), "SUMIF(") {
				hasSum = true
			}
		}
	}
	if !hasBeam || !hasFormulas || !hasSum {
		t.Fatalf("missing audit beam=%v formulas=%v sum=%v", hasBeam, hasFormulas, hasSum)
	}
	if stat, e := os.Stat(file); e != nil || stat.Size() < 3000 {
		t.Fatalf("invalid export %v %v", stat, e)
	}
}
