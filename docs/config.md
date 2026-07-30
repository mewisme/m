# Configuration

Layered Mew configuration with source provenance on every effective value.
Format: JSONC ([ADR 0003](adr/0003-config-jsonc.md)).

## Precedence (low → high)

1. `defaults` — built-in
2. `global` — user `config.jsonc`
3. `project` — project `m.jsonc`
4. `env` — `MEW_*` and mapped env keys
5. `cli` — `--offline`, `--prefer-offline`, `--config` overlays

## Files

| Layer | Path |
|---|---|
| Global | `$MEW_CONFIG_DIR/config.jsonc` or platform default (`~/.config/github.com/mewisme/mew/…`, `%AppData%\mew\…`) |
| Project | `<project-root>/m.jsonc` |

`MEW_HOME` influences derived dirs when specific overrides are unset.

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
| `ui.output` | string | `auto` — presentation output mode |
| `ui.color` | string | `auto` |
| `ui.progress` | string | `auto` |
| `ui.unicode` | string | `auto` |
| `ui.interactive` | string | `auto` — `auto`\|`always`\|`never` prompt policy (`MEW_INTERACTIVE`) |
| `ui.accessible` | bool | `false` — numbered prompts + append-only output (`MEW_ACCESSIBLE`) |
| `ui.summary` | bool | `true` |
| `ui.theme` | string | empty/`auto` — `auto`\|`light`\|`dark`\|`accessible`\|`none` |
| `ui.pager` | string | empty — optional pager command for `m help` topics (`MEW_PAGER` / `PAGER`) |
| `log.level` | string | `error` |

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
| `network.timeout_ms` | int | `60000` |
| `network.proxy` | string | empty — http/https proxy URL (else env) |
| `network.ca_file` | string | empty — optional PEM CA bundle |
| `registry.auth_token_env` | string | empty — **env var name only**, never a secret |

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
m config get <key>
m config set <key> <value> [--global]
m config list [--sources]
```

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
