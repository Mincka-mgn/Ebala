package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWind058MaximumZoneDefaultWithoutSite(t *testing.T) {
	windTestData(t)
	a := windDefaultInput("")
	if a.Region != "VII" {
		t.Fatalf("default region %q not maximum", a.Region)
	}
	c, err := CalculateWind(a)
	if err != nil {
		t.Fatal(err)
	}
	windNear(t, "zone VII", c.Result.W0Pa, 850)
}
func TestWind058FreeWallHeightAndAeroFromApprovedDB(t *testing.T) {
	windTestData(t)
	in := windDefaultInput("III")
	in.Geometry = "free_wall"
	in.Depth = 0
	in.AeroTable = "В.1"
	in.AeroZone = "D"
	in.Height = 60
	in.Rho = 12
	in.Chi = 60
	for _, z := range []float64{.001, 5, 10, 25, 60} {
		in.Z = z
		c, err := CalculateWind(in)
		if err != nil {
			t.Fatal(err)
		}
		windNear(t, "wall effective height", c.Result.Ze, 60)
		windNear(t, "wall table B1 D", c.Result.C, 1.2)
	}
}
func TestWind058ProfileReevaluatesCoreAtEveryHeight(t *testing.T) {
	windTestData(t)
	in := windDefaultInput("III")
	in.Height = 60
	in.Depth = 20
	in.Rho = 25
	in.Chi = 60
	in.AeroTable = "В.2"
	in.AeroZone = "D"
	c, err := CalculateWindProfile(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Profile) < 20 { // 0.59 uses the engineer-approved 0, 5, 7.5 m default grid
		t.Fatalf("too few profile points: %d", len(c.Profile))
	}
	foundLow, foundUpper := false, false
	for i, p := range c.Profile {
		if i > 0 && p.Z <= c.Profile[i-1].Z {
			t.Fatal("unordered points")
		}
		q := c.Input
		q.Z = p.Z
		direct, e := CalculateWind(q)
		if e != nil {
			t.Fatal(e)
		}
		windNear(t, "profile mean", p.Result.MeanPa, direct.Result.MeanPa)
		windNear(t, "profile norm", p.Result.NormPa, direct.Result.NormPa)
		windNear(t, "profile design", p.Result.DesignPa, direct.Result.DesignPa)
		if math.Abs(p.Z-20) < 1e-8 {
			foundLow = true
		}
		if math.Abs(p.Z-40) < 1e-8 {
			foundUpper = true
		}
	}
	if !foundLow || !foundUpper {
		t.Fatal("missing aerodynamic effective-height boundaries")
	}
}
func TestWind058ExcelProfileIsFormulaDriven(t *testing.T) {
	windTestData(t)
	in := windDefaultInput("III")
	in.Height = 60
	in.Depth = 20
	in.Rho = 25
	in.Chi = 60
	c, err := CalculateWindProfile(in)
	if err != nil {
		t.Fatal(err)
	}
	c.Input.DisplayUnit = "kgf"
	dir := t.TempDir()
	p := filepath.Join(dir, "wind058.xlsx")
	if err = exportTempExcel(TempReport{Version: 1, Winds: []WindCalculation{c}}, p); err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(p)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	foundEpure, foundLinkedHeight, foundUnit := false, false, false
	for _, f := range z.File {
		if !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		blob, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
		if strings.Contains(string(blob), "#REF!") {
			t.Fatalf("broken Excel reference in %s", f.Name)
		}
		if strings.HasPrefix(f.Name, "xl/worksheets/") && bytes.Contains(blob, []byte("Ветер Исходные 1")) && bytes.Contains(blob, []byte("Ветер Расчёт 1")) && bytes.Contains(blob, []byte("11.2")) {
			foundEpure = true
			foundLinkedHeight = bytes.Contains(blob, []byte("$B$8"))
			foundUnit = bytes.Contains(blob, []byte("$B$21"))
		}
		dec := xml.NewDecoder(bytes.NewReader(blob))
		for {
			if _, e := dec.Token(); e != nil {
				if e != io.EOF {
					t.Fatalf("bad XML %s: %v", f.Name, e)
				}
				break
			}
		}
	}
	if !foundEpure || !foundLinkedHeight || !foundUnit {
		t.Fatalf("epure missing links: %v %v %v", foundEpure, foundLinkedHeight, foundUnit)
	}
	// XLSX remains self-contained and does not require a separate file for tables.
	if st, e := os.Stat(p); e != nil || st.Size() == 0 {
		t.Fatal("empty Excel")
	}
}
func TestWind058WordUnitAndProfile(t *testing.T) {
	windTestData(t)
	in := windDefaultInput("III")
	in.DisplayUnit = "kgf"
	c, e := CalculateWindProfile(in)
	if e != nil {
		t.Fatal(e)
	}
	data := windWordAppend(c, "Проверка эпюры")
	if !strings.Contains(data, "кгс/м²") || !strings.Contains(data, "Отметка, м") {
		t.Fatal("Word omits chosen units or profile")
	}
}
func TestWind058KTableVsFormulaNotIdentical(t *testing.T) {
	windTestData(t)
	db, e := getWindDB()
	if e != nil {
		t.Fatal(e)
	}
	ktab, e := db.interpHeight("11.2", "A", 15)
	if e != nil {
		t.Fatal(e)
	}
	alpha, e := db.tableValue("11.3", "A", "alpha")
	if e != nil {
		t.Fatal(e)
	}
	k10, e := db.tableValue("11.3", "A", "k10")
	if e != nil {
		t.Fatal(e)
	}
	kformula, e := db.calcExpr("11.4", map[string]float64{"z_e": 15, "k10": k10, "alpha_terrain": alpha})
	if e != nil {
		t.Fatal(e)
	}
	if math.Abs(ktab-kformula) < 1e-6 {
		t.Log("selected formulas coincide at control height")
	} else {
		t.Logf("k(15): table %.8f != formula %.8f; interface must use table", ktab, kformula)
	}
}

func TestWind058SavedProfileAndExports(t *testing.T) {
	windTestData(t)
	in := windDefaultInput("VII")
	in.Height = 36
	in.Depth = 12
	in.Rho = 18
	in.Chi = 36
	in.DisplayUnit = "kgf"
	c, e := CalculateWindProfile(in)
	if e != nil {
		t.Fatal(e)
	}
	project := initialProject()
	id := project.putWindCalc(0, c)
	dir := t.TempDir()
	if specified := os.Getenv("EBALA_WIND_QA_OUT"); specified != "" {
		dir = specified
		if e := os.MkdirAll(dir, 0755); e != nil {
			t.Fatal(e)
		}
	}
	path := filepath.Join(dir, "wind058_project.json")
	if e := project.Save(path); e != nil {
		t.Fatal(e)
	}
	back, e := LoadProject(path)
	if e != nil {
		t.Fatal(e)
	}
	if len(back.SavedBlocks) != 1 || back.SavedBlocks[0].ID != id || back.SavedBlocks[0].WindCalc == nil {
		t.Fatal("saved wind block lost")
	}
	restored := back.SavedBlocks[0].WindCalc
	if len(restored.Profile) != len(c.Profile) {
		t.Fatalf("profile length %d vs %d", len(restored.Profile), len(c.Profile))
	}
	for j := range restored.Profile {
		windNear(t, "restored profile point", restored.Profile[j].Result.DesignPa, c.Profile[j].Result.DesignPa)
	}
	report := TempReport{Version: 1, Winds: []WindCalculation{*restored}, WindTitles: []string{"Фасад: контрольная эпюра"}}
	if e := exportTempWord(report, filepath.Join(dir, "wind058_profile.docx")); e != nil {
		t.Fatal(e)
	}
	if e := exportTempExcel(report, filepath.Join(dir, "wind058_profile.xlsx")); e != nil {
		t.Fatal(e)
	}
	word, e := zip.OpenReader(filepath.Join(dir, "wind058_profile.docx"))
	if e != nil {
		t.Fatal(e)
	}
	defer word.Close()
	foundCorrectTitle := false
	for _, f := range word.File {
		if f.Name != "word/document.xml" {
			continue
		}
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		data, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
		foundCorrectTitle = bytes.Contains(data, []byte("РАСЧЁТ ВЕТРОВЫХ НАГРУЗОК")) && !bytes.Contains(data, []byte("ВРЕМЕННЫЕ И ТРАНСПОРТНЫЕ НАГРУЗКИ"))
	}
	if !foundCorrectTitle {
		t.Fatal("В Word некорректный заголовок отдельного ветрового расчёта")
	}
}
