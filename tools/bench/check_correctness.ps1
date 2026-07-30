param(
  [ValidateSet('cold', 'warm')]
  [string]$Mode = 'warm'
)

$ErrorActionPreference = 'Stop'
$root = Resolve-Path (Join-Path $PSScriptRoot '..', '..')
$outPath = Join-Path $root 'bench-result.json'
$minSamples = 7

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

if (-not $result.case) { Write-Error 'bench JSON missing case' }
if (-not $result.mode) { Write-Error 'bench JSON missing mode' }
if ($result.samples.Count -lt $minSamples) {
  Write-Error "bench samples=$($result.samples.Count) require >= $minSamples"
}
if (-not $result.fixtureDigest) {
  Write-Error 'bench JSON missing fixtureDigest'
}
if (-not $result.medianMs) {
  Write-Error 'bench JSON missing medianMs'
}

$artifact = [ordered]@{
  checkedAt = (Get-Date).ToUniversalTime().ToString('o')
  mode      = $Mode
  result    = $result
}
$artifact | ConvertTo-Json -Depth 8 | Set-Content -Path $outPath -Encoding utf8NoBOM

Write-Host "ok: bench correctness case=$($result.case) samples=$($result.samples.Count) medianMs=$($result.medianMs)"
