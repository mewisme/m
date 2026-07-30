# Linux Docker evidence slot (UX-0008)

**Status:** unavailable on the Windows measurement host (re-checked 2026-07-31).

Docker client is present, but the Linux engine pipe is missing
(`npipe:////./pipe/dockerDesktopLinuxEngine`). Do not invent Linux results.

## Acceptance (either)

1. Local Docker Linux preflight with a pinned image matching CI Go/Node, or
2. A green Ubuntu GitHub Actions job `conformance-cli-ux` on the exact
   certification SHA on `origin/main`.

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

Record: image name + digest, commit SHA, command output path, binary sizes,
startup median/p95 for the same four commands as the Windows artifact.
