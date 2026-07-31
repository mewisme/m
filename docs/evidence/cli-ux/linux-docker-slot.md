# Linux Docker evidence slot (UX-0008)

**Status:** satisfied via Ubuntu GitHub Actions (local Docker daemon unavailable).

## Local Docker

Re-checked 2026-07-31: Docker client present, Linux engine pipe missing
(`npipe:////./pipe/dockerDesktopLinuxEngine`). No local Linux Docker run.

## CI substitute (accepted)

| Field | Value |
|---|---|
| Commit SHA | `6641a4f8417c3c2e2160548f997153d64da0a477` |
| Workflow run ID | `30591819891` |
| Job name | `conformance-cli-ux` (ubuntu-latest) |
| Result | **success** (`m conformance run cli-ux --json`) |

Full workflow on that SHA: success (48/48 jobs).
