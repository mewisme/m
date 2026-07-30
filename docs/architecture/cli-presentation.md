# CLI presentation architecture

Human and structured CLI output is owned by [`internal/presentation`](../../internal/presentation).
Product packages emit semantic diagnostics; they do not import Charm or drive
terminal drawing.

## Output modes

| Mode | Selection | Properties |
|---|---|---|
| `auto` | Default | Rich when `RichEligible`; otherwise plain |
| `rich` | `--output=rich` | Styled human output; fails closed if stderr is not an interactive eligible TTY |
| `plain` | `--output=plain`, CI, pipe, dumb, accessible, legacy auto | Append-only, zero ANSI / cursor control |
| `json` | `--output=json` | Structured stdout only; no human prefix |
| `ndjson` | `--output=ndjson` | Line-delimited events on stdout |
| `silent` | `--output=silent` | No progress/summary; required errors still surface |

Resolution lives in `presentation.Resolve` (`internal/presentation/resolve.go`).
Precedence for mode: CLI `--output` → `--reporter` → `MEW_OUTPUT` /
`MEW_LOG_FORMAT` → `ui.output` → `auto`.

`RichEligible` requires stderr TTY, non-CI, non-dumb terminal, and not
accessible / screen-reader.

## Streams

| Stream | Owns |
|---|---|
| stdout | Machine payloads (JSON/NDJSON), help text when written to stdout, completion scripts, child stdout passthrough |
| stderr | Human progress, notices, ErrorView, prompts, live Bubble Tea frames |

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

`--accessible` / `MEW_ACCESSIBLE` / `ui.accessible` forces append-only plain
output and numbered prompts. See [`docs/accessibility.md`](../accessibility.md).

## Rollout stages

| Stage | Status |
|---|---|
| 1–3 | Historical — UX-0001–0003 foundation, design system, static output |
| 4 Default rich auto | **Current** — rich only when `RichEligible`; CI/pipe/accessible/legacy → plain |
| 5 Cleanup | **Deferred** — remove `--presentation-legacy` / `MEW_PRESENTATION=legacy` after one stable release window and green `cli-ux` matrix on Ubuntu, Windows, and macOS |

Permanent controls independent of Stage 5: `--output=plain`, structured
reporters (`json` / `ndjson` / `silent`), and `--accessible`.

## Certification

- Conformance: `m conformance run cli-ux --json`
- Evidence: [`docs/evidence/cli-ux/`](../evidence/cli-ux/)
- Plans: [`plans/ux/`](../../plans/ux/)
