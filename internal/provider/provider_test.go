package provider

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveOptionsUsesExplicitPathAndFormat(t *testing.T) {
	t.Parallel()

	resolved, err := ResolveOptions(Options{
		Path:   "/tmp/providers.yaml",
		Format: "yaml",
		Home:   "/home/tester",
	})
	if err != nil {
		t.Fatalf("ResolveOptions returned error: %v", err)
	}

	if resolved.Path != "/tmp/providers.yaml" {
		t.Fatalf("path = %q, want explicit path", resolved.Path)
	}
	if resolved.Format != FormatYAML {
		t.Fatalf("format = %q, want yaml", resolved.Format)
	}
}

func TestResolveOptionsDefaultsToJSONPath(t *testing.T) {
	t.Parallel()

	resolved, err := ResolveOptions(Options{Home: "/home/tester"})
	if err != nil {
		t.Fatalf("ResolveOptions returned error: %v", err)
	}

	if resolved.Path != "/home/tester/totp.json" {
		t.Fatalf("path = %q, want default json path", resolved.Path)
	}
	if resolved.Format != FormatJSON {
		t.Fatalf("format = %q, want json", resolved.Format)
	}
}

func TestResolveOptionsDefaultPathFollowsExplicitFormat(t *testing.T) {
	t.Parallel()

	resolved, err := ResolveOptions(Options{Format: "toml", Home: "/home/tester"})
	if err != nil {
		t.Fatalf("ResolveOptions returned error: %v", err)
	}

	if resolved.Path != "/home/tester/totp.toml" {
		t.Fatalf("path = %q, want default toml path", resolved.Path)
	}
	if resolved.Format != FormatTOML {
		t.Fatalf("format = %q, want toml", resolved.Format)
	}
}

func TestResolveOptionsDetectsFormatFromExtension(t *testing.T) {
	t.Parallel()

	tests := map[string]Format{
		"~/totp.json": FormatJSON,
		"~/totp.yaml": FormatYAML,
		"~/totp.yml":  FormatYAML,
		"~/totp.toml": FormatTOML,
		"~/totp.data": FormatJSON,
	}

	for path, want := range tests {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			resolved, err := ResolveOptions(Options{Path: path, Home: "/home/tester"})
			if err != nil {
				t.Fatalf("ResolveOptions returned error: %v", err)
			}
			if resolved.Format != want {
				t.Fatalf("format = %q, want %q", resolved.Format, want)
			}
		})
	}
}

func TestParseRejectsInvalidFormat(t *testing.T) {
	t.Parallel()

	_, err := ParseFormat("xml")
	if err == nil {
		t.Fatal("ParseFormat returned nil error for invalid format")
	}
}

func TestLoadParsesJSONYAMLandTOML(t *testing.T) {
	t.Parallel()

	files := []string{"providers.json", "providers.yaml", "providers.toml"}

	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join("testdata", name)
			resolved, err := ResolveOptions(Options{Path: path})
			if err != nil {
				t.Fatalf("ResolveOptions returned error: %v", err)
			}

			providers, err := Load(resolved.Path, resolved.Format)
			if err != nil {
				t.Fatalf("Load returned error: %v", err)
			}

			if len(providers) != 1 {
				t.Fatalf("len(providers) = %d, want 1", len(providers))
			}
			got := providers[0]
			if got.DisplayLabel() != "GitHub/trezorg" {
				t.Fatalf("DisplayLabel = %q, want GitHub/trezorg", got.DisplayLabel())
			}
			if got.Algorithm != AlgorithmSHA1 {
				t.Fatalf("Algorithm = %q, want SHA1", got.Algorithm)
			}
			if got.Digits != 6 {
				t.Fatalf("Digits = %d, want 6", got.Digits)
			}
			if got.TOTPSecret != "JBSWY3DPEHPK3PXP" {
				t.Fatalf("TOTPSecret = %q, want fixture secret", got.TOTPSecret)
			}
		})
	}
}

func TestValidateProviderRejectsMissingSecret(t *testing.T) {
	t.Parallel()

	_, err := Normalize(RawProvider{Issuer: "GitHub", Type: "TOTP"})
	if err == nil {
		t.Fatal("Normalize returned nil error for missing TOTP secret")
	}
	if !strings.Contains(err.Error(), "totpSecret") {
		t.Fatalf("error = %q, want mention totpSecret", err.Error())
	}
}

func TestValidateProviderRejectsUnsupportedType(t *testing.T) {
	t.Parallel()

	_, err := Normalize(RawProvider{
		Issuer:     "GitHub",
		Type:       "HOTP",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	})
	if err == nil {
		t.Fatal("Normalize returned nil error for unsupported type")
	}
	if !strings.Contains(err.Error(), "type") {
		t.Fatalf("error = %q, want mention type", err.Error())
	}
}

func TestDisplayLabelFallsBackToSingleAvailableField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider Provider
		want     string
	}{
		{name: "issuer and name", provider: Provider{Issuer: "GitHub", Name: "trezorg"}, want: "GitHub/trezorg"},
		{name: "issuer only", provider: Provider{Issuer: "GitHub"}, want: "GitHub"},
		{name: "name only", provider: Provider{Name: "trezorg"}, want: "trezorg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.provider.DisplayLabel(); got != tt.want {
				t.Fatalf("DisplayLabel = %q, want %q", got, tt.want)
			}
		})
	}
}
