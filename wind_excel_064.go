package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// wind064PolishWorkbook applies the approved engineering workbook navigation
// and tab colors without changing cell values, formulas or sheet identities.
// Sheet names are changed before the worksheets and workbook are rewritten,
// so all Excel formula references continue to point to their original tables.
func wind064PolishWorkbook(entries map[string]string) error {
	wb := entries["xl/workbook.xml"]
	start, end := strings.Index(wb, "<sheets>"), strings.Index(wb, "</sheets>")
	if start < 0 || end < start {
		return fmt.Errorf("Не найден список листов Excel")
	}
	oldNames := []string{
		"Ветер 11.1", "Ветер 11.2", "Ветер 11.3", "Ветер 11.4", "Ветер 11.5", "Ветер 11.6",
		"Ветер В.1", "Ветер В.2", "Ветер В.3а 1", "Ветер В.3а 2", "Ветер В.3б",
	}
	// Change the complete quoted references and workbook sheet titles. Unicode
	// XML text nodes and Excel formulas both use the same exact sheet names.
	for _, found := range snowWorkbookSheet.FindAllString(wb, -1) {
		m := regexp.MustCompile(`name="([^"]+)"`).FindStringSubmatch(found)
		if len(m) == 2 && (strings.HasPrefix(m[1], "Таблица ") || strings.Contains(m[1], " табл.")) {
			oldNames = append(oldNames, m[1])
		}
	}
	for _, old := range oldNames {
		to := "СП " + old
		for key, val := range entries {
			if strings.HasSuffix(key, ".xml") {
				entries[key] = strings.ReplaceAll(val, old, to)
			}
		}
	}
	wb = entries["xl/workbook.xml"]
	start, end = strings.Index(wb, "<sheets>"), strings.Index(wb, "</sheets>")
	content := wb[start+len("<sheets>") : end]
	items := snowWorkbookSheet.FindAllString(content, -1)
	if len(items) == 0 || strings.TrimSpace(snowWorkbookSheet.ReplaceAllString(content, "")) != "" {
		return fmt.Errorf("Не удалось разобрать листы Excel")
	}
	nameRx := regexp.MustCompile(`name="([^"]+)"`)
	idRx := regexp.MustCompile(`r:id="(rId[0-9]+)"`)
	relRx := regexp.MustCompile(`<Relationship\s+[^>]*?/>`)
	attrID := regexp.MustCompile(`\bId="([^"]+)"`)
	attrTarget := regexp.MustCompile(`\bTarget="([^"]+)"`)
	rels := map[string]string{}
	for _, rec := range relRx.FindAllString(entries["xl/_rels/workbook.xml.rels"], -1) {
		id, target := attrID.FindStringSubmatch(rec), attrTarget.FindStringSubmatch(rec)
		if len(id) == 2 && len(target) == 2 && strings.HasPrefix(target[1], "worksheets/") {
			rels[id[1]] = "xl/" + target[1]
		}
	}
	type item struct {
		sheet, name, id, path string
		order, rank           int
	}
	sheets := make([]item, 0, len(items))
	for i, s := range items {
		name, id := nameRx.FindStringSubmatch(s), idRx.FindStringSubmatch(s)
		if len(name) != 2 || len(id) != 2 {
			return fmt.Errorf("Неверное описание листа %s", s)
		}
		path := rels[id[1]]
		if _, ok := entries[path]; path == "" || !ok {
			return fmt.Errorf("Отсутствует файл листа %s: %s", name[1], path)
		}
		rank := wind064SheetRank(name[1])
		sheets = append(sheets, item{sheet: s, name: name[1], id: id[1], path: path, order: i, rank: rank})
	}
	sort.SliceStable(sheets, func(i, j int) bool { return sheets[i].rank < sheets[j].rank })
	names := map[string]bool{}
	var ordered strings.Builder
	for _, s := range sheets {
		if names[s.name] {
			return fmt.Errorf("Дублируется лист %s", s.name)
		}
		names[s.name] = true
		ordered.WriteString(s.sheet)
		color := "FF2D79BF" // blue calculation
		switch s.rank {
		case 0, 1:
			color = "FFE44C4C" // red input or contents
		case 3:
			color = "FF46A86A" // green result
		case 4:
			color = "FF888888" // gray normative reference
		}
		xml := entries[s.path]
		pr := `<sheetPr><tabColor rgb="` + color + `"/></sheetPr>`
		if i := strings.Index(xml, "<sheetPr>"); i >= 0 {
			if j := strings.Index(xml[i:], "</sheetPr>"); j >= 0 {
				xml = xml[:i] + pr + xml[i+j+len("</sheetPr>"):]
			} else {
				return fmt.Errorf("Ошибка sheetPr %s", s.name)
			}
		} else {
			open := strings.Index(xml, "<worksheet")
			if open < 0 {
				return fmt.Errorf("Неверный XML листа %s", s.name)
			}
			close := strings.Index(xml[open:], ">")
			if close < 0 {
				return fmt.Errorf("Незакрытый XML листа %s", s.name)
			}
			at := open + close + 1
			xml = xml[:at] + pr + xml[at:]
		}
		entries[s.path] = xml
	}
	entries["xl/workbook.xml"] = wb[:start+len("<sheets>")] + ordered.String() + wb[end:]
	return nil
}

func wind064SheetRank(name string) int {
	switch {
	case name == "Исходные данные":
		return 0
	case strings.HasPrefix(name, "СП "), strings.HasPrefix(name, "Таблица "), strings.Contains(name, " табл."), strings.Contains(name, "- параметры"):
		return 4
	case strings.Contains(name, "Исходные"), strings.Contains(name, "исходные"), strings.Contains(name, " Вход"), strings.Contains(name, "паспорт"):
		return 1
	case strings.Contains(name, "Эпюра"), strings.Contains(name, "Итоги"), strings.Contains(name, "Итоговые"), strings.Contains(name, "Свод"), strings.Contains(name, "Перечень загружений"):
		return 3
	default:
		return 2
	}
}
