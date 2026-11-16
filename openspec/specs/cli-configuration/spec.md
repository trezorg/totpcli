## Purpose

Define command parsing, environment variable configuration, precedence, and help behavior for the `totpcli` command.
## Requirements
### Requirement: Cobra command parsing
The CLI SHALL parse commands and flags through a Cobra root command named `totpcli`. The CLI SHALL preserve root invocation as the default TOTP display behavior, SHALL expose a `totp` subcommand for the same behavior, SHALL expose `--totp`, `--search`, `--search-engine`, and `--no-tui` flags for TOTP display behavior, SHALL expose an `add` subcommand for direct TOTP provider creation behavior, SHALL expose a `delete` subcommand for direct TOTP provider deletion behavior, SHALL expose an `import` subcommand for Google Authenticator QR provider import behavior, SHALL expose an `import-file` subcommand for provider file import behavior, SHALL expose a `show` subcommand for SQLite table inspection behavior, and SHALL expose an `export` subcommand for provider file export behavior. The CLI SHALL expose `--format` only on subcommands that define format-specific behavior, including `import-file`, `show`, and `export`. The CLI SHALL NOT expose `--format` on root invocation, the `totp` subcommand, the `add` subcommand, the `delete` subcommand, or the `import` subcommand.

#### Scenario: Existing TOTP flags still launch the TUI
- **WHEN** the user starts the application with `--totp /tmp/providers.sqlite --search git`
- **THEN** the CLI loads providers from `/tmp/providers.sqlite` and starts the TUI with `git` as the initial search text

#### Scenario: TOTP subcommand launches the TUI
- **WHEN** the user starts the application with `totp --totp /tmp/providers.sqlite --search git`
- **THEN** the CLI loads providers from `/tmp/providers.sqlite` and starts the TUI with `git` as the initial search text

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

#### Scenario: Add subcommand routes provider creation behavior
- **WHEN** the user starts the application with `add --issuer GitHub --name trezorg --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ --totp /tmp/providers.sqlite`
- **THEN** the CLI creates one TOTP provider record instead of starting the TUI

#### Scenario: Delete subcommand routes provider deletion behavior
- **WHEN** the user starts the application with `delete --issuer GitHub --name trezorg --totp /tmp/providers.sqlite`
- **THEN** the CLI deletes one matching TOTP provider record instead of starting the TUI

#### Scenario: Import subcommand routes QR import behavior
- **WHEN** the user starts the application with `import /tmp/export.jpg --totp /tmp/providers.sqlite`
- **THEN** the CLI runs QR provider import behavior instead of starting the TUI

#### Scenario: Import-file subcommand routes provider file import behavior
- **WHEN** the user starts the application with `import-file /tmp/providers.yaml --format yaml --totp /tmp/providers.sqlite`
- **THEN** the CLI imports providers from `/tmp/providers.yaml` into `/tmp/providers.sqlite`

#### Scenario: Show subcommand routes SQLite inspection behavior
- **WHEN** the user starts the application with `show --format json --totp /tmp/providers.sqlite`
- **THEN** the CLI outputs SQLite table data from `/tmp/providers.sqlite` as JSON

#### Scenario: Export subcommand routes provider export behavior
- **WHEN** the user starts the application with `export --output /tmp/backup.json --totp /tmp/providers.sqlite`
- **THEN** the CLI writes providers from `/tmp/providers.sqlite` to `/tmp/backup.json`

#### Scenario: Format flag is rejected on TOTP workflow
- **WHEN** the user starts the application with `totp --format json`
- **THEN** the CLI reports a command usage error and exits with a non-zero usage status

#### Scenario: Format flag is rejected on add workflow
- **WHEN** the user starts the application with `add --issuer GitHub --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ --format json`
- **THEN** the CLI reports a command usage error and exits with a non-zero usage status

#### Scenario: Format flag is rejected on delete workflow
- **WHEN** the user starts the application with `delete --issuer GitHub --format json`
- **THEN** the CLI reports a command usage error and exits with a non-zero usage status

#### Scenario: Unknown flag reports a usage error
- **WHEN** the user starts the application with an unsupported flag
- **THEN** the CLI reports a usage error and exits with a non-zero usage status

### Requirement: Environment variable configuration
The CLI SHALL support `TOTPCLI_TOTP_PATH`, `TOTPCLI_SEARCH`, `TOTPCLI_SEARCH_ENGINE`, and `TOTPCLI_NO_TUI` as environment variable fallbacks for provider database path, initial search text, provider search engine, and non-interactive mode. The CLI SHALL NOT support `TOTPCLI_FORMAT` for root invocation, the `totp` subcommand, or the `import` subcommand.

#### Scenario: Provider database path comes from environment
- **WHEN** `TOTPCLI_TOTP_PATH` is set and the user does not provide `--totp`
- **THEN** the CLI uses `TOTPCLI_TOTP_PATH` as the provider database path input

#### Scenario: Initial search comes from environment
- **WHEN** `TOTPCLI_SEARCH` is set and the user does not provide `--search`
- **THEN** the TUI starts with the environment value as the initial search text

#### Scenario: Search engine comes from environment
- **WHEN** `TOTPCLI_SEARCH_ENGINE` is set to `lexical` and the user does not provide `--search-engine`
- **THEN** the CLI starts the TUI with lexical provider search

#### Scenario: No TUI mode comes from environment
- **WHEN** `TOTPCLI_NO_TUI` is set to a supported truthy value and the user does not provide `--no-tui`
- **THEN** the CLI prints non-interactive token output instead of starting the TUI

### Requirement: Configuration precedence
The CLI SHALL resolve configuration using command-line flags first, environment variables second, and built-in or input-derived defaults third.

#### Scenario: Flag overrides environment variable
- **WHEN** `TOTPCLI_SEARCH` is set to `aws` and the user starts the application with `--search git`
- **THEN** the TUI starts with `git` as the initial search text

#### Scenario: Search engine flag overrides environment variable
- **WHEN** `TOTPCLI_SEARCH_ENGINE` is set to `lexical` and the user starts the application with `--search-engine digram`
- **THEN** the CLI starts the TUI with digram provider search

#### Scenario: Provider database flag overrides environment database path
- **WHEN** `TOTPCLI_TOTP_PATH` is set to `/tmp/env.sqlite` and the user starts the application with `--totp /tmp/flag.sqlite`
- **THEN** the CLI uses `/tmp/flag.sqlite` as the provider database path

#### Scenario: Environment variable overrides default database path
- **WHEN** `TOTPCLI_TOTP_PATH` is set to `/tmp/env.sqlite` and the user does not provide `--totp`
- **THEN** the CLI uses `/tmp/env.sqlite` as the provider database path

#### Scenario: Default search engine is digram
- **WHEN** the user does not provide `--search-engine` and `TOTPCLI_SEARCH_ENGINE` is unset
- **THEN** the CLI starts the TUI with digram provider search

#### Scenario: No TUI flag overrides disabled environment value
- **WHEN** `TOTPCLI_NO_TUI` is set to a supported falsy value and the user starts the application with `--no-tui`
- **THEN** the CLI prints non-interactive token output instead of starting the TUI

#### Scenario: Default mode starts TUI
- **WHEN** the user does not provide `--no-tui` and `TOTPCLI_NO_TUI` is unset
- **THEN** the CLI starts the TUI

### Requirement: Cobra help output
The CLI SHALL provide Cobra-generated help output that documents available commands, flags, and related environment variables.

#### Scenario: Help output lists supported commands
- **WHEN** the user starts the application with `--help`
- **THEN** the CLI prints help output that includes the `totp`, `add`, `delete`, `import`, `import-file`, `show`, and `export` commands

#### Scenario: Help output lists supported TOTP flags
- **WHEN** the user starts the application with `--help`
- **THEN** the CLI prints help output that includes `--totp`, `--search`, `--search-engine`, and `--no-tui`, and does not include `--format`

#### Scenario: TOTP subcommand help lists supported TOTP flags
- **WHEN** the user starts the application with `totp --help`
- **THEN** the CLI prints help output that includes `--totp`, `--search`, `--search-engine`, and `--no-tui`, and does not include `--format`

#### Scenario: Add subcommand help lists add options
- **WHEN** the user starts the application with `add --help`
- **THEN** the CLI prints help output that includes provider creation behavior, `--totp`, `--issuer`, `--name`, `--totp-secret`, `--secret`, `--algorithm`, and `--digits`
- **AND** the help output documents default `SHA1`, default `SIX`, and default `TOTP`
- **AND** the help output does not include `--format` or `--type`

#### Scenario: Delete subcommand help lists delete options
- **WHEN** the user starts the application with `delete --help`
- **THEN** the CLI prints help output that includes provider deletion behavior, `--totp`, `--issuer`, `--name`, and `--totp-secret`
- **AND** the help output does not include `--format`, `--type`, or `--id`

#### Scenario: Import subcommand help lists import options
- **WHEN** the user starts the application with `import --help`
- **THEN** the CLI prints help output that includes the QR image argument, `--totp`, and SQLite import behavior, and does not include `--format`

#### Scenario: Import-file subcommand help lists import-file options
- **WHEN** the user starts the application with `import-file --help`
- **THEN** the CLI prints help output that includes provider file arguments, `--totp`, and `--format`

#### Scenario: Show subcommand help lists show options
- **WHEN** the user starts the application with `show --help`
- **THEN** the CLI prints help output that includes SQLite table inspection behavior, `--totp`, and `--format`

#### Scenario: Export subcommand help lists export options
- **WHEN** the user starts the application with `export --help`
- **THEN** the CLI prints help output that includes provider export behavior, `--totp`, `--output`, `--format`, and `--force`, including restore guidance through `import-file`

#### Scenario: Help output lists environment variables
- **WHEN** the user starts the application with `--help`
- **THEN** the CLI help output includes `TOTPCLI_TOTP_PATH`, `TOTPCLI_SEARCH`, `TOTPCLI_SEARCH_ENGINE`, and `TOTPCLI_NO_TUI`

### Requirement: Search engine configuration validation
The CLI SHALL validate the resolved search engine name before loading the TUI. Supported values SHALL include `digram` and `lexical`.

#### Scenario: Unsupported search engine reports usage error
- **WHEN** the user starts the application with `--search-engine unknown`
- **THEN** the CLI reports a usage error and exits with a non-zero usage status

