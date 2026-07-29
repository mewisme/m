# PM-core security (0031)

Subset of the program threat model (**0082**) covering package-manager core
surfaces shipped in MVPs **0010–0030**. Full threat-model program remains
**0082**; runner, runtime, shim, and updater boundaries are out of scope here.

See also: [`core-certification.md`](core-certification.md),
[`patch.md`](patch.md), [`lifecycle.md`](lifecycle.md), [`audit.md`](audit.md).

## Trust boundaries

| Boundary | Hostile input | Controls |
|---|---|---|
| Registry HTTP | Packuments, tarballs, redirects | TLS, size limits, integrity (SRI/shasum), bounded workers |
| Git / hosted / file sources | Repos, archives, path refs | Protocol validation, shallow fetch policy, no hook execution on fetch |
| Archive extraction | `npm pack` tarballs | Path traversal guards, max entry count/size, symlink policy |
| Patch application | Unified diff files | Path sandbox, Option B preflight, atomic stage + transaction |
| Lockfile parse | Incumbent + `m.lock` bytes | Typed errors, duplicate-key rejection, bounded YAML depth (pnpm) |
| Lifecycle scripts | `preinstall` / `postinstall` / … | Trust list, sandbox (platform-dependent), rollback on failure |
| Diagnostics | Errors, reporters, `--json` | Credential redaction, no secrets in plans or journals |
| Policy evaluation | `mew.policy.json` | Deny lists, waivers with expiry, install-time enforcement |
| Provenance | Attestation bundles | Fixture-based verification; live Sigstore deferred |

## Archive extraction

Implementation: `internal/archive`.

- Reject absolute paths, `..` traversal, and entries outside the extraction root
- Enforce maximum uncompressed size and entry count (see security fuzz rules)
- Verify integrity digest before extraction when lock/registry provides one
- Tests: `internal/archive` security tests, integration evil-tarball fixtures

**Residual risk:** extraction safety does not make package code safe to execute;
lifecycle trust remains a separate gate.

## Registry authentication and redaction

Implementation: `internal/registry`, `internal/fetch`, `internal/diagnostics`.

- Bearer tokens and `_auth` credentials must not appear in stderr, JSON
  reports, transaction journals, or crash artifacts
- Reporter redaction rules: [`reporters.md`](reporters.md)
- Redirect limits and custom CA handling documented in [`registry.md`](registry.md)

## Patch sandbox

Implementation: `internal/archive/patch_*.go`, `internal/resolver/patch.go`.

Pass 20 shipped controls (evidence: `.agents/stabilization-pass20-score.md`):

- `resolvePatchTarget` + `fsx.GuardAncestors` — patch paths cannot escape package root
- Store copy-on-write — patched derivatives staged, not written into global store paths
- Fail-closed preflight — unsupported diff ops rejected before apply
- Byte-derived `patch_hash` — identity from patch file bytes, not path strings
- Atomic apply via transaction commit/rollback

## Policy gate

Implementation: `internal/policy`, `m policy check`, install pre-commit hook.

- Deny packages and licenses from org policy file
- Waivers require explicit expiry (RFC3339)
- `severityThreshold: error` blocks install with `ERR_M_POLICY`

## Provenance (fixture verification)

Implementation: `internal/provenance`, `m verify provenance`.

- Verifies npm-style provenance attestations against fixture bundles in tests
- `m publish --provenance` records publish-time hook metadata (0027)
- **Not in 0031 scope:** live Sigstore registry verification, feed signing for advisories

## Lifecycle and script execution

See [`lifecycle.md`](lifecycle.md). Untrusted packages block script execution
until `m trust` / `m approve-builds`. Sandbox reduces filesystem and network
reach but is not a complete containment boundary — document as best-effort.

## Mandatory review triggers

Changes touching the following require security review per 0082 discipline:

- Archive extraction limits or path rules
- Patch apply semantics or sandbox roots
- Registry auth header handling or logging
- Policy schema or install enforcement order
- Provenance verification trust roots
- Transaction journal fields that may contain user paths or env

## Evidence map

| Control | Tests / docs |
|---|---|
| Evil tarball | `internal/archive` tests, fuzz |
| Patch traversal | `patchapply_security_test.go`, pass 20 |
| Auth redaction | `internal/diagnostics`, reporter tests |
| Policy deny | `tests/integration/policy_test.go` |
| Provenance fixture | `tests/integration/provenance_test.go` |
| Audit offline digest | `tests/integration/audit_test.go` |
| SBOM redaction | `tests/integration/sbom_test.go` |
