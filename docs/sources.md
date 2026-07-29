# Non-registry dependency sources

MVP **0027**. Git, local path, and tarball specifiers resolve into the lock
graph, record provenance in `m.lock` extensions, and materialize during install.

See also: [`resolver.md`](resolver.md), [`lockfile.md`](lockfile.md),
[`fetch.md`](fetch.md), [`patch.md`](patch.md), [`install.md`](install.md).

## Protocol matrix

| Specifier prefix | Resolve | Install | Lock extension |
|---|---|---|---|
| `git+https:`, `git+ssh:`, `git:`, `github:` | yes | shallow git fetch at pinned commit | `mew.resolver/git` |
| `file:` | yes | copy from project-relative path | `mew.resolver/local` |
| `tarball:` | yes | extract local `.tgz` into stage | `mew.resolver/local` |
| `link:` | yes | symlink/junction to local path | `mew.resolver/local` |
| `portal:` | yes | copy from local path (portal semantics) | `mew.resolver/local` |
| `workspace:` | yes | member tree (workspaces gate) | `mew.resolver/local` |

Registry (`semver` range), `npm:` alias, and `catalog:` behave as before (0020).

### `github:` shortcut

`github:user/repo#ref` normalizes to
`https://github.com/user/repo.git#ref` before URL validation and fetch.

### Git refs

- Full 40-character commit SHAs pin without network when already in the lock.
- Branch and tag refs resolve via `git ls-remote` (or lock hints) before fetch.
- Invalid refs fail at parse/resolve with `ERR_M_RESOLVE` or `ERR_M_MANIFEST`.

## Install materialization

```text
resolve → extensions (git/local/patches) → fetchSourcePackages → link
```

| Source | Materialization |
|---|---|
| Git | `fetch.FetchGit`: `git init` + shallow `fetch --depth 1` + `checkout` into stage dir |
| `file:` / `portal:` | Absolute path under project root (no copy) |
| `tarball:` | `fetch.MaterializeLocalTarball` extracts into stage |
| `link:` | Target directory via `buildLocalExtractDirs` |
| `workspace:` | Member package directory |

Git fetch runs with hooks disabled (`core.hooksPath` → `/dev/null` or `NUL`) and
does **not** initialize submodules. Offline mode rejects git network fetch with
`ERR_M_NETWORK`.

Local paths resolve relative to the declaring importer manifest directory, then
project root. Tarball paths are project-relative POSIX paths recorded in the lock.

## Lock extensions

### `mew.resolver/git`

Maps package keys (`name@version`) to pinned remotes:

```json
{
  "sample-pkg@1.0.0": {
    "url": "file:///…/sample-pkg.git",
    "commit": "1e92b302cc5df841ccc7a74c7d88e8d2c2e13535"
  }
}
```

### `mew.resolver/local`

```json
{
  "vendor-pkg@1.0.0": { "protocol": "file", "path": "vendor/pkg" }
}
```

Protocols: `workspace`, `file`, `link`, `portal`, `tarball`. Optional
`integrity` is reserved for future tarball pinning.

### `mew.resolver/patches`

Maps package keys to committed patch files (see [`patch.md`](patch.md)):

```json
{
  "pkg-a@1.0.0": { "path": "patches/pkg-a@1.0.0.patch", "hash": "…" }
}
```

Patches apply after fetch/extract and before link (`internal/app/install_helpers.go`).

## CLI

```text
m add <name>@git+https://example.com/repo.git#<commit>
m add github:user/repo#<ref>
m install    # materializes git/file/tarball sources from lock extensions
m patch <pkg> [--commit] [--edit-dir <dir>]
```

`m patch` extracts a package into a temp edit dir; `--commit` writes
`patches/<name>@<version>.patch` and updates `patchedDependencies` in
`package.json`.

## Go surfaces

| Package | Entry points |
|---|---|
| `internal/manifest` | `ParseSpecifier`, protocol constants |
| `internal/resolver` | `processGit`, `processLocal`, `DecodeGitSources`, `DecodeLocalSources` |
| `internal/fetch` | `FetchGit`, `MaterializeLocalTarball` |
| `internal/app` | `fetchSourcePackages`, patch apply in install helpers |

## Fixtures

| Path | Exercises |
|---|---|
| `fixtures/sources/file-dep` | `file:` install + frozen `m ci` |
| `fixtures/sources/git-dep` | Git dep at pinned commit + lock metadata |
| `fixtures/sources/patch-left-pad` | Patch workflow fixture tree |
| `tests/integration/sources_test.go` | File and git acceptance |
| `tests/integration/patch_test.go` | Deterministic patched install |

## Intentional limits (v1)

- Git submodules are not fetched.
- Git lifecycle scripts and hooks are not executed during fetch.
- `link:` install uses directory paths from the lock graph; full npm `link:`
  parity for global link targets is not guaranteed.
- Remote git over SSH requires `git` on `PATH` and host key/trust configuration
  outside Mew.
