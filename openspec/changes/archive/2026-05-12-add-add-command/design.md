## Context

The CLI already stores providers in SQLite and normalizes provider fields through `provider.Normalize`. Imports use repository methods to initialize storage and upsert multiple raw providers. The new command should add one provider without requiring a temporary JSON/YAML/TOML file and should preserve the same validation semantics users already get through imports.

## Goals / Non-Goals

**Goals:**
- Add a focused `totpcli add` command for one TOTP provider.
- Keep mandatory flags minimal: require `--totp-secret` and at least one of `--issuer` or `--name`.
- Derive fields that are already defaults or storage-managed: `algorithm=SHA1`, `digits=SIX`, `type=TOTP`, `totp_secret_key`, `id`, `created_at`, and `updated_at`.
- Update duplicates for manual adds so repeated commands can correct existing provider values.

**Non-Goals:**
- Add HOTP or non-TOTP provider support.
- Add interactive prompting.
- Add bulk creation; `import-file` remains the bulk provider-file path.
- Change the existing SQLite schema.

## Decisions

1. Expose provider identity and TOTP secret as CLI flags.

   The command will accept `--issuer`, `--name`, `--totp-secret`, `--secret`, `--algorithm`, `--digits`, and `--totp`. `--totp-secret` is mandatory because the application cannot generate TOTP codes without it. At least one of `--issuer` or `--name` is mandatory because provider display and duplicate detection need an identity. `--secret` remains optional because it is part of the existing record model but is not required for token generation.

   Alternative considered: positional issuer/name arguments. Flags are clearer for scripting and avoid ambiguity when one of issuer or name is intentionally omitted.

2. Derive unsupported or storage-owned fields.

   The command will not expose `--type`; it will build a raw provider with `Type: "TOTP"` before normalization. It will let existing normalization default omitted `--algorithm` to `SHA1` and omitted `--digits` to six digits. SQLite will derive `id`, timestamps, and `totp_secret_key`.

   Alternative considered: expose every database column as a flag. That would leak implementation details and allow invalid or misleading values for columns the database or provider normalization should control.

3. Add repository single-provider save separate from bulk import upsert.

   Storage should provide a single-provider operation that normalizes input, canonicalizes the TOTP secret for duplicate detection, inserts a new provider, and updates the existing provider when the unique key already exists. This keeps `add` useful both for first-time creation and for correcting algorithm, digits, secret metadata, or the displayed TOTP secret form for an existing provider.

   Alternative considered: reuse `UpsertProviders` for `add`. That would be less code, but the bulk import result shape is more than the command needs and would couple command output to import reporting.

4. Use the same database path resolution as TOTP display.

   `add` will resolve `--totp` and `TOTPCLI_TOTP_PATH` with the default `~/totp.sqlite`. That matches commands that operate on the default provider database instead of deriving a database path from an input file.

## Risks / Trade-offs

- Duplicate detection depends on normalized issuer, name, and canonical TOTP secret. Mitigation: reuse storage normalization/canonicalization paths already used by imports.
- Users may expect a `--type` flag because the database has a `type` column. Mitigation: document that `add` creates TOTP providers only and derives the stored type.
- Manual command lines can expose TOTP secrets in shell history. Mitigation: keep this change non-interactive for now and document the risk in help text where practical; interactive secret entry can be a later change.
