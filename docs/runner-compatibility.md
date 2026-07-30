# Runner compatibility and certification

MVP **0046** certifies the runner stack delivered by MVPs **0040–0045** through
a deterministic conformance harness. This document records compatibility status
per surface without vague parity language.

## Status vocabulary

| Status | Meaning |
|---|---|
| `certified` | Required conformance suites pass on all applicable platforms |
| `certified-extension` | Mew-only behavior, certified with documented divergence |
| `experimental` | Shipped behind gates; waiver or follow-up MVP required |
| `partial` | Certified with documented platform or capability limits |
| `probe-only` | Optional probe suite; does not block certification |
| `platform-waived` | Required behavior waived on a platform with dated waiver |
| `deferred` | Not shipped; no certification claim |

## Certified surfaces (0046)

| Surface | Status | Evidence |
|---|---|---|
| `m run` script runner | certified | `runner-launch-argv-*`, `runner-stdio`, `runner-corpus-layouts` |
| Workspace script orchestration | certified | `runner-workspace-*` |
| `m exec` local binaries | certified | `runner-envexec-equivalence`, `runner-envexec-stress` |
| `mx` consent and fetch boundaries | certified | `runner-mx-security` (`networkPolicy: local-fixture`) |
| Snapshot/capsule offline boundaries | certified | `runner-snapshot-security`, `runner-capsule-security` |
| `environment-prepared` event v1 | certified | `runner-event-schema` |
| Environment inspect JSON v1 | certified | `runner-inspect-schema` |
| Import architecture boundaries | certified | `runner-import-boundaries` |
| Process supervision short soak | certified | `runner-process-soak`, `runner-process-tree-*` |

## Certified extensions (Mew-only)

| Extension | Status | Notes |
|---|---|---|
| Direct `m <script>` shortcuts | experimental | `runner-direct-dispatch-gates`; waiver `waiver-direct-dispatch-gates` until MVP 0050 |
| Dispatch collision handling | experimental | `runner-dispatch-collisions`; waiver `waiver-direct-dispatch` documents gate status |

Direct shortcuts are intentional Mew extensions. Certification documents
collision precedence and gate behavior; it does not claim Nub ships this
surface.

## Partial / platform-specific

| Capability | Linux | macOS | Windows |
|---|---|---|---|
| Child exit propagation | certified | certified | certified |
| SIGINT exit 130 | certified | certified | not applicable |
| Unix signal mapping | certified | certified | mapped Windows behavior |
| Process-group cleanup | certified | certified | Job Object policy |
| Grandchild cleanup when owned | certified | certified | certified when Job Object owned |
| Forced termination fallback | certified | certified | certified |
| Pipe closure on cancel | certified | certified | certified |
| Windows Mode B launch | not applicable | not applicable | certified (`runner-windows-modeb`) |

When a capability is `not applicable`, certification uses platform-split suites
instead of runtime `t.Skip()` in required suites.

## Deferred

| Feature | Status | Owner MVP |
|---|---|---|
| Interactive script/bin picker (`runner.interactive-select`) | deferred | 0090 |
| Shell execution mode | deferred | — |
| Public registry in certification | forbidden | — |

## Harness contract

```bash
m conformance run runner [--json] [--output report.json] [--group <group>] [--filter <suite-id>]
m conformance verify runner --report <path>... --output runner-certification-summary.json
```

Manifest: [`tests/conformance/runner-matrix/manifest.json`](../tests/conformance/runner-matrix/manifest.json)

Waivers: [`tests/conformance/runner-matrix/waivers.v1.json`](../tests/conformance/runner-matrix/waivers.v1.json)

Rules:

- `expectedTests` must match the suite regex exactly (preflight `go test -list`)
- Suites run sequentially in locked group order
- Each suite uses `fresh` isolation (no shared cache between suites)
- No public registry access in certification (`networkPolicy` enforced)
- No suite-level parallelism

## Benchmarks

```bash
m benchmark runner --profile smoke --json
m benchmark runner --profile full --json --compare benchmarks/runner-baseline.json
```

Benchmarks use local fixtures only. Baseline comparison is informational;
`--compare` never mutates the baseline file.

## Runtime handoff

Stable surfaces for MVP **0050+** are documented in
[`architecture/runner-runtime-handoff.md`](architecture/runner-runtime-handoff.md).
