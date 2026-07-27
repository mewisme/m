# Testing strategy

Hermetic fixtures, clean-home isolation, local registry, fuzz smoke, and
conformance stubs (MVP 0008). Normal CI must never depend on the public npm
registry. Large ecosystem corpora belong in scheduled jobs (MVP 0080).

Source: [`plans/0008-testing-strategy.md`](../plans/0008-testing-strategy.md).

## Layout

```text
fixtures/
  registry/v1/
    manifest.json           # SHA-256 index (fail closed on mismatch)
    packuments/             # npm-shaped packument JSON
    tarballs/               # synthetic *.tgz (not downloaded from npm)
  projects/
    basic-cjs/ basic-esm/ typescript-app/ workspace-simple/
  identity/                 # lockfile identity cases (0006)
  security/evil-archives/   # known-bad member names; never extract in prod
tests/
  integration/              # clean-home + local registry smoke
  conformance/              # differential harness + inventory stub
internal/testkit/           # TempHome, LoadRegistry, DiffReport, faults, FS probe
```

```mermaid
flowchart LR
  fix[Fixtures] --> reg[LocalRegistry]
  reg --> mew[MewUnderTest]
  mew --> cmp[Compare]
  cmp --> ref[ReferencePM]
```

## Clean-home contract

`testkit.CleanEnv` / `TempHome` set:

- `HOME`, `USERPROFILE`
- `XDG_CACHE_HOME`, `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`
- `MEW_HOME`, `MEW_CACHE_DIR`, `MEW_STORE_DIR`, `MEW_CONFIG_DIR`

All under a `t.TempDir()`. Tests must copy fixtures into the temp tree
(`CopyFixture`) before mutating. Do not write into the developer’s real home.

## Local fixture registry

1. Edit blobs under `fixtures/registry/v1/`.
2. Update `manifest.json` SHA-256 for every blob path.
3. `LoadRegistry` verifies checksums on load; mismatch fails the test.
4. `Start` serves `GET /{name}` (packument) and `GET /{name}/-/{file}.tgz`.

Synthetic packages only. Generator bytes are checked in; do not fetch from
`registry.npmjs.org` in tests or CI.

Smoke today: HTTP fetch packument + tarball + integrity check. Full `m install`
against the fixture registry waits for later install MVPs.

## How to add a fixture

1. Create a directory under `fixtures/<area>/`.
2. Prefer hand-authored text. For binaries, record SHA-256 in a nearby
   `manifest.json` (registry) or digests file.
3. Document any generator command in a fixture `README.md`.
4. Cover with a test that uses `CopyFixture` or `LoadRegistry`.
5. Never mutate checked-in fixtures from tests.

## Differential comparison

Schema: `testkit.DiffReport` (`schemaVersion`, `skipped`, `skipReason`, `mew`,
`reference`, `diffs[]`).

- Normalize with `NormalizeOutput` (absolute paths, CRLF, ISO timestamps).
- If `npm` / `nub` is absent, conformance smoke writes a skipped report and still
  validates the schema.
- Documented reference pins: **npm 10**, **pnpm 9**; Nub pin field in
  [`tests/conformance/inventory.json`](../tests/conformance/inventory.json).

```powershell
make conformance
# or
go test ./tests/conformance/... -count=1
```

## Fuzz smoke

| Target | Package | Notes |
|---|---|---|
| `FuzzParseJSON` | `internal/manifest` | malformed package.json |
| `FuzzDecodeGraph` | `internal/graph` | truncated/garbage graphs |
| `FuzzLoadConfig` | `internal/config` | hostile JSONC |

Deferred until packages exist: archive path fuzz (`internal/archive`), semver
ranges (`internal/semver`).

```powershell
make fuzz-smoke
```

## Stabilization pass 2 suites (0017–0020)

| Area | Package / path | What it proves |
|---|---|---|
| Project lock contention | `internal/transaction/lock_proc_test.go` | 20-process exclusive lock, stale recovery, ctx cancel |
| Crash recovery | `internal/transaction/inject_test.go`, `tests/integration/txn_inject_test.go`, `tests/integration/txn_crash_test.go` | Kill-boundary recovery, idempotent `m recover` |
| Store import locks | `internal/store/import_proc_test.go` | External `.locks/<algo>/<hex>.lock`, GC safety |
| Tree manifest security | `internal/store/treemanifest_security_test.go` | Bidirectional verify, hostile manifests, legacy re-import |
| Graph aliases | `internal/graph/alias_test.go`, `fixtures/resolver/aliases/` | `Edge.Name` round-trip |
| Peer nearest + instances | `internal/resolver/peers_nearest_test.go`, `peers_instances_test.go` | Nearest provider, dual peer-context nodes |
| Incremental update | `internal/resolver/incremental_diff_test.go` | Edge-keyed closure, fingerprint drift |
| Path guards | `internal/fsx`, `internal/transaction/paths_test.go` | Ancestor symlink/junction rejection |
| Registry cancel | `internal/registry/bounded_test.go` | `Packument` / `Packuments` ctx cancellation |
| Isolated linker | `internal/linker/isolated/fixture_test.go`, `tests/integration/isolated_test.go` | StoreID, phantom `require()` |

Cross-process tests use file-based coordination and bounded timeouts; they may be
slower on shared CI runners.

Race checks (when CGO enabled):

```powershell
$env:CGO_ENABLED = "1"
go test -race ./internal/transaction/... ./internal/store/... ./internal/resolver/... -count=1
```

On Windows without CGO, race builds are skipped; concurrency is covered by
cross-process lock/import tests instead.

## Failure injection

- `FaultyRoundTripper` — network cut after N requests
- `LimitedWriter` — `ENOSPC` after N bytes

## Filesystem probes

`ProbeFS` reports symlink support, junction (Windows stub), and case sensitivity.

## Required metadata for differential runs

Record in reports or job logs: OS, Go version (`go env GOVERSION`), and
reference tool versions (`npm --version`, `pnpm --version`, `nub --version`
when present).

## CI policy

| Suite | Normal PR CI | Scheduled / 0080 |
|---|---|---|
| `go test ./...` | yes (hermetic) | yes |
| `golangci-lint run` | yes (Ubuntu) | yes |
| Fixture registry | local only | local only |
| Public npm | never | never |
| Large ecosystem corpus | no | yes |
| Reference PM differential | skip if absent | pin tools in image |
