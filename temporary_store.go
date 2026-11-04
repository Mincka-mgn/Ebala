package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

type TempWheel struct {
	XMM    float64 `json:"x_mm"`
	PKN    float64 `json:"p_kN"`
	XKnown bool    `json:"x_known,omitempty"`
}
type TempEntry struct {
	ID     string                 `json:"id"`
	Group  string                 `json:"group"`
	Title  string                 `json:"title"`
	Source string                 `json:"source"`
	Params map[string]interface{} `json:"params"`
	Notes  string                 `json:"notes"`
	Status string                 `json:"status"`
	Code   string                 `json:"code"`
	Wheels []TempWheel            `json:"wheels,omitempty"`
}
type TempCatalog struct {
	Version int         `json:"version"`
	Sources []string    `json:"sources"`
	Records []TempEntry `json:"records"`
}
type TempSelection struct {
	Group  string                 `json:"group"`
	Title  string                 `json:"title"`
	Source string                 `json:"source"`
	Params map[string]interface{} `json:"params,omitempty"`
	Notes  string                 `json:"notes,omitempty"`
	Status string                 `json:"status,omitempty"`
	Wheels []TempWheel            `json:"wheels,omitempty"`
}

// WeightReportInput preserves editable layer geometry for the independent Excel ledger.
// SavedReportBlock already contains these input rows; no change to project files.
type WeightReportInput struct {
	Title string      `json:"title"`
	Rows  []WeightRow `json:"rows"`
}

type TempReport struct {
	Snows        []SnowCalc           `json:"snows,omitempty"`
	Ices         []IceLoadCalc        `json:"ices,omitempty"`
	Snow         *SnowCalc            `json:"snow,omitempty"`
	Ice          *IceLoadCalc         `json:"ice_load,omitempty"`
	Version      int                  `json:"version"`
	Project      string               `json:"project"`
	Sections     []TempSelection      `json:"sections"`
	SiteLines    []string             `json:"site_lines,omitempty"`
	Site         *Site                `json:"site_input,omitempty"`
	WeightRows   []ProjectWeightLine  `json:"weight_rows,omitempty"`
	WeightBlocks []ProjectWeightBlock `json:"weight_blocks,omitempty"`
	WeightInputs []WeightReportInput  `json:"weight_inputs,omitempty"`
	CraneGroups  []CraneReportGroup   `json:"crane_groups,omitempty"`
	SnowTitles   []string             `json:"snow_titles,omitempty"`
	IceTitles    []string             `json:"ice_titles,omitempty"`
}

func loadTempCatalog(path string) (TempCatalog, error) {
	data, e := os.ReadFile(path)
	if e != nil {
		return TempCatalog{}, e
	}
	var c TempCatalog
	if e = json.Unmarshal(data, &c); e != nil {
		return c, e
	}
	if c.Version != 1 || len(c.Records) < 40 {
		return c, fmt.Errorf("неверная версия или пустой справочник временных нагрузок")
	}
	seen := map[string]bool{}
	for _, v := range c.Records {
		if v.ID == "" || v.Title == "" || v.Source == "" || seen[v.ID] {
			return c, fmt.Errorf("ошибка идентификаторов каталога: %s", v.ID)
		}
		seen[v.ID] = true
	}
	return c, nil
}
func tempNumberText(v interface{}) string {
	switch n := v.(type) {
	case float64:
		return strconv.FormatFloat(n, 'f', -1, 64)
	case string:
		return n
	case int:
		return strconv.Itoa(n)
	case bool:
		if n {
			return "да"
		}
		return "нет"
	default:
		return fmt.Sprintf("%v", v)
	}
}

var tempParamNames = map[string]string{"q_kPa": "Равномерная нагрузка, кПа", "P_kN": "Сосредоточенная нагрузка, кН", "line_load_kN_per_m": "Погонная нагрузка, кН/м", "gamma_f": "Коэффициент надёжности γf", "wheel_count": "Колёс на одной нитке", "wheel_count_one_rail": "Колёс на одной нитке", "capacity_t": "Грузоподъёмность, т", "crane_capacity_t": "Грузоподъёмность, т", "crane_span_m": "Пролёт крана, м", "p_wheel_kN": "Макс. нагрузка на колесо, кН", "wheel_load_F_kN": "F, кН", "wheel_load_F1_kN": "F1, кН", "horizontal_load_Qb_kN": "Qб, кН", "wheelbase_mm": "База колёс, мм", "base_A_mm": "База A, мм", "axle_load_kN": "Нагрузка на ось, кН", "distributed_load_both_tracks_kN_per_m": "Распределённая на две колеи, кН/м", "K": "Класс нагрузки K", "axle_spacing_m": "Расстояние между осями, м", "axle_count": "Число осей", "concentrated_upper_kN": "Сосредоточенная нагрузка (верхняя оценка), кН", "distributed_kN_per_m": "Распределённая нагрузка, кН/м", "empty_rolling_stock_kN_per_m": "Порожний состав, кН/м", "axles_per_wagon": "Осей на вагон", "loaded_wagon_kN": "Нагруженный вагон, кН", "empty_wagon_kN": "Порожний вагон, кН", "front_axle_kN": "Передняя ось, кН", "rear_axle_kN": "Задняя ось, кН", "track_width_rear_m": "Колея задних колёс, м"}

func tempParamLines(params map[string]interface{}) []string {
	if len(params) == 0 {
		return nil
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		p := tempParamNames[k]
		if p == "" {
			p = k
		}
		out = append(out, p+": "+tempNumberText(params[k]))
	}
	return out
}
func tempSectionFromEntry(e TempEntry) TempSelection {
	w := append([]TempWheel(nil), e.Wheels...)
	for i := range w {
		w[i].XKnown = true
	}
	return TempSelection{Group: e.Group, Title: e.Title, Source: e.Source, Params: e.Params, Notes: e.Notes, Status: e.Status, Wheels: w}
}
func tempSectionDetails(s TempSelection) string {
	lines := []string{s.Title, "Источник: " + s.Source, "Статус: " + s.Status}
	lines = append(lines, tempParamLines(s.Params)...)
	if len(s.Wheels) > 0 {
		lines = append(lines, fmt.Sprintf("Колёс на одной нитке пути: %d", len(s.Wheels)))
		for i, w := range s.Wheels {
			if w.XKnown {
				lines = append(lines, fmt.Sprintf("Колесо %d: x=%g мм, P=%g кН", i+1, w.XMM, w.PKN))
			} else {
				lines = append(lines, fmt.Sprintf("Колесо %d: x не задана, P=%g кН", i+1, w.PKN))
			}
		}
	}
	if s.Notes != "" {
		lines = append(lines, "Примечание: "+s.Notes)
	}
	return strings.Join(lines, "\r\n")
}
func parseWheelVector(s string, count int, label string, strictlyPositive bool) ([]float64, error) {
	s = strings.NewReplacer(";", " ", "\r", " ", "\n", " ", "\t", " ").Replace(strings.TrimSpace(s))
	fields := strings.Fields(s)
	if count < 1 || count > 64 {
		return nil, fmt.Errorf("%s: число колёс должно быть от 1 до 64", label)
	}
	if len(fields) != count {
		return nil, fmt.Errorf("%s: ожидается %d значений, введено %d. Разделяйте точкой с запятой или пробелом", label, count, len(fields))
	}
	v := make([]float64, 0, count)
	for _, raw := range fields {
		n, e := strconv.ParseFloat(strings.ReplaceAll(raw, ",", "."), 64)
		if e != nil || math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || (strictlyPositive && n <= 0) {
			return nil, fmt.Errorf("%s: недопустимое число %q", label, raw)
		}
		v = append(v, n)
	}
	return v, nil
}
func manualCrane(title string, count int, forces, positions string) (TempSelection, error) {
	if strings.TrimSpace(title) == "" {
		return TempSelection{}, fmt.Errorf("Укажите обозначение крана")
	}
	f, e := parseWheelVector(forces, count, "Нагрузки P, кН", true)
	if e != nil {
		return TempSelection{}, e
	}
	pos := make([]float64, count)
	if strings.TrimSpace(positions) != "" {
		pos, e = parseWheelVector(positions, count, "Координаты колёс, мм", false)
		if e != nil {
			return TempSelection{}, e
		}
		if pos[0] != 0 {
			return TempSelection{}, fmt.Errorf("Первая координата должна быть равна 0 мм")
		}
		for i := 1; i < count; i++ {
			if pos[i] <= pos[i-1] {
				return TempSelection{}, fmt.Errorf("Координаты колёс должны строго возрастать")
			}
		}
	}
	wheels := make([]TempWheel, count)
	for i := range wheels {
		wheels[i] = TempWheel{XMM: pos[i], PKN: f[i], XKnown: strings.TrimSpace(positions) != ""}
	}
	notes := "Ручное задание нагрузок."
	if strings.TrimSpace(positions) == "" {
		notes += " Продольные координаты не заданы, геометрия схемы неполная."
	}
	return TempSelection{Group: "Краны · ручной ввод", Title: strings.TrimSpace(title), Source: "Задано пользователем", Params: map[string]interface{}{"wheel_count": count}, Notes: notes, Status: "user_defined", Wheels: wheels}, nil
}
func validateTempReport(r TempReport) error {
	if r.Version != 1 {
		return fmt.Errorf("неподдерживаемый формат файла")
	}
	if len(r.Sections) > 1000 {
		return fmt.Errorf("слишком много записей")
	}
	for _, group := range r.CraneGroups {
		if len(group.Cranes) == 0 {
			return fmt.Errorf("Набор мостовых кранов «%s» пуст", group.Title)
		}
		for _, placement := range group.Placements {
			if len(placement.Result.Cases) == 0 {
				return fmt.Errorf("Расстановка «%s» для набора «%s» требует повторного расчёта", placement.Title, group.Title)
			}
		}
	}
	for _, s := range r.Sections {
		if s.Group == "" {
			return fmt.Errorf("некорректная запись отчёта")
		}
		if len(s.Wheels) > 64 {
			return fmt.Errorf("слишком много колёс")
		}
	}
	return nil
}
func saveTempReport(path string, r TempReport) error {
	if e := validateTempReport(r); e != nil {
		return e
	}
	b, e := json.MarshalIndent(r, "", "  ")
	if e != nil {
		return e
	}
	return writeAtomic(path, append(b, '\n'))
}
func readTempReport(path string) (TempReport, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return TempReport{}, e
	}
	var r TempReport
	if e = json.Unmarshal(b, &r); e != nil {
		return r, e
	}
	return r, validateTempReport(r)
}
