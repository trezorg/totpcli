## 1. Command Structure

- [x] 1.1 Refactor current root TOTP execution into a reusable command action.
- [x] 1.2 Add a `totp` subcommand that uses the same flags, environment fallbacks, and behavior as root invocation.
- [x] 1.3 Add an `import <qr-image>` subcommand with stdout default output, `--totp`, default merge mode, and `--override`.
- [x] 1.4 Update Cobra help text to describe `totp`, `import`, related flags, and environment variables.

## 2. Import Pipeline

- [x] 2.1 Add an importer package with interfaces for QR image decoding, migration payload decoding, provider conversion, and provider file writing.
- [x] 2.2 Implement QR image decoding from a local image path to a decoded URI string.
- [x] 2.3 Implement `otpauth-migration` URI parsing, `data` extraction, URL decoding, and base64 decoding.
- [x] 2.4 Implement Google Authenticator migration protobuf wire decoding for version 1 and 2 payloads without copying external GPL proto files.
- [x] 2.5 Convert supported TOTP accounts into existing provider records with base32 `totpSecret`, issuer, name, algorithm, digits, and type.
- [x] 2.6 Reject unsupported account types, algorithms, digit counts, missing secrets, malformed migration URIs, and QR decode failures with actionable errors.

## 3. Provider File Writing

- [x] 3.1 Implement JSON provider loading for import merge mode, treating a missing output file as an empty provider list.
- [x] 3.2 Implement deduplication by trimmed issuer, trimmed name, and normalized `totpSecret`.
- [x] 3.3 Implement default merge mode that preserves existing provider order and appends non-duplicate imports in QR payload order.
- [x] 3.4 Implement override mode that replaces the output JSON with only imported providers.
- [x] 3.5 Write provider JSON atomically so parse, validation, conversion, or write failures leave the existing file unchanged.
- [x] 3.6 Print import success counts to stdout and import errors to stderr with non-zero exit codes.

## 4. Tests

- [x] 4.1 Add CLI tests for root compatibility, `totp` subcommand behavior, import routing, help output, and usage errors.
- [x] 4.2 Add importer tests for migration URI parsing, protobuf decoding, account conversion, unsupported values, and malformed payloads.
- [x] 4.3 Add provider writer tests for merge, override, deduplication, missing output file creation, invalid existing JSON, and unchanged file behavior on failure.
- [x] 4.4 Add an end-to-end import command test using a QR decoder test double and the existing provider loader.

## 5. Verification

- [x] 5.1 Run `make test`.
- [x] 5.2 Run `make lint`.
- [x] 5.3 Run `make all`.
