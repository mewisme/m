# MVP 0000–0031 stabilization certification

Recorded **2026-07-30** after the stabilization pass (plan progress safety,
identity, evidence, performance gate honesty, local certification alignment,
inventory consistency, advisory range cleanup).

## Commits

| Role | SHA |
|---|---|
| Reviewed baseline | `050291cafb0ee0bed95b2edd8684e0149d1244a7` |
| Stabilization series (6 commits) | `02e8858` … `6bc9dfa` |
| Follow-up fixes | `dba1cc8`, `1ecbcff`, `4513ddb` |
| **Final implementation SHA** | `fa49800795752e4283b46399348ef921d01255b3` |

## CI evidence

| Field | Value |
|---|---|
| Workflow run ID | [`30538066438`](https://github.com/mewisme/mew/actions/runs/30538066438) |
| `head_sha` | `fa49800795752e4283b46399348ef921d01255b3` |
| Conclusion | **success** (full `ci` workflow on `main`; `platform-lock` Windows shard rerun) |

## Gates added or corrected

- `plan-generation-idempotency` — blocking; `pwsh tools/ci/verify-plan-generation.ps1`
- `markdown-link-check` — blocking; `go run ./tools/check-links`
- `bench-correctness` — blocking install bench JSON/schema gate
- `bench-regression` — advisory until Ubuntu baseline exists (`benchmarks/waivers.json`)

## Local preflight (Windows host, 2026-07-30)

```powershell
$env:CGO_ENABLED = "0"
go test ./... -count=1
go vet ./...
go build ./cmd/m ./cmd/mx
go run ./tools/check-license
go run ./tools/check-deps
go run ./tools/conformance/verify-fixtures
go run ./tools/ci/verify-crash-shards
go run ./tools/check-links
pwsh tools/ci/verify-plan-generation.ps1
```

All commands above passed on the final implementation SHA before the evidence
commit.

## Residual limitations

- `bench-regression` remains advisory on shared `ubuntu-latest` until a stable
  platform-matched baseline is captured and waivers expire.
- Race and long crash suites are documented as separate expensive targets
  (`make core-cert-crash`), not part of the default `make core-cert` loop.
