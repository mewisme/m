# 0051 — Runtime MVP 2 — Go Transform Service and TypeScript Execution

## Document Control

| Item | Detail |
|---|---|
| Phase | Runtime / MVP 2 |
| Primary objective | Execute TypeScript through stock Node using a Go-native transform pipeline and a small embedded Node loader bridge. |
| Required predecessors | 0050 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Execute TypeScript through stock Node using a Go-native transform pipeline and a small embedded Node loader bridge.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0050 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub OXC N-API transform addon and loader hooks
- Nub zero-config TS, tsconfig discovery, and transpile cache

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m app.ts
```
```bash
m src/index.mts
```
```bash
m src/tool.cts
```

## In Scope

- TS, MTS, CTS, TSX parsing path foundations.
- tsconfig discovery, extends chain, compiler-option subset, path mapping handoff, and source-map generation.
- Go transform service lifecycle and local IPC.
- Content-addressed transpile cache.
- Fallback behavior for unsupported syntax/options.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Prefer an embedded Go transform library such as esbuild after parity evaluation.
- Synchronous Node hooks require a bounded low-latency local protocol or an alternate precompile strategy; complete the spike before freezing the protocol.
- Transform output is never treated as trusted executable input beyond the user source it represents.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Benchmark candidate Go transformers (e.g. esbuild) against Nub/OXC corpus
- [ ] Implement tsconfig parser, extends chain, and normalized compiler-option subset
- [ ] Add TS syntax corpus across supported Node versions
- [ ] Keep transform output scoped to user source representation only

### Core logic

- [ ] Define transform request/response protocol with version, digest, options, errors, maps
- [ ] Implement path mapping handoff to later resolver (0053)
- [ ] Test IPC corruption, timeout, service crash, and concurrent transforms
- [ ] Spike synchronous loader IPC latency before freezing protocol

### CLI / UX

- [ ] Implement transform service startup, auth token, health check, cancellation
- [ ] Implement content-addressed transpile cache with atomic publication
- [ ] Add source-map stack trace integration tests
- [ ] Use BLAKE3 or reviewed fast digest for cache keys

### Tests & fixtures

- [ ] Implement idle shutdown and crash recovery for transform service
- [ ] Map diagnostics to original source locations
- [ ] Benchmark warm-cache transform latency against budget

### Docs & observability

- [ ] Implement Node loader bridge and format detection for .ts/.mts/.cts
- [ ] Implement fallback for unsupported syntax/options with clear errors
- [ ] Document unsupported-feature report vs Nub/OXC

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: m app.ts executes TypeScript through stock Node without separate tsc step
- [ ] Acceptance: Transform service recovers from crash without wedging Node loader
- [ ] Acceptance: Transpile cache hits produce identical output bytes
- [ ] Acceptance: Diagnostics reference original TypeScript sources
- [ ] Acceptance: Unsupported options fail with actionable messages
- [ ] Fixture ready: `fixtures/transform/ts-corpus — syntax coverage`
- [ ] Fixture ready: `fixtures/transform/tsconfig-extends — discovery chain`
- [ ] Fixture ready: `fixtures/transform/ipc-failure — timeout/crash/corruption`
- [ ] Fixture ready: `fixtures/transform/cache-warm — latency benchmarks`
- [ ] Fixture ready: `fixtures/transform/unsupported — fallback diagnostics`


Required test layers:

- Unit tests for parsing, normalization, deterministic ordering, and error classification.
- Golden tests for manifests, lockfiles, command output, and migration reports.
- Integration tests against local fixture registries and isolated temporary homes.
- Failure-injection tests for network interruption, disk exhaustion, permission errors, process termination, and corrupted cache entries.
- Cross-platform tests for Linux, macOS, and Windows, including path length, case sensitivity, junctions, symlinks, and executable shims.
- Conformance tests comparing intentional compatibility surfaces with the corresponding Nub or package-manager behavior.

## Performance Requirements

- Keep warm transform startup overhead low enough for command-line scripts; define and enforce a measured budget.
- Batch or reuse service connections within one process tree.
- Use BLAKE3 or a reviewed fast digest for cache keys while preserving registry integrity algorithms separately.

All performance claims must be backed by reproducible benchmark commands, machine metadata, cold/warm cache separation, and multiple samples. Performance regressions on critical paths require an explicit waiver.

## Security and Trust Requirements

- Validate all external input and fail closed on malformed or ambiguous data.
- Use least-privilege filesystem access and redact credentials in diagnostics.
- Maintain integrity verification before extraction or execution.

Secrets must never be written to logs, lockfiles, snapshots, telemetry, crash reports, or plan files. Archive extraction, script execution, registry authentication, and path construction must be treated as hostile-input boundaries.

## Risks and Mitigations

- Synchronous loader IPC may be too expensive or deadlock-prone: retain precompile and native-addon alternatives behind an architecture spike.
- Go transformer parity may lag OXC: maintain an unsupported-feature report and optional fallback strategy.
- Decorator metadata may require a specialized transform stage.

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
| Go transform service | Nub OXC N-API | Go-native transform + IPC | 0051 |
| TypeScript execution | Nub zero-config TS | m app.ts without tsc step | 0051 |
| tsconfig discovery | Nub | extends chain and option subset | 0051 |
| Transpile cache | Nub | content-addressed cache | 0051 |

## Go Package Map

**Packages / paths:**

- `internal/transform`
- `internal/runtime`
- `runtime/loader-bridge`
- `cmd/m`

**Forbidden import edges:**

- internal/resolver
- internal/linker

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  entry[m app.ts] --> loader[NodeLoaderBridge]
  loader --> ipc[TransformIPC]
  ipc --> svc[GoTransformService]
  svc --> cache[TranspileCache]
  cache --> node[StockNode]
```

## Commands and Flags

| Command | Flags | Notes |
|---|---|---|
| `m app.ts` | auto format detect | TS/MTS/CTS entry |
| `m src/index.mts` | module type aware | ESM TypeScript |
| `m src/tool.cts` | CJS TypeScript | CTS execution |

Synchronous hooks require bounded low-latency IPC or precompile strategy.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Transform IPC protocol spec | Versioned request/response |
| Transpile cache store | Content-addressed outputs + source maps |

## Concrete Test Fixtures

- `fixtures/transform/ts-corpus — syntax coverage`
- `fixtures/transform/tsconfig-extends — discovery chain`
- `fixtures/transform/ipc-failure — timeout/crash/corruption`
- `fixtures/transform/cache-warm — latency benchmarks`
- `fixtures/transform/unsupported — fallback diagnostics`

## Acceptance Scenarios

1. m app.ts executes TypeScript through stock Node without separate tsc step
2. Transform service recovers from crash without wedging Node loader
3. Transpile cache hits produce identical output bytes
4. Diagnostics reference original TypeScript sources
5. Unsupported options fail with actionable messages

## Nub Conformance Targets

- Nub zero-config TypeScript execution | parity
- Nub tsconfig discovery | parity
- Nub transpile cache behavior | parity

## Open Decisions

- Final Go transformer library choice after benchmark spike
- IPC vs precompile strategy for synchronous hooks

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
