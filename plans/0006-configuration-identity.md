# 0006 — Configuration and Project Identity Model

## Document Control

| Item | Detail |
|---|---|
| Phase | Foundation |
| Primary objective | Implement layered configuration and package-manager identity detection without reading branded configuration from an unrelated incumbent manager. |
| Required predecessors | 0004, 0005 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Implement layered configuration and package-manager identity detection without reading branded configuration from an unrelated incumbent manager.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0004 before starting this MVP.
- Complete and merge 0005 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub global and project JSONC configuration
- Nub identity versus compatibility project model
- Package-manager detection by `packageManager`, `devEngines`, installed version, and lockfile signal

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m config get registry
```
```bash
m config set install.linker auto
```
```bash
m config list --sources
```

## In Scope

- Defaults, global config, project config, environment, and CLI precedence.
- Neutral Mew project configuration in `m.jsonc` or a final ADR-selected name.
- Compatibility adapters for npmrc, pnpm, Yarn, Bun, and Nub configuration.
- Specific package-manager major-version models.
- Credential references separated from non-secret configuration.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Use typed configuration with source provenance on every effective value.
- Preserve comments and ordering when modifying user-owned JSONC or compatible formats.
- Do not read pnpm-specific files for an Mew-identity project unless explicitly importing.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Define config layer precedence: defaults, global, project, env, CLI
- [ ] Implement identity detector for packageManager, devEngines, lockfile, Mew native
- [ ] Precedence table tests
- [ ] Preserve comments and ordering when modifying user-owned JSONC

### Core logic

- [ ] Define identity detection order matching AGENTS.md
- [ ] Implement offline/prefer-offline flags in config model
- [ ] Identity detection fixtures for each lockfile type
- [ ] Separate credential references from non-secret configuration

### CLI / UX

- [ ] List owned config keys vs pass-through npmrc keys
- [ ] Validate unknown keys policy (warn vs fail)
- [ ] Malformed config fail-closed tests
- [ ] Add unsupported-config diagnostics that never silently ignore safety-critical options

### Tests & fixtures

- [ ] Forbid reading unrelated branded PM config as authority
- [ ] Specify config command grammar
- [ ] Document every public config key

### Docs & observability

- [ ] Implement layered loader with deterministic merge
- [ ] Effective-config debug output with redaction
- [ ] Document identity detection with examples

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Detection order matches AGENTS.md
- [ ] Acceptance: Conflicting signals produce explicit errors, not silent picks
- [ ] Acceptance: Env overrides project overrides user as documented
- [ ] Acceptance: pnpm-specific files are not read for Mew-identity projects unless importing
- [ ] Fixture ready: `fixtures/identity/npm-lock`
- [ ] Fixture ready: `fixtures/identity/pnpm-lock`
- [ ] Fixture ready: `fixtures/identity/nub-lock`
- [ ] Fixture ready: `fixtures/identity/packageManager-field`
- [ ] Fixture ready: `fixtures/identity/conflict-signals`


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
| Layered config | Nub/npmrc-like | Mew config layers | 0006 |
| PM identity detection | Nub | packageManager + lockfile order | 0006 |
| Source-aware config | Nub JSONC | Provenance on effective values | 0006 |

## Go Package Map

**Packages / paths:**

- `internal/config`
- `internal/project`

**Forbidden import edges:**

- internal/resolver
- internal/linker
- internal/fetch

## Data Flow

```mermaid
flowchart LR
  env[Env] --> merge[ConfigMerge] --> identity[IdentityDetect] --> effective[EffectiveConfig]
```

## Commands and Flags

| Command | Notes |
|---|---|
| `m config get/set/list` | May land with 0026; define keys now |
| `m config list --sources` | Show layer provenance |
| Global flags | `--config`, `--cwd`, `--offline` |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| User config file | `~/.config/github.com/mewisme/m/config.toml` (final path per naming doc) |
| Project config | Neutral names where convention exists |
| Effective config dump | Debug only with redaction |

## Concrete Test Fixtures

- `fixtures/identity/npm-lock`
- `fixtures/identity/pnpm-lock`
- `fixtures/identity/nub-lock`
- `fixtures/identity/packageManager-field`
- `fixtures/identity/conflict-signals`

## Acceptance Scenarios

1. Detection order matches AGENTS.md
2. Conflicting signals produce explicit errors, not silent picks
3. Env overrides project overrides user as documented
4. pnpm-specific files are not read for Mew-identity projects unless importing

## Nub Conformance Targets

- packageManager field precedence | parity
- No foreign branded config authority | extension

## Open Decisions

- TOML vs JSON vs yaml for Mew-native config file
- Final neutral project config filename (m.jsonc vs ADR-selected name)

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
