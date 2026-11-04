package main

import (
	"strings"
	"testing"
)

// These tests are prepared for the final validation pass. They deliberately
// were not executed during the source-only phase 8.
func TestExcelPhase8GableReserveAndApplicability(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "gable"
	s.Angle = 10
	s.Cases = []SnowCase{{Title: "1. Равномерная", Source: "Б.1"}}
	plan := snowExcelVariants(s)
	if len(plan) != 6 {
		t.Fatalf("variants: got %d, want 6", len(plan))
	}
	titles := map[string]bool{}
	for _, p := range plan {
		titles[p.Case.Title] = true
	}
	for _, title := range []string{
		"2. Неравномерная", "3. Неравномерная (зеркальная)",
		"Частичное загружение левого ската", "Частичное загружение правого ската",
		"Вариант 3 Б.1. Накопление у конька",
	} {
		if !titles[title] {
			t.Fatalf("reserve absent: %s", title)
		}
	}
	if !strings.Contains(plan[1].Applicability, ">=15") {
		t.Fatal("angle condition absent")
	}
	text := snowExcelVariantDataWithStatus(s, plan[1].Case, 1, plan[1].Applicability)
	if !strings.Contains(text, `IF(AND($B$3=`) {
		t.Fatal("mu is not gated by live applicability cell")
	}
	if !strings.Contains(snowExcelInputData(s), "Оборудование у конька") {
		t.Fatal("ridge-equipment input absent")
	}
}

func TestExcelPhase8ReservedStepWithInitiallyEqualRoofs(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "flat"
	s.RightRoof = &SnowRoofPart{Shape: "gable", Span: 18, RoofHeight: 8, Angle: 10}
	s.Cases = []SnowCase{{Title: "1. Основное загружение обоих пролётов", Source: "Б.1"}}
	plan := snowExcelVariants(s)
	var foundStep, foundRight bool
	for _, p := range plan {
		if strings.HasPrefix(p.Case.Title, "Занос у перепада высот:") {
			foundStep = p.Reserved && strings.Contains(p.Applicability, "'Перепад Б.8'!$B$25")
		}
		if p.Case.Title == "Правый пролёт: 2. Неравномерная" {
			foundRight = p.Reserved && strings.Contains(p.Applicability, "'Коэффициенты'!$B$55")
		}
	}
	if !foundStep || !foundRight {
		t.Fatal("reserved step/right-gable variants absent")
	}
}

func TestExcelPhase8CatalogueDoesNotContainGoLoads(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "gable"
	s.Cases = []SnowCase{{Title: "1. Равномерная", Source: "Б.1", Intervals: []SnowInterval{{X1: 0, X2: 18, Mu: 0.73129}}}}
	catalogue := snowExcelVariantCatalogue(snowExcelVariants(s), "", snowExcelGridSamples)
	if strings.Contains(catalogue, "0.73129") {
		t.Fatal("Go mu leaked into dynamic catalogue")
	}
	if !strings.Contains(catalogue, "COUNTIF") {
		t.Fatal("dynamic variant count absent")
	}
	if !strings.Contains(catalogue, "MAX(") {
		t.Fatal("Excel maxima formula absent")
	}
}
