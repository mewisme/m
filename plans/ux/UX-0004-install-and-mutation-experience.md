---
name: UX-0004 Install and Mutation Experience
overview: Add polished rich and plain progress for install-family operations, model resolve/fetch/link/lifecycle/validate/commit/rollback phases semantically, produce concise package and lifecycle summaries, and preserve transaction correctness, rollback evidence, CI logs, and structured output.
todos:
  - id: p1-events
    content: Define typed install and mutation operation events with stable IDs, metrics, and phase state
    status: pending
  - id: p2-live-controller
    content: Implement inline live rendering using Bubble Tea/Bubbles only when capabilities permit
    status: pending
  - id: p3-plain
    content: Implement deterministic append-only phase logging for CI, redirects, accessibility, and fallback
    status: pending
  - id: p4-mutations
    content: Migrate install, add, remove, update, ci, dedupe, prune, recovery, and rollback presentation
    status: pending
  - id: p5-summaries
    content: Add package deltas, cache/fetch metrics, lifecycle warnings, transaction outcome, and duration summaries
    status: pending
  - id: p6-failure
    content: Coordinate failures, rollback, cancellation, broken pipe, renderer failure, and cleanup
    status: pending
  - id: p7-cert
    content: Add TTY, non-TTY, transaction, stream, race, cancellation, Windows, and structured-output tests
    status: pending
isProject: false
---

# UX-0004 — Install and Mutation Experience

## Goal

Modernize the user experience of install-family operations without changing resolution, fetch, linking, lifecycle, transaction, recovery, lockfile, or store semantics.

The UI must make long-running operations understandable while preserving the installer as the source of truth. The renderer observes typed events; it never drives transaction state.

## Commands in scope

Confirm exact current command surface before implementation. Expected families:

```text
m install / m i
m add
m remove / m rm
m update
m ci
m dedupe
m prune
m recover
m rollback
m snapshot restore where mutation progress is shared
```

## User-visible deliverables

### Interactive install

```text
  Resolving dependencies
  Fetching packages        31/42
  Linking dependencies     pending
  Running approved scripts pending
  Validating installation  pending
  Committing changes       pending
```

Final frame:

```text
✓ Installed 126 packages in 1.8s

  Added       4
  Updated     2
  Removed     1
  Reused      119
  Downloaded  7

! 3 lifecycle scripts were blocked
→ Run `m builds` to review them
```

### Plain/CI install

```text
resolve started
resolve completed duration=118ms
fetch started packages=42
fetch completed downloaded=7 reused=35 duration=620ms
link completed packages=126 duration=410ms
lifecycle completed ran=2 blocked=3
validate completed
commit completed
installed added=4 updated=2 removed=1 duration=1.8s
warning lifecycle-blocked count=3
```

## Explicitly out of scope

- Changing install ordering or concurrency.
- Changing transaction commit/rollback semantics.
- Pretending an operation is atomic when existing transaction evidence does not prove it.
- Full-screen dashboard or alternate screen.
- Public-registry-dependent UI tests.
- Interactive package search.
- Hiding lifecycle trust or policy failures.
- Hard timing promises.

## Rendering strategy

### Rich mode

Use inline Bubble Tea only for live state that cannot be represented cleanly with static Lip Gloss output.

Potential dependencies:

```go
tea "charm.land/bubbletea/v2"
"charm.land/bubbles/v2/spinner"
"charm.land/bubbles/v2/progress"
```

The renderer:

- writes live status to stderr;
- does not use alternate screen;
- does not own stdout;
- does not own stdin unless no child process or prompt needs it;
- leaves a concise final frame;
- stops and restores terminal state on every exit path.

### Plain mode

Append-only semantic phase lines. No spinner, cursor movement, repaint, or ANSI.

### Structured modes

JSON and NDJSON continue through existing reporters and versioned event schemas. Live renderer is not created.

## Install operation model

```go
type InstallPhase string

const (
    PhaseResolve   InstallPhase = "resolve"
    PhaseFetch     InstallPhase = "fetch"
    PhaseLink      InstallPhase = "link"
    PhaseLifecycle InstallPhase = "lifecycle"
    PhaseValidate  InstallPhase = "validate"
    PhaseCommit    InstallPhase = "commit"
    PhaseRollback  InstallPhase = "rollback"
    PhaseCleanup   InstallPhase = "cleanup"
)
```

Each phase has a stable event lifecycle:

```text
started -> progress* -> completed
started -> failed
started -> cancelled
started -> skipped, only when product semantics explicitly mark it skipped
```

The renderer must not infer completion because the next phase started.

## Event contracts

### Phase started

```go
type InstallPhaseStartedEvent struct {
    V       int
    OpID    string
    TxID    *string
    Phase   string
    Label   string
    Total   *int64
    Unit    string
}
```

### Phase progress

```go
type InstallPhaseProgressEvent struct {
    V         int
    OpID      string
    Phase     string
    Completed int64
    Total     *int64
    Bytes     int64
    Detail    string
}
```

### Phase completed

```go
type InstallPhaseCompletedEvent struct {
    V          int
    OpID       string
    Phase      string
    DurationMs int64
    Metrics    InstallMetrics
}
```

### Transaction outcome

```go
type MutationOutcomeEvent struct {
    V                         int
    OpID                      string
    TxID                      *string
    Committed                 bool
    RolledBack                bool
    RecoveryRequired          bool
    CleanupIncomplete         bool
    StoreMaintenanceRequired  bool
}
```

Rules:

- IDs are safe/redacted and optional in default human output.
- Durations are supplied by operation instrumentation.
- Product state determines transaction outcome.
- Event ordering follows actual phase transitions.
- Machine schemas are versioned.

## Live model

```go
type InstallModel struct {
    Mode        Mode
    Width       int
    Phases      []PhaseView
    Active      string
    Summary     *MutationSummary
    Warning     []Notice
    Failed      *ErrorView
    Cancelled   bool
    Closed      bool
}
```

### Rendering rules

- Show current phase and a small amount of context.
- Completed phases may collapse to checkmarked lines.
- Pending phases remain visible only when this improves orientation; avoid a tall dashboard.
- Show counts only when authoritative totals exist.
- Indeterminate operations use spinner, not fake percentages.
- Do not display package-by-package fetch spam in default mode.
- Do not expose cache/store paths in default mode.
- Do not display transaction IDs unless debug mode is active.

## Phase-specific UX

### Resolve

Possible metrics:

```text
importers
packages
peer contexts
optional packages skipped
workspace packages
```

Do not print every resolver decision; link to `m explain` or debug traces.

### Fetch

Possible metrics:

```text
packages total
metadata hits
blob hits
artifacts downloaded
bytes downloaded
packages reused
```

Progress bar is allowed only when total is known and stable.

### Link

Possible metrics:

```text
packages linked
bins created
hardlinks/reflinks/copies/symlinks/junctions
layout mode
```

Default summary should not dump every placement strategy.

### Lifecycle

Possible metrics:

```text
scripts approved
scripts executed
scripts cached
scripts blocked
scripts failed
```

Security-relevant blocked/failed state remains visible.

### Validate

Show concise validation state. Detailed integrity failure uses UX-0003 ErrorView.

### Commit

Do not report success before transaction commit is proven.

### Rollback

Show rollback only after failure/cancellation triggers it. Final messaging must reflect actual outcome:

```text
! Installation failed; project changes were rolled back
```

or:

```text
× Installation failed
  Recovery is required before the next mutation.
  Code  ERR_M_TRANSACTION
```

Do not use “no changes were committed” unless transaction state proves it.

## Mutation summaries

```go
type MutationSummary struct {
    Operation    string
    Added        []PackageDelta
    Updated      []PackageDelta
    Removed      []PackageDelta
    Reused       int
    Downloaded   int
    ScriptsRun   int
    ScriptsBlock int
    LockChanged  bool
    ManifestChanged bool
    DurationMs   int64
}
```

### `m add`

```text
✓ Added zod 4.0.14

  Dependency  ^4.0.14
  Packages    +1
  Lockfile    updated
  Duration    620ms
```

### `m update`

```text
✓ Updated 3 packages

  react        19.1.0 -> 19.1.1
  vite          7.0.2 -> 7.0.4
  typescript    5.8.2 -> 5.8.3
```

### `m remove`

```text
✓ Removed lodash

  Packages    -1
  Lockfile    updated
```

### No-op

```text
✓ Dependencies are already up to date
```

Do not print a misleading “installed” summary for dry-run or plan-only operations.

## Dry-run UX

Dry-run presentation must say clearly that nothing was committed:

```text
Planned changes

  + zod ^4.0.14

No project files were changed.
```

This statement is permitted because dry-run semantics prove no commit occurred.

## Lifecycle output coordination

When lifecycle child output is streamed:

- suspend or minimize repaint;
- write lifecycle child stdout/stderr according to existing policy;
- resume phase rendering after child completion;
- never insert spinner frames into a partial child line;
- preserve child exit and failure information.

Aggregate mode may show compact script rows only if current runner/lifecycle contracts provide deterministic aggregation.

## Cancellation

On Ctrl+C or context cancellation:

1. stop spinner/timers;
2. restore cursor and terminal state;
3. propagate cancellation to product operation;
4. wait for bounded transaction rollback/cleanup;
5. render final outcome from actual mutation result;
6. return existing cancellation/transaction exit mapping.

Do not close the renderer before final rollback facts are available unless output itself failed.

## Broken pipe

Since progress is on stderr, stdout broken pipes may occur in commands that also emit result data.

Requirements:

- honor existing broken-pipe behavior;
- stop renderer;
- continue or cancel product work according to command contract;
- release transaction resources;
- never leave hidden cursor state;
- avoid secondary error spam.

## Renderer failure policy

Recommended:

- auto rich mode: downgrade once to plain, emit a debug diagnostic, continue product work;
- forced rich mode: return typed output error if failure occurs before mutation starts;
- after mutation starts, presentation failure must not abandon transaction cleanup; product operation and cleanup complete, then error precedence follows UX-0001.

Freeze exact policy before implementation.

## Implementation phases

### Phase 1 — Instrumentation audit

Map current install phase events and metrics. Reuse existing events where possible.

### Phase 2 — Typed phase events

Add missing typed fields without putting presentation strings in domain packages.

### Phase 3 — Plain renderer

Implement append-only phase and summary output first. Certify CI behavior before live UI.

### Phase 4 — Inline live renderer

Add Bubble Tea/Bubbles behind capability and command classification gates.

### Phase 5 — Mutation command summaries

Migrate add/remove/update/ci/dedupe/prune and recovery-related outputs.

### Phase 6 — Failure and cleanup certification

Test rollback, partial commit, cleanup warning, recovery-required, cancellation, and renderer failure paths.

## Tests

### Unit

- phase state machine;
- duplicate/out-of-order event handling;
- progress with known and unknown totals;
- final summary construction;
- dry-run summary;
- rollback message selection;
- no misleading success before commit;
- renderer downgrade.

### Integration

- warm install;
- cold install with local registry fixture;
- no-op install;
- add/remove/update;
- frozen/CI failure;
- lifecycle blocked;
- lifecycle failure;
- commit failure injection;
- rollback success;
- rollback incomplete/recovery required;
- Ctrl+C in each major phase;
- redirected stdout/stderr;
- `NO_COLOR`;
- CI environment;
- Windows console.

### Race

```sh
go test -race ./internal/presentation/... ./internal/diagnostics/... ./internal/app/... ./internal/transaction/...
```

### Golden

Golden only final static frames and plain output. Do not freeze spinner frames or real durations. Normalize:

```text
duration
transaction ID
temp path
cache digest
platform path separators
```

## Acceptance criteria

- Interactive install has useful inline progress.
- Plain/CI output is deterministic and append-only.
- Progress stays on stderr.
- Machine output remains valid.
- Transaction success is never reported early.
- Rollback/recovery messages reflect real outcome.
- Lifecycle trust warnings remain visible.
- Cancellation restores terminal state and preserves mutation safety.
- Renderer failure cannot strand locks or transaction journals.
- No public network is needed by tests.

## Risks

| Risk | Mitigation |
|---|---|
| Progress events influence transaction timing | asynchronous observer with bounded channel and non-blocking/defined backpressure policy |
| Live repaint corrupts lifecycle output | suspend/minimize around child output and partial-line tests |
| UI claims rollback succeeded incorrectly | render only from typed outcome result |
| Renderer crash strands mutation | controller cleanup independent of transaction cleanup |
| Unknown totals produce fake precision | indeterminate spinner only |
| CI logs become noisy | plain compact phase output and log-level control |

## Estimated effort

**20–32 focused engineering hours**, excluding installer defects uncovered by new cancellation and stream tests.
