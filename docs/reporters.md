# Reporters and diagnostics

Mew separates user-facing diagnostics from internal logs. Progress is modeled as
events; terminal rendering is a reporter concern.

## Formats

| Mode | Flag / env | Behavior |
|---|---|---|
| `default` / `plain` / `rich` | `--output auto\|plain\|rich` or `--reporter default` | Human text on stderr; `rich` enables design-system styling when capable |
| `ndjson` | `--output ndjson` or `--reporter ndjson` or `MEW_LOG_FORMAT=ndjson` | One JSON object per line on stdout |
| `json` | `--output json` or `--reporter json` | Single JSON error document on stdout |
| `silent` | `--output silent` or `--reporter silent` | No progress; errors still on stderr |

Resolution is centralized in [`internal/presentation`](../internal/presentation). Conflicting
`--output` and `--reporter` values return a usage error.

## Flags and environment

| Flag / env | Purpose |
|---|---|
| `--output` | Canonical output mode (`auto`, `rich`, `plain`, `json`, `ndjson`, `silent`) |
| `--reporter` | Legacy alias for structured/human reporter selection |
| `MEW_OUTPUT` | Default output mode when `--output` unset |
| `MEW_LOG_FORMAT` | Legacy env alias (same precedence tier as `MEW_OUTPUT`) |
| `--progress` / `MEW_PROGRESS` / `ui.progress` | Progress policy (`auto`, `always`, `never`) |
| `--unicode` / `MEW_UNICODE` / `ui.unicode` | Unicode symbol policy |
| `--interactive` / `MEW_INTERACTIVE` / `ui.interactive` | Interactive UI policy |
| `--accessible` / `MEW_ACCESSIBLE` / `ui.accessible` | Append-only accessible output |
| `--log-level` / `MEW_LOG_LEVEL` / `log.level` | Diagnostic verbosity |
| `--no-summary` / `ui.summary` | Suppress command summaries |
| `--presentation-legacy` / `MEW_PRESENTATION=legacy` | Hidden rollout switch (forces legacy human path) |
| `--debug` / `MEW_DEBUG` / `M_LOG=debug` | Verbose debug lines |
| `--color` / `--no-color` / `NO_COLOR` / `ui.color` | ANSI color policy |
| `ui.theme` | Theme palette: `auto`, `light`, `dark`, `accessible`, `none` |
| `--unsafe-diagnostics` | Disable redaction (hidden; dangerous) |

### Color and `NO_COLOR`

Precedence for color:

1. `--color=always|never`
2. `MEW_COLOR`
3. `ui.color`
4. `NO_COLOR` / `--no-color` force never **unless** `--color=always` was set
5. Auto requires a suitable stdout TTY, non-dumb `TERM`, and a color-capable profile

Explicit `--color=always` overrides `NO_COLOR`. Plain / pipe / CI / accessible paths use the first-class plain renderer (zero ANSI); rich output is never stripped to fake plain.

### Width and Unicode

Terminal width is detected once per invocation (default 80; clamped 20–500). Tables stack below ~60 columns. Unicode symbols are selected via `--unicode` / `MEW_UNICODE` / `ui.unicode`, with ASCII fallbacks (`OK`, `WARN`, `ERROR`, `->`).

Command-local `--json` emits command **result** documents and must not be combined with
global `--output=json` or `--output=ndjson`.

## Progress events

Schema: [`schemas/diagnostics/progress-event.schema.json`](../schemas/diagnostics/progress-event.schema.json).

```json
{"v":1,"type":"progress","phase":"fetch","package":"left-pad@1.0.0","bytes":50,"total_bytes":100,"op_id":"op-1","tx_id":null}
```

NDJSON writes are mutex-guarded so concurrent progress is line-atomic.

Additional NDJSON event schemas (v1): `operation-started`, `operation-progress`,
`operation-completed`, and `notice` under [`schemas/diagnostics/`](../schemas/diagnostics/).

### Install progress (UX-0004)

Install-family mutations emit typed `Operation*` events for phases: `resolve`,
`fetch`, `link`, `lifecycle`, `validate`, `commit`, `rollback`, `cleanup`.

| Effective mode | Progress rendering |
|---|---|
| `plain` / CI / accessible / redirected stderr | Append-only zero-ANSI lines on stderr (`resolve started`, `fetch completed duration=…`) |
| `rich` with `--progress=auto` and stderr TTY | Inline Bubble Tea renderer on stderr (no alt screen, no stdin ownership) |
| `--progress=never` | No phase lines; lifecycle/security `Notice` events still print |
| `--progress=always` without stderr TTY | Usage error before mutation start |
| Auto-rich when live UI cannot start | One-shot downgrade to plain + debug notice; product continues |

Final human mutation summary is written to **stdout** via `StaticRenderer`
(`Installed N packages in …` with Added/Updated/Removed metrics). `--no-summary`
suppresses that success summary but never suppresses errors or lifecycle/security
notices. JSON/NDJSON result documents are unchanged (additive `InstallResult`
metric fields only).

### Runner and workspace presentation (UX-0005)

Human reporters invoke presentation hooks for:

- `environment-prepared` → execution prep banner on stderr (safe labels only)
- `workspace-task` / `workspace-summary` → append-only status + final counts
- human-only `prep-stage` / `exec-prep` / `exec-summary` Progress types (not emitted on NDJSON)

JSON/NDJSON keep the frozen `environment-prepared`, `workspace-task`, and
`workspace-summary` schemas. Reporter failure after lease acquire still releases
the lease. Live install progress Suspend/Resume also covers runner children.

## Errors

Schema: [`schemas/diagnostics/error.schema.json`](../schemas/diagnostics/error.schema.json).
Codes: [`errors.md`](errors.md).

Human errors (default/plain/rich reporters) map typed `apperr` failures to an
`ErrorView` in `internal/presentation`: title, message, optional context rows,
`ERR_M_*` code, and up to three deterministic hints. Rich mode applies semantic
error styling; plain mode prefixes `ERROR` with no ANSI. Debug mode may append a
short cause chain.

`diagnostics.Options.HumanErrorRender` injects the presentation renderer without
import cycles. When `--presentation-legacy` or `MEW_PRESENTATION=legacy` is set,
human errors keep the legacy `formatHumanError` path (including optional red ANSI).

JSON and NDJSON error documents are unchanged.

## Redaction

Default and debug modes always redact URL userinfo, Bearer tokens, and common
`*_TOKEN=` / `*_PASSWORD=` shapes. Only `--unsafe-diagnostics` skips redaction.

## Debug bundles

Future `m doctor report` archives must require explicit consent and apply the
same redaction rules. Not implemented in MVP 0005.

## Tracing

[`internal/trace`](../internal/trace) provides in-process spans without OpenTelemetry.
OTel remains an optional future adapter.
