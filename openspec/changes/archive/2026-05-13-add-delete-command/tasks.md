## 1. Storage Repository

- [x] 1.1 Add delete selector and typed not-found, ambiguous, and validation errors in `internal/storage`.
- [x] 1.2 Implement repository deletion by trimmed issuer/name and optional canonical TOTP secret inside one transaction.
- [x] 1.3 Return the deleted normalized provider from the storage delete operation for CLI reporting.
- [x] 1.4 Add storage tests for single delete, canonical TOTP secret matching, not found, ambiguous selector, invalid selector, and unchanged rows on failure.

## 2. CLI Command

- [x] 2.1 Extend the CLI repository interface with the delete operation.
- [x] 2.2 Add `totpcli delete` with `--totp`, `--issuer`, `--name`, and `--totp-secret` flags.
- [x] 2.3 Validate delete selectors before storage mutation, including required identity and valid optional TOTP secret.
- [x] 2.4 Resolve delete database paths from `--totp`, `TOTPCLI_TOTP_PATH`, or default `~/totp.sqlite`.
- [x] 2.5 Map storage not-found and ambiguous outcomes to clear non-zero command errors without deleting records.
- [x] 2.6 Print successful delete output containing the resolved database path and deleted provider label.

## 3. CLI Tests

- [x] 3.1 Add command routing tests proving `delete` runs provider deletion instead of TOTP display.
- [x] 3.2 Add delete success tests for issuer/name, issuer-only, name-only, TOTP secret disambiguation, explicit path, environment path, default path, and success summary output.
- [x] 3.3 Add delete failure tests for missing identity, invalid TOTP secret, not found, ambiguous selector, partial selector mismatch, and unchanged database state.
- [x] 3.4 Add help and flag tests for `delete --help`, root help command listing, and rejection of unsupported `--format`, `--type`, and `--id` flags.

## 4. Verification

- [x] 4.1 Run Go formatting on changed Go files.
- [x] 4.2 Run focused Go tests for `internal/storage` and `internal/cli`.
- [x] 4.3 Run the project checks from the documented workflow, including `make test` and `make lint`.
