package main

import (
	"fmt"
	"math"
	"strings"
)

// IceWindInput stores user-confirmed wind parameters. The wind load in this
// module is the MEAN component under 11.1.2–11.1.3. A verified pulsation
// increment can be entered from an independent full wind calculation.
// A missing increment does not imply that the pulsation is zero.
type IceWindInput struct {
	Enabled       bool    `json:"enabled,omitempty"`
	Flexible      bool    `json:"flexible,omitempty"` // selected structural classification; ψ per 12.3б
	Zone          string  `json:"zone,omitempty"`
	Terrain       string  `json:"terrain,omitempty"` // A,B,C from 11.1.6
	Drag          float64 `json:"drag,omitempty"`    // c, independently selected from appendix V
	DragSource    string  `json:"drag_source,omitempty"`
	DragIced      float64 `json:"drag_iced,omitempty"`   // 0 = same proven c as original element
	PulseRatio    float64 `json:"pulse_ratio,omitempty"` // wp / wm from separately verified 11.1.8 analysis
	PulseVerified bool    `json:"pulse_verified,omitempty"`
	PulseSource   string  `json:"pulse_source,omitempty"`
}

const iceWindGamma = 1.4 // SP20 11.4
func (w IceWindInput) validate() error {
	if !w.Enabled {
		return nil
	}
	if _, ok := windForZone(w.Zone); !ok {
		return fmt.Errorf("Для совместного расчёта выберите ветровой район по табл. 11.1")
	}
	if w.Terrain != "A" && w.Terrain != "B" && w.Terrain != "C" {
		return fmt.Errorf("Выберите тип местности A, B или C по п. 11.1.6")
	}
	if w.Drag <= 0 || w.Drag > 10 || math.IsNaN(w.Drag) || math.IsInf(w.Drag, 0) {
		return fmt.Errorf("Укажите обоснованный аэродинамический коэффициент c по приложению В СП 20.13330.2016")
	}
	if w.DragIced < 0 || w.DragIced > 10 || math.IsNaN(w.DragIced) || math.IsInf(w.DragIced, 0) {
		return fmt.Errorf("Недопустимый аэродинамический коэффициент обледеневшего элемента")
	}
	if w.PulseVerified && (w.PulseRatio < 0 || w.PulseRatio > 10 || math.IsNaN(w.PulseRatio) || strings.TrimSpace(w.PulseSource) == "") {
		return fmt.Errorf("Для пульсационной составляющей укажите подтверждённое значение wp/wm и источник")
	}
	return nil
}
func iceWindK(h float64, terrain string) (float64, error) {
	heights := []float64{5, 10, 20, 40, 60, 80, 100, 150, 200, 250, 300}
	var values []float64
	switch terrain {
	case "A":
		values = []float64{.75, 1, 1.25, 1.5, 1.7, 1.85, 2, 2.25, 2.45, 2.65, 2.75}
	case "B":
		values = []float64{.5, .65, .85, 1.1, 1.3, 1.45, 1.6, 1.9, 2.1, 2.3, 2.5}
	case "C":
		values = []float64{.4, .4, .55, .8, 1, 1.15, 1.25, 1.55, 1.8, 2, 2.2}
	default:
		return 0, fmt.Errorf("Неизвестная местность ветра %q", terrain)
	}
	if h > 300 {
		return 0, fmt.Errorf("Для ветрового расчёта выше 300 м k(z) требует научно-технического сопровождения по п. 11.1.6")
	}
	return iceLinear(h, heights, values), nil
}
func iceWindAtHeight(s IceLoadCalc, p *IceLoadPoint) error {
	k, err := iceWindK(p.HeightM, s.Wind.Terrain)
	if err != nil {
		return err
	}
	w0, _ := windForZone(s.Wind.Zone)
	// w0 is in kPa, so use 1000 Pa/kPa. Applied as w0*k*c.
	meanPa := 1000 * w0 * k * s.Wind.Drag
	meanPaIced := meanPa
	if s.Wind.DragIced > 0 {
		meanPaIced = 1000 * w0 * k * s.Wind.DragIced
	}
	// For round elements, linear wind load uses reference projected diameter (m).
	// For nonround construction members this is wind pressure (Pa).
	original, iced := 1., 1.
	if s.IsRound() {
		original = s.DiameterMM / 1000
		iced = (s.DiameterMM + 2*p.EffectiveMM) / 1000
	}
	p.WindK = k
	if s.IsRound() {
		p.IcedDMM = s.DiameterMM + 2*p.EffectiveMM
	}
	p.WindMean = meanPaIced * iced
	increase := 1.0
	if s.Wind.PulseVerified {
		increase += s.Wind.PulseRatio
	}
	p.WindNorm = p.WindMean * increase
	psi := .6
	if s.IsFlexible() {
		psi = .25
	}
	p.WindDesign = p.WindNorm * iceWindGamma * psi
	p.WindLeading = meanPa * original * increase * iceWindGamma
	return nil
}
func iceWindCompleteness(s IceLoadCalc) string {
	if !s.Wind.Enabled {
		return "Ветровой расчёт не задан. Выполнен только расчёт гололёда."
	}
	if !s.Wind.PulseVerified {
		return "ОГРАНИЧЕНИЕ: учтена только средняя составляющая ветра. Пульсации/динамика по п. 11.1.8 не рассчитаны; ветровые результаты не являются полной нормативной нагрузкой."
	}
	return "Пульсационная составляющая принята по отдельно подтверждённому расчёту: " + s.Wind.PulseSource + ". Проверка применимости аэродинамического коэффициента, резонанса и специальных воздействий остаётся за проектировщиком."
}
