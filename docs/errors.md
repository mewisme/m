# Error codes

Stable machine-readable codes for Mew CLI failures. Pattern: `ERR_M_<DOMAIN>_<DETAIL>`
(see [`naming.md`](naming.md)). Nub `ERR_NUB_*` codes are behavioral references only.

## Registry (MVP 0005)

| Code | Exit | Meaning |
|---|---|---|
| `ERR_M_OK` | 0 | Sentinel; not used for failures |
| `ERR_M_USAGE` | 2 | Invalid arguments or flag misuse |
| `ERR_M_CANCELLED` | 130 | Context canceled / interrupt |
| `ERR_M_INTERNAL` | 1 | Unexpected failure |
| `ERR_M_INTERNAL_PANIC` | 1 | Panic recovered at command boundary |
| `ERR_M_IO` | 1 | Filesystem I/O |
| `ERR_M_CONFIG` | 1 | Configuration (seed for 0006) |
| `ERR_M_NETWORK` | 1 | Network / registry (seed) |
| `ERR_M_INTEGRITY` | 1 | Checksum / integrity (seed) |
| `ERR_M_LOCKFILE` | 1 | Lockfile parse, checksum, graph, or frozen manifest drift (MVP 0015) |
| `ERR_M_UNIMPLEMENTED` | 1 | Reserved command stub not yet implemented (MVP 0010) |
| `ERR_M_MANIFEST` | 1 | package.json parse / validate (MVP 0011) |
| `ERR_M_NOT_FOUND` | 1 | Project root or package.json missing (MVP 0011) |
| `ERR_M_RESOLVE` | 1 | Dependency resolution failure: unsatisfiable range, cycle, missing packument, or limit exceeded (MVP 0013) |
| `ERR_M_TRANSACTION` | 1 | Transaction journal, commit, rollback, recovery, or project lock failure (MVP 0017) |
| `ERR_M_STORE` | 1 | Global content store import, verify, or prune failure (MVP 0018) |

### Transaction detail (0017 journal v3)

| Situation | Code | Notes |
|---|---|---|
| Concurrent install (`lock` held) | `ERR_M_TRANSACTION` | Another process holds `.mew/txn/lock` |
| Lock wait cancelled | `ERR_M_CANCELLED` | Context cancelled during `AcquireProjectLock` |
| Commit / publish failure | `ERR_M_TRANSACTION` | Roll back via `m recover` when incomplete |
| Recovery failure | `ERR_M_TRANSACTION` | Partial `node_modules` rename may need manual cleanup |
| Symlink/junction in guarded path | `ERR_M_TRANSACTION` | Ancestor guard on `.mew` / `node_modules` / snapshots |
| Post-commit prune failure | `ERR_M_IO` | Install already committed; retry prune or `m snapshot list` |
| StoreID collision during isolated layout | `ERR_M_INTEGRITY` | Collision-resistant digest still collided (extremely rare) |

Unknown codes map to exit **1**.

## Go API

Package [`internal/apperr`](../internal/apperr): `New`, `Wrap`, `CodeOf`, `ExitCode`.

Every public failure path should return an `*apperr.Error` (or wrap into one at the CLI boundary).

## Debug bundles

Future `m doctor report` archives must require explicit consent and apply the same
redaction rules as reporters. Spec pointer: [`reporters.md`](reporters.md).
Not implemented in 0005.
