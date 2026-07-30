# 0017 — Core MVP 8 — Transactional Install and Instant Rollback

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 8 |
| Primary objective | Make dependency mutations atomic at the product level and introduce install journals, snapshots, history, recovery, and instant rollback. |
| Required predecessors | 0016 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Make dependency mutations atomic at the product level and introduce install journals, snapshots, history, recovery, and instant rollback.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0016 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Mew signature feature; Nub behavior used only as baseline for normal install semantics

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m install
```
```bash
m history
```
```bash
m rollback
```
```bash
m recover
```

## In Scope

- Prepare, validate, and commit phases.
- Journaled manifest, lockfile, modules metadata, and node_modules transitions.
- Crash recovery after process kill or power-loss simulation.
- Snapshot history with retention and garbage collection.
- Rollback without network access when required blobs remain in store.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Never claim multi-file filesystem atomicity; implement recoverable transactions with a single authoritative journal and ordered commit protocol.
- Use rename-exchange where available, then portable rename choreography with recovery markers.
- Snapshots reference immutable store content rather than duplicating it.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Define transaction phases: inspect, resolve, plan, fetch, stage, validate, commit
- [ ] Create snapshot on successful commit with monotonic ID
- [ ] Add failure injection: kill process mid-commit, disk full, permission denied
- [ ] Emit transaction progress events to diagnostics reporter

### Core logic

- [ ] Journal every filesystem mutation with inverse operations
- [ ] Implement m snapshot list and m snapshot restore
- [ ] Add tests proving old node_modules works after failed install
- [ ] Support dry-run generating plan without journal writes

### CLI / UX

- [ ] Keep original manifest, lockfile, and node_modules until commit succeeds
- [ ] Validate staged tree before commit: integrity, bins, expected packages
- [ ] Document journal format and retention policy

### Tests & fixtures

- [ ] Implement rollback applying journal in reverse on any failure
- [ ] Integrate transaction boundary with install/add/remove from 0016
- [ ] Limit journal size with rotation policy

### Docs & observability

- [ ] Implement crash recovery: detect incomplete journal and offer recover
- [ ] Ensure partial fetch does not mutate committed state
- [ ] Never delete committed state without successful staging validation

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Failed install leaves prior node_modules intact and usable
- [ ] Acceptance: Interrupted commit can be recovered or cleanly rolled back
- [ ] Acceptance: Snapshot restore returns project to prior dependency state
- [ ] Acceptance: Journal records sufficient ops for full rollback
- [ ] Acceptance: Commit is atomic: no half-updated lockfile visible
- [ ] Fixture ready: `testdata/transaction/journal-samples/`
- [ ] Fixture ready: `fixtures/transaction/interrupted-install/`
- [ ] Fixture ready: `fixtures/transaction/rollback-hoisted/`


Required test layers:

- Unit tests for parsing, normalization, deterministic ordering, and error classification.
- Golden tests for manifests, lockfiles, command output, and migration reports.
- Integration tests against local fixture registries and isolated temporary homes.
- Failure-injection tests for network interruption, disk exhaustion, permission errors, process termination, and corrupted cache entries.
- Cross-platform tests for Linux, macOS, and Windows, including path length, case sensitivity, junctions, symlinks, and executable shims.
- Conformance tests comparing intentional compatibility surfaces with the corresponding Nub or package-manager behavior.

## Performance Requirements

- Commit and rollback should be dominated by filesystem relinking, not redownload.
- Snapshots must be metadata-light and deduplicate immutable content.
- Recovery scan must be bounded to affected projects and transactions.

All performance claims must be backed by reproducible benchmark commands, machine metadata, cold/warm cache separation, and multiple samples. Performance regressions on critical paths require an explicit waiver.

## Security and Trust Requirements

- Authenticate transaction paths against the project root.
- Protect journals from symlink substitution and untrusted ownership.
- Redact original registry credentials from persisted plans and history.

Secrets must never be written to logs, lockfiles, snapshots, telemetry, crash reports, or plan files. Archive extraction, script execution, registry authentication, and path construction must be treated as hostile-input boundaries.

## Risks and Mitigations

- Compatibility drift: mitigate with fixture-based conformance tests.
- Cross-platform divergence: mitigate with platform-specific CI and filesystem probes.
- Premature abstraction: require at least two concrete callers before generalizing an interface.

## Deliverables

- [ ] Production implementation and public interfaces.
- [ ] Unit, integration, conformance, and failure-injection tests.
- [ ] User documentation and migration notes where behavior is public.
- [ ] Benchmark baseline and diagnostic instrumentation.

## Exit Criteria

- [ ] All required tests pass on supported operating systems.
- [ ] No unresolved correctness, integrity, or data-loss issue remains.
- [ ] Public behavior and intentional deviations are documented.
- [ ] The next dependent MVP can consume stable interfaces without reaching into internals.



<!-- ENRICHMENT:BEGIN -->

## Feature Inventory Links

Rows this MVP owns or primarily advances (from `0002` inventory themes):

| Feature | Nub baseline | Mew target | Primary MVP |
|---|---|---|---|
| Atomic mutations | Nub transactions | inspect-resolve-plan-commit | 0017 |
| Install journal | Nub journal | Recoverable operation log | 0017 |
| Snapshot history | Nub snapshots | Point-in-time restore | 0017, 0028 |
| Rollback on failure | Nub | Old state remains usable | 0017 |

## Go Package Map

**Packages / paths:**

- `internal/transaction`
- `internal/app`
- `internal/linker`
- `internal/manifest`
- `internal/lockfile`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  plan[InstallPlan] --> stage[StagingArea] --> journal[Journal] --> commit[Commit] --> rollback[RollbackOnFailure]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m install` | `--journal` | Enhanced diagnostics |
| `m snapshot` | `list`, `restore <id>` | History management |
| `m recover` | — | Resume interrupted transaction |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Transaction journal | ~/.cache/github.com/mewisme/mew/journal or project-local |
| Snapshot store | Manifest + lock + node_modules metadata |
| Staging directories | Pre-commit install tree |

## Concrete Test Fixtures

- `testdata/transaction/journal-samples/`
- `fixtures/transaction/interrupted-install/`
- `fixtures/transaction/rollback-hoisted/`

## Acceptance Scenarios

1. Failed install leaves prior node_modules intact and usable
2. Interrupted commit can be recovered or cleanly rolled back
3. Snapshot restore returns project to prior dependency state
4. Journal records sufficient ops for full rollback
5. Commit is atomic: no half-updated lockfile visible

## Nub Conformance Targets

- Nub transaction atomicity | parity
- Snapshot restore semantics | parity
- Journal recovery behavior | parity

## Open Decisions

- Project-local vs global journal storage location
- Snapshot retention count and pruning policy

<!-- ENRICHMENT:END -->

## AI-Agent Handoff Contract

- Read 0000, 0003, 0005, 0007, 0008, and the immediate predecessor before changing code.
- Prefer small vertical pull requests over broad mechanical ports.
- Never copy Rust architecture blindly; preserve behavior and invariants using idiomatic Go.
- Update the feature matrix and conformance inventory when behavior changes.

Before submitting work, an agent must provide:

1. A concise behavior summary and the exact compatibility target.
2. A list of files and public interfaces changed.
3. Commands used for tests, benchmarks, and static analysis.
4. Known gaps, deferred cases, and platform limitations.
5. Evidence that generated files and fixtures are deterministic.
6. A rollback note for any persistent-format or migration change.
