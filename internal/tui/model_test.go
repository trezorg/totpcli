package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/trezorg/totpcli/internal/provider"
	"github.com/trezorg/totpcli/internal/search"
	"github.com/trezorg/totpcli/internal/totpcode"
)

func init() {
	lipgloss.SetColorProfile(termenv.TrueColor)
}

func TestNewModelInitializesSearchFromCLIValue(t *testing.T) {
	t.Parallel()

	model := newTestModel("git")

	if model.SearchValue() != "git" {
		t.Fatalf("SearchValue = %q, want git", model.SearchValue())
	}
	assertLabels(t, model.Filtered(), []string{"GitHub/trezorg", "GitLab"})
}

func TestSetSearchUpdatesFilteredResults(t *testing.T) {
	t.Parallel()

	model := newTestModel("")
	model.SetSearch("aws")

	assertLabels(t, model.Filtered(), []string{"AWS/root"})

	model.SetSearch("")
	assertLabels(t, model.Filtered(), []string{"GitHub/trezorg", "AWS/root", "GitLab"})
}

func TestViewRendersProviderLabels(t *testing.T) {
	t.Parallel()

	view := newTestModel("").View()
	for _, want := range []string{"GitHub/trezorg", "AWS/root", "GitLab"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing provider label %q:\n%s", want, view)
		}
	}
}

func TestViewStylesProviderList(t *testing.T) {
	t.Parallel()

	view := newTestModel("").View()
	if !strings.Contains(view, "\x1b[") {
		t.Fatalf("View() missing ANSI styling:\n%s", view)
	}
}

func TestSelectHighlightedProviderGeneratesCode(t *testing.T) {
	t.Parallel()

	generator := &stubGenerator{results: []totpcode.Result{{Code: "123456", Remaining: 17}}}
	model := NewModel(sampleProviders(), search.Lexical{}, "", generator, fixedNow)
	model.MoveDown()

	if ok := model.Select(); !ok {
		t.Fatal("Select returned false, want true")
	}

	if got := model.Selected().DisplayLabel(); got != "AWS/root" {
		t.Fatalf("Selected = %q, want AWS/root", got)
	}
	if model.Code() != "123456" {
		t.Fatalf("Code = %q, want 123456", model.Code())
	}
	if model.Remaining() != 17 {
		t.Fatalf("Remaining = %d, want 17", model.Remaining())
	}
}

func TestViewStylesGeneratedCode(t *testing.T) {
	t.Parallel()

	model := newTestModel("")
	model.Select()

	view := model.View()
	for _, want := range []string{"GitHub/trezorg", "Code:", "123456", "Remaining:", "17s"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q:\n%s", want, view)
		}
	}
	if !strings.Contains(view, "\x1b[") {
		t.Fatalf("View() missing ANSI styling:\n%s", view)
	}
}

func TestSelectWithNoResultsDoesNotGenerateCode(t *testing.T) {
	t.Parallel()

	generator := &stubGenerator{results: []totpcode.Result{{Code: "123456", Remaining: 17}}}
	model := NewModel(sampleProviders(), search.Lexical{}, "", generator, fixedNow)
	model.SetSearch("missing")

	if ok := model.Select(); ok {
		t.Fatal("Select returned true for empty results")
	}
	if generator.calls != 0 {
		t.Fatalf("generator calls = %d, want 0", generator.calls)
	}
}

func TestViewStylesEmptyResults(t *testing.T) {
	t.Parallel()

	model := newTestModel("")
	model.SetSearch("missing")

	view := model.View()
	if !strings.Contains(view, "No providers found.") {
		t.Fatalf("View() missing empty state:\n%s", view)
	}
	if !strings.Contains(view, "\x1b[") {
		t.Fatalf("View() missing ANSI styling:\n%s", view)
	}
}

func TestRefreshUpdatesSelectedProviderCode(t *testing.T) {
	t.Parallel()

	generator := &stubGenerator{
		results: []totpcode.Result{
			{Code: "111111", Remaining: 1},
			{Code: "222222", Remaining: 30},
		},
	}
	model := NewModel(sampleProviders(), search.Lexical{}, "", generator, fixedNow)
	model.Select()
	model.Refresh()

	if model.Code() != "222222" {
		t.Fatalf("Code = %q, want refreshed code", model.Code())
	}
	if model.Remaining() != 30 {
		t.Fatalf("Remaining = %d, want 30", model.Remaining())
	}
}

func TestGenerationErrorIsDisplayed(t *testing.T) {
	t.Parallel()

	model := NewModel(sampleProviders(), search.Lexical{}, "", &stubGenerator{err: errors.New("bad secret")}, fixedNow)
	model.Select()

	if model.Err() == nil {
		t.Fatal("Err = nil, want generation error")
	}
	if !strings.Contains(model.View(), "bad secret") {
		t.Fatalf("View() missing error:\n%s", model.View())
	}
}

func TestViewStylesGenerationError(t *testing.T) {
	t.Parallel()

	model := NewModel(sampleProviders(), search.Lexical{}, "", &stubGenerator{err: errors.New("bad secret")}, fixedNow)
	model.Select()

	view := model.View()
	if !strings.Contains(view, "bad secret") {
		t.Fatalf("View() missing error:\n%s", view)
	}
	if !strings.Contains(view, "\x1b[") {
		t.Fatalf("View() missing ANSI styling:\n%s", view)
	}
}

func newTestModel(initialSearch string) Model {
	return NewModel(sampleProviders(), search.Lexical{}, initialSearch, &stubGenerator{
		results: []totpcode.Result{{Code: "123456", Remaining: 17}},
	}, fixedNow)
}

func sampleProviders() []provider.Provider {
	return []provider.Provider{
		{Issuer: "GitHub", Name: "trezorg", Algorithm: provider.AlgorithmSHA1, Digits: 6, TOTPSecret: "JBSWY3DPEHPK3PXP"},
		{Issuer: "AWS", Name: "root", Algorithm: provider.AlgorithmSHA1, Digits: 6, TOTPSecret: "JBSWY3DPEHPK3PXP"},
		{Issuer: "GitLab", Algorithm: provider.AlgorithmSHA1, Digits: 6, TOTPSecret: "JBSWY3DPEHPK3PXP"},
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

func fixedNow() time.Time {
	return time.Unix(30, 0)
}

type stubGenerator struct {
	results []totpcode.Result
	err     error
	calls   int
}

func (s *stubGenerator) Generate(provider.Provider, time.Time) (totpcode.Result, error) {
	s.calls++
	if s.err != nil {
		return totpcode.Result{}, s.err
	}
	if len(s.results) == 0 {
		return totpcode.Result{}, nil
	}
	result := s.results[0]
	s.results = s.results[1:]
	return result, nil
}
