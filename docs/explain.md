# Explain commands

Read-only resolver diagnostics: why a package version was selected, or why a peer
dependency could not be satisfied. No manifest, lockfile, or `node_modules`
mutation (MVP **0028**).

## Commands

```text
m explain <name> [--json]
m explain peer <name> [--json]
```

| Subcommand | Purpose |
|---|---|
| `explain <name>` | Version selection path for a resolved package name |
| `explain peer <name>` | Peer conflict tree when strict peers fail (or “no peer conflict” when satisfied) |

`m explain` with no arguments prints help.

## Trace schema

`m explain <name> --json` emits `resolver.PackageExplanation`:

| Field | Type | Meaning |
|---|---|---|
| `package` | string | Requested package name |
| `decisions` | `ResolutionDecision[]` | Matching decision trace entries |
| `paths` | `ImportPath[]` | Importer → instance chains (`importer`, `chain`) |
| `conflict` | `ConflictTree` | Present when resolve failed on a peer with this name |

Each `ResolutionDecision` (see [`data-model.md`](data-model.md)) includes:

| Field | Meaning |
|---|---|
| `package` | Requested name |
| `requested` | Range or specifier |
| `candidates` | Considered versions |
| `selected` | Chosen version or package key |
| `reason` | Selection reason code (see below) |
| `rejected` | Policy-filtered versions |
| `peerProviders` | Active peer provider context |
| `overrideFrom` | Override specifier when npm `overrides` rewrote the edge |

### Reason codes

Human output maps `reason` through `ReasonDetailFor`. Common values:

| Reason | Meaning |
|---|---|
| `reuse-key` | Incremental reuse from prior lock |
| `hint` | Prior graph hint |
| `max-satisfying` | Highest semver match in range |
| `tag-or-exact` | Exact version or dist-tag |
| `workspace` | `workspace:` protocol member |
| `platform-skipped` | Optional dep skipped for OS/CPU/libc |

When a reason implies a user-facing failure, the detail may include an
`ERR_M_*` hint (for example `platform-skipped` → `ERR_M_RESOLVE`). Full code
list: [`errors.md`](errors.md).

## Peer conflict tree

`m explain peer <name> --json` emits `resolver.ConflictTree`:

| Field | Meaning |
|---|---|
| `peer` | Peer package name |
| `root` | `ConflictNode` root (constraint, importer, search path, candidates, rejected, remediation, children) |

Golden fixtures: [`testdata/resolver/explain/`](../testdata/resolver/explain/),
[`fixtures/explain/`](../fixtures/explain/).

## Relation to `m resolve --trace`

| Command | Scope |
|---|---|
| `m resolve --trace` | Full resolve; all decision lines |
| `m explain <pkg>` | One package: decisions + import paths |
| `m explain peer <pkg>` | Peer failure only |

Use `m explain` for targeted “why this version?” questions; use `m resolve
--trace` when auditing the entire graph.

## Error codes

| Situation | Code |
|---|---|
| Missing project / manifest | `ERR_M_NOT_FOUND`, `ERR_M_MANIFEST` |
| Package not in graph | `ERR_M_NOT_FOUND` (`resolver.explain`) |
| Unsatisfiable resolve (non-peer explain) | `ERR_M_RESOLVE` |
| Registry / network | `ERR_M_NETWORK` |
| Invalid usage | `ERR_M_USAGE` |

## Deferred

- `m why` alias — use `m explain` (npm parity name deferred)
- `m shell --snapshot` / `m run --snapshot` — MVP **0045**

See also [`resolver.md`](resolver.md), [`plan.md`](plan.md).
