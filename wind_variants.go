package main

import (
	"fmt"
	"math"
)

// A wind variant is a single wind direction. Its simultaneous wall zones
// belong to the same variant; they are not additive alternatives.
// Each zone retains a complete core result and calculation profile so that
// no graphical or reporting code has to invent aerodynamic coefficients.
type WindZoneCalculation struct {
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	From        float64         `json:"from_m"`
	To          float64         `json:"to_m"`
	Surface     string          `json:"surface,omitempty"`     // wall or roof
	Alternative string          `json:"alternative,omitempty"` // complete roof load case 1/2
	Rectangles  []WindRoofRect  `json:"rectangles,omitempty"`
	Calculation WindCalculation `json:"calculation"`
}
type WindVariant struct {
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	WindAlong string                `json:"wind_along"`
	Reverse   bool                  `json:"reverse,omitempty"`
	AlongM    float64               `json:"along_m"`
	AcrossM   float64               `json:"across_m"`
	E         float64               `json:"e_m,omitempty"`
	Zones     []WindZoneCalculation `json:"zones"`
}

// WindProfileHeights is the independently tested sampling convention.
// Step == 0 is the approved default: 0, 5, 7.5, ... H.
// Positive step is a maximum interval, with 0, H and 5 m mandatory.
func WindProfileHeights(height, step float64) ([]float64, error) {
	if math.IsNaN(height) || math.IsInf(height, 0) || height <= 0 {
		return nil, fmt.Errorf("Высота сооружения должна быть положительной")
	}
	if math.IsNaN(step) || math.IsInf(step, 0) || step < 0 {
		return nil, fmt.Errorf("Шаг эпюры должен быть положительным или автоматическим")
	}
	if height > 100000 || (step > 0 && height/step > 20000) {
		return nil, fmt.Errorf("Слишком много точек эпюры; увеличьте шаг")
	}
	zs := []float64{0}
	start := 0.0
	appendTo := func(end, interval float64) {
		if end <= start {
			return
		}
		for start+interval < end-1e-9 {
			start += interval
			zs = append(zs, start)
		}
		if end > start+1e-9 {
			zs = append(zs, end)
			start = end
		}
	}
	if step == 0 {
		if height <= 5 {
			appendTo(height, height)
		} else {
			appendTo(5, 5)
			appendTo(height, 2.5)
		}
	} else {
		if height <= 5 {
			appendTo(height, step)
		} else {
			appendTo(5, step)
			appendTo(height, step)
		}
	}
	return zs, nil
}

func windZoneDefinition(base WindInput, code, name string, from, to float64, rho float64) (WindZoneCalculation, error) {
	in := base
	in.AeroZone = code
	if base.Geometry == "building" {
		in.AeroTable = "В.2"
	} else {
		in.AeroTable = "В.1"
		in.Depth = 0
	}
	in.Rho = rho
	in.Chi = base.Height
	calc, err := CalculateWindProfile(in)
	if err != nil {
		return WindZoneCalculation{}, fmt.Errorf("Зона %s: %w", code, err)
	}
	return WindZoneCalculation{Code: code, Name: name, From: from, To: to, Calculation: calc}, nil
}

func windBuildingVariant(base WindInput, id, name string, along, across float64, reverse bool) (WindVariant, error) {
	in := base
	in.Geometry = "building"
	in.Depth = across                    // СП 20.13330.2016, 11.1.5: поперечный размер к ветру, НЕ вдоль потока
	e := math.Min(across, 2*base.Height) // approved figure В.3
	v := WindVariant{ID: id, Name: name, WindAlong: id, AlongM: along, AcrossM: across, E: e, Reverse: reverse}
	a := math.Min(along, e/5)
	b := math.Min(along, e)
	specs := []struct {
		code, name        string
		start, end, width float64
	}{
		{"D", "Наветренная торцевая стена", 0, across, across},
		{"E", "Подветренная торцевая стена", 0, across, across},
		{"A", "Боковая стена: краевая зона A", 0, a, a},
		{"B", "Боковая стена: зона B", a, b, b - a},
		{"C", "Боковая стена: зона C", b, along, along - b},
	}
	for _, s := range specs {
		if s.width < 1e-9 {
			continue
		} // an empty zone is not a load case
		start, end := s.start, s.end
		if reverse && s.code != "D" && s.code != "E" {
			start, end = along-s.end, along-s.start
		}
		zone, err := windZoneDefinition(in, s.code, s.name, start, end, s.width)
		if err != nil {
			return WindVariant{}, err
		}
		v.Zones = append(v.Zones, zone)
	}
	return v, nil
}

// CalculateWindVariants uses exactly the verified single-zone mathematical
// core, with no new wind formulas. The roof sketch is geometry guidance only;
// roof coefficients are intentionally not synthesized from wall coefficients.
func CalculateWindVariants(base WindInput) (WindCalculation, error) {
	if base.Height <= 0 || base.Rho <= 0 {
		return WindCalculation{}, fmt.Errorf("Укажите высоту H и поперечный размер L сооружения больше нуля")
	}
	// Unusual geometry is warned about, not force-clamped or refused; roof
	// coefficients are not part of the currently implemented wall cases.
	if base.ProfileStep < 0 {
		return WindCalculation{}, fmt.Errorf("Шаг эпюры должен быть положительным")
	}
	if base.PulseMode != "mean" {
		base.PulseMode = "nonresonant"
	}
	base.KMode = "table"
	var variants []WindVariant
	if base.Geometry == "building" {
		if base.Depth <= 0 {
			return WindCalculation{}, fmt.Errorf("Укажите длину здания B больше нуля")
		}
		// Два самостоятельных направления ветра для симметричного в плане здания.
		// B — вдоль конька, L — поперёк конька. Противоположные направления
		// не создают дублирующих загружений.
		for _, c := range []struct {
			id, name      string
			along, across float64
			reverse       bool
		}{
			{"RIDGE", "Ветер вдоль конька", base.Depth, base.Rho, false},
			{"CROSS", "Ветер поперёк конька", base.Rho, base.Depth, false},
		} {
			v, err := windBuildingVariant(base, c.id, c.name, c.along, c.across, c.reverse)
			if err != nil {
				return WindCalculation{}, err
			}
			variants = append(variants, v)
		}
	} else if base.Geometry == "free_wall" {
		v := WindVariant{ID: "W", Name: "Отдельно стоящая стена", WindAlong: "W", AlongM: base.Rho, AcrossM: base.Rho}
		borders := []float64{0, 0.3 * base.Height, 2.3 * base.Height, 6.3 * base.Height, base.Rho}
		names := []string{"Краевая зона A", "Зона B", "Зона C", "Зона D"}
		for j, zone := range []string{"A", "B", "C", "D"} {
			from := math.Min(borders[j], base.Rho)
			to := math.Min(borders[j+1], base.Rho)
			if to-from <= 1e-9 {
				continue
			}
			z, err := windZoneDefinition(base, zone, names[j], from, to, to-from)
			if err != nil {
				return WindCalculation{}, err
			}
			v.Zones = append(v.Zones, z)
		}
		variants = append(variants, v)
	} else {
		return WindCalculation{}, fmt.Errorf("Выберите «Здание» или «Отдельно стоящая стена»")
	}
	if len(variants) == 0 || len(variants[0].Zones) == 0 {
		return WindCalculation{}, fmt.Errorf("Нет расчётных зон для заданной геометрии")
	}
	if base.Geometry == "building" && base.RoofRise > 0 {
		beta, e := windRoofAngles(base)
		if e != nil {
			return WindCalculation{}, e
		}
		db, e := getWindDB()
		if e != nil {
			return WindCalculation{}, e
		}
		forkMin, forkMax, e := db.roofForkBounds()
		if e != nil {
			return WindCalculation{}, e
		}
		for vi := range variants {
			alts := []int{1}
			if variants[vi].ID == "CROSS" && beta >= forkMin && beta <= forkMax {
				alts = []int{1, 2}
			}
			for _, alt := range alts {
				zones, e := windRoofZones(base, variants[vi], alt)
				if e != nil {
					return WindCalculation{}, e
				}
				variants[vi].Zones = append(variants[vi].Zones, zones...)
			}
		}
	}
	first := variants[0].Zones[0].Calculation
	first.Input = base
	first.Variants = variants
	first.SelectedVariant = 0
	for i, v := range variants {
		if v.ID == "CROSS" {
			first.SelectedVariant = i
			break
		}
	}
	first.SelectedZone = 0
	// The module offers an engineering choice, not a fabricated exemption
	// from the frequency criterion in SP 20.13330.2016, 11.1.8.
	if base.PulseMode == "mean" {
		first.Result.Warnings = append(first.Result.Warnings,
			"Пульсационная составляющая не учтена по решению инженера; см. СП 20.13330.2016, 11.1.2 и 11.1.8")
	} else if base.Geometry == "free_wall" {
		first.Result.Warnings = append(first.Result.Warnings,
			"Для отдельно стоящей стены примечание 1 к 11.1.8 не освобождает от определения частот; применение (11.5) требует обоснования")
	} else {
		first.Result.Warnings = append(first.Result.Warnings,
			"Режим (11.5): проверьте условия примечания 1 к 11.1.8 или обоснуйте выполнение f₁ > f_lim; конструктивный материал Ebala не запрашивает")
	}
	if base.Geometry == "building" && (base.RoofRise < 0 || base.RoofRise >= base.Height) {
		first.Result.Warnings = append(first.Result.Warnings, fmt.Sprintf("Нетипичное соотношение высот H=%.3f и h=%.3f м: проверьте геометрию двускатного покрытия", base.Height, base.RoofRise))
	}
	return first, nil
}

func (c WindCalculation) SelectedWindZone() *WindZoneCalculation {
	if len(c.Variants) == 0 {
		return nil
	}
	if c.SelectedVariant < 0 || c.SelectedVariant >= len(c.Variants) {
		return nil
	}
	v := &c.Variants[c.SelectedVariant]
	if c.SelectedZone < 0 || c.SelectedZone >= len(v.Zones) {
		return nil
	}
	return &v.Zones[c.SelectedZone]
}

// Expands a saved building/standalone-wall object for independent reporting.
// No zones are summed: simultaneously acting pressures remain separate
// surfaces within their common wind-direction heading.
func windReportZones(c WindCalculation, title string) ([]WindCalculation, []string) {
	if len(c.Variants) == 0 {
		return []WindCalculation{c}, []string{title}
	}
	var out []WindCalculation
	var names []string
	for _, v := range c.Variants {
		for _, z := range v.Zones {
			exported := z.Calculation
			exported.Input.RoofSpan = c.Input.Rho
			exported.Input.RoofLength = c.Input.Depth
			exported.Input.RoofRise = c.Input.RoofRise
			exported.Input.RoofFlow = v.ID
			exported.Input.StructuralType = c.Input.StructuralType
			exported.Input.Delta = c.Input.Delta
			out = append(out, exported)
			if z.Surface == "roof" {
				branch := ""
				if v.ID == "CROSS" && z.Alternative != "" {
					branch = ". Знаковый вариант " + z.Alternative
				}
				names = append(names, fmt.Sprintf("%s. %s%s. Крыша, зона %s: %s", title, v.Name, branch, z.Code, z.Name))
			} else {
				names = append(names, fmt.Sprintf("%s. %s. Стена, зона %s: %s (%s–%s м)", title, v.Name, z.Code, z.Name, decimalInput(z.From, 3), decimalInput(z.To, 3)))
			}
		}
	}
	return out, names
}
func windExpandReports(calculations []WindCalculation, titles []string) ([]WindCalculation, []string) {
	var out []WindCalculation
	var names []string
	for i, c := range calculations {
		title := fmt.Sprintf("Ветровая нагрузка %d", i+1)
		if i < len(titles) {
			title = titles[i]
		}
		zs, n := windReportZones(c, title)
		out = append(out, zs...)
		names = append(names, n...)
	}
	return out, names
}
