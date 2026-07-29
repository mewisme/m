# SBOM export

MVP **0030**. Emit a software bill of materials from the project lock graph.

See also: [`cli.md`](cli.md), [`lockfile.md`](lockfile.md).

## `m sbom`

```text
m sbom [--format cyclonedx|spdx] [--redact-internal] [--redact-pattern <regex>]
```

| Flag | Default | Effect |
|---|---|---|
| `--format` | `cyclonedx` | `cyclonedx` (JSON, BOM 1.5 subset) or `spdx` (tag-value 2.3 subset) |
| `--redact-internal` | off | Replace scoped/internal package names with redacted placeholders |
| `--redact-pattern` | — | Regex of package names to redact |

Stdout receives the document bytes. The command is read-only.

### Graph coverage

Every package instance in the lock graph is included (direct and transitive).
Components carry:

- npm `purl` (`pkg:npm/name@version`)
- integrity hashes when present on the lock entry
- license strings read from `node_modules/<pkg>/package.json` when installed

### CycloneDX

Minimal CycloneDX 1.5 JSON: `metadata.component`, `components[]` with `type`,
`name`, `version`, `purl`, and `hashes` from lock integrity.

Golden: `fixtures/sbom/medium-graph-cyclonedx-golden.json` (from
`fixtures/bench/medium-graph/` after install).

### SPDX

SPDX 2.3 tag-value subset: `SPDXVersion`, `DataLicense`, `DocumentName`,
`PackageName` / `PackageVersion` / `PackageDownloadLocation` per component.

### Redaction

`--redact-internal` redacts scoped names (`@scope/pkg` → placeholder).
`--redact-pattern` applies an additional regex. Redaction affects emitted names
and purls only; lock files on disk are unchanged.

## Limitations (v1)

- License fields come from installed `package.json` or are omitted when absent.
- No VEX, dependency-of relationships, or supplier metadata beyond project name.
- SPDX and CycloneDX outputs are validated by golden tests, not external schema CI.

## Related

- [`audit.md`](audit.md) — vulnerability scan
- [`policy.md`](policy.md) — org policy on licenses and packages
