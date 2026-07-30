# Linux Docker evidence slot (UX-0008)

**Status:** satisfied via Ubuntu GitHub Actions (local Docker daemon unavailable).

## Local Docker

Re-checked 2026-07-31: Docker client present, Linux engine pipe missing
(`npipe:////./pipe/dockerDesktopLinuxEngine`). No local Linux Docker run.

## CI substitute (accepted)

| Field | Value |
|---|---|
| Commit SHA | `b1c3bbfefcd07b1a94a67dca4d69ab1c620412ad` |
| Workflow run ID | `30590802511` |
| Job name | `conformance-cli-ux` (ubuntu-latest) |
| Result | **success** (`m conformance run cli-ux --json`) |

## When Docker is available

Use a pinned Linux image matching CI Go/Node, isolated `HOME`, `CGO_ENABLED=0`:

```powershell
docker run --rm `
  --mount "type=bind,source=<ABSOLUTE_REPOSITORY_PATH>,target=/workspace" `
  --mount type=volume,source=mew-go-mod-cache,target=/go/pkg/mod `
  --mount type=volume,source=mew-go-build-cache,target=/root/.cache/go-build `
  --workdir /workspace `
  --env CGO_ENABLED=0 `
  --env HOME=/tmp/mew-home `
  <PINNED_TEST_IMAGE> `
  bash -lc 'go build -o /tmp/m ./cmd/m && go build -o /tmp/mx ./cmd/mx && go run ./cmd/m conformance run cli-ux --json'
```
