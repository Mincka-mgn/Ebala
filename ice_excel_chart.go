package main

import (
	"fmt"
	"strings"
)

// iceExcelChart creates an editable native OOXML XY scatter plot, not a screenshot.
// The chart refers to the same formula cells used by the checking workbook.
func iceExcelChart(entries map[string]string, s IceLoadCalc, blockIndex ...int) error {
	suffix := ""
	if len(blockIndex) > 0 && blockIndex[0] > 0 {
		suffix = fmt.Sprintf(" %d", blockIndex[0])
	}

	n := iceExcelHeightCapacity
	if n == 0 {
		return fmt.Errorf("Нет точек для Excel-эпюры")
	}
	sheetID, err := snowExcelSheet(entries, "Эпюры нагрузок"+suffix, tempExcelRow(1, "На этом листе показаны высотные эпюры гололёда и ветра; диаграммы связаны с вычисляемыми значениями расчётного листа.")+tempExcelRow(2, "Вертикальная ось — высота, м; горизонтальная — интенсивность, "+s.NormUnit())+tempExcelRow(3, "Числовые ряды связаны с листом «Гололёд - расчёт». Изменение формул обновляет диаграмму."), false)
	if err != nil {
		return err
	}
	part := fmt.Sprintf("xl/worksheets/snow%d.xml", sheetID)
	entries[part] = strings.Replace(entries[part], "</worksheet>", `<drawing r:id="rId1"/></worksheet>`, 1)
	drawingN := 1
	for {
		if _, exists := entries[fmt.Sprintf("xl/drawings/drawing%d.xml", drawingN)]; !exists {
			break
		}
		drawingN++
	}
	chartN := 1
	for {
		if _, exists := entries[fmt.Sprintf("xl/charts/chart%d.xml", chartN)]; !exists {
			break
		}
		chartN++
	}
	drawingName := fmt.Sprintf("drawing%d.xml", drawingN)
	chartName := fmt.Sprintf("chart%d.xml", chartN)
	entries[fmt.Sprintf("xl/worksheets/_rels/snow%d.xml.rels", sheetID)] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/drawing" Target="../drawings/` + drawingName + `"/></Relationships>`
	entries["xl/drawings/"+drawingName] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><xdr:wsDr xmlns:xdr="http://schemas.openxmlformats.org/drawingml/2006/spreadsheetDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:c="http://schemas.openxmlformats.org/drawingml/2006/chart"><xdr:twoCellAnchor><xdr:from><xdr:col>0</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>5</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:from><xdr:to><xdr:col>17</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>39</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:to><xdr:graphicFrame macro=""><xdr:nvGraphicFramePr><xdr:cNvPr id="2" name="Высотная эпюра гололёда и ветра"/><xdr:cNvGraphicFramePr/></xdr:nvGraphicFramePr><xdr:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/></xdr:xfrm><a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/chart"><c:chart r:id="rId1"/></a:graphicData></a:graphic></xdr:graphicFrame><xdr:clientData/></xdr:twoCellAnchor></xdr:wsDr>`
	entries["xl/drawings/_rels/"+drawingName+".rels"] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/chart" Target="../charts/` + chartName + `"/></Relationships>`
	end := n + 4
	var series strings.Builder
	addSeries := func(index int, title, xcol, color string) {
		xRange := fmt.Sprintf("'Гололёд - расчёт%s'!$%s$5:$%s$%d", suffix, xcol, xcol, end)
		yRange := fmt.Sprintf("'Гололёд - расчёт%s'!$A$5:$A$%d", suffix, end)
		// Deliberately NO c:numCache: it would duplicate Go values and remain
		// stale if user changes height, step, temperature or terrain.
		fmt.Fprintf(&series, `<c:ser><c:idx val="%d"/><c:order val="%d"/><c:tx><c:v>%s</c:v></c:tx><c:spPr><a:ln w="28575"><a:solidFill><a:srgbClr val="%s"/></a:solidFill></a:ln></c:spPr><c:marker><c:symbol val="none"/></c:marker><c:xVal><c:numRef><c:f>%s</c:f></c:numRef></c:xVal><c:yVal><c:numRef><c:f>%s</c:f></c:numRef></c:yVal><c:smooth val="0"/></c:ser>`, index, index, tempXML(title), color, tempXML(xRange), tempXML(yRange))
	}
	addSeries(0, "Гололёд нормативный", "G", "222222")
	addSeries(1, "Гололёд расчётный", "H", "C22B37")
	addSeries(2, "Ветер при гололёде", "N", "2469BB")
	addSeries(3, "Ветер ведущий", "O", "147C4A")
	textTitle := func(v string) string {
		return `<c:title><c:tx><c:rich><a:bodyPr/><a:lstStyle/><a:p><a:pPr><a:defRPr/></a:pPr><a:r><a:rPr lang="ru-RU"/><a:t>` + tempXML(v) + `</a:t></a:r><a:endParaRPr lang="ru-RU"/></a:p></c:rich></c:tx><c:overlay val="0"/></c:title>`
	}
	entries["xl/charts/"+chartName] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><c:chartSpace xmlns:c="http://schemas.openxmlformats.org/drawingml/2006/chart" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><c:lang val="ru-RU"/><c:chart>` + textTitle("Высотные эпюры гололёда и ветра") + `<c:autoTitleDeleted val="0"/><c:plotArea><c:layout/><c:scatterChart><c:scatterStyle val="line"/><c:varyColors val="0"/>` + series.String() + `<c:axId val="500001"/><c:axId val="500002"/></c:scatterChart>` + `<c:valAx><c:axId val="500001"/><c:scaling><c:orientation val="minMax"/><c:min val="0"/></c:scaling>` + textTitle("Интенсивность, "+s.NormUnit()) + `<c:axPos val="b"/><c:majorGridlines/><c:numFmt formatCode="0.00" sourceLinked="0"/><c:tickLblPos val="nextTo"/><c:crossAx val="500002"/><c:crosses val="autoZero"/><c:crossBetween val="between"/></c:valAx>` + `<c:valAx><c:axId val="500002"/><c:scaling><c:orientation val="minMax"/></c:scaling>` + textTitle("Высота, м") + `<c:axPos val="l"/><c:majorGridlines/><c:numFmt formatCode="0.0" sourceLinked="0"/><c:tickLblPos val="nextTo"/><c:crossAx val="500001"/><c:crosses val="autoZero"/><c:crossBetween val="between"/></c:valAx></c:plotArea><c:legend><c:legendPos val="b"/><c:layout/></c:legend><c:plotVisOnly val="1"/><c:dispBlanksAs val="gap"/></c:chart></c:chartSpace>`
	types := entries["[Content_Types].xml"]
	types = strings.Replace(types, "</Types>", fmt.Sprintf(`<Override PartName="/xl/drawings/%s" ContentType="application/vnd.openxmlformats-officedocument.drawing+xml"/><Override PartName="/xl/charts/%s" ContentType="application/vnd.openxmlformats-officedocument.drawingml.chart+xml"/></Types>`, drawingName, chartName), 1)
	entries["[Content_Types].xml"] = types
	return nil
}
