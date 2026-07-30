# 0023 — Core MVP 14 — Nub and pnpm Lockfile Bridge

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 14 |
| Primary objective | Read, preserve, write, validate, diff, and explicitly migrate Nub and supported pnpm lockfile generations through the canonical graph. |
| Required predecessors | 0015, 0020, 0022 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Read, preserve, write, validate, diff, and explicitly migrate Nub and supported pnpm lockfile generations through the canonical graph.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0015 before starting this MVP.
- Complete and merge 0020 before starting this MVP.
- Complete and merge 0022 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub lockfile round-trip behavior
- Aube pnpm lock parser/writer
- Specific pnpm major-version compatibility models

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m lock validate
```
```bash
m lock diff
```
```bash
m lock migrate --from nub --to m
```
```bash
m install --lockfile-format preserve
```

## In Scope

- `nub.lock` first-class compatibility.
- pnpm lockfile versions used by supported pnpm majors.
- Importer, snapshot, peer suffix, settings, overrides, catalog, patch, and integrity semantics.
- Semantic diff and representability/loss reports.
- Preserve existing format unless migration is explicit.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Implement adapters as parse-to-canonical and canonical-to-format layers.
- Retain unrecognized but safely preservable data in adapter extensions.
- Validate emitted lockfiles by running pinned target package managers in conformance CI.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Detect nub.lock and pnpm-lock.yaml per identity rules from 0006
- [ ] Write round-trip safe pnpm-lock when identity is pnpm
- [ ] Add diff tool comparing canonical graph from two lock sources
- [ ] Fuzz parser smoke on lockfile corpora

### Core logic

- [ ] Implement nub.lock reader adapter to canonical graph
- [ ] Implement m migrate lock --to m.lock with dry-run report
- [ ] Support peer/importer metadata required for isolated layout
- [ ] Record adapter version in migration report

### CLI / UX

- [ ] Implement pnpm-lock.yaml reader for supported major generations
- [ ] Document lossy conversions explicitly in migration output
- [ ] Never overwrite incumbent lock without explicit migrate

### Tests & fixtures

- [ ] Preserve incumbent lockfile on install without user migrate
- [ ] Validate adapter output against resolver for drift detection
- [ ] Handle lockfile version unsupported with upgrade guidance

### Docs & observability

- [ ] Write round-trip safe nub.lock when project identity is Nub
- [ ] Add golden tests per lockfile generation fixture
- [ ] Integrate with transaction commit for lock writes

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Install on nub.lock project preserves nub.lock format
- [ ] Acceptance: pnpm-lock.yaml project installs without silent m.lock conversion
- [ ] Acceptance: m migrate lock --dry-run lists lossy fields
- [ ] Acceptance: Adapter round-trip nub.lock golden matches source
- [ ] Acceptance: Unsupported lock version returns actionable error
- [ ] Fixture ready: `fixtures/locks/nub/v1-basic`
- [ ] Fixture ready: `fixtures/locks/pnpm/v6`
- [ ] Fixture ready: `fixtures/locks/pnpm/v9`
- [ ] Fixture ready: `testdata/lockfile/migrate-reports/`


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
| nub.lock read | Nub native | Import to canonical graph | 0023 |
| pnpm-lock.yaml read | pnpm | Supported generations | 0023 |
| Lock preserve on detect | Mew policy | No silent conversion | 0023 |
| Explicit migrate | Nub migrate | nub/pnpm to m.lock | 0023 |

## Go Package Map

**Packages / paths:**

- `internal/lockfile`
- `internal/compat/nub`
- `internal/compat/pnpm`
- `internal/resolver`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  nublock[nub.lock/pnpm-lock] --> adapter[compat adapter] --> graph[CanonicalGraph] --> mlock[m.lock optional]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m install` | — | Uses detected lockfile identity |
| `m migrate lock` | `--to m.lock`, `--dry-run` | Explicit conversion |
| `m lock --diff` | — | Compare adapter output |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Preserved nub.lock / pnpm-lock.yaml | Unchanged unless migrate |
| Migration report | Lossy field documentation |

## Concrete Test Fixtures

- `fixtures/locks/nub/v1-basic`
- `fixtures/locks/pnpm/v6`
- `fixtures/locks/pnpm/v9`
- `testdata/lockfile/migrate-reports/`

## Acceptance Scenarios

1. Install on nub.lock project preserves nub.lock format
2. pnpm-lock.yaml project installs without silent m.lock conversion
3. m migrate lock --dry-run lists lossy fields
4. Adapter round-trip nub.lock golden matches source
5. Unsupported lock version returns actionable error

## Nub Conformance Targets

- nub.lock semantics | parity
- pnpm-lock.yaml supported generations | parity
- Lockfile preservation policy | extension

## Open Decisions

- Which pnpm lock major versions are certified in v1
- Whether m can write pnpm-lock.yaml or read-only initially

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
