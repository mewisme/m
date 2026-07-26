# 0028 — Core MVP 19 — Explainability, Plans, Semantic Diffs, and Time Travel

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 19 |
| Primary objective | Expose every resolver and installer decision, preview all mutations, compare dependency graphs semantically, and run or restore historical snapshots. |
| Required predecessors | 0017, 0020, 0026 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Expose every resolver and installer decision, preview all mutations, compare dependency graphs semantically, and run or restore historical snapshots.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0017 before starting this MVP.
- Complete and merge 0020 before starting this MVP.
- Complete and merge 0026 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub why/list behavior
- Mew signature explainable-resolver and rollback extensions

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m plan update
```
```bash
m explain react
```
```bash
m explain peer react
```
```bash
m lock diff
```
```bash
m history
```
```bash
m shell --snapshot ID
```
```bash
m run --snapshot ID dev
```

## In Scope

- Resolution candidate explanations.
- Install operation plans with downloads, reuse, scripts, risk, disk, and file changes.
- Semantic graph and lockfile diffs.
- Snapshot history queries and restoration.
- Ephemeral historical environments without changing the working tree.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Decision traces are generated during resolution, not reconstructed heuristically afterward.
- Plans are serializable, signed or checksummed, and validated against current state before application.
- Historical environments reference immutable content and explicit policy snapshots.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement m explain showing version selection reasoning
- [ ] Integrate snapshot list/restore from 0017 with UX polish
- [ ] Add plan preview tests matching actual install delta
- [ ] Performance: explain completes in <1s on large graph fixture

### Core logic

- [ ] Implement m explain peer for peer dependency conflicts
- [ ] Emit structured JSON for explain and plan for agents
- [ ] Document explain trace schema
- [ ] Add m history showing snapshot timeline

### CLI / UX

- [ ] Implement m plan previewing fetch/link/manifest changes
- [ ] Colorize human explain output via diagnostics reporter
- [ ] Never mutate state in explain/plan/diff commands

### Tests & fixtures

- [ ] Implement semantic diff between two lock graphs
- [ ] Support diff against npm/pnpm locks via adapters
- [ ] Support piping plan to file for CI review

### Docs & observability

- [ ] Compare m.lock revisions and incumbent lock formats
- [ ] Add golden tests for explain output on fixture graphs
- [ ] Link explain output to stable error codes

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: m explain prints version selection path for target package
- [ ] Acceptance: m plan --json matches actual install file changes on dry-run
- [ ] Acceptance: m diff lock detects semver bump between two locks
- [ ] Acceptance: m snapshot restore returns project to recorded state
- [ ] Acceptance: Explain/plan/diff never modify project files
- [ ] Fixture ready: `fixtures/explain/peer-conflict`
- [ ] Fixture ready: `fixtures/explain/override-chain`
- [ ] Fixture ready: `testdata/plan/install-delta-golden/`
- [ ] Fixture ready: `testdata/diff/lock-revisions/`


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
| m explain | Nub explain | Resolver decision traces | 0028 |
| m plan | Nub plan | Preview install mutations | 0028 |
| m diff | Nub diff | Semantic graph comparison | 0028 |
| Snapshot restore | 0017 snapshots | Historical state restore | 0028 |

## Go Package Map

**Packages / paths:**

- `internal/resolver`
- `internal/transaction`
- `internal/diagnostics`
- `internal/app`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  graph[lock graph] --> explain[explain engine] --> plan[plan preview] --> diff[semantic diff] --> snap[snapshot restore]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m explain <pkg>` | `--json` | Why version selected |
| `m plan` | `--json`, install flags | Dry mutation preview |
| `m diff lock` | `--from`, `--to` | Graph semantic diff |
| `m snapshot restore` | `<id>` | Restore historical state |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Explain trace output | Human and JSON formats |
| Plan file | Optional saved install plan |

## Concrete Test Fixtures

- `fixtures/explain/peer-conflict`
- `fixtures/explain/override-chain`
- `testdata/plan/install-delta-golden/`
- `testdata/diff/lock-revisions/`

## Acceptance Scenarios

1. m explain prints version selection path for target package
2. m plan --json matches actual install file changes on dry-run
3. m diff lock detects semver bump between two locks
4. m snapshot restore returns project to recorded state
5. Explain/plan/diff never modify project files

## Nub Conformance Targets

- Nub explain output | parity
- Nub plan preview | parity
- Semantic lock diff | parity

## Open Decisions

- Whether plan files are persisted by default
- Explain verbosity levels vs single default

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
