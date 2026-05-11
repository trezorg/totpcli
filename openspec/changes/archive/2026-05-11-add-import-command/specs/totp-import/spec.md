## ADDED Requirements

### Requirement: Import command input and output
The CLI SHALL provide `totpcli import <qr-image>` to import providers from a Google Authenticator export QR image. The import command SHALL print imported provider JSON to stdout when `--totp` is omitted or set to `-`. The import command SHALL write to the output provider JSON path when `--totp <path>` is provided.

#### Scenario: Explicit output path is used
- **WHEN** the user runs `totpcli import /tmp/export.jpg --totp /tmp/providers.json`
- **THEN** the CLI imports providers from `/tmp/export.jpg` into `/tmp/providers.json`

#### Scenario: Omitted output path writes stdout
- **WHEN** the user runs `totpcli import /tmp/export.jpg` without `--totp`
- **THEN** the CLI prints imported provider JSON to stdout and does not write the default provider file

#### Scenario: Dash output path writes stdout
- **WHEN** the user runs `totpcli import /tmp/export.jpg --totp -`
- **THEN** the CLI prints imported provider JSON to stdout and does not write a provider file

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
- **THEN** the CLI reports an import error and leaves the output provider file unchanged

#### Scenario: Missing data parameter is rejected
- **WHEN** the QR image decodes to an `otpauth-migration` URI without a `data` query parameter
- **THEN** the CLI reports an import error and leaves the output provider file unchanged

### Requirement: Imported provider conversion
The import command SHALL convert Google Authenticator TOTP accounts into provider records compatible with the existing provider loader. The command SHALL base32-encode secret bytes into `totpSecret`, preserve `name` and `issuer`, map supported algorithms to `SHA1`, `SHA256`, or `SHA512`, map supported digit counts to `SIX` or `EIGHT`, and set `type` to `TOTP`.

#### Scenario: TOTP account is converted
- **WHEN** a migration payload account has TOTP type, SHA1 algorithm, six digits, secret bytes, issuer, and name
- **THEN** the imported provider contains `type: "TOTP"`, `algorithm: "SHA1"`, `digits: "SIX"`, the same issuer and name, and a base32 `totpSecret`

#### Scenario: Unsupported account type is rejected
- **WHEN** a migration payload account has HOTP type
- **THEN** the CLI reports an unsupported account error and leaves the output provider file unchanged

#### Scenario: Unsupported algorithm is rejected
- **WHEN** a migration payload account uses an unsupported algorithm such as MD5
- **THEN** the CLI reports an unsupported algorithm error and leaves the output provider file unchanged

#### Scenario: Unsupported digit count is rejected
- **WHEN** a migration payload account uses an unsupported digit count such as seven digits
- **THEN** the CLI reports an unsupported digit count error and leaves the output provider file unchanged

### Requirement: Merge import mode
The import command SHALL merge imported providers into the existing JSON provider file by default. Merge mode SHALL treat a missing output file as an empty provider list, preserve existing provider order, append new imported providers in QR payload order, and skip imported providers that duplicate an existing provider by trimmed issuer, trimmed name, and normalized `totpSecret`.

#### Scenario: Imported providers are appended
- **WHEN** the output JSON file contains one existing provider and the QR image contains one distinct imported provider
- **THEN** the output JSON file contains the existing provider followed by the imported provider

#### Scenario: Duplicate imported provider is skipped
- **WHEN** the output JSON file already contains a provider with the same issuer, name, and `totpSecret` as an imported provider
- **THEN** the output JSON file keeps the existing provider and does not add a duplicate

#### Scenario: Missing output file is created
- **WHEN** the output JSON file does not exist and the user runs `totpcli import /tmp/export.jpg`
- **THEN** the CLI creates the output JSON file with the imported providers

### Requirement: Override import mode
The import command SHALL provide `--override` to replace the output JSON file with only the imported providers. If `--merge` is exposed as an explicit flag, the CLI SHALL reject commands that provide both `--merge` and `--override`.

#### Scenario: Override replaces existing providers
- **WHEN** the output JSON file contains existing providers and the user runs `totpcli import /tmp/export.jpg --override`
- **THEN** the output JSON file contains only providers imported from `/tmp/export.jpg`

#### Scenario: Conflicting modes are rejected
- **WHEN** the user runs `totpcli import /tmp/export.jpg --merge --override`
- **THEN** the CLI reports a command usage error and leaves the output provider file unchanged

### Requirement: Import reporting and failure behavior
When writing to an explicit provider file, the import command SHALL print a success summary to stdout that includes the output path, imported count, added count, skipped duplicate count, and final provider count. When printing imported providers to stdout, the import command SHALL print provider JSON without a success summary. The command SHALL print errors to stderr once, exit non-zero on import failure, and leave the output provider file unchanged when decoding, conversion, parsing, or validation fails.

#### Scenario: Successful merge reports counts
- **WHEN** the import command adds two imported providers and skips one duplicate
- **THEN** stdout reports the output path, `imported=3`, `added=2`, `skipped=1`, and the final provider count

#### Scenario: Invalid existing JSON is not overwritten
- **WHEN** merge mode reads an existing output file that is not valid provider JSON
- **THEN** the CLI reports a parse error, exits non-zero, and leaves the existing file unchanged

#### Scenario: QR decode failure is reported
- **WHEN** the QR image cannot be decoded
- **THEN** the CLI reports a QR decode error, exits non-zero, and leaves the output provider file unchanged
