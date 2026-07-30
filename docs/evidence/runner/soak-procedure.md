# Runner long-soak procedure (manual)

MVP **0046** certifies short soak thresholds in CI (`runner-process-soak`: 100
cycles, 5-minute cap). This procedure documents optional multi-day manual
evidence for process and handle leaks outside the blocking certification path.

## Prerequisites

- Clean checkout at the commit under test
- Node available on `PATH`
- Isolated `HOME` / `USERPROFILE` (do not use developer cache)
- `CGO_ENABLED=0`

## Short soak (certification)

Automated in the runner matrix:

```powershell
$env:CGO_ENABLED = "0"
go test ./tests/conformance/runner/... -count=1 -run '^TestProcessSoakShort$'
```

Thresholds (exact):

| Metric | Limit |
|---|---|
| Minimum cycles | 100 |
| Runner-owned goroutine delta after cleanup + 2 GC | ≤ 5 |
| Active execution leases | 0 |
| Owned staging directories | 0 |
| Runner-owned child processes | 0 |
| Request-owned temp roots | 0 |
| Total timeout | 5 minutes |

## Long soak (manual, optional)

1. Build Mew: `go build ./cmd/m`
2. Create isolated home: `$home = Join-Path $env:TEMP "mew-soak-$(Get-Random)"; New-Item -ItemType Directory -Path $home`
3. Loop script execution for 24–72 hours:

```powershell
$env:HOME = $home
$env:USERPROFILE = $home
$env:CGO_ENABLED = "0"
$fixture = "fixtures/runner/basic-scripts"
for ($i = 0; $i -lt 10000; $i++) {
  go run ./cmd/m --cwd $fixture run append-order -- $i
  if ($i % 100 -eq 0) { [GC]::Collect(); [GC]::WaitForPendingFinalizers() }
}
```

4. After the loop, verify:
   - No orphaned `node` or `m` child processes
   - Temp roots under the isolated home are empty or documented
   - FD/handle count stable versus baseline (record both numbers)

5. Record evidence under `docs/evidence/runner/0046/`:
   - commit SHA
   - OS/arch, Go and Node versions
   - cycle count and duration
   - goroutine delta if measured
   - FD/handle counts if available
   - pass/fail conclusion

Long soak does not block MVP 0046 ship; short soak in CI does.
