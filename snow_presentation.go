package main

import "strings"

// SnowLoadView is an engineering loading arrangement as seen by the user.
// A single B.3 loading arrangement consists of several longitudinal zones;
// the cross-sections for A/C/B are not independent alternative load cases.
type SnowLoadView struct {
	Title string
	Note  string
	Cases []SnowCase
	Zones []string
}

func snowPresentations(s SnowCalc) []SnowLoadView {
	out := make([]SnowLoadView, 0, len(s.Cases))
	if s.Shape == "skylight" && (s.RightRoof == nil || s.RightRoof.Shape == "none") {
		used := make(map[int]bool)
		for _, group := range []struct {
			prefixes []string
			title    string
		}{
			{[]string{"1A.", "1C."}, "Вариант 1. Совместное загружение зон A, B, C"},
			{[]string{"2A.", "2C."}, "Вариант 2. Совместное загружение зон A, B, C"},
		} {
			v := SnowLoadView{Title: group.title, Note: "Зоны A, B и C действуют одновременно по длине фонаря; зона B у торца фонаря принимается с μ = 1,0."}
			for _, prefix := range group.prefixes {
				for i, c := range s.Cases {
					if strings.HasPrefix(c.Title, prefix) {
						v.Cases = append(v.Cases, c)
						if strings.Contains(prefix, "A") {
							v.Zones = append(v.Zones, "A")
						} else {
							v.Zones = append(v.Zones, "C")
						}
						used[i] = true
					}
				}
			}
			if len(v.Cases) != 0 {
				v.Cases = append(v.Cases, snowCase("Зона B у торцов фонаря", snowNormSource+", Б.3, рисунок Б.5", "Зона B у торцов фонаря: μ = 1,0 в обоих вариантах", snowMuInterval(0, s.Span, 1)))
				v.Zones = append(v.Zones, "B")
				out = append(out, v)
			}
		}
		for i, c := range s.Cases {
			if !used[i] {
				out = append(out, SnowLoadView{Title: c.Title, Cases: []SnowCase{c}})
			}
		}
		return out
	}
	for _, c := range s.Cases {
		if s.Shape == "vault" && strings.Contains(strings.ToLower(c.Title), "зеркальн") {
			// The mirrored loading is still in the computational model, saved project
			// and detailed Excel; do not duplicate it in the on-screen/Word overview.
			continue
		}
		title := strings.ReplaceAll(c.Title, "Б.1. ", "")
		v := SnowLoadView{Title: title, Cases: []SnowCase{c}}
		if s.Shape == "vault" && strings.Contains(strings.ToLower(c.Title), "несимметричн") {
			v.Note = "Данная схема снегового загружения может применяться также в зеркальном отображении."
		}
		out = append(out, v)
	}
	return out
}

func (v SnowLoadView) MaxMu() float64 {
	max := 0.0
	for _, c := range v.Cases {
		for _, interval := range c.Intervals {
			if interval.Mu > max {
				max = interval.Mu
			}
		}
	}
	return max
}
