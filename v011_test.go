package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func Test011IndependentElevationAndNoDiagnosticInReport(t *testing.T) {
	for _, raw := range []string{"Сиськи", "80", "0", "-25"} {
		s := Site{Region: "Краснодарский край", Place: "Тихорецк"}
		s.SetElevationText(raw)
		if !siteReportHasInput(s) {
			t.Fatal("only elevation must count as filled")
		}
		rows, err := siteReportRows(s)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, r := range rows {
			if r.Parameter == "Абсолютная отметка h, м" && r.Value == raw {
				found = true
			}
		}
		if !found {
			t.Fatalf("lost height %q", raw)
		}
		word := filepath.Join(t.TempDir(), "check.docx")
		xlsx := filepath.Join(t.TempDir(), "check.xlsx")
		if e := ExportSiteWord(s, word); e != nil {
			t.Fatal(e)
		}
		if e := ExportSiteExcel(s, xlsx); e != nil {
			t.Fatal(e)
		}
		for _, tc := range []struct{ file, part string }{{word, "word/document.xml"}, {xlsx, "xl/worksheets/sheet1.xml"}} {
			data := zipPart(t, tc.file, tc.part)
			if !strings.Contains(data, raw) {
				t.Fatalf("lost %q in %s", raw, tc.file)
			}
			if strings.Contains(data, "Районы определены по предварительному справочнику") || strings.Contains(data, "Примечание: районы") || strings.Contains(data, "ПРЕДУПРЕЖДЕНИЕ") {
				t.Fatal("diagnostic leaked into the report")
			}
		}
	}
	empty := Site{}
	if siteReportHasInput(empty) {
		t.Fatal("empty form isn't empty")
	}
	word := filepath.Join(t.TempDir(), "empty.docx")
	if err := ExportSiteWord(empty, word); err != nil {
		t.Fatal(err)
	}
}
func Test011HTMLAndRawWikiFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/w/index.php":
			if r.URL.Query().Get("title") == "Майкоп" {
				fmt.Fprint(w, "{{НП России\n|Регион = Республика Адыгея\n|Высота центра НП = 220 м\n}}")
			} else {
				http.NotFound(w, r)
			}
		case "/wiki/Тихорецк":
			fmt.Fprint(w, `<html><table><tr><th>Субъект Федерации</th><td>Краснодарский край</td></tr><tr><th>Высота центра</th><td>80 м</td></tr></table></html>`)
		default:
			http.Error(w, "api blocked", 403)
		}
	}))
	defer server.Close()
	for _, test := range []struct {
		title, region string
		height        float64
	}{{"Майкоп", "Республика Адыгея", 220}, {"Тихорецк", "Краснодарский край", 80}} {
		h, _, err := wikiNonAPIFallback(context.Background(), server.Client(), server.URL+"/w/api.php", test.title, test.region)
		if err != nil || h != test.height {
			t.Fatalf("%s: h=%g %v", test.title, h, err)
		}
	}
}
