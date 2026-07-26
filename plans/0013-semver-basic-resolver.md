# 0013 — Core MVP 4 — npm Semver and Basic Dependency Resolver

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 4 |
| Primary objective | Resolve registry dependencies and transitive dependencies using npm-compatible semver and produce a deterministic canonical graph with decision traces. |
| Required predecessors | 0012 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Resolve registry dependencies and transitive dependencies using npm-compatible semver and produce a deterministic canonical graph with decision traces.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0012 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Aube resolver
- Nub use of npm-semantic ranges rather than Cargo-semver semantics

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m resolve --plan
```
```bash
m explain resolution react
```

## In Scope

- Exact versions, tags, caret, tilde, x-ranges, hyphen ranges, unions, prereleases, and common aliases.
- Recursive dependencies and dev-dependency importer semantics.
- Cycle-safe graph traversal and request deduplication.
- Minimum release age and deprecation filters as recorded policy decisions.
- Decision traces for selected and rejected candidates.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Resolver is pure over registry metadata, manifest input, policy, and optional previous graph.
- Separate candidate generation, policy filtering, preference scoring, and graph expansion.
- Use stable package keys and sorted traversal.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Integrate npm-compatible semver range parsing and satisfaction
- [ ] Handle scoped package names and registry URL routing
- [ ] Bound recursion depth and fan-out with clear limits
- [ ] Fail closed on missing packument or unsatisfiable range

### Core logic

- [ ] Resolve direct dependencies from normalized manifest declarations
- [ ] Select highest matching version for ^ ~ * and exact ranges
- [ ] Add unit tests for semver edge cases: prerelease, build metadata
- [ ] Avoid any node_modules or lockfile mutation in this MVP

### CLI / UX

- [ ] Expand transitive dependencies from registry packuments recursively
- [ ] Emit structured decision trace for each version choice
- [ ] Add integration tests resolving fixture registry graphs

### Tests & fixtures

- [ ] Produce deterministic canonical graph with stable node ordering
- [ ] Support resolution from empty graph (greenfield)
- [ ] Add golden tests for deterministic graph encoding

### Docs & observability

- [ ] Detect and report dependency cycles with full path
- [ ] Support resolution from partial lock hints (prepare for 0015)
- [ ] Document resolver input/output interfaces for lockfile adapter

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Simple ^ range resolves to expected highest compatible version
- [ ] Acceptance: Transitive closure matches fixture registry graph
- [ ] Acceptance: Identical inputs produce byte-identical canonical graph
- [ ] Acceptance: Unsatisfiable range returns stable error with package name
- [ ] Acceptance: Cycle detection reports full cycle path
- [ ] Fixture ready: `fixtures/registry/v1/transitive-a-b-c/`
- [ ] Fixture ready: `fixtures/projects/semver-ranges/`
- [ ] Fixture ready: `testdata/resolver/golden/graphs/`
- [ ] Fixture ready: `testdata/resolver/cycles/cycle-a-b.json`


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
| Semver range resolution | npm semver | Registry version selection | 0013 |
| Transitive closure | Aube resolver | Deterministic graph build | 0013 |
| Decision trace | Nub explain | Resolver trace events | 0013, 0028 |
| Cycle detection | npm | Fail with actionable path | 0013 |

## Go Package Map

**Packages / paths:**

- `internal/resolver`
- `internal/registry`
- `internal/manifest`

**Forbidden import edges:**

- internal/linker
- internal/transaction

## Data Flow

```mermaid
flowchart LR
  man[manifest] --> reg[registry] --> res[internal/resolver] --> graph[CanonicalGraph] --> trace[DecisionTrace]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m resolve` | `--json`, `--trace` | Dry resolution without install |
| `m why <pkg>` | — | Stub pointing to 0028 if not full |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Canonical dependency graph | Resolver output model |
| Decision trace log | Explain input |

## Concrete Test Fixtures

- `fixtures/registry/v1/transitive-a-b-c/`
- `fixtures/projects/semver-ranges/`
- `testdata/resolver/golden/graphs/`
- `testdata/resolver/cycles/cycle-a-b.json`

## Acceptance Scenarios

1. Simple ^ range resolves to expected highest compatible version
2. Transitive closure matches fixture registry graph
3. Identical inputs produce byte-identical canonical graph
4. Unsatisfiable range returns stable error with package name
5. Cycle detection reports full cycle path

## Nub Conformance Targets

- npm semver range semantics | parity
- Transitive resolution ordering | parity
- Scoped dependency resolution | parity

## Open Decisions

- Prerelease inclusion policy for ^ and ~ ranges
- Maximum resolution depth and breadth limits

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
