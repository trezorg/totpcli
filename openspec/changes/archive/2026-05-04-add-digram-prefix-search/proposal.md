## Why

Provider lists can grow large enough that the current lexical scan becomes less precise and less interesting as the only search option. A digram-based prefix search tree gives the CLI a faster default engine with room for ranked, typo-tolerant matching while preserving the existing pluggable search boundary.

## What Changes

- Add a digram prefix search engine that indexes provider issuer, name, and display label terms.
- Make the digram prefix search engine the default provider search engine.
- Keep lexical search available as an explicit fallback engine.
- Add CLI and environment configuration for choosing the search engine.
- Reject unsupported search engine names with a usage error before launching the TUI.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `totp-search`: add the digram prefix search engine, default engine behavior, and selectable search engine semantics.
- `cli-configuration`: add command-line and environment configuration for the search engine selector.

## Impact

- Affected specs: `openspec/specs/totp-search/spec.md`, `openspec/specs/cli-configuration/spec.md`.
- Affected code: `internal/search`, `internal/cli`, TUI model construction, and related tests.
- External dependencies: none expected; existing Go standard library data structures are sufficient for the initial implementation.
