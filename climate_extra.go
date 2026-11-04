package main

import "strings"

const snowOptionK = "Особый, по приложению К"
const snowOptionMountain = "Горные и малоизученные районы"

func isKChoice(v string) bool { return v == "Особый" || v == snowOptionK }

var snowOptions = []string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", snowOptionK, snowOptionMountain}
var windOptions = []string{"Iа", "I", "II", "III", "IV", "V", "VI", "VII", "Пользовательский"}
var iceOptions = []string{"I", "II", "III", "IV", "V"}

func validClimateRecord(r ClimateRecord) bool {
	return r.SnowK > 0 || r.SnowZone != "" || r.WindZone != "" || r.IceZone != ""
}
func specialSnowSite(s Site) bool {
	return s.MountainID == "" && strings.Contains(s.SnowNote, "приложение К") && s.SnowGround > 0
}
func snowChoice(s Site) string {
	if specialSnowSite(s) {
		return snowOptionK
	}
	return s.SnowZone
}
func snowValueForChoice(choice string, specialSite Site) (float64, bool) {
	if isKChoice(choice) {
		if specialSnowSite(specialSite) {
			return specialSite.SnowGround, true
		}
		return 0, false
	}
	return snowForZone(choice)
}
func windDisplayZone(zone string) string {
	if zone == "IA" {
		return "Iа"
	}
	return zone
}
func windInternalZone(zone string) string {
	if zone == "Iа" || zone == "Ia" || zone == "IA" {
		return "IA"
	}
	return romanZone(zone)
}
func climateZoneNotes(kind, zone string) string {
	switch kind {
	case "snow":
		return "СП 20.13330.2016, п. 10.2, табл. 10.1"
	case "wind":
		return "СП 20.13330.2016, п. 11.1.4, табл. 11.1"
	case "ice":
		return "СП 20.13330.2016, п. 12.2, табл. 12.1; для районов I и V значения минимальные"
	}
	return ""
}
