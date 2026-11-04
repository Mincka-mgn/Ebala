package main

import (
	"fmt"
	"strings"
)

// A project may contain several calculations of the same load type. The first
// worksheet is a navigation point, not a shadow copy of calculated figures.
// Original accepted numbers live on the individual *_source worksheets.
func excelPhase3InputIndex(entries map[string]string, r TempReport, regular, transport, cranes []TempSelection) error {
	// The legacy single-snow export owns "Исходные данные". Creating a
	// second sheet of that name would make the XLSX invalid. The newer
	// managed project format uses numbered source sheets and gets this index.
	if r.Snow != nil {
		return nil
	}
	var b strings.Builder
	b.WriteString(tempExcelInputRow(1, "На этом листе показано, где находятся принятые исходные данные всех включённых загружений объекта. Для редактирования параметров откройте соответствующий лист исходных данных; вычисления выполняются на расчётных листах Excel."))
	b.WriteString(tempExcelInputRow(2, "Раздел", "Лист с исходными данными", "Количество элементов / примечание"))
	row := 3
	add := func(section, sheet, note string) {
		b.WriteString(tempExcelInputRow(row, section, sheet, note))
		row++
	}
	if r.Site != nil {
		add("Местность", "Местность - исходные", "Климатические исходные параметры")
	}
	if n := len(weightReportBlocks(r)); n > 0 {
		add("Собственный вес", "Собственный вес - исходные", fmt.Sprintf("Конструкций: %d", n))
	}
	if len(regular) > 0 {
		add("Временные", "Временные - исходные", fmt.Sprintf("Загружений: %d", len(regular)))
	}
	if len(transport) > 0 {
		add("Транспорт", "Транспорт - исходные", fmt.Sprintf("Схем: %d", len(transport)))
	}
	if sources := excelPhase3CraneSources(r, cranes); len(sources) > 0 {
		add("Мостовые краны", "Краны - паспорт / Краны - исходные", fmt.Sprintf("Колёс: %d", len(sources)))
	}
	placements := 0
	for _, group := range r.CraneGroups {
		placements += len(group.Placements)
	}
	if placements > 0 {
		add("Расстановки мостовых кранов", "Р01 Вход / Р01 Балка / Р01 Колёса / Р01 Итоги (далее Р02…)",
			fmt.Sprintf("Независимых расчётов: %d. Численно найдены только положения колёс", placements))
	}
	if r.Snow != nil || len(r.Snows) > 0 {
		add("Снег", "Исходные данные / Исходные данные 1…", "Один набор исходных данных на загружение")
	}
	if r.Ice != nil || len(r.Ices) > 0 {
		add("Гололёд", "Гололёд - исходные / с номером загружения", "Один набор исходных данных на загружение")
	}
	if row == 3 {
		add("Общие сведения", "Нагрузки", "В ведомости отсутствуют расчёты из поддерживаемых модулей")
	}
	_, err := snowExcelSheet(entries, "Исходные данные", b.String(), false)
	return err
}
