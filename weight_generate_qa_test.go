//go:build qaqa

package main

import (
	"os"
	"testing"
)

func TestMakeQA(t *testing.T) {
	w := exampleWeight()
	os.MkdirAll("/mnt/data/ebala_v04_qa", 0755)
	if err := ExportWeightWord(w, "/mnt/data/ebala_v04_qa/weight.docx"); err != nil {
		t.Fatal(err)
	}
	if err := ExportWeightExcel(w, "/mnt/data/ebala_v04_qa/weight.xlsx"); err != nil {
		t.Fatal(err)
	}
}
