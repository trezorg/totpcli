package storage

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/trezorg/totpcli/internal/provider"
	"github.com/trezorg/totpcli/internal/totpsecret"
)

type ImportResult struct {
	OutputPath string
	Imported   int
	Added      int
	Overridden int
	Final      int
}

type TableSnapshot struct {
	Name    string           `json:"name" yaml:"name" toml:"name"`
	Columns []string         `json:"columns" yaml:"columns" toml:"columns"`
	Rows    []map[string]any `json:"rows" yaml:"rows" toml:"rows"`
}

var (
	ErrProviderNotFound              = errors.New("provider not found")
	ErrProviderDeleteAmbiguous       = errors.New("provider delete selector is ambiguous")
	ErrProviderDeleteSelectorInvalid = errors.New("provider delete selector is invalid")
)

type DeleteSelector struct {
	Issuer     string
	Name       string
	TOTPSecret string
}

type SQLiteRepository struct {
	path string
	db   *sql.DB
}

// ResolveDatabasePath expands ~ and ensures the path ends with .sqlite.
// If the path has a non-.sqlite extension, that extension is replaced.
func ResolveDatabasePath(input string, fallback string, home string) (string, error) {
	path := strings.TrimSpace(input)
	if path == "" {
		path = fallback
	}
	if path == "" {
		path = filepath.Join("~", "totp.sqlite")
	}

	expanded, err := provider.ExpandHome(path, home)
	if err != nil {
		return "", err
	}

	if strings.EqualFold(filepath.Ext(expanded), ".sqlite") {
		return expanded, nil
	}
	ext := filepath.Ext(expanded)
	if ext == "" {
		return expanded + ".sqlite", nil
	}
	return strings.TrimSuffix(expanded, ext) + ".sqlite", nil
}

// OpenSQLite opens a SQLite TOTP provider database. TOTP secrets are stored
// as plaintext; file permissions are restricted to 0600 as the sole
// at-rest protection.
func OpenSQLite(path string) (*SQLiteRepository, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("database path is required")
	}
	created, err := ensureSQLiteFilePermissions(path)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		if created {
			_ = os.Remove(path)
		}
		return nil, fmt.Errorf("open sqlite database %q: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	return &SQLiteRepository{path: path, db: db}, nil
}

func ensureSQLiteFilePermissions(path string) (bool, error) {
	_, statErr := os.Stat(path)
	created := errors.Is(statErr, os.ErrNotExist)
	if statErr != nil && !created {
		return false, fmt.Errorf("stat sqlite database file %q: %w", path, statErr)
	}
	if created {
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return false, fmt.Errorf("create sqlite database file %q: %w", path, err)
		}
		if err := file.Close(); err != nil {
			return true, fmt.Errorf("close sqlite database file %q: %w", path, err)
		}
		return true, nil
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return false, fmt.Errorf("restrict sqlite database file %q: %w", path, err)
	}
	return false, nil
}

func sqliteDSN(path string) string {
	return path + "?_pragma=busy_timeout%3d5000"
}

func (r *SQLiteRepository) Close() error {
	if r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *SQLiteRepository) Init() error {
	const schema = `
CREATE TABLE IF NOT EXISTS providers (
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
);`
	if _, err := r.db.Exec(schema); err != nil {
		return fmt.Errorf("initialize sqlite schema %q: %w", r.path, err)
	}
	return nil
}

func (r *SQLiteRepository) ListProviders() (providers []provider.Provider, err error) {
	rows, err := r.db.Query(`
SELECT id, secret, name, issuer, algorithm, digits, type, totp_secret
FROM providers
ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	// Named return err is set here so bare returns propagate the close error.
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close provider rows: %w", cerr)
		}
	}()

	for rows.Next() {
		var id int64
		var raw provider.RawProvider
		var digits int
		if err := rows.Scan(&id, &raw.Secret, &raw.Name, &raw.Issuer, &raw.Algorithm, &digits, &raw.Type, &raw.TOTPSecret); err != nil {
			return nil, fmt.Errorf("scan provider: %w", err)
		}
		raw.Digits = strconv.Itoa(digits)
		normalized, err := provider.Normalize(raw)
		if err != nil {
			return nil, fmt.Errorf("stored provider id %d: %w", id, err)
		}
		providers = append(providers, normalized)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate providers: %w", err)
	}
	return providers, nil
}

func (r *SQLiteRepository) InspectTables() (tables []TableSnapshot, err error) {
	rows, err := r.db.Query(`
SELECT name
FROM sqlite_schema
WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list sqlite tables: %w", err)
	}
	// Named return err is set here so bare returns propagate the close error.
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close table rows: %w", cerr)
		}
	}()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan sqlite table name: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite table names: %w", err)
	}

	tables = make([]TableSnapshot, 0, len(names))
	for _, name := range names {
		table, err := r.inspectTable(name)
		if err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func withTx(db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	committed = true
	return nil
}

func (r *SQLiteRepository) InsertProvider(raw provider.RawProvider) (provider.Provider, error) {
	normalized, err := provider.Normalize(raw)
	if err != nil {
		return provider.Provider{}, err
	}
	key, err := totpsecret.CanonicalBase32(normalized.TOTPSecret)
	if err != nil {
		return provider.Provider{}, err
	}

	if err := withTx(r.db, func(tx *sql.Tx) error {
		_, err := upsertOne(tx, normalized, key)
		return err
	}); err != nil {
		return provider.Provider{}, fmt.Errorf("insert provider: %w", err)
	}
	return normalized, nil
}

func (r *SQLiteRepository) DeleteProvider(selector DeleteSelector) (provider.Provider, error) {
	normalized, key, err := normalizeDeleteSelector(selector)
	if err != nil {
		return provider.Provider{}, err
	}

	var deleted provider.Provider
	if err := withTx(r.db, func(tx *sql.Tx) error {
		matches, err := selectDeleteCandidates(tx, normalized, key)
		if err != nil {
			return err
		}
		switch len(matches) {
		case 0:
			return ErrProviderNotFound
		case 1:
			deleted = matches[0].provider
		default:
			return ErrProviderDeleteAmbiguous
		}

		if _, err := tx.Exec(`DELETE FROM providers WHERE id = ?`, matches[0].id); err != nil {
			return fmt.Errorf("delete provider: %w", err)
		}
		return nil
	}); err != nil {
		return provider.Provider{}, fmt.Errorf("delete provider: %w", err)
	}
	return deleted, nil
}

func (r *SQLiteRepository) inspectTable(name string) (snapshot TableSnapshot, err error) {
	columns, err := r.tableColumns(name)
	if err != nil {
		return TableSnapshot{}, err
	}

	quoted, err := quoteSQLiteIdentifier(name)
	if err != nil {
		return TableSnapshot{}, fmt.Errorf("inspect sqlite table %q: %w", name, err)
	}
	query := fmt.Sprintf("SELECT * FROM %s ORDER BY rowid", quoted)
	rows, err := r.db.Query(query)
	if err != nil {
		return TableSnapshot{}, fmt.Errorf("inspect sqlite table %q: %w", name, err)
	}
	// Named return err is set here so bare returns propagate the close error.
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close inspect rows: %w", cerr)
		}
	}()

	snapshot = TableSnapshot{Name: name, Columns: columns}
	for rows.Next() {
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return TableSnapshot{}, fmt.Errorf("scan sqlite table %q row: %w", name, err)
		}

		row := make(map[string]any, len(columns))
		for i, column := range columns {
			row[column] = normalizeSQLiteValue(values[i])
		}
		snapshot.Rows = append(snapshot.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return TableSnapshot{}, fmt.Errorf("iterate sqlite table %q rows: %w", name, err)
	}
	return snapshot, nil
}

func (r *SQLiteRepository) tableColumns(name string) (columns []string, err error) {
	quoted, err := quoteSQLiteIdentifier(name)
	if err != nil {
		return nil, fmt.Errorf("inspect sqlite table %q columns: %w", name, err)
	}
	rows, err := r.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", quoted))
	if err != nil {
		return nil, fmt.Errorf("inspect sqlite table %q columns: %w", name, err)
	}
	// Named return err is set here so bare returns propagate the close error.
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close column rows: %w", cerr)
		}
	}()

	for rows.Next() {
		var cid int
		var columnName string
		var columnType string
		var notNull int
		var defaultValue any
		var primaryKey int
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, fmt.Errorf("scan sqlite table %q column: %w", name, err)
		}
		columns = append(columns, columnName)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sqlite table %q columns: %w", name, err)
	}
	return columns, nil
}

func (r *SQLiteRepository) UpsertProviders(imports []provider.RawProvider) (ImportResult, error) {
	result := ImportResult{OutputPath: r.path, Imported: len(imports)}
	if err := withTx(r.db, func(tx *sql.Tx) error {
		for i, raw := range imports {
			normalized, err := provider.Normalize(raw)
			if err != nil {
				return fmt.Errorf("provider %d: %w", i+1, err)
			}
			key, err := totpsecret.CanonicalBase32(normalized.TOTPSecret)
			if err != nil {
				return fmt.Errorf("provider %d: %w", i+1, err)
			}
			added, err := upsertOne(tx, normalized, key)
			if err != nil {
				return fmt.Errorf("upsert provider %d: %w", i+1, err)
			}
			if added {
				result.Added++
			} else {
				result.Overridden++
			}
		}

		if err := tx.QueryRow(`SELECT COUNT(*) FROM providers`).Scan(&result.Final); err != nil {
			return fmt.Errorf("count providers: %w", err)
		}
		return nil
	}); err != nil {
		return ImportResult{}, fmt.Errorf("import providers: %w", err)
	}
	return result, nil
}

// upsertOne inserts a new provider or updates an existing one matched by
// (issuer, name, totp_secret_key). On update, name and issuer are not
// overwritten because the match is on their normalized values; only the
// mutable fields (secret, algorithm, digits, type, totp_secret) change.
func upsertOne(tx *sql.Tx, normalized provider.Provider, key string) (bool, error) {
	var id int64
	err := tx.QueryRow(`
SELECT id
FROM providers
WHERE issuer = ? AND name = ? AND totp_secret_key = ?`,
		normalized.Issuer, normalized.Name, key).Scan(&id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.Exec(`
INSERT INTO providers (secret, name, issuer, algorithm, digits, type, totp_secret, totp_secret_key)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			normalized.Secret,
			normalized.Name,
			normalized.Issuer,
			string(normalized.Algorithm),
			normalized.Digits,
			normalized.Type,
			normalized.TOTPSecret,
			key,
		); err != nil {
			return false, fmt.Errorf("insert provider: %w", err)
		}
		return true, nil
	case err != nil:
		return false, fmt.Errorf("check duplicate provider: %w", err)
	default:
		if _, err := tx.Exec(`
UPDATE providers
SET secret = ?, algorithm = ?, digits = ?, type = ?, totp_secret = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?`,
			normalized.Secret,
			string(normalized.Algorithm),
			normalized.Digits,
			normalized.Type,
			normalized.TOTPSecret,
			id,
		); err != nil {
			return false, fmt.Errorf("update provider: %w", err)
		}
		return false, nil
	}
}

type deleteCandidate struct {
	id       int64
	provider provider.Provider
}

func normalizeDeleteSelector(selector DeleteSelector) (DeleteSelector, string, error) {
	normalized := DeleteSelector{
		Issuer:     strings.TrimSpace(selector.Issuer),
		Name:       strings.TrimSpace(selector.Name),
		TOTPSecret: strings.TrimSpace(selector.TOTPSecret),
	}
	if normalized.Issuer == "" && normalized.Name == "" {
		return DeleteSelector{}, "", fmt.Errorf("%w: at least one of issuer or name is required", ErrProviderDeleteSelectorInvalid)
	}
	if normalized.TOTPSecret == "" {
		return normalized, "", nil
	}
	key, err := totpsecret.CanonicalBase32(normalized.TOTPSecret)
	if err != nil {
		return DeleteSelector{}, "", fmt.Errorf("%w: %w", ErrProviderDeleteSelectorInvalid, err)
	}
	return normalized, key, nil
}

func selectDeleteCandidates(tx *sql.Tx, selector DeleteSelector, key string) (matches []deleteCandidate, err error) {
	query := `
SELECT id, secret, name, issuer, algorithm, digits, type, totp_secret
FROM providers
WHERE (? = '' OR issuer = ?)
  AND (? = '' OR name = ?)
  AND (? = '' OR totp_secret_key = ?)
ORDER BY id`
	rows, err := tx.Query(query,
		selector.Issuer, selector.Issuer,
		selector.Name, selector.Name,
		key, key,
	)
	if err != nil {
		return nil, fmt.Errorf("select provider delete candidates: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close provider delete candidates: %w", cerr)
		}
	}()

	for rows.Next() {
		var id int64
		var raw provider.RawProvider
		var digits int
		if err := rows.Scan(&id, &raw.Secret, &raw.Name, &raw.Issuer, &raw.Algorithm, &digits, &raw.Type, &raw.TOTPSecret); err != nil {
			return nil, fmt.Errorf("scan provider delete candidate: %w", err)
		}
		raw.Digits = strconv.Itoa(digits)
		normalized, err := provider.Normalize(raw)
		if err != nil {
			return nil, fmt.Errorf("stored provider id %d: %w", id, err)
		}
		matches = append(matches, deleteCandidate{id: id, provider: normalized})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider delete candidates: %w", err)
	}
	return matches, nil
}

func normalizeSQLiteValue(value any) any {
	switch typed := value.(type) {
	case []byte:
		return base64.StdEncoding.EncodeToString(typed)
	default:
		return typed
	}
}

func quoteSQLiteIdentifier(identifier string) (string, error) {
	if strings.ContainsRune(identifier, 0) {
		return "", fmt.Errorf("sqlite identifier %q contains null byte", identifier)
	}
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`, nil
}
