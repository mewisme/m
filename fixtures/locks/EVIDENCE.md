# Lock fixture evidence (Pass 15)

**Fetch date:** 2026-07-28  
**Host:** Windows 10 (win32), amd64  
**Node:** stock Node from CI/dev (see per-fixture `metadata.json` when present)

## pnpm

| Major | Pinned version | Source |
|-------|----------------|--------|
| 9 | 9.15.9 | [pnpm releases](https://github.com/pnpm/pnpm/releases), [npm `pnpm@9`](https://www.npmjs.com/package/pnpm/v/9.15.9) |
| 10 | 10.34.5 | [pnpm releases](https://github.com/pnpm/pnpm/releases), [npm `pnpm@10`](https://www.npmjs.com/package/pnpm/v/10.34.5) |
| 11 | 11.17.0 | [pnpm releases](https://github.com/pnpm/pnpm/releases), [npm `pnpm@11`](https://www.npmjs.com/package/pnpm/v/11.17.0) |

Pins live in [`tools/conformance/pnpm-versions.env`](../../tools/conformance/pnpm-versions.env).

### Generation commands (fixture script)

```powershell
.\tools\conformance\generate-lock-fixtures.ps1
```

Per family (after `pnpm` is on PATH at the pinned version):

```text
pnpm@9.15.9  install   → fixtures/locks/pnpm/generated/v9-basic/
pnpm@10.34.5 install   → fixtures/locks/pnpm/generated/v10-basic/
pnpm@11.17.0 install   → fixtures/locks/pnpm/generated/v11-basic/
```

**Do not** infer producer major from `lockfileVersion: '9.0'` alone. Use structural evidence
(package `checksum` for v10, `buildPolicy` for v11) or explicit `--pnpm-major`.

### Docs

- [pnpm lockfile](https://pnpm.io/git#lockfile)
- [pnpm 11 releases blog](https://pnpm.io/blog/releases/11.11-11.14)

## Nub

| Artifact | Source |
|----------|--------|
| `nub.lock` layout | [nubjs.com](https://nubjs.com) — pnpm v9-shaped YAML, distinct filename |

Existing hand-maintained fixture: `fixtures/locks/nub/v1-basic/`. Binary-generated Nub
families are deferred to Pass 15 Phase 9/10.

## Mew detection extension

Adapter-recorded metadata may appear under top-level extension key `mew.lockfile/detection`
(JSON: `format`, `producerMajor`). Used as evidence rank 4 in detection order.
