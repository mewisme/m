# Charm dependency evaluation (UX-0002 Lip Gloss pin)

**Date:** 2026-07-31  
**Status:** Lip Gloss v2.0.5 pinned for `internal/presentation` only.

## Proposed modules

| Module | Purpose | UX plan | Status |
|---|---|---|---|
| `charm.land/lipgloss/v2` | Static styling, tables, colors | UX-0002 | **Pinned v2.0.5** |
| `charm.land/bubbletea/v2` | Live inline renderers | UX-0004 | Deferred |
| `charm.land/bubbles/v2` | Progress spinners, components | UX-0004 | Deferred |
| `charm.land/huh/v2` | Accessible prompts | UX-0006 | Deferred |
| `charm.land/glamour/v2` | Markdown help | UX-0007 | Deferred |

## License

Lip Gloss v2.0.5 is MIT (`LICENSE` in module cache). Transitive Charm /
clipperhouse modules used by Lip Gloss are also MIT-compatible for Mew.
`go run ./tools/check-license` reports `ok: LICENSE is Apache-2.0`.

## Integration boundary

- Allowed importers: `internal/presentation` (and tests).
- `internal/cli` calls presentation APIs only (no direct Charm imports).
- Forbidden: domain packages (`internal/app`, `internal/runner`,
  `internal/transaction`, `internal/resolver`, `internal/linker`,
  `internal/store`, `internal/lifecycle`, …).
- Enforced by `internal/archcheck` import-edge tests.
- Do **not** import `charm.land/lipgloss/v2/compat` (global stdin/stdout probes).
- Do **not** call Lip Gloss global `Print`/`Fprint` helpers; use `Style.Render`
  / presentation renderers only.
- No package `init` terminal I/O in Mew code; avoid Lip Gloss global writers.

## Pin evidence (Windows, `CGO_ENABLED=0`)

| Binary | Before (bytes) | After (bytes) | Delta |
|---|---:|---:|---:|
| `cmd/m` | 17,117,184 | 18,257,920 | +1,140,736 (~1.09 MiB) |
| `cmd/mx` | 13,530,624 | 14,676,480 | +1,145,856 (~1.09 MiB) |

Startup smoke (local PowerShell, 7 runs, average wall time; no hard fail threshold):

| Command | Before avg (ms) | After avg (ms) |
|---|---:|---:|
| `m version` | 48.0 | 47.2 |
| `m --help` | — | 23.5 |
| `mx version` | — | 43.9 |

No material cold-start regression observed for `m version` on this host.

## Allowlist

`tools/allowlist/modules.txt` updated for Lip Gloss and transitive modules,
including: `charm.land/lipgloss/v2`, `github.com/charmbracelet/colorprofile`,
`github.com/charmbracelet/ultraviolet`, `github.com/charmbracelet/x/{ansi,term,termios,windows,exp/golden}`,
`github.com/clipperhouse/{displaywidth,stringish,uax29/v2}`,
`github.com/aymanbagabas/go-udiff`, `github.com/bits-and-blooms/bitset`,
`github.com/lucasb-eyer/go-colorful`, `github.com/mattn/go-runewidth`,
`github.com/muesli/cancelreader`, `github.com/rivo/uniseg`,
`github.com/xo/terminfo`, `golang.org/x/{exp,sync}`.

## Fallback

- `MEW_PRESENTATION=legacy` / hidden `--presentation-legacy` forces pre-Charm
  human reporter path until UX-0008 removes the switch.
- `auto` downgrades to plain on non-TTY, CI, `TERM=dumb`, or accessible mode.
- Plain renderer is first-class (zero ANSI), not rich-with-strip.

## References

- https://charm.land/libs/
- https://charm.land/blog/v2/
- Module: `charm.land/lipgloss/v2@v2.0.5` (Go ≥ 1.25; repo Go 1.26.5)
