# Stabilization Pass 3 — Quality Scorecard

**Baseline commit:** `b1fe1977b13ccaccab9a785d251b3fb913a7972e`  
**Started:** 2026-07-27  
**Gate:** ≥ 9.0 to unblock MVP 0021

## Score (updated each phase)

| Category | Max | Awarded | Evidence | Deductions |
|----------|-----|---------|----------|------------|
| Correctness | 2.5 | 2.3 | canonicalize.go; key-based incremental merge; policy drift tests pass | edge-case coverage gaps |
| Transaction durability | 1.5 | 1.4 | journal generations + head; directory lock v3; takeover; RestoreSnapshot in txn | macOS CI not yet green locally |
| Store integrity | 1.5 | 1.4 | directory import locks; strict manifest v2; ReconcileIndex + warn-on-upsert | — |
| Security | 1.0 | 0.9 | fsx guards; strict manifest path validation | — |
| Cross-platform | 1.0 | 0.75 | linux/darwin/windows identity split; platform-lock CI job added | macOS CI evidence pending |
| Test quality | 1.0 | 0.95 | journal_crash, lock_proc, import_proc, canonicalize, policy_drift tests | — |
| Maintainability | 0.75 | 0.7 | shared fsx/lockdir; genfile.go | — |
| Docs/status | 0.5 | 0.4 | store.md updated; CHECKLIST blocks 0021 | transaction/resolver docs pending |
| Performance | 0.25 | 0.2 | bounded workers | — |

**Estimated total:** ~8.8 / 10.0 (post phases 1–12)

## Phase status

- [x] Phase 0 — baseline recorded, 0021 blocked
- [x] Phase 1 — platform process identity
- [x] Phase 2 — directory project lock
- [x] Phase 3 — recovery takeover
- [x] Phase 4 — journal generations
- [x] Phase 5 — publication hardening
- [x] Phase 6 — store directory locks
- [x] Phase 7 — index contract
- [x] Phase 8 — strict tree manifest
- [x] Phase 9 — canonical peer dedup
- [x] Phase 10 — full-identity incremental
- [x] Phase 11 — policy drift
- [x] Phase 12 — platform CI jobs
- [ ] Phase 13 — docs, full gates, score ≥9.0, READY/BLOCKED report
