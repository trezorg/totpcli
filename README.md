# totpcli

Terminal UI for browsing TOTP providers, filtering them, and displaying the current TOTP code with its remaining lifetime. Providers are stored in a local SQLite database.

## Commands

```
totpcli [totp]        Browse TOTP providers (default)
totpcli add           Add a provider
totpcli delete        Delete a provider
totpcli import        Import from a Google Authenticator export QR image
totpcli import-file   Import from JSON, YAML, or TOML provider files
totpcli show          Show database contents
```

### TUI (`totpcli`, `totpcli totp`)

```sh
totpcli --totp ~/totp.sqlite --search github --search-engine digram
```

| Flag | Default | Description |
|------|---------|-------------|
| `--totp` | `~/totp.sqlite` | Provider database path |
| `--search` | | Initial search text |
| `--search-engine` | `digram` | `digram` or `lexical` |
| `--no-tui` | `false` | Print matching tokens and exit |

### Add (`totpcli add`)

```sh
totpcli add --issuer GitHub --name trezorg --totp-secret JBSWY3DPEHPK3PXP
```

| Flag | Default | Description |
|------|---------|-------------|
| `--totp` | `~/totp.sqlite` | Provider database path |
| `--issuer` | | Provider issuer |
| `--name` | | Provider account name |
| `--totp-secret` | | TOTP secret |
| `--secret` | | Secret metadata |
| `--algorithm` | `SHA1` | `SHA1`, `SHA256`, or `SHA512` |
| `--digits` | `SIX` | `SIX`, `EIGHT`, `6`, or `8` |

### Delete (`totpcli delete`)

```sh
totpcli delete --issuer GitHub --name trezorg
```

| Flag | Default | Description |
|------|---------|-------------|
| `--totp` | `~/totp.sqlite` | Provider database path |
| `--issuer` | | Provider issuer |
| `--name` | | Provider account name |
| `--totp-secret` | | TOTP secret for disambiguation |

### Import (`totpcli import`)

```sh
totpcli import ~/export.png --totp ~/totp.sqlite
```

| Flag | Default | Description |
|------|---------|-------------|
| `--totp` | `~/totp.sqlite` | Output database path |

### Import File (`totpcli import-file`)

```sh
totpcli import-file ~/totp.json --totp ~/totp.sqlite
```

| Flag | Default | Description |
|------|---------|-------------|
| `--totp` | `~/totp.sqlite` | Output database path |
| `--format` | detected from extension | `json`, `yaml`, or `toml` |

### Show (`totpcli show`)

```sh
totpcli show --format json --reveal-secrets
```

| Flag | Default | Description |
|------|---------|-------------|
| `--totp` | `~/totp.sqlite` | Provider database path |
| `--format` | `table` | `table`, `json`, `yaml`, or `toml` |
| `--reveal-secrets` | `false` | Show secret columns in full |

## Environment Variables

| Variable | Flag fallback |
|----------|-------------|
| `TOTPCLI_TOTP_PATH` | `--totp` |
| `TOTPCLI_SEARCH` | `--search` |
| `TOTPCLI_SEARCH_ENGINE` | `--search-engine` |
| `TOTPCLI_NO_TUI` | `--no-tui` |

Flags take precedence over environment variables, which take precedence over defaults.

## SQLite Storage

The database is created automatically with `0600` permissions. The `providers` table stores: `secret`, `name`, `issuer`, `algorithm`, `digits` (integer), `type`, `totp_secret`, `totp_secret_key` (canonical form for deduplication), plus `id`, `created_at`, and `updated_at`.

The database stores TOTP seeds in cleartext. Protect the file like password material: keep it on a trusted local disk, restrict backups and sync targets, and do not share `show --reveal-secrets` output.

Duplicates are matched by trimmed issuer, trimmed name, and normalized `totp_secret_key`. Existing records are overwritten with imported values.

## Provider File Formats

**JSON** — array of objects:
```json
[{"secret": "...", "name": "trezorg", "issuer": "GitHub", "algorithm": "SHA1", "digits": "SIX", "type": "TOTP", "totpSecret": "..."}]
```

**YAML** — array entries:
```yaml
- secret: "..."
  name: trezorg
  issuer: GitHub
  algorithm: SHA1
  digits: SIX
  type: TOTP
  totpSecret: "..."
```

**TOML** — `[[providers]]` array tables:
```toml
[[providers]]
secret = "..."
name = "trezorg"
issuer = "GitHub"
algorithm = "SHA1"
digits = "SIX"
type = "TOTP"
totpSecret = "..."
```

Supported algorithms: `SHA1`, `SHA256`, `SHA512`. Supported digits: `SIX`, `EIGHT`, `6`, `8`.
