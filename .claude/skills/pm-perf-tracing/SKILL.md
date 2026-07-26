---
name: pm-perf-tracing
description: Localize Mew package-manager performance bottlenecks with phase diagnostics, strategy tallies, Go benchmarks, runtime traces, and pprof. Use when install, resolve, fetch, extract, link, lifecycle, commit, or rollback is unexpectedly slow.
---

# Package-manager performance tracing

Measure before changing code.

## First layer: phase data

Run the operation with structured diagnostics:

```sh
MEW_DIAG_FILE=/tmp/mew-diag.jsonl \
MEW_DIAG_SUMMARY=1 \
MEW_DIAG_PHASES=1 \
./m install --offline
```

Expected phases include inspect, resolve, plan, fetch, verify, extract, store, link, lifecycle, validate, and commit.

## Measurement discipline

- use a verified clean `node_modules`
- warm the store, then use `--offline` for filesystem measurements
- check exit code for every sample
- compare identical fixtures and configurations
- report medians and distributions, not one run
- use strategy counts for linker questions
- separate cold network, warm cache, and no-op install cases

## Deep profiling

Use `go test -bench`, `go test -trace`, `go tool trace`, CPU profiles, heap profiles, mutex profiles, and block profiles. Keep diagnostics disabled by default and test their overhead.
