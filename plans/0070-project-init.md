# 0070 — Product MVP 1 — TypeScript-First Project Initialization

## Document Control

| Item | Detail |
|---|---|
| Phase | Product / MVP 1 |
| Primary objective | Create a fast, opinionated but transparent TypeScript-first project scaffold and a minimal manifest-only mode. |
| Required predecessors | 0011, 0031, 0051 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Create a fast, opinionated but transparent TypeScript-first project scaffold and a minimal manifest-only mode.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0011 before starting this MVP.
- Complete and merge 0031 before starting this MVP.
- Complete and merge 0051 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub TypeScript-first `init` scaffold and template guidance

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m init
```
```bash
m init --manifest-only
```
```bash
mx create-vite
```

## In Scope

- Interactive and noninteractive initialization.
- Package name, module type, source layout, TypeScript config, scripts, ignore files, and initial install.
- No hidden framework templates; delegate specialized templates to `mx create-*`.
- Transaction and rollback on failed initialization.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Generated projects are deterministic for identical answers and Mew version.
- Every generated choice is visible before commit.
- Do not overwrite nonempty directories without explicit policy.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Define scaffold templates as embedded versioned assets
- [ ] Implement dry-run file plan preview before write
- [ ] Use m.lock as default for greenfield projects
- [ ] Document every generated choice before commit

### Core logic

- [ ] Implement interactive and noninteractive init prompts/flags
- [ ] Integrate initial install via transaction boundary from core PM
- [ ] Preserve incumbent lockfile if detected in existing directory
- [ ] Benchmark init time cold vs warm cache

### CLI / UX

- [ ] Generate package.json with module type, scripts, and Mew identity defaults
- [ ] Rollback all writes on init failure or interruption
- [ ] Add golden tests for generated project trees
- [ ] Publish migration notes from npm create equivalents

### Tests & fixtures

- [ ] Generate tsconfig with sensible strict defaults
- [ ] Refuse overwrite of nonempty directories without explicit policy
- [ ] Smoke-test build/run on generated projects via 0051 runtime

### Docs & observability

- [ ] Generate source layout, gitignore, and editor config stubs
- [ ] Delegate specialized framework templates to mx create-* hints
- [ ] Test nonempty directory and interruption recovery

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: m init creates deterministic TS project that runs with m dev
- [ ] Acceptance: Failed init leaves no partial manifest or half-written tree
- [ ] Acceptance: manifest-only mode writes only package.json
- [ ] Acceptance: Nonempty directory policy enforced with clear errors
- [ ] Acceptance: Framework templates directed to mx, not hidden in m init
- [ ] Fixture ready: `fixtures/init/golden — deterministic tree snapshots`
- [ ] Fixture ready: `fixtures/init/manifest-only — minimal output`
- [ ] Fixture ready: `fixtures/init/nonempty — refusal/overwrite policy`
- [ ] Fixture ready: `fixtures/init/interrupt — rollback journal`
- [ ] Fixture ready: `fixtures/init/smoke — generated project build/run`


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
| m init scaffold | Nub TypeScript-first init | opinionated TS template | 0070 |
| manifest-only mode | Nub | --manifest-only | 0070 |
| Template delegation | Nub | mx create-* for frameworks | 0070 |
| Transactional init | Mew | rollback on failure | 0070 |

## Go Package Map

**Packages / paths:**

- `internal/app`
- `internal/manifest`
- `internal/transaction`
- `cmd/m (init)`
- `templates/init`

**Forbidden import edges:**

- internal/runtime (beyond generated project smoke)
- internal/transform (full pipeline)

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  prompt[InitPrompts] --> plan[FilePlan]
  plan --> txn[InstallTransaction]
  txn --> tree[ProjectTree]
  tree --> smoke[BuildRunSmoke]
```

## Commands and Flags

| Command | Flags | Notes |
|---|---|---|
| `m init` | interactive prompts | TypeScript-first scaffold |
| `m init --manifest-only` | | package.json only |
| `mx create-vite` | via 0044 | Delegated framework templates |

Generated projects deterministic for identical inputs and Mew version.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Embedded init templates (versioned) | Deterministic scaffold assets |
| Init file plan format | Dry-run preview output |

## Concrete Test Fixtures

- `fixtures/init/golden — deterministic tree snapshots`
- `fixtures/init/manifest-only — minimal output`
- `fixtures/init/nonempty — refusal/overwrite policy`
- `fixtures/init/interrupt — rollback journal`
- `fixtures/init/smoke — generated project build/run`

## Acceptance Scenarios

1. m init creates deterministic TS project that runs with m dev
2. Failed init leaves no partial manifest or half-written tree
3. manifest-only mode writes only package.json
4. Nonempty directory policy enforced with clear errors
5. Framework templates directed to mx, not hidden in m init

## Nub Conformance Targets

- Nub TypeScript-first init | parity
- Greenfield m.lock default | Mew policy

## Open Decisions

- Default module type: ESM vs dual package
- Interactive vs CI-default noninteractive init

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
