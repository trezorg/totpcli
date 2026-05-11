## Purpose

Define importing TOTP providers from Google Authenticator export QR images and provider files into SQLite storage.

## Requirements

### Requirement: Import command input and output
The CLI SHALL provide `totpcli import <qr-image>` to import providers from a Google Authenticator export QR image into SQLite storage. The import command SHALL resolve the output database path from `--totp`, `TOTPCLI_TOTP_PATH`, or the QR image filename with a `.sqlite` extension. The import command SHALL NOT print imported provider JSON to stdout.

#### Scenario: Explicit output database path is used
- **WHEN** the user runs `totpcli import /tmp/export.jpg --totp /tmp/providers.sqlite`
- **THEN** the CLI imports providers from `/tmp/export.jpg` into `/tmp/providers.sqlite`

#### Scenario: Omitted output database path is derived from QR image
- **WHEN** the user runs `totpcli import /tmp/export.jpg` without `--totp`
- **THEN** the CLI imports providers from `/tmp/export.jpg` into `/tmp/export.sqlite`

#### Scenario: Missing image path is rejected
- **WHEN** the user runs `totpcli import` without a QR image path
- **THEN** the CLI reports a command usage error and exits with a non-zero usage status

### Requirement: Google Authenticator export decoding
The import command SHALL decode the QR image to an `otpauth-migration://offline?data=...` URI, URL-decode and base64-decode the `data` query parameter, and parse Google Authenticator migration payload versions 1 and 2.

#### Scenario: Migration URI is decoded
- **WHEN** the QR image contains an `otpauth-migration://offline?data=...` URI with a valid version 1 or 2 payload
- **THEN** the CLI extracts every account from that payload for provider conversion

#### Scenario: Non-migration QR is rejected
- **WHEN** the QR image decodes to a URI that is not an `otpauth-migration` URI
- **THEN** the CLI reports an import error and leaves the output database unchanged

#### Scenario: Missing data parameter is rejected
- **WHEN** the QR image decodes to an `otpauth-migration` URI without a `data` query parameter
- **THEN** the CLI reports an import error and leaves the output database unchanged

### Requirement: Imported provider conversion
The import command SHALL convert Google Authenticator TOTP accounts into provider records compatible with the existing provider loader. The command SHALL base32-encode secret bytes into `totpSecret`, preserve `name` and `issuer`, map supported algorithms to `SHA1`, `SHA256`, or `SHA512`, map supported digit counts to `SIX` or `EIGHT`, and set `type` to `TOTP`.

#### Scenario: TOTP account is converted
- **WHEN** a migration payload account has TOTP type, SHA1 algorithm, six digits, secret bytes, issuer, and name
- **THEN** the imported provider contains `type: "TOTP"`, `algorithm: "SHA1"`, `digits: "SIX"`, the same issuer and name, and a base32 `totpSecret`

#### Scenario: Unsupported account type is rejected
- **WHEN** a migration payload account has HOTP type
- **THEN** the CLI reports an unsupported account error and leaves the output database unchanged

#### Scenario: Unsupported algorithm is rejected
- **WHEN** a migration payload account uses an unsupported algorithm such as MD5
- **THEN** the CLI reports an unsupported algorithm error and leaves the output database unchanged

#### Scenario: Unsupported digit count is rejected
- **WHEN** a migration payload account uses an unsupported digit count such as seven digits
- **THEN** the CLI reports an unsupported digit count error and leaves the output database unchanged

### Requirement: Merge import mode
The import command SHALL upsert imported providers into the SQLite database. Import mode SHALL treat a missing output database as an empty provider store, preserve existing non-duplicate providers, insert new imported providers in import order, and override imported providers that duplicate an existing provider by trimmed issuer, trimmed name, and normalized `totpSecret`.

#### Scenario: Imported providers are inserted
- **WHEN** the output database contains one existing provider and the QR image contains one distinct imported provider
- **THEN** the output database contains both providers

#### Scenario: Duplicate imported provider is overridden
- **WHEN** the output database already contains a provider with the same issuer, name, and `totpSecret` as an imported provider
- **THEN** the output database keeps one provider for that key and updates it with the imported provider values

#### Scenario: Missing output database is created
- **WHEN** the output database does not exist and the user runs `totpcli import /tmp/export.jpg`
- **THEN** the CLI creates the output database with the imported providers

### Requirement: Override import mode
The import command SHALL NOT expose a whole-database `--override` mode. Duplicate provider records SHALL be overridden automatically by import upsert behavior.

#### Scenario: Override flag is rejected
- **WHEN** the user runs `totpcli import /tmp/export.jpg --override`
- **THEN** the CLI reports a command usage error and leaves the output database unchanged

### Requirement: Import reporting and failure behavior
When writing to SQLite storage, the import command SHALL print a success summary to stdout that includes the output database path, imported count, added count, overridden duplicate count, and final provider count. The command SHALL print errors to stderr once, exit non-zero on import failure, and leave the output database records unchanged when decoding, conversion, parsing, validation, or repository writes fail.

#### Scenario: Successful import reports counts
- **WHEN** the import command adds two imported providers and overrides one duplicate
- **THEN** stdout reports the output database path, `imported=3`, `added=2`, `overridden=1`, and the final provider count

#### Scenario: Repository write failure leaves records unchanged
- **WHEN** the repository cannot commit imported providers
- **THEN** the CLI reports a repository error, exits non-zero, and leaves existing provider records unchanged

#### Scenario: QR decode failure is reported
- **WHEN** the QR image cannot be decoded
- **THEN** the CLI reports a QR decode error, exits non-zero, and leaves the output database unchanged

### Requirement: Import-file command input and output
The CLI SHALL provide `totpcli import-file <provider-file>...` to import providers from one or more JSON/YAML/TOML provider files into SQLite storage. The command SHALL resolve the output database path from `--totp`, `TOTPCLI_TOTP_PATH`, or the first input provider filename with a `.sqlite` extension.

#### Scenario: Single provider file is imported
- **WHEN** the user runs `totpcli import-file /tmp/providers.json --totp /tmp/providers.sqlite`
- **THEN** the CLI imports providers from `/tmp/providers.json` into `/tmp/providers.sqlite`

#### Scenario: Multiple provider files are imported
- **WHEN** the user runs `totpcli import-file /tmp/providers.json /tmp/extra.yaml --totp /tmp/providers.sqlite`
- **THEN** the CLI imports providers from both files into `/tmp/providers.sqlite`

#### Scenario: Missing provider file argument is rejected
- **WHEN** the user runs `totpcli import-file` without file arguments
- **THEN** the CLI reports a command usage error and exits with a non-zero usage status

### Requirement: Import-file format handling
The import-file command SHALL accept `--format json|yaml|toml` to force all input files to that format. When `--format` is omitted, the command SHALL auto-discover each input file format from its extension and SHALL fall back to JSON for unknown extensions.

#### Scenario: Explicit import-file format is used
- **WHEN** the user runs `totpcli import-file /tmp/providers.txt --format yaml`
- **THEN** the CLI parses `/tmp/providers.txt` as YAML

#### Scenario: Import-file format is auto-discovered
- **WHEN** the user runs `totpcli import-file /tmp/providers.toml` without `--format`
- **THEN** the CLI parses `/tmp/providers.toml` as TOML

### Requirement: Import-file duplicate handling and reporting
The import-file command SHALL validate parsed providers, upsert them into SQLite storage, override duplicate records using the same duplicate key as QR import, and print a success summary with output database path, imported count, added count, overridden duplicate count, and final provider count.

#### Scenario: Duplicate file provider is overridden
- **WHEN** an imported file provider duplicates an existing database provider
- **THEN** the command updates the existing database provider with the imported file provider values

#### Scenario: Successful file import reports counts
- **WHEN** the import-file command imports three providers, adds two, and overrides one duplicate
- **THEN** stdout reports the output database path, `imported=3`, `added=2`, `overridden=1`, and the final provider count
