package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/trezorg/totpcli/internal/provider"
	"github.com/trezorg/totpcli/internal/search/lexical"
	"github.com/trezorg/totpcli/internal/storage"
	"github.com/trezorg/totpcli/internal/totpcode"
	"github.com/trezorg/totpcli/internal/tui"
	"google.golang.org/protobuf/encoding/protowire"
	"gopkg.in/yaml.v3"
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

func TestRunTOTPSubcommandUsesTOTPBehavior(t *testing.T) {
	path := writeProviderFile(t, singleProviderJSON(), "providers.json")

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

	code := app.Run([]string{"totp", "--totp", path, "--search", "git"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	if launched.SearchValue() != "git" {
		t.Fatalf("SearchValue = %q, want git", launched.SearchValue())
	}
	assertLaunchedLabels(t, launched, []string{"GitHub/trezorg"})
}

func TestRunTOTPSubcommandNoTUIPrintsTokenOutput(t *testing.T) {
	path := writeProviderFile(t, singleProviderJSON(), "providers.json")

	var stdout bytes.Buffer
	app := App{
		Home:   t.TempDir(),
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Now:    fixedNow,
		RunTUI: func(tui.Model) error {
			t.Fatal("RunTUI called in no-tui mode")
			return nil
		},
	}

	code := app.Run([]string{"totp", "--totp", path, "--no-tui"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}
	assertTokenTable(t, stdout.String(), "GitHub/trezorg")
}

func TestRunImportSubcommandUsesDecoderAndWritesProviders(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "providers.json")
	dbPath := strings.TrimSuffix(outputPath, ".json") + ".sqlite"

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{
		Home:            t.TempDir(),
		Stdout:          &stdout,
		Stderr:          &stderr,
		ImportQRDecoder: stubQRDecoder{uri: testMigrationURI([]byte("hello"), "trezorg", "GitHub")},
		RunTUI: func(tui.Model) error {
			t.Fatal("RunTUI called for import command")
			return nil
		},
	}

	code := app.Run([]string{"import", "/tmp/export.png", "--totp", outputPath})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "imported=1") || !strings.Contains(stdout.String(), "added=1") || !strings.Contains(stdout.String(), dbPath) {
		t.Fatalf("stdout = %q, want import summary", stdout.String())
	}
	providers := loadProviderDatabase(t, dbPath)
	assertProviderLabels(t, providers, []string{"GitHub/trezorg"})
}

func TestRunImportConvertsNonSQLiteOutputPath(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "providers.yaml")
	dbPath := strings.TrimSuffix(outputPath, ".yaml") + ".sqlite"

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{
		Home:            t.TempDir(),
		Stdout:          &stdout,
		Stderr:          &stderr,
		ImportQRDecoder: stubQRDecoder{uri: testMigrationURI([]byte("hello"), "trezorg", "GitHub")},
	}

	code := app.Run([]string{"import", "/tmp/export.png", "--totp", outputPath})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), dbPath) {
		t.Fatalf("stdout = %q, want converted sqlite path %q", stdout.String(), dbPath)
	}
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("source-shaped output file exists or stat failed unexpectedly: %v", err)
	}
	assertProviderLabels(t, loadProviderDatabase(t, dbPath), []string{"GitHub/trezorg"})
}

func TestRunImportDefaultsToImageSQLitePath(t *testing.T) {
	imagePath := filepath.Join(t.TempDir(), "export.png")
	dbPath := strings.TrimSuffix(imagePath, ".png") + ".sqlite"

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{
		Home:            t.TempDir(),
		Stdout:          &stdout,
		Stderr:          &stderr,
		ImportQRDecoder: stubQRDecoder{uri: testMigrationURI([]byte("hello"), "trezorg", "GitHub")},
	}

	code := app.Run([]string{"import", imagePath})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), dbPath) || !strings.Contains(stdout.String(), "imported=1") {
		t.Fatalf("stdout = %q, want sqlite import summary", stdout.String())
	}
	assertProviderLabels(t, loadProviderDatabase(t, dbPath), []string{"GitHub/trezorg"})
}

func TestRunImportTOTPPFlagOverridesEnvironmentPath(t *testing.T) {
	envPath := writeProviderFile(t, singleProviderJSON(), "env.json")
	flagPath := filepath.Join(t.TempDir(), "flag.json")
	flagDBPath := strings.TrimSuffix(flagPath, ".json") + ".sqlite"
	t.Setenv("TOTPCLI_TOTP_PATH", envPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{
		Home:            t.TempDir(),
		Stdout:          &stdout,
		Stderr:          &stderr,
		ImportQRDecoder: stubQRDecoder{uri: testMigrationURI([]byte("world"), "root", "AWS")},
	}

	code := app.Run([]string{"import", "/tmp/export.png", "--totp", flagPath})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	assertProviderLabels(t, loadProviderDatabase(t, flagDBPath), []string{"AWS/root"})
	envDBPath, err := storage.ResolveDatabasePath(envPath, "", "")
	if err != nil {
		t.Fatalf("resolve env db path: %v", err)
	}
	assertProviderLabels(t, loadProviderDatabase(t, envDBPath), []string{"GitHub/trezorg"})
}

func TestRunImportRejectsOverrideFlag(t *testing.T) {
	outputPath := writeProviderFile(t, singleProviderJSON(), "providers.json")

	var stderr bytes.Buffer
	app := App{
		Home:            t.TempDir(),
		Stdout:          &bytes.Buffer{},
		Stderr:          &stderr,
		ImportQRDecoder: stubQRDecoder{uri: testMigrationURI([]byte("world"), "root", "AWS")},
	}

	code := app.Run([]string{"import", "/tmp/export.png", "--totp", outputPath, "--override"})
	if code != 2 {
		t.Fatalf("Run exit code = %d, want 2", code)
	}
}

func TestRunImportFileImportsProviderFiles(t *testing.T) {
	dir := t.TempDir()
	jsonPath := writeImportFile(t, dir, "providers.json", singleProviderJSON())
	yamlPath := writeImportFile(t, dir, "extra.yaml", `- name: root
  issuer: AWS
  algorithm: SHA1
  digits: SIX
  type: TOTP
  totpSecret: JBSWY3DPEHPK3PXP
`)
	outputPath := filepath.Join(dir, "store.sqlite")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &stdout, Stderr: &stderr}

	code := app.Run([]string{"import-file", jsonPath, yamlPath, "--totp", outputPath})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "imported=2") || !strings.Contains(stdout.String(), "added=2") {
		t.Fatalf("stdout = %q, want import-file summary", stdout.String())
	}
	assertProviderLabels(t, loadProviderDatabase(t, outputPath), []string{"GitHub/trezorg", "AWS/root"})
}

func TestRunImportFileDerivesDatabasePathFromFirstFile(t *testing.T) {
	dir := t.TempDir()
	inputPath := writeImportFile(t, dir, "providers.json", singleProviderJSON())
	dbPath := filepath.Join(dir, "providers.sqlite")

	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

	code := app.Run([]string{"import-file", inputPath})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	assertProviderLabels(t, loadProviderDatabase(t, dbPath), []string{"GitHub/trezorg"})
}

func TestRunImportFileUsesExplicitFormat(t *testing.T) {
	dir := t.TempDir()
	inputPath := writeImportFile(t, dir, "providers.txt", `- name: trezorg
  issuer: GitHub
  algorithm: SHA1
  digits: SIX
  type: TOTP
  totpSecret: JBSWY3DPEHPK3PXP
`)
	outputPath := filepath.Join(dir, "providers.sqlite")

	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

	code := app.Run([]string{"import-file", inputPath, "--format", "yaml", "--totp", outputPath})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	assertProviderLabels(t, loadProviderDatabase(t, outputPath), []string{"GitHub/trezorg"})
}

func TestRunShowSubcommandOutputsJSONTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.sqlite")
	seedProviderDatabase(t, path, []provider.RawProvider{{
		Name:       "trezorg",
		Issuer:     "GitHub",
		Algorithm:  "SHA1",
		Digits:     "SIX",
		Type:       "TOTP",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	}})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &stdout, Stderr: &stderr}

	code := app.Run([]string{"show", "--totp", path, "--format", "json"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	var decoded struct {
		Tables []storage.TableSnapshot `json:"tables"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("show json did not parse: %v\n%s", err, stdout.String())
	}
	if len(decoded.Tables) != 1 || decoded.Tables[0].Name != "providers" {
		t.Fatalf("tables = %#v, want providers table only", decoded.Tables)
	}
	if len(decoded.Tables[0].Rows) != 1 || decoded.Tables[0].Rows[0]["issuer"] != "GitHub" {
		t.Fatalf("provider rows = %#v, want GitHub provider", decoded.Tables[0].Rows)
	}
}

func TestRunShowDefaultsToTableOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.sqlite")
	seedProviderDatabase(t, path, []provider.RawProvider{{
		Name:       "trezorg",
		Issuer:     "GitHub",
		Algorithm:  "SHA1",
		Digits:     "SIX",
		Type:       "TOTP",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	}})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &stdout, Stderr: &stderr}

	code := app.Run([]string{"show", "--totp", path})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	for _, want := range []string{"providers", "issuer", "GitHub", "+", "|"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "sqlite_sequence") {
		t.Fatalf("stdout = %q, want internal SQLite table excluded", stdout.String())
	}
}

func TestRunShowRejectsUnsupportedFormatBeforeLoadingDatabase(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		NewRepository: func(string) (providerRepository, error) {
			t.Fatal("NewRepository called before show format validation")
			return nil, nil
		},
	}

	code := app.Run([]string{"show", "--totp", filepath.Join(t.TempDir(), "missing.sqlite"), "--format", "csv"})
	if code != 2 {
		t.Fatalf("Run exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unsupported show format") {
		t.Fatalf("stderr = %q, want unsupported show format error", stderr.String())
	}
}

func TestRunImportReportsUsageAndImportErrors(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"missing image": {"import"},
		"bad flag":      {"import", "/tmp/export.png", "--format", "json"},
	}

	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

			code := app.Run(args)
			if code != 2 {
				t.Fatalf("Run exit code = %d, want 2", code)
			}
			if stderr.String() == "" {
				t.Fatal("stderr is empty, want usage error")
			}
		})
	}

	t.Run("decode error", func(t *testing.T) {
		t.Parallel()

		var stderr bytes.Buffer
		app := App{
			Home:            t.TempDir(),
			Stdout:          &bytes.Buffer{},
			Stderr:          &stderr,
			ImportQRDecoder: stubQRDecoder{err: fmt.Errorf("QR decode failed")},
		}

		code := app.Run([]string{"import", "/tmp/export.png", "--totp", filepath.Join(t.TempDir(), "providers.json")})
		if code == 0 {
			t.Fatal("Run exit code = 0, want error")
		}
		if !strings.Contains(stderr.String(), "QR decode failed") {
			t.Fatalf("stderr = %q, want decode error", stderr.String())
		}
		if strings.Count(stderr.String(), "QR decode failed") != 1 {
			t.Fatalf("stderr = %q, want decode error printed once", stderr.String())
		}
	})
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

func TestRunIgnoresEnvironmentFormat(t *testing.T) {
	path := writeProviderFile(t, singleProviderJSON(), "providers.json")
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

func TestRunDefaultPathUsesSQLiteDatabase(t *testing.T) {
	home := t.TempDir()
	seedProviderDatabase(t, filepath.Join(home, "totp.sqlite"), []provider.RawProvider{{
		Name:       "trezorg",
		Issuer:     "GitHub",
		Algorithm:  "SHA1",
		Digits:     "SIX",
		Type:       "TOTP",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	}})

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
		"totp",
		"import",
		"import-file",
		"show",
		"--totp",
		"--search",
		"--search-engine",
		"--no-tui",
		"TOTPCLI_TOTP_PATH",
		"TOTPCLI_SEARCH",
		"TOTPCLI_SEARCH_ENGINE",
		"TOTPCLI_NO_TUI",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("help missing %q:\n%s", want, help)
		}
	}
}

func TestRunSubcommandHelpListsRelevantFlags(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"totp": {
			"--totp",
			"--search",
			"--search-engine",
			"--no-tui",
		},
		"import": {
			"qr-image",
			"--totp",
			"SQLite",
		},
		"import-file": {
			"provider-file",
			"--totp",
			"--format",
		},
		"show": {
			"SQLite",
			"--totp",
			"--format",
		},
	}

	for command, wants := range tests {
		t.Run(command, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			app := App{Home: t.TempDir(), Stdout: &stdout, Stderr: &stderr}

			code := app.Run([]string{command, "--help"})
			if code != 0 {
				t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
			}
			for _, want := range wants {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("%s help missing %q:\n%s", command, want, stdout.String())
				}
			}
		})
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

func TestRunRejectsFormatFlagOnTOTPWorkflow(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"root": {"--format", "xml"},
		"totp": {"totp", "--format", "xml"},
	}

	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

			code := app.Run(args)
			if code == 0 {
				t.Fatal("Run exit code = 0, want error")
			}
			if !strings.Contains(stderr.String(), "unknown flag") {
				t.Fatalf("stderr = %q, want unknown flag error", stderr.String())
			}
		})
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

func TestRunCreatesMissingProviderDatabase(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	var launched tui.Model
	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		RunTUI: func(model tui.Model) error {
			launched = model
			return nil
		},
	}

	code := app.Run([]string{"--totp", filepath.Join(t.TempDir(), "missing.json")})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if len(launched.Filtered()) != 0 {
		t.Fatalf("filtered = %#v, want no providers", launched.Filtered())
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

func TestParseShowFormat(t *testing.T) {
	t.Parallel()

	tests := map[string]showFormat{
		"":       showFormatTable,
		"table":  showFormatTable,
		"json":   showFormatJSON,
		"yaml":   showFormatYAML,
		"toml":   showFormatTOML,
		" JSON ": showFormatJSON,
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			got, err := parseShowFormat(input)
			if err != nil {
				t.Fatalf("parseShowFormat returned error: %v", err)
			}
			if got != want {
				t.Fatalf("parseShowFormat(%q) = %q, want %q", input, got, want)
			}
		})
	}

	if _, err := parseShowFormat("csv"); err == nil {
		t.Fatal("parseShowFormat returned nil error for unsupported format")
	}
}

func TestRenderShowTables(t *testing.T) {
	t.Parallel()

	tables := []storage.TableSnapshot{{
		Name:    "providers",
		Columns: []string{"id", "issuer", "missing"},
		Rows: []map[string]any{{
			"id":      int64(1),
			"issuer":  "GitHub",
			"missing": nil,
		}},
	}}

	t.Run("table", func(t *testing.T) {
		t.Parallel()

		var output bytes.Buffer
		if err := renderShowTables(&output, tables, showFormatTable); err != nil {
			t.Fatalf("renderShowTables returned error: %v", err)
		}
		got := output.String()
		for _, want := range []string{"providers", "id", "issuer", "missing", "GitHub", "|", "+"} {
			if !strings.Contains(got, want) {
				t.Fatalf("table output missing %q:\n%s", want, got)
			}
		}
	})

	t.Run("json", func(t *testing.T) {
		t.Parallel()

		var output bytes.Buffer
		if err := renderShowTables(&output, tables, showFormatJSON); err != nil {
			t.Fatalf("renderShowTables returned error: %v", err)
		}
		var decoded struct {
			Tables []storage.TableSnapshot `json:"tables"`
		}
		if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
			t.Fatalf("json output did not parse: %v\n%s", err, output.String())
		}
		assertRenderedShowTables(t, decoded.Tables)
	})

	t.Run("yaml", func(t *testing.T) {
		t.Parallel()

		var output bytes.Buffer
		if err := renderShowTables(&output, tables, showFormatYAML); err != nil {
			t.Fatalf("renderShowTables returned error: %v", err)
		}
		var decoded struct {
			Tables []storage.TableSnapshot `yaml:"tables"`
		}
		if err := yaml.Unmarshal(output.Bytes(), &decoded); err != nil {
			t.Fatalf("yaml output did not parse: %v\n%s", err, output.String())
		}
		assertRenderedShowTables(t, decoded.Tables)
	})

	t.Run("toml", func(t *testing.T) {
		t.Parallel()

		var output bytes.Buffer
		if err := renderShowTables(&output, tables, showFormatTOML); err != nil {
			t.Fatalf("renderShowTables returned error: %v", err)
		}
		var decoded struct {
			Tables []struct {
				Name    string   `toml:"name"`
				Columns []string `toml:"columns"`
				Rows    []struct {
					Values []struct {
						Column string `toml:"column"`
						Value  string `toml:"value"`
						Null   bool   `toml:"null"`
					} `toml:"values"`
				} `toml:"rows"`
			} `toml:"tables"`
		}
		if err := toml.Unmarshal(output.Bytes(), &decoded); err != nil {
			t.Fatalf("toml output did not parse: %v\n%s", err, output.String())
		}
		if len(decoded.Tables) != 1 {
			t.Fatalf("len(tables) = %d, want 1: %#v", len(decoded.Tables), decoded.Tables)
		}
		table := decoded.Tables[0]
		if table.Name != "providers" || strings.Join(table.Columns, ",") != "id,issuer,missing" {
			t.Fatalf("decoded table = %#v, want providers with id/issuer/missing", table)
		}
		if len(table.Rows) != 1 || len(table.Rows[0].Values) != 3 {
			t.Fatalf("decoded rows = %#v, want one row with three values", table.Rows)
		}
		if table.Rows[0].Values[1].Column != "issuer" || table.Rows[0].Values[1].Value != "GitHub" {
			t.Fatalf("issuer cell = %#v, want issuer=GitHub", table.Rows[0].Values[1])
		}
		if table.Rows[0].Values[2].Column != "missing" || !table.Rows[0].Values[2].Null {
			t.Fatalf("missing cell = %#v, want null missing", table.Rows[0].Values[2])
		}
	})
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

func assertProviderLabels(t *testing.T, providers []provider.Provider, want []string) {
	t.Helper()

	if len(providers) != len(want) {
		t.Fatalf("len(providers) = %d, want %d: %#v", len(providers), len(want), providers)
	}
	for i := range want {
		if got := providers[i].DisplayLabel(); got != want[i] {
			t.Fatalf("providers[%d] = %q, want %q", i, got, want[i])
		}
	}
}

func assertRenderedShowTables(t *testing.T, tables []storage.TableSnapshot) {
	t.Helper()

	if len(tables) != 1 {
		t.Fatalf("len(tables) = %d, want 1: %#v", len(tables), tables)
	}
	table := tables[0]
	if table.Name != "providers" {
		t.Fatalf("table name = %q, want providers", table.Name)
	}
	if strings.Join(table.Columns, ",") != "id,issuer,missing" {
		t.Fatalf("columns = %#v, want id/issuer/missing", table.Columns)
	}
	if len(table.Rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(table.Rows))
	}
	if table.Rows[0]["issuer"] != "GitHub" {
		t.Fatalf("issuer = %#v, want GitHub", table.Rows[0]["issuer"])
	}
	if table.Rows[0]["missing"] != nil {
		t.Fatalf("missing = %#v, want nil", table.Rows[0]["missing"])
	}
}

func loadProviderDatabase(t *testing.T, path string) []provider.Provider {
	t.Helper()

	repo, err := storage.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open provider database: %v", err)
	}
	defer func() {
		if err := repo.Close(); err != nil {
			t.Fatalf("close provider database: %v", err)
		}
	}()
	providers, err := repo.ListProviders()
	if err != nil {
		t.Fatalf("list provider database: %v", err)
	}
	return providers
}

func seedProviderDatabase(t *testing.T, path string, providers []provider.RawProvider) {
	t.Helper()

	repo, err := storage.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open provider database: %v", err)
	}
	defer func() {
		if err := repo.Close(); err != nil {
			t.Fatalf("close provider database: %v", err)
		}
	}()
	if err := repo.Init(); err != nil {
		t.Fatalf("initialize provider database: %v", err)
	}
	if _, err := repo.UpsertProviders(providers); err != nil {
		t.Fatalf("seed provider database: %v", err)
	}
}

func writeProviderFile(t *testing.T, content string, name string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write provider file: %v", err)
	}
	raw, err := provider.ParseRaw([]byte(content), provider.DetectFormat(path))
	if err != nil {
		return path
	}
	dbPath, err := storage.ResolveDatabasePath(path, "", "")
	if err != nil {
		t.Fatalf("resolve provider database path: %v", err)
	}
	seedProviderDatabase(t, dbPath, raw)
	return path
}

func writeImportFile(t *testing.T, dir string, name string, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write import file: %v", err)
	}
	return path
}

func testMigrationURI(secret []byte, name string, issuer string) string {
	var account []byte
	account = protowire.AppendBytes(protowire.AppendTag(account, 1, protowire.BytesType), secret)
	account = protowire.AppendString(protowire.AppendTag(account, 2, protowire.BytesType), name)
	account = protowire.AppendString(protowire.AppendTag(account, 3, protowire.BytesType), issuer)
	account = protowire.AppendVarint(protowire.AppendTag(account, 4, protowire.VarintType), 1)
	account = protowire.AppendVarint(protowire.AppendTag(account, 5, protowire.VarintType), 1)
	account = protowire.AppendVarint(protowire.AppendTag(account, 6, protowire.VarintType), 2)

	var payload []byte
	payload = protowire.AppendBytes(protowire.AppendTag(payload, 1, protowire.BytesType), account)
	payload = protowire.AppendVarint(protowire.AppendTag(payload, 2, protowire.VarintType), 1)
	return "otpauth-migration://offline?data=" + base64.StdEncoding.EncodeToString(payload)
}

type stubQRDecoder struct {
	uri string
	err error
}

func (s stubQRDecoder) Decode(string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.uri, nil
}
