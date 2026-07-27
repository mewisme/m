# Hoisted linker fixtures

Graph fixtures for `internal/linker/hoisted` placement tests. Each directory
contains `graph.json` with packages, edges, and expected placement paths
(relative to the project root).

Cases:

- `multi-version-x` — root and nested versions of the same package name
- `shared-child-blocked` — two parents each need a distinct nested copy
- `scoped-conflict` — scoped package version collision nesting
- `nested-bins` — root `.bin` only for root direct dependencies
- `cyclic-graph` — valid dependency cycle terminates placement walk
- `peer-context-instances` — same name/version, different peer provider keys
