package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Height of a Wikidata geographical entity (P2044). This describes the
// encyclopedia's settlement, NOT a surveying elevation of a building site.
// Search is restricted by region to avoid silently accepting a namesake.
type WikiElevation struct {
	Height float64 `json:"height"`
	Source string  `json:"source"`
	Entity string  `json:"entity"`
}

var errWikiNotFound = errors.New("В Википедии не нашлось")
var errWikiRegion = errors.New("Не удалось однозначно сопоставить населённый пункт и регион")

func wikiCacheFile() string {
	dir, e := os.UserCacheDir()
	if e != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "Poverka", "wiki_elevations.json")
}
func wikiCacheKey(place, region, district string) string {
	return normalizeSearch(place) + "|" + normalizeSearch(region) + "|" + normalizeSearch(district)
}
func readWikiCache() map[string]WikiElevation {
	b, e := os.ReadFile(wikiCacheFile())
	if e != nil {
		return map[string]WikiElevation{}
	}
	m := map[string]WikiElevation{}
	if json.Unmarshal(b, &m) != nil {
		return map[string]WikiElevation{}
	}
	return m
}
func cacheWikiHeight(place, region, district string, v WikiElevation) {
	m := readWikiCache()
	m[wikiCacheKey(place, region, district)] = v
	b, e := json.MarshalIndent(m, "", "  ")
	if e != nil {
		return
	}
	fn := wikiCacheFile()
	if os.MkdirAll(filepath.Dir(fn), 0700) != nil {
		return
	}
	_ = os.WriteFile(fn, b, 0600)
}

// Local diagnostic journal helps reproduce real network failures without
// installing developer tools on the user's computer.
func wikiTrace(format string, args ...any) {
	path := filepath.Join(filepath.Dir(wikiCacheFile()), "wiki_lookup.log")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s ", time.Now().Format("2006-01-02 15:04:05"))
	_, _ = fmt.Fprintf(f, format+"\n", args...)
}
func wikiGet(ctx context.Context, client *http.Client, endpoint string, value any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "PoverkaNagruzki/0.11 (elevation lookup; engineering calculator)")
	req.Header.Set("Accept", "application/json")
	res, err := client.Do(req)
	if err != nil {
		wikiTrace("API %s %s: %v", req.URL.Host, req.URL.Query().Get("action"), err)
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		wikiTrace("API %s %s: HTTP %d", req.URL.Host, req.URL.Query().Get("action"), res.StatusCode)
		return fmt.Errorf("ответ сервера: HTTP %d", res.StatusCode)
	}
	err = json.NewDecoder(io.LimitReader(res.Body, 2*1024*1024)).Decode(value)
	wikiTrace("API %s %s: HTTP 200, JSON: %v", req.URL.Host, req.URL.Query().Get("action"), err)
	return err
}

type wikiSearchReply struct {
	Search []struct{ ID, Label, Description string } `json:"search"`
}
type wikiEntityReply struct {
	Entities map[string]struct {
		Sitelinks map[string]struct {
			Title string `json:"title"`
		} `json:"sitelinks"`
		Claims map[string][]struct {
			MainSnak struct {
				DataValue struct {
					Value struct {
						Amount string `json:"amount"`
						Unit   string `json:"unit"`
					} `json:"value"`
				} `json:"datavalue"`
			} `json:"mainsnak"`
			Rank string `json:"rank"`
		} `json:"claims"`
	} `json:"entities"`
}

func chooseWikiItem(candidates wikiSearchReply, place, region, district string) (string, error) {
	p, r, d := normalizeSearch(place), normalizeSearch(region), normalizeSearch(district)
	// Search descriptions mention the administrative region for many villages;
	// if they do not, do NOT silently choose the first eponymous village.
	bestScore, bestID, tie := -1, "", false
	for _, c := range candidates.Search {
		title, description := normalizeSearch(c.Label), normalizeSearch(c.Description)
		if title != p {
			continue
		}
		score := 1
		if r != "" {
			// Full subject name or a significant stem, e.g. "Краснодарский край".
			parts := strings.Fields(r)
			token := ""
			for _, v := range parts {
				if v == "республика" || v == "область" || v == "край" || v == "автономный" || v == "округ" || v == "город" {
					continue
				}
				if len([]rune(v)) > len([]rune(token)) {
					token = v
				}
			}
			stem := []rune(token)
			if len(stem) > 6 {
				stem = stem[:len(stem)-3]
			}
			if strings.Contains(description, r) {
				score += 8
			} else if len(stem) > 0 && strings.Contains(description, string(stem)) {
				score += 6
			} else {
				continue
			} // cannot confirm administrative identity
		}
		if d != "" {
			words := strings.Fields(d)
			for _, word := range words {
				chars := []rune(word)
				if len(chars) > 5 && strings.Contains(description, string(chars[:len(chars)-3])) {
					score += 3
					break
				}
			}
		}
		if score > bestScore {
			bestScore, bestID, tie = score, c.ID, false
		} else if score == bestScore {
			tie = true
		}
	}
	if bestID == "" {
		return "", errWikiNotFound
	}
	if tie {
		return "", errWikiRegion
	}
	return bestID, nil
}
func parseWikiHeight(data wikiEntityReply, entity string) (float64, bool) {
	e, ok := data.Entities[entity]
	if !ok {
		return 0, false
	}
	for _, c := range e.Claims["P2044"] {
		if c.Rank == "deprecated" {
			continue
		}
		value := c.MainSnak.DataValue.Value
		n, err := strconv.ParseFloat(value.Amount, 64)
		if err != nil {
			continue
		}
		switch {
		case strings.HasSuffix(value.Unit, "/Q11573"):
			return n, true // metres
		case strings.HasSuffix(value.Unit, "/Q828224"):
			return n * 1000, true // km
		case strings.HasSuffix(value.Unit, "/Q3710"):
			return n * .3048, true // feet
		}
	}
	return 0, false
}

// Parses only an explicit settlement infobox field, never an arbitrary number from a page.
var wikiHeightPattern = regexp.MustCompile(`(?im)^\|\s*(?:высота(?:\s+центра)?(?:\s+нп|\s+насел[её]нного\s+пункта)?|абсолютная\s+высота)\s*=\s*([^\r\n|]+)`)
var firstNumber = regexp.MustCompile(`[-+]?\d+(?:[.,]\d+)?`)

func wikiInfoboxHeight(wikitext string) (float64, bool) {
	m := wikiHeightPattern.FindStringSubmatch(wikitext)
	if len(m) < 2 {
		return 0, false
	}
	value := firstNumber.FindString(m[1])
	if value == "" {
		return 0, false
	}
	h, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", "."), 64)
	return h, err == nil && h >= -500 && h <= 9000
}

// Only use Wikipedia's settlement infobox after verifying that the page
// actually belongs to the requested subject of the RF.
func wikipediaRegionMatches(wikitext, region string) bool {
	if strings.TrimSpace(region) == "" {
		return false
	}
	n := normalizeSearch(region)
	source := normalizeSearch(wikitext)
	words := strings.Fields(n)
	for _, word := range words {
		if word == "область" || word == "республика" || word == "край" || word == "автономный" || word == "округ" || word == "город" || word == "федерального" || word == "значения" {
			continue
		}
		r := []rune(word)
		if len(r) > 7 {
			r = r[:len(r)-3]
		} else if len(r) > 5 {
			r = r[:len(r)-2]
		}
		if len(r) >= 4 && strings.Contains(source, string(r)) {
			return true
		}
	}
	return false
}
func wikiWikipediaFallback(ctx context.Context, client *http.Client, wikiEndpoint, title, region string) (float64, bool, error) {
	if title == "" {
		return 0, false, nil
	}
	q := url.Values{}
	q.Set("action", "parse")
	q.Set("format", "json")
	q.Set("formatversion", "2")
	q.Set("prop", "wikitext")
	q.Set("page", title)
	var reply struct {
		Parse struct {
			Wikitext string `json:"wikitext"`
		} `json:"parse"`
		Error struct{ Code, Info string } `json:"error"`
	}
	if err := wikiGet(ctx, client, wikiEndpoint+"?"+q.Encode(), &reply); err != nil {
		return 0, false, err
	}
	if reply.Error.Code != "" {
		return 0, false, nil
	}
	if !wikipediaRegionMatches(reply.Parse.Wikitext, region) {
		return 0, false, nil
	}
	h, ok := wikiInfoboxHeight(reply.Parse.Wikitext)
	return h, ok, nil
}

// Read the article outside the MediaWiki JSON API. Some sites/proxies return
// filtered API data while serving the ordinary encyclopedia pages normally.
func wikiTextGet(ctx context.Context, client *http.Client, endpoint string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "PoverkaNagruzki/0.11 (elevation lookup)")
	req.Header.Set("Accept", "text/html, text/plain;q=0.9")
	resp, err := client.Do(req)
	if err != nil {
		wikiTrace("PAGE %s: %v", req.URL.String(), err)
		return "", fmt.Errorf("%s: %w", req.URL.Host, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		wikiTrace("PAGE %s: HTTP %d", req.URL.String(), resp.StatusCode)
		return "", fmt.Errorf("%s: HTTP %d", req.URL.Host, resp.StatusCode)
	}
	wikiTrace("PAGE %s: HTTP 200", req.URL.String())
	buf, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

var htmlRowPattern = regexp.MustCompile(`(?is)<tr\b[^>]*>.*?</tr>`)
var htmlTagPattern = regexp.MustCompile(`(?s)<[^>]+>`)
var htmlWhitespace = regexp.MustCompile(`\s+`)

// Read only an explicit table row headed "Высота центра". The text of the
// article isn't scanned for arbitrary elevations or other unrelated numbers.
func wikiArticleHeight(page string) (float64, bool) {
	for _, row := range htmlRowPattern.FindAllString(page, -1) {
		text := html.UnescapeString(htmlTagPattern.ReplaceAllString(row, " "))
		text = htmlWhitespace.ReplaceAllString(text, " ")
		low := strings.ToLower(text)
		pos := strings.Index(low, "высота центра")
		if pos < 0 {
			continue
		}
		part := text[pos+len("высота центра"):]
		v := firstNumber.FindString(part)
		if v == "" {
			continue
		}
		h, err := strconv.ParseFloat(strings.ReplaceAll(v, ",", "."), 64)
		if err == nil && h >= -500 && h <= 9000 {
			return h, true
		}
	}
	return 0, false
}
func wikiNonAPIFallback(ctx context.Context, client *http.Client, wikiEndpoint, title, region string) (float64, string, error) {
	if title == "" {
		return 0, "", errWikiNotFound
	}
	base := strings.TrimSuffix(wikiEndpoint, "/w/api.php")
	if base == wikiEndpoint {
		// Test server accepts all actions at one path.
		base = strings.TrimSuffix(wikiEndpoint, "/")
	}
	q := url.Values{"title": {title}, "action": {"raw"}}
	raw, rawErr := wikiTextGet(ctx, client, base+"/w/index.php?"+q.Encode())
	if rawErr == nil && wikipediaRegionMatches(raw, region) {
		if h, ok := wikiInfoboxHeight(raw); ok {
			return h, "исходный текст статьи", nil
		}
	}
	article := base + "/wiki/" + url.PathEscape(strings.ReplaceAll(title, " ", "_"))
	page, pageErr := wikiTextGet(ctx, client, article)
	if pageErr == nil && wikipediaRegionMatches(html.UnescapeString(htmlTagPattern.ReplaceAllString(page, " ")), region) {
		if h, ok := wikiArticleHeight(page); ok {
			return h, "карточка статьи (HTML)", nil
		}
	}
	if rawErr != nil && pageErr != nil {
		return 0, "", fmt.Errorf("исходный текст: %v; страница: %v", rawErr, pageErr)
	}
	return 0, "", errWikiNotFound
}

func fetchWikiElevation(ctx context.Context, client *http.Client, base, place, region, district string) (WikiElevation, error) {
	wikiEndpoint := "https://ru.wikipedia.org/w/api.php"
	if strings.HasPrefix(base, "http://127.0.0.1") || strings.HasPrefix(base, "http://localhost") {
		wikiEndpoint = base
	}
	if strings.TrimSpace(place) == "" {
		return WikiElevation{}, errWikiNotFound
	}

	// Wikidata is useful when the entity description contains the subject.
	// The previous version incorrectly treated a missing region in a *description*
	// as proof that the city does not exist (Maykop and Tikhoretsk).
	q := url.Values{}
	q.Set("action", "wbsearchentities")
	q.Set("format", "json")
	q.Set("language", "ru")
	q.Set("search", place)
	q.Set("type", "item")
	q.Set("limit", "50")
	search := wikiSearchReply{}
	err := wikiGet(ctx, client, base+"?"+q.Encode(), &search)
	identityError := errWikiNotFound
	if err == nil {
		entity, matchError := chooseWikiItem(search, place, region, district)
		if matchError != nil {
			identityError = matchError
		} else {
			q = url.Values{}
			q.Set("action", "wbgetentities")
			q.Set("format", "json")
			q.Set("ids", entity)
			q.Set("props", "claims|sitelinks")
			detail := wikiEntityReply{}
			if detailErr := wikiGet(ctx, client, base+"?"+q.Encode(), &detail); detailErr == nil {
				if h, found := parseWikiHeight(detail, entity); found && h >= -500 && h <= 9000 {
					return WikiElevation{Height: h, Entity: entity, Source: "Википедия / Wikidata (P2044), " + entity}, nil
				}
				if en, ok := detail.Entities[entity]; ok {
					if title := en.Sitelinks["ruwiki"].Title; title != "" {
						if h, ok, _ := wikiWikipediaFallback(ctx, client, wikiEndpoint, title, region); ok {
							return WikiElevation{Height: h, Entity: entity, Source: "Русская Википедия, карточка населённого пункта: " + title}, nil
						}
					}
				}
			}
		}
	}
	// Try the exact Russian Wikipedia article before declaring data absent.
	// Wikidata descriptions frequently omit the subject RF, which is NOT evidence
	// that the settlement or its height doesn't exist.
	if h, ok, _ := wikiWikipediaFallback(ctx, client, wikiEndpoint, place, region); ok {
		return WikiElevation{Height: h, Source: "Русская Википедия, карточка населённого пункта: " + place}, nil
	}
	h, how, fallbackErr := wikiNonAPIFallback(ctx, client, wikiEndpoint, place, region)
	if fallbackErr == nil {
		return WikiElevation{Height: h, Source: "Русская Википедия, " + how + ": " + place}, nil
	}
	if err != nil {
		return WikiElevation{}, fmt.Errorf("Wikidata API недоступен (%v); Википедия (%v)", err, fallbackErr)
	}
	if fallbackErr != nil && !errors.Is(fallbackErr, errWikiNotFound) {
		// Do not disguise an HTTP/connection failure as "no height in Wikipedia".
		return WikiElevation{}, fmt.Errorf("страница Википедии недоступна: %w", fallbackErr)
	}
	return WikiElevation{}, identityError
}

func lookupWikiElevation(place, region, district string) (WikiElevation, error) {
	wikiTrace("Проверка: %q, %q, %q", place, district, region)
	if v, ok := readWikiCache()[wikiCacheKey(place, region, district)]; ok {
		wikiTrace("Успешно: локальный кэш, высота %.2f м", v.Height)
		return v, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 10 * time.Second}
	result, err := fetchWikiElevation(ctx, client, "https://www.wikidata.org/w/api.php", place, region, district)
	if err == nil {
		cacheWikiHeight(place, region, district, result)
		wikiTrace("Успешно: %.2f м, %s", result.Height, result.Source)
	} else {
		wikiTrace("Результат: %v", err)
	}
	return result, err
}
