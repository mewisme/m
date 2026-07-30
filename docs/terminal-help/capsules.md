<!--
Ownership: curated terminal help for `m help capsules`.
Authoritative: docs/ (capsule command surfaces); no dedicated long-form doc yet.
-->

# Capsules

Capsules are portable dependency bundles for offline or verified restore flows.

## Commands

```text
m capsule --help
m pack --help
```

## Safety

- Verify integrity before trusting a capsule as an install source.
- Capsule restore is a mutation and must go through the transaction boundary.
- Do not treat a capsule label as proof of provenance by itself.

## See also

- `m help snapshots`
- `m help errors ERR_M_INTEGRITY`
- docs/compatibility-axes.md
