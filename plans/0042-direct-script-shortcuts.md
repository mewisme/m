# 0042 — Runner MVP 3 — Direct `m <script>` Shortcuts

## Document Control

| Item | Detail |
|---|---|
| Phase | Runner / Mew Extension |
| Primary objective | Allow exact package.json script names such as `m dev` and `m start` while preserving deterministic built-in command precedence. |
| Required predecessors | 0010, 0040 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Allow exact package.json script names such as `m dev` and `m start` while preserving deterministic built-in command precedence.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0010 before starting this MVP.
- Complete and merge 0040 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Intentional Mew divergence: Nub requires explicit `nub run <script>`

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m dev
```
```bash
m start
```
```bash
m build -- --mode production
```
```bash
m run add
```

## In Scope

- Exact script fallback after built-in command and alias resolution.
- Arguments passed without requiring a separator when parsing is unambiguous.
- Helpful suggestions for misspelled built-ins and scripts.
- Interactive script picker for bare `m` only when explicitly enabled or selected by UX policy.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Precedence: built-in command, built-in alias, exact script, optional local binary policy, suggestions, error.
- A script named `add` is reachable through `m run add`, never through `m add`.
- No fuzzy execution; fuzzy matches are suggestions only.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement two-pass CLI dispatch after built-in and alias resolution
- [ ] Implement Levenshtein or equivalent suggestion ranking (suggestions only)
- [ ] Build exhaustive collision matrix tests (built-in vs script names)
- [ ] Gate behavior behind experimental flag until stabilization

### Core logic

- [ ] Wire exact package.json script fallback into main m dispatch
- [ ] Reject fuzzy execution; suggestions never auto-run scripts
- [ ] Test argument ambiguity and global-flag interaction
- [ ] Update feature inventory with extension compatibility_class

### CLI / UX

- [ ] Preserve m run as unambiguous escape hatch for reserved names
- [ ] Implement optional local executable lookup behind explicit policy flag
- [ ] Test no-project and malformed-manifest behavior
- [ ] Ensure mx dispatch unaffected unless explicitly shared

### Tests & fixtures

- [ ] Implement direct argument forwarding without requiring `--` when unambiguous
- [ ] Add shell completion for dynamic manifest scripts
- [ ] Record intentional divergence from Nub in conformance inventory

### Docs & observability

- [ ] Implement reserved-name and built-in collision diagnostics
- [ ] Document one-letter m shell alias conflicts
- [ ] Benchmark dispatch overhead on cold vs warm manifest reads

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: m dev runs dev script when not a built-in command
- [ ] Acceptance: m add runs built-in add; m run add runs add script if present
- [ ] Acceptance: Misspelled commands show suggestions without executing
- [ ] Acceptance: Dispatch precedence matches documented charter order
- [ ] Acceptance: No fuzzy script execution occurs
- [ ] Fixture ready: `fixtures/dispatch/collision-matrix — built-in vs script exhaust`
- [ ] Fixture ready: `fixtures/dispatch/arg-ambiguity — global flags vs script args`
- [ ] Fixture ready: `fixtures/dispatch/no-project — missing package.json`
- [ ] Fixture ready: `fixtures/dispatch/malformed-manifest — parse error UX`
- [ ] Fixture ready: `fixtures/dispatch/suggestions — typo ranking golden`


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
| Direct m <script> shortcuts | Nub requires nub run | Intentional Mew extension | 0042 |
| Dispatch precedence | N/A | built-in, alias, script, local bin, suggest, error | 0042 |
| Collision diagnostics | Nub explicit run | m run add for script named add | 0042 |
| Script suggestions | npm-style | Levenshtein ranking, no fuzzy execution | 0042 |

## Go Package Map

**Packages / paths:**

- `internal/cli`
- `internal/runner`
- `internal/manifest`
- `cmd/m (dispatch)`

**Forbidden import edges:**

- internal/runtime
- internal/resolver
- internal/linker

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  argv[Argv] --> builtin[BuiltInCmd]
  builtin --> alias[BuiltInAlias]
  alias --> script[ExactScript]
  script --> localbin[LocalBinPolicy]
  localbin --> suggest[Suggestions]
  suggest --> err[Error]
```

## Commands and Flags

| Command | Flags | Notes |
|---|---|---|
| `m dev` | script args without `--` when unambiguous | Direct script shortcut |
| `m start` | same | Common lifecycle script |
| `m build -- --mode production` | `--` when needed | Arg forwarding rules |
| `m run add` | explicit run | Script colliding with built-in pm verb |

Precedence: built-in > alias > exact script > optional local bin > suggestions > error.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Dispatch precedence table | Documented in charter and user docs |
| Collision matrix test corpus | Exhaustive built-in vs script cases |

## Concrete Test Fixtures

- `fixtures/dispatch/collision-matrix — built-in vs script exhaust`
- `fixtures/dispatch/arg-ambiguity — global flags vs script args`
- `fixtures/dispatch/no-project — missing package.json`
- `fixtures/dispatch/malformed-manifest — parse error UX`
- `fixtures/dispatch/suggestions — typo ranking golden`

## Acceptance Scenarios

1. m dev runs dev script when not a built-in command
2. m add runs built-in add; m run add runs add script if present
3. Misspelled commands show suggestions without executing
4. Dispatch precedence matches documented charter order
5. No fuzzy script execution occurs

## Nub Conformance Targets

- Nub requires nub run <script> | divergence (documented extension)
- npm/pnpm script naming | reference for collision tests | defer

## Open Decisions

- Whether bare m opens interactive script picker in v1
- Opt-in flag for local bin lookup in direct dispatch

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
