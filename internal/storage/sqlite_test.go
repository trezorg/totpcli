package storage

import (
	"path/filepath"
	"testing"

	"github.com/trezorg/totpcli/internal/provider"
)

func TestResolveDatabasePath(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    string
		fallback string
		home     string
		want     string
	}{
		"explicit sqlite": {
			input: "/tmp/providers.sqlite",
			home:  "/home/tester",
			want:  "/tmp/providers.sqlite",
		},
		"explicit non sqlite": {
			input: "/tmp/providers.json",
			home:  "/home/tester",
			want:  "/tmp/providers.sqlite",
		},
		"explicit no extension": {
			input: "/tmp/providers",
			home:  "/home/tester",
			want:  "/tmp/providers.sqlite",
		},
		"home expansion": {
			input: "~/totp.json",
			home:  "/home/tester",
			want:  "/home/tester/totp.sqlite",
		},
		"fallback": {
			fallback: "~/totp.json",
			home:     "/home/tester",
			want:     "/home/tester/totp.sqlite",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := ResolveDatabasePath(tt.input, tt.fallback, tt.home)
			if err != nil {
				t.Fatalf("ResolveDatabasePath returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("path = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSQLiteRepositoryInitializesSchemaAndListsProviders(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "providers.sqlite")
	repo, err := OpenSQLite(path)
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := repo.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	})

	if err := repo.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	result, err := repo.UpsertProviders([]provider.RawProvider{{
		Name:       " trezorg ",
		Issuer:     " GitHub ",
		Algorithm:  "SHA1",
		Digits:     "SIX",
		Type:       "TOTP",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	}})
	if err != nil {
		t.Fatalf("UpsertProviders returned error: %v", err)
	}
	if result.Imported != 1 || result.Added != 1 || result.Overridden != 0 || result.Final != 1 {
		t.Fatalf("result = %#v, want imported=1 added=1 overridden=0 final=1", result)
	}

	providers, err := repo.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("len(providers) = %d, want 1", len(providers))
	}
	got := providers[0]
	if got.DisplayLabel() != "GitHub/trezorg" {
		t.Fatalf("DisplayLabel = %q, want GitHub/trezorg", got.DisplayLabel())
	}
	if got.Algorithm != provider.AlgorithmSHA1 {
		t.Fatalf("Algorithm = %q, want SHA1", got.Algorithm)
	}
	if got.Digits != 6 {
		t.Fatalf("Digits = %d, want 6", got.Digits)
	}
}

func TestSQLiteRepositoryOverridesDuplicateProviders(t *testing.T) {
	t.Parallel()

	repo := openTestSQLite(t)

	_, err := repo.UpsertProviders([]provider.RawProvider{{
		Name:       "trezorg",
		Issuer:     "GitHub",
		Algorithm:  "SHA1",
		Digits:     "SIX",
		Type:       "TOTP",
		TOTPSecret: "NBUQ====",
	}})
	if err != nil {
		t.Fatalf("UpsertProviders initial returned error: %v", err)
	}

	result, err := repo.UpsertProviders([]provider.RawProvider{{
		Name:       " trezorg ",
		Issuer:     " GitHub ",
		Algorithm:  "SHA256",
		Digits:     "EIGHT",
		Type:       "TOTP",
		TOTPSecret: "NBUQ",
	}})
	if err != nil {
		t.Fatalf("UpsertProviders duplicate returned error: %v", err)
	}
	if result.Imported != 1 || result.Added != 0 || result.Overridden != 1 || result.Final != 1 {
		t.Fatalf("result = %#v, want imported=1 added=0 overridden=1 final=1", result)
	}

	providers, err := repo.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("len(providers) = %d, want 1", len(providers))
	}
	if providers[0].Algorithm != provider.AlgorithmSHA256 {
		t.Fatalf("Algorithm = %q, want SHA256", providers[0].Algorithm)
	}
	if providers[0].Digits != 8 {
		t.Fatalf("Digits = %d, want 8", providers[0].Digits)
	}
}

func TestSQLiteRepositoryRollsBackInvalidImport(t *testing.T) {
	t.Parallel()

	repo := openTestSQLite(t)

	if _, err := repo.UpsertProviders([]provider.RawProvider{{
		Name:       "trezorg",
		Issuer:     "GitHub",
		Algorithm:  "SHA1",
		Digits:     "SIX",
		Type:       "TOTP",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	}}); err != nil {
		t.Fatalf("UpsertProviders initial returned error: %v", err)
	}

	_, err := repo.UpsertProviders([]provider.RawProvider{
		{Name: "root", Issuer: "AWS", Algorithm: "SHA1", Digits: "SIX", Type: "TOTP", TOTPSecret: "JBSWY3DPEHPK3PXP"},
		{Name: "broken", Issuer: "GitHub", Algorithm: "MD5", Digits: "SIX", Type: "TOTP", TOTPSecret: "JBSWY3DPEHPK3PXP"},
	})
	if err == nil {
		t.Fatal("UpsertProviders returned nil error for invalid provider")
	}

	providers, err := repo.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	if len(providers) != 1 || providers[0].DisplayLabel() != "GitHub/trezorg" {
		t.Fatalf("providers = %#v, want original provider only", providers)
	}
}

func openTestSQLite(t *testing.T) *SQLiteRepository {
	t.Helper()

	repo, err := OpenSQLite(filepath.Join(t.TempDir(), "providers.sqlite"))
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := repo.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	})
	if err := repo.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	return repo
}
