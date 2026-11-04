package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnow032ReportArtifacts(t *testing.T) {
	out := os.Getenv("EBALA_QA_OUT")
	if out == "" {
		return
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	samples := map[string]SnowCalc{}
	a := sampleSnow("gable")
	a.Angle = 10
	a.RightRoof = &SnowRoofPart{Shape: "vault", Span: 22, Rise: 2.8, SlopeMode: "height", EdgeLevel: 6}
	samples["paired"] = a
	d := sampleSnow("dome")
	d.Span = 24
	d.Rise = 3.5
	d.DomeObstacles = true
	samples["dome"] = d
	c := sampleSnow("cone")
	c.Span = 24
	c.Angle = 14
	samples["cone"] = c
	for k, s := range samples {
		s.Region = "III"
		s.Ground = 1.5
		s.JanuarySet = true
		s.JanuaryTemp = -12
		s.WinterWindSet = true
		s.WinterWind = 4.5
		got, err := SnowCompute(s)
		if err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		if err = ExportSnowWord(got, filepath.Join(out, k+".docx")); err != nil {
			t.Fatal(err)
		}
		if err = ExportSnowExcel(got, filepath.Join(out, k+".xlsx")); err != nil {
			t.Fatal(err)
		}
	}
}
