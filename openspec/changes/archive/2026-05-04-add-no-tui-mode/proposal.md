## Why

Users need a non-interactive way to get current TOTP codes from scripts or quick terminal invocations without entering the TUI. This should reuse the existing provider loading and search behavior while exiting immediately after printing the matching codes.

## What Changes

- Add a `--no-tui` flag that disables the interactive TUI.
- When `--no-tui` is enabled, print a table of each matching provider with its current token and remaining seconds in the current TOTP period.
- Apply the existing search configuration before printing output, so `--search` limits non-TUI output to matching providers.
- Exit immediately after printing non-TUI output; do not wait for token refreshes.
- Preserve existing TUI behavior when `--no-tui` is not provided.

## Capabilities

### New Capabilities
- `totp-noninteractive-output`: Non-interactive token output for loaded providers, including filtered output and remaining time.

### Modified Capabilities
- `cli-configuration`: Add parsing, help, environment fallback, and precedence behavior for `--no-tui`.

## Impact

- CLI configuration and root command execution.
- Provider loading and search pipeline reuse.
- TOTP code generation and remaining time calculation in a non-TUI presentation path.
- Tests for CLI flag parsing, non-interactive output, filtering, and immediate exit behavior.
