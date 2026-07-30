# 0015 — Core MVP 6 — Native `m.lock` Format

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 6 |
| Primary objective | Design and implement Mew’s deterministic native lockfile with complete graph, importer, policy, integrity, peer-context, and compatibility metadata. |
| Required predecessors | 0007, 0013 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Design and implement Mew’s deterministic native lockfile with complete graph, importer, policy, integrity, peer-context, and compatibility metadata.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0007 before starting this MVP.
- Complete and merge 0013 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub native lock identity concepts
- Aube lockfile graph representation

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m lock validate
```
```bash
m lock format
```
```bash
m install --frozen-lockfile
```

## In Scope

- Versioned `m.lock` schema.
- Importers and workspace package records.
- Resolved package snapshots, integrity, tarball, dependency edges, peers, platform constraints, build policy, patches, and source type.
- Settings checksum and manifest specifiers.
- Forward-compatible unknown-field handling policy.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Choose a deterministic human-reviewable syntax through ADR; YAML is acceptable only with a controlled encoder.
- Reader must tolerate supported older versions; writer emits one canonical current version.
- Semantic hash excludes presentation-only data.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Define m.lock schema version and top-level document structure
- [ ] Implement lockfile read parser with forward-compatible unknown field handling
- [ ] Add migration stub for future schema bumps
- [ ] Integrate with internal/resolver graph model from 0007

### Core logic

- [ ] Serialize canonical graph to importer-aware lock sections
- [ ] Implement lockfile write from resolver output without data loss
- [ ] Document m.lock field reference for adapter MVPs
- [ ] Add fuzz smoke tests for parser robustness

### CLI / UX

- [ ] Record package identity, resolution, integrity, and dependency edges
- [ ] Validate lockfile against manifest on --frozen-lockfile
- [ ] Support peer-context placeholders for 0020 without full peer resolution

### Tests & fixtures

- [ ] Include settings block for linker mode and resolver policy placeholders
- [ ] Detect lockfile/manifest drift with actionable diff summary
- [ ] Never embed secrets or auth tokens in lockfile

### Docs & observability

- [ ] Implement deterministic key ordering and stable encoding
- [ ] Add golden tests for round-trip encode/decode
- [ ] Reject ambiguous duplicate package entries

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Resolver graph round-trips through m.lock losslessly
- [ ] Acceptance: Generated m.lock is byte-identical across platforms for same input
- [ ] Acceptance: Frozen lockfile mode fails when manifest changes
- [ ] Acceptance: Corrupt lockfile returns stable parse error code
- [ ] Acceptance: Schema version field present on every document
- [ ] Fixture ready: `testdata/lockfile/mlock/golden/basic/`
- [ ] Fixture ready: `testdata/lockfile/mlock/golden/workspace/`
- [ ] Fixture ready: `fixtures/projects/mlock-greenfield/`
- [ ] Fixture ready: `testdata/lockfile/mlock/corrupt/`


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
| m.lock format | Mew native | Versioned deterministic graph | 0015 |
| Importer sections | pnpm importers | Per-package manifest snapshot | 0015 |
| Integrity records | npm lock integrity | Per-package hash | 0015 |
| Settings block | Nub lock settings | Linker/resolver policy | 0015 |

## Go Package Map

**Packages / paths:**

- `internal/lockfile`
- `internal/lockfile/mlock`
- `internal/resolver`

**Forbidden import edges:**

- internal/linker
- internal/transaction

## Data Flow

```mermaid
flowchart LR
  graph[CanonicalGraph] --> mlock[internal/lockfile/mlock] --> disk[m.lock] --> read[LockfileRead]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m lock` | `--frozen`, `--fix` | Generate or validate m.lock |
| `m install --frozen-lockfile` | — | Prepare for 0016 |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| `m.lock` | Native deterministic lockfile |
| Lockfile schema version field | Migration anchor |

## Concrete Test Fixtures

- `testdata/lockfile/mlock/golden/basic/`
- `testdata/lockfile/mlock/golden/workspace/`
- `fixtures/projects/mlock-greenfield/`
- `testdata/lockfile/mlock/corrupt/`

## Acceptance Scenarios

1. Resolver graph round-trips through m.lock losslessly
2. Generated m.lock is byte-identical across platforms for same input
3. Frozen lockfile mode fails when manifest changes
4. Corrupt lockfile returns stable parse error code
5. Schema version field present on every document

## Nub Conformance Targets

- pnpm lock importer concepts | parity
- npm lock integrity field semantics | parity
- Deterministic lockfile ordering | parity

## Open Decisions

- Exact m.lock file name and location (root only vs per-importer)
- Whether to embed full packument metadata or minimal dist fields

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
