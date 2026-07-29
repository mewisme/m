# Pack and publish

MVP **0027**. Deterministic `m pack` tarball creation and authenticated
`m publish` to npm-compatible registries.

See also: [`registry.md`](registry.md), [`fetch.md`](fetch.md),
[`manifest.md`](manifest.md).

## `m pack`

Creates `name-version.tgz` using npm file-selection rules.

```text
m pack [package-dir] [--pack-destination <dir>]
```

| Input | Behavior |
|---|---|
| `package-dir` | Directory containing `package.json` (default: project root) |
| `--pack-destination` | Output directory (default: current working directory) |

Stdout prints the tarball basename (one line).

### File selection

1. If `package.json` `files` is non-empty, only listed paths (files or
   directories) are candidates.
2. Otherwise every file under the package root is a candidate.
3. `.npmignore` patterns (and default npm ignores) filter candidates.
4. `package.json` is always included.

Paths inside the tarball use the npm `package/` prefix. Entries are sorted;
tar and gzip mtimes are set to the Unix epoch for reproducible bytes.

### Validation

- `name` and `version` are required in `package.json`.
- Tarball filename follows npm rules (`@scope/pkg` → `scope-pkg-version.tgz`).

### Sandbox (Pass 32)

`m pack` enforces a fail-closed root boundary before tar creation:

- Reject absolute, drive, UNC, `..`, and empty path components
- Reject symlinks and reparse points (Windows junctions included)
- Preserve executable bits on regular files
- Exclude output tarball, `.git`, `node_modules`, `.mew`, and temp files
- Limits: 100k files, 512 MiB per file, 2 GiB total, 4096-byte path length

Implementation: `internal/pack/sandbox.go`. Hostile-path tests:
`internal/pack/sandbox_test.go`, `sandbox_windows_test.go`.

## `m publish`

Packs (unless a tarball is given) and `PUT`s to the configured registry.

```text
m publish [tarball.tgz]
  [--dry-run]
  [--tag <dist-tag>]
  [--access public|restricted]
  [--otp <code>]
  [--provenance]
  [--pack-destination <dir>]
  [--json]
```

| Flag | Default | Notes |
|---|---|---|
| `--dry-run` | off | Validate tarball and print plan; no network `PUT` |
| `--tag` | `latest` | npm dist-tag |
| `--access` | — | Required semantics for scoped packages when registry expects it |
| `--otp` | — | Sent as `npm-otp` header for registry 2FA |
| `--provenance` | off | Requires a configured `ProvenanceAttest` provider; fails with `ERR_M_UNSUPPORTED` before registry upload when unset |
| `--pack-destination` | cwd | Used when packing before publish |
| `--json` | off | Emit `PublishResult` JSON |

Registry URL and auth follow project config (`registry`, `registry.auth_token_env`).
Publish errors redact bearer tokens and credentials in diagnostics
(`registry.redactPublishErr`).

### Dry-run output

Human mode prints a single plan line:

```text
publish name@version → https://registry.example tag=latest (12345 bytes) (dry-run)
```

### Successful publish

```text
+ name@version
```

## Flow

```text
ResolvePackTarball (arg or m pack)
  → read package/package.json from .tgz
  → validate name/version/tarball name
  → [optional] ProvenanceAttest hook
  → registry.Publish (PUT with _attachments)
```

`--dry-run` stops before `registry.Publish`.

## Go surfaces

| Package / symbol | Role |
|---|---|
| `internal/pack` | `Pack`, `ListFiles`, `ReadPackageJSONFromTarball` |
| `internal/app` | `Pack`, `Publish`, `ResolvePackTarball`, `ProvenanceAttest` hook type |
| `internal/registry` | `Publish`, `PublishOptions` |
| `internal/cli` | `m pack`, `m publish` commands |

## Fixtures and tests

| Path | Exercises |
|---|---|
| `fixtures/pack/minimal-package` | Pack file list golden |
| `testdata/pack/minimal-package-files.json` | Expected packed paths |
| `tests/integration/pack_test.go` | `ListFiles` + CLI pack |
| `tests/integration/publish_test.go` | Dry-run (no PUT) and authenticated PUT + OTP |

## Provenance and verification

| Surface | Behavior |
|---|---|
| `m publish --provenance` | Calls `ProvenanceAttest` hook when configured; **fail closed** (`ERR_M_UNSUPPORTED`) before upload when no provider is set |
| `m verify provenance` | Verifies DSSE/Sigstore-bundle attestations with **explicit trust policy** (`TrustConfiguredKey` in production; fixture key for tests only) |
| Live Sigstore / Fulcio | **Not supported** — fixture attestations in tests are not production Sigstore verification |

## Intentional limits (v1)

- No live Sigstore publish provider or registry attestation upload in 0027/0030.
- Workspace publish filters and monorepo publish orchestration are not in 0027.
- `m publish` does not run prepublish lifecycle scripts; pack contents reflect
  the tree on disk at invocation time.
