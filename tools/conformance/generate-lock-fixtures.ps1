#Requires -Version 7.0
<#
.SYNOPSIS
  Regenerate committed lock bridge conformance fixtures from pinned pnpm binaries.

.DESCRIPTION
  Copies fixture families into fixtures/locks/generated/ and writes metadata.json
  with producer version, command, and SHA-256 of the lockfile. Requires pnpm on PATH
  matching tools/conformance/pnpm-versions.env when -Generate is set.
#>
param(
    [switch]$Generate
)

$ErrorActionPreference = 'Stop'
$root = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
$versions = Join-Path $PSScriptRoot 'pnpm-versions.env'
if (-not (Test-Path $versions)) { throw "missing $versions" }
Get-Content $versions | ForEach-Object {
    if ($_ -match '^(PNPM\d+_VERSION)=(.+)$') { Set-Variable -Name $Matches[1] -Value $Matches[2].Trim() }
}

function Write-Metadata($dest, $major, $pnpmVersion, $command) {
    $lock = Join-Path $dest 'pnpm-lock.yaml'
    if (-not (Test-Path $lock)) { throw "missing lock at $lock" }
    $hash = (Get-FileHash $lock -Algorithm SHA256).Hash.ToLower()
    $meta = [ordered]@{
        producer         = 'pnpm'
        producerVersion  = $pnpmVersion
        producerMajor    = [int]$major
        node             = (node -v 2>$null)
        os               = [System.Runtime.InteropServices.RuntimeInformation]::OSDescription
        arch             = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower()
        lockfileVersion  = '9.0'
        generationSignals = @("lockfileVersion=9.0")
        confidence       = if ($major -eq '9') { 'inferred' } else { 'certain' }
        command          = $command
        timestamp        = (Get-Date -Format o)
        lockfileSha256   = $hash
    }
    $meta | ConvertTo-Json -Depth 6 | Set-Content (Join-Path $dest 'metadata.json') -Encoding utf8NoBOM
}

foreach ($major in 9,10,11) {
    $src = Join-Path $root "fixtures/locks/pnpm/v$major"
    $dest = Join-Path $root "fixtures/locks/generated/pnpm-$major/basic"
    New-Item -ItemType Directory -Force -Path $dest | Out-Null
    if ($Generate) {
        $ver = (Get-Variable "PNPM${major}_VERSION").Value
        $work = Join-Path ([System.IO.Path]::GetTempPath()) "mew-lock-fix-$major"
        if (Test-Path $work) { Remove-Item -Recurse -Force $work }
        New-Item -ItemType Directory -Force -Path $work | Out-Null
        Copy-Item (Join-Path $src 'package.json') $work
        Push-Location $work
        try {
            corepack enable 2>$null | Out-Null
            corepack prepare "pnpm@$ver" --activate | Out-Null
            pnpm install --lockfile-only | Out-Null
        } finally { Pop-Location }
        Copy-Item (Join-Path $work 'package.json') $dest
        Copy-Item (Join-Path $work 'pnpm-lock.yaml') $dest
        $cmd = "corepack prepare pnpm@$ver --activate; pnpm install --lockfile-only"
    } else {
        Copy-Item (Join-Path $src 'package.json') $dest -Force
        Copy-Item (Join-Path $src 'pnpm-lock.yaml') $dest -Force
        $ver = (Get-Variable "PNPM${major}_VERSION").Value
        $cmd = "committed from fixtures/locks/pnpm/v$major (pnpm@$ver reference)"
    }
    Write-Metadata $dest $major $ver $cmd
}

$nubSrc = Join-Path $root 'fixtures/locks/nub/v1-basic'
$nubDest = Join-Path $root 'fixtures/locks/generated/nub-basic'
New-Item -ItemType Directory -Force -Path $nubDest | Out-Null
Copy-Item (Join-Path $nubSrc '*') $nubDest -Force
$nubLock = Join-Path $nubDest 'nub.lock'
$nubHash = (Get-FileHash $nubLock -Algorithm SHA256).Hash.ToLower()
[ordered]@{
    producer        = 'nub'
    producerVersion = 'manual-evidence'
    producerMajor   = 9
    node            = (node -v 2>$null)
    os              = [System.Runtime.InteropServices.RuntimeInformation]::OSDescription
    arch            = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower()
    lockfileVersion = '9.0'
    generationSignals = @('pnpm-v9-shaped', 'nub.lock')
    confidence      = 'manual'
    command         = 'committed from fixtures/locks/nub/v1-basic'
    timestamp       = (Get-Date -Format o)
    lockfileSha256  = $nubHash
} | ConvertTo-Json -Depth 6 | Set-Content (Join-Path $nubDest 'metadata.json') -Encoding utf8NoBOM

Write-Host "ok: fixtures/locks/generated refreshed"
