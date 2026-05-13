package storage

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trezorg/totpcli/internal/provider"
	_ "modernc.org/sqlite"
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

func TestOpenSQLiteCreatesDatabaseWithOwnerOnlyPermissions(t *testing.T) {
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

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat database: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("database mode = %v, want 0600", got)
	}
}

func TestOpenSQLiteConfiguresBusyTimeout(t *testing.T) {
	t.Parallel()

	repo := openTestSQLite(t)

	var timeout int
	if err := repo.db.QueryRow(`PRAGMA busy_timeout`).Scan(&timeout); err != nil {
		t.Fatalf("query busy timeout: %v", err)
	}
	if timeout != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", timeout)
	}
}

func TestOpenSQLiteSerializesConnections(t *testing.T) {
	t.Parallel()

	repo := openTestSQLite(t)

	if got := repo.db.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("MaxOpenConnections = %d, want 1", got)
	}
}

func TestOpenSQLiteUsesBusyTimeoutForExistingDatabase(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "providers.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open setup database: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE setup (id INTEGER PRIMARY KEY)`); err != nil {
		_ = db.Close()
		t.Fatalf("create setup database: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close setup database: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod setup database: %v", err)
	}

	repo, err := OpenSQLite(path)
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := repo.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	})

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat database: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("database mode = %v, want 0600", got)
	}
}

func TestSQLiteRepositoryListProvidersReportsCorruptRowID(t *testing.T) {
	t.Parallel()

	repo := openTestSQLite(t)
	if _, err := repo.db.Exec(`
INSERT INTO providers (secret, name, issuer, algorithm, digits, type, totp_secret, totp_secret_key)
VALUES ('', 'broken', 'GitHub', 'MD5', 6, 'TOTP', 'JBSWY3DPEHPK3PXP', 'JBSWY3DPEHPK3PXP')`); err != nil {
		t.Fatalf("seed corrupt provider: %v", err)
	}

	_, err := repo.ListProviders()
	if err == nil {
		t.Fatal("ListProviders returned nil error for corrupt row")
	}
	if !strings.Contains(err.Error(), "stored provider id 1") {
		t.Fatalf("error = %q, want mention stored provider id 1", err.Error())
	}
	if !errors.Is(err, provider.ErrUnsupportedAlgorithm) {
		t.Fatalf("error = %q, want wrapped ErrUnsupportedAlgorithm", err.Error())
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

func TestSQLiteRepositoryRejectsInvalidCanonicalTOTPSecret(t *testing.T) {
	t.Parallel()

	repo := openTestSQLite(t)

	_, err := repo.UpsertProviders([]provider.RawProvider{{
		Name:       "trezorg",
		Issuer:     "GitHub",
		Algorithm:  "SHA1",
		Digits:     "SIX",
		Type:       "TOTP",
		TOTPSecret: "INVALID0",
	}})
	if err == nil {
		t.Fatal("UpsertProviders returned nil error for invalid base32 secret")
	}
	if !strings.Contains(err.Error(), "totp secret") {
		t.Fatalf("error = %q, want mention totp secret", err.Error())
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

func TestSQLiteRepositoryInsertsProvider(t *testing.T) {
	t.Parallel()

	repo := openTestSQLite(t)

	added, err := repo.InsertProvider(provider.RawProvider{
		Name:       " trezorg ",
		Issuer:     " GitHub ",
		TOTPSecret: "JBSWY3DPEHPK3PXP",
	})
	if err != nil {
		t.Fatalf("InsertProvider returned error: %v", err)
	}
	if added.DisplayLabel() != "GitHub/trezorg" {
		t.Fatalf("DisplayLabel = %q, want GitHub/trezorg", added.DisplayLabel())
	}
	if added.Algorithm != provider.AlgorithmSHA1 {
		t.Fatalf("Algorithm = %q, want SHA1", added.Algorithm)
	}
	if added.Digits != 6 {
		t.Fatalf("Digits = %d, want 6", added.Digits)
	}
	if added.Type != "TOTP" {
		t.Fatalf("Type = %q, want TOTP", added.Type)
	}

	providers, err := repo.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	if len(providers) != 1 || providers[0].DisplayLabel() != "GitHub/trezorg" {
		t.Fatalf("providers = %#v, want GitHub/trezorg", providers)
	}
}

func TestSQLiteRepositoryUpdatesDuplicateInsertProvider(t *testing.T) {
	t.Parallel()

	repo := openTestSQLite(t)

	if _, err := repo.InsertProvider(provider.RawProvider{
		Name:       "trezorg",
		Issuer:     "GitHub",
		Algorithm:  "SHA1",
		Digits:     "SIX",
		Type:       "TOTP",
		TOTPSecret: "NBUQ====",
	}); err != nil {
		t.Fatalf("InsertProvider initial returned error: %v", err)
	}

	_, err := repo.InsertProvider(provider.RawProvider{
		Name:       " trezorg ",
		Issuer:     " GitHub ",
		Secret:     "updated metadata",
		Algorithm:  "SHA256",
		Digits:     "EIGHT",
		Type:       "TOTP",
		TOTPSecret: "NBUQ",
	})
	if err != nil {
		t.Fatalf("InsertProvider duplicate returned error: %v", err)
	}

	providers, err := repo.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("len(providers) = %d, want 1", len(providers))
	}
	if providers[0].Algorithm != provider.AlgorithmSHA256 {
		t.Fatalf("Algorithm = %q, want updated SHA256", providers[0].Algorithm)
	}
	if providers[0].Digits != 8 {
		t.Fatalf("Digits = %d, want updated 8", providers[0].Digits)
	}
	if providers[0].Secret != "updated metadata" {
		t.Fatalf("Secret = %q, want updated metadata", providers[0].Secret)
	}
}

func TestSQLiteRepositoryRejectsInvalidInsertProvider(t *testing.T) {
	t.Parallel()

	repo := openTestSQLite(t)

	_, err := repo.InsertProvider(provider.RawProvider{
		Name:   "trezorg",
		Issuer: "GitHub",
	})
	if err == nil {
		t.Fatal("InsertProvider returned nil error for invalid provider")
	}

	providers, err := repo.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	if len(providers) != 0 {
		t.Fatalf("providers = %#v, want no providers", providers)
	}
}

func TestSQLiteRepositoryDeletesSingleProvider(t *testing.T) {
	t.Parallel()

	repo := openTestSQLite(t)
	seedTestProviders(t, repo, []provider.RawProvider{
		{Name: "trezorg", Issuer: "GitHub", Type: "TOTP", TOTPSecret: "JBSWY3DPEHPK3PXP"},
		{Issuer: "AWS", Type: "TOTP", TOTPSecret: "JBSWY3DPEHPK3PXP"},
	})

	deleted, err := repo.DeleteProvider(DeleteSelector{Issuer: " GitHub ", Name: " trezorg "})
	if err != nil {
		t.Fatalf("DeleteProvider returned error: %v", err)
	}
	if deleted.DisplayLabel() != "GitHub/trezorg" {
		t.Fatalf("deleted label = %q, want GitHub/trezorg", deleted.DisplayLabel())
	}

	providers, err := repo.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	if len(providers) != 1 || providers[0].DisplayLabel() != "AWS" {
		t.Fatalf("providers = %#v, want AWS only", providers)
	}
}

func TestSQLiteRepositoryDeletesWithCanonicalTOTPSecretSelector(t *testing.T) {
	t.Parallel()

	repo := openTestSQLite(t)
	seedTestProviders(t, repo, []provider.RawProvider{
		{Name: "trezorg", Issuer: "GitHub", Type: "TOTP", TOTPSecret: "NBUQ===="},
		{Name: "root", Issuer: "GitHub", Type: "TOTP", TOTPSecret: "JBSWY3DPEHPK3PXP"},
	})

	deleted, err := repo.DeleteProvider(DeleteSelector{Issuer: "GitHub", TOTPSecret: "nbuq"})
	if err != nil {
		t.Fatalf("DeleteProvider returned error: %v", err)
	}
	if deleted.DisplayLabel() != "GitHub/trezorg" {
		t.Fatalf("deleted label = %q, want GitHub/trezorg", deleted.DisplayLabel())
	}

	providers, err := repo.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	if len(providers) != 1 || providers[0].DisplayLabel() != "GitHub/root" {
		t.Fatalf("providers = %#v, want GitHub/root only", providers)
	}
}

func TestSQLiteRepositoryDeleteProviderErrorsWithoutDeleting(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		selector DeleteSelector
		wantErr  error
	}{
		"not found": {
			selector: DeleteSelector{Issuer: "Missing"},
			wantErr:  ErrProviderNotFound,
		},
		"ambiguous": {
			selector: DeleteSelector{Issuer: "GitHub"},
			wantErr:  ErrProviderDeleteAmbiguous,
		},
		"invalid selector": {
			selector: DeleteSelector{},
			wantErr:  ErrProviderDeleteSelectorInvalid,
		},
		"invalid totp secret": {
			selector: DeleteSelector{Issuer: "GitHub", TOTPSecret: "not-base32"},
			wantErr:  ErrProviderDeleteSelectorInvalid,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repo := openTestSQLite(t)
			seedTestProviders(t, repo, []provider.RawProvider{
				{Name: "trezorg", Issuer: "GitHub", Type: "TOTP", TOTPSecret: "JBSWY3DPEHPK3PXP"},
				{Name: "root", Issuer: "GitHub", Type: "TOTP", TOTPSecret: "NBUQ===="},
			})

			_, err := repo.DeleteProvider(tt.selector)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("DeleteProvider error = %v, want %v", err, tt.wantErr)
			}
			providers, err := repo.ListProviders()
			if err != nil {
				t.Fatalf("ListProviders returned error: %v", err)
			}
			if len(providers) != 2 {
				t.Fatalf("providers = %#v, want unchanged two providers", providers)
			}
		})
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

func seedTestProviders(t *testing.T, repo *SQLiteRepository, providers []provider.RawProvider) {
	t.Helper()

	if _, err := repo.UpsertProviders(providers); err != nil {
		t.Fatalf("seed test providers: %v", err)
	}
}
