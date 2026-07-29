# Performance tuning

MVP **0029**. Measure install phases, tune concurrency, and guard regressions
with the bench harness.

## Install phase timing

Pass `--debug` (or set `MEW_DEBUG=1` / `M_LOG=debug`) to emit per-phase elapsed
time at the end of each install transaction phase:

```text
debug: install phase phase=resolve elapsed=120ms
debug: install phase phase=fetch elapsed=1.2s
debug: install phase phase=link elapsed=450ms
debug: install phase phase=validate elapsed=80ms
debug: install phase phase=commit elapsed=200ms
```

Phases: `resolve`, `fetch`, `link`, `validate`, `commit`. Progress events
(`phase=…` on stderr) continue regardless of debug mode.

NDJSON / JSON reporters emit structured `type=debug` records with `phase` and
`elapsed` attributes when debug is enabled.

## Worker concurrency

Fetch and registry metadata use the same default worker count:

```text
workers = min(max(runtime.NumCPU(), 1), 16)
```

Implemented in `fetch.DefaultWorkers()` and `registry.defaultMaxWorkers()`.

| Surface | Default |
|---|---|
| Tarball download pool | `fetch.Downloader.Workers` |
| Registry `Packuments` batch | `registry.Options.MaxWorkers` |

`ponytail:` `MEW_FETCH_WORKERS` env override is deferred to the cross-cutting
performance program (MVP **0081**); today only the NumCPU default applies.

## Hot-path profiling (Phase 5)

After baseline timing landed, resolver `selectVersion` buffer reuse and linker
placement `strings.Split` avoidance were evaluated against `m bench install`
and package-level `go test -bench` suites. Neither showed a clear win on the
checked-in medium-graph fixture, so no hot-path diff was applied.

`ponytail:` revisit when a 1k+ workspace fixture corpus exists (MVP **0081**)
or bench medians regress without an obvious network cause.

Integrity verification, tarball hash checks, and store validation are never
skipped for speed.

## `m bench install`

End-to-end install benchmark against a fixture project (default
`fixtures/bench/medium-graph`):

```text
m bench install              # cold (default): clear bench home cache first
m bench install --cold
m bench install --warm       # reuse cache from prior bench run
m bench install --fixture <path>
m bench install --json       # BenchResult JSON on stdout
```

Bench runs use an isolated home under `.cache/mew/bench/<fixture>/` and a local
fixture registry. Lifecycle scripts are ignored (`IgnoreScripts`).

JSON shape:

```json
{
  "case": "medium-graph-cold",
  "mode": "cold",
  "phases": { "resolve": 120, "fetch": 800, "link": 200 },
  "totalMs": 1500
}
```

Phase keys mirror install progress; values are milliseconds.

## Baselines and CI gate

Published medians live in [`benchmarks/install-baseline.json`](../benchmarks/install-baseline.json).

Check regression locally:

```powershell
pwsh tools/bench/check_regression.ps1 -Mode warm
pwsh tools/bench/check_regression.ps1 -Mode cold
```

Fails when `totalMs` exceeds baseline median by more than 10%. Set
`BENCH_WAIVER=1` to warn instead of fail (CI uses a non-blocking warm check
today).

Soak repeated installs:

```powershell
pwsh tools/soak/install-loop.ps1 -Count 10 -Mode cold
```

## Package-level benchmarks

Lower-level `go test -bench` suites (transaction, resolver, store, linker) are
documented in [`tests/benchmarks/README.md`](../tests/benchmarks/README.md).

## Related

- [`offline.md`](offline.md) — warm cache and capsule bootstrap
- [`cli.md`](cli.md) — `m bench`, `m capsule`
- [`install.md`](install.md) — install pipeline
