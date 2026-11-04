package main

import "testing"

func TestFilteredAndSpecialClimate(t *testing.T) {
	a := matchingRecords(climateRecords(), "Туапсе")
	if len(a) != 0 {
		t.Fatalf("empty place not excluded: %+v", a)
	}
	a = matchingRecords(climateRecords(), "Казань")
	if len(a) != 1 || a[0].SnowK != 2.3 {
		t.Fatalf("special Kazan: %+v", a)
	}
	s := climateFromRecord(a[0])
	if snowChoice(s) != snowOptionK || s.SnowZone != "IV" {
		t.Fatal(s)
	}
	for _, tc := range []struct {
		zone  string
		value float64
	}{{"I", .5}, {"IV", 2}, {"VIII", 4}} {
		v, ok := snowValueForChoice(tc.zone, s)
		if !ok || v != tc.value {
			t.Fatal(tc, v)
		}
	}
	if v, ok := windForZone(windInternalZone("Iа")); !ok || v != .17 {
		t.Fatal(v, ok)
	}
}
