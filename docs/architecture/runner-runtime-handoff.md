# Runner → runtime handoff (MVP 0046)

MVP **0046** certifies the runner stack and freezes the public contracts runtime
MVPs (**0050+**) may depend on. This document separates guaranteed surfaces from
internal implementation details.

## Guaranteed for runtime consumers

| Surface | Location | Contract |
|---|---|---|
| Process supervision | `internal/process` | Exit codes, signal forwarding, graceful then forced termination, pipe closure |
| Script execution | `internal/runner` (`Run`, `Exec`) | argv construction, env overlay, working directory, lifecycle hooks |
| Verified bin commands | `internal/binresolve` | Ownership verification before direct dispatch |
| Environment construction | `internal/runner` (`BuildEnv`) | npm-compatible script environment |
| `environment-prepared` v1 | `internal/diagnostics` | Event schema in [`runner-events.md`](../runner-events.md) |
| Inspect JSON v1 | `internal/runner/envexec` | Plan-only inspect output; schema in [`runner-events.md`](../runner-events.md) |
| Conformance manifest/report v1 | `internal/conformance` | Runner certification harness |
| Error code mapping | `internal/apperr`, `docs/errors.md` | Stable `ERR_M_*` at CLI boundary |

Runtime code should call these packages through their public APIs and must not
reach into conformance test helpers.

## Not guaranteed (internal)

- `internal/runner/envexec` orchestrator function signatures
- Provider-specific materialization helpers
- Private cache key formats beyond published digests
- Test-only exports in `conformance_export.go`
- Harness isolation environment variable names except documented probes

`internal/runner` must not import `internal/runner/envexec`.
`internal/runner/envexec` must not import `internal/app`.
Conformance packages must not become runtime dependencies.

Enforced by `TestRunnerImportBoundaries` in the runner certification matrix.

## Observable process guarantees

Certified behavior (see [`runner-compatibility.md`](../runner-compatibility.md)):

- Child exit code propagation
- Unix SIGINT → exit 130 where applicable
- Windows signal mapping documented in process capability matrix
- Owned process-group or Job Object cleanup
- Forced kill fallback after graceful grace
- Stdio pipe closure on cancellation

## Experimental surfaces (documented, not runtime prerequisites)

| Surface | Gate | Follow-up |
|---|---|---|
| Direct `m <script>` shortcuts | `MEW_EXPERIMENTAL_DIRECT_SCRIPTS` / config | MVP 0050 may consume verified dispatch outcomes |
| Interactive script picker | not shipped | deferred to 0090 |

## Sign-off checklist

- [x] Runner conformance matrix passes on Linux and Windows CI
- [x] macOS report via CI or committed local evidence under `docs/evidence/runner/0046/`
- [x] `environment-prepared` and inspect schemas frozen with golden tests
- [x] Import boundaries enforced
- [x] Waivers dated with `followUpMVP` where applicable
- [x] `foundation.runner-stabilization` marked shipped in feature inventory
