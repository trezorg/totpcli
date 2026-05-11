package storage

import (
	"encoding/base64"
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

func TestSQLiteRepositoryInspectsTables(t *testing.T) {
	t.Parallel()

	repo := openTestSQLite(t)
	if _, err := repo.db.Exec(`
CREATE TABLE audit (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	score REAL NOT NULL,
	payload BLOB,
	missing TEXT
);
INSERT INTO audit (id, name, score, payload, missing) VALUES
	(2, 'second', 7.5, x'6869', NULL),
	(1, 'first', 3.25, x'627965', 'present');`); err != nil {
		t.Fatalf("seed audit table: %v", err)
	}

	tables, err := repo.InspectTables()
	if err != nil {
		t.Fatalf("InspectTables returned error: %v", err)
	}

	if len(tables) != 2 {
		t.Fatalf("len(tables) = %d, want 2: %#v", len(tables), tables)
	}
	if tables[0].Name != "audit" || tables[1].Name != "providers" {
		t.Fatalf("table order = %q, %q; want audit, providers", tables[0].Name, tables[1].Name)
	}
	if tables[0].Columns[0] != "id" || tables[0].Columns[1] != "name" || tables[0].Columns[2] != "score" || tables[0].Columns[3] != "payload" || tables[0].Columns[4] != "missing" {
		t.Fatalf("audit columns = %#v, want id/name/score/payload/missing", tables[0].Columns)
	}
	if len(tables[0].Rows) != 2 {
		t.Fatalf("len(audit rows) = %d, want 2", len(tables[0].Rows))
	}
	first := tables[0].Rows[0]
	if first["id"] != int64(1) || first["name"] != "first" || first["score"] != 3.25 || first["payload"] != base64.StdEncoding.EncodeToString([]byte("bye")) || first["missing"] != "present" {
		t.Fatalf("first audit row = %#v, want normalized values", first)
	}
	second := tables[0].Rows[1]
	if second["id"] != int64(2) || second["name"] != "second" || second["score"] != 7.5 || second["payload"] != base64.StdEncoding.EncodeToString([]byte("hi")) || second["missing"] != nil {
		t.Fatalf("second audit row = %#v, want normalized values with nil missing", second)
	}
	if len(tables[1].Rows) != 0 {
		t.Fatalf("providers rows = %#v, want empty providers table", tables[1].Rows)
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
