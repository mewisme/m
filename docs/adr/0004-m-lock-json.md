# ADR 0004: Native lockfile is JSON (`m.lock`)

> **Status:** Accepted

## Context

MVP 0015 needs Mew's native lockfile at the project root. [`docs/naming.md`](../naming.md)
names **`m.lock`**. The canonical dependency graph from MVP 0007 already serializes as
deterministic JSON. Importer specifiers, linker settings, and policy snapshots must
round-trip for frozen installs (0016) and adapter MVPs (0023+).

## Decision

1. Use **deterministic JSON** (pretty, 2-space indent, trailing newline) for `m.lock`.
2. File location is **`m.lock` at the project root** only.
3. Top-level **`lockfileVersion`** (currently `1`) is independent of `graph.SchemaVersion`.
4. Package entries carry minimal dist: `PackageID`, `integrity`, `tarballUrl`.
5. **`importers[]`** hold sorted `specifiers[]` (`name`, `range`, `kind`) for frozen drift detection.
6. **`settings`** snapshot `linker` and `policy` for install handoff.
7. **`checksum`** is lowercase hex SHA-256 of canonical JSON over `settings`, `importers`,
   `packages`, and `edges` (excludes `checksum` and `lockfileVersion`).
8. Unknown top-level fields are preserved in `extensions` on read; writers omit empty `extensions`.
9. Atomic write uses temp file + rename (same pattern as `package.json`).

## Consequences

### Positive

- Byte-stable across OS; same encoder settings as `graph.EncodeJSON`.
- Semantic checksum detects tampering without relying on presentation-only fields.
- Forward-compatible via `extensions` and versioned `lockfileVersion`.

### Negative

- Large graphs produce large JSON files (acceptable for v1; compression deferred).

### Neutral

- `m install --frozen-lockfile` remains 0016; `m lock validate --frozen` is the 0015 validator.

## Alternatives considered

| Alternative | Why rejected |
|---|---|
| YAML | Extra encoder; harder to guarantee byte stability |
| TOML | Same |
| Embed full packuments | Bloat; registry is source of truth |

## Compatibility impact

| Axis | Impact |
|---|---|
| CLI | `m lock format`, `m lock validate [--frozen]` |
| Lockfile | new native `m.lock` v1 |
| Config | settings snapshot only |
| Runtime | none |
| Layout | root `m.lock` |

State: extension (Mew-native lockfile)

## Rollback

Stop writing `m.lock`; readers treat missing file as greenfield. Existing v1 files remain
parseable until a breaking `lockfileVersion` bump (requires migration MVP 0028).

## References

- Plan: `plans/0015-m-lock.md`
- Docs: `docs/lockfile.md`, `docs/data-model.md`
