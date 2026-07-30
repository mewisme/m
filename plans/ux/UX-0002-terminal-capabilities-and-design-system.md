---
name: UX-0002 Terminal Capabilities and Design System
overview: Implement deterministic terminal capability detection, rich/plain mode selection, semantic styling tokens, Unicode and ASCII symbols, responsive width handling, reusable tables and summaries, and a restrained Mew visual language built on Lip Gloss.
todos:
  - id: p1-capabilities
    content: Build injectable terminal capability detection for stdin/stdout/stderr, size, CI, dumb terminals, color, Unicode, and accessibility
    status: pending
  - id: p2-resolution
    content: Implement deterministic color/progress/unicode/interactive resolution using UX-0001 precedence
    status: pending
  - id: p3-theme
    content: Add semantic theme tokens, light/dark/accessible palettes, and no-color rendering
    status: pending
  - id: p4-symbols
    content: Add Unicode symbols with ASCII fallbacks and width-safe validation
    status: pending
  - id: p5-components
    content: Build reusable key-value blocks, tables, summaries, badges, deltas, notices, and hints
    status: pending
  - id: p6-width
    content: Implement width-aware layouts at 40, 60, 80, and 120 columns
    status: pending
  - id: p7-dependency
    content: Pin Lip Gloss v2 after dependency, binary-size, startup, and security review
    status: pending
  - id: p8-tests
    content: Add golden, no-ANSI, width, color-profile, Unicode, and Windows rendering tests
    status: pending
isProject: false
---

# UX-0002 — Terminal Capabilities and Design System

## Goal

Create the reusable visual and terminal-capability layer used by every later human-facing CLI improvement.

The design system must make Mew feel polished through hierarchy, spacing, consistency, and information density—not through excessive borders, gradients, animation, or terminal takeover.

## User-visible outcome

Static human output becomes consistently styled when appropriate and automatically degrades to clean plain text.

Examples:

```text
✓ Added zod 4.0.14

  Packages    +1
  Lockfile    updated
  Duration    620ms
```

No-color/ASCII fallback:

```text
OK Added zod 4.0.14

  Packages    +1
  Lockfile    updated
  Duration    620ms
```

## Explicitly out of scope

- Live multi-operation progress; see UX-0004 and UX-0005.
- Interactive prompts; see UX-0006.
- Markdown help; see UX-0007.
- User-authored themes.
- Full-screen interfaces.
- Terminal images, hyperlinks as required UX, mouse support, or clipboard integration.
- Conveying status by color alone.

## Charm dependency

Primary dependency:

```go
import "charm.land/lipgloss/v2"
```

Lip Gloss is used for pure styling and layout. New code should avoid global terminal detection hidden inside style definitions. Capability decisions are injected explicitly.

Before pinning:

- verify Go-version compatibility;
- record exact version;
- run license/security review;
- measure binary-size and startup impact;
- verify Windows support;
- verify no unexpected terminal I/O occurs during package initialization.

## Capability snapshot

```go
type Capabilities struct {
    StdinTTY      bool
    StdoutTTY     bool
    StderrTTY     bool
    Width         int
    Height        int
    CI            bool
    DumbTerminal  bool
    Unicode       bool
    Interactive   bool
    ScreenReader  bool
    ColorProfile  ColorProfile
    Background    BackgroundMode
    Hyperlinks    bool
}
```

### Snapshot rules

- Capture once during CLI setup.
- Make immutable for the invocation.
- Use injected readers/writers/environment in tests.
- Do not query global `os.Stdout` from rendering components.
- Default width to 80 only when size cannot be determined.
- Clamp width to a safe range before layout.
- Treat height as advisory; inline output must not depend on a minimum height.

## Terminal detection inputs

At minimum evaluate:

```text
stdin/stdout/stderr character-device state
terminal width and height
TERM
COLORTERM
NO_COLOR
CLICOLOR
CLICOLOR_FORCE
CI
GITHUB_ACTIONS
WT_SESSION
TERM_PROGRAM
MEW_* UI overrides
```

Do not infer screen-reader use from unreliable terminal heuristics. Accessibility mode is explicit through flag/config/environment, with safe auto behavior where repository policy allows.

## Resolution rules

### Color

Recommended precedence:

1. `--color=always|never`.
2. `MEW_COLOR`.
3. effective `ui.color`.
4. `NO_COLOR` disables color unless an explicit CLI force policy intentionally overrides it.
5. `CLICOLOR_FORCE` may enable color only when compatible with Mew policy.
6. auto requires a suitable output terminal and non-dumb terminal.

Document exact conflict behavior between `NO_COLOR` and explicit `--color=always`.

### Progress

Auto progress requires:

- output mode rich;
- stderr TTY;
- not CI;
- not dumb;
- not accessible append-only mode;
- command classified as live-render compatible.

### Unicode

Auto Unicode requires:

- non-dumb terminal;
- known-safe encoding/environment;
- no accessible/ASCII override;
- platform capability test passes.

Failure to prove support selects ASCII, not mojibake-prone Unicode.

### Interactivity

Interactive mode requires:

- stdin TTY;
- human output mode;
- not CI unless explicitly allowed;
- command contract permits prompting;
- no accessible restriction that requires an alternate prompt path.

## Theme model

```go
type Theme struct {
    Primary   lipgloss.Style
    Secondary lipgloss.Style
    Muted     lipgloss.Style
    Strong    lipgloss.Style

    Success   lipgloss.Style
    Warning   lipgloss.Style
    Error     lipgloss.Style
    Info      lipgloss.Style

    Command   lipgloss.Style
    Package   lipgloss.Style
    Version   lipgloss.Style
    Path      lipgloss.Style
    Code      lipgloss.Style
    Number    lipgloss.Style

    Added     lipgloss.Style
    Updated   lipgloss.Style
    Removed   lipgloss.Style
    Reused    lipgloss.Style

    Header    lipgloss.Style
    Label     lipgloss.Style
    Value     lipgloss.Style
}
```

### Theme modes

```text
auto
light
dark
accessible
none
```

`none` is the plain/no-style theme, not a separate output mode.

### Theme constraints

- Maximum practical semantic palette: primary, muted, success, warning, error, info, added, removed.
- Avoid coloring complete paragraphs.
- Preserve contrast on light and dark backgrounds.
- Bold must not be required for meaning.
- Do not use blink.
- Avoid italics for essential content because terminal support varies.
- Do not use background color for large blocks by default.
- Border use is exceptional, not default.

## Symbol model

```go
type Symbols struct {
    Success string
    Warning string
    Error   string
    Info    string
    Arrow   string
    Bullet  string
    Pending string
    Running string
    Skipped string
    Added   string
    Removed string
}
```

Recommended Unicode set:

```text
✓  !  ×  →  •  ○  ●  –  +  -
```

Recommended ASCII set:

```text
OK  WARN  ERROR  ->  *  .  *  -  +  -
```

Rules:

- Validate visible cell width.
- Never assume rune count equals terminal width.
- Symbols must not be the only carrier of status.
- Golden tests cover Unicode and ASCII variants.

## Typography and spacing rules

- Use one blank line between headline and detail block.
- Use two-space indentation for secondary metadata.
- Keep labels short and aligned only when width permits.
- Prefer sentence case.
- Use backticks for commands and exact configuration keys in plain text.
- Avoid trailing punctuation in compact status labels.
- Use full sentences in errors and explanatory notices.
- Default human summaries should fit within approximately 10–15 lines unless the command inherently returns a table.

## Reusable components

### Status line

```go
type StatusLine struct {
    Status Status
    Text   string
    Detail string
}
```

Example:

```text
✓ Installed 126 packages
```

### Key-value block

```go
type KeyValue struct {
    Key   string
    Value string
    Style ValueKind
}
```

Wide:

```text
  Packages    126
  Downloaded  7
  Reused      119
```

Narrow:

```text
Packages: 126
Downloaded: 7
Reused: 119
```

### Summary

```go
type Summary struct {
    Status  Status
    Title   string
    Metrics []KeyValue
    Notices []Notice
    Hints   []Hint
}
```

### Notice and hint

```text
! 3 lifecycle scripts were blocked
→ Run `m builds` to review them
```

### Package delta

```text
+ zod        4.0.14
~ react      19.1.0 -> 19.1.1
- lodash     4.17.21
```

ASCII must remain understandable without color.

### Table

A table component must support:

- column definitions;
- minimum and preferred width;
- alignment;
- truncation policy;
- wrapping policy;
- stacked fallback;
- stable sorting;
- no border by default;
- no color mode;
- plain text mode;
- screen-reader linear mode.

## Responsive layout

Test fixed widths:

```text
40
60
80
120
```

### Wide table

```text
PACKAGE       CURRENT   WANTED   LATEST   TYPE
react         19.1.0    19.1.1   19.1.1   dependency
typescript    5.8.2     5.8.3    5.8.3    dev
```

### Narrow stacked layout

```text
react
  current  19.1.0
  wanted   19.1.1
  latest   19.1.1

typescript
  current  5.8.2
  wanted   5.8.3
```

### Truncation policy

- Preserve package and command identity whenever possible.
- Truncate long paths from the middle when basename and root context are important.
- Include an ellipsis only when Unicode or ASCII policy provides a safe symbol.
- Never truncate error codes.
- Never truncate a value silently in machine output.
- Offer `--wide` or existing command-specific options only when the command already needs them; do not add global complexity prematurely.

## Plain renderer

The plain renderer is a first-class implementation, not rich output with ANSI stripped afterward.

Requirements:

- append-only;
- no cursor movement;
- no ANSI;
- deterministic line ordering;
- stable indentation;
- ASCII-safe when Unicode is disabled;
- same semantic content as rich static output;
- suitable for CI logs and screen readers.

## Human renderer API

```go
type StaticRenderer interface {
    Status(StatusLine) string
    KeyValues([]KeyValue) string
    Summary(Summary) string
    Notice(Notice) string
    Hint(Hint) string
    Table(TableModel) string
    PackageDeltas([]PackageDelta) string
}
```

Rendering functions should be pure where practical:

- input model + capabilities + theme -> string;
- no global environment access;
- no direct writes;
- no clocks;
- no random data.

## Implementation phases

### Phase 1 — Capability probes

Implement terminal and environment probes behind interfaces. Add platform-specific files only where required.

### Phase 2 — Mode-specific settings

Resolve color, Unicode, progress, theme, background, and accessibility into immutable settings.

### Phase 3 — Theme and symbols

Implement palettes and fallback symbols. Review contrast manually and with automated checks where feasible.

### Phase 4 — Static components

Implement status, key-value, notice, hint, summary, package-delta, and table components.

### Phase 5 — Width and wrapping

Implement visible-cell width, wrapping, middle truncation, stacked fallback, and tests.

### Phase 6 — Adoption pilot

Migrate a small low-risk command set:

```text
m version
m features
m project
m config list/get
```

Do not migrate install or runner progress in this plan.

## Test matrix

### Capability tests

| stderr TTY | stdout TTY | CI | TERM | accessible | expected |
|---:|---:|---:|---|---:|---|
| yes | yes | no | normal | no | rich capable |
| yes | no | no | normal | no | rich diagnostics, plain result |
| no | no | no | normal | no | plain |
| yes | yes | yes | normal | no | plain by auto |
| yes | yes | no | dumb | no | plain |
| yes | yes | no | normal | yes | accessible append-only |

### Rendering tests

- light theme snapshots;
- dark theme snapshots;
- accessible theme snapshots;
- no-color snapshots;
- Unicode snapshots;
- ASCII snapshots;
- widths 40/60/80/120;
- long package names;
- long Windows and POSIX paths;
- CJK and combining characters;
- invalid UTF-8 replacement policy;
- no ANSI when styles disabled.

### Startup benchmark

Measure:

```text
m version
m --help
mx version
```

Record before/after median and p95 over repeated local runs. No hard threshold is frozen until baseline evidence exists, but regressions must be reviewed.

## Acceptance criteria

- Capabilities are captured once and injectable.
- Plain output never contains ANSI or cursor control.
- Rich output is width-aware.
- State is understandable without color.
- ASCII fallback is complete.
- Static components are reusable and pure.
- Pilot commands no longer assemble ad-hoc styles.
- Lip Gloss dependency remains isolated to presentation packages.
- Windows, Linux, and macOS snapshots pass.

## Risks

| Risk | Mitigation |
|---|---|
| Background detection is unreliable | explicit theme override and conservative palette |
| Unicode renders inconsistently | capability gate and ASCII fallback |
| Tables become noisy | borderless default and stacked narrow layout |
| Styling leaks into redirected output | plain renderer and no-ANSI tests |
| Dependency increases startup size/time | benchmark and dependency review |
| Terminal-width calculations break on complex text | cell-width library behavior tests and normalization |

## Estimated effort

**14–22 focused engineering hours**.

## References

- Lip Gloss v2: https://github.com/charmbracelet/lipgloss
- Charm v2 overview: https://charm.land/blog/v2/
