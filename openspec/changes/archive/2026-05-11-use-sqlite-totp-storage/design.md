## Context

The CLI currently treats JSON/YAML/TOML provider files as the runtime source of truth. The root command and `totp` command load providers directly from the selected file, while the `import` command decodes Google Authenticator QR exports and writes JSON provider files. A recently added importer package already separates QR decoding, payload conversion, and JSON writing, but persistence is still file-oriented.

This change makes SQLite the runtime store. Provider files remain supported only as import inputs through a new `import-file` command. The implementation must preserve provider normalization rules, search behavior, TOTP generation behavior, and no-TUI output while changing how providers are loaded and imported.

## Goals / Non-Goals

**Goals:**

- Use a SQLite database as the source of truth for providers used by `totpcli` and `totpcli totp`.
- Initialize the database file and provider schema automatically before reads or writes.
- Keep all SQL access behind a repository boundary.
- Import Google Authenticator QR providers and file providers into SQLite with duplicate override semantics.
- Move JSON/YAML/TOML format selection to `import-file --format`, with extension-based discovery when omitted.
- Keep provider validation and TOTP generation behavior compatible with the existing provider model.

**Non-Goals:**

- Migrating an existing provider file automatically during normal TOTP display.
- Supporting remote databases or multi-user synchronization.
- Encrypting the SQLite database.
- Adding schema migrations beyond the initial schema creation needed for this change.

## Decisions

### Store normalized providers in a `providers` table

The database table will mirror the current provider fields: `secret`, `name`, `issuer`, `algorithm`, `digits`, `type`, and `totp_secret`. It will also include an integer primary key and timestamps for stable updates. `algorithm`, `digits`, and `type` will be stored in normalized runtime values (`SHA1`/`SHA256`/`SHA512`, `6`/`8`, `TOTP`) so reads can construct `provider.Provider` without reinterpreting legacy spellings.

Alternative considered: store raw JSON blobs per provider. That would reduce schema work but would hide duplicate keys, make SQL-level upsert harder, and weaken the repository boundary.

### Use a repository interface for persistence

The CLI and importer will depend on a provider repository with methods for schema initialization, listing providers, and upserting providers. A SQLite implementation will own the SQL driver, connection setup, `CREATE TABLE IF NOT EXISTS`, and transactions. Tests can use a fake repository for CLI behavior and a temporary SQLite database for repository behavior.

Alternative considered: call SQL directly from command handlers. That would be faster to implement but would spread persistence concerns through command parsing and make duplicate behavior harder to test in isolation.

### Derive database paths from existing filename inputs

`--totp` remains the user-facing storage selector, but it resolves to a SQLite path. If the selected path does not already end in `.sqlite`, the CLI replaces its extension with `.sqlite`. Without `--totp`, `totp` uses the current default provider filename base, `~/totp.sqlite`; `import` derives the database path from the QR image filename; `import-file` derives it from the first provider file argument.

Alternative considered: introduce a new `--database` flag and deprecate `--totp`. The requested change only removes `--format`, and keeping `--totp` limits the surface area while preserving existing environment-variable fallback behavior.

### Override duplicates by normalized provider identity

Imports will identify duplicates by trimmed issuer, trimmed name, and normalized `totpSecret`, matching the current merge duplicate definition. SQLite will enforce this through repository upsert logic, ideally backed by a unique index over the normalized duplicate key columns. When an import contains a duplicate, the imported record replaces mutable provider values instead of being skipped.

Alternative considered: preserve the existing skip-on-duplicate behavior. The requested behavior explicitly calls for overriding duplicates on import.

### Keep file parsing in `provider`, move file import orchestration to `import-file`

The existing JSON/YAML/TOML parsing code remains useful for `import-file`. Runtime TOTP loading will use the repository, while file parsing functions stay available for import-file conversion and validation.

Alternative considered: move all parsing into the importer package. That would couple QR import and provider-file import unnecessarily.

## Risks / Trade-offs

- Existing users may expect `--totp providers.json --format json` to load a file directly -> Remove `--format` from TOTP commands and document that provider files must be imported with `import-file`.
- SQLite driver choice can affect build portability -> Prefer a Go SQLite driver compatible with the project toolchain and cover setup in tests.
- Deriving database paths from image filenames may surprise users importing many QR images -> Keep explicit `--totp` as the override for all commands.
- Unique duplicate matching can be affected by whitespace and case choices -> Normalize the duplicate key consistently before upsert and cover it with repository tests.
- Auto-creating an empty database can hide a typo in a path -> The CLI should still report parent directory and permission errors from SQLite open/schema initialization.

## Migration Plan

1. Add the repository package, SQLite schema initialization, and tests.
2. Rewire TOTP loading to list providers from SQLite.
3. Rewire QR import to upsert converted providers into SQLite.
4. Add `import-file` for JSON/YAML/TOML provider files.
5. Remove TOTP/import `--format` handling and update help, README, and specs.

Rollback is straightforward before release: restore file-based provider loading and JSON importer writing, then remove the SQLite dependency and repository package.
