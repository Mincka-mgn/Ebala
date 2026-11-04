package main

import (
	"fmt"
	"math"
	"strings"
)

func snowWordHeading(label string) string {
	return `<w:p><w:pPr><w:keepNext/><w:keepLines/><w:spacing w:before="260" w:after="120"/></w:pPr><w:r><w:rPr><w:b/><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="20"/></w:rPr><w:t>` + tempXML(label) + `</w:t></w:r></w:p>`
}
func snowWordParagraph(s string) string {
	return `<w:p><w:pPr><w:spacing w:after="100"/></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:sz w:val="20"/></w:rPr><w:t xml:space="preserve">` + tempXML(s) + `</w:t></w:r></w:p>`
}
func snowNormReference(s string) string {
	if strings.Contains(s, snowNormSource) {
		return s
	}
	return s + " СП 20.13330.2016"
}
func snowWordCoefTrace(s SnowCalc) []string {
	o := []string{
		"Расчёт выполняется на горизонтальную проекцию покрытия по разделу 10 СП 20.13330.2016.",
		"Sg — нормативный вес снегового покрова на 1 м² горизонтальной поверхности земли. Sg = " + decimalInput(s.Ground, 3) + " кПа (снеговой район " + s.Region + ", таблица 10.1 СП 20.13330.2016).",
		"μ — коэффициент перехода от веса снежного покрова на земле к снеговой нагрузке на покрытие. Его значение определяется по схеме покрытия в приложении Б СП 20.13330.2016 и рассчитывается отдельно по каждому участку.",
		"ce — коэффициент сноса снега ветром, учитывающий уменьшение нагрузки при выполнении условий СП 20.13330.2016.",
		"ct — термический коэффициент; учитывает снижение нагрузки на неутеплённых покрытиях с тепловыделением при уклоне более 3% и отводе талой воды.",
		"γf — коэффициент надёжности по снеговой нагрузке, γf = " + decimalInput(s.Gamma, 3) + " (пункт 10.12 СП 20.13330.2016).",

		"В эпюрах подписи читаются так: чёрная нормативная (красная расчётная / зелёная пониженная), кПа. Например: 2,20 (3,08/1,10). Размеры в метрах.",
	}
	if s.JanuarySet {
		o = append(o, "Средняя температура января T = "+decimalInput(s.JanuaryTemp, 2)+" °C; источник: "+snowClimateOrigin(s)+".")
	} else {
		o = append(o, "Средняя температура января не задана; снижение снеговой нагрузки по условиям температуры не производится.")
	}
	if s.WinterWindSet {
		o = append(o, "Средняя скорость ветра холодного периода V = "+decimalInput(s.WinterWind, 2)+" м/с; источник: "+snowClimateOrigin(s)+".")
	} else {
		o = append(o, "Средняя скорость ветра холодного периода не задана.")
	}
	if s.CE < 1 {
		if s.Shape == "cone" || s.Shape == "dome" {
			o = append(o, fmt.Sprintf("Для круговой кровли при диаметре D = %s м и T≤−5 °C применяется п. 10.8 СП 20.13330.2016: ce = %s.", decimalInput(s.Span, 2), decimalInput(s.CE, 3)))
		} else {
			length := s.Span
			if s.Shape == "multi_gable" || s.Shape == "multi_vault" {
				length *= float64(s.Count)
			}
			b := math.Min(length, s.Width)
			lmax := math.Max(length, s.Width)
			lc := math.Min(100, 2*b-b*b/lmax)
			h := s.RoofHeight
			heights := []float64{5, 10, 20, 40, 60, 80, 100, 150, 200, 250, 300}
			a := []float64{.75, 1, 1.25, 1.5, 1.7, 1.85, 2, 2.25, 2.45, 2.65, 2.75}
			bb := []float64{.5, .65, .85, 1.1, 1.3, 1.45, 1.6, 1.9, 2.1, 2.3, 2.5}
			vals := bb
			if s.Terrain == "А" {
				vals = a
			}
			k := vals[0]
			if h >= 300 {
				k = vals[len(vals)-1]
			} else if h > 5 {
				for i := 1; i < len(heights); i++ {
					if h <= heights[i] {
						k = vals[i-1] + (h-heights[i-1])/(heights[i]-heights[i-1])*(vals[i]-vals[i-1])
						break
					}
				}
			}
			windCol := 0
			if s.WinterWind > 6 {
				windCol = 2
			} else if s.WinterWind > 4 {
				windCol = 1
			}
			tempRow := 0
			if s.JanuaryTemp < -25 {
				tempRow = 2
			} else if s.JanuaryTemp < -15 {
				tempRow = 1
			}
			kva := [][]float64{{1.4, 1.3, 1.3}, {1.4, 1.3, 1.2}, {1.3, 1.2, 1.2}}
			kvb := [][]float64{{1.4, 1.4, 1.3}, {1.4, 1.4, 1.3}, {1.4, 1.3, 1.2}}
			kv := kvb[tempRow][windCol]
			if s.Terrain == "А" {
				kv = kva[tempRow][windCol]
			}
			o = append(o, fmt.Sprintf("Снос ветром по п. 10.7 СП 20.13330.2016. Меньший размер b = %s м, больший L = %s м. Характерный размер lc = min(100; 2b−b²/L) = %s м (формула 10.3).", decimalInput(b, 2), decimalInput(lmax, 2), decimalInput(lc, 3)))
			o = append(o, fmt.Sprintf("Для типа местности %s и высоты H=%s м коэффициент k(H) = %s, таблица 11.2 СП 20.13330.2016 (с интерполяцией). По температуре T=%s °C и скорости V=%s м/с коэффициент kv=%s, таблица 10.2.", s.Terrain, decimalInput(h, 2), decimalInput(k, 4), decimalInput(s.JanuaryTemp, 2), decimalInput(s.WinterWind, 2), decimalInput(kv, 3)))
			o = append(o, fmt.Sprintf("ce = max(0,5; min(1,0; (kv−0,4√k)×(0,8+0,002lc))) = (%s−0,4√%s)×(0,8+0,002×%s), с ограничением = %s.", decimalInput(kv, 3), decimalInput(k, 4), decimalInput(lc, 3), decimalInput(s.CE, 3)))
		}
	} else {
		o = append(o, fmt.Sprintf("Для заданных условий снижение от сноса снега не применяется; ce = %s (пункты 10.6–10.9 СП 20.13330.2016).", decimalInput(s.CE, 3)))
	}
	if s.CT < 1 {
		o = append(o, fmt.Sprintf("Покрытие неутеплённое, с повышенным тепловыделением и выполнены требуемые условия: термический коэффициент ct = %s по п. 10.10 СП 20.13330.2016.", decimalInput(s.CT, 3)))
	} else {
		o = append(o, "Уменьшение по тепловыделению не применяется: ct = 1,000 (п. 10.10 СП 20.13330.2016).")
	}
	if s.ReducedApplicable() {
		o = append(o, fmt.Sprintf("Пониженная нормативная нагрузка по п. 10.11 СП 20.13330.2016: Sпониж = 0,5×μ×Sg = 0,5×μ×%s кПа; ce и ct не применяются.", decimalInput(s.Ground, 3)))
	} else {
		o = append(o, snowJanuaryReductionReason(s))
	}
	return o
}
func snowWordMuTrace(s SnowCalc, c SnowCase) []string {
	lines := []string{"Нормативное основание: " + snowNormReference(c.Source) + ". Схема «" + c.Title + "».", "Координата x отсчитывается от начала покрытия; x1 и x2 — границы участка распределённой нагрузки, м. Для криволинейных эпюр коэффициент μ вычисляется в середине элементарного участка. График представляет дискретную выборку расчётной функции; у парапета наклонная часть отображается прямой линией."}
	if s.RightRoof != nil && s.RightRoof.Shape != "none" {
		r := *s.RightRoof
		l := SnowRoofPart{Shape: s.Shape, Span: s.Span, Angle: s.Angle, Rise: s.Rise}
		lines = append(lines, fmt.Sprintf("Левый пролёт: %s, L1=%s м, α1=%s°; правый пролёт: %s, L2=%s м, α2=%s°. Ширина общей расчётной схемы L1+L2=%s м.", snowShapeTitle(l.Shape), decimalInput(l.Span, 3), decimalInput(l.Angle, 3), snowShapeTitle(r.Shape), decimalInput(r.Span, 3), decimalInput(r.resolvedAngle(), 3), decimalInput(l.Span+r.Span, 3)))
		for _, p := range []SnowRoofPart{l, r} {
			switch p.Shape {
			case "flat", "mono", "gable":
				lines = append(lines, fmt.Sprintf("Для %s при α=%s°: μ0=1 при α≤30°; μ0=(60−α)/30 при 30°<α<60°; μ0=0 при α≥60°; принято μ0=%s по приложению Б СП 20.13330.2016.", snowShapeTitle(p.Shape), decimalInput(p.resolvedAngle(), 3), decimalInput(snowRoofMu(p.resolvedAngle()), 4)))
			case "vault":
				R := p.Span*p.Span/(8*p.Rise) + p.Rise/2
				lines = append(lines, fmt.Sprintf("Для сводчатого пролёта L=%s м, f=%s м: R=L²/(8f)+f/2=%s м. Местный угол α(x)=arctg(|x−L/2|/√(R²−(x−L/2)²)). Симметричная схема: μ1=cos(1,5α) по Б.2 СП 20.13330.2016.", decimalInput(p.Span, 3), decimalInput(p.Rise, 3), decimalInput(R, 3)))
			}
		}
		if strings.Contains(c.Title, "перепада") {
			lines = append(lines, "При разных отметках кромок дополнительно учтён снеговой занос у перепада по Б.8 СП 20.13330.2016, как отдельный вариант загружения. Коэффициенты и ширина зоны уточняются по схеме Б.8.")
		}
		if c.Note != "" {
			lines = append(lines, "Принятый вариант: "+c.Note)
		}
		return lines
	}
	switch s.Shape {
	case "flat", "mono", "gable":
		lines = append(lines, fmt.Sprintf("Базовый коэффициент μ0 = 1,000 при α≤30°; μ0=(60°−α)/30° при 30°<α<60°; μ0=0 при α≥60°. При α=%s° получено μ0=%s. Неравномерные варианты определяются умножением μ0 на коэффициенты соответствующей схемы (0,75 и 1,25 либо 0,9 и 1,1).", decimalInput(s.Angle, 2), decimalInput(snowRoofMu(s.Angle), 3)))
	case "vault", "multi_vault":
		R := s.Span*s.Span/(8*s.Rise) + s.Rise/2
		lines = append(lines, fmt.Sprintf("Для кругового свода радиус R = L²/(8f)+f/2 = %s²/(8×%s)+%s/2 = %s м. В каждой точке α(x)=arctg(|x−L/2|/√(R²−(x−L/2)²)).", decimalInput(s.Span, 3), decimalInput(s.Rise, 3), decimalInput(s.Rise, 3), decimalInput(R, 3)))
		lines = append(lines, "Симметричная схема: μ1(x)=max(0; cos(1,5α(x))) по рисунку Б.2. Несимметричные схемы: линейная интерполяция μ2 между контрольными сечениями; на противоположной половине принимается множитель 0,5. Полные формулы по расчётным точкам приведены на листе варианта в Excel.")
	case "shed":
		lines = append(lines, "Шедовые покрытия по Б.4, рисунок Б.7: вариант 1 — μ=1,0 на всей горизонтальной проекции; вариант 2 — на половинах каждого шедового пролёта чередуются μ=1,4 и μ=0,6. Высота конька над землёй H входит в исходную геометрию, но не в коэффициент формы μ.")
	case "parapet":
		mu := math.Min(3, 2*s.H/s.Ground)
		lines = append(lines, fmt.Sprintf("Покрытие с парапетом, Б.13, рисунок Б.16: h=%s м, Sg=%s кПа; при h>Sg/2 местная нагрузка μ=2h/Sg, но не более 3; μmax=%s, b=2h=%s м, линейный спад до μ=1.", decimalInput(s.H, 3), decimalInput(s.Ground, 3), decimalInput(mu, 3), decimalInput(2*s.H, 3)))
		if s.Span > 24 || s.H > 1.2 {
			lines = append(lines, "По Б.13(в) дополнительно рассматривается независимый вариант снегозаноса у парапета по Б.8 (рисунок Б.11), если он неблагоприятнее.")
		}
	case "skylight":
		a := s.SkylightWidth
		b := (s.Span - a) / 2
		b1 := math.Min(b, s.H)
		mu1 := .8
		mu2 := math.Min(4, math.Min(3*s.H/s.Ground, 1+.1*a/b))
		mu3 := math.Min(4, math.Min(3*s.H/s.Ground, 1+.5*a/b1))
		lines = append(lines, fmt.Sprintf("Геометрия фонаря: L=%s м, a=%s м — ширина фонаря, hf=%s м — его высота. Ширина одного ската от карниза до фонаря b=(L−a)/2=(%s−%s)/2=%s м; локальная длина b1=min(b;hf)=%s м.", decimalInput(s.Span, 2), decimalInput(a, 2), decimalInput(s.H, 2), decimalInput(s.Span, 2), decimalInput(a, 2), decimalInput(b, 3), decimalInput(b1, 3)))
		lines = append(lines, fmt.Sprintf("Коэффициенты схемы по Б.3: μ1=0,8; μ2=min(4; 3hf/Sg; 1+0,1a/b)=min(4; %s; %s)=%s; μ3=min(4; 3hf/Sg; 1+0,5a/b1)=min(4; %s; %s)=%s. Здесь Sg=%s кПа; числовые коэффициенты распределяются по зонам согласно рисунку Б.5.", decimalInput(3*s.H/s.Ground, 3), decimalInput(1+.1*a/b, 3), decimalInput(mu2, 3), decimalInput(3*s.H/s.Ground, 3), decimalInput(1+.5*a/b1, 3), decimalInput(mu3, 3), decimalInput(s.Ground, 3)))
		_ = mu1
	case "dome":
		d, f := s.Span, s.Rise
		R := d*d/(8*f) + f/2
		alpha := 2 * math.Atan2(2*f, d) * 180 / math.Pi
		lines = append(lines, fmt.Sprintf("Сферический купол: D=%s м, f=%s м, f/D=%s; R=D²/(8f)+f/2=%s м; максимальный уклон α=%s°.", decimalInput(d, 3), decimalInput(f, 3), decimalInput(f/d, 4), decimalInput(R, 3), decimalInput(alpha, 3)))
		if len(c.Polar) > 0 {
			cr := 2.55 - math.Exp(.8-14*f/d)
			r1 := R * math.Sin(math.Pi/6)
			lines = append(lines, fmt.Sprintf("Формула (Б.10) СП 20.13330.2016: μ2=Cr1×(z/r1)²×sinβ при z≤r1; Cr1=2,55−exp(0,8−14f/D)=%s; r1=R×sin30°=%s м.", decimalInput(cr, 4), decimalInput(r1, 4)))
			lines = append(lines, "Для z>r1: μ2=1,5×sinβ при α=45°; μ2=0 при α≥60°. Между 30°, 45° и 60° применяется линейная интерполяция. Здесь α=arcsin(z/R), β — полярный угол.")
			if strings.HasPrefix(c.Title, "3.") {
				lines = append(lines, "Формула (Б.11) СП 20.13330.2016: μ2=3×√(2f/D)×sin(3α)×sinβ. Вариант применён при отмеченных препятствиях / шероховатости / защите от ветра.")
			}
			lines = append(lines, "На рисунке показан меридиональный разрез при β=90°; пространственное распределение по радиусу z и углу β раскрыто в отдельной полярной таблице Excel.")
		}
	case "cone":
		C := snowConeFactor(s.Angle)
		lines = append(lines, fmt.Sprintf("Коническое покрытие диаметром D=%s м, α=%s°. Равномерная схема: μ1 по таблице Б.3 СП 20.13330.2016. Пространственный вариант: μ2(z,β)=Cr2×(z/r)×sinβ; Cr2=2,15×∛(α×π/180) при 7°<α≤30° (формула Б.12), Cr2=1,7×30°/α при 30°<α<60° (формула Б.13). Для заданного угла Cr2=%s. Здесь r=D/2, z — радиальная координата, β — полярный угол.", decimalInput(s.Span, 3), decimalInput(s.Angle, 3), decimalInput(C, 3)))
		if len(c.Polar) > 0 {
			lines = append(lines, "Приведённая эпюра является только меридиональным разрезом при β=90°. Полное пространственное распределение по радиусу и углу β приведено в Excel на листе данного варианта.")
		}
	case "step", "wall":
		u, l := s.stepRoofs()
		lines = append(lines, fmt.Sprintf("Перепад высот: верхнее покрытие L1=%s м и α1=%s°, нижнее L2=%s м и α2=%s°, перепад h=%s м. Коэффициенты снегопереноса определяются отдельно для верхнего и нижнего покрытия по Б.8 СП 20.13330.2016.", decimalInput(u.Span, 2), decimalInput(u.resolvedAngle(), 2), decimalInput(l.Span, 2), decimalInput(l.resolvedAngle(), 2), decimalInput(s.H, 2)))
		if strings.Contains(c.Title, "перепада") || strings.Contains(c.Title, "высокой стены") {
			lines = append(lines, "Локальный занос формируется независимо от основного равномерного варианта; максимальное μ ограничивается нормативными условиями. Размер зоны b и используемые множители m1,m2 приведены ниже вместе с подстановкой.", "Промежуточные коэффициенты из расчётного ядра: "+c.Note)
		}
	case "multi_gable":
		lines = append(lines, "Коэффициенты и зоны загружения для нескольких пролётов выбираются по схемам Б.8 и Б.9 СП 20.13330.2016. Для каждого пролёта участки указаны в таблице ниже.")
	}
	if c.Note != "" && s.Shape != "step" && s.Shape != "wall" {
		lines = append(lines, "Дополнительные условия выбранной схемы: "+c.Note)
	}
	return lines
}
