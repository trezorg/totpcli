## ADDED Requirements

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
