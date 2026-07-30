# 0007 — Canonical Data Model and Core Interfaces

## Document Control

| Item | Detail |
|---|---|
| Phase | Foundation |
| Primary objective | Freeze canonical manifest, dependency graph, resolution, importer, package, policy, plan, snapshot, and lockfile models shared across the core. |
| Required predecessors | 0003, 0006 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Freeze canonical manifest, dependency graph, resolution, importer, package, policy, plan, snapshot, and lockfile models shared across the core.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0003 before starting this MVP.
- Complete and merge 0006 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Aube manifest, resolver, lockfile, linker, registry, workspace, and script models
- Nub lockfile compatibility layer

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

No new command is required in this MVP.

## In Scope

- Immutable canonical graph keyed by package identity plus peer context.
- Loss-report model for data that cannot be represented in a target lockfile.
- Decision trace model recording candidate filtering and version selection.
- Install plan model separating desired state, physical operations, and commit actions.
- Snapshot model for rollback and dependency time travel.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Keep source-format fields in adapter-owned extension maps, never in core algorithms.
- Use sorted slices or explicit deterministic encoders rather than Go map iteration.
- Version serialized internal caches independently from public lockfiles.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Freeze Manifest, Dependency, Importer, Package, Graph, Edge types
- [ ] Define ID schemes for packages and importers
- [ ] Specify explain/plan JSON shapes consumed by 0028
- [ ] Link types to owning packages

### Core logic

- [ ] Freeze ResolutionDecision and PeerContext types
- [ ] Define integrity and tarball URL fields
- [ ] Round-trip golden encoding tests
- [ ] Provide builders for tests while keeping production models immutable after validation

### CLI / UX

- [ ] Freeze Policy, Plan, Snapshot, Capsule descriptors
- [ ] Define migration-friendly version fields
- [ ] Ordering stability tests
- [ ] Version serialized internal caches independently from public lockfiles

### Tests & fixtures

- [ ] Define deterministic sort keys for all collections
- [ ] Keep source-format fields in adapter-owned extension maps
- [ ] Invalid graph rejection tests

### Docs & observability

- [ ] Define immutability rules for graph values
- [ ] Use sorted slices or explicit deterministic encoders rather than map iteration
- [ ] Publish data-model doc with diagrams

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: All later core MVPs can depend on these types without reaching into adapters
- [ ] Acceptance: Deterministic encoding byte-identical across platforms
- [ ] Acceptance: Version field present on every persistent model
- [ ] Acceptance: Peer-context identity collisions are detectable and rejected
- [ ] Fixture ready: `testdata/graph/simple-app.json`
- [ ] Fixture ready: `testdata/graph/peers.json`
- [ ] Fixture ready: `testdata/graph/workspace.json`
- [ ] Fixture ready: `testdata/graph/loss-report.json`


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
| Canonical graph model | Aube models | Go types | 0007 |
| Plan / snapshot models | Mew extension | Shared types | 0007, 0017, 0028 |
| Loss-report model | Lock adapters | Round-trip fidelity | 0007, 0023 |

## Go Package Map

**Packages / paths:**

- `internal/manifest`
- `internal/lockfile`
- `internal/resolver`
- `internal/policy`
- `internal/transaction`

**Forbidden import edges:**

- internal/fetch
- internal/linker
- internal/registry

## Data Flow

```mermaid
flowchart LR
  manifest[Manifest] --> graph[CanonicalGraph] --> lock[LockAdapters] --> plan[MutationPlan] --> tx[Transaction]
```

## Commands and Flags

N/A — data model freeze only.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Versioned Go types | Shared across core |
| Golden JSON/YAML encodings | Deterministic ordering tests |

## Concrete Test Fixtures

- `testdata/graph/simple-app.json`
- `testdata/graph/peers.json`
- `testdata/graph/workspace.json`
- `testdata/graph/loss-report.json`

## Acceptance Scenarios

1. All later core MVPs can depend on these types without reaching into adapters
2. Deterministic encoding byte-identical across platforms
3. Version field present on every persistent model
4. Peer-context identity collisions are detectable and rejected

## Nub Conformance Targets

- Aube canonical graph concepts | parity

## Open Decisions

- Exact package ID string format (link 0015)

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
