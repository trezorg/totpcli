## MODIFIED Requirements

### Requirement: Cobra command parsing
The CLI SHALL parse commands and flags through a Cobra root command named `totpcli`. The CLI SHALL preserve root invocation as the default TOTP display behavior, SHALL expose a `totp` subcommand for the same behavior, SHALL preserve the existing `--totp`, `--format`, `--search`, `--search-engine`, and `--no-tui` flags for TOTP display behavior, and SHALL expose an `import` subcommand for provider import behavior.

#### Scenario: Existing flags still launch the TUI
- **WHEN** the user starts the application with `--totp /tmp/providers.json --format json --search git`
- **THEN** the CLI loads `/tmp/providers.json` as JSON and starts the TUI with `git` as the initial search text

#### Scenario: TOTP subcommand launches the TUI
- **WHEN** the user starts the application with `totp --totp /tmp/providers.json --format json --search git`
- **THEN** the CLI loads `/tmp/providers.json` as JSON and starts the TUI with `git` as the initial search text

#### Scenario: Search engine flag selects engine
- **WHEN** the user starts the application with `--search-engine lexical`
- **THEN** the CLI starts the TUI with lexical provider search

#### Scenario: TOTP subcommand search engine flag selects engine
- **WHEN** the user starts the application with `totp --search-engine lexical`
- **THEN** the CLI starts the TUI with lexical provider search

#### Scenario: No TUI flag prints token output
- **WHEN** the user starts the application with `--no-tui`
- **THEN** the CLI loads providers and prints non-interactive token output instead of starting the TUI

#### Scenario: TOTP subcommand no TUI flag prints token output
- **WHEN** the user starts the application with `totp --no-tui`
- **THEN** the CLI loads providers and prints non-interactive token output instead of starting the TUI

#### Scenario: Import subcommand routes import behavior
- **WHEN** the user starts the application with `import /tmp/export.jpg --totp /tmp/providers.json`
- **THEN** the CLI runs provider import behavior instead of starting the TUI

#### Scenario: Unknown flag reports a usage error
- **WHEN** the user starts the application with an unsupported flag
- **THEN** the CLI reports a command usage error and exits with a non-zero usage status

### Requirement: Cobra help output
The CLI SHALL provide Cobra-generated help output that documents available commands, flags, and related environment variables.

#### Scenario: Help output lists supported commands
- **WHEN** the user starts the application with `--help`
- **THEN** the CLI prints help output that includes the `totp` and `import` commands

#### Scenario: Help output lists supported TOTP flags
- **WHEN** the user starts the application with `--help`
- **THEN** the CLI prints help output that includes `--totp`, `--format`, `--search`, `--search-engine`, and `--no-tui`

#### Scenario: TOTP subcommand help lists supported TOTP flags
- **WHEN** the user starts the application with `totp --help`
- **THEN** the CLI prints help output that includes `--totp`, `--format`, `--search`, `--search-engine`, and `--no-tui`

#### Scenario: Import subcommand help lists import options
- **WHEN** the user starts the application with `import --help`
- **THEN** the CLI prints help output that includes the QR image argument, `--totp`, merge behavior, and `--override`

#### Scenario: Help output lists environment variables
- **WHEN** the user starts the application with `--help`
- **THEN** the CLI help output includes `TOTPCLI_TOTP_PATH`, `TOTPCLI_FORMAT`, `TOTPCLI_SEARCH`, `TOTPCLI_SEARCH_ENGINE`, and `TOTPCLI_NO_TUI`
