package searchtest

import (
	"testing"

	"github.com/trezorg/totpcli/internal/provider"
)

func SampleProviders() []provider.Provider {
	return []provider.Provider{
		{Issuer: "GitHub", Name: "trezorg"},
		{Issuer: "AWS", Name: "root"},
		{Issuer: "GitLab"},
	}
}

func AssertLabels(t *testing.T, providers []provider.Provider, want []string) {
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
