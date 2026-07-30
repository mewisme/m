param(
  [ValidateSet('cold', 'warm')]
  [string]$Mode = 'warm'
)

$ErrorActionPreference = 'Stop'
$root = Resolve-Path (Join-Path $PSScriptRoot '..', '..')
$baselinePath = Join-Path $root 'benchmarks\install-baseline.json'
$waiversPath = Join-Path $root 'benchmarks\waivers.json'
$outPath = Join-Path $root 'bench-result.json'
$caseName = "medium-graph-$Mode"
$noiseBudget = 0.10
$minSamples = 7

if ($env:BENCH_WAIVER -eq '1') {
  Write-Warning 'BENCH_WAIVER=1 is deprecated; use benchmarks/waivers.json structured waivers'
}

function Get-RunnerClass {
  param($Result)
  if ($Result.runnerClass) { return $Result.runnerClass }
  if ($env:MEW_BENCH_RUNNER_CLASS) { return $env:MEW_BENCH_RUNNER_CLASS }
  if ($env:GITHUB_ACTIONS -eq 'true') {
    $runner = if ($env:RUNNER_OS) { $env:RUNNER_OS.ToLower() } else { 'unknown' }
    return "github-actions-$runner"
  }
  if ($IsWindows) { return 'local-windows' }
  if ($IsLinux) { return 'local-linux' }
  if ($IsMacOS) { return 'local-darwin' }
  return 'local-unknown'
}

function Test-Waiver {
  param($Waivers, $Result, [string]$RunnerClass)
  $today = Get-Date
  foreach ($w in $Waivers) {
    if ($w.case -ne $Result.case) { continue }
    if ($w.benchmarkMode -and $w.benchmarkMode -ne $Result.mode) { continue }
    if ($w.os -and $w.os -ne $Result.os) { continue }
    if ($w.arch -and $w.arch -ne $Result.arch) { continue }
    if ($w.runnerClass -and $w.runnerClass -ne $RunnerClass) { continue }
    if ($w.expires) {
      $exp = [datetime]::Parse($w.expires)
      if ($today -gt $exp) { continue }
    }
    return $w
  }
  return $null
}

if (-not (Test-Path $baselinePath)) {
  Write-Error "missing baseline file: $baselinePath"
}

Push-Location $root
try {
  $env:CGO_ENABLED = '0'
  $out = go run ./cmd/m bench install --$Mode --json 2>&1
  if ($LASTEXITCODE -ne 0) {
    Write-Error "bench install failed: $out"
  }
} finally {
  Pop-Location
}

$line = ($out | Select-Object -Last 1).ToString().Trim()
$result = $line | ConvertFrom-Json
$runnerClass = Get-RunnerClass -Result $result

if ($result.samples.Count -lt $minSamples) {
  Write-Error "bench samples=$($result.samples.Count) require >= $minSamples"
}

$baseline = Get-Content -Raw $baselinePath | ConvertFrom-Json
$entry = $baseline.cases | Where-Object {
  $_.name -eq $caseName -and
  $_.os -eq $result.os -and
  $_.arch -eq $result.arch -and
  $_.benchmarkMode -eq $Mode -and
  ((-not $_.runnerClass) -or ($_.runnerClass -eq $runnerClass))
} | Select-Object -First 1

$waivers = @()
if (Test-Path $waiversPath) {
  $waivers = (Get-Content -Raw $waiversPath | ConvertFrom-Json).waivers
}

if (-not $entry) {
  $waiver = Test-Waiver -Waivers $waivers -Result $result -RunnerClass $runnerClass
  if ($waiver) {
    Write-Host "WARN: no platform baseline for case=$caseName os=$($result.os) arch=$($result.arch) runner=$runnerClass; waiver owner=$($waiver.owner) reason=$($waiver.reason)"
    $artifact = [ordered]@{
      checkedAt   = (Get-Date).ToUniversalTime().ToString('o')
      mode        = $Mode
      runnerClass = $runnerClass
      waived      = $true
      waiver      = $waiver
      result      = $result
    }
    $artifact | ConvertTo-Json -Depth 8 | Set-Content -Path $outPath -Encoding utf8NoBOM
    exit 0
  }
  Write-Error "no baseline case for case=$caseName os=$($result.os) arch=$($result.arch) runner=$runnerClass mode=$Mode"
}

if ($entry.fixtureDigest -and $result.fixtureDigest -and $entry.fixtureDigest -ne $result.fixtureDigest) {
  Write-Error "fixtureDigest mismatch baseline=$($entry.fixtureDigest) current=$($result.fixtureDigest)"
}

$medianBaseline = [double]$entry.totalMsMedian
$p95Baseline = if ($null -ne $entry.totalMsP95) { [double]$entry.totalMsP95 } else { $medianBaseline }
$medianLimit = $medianBaseline * (1.0 + $noiseBudget)
$p95Limit = $p95Baseline * (1.0 + $noiseBudget)
$median = [double]$result.medianMs
$p95 = if ($null -ne $result.p95Ms) { [double]$result.p95Ms } else { [double]$result.totalMs }

Write-Host "case=$caseName runner=$runnerClass samples=$($result.samples.Count) medianMs=$median p95Ms=$p95 medianLimit=$medianLimit p95Limit=$p95Limit"

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
  $waiver = Test-Waiver -Waivers $waivers -Result $result -RunnerClass $runnerClass
  if ($waiver) {
    Write-Host "WARN: $message; structured waiver owner=$($waiver.owner)"
    $artifact = [ordered]@{
      checkedAt   = (Get-Date).ToUniversalTime().ToString('o')
      mode        = $Mode
      runnerClass = $runnerClass
      waived      = $true
      waiver      = $waiver
      regression  = $message
      result      = $result
    }
    $artifact | ConvertTo-Json -Depth 8 | Set-Content -Path $outPath -Encoding utf8NoBOM
    exit 0
  }
  if ($env:BENCH_WAIVER -eq '1') {
    Write-Warning "$message but BENCH_WAIVER=1 is deprecated"
    exit 0
  }
  Write-Error $message
}

$artifact = [ordered]@{
  checkedAt   = (Get-Date).ToUniversalTime().ToString('o')
  mode        = $Mode
  runnerClass = $runnerClass
  baseline    = $entry
  result      = $result
}
$artifact | ConvertTo-Json -Depth 8 | Set-Content -Path $outPath -Encoding utf8NoBOM
Write-Host 'ok: within baseline median and p95'
