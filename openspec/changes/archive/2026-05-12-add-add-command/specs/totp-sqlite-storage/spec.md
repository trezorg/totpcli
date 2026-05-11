## ADDED Requirements

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

## MODIFIED Requirements

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
