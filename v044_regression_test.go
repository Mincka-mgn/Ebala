package main

import (
	"archive/zip"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func zipDocumentPart(t *testing.T, fn, part string) string {
	t.Helper()
	r, err := zip.OpenReader(fn)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name != part {
			continue
		}
		rr, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rr)
		rr.Close()
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	t.Fatalf("part %s missing in %s", part, fn)
	return ""
}

func Test044ProjectSnapshotIndependentOfSite(t *testing.T) {
	p := initialProject()
	first := Site{Region: "Краснодарский край", Place: "Краснодар", SnowZone: "III", SnowGround: 1.5, IceZone: "IV", WindZone: "IV"}
	p.SetSite(first)
	snow := defaultSnowCalc()
	snow.Region = "III"
	snow.Ground = 1.5
	snow, err := SnowCompute(snow)
	if err != nil {
		t.Fatal(err)
	}
	sid := p.putSnowCalc(0, snow)
	ice := defaultIceLoad()
	ice.Region = "IV"
	ice.BaseMM = 15
	iid := p.putIceCalc(0, ice)
	p.renameBlock(sid, "Перекрытие")
	p.renameBlock(iid, "Трос")
	p.SetSite(Site{Region: "Место без карты", Place: "Неизвестное место"})
	if p.SavedBlocks[0].Snow.Region != "III" || p.SavedBlocks[0].Snow.Ground != 1.5 {
		t.Fatal("site rewrote snow snapshot")
	}
	if p.SavedBlocks[1].Ice.Region != "IV" || p.SavedBlocks[1].Ice.BaseMM != 15 {
		t.Fatal("site rewrote ice snapshot")
	}
	if p.SavedBlocks[0].Title != "Перекрытие" || p.SavedBlocks[1].Title != "Трос" {
		t.Fatal("lost labels")
	}
	// Incomplete independent drafts must not block JSON project persistence.
	p.Weight = &WeightCalc{Name: "Незавершённый пирог", Rows: []WeightRow{{Name: "Слой", Kind: "direct", GNorm: -1, Gamma: 1.1}}}
	fn := filepath.Join(t.TempDir(), "site-project.ebala")
	if err := p.Save(fn); err != nil {
		t.Fatalf("incomplete project cannot be saved: %v", err)
	}
	loaded, err := LoadProject(fn)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SavedBlocks[0].Snow.Region != "III" || loaded.SavedBlocks[1].Ice.BaseMM != 15 {
		t.Fatal("snapshots not restored")
	}
	if loaded.Weight == nil || loaded.Weight.Name != "Незавершённый пирог" {
		t.Fatal("draft not persisted")
	}
}

func Test044NamedWeightBlocksAndIndependentCopies(t *testing.T) {
	p := initialProject()
	floor := WeightCalc{Name: "Конструкция пола", Rows: []WeightRow{
		{Name: "Плиты", Kind: "direct", GNorm: 290, Gamma: 1.1},
		{Name: "Стяжка", Kind: "direct", GNorm: 76, Gamma: 1.3}}}
	roof := WeightCalc{Name: "Утеплённая кровля", Rows: []WeightRow{
		{Name: "Утеплитель", Kind: "direct", GNorm: 8, Gamma: 1.2}}}
	id := p.putWeightCalc(0, floor)
	p.putWeightCalc(0, roof)
	// Deliberately edit original memory after saving to verify snapshots are isolated.
	floor.Rows[0].GNorm = 9999
	if p.SavedBlocks[0].WeightCalc.Rows[0].GNorm != 290 {
		t.Fatal("stored weight shares mutable layer slice")
	}
	if p.SavedBlocks[0].Title != "Конструкция пола" || p.SavedBlocks[1].Title != "Утеплённая кровля" {
		t.Fatal("wrong default names")
	}
	p.renameBlock(id, "Перекрытие второго этажа")
	if p.SavedBlocks[0].WeightCalc.Name != "Перекрытие второго этажа" {
		t.Fatal("rename did not sync construction name")
	}
	r := p.selectedReport()
	if len(r.WeightBlocks) != 2 || r.WeightBlocks[0].Title != "Перекрытие второго этажа" || len(r.WeightBlocks[0].Rows) != 2 {
		t.Fatalf("lost separate takeoffs: %+v", r.WeightBlocks)
	}
	if diff := (r.WeightBlocks[0].Rows[0].Norm + r.WeightBlocks[0].Rows[1].Norm) - kgfM2ToKpa(366); math.Abs(diff) > 1e-9 {
		t.Fatal("normative subtotal")
	}
	dir := t.TempDir()
	word := filepath.Join(dir, "report.docx")
	excel := filepath.Join(dir, "report.xlsx")
	if err := exportTempWord(r, word); err != nil {
		t.Fatal(err)
	}
	if err := exportTempExcel(r, excel); err != nil {
		t.Fatal(err)
	}
	w := zipDocumentPart(t, word, "word/document.xml")
	x := zipDocumentPart(t, excel, "xl/worksheets/sheet1.xml")
	for _, name := range []string{"Перекрытие второго этажа", "Утеплённая кровля", "ИТОГО"} {
		if !strings.Contains(w, name) || !strings.Contains(x, name) {
			t.Fatalf("missing %s in group exports", name)
		}
	}
	if strings.Count(w, "ИТОГО") != 2 || strings.Count(x, "ИТОГО") != 2 {
		t.Fatal("each construction must have an independent subtotal")
	}
	if !strings.Contains(w, `<w:cantSplit/>`) || !strings.Contains(w, `<w:keepNext/>`) || !strings.Contains(w, `<w:tblHeader`) {
		t.Fatal("Word page-break controls missing")
	}
}

func Test044StandaloneWeightReportWithoutFooterFormula(t *testing.T) {
	fn := filepath.Join(t.TempDir(), "weight.docx")
	w := WeightCalc{Name: "Покрытие", Rows: []WeightRow{{Name: "Шифер", Kind: "direct", GNorm: 38, Gamma: 1.1}}}
	if err := ExportWeightWord(w, fn); err != nil {
		t.Fatal(err)
	}
	xml := zipDocumentPart(t, fn, "word/document.xml")
	if strings.Contains(xml, "γf,экв = Σgр") || strings.Contains(xml, "Пересчёт: gр") {
		t.Fatal("unwanted footer formulas in client Word")
	}
	if !strings.Contains(xml, "Общий коэффициент надёжности") {
		t.Fatal("equivalent factor summary missing")
	}
}

func Test044SnowRidgeOptionalScheme(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "gable"
	s.Angle = 20
	s.RidgeEquipment = true
	computed, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range computed.Cases {
		if strings.Contains(c.Source, "вариант 3") {
			found = true
		}
	}
	if !found {
		t.Fatal("ridge equipment must activate B.1 variant 3")
	}
	s.RidgeEquipment = false
	no, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range no.Cases {
		if strings.Contains(c.Source, "вариант 3") {
			t.Fatal("variant 3 without ridge equipment")
		}
	}
	// A second roof uses its own schemes, not single-span B.1 ridge variant.
	s.RidgeEquipment = true
	s.RightRoof = &SnowRoofPart{Shape: "gable", Span: 18, Width: 24, RoofHeight: 8, Angle: 20}
	paired, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range paired.Cases {
		if strings.Contains(c.Source, "рисунок Б.1, вариант 3") {
			t.Fatal("single-span ridge variant leaked into paired roof")
		}
	}
}

func Test044ReportWordingAndPlace(t *testing.T) {
	if got := formatSitePlace("Краснодарский край", "", "Усть-Лабинск"); got != "Краснодарский край, Усть-Лабинск" {
		t.Fatal(got)
	}
	if got := formatSitePlace("", "", "Усть-Лабинск"); got != "Усть-Лабинск" {
		t.Fatal(got)
	}
	site := Site{Region: "Краснодарский край", Place: "Усть-Лабинск", SeismicMap: "А", SeismicPoints: "7"}
	lines := strings.Join(projectSiteLines(site, initialProject()), "\n")
	if !strings.Contains(lines, "Сейсмичность региона по карте А: 7 баллов") || strings.Contains(lines, "баллов баллов") || strings.Contains(lines, ", ,") {
		t.Fatal(lines)
	}
	s := defaultSnowCalc()
	s.JanuarySet = true
	s.JanuaryTemp = 0
	reason := snowJanuaryReductionReason(s)
	if strings.Contains(reason, "не задана") || !strings.Contains(reason, "выше −5") {
		t.Fatal(reason)
	}
	s.JanuaryTemp = -5
	if !s.ReducedApplicable() {
		t.Fatal("T=-5 must allow reduction")
	}
	s.JanuarySet = false
	if !strings.Contains(snowJanuaryReductionReason(s), "не задана") {
		t.Fatal("missing-data wording")
	}
}

func Test044WindIaOptionAlreadyAvailable(t *testing.T) {
	w, ok := windForZone("Iа")
	if !ok || math.Abs(w-.17) > 1e-9 {
		t.Fatalf("wind Ia: %g %t", w, ok)
	}
}

func Test044ProjectSaveReadableJSON(t *testing.T) {
	fn := filepath.Join(t.TempDir(), "empty.ebala")
	p := initialProject()
	p.SetSite(Site{Region: "Область", Place: "Площадка"})
	if e := p.Save(fn); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(fn)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(b), `"region": "Область"`) {
		t.Fatal("site not saved")
	}
}
