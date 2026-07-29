param(
  [int]$Count = 10,
  [ValidateSet('cold', 'warm')]
  [string]$Mode = 'cold'
)

$ErrorActionPreference = 'Stop'
$root = Resolve-Path (Join-Path $PSScriptRoot '..', '..')
$flag = if ($Mode -eq 'cold') { '--cold' } else { '--warm' }

Push-Location $root
try {
  $env:CGO_ENABLED = '0'
  for ($i = 1; $i -le $Count; $i++) {
    Write-Host "install loop $i/$Count mode=$Mode"
    go run ./cmd/m bench install $flag --json
    if ($LASTEXITCODE -ne 0) {
      Write-Error "bench install failed on iteration $i"
    }
  }
} finally {
  Pop-Location
}

Write-Host "ok: $Count install iterations completed"
