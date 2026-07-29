# Supply-chain policy

MVP **0030**. Project-level org rules for denied packages and licenses, plus
install-time enforcement. Separate from resolver release-age policy and
lifecycle script trust.

See also: [`lifecycle.md`](lifecycle.md), [`resolver.md`](resolver.md),
[`errors.md`](errors.md).

## Org policy file

Mew searches, in order:

1. `mew.policy.json` at the project root
2. `.mew/policy.json`

Missing files mean no org policy (pass). JSON schema v1:

```json
{
  "schemaVersion": 1,
  "denyPackages": ["bad-pkg"],
  "denyLicenses": ["GPL-3.0"],
  "severityThreshold": "error",
  "waivers": [
    {
      "package": "legacy-pkg",
      "reason": "migration in progress",
      "expires": "2026-12-31T23:59:59Z"
    }
  ]
}
```

| Field | Default | Meaning |
|---|---|---|
| `denyPackages` | `[]` | Exact package names (case-insensitive) blocked |
| `denyLicenses` | `[]` | License strings or substrings matched against installed `package.json` `license` |
| `severityThreshold` | `error` | `warn` reports only; `error` fails `m policy check` and blocks install |
| `waivers` | `[]` | Temporary exemption by package name or lock key; optional RFC3339 `expires` |

## `m policy check`

```text
m policy check [--json]
```

Dry evaluation against the lock graph and `node_modules` license metadata. No
install or lock mutation. Exits non-zero with `ERR_M_POLICY` when
`severityThreshold` is `error` and violations exist.

JSON output: `{ "passed": bool, "violations": […] }` with sorted violations.

## Install gate

During `m install` (and mutation-family commands using the install transaction),
the **validate** phase calls org-policy evaluation after link planning. Error-
severity violations abort before commit with `ERR_M_POLICY` and a summary of
denied packages or licenses.

Licenses are read from staged extract directories when available, otherwise from
the target `node_modules` tree.

Fixture: `fixtures/policy/deny-gpl/`.

## Resolver policy (separate)

[`internal/policy/policy.go`](../internal/policy/policy.go) also defines
resolver-time rules consumed by [`resolver.md`](resolver.md):

- `MinimumReleaseAge` — reject freshly published versions
- `RejectDeprecated` — skip deprecated packument versions

These apply at **resolve** time and are independent of `mew.policy.json`.

## Lifecycle trust (separate)

[`lifecycle.md`](lifecycle.md) covers `m trust` / `m approve-builds` for
lifecycle script execution. Untrusted scripts return `ERR_M_POLICY` with
operation `lifecycle.*`, not `app.policy`.

## Limitations (v1)

- License matching is substring-based on the declared `license` field only.
- No SPDX expression parser, no OSI category rules, no registry provenance gate
  (see [`audit.md`](audit.md) and provenance verify).
- Waivers are file-based; no remote policy server.

## Related

- [`audit.md`](audit.md) — advisory scan
- [`sbom.md`](sbom.md) — SBOM export with license fields
