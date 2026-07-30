# MewJS development uninstaller (Windows). LF line endings required.
#Requires -Version 5.1
[CmdletBinding()]
param(
    [string]$InstallDir = '',
    [switch]$KeepPath,
    [switch]$KeepCompletion
)

$ErrorActionPreference = 'Stop'
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = (Resolve-Path (Join-Path $ScriptDir '..')).Path
Import-Module (Join-Path $ScriptDir 'lib/DevInstall.psm1') -Force
$script:DevInstallRepoRoot = $RepoRoot

if (-not $InstallDir) {
    $script:DevInstallInstallDir = Get-DevInstallDefaultInstallDirWindows
}
else {
    $script:DevInstallInstallDir = $InstallDir
}
$script:DevInstallCompletionBase = Get-DevInstallCompletionBaseFromInstallDir $script:DevInstallInstallDir

Write-DevInstallStage uninstall "removing installer-owned files from $script:DevInstallInstallDir"

if (-not $KeepCompletion) {
    Remove-DevInstallOwnedWindowsFiles -InstallDir $script:DevInstallInstallDir -Base $script:DevInstallCompletionBase
    Remove-DevInstallWindowsCompletionProfile
}
else {
    Remove-DevInstallOwnedWindowsFiles -InstallDir $script:DevInstallInstallDir -Base $script:DevInstallCompletionBase
}

if (-not $KeepPath) {
    Remove-DevInstallWindowsPath $script:DevInstallInstallDir
}
else {
    Write-DevInstallStage path 'kept (-KeepPath)'
}

Write-DevInstallStage done 'development uninstall complete'
