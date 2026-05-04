# totpcli

`totpcli` is a terminal UI for browsing configured TOTP providers, filtering them, and showing the current TOTP code with its remaining lifetime.

## Usage

```sh
go run ./cmd/totpcli --totp ~/totp.json --format json --search github
```

Flags:

- `--totp`: provider file path. Defaults to `~/totp.json`, or `~/totp.<format>` when `--format` is set.
- `--format`: provider file format, one of `json`, `yaml`, or `toml`. When omitted, the format is detected from `.json`, `.yaml`, `.yml`, or `.toml`; unknown extensions use JSON.
- `--search`: initial search text for the TUI filter.

## Provider Files

JSON is the default format:

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
