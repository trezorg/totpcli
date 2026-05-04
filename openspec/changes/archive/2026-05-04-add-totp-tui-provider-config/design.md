## Context

The project is a new Go CLI with an OpenSpec setup and minimal implementation. The requested feature spans startup flags, provider file parsing, terminal interaction, search behavior, and TOTP code generation. The design should keep domain logic independent from the TUI so config loading, filtering, and code generation can be tested without terminal rendering.

Provider files contain an array of records with fields such as `secret`, `name`, `issuer`, `algorithm`, `digits`, `type`, and `totpSecret`. JSON is the default format, while YAML and TOML are supported through explicit `--format` selection or automatic detection.

## Goals / Non-Goals

**Goals:**

- Load providers from JSON, YAML, or TOML with JSON as the default.
- Resolve provider file path from `--totp`, defaulting to `~/totp.json` when neither path nor format implies another format.
- Auto-detect format from a known file extension when `--format` is not set.
- Present providers in a TUI as `issuer/name`, falling back to `issuer`.
- Filter providers using a pluggable search engine with lexical search as the first implementation.
- Initialize the filter input from `--search`.
- Generate TOTP codes in-process and display the code with remaining time.

**Non-Goals:**

- Semantic search implementation.
- Editing, creating, or deleting provider records.
- Secret storage, encryption, or migration from browser/account exports.
- HOTP support beyond rejecting unsupported provider types clearly.
- Clipboard integration or automatic code copy.

## Decisions

### Use an in-process TOTP library

Use `github.com/pquerna/otp/totp` with `totp.GenerateCodeCustom` and `totp.ValidateOpts`. It supports base32 secrets, SHA1/SHA256/SHA512 algorithms, 6- and 8-digit codes, and configurable periods. This avoids requiring `oathtool` on PATH and keeps error handling testable.

Alternative considered: shelling out to `oathtool --base32 --totp`. That matches the requested command shape but adds an external runtime dependency, platform differences, and more fragile parsing. Keep external command execution out of the initial implementation unless a library gap is discovered.

### Split provider config from TUI state

Create a provider config package or module that handles path expansion, format resolution, parsing, validation, and normalized provider records. The TUI receives already validated provider data and does not parse files directly.

Alternative considered: parse config directly in `main` or the TUI model. That is simpler initially but makes tests and future format behavior harder to isolate.

### Use extension-based auto-detection

When `--format` is absent, detect `json`, `yaml`, `yml`, or `toml` from the provider filename. If the path has no known extension, use JSON. When `--totp` is absent, use `~/totp.<format>`, defaulting to `~/totp.json`.

Alternative considered: content sniffing. It can be useful, but ambiguous syntax between JSON/YAML and TOML error recovery would make startup behavior less predictable. Extension detection is enough for the requested behavior and keeps error messages clear.

### Normalize provider records before use

Map `digits` values such as `SIX`, `EIGHT`, `6`, and `8` to numeric digits. Map `algorithm` values such as `SHA1`, `SHA256`, and `SHA512` to the TOTP library options. Require `type` to be absent or `TOTP`, require `totpSecret`, and require at least `issuer` or `name` for display.

Alternative considered: pass raw strings through to generation. Normalization gives one place for validation and prevents TUI-specific error handling from spreading across the codebase.

### Keep search pluggable

Define a small interface, for example `SearchEngine.Search(query string, providers []Provider) []ProviderMatch`, with a lexical implementation that matches case-insensitively across display label, issuer, and name. Preserve original provider order for equal lexical matches.

Alternative considered: use the built-in filtering behavior from a TUI list component as the domain search. That would couple search semantics to UI dependencies and make future semantic search harder to introduce.

### Use Bubble Tea and Bubbles for the TUI

Use Bubble Tea's model/update/view loop and Bubbles components for text input and list behavior. The TUI model owns only interaction state: input value, filtered results, selected row, selected provider, generated code, and next refresh time.

Alternative considered: hand-render terminal controls. That would avoid a dependency but creates more work for input handling, resizing, and predictable terminal behavior.

## Risks / Trade-offs

- Unsupported `digits` values beyond 6 and 8 -> Return a startup or selection error that names the provider and accepted values.
- Provider files may include both `secret` and `totpSecret` -> Use `totpSecret` for code generation and treat `secret` as ignored metadata unless future requirements define it.
- Automatic format detection may surprise users with extensionless files -> Document that JSON is the fallback and allow `--format` to override it.
- TUI tests can be brittle if they assert full rendered strings -> Prefer testing model transitions and core services, with light rendering smoke tests.
- Remaining time must stay accurate while the screen is open -> Schedule periodic ticks and compute remaining seconds from the current time rather than decrementing local state.

## Migration Plan

This is a new application behavior, so no data migration is required. Implement the provider loader and TOTP generation first, then wire CLI flags, then add the TUI. Rollback is removing the new dependencies and command path before any release.

## Open Questions

None for the proposal. The initial implementation can choose exact package names and dependency versions while preserving these contracts.
