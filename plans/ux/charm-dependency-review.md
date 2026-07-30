# Charm dependency evaluation

**Date:** 2026-07-31  
**Status:** Lip Gloss v2.0.5 + Bubble Tea v2.0.8 + Bubbles v2.1.1 pinned for
`internal/presentation` only.

## Proposed modules

| Module | Purpose | UX plan | Status |
|---|---|---|---|
| `charm.land/lipgloss/v2` | Static styling, tables, colors | UX-0002 | **Pinned v2.0.5** |
| `charm.land/bubbletea/v2` | Live inline install renderer | UX-0004 | **Pinned v2.0.8** |
| `charm.land/bubbles/v2` | Spinner (progress bar reserved) | UX-0004 | **Pinned v2.1.1** |
| `charm.land/huh/v2` | Accessible prompts | UX-0006 | Deferred |
| `charm.land/glamour/v2` | Markdown help | UX-0007 | Deferred |

## License

Lip Gloss, Bubble Tea, and Bubbles are MIT (`LICENSE` in module cache). Transitive
Charm / clipperhouse modules are MIT-compatible for Mew.
`go run ./tools/check-license` reports `ok: LICENSE is Apache-2.0`.

## Integration boundary

- Allowed importers: `internal/presentation` (and tests).
- `internal/cli` calls presentation APIs only (no direct Charm imports).
- Forbidden: domain packages (`internal/app`, `internal/runner`,
  `internal/transaction`, `internal/resolver`, `internal/linker`,
  `internal/store`, `internal/lifecycle`, …).
- Enforced by `internal/archcheck` import-edge tests.
- Do **not** import `charm.land/lipgloss/v2/compat` (global stdin/stdout probes).
- Live Bubble Tea programs: `tea.WithOutput(stderr)`, `tea.WithInput(nil)`,
  no alternate screen, no signal ownership.
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

## Allowlist

`tools/allowlist/modules.txt` includes: `charm.land/lipgloss/v2`,
`charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, and transitive Charm /
clipperhouse modules listed in the file.

## Fallback

- `MEW_PRESENTATION=legacy` / hidden `--presentation-legacy` forces pre-Charm
  human reporter path until UX-0008 removes the switch.
- `auto` progress downgrades to plain on non-TTY, CI, `TERM=dumb`, or accessible mode.
- Forced `--progress=always` without a stderr TTY fails before mutation start
  (`RichUnsupportedError`).
- Mid-mutation live renderer failure: product cleanup completes; presentation
  error surfaces afterward.
- Plain renderer is first-class (zero ANSI), not rich-with-strip.

## References

- https://charm.land/libs/
- https://charm.land/blog/v2/
- Modules: `charm.land/lipgloss/v2@v2.0.5`, `charm.land/bubbletea/v2@v2.0.8`,
  `charm.land/bubbles/v2@v2.1.1` (Go ≥ 1.25; repo Go 1.26.5)
