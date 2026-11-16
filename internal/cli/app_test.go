package cli

import (
	"bytes"
	"database/sql"
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
	"github.com/trezorg/totpcli/internal/testutil"
	"github.com/trezorg/totpcli/internal/totpcode"
	"github.com/trezorg/totpcli/internal/tui"
	"google.golang.org/protobuf/encoding/protowire"
	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

const (
	addSHA1Secret   = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	addSHA256Secret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZA"
	addSHA512Secret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNA"
)

func TestRunLoadsProvidersAndLaunchesTUIWithSearch(t *testing.T) {
	t.Parallel()

	path := writeProviderFileAndDatabase(t, `[
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
	path := writeProviderFileAndDatabase(t, singleProviderJSON(), "providers.json")
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
	path := writeProviderFileAndDatabase(t, `[
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
	path := writeProviderFileAndDatabase(t, singleProviderJSON(), "providers.json")

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
	path := writeProviderFileAndDatabase(t, singleProviderJSON(), "providers.json")

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
	path := writeProviderFileAndDatabase(t, singleProviderJSON(), "providers.json")

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
		ImportQRDecoder: testutil.StubQRDecoder{URI: testMigrationURI([]byte("hello"), "trezorg", "GitHub")},
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

func TestRunAddSubcommandWritesProviderWithDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.sqlite")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &stdout, Stderr: &stderr}

	code := app.Run([]string{"add", "--issuer", "GitHub", "--name", "trezorg", "--totp-secret", addSHA1Secret, "--totp", path})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), path) || !strings.Contains(stdout.String(), "GitHub/trezorg") {
		t.Fatalf("stdout = %q, want add summary with path and provider label", stdout.String())
	}
	providers := loadProviderDatabase(t, path)
	assertProviderLabels(t, providers, []string{"GitHub/trezorg"})
	got := providers[0]
	if got.Algorithm != provider.AlgorithmSHA1 || got.Digits != 6 || got.Secret != "" || got.Type != "TOTP" {
		t.Fatalf("provider = %#v, want default SHA1, six digits, empty secret, TOTP type", got)
	}
}

func TestRunAddSubcommandWritesExplicitOptionalProviderFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.sqlite")

	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

	code := app.Run([]string{
		"add",
		"--issuer", "GitHub",
		"--name", "trezorg",
		"--totp-secret", addSHA512Secret,
		"--secret", "metadata",
		"--algorithm", "SHA512",
		"--digits", "8",
		"--totp", path,
	})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	providers := loadProviderDatabase(t, path)
	assertProviderLabels(t, providers, []string{"GitHub/trezorg"})
	got := providers[0]
	if got.Algorithm != provider.AlgorithmSHA512 || got.Digits != 8 || got.Secret != "metadata" || got.Type != "TOTP" {
		t.Fatalf("provider = %#v, want explicit SHA512, eight digits, metadata secret, TOTP type", got)
	}
}

func TestRunAddSubcommandAllowsIssuerOnlyAndNameOnly(t *testing.T) {
	dir := t.TempDir()

	tests := map[string]struct {
		args []string
		want string
	}{
		"issuer only": {
			args: []string{"--issuer", "AWS"},
			want: "AWS",
		},
		"name only": {
			args: []string{"--name", "trezorg"},
			want: "trezorg",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, strings.ReplaceAll(name, " ", "-")+".sqlite")
			args := append([]string{"add"}, tt.args...)
			args = append(args, "--totp-secret", addSHA1Secret, "--totp", path)

			var stderr bytes.Buffer
			app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

			code := app.Run(args)
			if code != 0 {
				t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
			}
			assertProviderLabels(t, loadProviderDatabase(t, path), []string{tt.want})
		})
	}
}

func TestRunAddSubcommandResolvesEnvironmentAndDefaultDatabasePaths(t *testing.T) {
	t.Run("environment", func(t *testing.T) {
		envPath := filepath.Join(t.TempDir(), "env.sqlite")
		t.Setenv("TOTPCLI_TOTP_PATH", envPath)

		var stderr bytes.Buffer
		app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

		code := app.Run([]string{"add", "--issuer", "GitHub", "--totp-secret", addSHA1Secret})
		if code != 0 {
			t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
		}
		assertProviderLabels(t, loadProviderDatabase(t, envPath), []string{"GitHub"})
	})

	t.Run("default", func(t *testing.T) {
		home := t.TempDir()
		defaultPath := filepath.Join(home, "totp.sqlite")

		var stderr bytes.Buffer
		app := App{Home: home, Stdout: &bytes.Buffer{}, Stderr: &stderr}

		code := app.Run([]string{"add", "--issuer", "GitHub", "--totp-secret", addSHA1Secret})
		if code != 0 {
			t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
		}
		assertProviderLabels(t, loadProviderDatabase(t, defaultPath), []string{"GitHub"})
	})
}

func TestRunAddSubcommandRejectsInvalidInputBeforeOpeningDatabase(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"missing secret":           {"add", "--issuer", "GitHub"},
		"empty secret":             {"add", "--issuer", "GitHub", "--totp-secret", ""},
		"missing identity":         {"add", "--totp-secret", addSHA1Secret},
		"invalid totp secret":      {"add", "--issuer", "GitHub", "--totp-secret", "not-base32"},
		"base32 encoded short key": {"add", "--issuer", "GitHub", "--totp-secret", "NBUQ"},
		"sha1 key too short":       {"add", "--issuer", "GitHub", "--totp-secret", "JBSWY3DPEHPK3PXP"},
		"sha256 key too short":     {"add", "--issuer", "GitHub", "--totp-secret", addSHA1Secret, "--algorithm", "SHA256"},
		"sha512 key too short":     {"add", "--issuer", "GitHub", "--totp-secret", addSHA256Secret, "--algorithm", "SHA512"},
		"unsupported algorithm":    {"add", "--issuer", "GitHub", "--totp-secret", addSHA1Secret, "--algorithm", "MD5"},
		"unsupported digits":       {"add", "--issuer", "GitHub", "--totp-secret", addSHA1Secret, "--digits", "7"},
	}

	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			app := App{
				Home:   t.TempDir(),
				Stdout: &bytes.Buffer{},
				Stderr: &stderr,
				NewRepository: func(string) (providerRepository, error) {
					t.Fatal("NewRepository called before add input validation")
					return nil, nil
				},
			}

			code := app.Run(args)
			if code != 2 {
				t.Fatalf("Run exit code = %d, want 2", code)
			}
			if stderr.String() == "" {
				t.Fatal("stderr is empty, want validation error")
			}
		})
	}
}

func TestRunAddSubcommandRejectsEmptyTOTPSecretWithFlagMessage(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		NewRepository: func(string) (providerRepository, error) {
			t.Fatal("NewRepository called before add input validation")
			return nil, nil
		},
	}

	code := app.Run([]string{"add", "--issuer", "GitHub", "--totp-secret", ""})
	if code != 2 {
		t.Fatalf("Run exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "totp secret is required") {
		t.Fatalf("stderr = %q, want totp secret is required", stderr.String())
	}
}

func TestRunAddSubcommandUpdatesDuplicateProvider(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.sqlite")
	seedProviderDatabase(t, path, []provider.RawProvider{{
		Name:       "trezorg",
		Issuer:     "GitHub",
		Algorithm:  "SHA1",
		Digits:     "SIX",
		Type:       "TOTP",
		TOTPSecret: addSHA256Secret,
	}})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &stdout, Stderr: &stderr}

	code := app.Run([]string{"add", "--issuer", " GitHub ", "--name", " trezorg ", "--totp-secret", addSHA256Secret, "--algorithm", "SHA256", "--digits", "EIGHT", "--secret", "metadata", "--totp", path})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "GitHub/trezorg") || !strings.Contains(stdout.String(), path) {
		t.Fatalf("stdout = %q, want add summary with path and provider label", stdout.String())
	}
	providers := loadProviderDatabase(t, path)
	assertProviderLabels(t, providers, []string{"GitHub/trezorg"})
	if providers[0].Algorithm != provider.AlgorithmSHA256 || providers[0].Digits != 8 || providers[0].Secret != "metadata" {
		t.Fatalf("provider = %#v, want duplicate updated", providers[0])
	}
}

func TestRunAddSubcommandRejectsUnsupportedFlags(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"type":   {"add", "--issuer", "GitHub", "--totp-secret", "JBSWY3DPEHPK3PXP", "--type", "TOTP"},
		"format": {"add", "--issuer", "GitHub", "--totp-secret", "JBSWY3DPEHPK3PXP", "--format", "json"},
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
			if !strings.Contains(stderr.String(), "unknown flag") {
				t.Fatalf("stderr = %q, want unknown flag error", stderr.String())
			}
		})
	}
}

func TestRunDeleteSubcommandDeletesProvider(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.sqlite")
	seedProviderDatabase(t, path, []provider.RawProvider{
		{Name: "trezorg", Issuer: "GitHub", Type: "TOTP", TOTPSecret: addSHA1Secret},
		{Issuer: "AWS", Type: "TOTP", TOTPSecret: addSHA1Secret},
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{
		Home:   t.TempDir(),
		Stdout: &stdout,
		Stderr: &stderr,
		RunTUI: func(tui.Model) error {
			t.Fatal("RunTUI called for delete command")
			return nil
		},
	}

	code := app.Run([]string{"delete", "--issuer", "GitHub", "--name", "trezorg", "--totp", path})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), path) || !strings.Contains(stdout.String(), "GitHub/trezorg") {
		t.Fatalf("stdout = %q, want delete summary with path and provider label", stdout.String())
	}
	assertProviderLabels(t, loadProviderDatabase(t, path), []string{"AWS"})
}

func TestRunDeleteSubcommandAllowsIssuerOnlyNameOnlyAndSecretDisambiguation(t *testing.T) {
	t.Run("issuer only", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "providers.sqlite")
		seedProviderDatabase(t, path, []provider.RawProvider{
			{Issuer: "AWS", Type: "TOTP", TOTPSecret: addSHA1Secret},
			{Name: "trezorg", Issuer: "GitHub", Type: "TOTP", TOTPSecret: addSHA1Secret},
		})

		var stderr bytes.Buffer
		app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

		code := app.Run([]string{"delete", "--issuer", "AWS", "--totp", path})
		if code != 0 {
			t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
		}
		assertProviderLabels(t, loadProviderDatabase(t, path), []string{"GitHub/trezorg"})
	})

	t.Run("name only", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "providers.sqlite")
		seedProviderDatabase(t, path, []provider.RawProvider{
			{Name: "trezorg", Type: "TOTP", TOTPSecret: addSHA1Secret},
			{Issuer: "AWS", Type: "TOTP", TOTPSecret: addSHA1Secret},
		})

		var stderr bytes.Buffer
		app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

		code := app.Run([]string{"delete", "--name", "trezorg", "--totp", path})
		if code != 0 {
			t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
		}
		assertProviderLabels(t, loadProviderDatabase(t, path), []string{"AWS"})
	})

	t.Run("totp secret disambiguation", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "providers.sqlite")
		seedProviderDatabase(t, path, []provider.RawProvider{
			{Name: "trezorg", Issuer: "GitHub", Type: "TOTP", TOTPSecret: addSHA1Secret},
			{Name: "root", Issuer: "GitHub", Type: "TOTP", TOTPSecret: addSHA256Secret},
		})

		var stderr bytes.Buffer
		app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

		code := app.Run([]string{"delete", "--issuer", "GitHub", "--totp-secret", addSHA1Secret, "--totp", path})
		if code != 0 {
			t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
		}
		assertProviderLabels(t, loadProviderDatabase(t, path), []string{"GitHub/root"})
	})
}

func TestRunDeleteSubcommandResolvesEnvironmentAndDefaultDatabasePaths(t *testing.T) {
	t.Run("environment", func(t *testing.T) {
		envPath := filepath.Join(t.TempDir(), "env.sqlite")
		seedProviderDatabase(t, envPath, []provider.RawProvider{{Issuer: "GitHub", Type: "TOTP", TOTPSecret: addSHA1Secret}})
		t.Setenv("TOTPCLI_TOTP_PATH", envPath)

		var stderr bytes.Buffer
		app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

		code := app.Run([]string{"delete", "--issuer", "GitHub"})
		if code != 0 {
			t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
		}
		assertProviderLabels(t, loadProviderDatabase(t, envPath), nil)
	})

	t.Run("default", func(t *testing.T) {
		home := t.TempDir()
		defaultPath := filepath.Join(home, "totp.sqlite")
		seedProviderDatabase(t, defaultPath, []provider.RawProvider{{Issuer: "GitHub", Type: "TOTP", TOTPSecret: addSHA1Secret}})

		var stderr bytes.Buffer
		app := App{Home: home, Stdout: &bytes.Buffer{}, Stderr: &stderr}

		code := app.Run([]string{"delete", "--issuer", "GitHub"})
		if code != 0 {
			t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
		}
		assertProviderLabels(t, loadProviderDatabase(t, defaultPath), nil)
	})
}

func TestRunDeleteSubcommandRejectsInvalidInputBeforeOpeningDatabase(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"missing identity":    {"delete"},
		"invalid totp secret": {"delete", "--issuer", "GitHub", "--totp-secret", "not-base32"},
	}

	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			app := App{
				Home:   t.TempDir(),
				Stdout: &bytes.Buffer{},
				Stderr: &stderr,
				NewRepository: func(string) (providerRepository, error) {
					t.Fatal("NewRepository called before delete input validation")
					return nil, nil
				},
			}

			code := app.Run(args)
			if code != 2 {
				t.Fatalf("Run exit code = %d, want 2", code)
			}
			if stderr.String() == "" {
				t.Fatal("stderr is empty, want validation error")
			}
		})
	}
}

func TestRunDeleteSubcommandReportsSelectionErrorsWithoutDeleting(t *testing.T) {
	tests := map[string]struct {
		args    []string
		wantErr string
	}{
		"not found": {
			args:    []string{"delete", "--issuer", "Missing"},
			wantErr: "no matching provider",
		},
		"ambiguous": {
			args:    []string{"delete", "--issuer", "GitHub"},
			wantErr: "ambiguous",
		},
		"partial selector": {
			args:    []string{"delete", "--issuer", "Git"},
			wantErr: "no matching provider",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "providers.sqlite")
			seedProviderDatabase(t, path, []provider.RawProvider{
				{Name: "trezorg", Issuer: "GitHub", Type: "TOTP", TOTPSecret: addSHA1Secret},
				{Name: "root", Issuer: "GitHub", Type: "TOTP", TOTPSecret: addSHA256Secret},
			})

			var stderr bytes.Buffer
			args := append(append([]string{}, tt.args...), "--totp", path)
			app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

			code := app.Run(args)
			if code == 0 {
				t.Fatal("Run exit code = 0, want error")
			}
			if !strings.Contains(stderr.String(), tt.wantErr) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.wantErr)
			}
			assertProviderLabels(t, loadProviderDatabase(t, path), []string{"GitHub/trezorg", "GitHub/root"})
		})
	}
}

func TestRunDeleteSubcommandRejectsUnsupportedFlags(t *testing.T) {
	t.Parallel()

	tests := map[string][]string{
		"format": {"delete", "--issuer", "GitHub", "--format", "json"},
		"type":   {"delete", "--issuer", "GitHub", "--type", "TOTP"},
		"id":     {"delete", "--id", "1"},
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
			if !strings.Contains(stderr.String(), "unknown flag") {
				t.Fatalf("stderr = %q, want unknown flag error", stderr.String())
			}
		})
	}
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
		ImportQRDecoder: testutil.StubQRDecoder{URI: testMigrationURI([]byte("hello"), "trezorg", "GitHub")},
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

func TestRunImportDefaultsToHomeDatabasePath(t *testing.T) {
	home := t.TempDir()
	imagePath := filepath.Join(t.TempDir(), "export.png")
	dbPath := filepath.Join(home, "totp.sqlite")
	t.Setenv("TOTPCLI_TOTP_PATH", "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{
		Home:            home,
		Stdout:          &stdout,
		Stderr:          &stderr,
		ImportQRDecoder: testutil.StubQRDecoder{URI: testMigrationURI([]byte("hello"), "trezorg", "GitHub")},
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
	envPath := writeProviderFileAndDatabase(t, singleProviderJSON(), "env.json")
	flagPath := filepath.Join(t.TempDir(), "flag.json")
	flagDBPath := strings.TrimSuffix(flagPath, ".json") + ".sqlite"
	t.Setenv("TOTPCLI_TOTP_PATH", envPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{
		Home:            t.TempDir(),
		Stdout:          &stdout,
		Stderr:          &stderr,
		ImportQRDecoder: testutil.StubQRDecoder{URI: testMigrationURI([]byte("world"), "root", "AWS")},
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
	outputPath := writeProviderFileAndDatabase(t, singleProviderJSON(), "providers.json")

	var stderr bytes.Buffer
	app := App{
		Home:            t.TempDir(),
		Stdout:          &bytes.Buffer{},
		Stderr:          &stderr,
		ImportQRDecoder: testutil.StubQRDecoder{URI: testMigrationURI([]byte("world"), "root", "AWS")},
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

func TestRunImportFileDefaultsToHomeDatabasePath(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	inputPath := writeImportFile(t, dir, "providers.json", singleProviderJSON())
	dbPath := filepath.Join(home, "totp.sqlite")
	t.Setenv("TOTPCLI_TOTP_PATH", "")

	var stderr bytes.Buffer
	app := App{Home: home, Stdout: &bytes.Buffer{}, Stderr: &stderr}

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
	if got := decoded.Tables[0].Rows[0]["totp_secret"]; got != redactedShowValue {
		t.Fatalf("totp_secret = %#v, want redacted value", got)
	}
	if got := decoded.Tables[0].Rows[0]["totp_secret_key"]; got != redactedShowValue {
		t.Fatalf("totp_secret_key = %#v, want redacted value", got)
	}
	if strings.Contains(stdout.String(), "JBSWY3DPEHPK3PXP") {
		t.Fatalf("stdout leaked TOTP secret:\n%s", stdout.String())
	}
}

func TestRunShowRevealSecretsOutputsJSONTables(t *testing.T) {
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

	code := app.Run([]string{"show", "--totp", path, "--format", "json", "--reveal-secrets"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	var decoded struct {
		Tables []storage.TableSnapshot `json:"tables"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("show json did not parse: %v\n%s", err, stdout.String())
	}
	if len(decoded.Tables) != 1 || len(decoded.Tables[0].Rows) != 1 {
		t.Fatalf("tables = %#v, want one provider row", decoded.Tables)
	}
	if got := decoded.Tables[0].Rows[0]["totp_secret"]; got != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("totp_secret = %#v, want revealed secret", got)
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
	if strings.Contains(stdout.String(), "JBSWY3DPEHPK3PXP") {
		t.Fatalf("stdout leaked TOTP secret:\n%s", stdout.String())
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

func TestRunExportSubcommandWritesProviderFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.sqlite")
	outPath := filepath.Join(t.TempDir(), "backup.json")
	seedProviderDatabase(t, path, []provider.RawProvider{{
		Secret:     "metadata",
		Name:       "trezorg",
		Issuer:     "GitHub",
		Algorithm:  "SHA512",
		Digits:     "EIGHT",
		Type:       "TOTP",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	}})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &stdout, Stderr: &stderr}

	code := app.Run([]string{"export", "--totp", path, "--output", outPath})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "exported 1 providers to "+outPath) || !strings.Contains(stdout.String(), "format=json") {
		t.Fatalf("stdout = %q, want export summary with path and format", stdout.String())
	}
	raw, err := provider.LoadRaw(outPath, provider.FormatJSON)
	if err != nil {
		t.Fatalf("LoadRaw returned error: %v", err)
	}
	if len(raw) != 1 {
		t.Fatalf("len(raw) = %d, want 1", len(raw))
	}
	want := provider.RawProvider{Secret: "metadata", Name: "trezorg", Issuer: "GitHub", Algorithm: "SHA512", Digits: "EIGHT", Type: "TOTP", TOTPSecret: "JBSWY3DPEHPK3PXP"}
	if raw[0] != want {
		t.Fatalf("raw[0] = %#v, want %#v", raw[0], want)
	}
	if mode := exportOutputMode(t, outPath); mode != 0o600 {
		t.Fatalf("mode = %o, want 600", mode)
	}
}

func TestRunExportRefusesExistingOutputWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.sqlite")
	outPath := filepath.Join(t.TempDir(), "backup.json")
	seedProviderDatabase(t, path, []provider.RawProvider{{
		Name:       "trezorg",
		Issuer:     "GitHub",
		Type:       "TOTP",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	}})
	if err := os.WriteFile(outPath, []byte("precious"), 0o644); err != nil {
		t.Fatalf("write existing output: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &stdout, Stderr: &stderr}

	code := app.Run([]string{"export", "--totp", path, "--output", outPath})
	if code != 1 {
		t.Fatalf("Run exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "--force") {
		t.Fatalf("stderr = %q, want --force hint", stderr.String())
	}
	if data, err := os.ReadFile(outPath); err != nil || string(data) != "precious" {
		t.Fatalf("output content = %q err = %v, want unchanged precious", data, err)
	}

	stdout.Reset()
	stderr.Reset()
	code = app.Run([]string{"export", "--totp", path, "--output", outPath, "--force"})
	if code != 0 {
		t.Fatalf("forced Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	raw, err := provider.LoadRaw(outPath, provider.FormatJSON)
	if err != nil || len(raw) != 1 {
		t.Fatalf("forced LoadRaw = %#v err = %v, want 1 provider", raw, err)
	}
	if mode := exportOutputMode(t, outPath); mode != 0o600 {
		t.Fatalf("forced mode = %o, want 600", mode)
	}
}

func TestRunExportDefaultsToHomeJSONPath(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(t.TempDir(), "providers.sqlite")
	seedProviderDatabase(t, path, []provider.RawProvider{{
		Name:       "trezorg",
		Issuer:     "GitHub",
		Type:       "TOTP",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	}})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{Home: home, Stdout: &stdout, Stderr: &stderr}

	code := app.Run([]string{"export", "--totp", path})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	wantPath := filepath.Join(home, "totp.json")
	if !strings.Contains(stdout.String(), wantPath) {
		t.Fatalf("stdout = %q, want default output path %q", stdout.String(), wantPath)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("default output missing: %v", err)
	}
}

func TestRunExportExplicitFormatWinsOverExtension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.sqlite")
	outPath := filepath.Join(t.TempDir(), "backup.json")
	seedProviderDatabase(t, path, []provider.RawProvider{{
		Name:       "trezorg",
		Issuer:     "GitHub",
		Type:       "TOTP",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	}})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &stdout, Stderr: &stderr}

	code := app.Run([]string{"export", "--totp", path, "--output", outPath, "--format", "yaml"})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "format=yaml") {
		t.Fatalf("stdout = %q, want format=yaml", stdout.String())
	}
	if _, err := provider.LoadRaw(outPath, provider.FormatYAML); err != nil {
		t.Fatalf("LoadRaw yaml returned error: %v", err)
	}
}

func TestRunExportEmptyDatabaseWritesEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.sqlite")
	seedProviderDatabase(t, path, nil)
	outPath := filepath.Join(t.TempDir(), "backup.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &stdout, Stderr: &stderr}

	code := app.Run([]string{"export", "--totp", path, "--output", outPath})
	if code != 0 {
		t.Fatalf("Run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	raw, err := provider.LoadRaw(outPath, provider.FormatJSON)
	if err != nil || len(raw) != 0 {
		t.Fatalf("LoadRaw = %#v err = %v, want zero providers", raw, err)
	}
	if !strings.Contains(stdout.String(), "exported 0 providers") {
		t.Fatalf("stdout = %q, want zero-provider summary", stdout.String())
	}
}

func TestRunExportRejectsMissingSource(t *testing.T) {
	for _, source := range []string{"flag", "env", "default"} {
		for _, force := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/force=%t", source, force), func(t *testing.T) {
				home := t.TempDir()
				path := filepath.Join(home, "totp.sqlite")
				outPath := filepath.Join(home, "backup.json")
				args := []string{"export", "--output", outPath}
				t.Setenv("TOTPCLI_TOTP_PATH", "")
				switch source {
				case "flag":
					path = filepath.Join(home, "missing.sqlite")
					args = append(args, "--totp", path)
				case "env":
					path = filepath.Join(home, "missing.sqlite")
					t.Setenv("TOTPCLI_TOTP_PATH", path)
				}
				const backup = `[{"name":"existing backup","totpSecret":"JBSWY3DPEHPK3PXP"}]`
				if force {
					if err := os.WriteFile(outPath, []byte(backup), 0o600); err != nil {
						t.Fatal(err)
					}
					args = append(args, "--force")
				}

				var stdout, stderr bytes.Buffer
				app := App{Home: home, Stdout: &stdout, Stderr: &stderr}
				if code := app.Run(args); code != 1 {
					t.Errorf("Run exit code = %d, want 1; stderr = %q", code, stderr.String())
				}
				if stdout.Len() != 0 {
					t.Errorf("stdout = %q, want no success summary", stdout.String())
				}
				if !strings.Contains(stderr.String(), path) {
					t.Errorf("stderr = %q, want source path %q", stderr.String(), path)
				}
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Errorf("source database was created: stat error = %v", err)
				}
				if force {
					data, err := os.ReadFile(outPath)
					if err != nil || string(data) != backup {
						t.Errorf("backup = %q, err = %v; want unchanged backup", data, err)
					}
				} else if _, err := os.Stat(outPath); !os.IsNotExist(err) {
					t.Errorf("output was created: stat error = %v", err)
				}
			})
		}
	}
}

func TestRunExportRejectsUnsupportedFormatBeforeOpeningDatabase(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	app := App{
		Home:   t.TempDir(),
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		NewRepository: func(string) (providerRepository, error) {
			t.Fatal("NewRepository called before export format validation")
			return nil, nil
		},
	}

	code := app.Run([]string{"export", "--totp", filepath.Join(t.TempDir(), "missing.sqlite"), "--format", "xml"})
	if code != 2 {
		t.Fatalf("Run exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unsupported format") {
		t.Fatalf("stderr = %q, want unsupported format error", stderr.String())
	}
}

func TestRunExportRejectsUnsupportedFlags(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &stderr}

	code := app.Run([]string{"export", "--totp", filepath.Join(t.TempDir(), "providers.sqlite"), "--search", "git"})
	if code == 0 {
		t.Fatal("Run exit code = 0, want non-zero for unsupported flag")
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("stderr = %q, want unknown flag error", stderr.String())
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
			ImportQRDecoder: testutil.StubQRDecoder{Err: fmt.Errorf("QR decode failed")},
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
	path := writeProviderFileAndDatabase(t, singleProviderJSON(), "providers.json")
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
	envPath := writeProviderFileAndDatabase(t, singleProviderJSON(), "env.json")
	flagPath := writeProviderFileAndDatabase(t, `[
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
	path := writeProviderFileAndDatabase(t, singleProviderJSON(), "providers.json")
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
	path := writeProviderFileAndDatabase(t, singleProviderJSON(), "providers.json")

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
	path := writeProviderFileAndDatabase(t, singleProviderJSON(), "providers.json")

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
	path := writeProviderFileAndDatabase(t, singleProviderJSON(), "providers.json")
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
	path := writeProviderFileAndDatabase(t, singleProviderJSON(), "providers.json")
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
	path := writeProviderFileAndDatabase(t, singleProviderJSON(), "providers.json")
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
		"add",
		"delete",
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
		"add": {
			"provider",
			"default SHA1",
			"default SIX",
			"type defaults to TOTP",
			"--totp",
			"--issuer",
			"--name",
			"--totp-secret",
			"--secret",
			"--algorithm",
			"--digits",
		},
		"delete": {
			"provider",
			"--totp",
			"--issuer",
			"--name",
			"--totp-secret",
		},
		"import": {
			"qr-image",
			"--totp",
			"SQLite",
			"~/totp.sqlite",
		},
		"import-file": {
			"provider-file",
			"--totp",
			"--format",
			"~/totp.sqlite",
		},
		"show": {
			"SQLite",
			"--totp",
			"--format",
			"--reveal-secrets",
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
			if command == "add" {
				for _, unwanted := range []string{"--format", "--type"} {
					if strings.Contains(stdout.String(), unwanted) {
						t.Fatalf("%s help includes %q unexpectedly:\n%s", command, unwanted, stdout.String())
					}
				}
			}
			if command == "delete" {
				for _, unwanted := range []string{"--format", "--type", "--id"} {
					if strings.Contains(stdout.String(), unwanted) {
						t.Fatalf("%s help includes %q unexpectedly:\n%s", command, unwanted, stdout.String())
					}
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
	path := writeProviderFileAndDatabase(t, twoProviderJSON(), "providers.json")

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
	path := writeProviderFileAndDatabase(t, twoProviderJSON(), "providers.json")

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
	path := writeProviderFileAndDatabase(t, twoProviderJSON(), "providers.json")

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
	path := writeProviderFileAndUnsafeDatabase(t, `[
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
	path := writeProviderFileAndUnsafeDatabase(t, `[
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

func TestShowFormatSet(t *testing.T) {
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

			var got showFormat
			if err := got.Set(input); err != nil {
				t.Fatalf("showFormat.Set returned error: %v", err)
			}
			if got != want {
				t.Fatalf("showFormat.Set(%q) = %q, want %q", input, got, want)
			}
		})
	}

	var f showFormat
	if err := f.Set("csv"); err == nil {
		t.Fatal("showFormat.Set returned nil error for unsupported format")
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

func TestRunExportImportFileRoundTrip(t *testing.T) {
	sourceDB := filepath.Join(t.TempDir(), "source.sqlite")
	targetDB := filepath.Join(t.TempDir(), "target.sqlite")
	backupPath := filepath.Join(t.TempDir(), "backup.yaml")
	seedProviderDatabase(t, sourceDB, []provider.RawProvider{
		{Name: "trezorg", Issuer: "GitHub", Algorithm: "SHA1", Digits: "SIX", Type: "TOTP", TOTPSecret: "JBSWY3DPEHPK3PXP"},
		{Secret: "metadata", Name: "ops", Issuer: "GitHub", Algorithm: "SHA512", Digits: "EIGHT", Type: "TOTP", TOTPSecret: "JBSWY3DPEHPK3PXQ"},
		{Name: "deploy", Issuer: "GitLab", Algorithm: "SHA1", Digits: "SIX", Type: "TOTP", TOTPSecret: "JBSWY3DPEHPK3PXA"},
		{Name: "deploy", Issuer: "GitLab", Algorithm: "SHA256", Digits: "SIX", Type: "TOTP", TOTPSecret: "JBSWY3DPEHPK3PXB"},
	})

	app := App{Home: t.TempDir(), Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}

	if code := app.Run([]string{"export", "--totp", sourceDB, "--output", backupPath}); code != 0 {
		t.Fatalf("export exit code = %d, want 0", code)
	}
	if code := app.Run([]string{"import-file", backupPath, "--totp", targetDB}); code != 0 {
		t.Fatalf("import-file exit code = %d, want 0", code)
	}

	source := indexProviders(loadProviderDatabase(t, sourceDB))
	target := indexProviders(loadProviderDatabase(t, targetDB))
	if len(source) != 4 || len(target) != 4 {
		t.Fatalf("providers = %d/%d, want 4/4: %v / %v", len(source), len(target), source, target)
	}
	for key, want := range source {
		got, ok := target[key]
		if !ok {
			t.Fatalf("round trip lost provider %q: %v", key, target)
		}
		if got != want {
			t.Fatalf("round trip changed provider %q: got %#v, want %#v", key, got, want)
		}
	}
}

func indexProviders(providers []provider.Provider) map[string]provider.Provider {
	indexed := make(map[string]provider.Provider, len(providers))
	for _, stored := range providers {
		indexed[stored.DisplayLabel()+"|"+stored.TOTPSecret] = stored
	}
	return indexed
}

func TestRunExportRejectsOutputMatchingSourceDatabase(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "totp.sqlite")
	seedProviderDatabase(t, dbPath, []provider.RawProvider{{
		Name:       "trezorg",
		Issuer:     "GitHub",
		Type:       "TOTP",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	}})
	before, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("read source database: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{Home: t.TempDir(), Stdout: &stdout, Stderr: &stderr}

	code := app.Run([]string{"export", "--totp", dbPath, "--output", dbPath, "--force"})
	if code != 2 {
		t.Fatalf("Run exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "must differ from the provider database") {
		t.Fatalf("stderr = %q, want source/output mismatch error", stderr.String())
	}
	if after, err := os.ReadFile(dbPath); err != nil || !bytes.Equal(before, after) {
		t.Fatalf("source database modified: equal=%v err=%v", bytes.Equal(before, after), err)
	}

	stdout.Reset()
	stderr.Reset()
	alias := filepath.Join(dir, "alias.json")
	if err := os.Link(dbPath, alias); err != nil {
		t.Fatalf("hard-link source database: %v", err)
	}
	code = app.Run([]string{"export", "--totp", dbPath, "--output", alias, "--force"})
	if code != 2 {
		t.Fatalf("alias Run exit code = %d, want 2", code)
	}
	if after, err := os.ReadFile(dbPath); err != nil || !bytes.Equal(before, after) {
		t.Fatalf("source database modified via alias: equal=%v err=%v", bytes.Equal(before, after), err)
	}
	if data, err := os.ReadFile(alias); err != nil || !bytes.Equal(before, data) {
		t.Fatalf("alias content = source db bytes equal=%v err=%v; want untouched hard link", bytes.Equal(before, data), err)
	}
}

func exportOutputMode(t *testing.T, path string) os.FileMode {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat export output %q: %v", path, err)
	}
	return info.Mode().Perm()
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

func writeProviderFileAndDatabase(t *testing.T, content string, name string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write provider file: %v", err)
	}
	raw, err := provider.ParseRaw([]byte(content), provider.DetectFormat(path))
	if err != nil {
		t.Fatalf("parse provider fixture: %v", err)
	}
	dbPath, err := storage.ResolveDatabasePath(path, "", "")
	if err != nil {
		t.Fatalf("resolve provider database path: %v", err)
	}
	seedProviderDatabase(t, dbPath, raw)
	return path
}

func writeProviderFileAndUnsafeDatabase(t *testing.T, content string, name string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write provider file: %v", err)
	}
	raw, err := provider.ParseRaw([]byte(content), provider.DetectFormat(path))
	if err != nil {
		t.Fatalf("parse provider fixture: %v", err)
	}
	dbPath, err := storage.ResolveDatabasePath(path, "", "")
	if err != nil {
		t.Fatalf("resolve provider database path: %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open unsafe provider database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close unsafe provider database: %v", err)
		}
	}()
	if _, err := db.Exec(`
CREATE TABLE providers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	secret TEXT NOT NULL DEFAULT '',
	name TEXT NOT NULL DEFAULT '',
	issuer TEXT NOT NULL DEFAULT '',
	algorithm TEXT NOT NULL,
	digits INTEGER NOT NULL,
	type TEXT NOT NULL,
	totp_secret TEXT NOT NULL,
	totp_secret_key TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(issuer, name, totp_secret_key)
);`); err != nil {
		t.Fatalf("create unsafe provider database: %v", err)
	}
	for _, record := range raw {
		digits := 6
		if strings.EqualFold(strings.TrimSpace(record.Digits), "EIGHT") || strings.TrimSpace(record.Digits) == "8" {
			digits = 8
		}
		if _, err := db.Exec(`
INSERT INTO providers (secret, name, issuer, algorithm, digits, type, totp_secret, totp_secret_key)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			strings.TrimSpace(record.Secret),
			strings.TrimSpace(record.Name),
			strings.TrimSpace(record.Issuer),
			strings.ToUpper(strings.TrimSpace(record.Algorithm)),
			digits,
			strings.ToUpper(strings.TrimSpace(record.Type)),
			strings.TrimSpace(record.TOTPSecret),
			strings.ToUpper(strings.TrimSpace(record.TOTPSecret)),
		); err != nil {
			t.Fatalf("seed unsafe provider database: %v", err)
		}
	}
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
