## 1. Storage

- [x] 1.1 Add repository API support for saving one provider and updating duplicates.
- [x] 1.2 Implement SQLite single-provider insert using existing normalization and canonical TOTP secret duplicate key behavior.
- [x] 1.3 Add storage tests for successful insert, duplicate update, and invalid provider rejection.

## 2. CLI Command

- [x] 2.1 Add `add` command options for `--totp`, `--issuer`, `--name`, `--totp-secret`, `--secret`, `--algorithm`, and `--digits`.
- [x] 2.2 Validate `add` input before opening or writing storage, including required `--totp-secret`, valid base32 TOTP secret with algorithm-specific decoded key length, required issuer-or-name identity, supported algorithm, and supported digits.
- [x] 2.3 Resolve the add database path from `--totp`, `TOTPCLI_TOTP_PATH`, or `~/totp.sqlite`.
- [x] 2.4 Persist the provider through the repository insert method with derived `type=TOTP`.
- [x] 2.5 Print a concise success summary with output database path and provider label.

## 3. CLI Tests

- [x] 3.1 Add CLI tests for successful add with explicit database path and defaulted algorithm, digits, secret, and type.
- [x] 3.2 Add CLI tests for issuer-only and name-only provider identity.
- [x] 3.3 Add CLI tests for environment and default database path resolution.
- [x] 3.4 Add CLI tests for validation failures and unchanged storage on missing secret, missing identity, invalid TOTP secret, base32 encoded short key, algorithm-specific short keys, unsupported algorithm, unsupported digits, duplicate provider update, `--type`, and `--format`.
- [x] 3.5 Add help tests that list the `add` command and add flags without listing `--format` or `--type`.

## 4. Verification

- [x] 4.1 Run `make test`.
- [x] 4.2 Run `make lint`.
