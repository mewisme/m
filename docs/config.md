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
| Global | `$MEW_CONFIG_DIR/config.jsonc` or platform default (`~/.config/github.com/mewisme/m/…`, `%AppData%\mew\…`) |
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

Environment:

| Variable | Effect |
|---|---|
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
The spec is an immutable snapshot of load inputs: CWD, resolved absolute
`--config` path (project vs global), env slice, and CLI overlays (`offline`,
`prefer-offline`, etc.).

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
