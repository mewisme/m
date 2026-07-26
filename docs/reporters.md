# Reporters and diagnostics

Mew separates user-facing diagnostics from internal logs. Progress is modeled as
events; terminal rendering is a reporter concern.

## Formats

| Mode | Flag / env | Behavior |
|---|---|---|
| `default` | `--reporter default` (default) | Human text on stderr; optional color |
| `ndjson` | `--reporter ndjson` or `MEW_LOG_FORMAT=ndjson` | One JSON object per line on stdout |
| `json` | `--reporter json` | Single JSON error document on stdout |
| `silent` | `--reporter silent` | No progress; errors still on stderr |

## Flags and environment

| Flag / env | Purpose |
|---|---|
| `--reporter` | Select reporter |
| `MEW_LOG_FORMAT` | Default reporter when flag unset |
| `--debug` / `MEW_DEBUG` / `M_LOG=debug` | Verbose debug lines |
| `--color` / `--no-color` / `NO_COLOR` | ANSI color policy |
| `--unsafe-diagnostics` | Disable redaction (hidden; dangerous) |

## Progress events

Schema: [`schemas/diagnostics/progress-event.schema.json`](../schemas/diagnostics/progress-event.schema.json).

```json
{"v":1,"type":"progress","phase":"fetch","package":"left-pad@1.0.0","bytes":50,"total_bytes":100,"op_id":"op-1","tx_id":null}
```

NDJSON writes are mutex-guarded so concurrent progress is line-atomic.

## Errors

Schema: [`schemas/diagnostics/error.schema.json`](../schemas/diagnostics/error.schema.json).
Codes: [`errors.md`](errors.md).

## Redaction

Default and debug modes always redact URL userinfo, Bearer tokens, and common
`*_TOKEN=` / `*_PASSWORD=` shapes. Only `--unsafe-diagnostics` skips redaction.

## Debug bundles

Future `m doctor report` archives must require explicit consent and apply the
same redaction rules. Not implemented in MVP 0005.

## Tracing

[`internal/trace`](../internal/trace) provides in-process spans without OpenTelemetry.
OTel remains an optional future adapter.
