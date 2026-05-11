## Context

`totpcli` currently exposes one Cobra root command that loads provider records, optionally filters them, and starts either the TUI or non-interactive token output. Provider loading already resolves `--totp`, `TOTPCLI_TOTP_PATH`, defaults, and JSON/YAML/TOML parsing. The requested change adds a second workflow: importing Google Authenticator export QR images into the provider JSON file used by the existing TOTP workflow.

Google Authenticator export QR codes contain an `otpauth-migration://offline?data=...` URI. The `data` query parameter is URL-decoded base64 protobuf data with account fields such as secret bytes, name, issuer, algorithm, digits, type, and batching metadata. The existing `/home/igor/development/google-authenticator-exporter` project validates that conversion shape and shows that `totpSecret` must be base32-encoded from the raw secret bytes. Observed Google Authenticator exports can use migration payload version 1 or 2 with the same account field layout used here.

## Goals / Non-Goals

**Goals:**

- Preserve `totpcli` with no subcommand as the default TOTP display behavior.
- Add `totpcli totp` as an explicit command for the current TOTP display behavior.
- Add `totpcli import <qr-image>` to print Google Authenticator export QR image contents as provider JSON, or write them to a provider JSON file when `--totp` is provided.
- Write imported providers as JSON records compatible with the existing provider loader.
- Support default merge behavior with deterministic deduplication.
- Support override behavior that replaces the output file with only imported providers.
- Keep QR image decoding, migration payload decoding, provider conversion, and file writing independently testable.

**Non-Goals:**

- Importing YAML or TOML output files.
- Importing HOTP accounts.
- Interactive prompts during import.
- Collecting multiple Google Authenticator batch QR images in one command.
- Reusing or copying GPL-licensed protobuf source from the Node exporter.

## Decisions

### Add subcommands while preserving root behavior

The Cobra root command will keep the existing flags and default `RunE`, so `totpcli --totp providers.json` continues to work. A `totp` subcommand will call the same execution path and accept the same TOTP display flags. An `import` subcommand will own import-specific arguments and flags.

Alternative considered: require all users to switch to `totpcli totp`. That would make the command tree cleaner but would break existing usage and scripts.

### Keep import output JSON-only and pipeable by default

The import command will print imported provider JSON to stdout when `--totp` is omitted or set to `-`. When `--totp` names a path, import reads and writes that target as JSON. This keeps the command pipeable by default, matches the requested resulting JSON shape, and avoids introducing write semantics for YAML and TOML before they are needed.

Alternative considered: use `--format` and write any supported provider format. That increases scope because TOML/YAML round trips need separate serialization rules and tests.

### Split import into small services

Implement import as separate units:

- QR image decoder: reads an image path and returns the decoded URI string, using the pure-Go decoder first and falling back to local `zbarimg` when available for high-density screenshots that the pure-Go detector cannot read.
- Migration payload decoder: parses the `otpauth-migration` URI and protobuf wire payload.
- Provider converter: maps Google Authenticator accounts into `provider.RawProvider` or `provider.Provider` records.
- Provider store writer: loads existing JSON providers, applies merge or override, and writes JSON atomically.

This keeps command tests focused on routing and exit codes while importer tests can cover fixtures and edge cases without running the full CLI.

Alternative considered: implement import directly inside `internal/cli`. That is faster initially but would mix Cobra concerns, image decoding, protobuf parsing, deduplication, and file writes in one place.

### Decode protobuf wire data without copying external proto files

Use `google.golang.org/protobuf/encoding/protowire` or an equivalent minimal decoder for the known Google Authenticator migration fields. The project already has protobuf available indirectly, and manual wire decoding avoids committing the GPL-licensed `.proto` from the reference exporter. Accept payload versions 1 and 2 for the observed compatible field layout.

Alternative considered: generate Go code from the reference `.proto`. That is straightforward technically, but copying a GPL-licensed proto into this project creates a licensing risk.

### Deduplicate on normalized provider identity

Merge mode will append imported providers only when no existing provider has the same trimmed issuer, trimmed name, and normalized `totpSecret`. Existing records keep their order. Imported records are appended in QR payload order.

Alternative considered: deduplicate by `totpSecret` only. That can collapse distinct entries that intentionally share a secret. Deduplicating by display fields plus secret is conservative and predictable.

### Use atomic JSON writes

Import writes will marshal pretty JSON, write to a temporary file in the output directory, and rename it over the target. Merge mode will treat a missing target as an empty provider list. Parse or validation failures will leave the existing target unchanged.

Alternative considered: write directly to the target file. That is simpler but risks corrupting the provider file if the process fails during write.

## Risks / Trade-offs

- QR decoding may fail on low-quality photos -> Return an actionable error that names the image path and confirms no provider file was changed.
- Google Authenticator batches can span multiple QR codes -> Import one QR image per command and report the batch index/size when present so users know more images may remain.
- Manual protobuf decoding can miss future Google fields -> Ignore unknown fields and fail only on unsupported values required for TOTP generation.
- JSON-only output may surprise users with YAML/TOML provider files -> Document import as JSON-only in command help and print JSON to stdout unless an explicit `--totp` file path is provided.
- Unsupported HOTP, MD5, or 7-digit accounts may appear in exports -> Reject unsupported accounts with provider-specific errors rather than writing records the TOTP generator cannot use.

## Migration Plan

No data migration is required. Existing `totpcli` root usage remains valid. The implementation can be rolled back by removing the `totp` and `import` subcommands, the importer package, and import-specific tests without changing the existing provider loader contract.

## Open Questions

None. The initial implementation should prefer the small internal importer pipeline and can choose exact package names while preserving these contracts.
