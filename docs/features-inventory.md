# Feature inventory overview

Machine-readable source: [`features/inventory.json`](../features/inventory.json)

Ongoing inventory curation (status tweaks, notes, test links) is normal maintenance
and does not imply MVP **0002** is incomplete — the schema, CLI, and consistency
tests shipped with that MVP.

Maintenance: [`features-maintenance.md`](features-maintenance.md)

## Summary by module

| Module | Features | Shipped in Mew |
|---|---|---|
| cross-cutting | 11 | 1 |
| distribution | 5 | 0 |
| executable | 6 | 0 |
| foundation | 14 | 2 |
| lifecycle | 4 | 0 |
| linker | 9 | 0 |
| lockfile | 10 | 0 |
| node-manager | 3 | 0 |
| package-manager | 8 | 0 |
| plugin | 1 | 0 |
| pm-manager | 1 | 0 |
| product | 2 | 0 |
| registry | 7 | 0 |
| resolver | 10 | 0 |
| runner | 5 | 0 |
| runtime | 13 | 0 |
| security | 4 | 0 |
| shim | 3 | 0 |
| workspace | 3 | 0 |

**Total:** 117 features across 19 modules.

## Mew extensions (Nub-absent or signature improvements)

| ID | Feature | MVP |
|---|---|---|
| `foundation.features-inventory` | feature inventory and parity matrix | 0002 |
| `lockfile.m-lock` | m.lock native format | 0015 |
| `lockfile.semantic-diff` | semantic diff and validation | 0028 |
| `lockfile.migration` | explicit lock migration and loss report | 0028 |
| `linker.reflink-planner` | reflink and automatic filesystem planning | 0018 |
| `linker.transactional-install` | transactional install and recovery | 0017 |
| `linker.rollback-history` | instant rollback and history | 0017 |
| `linker.time-travel` | dependency time travel | 0028 |
| `linker.capsules` | portable capsules | 0029 |
| `security.policy-as-code` | policy-as-code | 0030 |
| `runner.direct-shortcuts` | direct m dev / m start shortcuts | 0042 |
| `runner.interactive-select` | interactive script selection | 0090 |
| `exec.snapshot-capsule` | snapshot and capsule execution | 0045 |
| `cross.future-backlog` | post-parity future extensions backlog | 0090 |

## CLI

```powershell
go run ./cmd/m features --format table
go run ./cmd/m features --format json --module runner --status planned
go run ./cmd/m version
```

Command tree: `internal/cli` ([Cobra](https://cobra.dev/)).

Counts in the summary table are derived from `features/inventory.json` at MVP 0002 completion. Regenerate after inventory changes.
