param(
  [ValidateSet('cold', 'warm')]
  [string]$Mode = 'warm'
)

$ErrorActionPreference = 'Stop'
$root = Resolve-Path (Join-Path $PSScriptRoot '..', '..')
$baselinePath = Join-Path $root 'benchmarks\install-baseline.json'
$caseName = "medium-graph-$Mode"

if (-not (Test-Path $baselinePath)) {
  Write-Error "missing baseline file: $baselinePath"
}

$baseline = Get-Content -Raw $baselinePath | ConvertFrom-Json
$entry = $baseline.cases | Where-Object { $_.name -eq $caseName } | Select-Object -First 1
if (-not $entry) {
  Write-Error "baseline case not found: $caseName"
}

$flag = if ($Mode -eq 'cold') { '--cold' } else { '--warm' }
Push-Location $root
try {
  $env:CGO_ENABLED = '0'
  $out = go run ./cmd/m bench install $flag --json 2>&1
  if ($LASTEXITCODE -ne 0) {
    Write-Error "bench install failed: $out"
  }
} finally {
  Pop-Location
}

$line = ($out | Select-Object -Last 1).ToString().Trim()
$result = $line | ConvertFrom-Json
$median = [double]$entry.totalMsMedian
$limit = $median * 1.10
$total = [double]$result.totalMs

Write-Host "case=$caseName totalMs=$total median=$median limit=$limit"

if ($total -gt $limit) {
  if ($env:BENCH_WAIVER -eq '1') {
    Write-Host 'WARN: regression over 10% but BENCH_WAIVER=1'
    exit 0
  }
  Write-Error "bench regression: totalMs $total exceeds limit $limit (+10% of median $median)"
}

Write-Host 'ok: within baseline'
