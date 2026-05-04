package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/trezorg/totpcli/internal/provider"
	"github.com/trezorg/totpcli/internal/search/lexical"
	"github.com/trezorg/totpcli/internal/totpcode"
	"github.com/trezorg/totpcli/internal/tui"
)

func TestRunLoadsProvidersAndLaunchesTUIWithSearch(t *testing.T) {
	t.Parallel()

	path := writeProviderFile(t, `[
  {
    "name": "trezorg",
    "issuer": "GitHub",
    "algorithm": "SHA1",
    "digits": "SIX",
    "type": "TOTP",
    "totpSecret": "JBSWY3DPEHPK3PXP"
  },
  {
    "issuer": "AWS",
    "algorithm": "SHA1",
    "digits": "SIX",
    "type": "TOTP",
    "totpSecret": "JBSWY3DPEHPK3PXP"
  }
]`, "providers.json")

	var launched tui.Model
	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		RunTUI: func(model tui.Model) error {
			launched = model
			return nil
		},
	}

	code := app.Run([]string{"--totp", path, "--search", "git"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if launched.SearchValue() != "git" {
		t.Fatalf("SearchValue = %q, want git", launched.SearchValue())
	}
	filtered := launched.Filtered()
	if len(filtered) != 1 || filtered[0].DisplayLabel() != "GitHub/trezorg" {
		t.Fatalf("filtered = %#v, want GitHub/trezorg only", filtered)
	}
}

func TestRunUsesEnvironmentProviderPath(t *testing.T) {
	path := writeProviderFile(t, singleProviderJSON(), "providers.json")
	t.Setenv("TOTPCLI_TOTP_PATH", path)

	var launched tui.Model
	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		RunTUI: func(model tui.Model) error {
			launched = model
			return nil
		},
	}

	code := app.Run(nil)
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	assertLaunchedLabels(t, launched, []string{"GitHub/trezorg"})
}

func TestRunUsesEnvironmentSearch(t *testing.T) {
	path := writeProviderFile(t, `[
  {
    "name": "trezorg",
    "issuer": "GitHub",
    "algorithm": "SHA1",
    "digits": "SIX",
    "type": "TOTP",
    "totpSecret": "JBSWY3DPEHPK3PXP"
  },
  {
    "issuer": "AWS",
    "algorithm": "SHA1",
    "digits": "SIX",
    "type": "TOTP",
    "totpSecret": "JBSWY3DPEHPK3PXP"
  }
]`, "providers.json")
	t.Setenv("TOTPCLI_SEARCH", "aws")

	var launched tui.Model
	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		RunTUI: func(model tui.Model) error {
			launched = model
			return nil
		},
	}

	code := app.Run([]string{"--totp", path})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if launched.SearchValue() != "aws" {
		t.Fatalf("SearchValue = %q, want aws", launched.SearchValue())
	}
	assertLaunchedLabels(t, launched, []string{"AWS"})
}

func TestRunSearchEngineFlagSelectsEngine(t *testing.T) {
	path := writeProviderFile(t, singleProviderJSON(), "providers.json")

	var engineName string
	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		NewSearchEngine: func(name string, providers []provider.Provider) (tui.SearchEngine, error) {
			engineName = name
			return lexical.New(providers), nil
		},
		RunTUI: func(tui.Model) error {
			return nil
		},
	}

	code := app.Run([]string{"--totp", path, "--search-engine", "lexical"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if engineName != "lexical" {
		t.Fatalf("search engine = %q, want lexical", engineName)
	}
}

func TestRunUsesEnvironmentSearchEngine(t *testing.T) {
	path := writeProviderFile(t, singleProviderJSON(), "providers.json")
	t.Setenv("TOTPCLI_SEARCH_ENGINE", "lexical")

	var engineName string
	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		NewSearchEngine: func(name string, providers []provider.Provider) (tui.SearchEngine, error) {
			engineName = name
			return lexical.New(providers), nil
		},
		RunTUI: func(tui.Model) error {
			return nil
		},
	}

	code := app.Run([]string{"--totp", path})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if engineName != "lexical" {
		t.Fatalf("search engine = %q, want lexical", engineName)
	}
}

func TestRunFlagsOverrideEnvironment(t *testing.T) {
	envPath := writeProviderFile(t, singleProviderJSON(), "env.json")
	flagPath := writeProviderFile(t, `[
  {
    "name": "root",
    "issuer": "AWS",
    "algorithm": "SHA1",
    "digits": "SIX",
    "type": "TOTP",
    "totpSecret": "JBSWY3DPEHPK3PXP"
  }
]`, "flag.json")
	t.Setenv("TOTPCLI_TOTP_PATH", envPath)
	t.Setenv("TOTPCLI_SEARCH", "git")

	var launched tui.Model
	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		RunTUI: func(model tui.Model) error {
			launched = model
			return nil
		},
	}

	code := app.Run([]string{"--totp", flagPath, "--search", "aws"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if launched.SearchValue() != "aws" {
		t.Fatalf("SearchValue = %q, want aws", launched.SearchValue())
	}
	assertLaunchedLabels(t, launched, []string{"AWS/root"})
}

func TestRunSearchEngineFlagOverridesEnvironment(t *testing.T) {
	path := writeProviderFile(t, singleProviderJSON(), "providers.json")
	t.Setenv("TOTPCLI_SEARCH_ENGINE", "lexical")

	var engineName string
	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		NewSearchEngine: func(name string, providers []provider.Provider) (tui.SearchEngine, error) {
			engineName = name
			return lexical.New(providers), nil
		},
		RunTUI: func(tui.Model) error {
			return nil
		},
	}

	code := app.Run([]string{"--totp", path, "--search-engine", "digram"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if engineName != "digram" {
		t.Fatalf("search engine = %q, want digram", engineName)
	}
}

func TestRunDefaultSearchEngineIsDigram(t *testing.T) {
	path := writeProviderFile(t, singleProviderJSON(), "providers.json")

	var engineName string
	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		NewSearchEngine: func(name string, providers []provider.Provider) (tui.SearchEngine, error) {
			engineName = name
			return lexical.New(providers), nil
		},
		RunTUI: func(tui.Model) error {
			return nil
		},
	}

	code := app.Run([]string{"--totp", path})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if engineName != "digram" {
		t.Fatalf("search engine = %q, want digram", engineName)
	}
}

func TestRunNoTUIFlagSkipsTUI(t *testing.T) {
	path := writeProviderFile(t, singleProviderJSON(), "providers.json")

	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Now:    fixedNow,
		RunTUI: func(tui.Model) error {
			t.Fatal("RunTUI called in no-tui mode")
			return nil
		},
	}

	code := app.Run([]string{"--totp", path, "--no-tui"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
}

func TestRunUsesEnvironmentNoTUI(t *testing.T) {
	path := writeProviderFile(t, singleProviderJSON(), "providers.json")
	t.Setenv("TOTPCLI_NO_TUI", "true")

	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Now:    fixedNow,
		RunTUI: func(tui.Model) error {
			t.Fatal("RunTUI called when TOTPCLI_NO_TUI enables no-tui mode")
			return nil
		},
	}

	code := app.Run([]string{"--totp", path})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
}

func TestRunNoTUIFlagOverridesDisabledEnvironment(t *testing.T) {
	path := writeProviderFile(t, singleProviderJSON(), "providers.json")
	t.Setenv("TOTPCLI_NO_TUI", "false")

	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Now:    fixedNow,
		RunTUI: func(tui.Model) error {
			t.Fatal("RunTUI called when --no-tui overrides environment")
			return nil
		},
	}

	code := app.Run([]string{"--totp", path, "--no-tui"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
}

func TestRunReportsInvalidNoTUIEnvironmentBeforeLoadingProviders(t *testing.T) {
	t.Setenv("TOTPCLI_NO_TUI", "sometimes")

	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

	code := app.Run([]string{"--totp", filepath.Join(t.TempDir(), "missing.json")})
	if code != 2 {
		t.Fatalf("Run exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unsupported boolean") {
		t.Fatalf("stderr = %q, want unsupported boolean error", stderr.String())
	}
	if strings.Contains(stderr.String(), "read provider file") {
		t.Fatalf("stderr = %q, want validation before provider loading", stderr.String())
	}
}

func TestRunUsesEnvironmentFormat(t *testing.T) {
	path := writeProviderFile(t, `[[providers]]
name = "trezorg"
issuer = "GitHub"
algorithm = "SHA1"
digits = "SIX"
type = "TOTP"
totpSecret = "JBSWY3DPEHPK3PXP"
`, "providers.data")
	t.Setenv("TOTPCLI_FORMAT", "toml")

	var launched tui.Model
	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		RunTUI: func(model tui.Model) error {
			launched = model
			return nil
		},
	}

	code := app.Run([]string{"--totp", path})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	assertLaunchedLabels(t, launched, []string{"GitHub/trezorg"})
}

func TestRunDefaultPathFollowsEnvironmentFormat(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "totp.yaml")
	if err := os.WriteFile(path, []byte(`- name: trezorg
  issuer: GitHub
  algorithm: SHA1
  digits: SIX
  type: TOTP
  totpSecret: JBSWY3DPEHPK3PXP
`), 0o600); err != nil {
		t.Fatalf("write provider file: %v", err)
	}
	t.Setenv("TOTPCLI_FORMAT", "yaml")

	var launched tui.Model
	app := App{
		Home:   home,
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		RunTUI: func(model tui.Model) error {
			launched = model
			return nil
		},
	}

	code := app.Run(nil)
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	assertLaunchedLabels(t, launched, []string{"GitHub/trezorg"})
}

func TestRunHelpListsFlagsAndEnvironment(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &stdout, Stderr: &stderr}

	code := app.Run([]string{"--help"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}

	help := stdout.String()
	for _, want := range []string{
		"--totp",
		"--format",
		"--search",
		"--search-engine",
		"--no-tui",
		"TOTPCLI_TOTP_PATH",
		"TOTPCLI_FORMAT",
		"TOTPCLI_SEARCH",
		"TOTPCLI_SEARCH_ENGINE",
		"TOTPCLI_NO_TUI",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("help missing %q:\n%s", want, help)
		}
	}
}

func TestRunReportsUnknownFlagUsageError(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

	code := app.Run([]string{"--missing"})
	if code != 2 {
		t.Fatalf("Run exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("stderr = %q, want unknown flag error", stderr.String())
	}
}

func TestRunReportsInvalidFormat(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

	code := app.Run([]string{"--format", "xml"})
	if code == 0 {
		t.Fatal("Run exit code = 0, want error")
	}
	if !strings.Contains(stderr.String(), "unsupported format") {
		t.Fatalf("stderr = %q, want unsupported format error", stderr.String())
	}
}

func TestRunReportsInvalidSearchEngineBeforeLoadingProviders(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

	code := app.Run([]string{"--totp", filepath.Join(t.TempDir(), "missing.json"), "--search-engine", "unknown"})
	if code != 2 {
		t.Fatalf("Run exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unsupported search engine") {
		t.Fatalf("stderr = %q, want unsupported search engine error", stderr.String())
	}
	if strings.Contains(stderr.String(), "read provider file") {
		t.Fatalf("stderr = %q, want validation before provider loading", stderr.String())
	}
}

func TestRunReportsMissingProviderFile(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

	code := app.Run([]string{"--totp", filepath.Join(t.TempDir(), "missing.json")})
	if code == 0 {
		t.Fatal("Run exit code = 0, want error")
	}
	if !strings.Contains(stderr.String(), "read provider file") {
		t.Fatalf("stderr = %q, want provider file error", stderr.String())
	}
}

func TestRunNoTUIPrintsAllProviders(t *testing.T) {
	path := writeProviderFile(t, twoProviderJSON(), "providers.json")

	var stdout bytes.Buffer
	app := App{
		Home:   t.TempDir(),
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Now:    fixedNow,
	}

	code := app.Run([]string{"--totp", path, "--no-tui"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}

	assertTokenTable(t, stdout.String(), "GitHub/trezorg", "AWS")
}

func TestRunNoTUISearchPrintsFilteredProviders(t *testing.T) {
	path := writeProviderFile(t, twoProviderJSON(), "providers.json")

	var stdout bytes.Buffer
	app := App{
		Home:   t.TempDir(),
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Now:    fixedNow,
	}

	code := app.Run([]string{"--totp", path, "--no-tui", "--search", "git"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}

	assertTokenTable(t, stdout.String(), "GitHub/trezorg")
	if strings.Contains(stdout.String(), "AWS") {
		t.Fatalf("stdout = %q, want AWS filtered out", stdout.String())
	}
}

func TestRunNoTUIEmptySearchPrintsNoTokenRows(t *testing.T) {
	path := writeProviderFile(t, twoProviderJSON(), "providers.json")

	var stdout bytes.Buffer
	app := App{
		Home:   t.TempDir(),
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Now:    fixedNow,
	}

	code := app.Run([]string{"--totp", path, "--no-tui", "--search", "missing"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty output", stdout.String())
	}
}

func TestRunNoTUIMatchedInvalidProviderFailsCommand(t *testing.T) {
	path := writeProviderFile(t, `[
  {
    "name": "broken",
    "issuer": "GitHub",
    "algorithm": "SHA1",
    "digits": "SIX",
    "type": "TOTP",
    "totpSecret": "not-base32"
  }
]`, "providers.json")

	var stderr bytes.Buffer
	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		Now:    fixedNow,
	}

	code := app.Run([]string{"--totp", path, "--no-tui"})
	if code == 0 {
		t.Fatal("Run exit code = 0, want error")
	}
	if !strings.Contains(stderr.String(), "generate TOTP for GitHub/broken") {
		t.Fatalf("stderr = %q, want generation error", stderr.String())
	}
}

func TestRunNoTUIFilteredOutInvalidProviderDoesNotFailCommand(t *testing.T) {
	path := writeProviderFile(t, `[
  {
    "name": "valid",
    "issuer": "GitHub",
    "algorithm": "SHA1",
    "digits": "SIX",
    "type": "TOTP",
    "totpSecret": "JBSWY3DPEHPK3PXP"
  },
  {
    "name": "broken",
    "issuer": "AWS",
    "algorithm": "SHA1",
    "digits": "SIX",
    "type": "TOTP",
    "totpSecret": "not-base32"
  }
]`, "providers.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{
		Home:   t.TempDir(),
		Stdout: &stdout,
		Stderr: &stderr,
		Now:    fixedNow,
	}

	code := app.Run([]string{"--totp", path, "--no-tui", "--search", "valid"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}

	assertTokenTable(t, stdout.String(), "GitHub/valid")
	if strings.Contains(stdout.String(), "AWS/broken") {
		t.Fatalf("stdout = %q, want invalid provider filtered out", stdout.String())
	}
}

func singleProviderJSON() string {
	return `[
  {
    "name": "trezorg",
    "issuer": "GitHub",
    "algorithm": "SHA1",
    "digits": "SIX",
    "type": "TOTP",
    "totpSecret": "JBSWY3DPEHPK3PXP"
  }
]`
}

func twoProviderJSON() string {
	return `[
  {
    "name": "trezorg",
    "issuer": "GitHub",
    "algorithm": "SHA1",
    "digits": "SIX",
    "type": "TOTP",
    "totpSecret": "JBSWY3DPEHPK3PXP"
  },
  {
    "issuer": "AWS",
    "algorithm": "SHA1",
    "digits": "SIX",
    "type": "TOTP",
    "totpSecret": "JBSWY3DPEHPK3PXP"
  }
]`
}

func fixedNow() time.Time {
	return time.Unix(1_700_000_025, 0)
}

func assertTokenTable(t *testing.T, output string, labels ...string) {
	t.Helper()

	result, err := totpcode.Generate(provider.Provider{
		Issuer:     "Expected",
		Algorithm:  provider.AlgorithmSHA1,
		Digits:     6,
		Type:       "TOTP",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	}, fixedNow())
	if err != nil {
		t.Fatalf("generate expected token: %v", err)
	}
	for _, want := range []string{"Provider", "Token", "Remaining", "+", "|", result.Code, fmt.Sprintf("%ds", result.Remaining)} {
		if !strings.Contains(output, want) {
			t.Fatalf("stdout = %q, want table containing %q", output, want)
		}
	}
	for _, label := range labels {
		if !strings.Contains(output, label) {
			t.Fatalf("stdout = %q, want provider label %q", output, label)
		}
	}
}

func assertLaunchedLabels(t *testing.T, model tui.Model, want []string) {
	t.Helper()

	filtered := model.Filtered()
	if len(filtered) != len(want) {
		t.Fatalf("len(filtered) = %d, want %d: %#v", len(filtered), len(want), filtered)
	}
	for i := range want {
		if got := filtered[i].DisplayLabel(); got != want[i] {
			t.Fatalf("filtered[%d] = %q, want %q", i, got, want[i])
		}
	}
}

func writeProviderFile(t *testing.T, content string, name string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write provider file: %v", err)
	}
	return path
}
