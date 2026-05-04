## Purpose

Define the terminal UI behavior for browsing TOTP providers, filtering them, selecting a provider, and displaying generated codes.

## Requirements

### Requirement: Provider list display
The TUI SHALL display loaded TOTP providers as a selectable list. Each row SHALL display `issuer/name` when both fields are present, `issuer` when `name` is absent, and `name` when `issuer` is absent.

#### Scenario: Provider with issuer and name is displayed
- **WHEN** a provider has issuer `GitHub` and name `trezorg`
- **THEN** the TUI displays the provider as `GitHub/trezorg`

#### Scenario: Provider without name is displayed
- **WHEN** a provider has issuer `GitHub` and no name
- **THEN** the TUI displays the provider as `GitHub`

#### Scenario: Provider without issuer is displayed
- **WHEN** a provider has name `trezorg` and no issuer
- **THEN** the TUI displays the provider as `trezorg`

### Requirement: Search input
The TUI SHALL provide a text input for entering search text. As the user types, the TUI SHALL update the provider list using the configured search engine.

#### Scenario: Typing filters providers
- **WHEN** the user types letters into the search input
- **THEN** the visible provider list updates to the matching providers

#### Scenario: Clearing input restores providers
- **WHEN** the user clears the search input
- **THEN** the visible provider list returns to all providers

### Requirement: Initial search parameter
The CLI SHALL accept a `--search` parameter. When present, the TUI SHALL initialize the search input from that value and SHALL display the filtered provider list on startup.

#### Scenario: Startup search filters list
- **WHEN** the user starts the application with `--search git`
- **THEN** the TUI starts with `git` in the search input and shows only matching providers

### Requirement: Provider selection
The TUI SHALL allow the user to move through the filtered provider list and select the current provider with Enter.

#### Scenario: Enter selects provider
- **WHEN** a provider row is highlighted and the user presses Enter
- **THEN** the TUI selects that provider for code generation

#### Scenario: Enter with no result does not generate code
- **WHEN** the filtered provider list is empty and the user presses Enter
- **THEN** the TUI does not attempt to generate a TOTP code

### Requirement: TOTP code display
After provider selection, the TUI SHALL display the generated TOTP code for the selected provider and SHALL show the remaining seconds in the current 30-second period.

#### Scenario: Selected provider shows code
- **WHEN** the user selects a provider with a valid TOTP secret
- **THEN** the TUI displays the generated TOTP code

#### Scenario: Remaining time is displayed
- **WHEN** a generated TOTP code is shown
- **THEN** the TUI displays the number of seconds remaining before the code changes

#### Scenario: Code refreshes after period changes
- **WHEN** the current 30-second period expires while the code view is open
- **THEN** the TUI refreshes the displayed TOTP code and remaining time

### Requirement: TOTP generation error display
The TUI SHALL display an error state when code generation fails for the selected provider.

#### Scenario: Invalid selected provider shows error
- **WHEN** the user selects a provider whose TOTP code cannot be generated
- **THEN** the TUI displays an error instead of a code

### Requirement: Structured colorized TUI rendering
The TUI SHALL render search input, provider rows, selected provider state, generated codes, remaining time, empty results, help text, and errors through a named style schema. On color-capable terminals, the style schema SHALL use distinct colors or emphasis for selection, successful code output, remaining time, muted helper text, and errors.

#### Scenario: Selected provider row is visually distinct
- **WHEN** the TUI displays a provider list with a highlighted row
- **THEN** the highlighted row is visually distinct from non-highlighted rows

#### Scenario: Generated code is emphasized
- **WHEN** a selected provider has a generated TOTP code
- **THEN** the TUI displays the code with stronger emphasis than surrounding labels

#### Scenario: Error state is styled distinctly
- **WHEN** code generation fails for the selected provider
- **THEN** the TUI displays the error with error-specific styling

#### Scenario: Empty result state is styled distinctly
- **WHEN** the filtered provider list is empty
- **THEN** the TUI displays the empty result message with muted or informational styling

### Requirement: TUI visible content compatibility
The TUI SHALL preserve the visible provider labels, generated code text, remaining time text, error text, search behavior, selection behavior, and quit behavior while adding styling.

#### Scenario: Styled provider labels remain readable
- **WHEN** providers are displayed in the styled TUI
- **THEN** each visible provider row still contains the provider display label

#### Scenario: Styled code output keeps semantic values
- **WHEN** a selected provider has a generated code and remaining time
- **THEN** the styled output still contains the provider label, generated code, and remaining seconds
