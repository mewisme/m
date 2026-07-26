# Install

`m install`, `m add`, `m remove`, and `m ci` implement the first end-to-end
install path (MVP **0016**).

## Pipeline

```text
resolve → fetch → link (hoisted copy) → write m.lock → publish node_modules
```

Work happens under `<project>/.mew-stage/` until publish. On failure after
fetch begins, the live `node_modules` tree is left unchanged.

## Commands

| Command | Flags | Notes |
|---|---|---|
| `m install` / `i` | `--prod`, `--frozen-lockfile`, `--dry-run`, `--json` | Full install |
| `m add <pkg>` | `-D`, `-E`, `--json` | Manifest + lock + install |
| `m remove <pkg>` / `rm` | `--json` | Remove dep + reinstall |
| `m ci` | `--prod`, `--json` | `install --frozen-lockfile` |

## Layout

- **Hoisted** copy-based `node_modules` (no hardlinks/symlinks yet — **0018**)
- **`node_modules/.bin`** platform shims for declared `bin` entries

## Non-goals (0016)

- Lifecycle scripts (`preinstall`, `postinstall`) — **0021**
- Global content store / smart linking — **0018**
- Isolated virtual store — **0019**
- Full transactional journal — **0017** (rename-swap publish only today)
- `m update` — still stubbed

See also: [`lockfile.md`](lockfile.md), [`cli.md`](cli.md).
