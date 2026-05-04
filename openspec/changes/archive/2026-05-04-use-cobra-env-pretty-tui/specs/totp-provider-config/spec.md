## MODIFIED Requirements

### Requirement: Provider file path resolution
The system SHALL accept a `--totp` CLI parameter that sets the TOTP provider file path. When `--totp` is absent and `TOTPCLI_TOTP_PATH` is set, the system SHALL use `TOTPCLI_TOTP_PATH` as the provider file path. When both `--totp` and `TOTPCLI_TOTP_PATH` are absent, the system SHALL use `~/totp.<format>`, where `<format>` is the effective explicit format value from `--format` or `TOTPCLI_FORMAT`, or `json` when no format is provided.

#### Scenario: Explicit provider path is used
- **WHEN** the user starts the application with `--totp /tmp/providers.yaml --format yaml`
- **THEN** the system loads providers from `/tmp/providers.yaml`

#### Scenario: Environment provider path is used
- **WHEN** `TOTPCLI_TOTP_PATH` is set to `/tmp/providers.toml` and the user starts the application without `--totp`
- **THEN** the system loads providers from `/tmp/providers.toml`

#### Scenario: Flag provider path overrides environment provider path
- **WHEN** `TOTPCLI_TOTP_PATH` is set to `/tmp/env.json` and the user starts the application with `--totp /tmp/flag.json`
- **THEN** the system loads providers from `/tmp/flag.json`

#### Scenario: Default JSON provider path is used
- **WHEN** the user starts the application without `--totp`, without `--format`, and without related environment variables
- **THEN** the system loads providers from `~/totp.json`

#### Scenario: Default path follows explicit format
- **WHEN** the user starts the application with `--format toml` and without `--totp`
- **THEN** the system loads providers from `~/totp.toml`

#### Scenario: Default path follows environment format
- **WHEN** `TOTPCLI_FORMAT` is set to `yaml` and the user starts the application without `--totp` and without `--format`
- **THEN** the system loads providers from `~/totp.yaml`

### Requirement: Provider file format resolution
The system SHALL accept `--format json|yaml|toml` and `TOTPCLI_FORMAT=json|yaml|toml`. When `--format` is present, it SHALL override `TOTPCLI_FORMAT`. When neither is present, the system SHALL detect the provider file format from a `.json`, `.yaml`, `.yml`, or `.toml` extension and SHALL fall back to JSON when no known extension is present.

#### Scenario: Explicit format overrides path extension
- **WHEN** the user starts the application with `--totp ~/totp.txt --format yaml`
- **THEN** the system parses the provider file as YAML

#### Scenario: Explicit format overrides environment format
- **WHEN** `TOTPCLI_FORMAT` is set to `json` and the user starts the application with `--totp ~/totp.txt --format yaml`
- **THEN** the system parses the provider file as YAML

#### Scenario: Environment format overrides path extension
- **WHEN** `TOTPCLI_FORMAT` is set to `yaml` and the user starts the application with `--totp ~/totp.txt` and without `--format`
- **THEN** the system parses the provider file as YAML

#### Scenario: Format is detected from extension
- **WHEN** the user starts the application with `--totp ~/totp.toml` and without `--format`
- **THEN** the system parses the provider file as TOML

#### Scenario: Unknown extension falls back to JSON
- **WHEN** the user starts the application with `--totp ~/totp.data` and without `--format`
- **THEN** the system parses the provider file as JSON
