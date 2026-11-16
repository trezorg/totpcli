## ADDED Requirements

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
