# Verify plan generation is idempotent: two runs must not change plans/.

$ErrorActionPreference = 'Stop'
$root = Resolve-Path (Join-Path $PSScriptRoot '..' '..')
$gen = Join-Path $root 'plans\scripts\enrich-and-generate.ps1'

function Invoke-PlanGeneration {
  & pwsh -NoProfile -File $gen
  if ($LASTEXITCODE -ne 0) {
    throw "enrich-and-generate.ps1 failed with exit $LASTEXITCODE"
  }
}

Push-Location $root
try {
  Write-Host 'plan-generation idempotency: first run'
  Invoke-PlanGeneration
  git diff --exit-code plans/
  if ($LASTEXITCODE -ne 0) {
    throw 'plans/ changed after first generation run'
  }

  Write-Host 'plan-generation idempotency: second run'
  Invoke-PlanGeneration
  git diff --exit-code plans/
  if ($LASTEXITCODE -ne 0) {
    throw 'plans/ changed after second generation run (not idempotent)'
  }

  Write-Host 'ok: plan generation is idempotent'
}
finally {
  Pop-Location
}
