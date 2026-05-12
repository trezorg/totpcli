package digram

import (
	"testing"

	"github.com/trezorg/totpcli/internal/provider"
	"github.com/trezorg/totpcli/internal/search/searchtest"
)

func TestSearchMatchesIssuerPrefix(t *testing.T) {
	t.Parallel()

	got := New(searchtest.SampleProviders()).Search("git")

	searchtest.AssertLabels(t, got, []string{"GitHub/trezorg", "GitLab"})
}

func TestSearchMatchesNameSubstring(t *testing.T) {
	t.Parallel()

	got := New(searchtest.SampleProviders()).Search("zorg")

	searchtest.AssertLabels(t, got, []string{"GitHub/trezorg"})
}

func TestSearchMatchesDisplayLabel(t *testing.T) {
	t.Parallel()

	got := New(searchtest.SampleProviders()).Search("hub/tre")

	searchtest.AssertLabels(t, got, []string{"GitHub/trezorg"})
}

func TestSearchExcludesNonMatches(t *testing.T) {
	t.Parallel()

	got := New(searchtest.SampleProviders()).Search("missing")

	if len(got) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(got))
	}
}

func TestSearchReturnsAllProvidersForBlankQuery(t *testing.T) {
	t.Parallel()

	got := New(searchtest.SampleProviders()).Search(" \t\n")

	searchtest.AssertLabels(t, got, []string{"GitHub/trezorg", "AWS/root", "GitLab"})
}

func TestSearchPreservesProviderOrder(t *testing.T) {
	t.Parallel()

	got := New([]provider.Provider{
		{Issuer: "FirstGit"},
		{Issuer: "Other"},
		{Issuer: "SecondGit"},
		{Issuer: "ThirdGit"},
	}).Search("git")

	searchtest.AssertLabels(t, got, []string{"FirstGit", "SecondGit", "ThirdGit"})
}
