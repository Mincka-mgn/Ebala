package main

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

// WindRoofRect is a footprint of one normative roof zone on the plan.
// X is the direction of the wind, Y is transverse; coordinates are metres.
// A single normative zone may consist of several disconnected rectangles.
type WindRoofRect struct {
	X0 float64 `json:"x0_m"`
	X1 float64 `json:"x1_m"`
	Y0 float64 `json:"y0_m"`
	Y1 float64 `json:"y1_m"`
}

func windRoofRectsValid(rects []WindRoofRect) []WindRoofRect {
	out := make([]WindRoofRect, 0, len(rects))
	for _, r := range rects {
		if r.X1-r.X0 > 1e-8 && r.Y1-r.Y0 > 1e-8 {
			out = append(out, r)
		}
	}
	return out
}

// The aerodynamic table's support angles and its two-sign interval are
// loaded from the approved JSON, not copied as Go constants.
func (db *windDB) roofAxes(table string) ([]float64, error) {
	seen := map[float64]bool{}
	for _, row := range db.Refs {
		if row.TableID != table || row.Input1Name != "beta_deg" {
			continue
		}
		b, err := strconv.ParseFloat(row.Input1Value, 64)
		if err != nil {
			return nil, fmt.Errorf("Повреждён угол таблицы %s: %w", table, err)
		}
		seen[b] = true
	}
	axes := make([]float64, 0, len(seen))
	for a := range seen {
		axes = append(axes, a)
	}
	sort.Float64s(axes)
	if len(axes) < 2 {
		return nil, fmt.Errorf("Недостаточно нормативных узлов таблицы %s", table)
	}
	return axes, nil
}
func (db *windDB) roofForkBounds() (float64, float64, error) {
	minAngle, maxAngle := math.Inf(1), math.Inf(-1)
	seen := map[string]bool{}
	for _, row := range db.Refs {
		if row.TableID != "В.3а" || row.Variant != "2" || row.Input1Name != "beta_deg" {
			continue
		}
		a, err := strconv.ParseFloat(row.Input1Value, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("Неверный угол альтернативы В.3а: %w", err)
		}
		if a < minAngle {
			minAngle = a
		}
		if a > maxAngle {
			maxAngle = a
		}
		seen[row.SchemeID] = true
	}
	// All zones F–J need their alternative records at both endpoints so
	// complete cases can be formed without missing/implicit coefficients.
	for _, zone := range []string{"F", "G", "H", "I", "J"} {
		if !seen[zone] {
			return 0, 0, fmt.Errorf("Для зоны %s нет второго знакового варианта В.3а", zone)
		}
	}
	if math.IsInf(minAngle, 0) || math.IsInf(maxAngle, 0) || minAngle >= maxAngle {
		return 0, 0, fmt.Errorf("Неполные границы знаковой альтернативы В.3а")
	}
	return minAngle, maxAngle, nil
}
func (db *windDB) roofTableAxisGroups() (low, high, ridge []float64, forkMin, forkMax float64, err error) {
	cross, err := db.roofAxes("В.3а")
	if err != nil {
		return
	}
	ridge, err = db.roofAxes("В.3б")
	if err != nil {
		return
	}
	forkMin, forkMax, err = db.roofForkBounds()
	if err != nil {
		return
	}
	for _, a := range cross {
		if a <= forkMax {
			low = append(low, a)
		}
		if a >= forkMin {
			high = append(high, a)
		}
	}
	if len(low) < 2 || len(high) < 2 {
		err = fmt.Errorf("Не удалось сформировать интервалы таблицы В.3а")
	}
	return
}

// All normative aerodynamic coefficients are read from the unmodified
// SP20_wind_all_in_one.json. In В.3а the tabulated sign alternatives at
// beta=15..30 are two distinct *complete* load cases, not per-zone maxima.
// Above 30 the continuous positive branch starts at alternative 2 of 30°.
func (db *windDB) roofCoefficient(table, zone string, beta float64, alternative int) (float64, error) {
	if table != "В.3а" && table != "В.3б" {
		return 0, fmt.Errorf("Неизвестная таблица кровли %s", table)
	}
	if math.IsNaN(beta) || math.IsInf(beta, 0) || beta < 0 {
		return 0, fmt.Errorf("Недопустимый угол кровли β=%.3f°", beta)
	}
	forkMin, forkMax := 0.0, 0.0
	if table == "В.3а" {
		var err error
		forkMin, forkMax, err = db.roofForkBounds()
		if err != nil {
			return 0, err
		}
	}
	points := make(map[float64]map[int]float64)
	for _, r := range db.Refs {
		if r.TableID != table || r.SchemeID != zone || r.Input1Name != "beta_deg" {
			continue
		}
		angle, e1 := strconv.ParseFloat(r.Input1Value, 64)
		val, e2 := strconv.ParseFloat(r.ResultValue, 64)
		if e1 != nil || e2 != nil {
			return 0, fmt.Errorf("Повреждена нормативная запись %s зоны %s", table, zone)
		}
		branch := 1
		if table == "В.3а" {
			branch, e1 = strconv.Atoi(r.Variant)
			if e1 != nil {
				return 0, fmt.Errorf("Нет номера знакового варианта в таблице %s", table)
			}
		}
		if points[angle] == nil {
			points[angle] = map[int]float64{}
		}
		if _, present := points[angle][branch]; present {
			return 0, fmt.Errorf("Дублируется коэффициент %s/%s/%.1f/%d", table, zone, angle, branch)
		}
		points[angle][branch] = val
	}
	angles := make([]float64, 0, len(points))
	for x := range points {
		angles = append(angles, x)
	}
	sort.Float64s(angles)
	if len(angles) < 2 {
		return 0, fmt.Errorf("Нет табличных данных %s/%s", table, zone)
	}
	if beta < angles[0] || beta > angles[len(angles)-1] {
		return 0, fmt.Errorf("Нет данных %s/%s при β=%.3f°", table, zone, beta)
	}
	// For beta=15..30, branch 1 and branch 2 are independent. For beta>30,
	// continue from the positive branch at 30 degrees to the 45 degree row.
	branchAt := func(angle float64) int {
		if table == "В.3б" {
			return 1
		}
		if beta > forkMax && angle == forkMax {
			return 2
		}
		if beta >= forkMin && beta <= forkMax {
			return alternative
		}
		return 1
	}
	if table == "В.3а" && beta >= forkMin && beta <= forkMax && alternative != 1 && alternative != 2 {
		return 0, fmt.Errorf("Таблица В.3а: выберите целый вариант 1 или 2")
	}
	lookup := func(angle float64) (float64, error) {
		branch := branchAt(angle)
		x, ok := points[angle][branch]
		if !ok {
			return 0, fmt.Errorf("Для %s/%s, β=%.1f°, ветвь %d отсутствует", table, zone, angle, branch)
		}
		return x, nil
	}
	for k, angle := range angles {
		if math.Abs(beta-angle) < 1e-10 {
			return lookup(angle)
		}
		if k > 0 && beta < angle {
			lo := angles[k-1]
			hi := angle
			x, e := lookup(lo)
			if e != nil {
				return 0, e
			}
			y, e := lookup(hi)
			if e != nil {
				return 0, e
			}
			return x + (y-x)*(beta-lo)/(hi-lo), nil
		}
	}
	return lookup(angles[len(angles)-1])
}

func windRoofAngles(base WindInput) (float64, error) {
	if base.Rho <= 0 || base.Depth <= 0 || base.Height <= 0 || base.RoofRise < 0 {
		return 0, fmt.Errorf("Для двускатной крыши нужны L, B, H > 0, h ≥ 0")
	}
	beta := math.Atan2(2*base.RoofRise, base.Rho) * 180 / math.Pi
	// The calculation core validates the actual upper tabulated angle for
	// each requested aerodynamic scheme; no arbitrary extension is allowed.
	return beta, nil
}

func windRoofZoneCalculation(base WindInput, v WindVariant, code string, alt int, rects []WindRoofRect, label string) (WindZoneCalculation, error) {
	rects = windRoofRectsValid(rects)
	if len(rects) == 0 {
		return WindZoneCalculation{}, nil
	}
	in := base
	in.Geometry = "building"
	in.AeroZone = code
	in.AeroTable = "В.3б"
	if v.ID == "CROSS" {
		in.AeroTable = "В.3а"
	}
	in.RoofFlow = v.ID
	in.RoofBranch = strconv.Itoa(alt)
	in.RoofSpan = base.Rho
	in.RoofLength = base.Depth
	if v.ID == "CROSS" {
		in.Depth = base.Depth
	} else {
		in.Depth = base.Rho
	}
	// A roof zone is a horizontal/near-horizontal calculation surface, with
	// the two dimensions of the reference rectangle per table 11.7 (xoy).
	// For disconnected corner patches the dimensions of ONE patch are used.
	in.Chi = rects[0].X1 - rects[0].X0
	in.Rho = rects[0].Y1 - rects[0].Y0
	in.Z = base.Height
	in.RoofPitchDeg = math.Atan2(2*base.RoofRise, base.Rho) * 180 / math.Pi
	// The roof is evaluated at the upper equivalent height. Rendering two
	// identical edge/ridge ordinates does not fabricate a wall-height profile.
	c, err := CalculateWind(in)
	if err != nil {
		return WindZoneCalculation{}, fmt.Errorf("Крыша, %s, зона %s, вариант %d: %w", v.Name, code, alt, err)
	}
	if base.PulseMode != "mean" {
		c.Result.Warnings = append(c.Result.Warnings, "Для пульсаций зона покрытия представлена прямоугольной проекцией xoy (таблица 11.7); для существенно наклонных поверхностей проверить применимость этого допущения")
	}
	if base.RoofRise >= base.Height {
		c.Result.Warnings = append(c.Result.Warnings, fmt.Sprintf("Высота подъёма двускатной кровли h=%.3f м не меньше высоты конька H=%.3f м: проверьте геометрию; давление на уровне H рассчитано по заданным числам", base.RoofRise, base.Height))
	}
	// Five or more intervals between the eaves and ridge. The normative
	// roof pressure remains evaluated at H; the denser altitude grid does
	// NOT introduce a non-approved pressure law for the inclined surface.
	zs, err := WindRoofProfileHeights(base.Height, base.RoofRise, base.ProfileStep)
	if err != nil {
		return WindZoneCalculation{}, err
	}
	c.Profile = make([]WindProfilePoint, 0, len(zs))
	for _, z := range zs {
		c.Profile = append(c.Profile, WindProfilePoint{Z: z, Result: c.Result})
	}
	from, to := rects[0].X0, rects[0].X1
	for _, r := range rects {
		if r.X0 < from {
			from = r.X0
		}
		if r.X1 > to {
			to = r.X1
		}
	}
	return WindZoneCalculation{Code: code, Name: label, From: from, To: to, Surface: "roof", Alternative: strconv.Itoa(alt), Rectangles: rects, Calculation: c}, nil
}

func windRoofZones(base WindInput, v WindVariant, alt int) ([]WindZoneCalculation, error) {
	var defs []struct {
		code, name string
		rects      []WindRoofRect
	}
	rect := func(x0, x1, y0, y1 float64) WindRoofRect { return WindRoofRect{x0, x1, y0, y1} }
	if v.ID == "CROSS" {
		// Fig В.4a: flow across ridge, X=0..L, Y=0..B.
		L, B := base.Rho, base.Depth
		e := math.Min(B, 2*base.Height)
		a := math.Min(e/10, L/2)
		f := math.Min(e/4, B/2)
		defs = []struct {
			code, name string
			rects      []WindRoofRect
		}{
			{"F", "Крыша: наветренные углы", []WindRoofRect{rect(0, a, 0, f), rect(0, a, B-f, B)}},
			{"G", "Крыша: край наветренного ската", []WindRoofRect{rect(0, a, f, B-f)}},
			{"H", "Крыша: наветренный скат", []WindRoofRect{rect(a, L/2, 0, B)}},
			{"J", "Крыша: приконьковая полоса подветренного ската", []WindRoofRect{rect(L/2, math.Min(L, L/2+e/10), 0, B)}},
			{"I", "Крыша: подветренный скат", []WindRoofRect{rect(math.Min(L, L/2+e/10), L, 0, B)}},
		}
	} else {
		// Fig В.4b: flow along ridge, X=0..B, Y=0..L.
		B, L := base.Depth, base.Rho
		e := math.Min(L, 2*base.Height)
		a := math.Min(e/10, B)
		b := math.Min(e/2, B)
		f := math.Min(e/4, L/2)
		defs = []struct {
			code, name string
			rects      []WindRoofRect
		}{
			{"F", "Крыша: наветренные углы", []WindRoofRect{rect(0, a, 0, f), rect(0, a, L-f, L)}},
			{"G", "Крыша: наветренная полоса", []WindRoofRect{rect(0, a, f, L/2), rect(0, a, L/2, L-f)}},
			{"H", "Крыша: средняя полоса", []WindRoofRect{rect(a, b, 0, L/2), rect(a, b, L/2, L)}},
			{"I", "Крыша: остальная часть", []WindRoofRect{rect(b, B, 0, L/2), rect(b, B, L/2, L)}},
		}
	}
	zones := []WindZoneCalculation{}
	for _, d := range defs {
		z, err := windRoofZoneCalculation(base, v, d.code, alt, d.rects, d.name)
		if err != nil {
			return nil, err
		}
		if len(z.Rectangles) > 0 {
			zones = append(zones, z)
		}
	}
	return zones, nil
}
