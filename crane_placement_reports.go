package main

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"strconv"
	"strings"
)

func craneCaseInfluence(spans []float64, c CranePlacementCase, x float64) (float64, error) {
	if c.Units != "кН·м" && c.Units != "кН" {
		return 0, fmt.Errorf("неизвестная величина %q", c.Units)
	}
	mid := []float64{c.SectionMM / 1000}
	unit := []CranePoint{{XMM: x, PKN: 1}}
	state, err := solveCraneBeam(spans, unit, mid)
	if err != nil {
		return 0, err
	}
	if c.Units == "кН·м" {
		return state.moments[0], nil
	}
	n := 0
	_, err = fmt.Sscanf(c.Name[strings.Index(c.Name, "опоры ")+len("опоры "):], "%d", &n)
	if err != nil || n < 1 || n > len(spans)+1 {
		return 0, fmt.Errorf("нет номера опоры в %s", c.Name)
	}
	if strings.Contains(c.Name, "слева от опоры") {
		return state.shearLeft[n-1], nil
	}
	if strings.Contains(c.Name, "справа от опоры") {
		return state.shearRight[n-1], nil
	}
	return 0, fmt.Errorf("неизвестное сечение: %s", c.Name)
}

// Generate an actual PNG from the same BMP renderer used by the program window.
func craneCasePNG(spans []float64, c CranePlacementCase) ([]byte, error) {
	f, err := os.CreateTemp("", "ebala-crane-*.bmp")
	if err != nil {
		return nil, err
	}
	name := f.Name()
	f.Close()
	defer os.Remove(name)
	if err = cranePictureForCase(spans, c, name); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	if len(b) < 54 {
		return nil, fmt.Errorf("повреждён рисунок")
	}
	w := int(binary.LittleEndian.Uint32(b[18:22]))
	h := int(binary.LittleEndian.Uint32(b[22:26]))
	off := int(binary.LittleEndian.Uint32(b[10:14]))
	if w < 1 || h < 1 || w > 6000 || h > 3000 {
		return nil, fmt.Errorf("неверный размер рисунка")
	}
	stride := (w*3 + 3) &^ 3
	if len(b) < off+stride*h {
		return nil, fmt.Errorf("обрезан рисунок")
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		src := off + (h-1-y)*stride
		for x := 0; x < w; x++ {
			i := src + x*3
			img.Set(x, y, color.RGBA{R: b[i+2], G: b[i+1], B: b[i], A: 255})
		}
	}
	var out bytes.Buffer
	if err = png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func craneWordRow(values []string, widths []int) string {
	return strings.Replace(tempTableRow(values, widths), `<w:tr>`, `<w:tr><w:trPr><w:cantSplit/></w:trPr>`, 1)
}

func cranePlacementWordImageInline(id int) string {
	const cx int64 = 9000000
	const cy int64 = cx * 300 / 1040
	return fmt.Sprintf(`<w:p><w:r><w:drawing><wp:inline xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" distT="0" distB="0" distL="0" distR="0"><wp:extent cx="%d" cy="%d"/><wp:docPr id="%d" name="Положение колёс"/><a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:nvPicPr><pic:cNvPr id="0" name="Схема"/><pic:cNvPicPr/></pic:nvPicPr><pic:blipFill><a:blip xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" r:embed="rId%d"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p>`, cx, cy, id+1, id, cx, cy)
}

func exportCranePlacementWord(r CranePlacementResult, path string) error {
	if len(r.Cases) == 0 {
		return fmt.Errorf("нет рассчитанных положений кранов")
	}
	entries := map[string]string{}
	var b, rel strings.Builder
	b.WriteString(pBold("РАССТАНОВКА МОСТОВЫХ КРАНОВ. ОПАСНЫЕ ПОЛОЖЕНИЯ КОЛЁСНЫХ НАГРУЗОК"))
	var spans []string
	for _, x := range r.SpansMM {
		spans = append(spans, tempRU(x/1000, 3))
	}
	b.WriteString(pText("Подкрановая балка: " + strconv.Itoa(len(r.SpansMM)) + " пролёт(ов); длины: " + strings.Join(spans, "; ") + " м."))
	b.WriteString(pText("Расчётная схема: неразрезная балка, вертикальные перемещения опор запрещены, повороты свободны; EI одинаковое по пролётам."))
	b.WriteString(pText("Вертикальные силы колёс — нормативные, кН. Координаты — от левой опоры балки, мм."))
	for i, c := range r.Cases {
		if i > 0 {
			b.WriteString(`<w:p><w:r><w:br w:type="page"/></w:r></w:p>`)
		}
		b.WriteString(pBold(fmt.Sprintf("%d. %s", i+1, c.Name)))
		b.WriteString(pText("Экстремальное усилие: " + tempRU(c.Value, 3) + " " + c.Units + ". Сечение x=" + tempRU(c.SectionMM, 1) + " мм."))
		picture, err := craneCasePNG(r.SpansMM, c)
		if err != nil {
			return err
		}
		id := i + 1
		entries[fmt.Sprintf("word/media/position_%03d.png", id)] = string(picture)
		fmt.Fprintf(&rel, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/position_%03d.png"/>`, id, id)
		b.WriteString(cranePlacementWordImageInline(id))
		b.WriteString(pText("Схема " + strconv.Itoa(i+1) + ". Длины стрелок пропорциональны колёсным нагрузкам; размеры на схеме в миллиметрах."))
		widths := []int{800, 2600, 1800, 1800, 3100}
		b.WriteString(tempTableStart(widths))
		b.WriteString(craneWordRow([]string{"№", "Кран", "Координата x, мм", "Pн, кН", "Положение"}, widths))
		length := 0.0
		for _, L := range r.SpansMM {
			length += L
		}
		for j, p := range c.Points {
			pos := "На балке"
			if p.XMM < 0 || p.XMM > length {
				pos = "Вне пролётов"
			}
			b.WriteString(craneWordRow([]string{strconv.Itoa(j + 1), p.Crane, tempRU(p.XMM, 1), tempRU(p.PKN, 3), pos}, widths))
		}
		b.WriteString(`</w:tbl>`)

	}
	if len(r.Warnings) > 0 {
		b.WriteString(pBold("Принятые допущения"))
		for _, v := range r.Warnings {
			b.WriteString(pText(v))
		}
	}
	entries["[Content_Types].xml"] = `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Default Extension="png" ContentType="image/png"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`
	entries["_rels/.rels"] = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`
	entries["word/_rels/document.xml.rels"] = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + rel.String() + `</Relationships>`
	entries["word/document.xml"] = `<?xml version="1.0" encoding="UTF-8"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` + b.String() + `<w:sectPr><w:pgSz w:w="16838" w:h="11906" w:orient="landscape"/><w:pgMar w:top="780" w:bottom="720" w:left="750" w:right="750"/></w:sectPr></w:body></w:document>`
	return packageZip(path, entries)
}

// A complete Office-generated workbook is used as the OOXML package shell.
// Replacing only the two sheet payloads avoids Excel repair dialogs caused by
// minimal hand-built document parts. The resource is embedded in the executable.
//
//go:embed Resources/CraneExcelTemplate.xlsx
var craneExcelTemplate []byte

func craneExcelSheetRows(rows string) string {
	// Mapping to the existing styles in the verified template.
	swap := strings.NewReplacer(` s="0"`, ` s="0"`, ` s="1"`, ` s="9"`, ` s="2"`, ` s="5"`, ` s="3"`, ` s="11"`, ` s="4"`, ` s="11"`)
	return swap.Replace(rows)
}
func craneExcelPackage(path string, summary, details, beam string, detailWidths []float64) error {
	r, err := zip.NewReader(bytes.NewReader(craneExcelTemplate), int64(len(craneExcelTemplate)))
	if err != nil {
		return fmt.Errorf("шаблон Excel: %w", err)
	}
	files := make(map[string]string, len(r.File))
	for _, f := range r.File {
		z, err := f.Open()
		if err != nil {
			return err
		}
		b, readErr := io.ReadAll(z)
		closeErr := z.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		files[f.Name] = string(b)
	}
	files["xl/worksheets/sheet1.xml"] = sheetXML(craneExcelSheetRows(summary), []float64{7, 57, 23, 14, 19, 25})
	files["xl/worksheets/sheet2.xml"] = sheetXML(craneExcelSheetRows(details), detailWidths)
	files["xl/worksheets/sheet3.xml"] = sheetXML(craneExcelSheetRows(beam), []float64{30, 19, 19, 20, 20, 20, 18, 23, 20})
	// Keep the original Excel-generated package and its styles, add only one genuine sheet.
	const contentType = `<Override PartName="/xl/worksheets/sheet3.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`
	files["[Content_Types].xml"] = strings.Replace(files["[Content_Types].xml"], "</Types>", contentType+"</Types>", 1)
	const relationship = `<Relationship Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="/xl/worksheets/sheet3.xml" Id="rIdCraneBeamAudit"/>`
	files["xl/_rels/workbook.xml.rels"] = strings.Replace(files["xl/_rels/workbook.xml.rels"], "</Relationships>", relationship+"</Relationships>", 1)
	const sheet = `<x:sheet name="Балка" sheetId="3" r:id="rIdCraneBeamAudit" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"/>`
	files["xl/workbook.xml"] = strings.Replace(files["xl/workbook.xml"], "</x:sheets>", sheet+"</x:sheets>", 1)
	return packageZip(path, files)
}

// Excel audit: the coordinates of found extrema are fixed. For each wheel the
// ordinate η is reconstructed from the three-moment equation by Excel formulas
// with complete forward/backward tridiagonal elimination. The Go computation
// only provides formula caches and an independent check against the FE solver.
func exportCranePlacementExcel(r CranePlacementResult, path string) error {
	return exportCranePlacementAuditExcel(r, path)
}
