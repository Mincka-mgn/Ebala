package main

// Independently testable wind-load calculation engine. Normative values and
// expression constants come exclusively from WindData JSON, never Go literals.
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

type WindInput struct {
	Region         string  `json:"region"`
	ManualW0       bool    `json:"manual_w0,omitempty"`
	W0KPa          float64 `json:"w0_kpa,omitempty"`
	Terrain        string  `json:"terrain"`
	Geometry       string  `json:"geometry"` // building or tower
	Height         float64 `json:"height_m"`
	Depth          float64 `json:"depth_m"`
	Z              float64 `json:"z_m"`
	AeroTable      string  `json:"aero_table"` // В.1, В.2 or manual
	AeroZone       string  `json:"aero_zone"`
	ManualC        float64 `json:"manual_c,omitempty"`
	KMode          string  `json:"k_mode"`     // table or formula
	PulseMode      string  `json:"pulse_mode"` // mean, nonresonant, dynamic
	Rho            float64 `json:"rho_m,omitempty"`
	Chi            float64 `json:"chi_m,omitempty"`
	Frequency      float64 `json:"frequency_hz,omitempty"`
	Delta          float64 `json:"delta,omitempty"`
	StructuralType string  `json:"structural_type,omitempty"` // user-selected type for δ per СП 20, 11.1.10
	Gamma          float64 `json:"gamma_f"`
	DisplayUnit    string  `json:"display_unit,omitempty"`
	RoofRise       float64 `json:"roof_rise_m,omitempty"`    // h: pitched roof rise; H is Height
	ProfileStep    float64 `json:"profile_step_m,omitempty"` // шаг расчётных отметок, по умолчанию 2,5 м
	RoofSpan       float64 `json:"roof_span_m,omitempty"`    // L: полный поперечный размер здания
	RoofLength     float64 `json:"roof_length_m,omitempty"`  // B: полная длина здания
	RoofFlow       string  `json:"roof_flow,omitempty"`      // RIDGE or CROSS
	RoofBranch     string  `json:"roof_branch,omitempty"`    // 1 or 2, В.3а
	RoofPitchDeg   float64 `json:"roof_pitch_deg,omitempty"` // вычислен из h и L, не отдельный исходный норматив
}
type WindResult struct {
	W0Pa        float64  `json:"w0_pa"`
	Ze          float64  `json:"equivalent_height_m"`
	K           float64  `json:"k"`
	C           float64  `json:"c"`
	Zeta        float64  `json:"zeta,omitempty"`
	Nu          float64  `json:"nu,omitempty"`
	Xi          float64  `json:"xi,omitempty"`
	Tg1         float64  `json:"tg1,omitempty"`
	FLim        float64  `json:"f_lim_hz,omitempty"`
	MeanPa      float64  `json:"mean_pa"`
	PulsePa     float64  `json:"pulse_pa"`
	NormPa      float64  `json:"norm_pa"`
	DesignPa    float64  `json:"design_pa"`
	FormulaIDs  []string `json:"formula_ids"`
	References  []string `json:"references"`
	Warnings    []string `json:"warnings,omitempty"`
	DataSHA256  string   `json:"data_sha256"`
	DataVersion string   `json:"data_version"`
}
type WindProfilePoint struct {
	Z      float64    `json:"z_m"`
	Result WindResult `json:"result"`
}
type WindCalculation struct {
	Input           WindInput          `json:"input"`
	Result          WindResult         `json:"result"`
	Profile         []WindProfilePoint `json:"profile,omitempty"`
	Variants        []WindVariant      `json:"variants,omitempty"`
	SelectedVariant int                `json:"selected_variant,omitempty"`
	SelectedZone    int                `json:"selected_zone,omitempty"`
}
type windFormula struct {
	ID            string `json:"id"`
	Expression    string `json:"expression"`
	SourceRef     string `json:"source_ref"`
	Approximation struct {
		ValidityTg1   []float64 `json:"validity_checked_Tg1"`
		ValidityDelta []float64 `json:"validity_checked_delta"`
	} `json:"approximation"`
}
type windTable struct {
	ID        string               `json:"id"`
	Kind      string               `json:"kind"`
	Columns   []string             `json:"columns"`
	Rows      [][]interface{}      `json:"rows"`
	Arguments map[string][]float64 `json:"arguments"`
	Values    [][]float64          `json:"values"`
}
type windRefRow struct {
	TableID     string `json:"table_id"`
	SchemeID    string `json:"scheme_id"`
	ResultName  string `json:"result_name"`
	ResultValue string `json:"result_value"`
	Variant     string `json:"variant"`
	Input1Name  string `json:"input_1_name"`
	Input1Value string `json:"input_1_value"`
	NormSection string `json:"norm_section"`
	Figure      string `json:"figure"`
	Condition   string `json:"condition"`
}
type windDB struct {
	Formulas map[string]windFormula
	Tables   map[string]windTable
	Refs     []windRefRow
	SHA      string
	Version  string
}

var windDBC struct {
	sync.Mutex
	path string
	db   *windDB
}

func windDataRoot() string {
	if v := os.Getenv("EBALA_WIND_DATA"); v != "" {
		return v
	}
	exe, e := os.Executable()
	if e == nil {
		return filepath.Join(filepath.Dir(exe), "WindData")
	}
	return "WindData"
}
func loadWindDB(root string) (*windDB, error) {
	files := []string{"formulas.json", "tables_section11.json", "SP20_wind_all_in_one.json"}
	db := &windDB{Formulas: map[string]windFormula{}, Tables: map[string]windTable{}, Version: "Ebala_Wind_Mathematics_v1_1"}
	var manifest struct {
		Files map[string]string `json:"files"`
	}
	mb, me := os.ReadFile(filepath.Join(root, "MANIFEST_SHA256.json"))
	if me != nil {
		return nil, fmt.Errorf("Нет контрольного манифеста ветровой базы: %w", me)
	}
	if me = json.Unmarshal(mb, &manifest); me != nil {
		return nil, me
	}
	h := sha256.New()
	for _, name := range files {
		b, e := os.ReadFile(filepath.Join(root, name))
		if e != nil {
			return nil, fmt.Errorf("Ветровая база %s: %w", name, e)
		}
		digest := sha256.Sum256(b)
		if manifest.Files[name] == "" || manifest.Files[name] != hex.EncodeToString(digest[:]) {
			return nil, fmt.Errorf("Ветровая база %s: контрольная сумма не совпадает с манифестом", name)
		}
		h.Write([]byte(name))
		h.Write(b)
		switch name {
		case "formulas.json":
			var d struct {
				Formulas []windFormula `json:"formulas"`
			}
			if e = json.Unmarshal(b, &d); e != nil {
				return nil, e
			}
			for _, f := range d.Formulas {
				if _, ok := db.Formulas[f.ID]; ok {
					return nil, fmt.Errorf("Повтор формулы %s", f.ID)
				}
				db.Formulas[f.ID] = f
			}
		case "tables_section11.json":
			var d struct {
				Tables []windTable `json:"tables"`
			}
			if e = json.Unmarshal(b, &d); e != nil {
				return nil, e
			}
			for _, t := range d.Tables {
				db.Tables[t.ID] = t
			}
		case "SP20_wind_all_in_one.json":
			var d struct {
				Tables struct {
					Rows []windRefRow `json:"rows"`
				} `json:"tables"`
				DatasetVersion string `json:"dataset_version"`
			}
			if e = json.Unmarshal(b, &d); e != nil {
				return nil, e
			}
			db.Refs = d.Tables.Rows
			db.Version += "/reference:" + d.DatasetVersion
		}
	}
	db.SHA = hex.EncodeToString(h.Sum(nil))
	if len(db.Formulas) < 100 || len(db.Tables) < 8 || len(db.Refs) < 180 {
		return nil, errors.New("Ветровая база неполная")
	}
	return db, nil
}
func getWindDB() (*windDB, error) {
	root := windDataRoot()
	windDBC.Lock()
	defer windDBC.Unlock()
	if windDBC.db != nil && windDBC.path == root {
		return windDBC.db, nil
	}
	db, e := loadWindDB(root)
	if e != nil {
		return nil, e
	}
	windDBC.db = db
	windDBC.path = root
	return db, nil
}
func numberCell(v interface{}) (float64, error) {
	f, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("Значение таблицы нечисловое: %v", v)
	}
	return f, nil
}
func (db *windDB) tableValue(table, col, rowLabel string) (float64, error) {
	t, ok := db.Tables[table]
	if !ok {
		return 0, fmt.Errorf("Нет таблицы %s", table)
	}
	ci := -1
	for i, k := range t.Columns {
		if k == col {
			ci = i
		}
	}
	if ci < 0 {
		return 0, fmt.Errorf("Нет столбца %s в %s", col, table)
	}
	for _, row := range t.Rows {
		if len(row) <= ci {
			continue
		}
		if fmt.Sprint(row[0]) == rowLabel {
			return numberCell(row[ci])
		}
	}
	return 0, fmt.Errorf("Нет строки %s в таблице %s", rowLabel, table)
}
func (db *windDB) refValue(table, zone string) (float64, error) {
	count := 0
	var v float64
	for _, r := range db.Refs {
		if r.TableID == table && r.SchemeID == zone && r.Condition == "" {
			x, e := strconv.ParseFloat(strings.ReplaceAll(r.ResultValue, ",", "."), 64)
			if e != nil {
				continue
			}
			v = x
			count++
		}
	}
	if count != 1 {
		return 0, fmt.Errorf("В справочнике %s зона %s имеет %d однозначных значений", table, zone, count)
	}
	return v, nil
}
func bracketAxis(xs []float64, x float64) (int, int, float64, error) {
	if len(xs) < 2 {
		return 0, 0, 0, errors.New("Недостаточно узлов таблицы")
	}
	if x <= xs[0] {
		return 0, 1, (x - xs[0]) / (xs[1] - xs[0]), nil
	}
	for i := 1; i < len(xs); i++ {
		if x <= xs[i] {
			return i - 1, i, (x - xs[i-1]) / (xs[i] - xs[i-1]), nil
		}
	}
	n := len(xs)
	return n - 2, n - 1, (x - xs[n-2]) / (xs[n-1] - xs[n-2]), nil
}
func (db *windDB) interpHeight(table, terrain string, z float64) (float64, error) {
	t, ok := db.Tables[table]
	if !ok {
		return 0, fmt.Errorf("Нет таблицы %s", table)
	}
	ix := -1
	for j, col := range t.Columns {
		if col == terrain {
			ix = j
		}
	}
	if ix < 1 {
		return 0, fmt.Errorf("Неизвестная местность %s", terrain)
	}
	first, e := numberCell(t.Rows[0][0])
	if e != nil {
		return 0, e
	}
	if z <= first {
		return numberCell(t.Rows[0][ix])
	} // normative <=5 band from JSON table
	for k := 1; k < len(t.Rows); k++ {
		b, e := numberCell(t.Rows[k][0])
		if e != nil {
			return 0, e
		}
		if z <= b || k == len(t.Rows)-1 {
			a, _ := numberCell(t.Rows[k-1][0])
			v0, _ := numberCell(t.Rows[k-1][ix])
			v1, _ := numberCell(t.Rows[k][ix])
			return v0 + (z-a)/(b-a)*(v1-v0), nil
		}
	}
	return 0, errors.New("Пустая таблица высот")
}
func (db *windDB) interpNu(rho, chi float64) (float64, error) {
	t, ok := db.Tables["11.6"]
	if !ok {
		return 0, errors.New("Нет таблицы 11.6")
	}
	xs := t.Arguments["rho_m"]
	ys := t.Arguments["chi_m"]
	i, j, ax, e := bracketAxis(xs, rho)
	if e != nil {
		return 0, e
	}
	m, n, ay, e := bracketAxis(ys, chi)
	if e != nil {
		return 0, e
	}
	if len(t.Values) != len(xs) || len(t.Values[i]) != len(ys) {
		return 0, errors.New("Повреждена таблица 11.6")
	}
	return (1-ax)*((1-ay)*t.Values[i][m]+ay*t.Values[i][n]) + ax*((1-ay)*t.Values[j][m]+ay*t.Values[j][n]), nil
}
func (db *windDB) calcExpr(id string, vars map[string]float64) (float64, error) {
	f, ok := db.Formulas[id]
	if !ok {
		return 0, fmt.Errorf("Не найдена формула %s", id)
	}
	ep := &windExprParser{s: f.Expression, vars: vars}
	x, e := ep.expression()
	if e != nil {
		return 0, fmt.Errorf("Формула %s: %w", id, e)
	}
	ep.spaces()
	if ep.p != len(ep.s) {
		return 0, fmt.Errorf("Формула %s: неизвестный фрагмент %q", id, ep.s[ep.p:])
	}
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return 0, fmt.Errorf("Формула %s: не определено конечное значение", id)
	}
	return x, nil
}

type windExprParser struct {
	s    string
	p    int
	vars map[string]float64
}

func (e *windExprParser) spaces() {
	for e.p < len(e.s) && unicode.IsSpace(rune(e.s[e.p])) {
		e.p++
	}
}
func (e *windExprParser) take(s string) bool {
	e.spaces()
	if strings.HasPrefix(e.s[e.p:], s) {
		e.p += len(s)
		return true
	}
	return false
}
func (e *windExprParser) expression() (float64, error) {
	x, err := e.term()
	if err != nil {
		return 0, err
	}
	for {
		if e.take("+") {
			y, er := e.term()
			if er != nil {
				return 0, er
			}
			x += y
		} else if e.take("-") {
			y, er := e.term()
			if er != nil {
				return 0, er
			}
			x -= y
		} else {
			return x, nil
		}
	}
}
func (e *windExprParser) term() (float64, error) {
	x, err := e.unary()
	if err != nil {
		return 0, err
	}
	for {
		e.spaces()
		if strings.HasPrefix(e.s[e.p:], "**") {
			return x, nil
		}
		if e.take("*") {
			y, er := e.unary()
			if er != nil {
				return 0, er
			}
			x *= y
		} else if e.take("/") {
			y, er := e.unary()
			if er != nil {
				return 0, er
			}
			if y == 0 {
				return 0, errors.New("деление на ноль")
			}
			x /= y
		} else {
			return x, nil
		}
	}
}
func (e *windExprParser) unary() (float64, error) {
	if e.take("+") {
		return e.unary()
	}
	if e.take("-") {
		v, er := e.unary()
		return -v, er
	}
	return e.power()
}
func (e *windExprParser) power() (float64, error) {
	x, err := e.atom()
	if err != nil {
		return 0, err
	}
	if e.take("**") {
		y, er := e.unary()
		if er != nil {
			return 0, er
		}
		return math.Pow(x, y), nil
	}
	return x, nil
}
func (e *windExprParser) atom() (float64, error) {
	e.spaces()
	if e.take("(") {
		x, er := e.expression()
		if er != nil {
			return 0, er
		}
		if !e.take(")") {
			return 0, errors.New("нет закрывающей скобки")
		}
		return x, nil
	}
	start := e.p
	for e.p < len(e.s) && (e.s[e.p] >= '0' && e.s[e.p] <= '9' || e.s[e.p] == '.') {
		e.p++
	}
	if e.p > start {
		return strconv.ParseFloat(e.s[start:e.p], 64)
	}
	for e.p < len(e.s) && (e.s[e.p] >= 'a' && e.s[e.p] <= 'z' || e.s[e.p] >= 'A' && e.s[e.p] <= 'Z' || e.s[e.p] >= '0' && e.s[e.p] <= '9' || e.s[e.p] == '_') {
		e.p++
	}
	if e.p == start {
		return 0, fmt.Errorf("неизвестный символ у позиции %d", e.p)
	}
	name := e.s[start:e.p]
	if name == "sqrt" {
		if !e.take("(") {
			return 0, errors.New("sqrt требует скобки")
		}
		x, er := e.expression()
		if er != nil {
			return 0, er
		}
		if !e.take(")") {
			return 0, errors.New("sqrt без закрывающей скобки")
		}
		if x < 0 {
			return 0, errors.New("sqrt отрицательного значения")
		}
		return math.Sqrt(x), nil
	}
	if name == "pi" {
		return math.Pi, nil
	}
	v, ok := e.vars[name]
	if !ok {
		return 0, fmt.Errorf("неизвестная переменная %s", name)
	}
	return v, nil
}
func (db *windDB) calculate(in WindInput) (WindResult, error) {
	var out WindResult
	out.DataSHA256 = db.SHA
	out.DataVersion = db.Version
	track := func(ids ...string) {
		for _, id := range ids {
			for _, old := range out.FormulaIDs {
				if id == old {
					goto next
				}
			}
			out.FormulaIDs = append(out.FormulaIDs, id)
			if f, ok := db.Formulas[id]; ok {
				out.References = append(out.References, "СП 20.13330.2016, "+f.SourceRef)
			}
		next:
		}
	}
	warn := func(msg string) { out.Warnings = append(out.Warnings, msg) }
	if in.Terrain != "A" && in.Terrain != "B" && in.Terrain != "C" {
		return out, errors.New("Тип местности должен быть A, B или C")
	}
	for name, v := range map[string]float64{"высота": in.Height, "глубина": in.Depth, "отметка": in.Z, "γf": in.Gamma, "c": in.ManualC, "ρ": in.Rho, "χ": in.Chi, "частота": in.Frequency, "δ": in.Delta} {
		if math.IsInf(v, 0) || math.IsNaN(v) {
			return out, fmt.Errorf("Недопустимое число: %s", name)
		}
	}
	if in.Height <= 0 || in.Z < 0 || (in.Geometry == "tower" && in.Z <= 0) {
		return out, errors.New("Высота сооружения и расчётная отметка должны быть положительными")
	}
	if in.Z > in.Height {
		warn("Расчётная отметка z выше высоты сооружения h")
	}
	if in.Gamma <= 0 {
		return out, errors.New("γf должен быть положительным")
	}
	if in.Geometry == "free_wall" {
		// Appendix В.1.1: independent wall uses the full wall height for z_e.
		out.Ze = in.Height
		track("B.1.1.wall")
	} else if in.Geometry == "tower" {
		out.Ze = in.Z
		track("11.1.5.tower")
	} else if in.Geometry == "building" {
		if in.Depth <= 0 {
			return out, errors.New("Для здания требуется положительный размер d вдоль ветра")
		}
		switch {
		case in.Height <= in.Depth:
			out.Ze = in.Height
			track("11.1.5.building_low")
		case in.Height <= 2*in.Depth:
			if in.Z >= in.Height-in.Depth {
				out.Ze = in.Height
				track("11.1.5.building_mid_upper")
			} else {
				out.Ze = in.Depth
				track("11.1.5.building_mid_lower")
			}
		default:
			if in.Z >= in.Height-in.Depth {
				out.Ze = in.Height
				track("11.1.5.building_high_upper")
			} else if in.Z <= in.Depth {
				out.Ze = in.Depth
				track("11.1.5.building_high_lower")
			} else {
				out.Ze = in.Z
				track("11.1.5.building_high_middle")
			}
		}
	} else {
		return out, errors.New("Неизвестная геометрия; допустимы building и tower")
	}
	if in.ManualW0 {
		out.W0Pa = in.W0KPa * 1000
		warn("Базовое давление w₀ назначено вручную")
	} else {
		w0, e := db.tableValue("11.1", "w0_kPa", in.Region)
		if e != nil {
			return out, e
		}
		out.W0Pa = w0 * 1000
		track("11.1.lookup_w0")
	}
	if out.W0Pa < 0 {
		return out, errors.New("Отрицательное базовое давление w₀")
	}
	switch in.AeroTable {
	case "manual":
		out.C = in.ManualC
		warn("Аэродинамический коэффициент c задан вручную")
	case "В.3а", "В.3б":
		branch := 1
		if in.RoofBranch == "2" {
			branch = 2
		}
		var er error
		out.C, er = db.roofCoefficient(in.AeroTable, in.AeroZone, in.RoofPitchDeg, branch)
		if er != nil {
			return out, er
		}
		track("B.1.2.roof_" + in.AeroTable)
		out.References = append(out.References, "СП 20.13330.2016, таблица "+in.AeroTable+", рисунок В.4")
	case "В.1", "В.2":
		var er error
		out.C, er = db.refValue(in.AeroTable, in.AeroZone)
		if er != nil {
			return out, er
		}
		if in.AeroTable == "В.1" {
			track("B.1.1.wall")
		} else {
			track("B.1.2.side_wall")
		}
		out.References = append(out.References, "СП 20.13330.2016, таблица "+in.AeroTable)
	default:
		return out, errors.New("Неизвестный источник аэродинамического коэффициента")
	}
	if t, ok := db.Tables["11.2"]; ok && len(t.Rows) > 0 {
		if maxH, e := numberCell(t.Rows[len(t.Rows)-1][0]); e == nil && out.Ze > maxH {
			warn(fmt.Sprintf("zₑ = %.3f м превышает предельную табличную высоту %.3f м (табл. 11.2)", out.Ze, maxH))
		}
	}
	var e error
	if in.KMode == "formula" {
		alpha, er := db.tableValue("11.3", ""+in.Terrain, "alpha")
		if er != nil {
			return out, er
		}
		k10, er := db.tableValue("11.3", in.Terrain, "k10")
		if er != nil {
			return out, er
		}
		out.K, e = db.calcExpr("11.4", map[string]float64{"z_e": out.Ze, "k10": k10, "alpha_terrain": alpha})
		track("11.4")
	} else if in.KMode == "table" {
		out.K, e = db.interpHeight("11.2", in.Terrain, out.Ze)
		track("11.2.lookup_k")
	} else {
		return out, errors.New("Выберите табличный или аналитический режим k")
	}
	if e != nil {
		return out, e
	}
	if out.K < 0 {
		return out, errors.New("Коэффициент k отрицателен")
	}
	out.MeanPa, e = db.calcExpr("11.2", map[string]float64{"w0": out.W0Pa, "k": out.K, "c_aerodynamic": out.C})
	if e != nil {
		return out, e
	}
	track("11.2")
	if in.PulseMode != "mean" {
		if in.Rho <= 0 || in.Chi <= 0 {
			return out, errors.New("Для пульсаций требуются размеры ρ и χ больше нуля, таблица 11.6")
		}
		out.Zeta, e = db.interpHeight("11.4", in.Terrain, out.Ze)
		if e != nil {
			return out, e
		}
		track("11.4.lookup_zeta")
		out.Nu, e = db.interpNu(in.Rho, in.Chi)
		if e != nil {
			return out, e
		}
		track("11.6.lookup_v")
		if tab, ok := db.Tables["11.6"]; ok {
			xs := tab.Arguments["rho_m"]
			ys := tab.Arguments["chi_m"]
			if len(xs) > 1 && len(ys) > 1 && (in.Rho < xs[0] || in.Rho > xs[len(xs)-1] || in.Chi < ys[0] || in.Chi > ys[len(ys)-1]) {
				warn("Размеры ρ, χ за пределами табл. 11.6: использована линейная экстраполяция")
			}
		}
		warn("Для табл. 11.6 принята билинейная интерполяция; СП прямо не предписывает её метод")
		if in.PulseMode == "nonresonant" {
			out.PulsePa, e = db.calcExpr("11.5", map[string]float64{"w_m": out.MeanPa, "zeta": out.Zeta, "v_corr": out.Nu})
			track("11.5")
		} else if in.PulseMode == "dynamic" {
			if in.Frequency <= 0 || in.Delta <= 0 {
				return out, errors.New("Для динамики необходимы f₁>0 и δ>0")
			}
			kd := out.K // equivalent height for the local zone; h-based dynamic coefficient tracked explicitly
			if in.Geometry == "building" {
				if in.KMode == "table" {
					kd, e = db.interpHeight("11.2", in.Terrain, in.Height)
				} else {
					alpha, _ := db.tableValue("11.3", in.Terrain, "alpha")
					k10, _ := db.tableValue("11.3", in.Terrain, "k10")
					kd, e = db.calcExpr("11.4", map[string]float64{"z_e": in.Height, "k10": k10, "alpha_terrain": alpha})
				}
			}
			if e != nil {
				return out, e
			}
			out.Tg1, e = db.calcExpr("11.8", map[string]float64{"w0": out.W0Pa, "k_dynamic": kd, "gamma_f": in.Gamma, "f1": in.Frequency})
			if e != nil {
				return out, e
			}
			track("11.8")
			out.Xi, e = db.calcExpr("11.1.xi_davenport_approx", map[string]float64{"Tg1": out.Tg1, "delta_damp": in.Delta})
			if e != nil {
				return out, e
			}
			track("11.1.xi_davenport_approx")
			out.PulsePa, e = db.calcExpr("11.7", map[string]float64{"w_m": out.MeanPa, "xi": out.Xi, "zeta": out.Zeta, "v_corr": out.Nu})
			track("11.7")
			limits := db.Formulas["11.1.xi_davenport_approx"].Approximation
			if len(limits.ValidityTg1) == 2 && len(limits.ValidityDelta) > 0 {
				ds := limits.ValidityDelta
				if out.Tg1 < limits.ValidityTg1[0] || out.Tg1 > limits.ValidityTg1[1] || in.Delta < ds[0] || in.Delta > ds[len(ds)-1] {
					warn(fmt.Sprintf("Аппроксимация ξ вне проверенной области Tg₁ [%.3f; %.3f], δ [%.3f; %.3f]", limits.ValidityTg1[0], limits.ValidityTg1[1], ds[0], ds[len(ds)-1]))
				}
			}
			lim, eLim := db.tableValue("11.5", "T_g_lim", strconv.FormatFloat(in.Delta, 'f', -1, 64))
			if eLim == nil {
				out.FLim, e = db.calcExpr("11.9a", map[string]float64{"w0": out.W0Pa, "k_dynamic": kd, "gamma_f": in.Gamma, "Tg_lim": lim})
				if e != nil {
					return out, e
				}
				track("11.9a")
				if in.Frequency >= out.FLim {
					warn("Для f₁≥f_lim следует проверить выбор безрезонансной ветви 11.5")
				}
			} else {
				warn("δ не совпадает с узлом табл. 11.5: f_lim не определялось; интерполяция Tg_lim не утверждена")
			}
			warn("ξ: согласованная инженерная аппроксимация рис. 11.1, не нормативная формула")
		} else {
			return out, errors.New("Неизвестный режим пульсаций")
		}
		if e != nil {
			return out, e
		}
		if in.PulseMode == "nonresonant" {
			if in.Frequency > 0 && in.Delta > 0 {
				lim, eLim := db.tableValue("11.5", "T_g_lim", strconv.FormatFloat(in.Delta, 'f', -1, 64))
				if eLim == nil {
					kd := out.K
					if in.Geometry == "building" {
						if in.KMode == "table" {
							kd, e = db.interpHeight("11.2", in.Terrain, in.Height)
						} else {
							alpha, errA := db.tableValue("11.3", in.Terrain, "alpha")
							k10, errK := db.tableValue("11.3", in.Terrain, "k10")
							if errA != nil {
								return out, errA
							}
							if errK != nil {
								return out, errK
							}
							kd, e = db.calcExpr("11.4", map[string]float64{"z_e": in.Height, "k10": k10, "alpha_terrain": alpha})
						}
						if e != nil {
							return out, e
						}
					}
					out.FLim, e = db.calcExpr("11.9a", map[string]float64{"w0": out.W0Pa, "k_dynamic": kd, "gamma_f": in.Gamma, "Tg_lim": lim})
					if e == nil {
						track("11.9a")
						if in.Frequency <= out.FLim {
							warn("f₁≤f_lim: безрезонансная ветвь может быть неприменима")
						}
					}
				}
			} else {
				warn("Частота f₁ не задана: условие применения безрезонансной ветви f₁>f_lim не проверено")
			}
		}
	} else {
		warn("Рассчитана только средняя составляющая wₘ. Основная ветровая нагрузка требует проверки пульсаций по 11.1.8")
	}
	out.NormPa, e = db.calcExpr("11.1", map[string]float64{"w_m": out.MeanPa, "w_g": out.PulsePa})
	if e != nil {
		return out, e
	}
	track("11.1")
	out.DesignPa, e = db.calcExpr("11.4.gamma", map[string]float64{"load_type": 0}) // selection formula returns normative gamma as number
	if e != nil {
		return out, e
	}
	track("11.4.gamma")
	if in.Gamma != out.DesignPa {
		warn("Коэффициент γf переопределён пользователем относительно 11.4")
	}
	out.DesignPa = out.NormPa * in.Gamma
	return out, nil
}
func CalculateWind(in WindInput) (WindCalculation, error) {
	db, e := getWindDB()
	if e != nil {
		return WindCalculation{}, e
	}
	r, e := db.calculate(in)
	if e != nil {
		return WindCalculation{}, e
	}
	return WindCalculation{Input: in, Result: r}, nil
}
func windDefaultInput(region string) WindInput {
	if region == "" {
		region = "VII" // maximum region; no location-based assumption
	}
	if region == "IA" || region == "Iа" {
		region = "Ia"
	}
	in := WindInput{Region: region, DisplayUnit: "kpa", Terrain: "B", Geometry: "building", Height: 20, Depth: 30, Z: 20, AeroTable: "В.2", AeroZone: "D", KMode: "table", PulseMode: "mean", Rho: 10, Chi: 20, ProfileStep: 2.5}
	in.StructuralType = "rc"
	if db, e := getWindDB(); e == nil {
		if gamma, err := db.calcExpr("11.4.gamma", map[string]float64{}); err == nil {
			in.Gamma = gamma
		}
		if t, ok := db.Tables["11.5"]; ok && len(t.Rows) > 0 {
			// The listed RC/masonry group has the largest damping decrement.
			if delta, err := numberCell(t.Rows[len(t.Rows)-1][0]); err == nil {
				in.Delta = delta
			}
		}
	}
	return in
}

// WindLoadProfile evaluates every ordinate through the original, unmodified
// mathematical core. It never interpolates pressure values between calls.
// Chart sample heights and geometric transitions are deterministic.
func CalculateWindProfile(in WindInput) (WindCalculation, error) {
	in.KMode = "table"
	if in.PulseMode != "mean" {
		in.PulseMode = "nonresonant"
	}
	in.Z = in.Height
	primary, err := CalculateWind(in)
	if err != nil {
		return WindCalculation{}, err
	}
	zs, err := WindProfileHeights(in.Height, in.ProfileStep)
	if err != nil {
		return WindCalculation{}, err
	}
	// Preserve geometric regime transitions explicitly. The jump at H-d is
	// drawn as a discontinuity, not blended into one long diagonal.
	if in.Geometry == "building" && in.Height > 2*in.Depth {
		for _, edge := range []float64{in.Depth, in.Height - in.Depth} {
			if edge > 0 && edge < in.Height {
				zs = append(zs, edge)
				// A point immediately before the upper jump makes the limit
				// visible in a height-pressure diagram.
				if edge == in.Height-in.Depth {
					zs = append(zs, math.Nextafter(edge, 0))
				}
			}
		}
	} else if in.Geometry == "building" && in.Height > in.Depth {
		edge := in.Height - in.Depth
		if edge > 0 && edge < in.Height {
			zs = append(zs, edge, math.Nextafter(edge, 0))
		}
	}
	// Sorting is deterministic and keeps exact mandatory sampling points.
	sort.Float64s(zs)
	last := math.Inf(-1)
	for _, z := range zs {
		if z == last {
			continue
		}
		last = z
		snapshot := in
		snapshot.Z = z
		result, e := CalculateWind(snapshot)
		if e != nil {
			return WindCalculation{}, fmt.Errorf("Эпюра на отметке %.5f м: %w", z, e)
		}
		primary.Profile = append(primary.Profile, WindProfilePoint{Z: z, Result: result.Result})
	}
	return primary, nil
}

// Site terrain is intentionally independent from wind-region lookup.
func windTerrainKey(label string) string {
	switch {
	case label == "A" || strings.HasPrefix(label, "А —") || strings.HasPrefix(label, "A —"):
		return "A"
	case label == "B" || strings.HasPrefix(label, "В —") || strings.HasPrefix(label, "B —"):
		return "B"
	case label == "C" || strings.HasPrefix(label, "С —") || strings.HasPrefix(label, "C —"):
		return "C"
	}
	return ""
}
func windSiteTerrainLabel(key string) string {
	switch key {
	case "A":
		return "А — открытая"
	case "B":
		return "В — городская"
	case "C":
		return "С — плотная застройка"
	}
	return "Не выбран"
}
