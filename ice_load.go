package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// IceLoadCalc implements SP 20.13330.2016 (changes 1-6), section 12.
// Values are stored in SI units; d, b and effective thickness are in mm.
// Circular members (5..70 mm): line load i [N/m], formula 12.1.
// Other members: surface load i' [Pa], formula 12.2.
type IceLoadCalc struct {
	Enabled        bool           `json:"enabled"`
	Region         string         `json:"region"`
	BaseMM         float64        `json:"base_thickness_mm"`
	BaseFromStudy  bool           `json:"base_from_study,omitempty"`
	StudyReference string         `json:"study_reference,omitempty"`
	Shape          string         `json:"shape"` // round, surface_small, surface_study
	DiameterMM     float64        `json:"diameter_mm,omitempty"`
	Mu2            float64        `json:"mu2,omitempty"`
	HeightStart    float64        `json:"height_start_m"`
	HeightEnd      float64        `json:"height_end_m"`
	HeightStep     float64        `json:"height_step_m"`
	HighZone       string         `json:"high_zone,omitempty"`            // ordinary, asian_I, special
	Horizontal     bool           `json:"horizontal_suspended,omitempty"` // legacy file compatibility
	HighValuesMM   []float64      `json:"high_values_mm,omitempty"`       // data from approved investigations at 200/300/400m
	Wind           IceWindInput   `json:"wind,omitempty"`
	DensityKgM3    float64        `json:"density_kg_m3,omitempty"` // 0=900 kg/m3, SP20 p.12.2
	Points         []IceLoadPoint `json:"points,omitempty"`
}

type IceLoadPoint struct {
	HeightM     float64 `json:"height_m"`
	BaseMM      float64 `json:"base_mm"`
	K           float64 `json:"k"`
	Mu1         float64 `json:"mu1"`
	Mu2         float64 `json:"mu2"`
	EffectiveMM float64 `json:"effective_mm"`
	Norm        float64 `json:"norm"` // N/m for round, Pa for other elements
	Design      float64 `json:"design"`
	Basis       string  `json:"basis"`
	WindK       float64 `json:"wind_k,omitempty"`
	IcedDMM     float64 `json:"iced_d_mm,omitempty"`
	WindNorm    float64 `json:"wind_norm,omitempty"`    // case B, N/m or Pa
	WindDesign  float64 `json:"wind_design,omitempty"`  // case B (gamma and psi applied)
	WindLeading float64 `json:"wind_leading,omitempty"` // case A, design with no ice
	WindMean    float64 `json:"wind_mean,omitempty"`    // iced-section mean normative pressure/line load
}

const iceDensity = 0.9  // g/cm3 = 900 kg/m3, SP20 p.12.2
const iceGravity = 9.81 // m/s2
const iceGamma = 1.8    // SP20 p.12.5
func defaultIceLoad() IceLoadCalc {
	return IceLoadCalc{Region: "IV", BaseMM: 15, Shape: "round", DiameterMM: 10, HeightStart: 5, HeightEnd: 100, HeightStep: 5, HighZone: "ordinary", Mu2: .6, Wind: IceWindInput{Terrain: "A"}}
}
func iceMinZone(z string) (float64, bool) {
	switch romanZone(z) {
	case "I":
		return 3, true
	case "II":
		return 5, true
	case "III":
		return 10, true
	case "IV":
		return 15, true
	case "V":
		return 20, true
	}
	return 0, false
}
func iceLinear(x float64, xs, ys []float64) float64 {
	if x <= xs[0] {
		return ys[0]
	}
	for i := 1; i < len(xs); i++ {
		if x <= xs[i] {
			return ys[i-1] + (ys[i]-ys[i-1])*(x-xs[i-1])/(xs[i]-xs[i-1])
		}
	}
	return ys[len(ys)-1]
}
func iceMu1(d float64) float64 {
	return iceLinear(d, []float64{5, 10, 20, 30, 50, 70}, []float64{1.1, 1, .9, .8, .7, .6})
}
func iceHeightK(h float64) float64 {
	return iceLinear(h, []float64{5, 10, 20, 30, 50, 70, 100}, []float64{.8, 1, 1.2, 1.4, 1.6, 1.8, 2})
}
func iceHighTable(zone string) ([]float64, error) {
	switch zone {
	case "ordinary":
		return []float64{35, 45, 60}, nil
	case "asian_I":
		return []float64{15, 20, 25}, nil
	default:
		return nil, fmt.Errorf("Высотные значения для данной местности требуют специальных обследований (таблица 12.2 и карты 3г–3е СП 20.13330.2016)")
	}
}
func (s IceLoadCalc) Validate() error {
	zone := romanZone(s.Region)
	minimum, ok := iceMinZone(zone)
	if !ok {
		return fmt.Errorf("Выберите гололёдный район I–V")
	}
	if !s.IsRound() && s.Shape != "surface_small" && s.Shape != "surface_study" {
		return fmt.Errorf("Выберите вид обледеневаемого элемента")
	}
	for _, v := range []float64{s.BaseMM, s.DiameterMM, s.Mu2, s.HeightStart, s.HeightEnd, s.HeightStep} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("Некорректные числовые параметры")
		}
	}
	if s.HeightStart < 5 || s.HeightEnd < s.HeightStart || s.HeightEnd > 400 || s.HeightStep <= 0 {
		return fmt.Errorf("Высоты по разделу 12: от 5 до 400 м, шаг больше нуля. Ниже 5 м и выше 400 м методика не реализована без специальных данных")
	}
	if s.HeightStep > .0001 && (s.HeightEnd-s.HeightStart)/s.HeightStep > 300 {
		return fmt.Errorf("Для эпюры слишком малый шаг: более 300 отсчётов")
	}
	if s.BaseMM < minimum {
		return fmt.Errorf("b=%g мм ниже минимального значения %g мм для %s района (табл. 12.1)", s.BaseMM, minimum, zone)
	}
	if (zone == "I" || zone == "V") && !s.BaseFromStudy {
		return fmt.Errorf("Для %s района табл. 12.1 устанавливает только нижнюю границу b; требуется подтверждённое значение", zone)
	}
	if zone == "V" && !s.BaseFromStudy {
		return fmt.Errorf("Для V района необходимы специальные наблюдения")
	}
	if s.BaseFromStudy && strings.TrimSpace(s.StudyReference) == "" {
		return fmt.Errorf("Укажите источник подтверждённой толщины гололёда")
	}
	if s.IsRound() && (s.DiameterMM < 5 || s.DiameterMM > 70) {
		return fmt.Errorf("Таблица 12.4 задаёт μ1 для диаметров 5–70 мм; вне диапазона автоматический расчёт не предусмотрен")
	}
	if s.Shape == "surface_study" && (s.Mu2 <= 0 || s.Mu2 > 1) {
		return fmt.Errorf("Для элемента без малых размеров сечения введите μ2 по исследованиям (0 < μ2 ≤ 1)")
	}
	if len(s.HighValuesMM) == 3 && strings.TrimSpace(s.StudyReference) == "" {
		return fmt.Errorf("Для специальных высотных толщин укажите источник наблюдений")
	}
	if (s.DensityKgM3 != 0 && s.DensityKgM3 != 900) && (s.DensityKgM3 <= 0 || math.IsNaN(s.DensityKgM3) || math.IsInf(s.DensityKgM3, 0) || strings.TrimSpace(s.StudyReference) == "") {
		return fmt.Errorf("Изменение нормативной плотности льда требует положительного значения и источника фактических данных")
	}
	if s.HeightEnd > 100 || s.HeightStart > 100 {
		if _, e := s.highTable(); e != nil {
			return e
		}
		if (zone == "V" && len(s.HighValuesMM) != 3) || s.HighZone == "asian_I" && zone != "I" {
			return fmt.Errorf("Уточните тип местности и высотные гололёдные данные по таблице 12.2")
		}
		// Northern European, mountainous and complex-terrain areas have no universal table value.
		if s.HighZone != "ordinary" && s.HighZone != "asian_I" && len(s.HighValuesMM) != 3 {
			return fmt.Errorf("Высотные данные требуют специального обследования")
		}
	}
	if err := s.Wind.validate(); err != nil {
		return err
	}
	return nil
}
func (s IceLoadCalc) atHeight(h float64) (IceLoadPoint, error) {
	b := s.BaseMM
	k := 1.0
	basis := "Таблица 12.3, линейная интерполяция"
	if h <= 100 {
		k = iceHeightK(h)
	} else if h < 200 {
		high, e := s.highTable()
		if e != nil {
			return IceLoadPoint{}, e
		}
		// SP20 p.12.2: exp(0.007h), restricted to b2/b (b2 from table 12.2).
		k = math.Min(math.Exp(.007*h), high[0]/b)
		basis = "Пункт 12.2: k=min(exp(0,007h); b₂/b), 100<h<200 м"
	} else {
		high, e := s.highTable()
		if e != nil {
			return IceLoadPoint{}, e
		}
		hs := []float64{200, 300, 400}
		if h >= 400 {
			b = high[2]
		} else {
			j := 0
			if h >= 300 {
				j = 1
			}
			b = high[j]
			k = 1 + .01*(high[j+1]/high[j]-1)*(h-hs[j])
		}
		basis = "Таблица 12.2 и пункт 12.2: b=b₁; k=1+0,01(b₂/b₁−1)(h−h₁)"
	}
	if (s.Horizontal || s.Shape == "round_wire") && h >= 200 && s.IsRound() {
		b *= .9
		basis += "; для горизонтального подвешенного элемента b × 0,9"
	}
	p := IceLoadPoint{HeightM: h, BaseMM: b, K: k, Mu1: 1, Mu2: 1, Basis: basis}
	if s.IsRound() {
		p.Mu1 = iceMu1(s.DiameterMM)
		thick := b * k * p.Mu1
		p.EffectiveMM = thick
		// pi * t * (d+t) [mm2] * rho[g/cm3] * g[m/s2] * 1e-3 => N/m.
		p.Norm = math.Pi * thick * (s.DiameterMM + thick) * s.density() * iceGravity * 1e-3
	} else {
		if s.Shape == "surface_small" {
			p.Mu2 = .6
		} else {
			p.Mu2 = s.Mu2
		}
		p.EffectiveMM = b * k
		// b[mm] * rho[g/cm3] * g[m/s2] => Pa.
		p.Norm = b * k * p.Mu2 * s.density() * iceGravity
	}
	p.Design = p.Norm * iceGamma
	if s.Wind.Enabled {
		if err := iceWindAtHeight(s, &p); err != nil {
			return p, err
		}
	}
	return p, nil
}
func IceCompute(s IceLoadCalc) (IceLoadCalc, error) {
	s.Region = romanZone(s.Region)
	if e := s.Validate(); e != nil {
		return s, e
	}
	levels := []float64{s.HeightStart, s.HeightEnd}
	for h := s.HeightStart + s.HeightStep; h < s.HeightEnd-1e-8; h += s.HeightStep {
		levels = append(levels, h)
	}
	for _, h := range []float64{5, 10, 20, 30, 50, 70, 100, 200, 300, 400} {
		if h > s.HeightStart && h < s.HeightEnd {
			levels = append(levels, h)
		}
	}
	sort.Float64s(levels)
	s.Points = nil
	for _, h := range levels {
		if len(s.Points) > 0 && math.Abs(h-s.Points[len(s.Points)-1].HeightM) < 1e-7 {
			continue
		}
		p, e := s.atHeight(h)
		if e != nil {
			return s, e
		}
		s.Points = append(s.Points, p)
	}
	return s, nil
}
func iceHeightZoneName(k string) string {
	switch k {
	case "asian_I":
		return "I гололёдный район азиатской части"
	case "special":
		return "Особая климатическая местность"
	default:
		return "Обычная местность (таблица 12.2)"
	}
}
func (s IceLoadCalc) IsRound() bool    { return s.Shape == "round" || s.Shape == "round_wire" }
func (s IceLoadCalc) IsFlexible() bool { return s.Shape == "round_wire" || s.Wind.Flexible }
func (s IceLoadCalc) density() float64 {
	if s.DensityKgM3 > 0 {
		return s.DensityKgM3 / 1000
	}
	return iceDensity
}
func (s IceLoadCalc) highTable() ([]float64, error) {
	if len(s.HighValuesMM) == 3 {
		for _, b := range s.HighValuesMM {
			if b <= 0 || math.IsNaN(b) || math.IsInf(b, 0) {
				return nil, fmt.Errorf("Некорректные данные b для 200–400 м")
			}
		}
		return s.HighValuesMM, nil
	}
	return iceHighTable(s.HighZone)
}
func (s IceLoadCalc) NormUnit() string {
	if s.IsRound() {
		return "Н/м"
	}
	return "Па"
}
func (s IceLoadCalc) EngineeringUnit() string {
	if s.IsRound() {
		return "кН/м"
	}
	return "кПа"
}
func (s IceLoadCalc) Description() string {
	if s.IsRound() {
		if s.IsFlexible() {
			return "Подвешенный гибкий круглый элемент, d=" + decimalInput(s.DiameterMM, 1) + " мм"
		}
		return "Круглый элемент, d=" + decimalInput(s.DiameterMM, 1) + " мм"
	}
	return "Элемент строительной конструкции, поверхностная нагрузка"
}
