<!--
Ownership: curated terminal help for `m help configuration`.
Authoritative: docs/config.md
-->

# Configuration

Mew loads layered JSONC configuration with provenance on every effective value.

## Precedence (low → high)

1. built-in defaults
2. global `config.jsonc`
3. project `m.jsonc`
4. `MEW_*` environment
5. CLI flags / `--config` overlay

## Inspect

```text
m config list
m config list --sources
m config get store.dir
```

## UI-related keys

| Key | Meaning |
|---|---|
| `ui.output` | `auto` \| `rich` \| `plain` \| `json` \| `ndjson` \| `silent` |
| `ui.color` | color policy |
| `ui.accessible` | accessible append-only mode |
| `ui.interactive` | prompt policy |
| `ui.pager` | optional pager command for topic help |

## See also

- docs/config.md
- docs/cli.md
