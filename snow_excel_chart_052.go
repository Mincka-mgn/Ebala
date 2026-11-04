package main

import (
	"fmt"
	"strings"
)

// snowExcelLiveChart places a native Excel XY diagram referencing calculated
// loads. A change to the inputs updates the diagram after spreadsheet recalc.
// No cached Go loads or embedded static epure are written to this chart.
func snowExcelLiveChart(entries map[string]string, variant, chartSheet string, count int) error {
	if count < 1 {
		return nil
	}
	body := tempExcelRow(1, "На этом листе построены живые эпюры снеговой нагрузки по формулам листа «"+variant+"». Они пересчитываются при изменении исходных данных.") +
		tempExcelRow(2, "X: горизонтальная координата, м; Y: нагрузка, кПа") +
		tempExcelRow(3, "Нормативная и расчётная эпюры связаны с вычисляемыми ячейками, а не с готовым PNG.")
	id, err := snowExcelSheet(entries, chartSheet, body, false)
	if err != nil {
		return err
	}
	part := fmt.Sprintf("xl/worksheets/snow%d.xml", id)
	entries[part] = strings.Replace(entries[part], "</worksheet>", `<drawing r:id="rId1"/></worksheet>`, 1)
	dn := 1
	for {
		if _, exists := entries[fmt.Sprintf("xl/drawings/drawing%d.xml", dn)]; !exists {
			break
		}
		dn++
	}
	cn := 1
	for {
		if _, exists := entries[fmt.Sprintf("xl/charts/chart%d.xml", cn)]; !exists {
			break
		}
		cn++
	}
	drawing := fmt.Sprintf("drawing%d.xml", dn)
	chart := fmt.Sprintf("chart%d.xml", cn)
	entries[fmt.Sprintf("xl/worksheets/_rels/snow%d.xml.rels", id)] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/drawing" Target="../drawings/` + drawing + `"/></Relationships>`
	entries["xl/drawings/"+drawing] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><xdr:wsDr xmlns:xdr="http://schemas.openxmlformats.org/drawingml/2006/spreadsheetDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:c="http://schemas.openxmlformats.org/drawingml/2006/chart"><xdr:twoCellAnchor><xdr:from><xdr:col>0</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>5</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:from><xdr:to><xdr:col>18</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>40</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:to><xdr:graphicFrame macro=""><xdr:nvGraphicFramePr><xdr:cNvPr id="2" name="Эпюра снеговых нагрузок"/><xdr:cNvGraphicFramePr/></xdr:nvGraphicFramePr><xdr:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/></xdr:xfrm><a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/chart"><c:chart r:id="rId1"/></a:graphicData></a:graphic></xdr:graphicFrame><xdr:clientData/></xdr:twoCellAnchor></xdr:wsDr>`
	entries["xl/drawings/_rels/"+drawing+".rels"] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/chart" Target="../charts/` + chart + `"/></Relationships>`
	var series strings.Builder
	for i, desc := range []struct{ title, col, color string }{{"Нормативная, кПа", "D", "164B74"}, {"Расчётная, кПа", "E", "B53B3B"}} {
		xrange := fmt.Sprintf("'%s'!$H$5:$H$%d", variant, count+4)
		yrange := fmt.Sprintf("'%s'!$%s$5:$%s$%d", variant, desc.col, desc.col, count+4)
		fmt.Fprintf(&series, `<c:ser><c:idx val="%d"/><c:order val="%d"/><c:tx><c:v>%s</c:v></c:tx><c:spPr><a:ln w="25000"><a:solidFill><a:srgbClr val="%s"/></a:solidFill></a:ln></c:spPr><c:marker><c:symbol val="none"/></c:marker><c:xVal><c:numRef><c:f>%s</c:f></c:numRef></c:xVal><c:yVal><c:numRef><c:f>%s</c:f></c:numRef></c:yVal><c:smooth val="0"/></c:ser>`, i, i, tempXML(desc.title), desc.color, tempXML(xrange), tempXML(yrange))
	}
	chartXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><c:chartSpace xmlns:c="http://schemas.openxmlformats.org/drawingml/2006/chart" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><c:lang val="ru-RU"/><c:chart><c:autoTitleDeleted val="1"/><c:plotArea><c:layout/><c:scatterChart><c:scatterStyle val="line"/><c:varyColors val="0"/>` + series.String() + `<c:axId val="620051"/><c:axId val="620052"/></c:scatterChart><c:valAx><c:axId val="620051"/><c:scaling><c:orientation val="minMax"/></c:scaling><c:axPos val="b"/><c:majorGridlines/><c:crossAx val="620052"/><c:crosses val="autoZero"/><c:crossBetween val="between"/></c:valAx><c:valAx><c:axId val="620052"/><c:scaling><c:orientation val="minMax"/></c:scaling><c:axPos val="l"/><c:majorGridlines/><c:crossAx val="620051"/><c:crosses val="autoZero"/><c:crossBetween val="between"/></c:valAx></c:plotArea><c:legend><c:legendPos val="b"/><c:layout/></c:legend><c:plotVisOnly val="1"/></c:chart></c:chartSpace>`
	entries["xl/charts/"+chart] = chartXML
	types := entries["[Content_Types].xml"]
	types = strings.Replace(types, "</Types>", fmt.Sprintf(`<Override PartName="/xl/drawings/%s" ContentType="application/vnd.openxmlformats-officedocument.drawing+xml"/><Override PartName="/xl/charts/%s" ContentType="application/vnd.openxmlformats-officedocument.drawingml.chart+xml"/></Types>`, drawing, chart), 1)
	entries["[Content_Types].xml"] = types
	return nil
}
