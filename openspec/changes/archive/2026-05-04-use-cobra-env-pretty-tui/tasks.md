## 1. Dependencies and Command Structure

- [x] 1.1 Add Cobra to the Go module and tidy module files.
- [x] 1.2 Replace direct `flag` parsing in `internal/cli.App.Run` with a Cobra root command named `totpcli`.
- [x] 1.3 Preserve the existing testable `App.Run(args []string) int` boundary and injected stdout, stderr, home, and TUI runner dependencies.

## 2. CLI Configuration Behavior

- [x] 2.1 Define `--totp`, `--format`, and `--search` Cobra flags with help text that documents related environment variables.
- [x] 2.2 Implement effective configuration resolution where changed flags override `TOTPCLI_TOTP_PATH`, `TOTPCLI_FORMAT`, and `TOTPCLI_SEARCH`, and environment variables override defaults.
- [x] 2.3 Preserve usage-error handling for invalid flags and invalid formats with non-zero usage exit status.
- [x] 2.4 Add CLI tests for flag compatibility, env fallback behavior, flag-over-env precedence, help output, and unknown flag errors.

## 3. Provider Resolution Coverage

- [x] 3.1 Ensure provider path resolution receives the effective path from CLI configuration and continues to expand `~`.
- [x] 3.2 Ensure provider format resolution receives the effective format from CLI configuration and continues to detect format from path when no explicit flag or env format exists.
- [x] 3.3 Add or update tests covering `TOTPCLI_TOTP_PATH`, `TOTPCLI_FORMAT`, flag precedence, and default path behavior.

## 4. TUI Styling

- [x] 4.1 Add a small Lip Gloss style schema for provider rows, selected rows, generated codes, remaining time, empty results, help text, and errors.
- [x] 4.2 Update `Model.View` to render through the style schema while preserving visible provider labels, code text, remaining seconds, and error messages.
- [x] 4.3 Add or update TUI tests so semantic content remains readable after styling and selected, empty, code, and error states are covered.

## 5. Documentation and Verification

- [x] 5.1 Update README usage documentation with Cobra help behavior and environment variable names.
- [x] 5.2 Run `gofmt` on edited Go files.
- [x] 5.3 Run `go test ./...`.
- [x] 5.4 Run project pre-commit checks if a project check command is available.
