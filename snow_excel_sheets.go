package main

import (
	"fmt"
	"math"
	"strings"
)

// The geometry and every load arrangement have their own worksheet.
// This is added to the ordinary workbook as well as the standalone snow workbook.
func snowExcelAddFigureSheets(entries map[string]string, s SnowCalc) error {
	type figure struct {
		title string
		data  []byte
	}
	roof, err := snowRoofImage(s)
	if err != nil {
		return err
	}
	figs := []figure{{"Схема покрытия", roof}}
	for i, c := range s.Cases {
		p, e := snowImage(s, c)
		if e != nil {
			return e
		}
		figs = append(figs, figure{fmt.Sprintf("Эпюра %d", i+1), p})
		if len(c.Cross) > 0 {
			cross := c
			cross.Intervals = c.Cross
			cross.Cross = nil
			cross.Title = "Поперечная эпюра"
			if s.Shape == "sphere_cone" {
				cross.Title = "Сечение по зонам 2–4–4–2"
			}
			image, er := snowImage(s, cross)
			if er != nil {
				return er
			}
			figs = append(figs, figure{fmt.Sprintf("Поперечная %d", i+1), image})
		}
	}
	types := entries["[Content_Types].xml"]
	books := entries["xl/workbook.xml"]
	bookRels := entries["xl/_rels/workbook.xml.rels"]
	// The base workbook already owns sheet1 and drawing1 (if used by other modules).
	if !strings.Contains(types, `<Default Extension="png"`) {
		types = strings.Replace(types, `</Types>`, `<Default Extension="png" ContentType="image/png"/></Types>`, 1)
	}
	for i, f := range figs {
		sid := i + 2
		did := i + 2
		sheeti := fmt.Sprintf("sheet%d.xml", sid)
		drawi := fmt.Sprintf("drawing%d.xml", did)
		media := fmt.Sprintf("snow_figure_%d.png", i)
		entries["xl/media/"+media] = string(f.data)
		entries["xl/worksheets/"+sheeti] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheetFormatPr defaultRowHeight="21"/><cols><col min="1" max="12" width="17" customWidth="1"/></cols><sheetData>` + tempExcelRow(1, "На этом листе показана графическая схема: "+f.title) + tempExcelRow(2, "Все числовые значения приведены в кПа. Размеры в метрах.") + `</sheetData><drawing r:id="rId1"/></worksheet>`
		entries["xl/worksheets/_rels/"+sheeti+".rels"] = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/drawing" Target="../drawings/` + drawi + `"/></Relationships>`
		entries["xl/drawings/"+drawi] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><xdr:wsDr xmlns:xdr="http://schemas.openxmlformats.org/drawingml/2006/spreadsheetDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><xdr:twoCellAnchor editAs="oneCell"><xdr:from><xdr:col>1</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>3</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:from><xdr:to><xdr:col>12</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>29</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:to><xdr:pic><xdr:nvPicPr><xdr:cNvPr id="1" name="Снеговая нагрузка"/><xdr:cNvPicPr/></xdr:nvPicPr><xdr:blipFill><a:blip r:embed="rId1"/><a:stretch><a:fillRect/></a:stretch></xdr:blipFill><xdr:spPr><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></xdr:spPr></xdr:pic><xdr:clientData/></xdr:twoCellAnchor></xdr:wsDr>`
		entries["xl/drawings/_rels/"+drawi+".rels"] = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/` + media + `"/></Relationships>`
		types = strings.Replace(types, "</Types>", fmt.Sprintf(`<Override PartName="/xl/worksheets/%s" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/drawings/%s" ContentType="application/vnd.openxmlformats-officedocument.drawing+xml"/></Types>`, sheeti, drawi), 1)
		books = strings.Replace(books, "</sheets>", fmt.Sprintf(`<sheet name="%s" sheetId="%d" r:id="rId%d"/></sheets>`, f.title, sid, sid), 1)
		bookRels = strings.Replace(bookRels, "</Relationships>", fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/%s"/></Relationships>`, sid, sheeti), 1)
	}
	// One complete numerical sheet per spatial variant, including variant 1.
	// Each displayed load is a formula referring to the computed mu and user
	// coefficients. Angular and radial grids are not thinned for Excel.
	nextSid := len(figs) + 2
	for caseIndex, c := range s.Cases {
		if len(c.Polar) == 0 {
			continue
		}
		sid := nextSid
		nextSid++
		var body strings.Builder
		title := fmt.Sprintf("Полярная сетка Б.15а — вариант %d", caseIndex+1)
		body.WriteString(tempExcelRow(1, title))
		body.WriteString(tempExcelRow(2, "Sg, кПа", xnum(s.Ground), "ce", xnum(s.CE), "ct", xnum(s.CT), "γf", xnum(s.Gamma)))
		body.WriteString(tempExcelRow(3, "β, град", "От края, м", "Радиус r, м", "Зона", "μ по СП", "Sн=μ·Sg·ce·ct", "Sр=γf·Sн", "Основание μ"))
		for i, pt := range c.Polar {
			row := i + 4
			zone := "—"
			if pt.Zone > 0 {
				zone = fmt.Sprintf("%d", pt.Zone)
			}
			remark := "Таблица Б.3: местный уклон"
			if caseIndex > 0 {
				remark = fmt.Sprintf("Б.12(б), зона %d", pt.Zone)
			}
			body.WriteString(tempExcelRow(row, xnum(pt.Beta), xnum(s.Span/2-pt.Radius), xnum(pt.Radius), zone, xnum(pt.Mu), fmt.Sprintf("=E%d*$B$2*$D$2*$F$2", row), fmt.Sprintf("=F%d*$H$2", row), remark))
		}
		name := fmt.Sprintf("sheet%d.xml", sid)
		entries["xl/worksheets/"+name] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetFormatPr defaultRowHeight="19"/><cols><col min="1" max="8" width="26" customWidth="1"/></cols><sheetData>` + body.String() + `</sheetData></worksheet>`
		types = strings.Replace(types, "</Types>", fmt.Sprintf(`<Override PartName="/xl/worksheets/%s" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`, name), 1)
		books = strings.Replace(books, "</sheets>", fmt.Sprintf(`<sheet name="Сетка Б.15а вар.%d" sheetId="%d" r:id="rId%d"/></sheets>`, caseIndex+1, sid, sid), 1)
		bookRels = strings.Replace(bookRels, "</Relationships>", fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/%s"/></Relationships>`, sid, name), 1)
	}

	entries["[Content_Types].xml"] = types
	entries["xl/workbook.xml"] = books
	entries["xl/_rels/workbook.xml.rels"] = bookRels
	return nil
}

func snowConeFactor(alpha float64) float64 {
	if alpha <= 30 {
		return 2.15 * math.Cbrt(alpha*math.Pi/180)
	}
	return 1.7 * (30 / alpha)
}
