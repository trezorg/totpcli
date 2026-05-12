package search

import "strings"

// Normalize lowercases and trims whitespace for search matching.
func Normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
