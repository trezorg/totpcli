## ADDED Requirements

### Requirement: Search engine abstraction
The system SHALL filter providers through a pluggable search engine abstraction. The initial application SHALL provide a lexical search implementation and SHALL allow future search engines to be added without changing TUI selection behavior.

#### Scenario: TUI uses configured search engine
- **WHEN** the user changes the search input
- **THEN** the TUI requests filtered providers from the configured search engine

#### Scenario: Search engine can be replaced
- **WHEN** a different search engine implementation satisfies the search abstraction
- **THEN** the application can use that implementation without changing provider config loading

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
