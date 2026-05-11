## Why

Users need a supported way to move Google Authenticator exports into `totpcli` without hand-decoding QR payloads or manually editing provider JSON. Adding an import command also makes the CLI shape explicit by preserving the existing TOTP behavior under a `totp` command while keeping the current root invocation as the default path.

## What Changes

- Add a `totp` command for the current TOTP display behavior, with root `totpcli` invocation continuing to run the same behavior by default.
- Add an `import` command that accepts a Google Authenticator export QR image path and writes providers to the configured provider JSON file.
- Print imported provider JSON to stdout when `--totp` is omitted or set to `-`; write to the explicit `--totp` path when provided.
- Merge imported providers into the existing provider file by default, deduplicating existing entries.
- Add an override mode that replaces the output file with only the imported providers.
- Decode the QR image through a local QR decoding step and convert Google Authenticator export payloads into the existing provider record shape.

## Capabilities

### New Capabilities

- `totp-import`: Import Google Authenticator export QR images into the `totpcli` provider file.

### Modified Capabilities

- `cli-configuration`: Add explicit `totp` and `import` subcommands while preserving the existing root command behavior.

## Impact

- CLI command tree and help output.
- Provider file read/write behavior for JSON import targets.
- New import parsing and QR decoding path.
- Tests for command routing, import path resolution, merge/override behavior, deduplication, and Google Authenticator payload conversion.
