package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

var project = initialProject()

type Layer struct {
	Name       string  `json:"name"`
	Thickness  float64 `json:"thickness_m"`
	UnitWeight float64 `json:"unit_weight_kn_m3"`
	GammaF     float64 `json:"gamma_f"`
}

// WeightRow holds a single editable item of the surface-load takeoff.
// Kind = "layer" (thickness_mm * density_kg_m3 / 1000) or "direct" (kgf/m2).
type WeightRow struct {
	Name       string  `json:"name"`
	Kind       string  `json:"kind"`
	Thickness  float64 `json:"thickness"`
	Density    float64 `json:"density"`
	GNorm      float64 `json:"g_norm"`
	Gamma      float64 `json:"gamma"`
	NormEdited bool    `json:"norm_edited,omitempty"`
}

// SwitchWeightKind switches between thickness × density and an entered
// load per m² without changing the currently computed normative load.
func SwitchWeightKind(r WeightRow, kind string) WeightRow {
	if r.Kind == kind || (kind != "direct" && kind != "layer") {
		return r
	}
	norm := r.NormKgf()
	if kind == "direct" {
		r.Kind = "direct"
		r.GNorm = norm
	} else {
		r.Kind = "layer"
		if r.Thickness <= 0 {
			r.Thickness = 100
		}
		r.Density = norm * 1000 / r.Thickness
	}
	return r
}

// EditWeightField updates one quantity while preserving the requested direction
// of conversion. Direct-weight rows do not require a fictitious thickness.
func EditWeightField(r WeightRow, field string, value float64) (WeightRow, error) {
	if !num(value) || value < 0 {
		return r, fmt.Errorf("Недопустимое значение")
	}
	switch field {
	case "thickness":
		if r.Kind == "layer" {
			r.Thickness = value
			r.NormEdited = true
		}
	case "density":
		if r.Kind == "layer" {
			r.Density = value
			r.NormEdited = true
		}
	case "norm":
		if r.Kind == "layer" && r.Thickness > 0 {
			r.Density = value * 1000 / r.Thickness
		} else {
			r.Kind = "direct"
			r.GNorm = value
		}
		r.NormEdited = true
	case "design":
		if r.NormEdited {
			if r.NormKgf() == 0 {
				if value != 0 {
					return r, fmt.Errorf("Сначала задайте ненулевую нормативную нагрузку")
				}
			}
			if r.NormKgf() > 0 {
				r.Gamma = value / r.NormKgf()
			}
		} else {
			if r.Gamma <= 0 {
				return r, fmt.Errorf("Недопустимый коэффициент")
			}
			norm := value / r.Gamma
			if r.Kind == "layer" && r.Thickness > 0 {
				r.Density = norm * 1000 / r.Thickness
			} else {
				r.Kind = "direct"
				r.GNorm = norm
			}
		}
	case "gamma":
		if value <= 0 {
			return r, fmt.Errorf("Коэффициент должен быть больше нуля")
		}
		r.Gamma = value
	default:
		return r, fmt.Errorf("Неизвестное поле")
	}
	if r.Gamma <= 0 {
		return r, fmt.Errorf("Коэффициент должен быть больше нуля")
	}
	return r, nil
}

type WeightCalc struct {
	Name string      `json:"name"`
	Rows []WeightRow `json:"rows"`
}

func (r WeightRow) NormKgf() float64 {
	if r.Kind == "layer" {
		return r.Thickness * r.Density / 1000
	}
	return r.GNorm
}
func (w WeightCalc) Validate() error {
	for i, r := range w.Rows {
		if strings.TrimSpace(r.Name) == "" || (r.Kind != "layer" && r.Kind != "direct") ||
			!num(r.Thickness) || !num(r.Density) || !num(r.GNorm) || !num(r.Gamma) ||
			r.Thickness < 0 || r.Density < 0 || r.GNorm < 0 || r.Gamma <= 0 || !num(r.NormKgf()) || !num(r.NormKgf()*r.Gamma) {
			return fmt.Errorf("Проверьте строку собственного веса № %d", i+1)
		}
	}
	return nil
}
func (w WeightCalc) Totals() (normKgf, designKgf, gammaEq float64) {
	for _, r := range w.Rows {
		n := r.NormKgf()
		normKgf += n
		designKgf += n * r.Gamma
	}
	if normKgf > 0 {
		gammaEq = designKgf / normKgf
	}
	return
}
func (p Project) CurrentWeight() WeightCalc {
	if p.Weight != nil {
		return *p.Weight
	}
	w := WeightCalc{Name: "Покрытие", Rows: make([]WeightRow, 0, len(p.Layers))}
	for _, l := range p.Layers {
		w.Rows = append(w.Rows, WeightRow{Name: l.Name, Kind: "layer", Thickness: l.Thickness * 1000, Density: l.UnitWeight * kgfPerKN, Gamma: l.GammaF})
	}
	return w
}

type Wind struct {
	W0     float64 `json:"w0_kpa"`
	KZ     float64 `json:"k_z"`
	C      float64 `json:"c"`
	GammaF float64 `json:"gamma_f"`
}
type Site struct {
	Region                 string  `json:"region,omitempty"`
	District               string  `json:"district,omitempty"`
	Place                  string  `json:"place,omitempty"`
	SnowZone               string  `json:"snow_zone,omitempty"`
	WindZone               string  `json:"wind_zone,omitempty"`
	IceZone                string  `json:"ice_zone,omitempty"`
	SnowGround             float64 `json:"snow_ground_kpa,omitempty"`
	SnowNote               string  `json:"snow_reference,omitempty"`
	MountainID             string  `json:"mountain_area_id,omitempty"`
	ElevationM             float64 `json:"site_elevation_m,omitempty"`
	ElevationRaw           string  `json:"elevation_user_text,omitempty"` // exactly what the user typed for reports
	ElevationKnown         bool    `json:"elevation_known,omitempty"`
	ElevationSource        string  `json:"elevation_source,omitempty"`
	Reference              string  `json:"zone_reference,omitempty"`
	SeismicMap             string  `json:"seismic_map,omitempty"`
	SeismicRefPlace        string  `json:"seismic_reference_place,omitempty"`
	SeismicRefRegion       string  `json:"seismic_reference_region,omitempty"`
	SeismicPoints          string  `json:"seismic_intensity_points,omitempty"`
	ClimateJanuary         float64 `json:"climate_january_c,omitempty"`
	ClimateWind            float64 `json:"climate_cold_wind_ms,omitempty"`
	ClimateJanuarySet      bool    `json:"climate_january_set,omitempty"`
	ClimateWindSet         bool    `json:"climate_wind_set,omitempty"`
	ClimateReferencePlace  string  `json:"climate_reference_place,omitempty"`
	ClimateReferenceRegion string  `json:"climate_reference_region,omitempty"`
	ClimateOrigin          string  `json:"climate_origin,omitempty"`
}
type ClimateOverride struct {
	Mode    string  `json:"mode,omitempty"`
	Enabled bool    `json:"enabled"`
	Zone    string  `json:"zone,omitempty"`
	Value   float64 `json:"value,omitempty"`
}

// SavedReportBlock is an independent, editable calculation in the project report.
// Snapshots allow multiple roofs or ice elements without overwriting older input.
type SavedReportBlock struct {
	ID         int                   `json:"id"`
	Kind       string                `json:"kind"` // weight, snow, ice, temporary
	Title      string                `json:"title"`
	Weight     *ProjectWeightLine    `json:"weight,omitempty"`
	WeightCalc *WeightCalc           `json:"weight_calculation,omitempty"`
	Snow       *SnowCalc             `json:"snow,omitempty"`
	Ice        *IceLoadCalc          `json:"ice,omitempty"`
	Temporary  *TempSelection        `json:"temporary,omitempty"`
	CraneSet   []TempSelection       `json:"crane_set,omitempty"`
	Placement  *CranePlacementResult `json:"crane_placement,omitempty"`
	ParentID   int                   `json:"parent_id,omitempty"`
	SpansText  string                `json:"spans_text,omitempty"`
}
type Project struct {
	Version              int                `json:"format_version"`
	Layers               []Layer            `json:"layers"`
	Weight               *WeightCalc        `json:"weight,omitempty"`
	Wind                 Wind               `json:"wind"`
	Site                 Site               `json:"site,omitempty"`
	WindOverride         bool               `json:"wind_override,omitempty"`
	WindZoneOverride     string             `json:"wind_zone_override,omitempty"`
	Snow                 ClimateOverride    `json:"snow_override,omitempty"`
	ReportManaged        bool               `json:"report_managed,omitempty"`
	SavedBlocks          []SavedReportBlock `json:"saved_blocks,omitempty"`
	SuppressedSelections []string           `json:"suppressed_selections,omitempty"`
	SnowCalc             *SnowCalc          `json:"snow_calculation,omitempty"`
	IceLoad              *IceLoadCalc       `json:"ice_load_calculation,omitempty"`
	Ice                  ClimateOverride    `json:"ice_override,omitempty"`
}

func (p Project) SnowInputs() (string, float64, bool) {
	if p.Snow.Enabled {
		return p.Snow.Zone, p.Snow.Value, true
	}
	return p.Site.SnowZone, p.Site.EffectiveSnow(), false
}
func (p Project) IceInputs() (string, float64, bool) {
	if p.Ice.Enabled {
		return p.Ice.Zone, p.Ice.Value, true
	}
	v, _ := iceForZone(p.Site.IceZone)
	return p.Site.IceZone, v, false
}

// Site is the source of defaults for the NEXT calculation. Saved blocks are
// immutable snapshots until their own calculation is explicitly recomputed.
// Do not rebase historical snow/ice calculations (or invalidate an unfinished
// one) on a change of region: a project can legitimately contain several sites.
func (p *Project) SetSite(s Site) {
	previous := p.Site
	changed := previous.Region != s.Region || previous.District != s.District || previous.Place != s.Place ||
		previous.SnowZone != s.SnowZone || previous.SnowGround != s.SnowGround || previous.WindZone != s.WindZone || previous.IceZone != s.IceZone ||
		previous.MountainID != s.MountainID || previous.ElevationM != s.ElevationM || previous.ClimateJanuary != s.ClimateJanuary || previous.ClimateWind != s.ClimateWind
	p.Site = s
	if changed {
		p.Snow = ClimateOverride{}
		p.Ice = ClimateOverride{}
		p.WindOverride = false
		p.WindZoneOverride = ""
	}
	if !p.WindOverride {
		if v, ok := windForZone(s.WindZone); ok {
			p.Wind.W0 = v
		} else {
			// Unknown climate zone is an unassigned current wind input;
			// completed wind calculations are stored separately.
			p.Wind.W0 = 0
		}
	}
}
func snowRebaseSite(v SnowCalc, s Site) SnowCalc {
	v.Region, v.Ground, v.GroundLocal = s.SnowZone, s.EffectiveSnow(), false
	if v.ClimateOrigin != "manual" {
		v.InheritClimate(s)
	}
	if v.Ground > 0 {
		if result, err := SnowCompute(v); err == nil {
			return result
		}
	}
	v.Cases = nil // Never present values computed for the previous locality.
	return v
}
func iceRebaseSite(v IceLoadCalc, s Site) IceLoadCalc {
	v.Region = s.IceZone
	v.BaseMM, _ = iceForZone(s.IceZone)
	v.BaseFromStudy = false
	v.StudyReference = ""
	v.Wind.Zone = s.WindZone
	if v.BaseMM > 0 {
		if result, err := IceCompute(v); err == nil {
			return result
		}
	}
	v.Points = nil
	return v
}

type Result struct {
	SWN, SWD, WN, WD float64
}

// Conversion based on standard gravity g0=9.80665 m/s2.
// 1 kPa = 1000/9.80665 kgf/m2. The SAME factor applies to kN/m3 -> kgf/m3.
const kgfPerKN = 1000.0 / 9.80665

func decimalInput(v float64, maxDecimals int) string {
	s := strconv.FormatFloat(v, 'f', maxDecimals, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	if s == "" || s == "-" {
		s = "0"
	}
	return strings.ReplaceAll(s, ".", ",")
}
func kpaToKgfM2(v float64) float64 { return v * kgfPerKN }
func kgfM2ToKpa(v float64) float64 { return v / kgfPerKN }

func initialProject() Project {
	return Project{Version: 1, Layers: []Layer{{"Стяжка (на строительной площадке)", .05, 20, 1.3}, {"Утеплитель", .10, 1.5, 1.2}, {"Гидроизоляция", .005, 12, 1.2}}, Wind: Wind{.30, 1, .8, 1.4}, WindOverride: false}
}
func num(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func (p Project) Calc() (Result, error) {
	var r Result
	if p.Weight != nil {
		if err := p.Weight.Validate(); err != nil {
			return r, err
		}
		gn, gr, _ := p.Weight.Totals()
		r.SWN = kgfM2ToKpa(gn)
		r.SWD = kgfM2ToKpa(gr)
	} else {
		if len(p.Layers) == 0 {
			return Result{}, errors.New("Нужен хотя бы один слой")
		}
		for i, l := range p.Layers {
			if strings.TrimSpace(l.Name) == "" || !num(l.Thickness) || !num(l.UnitWeight) || !num(l.GammaF) || l.Thickness < 0 || l.UnitWeight < 0 || l.GammaF < 0 {
				return r, fmt.Errorf("Недопустимые данные слоя № %d", i+1)
			}
			r.SWN += l.Thickness * l.UnitWeight
			r.SWD += l.Thickness * l.UnitWeight * l.GammaF
		}
	}
	w := p.Wind
	if !num(w.W0) || !num(w.KZ) || !num(w.C) || !num(w.GammaF) || w.W0 < 0 || w.KZ < 0 || w.GammaF < 0 {
		return r, errors.New("Проверьте параметры ветра")
	}
	r.WN = w.W0 * w.KZ * w.C
	r.WD = r.WN * w.GammaF
	return r, nil
}
func (p Project) Save(path string) error {
	// A project is a collection of saved calculations and incomplete drafts.
	// Project serialization must never validate the unrelated working modules.
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0644)
}
func LoadProject(path string) (Project, error) {
	var p Project
	b, err := os.ReadFile(path)
	if err != nil {
		return p, err
	}
	if err = json.Unmarshal(b, &p); err != nil {
		return p, err
	}
	if p.Version != 1 {
		return p, errors.New("Неподдерживаемая версия проекта")
	}
	// Files created in v0.1/0.2 have no site. Preserve their manually entered wind.
	if p.Site.Region == "" && p.Site.Place == "" {
		p.WindOverride = true
	}
	return p, nil
}
func parseNumber(s string) (float64, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || !num(v) {
		return 0, fmt.Errorf("Требуется число, получено: %q", s)
	}
	return v, nil
}
func displayNumber(v float64) string {
	return strings.ReplaceAll(strconv.FormatFloat(v, 'f', 4, 64), ".", ",")
}
