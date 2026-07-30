# 0021 — Core MVP 12 — Lifecycle Scripts, Trust, and Sandbox Policy

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 12 |
| Primary objective | Run required package lifecycle scripts under explicit trust policy, capability restrictions, reproducible build caching, and complete audit logs. |
| Required predecessors | 0018, 0020 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Run required package lifecycle scripts under explicit trust policy, capability restrictions, reproducible build caching, and complete audit logs.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0018 before starting this MVP.
- Complete and merge 0020 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Aube build-policy and trusted-dependency flow
- Nub minimum-release-age and build approval UX

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m trust --interactive
```
```bash
m trust esbuild
```
```bash
m install --ignore-scripts
```
```bash
m builds list
```

## In Scope

- preinstall, install, postinstall, prepare, and supported package-manager lifecycle semantics.
- Default trust policy, per-package approvals, network and filesystem capabilities.
- Non-interactive CI behavior that fails closed.
- Build output identity and reusable build cache.
- Platform-specific sandbox backends with explicit degradation reports.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Separate decision, execution, sandbox, and build-output capture.
- Never execute scripts directly in immutable base store objects.
- Include package content, command, platform, toolchain, policy, and selected environment in build cache keys.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement lifecycle script discovery from package.json scripts field
- [ ] Propagate correct PATH and node_modules/.bin for script context
- [ ] Redact secrets from script environment in logs
- [ ] Never execute lifecycle during --dry-run

### Core logic

- [ ] Run preinstall/install/postinstall/prepare in npm order
- [ ] Cache reproducible build script outputs keyed by inputs
- [ ] Add tests with benign fixture scripts
- [ ] Support m approve-builds to add package to trust list

### CLI / UX

- [ ] Enforce ignore-scripts flag and config
- [ ] Write audit log entry for every script: package, script, exit code
- [ ] Add failure tests: script exit non-zero triggers rollback

### Tests & fixtures

- [ ] Implement trust policy: prompt or allowlist for unknown scripts
- [ ] Fail install on script failure with rollback via transaction
- [ ] Document lifecycle policy for CI (--ignore-scripts default?)

### Docs & observability

- [ ] Execute scripts in sandbox with restricted env and filesystem
- [ ] Support Windows cmd/sh and Unix sh shebang resolution
- [ ] Integrate with isolated linker bin paths

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: postinstall script runs after package materialized
- [ ] Acceptance: Failing lifecycle script triggers full install rollback
- [ ] Acceptance: --ignore-scripts skips all lifecycle execution
- [ ] Acceptance: Untrusted package prompts or blocks per policy
- [ ] Acceptance: Audit log records script executions
- [ ] Fixture ready: `fixtures/lifecycle/postinstall-write-file`
- [ ] Fixture ready: `fixtures/lifecycle/failing-script`
- [ ] Fixture ready: `fixtures/lifecycle/native-addon-build-stub`


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

- Deny scripts by policy unless explicitly trusted or covered by a reviewed default-trust baseline.
- Strip registry tokens, cloud credentials, SSH agents, and unrelated environment variables by default.
- Treat unsupported sandbox capabilities as a visible policy decision, never as silent success.

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
| Lifecycle scripts | npm scripts | preinstall/install/postinstall | 0021 |
| Trust policy | Nub policy | Allow/deny script execution | 0021 |
| Sandbox restrictions | Nub | Capability-limited execution | 0021 |
| Build reproducibility cache | Nub | Cache lifecycle outputs | 0021 |

## Go Package Map

**Packages / paths:**

- `internal/lifecycle`
- `internal/policy`
- `internal/app`
- `internal/process`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  install[commit] --> policy[policy check] --> life[internal/lifecycle] --> sandbox[SandboxedExec] --> audit[AuditLog]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m install` | `--ignore-scripts` | Skip all lifecycle scripts |
| `m config set ignore-scripts true` | — | Persistent policy |
| `m approve-builds` | — | Trust package build scripts |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Lifecycle audit log | Per-script execution record |
| Trusted packages list | Policy allowlist store |

## Concrete Test Fixtures

- `fixtures/lifecycle/postinstall-write-file`
- `fixtures/lifecycle/failing-script`
- `fixtures/lifecycle/native-addon-build-stub`

## Acceptance Scenarios

1. postinstall script runs after package materialized
2. Failing lifecycle script triggers full install rollback
3. --ignore-scripts skips all lifecycle execution
4. Untrusted package prompts or blocks per policy
5. Audit log records script executions

## Nub Conformance Targets

- npm lifecycle script order | parity
- ignore-scripts behavior | parity
- Nub trust policy model | parity

## Open Decisions

- Default trust policy for new projects (prompt vs deny)
- Sandbox depth: full sandbox vs path-restricted only

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
