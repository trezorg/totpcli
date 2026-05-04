## 1. Search Engine Core

- [x] 1.1 Update the search abstraction so the TUI can use provider-bound engines without passing the provider list on each query.
- [x] 1.2 Add engine name parsing and a factory that supports `digram` and `lexical`, defaults to `digram`, and rejects unsupported names.
- [x] 1.3 Implement the digram prefix-tree engine over normalized issuer, name, and display label fields.
- [x] 1.4 Preserve empty-query behavior and stable provider input order in both digram and lexical engines.

## 2. CLI Configuration

- [x] 2.1 Add `--search-engine` to the Cobra root command.
- [x] 2.2 Add `TOTPCLI_SEARCH_ENGINE` as the environment fallback for the search engine selector.
- [x] 2.3 Apply existing precedence rules so flags override environment variables and environment variables override defaults.
- [x] 2.4 Return a usage-style error for unsupported search engine names before launching the TUI.

## 3. Tests and Documentation

- [x] 3.1 Add search package tests for digram issuer, name substring, display label, empty query, stable order, and unsupported engine names.
- [x] 3.2 Update CLI tests for flag selection, environment selection, precedence, help output, and invalid engine errors.
- [x] 3.3 Update TUI tests for the provider-bound search abstraction if existing tests depend on the old interface.
- [x] 3.4 Update README usage and environment variable documentation for `--search-engine` and `TOTPCLI_SEARCH_ENGINE`.

## 4. Verification

- [x] 4.1 Run `gofmt` on changed Go files.
- [x] 4.2 Run `go test ./...`.
- [x] 4.3 Run the project pre-commit/check command if one is available.
