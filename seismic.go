package main

import (
	"bufio"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// Values are the intensity of ОСР-2015 A/B/C for *average soil conditions*.
// "-" in appendix A means no value specified for that map, not zero points.
type SeismicRecord struct{ Region, Place, A, B, C string }

func (r SeismicRecord) Value(m string) string {
	switch m {
	case "А":
		return r.A
	case "В":
		return r.B
	case "С":
		return r.C
	}
	return ""
}

const seismicNorm = "СП 14.13330.2018 (изм. № 2–5), п. 4.3, приложение А; ОСР-2015"

var seismicCatalog []SeismicRecord
var seismicReviews = map[string]string{}
var seismicCatalogStatus = "Справочник не загружен. Положите файл в Maps\\Seismic\\settlements.tsv и нажмите «Обновить»."

// Small verified seed is never presented as a *complete* appendix A.
func seismicSeed() []SeismicRecord {
	return []SeismicRecord{
		{"Республика Адыгея", "Адыгейск", "8", "8", "9"}, {"Республика Адыгея", "Гиагинская", "7", "8", "8"},
		{"Республика Адыгея", "Каменномостский", "8", "8", "9"}, {"Республика Адыгея", "Кошехабль", "7", "7", "8"},
		{"Республика Адыгея", "Красногвардейское", "7", "7", "8"}, {"Республика Адыгея", "Майкоп", "7", "8", "9"},
		{"Республика Адыгея", "Тульский", "7", "8", "9"}, {"Республика Адыгея", "Энем", "8", "8", "9"},
		{"Республика Адыгея", "Яблоновский", "8", "8", "9"},
		{"Республика Алтай", "Акташ", "9", "9", "10"}, {"Республика Алтай", "Горно-Алтайск", "8", "8", "9"},
		{"Республика Алтай", "Чемал", "8", "9", "10"},
		{"Республика Крым", "Симферополь", "7", "8", "9"}, {"Республика Крым", "Ялта", "8", "9", "10"},
		{"Республика Крым", "Джанкой", "6", "7", "7"}, {"Республика Крым", "Судак", "8", "9", "9"},
		{"Севастополь", "Севастополь", "8", "9", "9"},
		{"Республика Саха (Якутия)", "Якутск", "6", "7", "8"},
		{"Республика Северная Осетия - Алания", "Владикавказ", "8", "9", "10"},
	}
}

func normalizeGeo(s string) string {
	s = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), "ё", "е"))
	s = strings.NewReplacer("–", "-", "—", "-", "-", " ", "  ", " ", "г. ", "", "город ", "").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}
func canonicalSeismicRegion(raw string) string {
	n := normalizeGeo(raw)
	// Lists sometimes use different conventional names.
	aliases := map[string]string{
		"республика северная осетия алания": "Республика Северная Осетия - Алания", "республика северная осетия - алания": "Республика Северная Осетия - Алания",
		"республика саха": "Республика Саха (Якутия)", "республика саха (якутия)": "Республика Саха (Якутия)",
		"республика татарстан (татарстан)": "Республика Татарстан", "республика татарстан": "Республика Татарстан",
		"севастополь - (город федерального подчинения)": "Севастополь", "город севастополь": "Севастополь",
		"москва - (город федерального подчинения)": "Москва", "санкт-петербург - (город федерального подчинения)": "Санкт-Петербург",
	}
	if v := aliases[n]; v != "" {
		return v
	}
	for _, line := range strings.Split(strings.TrimSpace(regionCapitals), "\n") {
		a := strings.SplitN(strings.TrimSpace(line), "|", 2)
		if len(a) != 2 {
			continue
		}
		cand := strings.TrimSpace(a[0])
		cn := normalizeGeo(cand)
		if cn == n || n == cn+" (татарстан)" || strings.HasPrefix(n, cn+" - (") {
			return cand
		}
	}
	return ""
}
func canonicalRegionMatch(a, b string) bool {
	return canonicalSeismicRegion(a) != "" && canonicalSeismicRegion(a) == canonicalSeismicRegion(b) || normalizeGeo(a) == normalizeGeo(b)
}
func validIntensity(s string) bool {
	switch strings.TrimSpace(s) {
	case "-", "6", "7", "8", "9", "10", "11", "12":
		return true
	}
	return false
}
func seismicDedup(rows []SeismicRecord) []SeismicRecord {
	unique := map[string]SeismicRecord{}
	for _, r := range rows {
		r.Region = canonicalSeismicRegion(r.Region)
		r.Place = strings.TrimSpace(r.Place)
		if r.Region == "" || r.Place == "" || !validIntensity(r.A) || !validIntensity(r.B) || !validIntensity(r.C) {
			continue
		}
		if len(r.Place) > 120 {
			continue
		}
		key := normalizeGeo(r.Region) + "|" + normalizeGeo(r.Place)
		unique[key] = r
	}
	out := make([]SeismicRecord, 0, len(unique))
	for _, v := range unique {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Region != out[j].Region {
			return out[i].Region < out[j].Region
		}
		return out[i].Place < out[j].Place
	})
	return out
}

// Local list is a transparent, editable source file: Регион|Населённый пункт|А|В|С.
func parseSeismicTSV(text string) []SeismicRecord {
	var rows []SeismicRecord
	sc := bufio.NewScanner(strings.NewReader(strings.TrimPrefix(text, "\ufeff")))
	sc.Buffer(make([]byte, 1024), 1024*1024)
	seismicReviews = map[string]string{}
	for sc.Scan() {
		p := strings.Split(sc.Text(), "|")
		if len(p) < 5 || strings.HasPrefix(strings.TrimSpace(p[0]), "#") {
			continue
		}
		r := SeismicRecord{strings.TrimSpace(p[0]), strings.TrimSpace(p[1]), strings.TrimSpace(p[2]), strings.TrimSpace(p[3]), strings.TrimSpace(p[4])}
		rows = append(rows, r)
		if len(p) >= 6 && strings.TrimSpace(p[5]) != "" {
			seismicReviews[normalizeGeo(r.Region)+"|"+normalizeGeo(r.Place)] = strings.TrimSpace(p[5])
		}
	}
	return seismicDedup(rows)
}
func seismicTSV(rows []SeismicRecord) string {
	var b strings.Builder
	b.WriteString("# СП 14.13330.2018, приложение А: регион|населённый пункт|А|В|С\n")
	for _, r := range seismicDedup(rows) {
		fmt.Fprintf(&b, "%s|%s|%s|%s|%s\n", r.Region, r.Place, r.A, r.B, r.C)
	}
	return b.String()
}
func seismicFilePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "Maps", "Seismic", "settlements.tsv"), nil
}
func saveSeismicCatalog(rows []SeismicRecord) error {
	p, e := seismicFilePath()
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(p), 0755); e != nil {
		return e
	}
	return os.WriteFile(p, []byte(seismicTSV(rows)), 0644)
}
func loadSeismicCatalog() {
	seismicCatalog = seismicSeed()
	seismicReviews = map[string]string{}
	seismicCatalogStatus = "Файл Maps\\Seismic\\settlements.tsv отсутствует; показана только проверочная выборка."
	p, e := seismicFilePath()
	if e != nil {
		return
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return
	}
	v := parseSeismicTSV(string(b))
	if len(v) > 0 {
		seismicCatalog = v
		seismicCatalogStatus = fmt.Sprintf("Локальный справочник: %d населённых пунктов. %s", len(v), p)
	}
}

// Empty query: show the selected region (or all when it is unknown).
// Nonempty query: search both region and place across the whole country.
// Thus "Ростов" shows EVERY entry from Ростовская область.
func seismicMatches(region, term string) (rows []SeismicRecord, regionAvailable bool) {
	wanted := canonicalSeismicRegion(region)
	for _, r := range seismicCatalog {
		if wanted != "" && canonicalRegionMatch(r.Region, wanted) {
			regionAvailable = true
			break
		}
	}
	needle := normalizeGeo(term)
	for _, r := range seismicCatalog {
		if needle == "" {
			if regionAvailable && !canonicalRegionMatch(r.Region, wanted) {
				continue
			}
		} else if !strings.Contains(normalizeGeo(r.Region), needle) && !strings.Contains(normalizeGeo(r.Place), needle) {
			continue
		}
		rows = append(rows, r)
	}
	return
}
func seismicAutoMatch(region, place string) (SeismicRecord, bool) {
	for _, r := range seismicCatalog {
		if canonicalRegionMatch(r.Region, region) && normalizeGeo(r.Place) == normalizeGeo(place) {
			return r, true
		}
	}
	return SeismicRecord{}, false
}

var seismicTDRE = regexp.MustCompile(`(?is)<t[dh]\b[^>]*>(.*?)</t[dh]>`)
var seismicTrRE = regexp.MustCompile(`(?is)<tr\b[^>]*>(.*?)</tr>`)
var seismicTags = regexp.MustCompile(`(?is)<[^>]+>`)
var seismicWhitespace = regexp.MustCompile(`\s+`)

func flattenHTML(s string) string {
	return strings.TrimSpace(seismicWhitespace.ReplaceAllString(html.UnescapeString(seismicTags.ReplaceAllString(s, " ")), " "))
}
func parseSeismicHTML(src string) []SeismicRecord {
	// Appendices generally display three (place/A/B/C) blocks per table row.
	start := strings.Index(strings.ToLower(src), "приложение а")
	if start >= 0 {
		src = src[start:]
	}
	var rows []SeismicRecord
	region := ""
	for _, t := range seismicTrRE.FindAllStringSubmatch(src, -1) {
		cells := seismicTDRE.FindAllStringSubmatch(t[1], -1)
		if len(cells) == 0 {
			continue
		}
		var values []string
		for _, c := range cells {
			values = append(values, flattenHTML(c[1]))
		}
		if len(values) == 1 {
			if v := canonicalSeismicRegion(values[0]); v != "" {
				region = v
			}
			continue
		}
		// Region headings may be in the first cell with colspan.
		if v := canonicalSeismicRegion(values[0]); v != "" {
			region = v
			continue
		}
		if region == "" {
			continue
		}
		for i := 0; i+3 < len(values); i += 4 {
			r := SeismicRecord{region, values[i], values[i+1], values[i+2], values[i+3]}
			if validIntensity(r.A) && validIntensity(r.B) && validIntensity(r.C) {
				rows = append(rows, r)
			}
		}
	}
	return seismicDedup(rows)
}
func seismicDecode(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	var out strings.Builder
	for _, c := range b {
		switch {
		case c < 128:
			out.WriteByte(c)
		case c >= 0xC0:
			out.WriteRune(rune(0x0410 + int(c) - 0xC0))
		case c == 0xA8:
			out.WriteRune('Ё')
		case c == 0xB8:
			out.WriteRune('ё')
		case c == 0xAB:
			out.WriteRune('«')
		case c == 0xBB:
			out.WriteRune('»')
		case c == 0xB9:
			out.WriteRune('№')
		case c == 0x97:
			out.WriteRune('—')
		case c == 0x96:
			out.WriteRune('–')
		default:
			out.WriteByte(' ')
		}
	}
	return out.String()
}
func seismicDisplayPoints(s string) string {
	s = strings.TrimSpace(s)
	if s == "-" {
		return "5 и менее"
	}
	return s
}
func seismicReportIntensity(s string) string {
	if s == "5 и менее" {
		return "5 и менее баллов"
	}
	return s + " баллов"
}
