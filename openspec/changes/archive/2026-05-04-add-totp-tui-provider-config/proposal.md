## Why

The CLI needs a usable interface for browsing configured TOTP providers, filtering them quickly, and generating one-time codes without manually invoking external tools. Provider configuration should support the existing JSON shape while allowing YAML and TOML for users who prefer those formats.

## What Changes

- Add a TOTP provider configuration loader that reads JSON by default and can also process YAML and TOML.
- Add `--format json|yaml|toml` with automatic format detection when it is omitted.
- Add `--totp` to set the provider file path, defaulting to `~/totp.{format}` using JSON when format is not otherwise known.
- Support provider entries containing `secret`, `name`, `issuer`, `algorithm`, `digits`, `type`, and `totpSecret`.
- Add a TUI that lists providers as `issuer/name`, or just `issuer` when `name` is absent.
- Add lexical filtering from an input field, initialized by optional `--search`.
- Define search through a pluggable search engine interface so semantic search can be added later.
- Generate TOTP codes from the selected provider using a Go library if available, with external `oathtool` use only as a fallback decision during implementation.
- Show the selected TOTP code and remaining validity time.

## Capabilities

### New Capabilities
- `totp-provider-config`: Loading and validating TOTP provider files across JSON, YAML, and TOML formats.
- `totp-search`: Filtering provider records through a pluggable search engine, initially lexical only.
- `totp-tui`: Terminal UI behavior for browsing providers, selecting one, and showing the generated TOTP code with remaining time.

### Modified Capabilities

None.

## Impact

- CLI flags and startup behavior for provider file location, config format, and initial search text.
- New configuration parsing dependencies for YAML and TOML.
- New TUI dependency for terminal rendering and interaction.
- New TOTP generation dependency if an appropriate Go library is selected during implementation.
- Tests covering config parsing, format detection, search behavior, TOTP option mapping, and TUI-facing state transitions.
