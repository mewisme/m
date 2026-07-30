# 0061 — Manager MVP 2 — Package-Manager Meta-Manager

## Document Control

| Item | Detail |
|---|---|
| Phase | Managers / MVP 2 |
| Primary objective | Detect, download, cache, pin, invoke, and migrate external package managers as a Corepack replacement and compatibility tool. |
| Required predecessors | 0023, 0024, 0025, 0060 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Detect, download, cache, pin, invoke, and migrate external package managers as a Corepack replacement and compatibility tool.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0023 before starting this MVP.
- Complete and merge 0024 before starting this MVP.
- Complete and merge 0025 before starting this MVP.
- Complete and merge 0060 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub `pm which`, `pin`, `migrate`, `update`, and cache behavior

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m pm which
```
```bash
m pm pin pnpm@10
```
```bash
m pm migrate bun
```
```bash
m pm update
```
```bash
m pm cache list
```

## In Scope

- Detection from packageManager, devEngines, lockfiles, installed executables, and config.
- Verified package-manager acquisition.
- Version pinning and project updates.
- Lockfile/config migration planning and execution.
- External PM invocation under selected Node.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- External managers run in isolated verified environments.
- Migration uses lock adapters and always produces a semantic loss report and rollback snapshot.
- Per-major compatibility models drive config writes.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement package-manager identity and version resolver
- [ ] Implement m pm pin and update commands with per-major models
- [ ] Never silently rewrite incumbent lockfile identity
- [ ] Document Corepack replacement positioning

### Core logic

- [ ] Detect from packageManager, devEngines, lockfiles, executables, config
- [ ] Implement migration planner using lock adapters from core PM MVPs
- [ ] Preserve m.lock as native; support npm/pnpm/Yarn/Bun round trips
- [ ] Benchmark PM invocation startup overhead

### CLI / UX

- [ ] Implement verified acquisition and content-addressed PM cache
- [ ] Execute migrations inside transaction with rollback snapshot
- [ ] Add pinned manager install and invocation tests
- [ ] Integrate trust policy for downloaded manager artifacts

### Tests & fixtures

- [ ] Run external managers in isolated verified environments
- [ ] Produce semantic loss report for every migration
- [ ] Add migration corpus with rollback verification

### Docs & observability

- [ ] Select Node version via 0060 for external PM invocation
- [ ] Implement cache inspection and prune commands
- [ ] Test ambiguous identity detection and diagnostics

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: m pm which reports correct manager for incumbent lockfiles
- [ ] Acceptance: Pinned manager version used for invocation
- [ ] Acceptance: Migration produces loss report and rollback snapshot
- [ ] Acceptance: Failed migration restores prior manifest/lock state
- [ ] Acceptance: External PM runs under selected Node from 0060
- [ ] Fixture ready: `fixtures/pm/detect — identity from lockfiles and manifest`
- [ ] Fixture ready: `fixtures/pm/pin-invoke — pinned pnpm/npm/yarn/bun runs`
- [ ] Fixture ready: `fixtures/pm/migrate — adapter migration corpus`
- [ ] Fixture ready: `fixtures/pm/rollback — failed migration recovery`
- [ ] Fixture ready: `fixtures/pm/ambiguous — conflicting identity signals`


Required test layers:

- Unit tests for parsing, normalization, deterministic ordering, and error classification.
- Golden tests for manifests, lockfiles, command output, and migration reports.
- Integration tests against local fixture registries and isolated temporary homes.
- Failure-injection tests for network interruption, disk exhaustion, permission errors, process termination, and corrupted cache entries.
- Cross-platform tests for Linux, macOS, and Windows, including path length, case sensitivity, junctions, symlinks, and executable shims.
- Conformance tests comparing intentional compatibility surfaces with the corresponding Nub or package-manager behavior.

## Performance Requirements

- Add benchmarks for every newly introduced hot path.
- Avoid unbounded goroutines, file descriptors, memory growth, or registry requests.
- Publish baseline measurements in repository benchmark artifacts.

All performance claims must be backed by reproducible benchmark commands, machine metadata, cold/warm cache separation, and multiple samples. Performance regressions on critical paths require an explicit waiver.

## Security and Trust Requirements

- Validate all external input and fail closed on malformed or ambiguous data.
- Use least-privilege filesystem access and redact credentials in diagnostics.
- Maintain integrity verification before extraction or execution.

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
| PM meta-manager | Nub pm which/pin | Corepack replacement | 0061 |
| Manager detection | Nub | packageManager, lockfiles, config | 0061 |
| Verified acquisition | Nub | download/cache external PMs | 0061 |
| Migration planner | Nub pm migrate | lock adapter migrations | 0061 |

## Go Package Map

**Packages / paths:**

- `internal/pmmanager`
- `internal/compat`
- `internal/lockfile`
- `cmd/m (pm subcommand)`

**Forbidden import edges:**

- internal/runtime
- internal/transform

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  detect[PMDetect] --> acquire[VerifiedAcquire]
  acquire --> invoke[PMInvoke]
  invoke --> migrate[MigrationPlanner]
  migrate --> report[LossReport]
```

## Commands and Flags

| Command | Flags | Notes |
|---|---|---|
| `m pm which` | | Detect active package manager |
| `m pm pin pnpm@10` | pin | Pin manager version in project |
| `m pm migrate bun` | migrate | Plan/execute migration |
| `m pm update` | | Update pinned manager |
| `m pm cache list` | | Inspect PM cache |

Migrations produce semantic loss report and rollback snapshot.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| PM acquisition cache | Verified external manager installs |
| Migration rollback snapshot | Pre-migration state journal |

## Concrete Test Fixtures

- `fixtures/pm/detect — identity from lockfiles and manifest`
- `fixtures/pm/pin-invoke — pinned pnpm/npm/yarn/bun runs`
- `fixtures/pm/migrate — adapter migration corpus`
- `fixtures/pm/rollback — failed migration recovery`
- `fixtures/pm/ambiguous — conflicting identity signals`

## Acceptance Scenarios

1. m pm which reports correct manager for incumbent lockfiles
2. Pinned manager version used for invocation
3. Migration produces loss report and rollback snapshot
4. Failed migration restores prior manifest/lock state
5. External PM runs under selected Node from 0060

## Nub Conformance Targets

- Nub pm which/pin/migrate/update | parity
- Nub PM cache behavior | parity
- Incumbent lockfile preservation | Mew policy | extension

## Open Decisions

- Which external PM majors are certified in v1
- Auto-migrate on m install vs explicit m pm migrate

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
