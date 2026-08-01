# MewJS development installer (Windows). LF line endings required.
#Requires -Version 5.1
[CmdletBinding()]
param(
    [switch]$BuildOnly,
    [switch]$SkipPath,
    [switch]$SkipCompletion,
    [switch]$SkipVerify,
    [string]$InstallDir = '',
    [string]$GoOS = '',
    [string]$GoArch = '',
    [string]$Version = '',
    [switch]$Force
)

$ErrorActionPreference = 'Stop'
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = (Resolve-Path (Join-Path $ScriptDir '..')).Path
Import-Module (Join-Path $ScriptDir 'lib/DevInstall.psm1') -Force
$script:DevInstallRepoRoot = $RepoRoot
Set-Location $script:DevInstallRepoRoot

Write-DevInstallStage detect 'detecting host and target'
$detect = Invoke-DevInstallDetect -GoOS $GoOS -GoArch $GoArch
Write-DevInstallStage detect "host $($detect.HostOS)/$($detect.HostArch)"
Write-DevInstallStage detect "target $($detect.TargetOS)/$($detect.TargetArch)"
if ($detect.Emulated) {
    Write-DevInstallStage detect 'warning: target arch differs from host (emulation/cross)'
}

Write-DevInstallStage check 'validating prerequisites'
Test-DevInstallGo
Test-DevInstallRepo
Resolve-DevInstallMetadata -VersionOverride $Version
$dirtyLabel = if ($script:DevInstallDirty) { '+dirty' } else { '' }
Write-DevInstallStage check "version=$script:DevInstallVersion$dirtyLabel commit=$script:DevInstallShortCommit target=$script:DevInstallTargetGoOS/$script:DevInstallTargetGoArch"

Invoke-DevInstallBuild

if ($BuildOnly) {
    Write-DevInstallStage done 'build-only complete'
    $script:DevInstallInstallDir = ''
    $script:DevInstallCompletionBase = ''
    Write-DevInstallSummary
    exit 0
}

if (-not $script:DevInstallCanInstall) {
    Write-DevInstallError 'cannot install cross-compiled binaries on this host (use -BuildOnly)'
}

if (-not $InstallDir) {
    $script:DevInstallInstallDir = Get-DevInstallDefaultInstallDirWindows
}
else {
    $script:DevInstallInstallDir = $InstallDir
}
$script:DevInstallCompletionBase = Get-DevInstallCompletionBaseFromInstallDir $script:DevInstallInstallDir

Write-DevInstallStage install "installing to $script:DevInstallInstallDir"
Install-DevInstallWindowsFiles -InstallDir $script:DevInstallInstallDir -Force $Force.IsPresent

if (-not $SkipPath) {
    Write-DevInstallStage path 'updating user PATH'
    Add-DevInstallWindowsPath $script:DevInstallInstallDir
}
else {
    Write-DevInstallStage path 'skipped (-SkipPath)'
}

if (-not $SkipCompletion) {
    Write-DevInstallStage completion "generating completions in $script:DevInstallCompletionBase"
    Invoke-DevInstallGenerateCompletionsWindows -InstallDir $script:DevInstallInstallDir -Base $script:DevInstallCompletionBase
    Set-DevInstallWindowsCompletionProfile $script:DevInstallCompletionBase
}
else {
    Write-DevInstallStage completion 'skipped (-SkipCompletion)'
    $script:DevInstallCompletionBase = ''
}

if (-not $SkipVerify) {
    Write-DevInstallStage verify 'running verification'
    Test-DevInstallVerifyWindows -InstallDir $script:DevInstallInstallDir -Base $script:DevInstallCompletionBase
}
else {
    Write-DevInstallStage verify 'skipped (-SkipVerify)'
}

Write-DevInstallStage done 'development install complete'
Write-DevInstallSummary
