## 1. Provider File Serialization

- [x] 1.1 Add `Encode` for JSON, YAML, and TOML output from `[]RawProvider`, symmetric with `ParseRaw` and emitting parseable empty documents.
- [x] 1.2 Add `Provider.ToRaw` mapping digits to `SIX`/`EIGHT` words.
- [x] 1.3 Add `Save` with `0600` permissions, an `ErrFileExists` refusal without overwrite, and permission tightening on forced overwrite.
- [x] 1.4 Add provider package tests for round-tripping all formats, empty documents, digit words, permissions, and overwrite refusal.

## 2. Export CLI Command

- [x] 2.1 Add the `export` Cobra subcommand with `--totp`, `--output`/`-o`, `--format`, and `--force` flags.
- [x] 2.2 Resolve format from explicit `--format` first, then `--output` extension, defaulting to JSON, and default the output path to `~/totp.<format>`.
- [x] 2.3 Reject unsupported formats with a usage error before opening the database.
- [x] 2.4 Report refusal as an operational error mentioning `--force` while leaving the existing file intact.
- [x] 2.5 Update CLI help behavior and tests to include `export` while keeping unrelated flags rejected.

## 3. Round Trip, Docs, and Verification

- [x] 3.1 Add an export followed by `import-file` acceptance test covering algorithm/digits variants and duplicate issuer/name with distinct secrets.
- [x] 3.2 Document `export` in the README with flags and the restore command.
- [x] 3.3 Run `make all` and confirm `gofmt`, `go vet`, `golangci-lint`, and tests pass.
- [x] 3.4 Run `openspec status --change add-export-command` and confirm the change is apply-ready.
