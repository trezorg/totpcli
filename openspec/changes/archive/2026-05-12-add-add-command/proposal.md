## Why

Users can import provider records from files or QR exports, but cannot add a single TOTP provider directly from the command line. A direct `add` command makes SQLite-backed storage useful for manual setup, scripting, and correcting small provider sets without preparing an import file.

## What Changes

- Add a `totpcli add` subcommand that creates one provider record in SQLite storage.
- Accept provider fields as CLI flags and reuse existing provider normalization where possible.
- Require `--totp-secret` and at least one display identity flag, `--issuer` or `--name`.
- Make `--algorithm` and `--digits` optional by deriving them from existing provider defaults: `SHA1` and `SIX`.
- Derive the stored provider `type` as `TOTP`; no `--type` flag is exposed because the app does not support creating non-TOTP records.
- Resolve the target SQLite database from `--totp`, `TOTPCLI_TOTP_PATH`, or `~/totp.sqlite`.
- Update duplicate manual adds using the same duplicate key as imports.

## Capabilities

### New Capabilities
- `totp-add`: Defines direct creation of a single TOTP provider record from CLI parameters.

### Modified Capabilities
- `cli-configuration`: Add `add` to the Cobra command surface and help behavior.
- `totp-sqlite-storage`: Add repository behavior for single-provider add/update by duplicate key.

## Impact

- Affected code: `internal/cli`, `internal/storage`, and focused CLI/storage tests.
- User-facing API: new `totpcli add` command and flags.
- Storage: no schema migration expected; the existing `providers` table already contains the required fields and duplicate key.
- Dependencies: no new third-party dependency expected.
