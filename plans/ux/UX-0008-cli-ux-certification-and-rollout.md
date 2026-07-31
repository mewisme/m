---
name: UX-0008 CLI UX Certification and Rollout
overview: Certify the complete Mew presentation system across platforms, terminals, output modes, widths, accessibility, child processes, cancellation, structured schemas, startup performance, dependency boundaries, documentation, rollout controls, and inventory status.
todos:
  - id: p1-matrix
    content: Build versioned CLI UX conformance matrix covering commands, modes, terminals, platforms, streams, widths, and failures
    status: pending
  - id: p2-platform
    content: Produce Linux, macOS, and Windows terminal evidence including PTY/console cases
    status: pending
  - id: p3-structured
    content: Certify JSON, NDJSON, completion, pipe, redirection, no-ANSI, redaction, and exit-code compatibility
    status: pending
  - id: p4-performance
    content: Benchmark startup, binary size, render throughput, high-volume output, and live-render overhead
    status: pending
  - id: p5-accessibility
    content: Certify no-color, ASCII, append-only, screen-reader, narrow-width, and non-interactive behavior
    status: pending
  - id: p6-rollout
    content: Define default enablement, rollback switch, telemetry-free diagnostics, migration cleanup, and fallback removal
    status: pending
  - id: p7-docs
    content: Update CLI, runner, testing, accessibility, configuration, contributor, and architecture documentation
    status: pending
  - id: p8-ship
    content: Update the standalone CLI UX index/checklist, product inventory, and integration links; remove legacy presentation paths only after all ship gates pass
    status: pending
isProject: false
---

# UX-0008 — CLI UX Certification and Rollout

## Goal

Turn the presentation work from plans UX-0001 through UX-0007 into a certified product contract rather than a collection of attractive snapshots.

Certification must prove that rich human output improves usability without regressing automation, machine output, security, package-manager behavior, runner behavior, terminal ownership, accessibility, startup performance, or cross-platform operation.

## User-visible deliverables

- Rich human output enabled by default only in supported interactive terminals.
- Plain append-only output in CI, pipes, redirects, dumb terminals, and accessible mode.
- Stable JSON and NDJSON output.
- Documented global UI controls.
- Consistent errors, help, progress, summaries, prompts, runner output, and workspace output.
- Platform evidence for Linux, macOS, and Windows.

## Explicitly out of scope

- Adding product features to make screenshots more impressive.
- Relaxing errors or trust policies for smoother demos.
- Full-screen default UI.
- Collecting user telemetry.
- Hard performance marketing claims.
- Certifying terminal applications not covered by declared capability matrix.
- Keeping duplicate legacy/new renderers indefinitely.

## Certification matrix

Create a versioned manifest, for example:

```text
tests/conformance/cli-ux/manifest.json
```

Dimensions:

```text
command family
output mode
color mode
progress mode
Unicode mode
interactive mode
accessibility mode
stdin/stdout/stderr TTY state
CI state
terminal width
platform
success/failure/cancellation/broken-pipe outcome
child stream behavior
```

## Required suite groups

### Foundation

- output resolution;
- terminal capability snapshot;
- config/environment/CLI precedence;
- import boundaries;
- renderer lifecycle;
- redaction.

### Static

- version/features/project/config;
- errors and hints;
- help and completions;
- tables and summaries;
- doctor/list/outdated/explain/audit/policy representative commands.

### Mutation

- install phases;
- add/remove/update;
- lifecycle output;
- commit/rollback/recovery;
- cancellation and cleanup.

### Runner

- single script/bin streams;
- workspace aggregate/stream;
- interactive child;
- signals and exit codes;
- `mx` consent and cache;
- snapshot/capsule execution.

### Prompt

- TTY rich prompt;
- accessible prompt;
- non-TTY failure;
- CI suppression;
- structured-mode suppression;
- cancellation and EOF.

### Help

- topic rendering;
- no-color;
- pager available/missing/failure;
- link validation.

### Performance

- startup;
- binary size;
- static render throughput;
- high-volume child output;
- live update frequency and CPU usage.

## Output-mode conformance

For every representative command:

| Mode | Required properties |
|---|---|
| auto interactive | rich when eligible; no machine corruption |
| rich forced | styled output or locked unsupported behavior |
| plain | append-only, no ANSI/cursor control |
| json | valid schema, no human text |
| ndjson | valid line-delimited events, no partial records |
| silent | no progress/summary; required result/error contract preserved |
| accessible | append-only, no animation, readable without color |

## Stream certification

Capture stdout and stderr separately.

Assert:

- progress never contaminates stdout;
- JSON/NDJSON stdout contains only structured output;
- child stdout remains stdout;
- child stderr remains stderr;
- warnings/errors follow documented stream policy;
- help/completion output remains valid;
- no ANSI in redirected plain output;
- no cursor control outside eligible live terminal output.

Use byte-level escape detection, not visual inspection only.

## Terminal control certification

Test cleanup after:

```text
success
product error
usage error
renderer startup failure
renderer write failure
Ctrl+C
context deadline
child signal exit
broken pipe
panic recovered at CLI boundary, if such recovery exists
```

Assert:

- cursor visible;
- terminal mode restored;
- no renderer goroutine leak;
- no active timer leak;
- no process/lease/transaction resource leak;
- final newline policy respected.

## Platform matrix

Required evidence:

```text
Linux
macOS
Windows
```

### Linux/macOS

- PTY tests;
- signals;
- process groups;
- ANSI/no-color;
- pager behavior;
- terminal resize.

### Windows

- Windows Terminal/ConPTY where CI tooling permits;
- `.cmd` and executable launch;
- Ctrl+C/process behavior;
- ANSI capability fallback;
- Unicode/ASCII glyphs;
- width detection;
- pager absence;
- path wrapping.

Do not mark a platform fully certified if only plain redirected snapshots were tested.

## Terminal-width matrix

Required widths:

```text
40
60
80
120
```

Optional stress widths:

```text
20
200
```

At required widths:

- no panic;
- no negative widths;
- error code visible;
- package/command identity preserved where possible;
- tables stack or truncate according to policy;
- prompt remains operable;
- no unbounded line from Mew-generated content.

Child output is not reformatted to width.

## Color and Unicode matrix

Test:

```text
auto color
always color
never color
NO_COLOR
CLICOLOR
CLICOLOR_FORCE
Unicode auto/always/never
ASCII fallback
light/dark/accessible themes
TERM=dumb
```

Status must remain understandable in monochrome ASCII output.

## Accessibility certification

Required evidence:

- append-only reading order;
- no cursor repaint;
- no animation;
- no color-only state;
- numbered prompts;
- safe defaults;
- screen-reader-oriented documentation;
- width 40;
- output captured as linear text;
- plain error/action hints.

Automated tests cannot fully certify screen-reader usability. Include a documented manual review checklist and evidence artifact.

## Machine-output compatibility

Inventory all JSON/NDJSON schemas touched by the program.

For each:

- schema version unchanged or intentionally bumped;
- exact field compatibility documented;
- no ANSI;
- no human prefix/suffix;
- line atomicity for NDJSON;
- redaction preserved;
- broken reporter behavior tested;
- ordering deterministic where contract requires.

Use golden/schema tests rather than comparing pretty human output.

## Completion certification

Shell completions must remain:

- ANSI-free;
- stable;
- fast;
- free of progress and prompt output;
- valid for supported shells;
- independent of rich human renderer startup where possible.

## Performance and footprint

### Startup benchmarks

Measure before/after:

```text
m version
m --help
m features --json
mx version
```

Record:

- median;
- p95;
- sample count;
- OS/arch;
- build mode;
- binary size;
- exact dependency versions.

### Live-render benchmarks

Measure:

- updates per second;
- CPU utilization during idle spinner;
- memory allocations;
- 10,000 progress events;
- high-volume child output with renderer suspended;
- terminal-width resize events.

Set thresholds only after baseline. Ship gate requires reviewed, documented regression bounds.

## Dependency certification

For every added Charm module:

- exact version pinned;
- license compatible;
- vulnerability scan passes;
- dependency purpose documented;
- no duplicate major versions without justification;
- binary/startup impact recorded;
- update owner and review date assigned;
- imported only by allowed packages.

## Architecture certification

Executable test must verify:

```text
product/domain packages do not import internal/presentation
product/domain packages do not import charm.land modules
internal/presentation may import internal/diagnostics
internal/diagnostics does not import internal/presentation
structured reporters do not depend on live Bubble Tea models
```

## Rollout plan

### Stage 1 — Opt-in developer preview

- rich output behind config/environment gate;
- plain remains default;
- gather repository-internal evidence only, no telemetry;
- document known limitations.

### Stage 2 — Auto mode for selected static commands

- enable low-risk commands;
- keep install/runner plain unless explicitly enabled;
- compare CI and user-reported behavior.

### Stage 3 — Install and runner auto mode

- enable after transaction and child-stream gates pass;
- preserve emergency plain fallback.

### Stage 4 — Default rich auto

- rich only when capabilities qualify;
- CI/pipe remain plain;
- rollback switch available for one release/milestone window.

### Stage 5 — Cleanup

- remove legacy renderer and temporary switch after stability window;
- keep permanent `--output=plain`.

**Stage 5 is now complete.** The legacy presentation path (`--presentation-legacy`,
`MEW_PRESENTATION=legacy`), `--reporter`, tri-state flags, and all
presentation-related environment variables and config keys have been removed.
Rich output is the default, controlled only by explicit CLI flags.

## Rollback strategy

A presentation rollback must not require reverting domain changes.

Maintain:

- plain renderer as permanent fallback (use `--output=plain`);
- structured reporters independent from rich renderer;
- ability to disable live Bubble Tea via `--no-progress` while retaining static Lip Gloss output.

## Documentation updates

At minimum:

```text
docs/cli.md
docs/config.md
docs/errors.md
docs/testing.md
docs/runner.md
docs/architecture/cli-presentation.md
docs/accessibility.md
CONTRIBUTING.md
README.md, concise user-facing update only
plans/ux/INDEX.md
plans/ux/CHECKLIST.md
features/inventory.json
plans/INDEX.md, link to the standalone program only when desired
```

Document:

- output modes;
- stream ownership;
- environment/config precedence;
- no-color;
- accessibility;
- progress behavior;
- prompt behavior;
- structured output;
- troubleshooting;
- dependency architecture.

## Inventory and status

Add or update feature entries equivalent to:

```text
cli.presentation-foundation
cli.rich-human-output
cli.accessible-output
cli.rich-errors
cli.install-progress
cli.runner-progress
cli.prompt-system
cli.markdown-help
cli.ux-certification
```

Do not mark shipped until applicable tests and platform evidence pass.

## Verification commands

Adapt to repository tooling:

```sh
gofmt -w <changed-go-files>
go test ./internal/diagnostics/... ./internal/presentation/... ./internal/cli/... -count=1
go test ./internal/app/... ./internal/transaction/... ./internal/runner/... ./internal/process/... -count=1
go test ./tests/integration/... ./tests/conformance/... -count=1
go test -race ./internal/diagnostics/... ./internal/presentation/... ./internal/cli/... ./internal/runner/... ./internal/process/... -count=1
go test ./... -count=1
go vet ./...
golangci-lint run ./...
govulncheck ./...
go build ./cmd/m ./cmd/mx
```

Add platform-specific conformance commands and artifact upload.

## Ship criteria

The CLI UX program is complete only when:

- all representative commands use centralized presentation;
- rich auto mode is limited to eligible terminals;
- CI/pipe/plain output has no ANSI/cursor control;
- JSON/NDJSON schemas pass;
- child stdio/signals/exit codes pass;
- transaction rollback/recovery presentation is truthful;
- non-TTY prompts fail closed;
- accessible mode passes automated and manual evidence;
- width matrix passes;
- Linux/macOS/Windows evidence exists;
- renderer cleanup has no leaks;
- import-boundary test passes;
- vulnerability/license review passes;
- startup and binary-size regressions are reviewed and within approved bounds;
- standalone program documentation, local checklist, integration links, and product inventory are updated;
- temporary legacy path has an explicit removal decision.

## Risks

| Risk | Mitigation |
|---|---|
| Beautiful snapshots hide stream regressions | byte-level stdout/stderr conformance |
| Rich default breaks CI | auto capability rules and plain CI tests |
| Child terminal behavior regresses | PTY/console and signal matrix |
| Accessibility remains theoretical | append-only implementation and manual evidence |
| Charm upgrades introduce breaking behavior | exact pins and dependency review cadence |
| Binary/startup cost is excessive | benchmarks and optional dependency staging |
| Legacy renderer never gets removed | rollout milestones and removal gate |
| Platform evidence is incomplete | aggregate certification report requiring all declared platforms |

## Estimated effort

**20–32 focused engineering hours** for certification, rollout infrastructure, evidence, and documentation, excluding defects discovered in earlier plans.
