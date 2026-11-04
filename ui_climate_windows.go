//go:build windows

package main

import (
	"fmt"
	"strings"
	"unsafe"
)

var allPlaces = climateRecords()
var foundPlaces []ClimateRecord
var selectedPlace = -1
var draftSite Site

func searchPlaces() {
	foundPlaces = matchingRecords(allPlaces, get(601))
	send(602, LB_RESETCONTENT, 0, 0)
	for _, r := range foundPlaces {
		pSend.Call(controls[602], LB_ADDSTRING, 0, uintptr(unsafe.Pointer(ptr(recordTitle(r)))))
	}
	set(626, fmt.Sprintf("Найдено записей: %d. ★ — значение Sg по приложению К", len(foundPlaces)))
	selectedPlace = -1
}
func selectedPlaceRow() {
	idx := int(int32(send(602, LB_GETCURSEL, 0, 0)))
	if idx < 0 || idx >= len(foundPlaces) {
		return
	}
	selectedPlace = idx
	draftSite = climateFromRecord(foundPlaces[idx])
	if r, ok := seismicAutoMatch(draftSite.Region, draftSite.Place); ok {
		draftSite.SeismicMap = "А"
		draftSite.SeismicRefPlace = r.Place
		draftSite.SeismicRefRegion = r.Region
		draftSite.SeismicPoints = seismicDisplayPoints(r.A)
	}
	project.SetSite(draftSite)
	fillSiteInputs(draftSite)
	previewSite(draftSite)
	projectHomeOverview()
}
func known(s string) string {
	if strings.TrimSpace(s) == "" {
		return "не установлен"
	}
	return s
}
func knownFloat(f float64, unit string) string {
	if f <= 0 {
		return "не задано"
	}
	return decimalInput(f, 3) + " " + unit
}
func windPreview(zone string) string {
	v, ok := windForZone(zone)
	if !ok {
		return "не задано"
	}
	return decimalInput(v, 3) + " кПа"
}
func icePreview(zone string) string {
	v, ok := iceForZone(zone)
	if !ok {
		return "не задано"
	}
	return decimalInput(v, 1) + " мм"
}
func seismicSiteReferenceLine(s Site) string {
	if strings.TrimSpace(s.SeismicRefPlace) == "" {
		return ""
	}
	note := "Сейсмика по нас. пункту " + s.SeismicRefPlace
	if s.SeismicRefRegion != "" && !canonicalRegionMatch(s.SeismicRefRegion, s.Region) {
		note += " (" + s.SeismicRefRegion + ")"
	}
	return note
}
func siteClimateSummary(s Site) string {
	base := "Снег: " + snowChoice(s) + ", ветер: " + known(s.WindZone) + ", гололёд: " + known(s.IceZone)
	if !s.ClimateJanuarySet && !s.ClimateWindSet {
		return base
	}
	ref := "вручную"
	if s.ClimateOrigin == "sp131" && s.ClimateReferencePlace != "" {
		ref = s.ClimateReferencePlace + " (СП 131.13330.2025)"
	}
	return base + " · T января / V холодн.: " + ref
}
func previewSite(s Site) {
	snowLabel := "район " + known(s.SnowZone)
	if specialSnowSite(s) {
		snowLabel = snowOptionK
	}
	set(620, "СНЕГ: "+snowLabel+" · Sg = "+knownFloat(s.EffectiveSnow(), "кПа"))
	set(621, "ВЕТЕР: район "+known(s.WindZone)+" · w₀ = "+windPreview(s.WindZone))
	if specialSnowSite(s) {
		set(624, "★ Особый, по приложению К; нормативное значение Sg")
	} else if s.SnowZone == snowOptionMountain {
		set(624, "Горные и малоизученные районы: Sg устанавливается отдельно")
	} else {
		set(624, "Таблица 10.1; подтвердить границы по карте")
	}
	set(622, "ГОЛОЛЁД: "+known(s.IceZone)+" · b = "+icePreview(s.IceZone))
	seis := "не указана"
	if s.SeismicMap != "" && s.SeismicPoints != "" {
		seis = "карта " + s.SeismicMap + ", " + seismicReportIntensity(s.SeismicPoints)
		if seismicReviews[normalizeGeo(s.SeismicRefRegion)+"|"+normalizeGeo(s.SeismicRefPlace)] != "" {
			seis += " · ⚠ сверить с СП 14"
		}
	}
	set(623, "СЕЙСМИКА: "+seis)
	set(660, seismicSiteReferenceLine(s))
}
func fillSiteInputs(s Site) {
	changingUnits = true
	for id, v := range map[int]string{631: s.Region, 632: s.District, 633: s.Place} {
		set(id, v)
	}
	setZone(634, snowChoice(s))
	setZone(636, s.WindZone)
	setZone(637, s.IceZone)
	if s.SnowGround > 0 {
		set(635, decimalInput(s.SnowGround, 3))
	} else {
		set(635, "")
	}
	if s.ClimateJanuarySet {
		set(662, decimalInput(s.ClimateJanuary, 1))
	} else {
		set(662, "")
	}
	if s.ClimateWindSet {
		set(663, decimalInput(s.ClimateWind, 1))
	} else {
		set(663, "")
	}
	set(638, "w₀ = "+windPreview(s.WindZone))
	set(639, "b = "+icePreview(s.IceZone))
	setZone(654, s.SeismicMap)
	setZone(655, seismicDisplayPoints(s.SeismicPoints))
	changingUnits = false
	fillMountainControls(s)
}
func readSiteManual() (Site, error) {
	choice := get(634)
	snowZone := choice
	if isKChoice(choice) {
		snowZone = draftSite.SnowZone
	}
	windZone := windInternalZone(get(636))
	iceZone := get(637)
	s := Site{Region: strings.TrimSpace(get(631)), District: strings.TrimSpace(get(632)), Place: strings.TrimSpace(get(633)), SnowZone: snowZone, WindZone: windZone, IceZone: iceZone, Reference: "Климатические зоны назначены пользователем"}
	if isKChoice(choice) {
		if !specialSnowSite(draftSite) || s.Region != draftSite.Region || s.Place != draftSite.Place {
			return s, fmt.Errorf("Режим «Особый, по приложению К» разрешён только для выбранного города приложения К; если изменили населённый пункт, выберите снеговой район")
		}
		s.SnowNote = climateSnowSource
		s.SnowGround = draftSite.SnowGround
	} else if choice == snowOptionMountain {
		s.SnowNote = "СП 20.13330.2016, карта 1 приложения Е: горные и малоизученные районы"
	} else if choice != "" {
		v, ok := snowForZone(choice)
		if !ok {
			return s, fmt.Errorf("Выберите снеговой район из списка")
		}
		s.SnowGround = v
		s.SnowNote = climateZoneNotes("snow", choice)
	}
	if v, err := parseNumber(get(635)); err == nil && v >= 0 {
		// Explicit numeric edits are allowed for the assessment of an existing structure.
		if v != s.SnowGround {
			s.SnowGround = v
			s.SnowNote = "Sg изменено вручную пользователем"
		}
	}
	if s.WindZone != "" {
		if _, ok := windForZone(s.WindZone); !ok {
			return s, fmt.Errorf("Ветровой район не поддерживается")
		}
	}
	if s.IceZone != "" {
		if _, ok := iceForZone(s.IceZone); !ok {
			return s, fmt.Errorf("Гололёдный район не поддерживается")
		}
	}
	// Preserve the selected climate reference unless values were edited manually.
	s.ClimateJanuary = draftSite.ClimateJanuary
	s.ClimateWind = draftSite.ClimateWind
	s.ClimateJanuarySet = draftSite.ClimateJanuarySet
	s.ClimateWindSet = draftSite.ClimateWindSet
	s.ClimateOrigin = draftSite.ClimateOrigin
	s.ClimateReferencePlace = draftSite.ClimateReferencePlace
	s.ClimateReferenceRegion = draftSite.ClimateReferenceRegion
	jraw := strings.TrimSpace(get(662))
	vraw := strings.TrimSpace(get(663))
	if jraw != "" || vraw != "" {
		var j, v float64
		if jraw != "" {
			val, e := parseNumber(jraw)
			if e != nil {
				return s, fmt.Errorf("Температура января: %w", e)
			}
			j = val
		}
		if vraw != "" {
			val, e := parseNumber(vraw)
			if e != nil || val < 0 {
				return s, fmt.Errorf("Проверьте скорость ветра холодного периода")
			}
			v = val
		}
		if (jraw != "") != s.ClimateJanuarySet || (vraw != "") != s.ClimateWindSet || (jraw != "" && j != s.ClimateJanuary) || (vraw != "" && v != s.ClimateWind) {
			s.SetSnowClimateManual(j, v, jraw != "", vraw != "")
		}
	} else if s.ClimateJanuarySet || s.ClimateWindSet {
		s.SetSnowClimateManual(0, 0, false, false)
	}
	readMountainFields(&s)
	s.SeismicMap = get(654)
	s.SeismicRefPlace = draftSite.SeismicRefPlace
	s.SeismicRefRegion = draftSite.SeismicRefRegion
	s.SeismicPoints = get(655)
	if _, _, _, applied, _ := s.ElevationDetails(); applied {
		s.SnowNote = "Снег уточнён по таблице Е.1 и примечанию к карте 1 приложения Е"
	}
	return s, nil
}
func currentSiteForReport() (Site, error) {
	// Report is based on the current selection AND unsaved edits, not project.Site.
	return readSiteManual()
}

// Preserve the currently entered site parameters without a separate Apply button.
// Incomplete numeric edits remain on screen and are reconsidered on leaving the page.
func siteSaveSelectionSilently() {
	if activePage != "site" || changingUnits {
		return
	}
	if s, e := readSiteManual(); e == nil {
		project.SetSite(s)
		draftSite = s
		projectHomeOverview()
	}
}

func applySiteManual() {
	s, err := readSiteManual()
	if err != nil {
		message(err.Error(), true)
		return
	}
	project.SetSite(s)
	draftSite = s
	refreshClimateUI()
	note := "Характеристики местности сохранены. При районировании по картам СП проверьте выбранную территорию."
	if w := s.ElevationWarning(); w != "" {
		note += "\n\nПредупреждение: " + w
	}
	message(note, false)
}
func refreshClimateUI() {
	s := project.Site
	draftSite = s
	fillSiteInputs(s)
	previewSite(s)
	displayName := formatSitePlace(s.Region, s.District, s.Place)
	if displayName == "" {
		displayName = "местность не выбрана"
	}
	set(625, "ТЕКУЩАЯ МЕСТНОСТЬ: "+displayName)
	set(627, siteClimateSummary(s))
	set(645, mountainStatus(s))
	if !project.WindOverride {
		if v, ok := windForZone(s.WindZone); ok {
			project.Wind.W0 = v
		} else {
			project.Wind.W0 = 0
		}
	}
	changingUnits = true
	set(201, xnum(project.Wind.W0))
	set(205, decimalInput(kpaToKgfM2(project.Wind.W0), 3))
	mode := ""
	if project.WindOverride {
		mode = project.WindZoneOverride
		if mode == "" {
			mode = "Пользовательский"
		}
	} else {
		mode = s.WindZone
	}
	setZone(206, mode)
	changingUnits = false
	if project.WindOverride {
		set(210, "ЛОКАЛЬНАЯ ПРАВКА: w₀ = "+knownFloat(project.Wind.W0, "кПа"))
	} else if s.WindZone != "" {
		set(210, "ИЗ МЕСТНОСТИ: ветровой район "+s.WindZone+" · "+climateZoneNotes("wind", s.WindZone))
	} else {
		set(210, "ВНИМАНИЕ: ветровой район не выбран; для расчёта назначьте его по карте")
	}
	snowZone, sg, snowOv := project.SnowInputs()
	snowMode := snowZone
	if snowOv && isKChoice(project.Snow.Mode) {
		snowMode = snowOptionK
	} else if !snowOv && specialSnowSite(s) {
		snowMode = snowOptionK
	}
	changingUnits = true
	setZone(701, snowMode)
	if sg > 0 {
		set(702, decimalInput(sg, 3))
	} else {
		set(702, "")
	}
	changingUnits = false
	status := "ИЗ МЕСТНОСТИ"
	if snowOv {
		status = "ЛОКАЛЬНАЯ ПРАВКА"
	}
	zoneDescription := "Снеговой район " + known(snowZone)
	if snowMode == snowOptionK {
		zoneDescription = snowOptionK
	}
	set(705, "Источник: "+status+". "+zoneDescription+", Sg = "+knownFloat(sg, "кПа"))
	iceZone, b, iceOv := project.IceInputs()
	changingUnits = true
	setZone(801, iceZone)
	if iceZone != "" {
		set(802, decimalInput(b, 2))
	} else {
		set(802, "")
	}
	changingUnits = false
	iceStatus := "ИЗ МЕСТНОСТИ"
	if iceOv {
		iceStatus = "ЛОКАЛЬНАЯ ПРАВКА"
	}
	set(805, "Источник: "+iceStatus+". Район "+known(iceZone)+", b = "+knownFloat(b, "мм"))
	set(628, "Районы СП: снег "+known(s.SnowZone)+" / ветер "+known(s.WindZone)+" / гололёд "+known(s.IceZone))
	renderResults()
}
func useSiteWind() {
	v, ok := windForZone(project.Site.WindZone)
	if !ok {
		message("Сначала назначьте ветровой район в местности.", true)
		return
	}
	project.WindOverride = false
	project.WindZoneOverride = ""
	project.Wind.W0 = v
	refreshClimateUI()
}
func applySnowManual() {
	choice := get(701)
	zone := choice
	if isKChoice(choice) {
		if !specialSnowSite(project.Site) {
			message("Режим «Особый, по приложению К» доступен только для городов из приложения К.", true)
			return
		}
		zone = project.Site.SnowZone
	}
	if zone != snowOptionMountain {
		if _, ok := snowForZone(zone); !ok {
			message("Выберите снеговой район I–VIII или горную категорию", true)
			return
		}
	}
	value, err := parseNumber(get(702))
	if err != nil || value < 0 {
		message("Введите допустимый Sg, кПа", true)
		return
	}
	project.Snow = ClimateOverride{Enabled: true, Zone: zone, Value: value, Mode: choice}
	refreshClimateUI()
}
func useSiteSnow() { project.Snow = ClimateOverride{}; refreshClimateUI() }
func applyIceManual() {
	zone := get(801)
	if _, ok := iceForZone(zone); !ok {
		message("Выберите гололёдный район I–V", true)
		return
	}
	value, err := parseNumber(get(802))
	if err != nil || value <= 0 {
		message("Введите толщину стенки гололёда, мм", true)
		return
	}
	project.Ice = ClimateOverride{Enabled: true, Zone: zone, Value: value}
	refreshClimateUI()
}
func useSiteIce() { project.Ice = ClimateOverride{}; refreshClimateUI() }
func isClimateCombo(id int) bool {
	switch id {
	case 634, 636, 637, 643, 654, 655, 701, 801, 206:
		return true
	}
	return false
}
func climateComboChanged(id int) {
	if changingUnits {
		return
	}
	choice := get(id)
	switch id {
	case 654:
		// Map changes intensity only if a definite reference settlement is known.
		if r, ok := seismicAutoMatch(draftSite.SeismicRefRegion, draftSite.SeismicRefPlace); ok {
			changingUnits = true
			setZone(655, seismicDisplayPoints(r.Value(choice)))
			changingUnits = false
		} else if r, ok := seismicAutoMatch(get(631), get(633)); ok {
			draftSite.SeismicRefPlace, draftSite.SeismicRefRegion = r.Place, r.Region
			changingUnits = true
			setZone(655, seismicDisplayPoints(r.Value(choice)))
			changingUnits = false
		}
		draftSite.SeismicMap, draftSite.SeismicPoints = choice, get(655)
		showSeismicSiteSummary()
	case 655:
		draftSite.SeismicMap, draftSite.SeismicPoints = get(654), choice
		showSeismicSiteSummary()
	case 643:
		// Choosing a mountain category must NOT silently replace the ordinary
		// snow zone. If height is missing/invalid, its original Sg is preserved.
		changed, _ := readSiteManual()
		set(645, mountainStatus(changed))
	case 634:
		if isKChoice(choice) {
			if v, ok := snowValueForChoice(choice, draftSite); ok {
				set(635, decimalInput(v, 3))
			} else {
				set(635, "")
			}
		} else if choice == snowOptionMountain {
			set(635, "")
		} else if v, ok := snowForZone(choice); ok {
			set(635, decimalInput(v, 3))
		}
		set(622, "Снеговое районирование: "+choice)
	case 636:
		set(638, "w₀ = "+windPreview(windInternalZone(choice)))
	case 637:
		set(639, "b = "+icePreview(choice))
	case 701:
		if v, ok := snowValueForChoice(choice, project.Site); ok {
			set(702, decimalInput(v, 3))
		} else {
			set(702, "")
		}
	case 801:
		if v, ok := iceForZone(choice); ok {
			set(802, decimalInput(v, 1))
		}
	case 206:
		if choice == "Пользовательский" {
			project.WindZoneOverride = choice
			project.WindOverride = true
			return
		}
		zone := windInternalZone(choice)
		if v, ok := windForZone(zone); ok {
			project.WindOverride = true
			project.WindZoneOverride = zone
			project.Wind.W0 = v
			changingUnits = true
			set(201, decimalInput(v, 3))
			set(205, decimalInput(kpaToKgfM2(v), 3))
			changingUnits = false
			set(210, "ЛОКАЛЬНЫЙ ВЕТРОВОЙ РАЙОН "+choice+" · w₀="+decimalInput(v, 3)+" кПа")
			renderResults()
		}
	}
}
func showSeismicSiteSummary() {
	s := draftSite
	s.SeismicMap, s.SeismicPoints = get(654), get(655)
	if s.SeismicMap == "" || s.SeismicPoints == "" {
		set(623, "СЕЙСМИКА: не выбрана")
	} else {
		set(623, "СЕЙСМИКА: карта "+s.SeismicMap+", "+seismicReportIntensity(s.SeismicPoints))
	}
	set(660, seismicSiteReferenceLine(s))
}
func buildSite() {
	buildingPage = "site"
	navigation("Характеристики местности", "Выберите местность, укажите нагрузки и сейсмику; при необходимости уточните их по справочникам")
	label("ПОИСК: город, область, район", 22, 98, 285, 25)
	label("ВНИМАНИЕ! База неточна! Проверять по картам обязательно!", 314, 95, 824, 32)
	edit(601, "", 22, 125, 710, 32)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 744, 127, 388, 46, 626)
	create("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|LBS_NOTIFY|LBS_OWNERDRAWFIXED|LBS_HASSTRINGS, 22, 167, 710, 236, 602)
	label("ХАРАКТЕРИСТИКИ ВЫБРАННОЙ МЕСТНОСТИ", 746, 173, 386, 26)
	for i := 0; i < 4; i++ {
		create("STATIC", "", WS_CHILD|WS_VISIBLE, 746, 201+i*34, 381, 30, 620+i)
	}
	button(651, "Карта снега", 745, 354, 119, 36)
	button(652, "Карта ветра", 866, 354, 119, 36)
	button(653, "Карта гололёда", 987, 354, 139, 36)
	label("ПАРАМЕТРЫ ПРОЕКТА", 22, 418, 1090, 24)
	label("Область / субъект РФ", 22, 446, 342, 21)
	edit(631, "", 22, 469, 347, 29)
	label("Район области", 386, 446, 350, 21)
	edit(632, "", 386, 469, 344, 29)
	label("Населённый пункт", 747, 446, 350, 21)
	edit(633, "", 747, 469, 256, 29)
	button(647, "Сбросить", 1010, 469, 100, 29)
	label("Снеговой район", 22, 508, 279, 21)
	combo(634, snowOptions, 22, 532, 270)
	label("Sg, кПа", 303, 508, 108, 21)
	edit(635, "", 303, 532, 95, 28)
	label("Ветровой район", 410, 508, 140, 21)
	combo(636, windOptions[:len(windOptions)-1], 410, 532, 120)
	label("Гололёд", 546, 508, 126, 21)
	combo(637, iceOptions, 546, 532, 123)
	label("Сейсмика: карта", 686, 508, 142, 21)
	combo(654, []string{"А", "В", "С"}, 686, 532, 103)
	label("Балльность", 806, 508, 132, 21)
	combo(655, seismicIntensityOptions, 806, 532, 126)
	button(1000, "Выбрать по СП 14", 948, 528, 161, 38)
	button(661, "Температура и ветер по СП", 22, 572, 180, 34)
	label("T января, °C", 211, 578, 96, 23)
	edit(662, "", 312, 573, 80, 29)
	label("V зимой, м/с", 407, 578, 110, 23)
	edit(663, "", 520, 573, 81, 29)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 615, 570, 108, 28, 638)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 724, 570, 82, 28, 639)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 811, 570, 298, 36, 660)
	label("СНЕГ В ГОРНЫХ РАЙОНАХ (таблица Е.1 СП 20.13330.2016)", 22, 613, 666, 22)
	combo(643, []string{"Не применять (обычная местность)"}, 22, 640, 666)
	label("Абсолютная отметка, м", 709, 613, 401, 22)
	edit(644, "", 709, 640, 195, 29)
	button(646, "Проверить в Википедии", 913, 640, 196, 30)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 22, 677, 1100, 43, 645)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 22, 726, 1085, 22, 625)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 22, 751, 1085, 20, 627)
	footerSite()
}
func buildSnowInputs() { buildSnowUI() }
func buildIceInputs() {
	buildingPage = "ice"
	navigation("Гололёд и ветер", "СП 20.13330.2016 · разделы 11–12 · горизонтальная и вертикальная нагрузки")
	label("Гололёдный район", 25, 111, 200, 24)
	combo(801, iceOptions, 25, 138, 178)
	label("b, мм", 216, 111, 130, 24)
	edit(802, "", 216, 138, 145, 30)
	button(iceSiteButton, "Из местности", 374, 138, 180, 31)

	button(iceExtraButton, "Специальные условия расчёта", 774, 136, 303, 35)
	label("Тип обледеневаемого элемента", 25, 186, 510, 23)
	combo(iceKind, []string{"Круглый элемент (прочий)", "Горизонтальный провод/трос (гибкий)", "Остальной элемент: малое сечение", "Остальной элемент: μ₂ (ручной коэффициент)"}, 25, 211, 490)
	create("STATIC", "d, мм (круглый элемент)", WS_CHILD|WS_VISIBLE, 534, 186, 260, 24, iceDiameterLabel)
	edit(iceDiameter, "10", 534, 211, 164, 30)
	create("STATIC", "Коэффициент μ₂", WS_CHILD|WS_VISIBLE, 716, 186, 300, 24, iceMuTwoLabel)
	edit(iceMuTwo, "0,6", 716, 211, 164, 30)
	label("Отметка от, м", 25, 259, 210, 25)
	edit(iceHeightFrom, "5", 25, 286, 165, 31)
	label("До, м", 207, 259, 200, 25)
	edit(iceHeightTo, "100", 207, 286, 165, 31)
	label("Шаг, м", 389, 259, 180, 25)
	edit(iceHeightStep, "5", 389, 286, 143, 31)

	heading("ВЕТРОВАЯ НАГРУЗКА", 25, 340, 780, 27)
	create("STATIC", "Ветровой район", WS_CHILD|WS_VISIBLE, 25, 373, 210, 22, iceWindZoneLabel)
	combo(iceWindZone, append([]string{"— без ветра"}, windOptions[:len(windOptions)-1]...), 25, 398, 178)
	create("STATIC", "Местность", WS_CHILD|WS_VISIBLE, 218, 373, 196, 22, iceWindTerrainLabel)
	combo(iceWindTerrain, []string{"А — открытые территории", "В — города, леса, препятствия >10 м", "С — плотная застройка >25 м"}, 218, 398, 335)
	create("STATIC", "Cx — сопротивление потоку воздуха", WS_CHILD|WS_VISIBLE, 568, 373, 450, 24, iceWindDragLabel)
	edit(iceWindDrag, "1,2", 568, 398, 110, 31)
	button(iceDragReset, "Сброс Cx", 684, 398, 118, 31)

	button(iceCalculateButton, "Рассчитать", 25, 811, 165, 38)
	button(iceAddButton, "Сохранить в ведомость и новый расчёт", 198, 811, 399, 38)
	button(iceRemoveButton, "Сбросить нагрузки", 602, 811, 176, 38)
	button(iceWordButton, "Отчёт Word", 783, 811, 144, 38)
	button(iceExcelButton, "Расчёт Excel", 931, 811, 170, 38)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 25, 466, 1050, 24, iceCalcStatus)
	create("STATIC", "", WS_CHILD|WS_VISIBLE|0x000E, 25, 503, 1040, 245, iceChartImage)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 25, 753, 1060, 49, iceChartStatus)
}
func buildIceAdvanced() {
	buildingPage = "ice_extra"
	navigation("Гололёд: специальные условия расчёта", "")
	heading("Особые высотные районы", 25, 107, 740, 31)
	label("Тип территории по таблице 12.2 и картам 3г–3е", 25, 148, 900, 25)
	combo(iceExtraCategory, []string{"Обычная", "I район азиатской части", "Специальная"}, 25, 182, 470)
	label("Толщина гололёда на отметках 200/300/400 м, мм (если есть отдельные метеоизыскания)", 25, 235, 930, 24)
	edit(iceExtraHigh200, "", 25, 265, 152, 31)
	edit(iceExtraHigh300, "", 195, 265, 152, 31)
	edit(iceExtraHigh400, "", 365, 265, 152, 31)
	label("Плотность льда ρ, кг/м³ (900 по СП; иное по специальным данным)", 545, 235, 520, 24)
	edit(iceExtraDensity, "900", 545, 265, 150, 31)
	button(iceExtraDensityReset, "Сброс 900", 705, 265, 155, 31)
	label("Источник специальных данных (чтобы включить обоснование в отчёт)", 25, 332, 1000, 26)
	edit(iceExtraStudy, "", 25, 362, 1000, 32)
	heading("Учет ветра: дополнительные исходные данные", 25, 447, 790, 30)
	create("BUTTON", "Гибкий элемент (оттяжка, ванта, трос); ψ = 0,25 при гололёде", WS_CHILD|WS_VISIBLE|0x00000003, 25, 408, 980, 32, iceExtraFlexible)
	label("Обоснование принятого Cx (откуда взяли значение)", 25, 540, 1000, 25)
	edit(iceExtraCSource, "", 25, 567, 970, 33)
	label("Cx обледеневшей конструкции (если отличается от 1,2)", 25, 480, 700, 28)
	edit(iceExtraDragIced, "", 25, 507, 187, 33)
	create("BUTTON", "Пульсационная составляющая определена расчётом", WS_CHILD|WS_VISIBLE|0x00000003, 25, 615, 950, 31, iceExtraPulseCheck)
	label("Отношение пульсационной и средней составляющих wp/wm", 25, 660, 940, 24)
	edit(iceExtraPulse, "", 25, 685, 160, 30)
	label("Источник расчёта пульсаций по п. 11.1.8 СП 20.13330.2016", 208, 660, 790, 22)
	edit(iceExtraPulseSource, "", 207, 685, 819, 30)

	button(iceExtraBack, "Сохранить и вернуться к гололёду", 25, 805, 500, 43)
}
