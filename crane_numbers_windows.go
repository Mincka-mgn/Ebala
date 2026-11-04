//go:build windows

package main

import "fmt"

func tempNextCraneNumber() int {
	maxN := 0
	for _, s := range tempSections.Sections {
		if !tempIsCrane(s) {
			continue
		}
		var number int
		if n, err := fmt.Sscanf(s.Title, "Кран %d", &number); err == nil && n == 1 && number > maxN {
			maxN = number
		}
	}
	if maxN < 0 {
		maxN = 0
	}
	return maxN + 1
}
