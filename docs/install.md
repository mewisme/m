# Install

`m install`, `m add`, `m remove`, and `m ci` implement the first end-to-end
install path (MVP **0016**).

## Pipeline

```text
resolve → fetch (blob cache) → [optional: import to global store] → link (hoisted copy or smart link) → validate → commit (journal) → snapshot
```

Work happens under `<project>/.mew/txn/<id>/stage/` until commit. On failure,
live `package.json`, `m.lock`, and `node_modules` are restored from journal
backups.

## Commands

| Command | Flags | Notes |
|---|---|---|
| `m install` / `i` | `--prod`, `--frozen-lockfile`, `--dry-run`, `--journal`, `--json` | Full install |
| `m add <pkg>` | `-D`, `-E`, `--json` | Manifest + lock + install (atomic commit) |
| `m remove <pkg>` / `rm` | `--json` | Remove dep + reinstall |
| `m ci` | `--prod`, `--json` | `install --frozen-lockfile` |
| `m snapshot` | `list`, `restore <id>` | Snapshot history |
| `m recover` | — | Recover interrupted transaction |
| `m rollback` | — | Restore previous snapshot |

## Layout

- **Hoisted** copy-based `node_modules` (no hardlinks/symlinks yet — **0018**)
- **`node_modules/.bin`** platform shims for declared `bin` entries

## Non-goals (0016)

- Lifecycle scripts (`preinstall`, `postinstall`) — **0021**
- Global content store / smart linking — **0018**
- Isolated virtual store — **0019**
- `m update` — still stubbed

See also: [`lockfile.md`](lockfile.md), [`transaction.md`](transaction.md), [`cli.md`](cli.md).
