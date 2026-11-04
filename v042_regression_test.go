package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func Test042DefaultsAndReportBlocks(t *testing.T) {
	if s := defaultSnowCalc(); s.Region != "VIII" || s.Ground != 4 {
		t.Fatalf("snow default %+v", s)
	}
	if s := defaultIceLoad(); s.Region != "IV" || s.BaseMM != 15 {
		t.Fatalf("ice default %+v", s)
	}
	p := initialProject()
	p.Weight = &WeightCalc{Rows: []WeightRow{{Name: "Пол", Kind: "direct", GNorm: 300, Gamma: 1.1}}}
	p.initReportBlocks(nil)
	if len(p.SavedBlocks) != 1 || p.SavedBlocks[0].WeightCalc == nil {
		t.Fatalf("weight construction duplicated: %+v", p.SavedBlocks)
	}
	a := defaultSnowCalc()
	a.Enabled = true
	id := p.putSnowCalc(0, a)
	a.Ground = 3
	if p.putSnowCalc(id, a) != id || len(p.SavedBlocks) != 2 {
		t.Fatal("repeat calculation made duplicate")
	}
	p.renameBlock(id, "")
	if p.SavedBlocks[1].Title != "" {
		t.Fatal("empty report name not preserved")
	}
	next := p.putSnowCalc(0, defaultSnowCalc())
	if len(p.SavedBlocks) != 3 || next == id {
		t.Fatal("second snow load not created independently")
	}
	path := filepath.Join(t.TempDir(), "project.json")
	if err := p.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadProject(path)
	if err != nil || len(loaded.SavedBlocks) != 3 || loaded.SavedBlocks[1].Title != "" {
		t.Fatalf("persistence: %v", err)
	}
}
func Test042LockFallback(t *testing.T) {
	p := filepath.Join(t.TempDir(), "Расчёт.xlsx")
	calls := 0
	saved, err := writeWithLockFallback(p, func(file string) error {
		calls++
		if file == p {
			return &os.PathError{Op: "open", Path: file, Err: syscall.Errno(32)}
		}
		return os.WriteFile(file, []byte("test"), 0644)
	})
	if err != nil || saved != numberedFileName(p, 1) || calls != 2 {
		t.Fatalf("fallback %q %d %v", saved, calls, err)
	}
	saved, err = writeWithLockFallback(p, func(file string) error { return os.WriteFile(file, []byte("replaced"), 0644) })
	if err != nil || saved != p {
		t.Fatal("ordinary existing file should be overwritten")
	}
	bad := errors.New("read-only directory")
	_, err = writeWithLockFallback(p, func(file string) error { return bad })
	if !errors.Is(err, bad) {
		t.Fatal("cannot hide non-lock errors")
	}
}
func Test042CompactReservoirTable(t *testing.T) {
	s := defaultSnowCalc()
	s.Shape = "cone"
	s.Angle = 25
	s.Ground = 2
	s.Span = 20
	s, err := SnowCompute(s)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range s.Cases {
		// The historic expectation concerns the B.12 drift field, not the
		// separate partial 1/2 and 1/4-area patterns added under p. 10.4.
		if len(c.Polar) == 0 || !strings.HasPrefix(c.Title, "2.") {
			continue
		}
		found = true
		lines := snowWordReservoirRows(s, c)
		if len(lines) < 15 || len(lines) > 55 {
			t.Fatalf("compact rows=%d", len(lines))
		}
		if lines[0][1] != "0…360" || lines[0][2] != "0" {
			t.Fatalf("center: %v", lines[0])
		}
		if !strings.Contains(lines[len(lines)-1][1], "195…360") {
			t.Fatalf("zero sector missing: %v", lines[len(lines)-1])
		}
	}
	if !found {
		t.Fatal("no polar snow variant")
	}
}
