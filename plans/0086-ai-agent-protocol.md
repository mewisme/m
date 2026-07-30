# 0086 — Cross-Cutting — AI Agent Implementation Protocol

## Document Control

| Item | Detail |
|---|---|
| Phase | Cross-Cutting |
| Primary objective | Give coding agents a deterministic workflow for implementing MVPs without losing architectural intent, compatibility context, or verification evidence. |
| Required predecessors | 0004, 0008, 0009 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Give coding agents a deterministic workflow for implementing MVPs without losing architectural intent, compatibility context, or verification evidence.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0004 before starting this MVP.
- Complete and merge 0008 before starting this MVP.
- Complete and merge 0009 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub AGENTS.md convention and repository guidance

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

No new command is required in this MVP.

## In Scope

- Required reading order.
- Task scoping and predecessor checks.
- Behavior-first research.
- Small pull request policy.
- Test and benchmark evidence.
- Persistent-format safeguards.
- Security escalation triggers.
- Handoff and status reporting.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Agents may propose ADRs but must not silently decide irreversible compatibility or format policy.
- Agents must never claim completion without running documented gates.
- Generated code and fixtures must identify their generator.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Required reading order for agents
- [ ] Persistent-format safeguards
- [ ] Thread file schema status values
- [ ] Clean-home test reminder

### Core logic

- [ ] Predecessor checks before coding
- [ ] Security escalation triggers
- [ ] Forbid secrets in threads
- [ ] Align with AGENTS.md non-negotiables

### CLI / UX

- [ ] Behavior-first research rules
- [ ] Handoff status reporting
- [ ] Link to plan files and inventory updates
- [ ] Document when to stop for human decisions

### Tests & fixtures

- [ ] Small PR policy
- [ ] Evidence template
- [ ] No force-push/default-branch rules reminder

### Docs & observability

- [ ] Test and benchmark evidence requirements
- [ ] Review checklist
- [ ] Windows verification reminder

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Agents have deterministic workflow doc
- [ ] Acceptance: Evidence template lists 6 handoff items
- [ ] Acceptance: Human-owned decisions called out
- [ ] Fixture ready: `docs/agents/evidence-template.md`
- [ ] Fixture ready: `docs/agents/thread-template.md`


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
| AI agent protocol | Nub AGENTS | Deterministic MVP workflow | 0086 |

## Go Package Map

**Packages / paths:**

- `AGENTS.md`
- `docs/agents/`
- `.claude/skills/`
- `.agents/`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  read[RequiredReading] --> scope[ScopeCheck] --> impl[SmallPRs] --> evidence[Evidence] --> handoff[Handoff]
```

## Commands and Flags

N/A — agent process. Threads under `.agents/threads/`.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Evidence template | PR/agent handoff |
| Thread template | Multi-turn state |

## Concrete Test Fixtures

- `docs/agents/evidence-template.md`
- `docs/agents/thread-template.md`

## Acceptance Scenarios

1. Agents have deterministic workflow doc
2. Evidence template lists 6 handoff items
3. Human-owned decisions called out

## Nub Conformance Targets

- Agent workflow | extension

## Open Decisions

- Whether to require thread files for every MVP

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
