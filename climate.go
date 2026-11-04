package main

import (
	_ "embed"
	"fmt"
	"strings"
	"unicode"
)

//go:embed climate_places.tsv
var placesText string

//go:embed region_capitals.tsv
var regionCapitals string

type ClimateRecord struct {
	Place, Region, District         string
	SnowZone, WindZone, IceZone     string
	SnowK                           float64
	IsDistrict, IsCapital, IsRegion bool
}

// Pilot city wind/snow zones are imported from a secondary published directory,
// and MUST be verified against maps E.1-E.3 at the exact site before final use.
const climateZoneSource = "Предварительный справочник EVRAZ; сверить по картам приложения Е СП 20.13330.2016"
const climateSnowSource = "СП 20.13330.2016, п. 10.2, приложение К (таблица К.1)"

var snowKTable = map[string]float64{
	"Альметьевск": 1.85, "Ангарск": 1.05, "Арзамас": 1.6, "Архангельск": 1.8, "Астрахань": .4,
	"Ачинск": 1.25, "Барнаул": 1.55, "Белгород": 1.55, "Бийск": 2.15, "Благовещенск": .5,
	"Братск": 1.25, "Брянск": 1.6, "Великие Луки": 1.1, "Великий Новгород": 1.55, "Владикавказ": .65,
	"Владимир": 1.85, "Волгоград": 1.0, "Волгодонск": .85, "Волжский": 1.0, "Вологда": 1.65,
	"Воронеж": 1.55, "Грозный": .45, "Димитровград": 2.05, "Екатеринбург": 1.35, "Елец": 1.35,
	"Златоуст": 1.85, "Иваново": 1.7, "Ижевск": 2.15, "Иркутск": 1.05, "Йошкар-Ола": 1.8,
	"Казань": 2.3, "Калининград": .8, "Калуга": 1.9, "Каменск-Уральский": 1.25, "Камышин": 1.15,
	"Кемерово": 1.8, "Киров": 2.1, "Киселёвск": 1.65, "Ковров": 1.6, "Коломна": 1.45,
	"Комсомольск-на-Амуре": 1.25, "Копейск": 1.2, "Кострома": 1.8, "Краснодар": 1.1, "Красноярск": 1.35,
	"Курган": 1.3, "Курск": 1.25, "Кызыл": .5, "Липецк": 1.5, "Магадан": 1.35,
	"Магнитогорск": 1.3, "Майкоп": .9, "Махачкала": .6, "Миасс": 1.1, "Москва": 1.45,
	"Мурманск": 3.2, "Муром": 1.55, "Набережные Челны": 2.25, "Невинномысск": .75, "Нефтекамск": 2.05,
	"Нефтеюганск": 1.8, "Нижневартовск": 2.3, "Нижнекамск": 2.1, "Нижний Новгород": 2.1, "Нижний Тагил": 1.5,
	"Новокузнецк": 1.8, "Новокуйбышевск": 1.6, "Новомосковск": 1.45, "Новосибирск": 1.6, "Новочебоксарск": 1.95,
	"Новочеркасск": .85, "Новошахтинск": .8, "Новый Уренгой": 2.55, "Норильск": 2.4, "Омск": 1.35,
	"Оренбург": 1.25, "Орск": 1.2, "Орёл": 1.4, "Пенза": 1.45, "Первоуральск": 1.4,
	"Пермь": 1.95, "Петрозаводск": 1.7, "Петропавловск-Камчатский": 4.1, "Прокопьевск": 1.6, "Псков": 1.3,
	"Ростов-на-Дону": .85, "Рубцовск": 1.0, "Рыбинск": 2.0, "Рязань": 1.55, "Салават": 2.45,
	"Самара": 1.6, "Санкт-Петербург": 1.3, "Саранск": 1.6, "Саратов": 1.4, "Северодвинск": 2.25,
	"Серпухов": 1.5, "Смоленск": 1.6, "Ставрополь": .95, "Старый Оскол": 1.55, "Стерлитамак": 2.2,
	"Сургут": 1.8, "Сызрань": 1.55, "Сыктывкар": 2.45, "Таганрог": .85, "Тамбов": 1.4,
	"Тверь": 1.6, "Тобольск": 1.55, "Тольятти": 1.65, "Томск": 2.15, "Тула": 1.5,
	"Тюмень": 1.6, "Улан-Удэ": .45, "Ульяновск": 1.4, "Уссурийск": .7, "Уфа": 2.45,
	"Ухта": 2.15, "Хабаровск": 1.1, "Хасавюрт": .65, "Чебоксары": 1.95, "Челябинск": 1.2,
	"Череповец": 1.85, "Черкесск": .6, "Чита": .4, "Шахты": .8, "Элиста": .7,
	"Энгельс": 1.4, "Южно-Сахалинск": 3.85, "Якутск": .7, "Ярославль": 1.8,
}

var snowByZone = map[string]float64{"I": .5, "II": 1, "III": 1.5, "IV": 2, "V": 2.5, "VI": 3, "VII": 3.5, "VIII": 4}
var windByZone = map[string]float64{"IA": .17, "I": .23, "II": .30, "III": .38, "IV": .48, "V": .60, "VI": .73, "VII": .85}
var iceByZone = map[string]float64{"I": 3, "II": 5, "III": 10, "IV": 15, "V": 20}

func romanZone(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "А", "A")
	if s == "1A" || s == "IA" {
		return "IA"
	}
	if s == "" {
		return ""
	}
	// Leave only valid roman notation: I, II, ..., VIII.
	if _, ok := snowByZone[s]; ok {
		return s
	}
	return ""
}
func snowForZone(s string) (float64, bool) { v, ok := snowByZone[romanZone(s)]; return v, ok }
func windForZone(s string) (float64, bool) { v, ok := windByZone[romanZone(s)]; return v, ok }
func iceForZone(s string) (float64, bool)  { v, ok := iceByZone[romanZone(s)]; return v, ok }

func climateRecords() []ClimateRecord {
	// The two EVRAZ tables are a search accelerator, never authoritative input.
	// Supplement with the previous list only for places not in the editable catalog.
	legacy := make([]ClimateRecord, 0)
	kRegions := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(placesText), "\n") {
		cols := strings.Split(strings.TrimSpace(line), "|")
		if len(cols) != 4 {
			continue
		}
		r := ClimateRecord{Place: cols[0], Region: cols[1], SnowZone: cols[2], WindZone: cols[3]}
		kRegions[normalizeSearch(r.Place)] = normalizeSearch(r.Region)
		// Legacy list is retained for coverage when not present in the new files.
		if normalizeSearch(r.Place) == "волжский" && r.Region == "Самарская область" {
			r.SnowK = 0
		} else if k, ok := snowKTable[r.Place]; ok && k > 0 {
			r.SnowK = k
		}
		legacy = append(legacy, r)
	}
	cities := loadExternalClimateCatalog()
	if len(cities) == 0 {
		cities = legacy
	} else {
		present := map[string]bool{}
		for i := range cities {
			r := &cities[i]
			key := normalizeSearch(r.Region) + "|" + normalizeSearch(r.Place)
			present[key] = true
			// Apply K only for a city in its original region, not by a name match alone.
			if k, ok := snowKTable[r.Place]; ok && k > 0 && kRegions[normalizeSearch(r.Place)] == normalizeSearch(r.Region) {
				r.SnowK = k
			}
		}
		for _, old := range legacy {
			key := normalizeSearch(old.Region) + "|" + normalizeSearch(old.Place)
			if !present[key] {
				cities = append(cities, old)
				present[key] = true
			}
		}
	}
	// Region entries can be selected without auto-filling a misleading capital.
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(regionCapitals), "\n") {
		cols := strings.SplitN(strings.TrimSpace(line), "|", 2)
		if len(cols) != 2 {
			continue
		}
		reg := strings.TrimSpace(cols[0])
		key := normalizeSearch(reg)
		if seen[key] {
			continue
		}
		seen[key] = true
		cities = append(cities, ClimateRecord{Region: reg, IsRegion: true})
	}
	return cities
}
func normalizeSearch(s string) string {
	s = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), "ё", "е"))
	var b strings.Builder
	for _, c := range s {
		if unicode.IsLetter(c) || unicode.IsNumber(c) {
			b.WriteRune(c)
		} else {
			b.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
func matchingRecords(records []ClimateRecord, q string) []ClimateRecord {
	q = normalizeSearch(q)
	var out []ClimateRecord
	// Put independent region entries first in search results.
	for _, isRegion := range []bool{true, false} {
		for _, r := range records {
			if r.IsRegion != isRegion {
				continue
			}
			s := normalizeSearch(r.Place + " " + r.Region + " " + r.District)
			if (validClimateRecord(r) || r.IsRegion) && (q == "" || strings.Contains(s, q)) {
				out = append(out, r)
			}
		}
	}
	return out
}
func recordTitle(r ClimateRecord) string {
	if r.IsRegion {
		return "Регион: " + r.Region
	}
	if r.IsDistrict {
		return fmt.Sprintf("Район: %s  ·  %s  [без авто-районов]", r.District, r.Region)
	}
	mark := ""
	if r.SnowK > 0 {
		mark = "★ К.1  "
	}
	return fmt.Sprintf("%s%s  ·  %s", mark, r.Place, r.Region)
}
func climateFromRecord(r ClimateRecord) Site {
	sg, ok := snowForZone(r.SnowZone)
	if !ok {
		sg = 0
	}
	source := "По табл. 10.1; районирование уточнить по карте прил. Е"
	if r.SnowK > 0 {
		sg = r.SnowK
		source = climateSnowSource
	}
	return Site{Region: r.Region, District: r.District, Place: r.Place, SnowZone: r.SnowZone, WindZone: r.WindZone, IceZone: r.IceZone, SnowGround: sg, SnowNote: source, Reference: climateZoneSource}
}
