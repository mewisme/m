---
name: soak
description: Run extended Mew compatibility and reliability testing across real projects, lockfile identities, Node versions, repeated installs, interrupted transactions, and long-lived watchers. Use after focused tests pass and before claiming broad readiness.
---

# Soak testing

Build a declared matrix and keep raw results. Include representative native Mew, Nub, npm, pnpm, Bun, Yarn Classic, and Yarn Berry projects as support lands.

Exercise:

- repeated install/no-op/install cycles
- offline restoration
- interrupted stage and commit recovery
- rollback after manifest and lock changes
- workspace filters and recursive scripts
- lifecycle approvals
- `mx` cache reuse
- watcher restarts and signal handling
- platform and Node-version tiers

A soak failure becomes a minimal fixture before it becomes a fix. Do not debug only inside the large project.
