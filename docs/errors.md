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
| `ERR_M_POLICY` | 1 | Lifecycle script trust block (0021) or org supply-chain policy violation (0030) |
| `ERR_M_PNP_UNSUPPORTED` | 1 | Yarn Berry PnP install blocked (MVP 0025; see [`yarn-lockfile.md`](yarn-lockfile.md)) |
| `ERR_M_INTEGRITY` | 1 | Ambiguous incomplete transaction state, tree manifest collision, or verification failure |

### Transaction detail (0017 journal v3)

| Situation | Code | Notes |
|---|---|---|
| Concurrent install (`lock` held) | `ERR_M_TRANSACTION` | Another process holds `.mew/txn/lock` |
| Lock wait cancelled | `ERR_M_CANCELLED` | Context cancelled during `AcquireProjectLock` |
| Multiple incomplete `committing` journals | `ERR_M_INTEGRITY` | Directory scan found ambiguous state |
| Incomplete txn after preflight recovery | `ERR_M_INTEGRITY` | `BeginMutation` refused to start |
| Lock release without ownership | `ERR_M_TRANSACTION` | `ReleaseDirLock` returned `ReleaseNotOwner` / `ReleaseMissingOwner` |
| Commit / publish failure | `ERR_M_TRANSACTION` | Roll back via `m recover` when incomplete |
| Recovery failure | `ERR_M_TRANSACTION` | Partial `node_modules` rename may need manual cleanup |
| Symlink/junction in guarded path | `ERR_M_TRANSACTION` | Ancestor guard on `.mew` / `node_modules` / snapshots |
| Post-commit prune failure | `ERR_M_IO` | Install already committed; retry prune or `m snapshot list` |
| Post-commit cleanup incomplete | `ERR_M_TRANSACTION` | Lock released or `current` clear failed after commit; run `m recover` |
| Store import lock release failure | (warning only) | `ImportResult.CleanupWarnings`; published tree remains valid |
| StoreID collision during isolated layout | `ERR_M_INTEGRITY` | Collision-resistant digest still collided (extremely rare) |
| Windows directory sync denied | (none — no-op) | `fsx.SyncDir` ignores access-denied on directory handles; file sync still runs |

Unknown codes map to exit **1**.

## Supply-chain security (MVP 0030)

| Situation | Code | Notes |
|---|---|---|
| Advisory cache missing with `--offline` | `ERR_M_NETWORK` | `app.audit`; seed `<cache>/advisory/osv.json` |
| Advisory cache missing (online) | `ERR_M_NOT_FOUND` | Same path; copy or refresh advisory DB |
| Org policy violation on `m policy check` | `ERR_M_POLICY` | `policy.check`; use `--json` for violations |
| Org policy violation on install validate | `ERR_M_POLICY` | `app.policy` / `install`; transaction rolls back |
| Invalid `mew.policy.json` | `ERR_M_CONFIG` | `policy.load` / `policy.normalize` |
| Provenance attestation mismatch | `ERR_M_INTEGRITY` | `verify.provenance` / `app.provenance` |
| Unknown SBOM format | `ERR_M_USAGE` | `app.sbom`; use `cyclonedx` or `spdx` |

See [`audit.md`](audit.md), [`sbom.md`](sbom.md), [`policy.md`](policy.md).

## Go API

Package [`internal/apperr`](../internal/apperr): `New`, `Wrap`, `CodeOf`, `ExitCode`.

Every public failure path should return an `*apperr.Error` (or wrap into one at the CLI boundary).

## Debug bundles

Future `m doctor report` archives must require explicit consent and apply the same
redaction rules as reporters. Spec pointer: [`reporters.md`](reporters.md).
Not implemented in 0005.
