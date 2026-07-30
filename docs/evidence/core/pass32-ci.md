# Pass 32 CI evidence

Reviewed and recorded **2026-07-30**.

## Commit and workflow

| Field | Value |
|---|---|
| Final `origin/main` SHA | `f0ce96df82b262819334a121c584b93b1aeaa309` |
| Workflow run ID | [`30487309379`](https://github.com/mewisme/mew/actions/runs/30487309379) |
| `head_sha` | `f0ce96df82b262819334a121c584b93b1aeaa309` |
| Code SHA (Pass 32 fixes) | `f19f3f73bd7dc4169a8a95c598a645b2077b9539` (run [`30486713425`](https://github.com/mewisme/mew/actions/runs/30486713425)) |

## Matrix summary

38 jobs succeeded across ubuntu, macOS, and Windows covering: test, race,
cross-build, platform-lock, crash integration (including Windows shards),
pnpm 9/10/11 conformance, npm/bun/yarn/nub lock bridges, core-stabilization,
fixture verification, gate probes, and lint/vuln/allowlist gates.

## Environment

- Go **1.26.x**
- Node **22**
- pnpm pins from `tools/conformance/pnpm-versions.env` (9.15.9 / 10.34.5 / 11.17.0)

## Artifacts

- Core certification report artifact name: `core-certification-report` (`core-stabilization` job)

## Known waivers (advisory, not blocking Pass 32)

- `bench-regression` and core-stabilization bench steps: `continue-on-error: true`
  (Windows-only baseline; see [`docs/performance.md`](../../performance.md))
- Windows `platform-lock`: one failed-job rerun on the same SHA (transient
  dual-winner flake)

## Reproduction commands

```powershell
CGO_ENABLED=0 go test ./... -count=1
CGO_ENABLED=0 go vet ./...
CGO_ENABLED=0 go build ./cmd/m ./cmd/mx
go run ./tools/conformance/verify-fixtures
go run ./tools/ci/verify-crash-shards
go run ./cmd/m conformance run core --json
```

See [`docs/core-certification.md`](../../core-certification.md) for the full gate matrix.
