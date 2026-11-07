//go:build windows

package main

import (
	"fmt"
	"math"
	"strings"
	"syscall"
	"unsafe"
)

const (
	snowShape             = 14001
	snowRegion            = 701
	snowGround            = 702
	snowRestore           = 703
	snowApply             = 704
	snowTerrain           = 14005
	snowSpan              = 14006
	snowWidth             = 14007
	snowAngle             = 14008
	snowRise              = 14009
	snowCount             = 14010
	snowStep              = 14011
	snowUpper             = 14012
	snowLower             = 14013
	snowCE                = 14014
	snowCT                = 14015
	snowRidge             = 14016
	snowSensitive         = 14017
	snowCalcButton        = 14018
	snowCasesList         = 14019
	snowDiagram           = 14020
	snowDetail            = 14021
	snowSave              = 14022
	snowRemove            = 14023
	snowExportWord        = 14024
	snowExportExcel       = 14025
	snowGamma             = 14026
	snowLowerShape        = 14027
	snowStatus            = 705
	snowRoofHeight        = 14028
	snowJanuary           = 14029
	snowWinterWind        = 14030
	snowExposed           = 14031
	snowWarm              = 14032
	snowGammaReset        = 14033
	snowSketch            = 14034
	snowSkylightA         = 14035
	snowModeAngle         = 14036
	snowModeRise          = 14037
	snowRoofUShape        = 14100
	snowRoofUSpan         = 14101
	snowRoofUMode         = 14102
	snowRoofUValue        = 14103
	snowRoofULevel        = 14104
	snowRoofLShape        = 14105
	snowRoofLSpan         = 14106
	snowRoofLMode         = 14107
	snowRoofLValue        = 14108
	snowRoofLLevel        = 14109
	snowDomeObstacles     = 14110
	snowSlopeMode         = 14250
	snowSlopeValue        = 14251
	snowRightA            = 14252
	snowRightHF           = 14253
	snowRightWarm         = 14254
	snowUnits             = 14255
	snowSlopeMainLabel    = 14256
	snowRoofLModeLabel    = 14257
	snowLeftRoofCaption   = 14258
	snowRightRoofCaption  = 14259
	snowGroundLabel       = 14260
	snowMountainButton    = 14261
	snowMountainState     = 14262
	snowTabInputs         = 14263
	snowTabResults        = 14264
	snowZonePlan          = 14265
	snowNormDiagram       = 14266
	snowCasesCaption      = 14267
	snowSphereAlpha1      = 14268
	snowSphereBetaStep    = 14269
	snowSphereBoundsTitle = 14270
	snowSphereConeTitle   = 14271
	snowSphereDomeTitle   = 14272
	snowSphereStrict      = 14273
	snowSphereWarning     = 14274
)

// СТАТУС СХЕМЫ "ПЕРЕПАД ВЫСОТ" (shape="step"): ЗАГЛУШКА / ЭКСПЕРИМЕНТ.
// Начиная с 0.47 пункт скрыт из выбора пользователем до отдельной
// нормативной проработки нескольких вариантов Б.8. Код сохраняется,
// в том числе для внутренней проверки двух соседних пролётов.
// Не путать с самостоятельной схемой "Покрытие с парапетом" (shape="parapet", Б.13):
// её скрывать нельзя. При аудите НЕ считать код "step" готовым функционалом.
var snowKinds = []string{"Плоская", "Односкатная", "Двускатная", "Сводчатая", "С фонарём", "Резервуар: коническая", "Резервуар: сферическая", "Многопролётная двускатная (одинаковые)", "Многопролётная сводчатая (одинаковые)", "Шедовое покрытие", "Покрытие с парапетом"}
var snowKindKeys = []string{"flat", "mono", "gable", "vault", "skylight", "cone", "dome", "multi_gable", "multi_vault", "shed", "parapet"}
var snowDraft = defaultSnowCalc()
var snowCurrentBlockID int
var snowReturnFromMountain bool
var snowBeforeMountainSite Site
var snowDrawCase = 0
var snowShownView = 0
var snowCurrentTab = 0 // 0 = input, 1 = results
var snowInputHandles []uintptr
var snowResultHandles []uintptr
var snowBackButton uintptr

var snowSphereBelowGroundWarned bool
var snowSphereInvalidAngleWarned bool
var snowPrevSlopeMode = 0
var snowPrevRightSlopeMode = 0

// The control displays either kPa or kgf/m², but internally Sg is ALWAYS kPa.
// Preserve full precision across unit toggles when the field was not edited.
var snowGroundFieldLast string
var snowGroundFieldKpa float64
var snowGroundFieldUnit string

func snowGroundUnitText(kpa float64, unit string) string {
	if unit == "kgf" {
		return decimalInput(kpa*kgfPerKN, 1)
	}
	return decimalInput(kpa, 3)
}
func snowSetGroundKpa(kpa float64, unit string) {
	snowGroundFieldKpa = kpa
	snowGroundFieldUnit = unit
	snowGroundFieldLast = snowGroundUnitText(kpa, unit)
	set(snowGround, snowGroundFieldLast)
}
func snowReadGroundKpa() (float64, error) {
	raw := strings.TrimSpace(get(snowGround))
	if raw == snowGroundFieldLast {
		return snowGroundFieldKpa, nil
	}
	n, err := parseNumber(raw)
	if raw == "" {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if snowGroundFieldUnit == "kgf" {
		return kgfM2ToKpa(n), nil
	}
	return n, nil
}
func snowGroundUpdateUnitLabel(unit string) {
	if unit == "kgf" {
		set(snowGroundLabel, "Sg, кгс/м²")
	} else {
		set(snowGroundLabel, "Sg, кПа")
	}
}

var invalidateSnow = u32.NewProc("InvalidateRect")

func snowIndex(shape string) int {
	if shape == "wall" || shape == "step" || shape == "sphere_cone" {
		// Hidden standalone B.8 prototype: do not expose it through the selector.
		return 0
	}
	for i, k := range snowKindKeys {
		if k == shape {
			return i
		}
	}
	return 2
}
func snowSel(id int) int {
	v := send(id, 0x147, 0, 0)
	if int32(v) == -1 {
		return -1
	}
	return int(v)
}
func snowSetSel(id, n int) { send(id, 0x14e, uintptr(n), 0) }
func snowBox(id int, checked bool) {
	n := uintptr(0)
	if checked {
		n = 1
	}
	send(id, 0x00F1, n, 0)
}
func snowIsChecked(id int) bool { return send(id, 0x00F0, 0, 0) != 0 }
func snowNumber(id int) (float64, error) {
	s := strings.TrimSpace(get(id))
	if s == "" {
		return 0, nil
	}
	return parseNumber(s)
}

// A field is a label plus the appropriate edit or selection control.
// Hidden controls retain their values for switching between roof templates.
var snowFieldIDs = []int{snowSpan, snowWidth, snowAngle, snowRise, snowCount, snowStep, snowUpper, snowLower, snowLowerShape, snowRoofHeight, snowSkylightA}

func snowFieldLabel(id int) int { return id + 1000 }
func snowFieldName(id int) string {
	switch id {
	case snowSpan:
		return "L, м (пролёт)"
	case snowWidth:
		return "B, м (длина)"
	case snowAngle:
		return "α, ° (уклон)"
	case snowRise:
		return "f, м (подъём)"
	case snowCount:
		return "n, шт. (пролётов)"
	case snowStep:
		return "h, м (перепад)"
	case snowUpper:
		return "L1, м (верх)"
	case snowLower:
		return "L2, м (низ)"
	case snowLowerShape:
		return "Нижняя кровля"
	case snowRoofHeight:
		return "H конька, м"
	case snowSkylightA:
		return "a, м (фонарь)"
	}
	return "Размер"
}

var snowStepControls = []int{snowRoofUShape, snowRoofUSpan, snowRoofUMode, snowRoofUValue, snowRoofULevel, snowRoofLShape, snowRoofLSpan, snowRoofLMode, snowRoofLValue, snowRoofLLevel}

// Both roof columns use identical controls. The right column can be disabled;
// all values remain available for editing when re-enabled.
func snowRoofPartSelection(part SnowRoofPart, shapeCtrl, spanCtrl, modeCtrl, valueCtrl, heightCtrl int) {
	keys := []string{"flat", "mono", "gable", "vault", "skylight"}
	i := 0
	for n, k := range keys {
		if part.Shape == k {
			i = n
			break
		}
	}
	if shapeCtrl == snowRoofLShape {
		i++
	}
	snowSetSel(shapeCtrl, i)
	set(spanCtrl, decimalInput(part.Span, 3))
	snowSetSel(modeCtrl, 0)
	v := part.Angle
	if part.Shape == "vault" || part.SlopeMode == "height" {
		snowSetSel(modeCtrl, 1)
		v = part.Rise
	}
	set(valueCtrl, decimalInput(v, 3))
	h := part.RoofHeight
	if h <= 0 {
		h = part.EdgeLevel + part.Rise
	}
	set(heightCtrl, decimalInput(h, 3))
}
func snowReadRoofPart(shapeCtrl, spanCtrl, modeCtrl, valueCtrl, heightCtrl int) (SnowRoofPart, error) {
	p := SnowRoofPart{}
	shapes := []string{"flat", "mono", "gable", "vault", "skylight"}
	i := snowSel(shapeCtrl)
	if shapeCtrl == snowRoofLShape {
		i--
	}
	if i < 0 || i >= len(shapes) {
		return p, fmt.Errorf("выберите тип покрытия")
	}
	p.Shape = shapes[i]
	var err error
	if p.Span, err = snowNumber(spanCtrl); err != nil {
		return p, fmt.Errorf("L: %w", err)
	}
	if p.RoofHeight, err = snowNumber(heightCtrl); err != nil {
		return p, fmt.Errorf("H конька: %w", err)
	}
	if p.Span <= 0 {
		return p, fmt.Errorf("L должна быть больше нуля")
	}
	if p.RoofHeight <= 0 {
		return p, fmt.Errorf("H конька должна быть больше нуля")
	}
	v, err := snowNumber(valueCtrl)
	if err != nil {
		return p, fmt.Errorf("уклон/подъём: %w", err)
	}
	if p.Shape == "vault" || snowSel(modeCtrl) == 1 {
		p.SlopeMode = "height"
		p.Rise = v
	} else {
		p.SlopeMode = "angle"
		p.Angle = v
	}
	if p.Shape == "flat" {
		p.Angle = 0
		p.Rise = 0
	} else if p.SlopeMode == "angle" {
		if v < 0 || v >= 90 {
			return p, fmt.Errorf("угол α должен быть в пределах 0–90°")
		}
		run := p.Span
		if p.Shape == "gable" || p.Shape == "skylight" {
			run = p.Span / 2
		}
		p.Rise = run * math.Tan(v*math.Pi/180)
	}
	if p.Shape == "vault" && (p.Rise <= 0 || p.Rise > p.Span/2) {
		return p, fmt.Errorf("стрела свода f должна удовлетворять 0 < f ≤ L/2")
	}
	if p.Shape == "skylight" {
		p.SkylightWidth, _ = snowNumber(snowRightA)
		p.LanternHeight, _ = snowNumber(snowRightHF)
		run := (p.Span - p.SkylightWidth) / 2
		if run <= 0 || p.LanternHeight <= 0 {
			return p, fmt.Errorf("фонарь: требуются 0 < a < L и hf > 0")
		}
		if p.SlopeMode == "angle" {
			p.Rise = run * math.Tan(p.Angle*math.Pi/180)
		}
	}
	p.Angle = p.resolvedAngle()
	p.EdgeLevel = p.RoofHeight - p.Rise
	if p.EdgeLevel < 0 {
		return p, fmt.Errorf("H конька должна быть не меньше подъёма кровли")
	}
	p.WarmRoof = snowIsChecked(snowRightWarm)
	return p, nil
}
func snowFieldList(shape string) []int {
	switch shape {
	case "flat":
		return []int{snowRoofHeight, snowSpan, snowWidth}
	case "mono", "gable":
		return []int{snowRoofHeight, snowSpan, snowWidth, snowAngle, snowRise}
	case "vault":
		return []int{snowRoofHeight, snowSpan, snowWidth, snowRise}
	case "multi_gable":
		return []int{snowRoofHeight, snowSpan, snowWidth, snowAngle, snowRise, snowCount}
	case "multi_vault":
		return []int{snowRoofHeight, snowSpan, snowWidth, snowRise, snowCount}
	case "wall":
		return []int{snowRoofHeight, snowWidth, snowStep, snowLower, snowLowerShape, snowAngle, snowRise}
	case "step":
		return []int{snowWidth}
	case "parapet":
		return []int{snowRoofHeight, snowSpan, snowWidth, snowStep}
	case "skylight":
		return []int{snowRoofHeight, snowSpan, snowWidth, snowSkylightA, snowStep, snowAngle, snowRise}
	case "cone":
		return []int{snowRoofHeight, snowSpan, snowWidth, snowAngle, snowRise}
	case "dome":
		return []int{snowRoofHeight, snowSpan, snowWidth, snowRise}
	case "sphere_cone":
		return []int{snowRoofHeight, snowSpan, snowSkylightA, snowStep, snowSphereAlpha1, snowSphereBetaStep}
	}
	return []int{snowRoofHeight, snowSpan, snowWidth}
}
func snowShow(id int, show bool) {
	if controls[id] == 0 {
		return
	}
	cmd := uintptr(0)
	if show && activePage == "snow" {
		cmd = 5
	}
	pShow.Call(controls[id], cmd)
}

func snowSlopeEligible(shape string) bool {
	switch shape {
	case "mono", "gable", "multi_gable", "skylight", "cone", "sphere_cone", "wall", "step":
		return true
	}
	return false
}
func snowUseHeightInput() bool {
	return snowIsChecked(snowModeRise)
}
func snowEnableArg(v bool) uintptr {
	if v {
		return 1
	}
	return 0
}

// Unified geometry layout. No radio buttons and no shared geometry controls
// stacked on top of independent roof inputs.
func snowLayout() {
	i := snowSel(snowShape)
	shape := snowDraft.Shape
	if i >= 0 && i < len(snowKindKeys) {
		shape = snowKindKeys[i]
	}
	reservoir := shape == "cone" || shape == "dome" || shape == "sphere_cone"
	stepMode := shape == "step"
	if stepMode && snowSel(snowRoofLShape) == 0 {
		snowSetSel(snowRoofLShape, 1)
	}
	pairAllowed := shape == "flat" || shape == "mono" || shape == "gable" || shape == "vault" || shape == "skylight"
	if !pairAllowed {
		snowSetSel(snowRoofLShape, 0)
	}
	enable := u32.NewProc("EnableWindow")
	enable.Call(controls[snowRoofLShape], snowEnableArg(pairAllowed))
	rightOn := (pairAllowed || stepMode) && snowSel(snowRoofLShape) > 0
	for _, id := range []int{snowRoofLSpan, snowRoofLLevel, snowRoofLMode, snowRoofLValue, snowRightWarm, snowUpper} {
		enable.Call(controls[id], snowEnableArg(rightOn))
	}
	rightShape := "none"
	if rightOn {
		choices := []string{"none", "flat", "mono", "gable", "vault", "skylight"}
		j := snowSel(snowRoofLShape)
		if j < len(choices) {
			rightShape = choices[j]
		}
	}
	// Only relevant parameters are visible: each roof gets its own lantern.
	snowShow(snowSkylightA, shape == "skylight" || shape == "sphere_cone")
	snowShow(snowFieldLabel(snowSkylightA), shape == "skylight" || shape == "sphere_cone")
	snowShow(snowStep, shape == "skylight" || shape == "parapet" || shape == "sphere_cone")
	snowShow(snowFieldLabel(snowStep), shape == "skylight" || shape == "parapet" || shape == "sphere_cone")
	snowShow(snowRightA, rightShape == "skylight")
	snowShow(snowFieldLabel(snowRightA), rightShape == "skylight")
	snowShow(snowRightHF, rightShape == "skylight")
	snowShow(snowFieldLabel(snowRightHF), rightShape == "skylight")
	snowShow(snowDomeObstacles, shape == "dome")
	if controls[snowZonePlan] != 0 {
		invalidateSnow.Call(controls[snowZonePlan], 0, 1)
	}
	snowShow(snowRidge, shape == "gable" && !rightOn)
	snowShow(snowCount, shape == "multi_gable" || shape == "multi_vault" || shape == "shed")
	snowShow(snowFieldLabel(snowCount), shape == "multi_gable" || shape == "multi_vault" || shape == "shed")
	isVault := shape == "vault" || shape == "dome" || shape == "multi_vault"
	isFlat := shape == "flat" || shape == "shed" || shape == "parapet"
	if isVault {
		snowSetSel(snowSlopeMode, 1)
	}
	enable.Call(controls[snowSlopeMode], snowEnableArg(!isFlat && !isVault))
	enable.Call(controls[snowSlopeValue], snowEnableArg(!isFlat))
	for _, id := range []int{snowSlopeMainLabel, snowSlopeMode, snowSlopeValue} {
		snowShow(id, shape != "parapet" && shape != "shed")
	}

	if rightShape == "vault" {
		snowSetSel(snowRoofLMode, 1)
	}
	enable.Call(controls[snowRoofLMode], snowEnableArg(rightOn && rightShape != "flat" && rightShape != "vault"))
	enable.Call(controls[snowRoofLValue], snowEnableArg(rightOn && rightShape != "flat"))

	if reservoir {
		set(snowFieldLabel(snowSpan), "D, м (диаметр)")
	} else if shape == "shed" {
		set(snowFieldLabel(snowSpan), "l, м (шаг шеда)")
	} else {
		set(snowFieldLabel(snowSpan), "L₁, м")
	}
	if shape == "dome" {
		snowSetSel(snowSlopeMode, 1)
	}
	if shape == "sphere_cone" {
		set(snowFieldLabel(snowSkylightA), "r конуса, м")
		set(snowFieldLabel(snowStep), "f от стенки, м")
	} else if shape == "skylight" {
		set(snowFieldLabel(snowSkylightA), "a фонаря, м")
		set(snowFieldLabel(snowStep), "hf, м")
	} else if shape == "parapet" {
		set(snowFieldLabel(snowStep), "h парапета, м")
	}
	// Б.15а: independent visual groups; no length B for a circular building.
	sp := shape == "sphere_cone"
	for _, id := range []int{snowSphereAlpha1, snowSphereBetaStep, snowSphereBoundsTitle, snowSphereConeTitle, snowSphereDomeTitle, snowSphereStrict, snowSphereWarning, snowFieldLabel(snowSphereAlpha1), snowFieldLabel(snowSphereBetaStep)} {
		snowShow(id, sp)
	}
	if sp {
		snowShow(snowWidth, false)
		snowShow(snowFieldLabel(snowWidth), false)
		snowShow(snowRoofLShape, false)
		snowShow(snowRightRoofCaption, false)
		snowShow(snowWarm, false)
		snowShow(snowRightWarm, false)
		// Hide the entire unused adjacent-roof column, including its labels.
		for _, id := range []int{snowRoofLLevel, snowRoofLSpan, snowUpper, snowRoofLMode, snowRoofLValue, snowRoofLModeLabel, snowFieldLabel(snowRoofLLevel), snowFieldLabel(snowRoofLSpan), snowFieldLabel(snowUpper), snowRightA, snowRightHF, snowFieldLabel(snowRightA), snowFieldLabel(snowRightHF)} {
			snowShow(id, false)
		}
		snowShow(snowSlopeMainLabel, true)
		set(snowLeftRoofCaption, "КРУГОВОЕ ПОКРЫТИЕ")
		set(snowSlopeMainLabel, "Уклон / подъём конуса")
		for _, item := range []struct{ id, y int }{{snowRoofHeight, 426}, {snowSpan, 458}, {snowSkylightA, 527}, {snowStep, 628}} {
			tempMove(snowFieldLabel(item.id), 24, item.y, 145, 22)
			tempMove(item.id, 173, item.y-3, 86, 26)
		}
		tempMove(snowSlopeMainLabel, 24, 561, 175, 22)
		tempMove(snowSlopeMode, 201, 556, 112, 180)
		tempMove(snowSlopeValue, 322, 556, 100, 26)
		tempMove(snowFieldLabel(snowSphereAlpha1), 24, 661, 210, 22)
		tempMove(snowSphereAlpha1, 240, 658, 94, 26)
		tempMove(snowFieldLabel(snowSphereBetaStep), 24, 697, 210, 22)
		tempMove(snowSphereBetaStep, 240, 694, 94, 26)
	} else {
		set(snowLeftRoofCaption, "ЛЕВАЯ КРОВЛЯ")
		set(snowSlopeMainLabel, "Уклон / подъём")
		for _, item := range []struct{ id, y int }{{snowRoofHeight, 412}, {snowSpan, 443}, {snowWidth, 474}, {snowSkylightA, 541}, {snowStep, 572}} {
			tempMove(snowFieldLabel(item.id), 24, item.y, 111, 22)
			tempMove(item.id, 137, item.y-3, 122, 26)
		}
		tempMove(snowSlopeMainLabel, 24, 504, 112, 22)
		tempMove(snowSlopeMode, 136, 499, 72, 180)
		tempMove(snowSlopeValue, 211, 499, 48, 26)
		// Restore visibility of the normal first/second roof headers and inputs.
		snowShow(snowWidth, true)
		snowShow(snowFieldLabel(snowWidth), true)
		snowShow(snowRoofLShape, true)
		snowShow(snowRightRoofCaption, true)
		snowShow(snowWarm, true)
		snowShow(snowRightWarm, true)
		for _, id := range []int{snowRoofLLevel, snowRoofLSpan, snowUpper, snowRoofLMode, snowRoofLValue, snowRoofLModeLabel, snowFieldLabel(snowRoofLLevel), snowFieldLabel(snowRoofLSpan), snowFieldLabel(snowUpper)} {
			snowShow(id, true)
		}
	}
	if controls[snowSketch] != 0 {
		invalidateSnow.Call(controls[snowSketch], 0, 1)
	}
}

// Each tab shows its own native controls. The common application header and
// navigation remain visible, without embedding a browser or modal window.
func snowSetTab(n int) {
	snowCurrentTab = n
	if activePage != "snow" {
		return
	}
	for _, h := range snowInputHandles {
		if h != 0 {
			pShow.Call(h, snowVisibleTab(n == 0))
		}
	}
	for _, h := range snowResultHandles {
		if h != 0 {
			pShow.Call(h, snowVisibleTab(n == 1))
		}
	}
	// Back is hierarchical: results -> inputs -> home.
	if snowBackButton != 0 {
		caption := "← На главную"
		if n == 1 {
			caption = "← К исходным данным"
		}
		pSetText.Call(snowBackButton, uintptr(unsafe.Pointer(ptr(caption))))
	}
	if n == 0 {
		snowLayout()
	}
	snowResize()
}

// The normative figure and the chart are independent child windows.
// Neither the list nor its caption can overlap the figure or be covered by
// a repaint of the chart when the selected variant changes.
func snowResize() {
	if hwnd == 0 || activePage != "snow" {
		return
	}
	var r tempRect
	tempGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	w, h := int(r.Right), int(r.Bottom)
	if w < 820 || h < 660 {
		return
	}
	right := w - 24
	leftW := (w - 72) * 35 / 100
	if leftW < 280 {
		leftW = 280
	}
	if leftW > 405 {
		leftW = 405
	}
	chartX := 24 + leftW + 16
	chartW := right - chartX
	// Button panel anchored to the lower right corner of the client rectangle.
	btnTop := h - 117
	btnW := (chartW - 12) / 2
	tempMove(snowSave, chartX, btnTop, btnW, 38)
	tempMove(snowRemove, chartX+btnW+12, btnTop, chartW-btnW-12, 38)
	tempMove(snowExportWord, chartX, btnTop+46, btnW, 36)
	tempMove(snowExportExcel, chartX+btnW+12, btnTop+46, chartW-btnW-12, 36)
	// The calculation button is fixed at the bottom on the inputs tab.
	tempMove(snowCalcButton, 24, h-69, 495, 40)
	// The geometry panels on the inputs tab consume free horizontal space.
	tempMove(snowSketch, 547, 164, w-571, 260)
	tempMove(snowZonePlan, 547, 480, w-571, tempMax(150, h-560))
	// Results: normative geometry fills the LEFT, chart + selection stay RIGHT.
	listTop := btnTop - 128
	chartTop := 180
	chartH := listTop - chartTop - 28
	if chartH < 290 {
		chartH = 290
	}
	tempMove(snowNormDiagram, 24, 177, leftW, btnTop-188)
	tempMove(snowDiagram, chartX, chartTop, chartW, chartH)
	tempMove(snowCasesCaption, chartX, listTop-22, chartW, 20)
	listW := chartW * 56 / 100
	tempMove(snowCasesList, chartX, listTop, listW-8, 101)
	tempMove(snowDetail, chartX+listW, listTop, chartW-listW, 101)
	// Units and tabs follow the expanding window.
	tempMove(snowUnits, w-151, 125, 126, 220)
	tempMove(snowTabInputs, w-574, 91, 269, 33)
	tempMove(snowTabResults, w-293, 91, 269, 33)
	if controls[snowNormDiagram] != 0 {
		invalidateSnow.Call(controls[snowNormDiagram], 0, 1)
	}
	if controls[snowDiagram] != 0 {
		invalidateSnow.Call(controls[snowDiagram], 0, 1)
	}
}
func snowVisibleTab(b bool) uintptr {
	if b {
		return 5
	}
	return 0
}

func buildSnowUI() {
	buildingPage = "snow"
	navigation("Снеговая нагрузка", "СП 20.13330.2016, раздел 10 и приложение Б · нагрузка на горизонтальную проекцию")
	snowBackButton = controls[399]
	startInput := len(pageControls["snow"])
	label("КЛИМАТИЧЕСКИЕ ДАННЫЕ И КОЭФФИЦИЕНТЫ", 24, 96, 495, 22)
	label("Снеговой район", 24, 120, 160, 19)
	combo(snowRegion, snowOptions, 24, 139, 153)
	create("STATIC", "Sg, кПа", WS_CHILD|WS_VISIBLE, 189, 120, 111, 19, snowGroundLabel)
	edit(snowGround, "", 189, 139, 93, 26)
	button(snowRestore, "Из местности", 291, 139, 107, 29)
	create("STATIC", "Горный район: НЕТ", WS_CHILD|WS_VISIBLE, 405, 116, 114, 22, snowMountainState)
	button(snowMountainButton, "Горные районы", 405, 139, 114, 29)
	label("Тип местности (п. 11.1.6 СП 20.13330.2016)", 24, 174, 491, 18)
	combo(snowTerrain, []string{"А — открытые побережья, степи, тундра", "В — города и леса, препятствия выше 10 м", "С — плотная застройка, здания выше 25 м"}, 24, 192, 495)
	button(14038, "Температура и ветер по СП", 24, 230, 194, 28)
	label("T января, °C", 228, 215, 130, 18)
	edit(snowJanuary, "", 228, 234, 129, 25)
	label("V холодного периода, м/с", 364, 215, 170, 18)
	edit(snowWinterWind, "", 364, 234, 155, 25)
	label("γf — коэффициент надёжности", 24, 273, 235, 20)
	edit(snowGamma, "1,4", 262, 269, 73, 27)
	button(snowGammaReset, "Сброс до нормы 1,4", 343, 269, 176, 27)
	create("STATIC", "", WS_CHILD, 24, 300, 0, 0, snowStatus)
	create("BUTTON", "Крыша защищена от прямого ветра", WS_CHILD|WS_VISIBLE|0x00000003, 24, 301, 247, 18, snowExposed)
	// Additional uneven snow-loading variants are produced automatically; no checkbox.
	label("ГЕОМЕТРИЯ КРОВЛИ", 24, 321, 495, 23)
	create("STATIC", "ЛЕВАЯ КРОВЛЯ", WS_CHILD|WS_VISIBLE, 24, 345, 237, 22, snowLeftRoofCaption)
	create("STATIC", "ПРАВАЯ КРОВЛЯ", WS_CHILD|WS_VISIBLE, 271, 345, 248, 22, snowRightRoofCaption)
	combo(snowShape, snowKinds, 24, 369, 235)
	combo(snowRoofLShape, []string{"Нет", "Плоская", "Односкатная", "Двускатная", "Сводчатая", "С фонарём"}, 271, 369, 248)
	// Same rows and sizes, no overlapping controls.
	field := func(id int, name string, x, y int, ww int) {
		create("STATIC", name, WS_CHILD|WS_VISIBLE, x, y, 111, 22, snowFieldLabel(id))
		edit(id, "", x+113, y-3, ww-113, 26)
	}
	field(snowRoofHeight, "H конька, м", 24, 412, 235)
	field(snowRoofLLevel, "H конька, м", 271, 412, 248)
	field(snowSpan, "L₁, м", 24, 443, 235)
	field(snowRoofLSpan, "L₂, м", 271, 443, 248)
	field(snowWidth, "B, м (длина)", 24, 474, 235)
	field(snowUpper, "B, м (длина)", 271, 474, 248)
	create("STATIC", "Уклон / подъём", WS_CHILD|WS_VISIBLE, 24, 504, 112, 22, snowSlopeMainLabel)
	create("STATIC", "Уклон / подъём", WS_CHILD|WS_VISIBLE, 271, 504, 112, 22, snowRoofLModeLabel)
	combo(snowSlopeMode, []string{"α, град", "h / f, м"}, 136, 499, 72)
	combo(snowRoofLMode, []string{"α, град", "h / f, м"}, 383, 499, 76)
	edit(snowSlopeValue, "", 211, 499, 48, 26)
	edit(snowRoofLValue, "", 462, 499, 57, 26)
	field(snowSkylightA, "a фонаря, м", 24, 541, 235)
	field(snowRightA, "a фонаря, м", 271, 541, 248)
	field(snowStep, "hf фонаря, м", 24, 572, 235)
	field(snowRightHF, "hf фонаря, м", 271, 572, 248)
	create("BUTTON", "Неутеплённая с повышенным тепловыделением", WS_CHILD|WS_VISIBLE|0x00000003|BS_MULTILINE, 24, 607, 237, 38, snowWarm)
	create("BUTTON", "Неутеплённая с повышенным тепловыделением", WS_CHILD|WS_VISIBLE|0x00000003|BS_MULTILINE, 271, 607, 248, 38, snowRightWarm)
	create("BUTTON", "Купол: препятствия, шероховатость, защита от ветра", WS_CHILD|WS_VISIBLE|0x00000003, 24, 647, 495, 25, snowDomeObstacles)
	create("BUTTON", "Ходовые мостики или светоаэрационные устройства по коньку", WS_CHILD|0x00000003, 24, 645, 495, 30, snowRidge)
	create("STATIC", "n пролётов", WS_CHILD|WS_VISIBLE, 24, 647, 130, 22, snowFieldLabel(snowCount))
	edit(snowCount, "2", 156, 645, 101, 26)
	label("ОБОЗНАЧЕНИЯ РАЗМЕРОВ", 547, 136, 560, 22)
	create("STATIC", "", WS_CHILD|WS_VISIBLE|0x0000000D, 547, 164, 560, 260, snowSketch)
	label("ПЛАН РАСПОЛОЖЕНИЯ ЗОН (ПРИ НАЛИЧИИ)", 547, 453, 560, 22)
	create("STATIC", "", WS_CHILD|WS_VISIBLE|0x0000000D, 547, 480, 560, 250, snowZonePlan)
	create("STATIC", "ГАБАРИТЫ", WS_CHILD|WS_VISIBLE, 24, 400, 330, 22, snowSphereBoundsTitle)
	create("STATIC", "КОНИЧЕСКАЯ ЧАСТЬ", WS_CHILD|WS_VISIBLE, 24, 498, 330, 22, snowSphereConeTitle)
	create("STATIC", "СФЕРИЧЕСКАЯ ЧАСТЬ", WS_CHILD|WS_VISIBLE, 24, 598, 330, 22, snowSphereDomeTitle)
	create("STATIC", "α₁, град (касательная у стенки)", WS_CHILD|WS_VISIBLE, 24, 661, 210, 22, snowFieldLabel(snowSphereAlpha1))
	edit(snowSphereAlpha1, "", 240, 658, 94, 26)
	create("STATIC", "Шаг угла β, град", WS_CHILD|WS_VISIBLE, 24, 697, 210, 22, snowFieldLabel(snowSphereBetaStep))
	edit(snowSphereBetaStep, "15", 240, 694, 94, 26)
	create("BUTTON", "Считать строго по формуле", WS_CHILD|WS_VISIBLE|0x00000003, 24, 727, 480, 24, snowSphereStrict)
	create("STATIC", "", WS_CHILD|WS_VISIBLE|0x0000000D, 24, 755, 495, 78, snowSphereWarning)
	button(snowCalcButton, "РАССЧИТАТЬ", 24, 850, 495, 40)
	snowInputHandles = append([]uintptr(nil), pageControls["snow"][startInput:]...)
	startResult := len(pageControls["snow"])
	label("СХЕМА И ЭПЮРА СНЕГОВЫХ НАГРУЗОК", 24, 121, 720, 24)
	label("Единицы", 909, 125, 68, 20)
	combo(snowUnits, []string{"кПа", "кгс/м²"}, 982, 125, 125)
	create("STATIC", "", WS_CHILD|WS_VISIBLE|0x0000000D, 24, 177, 380, 600, snowNormDiagram)
	create("STATIC", "", WS_CHILD|WS_VISIBLE|0x0000000D, 420, 180, 687, 440, snowDiagram)
	create("STATIC", "РАСЧЁТНЫЕ ВАРИАНТЫ ЗАГРУЖЕНИЯ", WS_CHILD|WS_VISIBLE, 420, 635, 680, 20, snowCasesCaption)
	create("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|LBS_NOTIFY, 420, 660, 370, 101, snowCasesList)
	create("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|ES_MULTILINE|ES_READONLY|WS_VSCROLL, 802, 660, 305, 101, snowDetail)
	button(snowSave, "Сохранить в ведомость и новый расчёт", 547, 758, 349, 39)
	button(snowRemove, "Сбросить нагрузки", 906, 758, 201, 39)
	button(snowExportWord, "Отчёт Word", 547, 805, 270, 36)
	button(snowExportExcel, "Отчёт Excel", 832, 805, 275, 36)
	snowResultHandles = append([]uintptr(nil), pageControls["snow"][startResult:]...)
	button(snowTabInputs, "ИСХОДНЫЕ ДАННЫЕ", 547, 91, 269, 33)
	button(snowTabResults, "РЕЗУЛЬТАТЫ РАСЧЁТА", 828, 91, 279, 33)
	snowSetTab(0)
}
func snowFill(s SnowCalc) {
	snowDraft = s
	if s.Shape != "sphere_cone" {
		snowSphereBelowGroundWarned = false
		snowSphereInvalidAngleWarned = false
	}
	snowBox(snowRidge, s.RidgeEquipment)
	snowSetSel(snowShape, snowIndex(s.Shape))
	zone := s.Region
	if !s.GroundLocal && project.Snow.Enabled {
		zone, _, _ = project.SnowInputs()
	}
	for i, name := range snowOptions {
		if name == zone {
			snowSetSel(snowRegion, i)
			break
		}
	}
	unit := "kpa"
	if s.Units == "kgf" {
		unit = "kgf"
	}
	snowSetGroundKpa(s.Ground, unit)
	snowGroundUpdateUnitLabel(unit)
	set(snowRoofHeight, decimalInput(s.RoofHeight, 3))
	set(snowSpan, decimalInput(s.Span, 3))
	set(snowWidth, decimalInput(s.Width, 3))
	set(snowGamma, decimalInput(s.Gamma, 3))
	set(snowCount, fmt.Sprint(s.Count))
	set(snowSkylightA, decimalInput(s.SkylightWidth, 3))
	set(snowStep, decimalInput(s.H, 3))
	if s.BetaStep <= 0 {
		s.BetaStep = 15
	}
	set(snowSphereBetaStep, decimalInput(s.BetaStep, 3))
	snowBox(snowSphereStrict, s.SphereStrictFormula)
	if s.Shape == "sphere_cone" && s.Span/2 > s.SkylightWidth && s.Angle > 0 {
		set(snowSphereAlpha1, decimalInput(sphereConeTangentAngle(s), 2))
	} else {
		set(snowSphereAlpha1, "")
	}
	mode := 0
	v := s.Angle
	if s.SlopeMode == "height" || s.Shape == "vault" || s.Shape == "dome" || s.Shape == "multi_vault" {
		mode = 1
		v = s.Rise
	}
	snowSetSel(snowSlopeMode, mode)
	snowPrevSlopeMode = mode
	set(snowSlopeValue, decimalInput(v, 3))
	if s.JanuarySet {
		set(snowJanuary, decimalInput(s.JanuaryTemp, 2))
	} else {
		set(snowJanuary, "")
	}
	if s.WinterWindSet {
		set(snowWinterWind, decimalInput(s.WinterWind, 2))
	} else {
		set(snowWinterWind, "")
	}
	terr := 1
	if s.Terrain == "А" {
		terr = 0
	}
	if s.Terrain == "С" {
		terr = 2
	}
	snowSetSel(snowTerrain, terr)
	snowBox(snowExposed, !s.Exposed)
	snowBox(snowWarm, s.WarmRoof)
	snowBox(snowDomeObstacles, s.DomeObstacles)
	snowSetSel(snowUnits, 0)
	if s.Units == "kgf" {
		snowSetSel(snowUnits, 1)
	}
	if s.Shape == "step" {
		u, l := s.stepRoofs()
		if u.RoofHeight == 0 {
			u.RoofHeight = u.EdgeLevel + u.Rise
		}
		if l.RoofHeight == 0 {
			l.RoofHeight = l.EdgeLevel + l.Rise
		}
		if u.RoofHeight == 0 {
			u.RoofHeight = s.RoofHeight
		}
		set(snowRoofHeight, decimalInput(u.RoofHeight, 3))
		set(snowSpan, decimalInput(u.Span, 3))
		set(snowSlopeValue, decimalInput(u.Angle, 3))
		snowSetSel(snowSlopeMode, 0)
		// The left 'Перепад высот' template retains legacy upper roof data.
		snowRoofPartSelection(l, snowRoofLShape, snowRoofLSpan, snowRoofLMode, snowRoofLValue, snowRoofLLevel)
		snowPrevRightSlopeMode = snowSel(snowRoofLMode)
		set(snowUpper, decimalInput(s.Width, 3))
		set(snowSkylightA, "0")
		set(snowStep, "0")
	} else if s.RightRoof != nil && s.RightRoof.Shape != "none" {
		snowRoofPartSelection(*s.RightRoof, snowRoofLShape, snowRoofLSpan, snowRoofLMode, snowRoofLValue, snowRoofLLevel)
		set(snowUpper, decimalInput(s.RightRoof.Width, 3))
		set(snowRightA, decimalInput(s.RightRoof.SkylightWidth, 3))
		set(snowRightHF, decimalInput(s.RightRoof.LanternHeight, 3))
		snowBox(snowRightWarm, s.RightRoof.WarmRoof)
		snowPrevRightSlopeMode = snowSel(snowRoofLMode)
	} else {
		snowSetSel(snowRoofLShape, 0)
		set(snowRoofLLevel, decimalInput(s.RoofHeight, 3))
		set(snowRoofLSpan, decimalInput(s.Span, 3))
		set(snowUpper, decimalInput(s.Width, 3))
		snowSetSel(snowRoofLMode, 0)
		snowPrevRightSlopeMode = 0
		set(snowRoofLValue, "0")
		set(snowRightA, "0")
		set(snowRightHF, "0")
		snowBox(snowRightWarm, false)
	}
	snowLayout()
	snowUpdateStatus()
}
func snowBringSelectorsForward() {
	p := u32.NewProc("SetWindowPos")
	for _, id := range []int{snowShape, snowRoofLShape, snowSlopeMode, snowRoofLMode, snowRegion, snowTerrain, snowUnits} {
		if controls[id] != 0 {
			p.Call(controls[id], 0, 0, 0, 0, 0, 0x13)
		}
	}
}
func snowPageEntered() {
	if snowReturnFromMountain {
		current := project.Site
		changed := current.MountainID != snowBeforeMountainSite.MountainID || current.ElevationText() != snowBeforeMountainSite.ElevationText() || current.Region != snowBeforeMountainSite.Region || current.SnowZone != snowBeforeMountainSite.SnowZone
		// Only a confirmed change of mountain/site data replaces an earlier manual snow value.
		if changed {
			project.Snow = ClimateOverride{}
			snowDraft.GroundLocal = false
		}
		snowReturnFromMountain = false
	}
	s := snowDraft
	if project.SnowCalc != nil {
		s = *project.SnowCalc
	}
	// The Snow module always inherits an existing site's zone unless a user explicitly overrode it.
	zone, sg, over := project.SnowInputs()
	if !s.GroundLocal && sg > 0 {
		s.Region, s.Ground = zone, sg
	}
	if s.Ground == 0 && s.Region == "" && zone == "" && project.Site.Region == "" && project.Site.Place == "" {
		s.Region = "VIII"
	}
	if s.Ground == 0 && s.Region != "" {
		if v, ok := snowForZone(s.Region); ok {
			s.Ground = v
		}
	}
	s.GroundLocal = over
	if s.ClimateOrigin != "manual" && (project.Site.ClimateJanuarySet || project.Site.ClimateWindSet) {
		s.InheritClimate(project.Site)
	}
	snowFill(s)
	snowBringSelectorsForward()
	snowCalculate(false)
	snowSetTab(0)
}
func snowUpdateStatus() {
	if project.Site.MountainID != "" {
		set(snowMountainState, "Горный район: ДА")
	} else {
		set(snowMountainState, "Горный район: НЕТ")
	}
	zone, sg, over := project.SnowInputs()
	name := "Район не указан. Выберите I–VIII и проверьте Sg."
	if sg > 0 {
		src := "характеристик местности"
		if over {
			src = "локального выбора"
		}
		unit := "кПа"
		if snowSel(snowUnits) == 1 {
			unit = "кгс/м²"
		}
		name = fmt.Sprintf("Sg = %s %s, район %s · источник: %s", snowGroundUnitText(sg, snowDraft.Units), unit, zone, src)
	}
	if project.SnowCalc != nil && project.SnowCalc.Enabled {
		name += " · ДОБАВЛЕНО В ВЕДОМОСТЬ"
	}
	set(snowStatus, name)
}
func snowRead() (SnowCalc, error) {
	s := snowDraft
	if i := snowSel(snowShape); i >= 0 && i < len(snowKindKeys) {
		s.Shape = snowKindKeys[i]
	}
	if i := snowSel(snowRegion); i >= 0 && i < len(snowOptions) {
		s.Region = snowOptions[i]
	}
	var err error
	fields := []struct {
		id   int
		v    *float64
		name string
	}{
		{snowGamma, &s.Gamma, "γf"},
		{snowRoofHeight, &s.RoofHeight, "H конька левой кровли"},
		{snowSpan, &s.Span, "L левой кровли"}, {snowWidth, &s.Width, "B левой кровли"},
	}
	for _, f := range fields {
		// У круглого покрытия Б.15а нет отдельной длины B.
		// Не считывать и не проверять скрытое поле прямоугольной кровли.
		if s.Shape == "sphere_cone" && f.id == snowWidth {
			continue
		}
		if *f.v, err = snowNumber(f.id); err != nil {
			return s, fmt.Errorf("%s: %w", f.name, err)
		}
	}
	if s.Ground, err = snowReadGroundKpa(); err != nil {
		return s, fmt.Errorf("Sg: %w", err)
	}
	if s.RoofHeight <= 0 {
		return s, fmt.Errorf("H конька левой кровли должна быть больше нуля")
	}
	if s.Shape == "sphere_cone" {
		s.Width = s.Span
	}
	if s.Span <= 0 || s.Width <= 0 {
		return s, fmt.Errorf("Размеры L и B левой кровли должны быть положительными")
	}
	if s.Shape == "multi_gable" || s.Shape == "multi_vault" || s.Shape == "shed" {
		c, e := snowNumber(snowCount)
		if e != nil || c != math.Floor(c) {
			return s, fmt.Errorf("Число пролётов должно быть целым")
		}
		s.Count = int(c)
	}
	if j := snowSel(snowTerrain); j >= 0 && j < 3 {
		s.Terrain = []string{"А", "В", "С"}[j]
	}
	s.SlopeMode = "angle"
	s.Angle = 0
	s.Rise = 0
	if snowSel(snowSlopeMode) == 1 || s.Shape == "vault" || s.Shape == "dome" || s.Shape == "multi_vault" {
		s.SlopeMode = "height"
	}
	v, e := snowNumber(snowSlopeValue)
	if e != nil {
		return s, fmt.Errorf("Уклон/подъём левой кровли: %w", e)
	}
	if s.SlopeMode == "height" {
		s.Rise = v
	} else {
		s.Angle = v
	}
	if s.Shape == "sphere_cone" {
		s.SkylightWidth, e = snowNumber(snowSkylightA)
		if e != nil {
			return s, fmt.Errorf("Радиус конуса r: %w", e)
		}
		s.H, e = snowNumber(snowStep)
		if e != nil {
			return s, fmt.Errorf("Подъём f от стенки до конька: %w", e)
		}
		s.SphereStrictFormula = snowIsChecked(snowSphereStrict)
		s.BetaStep, e = snowNumber(snowSphereBetaStep)
		if e != nil || s.BetaStep <= 0 || s.BetaStep > 180 {
			return s, fmt.Errorf("Шаг β должен быть больше 0 и не превышать 180°")
		}
		if s.SlopeMode == "height" && s.SkylightWidth > 0 {
			s.Angle = math.Atan2(s.Rise, s.SkylightWidth) * 180 / math.Pi
		}
	}
	if s.Shape == "parapet" {
		s.H, e = snowNumber(snowStep)
		if e != nil {
			return s, fmt.Errorf("Высота парапета: %w", e)
		}
	}
	if s.Shape == "skylight" {
		s.SkylightWidth, e = snowNumber(snowSkylightA)
		if e != nil {
			return s, fmt.Errorf("Ширина фонаря: %w", e)
		}
		s.H, e = snowNumber(snowStep)
		if e != nil {
			return s, fmt.Errorf("Высота фонаря hf: %w", e)
		}
	}
	if s.Shape == "cone" || s.Shape == "dome" || s.Shape == "sphere_cone" {
		s.RightRoof = nil
	}
	s.DomeObstacles = snowIsChecked(snowDomeObstacles)
	s.Exposed = !snowIsChecked(snowExposed)
	s.WarmRoof = snowIsChecked(snowWarm)
	s.Units = "kpa"
	if snowSel(snowUnits) == 1 {
		s.Units = "kgf"
	}
	s.RidgeEquipment = s.Shape == "gable" && (snowSel(snowRoofLShape) <= 0) && snowIsChecked(snowRidge)
	s.RoofSensitive = false // automatic optional variants; no user checkbox
	s.JanuarySet = strings.TrimSpace(get(snowJanuary)) != ""
	if s.JanuarySet {
		if s.JanuaryTemp, err = snowNumber(snowJanuary); err != nil {
			return s, fmt.Errorf("Температура января: %w", err)
		}
	}
	s.WinterWindSet = strings.TrimSpace(get(snowWinterWind)) != ""
	if s.WinterWindSet {
		if s.WinterWind, err = snowNumber(snowWinterWind); err != nil {
			return s, fmt.Errorf("Скорость ветра: %w", err)
		}
	}
	if s.JanuarySet != snowDraft.JanuarySet || s.JanuarySet && s.JanuaryTemp != snowDraft.JanuaryTemp || s.WinterWindSet != snowDraft.WinterWindSet || s.WinterWindSet && s.WinterWind != snowDraft.WinterWind {
		s.ClimateOrigin = "manual"
		s.ClimateReferencePlace = ""
		s.ClimateReferenceRegion = ""
	}
	if snowSel(snowRoofLShape) > 0 && (s.Shape == "flat" || s.Shape == "mono" || s.Shape == "gable" || s.Shape == "vault" || s.Shape == "skylight" || s.Shape == "step") {
		r, e := snowReadRoofPart(snowRoofLShape, snowRoofLSpan, snowRoofLMode, snowRoofLValue, snowRoofLLevel)
		if e != nil {
			return s, fmt.Errorf("Правая кровля: %w", e)
		}
		if r.Width, e = snowNumber(snowUpper); e != nil {
			return s, fmt.Errorf("B правой кровли: %w", e)
		}
		if r.Width <= 0 {
			return s, fmt.Errorf("B правой кровли должно быть больше нуля")
		}
		if s.Shape == "step" {
			upper := SnowRoofPart{Shape: "flat", Span: s.Span, RoofHeight: s.RoofHeight, EdgeLevel: s.RoofHeight, Width: s.Width}
			s.UpperRoof = upper
			s.LowerRoof = r
			if upper.EdgeLevel <= r.EdgeLevel {
				return s, fmt.Errorf("При схеме перепада верхняя кровля должна быть выше нижней")
			}
			s.H = upper.EdgeLevel - r.EdgeLevel
			s.Upper = upper.Span
			s.Lower = r.Span
			s.LowerShape = r.Shape
			s.RightRoof = nil
			s.RoofHeight = upper.RoofHeight
			s.Span = upper.Span + r.Span
		} else {
			s.RightRoof = &r
		}
	} else if s.Shape == "step" {
		return s, fmt.Errorf("Для перепада высот выберите тип правой (нижней) кровли")
	} else {
		s.RightRoof = nil
	}
	if s.Shape == "flat" {
		s.Angle = 0
		s.Rise = 0
	}
	return s, nil
}

// The load engine stays in kPa. User interface only changes presentation.
func snowUIUnits() (float64, string) {
	if snowSel(snowUnits) == 1 {
		return 101.9716212978, "кгс/м²"
	}
	return 1, "кПа"
}
func snowUILoad(v float64) string {
	k, _ := snowUIUnits()
	if k > 1 {
		return decimalInput(v*k, 1)
	}
	return decimalInput(v*k, 3)
}
func snowRefreshCases() {
	previous := int(send(snowCasesList, LB_GETCURSEL, 0, 0))
	if previous == -1 {
		previous = snowShownView
	}
	send(snowCasesList, LB_RESETCONTENT, 0, 0)
	for i, v := range snowPresentations(snowDraft) {
		_, unit := snowUIUnits()
		name := fmt.Sprintf("%s   ·   μmax = %s   ·   Sр = %s %s", v.Title, decimalInput(v.MaxMu(), 3), snowUILoad(snowDraft.QDesign(v.MaxMu())), unit)
		pSend.Call(controls[snowCasesList], LB_ADDSTRING, 0, uintptr(unsafe.Pointer(ptr(name))))
		_ = i
	}
	if len(snowDraft.Cases) > 0 {
		if previous < 0 || previous >= len(snowPresentations(snowDraft)) {
			previous = 0
		}
		snowDrawCase, snowShownView = previous, previous
		send(snowCasesList, LB_SETCURSEL, uintptr(previous), 0)
	}
	snowShowCase()
}
func snowShowCase() {
	views := snowPresentations(snowDraft)
	if len(views) == 0 {
		set(snowDetail, "Выберите исходные данные и нажмите «Рассчитать».")
	} else {
		i := int(send(snowCasesList, LB_GETCURSEL, 0, 0))
		if i >= 0 && i < len(views) {
			snowShownView = i
		}
		if snowShownView >= len(views) {
			snowShownView = 0
		}
		v := views[snowShownView]
		note := v.Note
		if note != "" {
			note += "\r\n"
		}
		for j, c := range v.Cases {
			zone := ""
			if j < len(v.Zones) {
				zone = "Зона " + v.Zones[j] + ": "
			}
			m := snowDraft.MaxMu(c)
			note += fmt.Sprintf("%sμmax=%s; Sн=%s; Sр=%s\r\n", zone, decimalInput(m, 3), snowUILoad(snowDraft.Q(m)), snowUILoad(snowDraft.QDesign(m)))
		}
		note += fmt.Sprintf("ce=%s; ct=%s; γf=%s.", decimalInput(snowDraft.CE, 3), decimalInput(snowDraft.CT, 3), decimalInput(snowDraft.Gamma, 3))
		set(snowDetail, note)
	}
	if controls[snowDiagram] != 0 {
		invalidateSnow.Call(controls[snowDiagram], 0, 1)
	}
}
func snowCalculate(showError bool) bool {
	s, e := snowRead()
	if e == nil {
		s, e = SnowCompute(s)
	}
	if e != nil {
		if showError {
			message(e.Error(), true)
		}
		return false
	}
	snowDraft = s
	if s.Shape == "sphere_cone" {
		set(snowSpan, decimalInput(s.Span, 3))
		set(snowSphereAlpha1, decimalInput(sphereConeTangentAngle(s), 2))
	}
	snowRefreshCases()
	if showError {
		snowSetTab(1)
	}
	if showError {
		project.initReportBlocks(tempSections.Sections)
		snowCurrentBlockID = project.putSnowCalc(snowCurrentBlockID, s)
		projectHomeOverview()
	}
	return true
}
func snowAction(id int) bool {
	switch id {
	case snowTabInputs:
		snowSetTab(0)
		return true
	case snowTabResults:
		snowSetTab(1)
		return true
	case snowWarm, snowDomeObstacles, snowExposed, snowSphereStrict:
		return true
	case snowGammaReset:
		set(snowGamma, "1,4")
		return true
	case snowCalcButton:
		snowCalculate(true)
		return true
	case snowMountainButton:
		if draft, er := snowRead(); er == nil {
			snowDraft = draft
		}
		siteReturnPage = "snow"
		snowBeforeMountainSite = project.Site
		snowReturnFromMountain = true
		switchPage("site")
		return true
	case snowRestore:
		if project.Site.SnowZone == "" || project.Site.EffectiveSnow() <= 0 {
			if draft, er := snowRead(); er == nil {
				snowDraft = draft
			}
			siteReturnPage = "snow"
			switchPage("site")
			return true
		}
		project.Snow = ClimateOverride{}
		zone, sg, _ := project.SnowInputs()
		snowDraft.GroundLocal = false
		snowDraft.Region = zone
		snowDraft.Ground = sg
		snowFill(snowDraft)
		snowCalculate(false)
		refreshClimateUI()
		return true
	case snowApply:
		s, e := snowRead()
		if e != nil {
			message(e.Error(), true)
			return true
		}
		if isKChoice(s.Region) {
			if !specialSnowSite(project.Site) {
				message("Для приложения К сначала выберите соответствующий населённый пункт в характеристиках местности", true)
				return true
			}
			s.Region = project.Site.SnowZone
		}
		if s.Region != snowOptionMountain {
			if _, ok := snowForZone(s.Region); !ok {
				message("Выберите снеговой район I–VIII", true)
				return true
			}
		}
		if s.Ground <= 0 {
			message("Укажите нормативный вес снега Sg в выбранных единицах", true)
			return true
		}
		project.Snow = ClimateOverride{Enabled: true, Zone: s.Region, Value: s.Ground, Mode: "Ручной выбор в снеговом модуле"}
		s.GroundLocal = true
		snowDraft = s
		snowUpdateStatus()
		refreshClimateUI()
		return true
	case snowSave:
		if !snowCalculate(true) {
			return true
		}
		name, ok := askReportBlockName(project.suggestedCurrentName("snow", snowCurrentBlockID))
		if !ok {
			return true
		}
		project.renameBlock(snowCurrentBlockID, name)
		snowDraft = defaultSnowCalc()
		project.SnowCalc = nil
		snowCurrentBlockID = 0
		editingReportBlockID = 0
		snowFill(snowDraft)
		snowUpdateStatus()
		snowSetTab(0)
		projectHomeOverview()
		return true
	case snowRemove:
		// Only clear the working draft; never delete a saved block here.
		snowCurrentBlockID = 0
		editingReportBlockID = 0
		project.SnowCalc = nil
		snowDraft = defaultSnowCalc()
		snowFill(snowDraft)
		snowUpdateStatus()
		snowSetTab(0)
		projectHomeOverview()
		return true

	case snowExportWord, snowExportExcel:
		if !snowCalculate(true) {
			return true
		}
		ext, name := "docx", "Снеговая нагрузка.docx"
		if id == snowExportExcel {
			ext, name = "xlsx", "Снеговая нагрузка.xlsx"
		}
		path := chooseSave(ext, strings.ToUpper(ext), name, false)
		if path == "" {
			return true
		}
		saveAndNotify(path, func(file string) error {
			if id == snowExportWord {
				return ExportSnowWord(snowDraft, file)
			}
			return ExportSnowExcel(snowDraft, file)
		})
		return true
	}
	return false
}

// Convert an already entered pitch to a rise and back without changing the roof.
func snowConvertSlope(modeCtrl, valueCtrl int, oldMode, newMode int, run float64) {
	if oldMode == newMode || run <= 0 {
		return
	}
	v, e := snowNumber(valueCtrl)
	if e != nil || (v < 0 && snowSel(snowShape) != 1) {
		return
	}
	if oldMode == 0 && newMode == 1 {
		if math.Abs(v) >= 90 {
			return
		}
		v = run * math.Tan(v*math.Pi/180)
	} else if oldMode == 1 && newMode == 0 {
		v = math.Atan2(v, run) * 180 / math.Pi
	}
	set(valueCtrl, decimalInput(v, 3))
}
func snowSlopeRun(shape string, span, lantern float64) float64 {
	switch shape {
	case "gable", "cone", "multi_gable", "dome":
		return span / 2
	case "sphere_cone":
		return lantern
	case "skylight":
		return (span - lantern) / 2
	default:
		return span
	}
}

// One warning per continuous underground-roof condition; never repeat on Calculate.
func snowSphereRefreshGeometry(lastEdited int) {
	H, e1 := snowNumber(snowRoofHeight)
	d, e2 := snowNumber(snowSpan)
	r, e3 := snowNumber(snowSkylightA)
	f, e4 := snowNumber(snowStep)
	v, e5 := snowNumber(snowSlopeValue)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil || e5 != nil || H <= 0 || d <= 0 || r <= 0 {
		return
	}
	s := defaultSnowCalc()
	s.Shape = "sphere_cone"
	s.RoofHeight = H
	s.Span = d
	s.SkylightWidth = r
	s.H = f
	if snowSel(snowSlopeMode) == 1 {
		s.Rise = v
		s.Angle = math.Atan2(v, r) * 180 / math.Pi
	} else {
		s.Angle = v
	}
	if s.Angle <= 0 || s.Angle >= 90 {
		return
	}
	if lastEdited == snowSphereAlpha1 {
		a1, er := snowNumber(snowSphereAlpha1)
		if er != nil || a1 <= 0 || a1 >= 180 {
			return
		}
		s.H = sphereConeRiseForAngle1(s, a1)
		set(snowStep, decimalInput(s.H, 3))
	}
	old, newD, changed := sphereConeExpandDiameter(&s)
	if changed {
		set(snowSpan, decimalInput(newD, 3))
		if old <= 2*r {
			message(fmt.Sprintf("Внимание! Для сферы с конусом требуется D > 2r. Диаметр D автоматически увеличен с %s до %s м.", decimalInput(old, 3), decimalInput(newD, 3)), false)
		} else {
			message(fmt.Sprintf("Внимание! При заданных размерах касательная сферической части имела бы обратный уклон (α₁ > 90°). Диаметр D автоматически увеличен с %s до %s м.", decimalInput(old, 3), decimalInput(newD, 3)), false)
		}
	}
	set(snowSphereAlpha1, decimalInput(sphereConeTangentAngle(s), 2))
	if sphereConeTangentAngle(s) <= s.Angle {
		if !snowSphereInvalidAngleWarned {
			snowSphereInvalidAngleWarned = true
			message("Внимание! Для нормативной схемы Б.15а должно соблюдаться α₁ > α. При заданных размерах это условие не выполняется. Расчёт выполним условно; проверьте геометрию и применимость схемы СП.", false)
		}
	} else {
		snowSphereInvalidAngleWarned = false
	}
	if s.RoofHeight-s.H < 0 {
		if !snowSphereBelowGroundWarned {
			snowSphereBelowGroundWarned = true
			message("Внимание! При такой геометрии отметка низа кровли оказалась ниже отметки земли. Посчитаем так, но убедитесь, что у вас действительно заглублённое в землю сооружение.", false)
		}
	} else {
		snowSphereBelowGroundWarned = false
	}
}

func snowNotification(id, code int) bool {
	if activePage != "snow" {
		return false
	}
	if (id == snowShape || id == snowRoofLShape) && code == CBN_SELCHANGE {
		if id == snowShape {
			j := snowSel(snowShape)
			if j >= 0 && j < len(snowKindKeys) {
				sh := snowKindKeys[j]
				if sh == "vault" || sh == "dome" || sh == "multi_vault" {
					set(snowSlopeValue, "1,5")
					snowSetSel(snowSlopeMode, 1)
					snowPrevSlopeMode = 1
				}
				if sh == "sphere_cone" {
					set(snowRoofHeight, "12")
					set(snowSpan, "18")
					set(snowSkylightA, "6")
					set(snowStep, "6")
					set(snowSlopeValue, "18")
					set(snowSphereBetaStep, "15")
					snowSetSel(snowSlopeMode, 0)
					snowPrevSlopeMode = 0
					snowSphereRefreshGeometry(0)
				}
				if sh == "skylight" {
					if v, _ := snowNumber(snowSkylightA); v <= 0 {
						set(snowSkylightA, "6")
					}
					if v, _ := snowNumber(snowStep); v <= 0 {
						set(snowStep, "2")
					}
				}
			}
		} else {
			j := snowSel(snowRoofLShape)
			if j == 4 {
				set(snowRoofLValue, "1,5")
				snowSetSel(snowRoofLMode, 1)
				snowPrevRightSlopeMode = 1
			}
			if j == 5 {
				if v, _ := snowNumber(snowRightA); v <= 0 {
					set(snowRightA, "6")
				}
				if v, _ := snowNumber(snowRightHF); v <= 0 {
					set(snowRightHF, "2")
				}
			}
		}
		snowLayout()
		return true
	}
	if id == snowSlopeMode && code == CBN_SELCHANGE {
		newMode := snowSel(snowSlopeMode)
		shape := snowDraft.Shape
		if i := snowSel(snowShape); i >= 0 && i < len(snowKindKeys) {
			shape = snowKindKeys[i]
		}
		span, _ := snowNumber(snowSpan)
		a, _ := snowNumber(snowSkylightA)
		snowConvertSlope(snowSlopeMode, snowSlopeValue, snowPrevSlopeMode, newMode, snowSlopeRun(shape, span, a))
		if shape == "sphere_cone" {
			snowSphereRefreshGeometry(snowSlopeValue)
		}
		snowPrevSlopeMode = newMode
		snowLayout()
		return true
	}
	if id == snowRoofLMode && code == CBN_SELCHANGE {
		newMode := snowSel(snowRoofLMode)
		shapes := []string{"none", "flat", "mono", "gable", "vault", "skylight"}
		j := snowSel(snowRoofLShape)
		shape := "flat"
		if j >= 0 && j < len(shapes) {
			shape = shapes[j]
		}
		span, _ := snowNumber(snowRoofLSpan)
		a, _ := snowNumber(snowRightA)
		snowConvertSlope(snowRoofLMode, snowRoofLValue, snowPrevRightSlopeMode, newMode, snowSlopeRun(shape, span, a))
		snowPrevRightSlopeMode = newMode
		snowLayout()
		return true
	}
	if code == EN_KILLFOCUS && (id == snowRoofHeight || id == snowSpan || id == snowSkylightA || id == snowStep || id == snowSphereAlpha1 || id == snowSlopeValue) {
		if snowSel(snowShape) >= 0 && snowSel(snowShape) < len(snowKindKeys) && snowKindKeys[snowSel(snowShape)] == "sphere_cone" {
			snowSphereRefreshGeometry(id)
		}
		return false
	}
	if id == snowUnits && code == CBN_SELCHANGE {
		// Read using PREVIOUS display units; the combo has already switched.
		sg, err := snowReadGroundKpa()
		if err != nil {
			message("Не удалось преобразовать Sg: "+err.Error(), true)
			return true
		}
		unit := "kpa"
		if snowSel(snowUnits) == 1 {
			unit = "kgf"
		}
		snowDraft.Units = unit
		snowSetGroundKpa(sg, unit)
		snowGroundUpdateUnitLabel(unit)
		snowUpdateStatus()
		snowRefreshCases()
		return true
	}
	if id == snowCasesList && code == LBN_SELCHANGE {
		snowShowCase()
		return true
	}
	if id == snowRegion && code == CBN_SELCHANGE {
		i := snowSel(snowRegion)
		if i >= 0 && i < len(snowOptions) {
			if v, ok := snowValueForChoice(snowOptions[i], project.Site); ok {
				snowSetGroundKpa(v, snowDraft.Units)
			} else {
				snowSetGroundKpa(0, snowDraft.Units)
				set(snowGround, "")
			}
		}
		return true
	}
	return false
}

// Win32 owner-draw uses the native window DC, without a browser or embedded WebView.
type snowRect struct{ Left, Top, Right, Bottom int32 }
type snowDrawStruct struct {
	CtlType, CtlID                uint32
	ItemID, ItemAction, ItemState uint32
	HWNDItem, HDC                 uintptr
	Rect                          snowRect
	ItemData                      uintptr
}

// Screen-only engineering warning. It must not enter DOCX/XLSX or saved cases.
func snowPaintSphereWarning(wp, lp uintptr) bool {
	if wp != snowSphereWarning || lp == 0 {
		return false
	}
	d := (*snowDrawStruct)(unsafe.Pointer(lp))
	dc := d.HDC
	brush, _, _ := g32.NewProc("CreateSolidBrush").Call(0x00FFFFFF)
	u32.NewProc("FillRect").Call(dc, uintptr(unsafe.Pointer(&d.Rect)), brush)
	g32.NewProc("DeleteObject").Call(brush)
	g32.NewProc("SetBkMode").Call(dc, 1)
	g32.NewProc("SetTextColor").Call(dc, 0x0000C0) // COLORREF: dark red
	if headingFont != 0 {
		prev, _, _ := g32.NewProc("SelectObject").Call(dc, headingFont)
		defer g32.NewProc("SelectObject").Call(dc, prev)
	}
	lines := []string{
		"ВНИМАНИЕ! В СП неоднозначна формула зоны 5:",
		"возможно, перепутаны углы α₁ и α₂.",
		"По умолчанию расчёт по геометрии (α₂);",
		"галочка включает буквальную формулу (α₁).",
	}
	for i, line := range lines {
		utf := syscall.StringToUTF16(line)
		g32.NewProc("TextOutW").Call(dc, uintptr(d.Rect.Left+3), uintptr(d.Rect.Top+2+19*int32(i)), uintptr(unsafe.Pointer(&utf[0])), uintptr(len(utf)-1))
	}
	return true
}

func snowPaintDiagram(wp, lp uintptr) bool {
	if wp != snowDiagram || lp == 0 {
		return false
	}
	d := (*snowDrawStruct)(unsafe.Pointer(lp))
	dc := d.HDC
	gp := g32.NewProc("CreatePen")
	goSel := g32.NewProc("SelectObject")
	del := g32.NewProc("DeleteObject")
	move := g32.NewProc("MoveToEx")
	line := g32.NewProc("LineTo")
	txt := g32.NewProc("TextOutW")
	bkg := g32.NewProc("SetBkMode")
	col := g32.NewProc("SetTextColor")
	fill := u32.NewProc("FillRect")
	brush := g32.NewProc("CreateSolidBrush")
	b, _, _ := brush.Call(0x00ffffff)
	fill.Call(dc, uintptr(unsafe.Pointer(&d.Rect)), b)
	del.Call(b)
	// Keep the entire calculated chart inside the separate RIGHT panel,
	// even after vertical resizing; native fonts remain readable.
	plotTop := int(d.Rect.Top)
	plotHeight := int(d.Rect.Bottom - d.Rect.Top)
	verticalScale := math.Min(1, float64(plotHeight)/390)
	fitY := func(y int) int { return plotTop + int(float64(y-plotTop)*verticalScale) }
	drawLine := func(x1, y1, x2, y2 int, c uint32) {
		y1, y2 = fitY(y1), fitY(y2)
		p, _, _ := gp.Call(0, 2, uintptr(c))
		old, _, _ := goSel.Call(dc, p)
		move.Call(dc, uintptr(x1), uintptr(y1), 0)
		line.Call(dc, uintptr(x2), uintptr(y2))
		goSel.Call(dc, old)
		del.Call(p)
	}
	drawText := func(x, y int, t string, c uint32) {
		y = fitY(y)
		u := syscall.StringToUTF16(t)
		if len(u) == 0 {
			return
		}
		col.Call(dc, uintptr(c))
		bkg.Call(dc, 1)
		txt.Call(dc, uintptr(x), uintptr(y), uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1))
	}
	ox := int(d.Rect.Left)
	oy := int(d.Rect.Top)
	w := int(d.Rect.Right - d.Rect.Left)
	if len(snowDraft.Cases) == 0 {
		drawText(ox+20, oy+30, "Введите данные покрытия и нажмите «Рассчитать»", 0x444444)
		return true
	}
	views := snowPresentations(snowDraft)
	if len(views) == 0 {
		return true
	}
	if snowShownView < 0 || snowShownView >= len(views) {
		snowShownView = 0
	}
	view := views[snowShownView]
	if len(view.Zones) > 0 {
		// Each A/C/B epure has the same numbered ordinates, vertical load
		// lines and characteristic coordinates as the ordinary roof charts.
		drawText(ox+14, oy+10, view.Title, 0x222222)
		x1, x2 := ox+132, ox+w-28
		maxMu := view.MaxMu()
		if maxMu <= 0 {
			maxMu = 1
		}
		n := len(view.Cases)
		rowH := (plotHeight - 54) / n
		if rowH < 83 {
			rowH = 83
		}
		length := snowDraft.Span
		if length <= 0 {
			length = 1
		}
		xp := func(x float64) int { return x1 + int(x/length*float64(x2-x1)) }
		for j, c := range view.Cases {
			base := oy + 46 + (j+1)*rowH - 21
			label := ""
			if j < len(view.Zones) {
				label = view.Zones[j]
			}
			drawText(ox+12, base-34, "Зона "+label, 0x222222)
			drawText(ox+12, base-16, "μmax="+decimalInput(snowDraft.MaxMu(c), 2), 0x333333)
			drawLine(x1, base, x2, base, 0x555555)
			// Mark both the endpoints of a step and the ordinary load
			// ordinates; the older charts do not consist of bare outlines.
			lastTextRight := x1 - 10
			for i, seg := range c.Intervals {
				xA, xB := xp(seg.X1), xp(seg.X2)
				v := seg.Mu
				y := base - int(46*v/maxMu)
				drawLine(xA, y, xB, y, 0xAA4422)
				if v > 0 {
					step := 31
					for x := xA; x <= xB; x += step {
						drawLine(x, y, x, base, 0xD5A78A)
					}
				}
				drawLine(xA, y, xA, base, 0xAA4422)
				if i == len(c.Intervals)-1 {
					drawLine(xB, y, xB, base, 0xAA4422)
				}
				if xB-xA >= 42 && xA >= lastTextRight+8 {
					caption := snowUILoad(snowDraft.Q(v)) + " (" + snowUILoad(snowDraft.QDesign(v)) + ")"
					xx := xA + 5
					if xx+len([]rune(caption))*7 > xB-3 {
						caption = decimalInput(v, 2)
					}
					drawText(xx, y-17, caption, 0x333333)
					lastTextRight = xx + len([]rune(caption))*7
				}
				if i > 0 && math.Abs(c.Intervals[i-1].Mu-v) > 1e-9 {
					drawText(xA-16, base+3, decimalInput(seg.X1, 1), 0x444444)
				}
			}
			drawText(x1, base+5, "0 м", 0x444444)
			drawText(x2-60, base+5, decimalInput(length, 1)+" м", 0x444444)
		}
		return true
	}

	c := view.Cases[0]
	if snowDraft.Shape == "sphere_cone" && len(c.Cross) > 0 {
		// Two characteristic diametral cuts of the same spatial load case.
		drawText(ox+15, oy+15, "Пространственное загружение Б.15а, зоны 1–6", 0x222222)
		plot := func(items []SnowInterval, top int, name string) {
			if len(items) == 0 {
				return
			}
			maxMu := 0.
			for _, v := range items {
				if v.Mu > maxMu {
					maxMu = v.Mu
				}
			}
			if maxMu <= 0 {
				maxMu = 1
			}
			x1, x2 := ox+34, ox+w-25
			base := oy + top + 113
			min, max := items[0].X1, items[len(items)-1].X2
			px := func(x float64) int { return x1 + int((x-min)/(max-min)*float64(x2-x1)) }
			py := func(mu float64) int { return base - int(mu/maxMu*80) }
			drawText(x1, oy+top+5, name, 0x222222)
			drawLine(x1, base, x2, base, 0x555555)
			for i, v := range items {
				a, b := px(v.X1), px(v.X2)
				if v.Mu > 0 && i%4 == 0 {
					drawLine((a+b)/2, py(v.Mu), (a+b)/2, base, 0xC9B2A4)
				}
				if i > 0 && math.Abs(items[i-1].Mu-v.Mu) > .12 {
					drawLine(a, py(items[i-1].Mu), a, py(v.Mu), 0xC07028)
				}
				drawLine(a, py(v.Mu), b, py(v.Mu), 0xC07028)
			}
			// Characteristic boundaries of two spherical shoulders and the cone.
			// The chain contains both consecutive dimensions and their coordinates.
			D, r := snowDraft.Span, snowDraft.SkylightWidth
			points := []float64{0, D/2 - r, D / 2, D/2 + r, D}
			for i, value := range points {
				x := px(value)
				drawLine(x, base+6, x, base+23, 0x555555)
				label := decimalInput(value, 2)
				if i == len(points)-1 {
					label += " м"
				}
				textX := x - len([]rune(label))*3
				if i == 0 {
					textX = x
				}
				if i == len(points)-1 {
					textX = x - len([]rune(label))*7
				}
				drawText(textX, base+24, label, 0x555555)
				if i > 0 && value-points[i-1] > 1e-5 {
					mid := px((value + points[i-1]) / 2)
					length := decimalInput(value-points[i-1], 2) + " м"
					drawText(mid-len([]rune(length))*4, base+44, length, 0x555555)
				}
			}
			drawText(x1+3, base-92, "μmax="+decimalInput(maxMu, 3)+"; Sн,max="+snowUILoad(snowDraft.Q(maxMu)), 0x333333)
		}
		plot(c.Intervals, 29, "Сечение по зонам 6–1")
		plot(c.Cross, 213, "Сечение по зонам 2–4–4–2")
		return true
	}
	minX, maxX, maxMu := 0., 0., 0.
	for _, v := range c.Intervals {
		minX = math.Min(minX, v.X1)
		maxX = math.Max(maxX, v.X2)
		maxMu = math.Max(maxMu, v.Mu)
	}
	L := maxX - minX
	if L <= 0 {
		return true
	}
	xLeft := ox + 35
	xRight := ox + w - 25
	plotW := xRight - xLeft
	pos := func(x float64) int { return xLeft + int((x-minX)/L*float64(plotW)) }
	roofY := oy + 104
	drawText(xLeft, oy+12, snowShapeTitle(snowDraft.Shape), 0x222222)
	drawLine(xLeft, roofY, xRight, roofY, 0x888888)
	if snowDraft.RightRoof != nil && snowDraft.RightRoof.Shape != "none" {
		leftLength := snowDraft.Span
		drawRoof := func(shape string, x1, x2 int) {
			mid := (x1 + x2) / 2
			switch shape {
			case "gable":
				drawLine(x1, roofY, mid, roofY-30, 0x222222)
				drawLine(mid, roofY-30, x2, roofY, 0x222222)
			case "mono":
				drawLine(x1, roofY, x2, roofY-30, 0x222222)
			case "vault":
				for z := 0; z < 30; z++ {
					x := x1 + (x2-x1)*z/30
					xx := x1 + (x2-x1)*(z+1)/30
					y := roofY - int(31*math.Sin(math.Pi*float64(z)/30))
					yy := roofY - int(31*math.Sin(math.Pi*float64(z+1)/30))
					drawLine(x, y, xx, yy, 0x222222)
				}
			case "skylight":
				a := float64(x2-x1) * .32 / 2
				l, r := int(float64(mid)-a), int(float64(mid)+a)
				drawLine(x1, roofY, l, roofY-14, 0x222222)
				drawLine(l, roofY-14, l, roofY-43, 0x222222)
				drawLine(l, roofY-43, mid, roofY-55, 0x222222)
				drawLine(mid, roofY-55, r, roofY-43, 0x222222)
				drawLine(r, roofY-43, r, roofY-14, 0x222222)
				drawLine(r, roofY-14, x2, roofY, 0x222222)
			}
		}
		sep := pos(leftLength)
		drawRoof(snowDraft.Shape, xLeft, sep)
		drawRoof(snowDraft.RightRoof.Shape, sep, xRight)
	} else {
		switch snowDraft.Shape {
		case "gable", "multi_gable":
			n := 1
			if snowDraft.Shape == "multi_gable" {
				n = snowDraft.Count
			}
			for i := 0; i < n; i++ {
				x1 := pos(float64(i) * L / float64(n))
				x2 := pos(float64(i+1) * L / float64(n))
				xm := (x1 + x2) / 2
				drawLine(x1, roofY, xm, roofY-37, 0x222222)
				drawLine(xm, roofY-37, x2, roofY, 0x222222)
			}
		case "mono":
			if snowDraft.Angle < 0 || snowDraft.Rise < 0 {
				drawLine(xLeft, roofY-35, xRight, roofY, 0x222222)
			} else {
				drawLine(xLeft, roofY, xRight, roofY-35, 0x222222)
			}
		case "vault", "multi_vault", "dome":
			n := 1
			if snowDraft.Shape == "multi_vault" {
				n = snowDraft.Count
				if n < 2 {
					n = 2
				}
			}
			for bay := 0; bay < n; bay++ {
				for j := 0; j < 40; j++ {
					u, v := float64(j)/40, float64(j+1)/40
					x1 := xLeft + int((float64(bay)+u)*float64(plotW)/float64(n))
					x2 := xLeft + int((float64(bay)+v)*float64(plotW)/float64(n))
					y1 := roofY - int(math.Sin(u*math.Pi)*40)
					y2 := roofY - int(math.Sin(v*math.Pi)*40)
					drawLine(x1, y1, x2, y2, 0x222222)
				}
			}
		case "parapet":
			drawLine(xLeft, roofY-42, xLeft, roofY, 0x222222)
			drawLine(xLeft, roofY, xRight, roofY, 0x222222)
		case "shed":
			n := snowDraft.Count
			if n < 2 {
				n = 2
			}
			for j := 0; j < n; j++ {
				a := pos(float64(j) * snowDraft.Span)
				b := pos(float64(j+1) * snowDraft.Span)
				drawLine(a, roofY, a+int(float64(b-a)*.95), roofY-35, 0x222222)
				drawLine(a+int(float64(b-a)*.95), roofY-35, b, roofY, 0x222222)
			}
		case "wall", "step":
			st := pos(0)
			drawLine(st, roofY-48, st, roofY, 0x222222)
			if snowDraft.Shape == "step" {
				upper, _ := snowDraft.stepRoofs()
				switch upper.Shape {
				case "gable":
					mid := (xLeft + st) / 2
					drawLine(xLeft, roofY-48, mid, roofY-76, 0x222222)
					drawLine(mid, roofY-76, st, roofY-48, 0x222222)
				case "mono":
					drawLine(xLeft, roofY-48, st, roofY-76, 0x222222)
				default:
					drawLine(xLeft, roofY-48, st, roofY-48, 0x222222)
				}
			}
			_, lower := snowDraft.stepRoofs()
			if lower.Shape == "gable" {
				mid := (st + xRight) / 2
				drawLine(st, roofY, mid, roofY-25, 0x222222)
				drawLine(mid, roofY-25, xRight, roofY, 0x222222)
			}
			if lower.Shape == "mono" {
				drawLine(st, roofY, xRight, roofY-25, 0x222222)
			}
			if lower.Shape == "vault" {
				for j := 1; j <= 64; j++ {
					x1 := st + (j-1)*(xRight-st)/64
					x2 := st + j*(xRight-st)/64
					y1 := roofY - int(25*math.Sin(math.Pi*float64(j-1)/64))
					y2 := roofY - int(25*math.Sin(math.Pi*float64(j)/64))
					drawLine(x1, y1, x2, y2, 0x222222)
				}
			}
		case "skylight":
			a := snowDraft.SkylightWidth
			left := pos((snowDraft.Span - a) / 2)
			right := pos((snowDraft.Span + a) / 2)
			drawLine(xLeft, roofY, left, roofY-18, 0x222222)
			drawLine(left, roofY-18, left, roofY-52, 0x222222)
			drawLine(left, roofY-52, (left+right)/2, roofY-68, 0x222222)
			drawLine((left+right)/2, roofY-68, right, roofY-52, 0x222222)
			drawLine(right, roofY-52, right, roofY-18, 0x222222)
			drawLine(right, roofY-18, xRight, roofY, 0x222222)
		case "sphere_cone":
			// Cone ridge and rounded spherical shoulders, consistent with B.15a.
			a := xLeft + plotW/6
			b := xRight - plotW/6
			m := (xLeft + xRight) / 2
			for j := 0; j < 16; j++ {
				xx1 := xLeft + j*(a-xLeft)/16
				xx2 := xLeft + (j+1)*(a-xLeft)/16
				y1 := roofY - int(18*math.Sin(float64(j)*math.Pi/32))
				y2 := roofY - int(18*math.Sin(float64(j+1)*math.Pi/32))
				drawLine(xx1, y1, xx2, y2, 0x222222)
				drawLine(xRight-(xx1-xLeft), y1, xRight-(xx2-xLeft), y2, 0x222222)
			}
			drawLine(a, roofY-18, m, roofY-42, 0x222222)
			drawLine(m, roofY-42, b, roofY-18, 0x222222)
		case "cone":
			drawLine(xLeft, roofY, xLeft+plotW/2, roofY-37, 0x222222)
			drawLine(xLeft+plotW/2, roofY-37, xRight, roofY, 0x222222)
		}
	}
	drawText(xLeft, oy+132, "Эпюра μ и вертикальная снеговая нагрузка · "+c.Title, 0x222222)
	zero := oy + 306
	drawLine(xLeft, zero, xRight, zero, 0x555555)
	if maxMu <= 0 {
		maxMu = 1
	}
	// Both B.13 local load and the independent B.8 parapet check are
	// continuous profiles. The conservative numeric intervals are NOT the
	// draw instructions (otherwise the extra check appears as a staircase).
	profile, continuous := snowParapetProfile(snowDraft, c)
	if continuous {
		for i := 1; i < len(profile); i++ {
			a, b := profile[i-1], profile[i]
			drawLine(pos(a.X), zero-int(110*a.Mu/maxMu), pos(b.X), zero-int(110*b.Mu/maxMu), 0xAD6334)
		}
		for j := 0; j <= 16; j++ {
			x := minX + L*float64(j)/16
			v := snowProfileMuAt(profile, x)
			drawLine(pos(x), zero, pos(x), zero-int(110*v/maxMu), 0xE0B89D)
		}
	} else {
		prevX, prevY := 0, 0
		for j := 0; j <= 160; j++ {
			x := minX + L*float64(j)/160
			v := snowMuAt(c, x)
			px, py := pos(x), zero-int(110*v/maxMu)
			if j > 0 {
				drawLine(prevX, prevY, px, py, 0xAD6334)
			}
			if j%8 == 0 {
				drawLine(px, zero, px, py, 0xE0B89D)
			}
			prevX, prevY = px, py
		}
	}
	// Distances and numeric labels accompany the epure, not just the text box.
	drawText(xLeft, zero+8, decimalInput(minX, 2)+" м", 0x444444)
	drawText(xRight-72, zero+8, decimalInput(maxX, 2)+" м", 0x444444)
	if snowDraft.Shape == "step" {
		drawText(pos(0)+4, zero+8, "0 / перепад", 0x444444)
	}
	// Show true computed boundaries, not a guessed centre of the roof.
	for _, edge := range snowEpureStartCoordinates(c) {
		if edge > minX+1e-6 && edge < maxX-1e-6 {
			x := pos(edge)
			drawLine(x, zero, x, zero+6, 0x555555)
			drawText(x-19, zero+9, decimalInput(edge, 2)+" м", 0x444444)
		}
	}
	if len(c.Intervals) <= 10 {
		last := 0
		for last < len(c.Intervals) {
			j := last
			for j+1 < len(c.Intervals) && math.Abs(c.Intervals[j+1].Mu-c.Intervals[last].Mu) < 1e-9 {
				j++
			}
			a, b := pos(c.Intervals[last].X1), pos(c.Intervals[j].X2)
			if b-a > 105 {
				val := c.Intervals[last].Mu
				label := snowUILoad(snowDraft.Q(val)) + " (" + snowUILoad(snowDraft.QDesign(val)) + ")"
				x := (a+b)/2 - 35
				y := zero - int(110*val/maxMu) - 23
				if y < oy+149 {
					y = oy + 149
				}
				drawText(x, y, label, 0x554035)
			}
			last = j + 1
		}
	}
	labelLoad := func(mu float64) string {
		red := "—"
		if snowDraft.ReducedApplicable() {
			red = snowUILoad(snowDraft.QReduced(mu))
		}
		if red == "—" {
			return fmt.Sprintf("%s (%s)", snowUILoad(snowDraft.Q(mu)), snowUILoad(snowDraft.QDesign(mu)))
		}
		return fmt.Sprintf("%s (%s/%s)", snowUILoad(snowDraft.Q(mu)), snowUILoad(snowDraft.QDesign(mu)), red)
	}
	type pnt struct{ x, mu float64 }
	pts := []pnt{}
	if len(c.Intervals) > 0 {
		pts = append(pts, pnt{c.Intervals[0].X1, c.Intervals[0].Mu})
		for k := 0; k < len(c.Intervals); k++ {
			pts = append(pts, pnt{c.Intervals[k].X2, c.Intervals[k].Mu})
			if k+1 < len(c.Intervals) && math.Abs(c.Intervals[k+1].Mu-c.Intervals[k].Mu) > 1e-9 {
				pts = append(pts, pnt{c.Intervals[k+1].X1, c.Intervals[k+1].Mu})
			}
		}
	}
	if len(c.Intervals) > 20 {
		pts = []pnt{{minX, c.Intervals[0].Mu}, {maxX, c.Intervals[len(c.Intervals)-1].Mu}, {(minX + maxX) / 2, snowDraft.MaxMu(c)}}
	}
	if continuous {
		pts = pts[:0]
		for _, p := range profile {
			pts = append(pts, pnt{p.X, p.Mu})
		}
	}
	lastX := -1e9
	for _, pt := range pts {
		if math.Abs(pt.x-lastX) < 0.02*L {
			continue
		}
		px := pos(pt.x)
		py := zero - int(110*pt.mu/maxMu) - 18
		if py < oy+140 {
			py = oy + 140
		}
		drawText(px-28, py, labelLoad(pt.mu), 0x333333)
		lastX = pt.x
	}
	low := "не рассчитывается"
	if snowDraft.ReducedApplicable() {
		low = snowUILoad(snowDraft.QReduced(snowDraft.MaxMu(c)))
	}
	_, unit := snowUIUnits()
	drawText(xRight-116, oy+348, "Нагрузка, "+unit, 0x222222)
	drawText(xLeft, oy+330, fmt.Sprintf("Sн,max = %s %s    Sр,max = %s %s    Sпониж = %s", snowUILoad(snowDraft.Q(snowDraft.MaxMu(c))), unit, snowUILoad(snowDraft.QDesign(snowDraft.MaxMu(c))), unit, low), 0x222222)
	return true
}

func snowTemperatureLabel(s SnowCalc) string {
	if !s.JanuarySet {
		return "не задана"
	}
	return decimalInput(s.JanuaryTemp, 1) + " °C"
}
func snowWindLabel(s SnowCalc) string {
	if !s.WinterWindSet {
		return "не задан"
	}
	return decimalInput(s.WinterWind, 1) + " м/с"
}
func snowExposureLabel(s SnowCalc) string {
	if s.CE < 1 {
		return "снижение учтено"
	}
	if !s.JanuarySet || !s.WinterWindSet {
		return "недостаточно климатических данных"
	}
	return "снижение не применяется"
}
func snowReducedLabel(s SnowCalc) string {
	if !s.ReducedApplicable() {
		return snowJanuaryReductionReason(s)
	}
	return "Пониженная нагрузка: 0,5·μ·Sg (п. 10.11 СП 20.13330.2016)."
}

// Draw a dimensioned technical sketch for the currently selected roof. All
// dimensions refer to the input fields; the drawing is not a scale model.
func snowPaintSketch(wp, lp uintptr) bool {
	if wp != snowSketch || lp == 0 {
		return false
	}
	d := (*snowDrawStruct)(unsafe.Pointer(lp))
	shape := snowDraft.Shape
	if i := snowSel(snowShape); i >= 0 && i < len(snowKindKeys) {
		shape = snowKindKeys[i]
	}
	left, top := int(d.Rect.Left), int(d.Rect.Top)
	w, h := int(d.Rect.Right-d.Rect.Left), int(d.Rect.Bottom-d.Rect.Top)
	white, _, _ := g32.NewProc("CreateSolidBrush").Call(0xffffff)
	u32.NewProc("FillRect").Call(d.HDC, uintptr(unsafe.Pointer(&d.Rect)), white)
	g32.NewProc("DeleteObject").Call(white)
	if j := snowSel(snowRoofLShape); j > 0 && (shape == "flat" || shape == "mono" || shape == "gable" || shape == "vault" || shape == "skylight" || shape == "step") {
		rshapes := []string{"none", "flat", "mono", "gable", "vault", "skylight"}
		if j < len(rshapes) {
			leftshape := shape
			if shape == "step" {
				leftshape = "flat"
			}
			snowPaintRoofAsset(d.HDC, left, top, w/2, h, leftshape, leftshape == "mono" && (snowDraft.Angle < 0 || snowDraft.Rise < 0))
			snowPaintRoofAsset(d.HDC, left+w/2, top, w-w/2, h, rshapes[j], snowDraft.RightRoof != nil && snowDraft.RightRoof.resolvedAngle() < 0)
			return true
		}
	}
	return snowPaintRoofAsset(d.HDC, left, top, w, h, shape, shape == "mono" && (snowDraft.Angle < 0 || snowDraft.Rise < 0))
}

func snowPaintZonePlan(wp, lp uintptr) bool {
	if wp != snowZonePlan || lp == 0 {
		return false
	}
	d := (*snowDrawStruct)(unsafe.Pointer(lp))
	r := d.Rect
	white, _, _ := g32.NewProc("CreateSolidBrush").Call(0xffffff)
	u32.NewProc("FillRect").Call(d.HDC, uintptr(unsafe.Pointer(&r)), white)
	g32.NewProc("DeleteObject").Call(white)
	shape := ""
	if j := snowSel(snowShape); j >= 0 && j < len(snowKindKeys) {
		shape = snowKindKeys[j]
	}
	if shape == "skylight" {
		two := snowSel(snowRoofLShape) == 5
		snowPaintLanternPlan(d.HDC, int(r.Left), int(r.Top), int(r.Right-r.Left), int(r.Bottom-r.Top), two)
	}
	return true
}

func snowPaintNormDiagram(wp, lp uintptr) bool {
	if wp != snowNormDiagram || lp == 0 {
		return false
	}
	d := (*snowDrawStruct)(unsafe.Pointer(lp))
	dc := d.HDC
	fill := u32.NewProc("FillRect")
	b, _, _ := g32.NewProc("CreateSolidBrush").Call(0x00ffffff)
	fill.Call(dc, uintptr(unsafe.Pointer(&d.Rect)), b)
	g32.NewProc("DeleteObject").Call(b)
	x, y := int(d.Rect.Left), int(d.Rect.Top)
	w, h := int(d.Rect.Right-d.Rect.Left), int(d.Rect.Bottom-d.Rect.Top)
	write := func(x, y int, s string) {
		u := syscall.StringToUTF16(s)
		g32.NewProc("SetBkMode").Call(dc, 1)
		g32.NewProc("SetTextColor").Call(dc, 0x222222)
		g32.NewProc("TextOutW").Call(dc, uintptr(x), uintptr(y), uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1))
	}
	shape := snowDraft.Shape
	if shape == "cone" || shape == "dome" || shape == "sphere_cone" {
		write(x+10, y+6, "Схема-пояснение из СП")
		snowPaintNormativeAsset(dc, x+4, y+31, w-8, h-36, shape)
		return true
	}
	write(x+10, y+6, "Геометрическая схема")
	if shape == "skylight" {
		two := snowDraft.RightRoof != nil && snowDraft.RightRoof.Shape == "skylight"
		snowPaintRoofAsset(dc, x+8, y+36, w-16, 170, shape, false)
		write(x+10, y+223, "План зон A, B, C")
		snowPaintLanternPlan(dc, x+5, y+250, w-10, h-260, two)
	} else {
		snowPaintRoofAsset(dc, x+8, y+40, w-16, tempMax(150, h-60), shape, false)
	}
	return true
}
