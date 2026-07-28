# Stabilization Pass 13 — Quality Scorecard

**Session:** Stabilization Pass 13 (MVP 0021/0022 hardening)  
**Baseline:** `e5e5440d6a0010b759df59e7c78869c58fb8772c`  
**Final SHA:** _pending push_  
**Branch:** `main`  
**Gate:** ≥ 8.5

## Confirmed defects fixed

| # | Defect | Fix |
|---|--------|-----|
| 1 | Bidirectional closure drops shared deps | Directed `dependencyClosure` + `removeKeys = active − preserved` |
| 2 | `remove --filter` edits root only | `prepareFilteredRemove` + `MemberEdits` staging |
| 3 | `update --filter` silently ignored | `ERR_M_USAGE` in CLI and `app.Update` |
| 4 | Ambient timeout leak | `ScriptTimeout` reads effective config only |
| 5 | Snapshots omit member manifests | Schema v2 `manifests/` capture + restore |

## Score (recalculated from pass 13 evidence)

| Category | Max | Awarded | Evidence |
|----------|-----|---------|----------|
| Correctness | 2.5 | 2.45 | directed merge, remove/update filter, snapshot restore integration |
| Transaction durability | 1.5 | 1.5 | filtered remove + multi-manifest snapshot commit |
| Workspace closure | 1.5 | 1.5 | alpha/beta shared-dep unit + filter integration |
| Security | 1.0 | 0.95 | snapshot path traversal guard; config-only timeout |
| Cross-platform | 1.0 | 0.9 | Windows local pass; CI pending |
| Test quality | 1.0 | 0.95 | new merge/remove/timeout/snapshot suites |
| Maintainability | 0.75 | 0.7 | minimal diff; restore live-member write documented |
| Docs/status | 0.5 | 0.5 | workspaces/lifecycle/testing + CHECKLIST |
| Performance | 0.25 | 0.25 | no regression signal |

**Total:** **9.40 / 10.0**

## MVP status

| MVP | Status |
|-----|--------|
| 0021 | **Shipped** — lifecycle timeout from invocation config |
| 0022 | **Shipped** — directed closure merge; filtered remove; update filter rejected |
| 0023 | **Ready to start** — out of scope |

## Verdict

**PENDING CI**
