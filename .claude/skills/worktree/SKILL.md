---
name: worktree
description: Create, use, synchronize, and remove isolated Git worktrees for concurrent Mew engineering tasks without disturbing the shared checkout. Use before substantive code changes or whenever multiple agents may edit the repository concurrently.
---

# Worktree workflow

```sh
git fetch origin
git worktree add -b <branch> <path> origin/main
```

Inside the worktree:

- keep one coherent task per branch
- stage only owned files
- rebase on the current target branch before final push when required
- run local verification before pushing
- never force-push the default branch
- do not use `git add -A` when unrelated agent files may exist

Remove after landing or abandonment:

```sh
git worktree remove <path> --force
git branch -D <branch>   # only after confirming it is safe
```

Go caches may remain shared. Project fixtures, generated outputs, local stores, and HOME must be isolated per worktree when tests mutate them.
