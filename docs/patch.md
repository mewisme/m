# Patch application

Mew applies pnpm-style unified patches during install using `internal/archive`.
Parsing and hunk application use [`github.com/bluekeyes/go-gitdiff`](https://github.com/bluekeyes/go-gitdiff).

## Supported operations (v1)

- Modify an **existing regular file** inside the package tree.
- Preserve the **existing file mode** on modification.

## Rejected operations

Preflight fails before any write when a patch would:

- create or delete files (`/dev/null` paths);
- rename or copy paths (`OldName` ≠ `NewName`);
- apply binary patches;
- change mode only (no text hunks);
- target symlinks, directories, or other non-regular files;
- escape the package root (traversal, absolute, drive-qualified, or UNC paths);
- duplicate or conflict on the same target path.

Rejected paths return `ERR_M_INTEGRITY` with a `PatchPathError` detail when path validation fails.

## API

| Function | Role |
|----------|------|
| `PreflightPlan` | Parse patch, validate paths and operations, build an apply plan |
| `ApplyPlan` | Execute a validated plan |
| `ApplyUnifiedPatch` | Preflight + apply into an existing directory |
| `ApplyPatchAtomic` | Copy source → work dir, apply, rename work → publish |

Atomic apply invokes transaction test hooks at each phase:

- `post_patch_copy`
- `post_patch_preflight`
- `post_patch_apply`
- `post_patch_publish`

On failure the work directory is removed; source and publish roots are not mutated.

## Resource limits

| Limit | Default |
|-------|---------|
| Patch file size | 1 MiB |
| Files per patch | 256 |
| Hunks per patch | 4096 |
| Path length | 4096 bytes |
| Line length | 65536 bytes |
| Output growth | 10× per-file input size |

Parse, preflight, copy, and apply honor `context.Context` cancellation.

## Install wiring

Install transactions call patch apply after fetch (see `internal/app/install_helpers.go`).
Copy-on-write staging and strict validation before apply are handled in later stabilization phases.
