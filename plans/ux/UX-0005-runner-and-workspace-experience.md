---
name: UX-0005 Runner and Workspace Experience
overview: Modernize m run, m exec, mx, workspace execution, snapshot/capsule execution, and process-status presentation while preserving child stdin/stdout/stderr, signals, exit codes, interactive applications, execution leases, and structured runner events.
todos:
  - id: p1-contract
    content: Audit runner, workspace, envexec, process supervisor, child stream, signal, and reporter contracts
    status: pending
  - id: p2-preparation
    content: Add concise environment preparation and source/cache/integrity presentation
    status: pending
  - id: p3-workspace
    content: Implement aggregate task rows and append-only stream mode with deterministic summaries
    status: pending
  - id: p4-child-control
    content: Add live-render suspend/resume around interactive children and terminal ownership transfer
    status: pending
  - id: p5-signals
    content: Preserve cancellation, signal forwarding, process-tree cleanup, and exit mapping
    status: pending
  - id: p6-modes
    content: Certify single task, workspace stream, workspace aggregate, JSON, NDJSON, no-color, CI, and accessible modes
    status: pending
  - id: p7-tests
    content: Add PTY/TTY, partial-line, binary-output, Windows shim, snapshot/capsule, mx consent, and race tests
    status: pending
isProject: false
---

# UX-0005 — Runner and Workspace Experience

## Goal

Improve the presentation of script and executable execution without turning Mew into a terminal proxy that changes child behavior.

Runner UX is the highest-risk area of the modernization program because the child process may own stdin, emit arbitrary stdout/stderr, use cursor control, enter raw mode, launch a full-screen TUI, or depend on exact signal behavior.

The plan therefore prioritizes **terminal ownership and stream correctness over visual progress**.

## Commands in scope

Confirm current grammar and shipped status:

```text
m run <script> [-- <args...>]
m run -r / workspace forms
m exec <bin> [args...]
m exec --package <owner> <bin> [args...]
m exec --snapshot <id> <bin> [args...]
m exec --capsule <path> <bin> [args...]
mx <package-spec> [args...]
mx -p <package>... <bin> [args...]
m env inspect ...
direct script/bin dispatch when enabled
```

## Non-negotiable contracts

- Child stdout remains stdout.
- Child stderr remains stderr.
- Child stdin remains available when the child contract requires it.
- Mew diagnostics and progress use stderr.
- Exit code and signal mapping remain unchanged.
- Environment preparation events remain versioned and redacted.
- Execution leases and cleanup are not controlled by presentation.
- Rich UI must suspend before an interactive child owns the terminal.
- No alternate screen around child execution.
- No host-PATH or command-resolution behavior changes.

## User-visible outcomes

### Single local executable

Before launch, when useful:

```text
→ Running eslint 9.12.0
  Source   project
  Package  eslint
```

Then raw child output.

Optional completion summary on stderr:

```text
✓ eslint completed in 1.2s
```

The summary is omitted when:

- `--no-summary`;
- child is an interactive/full-screen application;
- output mode is structured/silent;
- command contract says no wrapper output;
- summary would disrupt terminal state.

### `mx` cold preparation

```text
  Resolving eslint@latest
  Checking consent
  Fetching package
  Preparing environment

→ Running eslint 9.12.0
```

### `mx` warm cache

```text
→ Running eslint 9.12.0
  Environment  warm cache
```

### Snapshot/capsule

```text
→ Running eslint
  Source       snapshot 000014
  Environment  verified
  Network      disabled
```

Do not expose absolute cache or staging paths in default output.

## Execution presentation states

```go
type ExecutionStage string

const (
    StagePlan        ExecutionStage = "plan"
    StageConsent     ExecutionStage = "consent"
    StageFetch       ExecutionStage = "fetch"
    StageMaterialize ExecutionStage = "materialize"
    StageVerify      ExecutionStage = "verify"
    StageLease       ExecutionStage = "lease"
    StageResolveBin  ExecutionStage = "resolve-bin"
    StageLaunch      ExecutionStage = "launch"
    StageRunning     ExecutionStage = "running"
    StageCleanup     ExecutionStage = "cleanup"
)
```

Only show stages that materially help the user. Warm project execution should not print a long checklist.

## Environment prepared event

Reuse the frozen runner event contract. Presentation may map fields such as:

```text
source
identity digest, debug only
cache state
network used
preparation duration
verification state if available in authoritative schema
```

Rules:

- Human mode renders safe labels.
- JSON/NDJSON uses the frozen schema.
- No absolute paths or credentials.
- Event emission failure follows runner contract and cleanup policy.
- Presentation does not synthesize an event when preparation did not complete.

## Terminal ownership state machine

```text
Mew owns terminal
  -> prepare status
  -> stop live repaint
  -> restore cursor
  -> hand stdin/stdout/stderr to child
  -> wait/forward signals
  -> child exits
  -> reacquire presentation ownership if safe
  -> render optional summary
```

### Suspend requirements

Before child start:

- stop spinner timers;
- flush final pre-launch status;
- restore cursor visibility;
- disable renderer key handling;
- ensure no goroutine writes cursor commands;
- release any terminal raw mode owned by presentation.

### Resume requirements

After child exit:

- only resume if terminal is still usable;
- do not clear child output;
- append summary below child output;
- handle missing final newline safely;
- never rewrite child screen contents.

## Interactive child detection

Do not rely on command name lists alone.

Use explicit execution metadata where possible:

```go
type TerminalIntent string

const (
    TerminalAuto        TerminalIntent = "auto"
    TerminalInteractive TerminalIntent = "interactive"
    TerminalStream      TerminalIntent = "stream"
)
```

Potential sources:

- command options;
- stdin/stdout/stderr TTY state;
- known child launch mode;
- user override;
- script/workspace output mode.

Auto should be conservative: when uncertain and all streams are terminals, suspend rich UI and let the child own them.

## Single-task output policy

For one script/bin:

- no package prefix on raw output;
- no progress repaint while child runs;
- no line buffering introduced unless current process layer already does it;
- partial lines preserved;
- binary output not decoded by presentation;
- stderr remains stderr;
- Mew summary written only after child completion and only to stderr.

## Workspace modes

### Aggregate mode

Use compact task rows when child output is intentionally aggregated by existing runner policy.

```text
apps/api       ✓ test   2.1s
apps/web       ● test   running
packages/ui    × test   1.8s
packages/core  – test   not run
```

Final:

```text
× 1 of 4 tasks failed

  Failed
  packages/ui  test  exit 1

  Completed   2
  Failed      1
  Not run     1
  Duration    3.4s
```

### Stream mode

Append-only package-prefixed output:

```text
[apps/api] starting test
[apps/web] starting test
[apps/api] PASS
[apps/web] PASS
```

No live task table repaint in stream mode.

### Inherit mode

If a task directly inherits terminal streams, rich workspace rendering must be disabled or suspended according to existing semantics.

## Workspace event model

Use existing typed task and summary events. Extend only when necessary.

Required task states:

```text
queued
ready
running
completed
failed
cancelled
skipped
not-run
```

Presentation must not reinterpret scheduler ordering or failure selection.

### Ordering

- Task rows sorted by existing deterministic scheduler/report order.
- Completion timing does not redefine order.
- Final failed-task list follows canonical task order, not map iteration.
- Secondary failures remain visible according to runner contract.

## Partial lines and stream framing

Tests must cover:

- stdout partial line;
- stderr partial line;
- interleaved complete lines;
- no final newline;
- carriage returns;
- ANSI emitted by child;
- invalid UTF-8/binary bytes;
- high-volume output;
- cancellation mid-line.

Presentation must not insert a prefix into the middle of a partial line unless existing workspace stream framing owns that behavior.

## Signals and cancellation

The UI must not consume Ctrl+C in a way that changes ProcessSupervisor semantics.

Requirements:

- Bubble Tea key handling is inactive while child owns terminal.
- Parent interrupt reaches existing signal/cancellation path.
- Child and process-tree behavior remains certified by 0046.
- Summary uses actual child exit result.
- Signal exit codes remain platform-specific according to current contract.
- Renderer cleanup is bounded and cannot delay signal forwarding materially.

## `mx` consent coordination

Consent is handled by UX-0006 prompt adapter, but runner presentation coordinates stages.

Rules:

- No artifact fetch status before consent if product policy forbids artifact fetch before approval.
- Metadata resolution may be shown only according to existing security contract.
- Non-TTY failure is concise and actionable.
- Approved transition is explicit enough for auditability without exposing secrets.
- Structured modes receive versioned consent/status events only if already part of the product contract.

## Snapshot and capsule safety messaging

Human labels may state:

```text
Network disabled
Integrity verified
Source snapshot/capsule
```

Only when authoritative provider state proves these facts.

Do not claim authenticity when only integrity is verified.

## Direct-dispatch presentation

- Do not make experimental dispatch shortcuts look more canonical than `m run` or `m exec`.
- When an ambiguity/error occurs, show the deterministic precedence and canonical command to use.
- Gate-off behavior must not emit misleading lookup progress.
- Integrity errors stop dispatch and use structured ErrorView.

## Implementation phases

### Phase 1 — Contract audit

Inventory all execution/output modes, stream ownership, signal paths, and interactive-child behaviors.

### Phase 2 — Preparation presentation

Map project, DLX, snapshot, and capsule preparation events into concise human status.

### Phase 3 — Single-task handoff

Implement suspend/resume and raw child stream certification.

### Phase 4 — Workspace aggregate and stream renderers

Build separate implementations; do not combine them through condition-heavy rendering.

### Phase 5 — Signals, cancellation, and cleanup

Certify interrupt, deadline, child failure, reporter failure, lease release, and renderer cleanup.

### Phase 6 — Platform and PTY evidence

Run on Linux, macOS, and Windows, with PTY/console tests where tooling supports it.

## Tests

### Unit

- execution stage reducer;
- warm vs cold concise output;
- source/cache labels;
- terminal ownership state machine;
- task row ordering;
- summary counts;
- no summary policy;
- partial-line framing.

### Integration

- local bin success/failure;
- script success/failure;
- child stdin echo fixture;
- stdout/stderr fixture;
- partial-line fixture;
- ANSI child fixture;
- interactive child fixture;
- workspace stream and aggregate;
- Ctrl+C;
- deadline;
- Windows `.cmd` shim;
- `mx` consent denied/approved/warm/offline;
- snapshot/capsule no-network execution;
- broken pipe;
- JSON/NDJSON reporter.

### PTY/console

- cursor restored;
- no spinner while child runs;
- child raw mode works;
- resize does not corrupt final summary;
- Ctrl+C reaches child/supervisor;
- full-screen child leaves usable terminal.

### Race

```sh
go test -race ./internal/presentation/... ./internal/runner/... ./internal/process/... ./internal/diagnostics/...
```

## Acceptance criteria

- Raw child streams remain correct.
- Interactive children receive terminal ownership.
- Workspace aggregate and stream modes are distinct and predictable.
- Progress stays on stderr.
- Signals and exit codes are unchanged.
- Snapshot/capsule messaging does not overclaim security.
- `mx` consent ordering remains secure.
- Renderer cleanup releases no execution lease or resource incorrectly; product cleanup remains authoritative.
- Structured runner events remain valid.
- Windows shims and console behavior pass.

## Risks

| Risk | Mitigation |
|---|---|
| Bubble Tea consumes child input/signals | suspend before child start and PTY tests |
| Spinner corrupts partial child output | no repaint during single child and framing tests |
| Workspace rows reorder nondeterministically | canonical event ordering and stable sorting |
| Summary changes exit semantics | use existing execution result only |
| Security label overclaims verification | provider-derived typed fields and wording review |
| Full-screen child leaves terminal broken | suspend/raw-mode restoration and console evidence |

## Estimated effort

**24–38 focused engineering hours**, excluding runner/process defects discovered by PTY and Windows tests.
