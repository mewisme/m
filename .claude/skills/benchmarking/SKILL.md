---
name: benchmarking
description: Design and run reproducible Mew benchmarks for package management, script execution, runtime startup, and compatibility paths. Use before making or reviewing performance claims, comparing strategies, or preventing regressions.
---

# Benchmarking

Define the question before the benchmark. Control package graph, lockfile, cache state, network state, filesystem, Node version, CPU architecture, and process environment.

Benchmark separate scenarios:

- cold metadata and tarball fetch
- warm store install
- no-op install
- isolated and hoisted linking
- transaction commit and rollback
- lockfile read/write
- `m run` and direct shortcut startup
- `mx` local hit and cached DLX hit
- runtime cold and warm transform

Use enough samples, report median plus spread, keep raw data, and verify all runs exit successfully. Never hide a slower scenario or mix incompatible environments in one conclusion.
