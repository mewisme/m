---
name: nub-reference-sync
description: Synchronize Mew's behavioral reference inventory with newer Nub and Aube releases without copying Rust source. Use when upstream changes package-manager semantics, lockfile support, runtime behavior, CLI compatibility, security policy, or tests that Mew intends to match.
---

# Nub reference synchronization

Nub and Aube are evidence sources, not vendored dependencies.

## Workflow

1. Record the previous and target upstream commits or releases.
2. Inventory behavior-affecting changes from changelogs, code, tests, issues, and executable probes.
3. Map each change to Mew's feature matrix and compatibility targets.
4. Build differential fixtures for changes relevant to Mew.
5. Classify each item as adopt, intentionally diverge, defer, or not applicable.
6. Update Mew conformance fixtures, reference notes, and roadmap.
7. Implement adopted behavior in idiomatic Go through normal implementation threads.

Never translate a Rust diff line by line. Preserve Mew's atomic transaction, explainability, lock neutrality, and direct script-shortcut decisions even when the reference differs.
