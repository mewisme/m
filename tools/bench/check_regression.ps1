param(
  [ValidateSet('cold', 'warm')]
  [string]$Mode = 'warm'
)

$ErrorActionPreference = 'Stop'
$root = Resolve-Path (Join-Path $PSScriptRoot '..', '..')
$baselinePath = Join-Path $root 'benchmarks\install-baseline.json'
$caseName = "medium-graph-$Mode"
$noiseBudget = 0.10

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

$medianBaseline = [double]$entry.totalMsMedian
$p95Baseline = if ($null -ne $entry.totalMsP95) { [double]$entry.totalMsP95 } else { $medianBaseline }
$medianLimit = $medianBaseline * (1.0 + $noiseBudget)
$p95Limit = $p95Baseline * (1.0 + $noiseBudget)
$median = [double]$result.medianMs
$p95 = if ($null -ne $result.p95Ms) { [double]$result.p95Ms } else { [double]$result.totalMs }

Write-Host "case=$caseName samples=$($result.samples.Count) medianMs=$median p95Ms=$p95 medianLimit=$medianLimit p95Limit=$p95Limit"

if ($entry.fixtureDigest -and $result.fixtureDigest -and $entry.fixtureDigest -ne $result.fixtureDigest) {
  Write-Host "WARN: fixtureDigest mismatch baseline=$($entry.fixtureDigest) current=$($result.fixtureDigest)"
}

$medianRegression = $median -gt $medianLimit
$p95Regression = $p95 -gt $p95Limit
if ($medianRegression -or $p95Regression) {
  $parts = @()
  if ($medianRegression) {
    $parts += "medianMs $median exceeds limit $medianLimit (+$([int]($noiseBudget * 100))% of baseline $medianBaseline)"
  }
  if ($p95Regression) {
    $parts += "p95Ms $p95 exceeds limit $p95Limit (+$([int]($noiseBudget * 100))% of baseline $p95Baseline)"
  }
  $message = 'bench regression: ' + ($parts -join '; ')
  if ($env:BENCH_WAIVER -eq '1') {
    Write-Host "WARN: $message but BENCH_WAIVER=1"
    exit 0
  }
  Write-Error $message
}

Write-Host 'ok: within baseline median and p95'
