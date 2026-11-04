package main

type ProjectWeightBlock struct {
	Title string              `json:"title"`
	Rows  []ProjectWeightLine `json:"rows"`
}

type ProjectWeightLine struct {
	Name   string  `json:"name"`
	Norm   float64 `json:"norm_kpa"`
	Gamma  float64 `json:"gamma_f"`
	Design float64 `json:"design_kpa"`
}

func projectWeightLines(p Project) []ProjectWeightLine {
	w := p.CurrentWeight()
	out := make([]ProjectWeightLine, 0, len(w.Rows))
	for _, r := range w.Rows {
		if r.Name == "" {
			continue
		}
		out = append(out, ProjectWeightLine{r.Name, kgfM2ToKpa(r.NormKgf()), r.Gamma, kgfM2ToKpa(r.NormKgf() * r.Gamma)})
	}
	return out
}
func projectSiteLines(s Site, p Project) []string {
	// Do not manufacture site parameters; only include values entered/selected by the user.
	if s.Region == "" && s.District == "" && s.Place == "" && s.SnowZone == "" && s.WindZone == "" && s.IceZone == "" && s.SeismicMap == "" && !p.Snow.Enabled && !p.Ice.Enabled && !p.WindOverride {
		return nil
	}
	out := []string{}
	name := formatSitePlace(s.Region, s.District, s.Place)
	if name != "" {
		out = append(out, "Местность: "+name)
	}
	if s.SnowZone != "" {
		out = append(out, "Снеговой район "+s.SnowZone+", Sg = "+decimalInput(s.EffectiveSnow(), 3)+" кПа")
	}
	if s.WindZone != "" {
		if w, ok := windForZone(s.WindZone); ok {
			out = append(out, "Ветровой район "+s.WindZone+", w0 = "+decimalInput(w, 3)+" кПа")
		} else {
			out = append(out, "Ветровой район "+s.WindZone)
		}
	}
	if s.IceZone != "" {
		out = append(out, "Гололёдный район: "+s.IceZone)
	}
	if s.SeismicMap != "" && s.SeismicPoints != "" {
		out = append(out, "Сейсмичность региона по карте "+s.SeismicMap+": "+seismicReportIntensity(s.SeismicPoints))
	}
	if p.Snow.Enabled {
		out = append(out, "Снеговая нагрузка, уточнённое Sg = "+decimalInput(p.Snow.Value, 3)+" кПа")
	}
	if p.Ice.Enabled {
		out = append(out, "Гололёд: уточнённый район "+p.Ice.Zone)
	}
	if p.WindOverride {
		out = append(out, "Ветровое давление, локальная правка w0 = "+decimalInput(p.Wind.W0, 3)+" кПа")
	}
	return out
}
