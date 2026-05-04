## Context

`totpcli` is a Go terminal application that currently parses `--totp`, `--format`, and `--search` with the standard `flag` package before loading providers and launching the Bubble Tea TUI. Provider path and format resolution already live behind `provider.ResolveOptions`, while TUI state and rendering are contained in `internal/tui`.

This change crosses CLI configuration, provider option resolution, and terminal rendering. It should preserve existing behavior for current users while adding a clearer command surface, environment variable fallbacks, and more readable terminal output.

## Goals / Non-Goals

**Goals:**

- Replace direct `flag` usage with a Cobra root command.
- Preserve the existing `--totp`, `--format`, and `--search` flags and exit-code behavior.
- Add `TOTPCLI_TOTP_PATH`, `TOTPCLI_FORMAT`, and `TOTPCLI_SEARCH` as environment variable fallbacks.
- Keep precedence explicit: flags override environment variables; environment variables override built-in defaults.
- Use Lip Gloss styles to make the TUI easier to scan: selected provider, provider rows, code, remaining time, empty state, help text, and errors should have distinct presentation.
- Keep the current provider loading, search, generation, and key bindings unchanged.

**Non-Goals:**

- Add new provider file formats or change provider file schemas.
- Add Viper or persistent configuration files.
- Change TOTP generation logic, search matching, or selection semantics.
- Add theming, user-customizable palettes, or non-terminal UI surfaces.

## Decisions

### Use Cobra without Viper

Cobra provides command metadata, usage/help output, and flag handling while keeping dependency scope small. Environment variables will be read through `os.LookupEnv` and applied explicitly instead of adding Viper, because this app only needs three fallback values and no config file loading.

Alternative considered: Cobra plus Viper. That would make env binding conventional for larger CLIs, but it adds a broader configuration model that this application does not currently need.

### Keep CLI orchestration in `internal/cli.App`

`App.Run(args []string) int` should remain the testable entry point. It will construct a Cobra command with injected stdout, stderr, home directory, and TUI runner dependencies, then execute it with the provided args. The command handler will resolve configuration, load providers, construct the TUI model, and run Bubble Tea as it does today.

Alternative considered: move command construction into `cmd/totpcli/main.go`. Keeping it in `internal/cli` preserves the current test boundary and avoids putting application behavior in the package entry point.

### Resolve CLI values before provider options

The CLI layer should compute effective `totpPath`, `format`, and `initialSearch` by checking whether a Cobra flag was changed, then falling back to the corresponding environment variable, then falling back to existing defaults. `provider.ResolveOptions` should remain responsible for expanding `~`, detecting formats from paths, and validating explicit format strings.

Alternative considered: make `provider.ResolveOptions` read environment variables. That would hide process-global behavior inside provider resolution and make it harder to test precedence independently from CLI parsing.

### Centralize TUI styles

The TUI should define a small style schema near the model rendering code using Lip Gloss. The `View` method can keep producing a single string, but rows and sections should be rendered through named styles so tests can continue asserting semantic content without depending on every escape sequence.

Alternative considered: introduce separate view components. The current model is small, so local style helpers are sufficient for this change.

## Risks / Trade-offs

- Cobra help output differs from `flag` package errors -> Update tests to assert stable user-facing facts instead of the exact full usage text.
- ANSI color escape codes can make TUI string tests brittle -> Keep tests focused on visible labels and state, and add targeted tests for structured rendering only where useful.
- Environment variables can create hidden configuration in tests -> Use explicit test environment setup and cleanup through `t.Setenv`.
- Adding Cobra changes dependencies -> Run `go mod tidy` and full `go test ./...` after implementation.

## Migration Plan

Implement the change behind the same executable and existing flag names. Users can continue running the current commands without changes. If rollback is needed, removing Cobra command construction and env fallback logic restores the previous `flag` behavior because the provider, search, and TUI contracts remain intact.

## Open Questions

None. The proposal uses fixed environment variable names and keeps the visual update scoped to built-in styling.
