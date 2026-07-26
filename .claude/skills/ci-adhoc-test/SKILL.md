---
name: ci-adhoc-test
description: Add and run a temporary branch-scoped GitHub Actions probe for Mew behavior that requires Windows, macOS, another architecture, or a clean hosted environment. Use when local Docker cannot verify the target behavior and a full permanent CI job is not yet justified.
---

# CI ad-hoc test

Create a narrowly scoped workflow and fixture that run only on manual dispatch or the task branch. Print exact environment and command results. Upload useful logs or filesystem snapshots as artifacts.

Use this for:

- Windows command resolution, junctions, `.cmd` shims, and signals
- macOS clonefile/reflink behavior
- architecture-specific binaries
- clean-machine installer checks
- filesystem or permission behavior unavailable locally

Remove or convert the workflow after the investigation. Do not leave a high-cost permanent matrix without a documented reason.
