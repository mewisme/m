# 0031 — Core MVP 22 — Package-Manager Core Stabilization Gate

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / Stabilization |
| Primary objective | Certify the package-manager core for daily use before beginning runner and runtime parity work. |
| Required predecessors | 0010, 0011, 0012, 0013, 0014, 0015, 0016, 0017, 0018, 0019, 0020, 0021, 0022, 0023, 0024, 0025, 0026, 0027, 0028, 0029, 0030 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Certify the package-manager core for daily use before beginning runner and runtime parity work.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0010 before starting this MVP.
- Complete and merge 0011 before starting this MVP.
- Complete and merge 0012 before starting this MVP.
- Complete and merge 0013 before starting this MVP.
- Complete and merge 0014 before starting this MVP.
- Complete and merge 0015 before starting this MVP.
- Complete and merge 0016 before starting this MVP.
- Complete and merge 0017 before starting this MVP.
- Complete and merge 0018 before starting this MVP.
- Complete and merge 0019 before starting this MVP.
- Complete and merge 0020 before starting this MVP.
- Complete and merge 0021 before starting this MVP.
- Complete and merge 0022 before starting this MVP.
- Complete and merge 0023 before starting this MVP.
- Complete and merge 0024 before starting this MVP.
- Complete and merge 0025 before starting this MVP.
- Complete and merge 0026 before starting this MVP.
- Complete and merge 0027 before starting this MVP.
- Complete and merge 0028 before starting this MVP.
- Complete and merge 0029 before starting this MVP.
- Complete and merge 0030 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Complete Nub package-manager surface and supported lockfile formats

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m doctor
```
```bash
m conformance run core
```
```bash
m benchmark install
```

## In Scope

- Feature freeze and bug burn-down.
- Real-world project corpus.
- Cross-PM lockfile certification.
- Crash-safety campaign.
- Performance, memory, file-descriptor, and disk-use budgets.
- Documentation, migration, and recovery playbooks.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- No runner or runtime scope may bypass unresolved core corruption or compatibility issues.
- Stable release requires explicit support matrix and known-limitations document.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Run full conformance suite against Nub/npm/pnpm fixtures per 0080
- [ ] Implement m doctor health check for common misconfigurations
- [ ] Security audit of threat model 0082 items for PM core
- [ ] Sign-off checklist per 0087 definition of done

### Core logic

- [ ] Execute cross-platform integration matrix on Linux/macOS/Windows CI
- [ ] Publish core-certification.md with evidence links
- [ ] Documentation pass for all PM commands shipped
- [ ] Unblock 0040 runner MVP with stable install interfaces

### CLI / UX

- [ ] Run soak tests: 100+ install cycles on representative projects
- [ ] Freeze public CLI and m.lock schema for runner MVPs
- [ ] Verify transaction recovery on all supported platforms

### Tests & fixtures

- [ ] Fix all P0/P1 defects found during stabilization
- [ ] Review and close open decisions from 0010-0030
- [ ] Verify all lock adapters on certified fixture corpus

### Docs & observability

- [ ] Verify no critical data-loss or corruption paths remain
- [ ] Benchmark baselines recorded and regression gates green
- [ ] No new features: stabilization and fixes only

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Full core conformance suite passes on all CI platforms
- [ ] Acceptance: m doctor reports healthy state on clean fixture project
- [ ] Acceptance: No open P0/P1 defects in PM core scope
- [ ] Acceptance: core-certification.md published with test evidence
- [ ] Acceptance: 0040 can depend on install/layout interfaces without breakage
- [ ] Fixture ready: `tests/conformance/core-matrix/`
- [ ] Fixture ready: `fixtures/soak/representative-projects/`
- [ ] Fixture ready: `testdata/certification/sign-off-checklist.md`


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

- [ ] Zero known data-loss or silent-integrity issue.
- [ ] Certified read/write matrices are accurate and enforced by tests.
- [ ] Transactional recovery succeeds for every injected commit interruption.
- [ ] Core commands are documented and machine-readable output is versioned.
- [ ] Performance and resource budgets are enforced in CI.







<!-- ENRICHMENT:BEGIN -->

## Feature Inventory Links

Rows this MVP owns or primarily advances (from `0002` inventory themes):

| Feature | Nub baseline | Mew target | Primary MVP |
|---|---|---|---|
| Core certification gate | 0087 DoD | PM core daily-driver ready | 0031 |
| Conformance matrix | 0080 program | Nub/incumbent parity evidence | 0031 |
| Cross-platform soak | 0008 strategy | Linux/macOS/Windows | 0031 |
| Stabilization fixes only | Release policy | No new features | 0031 |

## Go Package Map

**Packages / paths:**

- `internal/app`
- `tests/conformance`
- `tests/integration`
- `docs/core-certification.md`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  mvps[0010-0030] --> conformance[conformance suite] --> soak[soak tests] --> cert[core certification] --> gate[runner MVPs unblocked]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `make core-cert` | — | Full certification command |
| `m doctor` | — | Environment and install health check |
| CI | core-stabilization workflow | Required green gate |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| docs/core-certification.md | Certification checklist and evidence |
| Conformance report | Pass/fail matrix per target |

## Concrete Test Fixtures

- `tests/conformance/core-matrix/`
- `fixtures/soak/representative-projects/`
- `testdata/certification/sign-off-checklist.md`

## Acceptance Scenarios

1. Full core conformance suite passes on all CI platforms
2. m doctor reports healthy state on clean fixture project
3. No open P0/P1 defects in PM core scope
4. core-certification.md published with test evidence
5. 0040 can depend on install/layout interfaces without breakage

## Nub Conformance Targets

- Nub PM core behavioral parity | parity
- 0087 definition of done | parity
- Cross-platform install reliability | parity

## Open Decisions

- Date for core v1 beta channel promotion
- Which Yarn/Berry features remain explicitly deferred post-0031

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
