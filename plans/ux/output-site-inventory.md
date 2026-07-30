# UX-0001 output-site inventory

Generated for the CLI presentation contract. Classifies current output paths before
migration to `internal/presentation`.

## Summary by package

| Package | Direct print sites | Primary role |
|---|---:|---|
| `internal/cli` | ~97 | Command results, tables, help, command-local `--json` |
| `internal/diagnostics` | 21 | Human/JSON/NDJSON/silent reporters, redaction |
| `internal/resolver` | 13 | `explain` human output |
| `internal/app` | 5 | Bench phase timing |
| `internal/runner` | few | Child output routing, workspace events |
| `internal/runner/dlx` | 1 | Consent prompt on stderr |
| `internal/features` | few | Feature inventory tables |
| `cmd/m`, `cmd/mx` | 0 | Thin entrypoints |

## Output-site families

### Machine output (stdout)

- Global `--reporter json|ndjson` via `diagnostics.NewReporter`
- Command-local `--json` on: `version`, `install`, `add`, `remove`, `update`, `ci`,
  `dedupe`, `prune`, `lock`, `plan`, `resolve`, `ls`, `outdated`, `diff`, `explain`,
  `audit`, `builds`, `bench`, `conformance`, `doctor`, `project`, `registry`,
  `store`, `fetch`, `features`, `dispatch` introspection, and others
- Child stdout via `diagnostics.ChildOutput` / `PrefixWriter` (human mode)

### Progress and status (stderr)

- `diagnostics.humanReporter.Progress` — phase lines, workspace-task, workspace-summary
- `app` install phase debug via `Reporter.Debug`
- Future: `Operation*` events (types added in UX-0001; emitters deferred to UX-0004)

### Errors (stderr or stdout)

- Human errors: stderr (`humanReporter`, `silentReporter`)
- JSON/NDJSON errors: stdout (`jsonReporter`, `ndjsonReporter`)
- CLI `execute` routes Cobra errors through `rep.Error`

### Prompts

- `internal/runner/dlx/prompt.go` — consent `[y/N]` on stderr
- `trust_cmd` — interactive approval path

### Help

- Cobra-generated help to stdout (`SetOut`)

## Stream ownership

| Content | Stream | Owner today |
|---|---|---|
| Command result / pipeline data | stdout | Command handlers, child stdout |
| JSON/NDJSON reporter events | stdout | `diagnostics` ndjson/json reporters |
| Human progress, workspace status | stderr | `diagnostics` human reporter |
| Human errors (default/silent) | stderr | `diagnostics` |
| JSON reporter errors | stdout | `diagnostics` json reporter |
| Debug lines | stderr | `diagnostics` |
| Child stderr | stderr | runner / human child-output routing |
| Help text | stdout | Cobra |

## Flag compatibility

| Legacy | Unified (`--output`) | Notes |
|---|---|---|
| `--reporter default` | `auto` or `plain` | `default`/`human` map to auto resolution |
| `--reporter json` | `json` | Diagnostics error document |
| `--reporter ndjson` | `ndjson` | Event stream |
| `--reporter silent` | `silent` | Suppress progress |
| `MEW_LOG_FORMAT` | `MEW_OUTPUT` | Env precedence below CLI |
| `--color auto\|always\|never` | same | Plus `NO_COLOR`, `CLICOLOR*` |
| Command `--json` | — | **Result encoding**; conflicts with `--output=json\|ndjson` |

Precedence (single resolver in `internal/presentation`):

1. Conflicting explicit structured flags → usage error
2. `--output`
3. `--reporter` / `MEW_LOG_FORMAT`
4. `MEW_OUTPUT`
5. `ui.output` config
6. Command-mandated structured mode (reserved)
7. `auto` → `rich` or `plain` from capabilities
8. Unknown mode → usage error

## Command classification

| Class | Examples |
|---|---|
| Static | `version`, `features`, `config list`, `help` |
| Progress | `install`, `add`, `update`, `fetch`, `link` |
| Child-process | `run`, `exec`, `mx` dispatch |
| Prompt | `trust`, DLX consent |
| Structured-only | `conformance run --json`, `bench --json` |

## Schema inventory

| Schema | Path |
|---|---|
| Progress event v1 | `schemas/diagnostics/progress-event.schema.json` |
| Error document v1 | `schemas/diagnostics/error.schema.json` |
| Operation started v1 | `schemas/diagnostics/operation-started.schema.json` |
| Operation progress v1 | `schemas/diagnostics/operation-progress.schema.json` |
| Operation completed v1 | `schemas/diagnostics/operation-completed.schema.json` |
| Notice v1 | `schemas/diagnostics/notice.schema.json` |

## Windows and TTY notes

- TTY detection uses `os.File` `ModeCharDevice` (works on Windows consoles).
- `CI`, `TERM`, `NO_COLOR` honored by `internal/presentation` capability snapshot.
- Redirected pipes are non-TTY → plain append-only output, no ANSI on errors when
  color is auto.

## Runner and workspace contracts (UX-0005)

### Single-child passthrough

| Path | Streams | Notes |
|---|---|---|
| `internal/runner/run.go` `DefaultRunner.Run` | raw stdin/stdout/stderr | No package prefix; `process.ExecSupervisor` Start/Wait |
| `internal/runner/exec.go` `Exec` | raw stdin/stdout/stderr | Local package bins; same supervisor |
| `internal/runner/envexec` | via `DefaultExecAdapter` → `runner.Exec` | Prepared event before launch; lease release on reporter failure |

Diagnostics and prep banners use **stderr only**. Child stdout/stderr stay byte-preserving.

### Workspace stream vs aggregate

| Mode | Framing | Status UI |
|---|---|---|
| `stream` | `PrefixWriter` (`internal/runner/mux.go`) prefixes complete lines with `[pkg] ` | Append-only task status on stderr; no live table |
| `aggregate` | `AggregateBuffer` collects per task; dump after scheduler | Append-only status during run; final summary after `WorkspaceSummary`; child blocks remain authoritative |
| inherit / interactive TTY task | raw TTYs to child | Force Suspend; disable aggregate live status |

`PrefixWriter` must not change: partial lines, CR/LF, invalid UTF-8, and truncation markers stay as-is.

### Signal path

- Parent cancel: `signal.NotifyContext` / command context → `ProcessSupervisor.Wait`
- Live install Bubble Tea uses `WithInput(nil)` + `WithoutSignals()` so UI never swallows Ctrl+C
- Runner must `Suspend` before child Start so any live progress is inactive
- Renderer `Close`/`Resume` bounded (≤2s cleanup context); must not delay signal forwarding

### Environment prepared + lease

- Event: frozen `EnvironmentPreparedEvent` (`prepared_event.go`)
- Human: presentation hook `OnEnvironmentPrepared` → `ExecutionPrepView` (safe labels only)
- NDJSON/JSON: unchanged schema emission
- Reporter failure after lease acquire → release lease + cleanup (existing tests)

### SuspendUI call sites

| Site | Status |
|---|---|
| Install lifecycle scripts (`app/install_lifecycle.go`) | Wired (UX-0004) |
| `app.Context.SuspendUI` / `ResumeUI` from CLI controller | Wired |
| `runner.Run` / `runner.Exec` via optional Suspend/Resume on options | UX-0005 |
| envexec before child Exec | UX-0005 |
| Workspace inherit/interactive path | UX-0005 when task owns raw TTYs |

### TerminalIntent

`auto|interactive|stream` in presentation: `auto` suspends when stdin+stdout+stderr are TTYs; `interactive` always suspends; `stream` does not start live progress for the command.

### Safe labels (human only)

Allowed: `project`, `warm cache`, `snapshot <short-id>`, `capsule`, `dlx`. Never absolute cache/staging paths. Digests debug-only. Claim `Network disabled` / `Integrity verified` only when envexec policy/source proves it.

## Migration status

- UX-0001: contract, resolver, controller, event types, inventory (this document)
- UX-0003: ErrorView human errors, grouped help, static command migrations
  (doctor, ls, outdated, plan, history, snapshot, cache/store status, audit,
  policy, verify, builds, pack, capsule); `writeStaticOut` bridge; archcheck
  allowlist for pending commands
- UX-0004: install `Operation*` emitters; plain append-only progress; Bubble Tea
  live install renderer; mutation summaries for install/add/remove/update/ci/
  dedupe/prune/plan/snapshot/capsule; `--no-summary` honored
- UX-0005: runner/workspace prep views, Suspend around child, workspace aggregate
  status, TerminalIntent, completion summary (see contracts above)
