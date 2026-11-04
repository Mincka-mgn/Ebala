package main

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

func transportImageData(key string) ([]byte, int, int, error) {
	file := filepath.Join("Resources", "Transport", key+".png")
	candidates := []string{filepath.Join(storageRoot(), file), file}
	for _, path := range candidates {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		conf, _, err := image.DecodeConfig(f)
		f.Close()
		if err == nil {
			return b, conf.Width, conf.Height, nil
		}
	}
	return nil, 0, 0, fmt.Errorf("нет рисунка транспортной схемы %s", key)
}
func tempWordImageInline(id int, w, h int) string {
	// Maximum width 9 inches; preserve aspect ratio (EMU dimensions).
	cx := int64(5500000)
	cy := cx * int64(h) / int64(w)
	if cy > 3100000 {
		cy = 3100000
		cx = cy * int64(w) / int64(h)
	}
	return fmt.Sprintf(`<w:p><w:r><w:drawing><wp:inline xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" distT="0" distB="0" distL="0" distR="0"><wp:extent cx="%d" cy="%d"/><wp:docPr id="%d" name="Схема транспортной нагрузки"/><a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:nvPicPr><pic:cNvPr id="0" name="Схема"/><pic:cNvPicPr/></pic:nvPicPr><pic:blipFill><a:blip xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" r:embed="rId%d"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p>`, cx, cy, id+1, id, cx, cy)
}
func tempWordTransportAppend(s TempSelection, imageNum *int, rel *strings.Builder, entries map[string]string) string {
	d := transportDetail(s)
	var b strings.Builder
	b.WriteString(pBold("Нагрузочная схема: " + d.Caption))
	widths := []int{4700, 5200}
	b.WriteString(tempTableStart(widths))
	for _, r := range d.Rows {
		b.WriteString(tempTableRow([]string{r[0], r[1]}, widths))
	}
	b.WriteString(`</w:tbl>`)
	for _, r := range d.Rules {
		b.WriteString(pText(r))
	}
	for _, key := range d.Figures {
		imageBytes, w, h, e := transportImageData(key)
		if e != nil {
			b.WriteString(pText("Рисунок не найден в комплекте программы"))
			continue
		}
		*imageNum++
		id := *imageNum
		path := fmt.Sprintf("word/media/transport_%d.png", id)
		entries[path] = string(imageBytes)
		fmt.Fprintf(rel, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/transport_%d.png"/>`, id, id)
		b.WriteString(tempWordImageInline(id, w, h))
		figCaption := map[string]string{"AK": "Рисунок 6.1а СП 35.13330.2011. Автомобильная нагрузка АК.", "NK": "Рисунок 6.1б СП 35.13330.2011. Тяжёлая нагрузка НК.", "Metro": "Рисунок 6.1в СП 35.13330.2011. Поезд метрополитена.", "Tram": "Рисунок 6.1г СП 35.13330.2011. Поезд трамвая."}[key]
		b.WriteString(pText(figCaption))
	}
	b.WriteString(pText("Нормативное основание: " + d.Source + ". Коэффициенты: п. 6.23, таблицы 6.9 и 6.10 по виду нагрузки."))
	return b.String()
}
func tempExcelTransportDetail(s TempSelection, add func(...string), row *int, imageNum *int, rels *strings.Builder, drawing *strings.Builder, entries map[string]string) {
	d := transportDetail(s)
	add("Подробные сведения: " + d.Caption)
	add("Параметр", "Значение")
	for _, p := range d.Rows {
		add(p[0], p[1])
	}
	for _, rule := range d.Rules {
		add("Условие применения", rule)
	}
	add("Нормативное основание", d.Source)
	for _, key := range d.Figures {
		bytes, w, h, e := transportImageData(key)
		if e != nil {
			continue
		}
		*imageNum++
		id := *imageNum
		entries[fmt.Sprintf("xl/media/transport_%d.png", id)] = string(bytes)
		fmt.Fprintf(rels, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/transport_%d.png"/>`, id, id)
		start := *row + 1
		spanRows := 12
		if h > 500 {
			spanRows = 18
		}
		// SpreadsheetDrawingML: images belong to this individual transport card.
		fmt.Fprintf(drawing, `<xdr:twoCellAnchor editAs="oneCell"><xdr:from><xdr:col>1</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>%d</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:from><xdr:to><xdr:col>6</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>%d</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:to><xdr:pic><xdr:nvPicPr><xdr:cNvPr id="%d" name="Схема СП 35"/><xdr:cNvPicPr/></xdr:nvPicPr><xdr:blipFill><a:blip r:embed="rId%d"/><a:stretch><a:fillRect/></a:stretch></xdr:blipFill><xdr:spPr><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></xdr:spPr></xdr:pic><xdr:clientData/></xdr:twoCellAnchor>`, start, start+spanRows, id, id)
		_ = w
		for j := 0; j < spanRows+2; j++ {
			add("")
		}
	}
	add("")
}
