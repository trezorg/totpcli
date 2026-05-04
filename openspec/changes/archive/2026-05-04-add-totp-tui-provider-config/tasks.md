## 1. Project Setup

- [x] 1.1 Add CLI, config parsing, TOTP, and TUI dependencies selected during implementation.
- [x] 1.2 Create package structure for provider config, search, TOTP generation, CLI startup, and TUI model.
- [x] 1.3 Add a main command entrypoint that can parse flags and report startup errors.

## 2. Provider Configuration

- [x] 2.1 Define raw and normalized provider types with display label, algorithm, digits, type, and secret fields.
- [x] 2.2 Implement `--format json|yaml|toml` parsing and validation.
- [x] 2.3 Implement `--totp` path resolution with `~/totp.<format>` defaults and home expansion.
- [x] 2.4 Implement extension-based format auto-detection for `.json`, `.yaml`, `.yml`, and `.toml`.
- [x] 2.5 Implement JSON, YAML, and TOML provider file parsing.
- [x] 2.6 Implement provider validation and normalization for `totpSecret`, display fields, `type`, `algorithm`, and `digits`.
- [x] 2.7 Add unit tests for path resolution, format resolution, parser behavior, and validation errors.

## 3. Search

- [x] 3.1 Define the search engine interface used by the TUI.
- [x] 3.2 Implement lexical search across issuer, name, and display label.
- [x] 3.3 Preserve provider input order for lexical search results.
- [x] 3.4 Add unit tests for empty queries, issuer matches, name matches, display label matches, case-insensitivity, and non-matches.

## 4. TOTP Generation

- [x] 4.1 Implement in-process TOTP code generation using normalized provider options.
- [x] 4.2 Map normalized SHA1, SHA256, SHA512, 6-digit, and 8-digit values to library options.
- [x] 4.3 Compute remaining seconds from the current 30-second period.
- [x] 4.4 Add unit tests for generation option mapping, invalid secrets, and remaining-time calculation.

## 5. TUI

- [x] 5.1 Build the TUI model with provider list, search input, filtered results, selected provider, generated code, and error state.
- [x] 5.2 Initialize the TUI search input from `--search`.
- [x] 5.3 Render provider labels as `issuer/name`, `issuer`, or `name` according to available fields.
- [x] 5.4 Update filtered results as the user edits the search input.
- [x] 5.5 Select the highlighted provider with Enter and generate its TOTP code.
- [x] 5.6 Refresh displayed code and remaining time when the TOTP period changes.
- [x] 5.7 Display a clear error state when code generation fails.
- [x] 5.8 Add TUI model tests for filtering, empty results, selection, code display, refresh behavior, and error display.

## 6. Integration and Verification

- [x] 6.1 Wire CLI startup to load providers, configure search, and launch the TUI.
- [x] 6.2 Add sample test fixtures for JSON, YAML, and TOML provider files.
- [x] 6.3 Run `go test ./...` and fix failures.
- [x] 6.4 Run the project pre-commit or check command if one exists.
- [x] 6.5 Update any user-facing documentation needed for `--totp`, `--format`, `--search`, and provider file examples.
