## Purpose

Define provider search behavior and the abstraction that allows future search engines to replace lexical search.

## Requirements

### Requirement: Search engine abstraction
The system SHALL filter providers through a pluggable search engine abstraction. The application SHALL provide digram and lexical search implementations and SHALL allow future search engines to be added without changing TUI selection behavior.

#### Scenario: TUI uses configured search engine
- **WHEN** the user changes the search input
- **THEN** the TUI requests filtered providers from the configured search engine

#### Scenario: Search engine can be replaced
- **WHEN** a different search engine implementation satisfies the search abstraction
- **THEN** the application can use that implementation without changing provider config loading

### Requirement: Digram prefix-tree provider filtering
The digram search engine SHALL match providers case-insensitively against issuer, name, and display label by querying an indexed digram prefix tree. The display label SHALL remain `issuer/name` when both values are present and SHALL be the single available value when only one is present.

#### Scenario: Digram search matches by issuer prefix
- **WHEN** the provider issuer is `GitHub` and the search query is `git`
- **THEN** digram search includes the provider in the results

#### Scenario: Digram search matches by name substring
- **WHEN** the provider name is `trezorg` and the search query is `zorg`
- **THEN** digram search includes the provider in the results

#### Scenario: Digram search matches by display label
- **WHEN** the provider display label is `GitHub/trezorg` and the search query is `hub/tre`
- **THEN** digram search includes the provider in the results

#### Scenario: Digram search excludes non-matches
- **WHEN** no searchable provider field contains the search query
- **THEN** digram search excludes the provider from the results

### Requirement: Digram empty search behavior
The digram search engine SHALL return all providers in their original order when the query is empty.

#### Scenario: Blank query returns all providers
- **WHEN** the digram search query is empty
- **THEN** digram search returns every provider in input order

### Requirement: Digram stable result order
The digram search engine SHALL preserve provider input order for matching records.

#### Scenario: Multiple providers match digram query
- **WHEN** several providers match the digram search query
- **THEN** digram search returns those providers in the same relative order as the provider file

### Requirement: Search engine selection
The system SHALL select a provider search engine by name. Supported engine names SHALL include `digram` and `lexical`. The default engine SHALL be `digram`.

#### Scenario: Default search engine is digram
- **WHEN** no search engine name is configured
- **THEN** the application uses the digram search engine

#### Scenario: Lexical search engine can be selected
- **WHEN** the configured search engine name is `lexical`
- **THEN** the application uses lexical provider filtering

#### Scenario: Unsupported search engine is rejected
- **WHEN** the configured search engine name is unsupported
- **THEN** the application reports a configuration error before launching the TUI

### Requirement: Lexical provider filtering
The lexical search engine SHALL match providers case-insensitively against issuer, name, and display label. The display label SHALL be `issuer/name` when both values are present and SHALL be the single available value when only one is present.

#### Scenario: Provider matches by issuer
- **WHEN** the provider issuer is `GitHub` and the search query is `git`
- **THEN** lexical search includes the provider in the results

#### Scenario: Provider matches by name
- **WHEN** the provider name is `trezorg` and the search query is `zorg`
- **THEN** lexical search includes the provider in the results

#### Scenario: Provider does not match
- **WHEN** no searchable provider field contains the search query
- **THEN** lexical search excludes the provider from the results

### Requirement: Empty search behavior
The lexical search engine SHALL return all providers in their original order when the query is empty.

#### Scenario: Blank query returns all providers
- **WHEN** the search query is empty
- **THEN** lexical search returns every provider in input order

### Requirement: Stable result order
The lexical search engine SHALL preserve provider input order for matching records.

#### Scenario: Multiple providers match
- **WHEN** several providers match the search query
- **THEN** lexical search returns those providers in the same relative order as the provider file
