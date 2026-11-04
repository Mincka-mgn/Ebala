package main

import "strings"

// formatSitePlace never emits separators for empty geographic levels.
func formatSitePlace(parts ...string) string {
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.Trim(part, ","))
		if part != "" {
			result = append(result, part)
		}
	}
	return strings.Join(result, ", ")
}
