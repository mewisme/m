# Command UI Migration Matrix

Audit of every visible `m` / `mx` command and its presentation behavior.  
Generated: 2026-08-02 | Branch: feat/github-cli-ui

## Legend

| Mark | Meaning |
|------|---------|
| ✓ | Migrated to shared primitives |
| — | Already used shared primitives (no change) |
| ⊘ | Intentionally unchanged (data export / machine contract) |

## Command Matrix

| Command | Result Type | Progress | Success | No-op | Warnings | Error | Summary | Table/List | Prompt | Child | Structured | Narrow | Accessible | Status |
|---------|------------|----------|---------|-------|----------|-------|---------|------------|--------|-------|------------|--------|------------|--------|
| `install` | InstallResult | Activity line | Summary | "Already up to date" | Scripts blocked, cleanup | Mapped ErrorView | Package deltas | — | Build approval | — | JSON | KeyValues stacked | Append-only | ✓ |
| `add` | InstallResult | Activity line | Summary | N/A | Same as install | Mapped ErrorView | Package deltas | — | Build approval | — | JSON | KeyValues stacked | Append-only | ✓ |
| `remove` | InstallResult | Activity line | Summary | N/A | Same as install | Mapped ErrorView | Package deltas | — | Build approval | — | JSON | KeyValues stacked | Append-only | ✓ |
| `ci` | InstallResult | Activity line | Summary | N/A | Same as install | Mapped ErrorView | Package deltas | — | — | — | JSON | KeyValues stacked | Append-only | ✓ |
| `update` | InstallResult | Activity line | Summary | "Already up to date" | Same as install | Mapped ErrorView | Package deltas | — | Build approval | — | JSON | KeyValues stacked | Append-only | ✓ |
| `run` | child exit | Suspended | Completion summary | — | — | Exit status | Duration | — | — | Raw pass-through | — | — | Append-only | — |
| `exec` | child exit | Suspended | Completion summary | — | — | Exit status | Duration | — | — | Raw pass-through | — | — | Append-only | — |
| `mx` | child exit | Prep + suspend | Compact | — | Consent | Exit status | Compact | — | Consent | Raw pass-through | — | — | Append-only | — |
| `pack` | pack line | None | — | — | — | — | — | — | — | — | JSON | — | Append-only | — |
| `publish` | publish result | None | Plan lines | — | — | Mapped ErrorView | — | — | — | — | JSON | — | Append-only | — |
| `plan` | plan output | None | — | — | — | Mapped ErrorView | — | Table | — | — | JSON | Stacked | Append-only | — |
| `explain` | conflict tree | None | StatusLine | — | — | Mapped ErrorView | — | — | — | — | JSON | — | Append-only | ✓ |
| `patch` | patch path | None | StatusLine | — | — | — | — | — | — | — | — | — | Append-only | ✓ |
| `config get` | config value | None | Raw value | — | — | Mapped ErrorView | — | — | — | — | JSON | — | — | ⊘ |
| `config set` | — | None | StatusLine | — | — | Mapped ErrorView | — | — | — | — | JSON | — | Append-only | — |
| `config list` | config table | None | — | — | — | — | — | Table | — | — | JSON | Stacked | Append-only | — |
| `registry *` | registry data | None | KeyValue lines | — | — | Mapped ErrorView | — | — | — | — | JSON | — | — | ⊘ |
| `project` | project info | None | KeyValue lines | — | — | — | — | — | — | — | JSON | — | — | ⊘ |
| `store status` | store status | None | KeyValue lines | — | — | — | — | — | — | — | JSON | — | — | ⊘ |
| `env` | env vars | None | Shell export | — | — | — | — | — | — | — | JSON | — | — | ⊘ |
| `sbom` | SBOM data | None | Raw output | — | — | — | — | — | — | — | JSON | — | — | ⊘ |
| `version` | version | None | Version text | — | — | — | — | — | — | — | JSON | — | — | — |
| `help` | help text | None | — | — | — | Mapped ErrorView | — | Command list | — | Pager | — | Wrapped | Plain | — |
| `builds` | builds table | None | — | — | — | Mapped ErrorView | — | Table | Approve | — | JSON | Stacked | Append-only | — |
| `audit` | audit table | None | — | — | — | Mapped ErrorView | — | Table | — | — | JSON | Stacked | Append-only | — |
| `doctor` | doctor report | None | KeyValue lines | — | Warnings | — | — | — | — | — | JSON | — | — | ⊘ |
| `conformance` | conformance | None | pass/fail line | — | — | — | — | — | — | — | JSON | — | — | ⊘ |
| `bench` | benchmark | None | key=value | — | — | — | — | — | — | — | JSON | — | — | ⊘ |
| `m <script>` | child exit | None | — | — | — | Exit status | — | — | — | Raw pass-through | — | — | Append-only | — |
| `recover` | mutation | Activity line | Summary | — | — | Mapped ErrorView | — | — | — | — | JSON | KeyValues stacked | Append-only | ✓ |

## Summary

- **Migrated**: 9 command families (install/add/remove/ci/update/explain/patch/recover + all install-family)
- **Already shared**: 8 command families (run/exec/mx/pack/publish/plan/config/help)
- **Intentionally raw**: 9 command families (config get/registry/project/store/env/sbom/doctor/conformance/bench) — data export / machine contracts
- **Total commands**: 26
