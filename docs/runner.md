# Package script runner (`m run`)

MVP **0040** implements `m run` for a single package root: npm-compatible
script lookup, lifecycle hooks, environment construction, argument forwarding,
process supervision, and reporter progress events.

Workspace orchestration (`-r`, `--filter`, stream/aggregate multiplexing) ships in
**0041** (this document section).

Direct `m <script>` shortcuts (same runner, gated) ship in **0042** — see
[Direct shortcuts](#direct-shortcuts-mew-extension) below.

## Direct shortcuts (Mew extension)

When `runner.direct_scripts.enabled` or `MEW_EXPERIMENTAL_DIRECT_SCRIPTS=1`:

```text
m dev
m build -- --mode production
m -r build
m --filter api... test
```

Uses the same `ScriptInvocation` path as `m run`. Built-ins and aliases always
win (`m add` runs the built-in; `m run add` runs the script). Workspace direct
shortcuts require both direct-script and workspace gates.

Globals before the selector are consumed by Mew; tokens after the selector
forward verbatim to the script (including `--reporter`, `--workspace-concurrency`,
etc. when placed after the script name).

## Workspace orchestration (`m -r run` / `--filter`)

Workspace mode runs when **either**:

- global `-r` / `--recursive` is set, **or**
- one or more global `--filter` patterns are present.

Requires `MEW_EXPERIMENTAL_WORKSPACES=1` or `workspaces.enabled`.

Examples:

```text
m -r run build
m --filter api... run test
m run lint --workspace-concurrency 4
m -r run build --workspace-order parallel --no-workspace-bail
m --filter pkg01 run build --workspace-output aggregate
```

### Flags

| Flag | Default | Meaning |
|---|---|---|
| `--workspace-concurrency N` | `GOMAXPROCS` | Max parallel tasks (`0` = `GOMAXPROCS`, capped by task count) |
| `--workspace-order` | `topological` | Scheduling mode (see below) |
| `--workspace-output` | `stream` | `stream` (prefixed lines) or `aggregate` (blocks after all tasks finish) |
| `--workspace-bail` | `true` | Stop on first failure; `--no-workspace-bail` continues all tasks |

Workspace-only flags without `-r` or `--filter` return `ERR_M_USAGE`.

### Order modes

| Mode | Behavior |
|---|---|
| `topological` | Respect workspace dependency edges; a package runs after its dependencies succeed |
| `reverse-topological` | Dependents before dependencies (within the selected subgraph) |
| `parallel` | Ignore edges for ordering; all selected packages may run concurrently (subject to concurrency cap) |
| `sequential` | One task at a time in stable path-sorted order (`--workspace-concurrency` forced to 1) |

Ready tasks dequeue in stable path order. Cycles in the selected subgraph fail
with `ERR_M_RESOLVE` before any child starts.

### Concurrency

`--workspace-concurrency 0` (default) uses `GOMAXPROCS`, capped by the number of
selected tasks. Negative values are rejected. The scheduler never fans out
unbounded goroutines.

### Failure policies

**Bail (default):** first failure stops scheduling, cancels in-flight children,
returns the triggering child exit code. Never-started tasks are `not-run`.

**Continue (`--no-workspace-bail`):** runs all selected tasks; exit code is the
**earliest failed stable scheduler index** (not max, not completion order).

### `--if-present`

Missing script in a member is `skip` (terminal success) and releases dependents.
Without the flag, a missing script fails that member (`ERR_M_NOT_FOUND`).

### Output modes

**Stream (default):** each complete stdout/stderr line is prefixed with
`[<package-name>] ` as it arrives. Long lines truncate at 1 MiB with
`[mew: output truncated]`.

**Aggregate:** buffers each member stream in memory (256 KiB), then spills to
`<project>/.mew/tmp/workspace-run/<task-key>/` (16 MiB hard cap per stream).
After all tasks finish, buffered output replays with `[<package-name>]` prefixes
(default reporter) or as `child-output` events (JSON/NDJSON).

Structured JSON/NDJSON emits `workspace-task`, `child-output`, and
`workspace-summary` events only — raw child bytes never land on structured stdout.

### Deferred

Changed-only filters and resume metadata are **not** implemented in 0041.

## Command shape

```text
m run <script> [--if-present] [-- <forwarded-args>...]
```

Examples:

```text
m run dev
m run build -- --mode production
m run "/^test:/"
m run test --if-present
```

Global `--reporter` applies (`default`, `silent`, `json`, `ndjson`). See
[`reporters.md`](reporters.md).

## Script lookup

1. **Exact name** — selector must match a `package.json` `scripts` key.
2. **Regex selector** — `/pattern/` syntax (leading and trailing `/`). All
   matching script names run in **lexicographic order** (e.g. `test:a` before
   `test:b`).
3. **Missing script** — `ERR_M_NOT_FOUND` (exit **1**) with an "Available
   scripts" list when any scripts exist.
4. **Invalid regex** — `ERR_M_USAGE` (exit **2**).

Use `m run <script>` to force script execution when a name will later collide
with a built-in (**0042** adds root dispatch without `run`).

## Lifecycle hooks

For primary script `dev`, stages run in order when defined:

```text
predev → dev → postdev
```

- Undefined hook keys are skipped.
- Regex selectors expand each matched primary name independently; matched
  primaries run sequentially in sorted name order.
- **Fail-fast:** the first non-zero exit stops later stages and scripts.
- **`--` forwarding** applies only to the primary stage body (`dev`), not
  `predev` or `postdev`.

## Environment variables

`m run` inherits the host environment (unlike dependency lifecycle scripts
under trust policy in **0021**). Mew sets or overrides:

| Variable | Meaning |
|---|---|
| `INIT_CWD` | Absolute project root at invocation |
| `npm_lifecycle_event` | Current stage name (`predev`, `dev`, …) |
| `npm_lifecycle_script` | Script body from `package.json` (before `--` append) |
| `npm_package_name` | `name` field |
| `npm_package_version` | `version` field |
| `npm_package_json` | Absolute path to `package.json` |
| `PATH` / `Path` | Prepends `node_modules/.bin` for the package |

Working directory for the child is the package directory (project root for a
single-package layout).

## Shell and quoting

- **Unix:** `sh -c <script>` (via `process.ExecSupervisor`).
- **Windows:** `%ComSpec% /c <script>` (typically `cmd.exe`).

Arguments after `--` are appended to the primary script with platform quoting
so a single shell invocation receives them.

## Exit codes

| Situation | Code | Exit |
|---|---|---|
| Success | — | **0** |
| Missing script | `ERR_M_NOT_FOUND` | **1** |
| Missing project / `package.json` | `ERR_M_NOT_FOUND` | **1** |
| Bad regex / usage | `ERR_M_USAGE` | **2** |
| Child non-zero | (none — `ExitHint`) | **child code** |
| Context cancel / interrupt | `ERR_M_CANCELLED` | **130** |

Child exit codes propagate through `apperr.ExitHint`; they are not remapped to
`ERR_M_*` codes.

## `--if-present`

When the selector matches no script (or expands to no hook stages), `m run
--if-present` exits **0** without running anything. Without the flag, the same
case returns `ERR_M_NOT_FOUND`.

## Signals and cancellation

On **Unix**, `SIGINT` / `SIGTERM` on the Mew process cancel the root context,
forward an interrupt to the child process tree, and surface `ERR_M_CANCELLED`
(exit **130**) when cancellation wins the race.

On **Windows**, signal propagation is **best-effort**:

- Mew sends `os.Interrupt` to the child before a tree kill on cancel.
- Interactive Ctrl+C behavior depends on `cmd.exe` / Node signal handling.
- Integration coverage for parent cancel is **Unix-only**; Windows tests skip
  `TestRunSignalCancel`.

Stdin, stdout, and stderr default to the parent TTY streams.

## Reporters

Progress events use `phase: "run"` and `package: <script-stage>` (NDJSON/JSON
via global `--reporter`). Workspace runs add `workspace-task`, `child-output`,
and `workspace-summary` events.

## Fixtures and tests

| Fixture | Covers |
|---|---|
| `fixtures/runner/basic-scripts/` | Hook order, regex multi-run, child exit code |
| `fixtures/runner/env-parity/` | `npm_*` and `INIT_CWD` golden keys |
| `fixtures/runner/shell-quoting/` | `--` argument forwarding |
| `fixtures/runner/signals/` | Parent cancel (Unix integration) |
| `fixtures/workspace-runner/dag-simple/` | Topological readiness chain |
| `fixtures/workspace-runner/cycle/` | Selected subgraph cycle rejection |
| `fixtures/workspace-runner/failure-bail/` | Bail vs continue |
| `fixtures/workspace-runner/large/` | Parallel stream / prefix stress |

Unit: `internal/runner/*_test.go`. Integration: `tests/integration/run_test.go`,
`tests/integration/workspace_run_test.go`, `tests/integration/exec_test.go`. CLI:
`internal/cli/run_cmd_test.go`, `internal/cli/exec_cmd_test.go`.

## Local binary execution (`m exec`)

MVP **0043** executes **local** package binaries without registry access.

```text
m exec <binary> [--package <dependency>] [-- <args>...]
```

### Resolution

1. Select exactly one importer (`--cwd`, or `--filter` when it matches one member).
2. `BinResolver` walks nearest `node_modules/.bin` levels (importer → ancestors).
3. Prefer generation-bound metadata in `node_modules/.mew/bins.v1.json`.
4. `PathValidator` + `PlatformLaunchBuilder` → `ProcessSupervisor`.

### Direct dispatch (gated)

When `runner.exec.direct_dispatch.enabled` or
`MEW_EXPERIMENTAL_EXEC_DIRECT_DISPATCH=1`, step 4 of root dispatch may run
`m <binary>` with **verified metadata only** (`OwnershipVerified=true`). Clean
miss falls through to suggestions; ambiguity and integrity errors stop dispatch.

Explicit `m exec` may use a validated unowned compatibility shim when metadata
is absent (never for direct dispatch).

### LaunchSpec and PnP trust boundary

`PlatformLaunchBuilder` converts a validated `BinCandidate` into a `LaunchSpec`
(`LaunchDirect`, `LaunchCmd`, `LaunchNode`) before `ProcessSupervisor` spawn.
Windows `.cmd` shims always launch via validated absolute `ComSpec`; Unix Node
shebangs use trusted absolute stock Node from Mew discovery (never PATH search).

PnP resolution uses a bounded helper subprocess with trusted absolute Node only
(no recursive `m exec` / bin lookup for `node`). Loading `.pnp.cjs` executes
project-controlled JavaScript under reduced I/O limits; Mew registry APIs are
not exposed (project code may still network independently).

### `m exec` vs `mx`

| | `m exec` | `mx` |
|---|---|---|
| Scope | Project-local bins | Local-first (Mode A unversioned), then isolated cache environment |
| Registry metadata | Never | Remote path only; allowed before consent |
| Registry artifacts | Never | Remote path only after consent or `--yes` |
| Project mutation | None | Never mutates user project, lockfile, or manifest |
| Lifecycle scripts | Project policy | Ephemeral policy only; fetch consent does not approve scripts |
| Consent | N/A | TTY prompt or `--yes`; non-TTY fails closed without `--yes` |
| Cache | N/A | Versioned environments under `<cache>/mx/exec/<digest>/` |
| Offline | N/A | Request index + warm environment + prior consent |

**mx argv**

- Mode A: `mx <package-spec> [args…]` — bin inferred from resolved package.
- Mode B: `mx -p <spec>… <command> [args…]` — explicit command; skips local-first.
- Reserved before DLX: `version`, `completion`, `cache` (including `mx cache prune`).
- Package named `cache` requires `mx -p cache <bin>`.

**Security boundary**

Consent occurs after metadata resolution and before tarball download, extraction,
store import, linking, lifecycle execution, or binary launch. Metadata probes may
run before consent; artifact probes must not.

## Unified execution (MVP **0045**)

`internal/runner/envexec` is the shared execution-environment layer behind
`m exec`, `mx`, snapshot execution, and capsule execution. `internal/app` is a
thin façade that builds `ExecutionRequest` values and calls
`envexec.Orchestrator`.

| Source | CLI | Network | Mutation | Cache |
|---|---|---|---|---|
| Project | `m exec <bin>` | forbidden | project-forbidden | none (uses project tree) |
| DLX | `mx …` | metadata/artifacts per 0044 consent | cache-only | `<cache>/mx/exec/<digest>/` |
| Snapshot | `m exec --snapshot <id> <bin>` | forbidden | cache-only | `<cache>/envexec/v1/snapshot/<digest>/` |
| Capsule | `m exec --capsule <path> <bin>` | forbidden | cache-only | `<cache>/envexec/v1/capsule/<digest>/` |

Snapshot and capsule execution never mutate the user project, never contact the
registry, and never run lifecycle scripts during materialization. Capsule trust
is integrity-only (no publisher signature claims).

Shared cache entries publish atomically with `ready.json`, `m.lock`,
`node_modules/.mew/bins.v1.json`, and `.mew/generation.json`. Warm validation
rejects artifact disagreement before launch.

`mx` remains DLX-only; there is no `mx --snapshot`.

Integration: `tests/integration/unified_exec_test.go`. Fixtures:
`fixtures/unified-exec/`.

## Related

- [`cli.md`](cli.md) — command registration and precedence
- [`errors.md`](errors.md) — `ERR_M_NOT_FOUND` for missing scripts
- [`compatibility-axes.md`](compatibility-axes.md) — `m run` and workspace orchestration parity
