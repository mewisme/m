# 0045 — Runner MVP 6 — Unified Execution and Snapshot Environments

## Document Control

| Item | Detail |
|---|---|
| Phase | Runner / MVP 6 |
| Primary objective | Unify `m exec`, `mx`, historical snapshots, and capsules behind one environment builder and executable resolver. |
| Required predecessors | 0028, 0029, 0043, 0044 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Unify `m exec`, `mx`, historical snapshots, and capsules behind one environment builder and executable resolver.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0028 before starting this MVP.
- Complete and merge 0029 before starting this MVP.
- Complete and merge 0043 before starting this MVP.
- Complete and merge 0044 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub exec/dlx split and nubx desugaring

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
mx --snapshot ID eslint .
```
```bash
m exec --capsule capsule.mcap tool
```

## In Scope

- Environment sources: current project, temporary package set, snapshot, or capsule.
- Uniform PATH, bin resolution, policy, reporter, and process supervision.
- Cache and provenance visibility.
- Explicit no-network and immutable modes.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- One execution request model with source-specific environment providers.
- Never merge incompatible dependency graphs implicitly.
- Expose environment identity in diagnostics.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Define ExecutionRequest and PreparedEnvironment interfaces
- [ ] Add capsule environment provider with integrity verification
- [ ] Implement cleanup leases for ephemeral execution roots
- [ ] Document provider selection and failure semantics

### Core logic

- [ ] Define environment provider contract for each source type
- [ ] Unify PATH, bin resolution, policy, reporter, and supervision
- [ ] Never merge incompatible dependency graphs implicitly
- [ ] Benchmark environment preparation across providers

### CLI / UX

- [ ] Refactor m exec local path onto shared environment builder
- [ ] Expose environment identity in diagnostics and structured events
- [ ] Add behavior equivalence tests across all providers
- [ ] Freeze public interfaces for 0046 stabilization

### Tests & fixtures

- [ ] Refactor mx DLX path onto shared executable resolver
- [ ] Implement explicit no-network and immutable execution modes
- [ ] Add leak and cleanup stress tests for ephemeral roots

### Docs & observability

- [ ] Add snapshot environment provider using lock adapters
- [ ] Add environment inspection command with provenance output
- [ ] Test concurrent execution isolation between environments

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: m exec and mx produce equivalent supervision through shared layer
- [ ] Acceptance: Snapshot and capsule providers verify integrity before execution
- [ ] Acceptance: Incompatible graphs never merge without explicit user action
- [ ] Acceptance: Environment inspect shows identity, provenance, and cache state
- [ ] Acceptance: Ephemeral roots cleaned up on success and failure
- [ ] Fixture ready: `fixtures/unified-exec/project — current importer baseline`
- [ ] Fixture ready: `fixtures/unified-exec/dlx — ephemeral package set`
- [ ] Fixture ready: `fixtures/unified-exec/snapshot — historical lock replay`
- [ ] Fixture ready: `fixtures/unified-exec/capsule — capsule.mcap execution`
- [ ] Fixture ready: `fixtures/unified-exec/isolation — concurrent env leaks`


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
| Unified execution model | Nub exec/dlx split | Single environment builder | 0045 |
| Snapshot execution | Nub | mx --snapshot ID | 0045 |
| Capsule execution | Nub capsules | m exec --capsule | 0045 |
| Environment inspection | Nub | provenance and cache visibility | 0045 |

## Go Package Map

**Packages / paths:**

- `internal/runner`
- `internal/lockfile`
- `internal/transaction`
- `cmd/m`
- `cmd/mx`

**Forbidden import edges:**

- internal/transform
- internal/runtime/augment

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  req[ExecutionRequest] --> provider[EnvProvider]
  provider --> prepare[PreparedEnvironment]
  prepare --> resolve[ExecutableResolver]
  resolve --> sup[ProcessSupervisor]
```

## Commands and Flags

| Command | Flags | Notes |
|---|---|---|
| `mx --snapshot ID eslint .` | `--snapshot` | Historical lock snapshot |
| `m exec --capsule capsule.mcap tool` | `--capsule` | Immutable capsule environment |
| `m env inspect` | subcommand | Environment identity and provenance |

Sources: current project, temp package set, snapshot, capsule. No implicit graph merge.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Execution request schema | Versioned unified request model |
| Environment identity digest | Diagnostics and cache keys |

## Concrete Test Fixtures

- `fixtures/unified-exec/project — current importer baseline`
- `fixtures/unified-exec/dlx — ephemeral package set`
- `fixtures/unified-exec/snapshot — historical lock replay`
- `fixtures/unified-exec/capsule — capsule.mcap execution`
- `fixtures/unified-exec/isolation — concurrent env leaks`

## Acceptance Scenarios

1. m exec and mx produce equivalent supervision through shared layer
2. Snapshot and capsule providers verify integrity before execution
3. Incompatible graphs never merge without explicit user action
4. Environment inspect shows identity, provenance, and cache state
5. Ephemeral roots cleaned up on success and failure

## Nub Conformance Targets

- Nub exec/dlx environment split | parity via unified model
- Nub snapshot/capsule concepts | parity where adopted

## Open Decisions

- Capsule format version and signing requirements
- Whether env inspect is stable CLI or debug-only in v1

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
