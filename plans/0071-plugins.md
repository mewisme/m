# 0071 — Product MVP 2 — External Command Plugin Convention

## Document Control

| Item | Detail |
|---|---|
| Phase | Product / MVP 2 |
| Primary objective | Support discoverable external `m-<verb>` commands without loading untrusted code into the Mew process. |
| Required predecessors | 0010, 0043, 0062 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Support discoverable external `m-<verb>` commands without loading untrusted code into the Mew process.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0010 before starting this MVP.
- Complete and merge 0043 before starting this MVP.
- Complete and merge 0062 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub external `nub-<verb>` plugin convention

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m hello
```
```bash
m plugin list
```
```bash
m plugin doctor
```

## In Scope

- PATH discovery of `m-<verb>` executables.
- Built-in command precedence.
- Plugin metadata handshake, version compatibility, completion, and structured events.
- Optional installation via `mx` or package-manager command.
- Trust and origin display.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Plugins are subprocesses, not Go binary plugins or an unstable in-process ABI.
- No plugin can shadow a built-in command.
- Environment and credentials are minimized according to plugin trust policy.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Define m-<verb> executable naming and handshake protocol
- [ ] Spawn plugins as subprocesses with minimized environment/credentials
- [ ] Document SDK examples in JavaScript, Go, and shell
- [ ] Audit plugin env surface against trust policy

### Core logic

- [ ] Discover plugins from PATH without loading untrusted code into m
- [ ] Implement structured plugin events and exit code propagation
- [ ] Test shadowing and version mismatch failures
- [ ] Benchmark plugin dispatch overhead

### CLI / UX

- [ ] Enforce built-in command precedence; plugins never shadow built-ins
- [ ] Implement m plugin list and doctor commands
- [ ] Test malicious plugin output/protocol handling
- [ ] Freeze handshake protocol before public SDK publish

### Tests & fixtures

- [ ] Implement plugin metadata cache for completion and doctor
- [ ] Display plugin trust and origin in diagnostics
- [ ] Test cross-platform executable discovery

### Docs & observability

- [ ] Implement version compatibility checks in handshake
- [ ] Support optional installation via mx or package-manager command
- [ ] Integrate completion for discovered plugin verbs

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: m hello runs m-hello on PATH when not a built-in
- [ ] Acceptance: Built-in commands always win over plugin names
- [ ] Acceptance: Plugin handshake rejects incompatible protocol versions
- [ ] Acceptance: m plugin doctor reports origin and trust metadata
- [ ] Acceptance: No untrusted code loaded into m process address space
- [ ] Fixture ready: `fixtures/plugins/hello-go — minimal Go m-hello plugin`
- [ ] Fixture ready: `fixtures/plugins/hello-js — Node plugin example`
- [ ] Fixture ready: `fixtures/plugins/shadow-builtin — must fail dispatch`
- [ ] Fixture ready: `fixtures/plugins/malicious-output — protocol hardening`
- [ ] Fixture ready: `fixtures/plugins/version-mismatch — compatibility errors`


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
| m-<verb> plugins | Nub nub-<verb> | external subprocess convention | 0071 |
| PATH discovery | Nub | discover m-* executables | 0071 |
| Built-in precedence | Nub | plugins cannot shadow built-ins | 0071 |
| Plugin handshake | Nub | metadata, version, completion | 0071 |

## Go Package Map

**Packages / paths:**

- `internal/cli`
- `internal/runner`
- `cmd/m (plugin subcommand)`

**Forbidden import edges:**

- internal/runtime
- internal/transform

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  argv[m hello] --> builtin[BuiltInCheck]
  builtin --> discover[PATHDiscovery]
  discover --> handshake[PluginHandshake]
  handshake --> spawn[PluginSubprocess]
```

## Commands and Flags

| Command | Flags | Notes |
|---|---|---|
| `m hello` | dispatches to m-hello | If on PATH |
| `m plugin list` | | Discovered plugins |
| `m plugin doctor` | | Compatibility and trust diagnostics |

Plugins are subprocesses; no in-process Go plugins or unstable ABI.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Plugin handshake protocol spec | Versioned metadata exchange |
| Plugin SDK examples | JS, Go, shell reference plugins |

## Concrete Test Fixtures

- `fixtures/plugins/hello-go — minimal Go m-hello plugin`
- `fixtures/plugins/hello-js — Node plugin example`
- `fixtures/plugins/shadow-builtin — must fail dispatch`
- `fixtures/plugins/malicious-output — protocol hardening`
- `fixtures/plugins/version-mismatch — compatibility errors`

## Acceptance Scenarios

1. m hello runs m-hello on PATH when not a built-in
2. Built-in commands always win over plugin names
3. Plugin handshake rejects incompatible protocol versions
4. m plugin doctor reports origin and trust metadata
5. No untrusted code loaded into m process address space

## Nub Conformance Targets

- Nub nub-<verb> plugin convention | parity

## Open Decisions

- Plugin trust tiers and signed plugin support in v1
- Whether plugins receive full project env or filtered subset

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
