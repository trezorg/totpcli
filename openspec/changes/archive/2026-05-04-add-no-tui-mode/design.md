## Context

The CLI currently resolves provider path, provider format, initial search text, and search engine from flags, environment variables, and defaults before loading providers and launching the TUI. The TUI model owns filtering state and selected-provider code refresh behavior, while TOTP generation itself already lives in `internal/totpcode` and returns both the generated code and remaining seconds.

`--no-tui` should add a second presentation path after the same configuration, loading, and search setup. It should not change the interactive path, and it should not introduce a long-running process.

## Goals / Non-Goals

**Goals:**
- Provide a non-interactive CLI mode that prints current tokens and remaining time for matching providers.
- Reuse existing provider loading, search engine selection, and TOTP generation behavior.
- Preserve flag-over-environment-over-default precedence for the new mode selection.
- Keep table output stable enough for tests and simple shell use.

**Non-Goals:**
- Add a watch mode or repeated refresh output.
- Add a new output format flag.
- Change provider file formats, search matching rules, or TUI behavior.

## Decisions

Use a boolean `--no-tui` flag with `TOTPCLI_NO_TUI` fallback.

This matches the existing CLI configuration style and lets non-interactive use be selected without adding a new subcommand. An alternative was a `print` subcommand, but that would split existing configuration behavior and make the current root command harder to reason about.

Create a small non-interactive output path in the CLI layer.

The CLI already coordinates configuration, provider loading, search engine construction, and TUI launch. Keeping non-TUI rendering there, or in a small helper owned by `internal/cli`, avoids making the TUI model responsible for non-interactive presentation. An alternative was to instantiate a TUI model and read `Filtered()`, but that couples batch output to UI state.

Render non-interactive results with `lipgloss/table` and ASCII borders.

Lip Gloss is already part of the application stack, and its table package provides table rendering without adding a separate table ecosystem. Alternatives considered were `olekukonko/tablewriter`, which is focused and feature-rich but adds another dependency family, and `jedib0t/go-pretty/table`, which is mature but broader than this output path needs.

Generate all matching rows from one captured `now`.

Using one timestamp produces consistent remaining seconds across the output and guarantees the command exits after one generation pass. An alternative was calling `time.Now` per provider, but that can produce inconsistent remaining values when output spans a boundary.

Print one line per successfully generated provider and return an error if any matching provider cannot generate a token.

Failing the command on invalid matched data keeps scripting behavior explicit. An alternative was printing partial successes and warning to stderr, but that is harder to detect in automation.

## Risks / Trade-offs

Invalid provider matched in non-TUI mode fails the whole command -> Tests should cover the exit code and stderr path.

The table format becomes a user-facing contract -> Keep headers and columns simple and documented by specs before implementation.

Environment parsing for booleans can be ambiguous -> Accept conventional truthy values and reject unsupported values as usage errors.
