package lexical

import (
	"testing"

	"github.com/trezorg/totpcli/internal/provider"
)

func TestSearchReturnsAllProvidersForBlankQuery(t *testing.T) {
	t.Parallel()

	providers := sampleProviders()
	got := New(providers).Search("")

	assertLabels(t, got, []string{"GitHub/trezorg", "AWS/root", "GitLab"})
}

func TestSearchMatchesIssuerNameAndDisplayLabelCaseInsensitively(t *testing.T) {
	t.Parallel()

	providers := sampleProviders()
	tests := map[string][]string{
		"git":            {"GitHub/trezorg", "GitLab"},
		"ZORG":           {"GitHub/trezorg"},
		"github/trezorg": {"GitHub/trezorg"},
	}

	for query, want := range tests {
		t.Run(query, func(t *testing.T) {
			t.Parallel()

			got := New(providers).Search(query)
			assertLabels(t, got, want)
		})
	}
}

func TestSearchExcludesNonMatches(t *testing.T) {
	t.Parallel()

	got := New(sampleProviders()).Search("missing")
	if len(got) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(got))
	}
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
