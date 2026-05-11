## 1. SQLite Repository

- [x] 1.1 Choose and add the SQLite driver dependency.
- [x] 1.2 Add a provider repository interface for schema initialization, provider listing, and provider upsert operations.
- [x] 1.3 Implement SQLite database opening and automatic provider table creation.
- [x] 1.4 Implement database path resolution that expands `~`, preserves `.sqlite`, and converts other extensions to `.sqlite`.
- [x] 1.5 Implement provider listing from SQLite into normalized `provider.Provider` values.
- [x] 1.6 Implement transactional import upsert with duplicate override by trimmed issuer, trimmed name, and normalized `totpSecret`.
- [x] 1.7 Add repository tests for schema creation, listing, path resolution, insert, duplicate override, and rollback on failure.

## 2. Provider and Import Services

- [x] 2.1 Preserve JSON/YAML/TOML provider parsing for import-file use while removing it from runtime TOTP loading.
- [x] 2.2 Update QR import service results to report imported, added, overridden, and final provider counts.
- [x] 2.3 Replace JSON provider writer usage with repository-backed SQLite import persistence.
- [x] 2.4 Add import-file service behavior for one or more provider files, explicit format, and extension-based format detection.
- [x] 2.5 Add tests for QR import SQLite persistence, file import parsing, duplicate override, and count reporting.

## 3. CLI Behavior

- [x] 3.1 Remove `--format` and `TOTPCLI_FORMAT` handling from root TOTP invocation and the `totp` command.
- [x] 3.2 Remove `--format` and `--override` behavior from the `import` command.
- [x] 3.3 Rewire root TOTP invocation and `totp` to initialize SQLite storage and load providers through the repository.
- [x] 3.4 Rewire `import` to derive the SQLite database path, initialize storage, import QR providers, upsert them, and print SQLite import counts.
- [x] 3.5 Add the `import-file <provider-file>...` command with `--format`, `--totp`, auto-discovery, SQLite path derivation, and import count output.
- [x] 3.6 Update Cobra help text and command descriptions for SQLite-backed storage.
- [x] 3.7 Add CLI tests for removed flags, database path precedence, TOTP repository loading, QR import, import-file, help output, and error handling.

## 4. Documentation and Verification

- [x] 4.1 Update README usage, provider file documentation, import workflow, environment variable list, and SQLite storage behavior.
- [x] 4.2 Run `gofmt` on changed Go files.
- [x] 4.3 Run `make test`.
- [x] 4.4 Run `make lint`.
- [x] 4.5 Run `openspec status --change use-sqlite-totp-storage`.
