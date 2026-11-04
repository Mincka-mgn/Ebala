package main

import (
	"strings"
)

func tempCraneKind(s TempSelection) string {
	if s.Params != nil {
		if name, ok := s.Params["crane_kind"].(string); ok {
			return strings.TrimSpace(name)
		}
	}
	return ""
}
