package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Every saved crane placement is a separate four-sheet calculation. Only
// locations found by the numerical optimizer and the accepted spans/wheel
// forces are source cells. Influence ordinates, reactions and final efforts
// are recalculated inside the combined Excel workbook.
func excelPhase4PlacementPrefix(index int) string {
	return fmt.Sprintf("Р%02d", index)
}

func excelPhase4PlacementSheetName(index int, kind string) string {
	return excelPhase4PlacementPrefix(index) + " " + kind
}

// The standalone exporter provides style IDs from a different template.
// Strip these IDs before snowExcelSheet applies the general-workbook styles;
// otherwise the generated <c> would have TWO s= attributes and invalid XML.
var phase4CellStyleRX = regexp.MustCompile(`(<c r="[^"]+") s="[0-9]+"`)

// Existing audit sheets deliberately use the short local names
// "Исходные данные", "Балка", "Колёса". Rewrite only quoted worksheet
// references, preserving numeric constants and every calculation expression.
func excelPhase4PlacementRename(body string, index int) string {
	return strings.NewReplacer(
		// The standalone sheet XML already escapes apostrophes. A matcher
		// searching only for a literal quote silently leaves broken references.
		"&#39;Исходные данные&#39;!", "&#39;"+excelPhase4PlacementSheetName(index, "Вход")+"&#39;!",
		"&#39;Балка&#39;!", "&#39;"+excelPhase4PlacementSheetName(index, "Балка")+"&#39;!",
		"&#39;Колёса&#39;!", "&#39;"+excelPhase4PlacementSheetName(index, "Колёса")+"&#39;!",
		"'Исходные данные'!", "'"+excelPhase4PlacementSheetName(index, "Вход")+"'!",
		"'Балка'!", "'"+excelPhase4PlacementSheetName(index, "Балка")+"'!",
		"'Колёса'!", "'"+excelPhase4PlacementSheetName(index, "Колёса")+"'!",
	).Replace(body)
}

func excelPhase4CranePlacementSheets(entries map[string]string, r TempReport) error {
	index := 0
	for _, group := range r.CraneGroups {
		for _, placement := range group.Placements {
			index++
			sheets, err := cranePlacementAuditSheets(placement.Result)
			if err != nil {
				return fmt.Errorf("расстановка %q: %w", placement.Title, err)
			}
			items := []struct{ kind, body string }{
				{"Вход", sheets.Inputs},
				{"Балка", sheets.Beam},
				{"Колёса", sheets.Detail},
				{"Итоги", sheets.Summary},
			}
			for _, item := range items {
				body := excelPhase4PlacementRename(item.body, index)
				// The general XLSX style catalogue owns styles 0..2.
				// All standalone style IDs are removed before the workbook style pass.
				body = phase4CellStyleRX.ReplaceAllString(body, "$1")
				if _, err := snowExcelSheet(entries,
					excelPhase4PlacementSheetName(index, item.kind), body, false); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
