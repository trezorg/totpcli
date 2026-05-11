## ADDED Requirements

### Requirement: Add command provider creation
The CLI SHALL provide `totpcli add` to create one TOTP provider record in the resolved SQLite database. The command SHALL accept `--issuer`, `--name`, `--totp-secret`, `--secret`, `--algorithm`, `--digits`, and `--totp` flags. The command SHALL create stored provider records with `type` set to `TOTP`.

#### Scenario: Provider is added with required identity and secret
- **WHEN** the user runs `totpcli add --issuer GitHub --name trezorg --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ --totp /tmp/providers.sqlite`
- **THEN** the CLI creates a provider record in `/tmp/providers.sqlite`
- **AND** the stored provider has issuer `GitHub`, name `trezorg`, and TOTP secret `GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ`
- **AND** the stored provider type is `TOTP`

#### Scenario: Provider may use only issuer
- **WHEN** the user runs `totpcli add --issuer AWS --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ --totp /tmp/providers.sqlite`
- **THEN** the CLI creates a provider record with issuer `AWS` and an empty name

#### Scenario: Provider may use only name
- **WHEN** the user runs `totpcli add --name trezorg --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ --totp /tmp/providers.sqlite`
- **THEN** the CLI creates a provider record with name `trezorg` and an empty issuer

### Requirement: Add command derived and optional fields
The `add` command SHALL derive omitted provider values consistently with provider normalization. When `--algorithm` is omitted, the command SHALL store `SHA1`. When `--digits` is omitted, the command SHALL store six digits. When `--secret` is omitted, the command SHALL store an empty provider `secret`. The command SHALL NOT expose a `--type` flag.

#### Scenario: Omitted optional values use defaults
- **WHEN** the user runs `totpcli add --issuer GitHub --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ --totp /tmp/providers.sqlite` without `--algorithm`, `--digits`, or `--secret`
- **THEN** the stored provider has algorithm `SHA1`
- **AND** the stored provider has six digits
- **AND** the stored provider has an empty `secret`
- **AND** the stored provider type is `TOTP`

#### Scenario: Explicit supported algorithm and digits are stored
- **WHEN** the user runs `totpcli add --issuer GitHub --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZA --algorithm SHA256 --digits EIGHT --totp /tmp/providers.sqlite`
- **THEN** the stored provider has algorithm `SHA256`
- **AND** the stored provider has eight digits

#### Scenario: Type flag is rejected
- **WHEN** the user runs `totpcli add --issuer GitHub --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ --type TOTP`
- **THEN** the CLI reports a command usage error
- **AND** the CLI exits with a non-zero usage status

### Requirement: Add command validation
The `add` command SHALL validate provider input before inserting or updating a record. The command SHALL require `--totp-secret`, SHALL require at least one of `--issuer` or `--name`, and SHALL validate that `--totp-secret` is valid base32, decodes to at least 20 bytes for `SHA1`, at least 32 bytes for `SHA256`, and at least 64 bytes for `SHA512`, and can generate a TOTP code with the resolved algorithm and digits. Unsupported algorithms, unsupported digit values, and invalid TOTP secrets SHALL produce a usage error and SHALL leave the database unchanged.

#### Scenario: Missing TOTP secret is rejected
- **WHEN** the user runs `totpcli add --issuer GitHub --totp /tmp/providers.sqlite` without `--totp-secret`
- **THEN** the CLI reports a usage error
- **AND** no provider record is created

#### Scenario: Missing identity is rejected
- **WHEN** the user runs `totpcli add --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ --totp /tmp/providers.sqlite` without `--issuer` or `--name`
- **THEN** the CLI reports a usage error
- **AND** no provider record is created

#### Scenario: Invalid TOTP secret is rejected
- **WHEN** the user runs `totpcli add --issuer GitHub --totp-secret not-base32 --totp /tmp/providers.sqlite`
- **THEN** the CLI reports a usage error
- **AND** no provider record is created

#### Scenario: Base32 encoded short TOTP key is rejected
- **WHEN** the user runs `totpcli add --issuer GitHub --totp-secret NBUQ --totp /tmp/providers.sqlite`
- **THEN** the CLI reports a usage error
- **AND** no provider record is created

#### Scenario: SHA1 TOTP key shorter than 20 bytes is rejected
- **WHEN** the user runs `totpcli add --issuer GitHub --totp-secret JBSWY3DPEHPK3PXP --totp /tmp/providers.sqlite`
- **THEN** the CLI reports a usage error
- **AND** no provider record is created

#### Scenario: SHA256 TOTP key shorter than 32 bytes is rejected
- **WHEN** the user runs `totpcli add --issuer GitHub --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ --algorithm SHA256 --totp /tmp/providers.sqlite`
- **THEN** the CLI reports a usage error
- **AND** no provider record is created

#### Scenario: SHA512 TOTP key shorter than 64 bytes is rejected
- **WHEN** the user runs `totpcli add --issuer GitHub --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZA --algorithm SHA512 --totp /tmp/providers.sqlite`
- **THEN** the CLI reports a usage error
- **AND** no provider record is created

#### Scenario: Unsupported algorithm is rejected
- **WHEN** the user runs `totpcli add --issuer GitHub --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ --algorithm MD5 --totp /tmp/providers.sqlite`
- **THEN** the CLI reports a usage error
- **AND** no provider record is created

#### Scenario: Unsupported digits are rejected
- **WHEN** the user runs `totpcli add --issuer GitHub --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ --digits 7 --totp /tmp/providers.sqlite`
- **THEN** the CLI reports a usage error
- **AND** no provider record is created

### Requirement: Add command database path and reporting
The `add` command SHALL resolve the target database path from `--totp`, `TOTPCLI_TOTP_PATH`, or the default `~/totp.sqlite`, using the same SQLite path normalization rules as other SQLite-backed commands. On success, the command SHALL print a concise summary to stdout that includes the output database path and the added provider label.

#### Scenario: Explicit database path is used
- **WHEN** the user runs `totpcli add --issuer GitHub --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ --totp /tmp/providers.sqlite`
- **THEN** the CLI writes the provider record to `/tmp/providers.sqlite`
- **AND** stdout reports `/tmp/providers.sqlite`

#### Scenario: Environment database path is used
- **WHEN** `TOTPCLI_TOTP_PATH` is set to `/tmp/env.sqlite` and the user runs `totpcli add --issuer GitHub --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ` without `--totp`
- **THEN** the CLI writes the provider record to `/tmp/env.sqlite`

#### Scenario: Default database path is used
- **WHEN** the user runs `totpcli add --issuer GitHub --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ` without `--totp` and without `TOTPCLI_TOTP_PATH`
- **THEN** the CLI writes the provider record to `~/totp.sqlite`

#### Scenario: Successful add reports provider label
- **WHEN** the user runs `totpcli add --issuer GitHub --name trezorg --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ --totp /tmp/providers.sqlite`
- **THEN** stdout reports that `GitHub/trezorg` was added

### Requirement: Add command duplicate handling
The `add` command SHALL update a provider that duplicates an existing record by trimmed issuer, trimmed name, and normalized `totpSecret`. Duplicate updates SHALL preserve a single provider record for the duplicate key.

#### Scenario: Duplicate provider is updated
- **WHEN** `/tmp/providers.sqlite` already contains a provider with issuer `GitHub`, name `trezorg`, and TOTP secret `GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZA`
- **AND** the user runs `totpcli add --issuer GitHub --name trezorg --totp-secret GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZA --algorithm SHA256 --digits EIGHT --secret metadata --totp /tmp/providers.sqlite`
- **THEN** the CLI updates the existing provider record with algorithm `SHA256`, eight digits, and secret metadata `metadata`
- **AND** no additional provider record is created
