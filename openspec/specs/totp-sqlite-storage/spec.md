## Purpose

Define SQLite-backed TOTP provider storage, database path resolution, repository boundaries, schema initialization, and import upsert behavior.

## Requirements

### Requirement: SQLite database path resolution
The system SHALL resolve the provider storage path to a SQLite database file. When a command receives an explicit provider storage path through `--totp` or `TOTPCLI_TOTP_PATH`, the system SHALL expand `~` and SHALL use the path unchanged when it ends in `.sqlite`; otherwise it SHALL replace the final extension with `.sqlite`. When no explicit storage path is provided, the system SHALL derive a `.sqlite` database path from the command's default or input filename.

#### Scenario: Explicit SQLite path is used unchanged
- **WHEN** the user runs `totpcli totp --totp /tmp/providers.sqlite`
- **THEN** the system uses `/tmp/providers.sqlite` as the provider database path

#### Scenario: Explicit non-SQLite path is converted
- **WHEN** the user runs `totpcli totp --totp /tmp/providers.json`
- **THEN** the system uses `/tmp/providers.sqlite` as the provider database path

#### Scenario: Default TOTP database path is used
- **WHEN** the user runs `totpcli totp` without `--totp` and without `TOTPCLI_TOTP_PATH`
- **THEN** the system uses `~/totp.sqlite` as the provider database path

#### Scenario: Import database path is derived from QR image
- **WHEN** the user runs `totpcli import /tmp/export.jpg` without `--totp` and without `TOTPCLI_TOTP_PATH`
- **THEN** the system uses `/tmp/export.sqlite` as the provider database path

#### Scenario: Import-file database path is derived from first input file
- **WHEN** the user runs `totpcli import-file /tmp/providers.yaml /tmp/extra.json` without `--totp` and without `TOTPCLI_TOTP_PATH`
- **THEN** the system uses `/tmp/providers.sqlite` as the provider database path

### Requirement: SQLite schema initialization
The system SHALL open the resolved SQLite database at command startup and SHALL create the provider table schema when the database file or table does not exist. The provider table schema SHALL represent the provider fields `secret`, `name`, `issuer`, `algorithm`, `digits`, `type`, and `totpSecret` and SHALL support duplicate detection for imports.

#### Scenario: Missing database is created
- **WHEN** the resolved SQLite database file does not exist
- **THEN** the system creates the database file and provider table before reading or writing providers

#### Scenario: Existing database is reused
- **WHEN** the resolved SQLite database file already contains the provider table
- **THEN** the system uses the existing table without dropping provider records

#### Scenario: Schema initialization failure is reported
- **WHEN** the provider table cannot be created because the database path is not writable
- **THEN** the command reports an error and exits non-zero

### Requirement: Provider repository boundary
Application code SHALL use a repository abstraction for SQLite communication. Command handlers and import services SHALL list providers, initialize storage, inspect tables, insert one manually added provider, and upsert imported providers through repository methods instead of issuing SQL statements directly.

#### Scenario: TOTP display reads through repository
- **WHEN** the TOTP workflow starts
- **THEN** it obtains providers by calling the provider repository for the resolved SQLite database

#### Scenario: Import writes through repository
- **WHEN** imported providers are persisted
- **THEN** the import workflow upserts providers by calling the provider repository

#### Scenario: Add writes through repository
- **WHEN** the add workflow persists a provider
- **THEN** the add workflow inserts the provider by calling the provider repository

### Requirement: Provider single insert
The repository SHALL support saving one provider record for direct manual creation. The repository SHALL normalize the provider with the same validation rules used by imports, SHALL canonicalize `totpSecret` for duplicate detection, SHALL insert non-duplicate providers, and SHALL update duplicate providers without creating an additional record.

#### Scenario: Manual provider is inserted
- **WHEN** the repository receives a provider with issuer `GitHub`, name `trezorg`, algorithm `SHA1`, digits `SIX`, type `TOTP`, and TOTP secret `JBSWY3DPEHPK3PXP`
- **THEN** the repository inserts one provider row
- **AND** the inserted row includes the normalized provider values
- **AND** the inserted row includes the canonical TOTP secret duplicate key

#### Scenario: Manual duplicate provider is updated
- **WHEN** the database already contains a provider with the same trimmed issuer, trimmed name, and normalized `totpSecret`
- **AND** the repository receives that provider for single insert
- **THEN** the repository updates the existing provider row with the new provider values
- **AND** no additional provider row is inserted

#### Scenario: Invalid manual provider is rejected
- **WHEN** the repository receives a provider without `totpSecret`
- **THEN** the repository returns a validation error
- **AND** no provider row is inserted

### Requirement: Provider import upsert
The repository SHALL detect duplicate providers by trimmed issuer, trimmed name, and normalized `totpSecret`. Import operations SHALL override duplicate records with the imported provider values and SHALL insert non-duplicate records.

#### Scenario: Duplicate provider is overridden
- **WHEN** the database contains a provider with the same trimmed issuer, trimmed name, and normalized `totpSecret` as an imported provider
- **THEN** the repository updates the existing provider record with the imported provider values

#### Scenario: Distinct provider is inserted
- **WHEN** the database does not contain a provider with the imported provider duplicate key
- **THEN** the repository inserts a new provider record

#### Scenario: Import reports overridden records
- **WHEN** an import overrides one existing provider and inserts two new providers
- **THEN** the import result reports one overridden provider, two added providers, and the final provider count
