# Install

`m install`, `m add`, `m remove`, and `m ci` implement the first end-to-end
install path (MVP **0016**).

## Pipeline

```text
resolve → fetch (blob cache) → [optional: import to global store] → link (hoisted or isolated) → validate → commit (journal) → snapshot
```

Work happens under `<project>/.mew/txn/<id>/stage/` until commit. On failure,
live `package.json`, `m.lock`, and `node_modules` are restored from journal
backups.

## Commands

| Command | Flags | Notes |
|---|---|---|
| `m install` / `i` | `--prod`, `--frozen-lockfile`, `--dry-run`, `--journal`, `--linker`, `--json` | Full install |
| `m add <pkg>` | `-D`, `-E`, `--linker`, `--json` | Manifest + lock + install (atomic commit) |
| `m remove <pkg>` / `rm` | `--linker`, `--json` | Remove dep + reinstall |
| `m ci` | `--prod`, `--linker`, `--json` | `install --frozen-lockfile` |
| `m update [pkg...]` | `--latest`, `--dry-run`, `--journal`, `--linker`, `--json` | Incremental lock refresh + install (transactional) |
| `m snapshot` | `list`, `restore <id>` | Snapshot history |
| `m recover` | — | Recover interrupted transaction |
| `m rollback` | — | Restore previous snapshot |

## Layout

- **Hoisted** (default): `auto` and `--linker=hoisted` — smart link or copy from global store when enabled (**0018**)
- **Isolated** (experimental): `--linker=isolated` + `MEW_EXPERIMENTAL_ISOLATED_LINKER=1` — pnpm-style `.pnpm` virtual store (**0019**); see [`linker.md`](linker.md)
- **`node_modules/.bin`** platform shims for declared `bin` entries

## Non-goals (0016)

- Lifecycle scripts (`preinstall`, `postinstall`) — **0021**

`m update` routes through the same install transaction as `m add` / `m remove`: resolve → fetch → link → validate → commit. `--latest` bumps manifest ranges in memory before resolve; `package.json` is written only at commit. `--dry-run` resolves and emits a mutation plan JSON (with `--json`) without touching disk.

See also: [`lockfile.md`](lockfile.md), [`transaction.md`](transaction.md), [`store.md`](store.md), [`linker.md`](linker.md), [`cli.md`](cli.md).
