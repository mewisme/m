# 0090 — Future Extensions Beyond Nub Parity

## Document Control

| Item | Detail |
|---|---|
| Phase | Future |
| Primary objective | Capture valuable post-parity ideas without allowing them to expand the ordered implementation critical path. |
| Required predecessors | 0087 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Capture valuable post-parity ideas without allowing them to expand the ordered implementation critical path.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0087 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Mew extension ideas discussed during planning

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

No new command is required in this MVP.

## In Scope

- Remote shared content store.
- Daemon mode only if measurements justify it.
- Organization policy service.
- Package graph query language.
- IDE/LSP integration.
- Signed plan approval workflows.
- Deterministic native-addon build farms.
- Reproducibility attestations.
- Dependency update automation.
- Interactive dependency graph UI.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- No future item enters implementation without a new indexed MVP, product case, security review, and dependency analysis.
- Avoid daemon or cloud architecture unless local single-binary behavior remains first-class.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] OPTIONAL: Capture post-parity ideas without expanding critical path
- [ ] OPTIONAL: Periodic backlog grooming
- [ ] OPTIONAL: Document promotion checklist

### Core logic

- [ ] OPTIONAL: Tag each idea with value/risk/effort
- [ ] OPTIONAL: Link rejected ideas with rationale
- [ ] OPTIONAL: Examples only — phantom dep analysis, extra templates

### CLI / UX

- [ ] OPTIONAL: Require charter update before promotion
- [ ] OPTIONAL: Keep out of stabilization gates
- [ ] OPTIONAL: Ensure INDEX marks 0090 as future

### Tests & fixtures

- [ ] OPTIONAL: Separate nice-to-have UX polish ideas
- [ ] OPTIONAL: Do not assign primary_mvp that blocks 0031/0046/0057
- [ ] OPTIONAL: No conformance dependency

### Docs & observability

- [ ] OPTIONAL: Note ideas that conflict with stock-Node boundary
- [ ] OPTIONAL: Agent must not implement backlog as if required
- [ ] OPTIONAL: Review during release planning only

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Backlog explicitly non-blocking
- [ ] Acceptance: No critical-path MVP lists 0090 as required predecessor
- [ ] Acceptance: Promotion requires human decision
- [ ] Fixture ready: `docs/backlog/future.md`


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
| Future backlog | Post-parity ideas | Non-blocking | 0090 |

## Go Package Map

**Packages / paths:**

- `docs/backlog/future.md`

**Forbidden import edges:**

- Must not pull backlog items onto critical path without charter change

## Data Flow

```mermaid
flowchart LR
  idea[Idea] --> park[Backlog] --> review[PeriodicReview] --> promote[OptionalPromote]
```

## Commands and Flags

N/A — non-blocking backlog. No ship gate depends on 0090.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Future backlog doc | Parking lot |

## Concrete Test Fixtures

- `docs/backlog/future.md`

## Acceptance Scenarios

1. Backlog explicitly non-blocking
2. No critical-path MVP lists 0090 as required predecessor
3. Promotion requires human decision

## Nub Conformance Targets

- Out of parity scope | deferred

## Open Decisions

- Which ideas if any to promote after GA

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
