package main

import (
	"fmt"
	"strings"
	"testing"
)

// Prepared for the final executable verification; the source-only stage
// deliberately does not compile or run these tests.
func TestExcelPhase9CompleteMultiGableTopology(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "multi_gable"
	s.Count = 2
	s.Angle = 10
	s.Cases = []SnowCase{{Title: "1. Равномерная по пролётам", Source: "Б.5"}}
	plan := snowExcelVariants(s)
	want := map[string]bool{}
	for _, v := range plan {
		want[v.Case.Title] = true
	}
	for bay := 1; bay <= 12; bay++ {
		for side := 1; side <= 2; side++ {
			title := fmt.Sprintf("Частичная нагрузка: пролёт %d, скат %d", bay, side)
			if !want[title] {
				t.Fatalf("missing Excel reserved bay %d, side %d", bay, side)
			}
		}
	}
	for _, title := range []string{
		"2. Неравномерная по пролётам", "3. Вариант по рисунку Б.9",
		"Частичная нагрузка: один скат в каждом пролёте, схема 4",
		"Частичная нагрузка: два смежных ската через пролёт, схема 2",
	} {
		if !want[title] {
			t.Fatalf("missing topology-independent arrangement %q", title)
		}
	}
	if snowExcelSampleCount(s) != 12*32 {
		t.Fatal("multi-bay grid is not reserved for 12 bays")
	}
	start, end := snowExcelSampleInterval(s, 11*32, snowExcelSampleCount(s))
	if !strings.Contains(start, "(11+0/32)") || !strings.Contains(end, "(11+1/32)") {
		t.Fatalf("12th bay coordinates are not calculated from input: %s / %s", start, end)
	}
	if !strings.Contains(snowExcelVariantDataWithStatus(s, plan[0].Case, 0, plan[0].Applicability), `ISNUMBER(A5),ISNUMBER(B5)`) {
		// Both zero and other numeric coordinates must be active; rows beyond n are blank.
		t.Fatal("inactive bay rows and numeric zero boundaries must be handled by formulas")
	}
}

func TestExcelPhase9MultiVaultAllAdjacentPairs(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "multi_vault"
	s.Count = 2
	s.Cases = []SnowCase{{Title: "1. Равномерная по сводам", Source: "Б.5"}}
	found := map[string]string{}
	for _, v := range snowExcelVariants(s) {
		found[v.Case.Title] = v.Applicability
	}
	for i := 1; i < 12; i++ {
		name := fmt.Sprintf("Частичная нагрузка: смежные своды %d и %d", i, i+1)
		if _, ok := found[name]; !ok {
			t.Fatal("missing reserve for", name)
		}
	}
	if !strings.Contains(found["Частичная нагрузка: смежные своды 11 и 12"], ">=12") {
		t.Fatal("last vault pair must depend on user-edited bay count")
	}
}

func TestExcelPhase9RoundPolarSectors(t *testing.T) {
	for _, kind := range []string{"cone", "dome", "sphere_cone"} {
		s := defaultSnowCalc()
		s.Shape = kind
		s.Angle = 5 // no sector in Go at export, all six needed in Excel reserve
		s.Cases = []SnowCase{{Title: "1. Равномерная", Source: "Б.10"}}
		n := 0
		for _, p := range snowExcelVariants(s) {
			if snowIsPartialSector(p.Case) {
				n++
				if !p.Reserved {
					t.Fatal("expected reserved Excel-only sector for", kind)
				}
			}
		}
		if n != 6 {
			t.Fatalf("%s: reserved %d sectors, expected 6", kind, n)
		}
		matrix := snowExcelSectorMatrix(s)
		for _, field := range []string{"μ 0–180°", "μ 180–360°", "μ 0–90°", "μ 90–180°", "μ 180–270°", "μ 270–360°", "Sн", "Sр"} {
			if !strings.Contains(matrix, field) {
				t.Fatalf("%s sector matrix missing %q", kind, field)
			}
		}
	}
}
