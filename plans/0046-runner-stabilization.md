# 0046 — Runner Stabilization Gate

## Document Control

| Item | Detail |
|---|---|
| Phase | Runner / Stabilization |
| Primary objective | Certify script and executable execution for interactive development, CI, workspaces, PnP, and cross-platform signal behavior. |
| Required predecessors | 0040, 0041, 0042, 0043, 0044, 0045 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Certify script and executable execution for interactive development, CI, workspaces, PnP, and cross-platform signal behavior.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0040 before starting this MVP.
- Complete and merge 0041 before starting this MVP.
- Complete and merge 0042 before starting this MVP.
- Complete and merge 0043 before starting this MVP.
- Complete and merge 0044 before starting this MVP.
- Complete and merge 0045 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Complete Nub runner behavior plus Mew direct-script extensions

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m conformance run runner
```
```bash
m benchmark runner
```

## In Scope

- Real-world script corpus.
- Shell and process semantics.
- Workspace scheduling soak.
- `mx` security and cache review.
- Documentation and completion.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Do not begin stable runtime augmentation until runner process supervision is reliable.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Assemble real-world script corpus across npm/pnpm/Yarn/Bun layouts
- [ ] Publish runner compatibility and divergence matrix
- [ ] Run multi-day process leak soak on supervisor
- [ ] Benchmark runner hot paths against published baselines

### Core logic

- [ ] Run cross-shell quoting and process semantics corpus on all platforms
- [ ] Verify no signal, exit-code, stdin, or output corruption bugs remain
- [ ] Run interactive TTY smoke on supported terminals
- [ ] Sign off interfaces consumed by runtime MVPs

### CLI / UX

- [ ] Soak long-lived processes and watch-like cancellation scenarios
- [ ] Fully test direct script shortcut collision behavior
- [ ] Run CI noninteractive behavior regression suite
- [ ] Update feature inventory statuses to shipped where certified

### Tests & fixtures

- [ ] Review executable trust UX for mx consent and policy surfaces
- [ ] Certify mx never fetches implicitly in non-TTY without consent
- [ ] Document known limitations and waivers with owners

### Docs & observability

- [ ] Freeze runner event schema with version field
- [ ] Verify workspace scheduler determinism and resource bounds
- [ ] Integrate runner conformance into CI stop-the-line gates

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: No known signal, exit-code, stdin, or output corruption bug
- [ ] Acceptance: Direct script shortcut collisions fully tested and documented
- [ ] Acceptance: mx never fetches implicitly in non-TTY without explicit consent
- [ ] Acceptance: Workspace scheduler deterministic and resource bounded
- [ ] Acceptance: Runner conformance suite passes on Linux, macOS, Windows
- [ ] Fixture ready: `tests/conformance/runner/corpus — real-world scripts`
- [ ] Fixture ready: `tests/conformance/runner/shell-matrix — cross-shell semantics`
- [ ] Fixture ready: `tests/conformance/runner/workspace-soak — scheduler stress`
- [ ] Fixture ready: `tests/conformance/runner/mx-security — consent/cache review`
- [ ] Fixture ready: `tests/conformance/runner/TTY-smoke — interactive terminals`


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

- [ ] No known signal, exit-code, stdin, or output corruption bug.
- [ ] Direct script shortcut collision behavior is fully tested.
- [ ] `mx` never fetches implicitly in non-TTY without explicit consent.
- [ ] Workspace scheduler is deterministic and resource bounded.



<!-- ENRICHMENT:BEGIN -->

## Feature Inventory Links

Rows this MVP owns or primarily advances (from `0002` inventory themes):

| Feature | Nub baseline | Mew target | Primary MVP |
|---|---|---|---|
| Runner stabilization gate | Nub runner surface | Certify 0040-0045 | 0046 |
| Conformance harness | Nub | m conformance run runner | 0046 |
| Divergence matrix | Charter | Direct script extension documented | 0046 |
| mx security review | Nubx | consent and cache certification | 0046 |

## Go Package Map

**Packages / paths:**

- `internal/runner`
- `internal/process`
- `cmd/m`
- `cmd/mx`
- `tests/conformance/runner`

**Forbidden import edges:**

- internal/resolver (new features)
- internal/runtime

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  corpus[RealWorldCorpus] --> run[RunnerStack]
  run --> compare[NubCompare]
  compare --> matrix[CompatibilityMatrix]
  matrix --> gate[StabilizationGate]
```

## Commands and Flags

| Command | Flags | Notes |
|---|---|---|
| `m conformance run runner` | suite selector | Full runner conformance |
| `m benchmark runner` | `--cold`, `--warm` | Published baselines |

Gate blocks stable runtime work (0050+) until exit criteria met.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Runner compatibility matrix | Parity/divergence/extension per feature |
| Frozen runner event schema | Stable machine-readable events |
| Benchmark baseline artifacts | CI regression tracking |

## Concrete Test Fixtures

- `tests/conformance/runner/corpus — real-world scripts`
- `tests/conformance/runner/shell-matrix — cross-shell semantics`
- `tests/conformance/runner/workspace-soak — scheduler stress`
- `tests/conformance/runner/mx-security — consent/cache review`
- `tests/conformance/runner/TTY-smoke — interactive terminals`

## Acceptance Scenarios

1. No known signal, exit-code, stdin, or output corruption bug
2. Direct script shortcut collisions fully tested and documented
3. mx never fetches implicitly in non-TTY without explicit consent
4. Workspace scheduler deterministic and resource bounded
5. Runner conformance suite passes on Linux, macOS, Windows

## Nub Conformance Targets

- Complete Nub runner behavior | parity except documented extensions
- Mew direct m <script> | extension | certified
- Nubx DLX security model | parity

## Open Decisions

- Waivers for known cross-platform shell differences
- Date to remove experimental gate on direct script shortcuts

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
