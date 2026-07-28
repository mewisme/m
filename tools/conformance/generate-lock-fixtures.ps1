#Requires -Version 7.0
<#
.SYNOPSIS
  Regenerate committed lock bridge conformance fixtures from pinned pnpm binaries.

.DESCRIPTION
  Reads family sources from fixtures/locks/sources/pnpm/<family>/ and writes
  fixtures/locks/generated/pnpm-{9,10,11}/<family>/ with honest metadata.json.
  Use -Generate to run isolated temp homes with exact pnpm@X.Y.Z via corepack.
#>
param(
    [switch]$Generate,
    [string[]]$Families = @(
        'basic', 'transitive', 'optional', 'peer-context', 'multi-version',
        'scoped', 'workspace', 'catalog', 'override', 'platform', 'importer-meta',
        'alias', 'patch', 'binary'
    ),
    [int[]]$Majors = @(9, 10, 11)
)

$ErrorActionPreference = 'Stop'
$root = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
$versions = Join-Path $PSScriptRoot 'pnpm-versions.env'
if (-not (Test-Path $versions)) { throw "missing $versions" }
Get-Content $versions | ForEach-Object {
    if ($_ -match '^(PNPM\d+_VERSION)=(.+)$') { Set-Variable -Name $Matches[1] -Value $Matches[2].Trim() }
}

function Write-Metadata($dest, $major, $family, $pnpmVersion, $command) {
    $lock = Join-Path $dest 'pnpm-lock.yaml'
    if (-not (Test-Path $lock)) { throw "missing lock at $lock" }
    $hash = (Get-FileHash $lock -Algorithm SHA256).Hash.ToLower()
    $meta = [ordered]@{
        producer          = 'pnpm'
        producerVersion   = $pnpmVersion
        producerMajor     = [int]$major
        family            = $family
        node              = (node -v 2>$null)
        os                = [System.Runtime.InteropServices.RuntimeInformation]::OSDescription
        arch              = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower()
        lockfileVersion   = '9.0'
        generationSignals = @("lockfileVersion=9.0", "family=$family")
        confidence        = 'certain'
        command           = $command
        timestamp         = (Get-Date -Format o)
        lockfileSha256    = $hash
    }
    $meta | ConvertTo-Json -Depth 6 | Set-Content (Join-Path $dest 'metadata.json') -Encoding utf8NoBOM
}

function Copy-FamilySource($src, $dest) {
    New-Item -ItemType Directory -Force -Path $dest | Out-Null
    Get-ChildItem -Path $src -Recurse -File | ForEach-Object {
        $rel = $_.FullName.Substring($src.Length).TrimStart('\', '/')
        $out = Join-Path $dest $rel
        $parent = Split-Path $out -Parent
        if (-not (Test-Path $parent)) { New-Item -ItemType Directory -Force -Path $parent | Out-Null }
        Copy-Item $_.FullName $out -Force
    }
}

function Invoke-PnpmLockOnly($workDir, $pnpmVersion) {
    Push-Location $workDir
    try {
        $env:COREPACK_ENABLE_DOWNLOAD_PROMPT = '0'
        corepack enable 2>$null | Out-Null
        corepack prepare "pnpm@$pnpmVersion" --activate | Out-Null
        $pnpm = Get-Command pnpm -ErrorAction Stop
        & $pnpm.Source install --lockfile-only --ignore-scripts 2>&1 | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "pnpm install failed in $workDir (exit $LASTEXITCODE)"
        }
    } finally { Pop-Location }
}

foreach ($major in $Majors) {
    $ver = (Get-Variable "PNPM${major}_VERSION").Value
    foreach ($family in $Families) {
        $src = Join-Path $root "fixtures/locks/sources/pnpm/$family"
        if (-not (Test-Path $src)) { throw "missing source family $src" }
        $dest = Join-Path $root "fixtures/locks/generated/pnpm-$major/$family"
        if ($Generate) {
            $work = Join-Path ([System.IO.Path]::GetTempPath()) "mew-lock-fix-$major-$family"
            if (Test-Path $work) { Remove-Item -Recurse -Force $work }
            Copy-FamilySource $src $work
            Invoke-PnpmLockOnly $work $ver
            if (-not (Test-Path (Join-Path $work 'pnpm-lock.yaml'))) {
                throw "pnpm did not write lockfile for $family major=$major"
            }
            Copy-FamilySource $work $dest
            $cmd = "corepack prepare pnpm@$ver --activate; pnpm install --lockfile-only --ignore-scripts (family=$family)"
        } else {
            if (-not (Test-Path (Join-Path $dest 'pnpm-lock.yaml'))) {
                Write-Warning "skip $dest — no committed lock; run with -Generate"
                continue
            }
            $cmd = "committed generated fixture (family=$family, pnpm@$ver)"
        }
        Write-Metadata $dest $major $family $ver $cmd
        Write-Host "ok: pnpm-$major/$family"
    }
}

# Nub fixture families — derived from pnpm-9 generated locks (pnpm-v9-shaped nub.lock)
$nubMap = [ordered]@{
    'nub-basic'      = 'basic'
    'nub-transitive' = 'transitive'
    'nub-workspace'  = 'workspace'
    'nub-catalog'    = 'catalog'
    'nub-peer'       = 'peer-context'
    'nub-optional'   = 'optional'
}
foreach ($entry in $nubMap.GetEnumerator()) {
    $family = $entry.Value
    $name = $entry.Key
    $pnpmDir = Join-Path $root "fixtures/locks/generated/pnpm-9/$family"
    $nubDest = Join-Path $root "fixtures/locks/generated/$name"
    if (-not (Test-Path (Join-Path $pnpmDir 'pnpm-lock.yaml'))) {
        Write-Warning "skip $name — missing pnpm-9/$family"
        continue
    }
    New-Item -ItemType Directory -Force -Path $nubDest | Out-Null
    Copy-Item (Join-Path $pnpmDir 'package.json') $nubDest -Force
    if (Test-Path (Join-Path $pnpmDir 'pnpm-workspace.yaml')) {
        Copy-Item (Join-Path $pnpmDir 'pnpm-workspace.yaml') $nubDest -Force
    }
    if (Test-Path (Join-Path $pnpmDir 'packages')) {
        Copy-Item (Join-Path $pnpmDir 'packages') (Join-Path $nubDest 'packages') -Recurse -Force
    }
    $lock = Get-Content (Join-Path $pnpmDir 'pnpm-lock.yaml') -Raw
    if ($lock -notmatch 'nubVersion:') {
        $lock = $lock.TrimEnd() + "`nnubVersion: `"1.0.0`"`n"
    }
    Set-Content (Join-Path $nubDest 'nub.lock') $lock -Encoding utf8NoBOM -NoNewline
    $nubHash = (Get-FileHash (Join-Path $nubDest 'nub.lock') -Algorithm SHA256).Hash.ToLower()
    [ordered]@{
        producer          = 'nub'
        producerVersion   = 'pnpm-9-shaped'
        producerMajor     = 9
        family            = $name
        node              = (node -v 2>$null)
        os                = [System.Runtime.InteropServices.RuntimeInformation]::OSDescription
        arch              = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower()
        lockfileVersion   = '9.0'
        generationSignals = @('pnpm-v9-shaped', 'nub.lock', "derived-from=pnpm-9/$family")
        confidence        = 'manual'
        command           = "derived from fixtures/locks/generated/pnpm-9/$family pnpm-lock.yaml"
        timestamp         = (Get-Date -Format o)
        lockfileSha256    = $nubHash
    } | ConvertTo-Json -Depth 6 | Set-Content (Join-Path $nubDest 'metadata.json') -Encoding utf8NoBOM
    Write-Host "ok: $name"
}

Write-Host "done: fixtures/locks/generated refreshed"
