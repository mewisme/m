# CLI presentation architecture

Human and structured CLI output is owned by [`internal/presentation`](../../internal/presentation).
Product packages emit semantic diagnostics; they do not import Charm or drive
terminal drawing.

## Output modes

| Mode | Selection | Properties |
|---|---|---|
| `rich` | Default (no flag needed) | Rich design-system styled output; TTY uses live renderer, non-TTY uses static-rich append-only |
| `plain` | `--output=plain` | Append-only, zero ANSI / cursor control |
| `json` | `--output=json` | Structured stdout only; no human prefix |
| `ndjson` | `--output=ndjson` | Line-delimited events on stdout |
| `silent` | `--output=silent` | No progress/summary; required errors still surface |

Rich is always the default. Resolution lives in `presentation.Resolve`
(`internal/presentation/resolve.go`) and is driven exclusively by CLI flags.
Environment variables and config keys no longer influence presentation.

## Streams

| Stream | Owns |
|---|---|
| stdout | Machine payloads (JSON/NDJSON), help text when written to stdout, completion scripts, child stdout passthrough |
| stderr | Human progress, notices, ErrorView, prompts, live Bubble Tea frames, static-rich append-only lines |

Progress and human notices must never contaminate stdout. Completions are
ANSI-free. Child single-task runs keep raw stdin/stdout/stderr ownership;
presentation Suspend/Resume brackets child Start/Wait.

## Charm boundary

Allowed: `internal/presentation` (and its tests), including
`internal/presentation/help`, `pager`, and `prompt`.

Forbidden: domain packages (`app`, `runner`, `transaction`, `resolver`,
`linker`, `store`, `lifecycle`, …) and `internal/prompt` (stdlib contract only).

Enforced by [`internal/archcheck`](../../internal/archcheck) and
[`forbidden-imports.md`](forbidden-imports.md).

Pinned modules (see [`plans/ux/charm-dependency-review.md`](../../plans/ux/charm-dependency-review.md)):

```text
charm.land/lipgloss/v2
charm.land/bubbletea/v2
charm.land/bubbles/v2
charm.land/huh/v2
charm.land/glamour/v2
```

Live programs use stderr output, no alternate screen, and no signal ownership.
Huh forms inject I/O; they do not probe global stdio in `init`.

## Lifecycle

`presentation.Controller` owns live UI lifetime: start (lazy where applicable),
update, Suspend/Resume around child I/O, and Close on success, cancel, product
error, or broken-pipe style paths. Renderer failure must not strand transaction
resources; product cleanup completes and presentation errors surface afterward.

## Accessibility

`--accessible` preserves rich output formatting while selecting the numbered
prompt adapter. See [`docs/accessibility.md`](../accessibility.md).

## Certification

- Conformance: `m conformance run cli-ux --json`
- Evidence: [`docs/evidence/cli-ux/`](../evidence/cli-ux/)
- Plans: [`plans/ux/`](../../plans/ux/)
