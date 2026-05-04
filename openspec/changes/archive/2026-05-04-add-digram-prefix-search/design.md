## Context

`totpcli` currently has a small `search.Engine` abstraction and a lexical implementation that scans every provider field on each input change. The TUI already depends on the abstraction rather than concrete lexical behavior, but the engine is constructed directly in `internal/cli`.

The requested change adds a prefix search tree based on digrams, makes it the default, and lets users select the engine from both CLI flags and environment variables. Web research found related options:

- `github.com/Vivino/go-autocomplete-trie` provides a Go trie with fuzzy, normalization, and Levenshtein behavior, but it is broader than the immediate need and would add a dependency for behavior we can implement simply.
- `github.com/lithammer/fuzzysearch` provides lightweight fuzzy matching and ranking over lists, but it is not an indexed prefix tree.
- Suggest Trees precompute top-k weighted suggestions in compressed tries, which is useful for ranked autocomplete but unnecessary while provider results remain stable input-order lists.
- n-gram inverted index research supports approximate string matching at larger scale, but also introduces merge/filter complexity and possible false positives that are not needed for local provider files.
- Go's `index/suffixarray` supports fast substring lookup over bytes, but it does not preserve provider-level result semantics or prefix-tree behavior.

## Goals / Non-Goals

**Goals:**

- Add a digram prefix-tree search engine that supports case-insensitive matching across issuer, name, and display label.
- Preserve current empty-query and stable input-order result behavior.
- Keep lexical search available under an explicit engine name.
- Resolve the selected engine through command-line flags, environment variables, and defaults using existing precedence rules.
- Keep the implementation dependency-free and small enough to test directly.

**Non-Goals:**

- Add fuzzy edit-distance matching, typo correction, personalization, or weighted ranking.
- Persist the search index across process runs.
- Change provider file formats or TUI interaction behavior.
- Add dynamic provider updates after the TUI starts.

## Decisions

### Use engine names `digram` and `lexical`

The CLI will accept `--search-engine digram|lexical`, with `TOTPCLI_SEARCH_ENGINE` as the environment fallback. The built-in default is `digram`.

Alternative considered: boolean flags such as `--lexical-search`. Named engines scale better when another implementation is added and align with the existing pluggable abstraction.

### Bind indexed engines to loaded providers

The current interface accepts providers on every search call. The digram engine should instead build its index once after providers are loaded, then answer queries against that index. Introduce a provider-bound search abstraction for the TUI, for example `Search(query string) []provider.Provider`, and construct it through a small factory that receives the selected engine name and provider list.

Alternative considered: keep `Search(query, providers)` and rebuild or cache internally. That keeps the interface stable but hides lifecycle state and risks rebuilding the tree during TUI input changes.

### Build a suffix-aware digram prefix tree

For each provider, normalize issuer, name, and display label by trimming and case-folding. Insert every suffix of each normalized field into a prefix tree whose path starts with the first rune and then advances by adjacent digrams. This preserves existing substring behavior while still using prefix-tree traversal for queries. Each node stores provider indexes in input order, deduplicated per provider.

Examples:

- `github` inserts suffix paths for `github`, `ithub`, `thub`, `hub`, `ub`, and `b`.
- Query `git` traverses `g`, `gi`, `it`.
- Query `hub` traverses `h`, `hu`, `ub`.

Alternative considered: index only full field prefixes. That is simpler but would regress existing searches such as matching `trezorg` with `zorg`.

### Preserve stable ordering instead of ranking

The digram engine returns matching providers in the original provider-file order. This keeps TUI behavior predictable and avoids inventing ranking rules before there is a product need.

Alternative considered: store top-k or weighted lists at each node, similar to Suggest Trees. That is useful for large autocomplete systems but premature for local TOTP provider selection.

### Validate engine configuration before TUI startup

The CLI resolves the effective engine name before creating the model. Unsupported values return a usage-style error and do not launch the TUI.

Alternative considered: silently fallback to the default. Explicit errors make configuration mistakes visible and match existing CLI validation expectations.

## Risks / Trade-offs

- Duplicate suffix insertion can increase memory use for long labels -> provider files are expected to be small, and node postings are provider indexes rather than provider copies.
- Unicode normalization beyond case-folding can be subtle -> keep initial normalization consistent with current lexical case-insensitive behavior and add tests for non-ASCII case-folding only if the implementation supports it.
- The interface change touches TUI tests and CLI model construction -> keep the factory small and update tests around observable behavior.
- The digram engine may look more complex than lexical scan -> isolate it in `internal/search` and keep lexical as a fallback for troubleshooting.

## Migration Plan

1. Add search engine name parsing and a provider-bound engine factory.
2. Implement and test the digram prefix tree in `internal/search`.
3. Update CLI configuration, help output, and tests for `--search-engine` and `TOTPCLI_SEARCH_ENGINE`.
4. Update TUI construction to receive the provider-bound engine.
5. Keep `lexical` available so users can opt out with `--search-engine lexical` if needed.
