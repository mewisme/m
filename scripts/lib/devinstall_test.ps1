#!/usr/bin/env pwsh
# Logic tests for DevInstall.psm1 (no Pester). LF line endings required.
#Requires -Version 5.1
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$lib = Join-Path $PSScriptRoot 'DevInstall.psm1'
Import-Module $lib -Force

$passed = 0
$failed = 0

function Assert-True {
    param([bool]$Cond, [string]$Name)
    if ($Cond) { $script:passed++; Write-Host "ok: $Name" } else { $script:failed++; Write-Host "FAIL: $Name" }
}

function Assert-Throws {
    param([scriptblock]$Block, [string]$Name)
    try { & $Block | Out-Null; $script:failed++; Write-Host "FAIL: $Name (no throw)" } catch { $script:passed++; Write-Host "ok: $Name" }
}

# 1. OS normalization
Assert-True ((ConvertTo-DevInstallOS 'linux') -eq 'linux') 'goos normalize linux'
Assert-True ((ConvertTo-DevInstallOS 'WINDOWS') -eq 'windows') 'goos normalize windows'
Assert-True ((ConvertTo-DevInstallOS 'freebsd') -eq 'unsupported') 'unsupported goos'

# 2. Arch normalization
Assert-True ((ConvertTo-DevInstallArch 'x86_64') -eq 'amd64') 'goarch normalize x86_64'
Assert-True ((ConvertTo-DevInstallArch 'aarch64') -eq 'arm64') 'goarch normalize aarch64'
Assert-True ((ConvertTo-DevInstallArch 'riscv64') -eq 'unsupported') 'unsupported goarch'

# 3. Windows PATH normalize/dedupe
$entries = @('C:\MewJS\bin', 'c:\mewjs\bin\', 'C:\Other', 'C:\MewJS\bin\')
$seen = @{}
$norm = foreach ($e in $entries) {
    $n = ConvertTo-DevInstallWindowsPathEntry $e
    $k = $n.ToLowerInvariant()
    if (-not $seen.ContainsKey($k)) { $seen[$k] = $true; $n }
}
Assert-True ($norm.Count -eq 2 -and $norm[0] -eq 'C:\MewJS\bin' -and $norm[1] -eq 'C:\Other') 'windows path dedupe'

# 4-5. Managed block insert/replace/remove
$tmpdir = Join-Path ([System.IO.Path]::GetTempPath()) "devinstall-ps-$PID"
New-Item -ItemType Directory -Path $tmpdir -Force | Out-Null
$profileFile = Join-Path $tmpdir 'profile.sh'
Set-Content -Path $profileFile -Value 'echo hello' -NoNewline
$start = $DevInstallMarkerInstallerStart
$end = $DevInstallMarkerInstallerEnd
Set-DevInstallManagedBlock $profileFile $start $end 'export PATH="/a/bin:$PATH"'
$once = Get-Content -Raw $profileFile
Assert-True ($once -match '/a/bin') 'path block insert'
Set-DevInstallManagedBlock $profileFile $start $end 'export PATH="/b/bin:$PATH"'
$twice = Get-Content -Raw $profileFile
Assert-True ((@($twice -split "`n" | Where-Object { $_ -match 'export PATH' }).Count -eq 1) -and ($twice -match '/b/bin')) 'path block replace no duplicate'
Remove-DevInstallManagedBlock $profileFile $start $end
$removed = Get-Content -Raw $profileFile
Assert-True (($removed -notmatch 'mewjs dev installer') -and ($removed -match 'echo hello')) 'path block remove preserves other'

$cStart = $DevInstallMarkerCompletionStart
$cEnd = $DevInstallMarkerCompletionEnd
$compFile = Join-Path $tmpdir 'comp.sh'
Set-Content -Path $compFile -Value 'keep' -NoNewline
Set-DevInstallManagedBlock $compFile $cStart $cEnd 'source /tmp/m'
Assert-True ((Get-Content -Raw $compFile) -match 'source /tmp/m') 'completion block insert'
Remove-DevInstallManagedBlock $compFile $cStart $cEnd
Assert-True ((Get-Content -Raw $compFile).Trim() -eq 'keep') 'completion block remove'

# 6. Shim content exact match
$lines = Get-DevInstallWindowsShimContent 'm'
$shim = ($lines -join "`n") + "`n"
Assert-True (($shim -match '@echo off') -and ($shim -match '%~dp0m\.exe') -and ($shim -match 'exit /b %ERRORLEVEL%')) 'shim content'

# 7-8. Detect canInstall / cross-compile
Get-DevInstallHost
Resolve-DevInstallTarget
Assert-True ($DevInstallCanInstall -eq $true) 'native canInstall true'
$crossOS = if ($DevInstallHostGoOS -eq 'linux') { 'darwin' } else { 'linux' }
Resolve-DevInstallTarget -FlagGoOS $crossOS -FlagGoArch $DevInstallHostGoArch
Assert-True ($DevInstallCanInstall -eq $false) 'cross-compile canInstall false'

# 9. Paths with spaces in managed block
$spaceFile = Join-Path $tmpdir 'space.sh'
Set-Content -Path $spaceFile -Value '# header' -NoNewline
Set-DevInstallManagedBlock $spaceFile $start $end 'export PATH="/path with spaces/bin:$PATH"'
Assert-True ((Get-Content -Raw $spaceFile) -match 'path with spaces') 'paths with spaces'

# 10. Managed-block idempotency
$idFile = Join-Path $tmpdir 'idem.sh'
New-Item -ItemType File -Path $idFile -Force | Out-Null
Set-DevInstallManagedBlock $idFile $start $end 'export PATH="/a/bin:$PATH"'
Set-DevInstallManagedBlock $idFile $start $end 'export PATH="/a/bin:$PATH"'
$markerCount = ([regex]::Matches((Get-Content -Raw $idFile), [regex]::Escape($start))).Count
Assert-True ($markerCount -eq 1) 'managed block idempotent'

# 11. UTF-8 no BOM write
$utfFile = Join-Path $tmpdir 'utf8.ps1'
Write-DevInstallUtf8NoBom $utfFile 'completion-data'
$bytes = [System.IO.File]::ReadAllBytes($utfFile)
Assert-True (($bytes.Length -gt 0) -and ($bytes[0] -ne 0xEF)) 'utf8 no bom'
Assert-True ((Get-Content -Raw $utfFile).Length -gt 0) 'utf8 non-empty'

# 12. Alias-completion wrapper presence
$aliasBlock = Get-DevInstallPowerShellCompletionBlock (Join-Path $tmpdir 'completions')
Assert-True (($aliasBlock -match 'mew') -and ($aliasBlock -match 'mewx')) 'alias completion wrappers'

# 13. Custom install dir paths
$customBase = Get-DevInstallCompletionBaseFromInstallDir 'C:\Custom\MewJS\bin'
Assert-True ($customBase -eq 'C:\Custom\MewJS\completions') 'custom install dir paths'

# 14. Source-build metadata never claims the current Git commit
$savedMewVersion = $env:MEW_VERSION
Remove-Item Env:MEW_VERSION -ErrorAction SilentlyContinue
Resolve-DevInstallMetadata
Assert-True ($DevInstallVersion -eq '0.0.0-dev') 'source build dev version'
Assert-True ($DevInstallCommit -eq '') 'source build commit unset'
Resolve-DevInstallMetadata -VersionOverride '1.2.3-local'
Assert-True ($DevInstallVersion -eq '1.2.3-local') 'source build version override'
Assert-True ($DevInstallCommit -eq '') 'source build override commit unset'
if ($null -ne $savedMewVersion) { $env:MEW_VERSION = $savedMewVersion }

Remove-Item -Recurse -Force $tmpdir -ErrorAction SilentlyContinue

Write-Host ''
Write-Host "results: $passed passed, $failed failed"
if ($failed -gt 0) { exit 1 }
Write-Host 'ok: all DevInstall logic tests passed'
