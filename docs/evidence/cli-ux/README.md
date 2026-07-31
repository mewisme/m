# CLI UX certification evidence (UX-0008)

Platform and performance evidence for the presentation system (UX-0001–0007)
and the UX-0008 certification gate.

## Evidence slots

| Platform | Status | Artifact |
|---|---|---|
| Windows native | **Measured** (this host) | [`windows-2026-07-31.md`](windows-2026-07-31.md) |
| Linux | **CI** — Ubuntu `conformance-cli-ux` on tip (local Docker unavailable) | [`linux-docker-slot.md`](linux-docker-slot.md) |
| macOS | **CI** — `conformance-cli-ux` on tip run `30591819891` | [`macos-ci-slot.md`](macos-ci-slot.md) |

## Related documents

- Performance baseline: [`plans/ux/performance-baseline.md`](../../../plans/ux/performance-baseline.md)
- Charm pins / license / vuln notes: [`plans/ux/charm-dependency-review.md`](../../../plans/ux/charm-dependency-review.md)
- Accessibility checklist: [`plans/ux/accessibility-evidence.md`](../../../plans/ux/accessibility-evidence.md)
- Architecture: [`docs/architecture/cli-presentation.md`](../../architecture/cli-presentation.md)

## Conformance entry point

```powershell
$env:CGO_ENABLED = "0"
go run ./cmd/m conformance run cli-ux --json
```

The versioned matrix lives at `tests/conformance/cli-ux/manifest.json`.
Platform certification: Windows local + Linux/macOS via `conformance-cli-ux`.
