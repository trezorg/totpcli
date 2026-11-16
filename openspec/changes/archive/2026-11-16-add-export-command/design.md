## Context

`totpcli` stores providers in SQLite and already parses JSON, YAML, and TOML provider files through `provider.ParseRaw` for `import-file`. The provider file formats are documented in the README. The database stores TOTP seeds in cleartext and is written with `0600` permissions; `show --reveal-secrets` output carries the same warning. There is no serialization counterpart to `ParseRaw`, so a text backup path does not exist.

## Goals / Non-Goals

**Goals:**
- Add `totpcli export` writing every stored provider to a JSON, YAML, or TOML provider file.
- Make the output restorable by the existing `import-file` command with no field loss on identity and secret data.
- Protect secret material: `0600` permissions and refuse-overwrite without `--force`.
- Reuse `--totp`/`TOTPCLI_TOTP_PATH` database path resolution.

**Non-Goals:**
- Do not export `id`, `created_at`, or `updated_at`; they are recreated on import.
- Do not encrypt or prompt for a passphrase.
- Do not add a new restore command; `import-file` restores.
- Do not stream or paginate; provider counts are small.

## Decisions

1. Encode through the existing `RawProvider` shape.

   `RawProvider` already carries struct tags for JSON, YAML, and TOML and is what `ParseRaw` consumes. Exporting that same shape (TOML wrapped in a `providers` array-of-tables) makes `export` and `import-file` symmetric by construction and keeps one canonical provider file format.

   Alternative considered: export full table snapshots like `show`. That preserves `id` and timestamps but `import-file` cannot consume it, so it would require a second new command and format for no user-visible benefit.

2. Enforce file safety inside `provider.Save`.

   The exist-check (`ErrFileExists` unless overwrite) and the `0600` permission belong next to the write, not in the CLI handler, so every future caller inherits the guarantees. A forced overwrite re-chmods to `0600` because a pre-existing user file may have looser permissions.

   Alternative considered: check-and-write in the Cobra handler. That leaves the library caller free to clobber secret files silently.

3. Digits export as words.

   `Provider.Digits` is an int; `RawProvider.Digits` accepts `SIX`, `EIGHT`, `6`, or `8`. Export `6` as `SIX` and `8` as `EIGHT` to match the README format examples and existing `add` defaults.

4. Explicit `--format` beats the `--output` extension; default output is `~/totp.<format>`.

   This mirrors `import-file` precedence and `provider.ResolveOptions` defaulting: JSON when nothing indicates otherwise.

## Risks / Trade-offs

- Export files are cleartext secret material -> `0600` permissions, overwrite refusal, and a README warning parallel to the database warning.
- Dropping `id`/timestamps means a restored database gets fresh ones -> acceptable; they carry no user-facing meaning.
- Exporting an empty database produces an empty document -> specified and tested as valid; importing it is a no-op.
