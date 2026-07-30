# 0029 — Core MVP 20 — Performance, Offline Operation, and Portable Capsules

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 20 |
| Primary objective | Optimize cold and warm installs, make offline behavior first-class, and package reproducible dependency environments for CI and containers. |
| Required predecessors | 0018, 0026, 0028 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Optimize cold and warm installs, make offline behavior first-class, and package reproducible dependency environments for CI and containers.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0018 before starting this MVP.
- Complete and merge 0026 before starting this MVP.
- Complete and merge 0028 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub/Aube cache and install performance behavior
- Mew smart planner and capsule extensions

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m install --offline
```
```bash
m capsule create
```
```bash
m capsule restore
```
```bash
m benchmark install
```

## In Scope

- Adaptive network, extraction, hashing, linking, and script concurrency.
- Resource-limit and cgroup-aware worker budgets.
- Complete offline resolution from lockfile and cache.
- Export/import of graph metadata and optional content blobs.
- Capsule identity for Node version, platform, graph, policy, and build outputs.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Prefer bounded pipelines with backpressure over phase-wide unbounded fan-out.
- Capsules are content-addressed, versioned, verifiable, and may be thin or self-contained.
- Offline failures identify exact missing objects.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Profile install phases: resolve, fetch, extract, link, lifecycle
- [ ] Implement m capsule restore for CI/container bootstrap
- [ ] Reduce allocator churn in resolver and linker
- [ ] Add soak test script for repeated install cycles

### Core logic

- [ ] Optimize hot paths identified by profiling
- [ ] Add benchmark harness m bench install with cold/warm modes
- [ ] Implement metadata batch fetch where registry supports
- [ ] Document performance tuning env vars

### CLI / UX

- [ ] Implement warm-cache fast path skipping redundant metadata fetches
- [ ] Publish baseline benchmark artifacts in repo
- [ ] Document offline workflow for air-gapped environments

### Tests & fixtures

- [ ] Make --offline first-class: preflight cache completeness check
- [ ] Add CI regression gate on critical path benchmarks
- [ ] Capsule integrity verification on restore

### Docs & observability

- [ ] Implement m capsule create bundling store + lock + metadata
- [ ] Tune worker pool defaults per CPU count
- [ ] Never sacrifice integrity for performance

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Warm install measurably faster than cold on benchmark fixture
- [ ] Acceptance: Offline install succeeds when capsule/cache complete
- [ ] Acceptance: Capsule round-trip produces identical node_modules hash
- [ ] Acceptance: Benchmark CI gate fails on >10% regression without waiver
- [ ] Acceptance: Phase timing diagnostics available via --debug
- [ ] Fixture ready: `fixtures/bench/medium-graph`
- [ ] Fixture ready: `fixtures/capsules/basic-export`
- [ ] Fixture ready: `testdata/offline/full-cache-project`


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
| Install performance | Nub benchmarks | Cold/warm cache paths | 0029 |
| Offline-first mode | Nub --offline | Metadata + tarball cache | 0029 |
| CI capsules | Nub capsules | Reproducible dep environments | 0029 |
| Benchmark suite | 0081 program | Published baselines | 0029 |

## Go Package Map

**Packages / paths:**

- `internal/fetch`
- `internal/store`
- `internal/registry`
- `internal/archive`
- `internal/app`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  cache[warm cache] --> fast[fast path] --> bench[benchmarks] --> capsule[capsule export] --> offline[offline install]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m install --offline` | — | No network required |
| `m capsule create` | `--output` | Export dep environment |
| `m capsule restore` | — | Import capsule |
| `m bench install` | — | Dev benchmark harness |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Capsule archive | Portable dep snapshot |
| Benchmark results JSON | CI performance regression gate |

## Concrete Test Fixtures

- `fixtures/bench/medium-graph`
- `fixtures/capsules/basic-export`
- `testdata/offline/full-cache-project`

## Acceptance Scenarios

1. Warm install measurably faster than cold on benchmark fixture
2. Offline install succeeds when capsule/cache complete
3. Capsule round-trip produces identical node_modules hash
4. Benchmark CI gate fails on >10% regression without waiver
5. Phase timing diagnostics available via --debug

## Nub Conformance Targets

- Nub install performance profile | parity
- Offline install behavior | parity
- Capsule reproducibility | extension

## Open Decisions

- Capsule format versioning and compression algorithm
- Benchmark machine pinning strategy in CI

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
