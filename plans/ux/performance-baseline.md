# CLI UX performance baseline (UX-0008)

**Review date:** 2026-07-31  
**Owner:** CLI UX certification (UX-0008)  
**Measurement commit:** `8d1ba8f91ccd6b0773abda9340e402edd099af59`  
**Host:** Windows 10 build 26200, go1.26.5 windows/amd64, `CGO_ENABLED=0`

Raw Windows artifact: [`docs/evidence/cli-ux/windows-2026-07-31.md`](../../docs/evidence/cli-ux/windows-2026-07-31.md).

## Method

- Release-ish local builds: `go build` of `./cmd/m` and `./cmd/mx` without CGO.
- 21 wall-time samples per command after a warm binary exists.
- Median and p95 from sorted samples (index `floor(0.95*(N-1))` for p95).
- No registry access; no install mutation.

## Startup

| Command | N | Median (ms) | p95 (ms) |
|---|---:|---:|---:|
| `m version` | 21 | 36.0 | 38.4 |
| `m --help` | 21 | 35.8 | 36.9 |
| `m features --format json` | 21 | 40.6 | 42.8 |
| `mx version` | 21 | 36.9 | 39.2 |

## Binary size

| Binary | Bytes |
|---|---:|
| `cmd/m` | 31,494,656 |
| `cmd/mx` | 27,945,984 |

Charm pin history and incremental deltas:
[`charm-dependency-review.md`](charm-dependency-review.md).

## Live render notes (advisory)

Bubble Tea install progress starts only on rich-eligible stderr TTYs. Plain,
CI, pipe, accessible, and legacy paths never start a live program. Idle CPU for
an inactive live install path is expected to be near zero after lazy start
(see UX-0004 live_install tests). No hard live-FPS or CPU marketing bound is
set in this baseline.

## Advisory bounds (reviewed)

These are **review thresholds**, not hard CI fail gates and not product
marketing claims. Re-measure after material Charm or presentation changes.

| Metric | Advisory bound | Notes |
|---|---|---|
| `m version` median | ≤ 150 ms | Windows local warm launch |
| `m --help` median | ≤ 200 ms | Includes grouped root help |
| `m features --format json` median | ≤ 250 ms | Full inventory encode |
| `mx version` median | ≤ 150 ms | |
| `cmd/m` size | ≤ 40 MiB | Current ~30 MiB with Glamour |
| `cmd/mx` size | ≤ 36 MiB | Current ~27 MiB |

Linux Docker and macOS CI slots should add platform rows without weakening
these Windows-reviewed bounds.

## Deferred

- Linux Docker / macOS startup tables (evidence slots only).
- Optional Go microbench for static render throughput (only if cheap and useful).
- Hard CI regression fail for startup (keep advisory like install bench compare).
