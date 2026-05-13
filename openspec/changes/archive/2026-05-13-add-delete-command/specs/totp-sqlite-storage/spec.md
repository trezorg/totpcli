## ADDED Requirements

### Requirement: Provider single delete
The repository SHALL support deleting one provider record selected by trimmed issuer, trimmed name, and optional normalized `totpSecret`. The repository SHALL delete the provider only when the selector matches exactly one row. No matching row, multiple matching rows, and invalid selector values SHALL return errors without deleting any provider row.

#### Scenario: Single matching provider is deleted
- **WHEN** the database contains one provider with issuer `GitHub`, name `trezorg`, and TOTP secret `JBSWY3DPEHPK3PXP`
- **AND** the repository receives a delete selector with issuer `GitHub` and name `trezorg`
- **THEN** the repository deletes that provider row
- **AND** no other provider row is deleted

#### Scenario: TOTP secret selector uses canonical matching
- **WHEN** the database contains one provider with issuer `GitHub` and canonical TOTP secret key for `JBSWY3DPEHPK3PXP`
- **AND** the repository receives a delete selector with issuer `GitHub` and equivalent normalized TOTP secret `jbswy3dpehpk3pxp`
- **THEN** the repository deletes that provider row

#### Scenario: No matching provider is not deleted
- **WHEN** the database does not contain a provider matching the delete selector
- **THEN** the repository returns a not-found error
- **AND** no provider row is deleted

#### Scenario: Ambiguous matching providers are not deleted
- **WHEN** the database contains two providers with issuer `GitHub`
- **AND** the repository receives a delete selector with only issuer `GitHub`
- **THEN** the repository returns an ambiguous selector error
- **AND** no provider row is deleted

#### Scenario: Invalid delete selector is rejected
- **WHEN** the repository receives a delete selector without issuer and without name
- **THEN** the repository returns a validation error
- **AND** no provider row is deleted

## MODIFIED Requirements

### Requirement: Provider repository boundary
Application code SHALL use a repository abstraction for SQLite communication. Command handlers and import services SHALL list providers, initialize storage, inspect tables, insert one manually added provider, delete one selected provider, and upsert imported providers through repository methods instead of issuing SQL statements directly.

#### Scenario: TOTP display reads through repository
- **WHEN** the TOTP workflow starts
- **THEN** it obtains providers by calling the provider repository for the resolved SQLite database

#### Scenario: Import writes through repository
- **WHEN** imported providers are persisted
- **THEN** the import workflow upserts providers by calling the provider repository

#### Scenario: Add writes through repository
- **WHEN** the add workflow persists a provider
- **THEN** the add workflow inserts the provider by calling the provider repository

#### Scenario: Delete writes through repository
- **WHEN** the delete workflow removes a provider
- **THEN** the delete workflow deletes the selected provider by calling the provider repository
