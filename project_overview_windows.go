//go:build windows

package main

import "fmt"

const (
	tempGeneralPlace   = 12901
	tempGeneralClimate = 12902
	tempGeneralWeight  = 12903
)

func projectHomeOverview() {
	site := formatSitePlace(project.Site.Region, project.Site.District, project.Site.Place)
	if site == "" {
		site = "местность не выбрана"
	}
	counts := map[string]int{}
	for _, b := range project.SavedBlocks {
		counts[b.Kind]++
	}
	set(403, "Местность: "+site)
	set(404, fmt.Sprintf("В составе отчёта: собственный вес — %d · снег — %d · гололёд — %d · прочие схемы — %d", counts["weight"], counts["snow"], counts["ice"], counts["temporary"]))
}
func projectOverviewRefresh() {
	s := project.Site
	site := formatSitePlace(s.Region, s.District, s.Place)
	if site == "" {
		site = "Местность не выбрана"
	}
	set(tempGeneralPlace, "Местность: "+site)
	snow, wind, ice := known(s.SnowZone), known(s.WindZone), known(s.IceZone)
	if z, sg, _ := project.SnowInputs(); sg > 0 {
		snow = known(z) + " (Sg=" + decimalInput(sg, 3) + " кПа)"
	}
	seismic := "не задана"
	if s.SeismicPoints != "" {
		seismic = "карта " + s.SeismicMap + ", " + seismicReportIntensity(s.SeismicPoints)
	}
	set(tempGeneralClimate, "Снег: "+snow+" · ветер: "+wind+" · гололёд: "+ice+" · сейсмика: "+seismic)
	compositionRefresh()

}
