# Nub Reference Snapshot

- Repository: `nubjs/nub`
- Reference branch: `main`
- Reference commit inspected during plan creation: `08a804359ef301ef8b9307f1258cc185b3270698`
- License observed in the repository workspace metadata: MIT
- Architecture observed: Rust CLI over stock Node, embedded Aube package-manager libraries, OXC-based native transform addon, and JavaScript runtime loader/preload assets.
- Important product boundary: Nub augments installed Node rather than shipping a custom JavaScript runtime.
- Important compatibility boundary: package-manager CLI compatibility and lockfile compatibility are independent axes.
- Important Mew divergence: Mew intentionally supports exact direct script shortcuts such as `m dev`; Nub intentionally requires `nub run dev`.

This snapshot is a research input. Before implementation begins, rerun the source inventory and record any relevant upstream changes.
