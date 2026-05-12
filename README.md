# totpcli

`totpcli` is a terminal UI for browsing configured TOTP providers, filtering them, and showing the current TOTP code with its remaining lifetime.

Providers are stored in a local SQLite database. JSON, YAML, and TOML provider files are supported as import inputs through `import-file`.

## Usage

Import an existing provider file:

```sh
go run ./cmd/totpcli import-file ~/totp.json
```

Open the TOTP browser:

```sh
go run ./cmd/totpcli --totp ~/totp.sqlite --search github --search-engine digram
```

Import a Google Authenticator export QR image:

```sh
go run ./cmd/totpcli import ~/export.png --totp ~/totp.sqlite
```

Flags for `totpcli` and `totpcli totp`:

- `--totp`: provider SQLite database path. Defaults to `~/totp.sqlite`.
- `--search`: initial search text for the TUI filter.
- `--search-engine`: provider search engine, either `digram` or `lexical`. Defaults to `digram`.
- `--no-tui`: print matching tokens and exit without starting the TUI.

Flags for `totpcli import`:

- `--totp`: output SQLite database path. When omitted, the database path is derived from the QR image filename with a `.sqlite` extension.

Flags for `totpcli import-file`:

- `--totp`: output SQLite database path. When omitted, the database path is derived from the first provider filename with a `.sqlite` extension.
- `--format`: provider file format, one of `json`, `yaml`, or `toml`. When omitted, the format is detected from `.json`, `.yaml`, `.yml`, or `.toml`; unknown extensions use JSON.

Flags for `totpcli show`:

- `--totp`: SQLite provider database path. Defaults to `~/totp.sqlite`.
- `--format`: output format, one of `table`, `json`, `yaml`, or `toml`. Defaults to `table`.
- `--reveal-secrets`: print secret columns in full. By default, secret columns are redacted.

Cobra provides command help:

```sh
go run ./cmd/totpcli --help
```

Environment variables can provide defaults when the matching flag is omitted:

- `TOTPCLI_TOTP_PATH`: provider SQLite database path fallback for `--totp`.
- `TOTPCLI_SEARCH`: initial provider search fallback for `--search`.
- `TOTPCLI_SEARCH_ENGINE`: provider search engine fallback for `--search-engine`.
- `TOTPCLI_NO_TUI`: non-interactive mode fallback for `--no-tui`.

Command-line flags take precedence over environment variables. Environment variables take precedence over built-in or input-derived defaults.

## SQLite Storage

The database is created automatically when it does not exist. It is created with owner-only `0600` permissions. The provider table stores the same provider fields used by the import file formats: `secret`, `name`, `issuer`, `algorithm`, `digits`, `type`, and `totpSecret`.

The SQLite database stores TOTP seeds in cleartext. Protect the database file like password material: keep it on a trusted local disk, restrict backups and sync targets, and do not share `show --reveal-secrets` output.

When importing, duplicates are matched by trimmed issuer, trimmed name, and normalized `totpSecret`. Duplicate records are overridden with the imported values.

## Provider Files

JSON provider files contain an array:

```json
[
  {
    "secret": "redacted",
    "name": "trezorg",
    "issuer": "GitHub",
    "algorithm": "SHA1",
    "digits": "SIX",
    "type": "TOTP",
    "totpSecret": "redacted"
  }
]
```

YAML uses the same field names:

```yaml
- secret: redacted
  name: trezorg
  issuer: GitHub
  algorithm: SHA1
  digits: SIX
  type: TOTP
  totpSecret: redacted
```

TOML uses provider array tables:

```toml
[[providers]]
secret = "redacted"
name = "trezorg"
issuer = "GitHub"
algorithm = "SHA1"
digits = "SIX"
type = "TOTP"
totpSecret = "redacted"
```

Supported algorithms are `SHA1`, `SHA256`, and `SHA512`. Supported digits values are `SIX`, `EIGHT`, `6`, and `8`.
