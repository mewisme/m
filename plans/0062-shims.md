# 0062 — Manager MVP 3 — Node, PM, and Self Shims

## Document Control

| Item | Detail |
|---|---|
| Phase | Managers / MVP 3 |
| Primary objective | Install safe cross-platform shims that resolve pinned Node, Mew, and external package-manager versions without unexpectedly augmenting plain Node calls. |
| Required predecessors | 0010, 0060, 0061 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Install safe cross-platform shims that resolve pinned Node, Mew, and external package-manager versions without unexpectedly augmenting plain Node calls.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0010 before starting this MVP.
- Complete and merge 0060 before starting this MVP.
- Complete and merge 0061 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub Node PATH-hijack contract
- Nub PM shims and self-shim provisioning

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m shim install
```
```bash
m shim status
```
```bash
m shim remove
```
```bash
m pm shim
```
```bash
m pm unshim
```

## In Scope

- POSIX and Windows shim installation.
- Node version resolution only for `node` shim; runtime augmentation remains on `m`/`mx`.
- Package manager dispatch from project identity.
- Optional Mew self-version pinning.
- Recursion prevention and emergency bypass environment variables.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Shims are tiny, auditable, and recoverable.
- Every shim has an explicit bypass and reports the resolved target in debug mode.
- Never replace an unrelated executable without confirmation and backup.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Design shim protocol and cross-platform installation paths
- [ ] Implement recursion prevention and emergency bypass env vars
- [ ] Test PATH precedence matrix across shells
- [ ] Benchmark shim dispatch overhead

### Core logic

- [ ] Implement POSIX launcher scripts and Windows exe/cmd strategy
- [ ] Implement shim status, repair, backup, and uninstall flows
- [ ] Test recursion and broken-pin recovery scenarios
- [ ] Coordinate with 0072 installers for default shim paths

### CLI / UX

- [ ] Resolve pinned Node version for node shim without runtime augmentation
- [ ] Never replace unrelated executables without confirmation and backup
- [ ] Test Windows quoting and executable extension handling
- [ ] Ensure plain node calls through shim do not inject Mew preloads

### Tests & fixtures

- [ ] Dispatch package-manager shims from project identity via 0061
- [ ] Report resolved target in debug/diagnostic mode
- [ ] Document node vs m/mx augmentation boundary clearly

### Docs & observability

- [ ] Support optional Mew self-version pinning in shims
- [ ] Integrate shell completion and PATH setup instructions
- [ ] Audit shim contents for minimal attack surface

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: node shim runs pinned stock Node without Mew augmentation
- [ ] Acceptance: m/mx shims resolve project-pinned Mew version when configured
- [ ] Acceptance: Recursion guards prevent shim loops
- [ ] Acceptance: m shim remove restores prior PATH state from backup
- [ ] Acceptance: Windows shims handle extensions and quoting correctly
- [ ] Fixture ready: `fixtures/shim/path-precedence — shell PATH matrix`
- [ ] Fixture ready: `fixtures/shim/recursion — MEW_SHIM_BYPASS scenarios`
- [ ] Fixture ready: `fixtures/shim/windows — cmd/ps1/exe quoting`
- [ ] Fixture ready: `fixtures/shim/broken-pin — recovery and repair`
- [ ] Fixture ready: `fixtures/shim/pm-dispatch — project manager routing`


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
| PATH shims | Nub Node hijack contract | node shim resolves pin only | 0062 |
| PM shims | Nub | pnpm/npm/yarn dispatch | 0062 |
| Self shims | Nub | optional Mew version pin | 0062 |
| Recursion guards | Nub | bypass env vars | 0062 |

## Go Package Map

**Packages / paths:**

- `internal/node`
- `internal/pmmanager`
- `cmd/m (shim subcommand)`

**Forbidden import edges:**

- internal/runtime/augment
- internal/transform

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  shim[ShimLauncher] --> resolve[VersionResolve]
  resolve --> target[TargetBinary]
  target --> guard[RecursionGuard]
  guard --> exec[DirectExec]
```

## Commands and Flags

| Command | Flags | Notes |
|---|---|---|
| `m shim install` | | Install shims to user PATH dir |
| `m shim status` | | Show shim targets and health |
| `m shim remove` | | Uninstall shims safely |
| `m pm shim` | | PM-specific shim install |
| `m pm unshim` | | Remove PM shims |

node shim selects version only; augmentation stays on m/mx.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Shim launcher binaries | Tiny auditable POSIX/Windows stubs |
| Shim installation manifest | Backup paths and repair metadata |

## Concrete Test Fixtures

- `fixtures/shim/path-precedence — shell PATH matrix`
- `fixtures/shim/recursion — MEW_SHIM_BYPASS scenarios`
- `fixtures/shim/windows — cmd/ps1/exe quoting`
- `fixtures/shim/broken-pin — recovery and repair`
- `fixtures/shim/pm-dispatch — project manager routing`

## Acceptance Scenarios

1. node shim runs pinned stock Node without Mew augmentation
2. m/mx shims resolve project-pinned Mew version when configured
3. Recursion guards prevent shim loops
4. m shim remove restores prior PATH state from backup
5. Windows shims handle extensions and quoting correctly

## Nub Conformance Targets

- Nub Node PATH-hijack contract | parity
- Nub PM shims and self-shim provisioning | parity

## Open Decisions

- Default shim install location per OS
- Whether github.com/mewisme/m/mewx alias shims ship in v1

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
