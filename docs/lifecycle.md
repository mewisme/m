# Lifecycle scripts

Mew runs npm-order lifecycle scripts (`preinstall`, `install`, `postinstall`, `prepare`) after packages are linked into the staged `node_modules` tree and before install validation commits.

## Experimental gate

Lifecycle execution is **off by default**. Enable with either:

- `MEW_EXPERIMENTAL_LIFECYCLE=1`, or
- `lifecycle.enabled: true` in `m.jsonc`

## Trust policy

Default `lifecycle.script_trust` is **`deny`**: unknown packages fail closed until approved.

```sh
m trust <package>
m approve-builds <package>
m trust --interactive
```

Untrusted blocks surface `ERR_M_POLICY`.

## Skipping scripts

```sh
m install --ignore-scripts
```

Config: `lifecycle.ignore_scripts: true`

Dry-run installs never execute lifecycle scripts.

## Restricted execution environment

Scripts run under a **restricted execution environment** (not a filesystem or network sandbox):

| Capability | Enforced |
|------------|----------|
| Package CWD | yes |
| Controlled PATH (`node_modules/.bin` prepended) | yes |
| Stripped env (explicit snapshot; secrets removed) | yes |
| Timeout (`lifecycle.script_timeout`, default `10m`) | yes |

`lifecycle.script_timeout` is read only from the **effective config** loaded at
invocation (`m.jsonc`, frozen env snapshot, CLI). Ambient `MEW_LIFECYCLE_SCRIPT_TIMEOUT`
after `app.New` does not override project config. Invalid values surface
`ERR_M_CONFIG`.

| Process-tree kill on timeout/cancel | best-effort |
| Filesystem isolation | **no** |
| Network isolation | **no** |

Env rules:

- Production lifecycle always passes an **explicit** env snapshot from the app context (`Explicit: true`).
- `Explicit` + empty vars → only controlled `PATH`/`Path` (no host inheritance).
- Secrets stripped include `NPM_TOKEN`, `GH_TOKEN`, `GITLAB_TOKEN`, `AWS_*`, `AZURE_*`, `GOOGLE_*`, and keys containing `TOKEN`, `SECRET`, `PASSWORD`, `PRIVATE_KEY`.

Shell dispatch: `sh -c` on Unix, `ComSpec /c` on Windows (resolved from script env, not ambient `os.Getenv`).

Audit entries include a `capabilities` field reporting the honest contract above.

## Audit log

Each executed script appends one JSON line to `.mew/lifecycle-audit.jsonl`:

```json
{"ts":"…","package":"lodash","script":"postinstall","exitCode":0,"durationMs":42,"cached":false,"restored":false,"capabilities":{"packageCWD":true,"controlledPATH":true,"strippedEnv":true,"timeout":true,"processTreeKill":true,"filesystemIsolation":false,"networkIsolation":false}}
```

```sh
m builds list
m builds list --json
```

## Prepare build cache

`prepare` scripts always execute. Marker files under `<cache>/lifecycle/` are **diagnostic metadata only** — they do not skip execution until output capture/restore exists.

## Integration point

`runInstallInSession` runs lifecycle after `post_link` and before `validate`. Failures abort the mutation and roll back via the existing transaction journal.

## CI guidance

- Set `MEW_EXPERIMENTAL_LIFECYCLE=1` only when you intend to run scripts.
- Prefer `m install --ignore-scripts` or `lifecycle.ignore_scripts: true` in CI until packages are explicitly trusted.
- Commit `.mew/trusted-packages.json` when your team approves a shared allowlist.

## Fixtures

| Fixture | Purpose |
|---|---|
| `fixtures/lifecycle/postinstall-write-file` | postinstall writes `marker.txt` |
| `fixtures/lifecycle/failing-script` | postinstall exits non-zero |
| `fixtures/lifecycle/native-addon-build-stub` | prepare stub |
| `fixtures/lifecycle/prepare-counter` | prepare counter (re-runs after marker present) |
