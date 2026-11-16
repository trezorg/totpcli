## Why

Users can add, import, delete, and show TOTP providers, but they cannot create a text backup of the SQLite provider database or move providers between machines. An `export` command writes the documented provider file formats (JSON, YAML, TOML), and the existing `import-file` command already restores them, so the round trip stays inside the CLI.

## What Changes

- Add an `export` subcommand that reads all providers from the resolved SQLite database and writes them to a provider file.
- Support `--format json|yaml|toml`; when omitted, detect the format from the `--output` extension and fall back to JSON.
- Add `--output` with a default of `~/totp.<format>` and a `-o` shorthand.
- Write export files with `0600` permissions because they contain cleartext TOTP secrets.
- Refuse to overwrite an existing output file unless `--force` is set.
- Restore via the existing `import-file` command; no new import command.

## Capabilities

### New Capabilities
- `totp-export`: Defines CLI behavior for exporting stored providers to JSON, YAML, or TOML provider files, including format selection, output file safety, and summary reporting.

### Modified Capabilities
- `cli-configuration`: Adds the `export` subcommand to Cobra command parsing, help output, and flag exposure rules.
- `totp-provider-config`: Adds provider file serialization and safe file writing next to existing parsing and normalization.

## Impact

- Affected code: serialization in `internal/provider`, Cobra command setup and tests in `internal/cli`.
- Dependencies: none; reuses `gopkg.in/yaml.v3`, `github.com/pelletier/go-toml/v2`, and the standard library.
- User-facing API: new `totpcli export` command with `--totp`, `--output`, `--format`, and `--force`.
