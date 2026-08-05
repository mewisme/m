# Configuration

Layered Mew configuration with source provenance on every effective value.
Format: JSONC ([ADR 0003](adr/0003-config-jsonc.md)).

## Precedence (low → high)

1. `defaults` — built-in
2. `user` — user `config.jsonc` (internal provenance label remains `global`)
3. `project` — project `m.jsonc`
4. `env` — `MEW_*` and mapped env keys
5. `cli` — `--offline`, `--prefer-offline`, `--config` overlays

### Raw layers vs effective configuration

Two different questions, two different answers:

- A **raw layer** (`user`, `project`) is what one file literally contains. It is
  what `m config set --local` writes and what `m config get` reports as
  `Previous` for that scope. A key absent from that file is absent, whatever any
  other layer says.
- **Effective** configuration is the merged result across all five layers, with
  the winning `Source` and `Path` recorded per value. It is what commands
  actually act on.

So writing a key to the user layer does not guarantee it takes effect: a project
file, an environment variable, or a CLI overlay above it still wins. `m config
set` prints both the previous scope value and the resulting effective value so
that difference is visible instead of surprising.

### Which scopes a key accepts

The key registry (`ConfigKeySpec`) is the single authority for each key's type,
default, secret status, deprecation, and the scopes it may be written to. Scope
enforcement reads `Scopes` from the registry, so a key cannot be declared in one
place and enforced from a stale second list. Writing a key to a scope it does not
declare fails with `ERR_M_USAGE` naming the allowed scopes.

## Files

| Layer | Path |
|---|---|
| User | `$MEW_CONFIG_DIR/config.jsonc`, else `$MEW_HOME/config/config.jsonc`, else platform default (`~/.config/mew/…`, `%AppData%\mew\…`) |
| Project | `<project-root>/m.jsonc` |

Portable setups set `MEW_HOME=…` (writes land in `$MEW_HOME/config/config.jsonc`).
Config is never written beside the `m` executable.

`MEW_HOME` influences derived dirs when specific overrides are unset.

### Product directories

One product directory name (`mew`) with fixed `config`, `cache`, and `store`
subdirectories on every platform. The names live in one place
(`internal/config/dirs.go`), so config, cache, and store cannot drift apart.

| Platform | Base |
|---|---|
| Linux | `$XDG_CONFIG_HOME/mew`, `$XDG_CACHE_HOME/mew`, `$XDG_DATA_HOME/mew` (else `~/.config`, `~/.cache`, `~/.local/share`) |
| macOS | `~/Library/Application Support/mew`, `~/Library/Caches/mew` |
| Windows | `%LocalAppData%\mew` (cache, store), `%AppData%\mew` (config) |

Precedence: `MEW_CONFIG_DIR` / `MEW_CACHE_DIR` / `MEW_STORE_DIR` override one
root each; `MEW_HOME` derives all three; platform defaults come last.

A store written by an older version under a previous vendor path is still
discovered and adopted, so upgrading does not orphan an existing store. An
explicit `MEW_STORE_DIR` always outranks that discovery, and the canonical path
wins when both exist.

## Owned keys (v1)

| Key | Type | Default |
|---|---|---|
| `registry` | string | `https://registry.npmjs.org` |
| `registries.@scope` | string | scoped registry URL (object `registries` in JSONC) |
| `install.linker` | string | `auto` (`auto` \| `hoisted` \| `isolated`) |
| `resolve.autoInstallPeers` | bool | `false` — when true, enqueue missing peers from the importer |
| `resolve.strictPeerDependencies` | bool | `true` — fail when required peers are unsatisfied |
| `offline` | bool | `false` |
| `prefer-offline` | bool | `false` |
| `cache.dir` | string | platform cache (empty = derive) |
| `store.dir` | string | platform store (empty = derive) |
| `link.use_global_store` | bool | `false` — experimental global store + smart linker (or `MEW_EXPERIMENTAL_GLOBAL_STORE=1`) |
| `lifecycle.enabled` | bool | `false` — run lifecycle scripts (or `MEW_EXPERIMENTAL_LIFECYCLE=1`) |
| `lifecycle.ignore_scripts` | bool | `false` — skip all lifecycle scripts |
| `lifecycle.script_trust` | string | `deny` (`allow` \| `deny` \| `ask`) — `ask` prompts when `--interactive` permits; choices: deny / allow-once (session) / trust-project |
| `workspaces.enabled` | bool | `false` — workspace install/filter (or `MEW_EXPERIMENTAL_WORKSPACES=1`) |
| `runner.direct_scripts.enabled` | bool | `false` — direct `m <script>` shortcuts (or `MEW_EXPERIMENTAL_DIRECT_SCRIPTS=1`) |
| `runner.exec.direct_dispatch.enabled` | bool | `false` — direct `m <binary>` shortcuts with verified bin metadata (or `MEW_EXPERIMENTAL_EXEC_DIRECT_DISPATCH=1`) |
| `runner.mx.cache.retention_days` | int | `7` — default prune retention for mx execution environments |
| `runner.mx.cache.dir` | string | empty — mx cache root (default `<cache>/mx`) |
| `ui.pager` | string | empty — optional pager command for `m help` topics (`MEW_PAGER` / `PAGER`) |
| `ui.theme` | string | `auto` — terminal color palette: `auto|light|dark`. `auto` follows OS appearance; `light`/`dark` force the palette. Accessibility and no-color policies override. |
| `log.level` | string | `error` |

Presentation modes, streams, and rollout stages:
[`architecture/cli-presentation.md`](architecture/cli-presentation.md).
Accessible mode: [`accessibility.md`](accessibility.md).

Presentation is controlled exclusively by CLI flags (`--output`, `--no-color`,
`--no-progress`, `--ascii`, `--accessible`, `--no-summary`). Environment
variables and config keys no longer influence presentation output.

### `ui.theme` wiring

`ui.theme` is the only config key that influences presentation. It is resolved
during invocation bootstrap (not at presentation resolve time) and stored in the
`config.Effective` snapshot. The resolved value (`auto`, `light`, or `dark`)
passes through `presentation.ResolveTheme`:

- `light` → `ThemeLight` (always light palette)
- `dark` → `ThemeDark` (always dark palette)
- `auto` → OS dark-mode detector → `ThemeDark` or `ThemeLight`; falls back to
  `ThemeLight` when detection is unavailable or fails

The resolved `ThemeMode` is embedded in `EffectiveSettings` and consumed by the
static renderer and live UI sink. Accessibility flags and `--no-color` override
the palette by disabling color output entirely — the theme mode still reflects
the user's preference for any non-color styling.

Environment:

| Variable | Effect |
|---|---|
| `MEW_EXPERIMENTAL_LIFECYCLE` | Maps to `lifecycle.enabled` |
| `MEW_EXPERIMENTAL_WORKSPACES` | Maps to `workspaces.enabled` |
| `MEW_MX_CACHE_DIR` | Overrides `runner.mx.cache.dir` |
| `MEW_EXPERIMENTAL_EXEC_DIRECT_DISPATCH` | Maps to `runner.exec.direct_dispatch.enabled` |
| `MEW_EXPERIMENTAL_GLOBAL_STORE` | Maps to `link.use_global_store` |
| `MEW_EXPERIMENTAL_ISOLATED_LINKER` | Set to `1` to allow `install.linker=isolated` |
| `MEW_RESOLVE_AUTO_INSTALL_PEERS` | Maps to `resolve.autoInstallPeers` |
| `MEW_RESOLVE_STRICT_PEER_DEPS` | Maps to `resolve.strictPeerDependencies` |

Every environment mapping is driven by the key registry, so a mapped variable is
always a real key and its value is coerced to that key's declared type. A value
that cannot be coerced, or that fails the key's validation, fails load with
`ERR_M_CONFIG` naming the variable — it is never silently dropped.

Network:

| Key | Type | Default |
|---|---|---|
| `network.timeout` | duration | `60s` — Go duration string (`60s`, `1m30s`) |
| `network.proxy` | string | empty — http/https proxy URL (else env) |
| `network.ca_file` | string | empty — optional PEM CA bundle |
| `registry.auth_token_env` | string | empty — **env var name only**, never a secret |

`network.timeout` is a duration, and it is applied to the HTTP client used for
registry, fetch, and publish traffic. The former `network.timeout_ms` integer key
is migrated: `m config migrate` rewrites `network.timeout_ms: 5000` to
`network.timeout: "5s"`.

Unknown keys under owned namespaces fail load with `ERR_M_CONFIG`.

## Credentials

Store references only (`registry.auth_token_env=NPM_TOKEN`). Raw tokens in config
values are rejected. `m config list --sources` redacts secret-shaped strings.

At runtime, `config.AuthToken` resolves the named variable from the invocation
`EnvSnapshot` on `app.Context` (not ambient `os.Environ()`). Resolver packument
fetch and tarball download both use this token. Windows lookups are
case-insensitive (`NPM_TOKEN` config matches `npm_token` in the snapshot).

### Explicit empty environment

`app.Options.Env: []string{}` produces an **initialized-empty** snapshot: path
overrides, offline flags, and registry tokens do not inherit the host process
environment. Use this in tests and hermetic CI to prove snapshot isolation.
`Options.Env == nil` (CLI default) snapshots `os.Environ()` once at `app.New`.

## Commands

```text
m config get <key> [--source]
m config set <key> <value> [--local | --file <path>] [--global]
m config unset <key> [--local | --file <path>] [--global]
m config list [--sources]
m config path [--local | --file <path>] [--global]
m config paths
```

### Write scopes

| Scope | Flag | Path |
|---|---|---|
| User (default) | _(none)_ | user `config.jsonc` via `GlobalConfigPath` |
| Project | `--local` | `<project-root>/m.jsonc` (requires `package.json`; no cwd fallback) |
| Explicit | `--file <path>` | path resolved against CLI `--cwd` |

`--local`, `--file`, and `--global` are mutually exclusive.
`--global` is a deprecated alias for the default user scope.
`--config` remains a read-only CLI overlay; it is not a write target.

`m config set` now writes user configuration by default.
Use `--local` to write `<project-root>/m.jsonc`.

`m config path` prints one write-target path. `m config paths` prints User and
Project lines (`Project` is `unavailable` when no project root is found).

`m config get --source` prints the value plus Source/Path. In `--output=json`,
the object is `{key,value,source,path}` with source `user` for the user layer.

`m config list` prints a table with `KEY`, `VALUE`, and `VALUES` (pipe-joined
allowed values, or `-` when free-form). `--sources` adds `SOURCE` and `PATH`
(user layer displayed as `user`).

`m config edit` is deferred (no editor runner yet).

Set/unset edit JSONC in place and **preserve comments**: the member is spliced
into the existing bytes, so surrounding `//` and `/* */` comments, key order, and
formatting survive. Files without comments are rewritten as deterministic pretty
JSON. `m config unset` also prunes a parent object it leaves empty. See
[ADR 0003](adr/0003-config-jsonc.md).

`m config migrate` is the exception: it moves keys between locations and cannot
place existing comments correctly, so it refuses a commented file and says to
apply the renames with `m config set` / `m config unset` instead.

`m config set` reports the value that was previously in the **target scope** and
the new **effective** value, which can differ when a higher layer (env or CLI)
still overrides the key being written.

## Global flags

| Flag | Effect |
|---|---|
| `--cwd` | Discovery / project root |
| `--config <path>` | Extra JSONC overlay (CLI layer) |
| `--offline` | Force `offline=true` |
| `--prefer-offline` | Force `prefer-offline=true` |

## Foreign branded config

For **mew** identity, `.npmrc`, `.yarnrc*`, `pnpm-workspace.yaml`, and
`.pnpmfile.cjs` are **not** config authority. See [`identity.md`](identity.md).

## Invocation bootstrap

Every normal invocation runs one configuration bootstrap before command dispatch.
Bootstrap merges defaults, user, project, environment, and CLI layers into one
`config.Effective` snapshot. That snapshot is immutable for the lifetime of the
invocation — commands do not re-read files or re-merge layers mid-invocation. The
bootstrap also resolves `ui.theme` and initializes the presentation controller
with the resolved theme mode.

A bootstrap failure (missing required file, malformed JSONC, duplicate keys,
unknown required key) exits non-zero before command dispatch. Bootstrap happens
once; if the bootstrap config is valid but a specific command needs a reload
(e.g., after a mutation lock is acquired), the reload uses the same frozen
`ConfigLoadSpec` with only `ProjectRoot` updated.

## Validation

### Shared validation

`config.Load` validates every loaded layer through a single `Validator` shared
across all scopes. The validator is the key registry (`ConfigKeySpec`): every key
must be a known owned key or within a known owned namespace. Unknown keys under
owned namespaces fail load with `ERR_M_CONFIG`. Keys outside owned namespaces are
preserved as passthrough (they do not fail load but are not in the effective map).

### Non-zero exit on invalid config

Invalid configuration fails non-zero. A malformed JSONC file, a duplicate key,
an unknown key under an owned namespace, or a value that fails type coercion all
produce `ERR_M_CONFIG` at load time. The error message names the file, the
specific key or line, and the reason — it does not produce a generic "invalid
config" message.

### Duplicate JSONC key rejection

`m.jsonc` and `config.jsonc` are parsed as JSONC. The parser rejects files
containing duplicate keys within the same object. The error names the duplicate
key and the file. This is a parse-time check, not a merge-time check — it catches
duplicates before any layer merging occurs.

### `m config validate`

`m config validate` loads and validates all layers without mutating anything.
Exit 0 means every layer is syntactically valid, every key in owned namespaces
is known, and every value passes type coercion. Exit non-zero means at least
one layer failed. The command reports the first error per layer.

`m config validate --strict` also requires that every key in every layer is
resolvable as a known config key — passthrough keys in non-owned namespaces
become errors.

## Migration

`m config migrate` rewrites legacy key names to their current canonical forms
deterministically. The migration is a pure function of the input file: given
the same input, it produces the same output every time. Known migrations:

- `network.timeout_ms` (int, ms) → `network.timeout` (duration string)
- Renamed key paths (no semantic change, only the key name)

Migration refuses to operate on files with comments (JSONC), since it rewrites
the entire document and cannot place existing comments correctly. The error
message directs the user to apply renames manually with `m config set` /
`m config unset`. A migration that would produce output identical to input is
a no-op (exit 0, no file written).

## Secret redaction

`m config list --sources` redacts values for keys registered as secret
(`Secret: true` in `ConfigKeySpec`) and any value that matches a secret-shaped
pattern (e.g., strings starting with `npm_` containing long base64-like
suffixes). Redacted values display as `<redacted>`.

The redaction is output-only: the underlying config value is intact and usable.
Structured output (`--output=json`) also redacts — the JSON value for a secret
key is the string `"<redacted>"`. Credential references
(`registry.auth_token_env`) are not themselves secrets (they name an env var),
but the resolved token is never displayed.

## Edit recovery

`m config edit` opens the target file in the configured editor (`EDITOR` or
`VISUAL`). If the editor exits non-zero or the file is unreadable after the
editor returns, the command restores the previous content from a backup taken
before the editor launched. If the backup restore also fails, the command
reports both errors and the path to the backup file so manual recovery is
possible. The editor subprocess inherits stdin/stdout/stderr so interactive
editors work.

## Reset confirmation

`m config reset` drops the target config file and replaces it with an empty
valid JSONC document (`{}` with a trailing newline). This is a destructive
operation. In interactive terminals, reset prompts for confirmation unless
`--yes` is passed. In non-interactive terminals (CI), reset without `--yes`
fails with `ERR_M_USAGE` explaining the confirmation requirement.

## Repair with malformed config

Commands that need to repair or reset configuration (`m config edit`,
`m config reset`, `m config migrate --force`) remain usable even when the
current configuration is malformed. These commands bypass normal config loading
and operate on the raw file bytes. Other config commands (`get`, `set`, `list`,
`validate`) require a loadable configuration and fail with `ERR_M_CONFIG` if
any layer is malformed.

## Mutation reload (`ConfigLoadSpec`)

`app.New` captures a `config.LoadSpec` on `app.Context` (`ConfigLoadSpec` field).
The spec is an immutable snapshot of load inputs: invocation CWD, discovered
project root, resolved absolute `--config` path (project vs global), frozen global
config path from env snapshot, env slice, and CLI overlays (`offline`,
`prefer-offline`, etc.).

Relative `--config` paths resolve against invocation CWD (`--cwd`), not the process
working directory after `app.New`. Explicit config classification uses
`config.IsPathWithin(projectRoot, path)` against the discovered project root
(`project.FindRoot`), not invocation CWD alone.

`GlobalConfigPathFromEnv` resolves the default global config path from the env
snapshot captured at `app.New`. Mutation reload uses the stored `GlobalPath` and
does not call `os.Getenv` for global discovery.

`MutationSession.ReloadEffectiveConfig` clones the stored spec and may only change
`ProjectRoot` after `FindRoot`. It must not call `os.Environ()` or rebuild CLI
overlays from the effective map.

### Explicit `--config` strictness

When `--config` resolves inside the project tree, `RequireProjectConfig` is set;
when it resolves outside (global overlay), `RequireGlobalConfig` is set. A missing
explicit file fails load with `ERR_M_CONFIG` (`explicit config file missing: …`).
Default discovery (`m.jsonc` absent) remains optional.

### Env snapshot

If `app.Options.Env` is nil, `app.New` snapshots `os.Environ()` once into
`ConfigLoadSpec`. Later `t.Setenv` / process env changes do not affect mutation
reload.

`Options.Env: []string{}` is an initialized-empty snapshot: `EnvSnapshot.Lookup`
never falls through to ambient env. `config.Load` preserves initialized-empty
snapshots and does not re-read `os.Environ()`.
