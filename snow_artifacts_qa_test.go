package main

import (
	"os"
	"testing"
)

func TestSnowDemoExportsQA(t *testing.T) {
	target := os.Getenv("SNOW_DEMO_OUT")
	if target == "" {
		return
	}
	s := sampleSnow("step")
	s.H = 3
	s.Upper = 20
	s.Lower = 18
	s.LowerShape = "gable"
	s.Angle = 20
	s, e := SnowCompute(s)
	if e != nil {
		t.Fatal(e)
	}
	if e = ExportSnowWord(s, target+".docx"); e != nil {
		t.Fatal(e)
	}
	if e = ExportSnowExcel(s, target+".xlsx"); e != nil {
		t.Fatal(e)
	}
}
