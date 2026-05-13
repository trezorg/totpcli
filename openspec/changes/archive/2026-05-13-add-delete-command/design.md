## Context

`totpcli` already routes workflows through Cobra commands in `internal/cli`, resolves provider database paths through `storage.ResolveDatabasePath`, and accesses SQLite through a repository boundary. Providers have user-facing identity fields (`issuer` and `name`) plus a TOTP secret that is already canonicalized for duplicate detection during add/import operations.

The delete workflow should remove stored records without requiring users to inspect or edit SQLite directly. It should also avoid introducing an interactive prompt because the existing command workflows are script-friendly and use explicit flags for behavior.

## Goals / Non-Goals

**Goals:**
- Add a focused `totpcli delete` command for removing one stored TOTP provider.
- Reuse `--totp` and `TOTPCLI_TOTP_PATH` path resolution with default `~/totp.sqlite`.
- Identify deletion candidates by trimmed `--issuer` and/or `--name`, with optional `--totp-secret` for disambiguation.
- Delete only when the selector resolves to exactly one provider.
- Keep all SQLite mutation inside `internal/storage`.

**Non-Goals:**
- Do not add bulk deletion, wildcard matching, fuzzy matching, or search-based deletion.
- Do not add interactive confirmation prompts.
- Do not expose database row IDs as a user-facing contract.
- Do not change provider schema or existing add/import duplicate behavior.

## Decisions

1. Use explicit identity flags instead of positional arguments.

   The command will accept `--issuer`, `--name`, `--totp-secret`, and `--totp`. At least one of `--issuer` or `--name` is required. Values are trimmed before matching so delete behavior is consistent with provider normalization and duplicate detection.

   Alternative considered: `totpcli delete <label>`. Labels such as `issuer/name` are convenient for display, but they are ambiguous when values contain separators and do not map cleanly to records with only issuer or only name.

2. Delete only exact, single matches.

   The repository will look for providers matching all supplied selector fields. If no provider matches, the command reports not found and exits non-zero without changing the database. If more than one provider matches, the command reports an ambiguous selector and exits non-zero, asking the user to provide more fields such as `--totp-secret`. Exactly one match is deleted in a transaction.

   Alternative considered: delete all matching rows. That is compact for cleanup scripts, but it makes a partial selector such as `--issuer GitHub` too dangerous for a first delete command.

3. Keep row IDs internal.

   SQLite rows already have an `id`, but existing provider-facing behavior does not expose it as an API. The delete command should not require users to discover database internals through `show`; it should operate on provider identity fields already used by add, import, display, and duplicate detection.

   Alternative considered: support `--id`. That would be precise, but it leaks storage implementation and would couple CLI behavior to the current SQLite schema.

4. Add a storage-level delete method.

   `internal/storage` should expose a single-provider delete method that receives a selector, performs matching and deletion inside SQLite, and returns the deleted provider or a typed result for not-found and ambiguous cases. The CLI should translate those outcomes into user-facing messages and exit codes.

   Alternative considered: list providers in `internal/cli`, select a match, and issue a lower-level delete by identity. That would split selection and mutation across layers and make it easier for command code to drift from repository constraints.

5. Report the deleted provider label and database path.

   Successful deletion should print a concise stdout summary including the resolved database path and the deleted provider display label. Usage errors and storage failures should continue to go to stderr through the existing Cobra error path.

   Alternative considered: silent success. Existing add/import commands report their effects, and delete is destructive enough that confirming the affected record is useful.

## Risks / Trade-offs

- Partial selectors can be ambiguous -> fail without mutation unless exactly one provider matches.
- Users may want bulk cleanup later -> leave bulk deletion out of scope and add it as a separate capability with stronger safeguards.
- Matching without row IDs can require an extra lookup before deletion -> acceptable for local SQLite provider databases and safer than exposing implementation IDs.
- TOTP secrets can appear in shell history when used for disambiguation -> keep `--totp-secret` optional and recommend issuer/name when they uniquely identify a record.

## Migration Plan

Implement as an additive CLI and repository change. Existing databases require no schema migration. Rollback is removing the new command route and repository delete method; existing records and commands remain compatible.

## Open Questions

- Should a later change add an interactive selection mode for ambiguous selectors?
- Should `show` eventually expose stable provider identifiers if users need deletion by exact storage row?
