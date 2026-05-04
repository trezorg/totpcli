## Why

The CLI currently uses Go's `flag` package and only accepts configuration through command-line flags, which limits help output quality and makes repeated use less convenient. The TUI also works functionally but lacks structured visual emphasis for scanning providers, codes, remaining time, and errors.

## What Changes

- Replace direct `flag` parsing with a Cobra command while preserving existing `--totp`, `--format`, and `--search` behavior.
- Add environment variable support for provider file path, provider file format, and initial search text.
- Define precedence so explicit flags override environment variables, and environment variables override defaults.
- Improve CLI help/usage output through Cobra-generated command metadata.
- Update the TUI view to use a structured, colorized schema for search, provider rows, selection, generated code, remaining time, empty results, and errors.
- Keep provider loading, search matching, TOTP generation, and existing key behavior unchanged.

## Capabilities

### New Capabilities
- `cli-configuration`: Cobra-based command parsing, environment variable support, help behavior, and configuration precedence.

### Modified Capabilities
- `totp-provider-config`: Provider file path and format resolution gains environment variable fallback behavior.
- `totp-tui`: Terminal output gains structured, colorized presentation requirements while preserving existing interactions.

## Impact

- Affected code: `internal/cli`, `cmd/totpcli`, `internal/provider`, `internal/tui`, and related tests.
- Dependencies: add Cobra for command parsing; existing Bubble Tea and Lip Gloss dependencies can support the TUI presentation work.
- User-facing behavior: existing flags remain compatible, env vars provide optional defaults, help output changes to Cobra formatting, and TUI rendering becomes colorized on capable terminals.
