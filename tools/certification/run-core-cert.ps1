param(
  [Parameter(Mandatory = $true)]
  [ValidateSet('core-cert-fast', 'core-cert', 'core-cert-security', 'core-cert-crash', 'core-cert-performance')]
  [string]$Target
)

$ErrorActionPreference = 'Stop'
$root = Resolve-Path (Join-Path $PSScriptRoot '..' '..')
$manifestPath = Join-Path $PSScriptRoot 'core-manifest.json'
$manifest = Get-Content -Raw $manifestPath -Encoding utf8 | ConvertFrom-Json

if (-not $manifest.targets.PSObject.Properties.Name.Contains($Target)) {
  throw "unknown target: $Target"
}

$stepIds = @($manifest.targets.$Target.steps)
Write-Host "core-cert target=$Target steps=$($stepIds.Count)"

$failures = 0
Push-Location $root
try {
  foreach ($stepId in $stepIds) {
    $step = $manifest.steps.$stepId
    if (-not $step) {
      throw "manifest missing step $stepId"
    }

    $skipStep = $false
    if ($step.requiresTools) {
      $missing = @()
      foreach ($tool in @($step.requiresTools)) {
        if ($tool -eq 'govulncheck') {
          if (-not (Get-Command govulncheck -ErrorAction SilentlyContinue)) {
            $skipStep = $true
          }
        }
        elseif ($tool -eq 'node') {
          if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
            $missing += $tool
          }
        }
        elseif ($tool -eq 'pnpm') {
          if (-not (Get-Command pnpm -ErrorAction SilentlyContinue)) {
            $missing += $tool
          }
        }
      }
      if ($skipStep) {
        Write-Host "skip $($step.id): required tool not installed"
        continue
      }
      if ($missing.Count -gt 0 -and $step.blocking) {
        Write-Error "step $($step.id) requires tools: $($missing -join ', ')"
      }
      if ($missing.Count -gt 0 -and -not $step.blocking) {
        Write-Host "skip $($step.id): missing tools $($missing -join ', ')"
        continue
      }
    }

    if ($step.env) {
      foreach ($p in $step.env.PSObject.Properties) {
        Set-Item -Path "env:$($p.Name)" -Value $p.Value
      }
    }

    Write-Host "==> $($step.id): $($step.command)"
    if ($step.shell -eq 'pwsh') {
      pwsh -NoProfile -Command $step.command
    }
    else {
      Invoke-Expression $step.command
    }
    if ($LASTEXITCODE -ne 0) {
      if ($step.blocking) {
        Write-Error "step $($step.id) failed with exit $LASTEXITCODE"
      }
      Write-Host "WARN: non-blocking step $($step.id) failed with exit $LASTEXITCODE"
      $failures++
    }
  }
}
finally {
  Pop-Location
}

if ($failures -gt 0) {
  Write-Host "completed with $failures non-blocking failure(s)"
}
Write-Host "ok: $Target"
