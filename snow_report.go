package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func snowGeometryNarrative(s SnowCalc) string {
	switch s.Shape {
	case "step":
		u, l := s.stepRoofs()
		return fmt.Sprintf("Верхняя кровля: %s, L1=%s м, α1=%s°, h1=%s м, Z1=%s м. Нижняя кровля: %s, L2=%s м, α2=%s°, h2=%s м, Z2=%s м. Перепад H=%s м. Длина B=%s м.", snowShapeTitle(u.Shape), decimalInput(u.Span, 2), decimalInput(u.resolvedAngle(), 2), decimalInput(u.Rise, 2), decimalInput(u.EdgeLevel, 2), snowShapeTitle(l.Shape), decimalInput(l.Span, 2), decimalInput(l.resolvedAngle(), 2), decimalInput(l.Rise, 2), decimalInput(l.EdgeLevel, 2), decimalInput(s.H, 2), decimalInput(s.Width, 2))
	case "skylight":
		return fmt.Sprintf("Кровля с фонарём: L=%s м, B=%s м, a=%s м (ширина фонаря), hf=%s м (высота фонаря), α=%s° или h=%s м (основной скат).", decimalInput(s.Span, 2), decimalInput(s.Width, 2), decimalInput(s.SkylightWidth, 2), decimalInput(s.H, 2), decimalInput(s.Angle, 2), decimalInput(s.Rise, 2))
	case "shed":
		return fmt.Sprintf("Шедовое покрытие: n=%d пролётов с шагом l=%s м; длина кровли B=%s м; отметка конька H=%s м над землёй.", s.Count, decimalInput(s.Span, 2), decimalInput(s.Width, 2), decimalInput(s.RoofHeight, 2))
	case "sphere_cone":
		return fmt.Sprintf("Сфера с конусом: H=%s м, f=%s м, D=%s м, α=%s°, r=%s м. Модель: касательное сопряжение сферической и конической частей; α₁=%s°, отметка стенки H−f=%s м; шаг β=%s°.", decimalInput(s.RoofHeight, 3), decimalInput(s.H, 3), decimalInput(s.Span, 3), decimalInput(s.Angle, 3), decimalInput(s.SkylightWidth, 3), decimalInput(sphereConeTangentAngle(s), 3), decimalInput(s.RoofHeight-s.H, 3), decimalInput(s.BetaStep, 3))
	case "parapet":
		return fmt.Sprintf("Плоское покрытие с парапетом: высота парапета h=%s м; длина участка L=%s м; ширина B=%s м.", decimalInput(s.H, 2), decimalInput(s.Span, 2), decimalInput(s.Width, 2))
	case "multi_gable", "multi_vault":
		return fmt.Sprintf("Число пролётов n=%d, L одного пролёта=%s м, B=%s м, уклон α=%s°, подъём f=%s м.", s.Count, decimalInput(s.Span, 2), decimalInput(s.Width, 2), decimalInput(s.Angle, 2), decimalInput(s.Rise, 2))
	default:
		return fmt.Sprintf("Размеры: L=%s м, B=%s м, α=%s°, f/h=%s м.", decimalInput(s.Span, 2), decimalInput(s.Width, 2), decimalInput(s.Angle, 2), decimalInput(s.Rise, 2))
	}
}
func snowWordHeaderRow(values []string, widths []int) string {
	row := tempTableRow(values, widths)
	return strings.Replace(row, "<w:tr>", `<w:tr><w:trPr><w:tblHeader/><w:cantSplit/></w:trPr>`, 1)
}
func snowWordRow(values []string, widths []int) string {
	return strings.Replace(tempTableRow(values, widths), "<w:tr>", `<w:tr><w:trPr><w:cantSplit/></w:trPr>`, 1)
}

var snowWordExtent = regexp.MustCompile(`cx="[0-9]+" cy="[0-9]+"`)

// Geometry images have exact height 3 cm while preserving original width/height.
func snowWordGeometryPicture(id int, data []byte) string {
	conf, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || conf.Height == 0 {
		return tempWordImageInline(id, 1400, 470)
	}
	cy := int64(2160000) // 6 cm at actual document scale
	cx := cy * int64(conf.Width) / int64(conf.Height)
	if cx > int64(5400000) {
		cx = 5400000
		cy = cx * int64(conf.Height) / int64(conf.Width)
	}
	pic := tempWordImageInline(id, conf.Width, conf.Height)
	pic = snowWordExtent.ReplaceAllString(pic, fmt.Sprintf(`cx="%d" cy="%d"`, cx, cy))
	return strings.Replace(pic, "<w:p>", `<w:p><w:pPr><w:keepNext/><w:keepLines/></w:pPr>`, 1)
}
func snowWordFormula(value string) string {
	return `<w:p><w:pPr><w:jc w:val="center"/><w:spacing w:before="120" w:after="120"/><w:keepLines/></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Cambria Math" w:hAnsi="Cambria Math"/><w:sz w:val="26"/></w:rPr><w:t>` + tempXML(value) + `</w:t></w:r></w:p>`
}
func snowWordTable(title string, headers []string, rows [][]string, widths []int) string {
	var out strings.Builder
	if title != "" {
		out.WriteString(snowWordHeading(title))
	}
	out.WriteString(tempTableStart(widths))
	if len(headers) > 0 {
		out.WriteString(snowWordHeaderRow(headers, widths))
	}
	for _, r := range rows {
		out.WriteString(snowWordRow(r, widths))
	}
	out.WriteString(`</w:tbl>`)
	return out.String()
}
func snowWordAppend(s SnowCalc, imageCount *int, rel *strings.Builder, entries map[string]string) (string, error) {
	if len(s.Cases) == 0 {
		return "", fmt.Errorf("отсутствуют снеговые загружения")
	}
	var b strings.Builder
	b.WriteString(snowWordHeading("Снеговые нагрузки. " + snowNormSource))
	b.WriteString(snowWordHeading("Геометрия покрытия"))
	b.WriteString(snowWordHeading("Схема геометрии покрытия и обозначения размеров"))
	roof, err := snowRoofImage(s)
	if err != nil {
		return "", err
	}
	*imageCount++
	imgID := *imageCount
	entries[fmt.Sprintf("word/media/snow_%d.png", imgID)] = string(roof)
	fmt.Fprintf(rel, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/snow_%d.png"/>`, imgID, imgID)
	b.WriteString(snowWordGeometryPicture(imgID, roof))
	// Plan of the longitudinal lantern zones is a separately approved resource.
	// The PNG preview is embedded in the DOCX; the editable original is shipped as SVG.
	if s.Shape == "skylight" {
		zoneFile := "snow_B3_lantern_zones_preview.png"
		if s.RightRoof != nil && s.RightRoof.Shape == "skylight" {
			zoneFile = "snow_B3_two_lantern_zones.png"
		}
		for _, p := range []string{filepath.Join(storageRoot(), "Resources", "SnowRoofs", zoneFile), filepath.Join("Resources", "SnowRoofs", zoneFile)} {
			if plan, e := os.ReadFile(p); e == nil {
				*imageCount++
				id := *imageCount
				entries[fmt.Sprintf("word/media/snow_%d.png", id)] = string(plan)
				fmt.Fprintf(rel, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/snow_%d.png"/>`, id, id)
				b.WriteString(snowWordHeading("План зон A, B, C у продольных фонарей"))
				b.WriteString(snowWordGeometryPicture(id, plan))
				break
			}
		}
	}
	if s.Shape == "dome" || s.Shape == "cone" || s.Shape == "sphere_cone" {
		fileName := map[string]string{"dome": "snow_B14_full.png", "cone": "snow_B15_full.png", "sphere_cone": "snow_B15a_full.png"}[s.Shape]
		for _, p := range []string{filepath.Join(storageRoot(), "Resources", "SnowRoofs", fileName), filepath.Join("Resources", "SnowRoofs", fileName)} {
			if data, e := os.ReadFile(p); e == nil {
				*imageCount++
				id := *imageCount
				entries[fmt.Sprintf("word/media/snow_%d.png", id)] = string(data)
				fmt.Fprintf(rel, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/snow_%d.png"/>`, id, id)
				b.WriteString(snowWordHeading("Схема-пояснение из СП"))
				b.WriteString(snowWordGeometryPicture(id, data))
				break
			}
		}
	}
	// Keep picture, caption and table together as far as the available page permits.
	caption := snowWordParagraph("Обозначения на схеме соответствуют величинам в следующей таблице. Схема условная, не в масштабе.")
	caption = strings.Replace(caption, "<w:pPr>", "<w:pPr><w:keepNext/>", 1)
	b.WriteString(caption)
	geo := [][]string{{"Тип покрытия", snowShapeTitle(s.Shape)}, {"Высота до конька H, м", decimalInput(s.RoofHeight, 3)}, {"Пролёт L / диаметр D, м", decimalInput(s.Span, 3)}}
	if s.Shape != "sphere_cone" {
		geo = append(geo, []string{"Длина покрытия B, м", decimalInput(s.Width, 3)})
	}
	if s.Shape == "cone" {
		geo = append(geo, []string{"Уклон конуса α, °", decimalInput(s.Angle, 3)}, []string{"Подъём кровли f, м", decimalInput(s.Rise, 3)})
	}
	if s.Shape == "sphere_cone" {
		geo = append(geo, []string{"Подъём кровли f от стенки до конька, м", decimalInput(s.H, 3)}, []string{"Радиус конуса r, м", decimalInput(s.SkylightWidth, 3)}, []string{"Уклон конуса α, °", decimalInput(s.Angle, 3)}, []string{"Подъём конуса r·tan α, м", decimalInput(s.SkylightWidth*math.Tan(s.Angle*math.Pi/180), 3)}, []string{"Угол касательной α₁, °", decimalInput(sphereConeTangentAngle(s), 3)}, []string{"Шаг β, °", decimalInput(s.BetaStep, 3)}, []string{"Отметка сопряжения со стенкой H−f, м", decimalInput(s.RoofHeight-s.H, 3)})
	}
	if s.Shape == "dome" {
		geo = append(geo, []string{"Подъём сферического купола f, м", decimalInput(s.Rise, 3)}, []string{"f/D", decimalInput(s.Rise/s.Span, 4)})
	}
	if s.Shape == "skylight" {
		geo = append(geo, []string{"Ширина фонаря a, м", decimalInput(s.SkylightWidth, 3)}, []string{"Высота фонаря hf, м", decimalInput(s.H, 3)}, []string{"Уклон основного покрытия α, °", decimalInput(s.Angle, 3)})
	}
	if s.Shape == "mono" || s.Shape == "gable" {
		geo = append(geo, []string{"Уклон покрытия α, °", decimalInput(s.Angle, 3)}, []string{"Подъём h, м", decimalInput(s.Rise, 3)})
	}
	if s.Shape == "vault" {
		geo = append(geo, []string{"Стрела подъёма f, м", decimalInput(s.Rise, 3)})
	}
	if s.Shape == "shed" {
		geo = append(geo, []string{"Число шедовых пролётов n", fmt.Sprint(s.Count)}, []string{"Шаг одного шеда l, м", decimalInput(s.Span, 3)}, []string{"Полная ширина n·l, м", decimalInput(float64(s.Count)*s.Span, 3)})
	}
	if s.Shape == "parapet" {
		geo = append(geo, []string{"Высота парапета h, м", decimalInput(s.H, 3)}, []string{"Расчётная ширина заноса b=2h, м", decimalInput(2*s.H, 3)})
	}
	if s.Shape == "step" {
		upper, lower := s.stepRoofs()
		geo = append(geo, []string{"Верхний участок L1, м", decimalInput(upper.Span, 3)}, []string{"Профиль верхнего участка", snowShapeTitle(upper.Shape)}, []string{"Отметка кромки Z1, м", decimalInput(upper.EdgeLevel, 3)}, []string{"Нижний участок L2, м", decimalInput(lower.Span, 3)}, []string{"Профиль нижнего участка", snowShapeTitle(lower.Shape)}, []string{"Отметка кромки Z2, м", decimalInput(lower.EdgeLevel, 3)}, []string{"Разность отметок h, м", decimalInput(s.H, 3)})
	}
	if s.RightRoof != nil && s.RightRoof.Shape != "none" {
		p := *s.RightRoof
		geo = append(geo, []string{"Тип второго пролёта", snowShapeTitle(p.Shape)}, []string{"Пролёт L2, м", decimalInput(p.Span, 3)}, []string{"Длина второго пролёта B2, м", decimalInput(p.Width, 3)}, []string{"Высота конька второго пролёта H2, м", decimalInput(p.RoofHeight, 3)}, []string{"Уклон второго пролёта α2, °", decimalInput(p.resolvedAngle(), 3)}, []string{"Подъём второго пролёта, м", decimalInput(p.Rise, 3)}, []string{"Отметка кромки второго пролёта, м", decimalInput(p.EdgeLevel, 3)})
		if p.Shape == "skylight" {
			geo = append(geo, []string{"Ширина фонаря правой кровли a2, м", decimalInput(p.SkylightWidth, 3)}, []string{"Высота фонаря правой кровли hf2, м", decimalInput(p.LanternHeight, 3)})
		}
		rightFeature := "Не относится к неутеплённым покрытиям с повышенным тепловыделением"
		if p.WarmRoof {
			rightFeature = "Неутеплённая конструкция с повышенным тепловыделением и отводом талой воды"
		}
		geo = append(geo, []string{"Особенности второго пролёта", rightFeature})
	}
	features := "Конструкция не относится к неутеплённым покрытиям с повышенным тепловыделением и отводом талой воды"
	if s.WarmRoof {
		features = "Неутеплённая конструкция с повышенным тепловыделением и отводом талой воды"
	}
	geo = append(geo, []string{"Особенности конструкции", features})
	b.WriteString(snowWordTable("", []string{"Геометрический параметр", "Значение"}, geo, []int{4100, 5400}))
	b.WriteString(snowWordHeading("Климатические характеристики и коэффициенты"))
	climate := [][]string{{"Снеговой район", s.SiteLabel(), "СП 20.13330.2016, таблица 10.1"}, {"Вес снегового покрова Sg, кПа", decimalInput(s.Ground, 3), "СП 20.13330.2016, таблица 10.1"}, {"Тип местности", s.Terrain, "СП 20.13330.2016, пункт 11.1.6"}, {"Средняя температура января, °C", snowTemperatureLabelCommon(s), snowClimateOrigin(s)}, {"Скорость ветра холодного периода, м/с", snowWindLabelCommon(s), snowClimateOrigin(s)}, {"Коэффициент сноса снега ce", decimalInput(s.CE, 3), "СП 20.13330.2016, пп. 10.6–10.9"}, {"Термический коэффициент ct", decimalInput(s.CT, 3), "СП 20.13330.2016, пункт 10.10"}, {"Коэффициент надёжности γf", decimalInput(s.Gamma, 3), "СП 20.13330.2016, пункт 10.12"}}
	if !s.ReducedApplicable() {
		climate = append(climate, []string{"Пониженная нормативная нагрузка", "Не вычисляется", snowJanuaryReductionReason(s)})
	}
	if s.CT == 1 {
		cause := "Не выполнены условия п. 10.10 СП 20.13330.2016: повышенное тепловыделение, уклон более 3% и отвод талой воды"
		if s.RightRoof != nil && s.RightRoof.Shape != "none" && (s.WarmRoof || s.RightRoof.WarmRoof) {
			cause = "У двух пролётов разные условия тепловыделения или уклона. Раздельное уменьшение не применяется: консервативно ct=1 для обоих пролётов."
		}
		climate = append(climate, []string{"Снижение по тепловыделению", "Не применяется", cause})
	} else {
		climate = append(climate, []string{"Снижение по тепловыделению", "Применяется, ct=0,8", "Пункт 10.10 СП 20.13330.2016"})
	}
	b.WriteString(snowWordTable("", []string{"Параметр", "Значение", "Источник / основание"}, climate, []int{3800, 1750, 4000}))
	explanations := [][]string{}
	for i, line := range snowWordCoefTrace(s) {
		explanations = append(explanations, []string{fmt.Sprint(i + 1), line})
	}
	b.WriteString(snowWordHeading("Основные расчётные формулы"))
	b.WriteString(snowWordFormula("Sₙ = μ · cₑ · cₜ · Sg"))
	b.WriteString(snowWordFormula("Sᵣ = γf · Sₙ"))
	if s.ReducedApplicable() {
		b.WriteString(snowWordFormula("Sпониж = 0,5 · μ · Sg"))
	}
	b.WriteString(snowWordTable("Расшифровка обозначений и расчёт коэффициентов", []string{"№", "Пояснение / формула / числовая подстановка"}, explanations, []int{500, 9000}))
	b.WriteString(snowWordHeading("Определение вариантов загружения"))
	views := snowPresentations(s)
	intro := fmt.Sprintf("Определено %d вариантов снегового загружения.", len(views))
	if s.Shape == "skylight" {
		intro += " Зоны A, B, C одной схемы по длине фонаря действуют совместно и не являются самостоятельными вариантами."
	}
	if s.Shape == "vault" {
		intro += " Зеркальная схема учтена в расчётном наборе и может быть приложена с противоположной стороны; повторная эпюра в отчёте опущена."
	}
	b.WriteString(snowWordParagraph(intro))
	summary := [][]string{}
	for i, v := range views {
		m := v.MaxMu()
		summary = append(summary, []string{fmt.Sprint(i + 1), v.Title, decimalInput(m, 3), decimalInput(s.Q(m), 3), decimalInput(s.QDesign(m), 3)})
	}
	b.WriteString(snowWordTable("", []string{"№", "Вариант", "μmax", "Нормативная Sₙ,max, кПа", "Расчётная Sᵣ,max, кПа"}, summary, []int{550, 4300, 1050, 1700, 1900}))
	for _, v := range views {
		b.WriteString(snowWordHeading(v.Title))
		if v.Note != "" {
			b.WriteString(snowWordParagraph(v.Note))
		}
		for j, c := range v.Cases {
			zone := ""
			if j < len(v.Zones) {
				zone = "Зона " + v.Zones[j] + ". "
			}
			if zone != "" {
				b.WriteString(snowWordHeading(zone + c.Title))
			}
			trace := [][]string{}
			for k, line := range snowWordMuTrace(s, c) {
				trace = append(trace, []string{fmt.Sprint(k + 1), line})
			}
			b.WriteString(snowWordTable("Нормативное основание и определение коэффициентов", []string{"№", "Исходные величины, формулы, расчётные значения и пояснения"}, trace, []int{500, 9000}))
			if s.Shape == "sphere_cone" {
				b.WriteString(snowWordTable("Полная числовая таблица (шаг β задан пользователем)", []string{"β, °", "От края, м", "Зона", "μ", "Sн, кПа", "Sр, кПа", "Sпониж, кПа"}, snowWordReservoirRows(s, c), []int{850, 1450, 850, 850, 1700, 1700, 2150}))
			} else if s.Shape == "cone" || s.Shape == "dome" {
				b.WriteString(snowWordTable("Распределение нагрузки по радиусу и углу β", []string{"Радиус r, м", "Угол β, °", "Коэффициент μ", "Нормативная Sₙ, кПа", "Расчётная Sᵣ, кПа", "Пониженная нормативная, кПа"}, snowWordReservoirRows(s, c), []int{1450, 1350, 1250, 1870, 1870, 1910}))
			} else {
				vals := [][]string{}
				for _, section := range c.Intervals {
					vals = append(vals, []string{decimalInput(section.X1, 3), decimalInput(section.X2, 3), decimalInput(section.Mu, 4), decimalInput(s.Q(section.Mu), 4), decimalInput(s.QDesign(section.Mu), 4), snowReducedCell(s, section.Mu)})
				}
				b.WriteString(snowWordTable("Числовые значения по участкам покрытия", []string{"Начало участка x₁, м", "Конец участка x₂, м", "Коэффициент μ", "Нормативная Sₙ, кПа", "Расчётная Sᵣ, кПа", "Пониженная нормативная, кПа"}, vals, []int{1510, 1510, 1100, 1770, 1770, 1890}))
			}
			img, e := snowImage(s, c)
			if e != nil {
				return "", e
			}
			*imageCount++
			n := *imageCount
			entries[fmt.Sprintf("word/media/snow_%d.png", n)] = string(img)
			fmt.Fprintf(rel, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/snow_%d.png"/>`, n, n)
			caption := "Эпюра снеговой нагрузки. " + zone + v.Title
			if len(c.Cross) > 0 {
				caption = "Продольная эпюра. " + zone + v.Title
				if s.Shape == "sphere_cone" {
					caption = "Сечение по зонам 6–1. " + zone + v.Title
				}
			}
			b.WriteString(snowWordHeading(caption))
			b.WriteString(tempWordImageInline(n, 1400, 470))
			if len(c.Cross) > 0 {
				cross := c
				cross.Intervals = c.Cross
				cross.Cross = nil
				cross.Title = "Поперечное сечение · " + v.Title
				if s.Shape == "sphere_cone" {
					cross.Title = "Сечение по зонам 2–4–4–2 · " + v.Title
				}
				img2, er := snowImage(s, cross)
				if er != nil {
					return "", er
				}
				*imageCount++
				n2 := *imageCount
				entries[fmt.Sprintf("word/media/snow_%d.png", n2)] = string(img2)
				fmt.Fprintf(rel, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/snow_%d.png"/>`, n2, n2)
				crossCaption := "Поперечная эпюра. " + zone + v.Title
				if s.Shape == "sphere_cone" {
					crossCaption = "Сечение по зонам 2–4–4–2. " + zone + v.Title
				}
				b.WriteString(snowWordHeading(crossCaption))
				b.WriteString(tempWordImageInline(n2, 1400, 470))
			}
		}
	}
	return b.String(), nil
}

// snowExcelAppend writes only auditable numeric cells on the main worksheet.
// Graphics are placed on dedicated worksheets by snowExcelAddFigureSheets.
func snowExcelAppend(s SnowCalc, add func(...string), row *int, imageCount *int, rels, drawing *strings.Builder, entries map[string]string) error {
	add("СНЕГОВЫЕ НАГРУЗКИ · СП 20.13330.2016")
	add("Тип покрытия", s.ReportName(), "Снеговой район", s.Region)
	add("Sg, кПа", "='Коэффициенты'!B2", "ce", "='Коэффициенты'!B19", "ct", "='Коэффициенты'!B21", "γf", "='Коэффициенты'!B3")
	factors := *row
	add("L, м", xnum(s.Span), "B, м", xnum(s.Width), "α, град", xnum(s.Angle), "h/f, м", xnum(s.Rise))
	add("H конька, м", xnum(s.RoofHeight), "T января", snowTemperatureLabelCommon(s), "V холодного периода", snowWindLabelCommon(s))
	add("Источник климата", snowClimateOrigin(s))
	add("Расчётное уравнение", "Sн = μ × ce × ct × Sg", "Sр = γf × Sн", "Sпониж = 0,5 × μ × Sg (если допускается)")
	add("Нормативные данные", "ce и ct определяются по разделу 10 СП 20.13330.2016; μ — по конкретной схеме приложения Б")
	add("Результаты", fmt.Sprintf("Выявлено %d возможных взаимоисключающих нормативных вариантов снегового загружения", len(s.Cases)))
	for i, c := range s.Cases {
		add("")
		add(fmt.Sprintf("ВАРИАНТ %d: %s", i+1, c.Title), c.Source)
		add("Основание и пояснение", c.Note)
		add("От, м", "До, м", "μ", "Sн, кПа", "Sр, кПа", "Sпониж, кПа")
		for _, v := range c.Intervals {
			next := *row + 1
			// B:factors = Sg, D:factors = ce, F:factors = ct, H:factors = γf.
			q := fmt.Sprintf("=C%d*$B$%d*$D$%d*$F$%d", next, factors, factors, factors)
			design := fmt.Sprintf("=D%d*$H$%d", next, factors)
			add(xnum(v.X1), xnum(v.X2), xnum(v.Mu), q, design, snowReducedFormula(s, next, factors))
		}
	}
	add("Пониженная нормативная нагрузка", snowReducedLabelCommon(s))
	return nil
}
func snowClimateOrigin(s SnowCalc) string {
	if s.ClimateOrigin == "sp131" {
		return "СП 131.13330.2025; " + s.ClimateReferenceRegion + "; " + s.ClimateReferencePlace
	}
	if s.ClimateOrigin == "manual" {
		return "Задано пользователем"
	}
	return "Значения введены пользователем"
}

func ExportSnowWord(s SnowCalc, path string) error {
	if len(s.Cases) == 0 {
		var e error
		s, e = SnowCompute(s)
		if e != nil {
			return e
		}
	}
	return exportTempWord(TempReport{Version: 1, Snow: &s}, path)
}
func ExportSnowExcel(s SnowCalc, path string) error {
	if len(s.Cases) == 0 {
		var e error
		s, e = SnowCompute(s)
		if e != nil {
			return e
		}
	}
	return exportTempExcel(TempReport{Version: 1, Snow: &s}, path)
}

func snowTemperatureLabelCommon(s SnowCalc) string {
	if !s.JanuarySet {
		return "не задана"
	}
	return decimalInput(s.JanuaryTemp, 2) + " °C"
}
func snowWindLabelCommon(s SnowCalc) string {
	if !s.WinterWindSet {
		return "не задана"
	}
	return decimalInput(s.WinterWind, 2) + " м/с"
}
func snowReducedLabelCommon(s SnowCalc) string {
	if !s.ReducedApplicable() {
		return snowJanuaryReductionReason(s)
	}
	return "0,5·μ·Sg; ce=ct=1 (п. 10.11 СП 20.13330.2016)"
}
func snowReducedCell(s SnowCalc, mu float64) string {
	if !s.ReducedApplicable() {
		return "—"
	}
	return decimalInput(s.QReduced(mu), 3)
}
func snowReducedFormula(s SnowCalc, row, groundRow int) string {
	if !s.ReducedApplicable() {
		return "—"
	}
	return fmt.Sprintf("=C%d*$B$%d*0.5", row, groundRow)
}
