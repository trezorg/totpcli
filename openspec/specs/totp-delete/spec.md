# totp-delete Specification

## Purpose
TBD - created by archiving change add-delete-command. Update Purpose after archive.
## Requirements
### Requirement: Delete command provider selection
The CLI SHALL provide `totpcli delete` to remove one TOTP provider record from the resolved SQLite database. The command SHALL accept `--issuer`, `--name`, `--totp-secret`, and `--totp` flags. The command SHALL require at least one of `--issuer` or `--name`, SHALL trim selector values before matching, and SHALL match only providers whose supplied selector fields exactly match the stored normalized provider fields.

#### Scenario: Provider is deleted by issuer and name
- **WHEN** `/tmp/providers.sqlite` contains one provider with issuer `GitHub` and name `trezorg`
- **AND** the user runs `totpcli delete --issuer GitHub --name trezorg --totp /tmp/providers.sqlite`
- **THEN** the CLI deletes that provider record from `/tmp/providers.sqlite`
- **AND** no other provider records are deleted

#### Scenario: Provider is deleted by issuer only when unique
- **WHEN** `/tmp/providers.sqlite` contains one provider with issuer `AWS` and an empty name
- **AND** the user runs `totpcli delete --issuer AWS --totp /tmp/providers.sqlite`
- **THEN** the CLI deletes that provider record

#### Scenario: Provider is deleted by name only when unique
- **WHEN** `/tmp/providers.sqlite` contains one provider with name `trezorg` and an empty issuer
- **AND** the user runs `totpcli delete --name trezorg --totp /tmp/providers.sqlite`
- **THEN** the CLI deletes that provider record

#### Scenario: TOTP secret disambiguates matching provider
- **WHEN** `/tmp/providers.sqlite` contains two providers with issuer `GitHub` and different TOTP secrets
- **AND** the user runs `totpcli delete --issuer GitHub --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ --totp /tmp/providers.sqlite`
- **THEN** the CLI deletes only the provider whose normalized TOTP secret matches `GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ`

### Requirement: Delete command validation and safety
The `delete` command SHALL validate the selector before deleting. Missing identity, invalid `--totp-secret`, no matching providers, and ambiguous matching providers SHALL produce a non-zero exit status and SHALL leave the database unchanged. The command SHALL NOT support wildcard, fuzzy, partial-substring, or bulk deletion.

#### Scenario: Missing identity is rejected
- **WHEN** the user runs `totpcli delete --totp /tmp/providers.sqlite` without `--issuer` or `--name`
- **THEN** the CLI reports a usage error
- **AND** no provider record is deleted

#### Scenario: Invalid TOTP secret is rejected
- **WHEN** the user runs `totpcli delete --issuer GitHub --totp-secret not-base32 --totp /tmp/providers.sqlite`
- **THEN** the CLI reports a usage error
- **AND** no provider record is deleted

#### Scenario: Missing provider is reported without deletion
- **WHEN** `/tmp/providers.sqlite` does not contain a provider with issuer `GitHub`
- **AND** the user runs `totpcli delete --issuer GitHub --totp /tmp/providers.sqlite`
- **THEN** the CLI reports that no matching provider was found
- **AND** no provider record is deleted

#### Scenario: Ambiguous selector is rejected
- **WHEN** `/tmp/providers.sqlite` contains two providers with issuer `GitHub`
- **AND** the user runs `totpcli delete --issuer GitHub --totp /tmp/providers.sqlite`
- **THEN** the CLI reports an ambiguous selector error
- **AND** no provider record is deleted

#### Scenario: Partial selector does not match
- **WHEN** `/tmp/providers.sqlite` contains one provider with issuer `GitHub`
- **AND** the user runs `totpcli delete --issuer Git --totp /tmp/providers.sqlite`
- **THEN** the CLI reports that no matching provider was found
- **AND** no provider record is deleted

### Requirement: Delete command database path and reporting
The `delete` command SHALL resolve the target database path from `--totp`, `TOTPCLI_TOTP_PATH`, or the default `~/totp.sqlite`, using the same SQLite path normalization rules as other SQLite-backed commands. On success, the command SHALL print a concise summary to stdout that includes the resolved database path and the deleted provider label.

#### Scenario: Explicit database path is used
- **WHEN** the user runs `totpcli delete --issuer GitHub --totp /tmp/providers.sqlite`
- **THEN** the CLI deletes the matching provider from `/tmp/providers.sqlite`
- **AND** stdout reports `/tmp/providers.sqlite`

#### Scenario: Environment database path is used
- **WHEN** `TOTPCLI_TOTP_PATH` is set to `/tmp/env.sqlite` and the user runs `totpcli delete --issuer GitHub` without `--totp`
- **THEN** the CLI deletes the matching provider from `/tmp/env.sqlite`

#### Scenario: Default database path is used
- **WHEN** the user runs `totpcli delete --issuer GitHub` without `--totp` and without `TOTPCLI_TOTP_PATH`
- **THEN** the CLI deletes the matching provider from `~/totp.sqlite`

#### Scenario: Successful delete reports provider label
- **WHEN** the user runs `totpcli delete --issuer GitHub --name trezorg --totp /tmp/providers.sqlite`
- **THEN** stdout reports that `GitHub/trezorg` was deleted

