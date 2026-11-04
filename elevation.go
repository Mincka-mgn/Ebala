package main

import (
	"fmt"
	"strings"
)

// Таблица Е.1 СП 20.13330.2016, изменения № 1–6.
// Подбор по региону лишь предлагает варианты. Принадлежность конкретной площадки
// горному району устанавливает пользователь по карте 1, а не название субъекта РФ.
type MountainArea struct {
	ID, Region, Caption, Zone string
	H0, Kh                    float64
}

var mountainAreas = []MountainArea{
	{"dagestan", "Республика Дагестан", "Горные районы Республики Дагестан", "II", 500, 1},
	{"krasnodar_adler", "Краснодарский край", "Адлерский район (горная территория)", "VII", 500, 2.15},
	{"krasnodar_other", "Краснодарский край", "Остальные ГОРНЫЕ районы Краснодарского края", "II", 500, 5.0},
	{"adygea", "Республика Адыгея", "Горные районы Республики Адыгея", "VII", 500, 2.15},
	{"stavropol", "Ставропольский край", "Горные районы Ставропольского края", "II", 500, 1},
	{"evenkia", "Красноярский край", "Эвенкийский автономный округ (горная территория)", "VI", 500, .35},
	{"kuzbass", "Кемеровская область", "Кузнецкий Алатау и Горная Шория (Кузбасс)", "VI,VII", 500, 2.25},
	{"kuzbass_kras", "Красноярский край", "Кузнецкий Алатау и Горная Шория (прилегающая территория)", "VI,VII", 500, 2.25},
	{"sayans", "Красноярский край", "Саянский и Куртушибинский хребты", "IV", 500, 3.15},
	{"north_yenisei", "Красноярский край", "Северо-Енисейский район", "VI", 500, .95},
	{"norilsk_ne", "Красноярский край", "Норильский промышленный район: к СВ от Норильска", "V", 500, 3.75},
	{"norilsk_s", "Красноярский край", "Норильский промышленный район: до 20 км к Ю от Норильска", "VI", 500, 2.5},
	{"bur_khamar", "Республика Бурятия", "Хребет Хамар-Дабан", "IV", 500, 1},
	{"bur_baikal", "Республика Бурятия", "Байкальский хребет", "IV", 500, 2.3},
	{"zab_udokan", "Забайкальский край", "Каларский район, хребет Удокан", "II", 1000, 1.8},
	{"yakutia", "Республика Саха (Якутия)", "Алданское нагорье", "III", 500, 1.35},
}

func mountainByID(id string) (MountainArea, bool) {
	for _, v := range mountainAreas {
		if v.ID == id {
			return v, true
		}
	}
	return MountainArea{}, false
}
func mountainForRegion(region string) []MountainArea {
	var out []MountainArea
	for _, v := range mountainAreas {
		if sameRegion(v.Region, region) {
			out = append(out, v)
		}
	}
	return out
}
func sameRegion(a, b string) bool {
	a, b = normalizeSearch(a), normalizeSearch(b)
	if a == b {
		return true
	}
	// Фактические варианты административных названий, встречающиеся в справочниках.
	if (strings.Contains(a, "адыге") && strings.Contains(b, "адыге")) ||
		(strings.Contains(a, "саха") && strings.Contains(b, "якут")) ||
		(strings.Contains(a, "кемеров") && strings.Contains(b, "кузбасс")) ||
		(strings.Contains(a, "эвенк") && strings.Contains(b, "красноярск")) {
		return true
	}
	return false
}

// ElevationText reproduces the user's entry. For legacy projects, derive it from
// the numeric field so old files retain their original appearance.
func (s Site) ElevationText() string {
	if s.ElevationRaw != "" {
		return s.ElevationRaw
	}
	if s.ElevationKnown || s.ElevationM != 0 {
		return decimalInput(s.ElevationM, 2)
	}
	return ""
}
func (s Site) ValidElevationForCalc() bool {
	if s.ElevationRaw == "" && (s.ElevationKnown || s.ElevationM != 0) {
		return s.ElevationM >= 0 && s.ElevationM <= 9000
	}
	return s.ElevationKnown && s.ElevationM >= 0 && s.ElevationM <= 9000
}

// Numeric value may be sanitized without changing ElevationRaw.
// A missing or malformed value means 0 m for calculations and no height uplift.
func (s *Site) SetElevationText(raw string) {
	s.ElevationRaw = raw
	s.ElevationM = 0
	s.ElevationKnown = false
	if strings.TrimSpace(raw) == "" {
		return
	}
	v, err := parseNumber(raw)
	if err == nil && v >= 0 && v <= 9000 {
		s.ElevationM = v
		s.ElevationKnown = true
	}
}
func (s Site) ElevationWarning() string {
	raw := s.ElevationText()
	if strings.TrimSpace(raw) == "" {
		if s.MountainID != "" {
			return "Абсолютная отметка не указана. В расчёте принимаем 0 м, высотную поправку отключаем."
		}
		return ""
	}
	if !s.ValidElevationForCalc() {
		return "Недопустимая отметка «" + raw + "». В расчёте принимаем 0 м, высотную поправку отключаем. Исходный текст сохраняется для отчёта."
	}
	return ""
}
func (s Site) ElevationDetails() (base, result float64, mountain MountainArea, applied bool, err error) {
	base = s.SnowGround
	if s.MountainID == "" {
		return base, base, mountain, false, nil
	}
	mountain, applied = mountainByID(s.MountainID)
	if !applied {
		return base, base, mountain, false, fmt.Errorf("Неизвестный горный район таблицы Е.1")
	}
	// Invalid or absent elevations are treated as 0 m for calculation,
	// without applying the height correction. Raw input remains intact for reports.
	if !s.ValidElevationForCalc() {
		return base, base, mountain, false, nil
	}
	// Для Е.1 исходный Sg берётся по снеговому району, указанному в самой таблице.
	if strings.Contains(mountain.Zone, ",") {
		if s.SnowZone != "VI" && s.SnowZone != "VII" {
			return base, base, mountain, false, fmt.Errorf("Для Кузнецкого Алатау выберите снеговой район VI или VII по карте")
		}
		base, _ = snowForZone(s.SnowZone)
	} else {
		base, _ = snowForZone(mountain.Zone)
	}
	result = base
	if s.ElevationM > mountain.H0 {
		result = base * (1 + .001*mountain.Kh*(s.ElevationM-mountain.H0))
	}
	return base, result, mountain, true, nil
}

func (s Site) EffectiveSnow() float64 {
	_, v, _, applied, err := s.ElevationDetails()
	if applied && err == nil {
		return v
	}
	return s.SnowGround
}

func (s Site) ElevationSummary() string {
	base, v, m, on, err := s.ElevationDetails()
	if !on || err != nil {
		return ""
	}
	if s.ElevationM <= m.H0 {
		return fmt.Sprintf("Горная территория: %s. h=%s м ≤ h₀=%s м; Sg=%s кПа без повышения (прим. 1 к карте 1).", m.Caption, decimalInput(s.ElevationM, 1), decimalInput(m.H0, 0), decimalInput(base, 3))
	}
	return fmt.Sprintf("Горная территория: %s; h=%s м; h₀=%s м; kₕ=%s; Sg=%s×[1+0,001×%s×(%s−%s)]=%s кПа (табл. Е.1).", m.Caption, decimalInput(s.ElevationM, 1), decimalInput(m.H0, 0), decimalInput(m.Kh, 2), decimalInput(base, 3), decimalInput(m.Kh, 2), decimalInput(s.ElevationM, 1), decimalInput(m.H0, 0), decimalInput(v, 3))
}
