# Generate pnpm lock fixtures using pinned versions from pnpm-versions.env.
# Does not replace hand-maintained fixtures under fixtures/locks/pnpm/{v6,v9,v10,v11}.

param(
    [string]$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
)

$ErrorActionPreference = "Stop"

$envFile = Join-Path $PSScriptRoot "pnpm-versions.env"
if (-not (Test-Path $envFile)) {
    throw "missing $envFile"
}

Get-Content $envFile | ForEach-Object {
    if ($_ -match '^\s*([A-Z0-9_]+)\s*=\s*(.+)\s*$') {
        Set-Item -Path "env:$($Matches[1])" -Value $Matches[2].Trim()
    }
}

function Invoke-PnpmFixture {
    param(
        [string]$Major,
        [string]$Version,
        [string]$PackageJson
    )
    $outDir = Join-Path $RepoRoot "fixtures\locks\pnpm\generated\v$Major-basic"
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null
    Copy-Item -Force $PackageJson (Join-Path $outDir "package.json")
    Push-Location $outDir
    try {
        Write-Host "generating $outDir with pnpm@$Version"
        npx --yes "pnpm@$Version" install --lockfile-only
        if ($LASTEXITCODE -ne 0) { throw "pnpm install failed for major $Major" }
    }
    finally {
        Pop-Location
    }
}

$pkgBasic = Join-Path $RepoRoot "fixtures\locks\pnpm\v9\package.json"
if (-not (Test-Path $pkgBasic)) {
    throw "missing seed package.json at $pkgBasic"
}

Invoke-PnpmFixture -Major "9" -Version $env:PNPM9_VERSION -PackageJson $pkgBasic
Invoke-PnpmFixture -Major "10" -Version $env:PNPM10_VERSION -PackageJson $pkgBasic
Invoke-PnpmFixture -Major "11" -Version $env:PNPM11_VERSION -PackageJson $pkgBasic

Write-Host "done — see fixtures/locks/pnpm/generated/"
