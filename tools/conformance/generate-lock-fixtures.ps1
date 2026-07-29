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
        'alias', 'alias-peer', 'patch', 'binary'
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

$digestTool = Join-Path $PSScriptRoot 'fixturemeta/cmd'
$registryURL = 'https://registry.npmjs.org'
$invocationID = [guid]::NewGuid().ToString()
$isolatedPolicy = 'temp-home-and-pnpm-store-per-family'

function Invoke-FixtureDigest($mode, $path) {
    $env:CGO_ENABLED = '0'
    $out = & go run $digestTool $mode $path 2>&1
    if ($LASTEXITCODE -ne 0) { throw "digest $mode $path failed: $out" }
    return ($out | Out-String).Trim()
}

function Get-WorkspaceManifestHashes($dir) {
    $packages = Join-Path $dir 'packages'
    if (-not (Test-Path $packages)) { return $null }
    $out = @{}
    Get-ChildItem -Path $packages -Recurse -Filter 'package.json' -File | ForEach-Object {
        $rel = $_.FullName.Substring($dir.Length).TrimStart('\', '/')
        $rel = $rel -replace '\\', '/'
        $out[$rel] = Invoke-FixtureDigest 'file' $_.FullName
    }
    if ($out.Count -eq 0) { return $null }
    return $out
}

function Get-PatchFileHashes($dir) {
    $patches = Join-Path $dir 'patches'
    if (-not (Test-Path $patches)) { return $null }
    $out = @{}
    Get-ChildItem -Path $patches -Recurse -Filter '*.patch' -File | ForEach-Object {
        $rel = $_.FullName.Substring($dir.Length).TrimStart('\', '/')
        $rel = $rel -replace '\\', '/'
        $out[$rel] = Invoke-FixtureDigest 'file' $_.FullName
    }
    if ($out.Count -eq 0) { return $null }
    return $out
}

function Write-Metadata($dest, $src, $major, $family, $pnpmVersion, $executablePath, $executableArgs, $invocationId) {
    $lock = Join-Path $dest 'pnpm-lock.yaml'
    if (-not (Test-Path $lock)) { throw "missing lock at $lock" }
    $pkg = Join-Path $dest 'package.json'
    if (-not (Test-Path $pkg)) { throw "missing package.json at $pkg" }
    $lockHash = Invoke-FixtureDigest 'file' $lock
    $pkgHash = Invoke-FixtureDigest 'file' $pkg
    $sourceDigest = Invoke-FixtureDigest 'source-tree' $src
    $wsYaml = Join-Path $dest 'pnpm-workspace.yaml'
    $wsHash = $null
    if (Test-Path $wsYaml) { $wsHash = Invoke-FixtureDigest 'file' $wsYaml }
    $wsManifests = Get-WorkspaceManifestHashes $dest
    $patchHashes = Get-PatchFileHashes $dest
    $cmd = "corepack prepare pnpm@$pnpmVersion --activate; pnpm install --lockfile-only --ignore-scripts (family=$family)"
    $meta = [ordered]@{
        producer                = 'pnpm'
        producerVersion         = $pnpmVersion
        producerMajor           = [int]$major
        family                  = $family
        node                    = (node -v 2>$null)
        os                      = [System.Runtime.InteropServices.RuntimeInformation]::OSDescription
        arch                    = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower()
        executablePath          = $executablePath
        executableArgs          = @($executableArgs)
        registry                = $registryURL
        timestamp               = (Get-Date -Format o)
        classification          = 'generated'
        lockfileVersion         = '9.0'
        lockfileSha256          = $lockHash
        packageJsonSha256       = $pkgHash
        sourceTreeDigest        = $sourceDigest
        invocationId            = $invocationId
        isolatedHomePolicy      = $isolatedPolicy
        command                 = $cmd
        confidence              = 'certain'
        generationSignals       = @("lockfileVersion=9.0", "family=$family", "invocationId=$invocationId")
    }
    if ($wsHash) { $meta.workspaceYamlSha256 = $wsHash }
    if ($wsManifests) { $meta.workspaceManifestSha256 = $wsManifests }
    if ($patchHashes) { $meta.patchFileSha256 = $patchHashes }
    $meta | ConvertTo-Json -Depth 8 | Set-Content (Join-Path $dest 'metadata.json') -Encoding utf8NoBOM
}

function Copy-FamilySource($src, $dest) {
    New-Item -ItemType Directory -Force -Path $dest | Out-Null
    Get-ChildItem -Path $src -Recurse -File | ForEach-Object {
        $rel = $_.FullName.Substring($src.Length).TrimStart('\', '/')
        if ($rel -match '^\.home(/|\\|$)' -or $rel -match '^\.pnpm-store(/|\\|$)') { return }
        $out = Join-Path $dest $rel
        $parent = Split-Path $out -Parent
        if (-not (Test-Path $parent)) { New-Item -ItemType Directory -Force -Path $parent | Out-Null }
        Copy-Item $_.FullName $out -Force
    }
}

function Invoke-PnpmLockOnly($workDir, $pnpmVersion, [ref]$executablePath, [ref]$executableArgs) {
    Push-Location $workDir
    try {
        $env:COREPACK_ENABLE_DOWNLOAD_PROMPT = '0'
        corepack enable 2>$null | Out-Null
        corepack prepare "pnpm@$pnpmVersion" --activate | Out-Null
        $pnpm = Get-Command pnpm -ErrorAction Stop
        $executablePath.Value = $pnpm.Source
        $args = @('install', '--lockfile-only', '--ignore-scripts')
        $executableArgs.Value = $args
        & $pnpm.Source @args 2>&1 | Out-Null
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
            $homeDir = Join-Path $work '.home'
            $storeDir = Join-Path $work '.pnpm-store'
            New-Item -ItemType Directory -Force -Path $homeDir, $storeDir | Out-Null
            $savedHome = $env:HOME
            $savedUserProfile = $env:USERPROFILE
            $savedPnpmHome = $env:PNPM_HOME
            $env:HOME = $homeDir
            $env:USERPROFILE = $homeDir
            $env:PNPM_HOME = Join-Path $homeDir 'pnpm'
            try {
                Copy-FamilySource $src $work
                $exePath = ''
                $exeArgs = @()
                Invoke-PnpmLockOnly $work $ver ([ref]$exePath) ([ref]$exeArgs)
                if (-not (Test-Path (Join-Path $work 'pnpm-lock.yaml'))) {
                    throw "pnpm did not write lockfile for $family major=$major"
                }
                Copy-FamilySource $work $dest
                Write-Metadata $dest $src $major $family $ver $exePath $exeArgs $invocationID
            } finally {
                if ($null -ne $savedHome) { $env:HOME = $savedHome } else { Remove-Item Env:HOME -ErrorAction SilentlyContinue }
                if ($null -ne $savedUserProfile) { $env:USERPROFILE = $savedUserProfile }
                if ($null -ne $savedPnpmHome) { $env:PNPM_HOME = $savedPnpmHome } else { Remove-Item Env:PNPM_HOME -ErrorAction SilentlyContinue }
                if (Test-Path $work) { Remove-Item -Recurse -Force $work }
            }
            Write-Host "ok: pnpm-$major/$family"
        } else {
            if (-not (Test-Path (Join-Path $dest 'pnpm-lock.yaml'))) {
                Write-Warning "skip $dest — no committed lock; run with -Generate"
                continue
            }
            if (-not (Test-Path (Join-Path $dest 'metadata.json'))) {
                Write-Warning "skip $dest — no metadata.json; run with -Generate"
                continue
            }
            Write-Host "ok: pnpm-$major/$family (verify-only)"
        }
    }
}

# Nub fixture families — derived from pnpm-9 generated locks (pnpm-v9-shaped nub.lock)
if ($Generate) {
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
    $sourceLockPath = Join-Path $pnpmDir 'pnpm-lock.yaml'
    $sourceLockHash = Invoke-FixtureDigest 'file' $sourceLockPath
    New-Item -ItemType Directory -Force -Path $nubDest | Out-Null
    Copy-Item (Join-Path $pnpmDir 'package.json') $nubDest -Force
    if (Test-Path (Join-Path $pnpmDir 'pnpm-workspace.yaml')) {
        Copy-Item (Join-Path $pnpmDir 'pnpm-workspace.yaml') $nubDest -Force
    }
    if (Test-Path (Join-Path $pnpmDir 'packages')) {
        $pkgDest = Join-Path $nubDest 'packages'
        if (Test-Path $pkgDest) { Remove-Item -Recurse -Force $pkgDest }
        Copy-Item (Join-Path $pnpmDir 'packages') $pkgDest -Recurse -Force
    }
    $lock = Get-Content $sourceLockPath -Raw
    if ($lock -notmatch 'nubVersion:') {
        $lock = $lock.TrimEnd() + "`nnubVersion: `"1.0.0`"`n"
    }
    Set-Content (Join-Path $nubDest 'nub.lock') $lock -Encoding utf8NoBOM -NoNewline
    $nubHash = Invoke-FixtureDigest 'file' (Join-Path $nubDest 'nub.lock')
    $pkgHash = Invoke-FixtureDigest 'file' (Join-Path $nubDest 'package.json')
    $srcRel = "fixtures/locks/generated/pnpm-9/$family"
    $derivCmd = "derived from $srcRel pnpm-lock.yaml; append nubVersion: 1.0.0"
    $wsHash = $null
    $wsYaml = Join-Path $nubDest 'pnpm-workspace.yaml'
    if (Test-Path $wsYaml) { $wsHash = Invoke-FixtureDigest 'file' $wsYaml }
    $wsManifests = Get-WorkspaceManifestHashes $nubDest
    $meta = [ordered]@{
        producer          = 'nub'
        producerVersion   = 'pnpm-9-shaped'
        producerMajor     = 9
        family            = $name
        node              = (node -v 2>$null)
        os                = [System.Runtime.InteropServices.RuntimeInformation]::OSDescription
        arch              = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower()
        executablePath    = 'derived'
        executableArgs    = @('nub.lock', 'from', 'pnpm-9', $family)
        registry          = $registryURL
        timestamp         = (Get-Date -Format o)
        classification    = 'derived'
        lockfileVersion   = '9.0'
        lockfileSha256    = $nubHash
        packageJsonSha256 = $pkgHash
        sourceFixture     = $srcRel
        sourceLockSha256  = $sourceLockHash
        derivationCommand = $derivCmd
        sourceTreeDigest  = (Invoke-FixtureDigest 'source-tree' (Join-Path $root "fixtures/locks/sources/pnpm/$family"))
        invocationId      = $invocationID
        isolatedHomePolicy = $isolatedPolicy
        command           = $derivCmd
        confidence        = 'manual'
        generationSignals = @('pnpm-v9-shaped', 'nub.lock', "derived-from=pnpm-9/$family", "invocationId=$invocationID")
    }
    if ($wsHash) { $meta.workspaceYamlSha256 = $wsHash }
    if ($wsManifests) { $meta.workspaceManifestSha256 = $wsManifests }
    $meta | ConvertTo-Json -Depth 8 | Set-Content (Join-Path $nubDest 'metadata.json') -Encoding utf8NoBOM
    Write-Host "ok: $name"
}
}

Write-Host "done: fixtures/locks/generated refreshed (invocationId=$invocationID)"
