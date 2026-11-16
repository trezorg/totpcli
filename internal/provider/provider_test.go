package provider

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
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

func TestLoadRawPreservesProviderFileFields(t *testing.T) {
	t.Parallel()

	path := filepath.Join("testdata", "providers.json")
	raw, err := LoadRaw(path, FormatJSON)
	if err != nil {
		t.Fatalf("LoadRaw returned error: %v", err)
	}
	if len(raw) != 1 {
		t.Fatalf("len(raw) = %d, want 1", len(raw))
	}
	if raw[0].Digits != "SIX" {
		t.Fatalf("Digits = %q, want SIX", raw[0].Digits)
	}
	if raw[0].TOTPSecret != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("TOTPSecret = %q, want fixture secret", raw[0].TOTPSecret)
	}
}

func TestValidateProviderRejectsMissingSecret(t *testing.T) {
	t.Parallel()

	_, err := Normalize(RawProvider{Issuer: "GitHub", Type: "TOTP"})
	if err == nil {
		t.Fatal("Normalize returned nil error for missing TOTP secret")
	}
	if !errors.Is(err, ErrTOTPSecretRequired) {
		t.Fatalf("error = %q, want ErrTOTPSecretRequired", err.Error())
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
	if !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("error = %q, want ErrUnsupportedType", err.Error())
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

func TestEncodeRoundTripsAllFormats(t *testing.T) {
	t.Parallel()

	raw := []RawProvider{{
		Secret:     "metadata",
		Name:       "trezorg",
		Issuer:     "GitHub",
		Algorithm:  "SHA512",
		Digits:     "EIGHT",
		Type:       "TOTP",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	}}
	for _, format := range []Format{FormatJSON, FormatYAML, FormatTOML} {
		t.Run(string(format), func(t *testing.T) {
			t.Parallel()

			data, err := Encode(raw, format)
			if err != nil {
				t.Fatalf("Encode returned error: %v", err)
			}
			parsed, err := ParseRaw(data, format)
			if err != nil {
				t.Fatalf("ParseRaw returned error: %v", err)
			}
			if !reflect.DeepEqual(parsed, raw) {
				t.Fatalf("round-trip = %#v, want %#v", parsed, raw)
			}
		})
	}
}

func TestEncodeEmptyProvidersAreParseable(t *testing.T) {
	t.Parallel()

	for _, format := range []Format{FormatJSON, FormatYAML, FormatTOML} {
		t.Run(string(format), func(t *testing.T) {
			t.Parallel()

			data, err := Encode(nil, format)
			if err != nil {
				t.Fatalf("Encode returned error: %v", err)
			}
			parsed, err := ParseRaw(data, format)
			if err != nil {
				t.Fatalf("ParseRaw(empty encode %q) returned error: %v", data, err)
			}
			if len(parsed) != 0 {
				t.Fatalf("parsed = %#v, want zero providers", parsed)
			}
		})
	}
}

func TestEncodeRejectsUnsupportedFormat(t *testing.T) {
	t.Parallel()

	if _, err := Encode(nil, "xml"); err == nil {
		t.Fatal("Encode returned nil error for unsupported format")
	}
}

func TestProviderToRawMapsDigitsWords(t *testing.T) {
	t.Parallel()

	six := Provider{
		Name:       "trezorg",
		Issuer:     "GitHub",
		Algorithm:  AlgorithmSHA1,
		Digits:     6,
		Type:       "TOTP",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	}.ToRaw()
	eight := Provider{
		Name:       "ops",
		Issuer:     "GitHub",
		Algorithm:  AlgorithmSHA256,
		Digits:     8,
		Type:       "TOTP",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	}.ToRaw()

	if six.Digits != "SIX" {
		t.Fatalf("six digits = %q, want SIX", six.Digits)
	}
	if eight.Digits != "EIGHT" {
		t.Fatalf("eight digits = %q, want EIGHT", eight.Digits)
	}
	if six.Algorithm != "SHA1" || eight.Algorithm != "SHA256" {
		t.Fatalf("algorithms = %q/%q, want SHA1/SHA256", six.Algorithm, eight.Algorithm)
	}
	if six.Name != "trezorg" || six.Issuer != "GitHub" || six.Type != "TOTP" || six.TOTPSecret != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("six = %#v, want identity fields copied verbatim", six)
	}
}

func TestSaveWrites0600AndRefusesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "totp.json")

	if err := Save(path, FormatJSON, []RawProvider{}, false); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if mode := fileMode(t, path); mode != 0o600 {
		t.Fatalf("mode = %o, want 600", mode)
	}

	err := Save(path, FormatJSON, []RawProvider{}, false)
	if !errors.Is(err, ErrFileExists) {
		t.Fatalf("second Save err = %v, want ErrFileExists", err)
	}

	if err := os.WriteFile(path, []byte("precious"), 0o600); err != nil {
		t.Fatalf("write existing output: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("loosen existing output permissions: %v", err)
	}
	if err := Save(path, FormatJSON, []RawProvider{}, true); err != nil {
		t.Fatalf("forced Save returned error: %v", err)
	}
	if mode := fileMode(t, path); mode != 0o600 {
		t.Fatalf("forced mode = %o, want 600", mode)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "[]" {
		t.Fatalf("forced content = %q err = %v, want empty JSON array", data, err)
	}
}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %q: %v", path, err)
	}
	return info.Mode().Perm()
}

func TestSaveConcurrentWritersProduceSingleFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "totp.json")
	const writers = 32

	start := make(chan struct{})
	results := make(chan error, writers)
	for range writers {
		go func() {
			<-start
			results <- Save(path, FormatJSON, []RawProvider{}, false)
		}()
	}
	close(start)

	var successes, refusals int
	for range writers {
		if err := <-results; err == nil {
			successes++
		} else if errors.Is(err, ErrFileExists) {
			refusals++
		} else {
			t.Fatalf("Save returned unexpected error: %v", err)
		}
	}
	if successes != 1 || refusals != writers-1 {
		t.Fatalf("successes = %d refusals = %d, want exactly 1 success and %d ErrFileExists refusals", successes, refusals, writers-1)
	}
}

func TestSaveRefusesDanglingSymlinkWithoutOverwrite(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("create symlink: %v", err)
	}

	err := Save(link, FormatJSON, []RawProvider{}, false)
	if !errors.Is(err, ErrFileExists) {
		t.Fatalf("Save dangling symlink err = %v, want ErrFileExists", err)
	}
	if _, statErr := os.Lstat(target); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("dangling symlink target was created: %v", statErr)
	}
}

func TestSaveForcedOverwriteReplacesFileAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "totp.json")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatalf("write existing output: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("loosen existing output permissions: %v", err)
	}
	oldFile, err := os.Open(path)
	if err != nil {
		t.Fatalf("open existing output: %v", err)
	}
	defer func() {
		_ = oldFile.Close()
	}()

	if err := Save(path, FormatJSON, []RawProvider{}, true); err != nil {
		t.Fatalf("forced Save returned error: %v", err)
	}

	viaOldDescriptor := make([]byte, 3)
	n, _ := oldFile.ReadAt(viaOldDescriptor, 0)
	if string(viaOldDescriptor[:n]) != "old" {
		t.Fatalf("open descriptor sees %q, want replaced old content preserved", viaOldDescriptor[:n])
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "[]" {
		t.Fatalf("path content = %q err = %v, want empty JSON array", data, err)
	}
	if mode := fileMode(t, path); mode != 0o600 {
		t.Fatalf("mode = %o, want 600", mode)
	}
}

func TestSaveForcedOverwriteDoesNotFollowSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("create symlink: %v", err)
	}

	if err := Save(link, FormatJSON, []RawProvider{}, true); err != nil {
		t.Fatalf("forced Save returned error: %v", err)
	}

	if data, err := os.ReadFile(target); err != nil || string(data) != "keep" {
		t.Fatalf("symlink target = %q err = %v, want untouched keep", data, err)
	}
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("link mode = %v err = %v, want replaced regular file", info, err)
	}
	if data, err := os.ReadFile(link); err != nil || string(data) != "[]" {
		t.Fatalf("link content = %q err = %v, want empty JSON array", data, err)
	}
}
