## Purpose

Define non-interactive TOTP output behavior for command-line use without launching the TUI.

## Requirements

### Requirement: Non-interactive token output
The CLI SHALL provide a non-interactive mode that prints a table containing the current TOTP token and remaining seconds for every loaded provider that matches the resolved search query.

#### Scenario: No search prints all providers
- **WHEN** the user starts the application with `--no-tui`
- **THEN** the CLI prints one table row for each loaded provider
- **AND** each table row includes the provider display label, current TOTP token, and remaining seconds
- **AND** the table includes `Provider`, `Token`, and `Remaining` headers

#### Scenario: Search filters printed providers
- **WHEN** the user starts the application with `--no-tui --search git`
- **THEN** the CLI prints table rows only for providers matching `git` through the configured search engine

#### Scenario: Empty search result prints no token rows
- **WHEN** the user starts the application with `--no-tui --search missing`
- **THEN** the CLI exits successfully without printing token rows

### Requirement: Non-interactive immediate exit
The CLI SHALL exit immediately after one non-interactive token generation pass and SHALL NOT wait for token refreshes.

#### Scenario: Command exits after printing current tokens
- **WHEN** the user starts the application with `--no-tui`
- **THEN** the CLI prints current token output
- **AND** the CLI exits without waiting for the current TOTP period to expire

### Requirement: Non-interactive generation errors
The CLI SHALL report a generation error and exit non-zero when a matching provider cannot generate a TOTP token.

#### Scenario: Invalid matched provider fails command
- **WHEN** the user starts the application with `--no-tui` and a matched provider has an invalid TOTP secret
- **THEN** the CLI reports the generation error
- **AND** the CLI exits with a non-zero status

#### Scenario: Invalid filtered-out provider does not fail command
- **WHEN** the user starts the application with `--no-tui --search valid` and an invalid provider does not match the search query
- **THEN** the CLI prints token output for matching valid providers
- **AND** the CLI exits successfully
