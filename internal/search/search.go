package search

import (
	"strings"

	"github.com/trezorg/totpcli/internal/provider"
)

type Engine interface {
	Search(query string, providers []provider.Provider) []provider.Provider
}

type Lexical struct{}

func (Lexical) Search(query string, providers []provider.Provider) []provider.Provider {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return append([]provider.Provider(nil), providers...)
	}

	results := make([]provider.Provider, 0, len(providers))
	for _, candidate := range providers {
		if matches(query, candidate) {
			results = append(results, candidate)
		}
	}
	return results
}

func matches(query string, candidate provider.Provider) bool {
	return strings.Contains(strings.ToLower(candidate.Issuer), query) ||
		strings.Contains(strings.ToLower(candidate.Name), query) ||
		strings.Contains(strings.ToLower(candidate.DisplayLabel()), query)
}
