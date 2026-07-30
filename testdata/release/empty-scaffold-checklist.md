# Empty-scaffold release checklist

Dry-run checklist for releasing the current Mew scaffold (foundation complete
through MVP 0009). No calendar date; run before tagging a 0.x alpha.

## Preconditions

- [ ] `docs/release-train.md` reviewed for channel and stop-the-line criteria
- [ ] `features/milestones.json` validates (acyclic; stabilization order)
- [ ] Feature inventory `foundation.release-train` is `shipped`

## Build and test

- [ ] `go test ./... -count=1`
- [ ] `go vet ./...`
- [ ] `go run ./cmd/m version`
- [ ] `go run ./cmd/m features --format table` (inventory loads)
- [ ] `make fuzz-smoke` (or `python tools/fuzz_smoke.py`)
- [ ] `make conformance` (or `go test ./tests/conformance/... -count=1`)

## Hermetic and experimental policy

- [ ] Normal CI does not call the public npm registry
- [ ] Fixture registry checksums verify (`testkit.LoadRegistry`)
- [ ] Experimental flag naming documented (`--experimental-*`, `MEW_EXPERIMENTAL_*`)
- [ ] No secrets in fixtures, lockfiles, or diagnostics goldens

## Stop-the-line (must be clear)

- [ ] Integrity failure is a release blocker
- [ ] Lock corruption is a release blocker
- [ ] Credential leak in logs/diagnostics is a release blocker

## Rollback note

- [ ] Tag points at a commit that can be abandoned without on-disk format
      migration (foundation has no public lock writer yet)
