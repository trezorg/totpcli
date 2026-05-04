package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func writeProviderFile(t *testing.T, content string, name string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write provider file: %v", err)
	}
	return path
}
