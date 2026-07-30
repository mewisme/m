# 0043 — Runner MVP 4 — Local Package Binary Execution

## Document Control

| Item | Detail |
|---|---|
| Phase | Runner / MVP 4 |
| Primary objective | Implement network-free local binary execution through `m exec`, workspace-aware `.bin` discovery, and robust platform shims. |
| Required predecessors | 0019, 0040 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Implement network-free local binary execution through `m exec`, workspace-aware `.bin` discovery, and robust platform shims.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0019 before starting this MVP.
- Complete and merge 0040 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub `exec` local-first behavior
- Node modules bin resolution and PnP bin helper

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m exec eslint .
```
```bash
m exec --package typescript tsc --version
```

## In Scope

- Current importer and ancestor/workspace binary lookup.
- Explicit package-to-bin selection.
- Node-modules and PnP execution adapters.
- PATH, cwd, TTY, signal, and exit-code preservation.
- No registry fallback for `m exec`.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Resolve executable identity before spawning.
- Use direct execution when possible and shell only when requested.
- Treat Windows command shims and Unix executable bits explicitly.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement bin index and lookup for current importer
- [ ] Resolve executable identity before spawning (shebang, extensions)
- [ ] Integrate with ProcessSupervisor from 0040
- [ ] Redact credentials from exec diagnostics

### Core logic

- [ ] Walk ancestor and workspace packages for .bin discovery
- [ ] Use direct process spawning; shell only when requested
- [ ] Add bin collision and multiple-bin fixtures
- [ ] Add stable error codes for missing/ambiguous bins

### CLI / UX

- [ ] Implement explicit package-to-bin selection with ambiguity errors
- [ ] Handle Windows cmd/PowerShell shims and Unix executable bits
- [ ] Add Windows shim and PnP conformance tests
- [ ] Ensure mx dlx does not weaken local-only exec contract

### Tests & fixtures

- [ ] Implement node_modules layout execution adapter
- [ ] Preserve PATH, cwd, TTY, signals, and exit codes
- [ ] Document m exec vs mx remote execution boundary

### Docs & observability

- [ ] Implement Yarn PnP bin resolution adapter
- [ ] Fail closed with install suggestion on local miss (no registry fetch)
- [ ] Benchmark bin lookup on large node_modules trees

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: m exec eslint runs local bin without network access
- [ ] Acceptance: Ambiguous package bins produce clear selection errors
- [ ] Acceptance: PnP projects resolve bins through adapter
- [ ] Acceptance: Windows shims execute with correct quoting
- [ ] Acceptance: Local miss suggests install; never silently fetches
- [ ] Fixture ready: `fixtures/exec/single-bin — eslint-style lookup`
- [ ] Fixture ready: `fixtures/exec/multi-bin — ambiguity errors`
- [ ] Fixture ready: `fixtures/exec/workspace — monorepo bin resolution`
- [ ] Fixture ready: `fixtures/exec/pnp — PnP bin execution`
- [ ] Fixture ready: `fixtures/exec/windows-shim — cmd/ps1 shim matrix`


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
| m exec local binaries | Nub exec | Network-free local execution | 0043 |
| Workspace bin lookup | Nub | ancestor and workspace .bin discovery | 0043 |
| PnP bin adapter | Nub PnP helper | PnP execution path | 0043 |
| No registry fallback | Nub local-first | m exec never fetches remotely | 0043 |

## Go Package Map

**Packages / paths:**

- `internal/runner`
- `internal/linker`
- `internal/compat`
- `cmd/m (exec subcommand)`

**Forbidden import edges:**

- internal/runtime
- internal/transform
- internal/resolver

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  cli[m exec] --> resolve[BinResolver]
  resolve --> layout[LayoutAdapter]
  layout --> spawn[DirectSpawn]
  spawn --> sup[ProcessSupervisor]
```

## Commands and Flags

| Command | Flags | Notes |
|---|---|---|
| `m exec eslint .` | bin name + args | Current importer bin lookup |
| `m exec --package typescript tsc --version` | `--package` | Explicit package-to-bin selection |

No network/registry fallback. PATH, cwd, TTY, signals preserved.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Bin index cache (in-memory) | Fast repeated exec in CI |
| Install suggestion templates | Actionable miss diagnostics |

## Concrete Test Fixtures

- `fixtures/exec/single-bin — eslint-style lookup`
- `fixtures/exec/multi-bin — ambiguity errors`
- `fixtures/exec/workspace — monorepo bin resolution`
- `fixtures/exec/pnp — PnP bin execution`
- `fixtures/exec/windows-shim — cmd/ps1 shim matrix`

## Acceptance Scenarios

1. m exec eslint runs local bin without network access
2. Ambiguous package bins produce clear selection errors
3. PnP projects resolve bins through adapter
4. Windows shims execute with correct quoting
5. Local miss suggests install; never silently fetches

## Nub Conformance Targets

- Nub exec local-first behavior | parity
- Nub PnP bin helper | parity
- Nub workspace bin discovery | parity

## Open Decisions

- Whether m exec supports global -g installed bins in v1
- Shell mode default on Windows for .cmd shims

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
