## ADDED Requirements

### Requirement: Provider file path resolution
The system SHALL accept a `--totp` CLI parameter that sets the TOTP provider file path. When `--totp` is absent, the system SHALL use `~/totp.<format>`, where `<format>` is the explicit `--format` value or `json` when no format is provided.

#### Scenario: Explicit provider path is used
- **WHEN** the user starts the application with `--totp /tmp/providers.yaml --format yaml`
- **THEN** the system loads providers from `/tmp/providers.yaml`

#### Scenario: Default JSON provider path is used
- **WHEN** the user starts the application without `--totp` and without `--format`
- **THEN** the system loads providers from `~/totp.json`

#### Scenario: Default path follows explicit format
- **WHEN** the user starts the application with `--format toml` and without `--totp`
- **THEN** the system loads providers from `~/totp.toml`

### Requirement: Provider file format resolution
The system SHALL accept `--format json|yaml|toml`. When `--format` is absent, the system SHALL detect the provider file format from a `.json`, `.yaml`, `.yml`, or `.toml` extension and SHALL fall back to JSON when no known extension is present.

#### Scenario: Explicit format overrides path extension
- **WHEN** the user starts the application with `--totp ~/totp.txt --format yaml`
- **THEN** the system parses the provider file as YAML

#### Scenario: Format is detected from extension
- **WHEN** the user starts the application with `--totp ~/totp.toml` and without `--format`
- **THEN** the system parses the provider file as TOML

#### Scenario: Unknown extension falls back to JSON
- **WHEN** the user starts the application with `--totp ~/totp.data` and without `--format`
- **THEN** the system parses the provider file as JSON

### Requirement: Provider data parsing
The system SHALL parse provider files containing an array of provider records. Each provider record SHALL support `secret`, `name`, `issuer`, `algorithm`, `digits`, `type`, and `totpSecret` fields.

#### Scenario: JSON provider file is parsed
- **WHEN** the provider file contains a JSON array with provider records
- **THEN** the system returns the parsed provider records

#### Scenario: YAML provider file is parsed
- **WHEN** the provider file contains a YAML sequence with provider records
- **THEN** the system returns the parsed provider records

#### Scenario: TOML provider file is parsed
- **WHEN** the provider file contains TOML array-of-table provider records
- **THEN** the system returns the parsed provider records

### Requirement: Provider validation and normalization
The system SHALL validate provider records before starting the TUI. The system SHALL require `totpSecret`, SHALL require `type` to be absent or `TOTP`, SHALL require at least one of `issuer` or `name`, SHALL normalize supported algorithms, and SHALL normalize `digits` values including `SIX`, `EIGHT`, `6`, and `8`.

#### Scenario: Valid provider is normalized
- **WHEN** a provider has `type` set to `TOTP`, `algorithm` set to `SHA1`, `digits` set to `SIX`, and a `totpSecret`
- **THEN** the system normalizes the provider to SHA1 with 6 digits

#### Scenario: Missing TOTP secret is rejected
- **WHEN** a provider record does not contain `totpSecret`
- **THEN** the system returns a validation error for that provider

#### Scenario: Unsupported provider type is rejected
- **WHEN** a provider record has `type` set to `HOTP`
- **THEN** the system returns a validation error for that provider

### Requirement: TOTP generation options
The system SHALL generate TOTP codes from the normalized provider using an in-process Go implementation. The system SHALL map normalized algorithm and digit values to the generation options and SHALL use a 30-second period.

#### Scenario: Six digit SHA1 code is generated
- **WHEN** a selected provider has a base32 `totpSecret`, algorithm SHA1, and 6 digits
- **THEN** the system generates a six digit TOTP code for the current 30-second period

#### Scenario: Invalid base32 secret is reported
- **WHEN** a selected provider has an invalid base32 `totpSecret`
- **THEN** the system returns a generation error for that provider
