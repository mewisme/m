# Stable Naming Conventions

Frozen identifiers for Mew public surfaces. Changes require an ADR and compatibility-axis update.

## Binaries

| Binary | Role | Alias status |
|---|---|---|
| `m` | Primary toolchain CLI | Required |
| `mx` | Package executable runner (Mewx) | Required |
| `mew` | Optional installer alias for `m` | Open decision (0072) |
| `mewx` | Optional installer alias for `mx` | Open decision (0072) |

## Lockfiles

| File | When used |
|---|---|
| `m.lock` | Mew-native projects; default for greenfield |
| `nub.lock` | Nub-identity projects; first-class compatibility |
| `package-lock.json` | npm identity |
| `pnpm-lock.yaml` | pnpm identity |
| `yarn.lock` | Yarn identity |
| `bun.lock` | Bun identity |

## Configuration

| File / key | Scope | Notes |
|---|---|---|
| `m.jsonc` | Project | Neutral Mew project config (name subject to ADR) |
| `.github.com/mewisme/m/` | Project | Mew-owned auxiliary data (policy, local state) |
| Global config | User | `~/.config/github.com/mewisme/m/config.jsonc` (Linux/macOS); `%AppData%\mew\config.jsonc` (Windows) |
| Pass-through | Project | `.npmrc`, incumbent manager configs via compatibility adapters only |

Mew does not read another package manager's branded config as authority for an Mew-identity project unless explicitly importing.

## Cache and store directories

Default roots (overridable; see environment variables):

| Purpose | Linux | macOS | Windows |
|---|---|---|---|
| Global cache | `$XDG_CACHE_HOME/mew` or `~/.cache/mew` | `~/Library/Caches/mew` | `%LocalAppData%\mew\cache` |
| Global content store | under cache or `~/.local/share/github.com/mewisme/m/store` | `~/Library/Application Support/github.com/mewisme/m/store` | `%LocalAppData%\mew\store` |
| Registry metadata cache | `<cache>/registry` | same | same |
| Transaction journal | `<cache>/journal` | same | same |

## Environment variables

| Variable | Purpose |
|---|---|
| `MEW_HOME` | Override Mew home root (config, cache, store derivation) |
| `MEW_STORE_DIR` | Override global content store location |
| `MEW_CACHE_DIR` | Override cache root |
| `MEW_CONFIG_DIR` | Override global config directory |
| `MEW_EXPERIMENTAL_<NAME>` | Enable experimental feature `<name>` |
| `MEW_LOG_FORMAT` | Structured log format (`default`, `json`, `ndjson`) |
| `MEW_DEBUG` | Verbose internal diagnostics |
| `M_LOG` | Shorthand for debug-level logging (matches Nub-style `M_LOG=debug`) |
| `MEW_SHIM_BYPASS` | Disable shim recursion guard (testing only) |

Incumbent manager variables (`npm_config_*`, `NPM_CONFIG_*`, etc.) are honored through compatibility adapters when project identity matches.

## Error codes

| Pattern | Example | Notes |
|---|---|---|
| `ERR_M_<DOMAIN>_<DETAIL>` | `ERR_M_LOCKFILE_AMBIGUOUS` | Stable machine-readable CLI codes |
| Exit code `0` | Success | |
| Exit code `1` | General failure | |
| Exit code `2` | Usage / invalid arguments | Reserved range; full map in MVP 0005 |

Nub `ERR_NUB_*` codes are behavioral references for diagnostics shape, not copied identifiers unless parity requires the same user-visible code in Nub-compatibility mode.

## Reserved command surfaces (identity only)

Until MVP 0010:

```bash
m --version
mx --version
```
