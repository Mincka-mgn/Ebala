//go:build windows

package main

import (
	"bytes"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const (
	windGeometryCombo      = 17201
	windPulseCombo         = 17202
	windUnitCombo          = 17203
	windInputTab           = 17204
	windResultTab          = 17205
	windNormFigure         = 17206
	windEpureFigure        = 17207
	windExplain            = 17208
	windEpureNumbers       = 17209
	windStepInput          = 17210
	windRoofInput          = 17211
	windDLabel             = 17212
	windWidthLabel         = 17213
	windW0Label            = 17214
	windRegLabel           = 17215
	windModeLabel          = 17216
	windReportSummary      = 17217
	windVariantCombo       = 17218
	windZoneCombo          = 17219
	windResultFigure       = 17220
	windResultInfo         = 17221
	windSiteLabel          = 17222
	windGammaLabel         = 17223
	windTypeLabel          = 17224
	windHeightLabel        = 17225
	windRoofLabel          = 17226
	windStepLabel          = 17227
	windPulseLabel         = 17228
	windResultVariantLabel = 17229
	windResultZoneLabel    = 17230
	windInputFigureLabel   = 17231
	windResultGraphLabel   = 17232
	windRestoreRegion      = 17233
	windTerrainHelp        = 17234
	windGammaHelp          = 17235
	windPulseHelp          = 17236
	windHelpBack           = 17237
	windHelpNorm           = 17238
	windHelpEstimate       = 17239
	windHelpDynamic        = 17240
	windHelpBody           = 17241
	windStructureLabel     = 17242
	windStructureCombo     = 17243
	windDampingLabel       = 17244
	windDampingInput       = 17245
)

var windTerrainOptions = []string{"A", "B", "C"}
var windObjectOptions = []string{"Здание с двускатной крышей", "Отдельно стоящая стена"}
var windPulseOptions = []string{"Не учитывать пульсацию", "Пульсация для невысоких массивных зданий"}

// СП 20.13330.2016, 11.1.10 (Изм. 5): engineer chooses structural
// class; damping is displayed and editable, but is NOT in formula (11.5).
var windStructureOptions = []string{
	"ЖБ, каменные, стальной каркас со стенами",
	"Смешанные без стен, стеклянные конструкции",
	"Стальное сооружение",
}

func windStructureID(name string) string {
	switch name {
	case windStructureOptions[1]:
		return "mixed"
	case windStructureOptions[2]:
		return "steel"
	default:
		return "rc"
	}
}
func windStructureName(code string) string {
	switch code {
	case "mixed":
		return windStructureOptions[1]
	case "steel":
		return windStructureOptions[2]
	default:
		return windStructureOptions[0]
	}
}

var windCurrentBlockID int
var windDraft WindInput
var windCurrentResult *WindCalculation
var windUnit = "kpa"
var windCurrentTab int
var windInputHandles []uintptr
var windResultHandles []uintptr
var windBackButton uintptr
var windUpdating bool

func windPressureUnitName() string {
	if windUnit == "kgf" {
		return "кгс/м²"
	}
	return "кПа"
}
func windPressureValue(v float64) float64 {
	if windUnit == "kgf" {
		return v * kgfPerKN
	}
	return v
}
func windPressureKPa(v float64) float64 {
	if windUnit == "kgf" {
		return kgfM2ToKpa(v)
	}
	return v
}
func windObjectGeometry() string {
	if get(windGeometryCombo) == windObjectOptions[1] {
		return "free_wall"
	}
	return "building"
}
func windNamedLabel(id int, s string, x, y, w int) {
	create("STATIC", s, WS_CHILD|WS_VISIBLE, x, y, w, 24, id)
}
func windBuildUI() {
	buildingPage = "wind"
	navigation("Ветровая нагрузка", "СП 20.13330.2016 · ветровая нагрузка")
	windBackButton = controls[399]
	windNamedLabel(windRegLabel, "ЕДИНИЦЫ ВСЕГО РАСЧЁТА", 22, 104, 257)
	combo(windUnitCombo, []string{"кПа", "кгс/м²"}, 285, 100, 165)
	button(windInputTab, "ИСХОДНЫЕ ДАННЫЕ", 540, 100, 270, 36)
	button(windResultTab, "РЕЗУЛЬТАТЫ И ЭПЮРЫ", 825, 100, 270, 36)
	start := len(pageControls["wind"])
	windNamedLabel(windSiteLabel, "Ветровой район", 22, 156, 230)
	combo(206, windOptions, 22, 181, 225)
	windNamedLabel(windModeLabel, "Тип местности", 263, 156, 230)
	combo(224, windTerrainOptions, 263, 181, 216)
	button(207, "Взять из местности", 22, 217, 245, 29)
	button(windTerrainHelp, "Типы местности A / B / C", 273, 217, 206, 29)
	button(windRestoreRegion, "Вернуть ветровой район", 22, 181, 225, 29)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 22, 249, 457, 24, 210)
	create("STATIC", "Базовое давление w₀", WS_CHILD|WS_VISIBLE, 22, 280, 235, 23, windW0Label)
	edit(201, "", 22, 307, 225, 29)
	windNamedLabel(windGammaLabel, "Коэффициент γf", 263, 280, 240)
	edit(204, "", 263, 307, 164, 29)
	button(windGammaHelp, "?", 432, 307, 47, 29)
	windNamedLabel(windTypeLabel, "ТИП СООРУЖЕНИЯ", 22, 350, 480)
	combo(windGeometryCombo, windObjectOptions, 22, 377, 457)
	create("STATIC", "Длина здания B (вдоль конька), м", WS_CHILD|WS_VISIBLE, 22, 425, 230, 23, windDLabel)
	edit(222, "", 22, 449, 225, 29)
	create("STATIC", "Ширина здания L (поперёк конька), м", WS_CHILD|WS_VISIBLE, 263, 425, 270, 23, windWidthLabel)
	edit(228, "", 263, 449, 216, 29)
	windNamedLabel(windHeightLabel, "H — до конька, м", 22, 493, 235)
	edit(221, "", 22, 519, 225, 29)
	windNamedLabel(windRoofLabel, "h — высота крыши, м", 263, 493, 245)
	edit(windRoofInput, "", 263, 519, 216, 29)
	windNamedLabel(windStepLabel, "Шаг эпюры Δz, м (по умолчанию 2,5)", 22, 567, 250)
	edit(windStepInput, "", 22, 594, 225, 29)
	windNamedLabel(windPulseLabel, "Учёт пульсации ветра", 22, 638, 460)
	combo(windPulseCombo, windPulseOptions, 22, 665, 457)
	windNamedLabel(windStructureLabel, "Конструктивная система здания", 22, 704, 332)
	combo(windStructureCombo, windStructureOptions, 22, 730, 334)
	windNamedLabel(windDampingLabel, "Декремент δ", 364, 704, 114)
	edit(windDampingInput, "", 364, 730, 114, 29)
	button(windPulseHelp, "Про пульсации", 263, 766, 216, 31)
	button(301, "РАССЧИТАТЬ", 22, 811, 457, 41)
	windNamedLabel(windInputFigureLabel, "ПОЯСНЯЮЩИЕ СХЕМЫ", 560, 158, 510)
	create("STATIC", "", WS_CHILD|WS_VISIBLE|0x0000000D, 560, 185, 530, 595, windNormFigure)
	windInputHandles = append([]uintptr(nil), pageControls["wind"][start:]...)
	start = len(pageControls["wind"])
	windNamedLabel(windResultVariantLabel, "Вариант направления ветра", 22, 156, 485)
	combo(windVariantCombo, []string{"Сначала выполните расчёт"}, 22, 183, 510)
	windNamedLabel(windResultZoneLabel, "Зона / расчётная поверхность", 552, 156, 535)
	combo(windZoneCombo, []string{"Выберите загружение"}, 552, 183, 535)
	windNamedLabel(windResultGraphLabel, "ЭПЮРА ВЕТРОВОГО ДАВЛЕНИЯ ПО ВЫСОТЕ", 22, 224, 750)
	create("STATIC", "", WS_CHILD|WS_VISIBLE|0x0000000D, 22, 251, 695, 439, windEpureFigure)
	create("STATIC", "", WS_CHILD|WS_VISIBLE|0x0000000D, 734, 251, 353, 265, windResultFigure)
	create("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|ES_MULTILINE|ES_READONLY|WS_VSCROLL, 734, 523, 353, 100, windResultInfo)
	create("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|ES_MULTILINE|ES_READONLY|WS_VSCROLL, 734, 585, 353, 125, windEpureNumbers)
	button(236, "Сохранить в ведомость и новый расчёт", 22, 729, 326, 38)
	button(235, "Сбросить нагрузки", 356, 729, 235, 38)
	button(237, "Отчёт Word", 600, 729, 231, 38)
	button(238, "Отчёт Excel", 838, 729, 250, 38)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 22, 781, 1066, 65, windReportSummary)
	windResultHandles = append([]uintptr(nil), pageControls["wind"][start:]...)
	if windDraft.Region == "" {
		windDraft = windDefaultInput(windDBRegion(project.Site.WindZone))
		if t := windTerrainKey(project.Site.WindTerrain); t != "" {
			windDraft.Terrain = t
		}
	}
	if windDraft.DisplayUnit == "kgf" {
		windUnit = "kgf"
	}
	windSetForm()
	windSetTab(0)
}

// Справочная страница устроена аналогично выбору карты сейсмичности:
// категории слева, подробное обоснование справа. Она не меняет расчёт.
func windBuildPulseHelp() {
	buildingPage = "wind_pulse_help"
	button(windHelpBack, "← К ветровой нагрузке", 22, 20, 235, 39)
	heading("Про пульсации ветра", 275, 17, 760, 42)
	label("СП 20.13330.2016, пункты 11.1.2, 11.1.8, 11.1.10 и 11.4", 275, 67, 795, 27)
	label("Когда требуются собственные частоты сооружения?", 22, 120, 1020, 28)
	create("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|ES_MULTILINE|ES_READONLY|WS_VSCROLL, 22, 158, 1063, 614, windHelpBody)
}

// Справка намеренно дана целиком, без внутренних кнопок: её читают
// начинающие инженеры, а не пользователи расчётного ядра.
func windHelpShow(section int) {
	content := `ЧТО ТАКОЕ ПУЛЬСАЦИИ ВЕТРА

Ветер не дует с неизменной скоростью: он усиливается и ослабевает. Поэтому СП 20.13330.2016 разделяет основную ветровую нагрузку на среднюю (условно постоянную) и пульсационную составляющие. Пульсации могут усиливать колебания здания. Для правильного учёта этого эффекта обычно необходимо знать, как само здание колеблется.

СОБСТВЕННАЯ ЧАСТОТА: ЧТО ЭТО?

Если слегка отклонить гибкое сооружение и отпустить, оно будет колебаться с определённой частотой. Первая собственная частота f₁ (в герцах) — частота самого медленного основного колебания; 1 Гц означает одно колебание в секунду. Чем жёстче конструкция при той же массе, тем, как правило, выше частота. Высокое и гибкое сооружение обычно колеблется медленнее, чем низкое и жёсткое.

СП устанавливает предельную частоту f_lim, зависящую от условий расчёта (п. 11.1.10). Если первая собственная частота ВЫШЕ этого предела, пульсации можно считать по формуле (11.5) без учёта резонансного усиления. Но чтобы проверить сравнение f₁ и f_lim, частоту обычно надо предварительно определить. Само по себе небольшое число этажей не доказывает выполнение условия.

1. КОГДА СП ПРЯМО РАЗРЕШАЕТ ОБОЙТИСЬ БЕЗ РАСЧЁТА ЧАСТОТ

Примечание 1 к п. 11.1.8 СП 20.13330.2016 разрешает использовать формулу (11.5) без самостоятельного расчёта частот для:
• многоэтажных ЖЕЛЕЗОБЕТОННЫХ зданий высотой до 40 м;
• одноэтажных ЖЕЛЕЗОБЕТОННЫХ производственных зданий высотой до 36 м, если высота меньше 1,5 пролёта.
В обоих случаях местность должна относиться к типу А или В. Например, железобетонное жилое здание высотой 24 м в местности В соответствует указанному условию по высоте и типу местности; остальные исходные данные также надо проверить. Стальной павильон той же высоты автоматически в этот перечень НЕ входит.

2. КОГДА ИНЖЕНЕР МОЖЕТ РАССМОТРЕТЬ УПРОЩЕНИЕ (С ОГОВОРКОЙ)

При поверочном или предварительном расчёте иногда рассматривают гипотезу, что здание достаточно жёсткое, а его резонансная реакция мала. Примеры для ОБСУЖДЕНИЯ, не нормативные разрешения:
• невысокое компактное железобетонное здание с частыми несущими стенами;
• невысокое каменное здание со связями и жёсткими дисками перекрытий;
• малоэтажное промышленное здание с жёсткой пространственной связевой системой и небольшими пролётами.

Подумайте, есть ли у здания гибкие колонны, большой свободный пролёт, лёгкое покрытие, нерегулярный каркас, надстройки, повреждения, малые поперечные размеры. Такие особенности ослабляют предположение о высокой жёсткости. Одинаковая высота двух зданий не означает одинаковые собственные частоты.

В европейском EN 1991-1-4, п. 6.2, существуют собственные упрощения для определения структурного коэффициента cₛcᵈ: например, для зданий ниже 15 м и для некоторых каркасных зданий с несущими стенами при определённых соотношениях размеров. Это ПРИМЕР ДРУГОЙ МЕТОДИКИ, а не разрешение российского СП 20.13330.2016 не рассчитывать частоты или безусловно использовать формулу (11.5). Нельзя подставлять коэффициенты Еврокода вместо коэффициентов СП.

Если сооружение не подходит под прямое примечание СП, выбор упрощения — осознанное инженерное допущение. При ответственном проектировании и сомнениях следует определить частоты расчётной моделью, а не полагаться на впечатление о массивности.

3. КОГДА БЕЗ ДИНАМИЧЕСКОГО РАСЧЁТА НЕ ОБОЙТИСЬ

При невыполнении частотного критерия п. 11.1.8 требуется учитывать динамическую реакцию. Для этого определяют собственные частоты и формы колебаний здания, задают его массу и жёсткость и учитывают демпфирование (затухание). Если несколько частот попадают в значимый диапазон, учитывают несколько форм колебаний. Это особенно важно для высоких и гибких зданий, башен, мачт, тонких и лёгких конструкций, а также конструкций со значительными динамическими эффектами.

ЧТО ВЫБРАТЬ В EBALA

«Не учитывать пульсацию» — программа рассчитывает только среднюю составляющую. Это НЕ означает, что СП освобождает объект от учёта пульсаций. Пользователь сам отвечает за принятое допущение.
«Пульсация для невысоких массивных зданий» — программа добавляет пульсации по формуле (11.5), без резонансного расчёта. Перед выбором убедитесь, что есть нормативное основание или обоснованная инженерная гипотеза.

ДЕКРЕМЕНТ КОЛЕБАНИЙ δ И КОНСТРУКТИВНЫЙ ТИП

Декремент δ показывает, как быстро затухают колебания сооружения. СП 20.13330.2016, п. 11.1.10, устанавливает следующие группы: 0,30 — железобетонные и каменные здания, а также здания со стальным или смешанным сталежелезобетонным каркасом при сплошных наружных ограждениях; 0,15 — стальные сооружения; 0,22 — стеклянные конструкции и смешанные сооружения со стальными и железобетонными несущими конструкциями без сплошных наружных ограждений. При наличии специальных демпфирующих устройств значение требует отдельного обоснования.

В поле «Конструктивная система здания» выберите подходящую группу, после чего значение δ подставится автоматически. Инженер может исправить его вручную. ВАЖНО: в упрощённой формуле (11.5) декремент напрямую не используется, поэтому изменение δ не меняет нагрузку в этом режиме. Декремент необходим для динамических критериев и других методов, которых текущий упрощённый режим не рассчитывает.

Нормативные ссылки: СП 20.13330.2016, пп. 11.1.2, 11.1.8–11.1.11, таблицы 11.4 и 11.6. Для сравнения: EN 1991-1-4, п. 6.2. Полная динамика по собственным частотам в этом модуле пока не реализована.`
	set(windHelpBody, strings.ReplaceAll(content, "\n", "\r\n"))
}

func windSetTab(n int) {
	windCurrentTab = n
	if activePage != "wind" {
		return
	}
	for _, h := range windInputHandles {
		if h != 0 {
			pShow.Call(h, windVisible(n == 0))
		}
	}
	for _, h := range windResultHandles {
		if h != 0 {
			pShow.Call(h, windVisible(n == 1))
		}
	}
	if windBackButton != 0 {
		title := "← На главную"
		if n == 1 {
			title = "← К исходным данным"
		}
		pSetText.Call(windBackButton, uintptr(unsafe.Pointer(ptr(title))))
	}
	if n == 1 {
		windRefreshVariantCombos()
		windShowResult()
	}
	windResize()
	if n == 0 {
		windUpdateControls()
	}
}
func windVisible(on bool) uintptr {
	if on {
		return 5
	}
	return 0
}
func windResize() {
	if hwnd == 0 || activePage != "wind" {
		return
	}
	var r tempRect
	tempGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	w, h := int(r.Right), int(r.Bottom)
	if w < 790 || h < 660 {
		return
	}
	// Scalable two-column layout with a minimum 350 px illustration pane.
	left := 520
	if w < 1080 {
		left = (w - 70) / 2
	}
	paneX := left + 40
	paneW := w - paneX - 24
	colA := left/2 - 30
	colB := left/2 - 23
	right := 22 + colA + 16
	if w < 1080 {
		tempMove(windUnitCombo, 282, 100, 105, 265)
		tempMove(windInputTab, w-385, 100, 181, 36)
		tempMove(windResultTab, w-196, 100, 174, 36)
	} else {
		tempMove(windUnitCombo, 285, 100, 165, 265)
		tempMove(windInputTab, w-571, 100, 265, 36)
		tempMove(windResultTab, w-296, 100, 270, 36)
	}
	tempMove(windInputFigureLabel, paneX, 158, paneW, 24)
	tempMove(windNormFigure, paneX, 185, paneW, tempMax(320, h-215))
	// Reposition every data label as a group; no stale fixed labels over edits.
	for _, p := range []struct{ id, x, y, ww, hh int }{
		{windSiteLabel, 22, 156, colA, 24}, {206, 22, 181, colA, 28},
		{windModeLabel, right, 156, colB, 24}, {224, right, 181, colB, 28},
		{207, 22, 217, (left - 52) * 55 / 100, 29}, {windTerrainHelp, 32 + (left-52)*55/100, 217, (left-52)*45/100 - 10, 29}, {210, 22, 249, left - 42, 24},
		{windRestoreRegion, 22, 181, colA, 29},
		{windW0Label, 22, 280, colA, 24}, {201, 22, 307, colA, 29},
		{windGammaLabel, right, 280, colB, 24}, {204, right, 307, colB - 55, 29}, {windGammaHelp, right + colB - 47, 307, 47, 29},
		{windTypeLabel, 22, 350, left - 42, 24}, {windGeometryCombo, 22, 377, left - 42, 28},
		{windDLabel, 22, 425, colA, 24}, {222, 22, 449, colA, 29},
		{windWidthLabel, right, 425, colB, 24}, {228, right, 449, colB, 29},
		{windHeightLabel, 22, 493, colA, 24}, {221, 22, 519, colA, 29},
		{windRoofLabel, right, 493, colB, 24}, {windRoofInput, right, 519, colB, 29},
		{windStepLabel, 22, 567, left - 42, 24}, {windStepInput, 22, 594, colA, 29},
		{windPulseLabel, 22, 638, left - 42, 24}, {windPulseCombo, 22, 665, left - 42, 28},
		{windStructureLabel, 22, 704, colA + 30, 24},
		{windStructureCombo, 22, 730, colA + 30, 245},
		{windDampingLabel, right + 30, 704, colB - 30, 24},
		{windDampingInput, right + 30, 730, colB - 30, 29},
		{windPulseHelp, right, 766, colB, 31}, {301, 22, 811, left - 42, 41},
	} {
		hh := p.hh
		if p.id == 206 || p.id == 224 || p.id == windGeometryCombo || p.id == windPulseCombo || p.id == windStructureCombo {
			hh = 265 // COMBOBOX: MoveWindow height includes the drop-down list
		}
		tempMove(p.id, p.x, p.y, p.ww, hh)
	}
	if windCurrentTab == 1 {
		half := (w - 58) / 2
		tempMove(windResultVariantLabel, 22, 156, half, 24)
		tempMove(windVariantCombo, 22, 183, half, 265)
		tempMove(windResultZoneLabel, 34+half, 156, half, 24)
		tempMove(windZoneCombo, 34+half, 183, half, 265)
		infoW := tempMax(350, w/3+60)
		plotW := w - infoW - 58
		chartH := tempMax(250, h-443)
		tempMove(windResultGraphLabel, 22, 224, plotW, 24)
		tempMove(windEpureFigure, 22, 251, plotW, chartH)
		x := plotW + 38
		tempMove(windResultFigure, x, 251, infoW, chartH*62/100)
		tempMove(windResultInfo, x, 258+chartH*62/100, infoW, 100)
		tempMove(windEpureNumbers, x, 365+chartH*62/100, infoW, tempMax(75, chartH*38/100-114))
		btnY := h - 158
		widths := []int{(w - 60) * 30 / 100, (w - 60) * 22 / 100, (w - 60) * 21 / 100, (w - 60) * 27 / 100}
		xx := 22
		for j, id := range []int{236, 235, 237, 238} {
			tempMove(id, xx, btnY, widths[j]-6, 38)
			xx += widths[j]
		}
		tempMove(windReportSummary, 22, h-101, w-46, 69)
	}
	if c := controls[windNormFigure]; c != 0 {
		invalidateSnow.Call(c, 0, 1)
	}
	if c := controls[windResultFigure]; c != 0 {
		invalidateSnow.Call(c, 0, 1)
	}
	if c := controls[windEpureFigure]; c != 0 {
		invalidateSnow.Call(c, 0, 1)
	}
}
func windReadForm() (WindInput, error) {
	s := windDraft
	zone := get(206)
	s.ManualW0 = windDraft.ManualW0 || zone == "Пользовательский"
	if !s.ManualW0 {
		s.Region = windDBRegion(zone)
	}
	s.Terrain = get(224)
	s.Geometry = windObjectGeometry()
	s.KMode = "table"
	s.PulseMode = "mean"
	if get(windPulseCombo) == windPulseOptions[1] {
		s.PulseMode = "nonresonant"
	}
	s.Frequency = 0
	if get(windStructureCombo) != "" {
		s.StructuralType = windStructureID(get(windStructureCombo))
	}
	if s.StructuralType == "" {
		s.StructuralType = "rc"
	}
	if s.PulseMode == "nonresonant" {
		value := strings.TrimSpace(get(windDampingInput))
		if value == "" {
			s.Delta = windStructureDamping(s.StructuralType)
		} else {
			d, e := parseNumber(value)
			if e != nil || d <= 0 {
				return s, fmt.Errorf("Декремент колебаний δ должен быть положительным числом")
			}
			s.Delta = d
		}
	}
	for _, p := range []struct {
		id   int
		dst  *float64
		name string
	}{
		{201, &s.W0KPa, "w₀"}, {204, &s.Gamma, "γf"},
		{221, &s.Height, "H"}, {222, &s.Depth, "B"}, {228, &s.Rho, "L"}, {windRoofInput, &s.RoofRise, "h"},
	} {
		str := strings.TrimSpace(get(p.id))
		if str == "" {
			*p.dst = 0
			continue
		}
		f, e := parseNumber(str)
		if e != nil {
			return s, fmt.Errorf("Поле %s: %w", p.name, e)
		}
		if p.id == 201 {
			f = windPressureKPa(f)
		}
		*p.dst = f
	}
	step := strings.TrimSpace(strings.ToLower(get(windStepInput)))
	if step == "" {
		s.ProfileStep = 2.5
	} else {
		n, e := parseNumber(step)
		if e != nil {
			return s, fmt.Errorf("Шаг эпюры: %w", e)
		}
		if n <= 0 {
			return s, fmt.Errorf("Шаг эпюры должен быть положительным")
		}
		s.ProfileStep = n
	}
	s.Z = s.Height
	s.DisplayUnit = windUnit
	s.Chi = s.Height
	if s.Geometry == "free_wall" {
		s.Depth = 0
		s.RoofRise = 0
	}
	if s.Height <= 0 {
		return s, fmt.Errorf("Укажите высоту сооружения H")
	}
	if s.Rho <= 0 {
		return s, fmt.Errorf("Укажите ширину L (для стены — её длину)")
	}
	if s.Geometry == "building" && s.Depth <= 0 {
		return s, fmt.Errorf("Укажите длину здания B")
	}
	if !s.ManualW0 {
		db, e := getWindDB()
		if e != nil {
			return s, e
		}
		v, e := db.tableValue("11.1", "w0_kPa", s.Region)
		if e != nil {
			return s, e
		}
		s.W0KPa = v
	}
	return s, nil
}
func windSetForm() {
	if controls[221] == 0 {
		return
	}
	changingUnits = true
	defer func() { changingUnits = false }()
	s := windDraft
	if s.Region == "" {
		s = windDefaultInput(windDBRegion(project.Site.WindZone))
		windDraft = s
	}
	if s.ManualW0 {
		setZone(206, "Пользовательский")
	} else {
		setZone(206, windDisplayInputRegion(s.Region))
	}
	setZone(224, s.Terrain)
	if s.Geometry == "free_wall" {
		setZone(windGeometryCombo, windObjectOptions[1])
	} else {
		setZone(windGeometryCombo, windObjectOptions[0])
	}
	setZone(windPulseCombo, windPulseOptions[0])
	if s.PulseMode != "mean" {
		setZone(windPulseCombo, windPulseOptions[1])
	}
	if s.StructuralType == "" {
		s.StructuralType = "rc"
	}
	setZone(windStructureCombo, windStructureName(s.StructuralType))
	if s.Delta <= 0 {
		s.Delta = windStructureDamping(s.StructuralType)
	}
	set(windDampingInput, decimalInput(s.Delta, 3))
	if windUnit == "kgf" {
		setZone(windUnitCombo, "кгс/м²")
	} else {
		setZone(windUnitCombo, "кПа")
	}
	if !s.ManualW0 {
		if db, e := getWindDB(); e == nil {
			if v, er := db.tableValue("11.1", "w0_kPa", s.Region); er == nil {
				s.W0KPa = v
				windDraft.W0KPa = v
			}
		}
	}
	set(201, decimalInput(windPressureValue(s.W0KPa), 5))
	set(204, decimalInput(s.Gamma, 5))
	set(221, decimalInput(s.Height, 4))
	set(222, decimalInput(s.Depth, 4))
	set(228, decimalInput(s.Rho, 4))
	set(windRoofInput, decimalInput(s.RoofRise, 4))
	if s.ProfileStep <= 0 {
		set(windStepInput, "2,5")
	} else {
		set(windStepInput, decimalInput(s.ProfileStep, 4))
	}
	if len(project.WindDraftRaw) > 0 {
		for id, value := range project.WindDraftRaw {
			switch id {
			case 206, 224, windGeometryCombo, windPulseCombo, windStructureCombo:
				setZone(id, value)
			case 201:
				if windUnit == "kpa" {
					set(id, value)
				}
			default:
				set(id, value)
			}
		}
	}
	set(windW0Label, "Базовое давление w₀, "+windPressureUnitName())
	windUpdateControls()
	windRefreshVariantCombos()
	windShowResult()
}
func windUpdateControls() {
	wall := windObjectGeometry() == "free_wall"
	pulse := get(windPulseCombo) == windPulseOptions[1]
	for _, id := range []int{windStructureLabel, windStructureCombo, windDampingLabel, windDampingInput} {
		if controls[id] != 0 {
			pShow.Call(controls[id], windVisible(pulse && activePage == "wind" && windCurrentTab == 0))
		}
	}
	pEnable.Call(controls[222], boolPtr(!wall))
	pEnable.Call(controls[windRoofInput], boolPtr(!wall))
	// Давление разрешено редактировать напрямую. При изменении исходное
	// районное значение перестаёт быть источником расчёта.
	pEnable.Call(controls[201], 1)
	manual := windDraft.ManualW0 || get(206) == "Пользовательский"
	if controls[windRegLabel] != 0 {
		pShow.Call(controls[windRegLabel], windVisible(!manual && activePage == "wind" && windCurrentTab == 0))
	}
	if controls[206] != 0 {
		pShow.Call(controls[206], windVisible(!manual && activePage == "wind" && windCurrentTab == 0))
	}
	if controls[windRestoreRegion] != 0 {
		pShow.Call(controls[windRestoreRegion], windVisible(manual && activePage == "wind" && windCurrentTab == 0))
	}
	if wall {
		set(windDLabel, "Для стены размер B не нужен")
		set(windWidthLabel, "Длина стены L, м")
		set(windHeightLabel, "Высота стены H, м")

	} else {
		set(windDLabel, "Длина здания B (вдоль конька), м")
		set(windWidthLabel, "Ширина здания L (поперёк конька), м")
		set(windHeightLabel, "H — до конька, м")

	}
	source := "Источник: таблица 11.1, район " + get(206)
	if manual {
		source = "Источник: инженер (w₀ задано вручную)"
	}
	set(210, source)
	if c := controls[windNormFigure]; c != 0 {
		invalidateSnow.Call(c, 0, 1)
	}
}
func boolPtr(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}
func windComboReset(id int, values []string, index int) {
	if controls[id] == 0 {
		return
	}
	pSend.Call(controls[id], 0x014B, 0, 0)
	for _, v := range values {
		pSend.Call(controls[id], CB_ADDSTRING, 0, uintptr(unsafe.Pointer(ptr(v))))
	}
	if len(values) > 0 {
		if index < 0 || index >= len(values) {
			index = 0
		}
		pSend.Call(controls[id], CB_SETCURSEL, uintptr(index), 0)
	}
}
func windRefreshVariantCombos() {
	if controls[windVariantCombo] == 0 || windUpdating {
		return
	}
	windUpdating = true
	defer func() { windUpdating = false }()
	if windCurrentResult == nil || len(windCurrentResult.Variants) == 0 {
		windComboReset(windVariantCombo, []string{"Сначала выполните расчёт"}, 0)
		windComboReset(windZoneCombo, []string{"Нет рассчитанных зон"}, 0)
		return
	}
	c := windCurrentResult
	if c.SelectedVariant >= len(c.Variants) || c.SelectedVariant < 0 {
		c.SelectedVariant = 0
	}
	v := c.Variants[c.SelectedVariant]
	if c.SelectedZone >= len(v.Zones) || c.SelectedZone < 0 {
		c.SelectedZone = 0
	}
	names := make([]string, len(c.Variants))
	for j, v := range c.Variants {
		names[j] = v.Name
	}
	windComboReset(windVariantCombo, names, c.SelectedVariant)
	windRefreshZoneCombo()
}
func windRefreshZoneCombo() {
	if windCurrentResult == nil || len(windCurrentResult.Variants) == 0 {
		return
	}
	c := windCurrentResult
	v := c.Variants[c.SelectedVariant]
	names := make([]string, len(v.Zones))
	for j, z := range v.Zones {
		if z.Surface == "roof" {
			variant := ""
			if v.ID == "CROSS" && z.Alternative != "" {
				variant = " [знаковый вариант " + z.Alternative + "]"
			}
			names[j] = fmt.Sprintf("Крыша%s, %s — %s", variant, z.Code, z.Name)
		} else {
			names[j] = fmt.Sprintf("Стены, %s — %s", z.Code, z.Name)
		}
	}
	windComboReset(windZoneCombo, names, c.SelectedZone)
}
func windUnitAll(c *WindCalculation, unit string) {
	c.Input.DisplayUnit = unit
	for vi := range c.Variants {
		for zi := range c.Variants[vi].Zones {
			c.Variants[vi].Zones[zi].Calculation.Input.DisplayUnit = unit
		}
	}
}

// Ручной ввод w₀ — самостоятельный источник данных, не привязанный к району.
func windManualPressureEdited(id, code int) bool {
	if activePage != "wind" || changingUnits || windUpdating || id != 201 || code != EN_CHANGE {
		return false
	}
	// Изменение создаёт самостоятельный режим, но не уничтожает ранее
	// выбранный район: инженер может восстановить его по кнопке.
	windDraft.ManualW0 = true
	windDraft.DisplayUnit = windUnit
	changingUnits = true
	setZone(206, "Пользовательский")
	changingUnits = false
	windUpdateControls()
	return true
}
func windComboCommand(id int) bool {
	if activePage != "wind" || changingUnits || windUpdating {
		return false
	}
	switch id {
	case windUnitCombo:
		// Always read the old unit before changing conversion.
		old := windUnit
		if in, e := windReadForm(); e == nil {
			windDraft = in
			project.WindDraft = &in
		}
		windUnit = "kpa"
		if get(windUnitCombo) == "кгс/м²" {
			windUnit = "kgf"
		}
		windDraft.DisplayUnit = windUnit
		project.WindDraft = &windDraft
		if windCurrentResult != nil {
			windUnitAll(windCurrentResult, windUnit)
			if windCurrentBlockID > 0 {
				project.putWindCalc(windCurrentBlockID, *windCurrentResult)
			}
		}
		if old != windUnit {
			project.WindDraftRaw = nil
		}
		windSetForm()
		return true
	case windVariantCombo:
		if windCurrentResult != nil && len(windCurrentResult.Variants) > 0 {
			idx := int(send(windVariantCombo, CB_GETCURSEL, 0, 0))
			if idx >= 0 && idx < len(windCurrentResult.Variants) {
				windCurrentResult.SelectedVariant = idx
				windCurrentResult.SelectedZone = 0
				if windCurrentBlockID > 0 {
					project.putWindCalc(windCurrentBlockID, *windCurrentResult)
				}
				windUpdating = true
				windRefreshZoneCombo()
				windUpdating = false
				windShowResult()
			}
		}
		return true
	case windZoneCombo:
		if windCurrentResult != nil && len(windCurrentResult.Variants) > 0 {
			idx := int(send(windZoneCombo, CB_GETCURSEL, 0, 0))
			v := windCurrentResult.Variants[windCurrentResult.SelectedVariant]
			if idx >= 0 && idx < len(v.Zones) {
				windCurrentResult.SelectedZone = idx
				if windCurrentBlockID > 0 {
					project.putWindCalc(windCurrentBlockID, *windCurrentResult)
				}
				windShowResult()
			}
		}
		return true
	case windGeometryCombo, windPulseCombo, windStructureCombo, 206, 224:
		if id == windStructureCombo {
			code := windStructureID(get(windStructureCombo))
			windDraft.StructuralType = code
			windDraft.Delta = windStructureDamping(code)
			set(windDampingInput, decimalInput(windDraft.Delta, 3))
		}
		if id == 206 && get(206) != "Пользовательский" {
			windDraft.ManualW0 = false
			windDraft.Region = windDBRegion(get(206))
			if db, e := getWindDB(); e == nil {
				if w0, e := db.tableValue("11.1", "w0_kPa", windDraft.Region); e == nil {
					changingUnits = true
					set(201, decimalInput(windPressureValue(w0), 5))
					changingUnits = false
					windDraft.W0KPa = w0
				}
			}
		}
		windUpdateControls()
		return true
	}
	return false
}
func windSyncSite() {
	s := project.Site
	if s.WindZone == "" || windTerrainKey(s.WindTerrain) == "" {
		return
	}
	windDraft.Region = windDBRegion(s.WindZone)
	windDraft.Terrain = windTerrainKey(s.WindTerrain)
	windDraft.ManualW0 = false
	if project.WindDraftRaw != nil {
		project.WindDraftRaw[206] = windDisplayInputRegion(windDraft.Region)
		project.WindDraftRaw[224] = windDraft.Terrain
	}
}
func windTakeFromSite() {
	if project.Site.WindZone == "" || windTerrainKey(project.Site.WindTerrain) == "" {
		windPreserveDraft()
		siteReturnPage = "wind"
		switchPage("site")
		message("Укажите в разделе «Местность» ветровой район и тип местности. После возврата они будут перенесены в расчёт ветра.", false)
		return
	}
	if in, e := windReadForm(); e == nil {
		windDraft = in
	}
	windSyncSite()
	windSetForm()
}
func windPreserveDraft() {
	raw := map[int]string{}
	for _, id := range []int{201, 204, 221, 222, 228, windRoofInput, windStepInput, windDampingInput, 206, 224, windGeometryCombo, windPulseCombo, windStructureCombo} {
		raw[id] = get(id)
	}
	project.WindDraftRaw = raw
	if in, e := windReadForm(); e == nil {
		windDraft = in
		project.WindDraft = &in
		project.WindDraftRaw = nil
	}
}
func windSelectedCalculation() (*WindCalculation, *WindVariant, *WindZoneCalculation) {
	if windCurrentResult == nil {
		return nil, nil, nil
	}
	c := windCurrentResult
	if len(c.Variants) == 0 {
		return c, nil, nil
	}
	if c.SelectedVariant < 0 || c.SelectedVariant >= len(c.Variants) {
		return c, nil, nil
	}
	v := &c.Variants[c.SelectedVariant]
	if c.SelectedZone < 0 || c.SelectedZone >= len(v.Zones) {
		return c, v, nil
	}
	z := &v.Zones[c.SelectedZone]
	return &z.Calculation, v, z
}
func windShowResult() {
	c, v, z := windSelectedCalculation()
	if c == nil {
		set(windEpureNumbers, "Эпюры ещё не построены.\r\nВведите геометрию и нажмите «Рассчитать».")
		set(windResultInfo, "")
		set(windReportSummary, "")
	} else {
		f := 1. / 1000.
		if windUnit == "kgf" {
			f = kgfPerKN / 1000
		}
		name := "Расчётная поверхность"
		if z != nil {
			name = z.Name
		}
		s := fmt.Sprintf("%s\r\nЗона %s; c = %.3f\r\n", name, c.Input.AeroZone, c.Result.C)
		if z != nil && z.Surface == "roof" {
			s += fmt.Sprintf("Кровля: β = %.2f°; %s, вариант %s.\r\n", c.Input.RoofPitchDeg, c.Input.AeroTable, z.Alternative)
			s += "Расчётный уровень ветрового давления: H (конёк)."
		} else {
			if z != nil {
				s += fmt.Sprintf("Участок стены: %.3f–%.3f м\r\n", z.From, z.To)
			}
			if v != nil && v.E > 0 {
				s += fmt.Sprintf("e = min(b, 2H) = %.3f м", v.E)
			}
		}
		if z != nil && v != nil {
			dims := windZoneLengthsText(v, z)
			if dims != "" {
				s += "\r\n" + dims
			}
		}
		set(windResultInfo, s)
		if z != nil && z.Surface == "roof" {
			set(windResultGraphLabel, "ЭПЮРА НА УЧАСТКЕ КРОВЛИ: ОТ КАРНИЗА ДО КОНЬКА")
		} else {
			set(windResultGraphLabel, "ЭПЮРА ВЕТРОВОГО ДАВЛЕНИЯ ПО ВЫСОТЕ")
		}
		var b strings.Builder
		fmt.Fprintf(&b, "z, м | wₘ | wₚ | wн | wрасч (%s)\r\n", windPressureUnitName())
		for _, point := range c.Profile {
			fmt.Fprintf(&b, "%6.2f | %6.3f | %6.3f | %6.3f | %6.3f\r\n", point.Z, point.Result.MeanPa*f, point.Result.PulsePa*f, point.Result.NormPa*f, point.Result.DesignPa*f)
		}
		warnings := append([]string(nil), c.Result.Warnings...)
		if windCurrentResult != nil && len(windCurrentResult.Variants) > 0 {
			warnings = append(warnings, windCurrentResult.Result.Warnings...)
		}
		if len(warnings) > 0 {
			b.WriteString("\r\nПРЕДУПРЕЖДЕНИЯ (не включаются в отчёт):\r\n")
			b.WriteString(strings.Join(warnings, "; \r\n"))
		}
		set(windEpureNumbers, b.String())
		set(windReportSummary, fmt.Sprintf("wₘ — средняя (статическая) составляющая; wₚ — пульсационная; wн — полная нормативная; wрасч — расчётная.\r\nВариантов: %d. Точек текущей эпюры: %d. Отрицательные значения — отсос, положительные — давление.", len(windCurrentResult.Variants), len(c.Profile)))
	}
	for _, id := range []int{windEpureFigure, windResultFigure} {
		if controls[id] != 0 {
			invalidateSnow.Call(controls[id], 0, 1)
		}
	}
}
func windCalculate() {
	in, e := windReadForm()
	if e != nil {
		message(e.Error(), true)
		return
	}
	c, e := CalculateWindVariants(in)
	if e != nil {
		message(e.Error(), true)
		return
	}
	windDraft = in
	project.WindDraft = &in
	project.WindDraftRaw = nil
	if in.Geometry == "building" {
		for i, v := range c.Variants {
			if v.ID == "CROSS" {
				c.SelectedVariant = i
				break
			}
		}
	}
	windCurrentResult = &c
	windCurrentBlockID = project.putWindCalc(windCurrentBlockID, c)
	compositionRefresh()
	windSetTab(1)
}
func windNew() {
	windCurrentBlockID = 0
	windCurrentResult = nil
	project.WindDraftRaw = nil
	windDraft = windDefaultInput(windDBRegion(project.Site.WindZone))
	if t := windTerrainKey(project.Site.WindTerrain); t != "" {
		windDraft.Terrain = t
	}
	project.WindDraft = &windDraft
	windSetForm()
	windSetTab(0)
}
func windSaveNew() {
	if windCurrentResult == nil {
		message("Сначала выполните расчёт", true)
		return
	}
	title := "Ветровая нагрузка"
	for _, b := range project.SavedBlocks {
		if b.ID == windCurrentBlockID {
			title = b.Title
			break
		}
	}
	if name, ok := askReportBlockName(title); ok {
		project.renameBlock(windCurrentBlockID, name)
		windNew()
		compositionRefresh()
	}
}
func windExportSelected(docx bool) {
	if windCurrentResult == nil {
		message("Сначала выполните расчёт", true)
		return
	}
	ext, pattern := "xlsx", "Расчёт Excel (XLSX)"
	if docx {
		ext, pattern = "docx", "Отчёт Word (DOCX)"
	}
	path := chooseSave(ext, pattern, "Ветровая нагрузка."+ext, false)
	if path == "" {
		return
	}
	report := TempReport{Version: 1, Winds: []WindCalculation{*windCurrentResult}, WindTitles: []string{"Ветровая нагрузка"}}
	if docx {
		saveAndNotify(path, func(file string) error { return exportTempWord(report, file) })
	} else {
		saveAndNotify(path, func(file string) error { return exportTempExcel(report, file) })
	}
}
func windCommand(id int) bool {
	switch id {
	case windInputTab:
		windSetTab(0)
		return true
	case windResultTab:
		windSetTab(1)
		return true
	case windRestoreRegion:
		windDraft.ManualW0 = false
		if windDraft.Region == "" {
			windDraft.Region = "VII"
		}
		changingUnits = true
		setZone(206, windDisplayInputRegion(windDraft.Region))
		changingUnits = false
		windSetForm()
		return true
	case windGammaHelp:
		message("СП 20.13330.2016, п. 11.4:\nДля основной и пиковой ветровых нагрузок коэффициент надёжности по нагрузке γf = 1,4.\nПри расчёте резонансного вихревого возбуждения γf = 1,0.\n\nВ пункте 11.4 нет правила автоматически повышать γf до 1,5 по высоте здания. Для специальных сооружений действуют соответствующие нормативные документы.\nВведённое значение допускается скорректировать вручную.", false)
		return true
	case windPulseHelp:
		switchPage("wind_pulse_help")
		windHelpShow(0)
		return true
	case windHelpBack:
		switchPage("wind")
		return true
	case windTerrainHelp:
		message("СП 20.13330.2016, п. 11.1.6:\nА — открытые побережья, сельская местность, степи, пустыни, лесостепь и тундра; застройка ниже 10 м.\nВ — городские территории, лесные массивы, регулярные препятствия выше 10 м.\nС — плотная городская застройка зданиями выше 25 м.\nТип определяется по наветренной стороне на расстоянии 30H при H < 60 м или 2 км при H > 60 м.", false)
		return true
	case 235:
		windNew()
		return true
	case 236:
		windSaveNew()
		return true
	case 237:
		windExportSelected(true)
		return true
	case 238:
		windExportSelected(false)
		return true
	}
	return false
}
func windPaintFigure(dc uintptr, x, y, w, h int, kind string) bool {
	path := filepath.Join(storageRoot(), "Resources", "Wind", kind)
	data, e := os.ReadFile(path)
	if e != nil {
		return false
	}
	config, e := png.DecodeConfig(bytes.NewReader(data))
	if e != nil || config.Width <= 0 || config.Height <= 0 {
		return false
	}
	scale := math.Min(float64(w)/float64(config.Width), float64(h)/float64(config.Height))
	ww, hh := int(float64(config.Width)*scale), int(float64(config.Height)*scale)
	return snowPaintPNG(dc, x+(w-ww)/2, y+(h-hh)/2, ww, hh, data)
}
func windDrawString(dc uintptr, x, y int, s string, color uintptr) {
	u := syscall.StringToUTF16(s)
	g32.NewProc("SetBkMode").Call(dc, 1)
	g32.NewProc("SetTextColor").Call(dc, color)
	g32.NewProc("TextOutW").Call(dc, uintptr(x), uintptr(y), uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1))
}
func windStroke(dc uintptr, points [][2]int, color uintptr, width int) {
	if len(points) < 2 {
		return
	}
	pen, _, _ := g32.NewProc("CreatePen").Call(0, uintptr(width), color)
	old, _, _ := g32.NewProc("SelectObject").Call(dc, pen)
	g32.NewProc("MoveToEx").Call(dc, uintptr(points[0][0]), uintptr(points[0][1]), 0)
	for _, p := range points[1:] {
		g32.NewProc("LineTo").Call(dc, uintptr(p[0]), uintptr(p[1]), 0)
	}
	g32.NewProc("SelectObject").Call(dc, old)
	g32.NewProc("DeleteObject").Call(pen)
}

// Draw roof strips for the selected normative sign alternative. The roof
// rectangles are stored in wind-aligned metres, not arbitrary screen pixels.
func windDrawRoofPlan(dc uintptr, left, top, pw, ph int, v *WindVariant, active *WindZoneCalculation) {
	if v == nil || active == nil || active.Surface != "roof" {
		return
	}
	xlength, ylength := v.AlongM, v.AcrossM
	if xlength <= 0 || ylength <= 0 {
		return
	}
	convert := func(r WindRoofRect) (int, int, int, int) {
		if v.ID == "RIDGE" {
			return left + int(float64(pw)*r.X0/xlength), top + int(float64(ph)*r.Y0/ylength), left + int(float64(pw)*r.X1/xlength), top + int(float64(ph)*r.Y1/ylength)
		}
		// In the CROSS view the wind arrow points from bottom to top.
		return left + int(float64(pw)*r.Y0/ylength), top + ph - int(float64(ph)*r.X1/xlength), left + int(float64(pw)*r.Y1/ylength), top + ph - int(float64(ph)*r.X0/xlength)
	}
	for _, z := range v.Zones {
		if z.Surface != "roof" || z.Alternative != active.Alternative {
			continue
		}
		for j, r := range z.Rectangles {
			x1, y1, x2, y2 := convert(r)
			if x2-x1 < 1 || y2-y1 < 1 {
				continue
			}
			col, thick := uintptr(0x777777), 1
			if z.Code == active.Code {
				col, thick = 0xA05020, 3
			}
			windStroke(dc, [][2]int{{x1, y1}, {x2, y1}, {x2, y2}, {x1, y2}, {x1, y1}}, col, thick)
			if j == 0 && x2-x1 >= 17 && y2-y1 >= 17 {
				windDrawString(dc, (x1+x2)/2-4, (y1+y2)/2-7, z.Code, 0x222222)
			}
		}
	}
}

// Schematic plan, drawn from the SAME WindVariant which drives the selected
// calculation. Never show normative images of unrelated structures here.
// All drawing coordinates are derived from the saved zone geometry. End-wall
// views omit side-wall zones, and roof views omit every wall zone.
func windDrawZoneDiagram(dc uintptr, x, y, w, h int, v *WindVariant, active *WindZoneCalculation) {
	if v == nil || w < 180 || h < 130 {
		return
	}
	roof := active != nil && active.Surface == "roof"
	endOnly := active != nil && active.Surface != "roof" && (active.Code == "D" || active.Code == "E")
	heading := "План здания. " + v.Name
	if v.ID == "W" {
		heading = v.Name
	}
	windDrawString(dc, x+6, y+4, heading, 0x222222)
	if v.ID == "W" {
		left, top, pw := x+20, y+75, w-60
		cuts := []float64{0, v.AlongM}
		for _, z := range v.Zones {
			cuts = append(cuts, z.From, z.To)
		}
		ax := windAxis(cuts, left, pw)
		windStroke(dc, [][2]int{{left, top}, {left + pw, top}}, 0x222222, 2)
		for _, z := range v.Zones {
			from, to := ax.px(z.From), ax.px(z.To)
			col, thick := uintptr(0x777777), 2
			if active != nil && active.Code == z.Code {
				col, thick = 0xA05020, 4
			}
			windStroke(dc, [][2]int{{from, top}, {to, top}}, col, thick)
			windDrawString(dc, (from+to)/2-5, top-25, z.Code, 0x222222)
			yy := top + 13
			windStroke(dc, [][2]int{{from, yy - 4}, {from, yy}, {to, yy}, {to, yy - 4}}, 0x777777, 1)
			windDrawString(dc, from, yy+9, decimalInput(z.To-z.From, 2)+" м", 0x333333)
		}
		windDrawString(dc, left+pw/2+8, top+88, "Ветер", 0x222222)
		windStroke(dc, [][2]int{{left + pw/2, top + 125}, {left + pw/2, top + 79}, {left + pw/2 - 6, top + 88}, {left + pw/2, top + 79}, {left + pw/2 + 6, top + 88}}, 0x202020, 2)
		return
	}
	left, top := x+86, y+58
	pw, ph := w-164, h-151
	if ph > 168 {
		ph = 168
	}
	if ph < 50 {
		ph = 50
	}
	if pw < 85 {
		pw = 85
	}
	alt := ""
	if roof {
		alt = active.Alternative
	}
	boxes, xCuts, yCuts := windDrawingRects(*v, roof, alt)
	ax, ay := windAxis(xCuts, left, pw), windAxis(yCuts, top, ph)
	windStroke(dc, [][2]int{{left, top}, {left + pw, top}, {left + pw, top + ph}, {left, top + ph}, {left, top}}, 0x222222, 2)
	if !endOnly {
		for _, r := range boxes {
			x1, x2 := ax.px(r.x0), ax.px(r.x1)
			y1, y2 := ay.px(r.y0), ay.px(r.y1)
			col, thick := uintptr(0x888888), 1
			if active != nil && active.Code == r.code {
				col, thick = 0xA05020, 3
			}
			if roof {
				windStroke(dc, [][2]int{{x1, y1}, {x2, y1}, {x2, y2}, {x1, y2}, {x1, y1}}, col, thick)
				if x2-x1 >= 17 && y2-y1 >= 17 {
					windDrawString(dc, (x1+x2)/2-4, (y1+y2)/2-8, r.code, 0x222222)
				} else {
					// A label outside the narrow patch with a short leader.
					tx := x1 + 2
					if x2-x1 < 17 {
						tx = left - 14
					}
					windDrawString(dc, tx, (y1+y2)/2-7, r.code, 0x222222)
					if x2-x1 < 17 {
						windStroke(dc, [][2]int{{tx + 9, (y1 + y2) / 2}, {x1 + 2, (y1 + y2) / 2}}, 0x777777, 1)
					}
				}
			} else if v.ID == "RIDGE" {
				// A/B/C are line segments along the side-wall contours.
				windStroke(dc, [][2]int{{x1, top}, {x2, top}}, col, thick)
				windStroke(dc, [][2]int{{x1, top + ph}, {x2, top + ph}}, col, thick)
				windStroke(dc, [][2]int{{x1, top - 4}, {x1, top + 4}}, 0x777777, 1)
				windStroke(dc, [][2]int{{x2, top - 4}, {x2, top + 4}}, 0x777777, 1)
				windDrawString(dc, (x1+x2)/2-4, top+5, r.code, 0x222222)
			} else {
				windStroke(dc, [][2]int{{left, y1}, {left, y2}}, col, thick)
				windStroke(dc, [][2]int{{left + pw, y1}, {left + pw, y2}}, col, thick)
				windStroke(dc, [][2]int{{left - 4, y1}, {left + 4, y1}}, 0x777777, 1)
				windStroke(dc, [][2]int{{left - 4, y2}, {left + 4, y2}}, 0x777777, 1)
				windDrawString(dc, left+4, (y1+y2)/2-8, r.code, 0x222222)
			}
		}
	}
	if !roof && endOnly {
		// End-wall result: the plan identifies only the requested end face.
		if v.ID == "RIDGE" {
			edge := left
			if active.Code == "E" {
				edge = left + pw
			}
			windStroke(dc, [][2]int{{edge, top}, {edge, top + ph}}, 0xA05020, 4)
			if active.Code == "D" {
				windDrawString(dc, edge-16, top+ph/2-12, "D", 0x222222)
			} else {
				windDrawString(dc, edge+4, top+ph/2-12, "E", 0x222222)
			}
		} else {
			edge := top + ph
			if active.Code == "E" {
				edge = top
			}
			windStroke(dc, [][2]int{{left, edge}, {left + pw, edge}}, 0xA05020, 4)
			windDrawString(dc, left+pw/2-5, edge+7, active.Code, 0x222222)
		}
	}
	// All building plans include a clearly identifiable ridge. It does not
	// exist when h==0. Never draw a dashed ridge or an axis-break mark.
	if active != nil && active.Calculation.Input.RoofRise > 0 {
		mid := top + ph/2
		windStroke(dc, [][2]int{{left, mid}, {left + pw, mid}}, 0x202020, 3)
		windDrawString(dc, left+pw-56, mid-20, "Конёк", 0x222222)
	}
	// Wind arrow and its caption form one grouped, centred symbol.
	if v.ID == "RIDGE" {
		cy := top + ph/2
		windStroke(dc, [][2]int{{left - 54, cy}, {left - 31, cy}, {left - 40, cy - 5}, {left - 31, cy}, {left - 40, cy + 5}}, 0x202020, 2)
		windDrawString(dc, left-71, cy+8, "Ветер", 0x222222)
	} else {
		cx := left + pw/2
		windStroke(dc, [][2]int{{cx, top + ph + 55}, {cx, top + ph + 27}, {cx - 6, top + ph + 35}, {cx, top + ph + 27}, {cx + 6, top + ph + 35}}, 0x202020, 2)
		windDrawString(dc, cx+11, top+ph+33, "Ветер", 0x222222)
	}
	if endOnly {
		return
	}
	// Dimension values are mandatory even for narrow physical strips.
	for k := 0; k < len(xCuts)-1; k++ {
		a, b := ax.px(xCuts[k]), ax.px(xCuts[k+1])
		yy := top - 9
		windStroke(dc, [][2]int{{a, yy + 3}, {a, yy}, {b, yy}, {b, yy + 3}}, 0x777777, 1)
		tx := (a+b)/2 - 14
		ty := yy - 19
		if b-a < 55 {
			tx = a
			ty = yy - 36 - 14*(k%2)
		}
		windDrawString(dc, tx, ty, decimalInput(xCuts[k+1]-xCuts[k], 2)+" м", 0x333333)
	}
	lastY := top - 19
	for k := 0; k < len(yCuts)-1; k++ {
		a, b := ay.px(yCuts[k]), ay.px(yCuts[k+1])
		xx := left + pw + 9
		windStroke(dc, [][2]int{{xx - 3, a}, {xx, a}, {xx, b}, {xx - 3, b}}, 0x777777, 1)
		mid := (a + b) / 2
		yy := mid - 8
		if yy < lastY+17 {
			yy = lastY + 17
		}
		windStroke(dc, [][2]int{{xx + 1, mid}, {xx + 5, yy + 7}}, 0x777777, 1)
		windDrawString(dc, xx+8, yy, decimalInput(yCuts[k+1]-yCuts[k], 2)+" м", 0x333333)
		lastY = yy
	}
	if roof {
		names := []string{}
		for _, z := range v.Zones {
			if z.Surface == "roof" && z.Alternative == alt {
				seen := false
				for _, q := range names {
					if q == z.Code {
						seen = true
						break
					}
				}
				if !seen {
					names = append(names, z.Code)
				}
			}
		}
		windDrawString(dc, x+5, top+ph+54, "Зоны кровли: "+strings.Join(names, ", "), 0x333333)
	}
}
func windPaint(wp, lp uintptr) bool {
	if lp == 0 || wp != windNormFigure && wp != windEpureFigure && wp != windResultFigure {
		return false
	}
	d := (*snowDrawStruct)(unsafe.Pointer(lp))
	dc := d.HDC
	rect := d.Rect
	white, _, _ := g32.NewProc("CreateSolidBrush").Call(0xffffff)
	u32.NewProc("FillRect").Call(dc, uintptr(unsafe.Pointer(&rect)), white)
	g32.NewProc("DeleteObject").Call(white)
	left, top := int(rect.Left), int(rect.Top)
	w, h := int(rect.Right-rect.Left), int(rect.Bottom-rect.Top)
	if wp == windNormFigure {
		if windObjectGeometry() == "free_wall" {
			windDrawString(dc, left+12, top+12, "ОТДЕЛЬНО СТОЯЩАЯ СТЕНА: L и H", 0x222222)
			windPaintFigure(dc, left+5, top+52, w-10, h*75/100, "wind_fence_H_L.png")
		} else {
			windDrawString(dc, left+12, top+8, "ТОРЦЕВОЙ ФАСАД: L, H и h", 0x222222)
			windPaintFigure(dc, left+8, top+34, w-16, h*44/100, "wind_house_front_L_H_h.png")
			windDrawString(dc, left+12, top+h/2, "ПРОДОЛЬНЫЙ ФАСАД: B, H и h", 0x222222)
			windPaintFigure(dc, left+8, top+h/2+27, w-16, h*44/100, "wind_house_side_B_H_h.png")
		}
		return true
	}
	c, v, z := windSelectedCalculation()
	if wp == windResultFigure {
		if v != nil {
			windDrawZoneDiagram(dc, left+3, top+3, w-6, h-6, v, z)
		}
		return true
	}
	if c == nil || len(c.Profile) == 0 {
		windDrawString(dc, left+18, top+18, "Эпюра появится после расчёта.", 0x222222)
		return true
	}
	x0 := left + 70
	y0 := top + 52
	chartW := w - 105
	chartH := h - 142
	if chartW < 120 || chartH < 110 {
		return true
	}
	minP, maxP := 0., 0.
	minObserved, maxObserved := 0., 0.
	for _, v := range c.Profile {
		for _, p := range []float64{v.Result.MeanPa, v.Result.NormPa, v.Result.DesignPa} {
			if p < minP {
				minP = p
			}
			if p > maxP {
				maxP = p
			}
			if p < minObserved {
				minObserved = p
			}
			if p > maxObserved {
				maxObserved = p
			}
		}
	}
	if maxP-minP < 1e-8 {
		minP -= 1
		maxP += 1
	}
	margin := (maxP - minP) * 0.08
	minP -= margin
	maxP += margin
	if maxP <= 0 {
		maxP = math.Max(1, -minP*.25)
	}
	if minP >= 0 {
		minP = -math.Max(1, maxP*.25)
	}
	scale := func(p float64) int { return x0 + int((p-minP)/(maxP-minP)*float64(chartW)) }
	zMin, zMax := c.Profile[0].Z, c.Profile[len(c.Profile)-1].Z
	if zMax <= zMin {
		zMax = zMin + 1
	}
	height := func(z float64) int { return y0 + int((zMax-z)/(zMax-zMin)*float64(chartH)) }
	for j := 0; j <= 4; j++ {
		z := zMin + (zMax-zMin)*float64(j)/4
		y := height(z)
		windStroke(dc, [][2]int{{x0, y}, {x0 + chartW, y}}, 0xE6E6E6, 1)
		windDrawString(dc, left+5, y-7, fmt.Sprintf("%.1f м", z), 0x222222)
	}
	zero := scale(0)
	windStroke(dc, [][2]int{{zero, y0}, {zero, y0 + chartH}}, 0x555555, 2)
	windDrawString(dc, left+12, top+9, "wₘ — серый; wн — синий; wрасч — чёрный", 0x222222)
	for _, curve := range []struct {
		f     func(WindResult) float64
		color uintptr
		width int
	}{
		{func(r WindResult) float64 { return r.MeanPa }, 0x808080, 1},
		{func(r WindResult) float64 { return r.NormPa }, 0xD07500, 2},
		{func(r WindResult) float64 { return r.DesignPa }, 0x121212, 3},
	} {
		line := make([][2]int, 0, len(c.Profile))
		for _, p := range c.Profile {
			line = append(line, [2]int{scale(curve.f(p.Result)), height(p.Z)})
		}
		windStroke(dc, line, curve.color, curve.width)
	}
	f := 1. / 1000.
	if windUnit == "kgf" {
		f = kgfPerKN / 1000
	}
	// Side labels belong BETWEEN the axis and its numeric tick marks.
	// They are anchored to opposite plot edges, not to the zero coordinate.
	labelY := y0 + chartH + 5
	if minObserved < -1e-12 {
		windDrawString(dc, x0, labelY, "Разрежение", 0x222222)
	}
	if maxObserved > 1e-12 && chartW > 220 {
		windDrawString(dc, x0+chartW-83, labelY, "Давление", 0x222222)
	}
	for j := 0; j <= 4; j++ {
		p := minP + (maxP-minP)*float64(j)/4
		windDrawString(dc, scale(p)-23, y0+chartH+29, fmt.Sprintf("%.2f", p*f), 0x222222)
	}
	windDrawString(dc, x0, y0+chartH+53, "Горизонтальная ось: "+windPressureUnitName(), 0x333333)
	return true
}
func windDBRegion(zone string) string {
	if zone == "IA" || zone == "Iа" || zone == "Ia" {
		return "Ia"
	}
	return zone
}
func windDisplayInputRegion(zone string) string {
	if zone == "IA" || zone == "Ia" {
		return "Iа"
	}
	return zone
}
