# macOS CI evidence slot (UX-0008)

**Status:** pending — fill after a green `conformance-cli-ux` run on the tip
SHA pushed for UX-0008 certification. No local macOS measurement on this host.

Record the GitHub Actions job that runs:

```text
go run ./cmd/m conformance run cli-ux --json
```

on `macos-latest` (workflow job name: `conformance-cli-ux`) for the **exact**
certification commit SHA already on `origin/main`.

## Required fields when filled

| Field | Example |
|---|---|
| Commit SHA | full 40-character SHA |
| Workflow run ID | numeric Actions run id |
| Job name | `conformance-cli-ux` (macos-latest) |
| OS / arch | darwin / amd64 or arm64 |
| Go version | from job log |
| Result | green / fail with root cause |

Do not claim local macOS certification from this Windows host.
