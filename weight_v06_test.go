package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV06OptionalWeightNameInReports(t *testing.T) {
	w := WeightCalc{Name: "", Rows: []WeightRow{{Name: "Безымянный слой", Kind: "direct", GNorm: 35, Gamma: 1.2}}}
	dir := t.TempDir()
	word := filepath.Join(dir, "w.docx")
	excel := filepath.Join(dir, "w.xlsx")
	if e := ExportWeightWord(w, word); e != nil {
		t.Fatal(e)
	}
	if e := ExportWeightExcel(w, excel); e != nil {
		t.Fatal(e)
	}
	xml := zipEntry(t, word, "word/document.xml")
	if strings.Contains(xml, "Конструкция:") {
		t.Fatal("unexpected construction-name line")
	}
	xml = zipEntry(t, excel, "xl/worksheets/sheet1.xml")
	if strings.Contains(xml, "СОБСТВЕННЫЙ ВЕС:") {
		t.Fatal("unexpected colon when name is blank")
	}
}
func TestV06ReorganizedCategories(t *testing.T) {
	b, e := os.ReadFile("Catalog/materials.json")
	if e != nil {
		t.Fatal(e)
	}
	var catalog weightTestCatalog
	if e = json.Unmarshal(b, &catalog); e != nil {
		t.Fatal(e)
	}
	ids := map[string]string{}
	for _, c := range catalog.Categories {
		ids[c.ID] = c.Group
	}
	for _, id := range []string{"c24", "c17", "c19", "c20", "c21", "c23"} {
		if ids[id] != "Покрытия и кровли" {
			t.Errorf("wrong roof category for %s: %s", id, ids[id])
		}
	}
	for _, v := range catalog.Items {
		if strings.Contains(strings.ToLower(v.Name), "асфальтобетон") && ids[v.CategoryID] != "Паро- и гидроизоляция" {
			t.Errorf("asphalt misclassified: %s", v.CategoryID)
		}
	}
}

type weightTestCatalog struct {
	Categories []struct {
		ID    string `json:"id"`
		Group string `json:"group"`
	} `json:"categories"`
	Items []struct {
		Name       string `json:"name"`
		CategoryID string `json:"category_id"`
	} `json:"items"`
}
