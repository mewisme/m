<!--
Ownership: curated terminal help for `m help configuration`.
Authoritative: docs/config.md
-->

# Configuration

Mew loads layered JSONC configuration with provenance on every effective value.

## Precedence (low → high)

1. built-in defaults
2. user `config.jsonc`
3. project `m.jsonc`
4. `MEW_*` environment
5. CLI flags / `--config` overlay

## Inspect

```text
m config list
m config list --sources
m config get store.dir
m config get store.dir --source
m config path
m config paths
```

`m config list` columns: `KEY`, `VALUE`, `VALUES` (pipe-joined allowed values,
or `-`). `--sources` adds `SOURCE` and `PATH` (user layer shown as `user`).

## Write

```text
m config set <key> <value>           # user config (default)
m config set <key> <value> --local   # <project-root>/m.jsonc
m config set <key> <value> --file p  # exact path vs --cwd
m config unset <key> [--local|--file]
```

`--global` is a deprecated alias for user scope. `--config` is read-overlay only.
`m config edit` is deferred.
## UI-related keys

| Key | Meaning |
|---|---|
| `ui.output` | `auto` \| `rich` \| `plain` \| `json` \| `ndjson` \| `silent` |
| `ui.color` | color policy |
| `ui.theme` | `auto` \| `light` \| `dark` \| `accessible` \| `none` — Glamour help style |
| `ui.accessible` | accessible append-only mode |
| `ui.interactive` | prompt policy |
| `ui.pager` | optional pager command for topic help |

## See also

- docs/config.md
- docs/cli.md
