## ADDED Requirements

### Requirement: Cobra command parsing
The CLI SHALL parse commands and flags through a Cobra root command named `totpcli`. The CLI SHALL preserve the existing `--totp`, `--format`, and `--search` flags.

#### Scenario: Existing flags still launch the TUI
- **WHEN** the user starts the application with `--totp /tmp/providers.json --format json --search git`
- **THEN** the CLI loads `/tmp/providers.json` as JSON and starts the TUI with `git` as the initial search text

#### Scenario: Unknown flag reports a usage error
- **WHEN** the user starts the application with an unsupported flag
- **THEN** the CLI reports a command usage error and exits with a non-zero usage status

### Requirement: Environment variable configuration
The CLI SHALL support `TOTPCLI_TOTP_PATH`, `TOTPCLI_FORMAT`, and `TOTPCLI_SEARCH` as environment variable fallbacks for provider path, provider format, and initial search text.

#### Scenario: Provider path comes from environment
- **WHEN** `TOTPCLI_TOTP_PATH` is set and the user does not provide `--totp`
- **THEN** the CLI uses `TOTPCLI_TOTP_PATH` as the provider file path

#### Scenario: Initial search comes from environment
- **WHEN** `TOTPCLI_SEARCH` is set and the user does not provide `--search`
- **THEN** the TUI starts with the environment value as the initial search text

### Requirement: Configuration precedence
The CLI SHALL resolve configuration using command-line flags first, environment variables second, and built-in defaults third.

#### Scenario: Flag overrides environment variable
- **WHEN** `TOTPCLI_SEARCH` is set to `aws` and the user starts the application with `--search git`
- **THEN** the TUI starts with `git` as the initial search text

#### Scenario: Environment variable overrides default
- **WHEN** `TOTPCLI_FORMAT` is set to `toml` and the user does not provide `--format`
- **THEN** the CLI resolves the provider file format as TOML

### Requirement: Cobra help output
The CLI SHALL provide Cobra-generated help output that documents available flags and related environment variables.

#### Scenario: Help output lists supported flags
- **WHEN** the user starts the application with `--help`
- **THEN** the CLI prints help output that includes `--totp`, `--format`, and `--search`

#### Scenario: Help output lists environment variables
- **WHEN** the user starts the application with `--help`
- **THEN** the CLI help output includes `TOTPCLI_TOTP_PATH`, `TOTPCLI_FORMAT`, and `TOTPCLI_SEARCH`
