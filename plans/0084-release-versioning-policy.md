# 0084 — Cross-Cutting — Versioning, Formats, and Support Policy

## Document Control

| Item | Detail |
|---|---|
| Phase | Cross-Cutting |
| Primary objective | Define compatibility promises for CLI, APIs, lockfiles, caches, plans, capsules, plugins, Node versions, and package-manager versions. |
| Required predecessors | 0009 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Define compatibility promises for CLI, APIs, lockfiles, caches, plans, capsules, plugins, Node versions, and package-manager versions.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0009 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub evolving v0 behavior and per-manager-major compatibility approach

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

No new command is required in this MVP.

## In Scope

- Semantic versioning policy.
- Experimental feature gates.
- Public versus internal formats.
- Reader/writer support windows.
- Cache invalidation and migration.
- Deprecation process and error-code stability.
- Supported Node, OS, architecture, and manager versions.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Readers outlive writers where practical.
- Public persistent formats require documented migration and rollback.
- Dropping a compatibility target requires evidence, notice, and explicit release notes.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Define semver promises for CLI
- [ ] Node version support windows
- [ ] Windows/macOS/Linux support statement
- [ ] Compatibility matrix published

### Core logic

- [ ] Define stability for Go public packages if any
- [ ] Package-manager adapter support windows
- [ ] Document what is not covered
- [ ] Align with 0087 DoD

### CLI / UX

- [ ] Lockfile version bump rules
- [ ] Experimental feature graduation rules
- [ ] Link to 0009 release channels
- [ ] Review cadence

### Tests & fixtures

- [ ] Cache version invalidation rules
- [ ] Breaking change communication process
- [ ] Upgrade/rollback testing requirements

### Docs & observability

- [ ] Plan/capsule/plugin compatibility
- [ ] Deprecation timeline minimums
- [ ] Agent must note format version impacts

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Every persistent format has version policy
- [ ] Acceptance: Node floor documented
- [ ] Acceptance: Experimental graduation criteria clear
- [ ] Fixture ready: `docs/versioning.md`
- [ ] Fixture ready: `docs/support-matrix.md`


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
| Versioning policy | Product | CLI/APIs/formats/Node/PMs | 0084 |
| Support windows | Product | Lock adapters + Node | 0084 |

## Go Package Map

**Packages / paths:**

- `docs/versioning.md`
- `docs/support-policy.md`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  change[ChangeType] --> class[CompatClass] --> version[VersionBump] --> notes[ReleaseNotes]
```

## Commands and Flags

Documents how `m --version` and lockfileVersion evolve.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Support matrix | Node + PM |
| Format version table | lock/cache/plan/capsule |

## Concrete Test Fixtures

- `docs/versioning.md`
- `docs/support-matrix.md`

## Acceptance Scenarios

1. Every persistent format has version policy
2. Node floor documented
3. Experimental graduation criteria clear

## Nub Conformance Targets

- Support policy | extension

## Open Decisions

- 0.x duration before 1.0

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
