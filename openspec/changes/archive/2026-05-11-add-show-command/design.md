## Context

`totpcli` currently resolves provider storage paths through `storage.ResolveDatabasePath`, opens SQLite with `storage.OpenSQLite`, and routes user-facing workflows through Cobra commands in `internal/cli`. Imports write provider records into SQLite, and TOTP display reads normalized providers through a repository boundary.

The `show` command is a read-only inspection workflow. It needs access to table names, column names, and row values rather than normalized provider records, so it should not overload `ListProviders`. The command still belongs in the same CLI and storage boundaries because it uses the same database path resolution and SQLite connection lifecycle.

## Goals / Non-Goals

**Goals:**
- Add `totpcli show` for read-only SQLite database inspection.
- Support `--format table|json|yaml|toml`, defaulting to `table`.
- Output every non-internal SQLite table in a deterministic order.
- Reuse existing `--totp` and `TOTPCLI_TOTP_PATH` path resolution.
- Keep format rendering testable without requiring a real terminal UI.

**Non-Goals:**
- Do not add SQL query input, filtering, pagination, or table selection.
- Do not expose SQLite internal tables such as `sqlite_sequence`.
- Do not generate TOTP tokens or apply search filtering in `show`.
- Do not change import or TOTP display behavior.

## Decisions

1. Add a storage-level table snapshot API.

   The SQLite repository should expose a read-only method that returns table snapshots containing table name, ordered columns, and ordered rows. This keeps raw SQL introspection inside `internal/storage`, where SQLite details already live, instead of embedding SQL in Cobra handlers.

   Alternative considered: query SQLite directly from `internal/cli`. That would be faster to wire, but it would weaken the existing repository boundary and make storage behavior harder to unit test.

2. Use one normalized in-memory shape for all renderers.

   Represent the result as a list of tables, each with columns and row values keyed by column name. JSON, YAML, TOML, and table output can all render from that structure. SQLite `NULL` values should remain distinguishable in machine-readable formats; table output can render them as an empty cell or a clear textual null marker.

   Alternative considered: build each format directly from `sql.Rows`. That would duplicate scanning and type conversion logic across renderers.

3. Keep `--format` scoped to `show`.

   Existing specs explicitly reject `--format` on root, `totp`, and `import`; `import-file` already has its own provider input format flag. The new `show --format` flag should not introduce a `TOTPCLI_FORMAT` fallback because existing CLI configuration avoids a global format environment variable.

   Alternative considered: reuse `TOTPCLI_FORMAT` globally. That would conflict with existing behavior and blur the meaning of `--format` across unrelated commands.

4. Prefer existing dependencies first.

   The module already includes `gopkg.in/yaml.v3` and `github.com/pelletier/go-toml/v2`. JSON can use the standard library. Table output can use the existing Charm table renderer already used for non-interactive token output unless implementation tests show it is unsuitable for multi-table output.

   Alternative considered: add a new table rendering library. That is unnecessary unless the existing renderer cannot handle the required output cleanly.

## Risks / Trade-offs

- Arbitrary SQLite values may include blobs or driver-specific types -> convert values through a small, documented normalization function and cover NULL, numeric, text, boolean, and blob cases in tests.
- TOML cannot represent every JSON/YAML shape exactly -> choose a TOML-friendly structure during implementation and test round-tripping/parsing of representative output.
- Large databases may produce very large output -> acceptable for this first inspection command because filtering and pagination are non-goals.
- Empty databases may be surprising if schema initialization creates tables -> specify and test the expected output for empty tables so users get deterministic results.
