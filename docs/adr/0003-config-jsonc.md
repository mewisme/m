# ADR 0003: Native config is JSONC (`m.jsonc`)

> **Status:** Accepted

## Context

MVP 0006 needs a Mew-native config format and project filename. Enrichment notes
suggested TOML vs JSON vs YAML. [`docs/naming.md`](../naming.md) already names
`m.jsonc` (project) and `config.jsonc` (global).

## Decision

1. Use **JSONC** for Mew-native global and project config.
2. Project file name is **`m.jsonc`** at the project root.
3. Global file is **`config.jsonc`** under the Mew config directory
   (`MEW_CONFIG_DIR`, else platform default from naming.md).
4. **Load** strips `//` and `/* */` comments then parses JSON.
5. **Set** rewrites deterministic pretty JSON. If the existing file contains
   comments outside strings, **Set fails closed** (`ERR_M_CONFIG`) instead of
   destroying comments. Full comment-preserving edit is deferred.

## Consequences

### Positive

- Aligns with naming.md; no new TOML/YAML dependency.
- Fail-closed Set avoids silent comment loss.

### Negative

- Users who hand-author comments must edit those files manually until a
  comment-preserving rewriter exists.

### Neutral

- Pass-through npmrc keys remain adapter-owned for non-Mew identities.
- CLI default write scope is user config; `--local` selects project `m.jsonc`
  (ADR path contract unchanged).

## Alternatives considered

| Alternative | Why rejected |
|---|---|
| TOML | Conflicts with naming.md; extra parser |
| YAML | Same |
| Silent comment strip on Set | Data-loss risk |

## Compatibility impact

| Axis | Impact |
|---|---|
| CLI | `m config` reads/writes JSONC |
| Lockfile | none |
| Config | new Mew-native files |
| Runtime | none |
| Layout | none |

State: extension (Mew-native config)

## Rollback

Stop writing `m.jsonc` / global `config.jsonc`; readers can ignore missing files.

## References

- Plan: `plans/0006-configuration-identity.md`
- Docs: `docs/config.md`, `docs/identity.md`
