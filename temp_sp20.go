package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type SP20Guide struct {
	Version    int       `json:"version"`
	Categories []string  `json:"categories"`
	Records    []SP20Row `json:"records"`
}
type SP20Row struct {
	ID         string  `json:"id"`
	SourceID   string  `json:"source_id"`
	Category   string  `json:"category"`
	Title      string  `json:"title"`
	Official   string  `json:"official"`
	Type       string  `json:"type"`
	Unit       string  `json:"unit"`
	Norm       float64 `json:"norm"`
	Gamma      float64 `json:"gamma"`
	Design     float64 `json:"design"`
	ExtraPoint float64 `json:"extra_point_kN"`
	Source     string  `json:"source"`
	Notes      string  `json:"notes"`
	Minimum    bool    `json:"minimum"`
	Editable   bool    `json:"editable,omitempty"`
}

func loadSP20Guide(path string) (SP20Guide, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return SP20Guide{}, e
	}
	var g SP20Guide
	if e = json.Unmarshal(b, &g); e != nil {
		return g, e
	}
	if g.Version != 1 || len(g.Records) < 30 {
		return g, fmt.Errorf("пустой справочник СП 20.13330.2016")
	}
	for _, r := range g.Records {
		if r.Title == "" || r.Unit == "" || (!r.Editable && r.Norm <= 0) || r.Gamma <= 0 || (!r.Editable && r.Design <= 0) {
			return g, fmt.Errorf("неполная строка СП 20.13330.2016: %s", r.ID)
		}
		if abs(r.Design-r.Norm*r.Gamma) > 0.0001 {
			return g, fmt.Errorf("ошибка расчёта в СП 20.13330.2016: %s", r.ID)
		}
	}
	return g, nil
}
func abs(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}
func sp20Selection(r SP20Row) TempSelection {
	p := map[string]interface{}{"normative": r.Norm, "gamma_f": r.Gamma, "design": r.Design, "unit": r.Unit, "load_kind": r.Type, "sp20_row": r.SourceID}
	if r.ExtraPoint > 0 {
		p["local_P_kN"] = r.ExtraPoint
	}
	n := r.Notes
	if r.Minimum {
		if n != "" {
			n += " "
		}
		n += "Значение не менее указанного; проверить технологическое задание."
	}
	return TempSelection{Group: "СП 20 · временные", Title: r.Title, Source: r.Source, Params: p, Notes: n}
}
func sp20DisplayRow(r SP20Row) string {
	n := r.Title
	rr := []rune(n)
	if len(rr) > 63 {
		n = string(rr[:60]) + "..."
	}
	return fmt.Sprintf("%-63s | %7.2f | %4.2f | %7.2f", n, r.Norm, r.Gamma, r.Design)
}
func sp20Details(r SP20Row) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\r\n\r\nНормативная: %s %s\r\nКоэффициент надёжности: %s\r\nРасчётная: %s %s", r.Title, strconv.FormatFloat(r.Norm, 'f', -1, 64), r.Unit, strconv.FormatFloat(r.Gamma, 'f', -1, 64), strconv.FormatFloat(r.Design, 'f', -1, 64), r.Unit)
	if r.ExtraPoint > 0 {
		fmt.Fprintf(&b, "\r\nМестная сосредоточенная: %g кН (отдельная проверка)", r.ExtraPoint)
	}
	if r.Minimum {
		b.WriteString("\r\nМинимальное значение. При наличии технологического задания может потребоваться большее.")
	}
	if r.Notes != "" {
		b.WriteString("\r\n" + r.Notes)
	}
	return b.String()
}

func sp20CustomSection(r SP20Row, title, raw, rawGamma string) (TempSelection, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return TempSelection{}, fmt.Errorf("Укажите название нагрузки")
	}
	normSet := strings.TrimSpace(raw) != ""
	var norm float64
	var e error
	if normSet {
		norm, e = parseNumber(raw)
		if e != nil || norm < 0 {
			return TempSelection{}, fmt.Errorf("Укажите неотрицательную нагрузку, кгс/м²")
		}
	}
	gamma, e := parseNumber(rawGamma)
	if e != nil || gamma <= 0 {
		return TempSelection{}, fmt.Errorf("Проверьте коэффициент надёжности")
	}
	r.Title = title
	r.Gamma = gamma
	r.Norm = norm * kgfM2ToKpa(1)
	r.Design = r.Norm * gamma
	s := sp20Selection(r)
	s.Params["is_defined"] = normSet
	s.Params["normative_reduced"] = r.Norm
	s.Params["design_reduced"] = r.Design
	s.Params["calc_scope"] = "По заданию пользователя"
	return s, nil
}
