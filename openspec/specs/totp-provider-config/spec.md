## Purpose

Define how TOTP provider storage is located and how imported provider files are parsed, validated, and normalized.
## Requirements
### Requirement: Provider file path resolution
The system SHALL accept a `--totp` CLI parameter that sets the TOTP provider database path input. When `--totp` is absent and `TOTPCLI_TOTP_PATH` is set, the system SHALL use `TOTPCLI_TOTP_PATH` as the provider database path input. When both `--totp` and `TOTPCLI_TOTP_PATH` are absent for TOTP display behavior, the system SHALL use `~/totp.sqlite`.

#### Scenario: Explicit provider database path is used
- **WHEN** the user starts the application with `--totp /tmp/providers.sqlite`
- **THEN** the system loads providers from `/tmp/providers.sqlite`

#### Scenario: Environment provider database path is used
- **WHEN** `TOTPCLI_TOTP_PATH` is set to `/tmp/providers.sqlite` and the user starts the application without `--totp`
- **THEN** the system loads providers from `/tmp/providers.sqlite`

#### Scenario: Flag provider database path overrides environment provider database path
- **WHEN** `TOTPCLI_TOTP_PATH` is set to `/tmp/env.sqlite` and the user starts the application with `--totp /tmp/flag.sqlite`
- **THEN** the system loads providers from `/tmp/flag.sqlite`

#### Scenario: Default SQLite provider path is used
- **WHEN** the user starts the application without `--totp` and without related environment variables
- **THEN** the system loads providers from `~/totp.sqlite`

### Requirement: Provider file format resolution
The system SHALL accept `--format json|yaml|toml` on the `import-file` command. When `--format` is present, it SHALL apply to every input file and override extension-based detection. When `--format` is absent, the system SHALL detect each provider file format from a `.json`, `.yaml`, `.yml`, or `.toml` extension and SHALL fall back to JSON when no known extension is present.

#### Scenario: Explicit import-file format overrides path extension
- **WHEN** the user starts the application with `import-file ~/totp.txt --format yaml`
- **THEN** the system parses `~/totp.txt` as YAML before importing providers into SQLite

#### Scenario: Format is detected from extension
- **WHEN** the user starts the application with `import-file ~/totp.toml` and without `--format`
- **THEN** the system parses `~/totp.toml` as TOML before importing providers into SQLite

#### Scenario: Unknown extension falls back to JSON
- **WHEN** the user starts the application with `import-file ~/totp.data` and without `--format`
- **THEN** the system parses `~/totp.data` as JSON before importing providers into SQLite

### Requirement: Provider data parsing
The system SHALL parse imported provider files containing an array of provider records. Each provider record SHALL support `secret`, `name`, `issuer`, `algorithm`, `digits`, `type`, and `totpSecret` fields.

#### Scenario: JSON provider file is parsed
- **WHEN** an imported provider file contains a JSON array with provider records
- **THEN** the system returns the parsed provider records

#### Scenario: YAML provider file is parsed
- **WHEN** an imported provider file contains a YAML sequence with provider records
- **THEN** the system returns the parsed provider records

#### Scenario: TOML provider file is parsed
- **WHEN** an imported provider file contains TOML array-of-table provider records
- **THEN** the system returns the parsed provider records

### Requirement: Provider validation and normalization
The system SHALL validate provider records before storing them in SQLite or generating TOTP codes. The system SHALL require `totpSecret`, SHALL require `type` to be absent or `TOTP`, SHALL require at least one of `issuer` or `name`, SHALL normalize supported algorithms, and SHALL normalize `digits` values including `SIX`, `EIGHT`, `6`, and `8`.

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

### Requirement: Provider file serialization
The system SHALL encode provider records into JSON, YAML, and TOML provider file documents symmetric with provider file parsing, including TOML array-of-table `providers` documents. Encoding zero providers SHALL produce a parseable empty document. The system SHALL map stored providers to provider records with six digits encoded as `SIX` and eight digits encoded as `EIGHT`.

#### Scenario: Encoded documents round-trip through parsing
- **WHEN** provider records are encoded as JSON, YAML, and TOML and parsed back
- **THEN** the parsed provider records equal the encoded records

#### Scenario: Empty provider list encodes a parseable document
- **WHEN** zero provider records are encoded in any supported format
- **THEN** the document parses to zero provider records

### Requirement: Provider file writing safety
The system SHALL write provider files with `0600` permissions because they contain cleartext TOTP secrets. When the target file already exists and overwrite is not requested, the system SHALL fail without modifying the file. When overwrite is requested, the system SHALL replace the file content and SHALL set its permissions to `0600`.

#### Scenario: Existing file is refused without overwrite
- **WHEN** the target provider file exists and overwrite is not requested
- **THEN** the system reports an output-file-exists error and leaves the file unchanged

#### Scenario: Created and overwritten files are owner-only
- **WHEN** a provider file is written to a new or existing path
- **THEN** the resulting file permissions are `0600`

