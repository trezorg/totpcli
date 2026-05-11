## 1. Storage Inspection

- [x] 1.1 Add storage data types for SQLite table snapshots, including table name, ordered columns, and ordered rows.
- [x] 1.2 Implement repository table inspection that lists non-internal SQLite tables in deterministic order.
- [x] 1.3 Implement row scanning and value normalization for common SQLite values, including NULL and blobs.
- [x] 1.4 Add storage tests for table discovery, internal table exclusion, empty tables, row ordering, and value normalization.

## 2. Output Rendering

- [x] 2.1 Add show output format parsing for `table`, `json`, `yaml`, and `toml`, defaulting to `table`.
- [x] 2.2 Implement table rendering with table names and column headers for each inspected table.
- [x] 2.3 Implement JSON, YAML, and TOML rendering from the shared table snapshot structure.
- [x] 2.4 Add renderer tests that validate table output and parse machine-readable JSON, YAML, and TOML output.

## 3. CLI Integration

- [x] 3.1 Add the `show` Cobra subcommand with `--totp` and `--format` flags.
- [x] 3.2 Reuse existing database path resolution and repository initialization for `show`.
- [x] 3.3 Ensure unsupported `show --format` values fail with a usage error before table data is read.
- [x] 3.4 Update CLI help behavior and tests to include `show` while keeping `--format` rejected on root, `totp`, and `import`.

## 4. Verification

- [x] 4.1 Run Go formatting on changed Go files.
- [x] 4.2 Run `make test`.
- [x] 4.3 Run `make lint`.
- [x] 4.4 Run `openspec status --change add-show-command` and confirm the change is apply-ready.
