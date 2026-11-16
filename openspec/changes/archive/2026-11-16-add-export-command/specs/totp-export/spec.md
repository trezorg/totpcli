## ADDED Requirements

### Requirement: Export format selection
The `export` command SHALL accept `--format json|yaml|toml`. When `--format` is present, it SHALL override extension-based detection. When `--format` is absent, the command SHALL detect the format from the `--output` file extension and SHALL fall back to JSON when no known extension is present. Unsupported format values SHALL fail with a usage error before the database is opened.

#### Scenario: Explicit export format overrides output extension
- **WHEN** the user runs `export --output ~/backup.json --format yaml`
- **THEN** the command writes a YAML provider file

#### Scenario: Format is detected from output extension
- **WHEN** the user runs `export --output ~/backup.toml` without `--format`
- **THEN** the command writes a TOML provider file

#### Scenario: Unknown output extension falls back to JSON
- **WHEN** the user runs `export --output ~/backup.data` without `--format`
- **THEN** the command writes a JSON provider file

#### Scenario: Unsupported export format fails before opening the database
- **WHEN** the user runs `export --format xml`
- **THEN** the command reports a usage error and exits with a usage error status without opening the database

### Requirement: Export output path resolution
The `export` command SHALL accept `--output` with a `-o` shorthand. When `--output` is absent, the command SHALL write to `~/totp.<format>`. The output path SHALL be used verbatim except for `~` expansion and SHALL NOT undergo SQLite path normalization.

#### Scenario: Default output path follows the format
- **WHEN** the user runs `export` with neither `--output` nor `--format`
- **THEN** the command writes to `~/totp.json`

#### Scenario: Explicit output path is preserved verbatim
- **WHEN** the user runs `export --output ~/backups/providers.bak`
- **THEN** the command writes exactly to `~/backups/providers.bak`

### Requirement: Export output file safety
The `export` command SHALL write output files with `0600` permissions. When the output file already exists and `--force` is absent, the command SHALL fail with an operational error that mentions `--force` and SHALL leave the existing file unchanged. When `--force` is present, the command SHALL overwrite the file and SHALL tighten its permissions to `0600`.

#### Scenario: Output matching the source database is rejected
- **WHEN** the output path identifies the source provider database, including through symlinks or hard links, even with `--force`
- **THEN** the command reports an error before writing and the source database is unchanged

#### Scenario: Existing output file is refused without force
- **WHEN** the output file exists and the user runs `export --output <path>` without `--force`
- **THEN** the command exits with an operational error status, reports the `--force` hint, and the existing file content is unchanged

#### Scenario: Forced overwrite tightens permissions
- **WHEN** the output file exists with permissions broader than `0600` and the user runs `export --output <path> --force`
- **THEN** the command overwrites the file and the resulting permissions are `0600`

#### Scenario: Export file is created owner-only
- **WHEN** the user runs `export --output <path>` and the file does not exist
- **THEN** the created file has `0600` permissions

### Requirement: Export content and summary
The `export` command SHALL write every stored provider as a provider file document parseable by `import-file`, mapping six digits to `SIX` and eight digits to `EIGHT`. An empty database SHALL produce a valid empty document. On success the command SHALL print a summary naming the provider count, output path, and format.

#### Scenario: Providers are exported in the provider file format
- **WHEN** the database contains providers and the user runs `export --output ~/backup.json`
- **THEN** the command writes a JSON array of provider records restorable by `import-file`

#### Scenario: Empty database exports an empty document
- **WHEN** the database contains no providers and the user runs `export`
- **THEN** the command writes a parseable empty document and exits successfully

### Requirement: Export source database resolution
The `export` command SHALL resolve its source database from `--totp`, then `TOTPCLI_TOTP_PATH`, then `~/totp.sqlite`, matching the other SQLite-backed commands.

#### Scenario: Export uses the resolved provider database
- **WHEN** the user runs `export --totp /tmp/providers.sqlite --output /tmp/backup.json`
- **THEN** the command reads providers from `/tmp/providers.sqlite`
