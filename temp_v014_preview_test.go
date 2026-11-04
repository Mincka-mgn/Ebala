package main

import (
	"path/filepath"
	"testing"
)

func TestV014Preview(t *testing.T) {
	root := t.TempDir()
	g, e := loadSP20Guide(filepath.Join("Catalog", "sp20_guide.json"))
	if e != nil {
		t.Fatal(e)
	}
	c, e := loadTempCatalog(filepath.Join("Catalog", "temporary_loads.json"))
	if e != nil {
		t.Fatal(e)
	}
	a, e := tempSP20Section(g.Records[0], TempFull, 0, 1)
	if e != nil {
		t.Fatal(e)
	}
	b := tempSectionFromEntry(c.Records[0])
	_ = b
	var car TempSelection
	car, e = tempManualSteps("Кран грузоподъёмностью 32 т", 4, "105; 105; 95; 95", "2500; 3400; 2500")
	if e != nil {
		t.Fatal(e)
	}
	var tr TempSelection
	for _, v := range c.Records {
		if v.Group == "СП 35 · транспорт" && v.Title == "Промышленный автомобиль АБ-51" {
			tr = tempSectionFromEntry(v)
			break
		}
	}
	if tr.Title == "" {
		t.Fatal("missing transport")
	}
	r := TempReport{Version: 1, Project: "Контрольный объект", Sections: []TempSelection{a, tr, car}}
	if e := exportTempWord(r, filepath.Join(root, "preview_014.docx")); e != nil {
		t.Fatal(e)
	}
	if e := exportTempExcel(r, filepath.Join(root, "preview_014.xlsx")); e != nil {
		t.Fatal(e)
	}
	blank := TempReport{Version: 1, Sections: []TempSelection{tempEmptyCranePreview()}}
	if exportTempWord(blank, filepath.Join(root, "blank_014.docx")) == nil {
		t.Fatal("empty should be rejected")
	}
}
func tempEmptyCranePreview() TempSelection {
	return TempSelection{Title: "Кран 1", Group: "Краны · ручной ввод"}
}
