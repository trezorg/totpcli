## Why

Users need a direct way to inspect the SQLite database that backs `totpcli` without opening an external SQLite shell or writing ad hoc queries. A built-in `show` command makes debugging imports, validating storage contents, and scripting database inspection consistent with the rest of the CLI.

## What Changes

- Add a `show` subcommand that reads the resolved SQLite database and outputs every supported SQLite table.
- Add a `--format` flag on `show` with supported values `table`, `json`, `yaml`, and `toml`.
- Default `show --format` to `table` when the user does not provide a format.
- Reuse existing SQLite path resolution through `--totp` and `TOTPCLI_TOTP_PATH`.
- Add Go library support for table, JSON, YAML, and TOML output as needed.

## Capabilities

### New Capabilities
- `totp-sqlite-show`: Defines SQLite table inspection and multi-format output behavior for the `show` command.

### Modified Capabilities
- `cli-configuration`: Expose the `show` subcommand, its `--format` flag, and related help and validation behavior.

## Impact

- Affected code: Cobra command setup in `internal/cli`, SQLite access in `internal/storage`, and command tests.
- Dependencies: likely add supported Go libraries for YAML, TOML, and table rendering while using the standard library for JSON where practical.
- User-facing API: new `totpcli show` command with `--format` and existing provider database path flags.
