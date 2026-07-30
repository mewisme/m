# macOS CI evidence slot (UX-0008)

**Status:** planned — no local macOS measurement.

Record the GitHub Actions `test` / `cli-ux` (or equivalent) job that runs:

```text
go run ./cmd/m conformance run cli-ux --json
```

on `macos-*` for the **exact** certification commit SHA already on `origin/main`.

## Required fields when filled

| Field | Example |
|---|---|
| Commit SHA | full 40-character SHA |
| Workflow run ID | numeric Actions run id |
| Job name | e.g. `test (macos-…)` or dedicated `cli-ux` |
| OS / arch | darwin / amd64 or arm64 |
| Go version | from job log |
| Result | green / fail with root cause |

Do not claim local macOS certification from this Windows host.
