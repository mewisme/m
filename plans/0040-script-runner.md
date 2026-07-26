# 0040 — Runner MVP 1 — Package Script Runner

## Document Control

| Item | Detail |
|---|---|
| Phase | Runner / MVP 1 |
| Primary objective | Implement `m run` with npm-compatible environment construction, lifecycle hooks, argument forwarding, signal propagation, and deterministic output. |
| Required predecessors | 0031 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Implement `m run` with npm-compatible environment construction, lifecycle hooks, argument forwarding, signal propagation, and deterministic output.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0031 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub script runner, environment handling, reporters, and process supervision
- Nub workspace script selection

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m run dev
```
```bash
m run build -- --mode production
```
```bash
m run "/^test:/"
```

## In Scope

- Script lookup and explicit missing-script diagnostics.
- pre/post hooks, PATH injection, INIT_CWD, npm lifecycle environment, shell selection, and argument forwarding.
- Regex script selectors where adopted.
- Human, silent, stream, aggregate, JSON, and NDJSON reporters.
- Signal and exit-code propagation.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Child process handling lives in a reusable supervisor.
- Environment construction is pure and testable.
- `m run` remains the unambiguous escape hatch for scripts colliding with built-ins.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Define ScriptRunner interface with context cancellation and stable error codes
- [ ] Implement argument forwarding with `--` separator semantics
- [ ] Implement per-package output prefix handling for future workspace use
- [ ] Add conformance fixtures against Nub script runner behavior

### Core logic

- [ ] Implement package.json script lookup with explicit missing-script diagnostics
- [ ] Implement reusable ProcessSupervisor with process groups
- [ ] Implement regex script selector parsing where adopted
- [ ] Document m run as unambiguous escape hatch for built-in collisions

### CLI / UX

- [ ] Implement pre/post lifecycle hook expansion with ordering guarantees
- [ ] Implement signal forwarding, cancellation, and exit-code propagation
- [ ] Implement shell completion from manifest scripts
- [ ] Benchmark script startup hot path without unbounded goroutines

### Tests & fixtures

- [ ] Implement pure npm-compatible environment builder (INIT_CWD, npm_* vars, PATH)
- [ ] Implement stdin/TTY preservation and output interleaving policy
- [ ] Add unit tests for env builder determinism and hook ordering

### Docs & observability

- [ ] Implement cross-platform shell selection and command quoting
- [ ] Implement human, silent, stream, aggregate, JSON, and NDJSON reporters
- [ ] Add integration tests for signal, exit code, and quoting fixtures

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: m run dev executes script with npm-compatible environment on Linux/macOS/Windows
- [ ] Acceptance: pre/post hooks run in documented order with correct failure propagation
- [ ] Acceptance: Signals forwarded to child; exit code matches child process
- [ ] Acceptance: m run remains explicit path when script name collides with built-in
- [ ] Acceptance: Reporter modes produce deterministic structured output in CI
- [ ] Fixture ready: `fixtures/runner/basic-scripts — pre/post hook ordering`
- [ ] Fixture ready: `fixtures/runner/shell-quoting — bash/cmd/PowerShell quoting matrix`
- [ ] Fixture ready: `fixtures/runner/signals — SIGINT/SIGTERM propagation`
- [ ] Fixture ready: `fixtures/runner/env-parity — npm lifecycle environment golden`
- [ ] Fixture ready: `fixtures/runner/reporters — output mode snapshots`


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
| m run script execution | Nub nub run | npm-compatible env + hooks | 0040 |
| Lifecycle pre/post hooks | Nub runner | pre/post expansion | 0040 |
| Reporter modes | Nub reporters | human/silent/stream/aggregate/json/ndjson | 0040 |
| Regex script selectors | Nub | /^pattern/ selection | 0040 |

## Go Package Map

**Packages / paths:**

- `internal/runner`
- `internal/process`
- `internal/manifest`
- `internal/project`
- `internal/lifecycle`
- `cmd/m (run subcommand)`

**Forbidden import edges:**

- internal/runtime
- internal/transform
- internal/resolver

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  cli[m run] --> lookup[ScriptLookup]
  lookup --> env[EnvBuilder]
  env --> hooks[LifecycleHooks]
  hooks --> sup[ProcessSupervisor]
  sup --> child[ShellOrBinChild]
```

## Commands and Flags

| Command | Flags | Notes |
|---|---|---|
| `m run <script>` | `--if-present`, reporter flags | Explicit script escape hatch |
| `m run build -- --flag` | `--` separator | Args after `--` forwarded to script |
| `m run "/^test:/"` | regex selector | Pattern-based script selection |

Environment: npm lifecycle vars, INIT_CWD, PATH injection. Exit codes: child exit code propagated.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Runner event schema (draft) | Structured task/output events |
| Shell quoting golden outputs | Cross-platform regression baselines |

## Concrete Test Fixtures

- `fixtures/runner/basic-scripts — pre/post hook ordering`
- `fixtures/runner/shell-quoting — bash/cmd/PowerShell quoting matrix`
- `fixtures/runner/signals — SIGINT/SIGTERM propagation`
- `fixtures/runner/env-parity — npm lifecycle environment golden`
- `fixtures/runner/reporters — output mode snapshots`

## Acceptance Scenarios

1. m run dev executes script with npm-compatible environment on Linux/macOS/Windows
2. pre/post hooks run in documented order with correct failure propagation
3. Signals forwarded to child; exit code matches child process
4. m run remains explicit path when script name collides with built-in
5. Reporter modes produce deterministic structured output in CI

## Nub Conformance Targets

- Nub script runner env + hooks | parity
- Nub reporters and prefix handling | parity
- Nub workspace script selection | defer to 0041

## Open Decisions

- Default shell on Windows: cmd vs PowerShell for npm script parity
- Whether regex script selectors ship in v1 or behind experimental gate

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
