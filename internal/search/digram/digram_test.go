package digram

import (
	"testing"

	"github.com/trezorg/totpcli/internal/provider"
)

func TestSearchMatchesIssuerPrefix(t *testing.T) {
	t.Parallel()

	got := New(sampleProviders()).Search("git")

	assertLabels(t, got, []string{"GitHub/trezorg", "GitLab"})
}

func TestSearchMatchesNameSubstring(t *testing.T) {
	t.Parallel()

	got := New(sampleProviders()).Search("zorg")

	assertLabels(t, got, []string{"GitHub/trezorg"})
}

func TestSearchMatchesDisplayLabel(t *testing.T) {
	t.Parallel()

	got := New(sampleProviders()).Search("hub/tre")

	assertLabels(t, got, []string{"GitHub/trezorg"})
}

func TestSearchExcludesNonMatches(t *testing.T) {
	t.Parallel()

	got := New(sampleProviders()).Search("missing")

	if len(got) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(got))
	}
}

func TestSearchReturnsAllProvidersForBlankQuery(t *testing.T) {
	t.Parallel()

	got := New(sampleProviders()).Search(" \t\n")

	assertLabels(t, got, []string{"GitHub/trezorg", "AWS/root", "GitLab"})
}

func TestSearchPreservesProviderOrder(t *testing.T) {
	t.Parallel()

	got := New([]provider.Provider{
		{Issuer: "FirstGit"},
		{Issuer: "Other"},
		{Issuer: "SecondGit"},
		{Issuer: "ThirdGit"},
	}).Search("git")

	assertLabels(t, got, []string{"FirstGit", "SecondGit", "ThirdGit"})
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
