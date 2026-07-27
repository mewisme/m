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

## Sandbox v1 (path-restricted)

Scripts run with:

- **CWD** — package root under staged `node_modules`
- **PATH** — `node_modules/.bin` prepended
- **Env** — invocation snapshot with registry tokens and common secrets stripped (`NPM_TOKEN`, `NODE_AUTH_TOKEN`, `AWS_*`, …)

Shell dispatch: `sh -c` on Unix, `ComSpec /c` on Windows.

OS-level sandboxing (namespaces, job objects) is deferred; see MVP plan deferrals.

## Audit log

Each executed script appends one JSON line to `.mew/lifecycle-audit.jsonl`:

```json
{"ts":"…","package":"lodash","script":"postinstall","exitCode":0,"durationMs":42}
```

```sh
m builds list
m builds list --json
```

## Prepare build cache

`prepare` script results are keyed by package integrity, script name, platform, and policy version under `<cache>/lifecycle/`. Cache hits skip re-execution (ponytail: marker files only; full output capture deferred).

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
| `fixtures/lifecycle/prepare-counter` | prepare cache counter |
| `fixtures/lifecycle/registry` | hermetic registry for integration tests |

Tests set `MEW_EXPERIMENTAL_LIFECYCLE=1` via `testkit.EnableLifecycle`.
