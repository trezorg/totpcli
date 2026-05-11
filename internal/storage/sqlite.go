package storage

import (
	"database/sql"
	"encoding/base32"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/trezorg/totpcli/internal/provider"
)

type ImportResult struct {
	OutputPath string
	Imported   int
	Added      int
	Overridden int
	Final      int
}

type Repository interface {
	Init() error
	ListProviders() ([]provider.Provider, error)
	UpsertProviders([]provider.RawProvider) (ImportResult, error)
}

type SQLiteRepository struct {
	path string
	db   *sql.DB
}

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

func OpenSQLite(path string) (*SQLiteRepository, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("database path is required")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database %q: %w", path, err)
	}
	return &SQLiteRepository{path: path, db: db}, nil
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

func (r *SQLiteRepository) ListProviders() ([]provider.Provider, error) {
	rows, err := r.db.Query(`
SELECT secret, name, issuer, algorithm, digits, type, totp_secret
FROM providers
ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var providers []provider.Provider
	for rows.Next() {
		var raw provider.RawProvider
		var digits int
		if err := rows.Scan(&raw.Secret, &raw.Name, &raw.Issuer, &raw.Algorithm, &digits, &raw.Type, &raw.TOTPSecret); err != nil {
			return nil, fmt.Errorf("scan provider: %w", err)
		}
		raw.Digits = strconv.Itoa(digits)
		normalized, err := provider.Normalize(raw)
		if err != nil {
			return nil, fmt.Errorf("stored provider %d: %w", len(providers)+1, err)
		}
		providers = append(providers, normalized)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate providers: %w", err)
	}
	return providers, nil
}

func (r *SQLiteRepository) UpsertProviders(imports []provider.RawProvider) (ImportResult, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return ImportResult{}, fmt.Errorf("begin provider import: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	result := ImportResult{OutputPath: r.path, Imported: len(imports)}
	for i, raw := range imports {
		normalized, err := provider.Normalize(raw)
		if err != nil {
			return ImportResult{}, fmt.Errorf("provider %d: %w", i+1, err)
		}
		key := canonicalBase32Secret(normalized.TOTPSecret)
		var id int64
		err = tx.QueryRow(`
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
				return ImportResult{}, fmt.Errorf("insert provider %d: %w", i+1, err)
			}
			result.Added++
		case err != nil:
			return ImportResult{}, fmt.Errorf("check duplicate provider %d: %w", i+1, err)
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
				return ImportResult{}, fmt.Errorf("update provider %d: %w", i+1, err)
			}
			result.Overridden++
		}
	}

	if err := tx.QueryRow(`SELECT COUNT(*) FROM providers`).Scan(&result.Final); err != nil {
		return ImportResult{}, fmt.Errorf("count providers: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ImportResult{}, fmt.Errorf("commit provider import: %w", err)
	}
	committed = true
	return result, nil
}

func canonicalBase32Secret(secret string) string {
	normalized := strings.ToUpper(strings.TrimSpace(secret))
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.TrimRight(normalized, "="))
	if err != nil {
		return normalized
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(decoded)
}
