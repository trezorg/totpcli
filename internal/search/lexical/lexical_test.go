package lexical

import (
	"testing"

	"github.com/trezorg/totpcli/internal/search/searchtest"
)

func TestSearchReturnsAllProvidersForBlankQuery(t *testing.T) {
	t.Parallel()

	providers := searchtest.SampleProviders()
	got := New(providers).Search("")

	searchtest.AssertLabels(t, got, []string{"GitHub/trezorg", "AWS/root", "GitLab"})
}

func TestSearchMatchesIssuerNameAndDisplayLabelCaseInsensitively(t *testing.T) {
	t.Parallel()

	providers := searchtest.SampleProviders()
	tests := map[string][]string{
		"git":            {"GitHub/trezorg", "GitLab"},
		"ZORG":           {"GitHub/trezorg"},
		"github/trezorg": {"GitHub/trezorg"},
	}

	for query, want := range tests {
		t.Run(query, func(t *testing.T) {
			t.Parallel()

			got := New(providers).Search(query)
			searchtest.AssertLabels(t, got, want)
		})
	}
}

func TestSearchExcludesNonMatches(t *testing.T) {
	t.Parallel()

	got := New(searchtest.SampleProviders()).Search("missing")
	if len(got) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(got))
	}
}
