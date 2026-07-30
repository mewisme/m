# Charm dependency evaluation

**Date:** 2026-07-31  
**Status:** Lip Gloss v2.0.5 + Bubble Tea v2.0.8 + Bubbles v2.1.1 + Huh v2.0.3 +
Glamour v2.0.1 pinned for `internal/presentation` only.

## Proposed modules

| Module | Purpose | UX plan | Status |
|---|---|---|---|
| `charm.land/lipgloss/v2` | Static styling, tables, colors | UX-0002 | **Pinned v2.0.5** |
| `charm.land/bubbletea/v2` | Live inline install renderer | UX-0004 | **Pinned v2.0.8** |
| `charm.land/bubbles/v2` | Spinner (progress bar reserved) | UX-0004 | **Pinned v2.1.1** |
| `charm.land/huh/v2` | Rich prompts (accessible adapter is Mew-owned) | UX-0006 | **Pinned v2.0.3** |
| `charm.land/glamour/v2` | Markdown help | UX-0007 | **Pinned v2.0.1** |

## License

Lip Gloss, Bubble Tea, Bubbles, Huh, and Glamour are MIT (`LICENSE` in module cache).
Transitive Charm / clipperhouse / catppuccin / goldmark / chroma modules are
MIT-compatible for Mew (bluemonday is BSD-3-Clause).
`go run ./tools/check-license` reports `ok: LICENSE is Apache-2.0`.

## Integration boundary

- Allowed importers: `internal/presentation` (and tests).
- `internal/cli` calls presentation APIs only (no direct Charm imports).
- Forbidden: domain packages (`internal/app`, `internal/runner`,
  `internal/transaction`, `internal/resolver`, `internal/linker`,
  `internal/store`, `internal/lifecycle`, …).
- Prompt contract: `internal/prompt` (stdlib only). Adapters:
  `internal/presentation/prompt`.
- Topic registry: `internal/help` (stdlib + embed only). Renderers/pager:
  `internal/presentation/help`, `internal/presentation/pager`.
- Enforced by `internal/archcheck` import-edge tests.
- Do **not** import `charm.land/lipgloss/v2/compat` (global stdin/stdout probes).
- Live Bubble Tea programs: `tea.WithOutput(stderr)`, `tea.WithInput(nil)`,
  no alternate screen, no signal ownership.
- Huh forms: `WithOutput(stderr)`, injected `WithInput`, no global I/O.
- No package `init` terminal I/O in Mew code.

## Pin evidence (Windows, `CGO_ENABLED=0`)

### Lip Gloss (UX-0002)

| Binary | Before (bytes) | After (bytes) | Delta |
|---|---:|---:|---:|
| `cmd/m` | 17,117,184 | 18,257,920 | +1,140,736 (~1.09 MiB) |
| `cmd/mx` | 13,530,624 | 14,676,480 | +1,145,856 (~1.09 MiB) |

### Bubble Tea + Bubbles (UX-0004)

Measured 2026-07-31 on the development host after pinning
`charm.land/bubbletea/v2@v2.0.8` and `charm.land/bubbles/v2@v2.1.1`:

| Binary | After Lip Gloss (bytes) | After Bubble Tea (bytes) | Delta |
|---|---:|---:|---:|
| `cmd/m` | 18,257,920 | 22,908,928 | +4,651,008 (~4.44 MiB) |
| `cmd/mx` | 14,676,480 | 19,343,872 | +4,667,392 (~4.45 MiB) |

Startup smoke (local PowerShell, 7 runs, average wall time; no hard fail threshold):

| Command | After Lip Gloss avg (ms) | After Bubble Tea avg (ms) |
|---|---:|---:|
| `m version` | 47.2 | 57.2 |
| `mx version` | 43.9 | 52.2 |

Cold-start cost is modest (~10 ms). Binary size increase is accepted for live
install progress; plain/CI paths do not start a Bubble Tea program.

### Huh (UX-0006)

Measured 2026-07-31 after pinning `charm.land/huh/v2@v2.0.3`:

| Binary | After Bubble Tea (bytes) | After Huh (bytes) | Delta |
|---|---:|---:|---:|
| `cmd/m` | 22,908,928 | 22,998,016 | +89,088 (~87 KiB) |
| `cmd/mx` | 19,343,872 | 19,434,496 | +90,624 (~88 KiB) |

Startup smoke (5 runs, average wall time):

| Command | After Bubble Tea avg (ms) | After Huh avg (ms) |
|---|---:|---:|
| `m version` | 57.2 | 68.4 |
| `mx version` | 52.2 | 61.2 |

Huh is presentation-only. Accessible numbered prompts are Mew-owned
(append-only) and do not require Huh's accessible mode. Non-interactive /
structured paths never construct a Huh form.

New allowlist entries from Huh: `charm.land/huh/v2`, `github.com/catppuccin/go`,
`github.com/charmbracelet/x/exp/ordered`, `github.com/charmbracelet/x/exp/strings`,
`github.com/mitchellh/hashstructure/v2`, plus optional transitive PTY helpers
(`x/xpty`, `creack/pty`, `x/conpty`) when present in `go.sum`.

### Glamour (UX-0007)

Measured 2026-07-31 after pinning `charm.land/glamour/v2@v2.0.1`:

| Binary | After Huh (bytes) | After Glamour (bytes) | Delta |
|---|---:|---:|---:|
| `cmd/m` | 22,998,016 | 31,422,464 | +8,424,448 (~8.03 MiB) |
| `cmd/mx` | 19,434,496 | 27,905,536 | +8,471,040 (~8.08 MiB) |

Startup smoke (5 runs, average wall time):

| Command | After Huh avg (ms) | After Glamour avg (ms) |
|---|---:|---:|
| `m version` | 68.4 | 94.4 |
| `mx version` | 61.2 | 86.5 |

Binary growth is material (goldmark/chroma/bluemonday). Mitigation already in
product policy: plain renderer is first-class; Glamour runs only for rich human
color TTYs on topic help. Accessible / CI / pipe / no-color stay plain. Topic
content is curated and embedded (no network fetch).

New allowlist entries from Glamour: `charm.land/glamour/v2`,
`github.com/alecthomas/chroma/v2`, `github.com/yuin/goldmark`,
`github.com/yuin/goldmark-emoji`, `github.com/microcosm-cc/bluemonday`,
`github.com/aymerick/douceur`, `github.com/gorilla/css`,
`github.com/dlclark/regexp2`, `github.com/charmbracelet/x/exp/slice`,
`golang.org/x/net`, `golang.org/x/text`, and related test-only assert modules
present in `go list -m all`.

## Allowlist

`tools/allowlist/modules.txt` includes: `charm.land/lipgloss/v2`,
`charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/huh/v2`,
`charm.land/glamour/v2`, and transitive Charm / clipperhouse / catppuccin /
goldmark / chroma modules listed in the file.

## Fallback

- `MEW_PRESENTATION=legacy` / hidden `--presentation-legacy` forces pre-Charm
  human reporter path until UX-0008 removes the switch.
- `auto` progress downgrades to plain on non-TTY, CI, `TERM=dumb`, or accessible mode.
- Forced `--progress=always` without a stderr TTY fails before mutation start
  (`RichUnsupportedError`).
- Mid-mutation live renderer failure: product cleanup completes; presentation
  error surfaces afterward.
- Plain renderer is first-class (zero ANSI), not rich-with-strip.
- Prompt `auto` never prompts in CI/structured/non-TTY; `always` still requires
  stdin TTY (typed error, no hang).
