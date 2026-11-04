package main

import (
	"fmt"
	"math"
	"strings"
)

func snowExcelAddAuditSheet(entries map[string]string, s SnowCalc) {
	wb := entries["xl/workbook.xml"]
	rels := entries["xl/_rels/workbook.xml.rels"]
	types := entries["[Content_Types].xml"]
	sid := strings.Count(wb, `<sheet `) + 1
	name := fmt.Sprintf("sheet%d.xml", sid)
	var b strings.Builder
	add := func(row int, values ...string) { b.WriteString(tempExcelRow(row, values...)) }
	add(1, "ПРОВЕРКА КОЭФФИЦИЕНТОВ (СП 20.13330.2016)")
	add(2, "Покрытие", s.ReportName(), "Основание", "Раздел 10 и приложение Б СП 20.13330.2016")
	add(3, "Sg, кПа", xnum(s.Ground), "γf", xnum(s.Gamma))
	add(4, "Уклон α, °", xnum(s.Angle), "Подъём h/f, м", xnum(s.Rise))
	if s.JanuarySet {
		add(5, "T января, °C", xnum(s.JanuaryTemp), "V холодного периода, м/с", xnum(s.WinterWind))
	} else {
		add(5, "T января, °C", "не задана", "V холодного периода, м/с", "не задана")
	}
	add(6, "H конька, м", xnum(s.RoofHeight), "Тип местности", s.Terrain)
	length := s.Span
	if s.Shape == "multi_gable" || s.Shape == "multi_vault" {
		length *= float64(s.Count)
	}
	add(7, "Размер в плане L, м", xnum(length))
	add(8, "Размер в плане B, м", xnum(s.Width))
	eligible := s.JanuarySet && s.JanuaryTemp <= -5 && s.WinterWindSet && s.WinterWind > 3 && s.RoofHeight > 0 && (s.Terrain == "А" || s.Terrain == "В")
	switch s.Shape {
	case "flat", "mono", "gable", "vault", "multi_gable", "multi_vault":
	default:
		eligible = false
	}
	if (s.Shape == "mono" || s.Shape == "gable" || s.Shape == "multi_gable") && s.Angle > 10 {
		eligible = false
	}
	if (s.Shape == "vault" || s.Shape == "multi_vault") && s.Rise/s.Span > .05 {
		eligible = false
	}
	eligibleN := "0"
	why := "Ветер не уменьшает нормативную нагрузку для заданных условий"
	if eligible {
		eligibleN = "1"
		why = "Все условия применения снижения выполнены"
	}
	add(9, "Допустимо снижение ce", eligibleN, "Причина", why)
	heights := []float64{5, 10, 20, 40, 60, 80, 100, 150, 200, 250, 300}
	kA := []float64{.75, 1, 1.25, 1.5, 1.7, 1.85, 2, 2.25, 2.45, 2.65, 2.75}
	kB := []float64{.5, .65, .85, 1.1, 1.3, 1.45, 1.6, 1.9, 2.1, 2.3, 2.5}
	ks := kB
	if s.Terrain == "А" {
		ks = kA
	}
	hi := 0
	for hi < len(heights)-1 && heights[hi] < s.RoofHeight {
		hi++
	}
	lo := hi
	if hi > 0 {
		lo = hi - 1
	}
	if s.RoofHeight <= 5 {
		lo = 0
		hi = 0
	}
	if s.RoofHeight >= 300 {
		lo = len(heights) - 1
		hi = lo
	}
	add(10, "Высота Hниже (табл. 11.2)", xnum(heights[lo]), "Hвыше", xnum(heights[hi]))
	add(11, "kниже (табл. 11.2)", xnum(ks[lo]), "kвыше", xnum(ks[hi]))
	add(12, "lc, м (формула 10.3)", "=MIN(100,2*MIN(B7,B8)-MIN(B7,B8)^2/MAX(B7,B8))")
	add(13, "k на высоте (таблица 11.2)", "=IF(D10=B10,B11,B11+(B6-B10)*(D11-B11)/(D10-B10))")
	kv := 1.4
	if s.JanuaryTemp < -25 {
		if s.Terrain == "А" {
			kv = 1.3
		} else {
			kv = 1.4
		}
	} else if s.JanuaryTemp < -15 {
		if s.Terrain == "А" {
			kv = 1.4
		} else {
			kv = 1.4
		}
	} else {
		kv = 1.4
	}
	windCol := 0
	if s.WinterWind > 6 {
		windCol = 2
	} else if s.WinterWind > 4 {
		windCol = 1
	}
	tr := 0
	if s.JanuaryTemp < -25 {
		tr = 2
	} else if s.JanuaryTemp < -15 {
		tr = 1
	}
	kva := [][]float64{{1.4, 1.3, 1.3}, {1.4, 1.3, 1.2}, {1.3, 1.2, 1.2}}
	kvb := [][]float64{{1.4, 1.4, 1.3}, {1.4, 1.4, 1.3}, {1.4, 1.3, 1.2}}
	if s.Terrain == "А" {
		kv = kva[tr][windCol]
	} else {
		kv = kvb[tr][windCol]
	}
	add(14, "kv (таблица 10.2)", xnum(kv), "Выбор по T января, V зимы, местности")
	if s.Shape == "cone" || s.Shape == "dome" {
		add(15, "ce, расчёт по п. 10.8", "=IF(B5> -5,1,IF(B7<=60,0.85,IF(B7>=100,1,0.85+0.00375*(B7-60))))", "ce в модели", xnum(s.CE))
	} else {
		add(15, "ce (п. 10.7)", "=IF(B9=0,1,MIN(1,MAX(0.5,(B14-0.4*SQRT(B13))*(0.8+0.002*B12))))", "ce в модели", xnum(s.CE))
	}
	warm := 0
	if s.WarmRoof {
		warm = 1
	}
	grad := snowRoofGradient(s)
	gradientOK := 0
	if grad > .03 {
		gradientOK = 1
	}
	add(16, "Признак крыши с тепловыделением", xnum(float64(warm)), "Уклон больше 3%", xnum(float64(gradientOK)))
	add(17, "ct (п. 10.10)", "=IF(AND(B16=1,D16=1),0.8,1)", "ct в модели", xnum(s.CT))
	add(18, "Пониженная (п. 10.11)", snowReducedLabelCommon(s))
	add(19, "Снеговая нагрузка", "Sн = μ × ce × ct × Sg", "Расчётная", "Sр = γf × Sн")
	add(20, "Формулы интерполяции k, lc и ce являются редактируемыми. μ по конкретной схеме раскрывается в листе «Нагрузки».")
	_ = math.Pi
	entries["xl/worksheets/"+name] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetFormatPr defaultRowHeight="22"/><cols><col min="1" max="1" width="38" customWidth="1"/><col min="2" max="2" width="40" customWidth="1"/><col min="3" max="3" width="23" customWidth="1"/><col min="4" max="4" width="58" customWidth="1"/></cols><sheetData>` + b.String() + `</sheetData></worksheet>`
	wb = strings.Replace(wb, "</sheets>", fmt.Sprintf(`<sheet name="Коэффициенты" sheetId="%d" r:id="rId%d"/></sheets>`, sid, sid), 1)
	rels = strings.Replace(rels, "</Relationships>", fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/%s"/></Relationships>`, sid, name), 1)
	types = strings.Replace(types, "</Types>", fmt.Sprintf(`<Override PartName="/xl/worksheets/%s" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`, name), 1)
	entries["xl/workbook.xml"] = wb
	entries["xl/_rels/workbook.xml.rels"] = rels
	entries["[Content_Types].xml"] = types
}
