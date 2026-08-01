# 0041 — Runner MVP 2 — Workspace Script Orchestration

## Document Control

| Item | Detail |
|---|---|
| Phase | Runner / MVP 2 |
| Primary objective | Run scripts across selected workspace packages with topology, concurrency control, failure policy, and structured output. |
| Required predecessors | 0022, 0040 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Run scripts across selected workspace packages with topology, concurrency control, failure policy, and structured output.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0022 before starting this MVP.
- Complete and merge 0040 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub recursive and filtered script runner
- Nub topological and streaming reporter behavior

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m -r run build
```
```bash
m --filter api... run test
```
```bash
m run lint --workspace-concurrency 4
```

## In Scope

- Workspace filter integration.
- Topological, reverse-topological, parallel, and sequential modes.
- Concurrency limits and resource-aware defaults.
- Bail, continue, resume, and changed-only behavior.
- Per-package output prefixes and summaries.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Use one scheduler over the workspace graph.
- Never deadlock on workspace cycles; diagnose and require an explicit cycle policy.
- Preserve child output and cancellation semantics.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Integrate workspace filter from 0022 into script runner dispatch
- [ ] Implement bail, continue, resume, and changed-only failure policies
- [ ] Add machine-readable task events for CI consumption
- [ ] Benchmark scheduler overhead on wide monorepos

### Core logic

- [ ] Implement task graph generation from workspace dependency graph
- [ ] Detect workspace cycles and fail with explicit cycle diagnostics
- [ ] Implement resume metadata for incremental workspace runs
- [ ] Document workspace runner flags and failure policy semantics

### CLI / UX

- [ ] Implement topological and reverse-topological scheduling modes
- [ ] Implement per-package output prefixes and summary aggregation
- [ ] Add synthetic DAG scheduling unit tests
- [ ] Ensure deterministic task ordering for identical inputs

### Tests & fixtures

- [ ] Implement parallel and sequential execution modes
- [ ] Preserve child cancellation and signal semantics from 0040
- [ ] Add cycle detection and failure propagation fixtures

### Docs & observability

- [ ] Implement concurrency limits with resource-aware defaults
- [ ] Implement readiness queue scheduler without deadlocks
- [ ] Stress-test large workspace output and cancellation

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: m -r run build executes packages in correct topological order
- [ ] Acceptance: Concurrency limit respected; no unbounded goroutine fan-out
- [ ] Acceptance: Workspace cycles diagnosed without deadlock
- [ ] Acceptance: Per-package output remains attributable under parallel execution
- [ ] Acceptance: Failure policies behave deterministically across platforms
- [ ] Fixture ready: `fixtures/workspace-runner/dag-simple — topological ordering`
- [ ] Fixture ready: `fixtures/workspace-runner/cycle — cycle detection diagnostics`
- [ ] Fixture ready: `fixtures/workspace-runner/large — output multiplex stress`
- [ ] Fixture ready: `fixtures/workspace-runner/failure-bail — bail vs continue matrix`
- [ ] Fixture ready: `fixtures/workspace-runner/changed-only — resume metadata`


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
| Workspace script orchestration | Nub recursive run | Topology + concurrency | 0041 |
| Workspace filters | Nub --filter | Package selection integration | 0041 |
| Failure policies | Nub bail/continue | bail, continue, resume, changed-only | 0041 |
| Output multiplexing | Nub streaming reporter | Per-package prefixes + summaries | 0041 |

## Go Package Map

**Packages / paths:**

- `internal/runner`
- `internal/workspace`
- `internal/process`
- `cmd/m (-r run, --filter)`

**Forbidden import edges:**

- internal/runtime
- internal/transform
- internal/linker

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  filter[WorkspaceFilter] --> graph[TaskGraph]
  graph --> sched[Scheduler]
  sched --> run[ScriptRunner]
  run --> mux[OutputMux]
  mux --> report[Reporter]
```

## Commands and Flags

| Command | Flags | Notes |
|---|---|---|
| `m -r run build` | `-r`, `--recursive` | Run across workspace packages |
| `m --filter api... run test` | `--filter` | Filter graph before scheduling |
| `m run lint --workspace-concurrency 4` | `--workspace-concurrency` | Bounded parallelism |

Policies: topological, reverse-topological, parallel, sequential, bail, continue, resume.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Workspace task event schema | Machine-readable scheduler events |
| Resume metadata store (optional) | Changed-only and resume state |

## Concrete Test Fixtures

- `fixtures/workspace-runner/dag-simple — topological ordering`
- `fixtures/workspace-runner/cycle — cycle detection diagnostics`
- `fixtures/workspace-runner/large — output multiplex stress`
- `fixtures/workspace-runner/failure-bail — bail vs continue matrix`
- `fixtures/workspace-runner/changed-only — resume metadata`

## Acceptance Scenarios

1. m -r run build executes packages in correct topological order
2. Concurrency limit respected; no unbounded goroutine fan-out
3. Workspace cycles diagnosed without deadlock
4. Per-package output remains attributable under parallel execution
5. Failure policies behave deterministically across platforms

## Nub Conformance Targets

- Nub recursive/filtered script runner | parity
- Nub topological scheduling | parity
- Nub streaming reporter behavior | parity

## Open Decisions

- Default workspace-concurrency heuristic for CI vs interactive
- Whether changed-only requires VCS integration in v1
- **Changed-only and resume (deferred)**: Changed-only selection and resume metadata are deferred to a follow-up plan (candidate: 0041-b). Rationale: these features require a versioned metadata schema, deterministic baseline identity, hermetic VCS-free fixture design, and stale/incompatible metadata rejection — each a non-trivial design surface. The current workspace runner ships with topological scheduling, concurrency control, bail/continue, and deterministic ordering. Decision date: 2026-08-02. Owner: stabilization-0040-0051.

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
