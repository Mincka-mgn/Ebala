package main

import (
	"archive/zip"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func almost(a, b float64) bool { return math.Abs(a-b) < 1e-12 }
func TestInitial(t *testing.T) {
	p := initialProject()
	r, e := p.Calc()
	if e != nil {
		t.Fatal(e)
	}
	if !almost(r.SWN, 1.21) || !almost(r.SWD, 1.552) || !almost(r.WN, .24) || !almost(r.WD, .336) {
		t.Fatalf("wrong results: %#v", r)
	}
}
func TestProjectRoundtrip(t *testing.T) {
	p := initialProject()
	name := filepath.Join(t.TempDir(), "project.json")
	if e := p.Save(name); e != nil {
		t.Fatal(e)
	}
	q, e := LoadProject(name)
	if e != nil {
		t.Fatal(e)
	}
	if len(q.Layers) != 3 || q.Wind.C != .8 {
		t.Fatal(q)
	}
}
func TestExcelWord(t *testing.T) {
	d := t.TempDir()
	p := initialProject()
	x := filepath.Join(d, "sample.xlsx")
	w := filepath.Join(d, "sample.docx")
	if e := ExportExcel(p, x); e != nil {
		t.Fatal(e)
	}
	if e := ExportWord(p, w); e != nil {
		t.Fatal(e)
	}
	checks := []struct{ filename, part, text string }{{x, "xl/worksheets/sheet1.xml", "B3*C3"}, {x, "xl/worksheets/sheet1.xml", "SUM(D3:D5)"}, {x, "xl/worksheets/sheet2.xml", "B3*B4*B5"}, {w, "word/document.xml", "ПОВЕРОЧНЫЙ"}, {w, "word/document.xml", "<m:oMath>"}}
	for _, c := range checks {
		z, e := zip.OpenReader(c.filename)
		if e != nil {
			t.Fatal(e)
		}
		found := false
		for _, f := range z.File {
			if f.Name == c.part {
				r, e := f.Open()
				if e != nil {
					t.Fatal(e)
				}
				b := make([]byte, f.UncompressedSize64)
				_, e = r.Read(b)
				r.Close()
				if e != nil && e.Error() != "EOF" {
					t.Fatal(e)
				}
				if !strings.Contains(string(b), c.text) {
					t.Errorf("%s no %q", c.part, c.text)
				}
				found = true
				break
			}
		}
		z.Close()
		if !found {
			t.Errorf("%s missing", c.part)
		}
	}
	if _, e := os.Stat(x); e != nil {
		t.Fatal(e)
	}
}

// XLSX's styles.xml has a fixed ECMA-376 element order. Excel rejects
// style tables when numFmts appears after fonts/fills/borders.
func TestStylesElementOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "check.xlsx")
	if err := ExportExcel(initialProject(), path); err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var styles string
	for _, f := range z.File {
		if f.Name != "xl/styles.xml" {
			continue
		}
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		styles = string(b)
	}
	if styles == "" {
		t.Fatal("no styles.xml")
	}
	names := []string{"<numFmts", "<fonts", "<fills", "<borders", "<cellStyleXfs", "<cellXfs", "<cellStyles"}
	prev := -1
	for _, name := range names {
		pos := strings.Index(styles, name)
		if pos <= prev {
			t.Fatalf("bad XLSX style order at %s", name)
		}
		prev = pos
	}
}

func TestUnitsAreReversible(t *testing.T) {
	for _, v := range []float64{0, 0.3, 1.21, 3.75, 100, 1234.56789} {
		back := kgfM2ToKpa(kpaToKgfM2(v))
		if math.Abs(back-v) > 1e-10 {
			t.Fatalf("unit roundtrip: %.12f != %.12f", back, v)
		}
	}
	if !almost(kpaToKgfM2(9.80665), 1000) {
		t.Fatal("9.80665 kPa must be 1000 kgf/m2")
	}
}

func TestNewExportsContainUnitFormulas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "units.xlsx")
	if err := ExportExcel(initialProject(), path); err != nil {
		t.Fatal(err)
	}
	f, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, tc := range []struct{ part, token string }{
		{"xl/worksheets/sheet1.xml", "D3*$J$3"},
		{"xl/worksheets/sheet1.xml", "F3*$J$3"},
		{"xl/worksheets/sheet2.xml", "B3*$F$3"},
		{"xl/worksheets/sheet2.xml", "B7*$F$3"},
		{"xl/worksheets/sheet2.xml", "B8*$F$3"},
	} {
		var found bool
		for _, z := range f.File {
			if z.Name != tc.part {
				continue
			}
			b, err := z.Open()
			if err != nil {
				t.Fatal(err)
			}
			content, err := io.ReadAll(b)
			b.Close()
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(content), tc.token) {
				t.Fatalf("missing %q in %s", tc.token, tc.part)
			}
			found = true
		}
		if !found {
			t.Fatalf("missing %s", tc.part)
		}
	}
}
