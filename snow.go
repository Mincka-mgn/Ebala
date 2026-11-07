package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// SnowCase is an independent load arrangement; different cases must not be summed.
// q values refer to horizontal roof projection, not inclined surface.
type SnowInterval struct {
	X1 float64 `json:"x1"`
	X2 float64 `json:"x2"`
	Mu float64 `json:"mu"`
}

// The JSON for this model is used in the ordinary project file as well as the reports.
type SnowPolarPoint struct {
	Radius float64 `json:"radius_m"`
	Beta   float64 `json:"beta_degrees"`
	Mu     float64 `json:"mu"`
	Zone   int     `json:"zone,omitempty"`
}
type SnowCase struct {
	Title     string           `json:"title"`
	Source    string           `json:"source"`
	Note      string           `json:"note,omitempty"`
	Intervals []SnowInterval   `json:"intervals"`
	Polar     []SnowPolarPoint `json:"polar,omitempty"`
	PeakMu    float64          `json:"peak_mu,omitempty"`
	Cross     []SnowInterval   `json:"cross_intervals,omitempty"`
}
type SnowRoofPart struct {
	Shape         string  `json:"shape,omitempty"`  // flat, mono, gable, vault
	Span          float64 `json:"span_m,omitempty"` // transverse length
	Angle         float64 `json:"angle_degrees,omitempty"`
	Rise          float64 `json:"rise_m,omitempty"`
	EdgeLevel     float64 `json:"edge_level_m,omitempty"` // roof elevation at abutment
	RoofHeight    float64 `json:"ridge_height_m,omitempty"`
	Width         float64 `json:"width_m,omitempty"`
	SkylightWidth float64 `json:"skylight_width_m,omitempty"`
	LanternHeight float64 `json:"lantern_height_m,omitempty"`
	WarmRoof      bool    `json:"warm_roof,omitempty"`
	SlopeMode     string  `json:"slope_mode,omitempty"`
}

func (p SnowRoofPart) resolvedAngle() float64 {
	if p.SlopeMode == "height" && p.Span > 0 {
		run := p.Span
		if p.Shape == "gable" {
			run = p.Span / 2
		} else if p.Shape == "skylight" {
			run = (p.Span - p.SkylightWidth) / 2
		}
		return math.Atan2(p.Rise, run) * 180 / math.Pi
	}
	return p.Angle
}
func (s SnowCalc) stepRoofs() (SnowRoofPart, SnowRoofPart) {
	u := s.UpperRoof
	l := s.LowerRoof
	if u.Span <= 0 {
		u.Span = s.Upper
		u.Shape = "flat"
	}
	if l.Span <= 0 {
		l.Span = s.Lower
		l.Shape = s.LowerShape
		l.Angle = s.Angle
		l.Rise = s.Rise
		l.SlopeMode = s.SlopeMode
	}
	if u.Shape == "" {
		u.Shape = "flat"
	}
	if l.Shape == "" {
		l.Shape = "flat"
	}
	return u, l
}

type SnowCalc struct {
	Enabled                bool          `json:"enabled"`
	Shape                  string        `json:"shape"`
	Region                 string        `json:"region,omitempty"`
	Ground                 float64       `json:"ground_kpa"`
	GroundLocal            bool          `json:"ground_local,omitempty"`
	Terrain                string        `json:"terrain"`
	LowerShape             string        `json:"lower_shape,omitempty"`
	UpperRoof              SnowRoofPart  `json:"upper_roof,omitempty"`
	LowerRoof              SnowRoofPart  `json:"lower_roof,omitempty"`
	RightRoof              *SnowRoofPart `json:"right_roof,omitempty"`
	Units                  string        `json:"units,omitempty"`
	DomeObstacles          bool          `json:"dome_obstacles,omitempty"`
	SlopeMode              string        `json:"slope_mode,omitempty"`
	BetaStep               float64       `json:"beta_step_degrees,omitempty"`
	SphereStrictFormula    bool          `json:"sphere_strict_formula,omitempty"`
	CE                     float64       `json:"ce"`
	CT                     float64       `json:"ct"`
	Gamma                  float64       `json:"gamma"`
	Span                   float64       `json:"span_m"`
	Width                  float64       `json:"width_m"`
	Angle                  float64       `json:"angle_degrees"`
	Rise                   float64       `json:"rise_m"`
	Count                  int           `json:"count"`
	H                      float64       `json:"height_step_m"`
	Upper                  float64       `json:"upper_catchment_m"`
	Lower                  float64       `json:"lower_width_m"`
	RoofSensitive          bool          `json:"roof_sensitive,omitempty"`
	SkylightWidth          float64       `json:"skylight_width_m,omitempty"`
	RoofHeight             float64       `json:"roof_height_m,omitempty"`
	JanuaryTemp            float64       `json:"january_temp_c,omitempty"`
	JanuarySet             bool          `json:"january_temp_set,omitempty"`
	WinterWind             float64       `json:"winter_wind_ms,omitempty"`
	WinterWindSet          bool          `json:"winter_wind_set,omitempty"`
	ClimateOrigin          string        `json:"climate_origin,omitempty"`
	ClimateReferencePlace  string        `json:"climate_reference_place,omitempty"`
	ClimateReferenceRegion string        `json:"climate_reference_region,omitempty"`
	Exposed                bool          `json:"wind_exposed,omitempty"`
	WarmRoof               bool          `json:"warm_roof_with_draining,omitempty"`
	RidgeEquipment         bool          `json:"ridge_equipment,omitempty"`
	Cases                  []SnowCase    `json:"cases,omitempty"`
}

const snowNormSource = "СП 20.13330.2016"

func defaultSnowCalc() SnowCalc {
	return SnowCalc{Region: "VIII", Ground: 4.0, Shape: "gable", Terrain: "В", LowerShape: "flat", SlopeMode: "angle", CE: 1, CT: 1, Gamma: 1.4, Span: 18, Width: 24, BetaStep: 15, RoofHeight: 8, SkylightWidth: 6, Angle: 10, Rise: 1.5, Count: 2, H: 2, Upper: 18, Lower: 18, Exposed: true}
}
func snowShapeTitle(t string) string {
	switch t {
	case "flat":
		return "Плоское покрытие"
	case "mono":
		return "Односкатное покрытие"
	case "gable":
		return "Двускатное покрытие"
	case "vault":
		return "Сводчатое покрытие"
	case "multi_gable":
		return "Двух- и многопролётное (двускатное)"
	case "multi_vault":
		return "Двух- и многопролётное (сводчатое)"
	case "skylight":
		return "Покрытие с фонарём"
	case "shed":
		return "Шедовое покрытие"
	case "wall":
		return "Примыкание к высокой стене"
	case "step":
		return "Перепад высоты двух покрытий"
	case "parapet":
		return "Покрытие с парапетом"
	case "cone":
		return "Резервуар с конической крышей"
	case "dome":
		return "Резервуар со сферической крышей"
	case "sphere_cone":
		return "Сфера с конусом"
	}
	return t
}
func snowRoofMu(alpha float64) float64 {
	alpha = math.Abs(alpha)
	if alpha <= 30 {
		return 1
	}
	if alpha >= 60 {
		return 0
	}
	return (60 - alpha) / 30
}
func snowMainRun(s SnowCalc) float64 {
	switch s.Shape {
	case "mono":
		return s.Span
	case "skylight":
		return math.Max(0, (s.Span-s.SkylightWidth)/2)
	case "sphere_cone":
		return s.SkylightWidth
	case "wall", "step":
		if s.LowerShape == "gable" {
			return s.Lower / 2
		}
		return s.Lower
	default:
		return s.Span / 2
	}
}
func snowResolveSlope(s *SnowCalc) {
	if s.SlopeMode == "" {
		s.SlopeMode = "angle"
	}
	run := snowMainRun(*s)
	if run <= 0 {
		return
	}
	sloped := s.Shape == "mono" || s.Shape == "gable" || s.Shape == "multi_gable" || s.Shape == "skylight" || s.Shape == "cone" || s.Shape == "sphere_cone" || s.Shape == "wall" || s.Shape == "step"
	if !sloped {
		return
	}
	if s.SlopeMode == "height" {
		if s.Rise != 0 {
			s.Angle = math.Atan(s.Rise/run) * 180 / math.Pi
		}
	} else if s.Angle != 0 {
		s.Rise = run * math.Tan(s.Angle*math.Pi/180)
	}
}
func snowParamsValid(s SnowCalc) error {
	if !num(s.Ground) || s.Ground <= 0 {
		return fmt.Errorf("Укажите снеговой район или положительный вес Sg")
	}
	for _, x := range []float64{s.CE, s.CT, s.Gamma, s.Span, s.Width, s.H, s.Upper, s.Lower, s.RoofHeight, s.WinterWind, s.SkylightWidth} {
		if !num(x) || x < 0 {
			return fmt.Errorf("Параметры покрытия должны быть неотрицательными (кроме направления уклона односкатной кровли)")
		}
	}
	if !num(s.Angle) || !num(s.Rise) || (s.Shape != "mono" && (s.Angle < 0 || s.Rise < 0)) {
		return fmt.Errorf("Отрицательный уклон допустим только для односкатной кровли")
	}
	if s.CE <= 0 || s.CT <= 0 || s.Gamma <= 0 || s.Span <= 0 || s.Width <= 0 {
		return fmt.Errorf("Укажите положительные размеры и коэффициенты")
	}
	if math.Abs(s.Angle) >= 90 {
		return fmt.Errorf("Уклон должен быть меньше 90°")
	}
	if math.Abs(s.Gamma-1.4) > 0.00001 {
		return fmt.Errorf("Для снеговой нагрузки γf=1,4 по п. 10.12 СП 20.13330.2016")
	}
	if s.CE < 1 && s.Terrain == "С" && s.Shape != "cone" && s.Shape != "dome" {
		return fmt.Errorf("Для типа местности С коэффициент сноса ce принимается равным 1,0 по п. 10.6")
	}
	if s.Shape == "vault" || s.Shape == "multi_vault" {
		if s.Rise <= 0 || s.Rise > s.Span/2 {
			return fmt.Errorf("Для кругового свода нужно 0 < f ≤ L/2")
		}
	}
	if s.Shape == "multi_gable" || s.Shape == "multi_vault" || s.Shape == "shed" {
		if s.Count < 2 || s.Count > 12 {
			return fmt.Errorf("Число пролётов: от 2 до 12")
		}
	}
	if s.Shape == "parapet" && s.H <= 0 {
		return fmt.Errorf("Укажите положительную высоту парапета h")
	}
	if s.Shape == "skylight" {
		if s.H <= 0 || s.SkylightWidth <= 0 || s.Span <= s.SkylightWidth {
			return fmt.Errorf("Для продольного фонаря нужны L>a>0 и высота h1>0")
		}
	}
	if s.Shape == "wall" || s.Shape == "step" {
		if s.Shape == "step" && s.UpperRoof.Span > 0 && s.LowerRoof.Span > 0 && s.UpperRoof.EdgeLevel <= s.LowerRoof.EdgeLevel {
			return fmt.Errorf("Уровень кромки верхней кровли должен быть выше нижней")
		}
		if s.H <= 0 || s.Lower <= 0 {
			return fmt.Errorf("Введите положительный перепад h и ширину нижнего покрытия")
		}
	}
	if s.RightRoof != nil && s.RightRoof.Shape != "none" {
		if s.Shape == "cone" || s.Shape == "dome" || s.Shape == "sphere_cone" || s.Shape == "multi_gable" || s.Shape == "multi_vault" {
			return fmt.Errorf("Для резервуаров и типовых многопролётных схем второй независимый пролёт не допускается")
		}
		if s.RightRoof.Span <= 0 {
			return fmt.Errorf("Для правого пролёта укажите L2 > 0")
		}
		if s.RightRoof.Shape != "flat" && s.RightRoof.Shape != "mono" && s.RightRoof.Shape != "gable" && s.RightRoof.Shape != "vault" && s.RightRoof.Shape != "skylight" {
			return fmt.Errorf("Тип правого пролёта не поддерживается")
		}
		if s.RightRoof.Shape == "skylight" && (s.RightRoof.LanternHeight <= 0 || s.RightRoof.SkylightWidth <= 0 || s.RightRoof.SkylightWidth >= s.RightRoof.Span) {
			return fmt.Errorf("Для правой кровли с фонарём укажите 0 < a < L и hf > 0")
		}
		if s.RightRoof.Shape == "vault" && (s.RightRoof.Rise <= 0 || s.RightRoof.Rise > s.RightRoof.Span/2) {
			return fmt.Errorf("Для правого свода нужно 0 < f ≤ L/2")
		}
		if math.Abs(s.RightRoof.resolvedAngle()) >= 90 {
			return fmt.Errorf("Уклон правого пролёта должен быть меньше 90°")
		}
	}
	if s.Shape == "sphere_cone" {
		if _, err := makeSphereConeGeometry(s); err != nil {
			return err
		}
	}
	if s.Shape == "dome" {
		if s.Rise <= 0 || s.Rise >= s.Span {
			return fmt.Errorf("Для сферического купола требуется 0 < f < D")
		}
		// For local surface inclinations α >= 60°, B.11 / table B.2
		// yields μ=0. Do not reject the WHOLE dome because only its steep
		// peripheral portion sheds the snow. The existing spherical-cap
		// parameterisation includes the overhanging annulus when rise > D/2.
	}

	return nil
}
func snowMuInterval(x1, x2, mu float64) SnowInterval {
	return SnowInterval{X1: x1, X2: x2, Mu: math.Max(0, mu)}
}
func snowCase(title, source, note string, parts ...SnowInterval) SnowCase {
	return SnowCase{Title: title, Source: source, Note: note, Intervals: parts}
}
func snowResample(s SnowCalc, mu func(x float64) float64, n int) []SnowInterval {
	out := make([]SnowInterval, 0, n)
	L := s.Span
	for i := 0; i < n; i++ {
		a := L * float64(i) / float64(n)
		b := L * float64(i+1) / float64(n)
		out = append(out, snowMuInterval(a, b, mu((a+b)/2)))
	}
	return out
}
func snowVaultAngle(L, f, x float64) float64 {
	r := L*L/(8*f) + f/2
	dx := math.Abs(x - L/2)
	q := math.Sqrt(math.Max(1e-10, r*r-dx*dx))
	return math.Atan2(dx, q) * 180 / math.Pi
}

// The need for an additional uneven arrangement depends on the supporting
// structure, which is not modelled here. Always show it as a review option.
// For roofs over 100 m in BOTH directions, B.1 note 2 mandates it.
func snowUnevenSource(s SnowCalc) (string, string, string) {
	if s.Span > 100 && s.Width > 100 {
		return "Обязательная неравномерная 0,9/1,1", "Б.1, примечание 2", "Обязательная дополнительная схема: оба размера покрытия более 100 м"
	}
	return "Проверочная неравномерная 0,9/1,1", "Б.1, примечание 1", "Дополнительная проверочная схема: необходимость зависит от чувствительности несущей конструкции"
}
func snowCases(s SnowCalc) ([]SnowCase, error) {
	snowResolveSlope(&s)
	if e := snowParamsValid(s); e != nil {
		return nil, e
	}
	if s.RightRoof != nil && s.RightRoof.Shape != "none" && (s.Shape == "flat" || s.Shape == "mono" || s.Shape == "gable" || s.Shape == "vault" || s.Shape == "skylight") {
		return snowPairedCases(s)
	}
	var cases []SnowCase
	add := func(title, clause, note string, parts ...SnowInterval) {
		cases = append(cases, snowCase(title, snowNormSource+", "+clause, note, parts...))
	}
	L, mu := s.Span, snowRoofMu(s.Angle)
	switch s.Shape {
	case "flat", "mono":
		add("1. Равномерная", "рисунок Б.1, вариант 1", "μ по таблице Б.1", snowMuInterval(0, L, mu))
		{
			title, source, note := snowUnevenSource(s)
			add(title, source, note, snowMuInterval(0, L/2, .9*mu), snowMuInterval(L/2, L, 1.1*mu))
		}
	case "gable":
		add("1. Равномерная", "рисунок Б.1, вариант 1", "μ по таблице Б.1", snowMuInterval(0, L, mu))
		if s.Angle >= 15 && s.Angle <= 40 {
			add("2. Неравномерная", "рисунок Б.1, вариант 2", "0,75μ на половине, 1,25μ на второй половине", snowMuInterval(0, L/2, .75*mu), snowMuInterval(L/2, L, 1.25*mu))
			add("3. Неравномерная (зеркальная)", "рисунок Б.1, вариант 2", "Обратное направление", snowMuInterval(0, L/2, 1.25*mu), snowMuInterval(L/2, L, .75*mu))
		}
		if math.Abs(s.Angle) > 20 {
			add("Частичное загружение левого ската", "п. 10.4; Б.1", "Дополнительный вариант неблагоприятной работы несущих элементов", snowMuInterval(0, L/2, mu), snowMuInterval(L/2, L, 0))
			add("Частичное загружение правого ската", "п. 10.4; Б.1", "Дополнительный зеркальный вариант", snowMuInterval(0, L/2, 0), snowMuInterval(L/2, L, mu))
		}
		if s.Angle >= 10 && s.Angle <= 30 && s.RidgeEquipment {
			add("Вариант 3 Б.1. Накопление у конька", "рисунок Б.1, вариант 3", "Применяется при ходовых мостиках/светоаэрационных устройствах у конька", snowMuInterval(0, L/2, .6), snowMuInterval(L/2, 3*L/4, 1.4), snowMuInterval(3*L/4, L, .6))
		}
		{
			title, source, note := snowUnevenSource(s)
			add(title, source, note, snowMuInterval(0, L/2, .9*mu), snowMuInterval(L/2, L, 1.1*mu))
		}
	case "vault":
		f := s.Rise
		v1 := snowResample(s, func(x float64) float64 { return math.Max(0, math.Cos(1.5*snowVaultAngle(L, f, x)*math.Pi/180)) }, 64)
		add("1. Симметричная на своде", "рисунок Б.2, вариант 1; формула Б.1", "Круговой свод: μ₁=cos(1,5α) в точках покрытия", v1...)
		if snowVaultAngle(L, f, 0) > 20 {
			lPart, rPart := append([]SnowInterval(nil), v1...), append([]SnowInterval(nil), v1...)
			for i := range lPart {
				if lPart[i].X1 >= L/2 {
					lPart[i].Mu = 0
				}
			}
			for i := range rPart {
				if rPart[i].X2 <= L/2 {
					rPart[i].Mu = 0
				}
			}
			add("Частичное загружение половины свода", "п. 10.4; Б.2", "Дополнительное частичное загружение", lPart...)
			add("Частичное загружение (зеркальное)", "п. 10.4; Б.2", "Дополнительное зеркальное частичное загружение", rPart...)
		}
		// For circular arches, μ2 is defined at the 30°, 60° and end sections; interpolation is linear.
		edge := snowVaultAngle(L, f, 0)
		r := L*L/(8*f) + f/2
		// For a circular vault B.2 defines μ₂ at A (30°), B (60°), C (end)
		// and requires LINEAR interpolation between those roof sections.
		type controlPoint struct{ x, q float64 }
		edgeQ := 2 * math.Sin(3*edge*math.Pi/180)
		if edge >= 60 {
			edgeQ = 0
		}
		pts := []controlPoint{{0, edgeQ}}
		if edge > 60 {
			pts = append(pts, controlPoint{L/2 - r*math.Sin(60*math.Pi/180), 0})
		}
		if edge > 30 {
			pts = append(pts, controlPoint{L/2 - r*math.Sin(30*math.Pi/180), 2})
		}
		pts = append(pts, controlPoint{L / 2, 0})
		mu2 := func(x float64) float64 {
			x = math.Min(x, L-x)
			for i := 1; i < len(pts); i++ {
				if x <= pts[i].x {
					q := (x - pts[i-1].x) / (pts[i].x - pts[i-1].x)
					return pts[i-1].q + (pts[i].q-pts[i-1].q)*q
				}
			}
			return 0
		}
		v2 := snowResample(s, func(x float64) float64 {
			if x <= L/2 {
				return mu2(x)
			}
			return .5 * mu2(x)
		}, 64)
		add("2. Несимметричная на своде", "рисунок Б.2, вариант 2; формула Б.1", "Линейная интерполяция между контрольными сечениями кругового свода", v2...)
		v3 := snowResample(s, func(x float64) float64 {
			if x <= L/2 {
				return .5 * mu2(x)
			}
			return mu2(x)
		}, 64)
		add("3. Несимметричная (зеркальная)", "рисунок Б.2, вариант 2", "Зеркальное направление снегопереноса", v3...)
	case "shed":
		// СП 20.13330.2016, Б.4, рисунок Б.7. Два варианта загружения:
		// 1) μ=1 по всей горизонтальной проекции;
		// 2) 1,4 и 0,6 на последовательных половинах каждого шага l.
		// H - высота конька над землёй, не амплитуда «зуба» и не фактор μ.
		// Для обычного шедового покрытия ce=1 по п. 10.6.
		L *= float64(s.Count)
		add("1. Равномерное загружение", "Б.4, рисунок Б.7, вариант 1", "μ=1,0 по всей горизонтальной проекции шедового покрытия", snowMuInterval(0, L, 1))
		arr := make([]SnowInterval, 0, 2*s.Count)
		for j := 0; j < s.Count; j++ {
			x := float64(j) * s.Span
			arr = append(arr, snowMuInterval(x, x+s.Span/2, 1.4), snowMuInterval(x+s.Span/2, x+s.Span, .6))
		}
		add("2. Неравномерное загружение шедов", "Б.4, рисунок Б.7, вариант 2", "На каждой паре полупролётов μ=1,4 и μ=0,6 одновременно", arr...)
	case "multi_gable":
		n := s.Count
		L *= float64(n)
		add("1. Равномерная по пролётам", "рисунок Б.8, вариант 1", "μ=1; ширина одного пролёта указана в исходных данных", snowMuInterval(0, L, 1))
		if s.Angle >= 15 {
			arr := make([]SnowInterval, 0, 4*n)
			for j := 0; j < 4*n; j++ {
				a := float64(j) * s.Span / 4
				b := a + s.Span/4
				v := .6
				if n == 2 {
					if (a+b)/2 > .5*s.Span && (a+b)/2 < 1.5*s.Span {
						v = 1.4
					}
				}
				if n >= 3 {
					u := math.Mod((a+b)/2, s.Span) / s.Span
					if u < .25 || u > .75 {
						v = 1.4
					}
				}
				arr = append(arr, snowMuInterval(a, b, v))
			}
			variant := 2
			if n >= 3 {
				variant = 3
			}
			add(fmt.Sprintf("2. Неравномерная (вариант %d)", variant), "рисунок Б.8", "Б.5: распределение 0,6/1,4 в пролётах", arr...)
		}
		if s.Angle >= 30 {
			arr := make([]SnowInterval, 0, n*32)
			for i := 0; i < n*32; i++ {
				a := float64(i) * s.Span / 32
				b := a + s.Span/32
				u := math.Mod((a+b)/2, s.Span) / s.Span
				v := 2 * math.Abs(2*u-1)
				arr = append(arr, snowMuInterval(a, b, v))
			}
			add("3. Вариант по рисунку Б.9", "Б.5, отсылка к рисунку Б.9, вариант 2", "Треугольная эпюра: максимум μ=2 у ендов, нуль в середине пролёта", arr...)
		}
		if math.Abs(s.Angle) > 20 {
			// П. 10.4: one slope in EVERY bay, including the reversed
			// and alternating directions. These are independent load cases.
			for scheme := 0; scheme < 4; scheme++ {
				arr := make([]SnowInterval, 0, 2*n)
				for bay := 0; bay < n; bay++ {
					loaded := 0
					switch scheme {
					case 1:
						loaded = 1
					case 2:
						loaded = bay % 2
					case 3:
						loaded = 1 - bay%2
					}
					for side := 0; side < 2; side++ {
						start := (float64(bay) + float64(side)/2) * s.Span
						value := 0.0
						if side == loaded {
							value = mu
						}
						arr = append(arr, snowMuInterval(start, start+s.Span/2, value))
					}
				}
				add(fmt.Sprintf("Частичная нагрузка: один скат в каждом пролёте, схема %d", scheme+1), "п. 10.4; Б.5", "Один нагруженный скат каждого пролёта; чередование и зеркальные направления", arr...)
			}
			// A second normative arrangement consists of TWO adjacent slopes
			// at the valley between bays, repeated after skipping a bay.
			for offset := 0; offset < 2 && offset < n-1; offset++ {
				arr := make([]SnowInterval, 0, 2*n)
				for j := 0; j < n; j++ {
					for k := 0; k < 2; k++ {
						start := (float64(j) + float64(k)/2) * s.Span
						value := 0.0
						for seam := offset; seam < n-1; seam += 2 {
							if j == seam && k == 1 || j == seam+1 && k == 0 {
								value = mu
							}
						}
						arr = append(arr, snowMuInterval(start, start+s.Span/2, value))
					}
				}
				add(fmt.Sprintf("Частичная нагрузка: два смежных ската через пролёт, схема %d", offset+1), "п. 10.4; Б.5", "Пары соседних скатов у ендов с чередованием пролётов", arr...)
			}
			// Additional local one-slope-per-selected-bay checks.
			for bay := 0; bay < n; bay++ {
				for side := 0; side < 2; side++ {
					arr := make([]SnowInterval, 0, 2*n)
					for j := 0; j < n; j++ {
						for k := 0; k < 2; k++ {
							x1 := float64(j)*s.Span + float64(k)*s.Span/2
							val := 0.0
							if j == bay && k == side {
								val = snowRoofMu(s.Angle)
							}
							arr = append(arr, snowMuInterval(x1, x1+s.Span/2, val))
						}
					}
					add(fmt.Sprintf("Частичная нагрузка: пролёт %d, скат %d", bay+1, side+1), "п. 10.4; Б.5", "Проверка односторонней загрузки ската", arr...)
				}
			}
		}
		{
			arr := make([]SnowInterval, 0, n)
			for i := 0; i < n; i++ {
				v := .9
				if i%2 == 1 {
					v = 1.1
				}
				arr = append(arr, snowMuInterval(float64(i)*s.Span, float64(i+1)*s.Span, v))
			}
			add("Проверочная неравномерная по пролётам", "Б.5", "Проверочная схема: необходимость зависит от чувствительности несущей конструкции", arr...)
		}
	case "multi_vault":
		n := s.Count
		L *= float64(n)
		add("1. Равномерная по сводам", "рисунок Б.9, вариант 1", "μ=1", snowMuInterval(0, L, 1))
		if s.Rise/s.Span > .1 {
			arr := make([]SnowInterval, 0, 64*n)
			for j := 0; j < 64*n; j++ {
				x1 := float64(j) * s.Span / 64
				x2 := x1 + s.Span/64
				u := math.Mod((x1+x2)/2, s.Span) / s.Span
				v := 2 * math.Abs(2*u-1)
				arr = append(arr, snowMuInterval(x1, x2, v))
			}
			add("2. Снег в ендовах", "рисунок Б.9, вариант 2", "Треугольная эпюра, μ=2 у ендов", arr...)
		}
		// СП 20.13330.2016 п.10.4: load the halves of two neighbouring vaults when
		// the largest vault roof slope exceeds 20 degrees.
		if snowVaultAngle(s.Span, s.Rise, 0) > 20 {
			for bay := 0; bay < n-1; bay++ {
				arr := make([]SnowInterval, 0, n*2)
				for j := 0; j < n; j++ {
					for k := 0; k < 2; k++ {
						x1 := (float64(j) + float64(k)/2) * s.Span
						val := 0.0
						if j == bay && k == 1 || j == bay+1 && k == 0 {
							val = 1
						}
						arr = append(arr, snowMuInterval(x1, x1+s.Span/2, val))
					}
				}
				add(fmt.Sprintf("Частичная нагрузка: смежные своды %d и %d", bay+1, bay+2), "п. 10.4; Б.6", "Две нагруженные половины соседних сводов", arr...)
			}
		}
	case "skylight":
		// SP20 B.3, Figs B.4 and B.5; closed longitudinal lantern, two wind zones.
		a := s.SkylightWidth
		b := (L - a) / 2
		b1 := math.Min(b, s.H) // B.2 for lantern roof of negligible rise (f=0)
		if b1 <= 0 {
			return nil, fmt.Errorf("Недостаточная ширина для зоны у фонаря")
		}
		mu1 := .8
		mu2 := math.Min(4, math.Min(3*s.H/s.Ground, 1+.1*a/b))
		mu3 := math.Min(4, math.Min(3*s.H/s.Ground, 1+.5*a/b1))
		add("1A. Равномерная (зона A)", "Б.3, рисунок Б.5, вариант 1", "В зоне А снег распределён равномерно", snowMuInterval(0, L, 1))
		add("1C. По бокам фонаря (зона C)", "Б.3, рисунок Б.5, вариант 1", fmt.Sprintf("a=%.2f; b=%.2f; μ1=%.3f; μ2=%.3f", a, b, mu1, mu2), snowMuInterval(0, b, mu2), snowMuInterval(b, b+a, mu1), snowMuInterval(b+a, L, mu2))
		add("2A. Накопление в центре (зона A)", "Б.3, рисунок Б.5, вариант 2", fmt.Sprintf("b1=%.2f; μ3=%.3f", b1, mu3), snowMuInterval(0, b-b1, 1), snowMuInterval(b-b1, b+a+b1, mu3), snowMuInterval(b+a+b1, L, 1))
		add("2C. У боковых стенок (зона C)", "Б.3, рисунок Б.5, вариант 2", "Две зоны переноса у боковых стен фонаря", snowMuInterval(0, b-b1, 1), snowMuInterval(b-b1, b, mu3), snowMuInterval(b, b+a, 0), snowMuInterval(b+a, b+a+b1, mu3), snowMuInterval(b+a+b1, L, 1))
		// Примечание 3 к Б.3: у фонарей на широких плоских скатах
		// требуется проверка дополнительного местного заноса как у перепада Б.8.
		// Не суммируем независимые варианты; оба направления рассматриваем отдельно.
		if b > 24 && math.Abs(s.Angle) < 15 {
			w := s
			w.Shape = "wall"
			w.RightRoof = nil
			w.H = s.H
			w.Span = b
			w.Upper = 0
			w.Lower = b
			w.LowerShape = "flat"
			w.UpperRoof = SnowRoofPart{Shape: "flat", Span: 0}
			w.LowerRoof = SnowRoofPart{Shape: "flat", Span: b, Width: s.Width, RoofHeight: s.RoofHeight}
			w.CE = 1 // п. 10.9(б): местный занос не снижается коэффициентом сноса
			other, err := snowCases(w)
			if err != nil {
				return nil, fmt.Errorf("Дополнительный расчёт по Б.3 и Б.8: %w", err)
			}
			for _, c := range other {
				if !strings.Contains(c.Title, "У высокой стены") {
					continue
				}
				muAt := func(u float64) float64 {
					for _, it := range c.Intervals {
						if u >= it.X1 && (u < it.X2 || u == b && math.Abs(it.X2-b) < 1e-7) {
							return it.Mu
						}
					}
					return 1
				}
				for _, side := range []string{"левого", "правого"} {
					arr := make([]SnowInterval, 0, 128)
					for i := 0; i < 128; i++ {
						x1 := L * float64(i) / 128
						x2 := L * float64(i+1) / 128
						mid := (x1 + x2) / 2
						mu := 1.
						if side == "левого" && mid < b {
							mu = muAt(b - mid)
						}
						if side == "правого" && mid > b+a {
							mu = muAt(mid - (b + a))
						}
						arr = append(arr, snowMuInterval(x1, x2, mu))
					}
					add("Дополнительное местное загружение у "+side+" ската фонаря", "Б.3, примечание 3; Б.8, рисунок Б.11", "Плоский широкий скат b>24 м, расчёт переноса снега по модели перепада высот; альтернативный, не суммируемый с основной схемой", arr...)
				}
			}
		}
		add("Частичное загружение у левого ската фонаря", "п. 10.4; Б.3", "Нагружена одна область шириной b", snowMuInterval(0, b, mu2), snowMuInterval(b, L, 0))
		add("Частичное загружение у правого ската фонаря", "п. 10.4; Б.3", "Нагружена одна область шириной b", snowMuInterval(0, b+a, 0), snowMuInterval(b+a, L, mu2))
	case "parapet":
		// СП 20.13330.2016, Б.13, рисунок Б.16: у парапета
		// μ=2h/S0, но не более 3; ширина местного отложения b=2h.
		// Эта схема применяется при h>S0/2. Для меньшего h остаётся μ=1.
		add("1. Основная нагрузка", "Б.13, рисунок Б.16", "Равномерная снеговая нагрузка без местного снегозаноса", snowMuInterval(0, L, 1))
		if s.H > s.Ground/2 {
			muP := math.Min(3, 2*s.H/s.Ground)
			b := math.Min(L, 2*s.H)
			pieces := make([]SnowInterval, 0, 64)
			for j := 0; j < 64; j++ {
				a := float64(j) * L / 64
				d := float64(j+1) * L / 64
				// The interval begins at the exact peak; stepping the linear
				// ramp conservatively preserves μmax for every saved/exported case.
				x := a
				v := 1.
				if x < b {
					v = muP + (1-muP)*x/(2*s.H)
				}
				pieces = append(pieces, snowMuInterval(a, d, v))
			}
			add("2. Местная нагрузка у парапета", "Б.13, рисунок Б.16", fmt.Sprintf("h=%s м, μmax=%s, b=2h=%s м; после b нагрузка μ=1", decimalInput(s.H, 3), decimalInput(muP, 3), decimalInput(2*s.H, 3)), pieces...)
		}
		// Б.13(в): при широком плоском покрытии или высоком парапете
		// дополнительно проверяется самостоятельная схема перепада Б.8.
		// Не складываем независимые эпюры Б.13 и Б.8.
		if L > 24 || s.H > 1.2 {
			w := s
			w.Shape = "wall"
			w.Upper = 0
			w.Lower = L
			w.LowerShape = "flat"
			w.RightRoof = nil
			w.LowerRoof = SnowRoofPart{Shape: "flat", Span: L, Width: s.Width, RoofHeight: s.RoofHeight}
			w.UpperRoof = SnowRoofPart{Shape: "flat", Span: 0}
			w.CE = 1 // п. 10.9(б): снос снега у парапета не уменьшает нагрузку
			if other, e := snowCases(w); e == nil {
				for _, c := range other {
					if strings.Contains(c.Title, "У высокой стены") {
						capMu := 3 * s.H / s.Ground
						for i := range c.Intervals {
							c.Intervals[i].Mu = math.Min(c.Intervals[i].Mu, capMu)
						}
						c.Title = "3. Дополнительная проверка снегозаноса"
						c.Source = snowNormSource + ", Б.13(в) и Б.8, рисунок Б.11"
						c.Note += fmt.Sprintf(". Независимый неблагоприятный вариант по Б.13(в), ограничение μ≤3h/S0=%s", decimalInput(capMu, 3))
						cases = append(cases, c)
					}
				}
			} else {
				return nil, fmt.Errorf("Не удалось рассчитать дополнительный занос по Б.13(в): %w", e)
			}
		}
	case "wall", "step":
		// ЗАГЛУШКА / ЭКСПЕРИМЕНТ: пользовательский режим shape="step"
		// («Здания с перепадом высот») ещё не прошёл отдельную нормативную
		// проработку. Не считать его завершённым модулем и не включать как
		// доступную схему в следующих релизах без согласования с пользователем.
		// ВАЖНО: здесь же используется вспомогательная модель shape="wall"
		// для отдельных проверок Б.8/Б.13; эта пометка НЕ означает, что нужно
		// отключать расчёт парапетов (shape="parapet").
		// B.8: raised roof and lowered roof are separate load cases. Never add a drift to base μ by default.
		upperRoof, lowerRoof := s.stepRoofs()
		h := math.Min(s.H, 8)
		u := math.Max(0, upperRoof.Span)
		d := lowerRoof.Span
		upperAngle := upperRoof.resolvedAngle()
		lowerAngle := lowerRoof.resolvedAngle()
		m1 := .4
		if s.Shape == "wall" {
			m1 = 0
		} else if upperAngle > 20 {
			m1 = .3
		}
		m2 := .4
		if lowerAngle > 20 || lowerRoof.Shape == "vault" && lowerRoof.Rise/d > 1./8 {
			m2 = .3
		}
		if d < 21 {
			k1 := math.Sqrt(d / 21)
			k2 := 1 - math.Min(35, lowerAngle)/35
			if lowerRoof.Shape == "flat" || lowerRoof.Shape == "vault" {
				k2 = 1
			}
			k3 := math.Max(.3, 1-lowerAngle/30)
			if lowerRoof.Shape == "flat" || lowerRoof.Shape == "vault" {
				k3 = 1
			}
			m2 = math.Max(.1, .5*k1*k2*k3)
		}
		// Point b) of B.8: the contributing distances are the roof widths, when there are no longitudinal skylights.
		// Per B.8(b), for a lower roof without a parapet, the snow-transport
		// length must not exceed three times its transverse width.
		dTransport := math.Min(d, 3*s.Width)
		mt := 1 + (m1*u+m2*dTransport)/h
		// S₀ = Sg in appendix B (10.4, note 2). ce and ct affect Sн,
		// never the B.8 drift geometry, upper bound or applicability.
		baseQ := s.Ground
		maxMu := math.Min(2*h/baseQ, 4)
		if math.Max(u, d) > 48 {
			maxMu = math.Min(2*h/baseQ, 4+2*math.Min(1, math.Max(0, (math.Max(u, d)-48)/24)))
		}
		if mt > maxMu {
			mt = maxMu
		}
		if mt < 1 {
			mt = 1
		}
		b := math.Min(2*h, 16.)
		if mt > 2*h/baseQ {
			den := 2*h/baseQ - 1 + 2*m2
			if den > 0 {
				b = 2 * h * (mt - 1 + 2*m2) / den
			}
		}
		b = math.Min(b, math.Min(5*h, 16))
		if b <= 0 {
			b = math.Min(d, 1)
		}
		mStart := 1 - 2*m2
		if b < d && mt > 2*h/baseQ {
			mStart = (d - .5*mt*b) / (d - .5*b)
			if mStart < .2 {
				mStart = .2
			}
		}
		if mStart < 0 {
			mStart = 0
		}
		prependUpper := func(c SnowCase) SnowCase {
			if s.Shape == "step" && u > 0 {
				var part []SnowInterval
				up := s
				up.Shape = upperRoof.Shape
				up.Span = u
				up.Angle = upperAngle
				up.Rise = upperRoof.Rise
				up.SlopeMode = "angle"
				bases, err := snowCases(up)
				if err == nil && len(bases) > 0 {
					for _, v := range bases[0].Intervals {
						part = append(part, snowMuInterval(v.X1-u, v.X2-u, v.Mu))
					}
				} else {
					part = []SnowInterval{snowMuInterval(-u, 0, 1)}
				}
				c.Intervals = append(part, c.Intervals...)
				c.Note += ". Верхнее покрытие задано независимо; x<0 — верхнее, x≥0 — нижнее"
			}
			return c
		}
		if lowerRoof.Shape == "mono" || lowerRoof.Shape == "gable" || lowerRoof.Shape == "vault" {
			subtype := s
			subtype.Shape = lowerRoof.Shape
			subtype.Span = d
			subtype.Angle = lowerAngle
			subtype.Rise = lowerRoof.Rise
			subtype.SlopeMode = "angle"
			if lowerRoof.Shape == "vault" && subtype.Rise > d/2 {
				return nil, fmt.Errorf("Для нижнего свода необходимо f ≤ d/2")
			}
			basics, err := snowCases(subtype)
			if err != nil {
				return nil, err
			}
			for _, c := range basics {
				c.Title = "Основное нижнее: " + c.Title
				c.Note += ". Схема Б.8, перечисление а (альтернатива заносу)"
				cases = append(cases, prependUpper(c))
			}
		} else {
			add("1. Основное загружение нижнего покрытия", "Б.8, перечисление а; Б.1", "Без местного заноса (отдельный расчётный вариант)", snowMuInterval(0, d, 1))
			cases[len(cases)-1] = prependUpper(cases[len(cases)-1])
		}
		if s.Shape == "step" && u > 0 && upperRoof.Shape != "flat" {
			top := s
			top.Shape = upperRoof.Shape
			top.Span = u
			top.Angle = upperAngle
			top.Rise = upperRoof.Rise
			top.SlopeMode = "angle"
			if topCases, err := snowCases(top); err == nil && len(topCases) > 1 {
				for _, tc := range topCases[1:] {
					parts := make([]SnowInterval, 0, len(tc.Intervals)+2)
					for _, v := range tc.Intervals {
						parts = append(parts, snowMuInterval(v.X1-u, v.X2-u, v.Mu))
					}
					// The lower roof carries the basic uniform arrangement, not an added drift.
					parts = append(parts, snowMuInterval(0, d, 1))
					cases = append(cases, snowCase("Основное верхнее: "+tc.Title, tc.Source, tc.Note+". Вариант верхней кровли рассматривается отдельно от снегового мешка", parts...))
				}
			}
		}
		// B.8 note 3 explicitly excludes local drift at small height differences.
		if s.H < baseQ/2 {
			return cases, nil
		}
		parts := []SnowInterval{}
		n := 30
		for i := 0; i < n; i++ {
			x1 := d * float64(i) / float64(n)
			x2 := d * float64(i+1) / float64(n)
			x := x1
			v := mStart
			if x < b {
				v = mt + (mStart-mt)*x/b
			}
			parts = append(parts, snowMuInterval(x1, x2, v))
		}
		add("2. У высокой стены / перепада", "рисунок Б.11; формулы Б.5–Б.6", fmt.Sprintf("μmax=%.3f; b=%.3f м; m₁=%.3f; m₂=%.3f. Перепад от стены слева, локальная эпюра на нижнем покрытии", mt, b, m1, m2), parts...)
		cases[len(cases)-1] = prependUpper(cases[len(cases)-1])
	case "cone":
		// B.12, figure B.15 and equations B.12–B.13. Variant 2 is spatial:
		// μ₂ = C_r2 * (z/r) * sin β, 0≤β≤180°; the meridional epure shown
		// in reports is ONLY the section at β=90°, not a blanket 2D load.
		uniform := snowRoofMu(s.Angle)
		add("1. Равномерная на конусе", "рисунок Б.15, вариант 1; таблица Б.3", "Коэффициент μ₁ по таблице Б.3; равномерно по круговой кровле", snowMuInterval(0, L, uniform))
		// At alpha >=60 the normative coefficient is zero on the whole
		// cone. Do not extrapolate the B.13 drift expression beyond 60°.
		if s.Angle >= 60 {
			return cases, nil
		}
		if s.Angle > 7 {
			c := 0.
			if s.Angle <= 30 {
				c = 2.15 * math.Cbrt(s.Angle*math.Pi/180)
			} else {
				c = 1.7 * (30 / s.Angle)
			}
			radius := L / 2
			arr := make([]SnowInterval, 0, 80)
			for i := 0; i < 80; i++ {
				a := L * float64(i) / 80
				b := L * float64(i+1) / 80
				z := (a+b)/2 - radius
				mu := 0.
				if z > 0 {
					mu = c * z / radius
				}
				arr = append(arr, snowMuInterval(a, b, mu))
			}
			add("2. Пространственный занос: β=90°", "рисунок Б.15, вариант 2; формулы (Б.12), (Б.13)", fmt.Sprintf("Показан только меридиональный разрез при β=90°; пространственный закон μ₂(z,β)=C·(z/r)·sinβ; C=%s. Полное распределение приведено в отдельной полярной таблице Excel.", decimalInput(c, 3)), arr...)
			p := &cases[len(cases)-1]
			for i := 0; i <= 12; i++ {
				r := radius * float64(i) / 12
				for j := 0; j <= 24; j++ {
					beta := 360 * float64(j) / 24
					mu := math.Max(0, c*r/radius*math.Sin(beta*math.Pi/180))
					p.Polar = append(p.Polar, SnowPolarPoint{Radius: r, Beta: beta, Mu: mu})
				}
			}
		}
	case "sphere_cone":
		return snowSphereConeCases(s)
	case "dome":
		d := s.Span
		f := s.Rise
		uniform := func(alpha float64) float64 { return snowRoofMu(alpha) }
		R := d*d/(8*f) + f/2
		r := d / 2
		aEnd := 2 * math.Atan2(2*f, d) * 180 / math.Pi
		// At f>D/2 the spherical cap has an overhang; its largest
		// horizontal projection is 2R, not the diameter between supports.
		// The steep outer annulus (including local slopes above 90°)
		// receives μ=0 under B.11. Preserve its zero region in the epure.
		projection := d
		if f > d/2 {
			projection = 2 * R
			r = projection / 2
		}
		chartCalc := s
		chartCalc.Span = projection
		pointAngle := func(z float64) float64 { return math.Asin(math.Min(1, math.Abs(z)/R)) * 180 / math.Pi }
		mu1 := func(x float64) float64 { return uniform(pointAngle(x - r)) }
		v1 := snowResample(chartCalc, mu1, 80)
		add("1. Симметричная на сферической крыше", "Б.11, рисунок Б.14, вариант 1; таблица Б.2", fmt.Sprintf("Коэффициент μ1 определяется по местному уклону α; R=%s м, угол у опоры=%s° (при α≥60° μ=0).", decimalInput(R, 3), decimalInput(aEnd, 2)), v1...)
		if f/d > .05 {
			cr := 2.55 - math.Exp(.8-14*f/d)
			r1 := R * math.Sin(math.Pi/6)
			mu2 := func(z, beta float64) float64 {
				if math.Sin(beta*math.Pi/180) <= 0 {
					return 0
				}
				a := pointAngle(z)
				v := 0.
				if z <= r1 {
					v = cr * math.Pow(z/r1, 2)
				} else if a <= 45 {
					v = cr + (a-30)/15*(1.5-cr)
				} else if a < 60 {
					v = 1.5 * (60 - a) / 15
				}
				return math.Max(0, v*math.Sin(beta*math.Pi/180))
			}
			arr := snowResample(chartCalc, func(x float64) float64 {
				if x <= r {
					return 0
				}
				return mu2(x-r, 90)
			}, 100)
			add("2. Пространственный занос на сферической крыше", "Б.11, рисунок Б.14, вариант 2; формула (Б.10)", fmt.Sprintf("μ2(z,β) = Cr1×(z/r1)²×sinβ в пределах z≤r1; Cr1=%s, r1=%s м; за r1 — линейная интерполяция по углу α (30°–60°). Эпюра — сечение β=90°, полная таблица по полярным координатам в Excel.", decimalInput(cr, 4), decimalInput(r1, 3)), arr...)
			p := &cases[len(cases)-1]
			for i := 0; i <= 20; i++ {
				z := r * float64(i) / 20
				for j := 0; j <= 24; j++ {
					beta := 360 * float64(j) / 24
					p.Polar = append(p.Polar, SnowPolarPoint{Radius: z, Beta: beta, Mu: mu2(z, beta)})
				}
			}
			if s.DomeObstacles {
				mu3 := func(z, beta float64) float64 {
					return math.Max(0, 3*math.Sqrt(2*f/d)*math.Sin(3*pointAngle(z)*math.Pi/180)*math.Sin(beta*math.Pi/180))
				}
				arr3 := snowResample(chartCalc, func(x float64) float64 {
					if x <= r {
						return 0
					}
					return mu3(x-r, 90)
				}, 100)
				add("3. Занос при препятствиях или шероховатости", "Б.11, рисунок Б.14, вариант 3; формула (Б.11)", "Вариант для шероховатых покрытий, фонарей, снегозадерживающих преград или защищённости от ветра; μ2=3√(2f/d)·sin(3α)·sinβ. Эпюра — только меридиональное сечение β=90°.", arr3...)
				p3 := &cases[len(cases)-1]
				for i := 0; i <= 20; i++ {
					z := r * float64(i) / 20
					for j := 0; j <= 24; j++ {
						beta := 360 * float64(j) / 24
						p3.Polar = append(p3.Polar, SnowPolarPoint{Radius: z, Beta: beta, Mu: mu3(z, beta)})
					}
				}
			}
		}

	default:
		return nil, fmt.Errorf("Неизвестный тип покрытия")
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("Для выбранного покрытия нет расчётных схем")
	}
	return cases, nil
}

// snowEdgeAtAbutment returns the actual roof elevation at the common wall.
// The sign of an one-pitch roof means the direction of inclination when
// walking from left to right. H for a mono roof denotes its upper edge.
func snowEdgeAtAbutment(p SnowRoofPart, rightEnd bool) float64 {
	if p.Shape == "flat" {
		return p.RoofHeight
	}
	if p.Shape == "mono" {
		positive := p.resolvedAngle() >= 0
		if positive == rightEnd {
			return p.RoofHeight
		}
		return p.RoofHeight - math.Abs(p.Rise)
	}
	return p.RoofHeight - math.Abs(p.Rise)
}

// snowPairedCases forms one horizontal axis for two independent neighbouring roof
// profiles. The same-level and step cases remain alternatives, never added.
func snowPairedCases(s SnowCalc) ([]SnowCase, error) {
	right := *s.RightRoof
	left := SnowRoofPart{Shape: s.Shape, Span: s.Span, Angle: s.Angle, Rise: s.Rise, SlopeMode: s.SlopeMode, RoofHeight: s.RoofHeight, Width: s.Width, SkylightWidth: s.SkylightWidth, LanternHeight: s.H, WarmRoof: s.WarmRoof}
	// The abutment is at the RIGHT end of the left roof and LEFT end of the right roof.
	left.EdgeLevel = snowEdgeAtAbutment(left, true)
	if right.RoofHeight > 0 {
		right.EdgeLevel = snowEdgeAtAbutment(right, false)
	}
	if right.RoofHeight == 0 && right.EdgeLevel == 0 {
		right.EdgeLevel = left.EdgeLevel
	}
	individual := func(p SnowRoofPart) ([]SnowCase, error) {
		x := s
		x.RightRoof = nil
		// Ridge-access accumulation in B.1 is an option of a standalone
		// single-span gable, not a generic option for adjoining roof parts.
		x.RidgeEquipment = false
		x.Shape = p.Shape
		x.Span = p.Span
		x.Angle = p.resolvedAngle()
		x.Rise = p.Rise
		x.SlopeMode = "angle"
		x.Count = 2
		x.SkylightWidth = p.SkylightWidth
		x.H = p.LanternHeight
		x.WarmRoof = p.WarmRoof
		x.RoofHeight = p.RoofHeight
		if p.Width > 0 {
			x.Width = p.Width
		}
		return snowCases(x)
	}
	lc, err := individual(left)
	if err != nil {
		return nil, err
	}
	rc, err := individual(right)
	if err != nil {
		return nil, err
	}
	move := func(parts []SnowInterval, delta float64) []SnowInterval {
		out := make([]SnowInterval, 0, len(parts))
		for _, v := range parts {
			out = append(out, snowMuInterval(v.X1+delta, v.X2+delta, v.Mu))
		}
		return out
	}
	combine := func(title, source, note string, l, r SnowCase) SnowCase {
		// Preserve the normative μ of each roof: reduced load must not carry
		// the thermal factor. A common ct is applied outside these intervals.
		p := append(move(l.Intervals, 0), move(r.Intervals, left.Span)...)
		return snowCase(title, source, note, p...)
	}
	out := []SnowCase{combine("1. Основное загружение обоих пролётов", "приложение Б, нормативные схемы каждого покрытия", "Загрузка каждого пролёта по его собственной форме; совместно показаны основные варианты", lc[0], rc[0])}
	for i := 1; i < len(lc); i++ {
		out = append(out, combine(fmt.Sprintf("Левый пролёт: %s", lc[i].Title), lc[i].Source, lc[i].Note, lc[i], rc[0]))
	}
	for i := 1; i < len(rc); i++ {
		out = append(out, combine(fmt.Sprintf("Правый пролёт: %s", rc[i].Title), rc[i].Source, rc[i].Note, lc[0], rc[i]))
	}
	// Two adjacent longitudinal lanterns can both exhibit uneven snow
	// distributions. The previous implementation only varied one roof at a
	// time and missed the documented C+C case (mu2 on BOTH side strips).
	// Combine corresponding exposure zones A/A and C/C; do not arbitrarily
	// mix A and C without the longitudinal alignment geometry.
	if left.Shape == "skylight" && right.Shape == "skylight" {
		zone := func(c SnowCase) string {
			if strings.HasPrefix(c.Title, "1A.") || strings.HasPrefix(c.Title, "2A.") {
				return "A"
			}
			if strings.HasPrefix(c.Title, "1C.") || strings.HasPrefix(c.Title, "2C.") {
				return "C"
			}
			return ""
		}
		for i := 1; i < len(lc); i++ {
			zi := zone(lc[i])
			if zi == "" {
				continue
			}
			for j := 1; j < len(rc); j++ {
				if zone(rc[j]) != zi {
					continue
				}
				out = append(out, combine(
					fmt.Sprintf("Оба фонаря, зона %s: левый %s + правый %s", zi, lc[i].Title, rc[j].Title),
					"СП 20.13330.2016, Б.3, рисунок Б.5 (совместное рассмотрение одного продольного участка)",
					"Совместное альтернативное загружение при совпадении продольных зон фонарей; для смещённых фонарей требуется проверить их положение в плане.",
					lc[i], rc[j]))
			}
		}
	}
	if math.Abs(left.EdgeLevel-right.EdgeLevel) > 1e-6 {
		step := s
		step.RightRoof = nil
		step.Shape = "step"
		step.CE = 1
		isLeftHigh := left.EdgeLevel > right.EdgeLevel
		if isLeftHigh {
			step.UpperRoof = left
			step.LowerRoof = right
		} else {
			step.UpperRoof = right
			step.LowerRoof = left
		}
		step.H = step.UpperRoof.EdgeLevel - step.LowerRoof.EdgeLevel
		step.Upper = step.UpperRoof.Span
		step.Lower = step.LowerRoof.Span
		// With longitudinal lanterns B.8 specifies l' = max(0,l* - 2d).
		// Their position within each roof is not explicitly captured in this model.
		// Full lengths are retained as a conservative bound; never describe the
		// result as an independently verified spatial lantern-step combination.
		step.LowerShape = step.LowerRoof.Shape
		if step.LowerRoof.Width > 0 {
			step.Width = step.LowerRoof.Width
		}
		step.Angle = step.LowerRoof.resolvedAngle()
		step.Rise = step.LowerRoof.Rise
		step.Span = left.Span + right.Span
		stepCases, err := snowCases(step)
		if err != nil {
			return nil, fmt.Errorf("перепад между пролётами: %w", err)
		}
		for _, c := range stepCases {
			if !strings.Contains(c.Title, "У высокой стены / перепада") {
				continue
			}
			c.Title = "Занос у перепада высот: " + c.Title
			if left.Shape == "skylight" || right.Shape == "skylight" {
				c.Note += " ПРЕДВАРИТЕЛЬНО: длина снегопереноса принята без сокращения продольным фонарём; требуется уточнить по Б.8 с фактическим расположением фонарей."
			}
			if isLeftHigh {
				for i := range c.Intervals {
					c.Intervals[i].X1 += left.Span
					c.Intervals[i].X2 += left.Span
				}
			} else {
				for i := range c.Intervals {
					v := &c.Intervals[i]
					x1 := v.X1
					x2 := v.X2
					v.X1 = left.Span - x2
					v.X2 = left.Span - x1
				}
			}
			sort.Slice(c.Intervals, func(i, j int) bool { return c.Intervals[i].X1 < c.Intervals[j].X1 })
			out = append(out, c)
		}
	}
	return out, nil
}

// SnowEnvironment computes the normative exposure/thermal coefficients from
// user-understandable conditions. Missing winter climate data MUST NOT imply
// a reduction. SP 20.13330.2016 clauses 10.5-10.11, 11.1.6.
func SnowEnvironment(s SnowCalc) SnowCalc {
	s.CE, s.CT = 1, 1
	// A single thermal coefficient cannot describe two different roofs.
	// Reduction is allowed only when BOTH roof segments satisfy 10.10;
	// for unlike constructions ct=1 is intentionally conservative.
	if s.RightRoof != nil && s.RightRoof.Shape != "none" && s.WarmRoof && s.RightRoof.WarmRoof && snowRoofGradient(s) > .03 && s.RightRoof.Rise/s.RightRoof.Span > .03 {
		s.CT = .8
	}
	if s.RightRoof == nil && s.WarmRoof && snowRoofGradient(s) > .03 {
		// The UI checkbox explicitly confirms uninsulated, high heat output and drainage.
		s.CT = .8
	}
	// 10.9: reduction unavailable if T January > -5, or if temperature not known.
	if !s.Exposed || !s.JanuarySet || s.JanuaryTemp > -5 {
		return s
	}
	if s.RightRoof != nil && s.RightRoof.Shape != "none" {
		return s
	}
	// 10.8: only for uniform load arrangements on spherical/conical roof in plan.
	if s.Shape == "cone" || s.Shape == "dome" || s.Shape == "sphere_cone" {
		if s.Shape == "sphere_cone" {
			return s
		}
		// Clause 10.8 is expressly restricted to UNIFORM load arrangements.
		// For nonuniform dome/cone designs retain ce=1 conservatively so a
		// single project-wide coefficient cannot understate the drift load.
		if (s.Shape == "cone" && s.Angle > 7) || (s.Shape == "dome" && s.Rise/s.Span > .05) {
			return s
		}
		d := s.Span
		switch {
		case d <= 60:
			s.CE = .85
		case d >= 100:
			s.CE = 1
		default:
			s.CE = .85 + .00375*(d-60)
		}
		return s
	}
	// 10.9(b): 10.7 wind reduction not applicable at walls and roof steps.
	if !s.WinterWindSet || s.WinterWind <= 3 || s.RoofHeight <= 0 || s.Terrain == "С" {
		return s
	}
	if s.Shape == "wall" || s.Shape == "step" || s.Shape == "skylight" {
		return s
	}
	if !(s.Shape == "flat" || s.Shape == "mono" || s.Shape == "gable" || s.Shape == "vault" || s.Shape == "multi_gable" || s.Shape == "multi_vault") {
		return s
	}
	if s.Shape == "mono" || s.Shape == "gable" || s.Shape == "multi_gable" {
		if math.Abs(s.Angle) > 10 {
			return s
		}
	} else if (s.Shape == "vault" || s.Shape == "multi_vault") && s.Rise/s.Span > .05 {
		return s
	}
	length := s.Span
	if s.Shape == "multi_gable" || s.Shape == "multi_vault" {
		length *= float64(s.Count)
	}
	if length <= 0 || s.Width <= 0 {
		return s
	}
	b := math.Min(length, s.Width)
	lmax := math.Max(length, s.Width)
	lc := math.Min(100, 2*b-b*b/lmax)
	h := s.RoofHeight
	heights := []float64{5, 10, 20, 40, 60, 80, 100, 150, 200, 250, 300}
	kA := []float64{.75, 1, 1.25, 1.5, 1.7, 1.85, 2, 2.25, 2.45, 2.65, 2.75}
	kB := []float64{.5, .65, .85, 1.1, 1.3, 1.45, 1.6, 1.9, 2.1, 2.3, 2.5}
	vals := kB
	if s.Terrain == "А" {
		vals = kA
	}
	k := vals[0]
	if h >= 300 {
		k = vals[len(vals)-1]
	} else if h > 5 {
		for i := 1; i < len(heights); i++ {
			if h <= heights[i] {
				q := (h - heights[i-1]) / (heights[i] - heights[i-1])
				k = vals[i-1] + q*(vals[i]-vals[i-1])
				break
			}
		}
	}
	// Table 10.2: January ranges [-15,-5],[-25,-15],(<-25);
	// winter wind ranges (3,4], (4,6], (>6).
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
	kvA := [][]float64{{1.4, 1.3, 1.3}, {1.4, 1.3, 1.2}, {1.3, 1.2, 1.2}}
	kvB := [][]float64{{1.4, 1.4, 1.3}, {1.4, 1.4, 1.3}, {1.4, 1.3, 1.2}}
	kv := kvB[tempRow][windCol]
	if s.Terrain == "А" {
		kv = kvA[tempRow][windCol]
	}
	ce := (kv - .4*math.Sqrt(k)) * (.8 + .002*lc)
	s.CE = math.Min(1, math.Max(.5, ce))
	return s
}
func snowRoofGradient(s SnowCalc) float64 {
	switch s.Shape {
	case "flat", "wall":
		return 0
	case "vault", "multi_vault", "dome":
		if s.Span > 0 {
			return 2 * s.Rise / s.Span
		}
		return 0
	default:
		return math.Abs(math.Tan(s.Angle * math.Pi / 180))
	}
}
func (s SnowCalc) ReducedApplicable() bool { return s.JanuarySet && s.JanuaryTemp <= -5 }
func SnowCompute(s SnowCalc) (SnowCalc, error) {
	if s.Shape == "sphere_cone" {
		if s.BetaStep == 0 {
			s.BetaStep = 15
		}
		if s.SlopeMode == "height" && s.SkylightWidth > 0 {
			s.Angle = math.Atan2(s.Rise, s.SkylightWidth) * 180 / math.Pi
		}
		sphereConeExpandDiameter(&s)
		s.Width = s.Span // круглое в плане покрытие, отдельной длины B нет
	}
	snowResolveSlope(&s)
	s = SnowEnvironment(s)
	cases, err := snowCases(s)
	if err != nil {
		return s, err
	}
	cases = snowAddSectorPartialCases(s, cases)
	s.Cases = cases
	return s, nil
}
func (s SnowCalc) Q(mu float64) float64        { return s.Ground * s.CE * s.CT * mu }
func (s SnowCalc) QDesign(mu float64) float64  { return s.Q(mu) * s.Gamma }
func (s SnowCalc) QReduced(mu float64) float64 { return .5 * s.Ground * mu }
func (s SnowCalc) MaxMu(c SnowCase) float64 {
	max := c.PeakMu
	for _, in := range c.Intervals {
		if in.Mu > max {
			max = in.Mu
		}
	}
	return max
}
func (s SnowCalc) ReportName() string {
	return "Снеговая нагрузка: " + snowShapeTitle(s.Shape)
}
func (s SnowCalc) FormulaLines() []string {
	lines := []string{fmt.Sprintf("Sн = μ·ce·ct·Sg = μ × %s × %s × %s кПа (п. 10.1 СП 20.13330.2016)", decimalInput(s.CE, 3), decimalInput(s.CT, 3), decimalInput(s.Ground, 3)), fmt.Sprintf("Sр = γf·Sн; γf = %s (п. 10.12 СП 20.13330.2016)", decimalInput(s.Gamma, 3)), fmt.Sprintf("Высота до конька H = %s м; ce = %s; ct = %s; тепловыделение: %s", decimalInput(s.RoofHeight, 2), decimalInput(s.CE, 3), decimalInput(s.CT, 3), yesSnow(s.WarmRoof)), "Нагрузки относятся к горизонтальной проекции покрытия (п. 10.1 СП 20.13330.2016)."}
	if s.ReducedApplicable() {
		lines = append(lines, "Пониженная нормативная: 0,5·μ·Sg; ce=ct=1 (п. 10.11 СП 20.13330.2016)")
	} else {
		lines = append(lines, snowJanuaryReductionReason(s))
	}
	if s.ClimateOrigin == "sp131" && s.ClimateReferencePlace != "" {
		ref := "Приняты климатические параметры по СП 131.13330.2025, таблицам 5.1 и 7.1: " + s.ClimateReferencePlace
		if s.ClimateReferenceRegion != "" {
			ref += " (" + s.ClimateReferenceRegion + ")"
		}
		lines = append(lines, ref)
	} else if s.ClimateOrigin == "manual" {
		lines = append(lines, "Климатические параметры заданы пользователем")
	}
	return lines
}
func yesSnow(ok bool) string {
	if ok {
		return "да"
	}
	return "нет"
}
func (s SnowCalc) SiteLabel() string {
	if strings.TrimSpace(s.Region) == "" {
		return "Sg введено пользователем"
	}
	return "Снеговой район " + s.Region
}

// CurrentSnowForReport refreshes inherited regional snow data when the site
// has changed after saving the roof setup. The project stores only explicit
// user choices; the reports always use the current climatic input.
func (p Project) CurrentSnowForReport() *SnowCalc {
	if p.SnowCalc == nil || !p.SnowCalc.Enabled {
		return nil
	}
	s := *p.SnowCalc
	if s.ClimateOrigin == "sp131" && (p.Site.ClimateJanuarySet || p.Site.ClimateWindSet) {
		s.InheritClimate(p.Site)
		if recalculated, e := SnowCompute(s); e == nil {
			s = recalculated
		}
	}
	if !s.GroundLocal {
		region, sg, _ := p.SnowInputs()
		if sg > 0 {
			s.Region, s.Ground = region, sg
			if calc, e := SnowCompute(s); e == nil {
				s = calc
			}
		}
	}
	return &s
}
