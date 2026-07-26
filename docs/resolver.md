# Resolver

Dry dependency resolution: registry metadata in, deterministic `graph.Graph` and
decision traces out. No `node_modules` mutation (0016). Lockfile write is via
`app.WriteLock` / `m lock` (0015).

## Inputs

| Input | Source |
|---|---|
| Root manifest | `project.Open` → `manifest.ToNormalized` |
| Registry | `registry.Client.Packument` via scoped URL routing |
| Policy | `ResolveOptions.Policy` — `MinimumReleaseAge`, `RejectDeprecated` |
| Hints | `ResolveOptions.Hints *graph.Graph` — prefer pinned versions when the range still satisfies (0015 prep) |

## Outputs

`resolver.Resolution`:

- `SchemaVersion`
- `Graph` — importers, packages, edges (validated / sorted)
- `Decisions` — per-request candidates, selected version, reason, policy rejects

## Rules (0013)

- Root importer resolves `dependencies` and `devDependencies`.
- Transitive packages expand **prod** `dependencies` only — never their `devDependencies`.
- Peers, optional deps, and overrides are ignored (0020).
- Semver: `internal/semver` over Masterminds/v3 (`^`, `~`, `*`, `x`, unions, hyphen ranges). Dist-tags resolve in the registry layer before range selection.
- Fail closed: missing packument, unsatisfiable range, dependency cycle, or limit exceeded → `ERR_M_RESOLVE`.
- Limits: `maxDepth=64`, `maxPackages=10000` (raise later for large monorepos).

## CLI

```text
m resolve [--plan] [--json] [--trace]
```

`--plan` is the dry-resolve mode (default). `--json` emits `Resolution`.
`--trace` prints human-readable decision lines. Full `m explain` is MVP 0028;
`--trace` / `Decisions` are the explain input.

## Lockfile handoff (0015)

`app.WriteLock` builds `m.lock` from `Resolution` plus manifest specifiers and
config settings. `app.ReadLockGraph` feeds `ResolveOptions.Hints` for
incremental resolve. See [`lockfile.md`](lockfile.md).

## Related

- [`docs/errors.md`](errors.md) — `ERR_M_RESOLVE`
- [`docs/architecture/interfaces.md`](architecture/interfaces.md) — `Resolver`
- [`docs/data-model.md`](data-model.md) — canonical graph
