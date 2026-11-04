package main

import (
	"fmt"
	"math"
	"strings"
)

// Each load scheme has different geometric and normative properties. The
// description is kept independent of the GUI and reused in both reports.
type TransportDetail struct {
	Kind    string
	Caption string
	Rows    [][2]string
	Rules   []string
	Figures []string
	Source  string
}

func transportKind(s TempSelection) string {
	t := strings.ToLower(s.Title)
	switch {
	case strings.Contains(t, "аб-51") || strings.Contains(t, "аб-74") || strings.Contains(t, "аб-151"):
		return "AB"
	case strings.Contains(t, "метрополит"):
		return "Metro"
	case strings.Contains(t, "трамва"):
		return "Tram"
	case strings.Contains(t, "железнодорож") || strings.Contains(t, "с14") || strings.Contains(t, "с11"):
		return "SK"
	case strings.Contains(t, "н14") || strings.Contains(t, "н11"):
		return "NK"
	case strings.Contains(t, "а14") || strings.Contains(t, "а11"):
		return "AK"
	}
	return ""
}
func transportFigures(s TempSelection) []string {
	switch transportKind(s) {
	case "AK":
		return []string{"AK"}
	case "NK":
		return []string{"NK"}
	case "Metro":
		return []string{"Metro"}
	case "Tram":
		return []string{"Tram"}
	case "AB":
		return nil // Таблицы 6.6 и 6.7 представлены точными значениями выбранного АБ
	}
	return nil
}
func transportLength(s TempSelection) float64 { return tempNumber(s, "load_length_m") }
func transportGammaSK(length float64) float64 {
	if length <= 0 {
		return 0
	}
	if length < 50 {
		return 1.30 - 0.15*length/50
	}
	if length < 150 {
		return 1.15 - 0.05*(length-50)/100
	}
	return 1.10
}
func transportGammaAB(s TempSelection) float64 {
	g := tempNumber(s, "gamma_selected")
	if g >= 1.1 && g <= 1.4 {
		return g
	}
	return 0
}
func transportDetail(s TempSelection) TransportDetail {
	kind := transportKind(s)
	d := TransportDetail{Kind: kind, Caption: s.Title, Source: tempReadableSource(s.Source), Figures: transportFigures(s)}
	add := func(k, v string) { d.Rows = append(d.Rows, [2]string{k, v}) }
	rule := func(s string) { d.Rules = append(d.Rules, s) }
	switch kind {
	case "AK":
		k := int(tempNumber(s, "K"))
		if k == 0 {
			k = 14
		}
		axle := float64(k) * 10.0
		distributed := float64(k)
		add("Схема", "Рисунок 6.1а СП 35.13330.2011")
		add("Состав схемы", "Двухосная тележка и равномерная нагрузка на две колеи")
		add("Класс нагрузки", fmt.Sprintf("А%d", k))
		add("Нормативная нагрузка на одну ось тележки", fmt.Sprintf("%s кН", tempRU(axle, 0)))
		add("Нормативная распределённая часть (обе колеи)", fmt.Sprintf("%s кН/м", tempRU(distributed, 0)))
		add("Коэффициент надёжности тележки", "1,50")
		add("Коэффициент надёжности распределённой части", "1,25")
		add("Расчётная нагрузка на одну ось тележки", fmt.Sprintf("%s кН", tempRU(axle*1.5, 1)))
		add("Расчётная распределённая часть (обе колеи)", fmt.Sprintf("%s кН/м", tempRU(distributed*1.25, 2)))
		add("Расстояние между осями тележки", "1,5 м (рисунок 6.1а СП 35.13330.2011)")
		add("От оси крайней полосы до ограждения", "не менее 1,5 м")
		add("Между осями соседних полос АК", "не менее 3,0 м")
		rule("Полосы АК размещаются в неблагоприятном положении. При загружении нескольких полос действуют дополнительные правила СП 35.")
		rule("Вместе с тележкой учитывается распределённая часть АК. Дополнительные коэффициенты, в том числе динамический, назначаются по расчётной ситуации.")
	case "NK":
		k := int(tempNumber(s, "K"))
		if k == 0 {
			k = 14
		}
		axle := 18.0 * float64(k)
		add("Схема", "Рисунок 6.1б СП 35.13330.2011")
		add("Состав схемы", "Одиночная четырёхосная тяжёлая тележка")
		add("Класс нагрузки", fmt.Sprintf("Н%d", k))
		add("Нормативная нагрузка на ось", tempRU(axle, 0)+" кН")
		add("Нормативная нагрузка от четырёх осей", tempRU(axle*4, 0)+" кН")
		add("Коэффициент надёжности", "1,10")
		add("Расчётная нагрузка на ось", tempRU(axle*1.10, 1)+" кН")
		add("Расчётная нагрузка от четырёх осей", tempRU(axle*4*1.10, 1)+" кН")
		add("Расстояние между соседними осями", "1,2 м (рисунок 6.1б СП 35.13330.2011)")
		add("От оси НК до кромки проезжей части", "не менее 1,75 м")
		add("Проверка сдвоенной НК", "между тележками 12 м; коэффициент 0,75")
		rule("НК не учитывается совместно с временной нагрузкой на тротуарах и при расчёте выносливости. Для II группы предельных состояний применяется коэффициент 0,8.")
		rule("Проверяют также предусмотренное СП 35 сдвоенное загружение НК с расстоянием между тележками 12 м и коэффициентом 0,75.")
	case "AB":
		ty := "АБ-51"
		if strings.Contains(strings.ToUpper(s.Title), "АБ-74") {
			ty = "АБ-74"
		}
		if strings.Contains(strings.ToUpper(s.Title), "АБ-151") {
			ty = "АБ-151"
		}
		// Geometry and loading are from tables 6.6 and 6.7 of SP 35.13330.2011.
		type ab struct {
			rear, front, base, outer, rearTrack, frontTrack, contactL, contactW, diam, edge, moving, parked float64
			minGap                                                                                          float64
		}
		p := map[string]ab{
			"АБ-51":  {333, 167, 3.5, 3.5, 2.4, 2.8, 0.4, 1.1, 1.5, 1.0, 1.9, 0.5, 20},
			"АБ-74":  {490, 235, 4.2, 3.8, 2.5, 2.8, 0.45, 1.30, 1.8, 1.2, 2.0, 0.7, 20},
			"АБ-151": {990, 490, 4.5, 5.4, 3.75, 4.1, 0.80, 1.65, 2.5, 1.6, 2.5, 1.0, 26},
		}[ty]
		add("Тип автомобиля", ty+": двухосный тяжёлый промышленный автомобиль")
		add("Источник параметров", "Таблицы 6.6 и 6.7 СП 35.13330.2011")
		add("Нормативная нагрузка на переднюю ось", tempRU(p.front, 0)+" кН")
		add("Нормативная нагрузка на заднюю ось", tempRU(p.rear, 0)+" кН")
		add("Расстояние между осями", tempRU(p.base, 1)+" м")
		add("Габарит по ширине задних колёс", tempRU(p.outer, 2)+" м")
		add("Колея передних / задних колёс", tempRU(p.frontTrack, 2)+" / "+tempRU(p.rearTrack, 2)+" м")
		add("Пятно контакта заднего колеса (длина × ширина)", tempRU(p.contactL, 2)+" × "+tempRU(p.contactW, 2)+" м")
		add("Диаметр колеса", tempRU(p.diam, 2)+" м")
		add("От ограждения до края колеса движущегося автомобиля", tempRU(p.edge, 2)+" м, не менее")
		add("Между задними колёсами соседних автомобилей при движении", tempRU(p.moving, 2)+" м, не менее")
		add("Между задними колёсами соседних стоящих автомобилей", tempRU(p.parked, 2)+" м, не менее")
		if g := transportGammaAB(s); g > 0 {
			add("Коэффициент надёжности, выбранный пользователем", tempRU(g, 2))
			add("Расчётная нагрузка на переднюю ось", tempRU(p.front*g, 1)+" кН")
			add("Расчётная нагрузка на заднюю ось", tempRU(p.rear*g, 1)+" кН")
		} else {
			add("Коэффициент надёжности", "1,10–1,40, зависит от плотности перевозимой породы (п. 6.23е)")
			add("Расчётная передняя ось при γf = 1,10 / 1,40", tempRU(p.front*1.1, 1)+" / "+tempRU(p.front*1.4, 1)+" кН")
			add("Расчётная задняя ось при γf = 1,10 / 1,40", tempRU(p.rear*1.1, 1)+" / "+tempRU(p.rear*1.4, 1)+" кН")
		}
		rule("По СП 35.13330.2011 рассматривают движущиеся и стоящие автомобили с предусмотренными для этих случаев условиями загружения.")
		rule("Коэффициент надёжности 1,10 или 1,40 и промежуточные значения назначаются по п. 6.23е с учётом плотности перевозимой породы. Значение не следует выбирать по желаемому запасу.")
	case "SK":
		k := int(tempNumber(s, "K"))
		if k == 0 {
			k = 14
		}
		add("Класс железнодорожной нагрузки", fmt.Sprintf("С%d", k))
		add("Нагрузочная модель", "Подвижной состав железной дороги. Расчётная интенсивность меняется в зависимости от линии влияния")
		add("Нормативная характеристика (ориентир)", tempRU(tempNumber(s, "distributed_kN_per_m"), 2)+" кН/м")
		add("Коэффициент надёжности для горизонтальной нагрузки", "1,20 при L = 0 м; 1,10 при L ≥ 50 м")
		add("Коэффициент для давления грунта от подвижного состава", "1,20 независимо от L")
		if l := transportLength(s); l > 0 {
			g := transportGammaSK(l)
			add("Длина загружения линии влияния", tempRU(l, 2)+" м")
			add("Коэффициент надёжности (вертикальная нагрузка)", tempRU(g, 3))
			add("Условное произведение справочной интенсивности на γf", tempRU(tempNumber(s, "distributed_kN_per_m")*g, 2)+" кН/м; не замена расчёта по приложению К")
		} else {
			add("Коэффициент надёжности для вертикальной нагрузки", "1,30 при L=0; 1,15 при L=50 м; 1,10 при L≥150 м")
		}
		rule("Значение из строки справочника не является универсальной равномерной нагрузкой для расчёта моста. Для корректной схемы требуются длина и форма линии влияния и приложение К.")
		rule("Для горизонтальных железнодорожных воздействий и давления грунта коэффициенты назначаются отдельно по таблице 6.9.")
	case "Metro", "Tram":
		n, empty := tempNumber(s, "loaded_wagon_kN"), tempNumber(s, "empty_wagon_kN")
		add("Схема", map[string]string{"Metro": "Рисунок 6.1в СП 35.13330.2011", "Tram": "Рисунок 6.1г СП 35.13330.2011"}[kind])
		add("Подвижной состав", map[string]string{"Metro": "Поезд метро", "Tram": "Поезд трамвая"}[kind])
		add("Оси одного вагона", "4")
		add("Нормативный вес гружёного вагона", tempRU(n, 0)+" кН")
		add("Нормативный вес порожнего вагона", tempRU(empty, 0)+" кН")
		if kind == "Metro" {
			add("Длина вагона по схеме", "19,2 м")
		} else {
			add("Длина вагона по схеме", "16,2 м")
			add("Максимальное число вагонов в одном поезде", "4")
			add("Межпоездное расстояние", "не менее 8,5 м")
		}
		if l := transportLength(s); l > 0 {
			g := math.Max(1.10, 1.3*(1-l/1000))
			add("Длина загружения", tempRU(l, 2)+" м")
			add("Коэффициент надёжности по формуле (6.23)", tempRU(g, 3))
			add("Расчётный вес гружёного вагона", tempRU(n*g, 2)+" кН")
			add("Расчётный вес порожнего вагона", tempRU(empty*g, 2)+" кН")
		} else {
			add("Коэффициент надёжности", "γf = max[1,10; 1,30 × (1 − L/1000)], L — длина загружения в метрах")
		}
		rule("Масса вагона и распределение веса между осями зависят от выбранной схемы. Расчётный вес вагона не является усилием в конкретном элементе мостового сооружения.")
	}
	return d
}
func transportSummaryFields(s TempSelection) (details, norm, gamma, calc string) {
	d := transportDetail(s)
	for _, r := range d.Rows {
		v := r[0]
		if strings.Contains(v, "Коэффициент надёжности") {
			if gamma == "" {
				gamma = r[1]
			}
			continue
		}
		if strings.HasPrefix(v, "Расчётная") || strings.HasPrefix(v, "Условное произведение") {
			if calc != "" {
				calc += "; "
			}
			calc += v + ": " + r[1]
			continue
		}
		if strings.HasPrefix(v, "Нормативная") {
			if norm != "" {
				norm += "; "
			}
			norm += v + ": " + r[1]
			continue
		}
		if strings.Contains(v, "класс") || strings.Contains(v, "Класс") || strings.Contains(v, "Расстояние между осями") || strings.Contains(v, "двухос") {
			if details != "" {
				details += "; "
			}
			details += v + ": " + r[1]
		}
	}
	if details == "" {
		details = d.Caption
	}
	if norm == "" {
		norm = "Нормативная схема"
	}
	if gamma == "" {
		gamma = "По условиям загружения"
	}
	if calc == "" {
		calc = "Требуется расчёт по схеме"
	}
	return
}
func transportDetailedText(s TempSelection, withSource bool) string {
	d := transportDetail(s)
	var lines []string
	lines = append(lines, d.Caption, "")
	for _, v := range d.Rows {
		lines = append(lines, v[0]+": "+v[1])
	}
	for _, r := range d.Rules {
		lines = append(lines, "", r)
	}
	if withSource {
		lines = append(lines, "", "Нормативное обоснование: "+d.Source)
	}
	return strings.Join(lines, "\r\n")
}
func transportNeedsLength(s TempSelection) bool {
	return transportKind(s) == "SK" || transportKind(s) == "Metro" || transportKind(s) == "Tram"
}
func transportNeedsABGamma(s TempSelection) bool { return transportKind(s) == "AB" }
func transportValidateGamma(v float64) bool      { return !math.IsNaN(v) && v >= 1.1 && v <= 1.4 }
