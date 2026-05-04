package lexical

import (
	"strings"

	"github.com/trezorg/totpcli/internal/provider"
)

type Engine struct {
	providers []provider.Provider
}

func New(providers []provider.Provider) Engine {
	return Engine{providers: append([]provider.Provider(nil), providers...)}
}

func (e Engine) Search(query string) []provider.Provider {
	query = normalize(query)
	if query == "" {
		return append([]provider.Provider(nil), e.providers...)
	}

	results := make([]provider.Provider, 0, len(e.providers))
	for _, candidate := range e.providers {
		if matches(query, candidate) {
			results = append(results, candidate)
		}
	}
	return results
}

func matches(query string, candidate provider.Provider) bool {
	return strings.Contains(normalize(candidate.Issuer), query) ||
		strings.Contains(normalize(candidate.Name), query) ||
		strings.Contains(normalize(candidate.DisplayLabel()), query)
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
