package search

import (
	"testing"

	"github.com/trezorg/totpcli/internal/provider"
)

func TestLexicalSearchReturnsAllProvidersForBlankQuery(t *testing.T) {
	t.Parallel()

	providers := sampleProviders()
	got := Lexical{}.Search("", providers)

	assertLabels(t, got, []string{"GitHub/trezorg", "AWS/root", "GitLab"})
}

func TestLexicalSearchMatchesIssuerNameAndDisplayLabelCaseInsensitively(t *testing.T) {
	t.Parallel()

	providers := sampleProviders()
	tests := map[string][]string{
		"git":            {"GitHub/trezorg", "GitLab"},
		"ZORG":           {"GitHub/trezorg"},
		"github/trezorg": {"GitHub/trezorg"},
	}

	for query, want := range tests {
		query := query
		want := want
		t.Run(query, func(t *testing.T) {
			t.Parallel()

			got := Lexical{}.Search(query, providers)
			assertLabels(t, got, want)
		})
	}
}

func TestLexicalSearchExcludesNonMatches(t *testing.T) {
	t.Parallel()

	got := Lexical{}.Search("missing", sampleProviders())
	if len(got) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(got))
	}
}

func TestSearchEngineInterfaceIsReplaceable(t *testing.T) {
	t.Parallel()

	var engine Engine = staticEngine{results: []provider.Provider{{Issuer: "Only"}}}
	got := engine.Search("ignored", sampleProviders())

	assertLabels(t, got, []string{"Only"})
}

func sampleProviders() []provider.Provider {
	return []provider.Provider{
		{Issuer: "GitHub", Name: "trezorg"},
		{Issuer: "AWS", Name: "root"},
		{Issuer: "GitLab"},
	}
}

func assertLabels(t *testing.T, providers []provider.Provider, want []string) {
	t.Helper()

	if len(providers) != len(want) {
		t.Fatalf("len(providers) = %d, want %d", len(providers), len(want))
	}
	for i := range want {
		if got := providers[i].DisplayLabel(); got != want[i] {
			t.Fatalf("providers[%d] = %q, want %q", i, got, want[i])
		}
	}
}

type staticEngine struct {
	results []provider.Provider
}

func (s staticEngine) Search(string, []provider.Provider) []provider.Provider {
	return s.results
}
