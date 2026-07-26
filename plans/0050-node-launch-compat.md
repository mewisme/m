# 0050 — Runtime MVP 1 — Node Launch and Compatibility Boundary

## Document Control

| Item | Detail |
|---|---|
| Phase | Runtime / MVP 1 |
| Primary objective | Launch the user-selected stock Node process from Go with predictable argument handling, preload injection, compatibility escape hatches, and embedded runtime assets. |
| Required predecessors | 0046 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Launch the user-selected stock Node process from Go with predictable argument handling, preload injection, compatibility escape hatches, and embedded runtime assets.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0046 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub augments stock Node through `--import`, `--require`, hooks, environment, and N-API; it does not embed or fork Node
- Nub `--node` and compatibility opt-out concepts
- Nub embedded runtime extraction cache

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m app.js
```
```bash
m --node app.js
```
```bash
m node-args -- --trace-warnings app.js
```

## In Scope

- JavaScript entrypoint detection.
- Node binary selection interface for later Node manager.
- Embedded CommonJS and ESM preload assets.
- Argument and V8 flag classification.
- Zero-augmentation escape hatch.
- Runtime-asset digest verification and versioned extraction.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Go owns orchestration; Node owns JavaScript execution.
- Embedded JavaScript remains small, versioned, tested, and generated into release artifacts.
- Never mutate or patch the Node binary.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement file-run dispatch without colliding with built-ins and scripts
- [ ] Inject ESM/CJS preloads through supported Node extension surfaces
- [ ] Add Node version matrix integration tests
- [ ] Benchmark cold asset extraction vs warm cache

### Core logic

- [ ] Implement Node discovery interface for later 0060 integration
- [ ] Implement --node and compatibility opt-out (zero augmentation)
- [ ] Add CJS/ESM entrypoint and argument forwarding fixtures
- [ ] Never mutate or patch the Node binary

### CLI / UX

- [ ] Implement Node argument and V8 flag classification/partitioning
- [ ] Forward signals and exit codes through shared ProcessSupervisor
- [ ] Test runtime asset corruption and re-extraction recovery
- [ ] Leave hooks for transform service without implementing TS yet

### Tests & fixtures

- [ ] Embed CommonJS and ESM preload assets via go:embed
- [ ] Detect JavaScript entrypoints (.js, .mjs, .cjs, later .ts via 0051)
- [ ] Parity-test opt-out mode against plain node invocation

### Docs & observability

- [ ] Extract, hash-verify, and garbage-collect runtime assets on disk
- [ ] Validate runtime asset digests before extraction or use
- [ ] Document augmentation boundary vs stock Node

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: m app.js launches stock Node with embedded preloads when augmentation enabled
- [ ] Acceptance: m --node app.js matches plain node behavior within documented tolerance
- [ ] Acceptance: Corrupted runtime assets rejected and re-extracted safely
- [ ] Acceptance: Signals and exit codes propagate correctly
- [ ] Acceptance: No Node source patching or private libnode embedding
- [ ] Fixture ready: `fixtures/runtime/entrypoints — cjs/esm/mjs argument matrix`
- [ ] Fixture ready: `fixtures/runtime/opt-out — plain Node parity`
- [ ] Fixture ready: `fixtures/runtime/assets-corrupt — digest failure recovery`
- [ ] Fixture ready: `fixtures/runtime/node-matrix — supported Node versions`
- [ ] Fixture ready: `fixtures/runtime/flags — V8 flag partitioning`


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
| Stock Node launch | Nub augments stock Node | No Node fork or embed | 0050 |
| Preload injection | Nub --import/--require | Embedded CJS/ESM assets | 0050 |
| --node escape hatch | Nub compat opt-out | Zero augmentation mode | 0050 |
| Runtime asset cache | Nub extraction cache | Digest-verified embed extract | 0050 |

## Go Package Map

**Packages / paths:**

- `internal/runtime`
- `internal/node`
- `internal/process`
- `runtime/`
- `cmd/m`

**Forbidden import edges:**

- internal/resolver
- internal/linker
- internal/transform (beyond embed bridge)

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  dispatch[FileRunDispatch] --> nodeSel[NodeSelection]
  nodeSel --> assets[RuntimeAssets]
  assets --> launch[NodeLaunch]
  launch --> preload[PreloadInject]
  preload --> child[StockNodeProcess]
```

## Commands and Flags

| Command | Flags | Notes |
|---|---|---|
| `m app.js` | entry detection | JavaScript file run |
| `m --node app.js` | `--node` | Plain stock Node, no augmentation |
| `m node-args -- --trace-warnings app.js` | `node-args` | V8/Node flag partition |

Go orchestrates; stock Node executes JS. Never patch Node binary.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Embedded runtime assets (go:embed) | Versioned preload modules |
| Runtime extraction cache | Content-addressed asset store |

## Concrete Test Fixtures

- `fixtures/runtime/entrypoints — cjs/esm/mjs argument matrix`
- `fixtures/runtime/opt-out — plain Node parity`
- `fixtures/runtime/assets-corrupt — digest failure recovery`
- `fixtures/runtime/node-matrix — supported Node versions`
- `fixtures/runtime/flags — V8 flag partitioning`

## Acceptance Scenarios

1. m app.js launches stock Node with embedded preloads when augmentation enabled
2. m --node app.js matches plain node behavior within documented tolerance
3. Corrupted runtime assets rejected and re-extracted safely
4. Signals and exit codes propagate correctly
5. No Node source patching or private libnode embedding

## Nub Conformance Targets

- Nub stock Node augmentation model | parity
- Nub --node opt-out | parity
- Nub embedded runtime cache | parity

## Open Decisions

- Node LTS floor for v1 certification (link 0084/0089)
- Default augmentation on vs opt-in experimental gate

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
