# Reporters and diagnostics

Mew separates user-facing diagnostics from internal logs. Progress is modeled as
events; terminal rendering is a reporter concern.

## Formats

| Mode | Flag | Behavior |
|---|---|---|
| `rich` | `--output=rich` (default) | Rich design-system styled output on stderr; TTY uses live renderer, non-TTY uses static-rich append-only |
| `plain` | `--output=plain` | Append-only zero-ANSI lines on stderr |
| `json` | `--output=json` | Single JSON error document on stdout |
| `ndjson` | `--output=ndjson` | One JSON object per line on stdout |
| `silent` | `--output=silent` | No progress; errors still on stderr |

Rich is always the default. The `auto`/`default`/`human` values are no longer
accepted — use `rich` explicitly if desired, or omit the flag for the default.
Resolution is centralized in [`internal/presentation`](../internal/presentation)
and driven only by explicit CLI flags. Environment variables and config keys no
longer influence presentation.

## Flags

| Flag | Purpose |
|---|---|
| `--output` | Canonical output mode: `rich` (default), `plain`, `json`, `ndjson`, `silent` |
| `--no-color` | Disable ANSI color |
| `--no-progress` | Disable progress output |
| `--ascii` | Use ASCII instead of Unicode symbols |
| `--accessible` | Accessible append-only output; preserves rich formatting |
| `--no-summary` | Suppress command summary output |
| `--log-level` | Diagnostic verbosity: `error` (default), `warn`, `info`, `debug` |
| `--debug` | Shorthand for `--log-level=debug` plus `MEW_DEBUG` / `M_LOG=debug` env vars |

### Color

- `--no-color` disables ANSI color entirely.
- Explicit `--output=plain` and structured modes (`json` / `ndjson` / `silent`) produce no ANSI.
- Rich output uses color when available on a TTY; `--no-color` forces plain symbols.

### Width and Unicode

Terminal width is detected once per invocation (default 80; clamped 20–500).
Tables stack below ~60 columns. Unicode symbols are used by default; `--ascii`
selects ASCII fallbacks (`OK`, `WARN`, `ERROR`, `->`).

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

| Output mode | Progress rendering |
|---|---|
| `rich` (TTY) | Inline Bubble Tea renderer on stderr (no alt screen, no stdin ownership) |
| `rich` (non-TTY) | Static-rich append-only lines on stderr with symbols and optional color |
| `plain` | Append-only zero-ANSI lines on stderr |
| `json` / `ndjson` / `silent` | No progress rendering |
| `--no-progress` | No phase lines; lifecycle/security `Notice` events still print |

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

Human errors (rich/plain reporters) map typed `apperr` failures to an
`ErrorView` in `internal/presentation`: title, message, optional context rows,
`ERR_M_*` code, and up to three deterministic hints. Rich mode applies semantic
error styling; plain mode prefixes `ERROR` with no ANSI. Debug mode may append a
short cause chain.

`diagnostics.Options.HumanErrorRender` injects the presentation renderer without
import cycles.

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
