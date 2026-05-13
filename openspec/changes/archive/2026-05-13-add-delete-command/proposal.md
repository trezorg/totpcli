## Why

Users can add, import, list, and inspect stored TOTP providers, but they cannot remove stale or incorrect records from the CLI. A delete command closes that lifecycle gap and keeps SQLite-backed provider storage manageable without manual database edits.

## What Changes

- Add a `totpcli delete` subcommand for removing one stored TOTP provider from the resolved SQLite database.
- Support identifying the provider to delete through the same user-facing identity fields used by provider display and creation.
- Require confirmation safeguards or explicit selection semantics so accidental deletion is avoided.
- Report successful deletion and not-found cases clearly through CLI output and exit status.
- Keep the command SQLite-backed and consistent with existing `--totp` path resolution.

## Capabilities

### New Capabilities
- `totp-delete`: Defines CLI behavior for deleting a stored TOTP provider, including identification, safety, reporting, and error handling.

### Modified Capabilities
- `cli-configuration`: Adds the `delete` subcommand to Cobra command parsing, help output, and flag exposure rules.
- `totp-sqlite-storage`: Adds repository support for deleting provider records from SQLite through application-level storage boundaries.

## Impact

- CLI command construction and tests in `internal/cli`.
- SQLite repository API and tests in `internal/storage`.
- Provider identity matching and validation rules shared with existing add/import/display workflows.
- OpenSpec requirements for command routing, help output, and SQLite repository behavior.
