package main

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestWind064StructuralDampingReference(t *testing.T) {
	windTestData(t)
	db, err := getWindDB()
	if err != nil {
		t.Fatal(err)
	}
	table := db.Tables["11.5"]
	for code, ix := range map[string]int{"steel": 0, "mixed": 1, "rc": 2} {
		want, err := numberCell(table.Rows[ix][0])
		if err != nil {
			t.Fatal(err)
		}
		if got := windStructureDamping(code); got != want {
			t.Fatalf("%s: δ %v vs СП %v", code, got, want)
		}
	}
}

func wind064ReadZip(t *testing.T, p string) map[string]string {
	t.Helper()
	z, err := zip.OpenReader(p)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	m := make(map[string]string, len(z.File))
	for _, f := range z.File {
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		buf := make([]byte, f.UncompressedSize64)
		n := 0
		for n < len(buf) {
			c, e := r.Read(buf[n:])
			n += c
			if e != nil && n < len(buf) {
				t.Fatal(e)
			}
		}
		r.Close()
		m[f.Name] = string(buf)
	}
	return m
}

func TestWind064ReportAcceptance(t *testing.T) {
	windTestData(t)
	for _, pulse := range []string{"mean", "nonresonant"} {
		t.Run(pulse, func(t *testing.T) {
			in := testRoofCaseBase(20)
			in.PulseMode = pulse
			calc, err := CalculateWindVariants(in)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			rpt := TempReport{Version: 1, Winds: []WindCalculation{calc}, WindTitles: []string{"Испытательная кровля"}}
			x := filepath.Join(dir, "wind.xlsx")
			d := filepath.Join(dir, "wind.docx")
			if err := exportTempExcel(rpt, x); err != nil {
				t.Fatal(err)
			}
			if err := exportTempWord(rpt, d); err != nil {
				t.Fatal(err)
			}
			files := wind064ReadZip(t, x)
			wb := files["xl/workbook.xml"]
			if !strings.HasPrefix(wb[strings.Index(wb, "<sheets>")+8:], `<sheet name="Исходные данные"`) {
				t.Fatal("Оглавление не первый лист")
			}
			if strings.Contains(wb, `name="Нагрузки"`) {
				t.Fatal("Непонятное имя итогового листа")
			}
			if !strings.Contains(wb, `name="Итоговые нагрузки"`) {
				t.Fatal("Нет итогового листа")
			}
			if strings.Index(wb, `name="СП Ветер 11.1"`) < strings.Index(wb, `name="Ветер Эпюра 1"`) {
				t.Fatal("СП не в конце книги")
			}
			if len(regexp.MustCompile(`<sheet name=`).FindAllString(wb, -1)) != 82 {
				t.Fatal("Неожиданное количество листов")
			}
			for _, co := range []string{"FFE44C4C", "FF2D79BF", "FF46A86A", "FF888888"} {
				found := false
				for path, xml := range files {
					if strings.HasPrefix(path, "xl/worksheets/") && strings.Contains(xml, `tabColor rgb="`+co+`"`) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Нет цвета %s", co)
				}
			}
			idxFound, geometryFound, dimensionsFound, technicalFound := false, false, false, false
			for path, xml := range files {
				if !strings.HasPrefix(path, "xl/worksheets/") {
					continue
				}
				if strings.Contains(xml, "Расчётных зон и знаковых вариантов: 23") && strings.Contains(xml, "HYPERLINK") {
					idxFound = true
				}
				if strings.Contains(xml, "Ширина здания L, м") && strings.Contains(xml, "Длина здания B, м") && strings.Contains(xml, `<v>30</v>`) {
					geometryFound = true
				}
				if strings.Contains(xml, "Ширина зоны поперёк ветра") && strings.Contains(xml, "Протяжённость зоны вдоль ветра") && strings.Contains(xml, "MIN(") {
					dimensionsFound = true
				}
				if strings.Contains(xml, "исторический режим 0.59") || strings.Contains(xml, "wo назначено вручную") || strings.Contains(xml, "w0 назначено вручную") {
					technicalFound = true
				}
			}
			if !idxFound || !geometryFound || !dimensionsFound || technicalFound {
				t.Errorf("Оглавление=%v геометрия=%v размеры зон=%v технические записи=%v", idxFound, geometryFound, dimensionsFound, technicalFound)
			}
			word := wind064ReadZip(t, d)["word/document.xml"]
			if strings.Contains(word, "одинаковых участков") || !strings.Contains(word, "ширина одного участка") {
				t.Fatal("Заголовок зоны F нарушает заданный формат")
			}
			if pulse == "mean" {
				// The user did not select pulsation: no ζ,ν,δ calculations or structural options.
				for path, xml := range files {
					if strings.HasPrefix(path, "xl/worksheets/") && strings.Contains(xml, "На этом листе выполнен независимый расчёт") {
						for _, q := range []string{"Нижний индекс ρ", "Логарифмический декремент колебаний", "Коэффициент пространственной корреляции ν"} {
							if strings.Contains(xml, q) {
								t.Errorf("Без пульсации остался расчёт %q в %s", q, path)
							}
						}
					}
				}
			}
			_ = fmt.Sprint
			_, _ = os.Stat(d)
		})
	}
}
