# Charter Consistency Checklist

Use this checklist when closing any MVP after 0001. All items must pass or be explicitly deferred with a plan reference.

## Identity and references

- [ ] Public docs use `m`, `mx`, and `m.lock` consistently
- [ ] Nub is described as a **behavioral reference**, not a source port target
- [ ] No claim of parity, performance, or security without fixture or benchmark evidence

## Compatibility

- [ ] Changed behavior classified as parity, intentional divergence, extension, or deferred
- [ ] [`compatibility-axes.md`](compatibility-axes.md) updated for affected axes
- [ ] Lockfile preservation rules respected (no silent format migration)
- [x] Direct script shortcut precedence shipped in MVP 0042
- [x] Local binary execution (`m exec`) and verified direct bin dispatch shipped in MVP 0043

## Naming

- [ ] New public identifiers match [`naming.md`](naming.md) or have an ADR
- [ ] Error codes follow `ERR_M_*` pattern (MVP 0005+)
- [ ] Environment variables use `MEW_*` or documented exceptions

## Safety

- [ ] Install-family mutations use the transaction boundary
- [ ] Integrity verification not weakened for test convenience
- [ ] Secrets not written to logs, lockfiles, or diagnostics

## Documentation

- [ ] User-visible changes documented in the same change
- [ ] Migration notes updated when lockfile or identity behavior changes
- [ ] Feature inventory (0002) updated when behavior ships

## Handoff (agents)

Before submitting work, provide:

1. Behavior summary and compatibility target
2. Files and public interfaces changed
3. Test, benchmark, and static-analysis commands run
4. Known gaps, deferred cases, platform limits
5. Determinism evidence for generated files
6. Rollback note for persistent-format changes
