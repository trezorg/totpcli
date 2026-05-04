## 1. CLI Configuration

- [x] 1.1 Add CLI tests for `--no-tui`, `TOTPCLI_NO_TUI`, flag precedence, default TUI mode, help output, and invalid boolean environment values.
- [x] 1.2 Add `--no-tui` parsing to the Cobra root command and resolve it with flag-over-environment-over-default precedence.
- [x] 1.3 Document `--no-tui` and `TOTPCLI_NO_TUI` in Cobra help text.

## 2. Non-Interactive Output

- [x] 2.1 Add tests for printing all providers with token and remaining seconds in `--no-tui` mode.
- [x] 2.2 Add tests for `--no-tui --search` printing only filtered providers and printing no token rows for empty results.
- [x] 2.3 Implement non-interactive output using the resolved search engine, one captured timestamp, and existing TOTP generation.
- [x] 2.4 Ensure `--no-tui` exits immediately after one generation pass and does not call the TUI runner.

## 3. Error Handling

- [x] 3.1 Add tests for matched invalid providers causing a non-zero exit and filtered-out invalid providers not failing the command.
- [x] 3.2 Return generation errors through the existing CLI error path with non-zero exit behavior.

## 4. Verification

- [x] 4.1 Run `gofmt` on changed Go files.
- [x] 4.2 Run `make test`.
- [x] 4.3 Run `make lint`.

## 5. Table Output

- [x] 5.1 Compare Go table rendering libraries and select the existing Lip Gloss table package.
- [x] 5.2 Add failing CLI tests that require no-TUI output to include table headers and ASCII borders.
- [x] 5.3 Render no-TUI token output as a table with `Provider`, `Token`, and `Remaining` columns.
