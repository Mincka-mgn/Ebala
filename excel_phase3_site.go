package main

import (
	"strings"
)

// General report requires the accepted Site snapshot, not textual strings
// containing already calculated climate numbers. Full normative tables are
// intentionally not repeated here: the independent snow/ice modules own them.
func excelPhase3SiteInputData(site *Site) string {
	if site == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(tempExcelInputRow(1, "На этом листе приведены принятые пользователем географические и климатические данные. Значения являются входами, а не результатами расчёта Excel; коэффициенты снега и гололёда вычисляются на своих расчётных листах."))
	b.WriteString(tempExcelInputRow(2, "Параметр", "Принятое значение", "Единица", "Источник / комментарий"))
	type data struct{ caption, value, unit, source string }
	snowInput, elevationInput := "", ""
	if site.SnowGround != 0 {
		snowInput = xnum(site.SnowGround)
	}
	if site.ElevationKnown || site.ElevationRaw != "" || site.ElevationM != 0 {
		elevationInput = xnum(site.ElevationM)
	}
	var rows = []data{
		{"Субъект РФ", site.Region, "", ""},
		{"Район", site.District, "", ""},
		{"Населённый пункт", site.Place, "", ""},
		{"Снеговой район", site.SnowZone, "", site.SnowNote},
		{"Принятое Sg (ручной ввод)", snowInput, "кПа", site.SnowNote},
		{"Ветровой район", site.WindZone, "", site.Reference},
		{"Гололёдный район", site.IceZone, "", site.Reference},
		{"Горный район (идентификатор)", site.MountainID, "", ""},
		{"Абсолютная отметка", elevationInput, "м", site.ElevationSource},
		{"Исходная запись отметки", site.ElevationRaw, "", ""},
		{"Сейсмическая карта", site.SeismicMap, "", site.SeismicRefPlace},
		{"Сейсмичность", site.SeismicPoints, "баллов", ""},
	}
	if site.ClimateJanuarySet {
		rows = append(rows, data{"Температура января", xnum(site.ClimateJanuary), "°C", site.ClimateOrigin})
	}
	if site.ClimateWindSet {
		rows = append(rows, data{"Ветер холодного периода", xnum(site.ClimateWind), "м/с", site.ClimateOrigin})
	}
	for i, row := range rows {
		b.WriteString(tempExcelInputRow(i+3, row.caption, row.value, row.unit, row.source))
	}
	return b.String()
}

func excelPhase3SiteSheets(entries map[string]string, site *Site) error {
	if site == nil {
		return nil
	}
	_, err := snowExcelSheet(entries, "Местность - исходные", excelPhase3SiteInputData(site), false)
	return err
}
