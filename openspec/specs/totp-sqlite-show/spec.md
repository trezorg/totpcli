## Purpose

Define SQLite table inspection behavior and multi-format output for the `show` command.

## Requirements

### Requirement: SQLite table inspection
The CLI SHALL provide a `show` command that opens the resolved SQLite provider database and outputs every non-internal SQLite table. The command SHALL include each table's name, ordered column names, and rows ordered deterministically by rowid when available. The command SHALL exclude SQLite internal tables whose names start with `sqlite_`.

#### Scenario: Show prints provider table contents
- **WHEN** the user runs `totpcli show --totp /tmp/providers.sqlite` against a database containing provider records
- **THEN** the CLI reads `/tmp/providers.sqlite`
- **AND** the output includes the `providers` table
- **AND** the output includes the provider table columns and row values

#### Scenario: Show excludes internal SQLite tables
- **WHEN** the user runs `totpcli show --totp /tmp/providers.sqlite`
- **THEN** the output does not include tables whose names start with `sqlite_`

#### Scenario: Empty table is still shown
- **WHEN** the user runs `totpcli show --totp /tmp/providers.sqlite` against a database with an empty `providers` table
- **THEN** the output includes the `providers` table name and column headers
- **AND** the command exits successfully

### Requirement: Show format selection
The `show` command SHALL support a `--format` flag with values `table`, `json`, `yaml`, and `toml`. When `--format` is omitted, the command SHALL use `table`. Unsupported formats SHALL produce a usage error and a non-zero usage status before reading table data.

#### Scenario: Default format is table
- **WHEN** the user runs `totpcli show --totp /tmp/providers.sqlite` without `--format`
- **THEN** the CLI renders SQLite table data in table format

#### Scenario: JSON format is selected
- **WHEN** the user runs `totpcli show --totp /tmp/providers.sqlite --format json`
- **THEN** the CLI renders SQLite table data as valid JSON

#### Scenario: YAML format is selected
- **WHEN** the user runs `totpcli show --totp /tmp/providers.sqlite --format yaml`
- **THEN** the CLI renders SQLite table data as valid YAML

#### Scenario: TOML format is selected
- **WHEN** the user runs `totpcli show --totp /tmp/providers.sqlite --format toml`
- **THEN** the CLI renders SQLite table data as valid TOML

#### Scenario: Unsupported show format reports usage error
- **WHEN** the user runs `totpcli show --format csv`
- **THEN** the CLI reports a usage error
- **AND** the CLI exits with a non-zero usage status

### Requirement: Show output shape
Machine-readable `show` output SHALL include a top-level collection of tables where each table has a `name`, ordered `columns`, and ordered `rows`. Each row SHALL preserve values by column name or by an equivalent TOML-compatible row representation that can be mapped back to column names. Table output SHALL include a visible table name and column headers for each table.

#### Scenario: Machine-readable output includes table metadata
- **WHEN** the user runs `totpcli show --totp /tmp/providers.sqlite --format json`
- **THEN** the output includes a top-level table collection
- **AND** each table entry includes its name, columns, and rows

#### Scenario: Table output identifies each table
- **WHEN** the user runs `totpcli show --totp /tmp/providers.sqlite --format table`
- **THEN** the output includes the table name before that table's rendered rows
- **AND** the rendered table includes column headers

### Requirement: Show error handling
The `show` command SHALL report database path, open, schema initialization, table discovery, row scanning, and output encoding failures as command errors and exit non-zero.

#### Scenario: Unreadable database path fails show
- **WHEN** the user runs `totpcli show --totp /root/providers.sqlite` and the database cannot be opened or initialized
- **THEN** the CLI reports the database error
- **AND** the CLI exits with a non-zero status

#### Scenario: Encoding failure fails show
- **WHEN** the selected output format cannot encode the inspected table data
- **THEN** the CLI reports the encoding error
- **AND** the CLI exits with a non-zero status
