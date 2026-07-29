param(
  [int]$Count = 100,
  [ValidateSet('cold', 'warm')]
  [string]$Mode = 'cold',
  [string]$Project = ''
)

$ErrorActionPreference = 'Stop'
$root = Resolve-Path (Join-Path $PSScriptRoot '..', '..')
$flag = if ($Mode -eq 'cold') { '--cold' } else { '--warm' }

$fixtureArgs = @()
if ($Project) {
  $fixtureArgs = @('--fixture', $Project)
  if ($Project -match 'workspace') {
    $env:MEW_EXPERIMENTAL_WORKSPACES = '1'
    $env:MEW_EXPERIMENTAL_ISOLATED_LINKER = '1'
  }
}

$label = if ($Project) { $Project } else { 'default' }

Push-Location $root
try {
  $env:CGO_ENABLED = '0'
  for ($i = 1; $i -le $Count; $i++) {
    Write-Host "install loop $i/$Count mode=$Mode project=$label"
    go run ./cmd/m bench install $flag --json @fixtureArgs
    if ($LASTEXITCODE -ne 0) {
      Write-Error "bench install failed on iteration $i"
    }
  }
} finally {
  Pop-Location
}

Write-Host "ok: $Count install iterations completed project=$label"
