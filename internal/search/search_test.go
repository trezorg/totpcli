package search

import (
	"testing"

	"github.com/trezorg/totpcli/internal/provider"
)

func TestLexicalSearchReturnsAllProvidersForBlankQuery(t *testing.T) {
	t.Parallel()

	providers := sampleProviders()
	got := NewLexical(providers).Search("")

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

			got := NewLexical(providers).Search(query)
			assertLabels(t, got, want)
		})
	}
}

func TestLexicalSearchExcludesNonMatches(t *testing.T) {
	t.Parallel()

	got := NewLexical(sampleProviders()).Search("missing")
	if len(got) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(got))
	}
}

func TestDigramSearchMatchesIssuerPrefix(t *testing.T) {
	t.Parallel()

	got := NewDigram(sampleProviders()).Search("git")

	assertLabels(t, got, []string{"GitHub/trezorg", "GitLab"})
}

func TestDigramSearchMatchesNameSubstring(t *testing.T) {
	t.Parallel()

	got := NewDigram(sampleProviders()).Search("zorg")

	assertLabels(t, got, []string{"GitHub/trezorg"})
}

func TestDigramSearchMatchesDisplayLabel(t *testing.T) {
	t.Parallel()

	got := NewDigram(sampleProviders()).Search("hub/tre")

	assertLabels(t, got, []string{"GitHub/trezorg"})
}

func TestDigramSearchExcludesNonMatches(t *testing.T) {
	t.Parallel()

	got := NewDigram(sampleProviders()).Search("missing")

	if len(got) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(got))
	}
}

func TestDigramSearchReturnsAllProvidersForBlankQuery(t *testing.T) {
	t.Parallel()

	got := NewDigram(sampleProviders()).Search(" \t\n")

	assertLabels(t, got, []string{"GitHub/trezorg", "AWS/root", "GitLab"})
}

func TestDigramSearchPreservesProviderOrder(t *testing.T) {
	t.Parallel()

	got := NewDigram([]provider.Provider{
		{Issuer: "FirstGit"},
		{Issuer: "Other"},
		{Issuer: "SecondGit"},
		{Issuer: "ThirdGit"},
	}).Search("git")

	assertLabels(t, got, []string{"FirstGit", "SecondGit", "ThirdGit"})
}

func TestNewEngineDefaultsToDigram(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine("", sampleProviders())
	if err != nil {
		t.Fatalf("NewEngine returned error: %v", err)
	}

	got := engine.Search("hub/tre")

	assertLabels(t, got, []string{"GitHub/trezorg"})
}

func TestParseEngineNameDefaultsToDigram(t *testing.T) {
	t.Parallel()

	name, err := ParseEngineName("")
	if err != nil {
		t.Fatalf("ParseEngineName returned error: %v", err)
	}
	if name != EngineDigram {
		t.Fatalf("ParseEngineName = %q, want %q", name, EngineDigram)
	}
}

func TestParseEngineNameRejectsUnsupportedName(t *testing.T) {
	t.Parallel()

	_, err := ParseEngineName("unknown")
	if err == nil {
		t.Fatal("ParseEngineName error = nil, want unsupported engine error")
	}
}

func TestNewEngineSelectsLexical(t *testing.T) {
	t.Parallel()

	engine, err := NewEngine(" lexical ", sampleProviders())
	if err != nil {
		t.Fatalf("NewEngine returned error: %v", err)
	}

	got := engine.Search("zorg")

	assertLabels(t, got, []string{"GitHub/trezorg"})
}

func TestNewEngineRejectsUnsupportedName(t *testing.T) {
	t.Parallel()

	_, err := NewEngine("unknown", sampleProviders())
	if err == nil {
		t.Fatal("NewEngine error = nil, want unsupported engine error")
	}
}

func TestSearchEngineInterfaceIsReplaceable(t *testing.T) {
	t.Parallel()

	var engine Engine = staticEngine{results: []provider.Provider{{Issuer: "Only"}}}
	got := engine.Search("ignored")

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

func (s staticEngine) Search(string) []provider.Provider {
	return s.results
}
