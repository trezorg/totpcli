## Why

Provider files are currently both the import format and the runtime storage format, which makes duplicate handling, updates, and future storage behavior depend on rewriting structured files. SQLite gives the CLI a durable local store with explicit schema ownership while keeping JSON/YAML/TOML files as import inputs.

## What Changes

- **BREAKING**: Remove `--format` from the root TOTP workflow and `totp` command; TOTP display reads providers from SQLite instead of parsing provider files directly.
- **BREAKING**: Change the `import` command so it writes imported QR providers into SQLite storage instead of a JSON provider file.
- Add an `import-file` command that imports one or more JSON/YAML/TOML provider files into SQLite storage.
- Move file format selection to `import-file --format`; when omitted, import-file detects format from each input file extension.
- Resolve the default SQLite database path from the input provider filename for `totp` and `import`, replacing the extension with `.sqlite`.
- Create the SQLite database file and provider table schema on startup when the database does not already exist.
- Detect duplicate providers during import and override existing records with imported values.
- Introduce a repository pattern for all SQLite reads, writes, duplicate checks, and schema initialization.

## Capabilities

### New Capabilities
- `totp-sqlite-storage`: SQLite database path resolution, schema initialization, provider repository behavior, and duplicate override semantics.

### Modified Capabilities
- `cli-configuration`: TOTP, import, and import-file command flags, defaults, and help behavior change around database-backed storage.
- `totp-provider-config`: Provider loading changes from direct file parsing to SQLite-backed runtime storage while preserving provider validation and normalization semantics.
- `totp-import`: Import behavior writes QR and file-based imports into SQLite storage, with file format handling moved to `import-file`.

## Impact

- CLI commands and help text in `internal/cli`.
- Provider parsing and loading boundaries in `internal/provider`.
- Import service behavior in `internal/importer`.
- New SQLite repository package or module for schema creation, provider persistence, duplicate detection, and lookup.
- New SQLite driver dependency.
- README usage, tests, and OpenSpec specs for changed command contracts.
