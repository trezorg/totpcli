package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trezorg/totpcli/internal/provider"
	"github.com/trezorg/totpcli/internal/search"
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
		NewSearchEngine: func(name string, providers []provider.Provider) (search.Engine, error) {
			engineName = name
			return search.NewLexical(providers), nil
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
		NewSearchEngine: func(name string, providers []provider.Provider) (search.Engine, error) {
			engineName = name
			return search.NewLexical(providers), nil
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
		NewSearchEngine: func(name string, providers []provider.Provider) (search.Engine, error) {
			engineName = name
			return search.NewLexical(providers), nil
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
		NewSearchEngine: func(name string, providers []provider.Provider) (search.Engine, error) {
			engineName = name
			return search.NewLexical(providers), nil
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
		"TOTPCLI_TOTP_PATH",
		"TOTPCLI_FORMAT",
		"TOTPCLI_SEARCH",
		"TOTPCLI_SEARCH_ENGINE",
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
