package main

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

func tempReadableSource(source string) string {
	source = strings.ReplaceAll(source, ".pdf PDF", "")
	source = strings.ReplaceAll(source, ".pdf", "")
	source = strings.ReplaceAll(source, "PDF стр.", "стр.")
	source = strings.ReplaceAll(source, "PDF", "")
	source = strings.ReplaceAll(source, "СП35 ", "СП 35.13330.2011, ")
	source = strings.ReplaceAll(source, "СП 35,", "СП 35.13330.2011,")
	source = strings.ReplaceAll(source, "СП 20.13330.2016,", "СП 20.13330.2016,")
	source = regexp.MustCompile(`(?i)(Выпуск\s+\d+)\s+стр\.`).ReplaceAllString(source, "$1, стр.")
	ws := regexp.MustCompile(`\s{2,}`)
	return strings.TrimSpace(ws.ReplaceAllString(source, " "))
}
func tempTransport(s TempSelection) bool { return s.Group == "СП 35 · транспорт" }
func tempDisplayNumber(v float64, unit string) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "—"
	}
	return strings.TrimRight(strings.TrimRight(tempRU(v, 2), "0"), ",") + " " + unit
}
func tempFloatValue(s TempSelection, key string) (float64, bool) {
	if s.Params == nil {
		return 0, false
	}
	v, ok := s.Params[key]
	if !ok || v == nil {
		return 0, false
	}
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case string:
		n, e := strconv.ParseFloat(strings.ReplaceAll(val, ",", "."), 64)
		return n, e == nil
	}
	return 0, false
}
func tempTransportFields(s TempSelection) (details, norm, gamma, calc string) {
	return transportSummaryFields(s)
}
func tempParamMini(s TempSelection, key, title, unit string) string {
	v, ok := tempFloatValue(s, key)
	if !ok {
		return ""
	}
	return title + " " + tempDisplayNumber(v, unit)
}
func tempTransportSummary(s TempSelection) (string, string) {
	details, norm, gamma, calc := tempTransportFields(s)
	return details + "; " + norm, "γf: " + gamma + "; " + calc
}
func tempSP35NormalTitle(s string) string {
	s = strings.ReplaceAll(s, "Железнодорожная нагрузка", "Железнодорожный состав")
	return s
}

func tempReportInclude(s TempSelection) bool {
	return s.Title != "" && s.Title != "Новая нагрузка"
}
