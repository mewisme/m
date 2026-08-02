# Shared MewJS development-install helpers for PowerShell.
# LF line endings required.

Set-StrictMode -Version Latest

$DevInstallMarkerInstallerStart = '# >>> mewjs dev installer >>>'
$DevInstallMarkerInstallerEnd = '# <<< mewjs dev installer <<<'
$DevInstallMarkerCompletionStart = '# >>> mewjs dev completion >>>'
$DevInstallMarkerCompletionEnd = '# <<< mewjs dev completion <<<'

$DevInstallRepoRoot = ''
$DevInstallHostGoOS = ''
$DevInstallHostGoArch = ''
$DevInstallTargetGoOS = ''
$DevInstallTargetGoArch = ''
$DevInstallVersion = ''
$DevInstallCommit = ''
$DevInstallBuildDate = ''
$DevInstallInstallDir = ''
$DevInstallCompletionBase = ''
$DevInstallCanInstall = $false
$DevInstallGoMinVersion = '1.26.5'

function Write-DevInstallStage {
    param([string]$Stage, [string]$Message)
    Write-Host "[$Stage] $Message"
}

function Write-DevInstallError {
    param([string]$Message)
    Write-Error "error: $Message" -ErrorAction Continue
    throw $Message
}

function ConvertTo-DevInstallOS {
    param([string]$Raw)
    switch -Regex ($Raw.ToLowerInvariant()) {
        '^(windows|mingw.*|msys.*|cygwin.*)$' { return 'windows' }
        '^linux$' { return 'linux' }
        '^darwin$' { return 'darwin' }
        default { return 'unsupported' }
    }
}

function ConvertTo-DevInstallArch {
    param([string]$Raw)
    switch ($Raw.ToLowerInvariant()) {
        'x86_64' { return 'amd64' }
        'amd64' { return 'amd64' }
        'aarch64' { return 'arm64' }
        'arm64' { return 'arm64' }
        default { return 'unsupported' }
    }
}

function Test-DevInstallMatrix {
    param([string]$OS, [string]$Arch)
    $osOk = @('windows', 'linux', 'darwin') -contains $OS
    $archOk = @('amd64', 'arm64') -contains $Arch
    return $osOk -and $archOk
}

function Get-DevInstallHost {
    $uname = if ($IsWindows) { 'windows' } elseif ($IsMacOS) { 'darwin' } else { 'linux' }
    $arch = ConvertTo-DevInstallArch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant())
    if ($arch -eq 'unsupported') {
        $arch = ConvertTo-DevInstallArch $env:PROCESSOR_ARCHITECTURE
    }
    $script:DevInstallHostGoOS = ConvertTo-DevInstallOS $uname
    $script:DevInstallHostGoArch = $arch
    if ($script:DevInstallHostGoOS -eq 'unsupported' -or $script:DevInstallHostGoArch -eq 'unsupported') {
        Write-DevInstallError "unsupported host OS/arch: $uname/$arch"
    }
}

function Resolve-DevInstallTarget {
    param(
        [string]$FlagGoOS = '',
        [string]$FlagGoArch = ''
    )
    $goos = if ($FlagGoOS) { $FlagGoOS } elseif ($env:GOOS) { $env:GOOS } else { $script:DevInstallHostGoOS }
    $goarch = if ($FlagGoArch) { $FlagGoArch } elseif ($env:GOARCH) { $env:GOARCH } else { $script:DevInstallHostGoArch }
    $script:DevInstallTargetGoOS = ConvertTo-DevInstallOS $goos
    $script:DevInstallTargetGoArch = ConvertTo-DevInstallArch $goarch
    if ($script:DevInstallTargetGoOS -eq 'unsupported' -or $script:DevInstallTargetGoArch -eq 'unsupported') {
        Write-DevInstallError "unsupported target GOOS/GOARCH: $goos/$goarch"
    }
    if (-not (Test-DevInstallMatrix $script:DevInstallTargetGoOS $script:DevInstallTargetGoArch)) {
        Write-DevInstallError "unsupported target matrix: $script:DevInstallTargetGoOS/$script:DevInstallTargetGoArch"
    }
    $script:DevInstallCanInstall = (
        $script:DevInstallTargetGoOS -eq $script:DevInstallHostGoOS -and
        $script:DevInstallTargetGoArch -eq $script:DevInstallHostGoArch
    )
}

function Test-DevInstallVersionGe {
    param([string]$Have, [string]$Want)
    $h = $Have.Split('.')
    $w = $Want.Split('.')
    for ($i = 0; $i -lt 3; $i++) {
        $hv = if ($i -lt $h.Length) { [int]$h[$i] } else { 0 }
        $wv = if ($i -lt $w.Length) { [int]$w[$i] } else { 0 }
        if ($hv -gt $wv) { return $true }
        if ($hv -lt $wv) { return $false }
    }
    return $true
}

function Test-DevInstallGo {
    $go = Get-Command go -ErrorAction SilentlyContinue
    if (-not $go) {
        Write-DevInstallError 'go not found on PATH'
    }
    $raw = (go version)
    $ver = ($raw -replace '^go version go', '').Split(' ')[0]
    if (-not (Test-DevInstallVersionGe $ver $DevInstallGoMinVersion)) {
        Write-DevInstallError "go version $ver < required $DevInstallGoMinVersion"
    }
}

function Resolve-DevInstallMetadata {
    param([string]$VersionOverride = '')
    if ($VersionOverride) {
        $script:DevInstallVersion = $VersionOverride
    }
    elseif ($env:MEW_VERSION) {
        $script:DevInstallVersion = $env:MEW_VERSION
    }
    else {
        $script:DevInstallVersion = 'dev'
    }

    # Resolve git commit from the working tree.
    $script:DevInstallCommit = ''
    $script:DevInstallShortCommit = ''
    $script:DevInstallDirty = $false
    $script:DevInstallBuildDate = [DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')

    Push-Location $script:DevInstallRepoRoot
    try {
        $gitCommit = (git rev-parse HEAD 2>$null)
        if ($LASTEXITCODE -eq 0 -and $gitCommit) {
            $script:DevInstallCommit = $gitCommit.Trim()
            $script:DevInstallShortCommit = $script:DevInstallCommit.Substring(0, [Math]::Min(7, $script:DevInstallCommit.Length))
            # Detect dirty state: any uncommitted changes in tracked files.
            $dirtyOut = (git status --porcelain 2>$null)
            if ($dirtyOut) {
                $script:DevInstallDirty = $true
            }
        }
    }
    finally {
        Pop-Location
    }
}

function Test-DevInstallRepo {
    $m = Join-Path $script:DevInstallRepoRoot 'cmd/m'
    $mx = Join-Path $script:DevInstallRepoRoot 'cmd/mx'
    if (-not (Test-Path $m) -or -not (Test-Path $mx)) {
        Write-DevInstallError 'cmd/m and cmd/mx must exist under repo root'
    }
    $binDir = Join-Path $script:DevInstallRepoRoot 'bin'
    if (-not (Test-Path $binDir)) {
        New-Item -ItemType Directory -Path $binDir -Force | Out-Null
    }
    if (-not (Test-Path $binDir -PathType Container)) {
        Write-DevInstallError "cannot create bin directory: $binDir"
    }
}

function Get-DevInstallBinExt {
    if ($script:DevInstallTargetGoOS -eq 'windows') { return '.exe' }
    return ''
}

function Get-DevInstallLdflags {
    $dirtyStr = if ($script:DevInstallDirty) { 'true' } else { 'false' }
    return "-X main.version=$script:DevInstallVersion -X main.commit=$script:DevInstallCommit -X main.shortCommit=$script:DevInstallShortCommit -X main.dirty=$dirtyStr -X main.buildDate=$script:DevInstallBuildDate -X main.targetOS=$script:DevInstallTargetGoOS -X main.targetArch=$script:DevInstallTargetGoArch"
}

function Invoke-DevInstallBuild {
    $ext = Get-DevInstallBinExt
    $env:CGO_ENABLED = '0'
    $env:GOOS = $script:DevInstallTargetGoOS
    $env:GOARCH = $script:DevInstallTargetGoArch
    $ldflags = Get-DevInstallLdflags
    $binDir = Join-Path $script:DevInstallRepoRoot 'bin'
    Write-DevInstallStage build "CGO_ENABLED=0 go build -> bin/m$ext, bin/mx$ext"
    & go build -ldflags $ldflags -o (Join-Path $binDir "m$ext") ./cmd/m
    if ($LASTEXITCODE -ne 0) { throw 'go build m failed' }
    & go build -ldflags $ldflags -o (Join-Path $binDir "mx$ext") ./cmd/mx
    if ($LASTEXITCODE -ne 0) { throw 'go build mx failed' }

    # Create mew.exe and mewx.exe as byte-identical copies of m.exe and mx.exe.
    # os.Args[0] determines the logical identity so copies preserve full identity.
    $mBin = Join-Path $binDir "m$ext"
    $mxBin = Join-Path $binDir "mx$ext"
    Copy-Item -Force $mBin (Join-Path $binDir "mew$ext")
    Copy-Item -Force $mxBin (Join-Path $binDir "mewx$ext")
    Write-DevInstallStage build "created mew$ext (copy of m$ext), mewx$ext (copy of mx$ext)"
}

function Get-DevInstallDefaultInstallDirWindows {
    $base = Join-Path $env:LOCALAPPDATA 'MewJS/bin'
    return $base
}

function Get-DevInstallProductRootFromInstallDir {
    param([string]$InstallDir)
    return (Split-Path -Parent $InstallDir)
}

function Get-DevInstallCompletionBaseFromInstallDir {
    param([string]$InstallDir)
    $root = Get-DevInstallProductRootFromInstallDir $InstallDir
    return Join-Path $root 'completions'
}

function Get-DevInstallWindowsShimContent {
    param([string]$Name)
    switch ($Name) {
        'm' { return @('@echo off', '"%~dp0m.exe" %*', 'exit /b %ERRORLEVEL%') }
        'mx' { return @('@echo off', '"%~dp0mx.exe" %*', 'exit /b %ERRORLEVEL%') }
        'mew' { return @('@echo off', '"%~dp0m.exe" %*', 'exit /b %ERRORLEVEL%') }
        'mewx' { return @('@echo off', '"%~dp0mx.exe" %*', 'exit /b %ERRORLEVEL%') }
        default { throw "unknown shim: $Name" }
    }
}

function Write-DevInstallUtf8NoBom {
    param([string]$Path, [string]$Content)
    $dir = Split-Path -Parent $Path
    if ($dir -and -not (Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }
    $tmp = "$Path.tmp.$PID"
    [System.IO.File]::WriteAllText($tmp, $Content, [System.Text.UTF8Encoding]::new($false))
    Move-Item -Force $tmp $Path
}

function Invoke-DevInstallAtomicCopy {
    param([string]$Source, [string]$Dest)
    $dir = Split-Path -Parent $Dest
    if ($dir -and -not (Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }
    $tmp = "$Dest.tmp.$PID"
    Copy-Item -Force $Source $tmp
    Move-Item -Force $tmp $Dest
}

function Set-DevInstallManagedBlock {
    param(
        [string]$FilePath,
        [string]$StartMarker,
        [string]$EndMarker,
        [string]$BlockContent
    )
    $dir = Split-Path -Parent $FilePath
    if ($dir -and -not (Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }
    if (-not (Test-Path $FilePath)) {
        New-Item -ItemType File -Path $FilePath -Force | Out-Null
    }
    $content = Get-Content -Raw -Path $FilePath
    if (-not $content) { $content = '' }
    $newBlock = "$StartMarker`n$BlockContent`n$EndMarker"
    if ($content.Contains($StartMarker)) {
        $before = $content.Substring(0, $content.IndexOf($StartMarker))
        $afterStart = $content.IndexOf($EndMarker)
        if ($afterStart -ge 0) {
            $after = $content.Substring($afterStart + $EndMarker.Length)
            if ($after.StartsWith("`n")) { $after = $after.Substring(1) }
            if ($after.StartsWith("`r`n")) { $after = $after.Substring(2) }
        }
        else { $after = '' }
        $joined = if ($after) { "$before$newBlock`n$after" } else { "$before$newBlock" }
        Write-DevInstallUtf8NoBom $FilePath $joined
    }
    else {
        $suffix = if ($content -and -not $content.EndsWith("`n")) { "`n" } else { '' }
        Write-DevInstallUtf8NoBom $FilePath "$content$suffix$newBlock`n"
    }
}

function Remove-DevInstallManagedBlock {
    param(
        [string]$FilePath,
        [string]$StartMarker,
        [string]$EndMarker
    )
    if (-not (Test-Path $FilePath)) { return }
    $content = Get-Content -Raw -Path $FilePath
    if (-not $content.Contains($StartMarker)) { return }
    $before = $content.Substring(0, $content.IndexOf($StartMarker))
    $afterStart = $content.IndexOf($EndMarker)
    if ($afterStart -ge 0) {
        $after = $content.Substring($afterStart + $EndMarker.Length)
        if ($after.StartsWith("`n")) { $after = $after.Substring(1) }
        if ($after.StartsWith("`r`n")) { $after = $after.Substring(2) }
    }
    else { $after = '' }
    $joined = if ($after) { "$($before.TrimEnd())`n$after" } else { $before.TrimEnd() }
    Write-DevInstallUtf8NoBom $FilePath $joined
}

function ConvertTo-DevInstallWindowsPathEntry {
    param([string]$PathEntry)
    $p = $PathEntry.TrimEnd('\')
    return $p
}

function Get-DevInstallUserPathEntries {
    $raw = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (-not $raw) { return @() }
    return $raw.Split(';') | Where-Object { $_ }
}

function Set-DevInstallUserPathEntries {
    param([string[]]$Entries)
    $joined = ($Entries -join ';')
    [Environment]::SetEnvironmentVariable('Path', $joined, 'User')
}

function Add-DevInstallWindowsPath {
    param([string]$InstallDir)
    $norm = ConvertTo-DevInstallWindowsPathEntry $InstallDir
    $entries = Get-DevInstallUserPathEntries
    $filtered = @()
    foreach ($e in $entries) {
        if ((ConvertTo-DevInstallWindowsPathEntry $e).ToLowerInvariant() -ne $norm.ToLowerInvariant()) {
            $filtered += $e
        }
    }
    $newEntries = @($norm) + $filtered
    Set-DevInstallUserPathEntries $newEntries
    $proc = $env:PATH -split ';'
    $procFiltered = @()
    foreach ($e in $proc) {
        if ((ConvertTo-DevInstallWindowsPathEntry $e).ToLowerInvariant() -ne $norm.ToLowerInvariant()) {
            $procFiltered += $e
        }
    }
    $env:PATH = ($norm + ';' + ($procFiltered -join ';')).TrimEnd(';')
}

function Remove-DevInstallWindowsPath {
    param([string]$InstallDir)
    $norm = ConvertTo-DevInstallWindowsPathEntry $InstallDir
    $entries = Get-DevInstallUserPathEntries
    $filtered = @()
    foreach ($e in $entries) {
        if ((ConvertTo-DevInstallWindowsPathEntry $e).ToLowerInvariant() -ne $norm.ToLowerInvariant()) {
            $filtered += $e
        }
    }
    Set-DevInstallUserPathEntries $filtered
}

function Install-DevInstallWindowsFiles {
    param(
        [string]$InstallDir,
        [bool]$Force = $false
    )
    $ext = Get-DevInstallBinExt
    $binDir = Join-Path $script:DevInstallRepoRoot 'bin'
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    # Install all four executable identities as byte-identical copies.
    # os.Args[0] determines the logical identity for help, headers, and hints.
    foreach ($name in @('m', 'mx', 'mew', 'mewx')) {
        Invoke-DevInstallAtomicCopy (Join-Path $binDir "$name$ext") (Join-Path $InstallDir "$name$ext")
    }

    # .cmd shims for terminal convenience (m, mx, mew, mewx).
    foreach ($shim in @('m', 'mx', 'mew', 'mewx')) {
        $shimPath = Join-Path $InstallDir "$shim.cmd"
        $lines = Get-DevInstallWindowsShimContent $shim
        Write-DevInstallUtf8NoBom $shimPath (($lines -join "`n") + "`n")
    }
}

function Get-DevInstallPowerShellCompletionBlock {
    param([string]$Base)
    $mScript = Join-Path $Base 'powershell/m.ps1'
    $mxScript = Join-Path $Base 'powershell/mx.ps1'
    return @(
        ". '$mScript'"
        ". '$mxScript'"
        'Register-ArgumentCompleter -CommandName mew -ScriptBlock { param($c,$p,$s) __mCompleter @c $p $s } -ErrorAction SilentlyContinue'
        'Register-ArgumentCompleter -CommandName mewx -ScriptBlock { param($c,$p,$s) __mxCompleter @c $p $s } -ErrorAction SilentlyContinue'
    ) -join "`n"
}

function Set-DevInstallWindowsCompletionProfile {
    param([string]$Base)
    $profile = $PROFILE.CurrentUserAllHosts
    if (-not $profile) { $profile = $PROFILE.CurrentUserCurrentHost }
    $block = Get-DevInstallPowerShellCompletionBlock $Base
    Set-DevInstallManagedBlock $profile $DevInstallMarkerCompletionStart $DevInstallMarkerCompletionEnd $block
    try {
        . (Join-Path $Base 'powershell/m.ps1')
        . (Join-Path $Base 'powershell/mx.ps1')
    }
    catch {
        Write-DevInstallStage completion 'warning: could not load completion in current session (profile may load later)'
    }
}

function Remove-DevInstallWindowsCompletionProfile {
    $profile = $PROFILE.CurrentUserAllHosts
    if (-not $profile) { $profile = $PROFILE.CurrentUserCurrentHost }
    Remove-DevInstallManagedBlock $profile $DevInstallMarkerCompletionStart $DevInstallMarkerCompletionEnd
}

function Invoke-DevInstallGenerateCompletionsWindows {
    param(
        [string]$InstallDir,
        [string]$Base
    )
    $ext = Get-DevInstallBinExt
    $mBin = Join-Path $InstallDir "m$ext"
    $mxBin = Join-Path $InstallDir "mx$ext"
    $psDir = Join-Path $Base 'powershell'
    New-Item -ItemType Directory -Path $psDir -Force | Out-Null

    $mOut = & $mBin completion powershell
    if (-not $mOut -or $mOut.Count -eq 0) {
        Write-DevInstallError 'empty powershell completion from m'
    }
    Write-DevInstallUtf8NoBom (Join-Path $psDir 'm.ps1') (($mOut -join "`n") + "`n")

    $mxOut = & $mxBin completion powershell
    if (-not $mxOut -or $mxOut.Count -eq 0) {
        Write-DevInstallError 'empty powershell completion from mx'
    }
    Write-DevInstallUtf8NoBom (Join-Path $psDir 'mx.ps1') (($mxOut -join "`n") + "`n")
}

function Test-DevInstallVerifyWindows {
    param(
        [string]$InstallDir,
        [string]$Base
    )
    foreach ($name in @('m', 'mx', 'mew', 'mewx')) {
        $cmd = Join-Path $InstallDir "$name.cmd"
        & $cmd version
        if ($LASTEXITCODE -ne 0) {
            Write-DevInstallError "verify failed: $cmd version"
        }
    }
    if ($Base) {
        $files = @(
            (Join-Path $Base 'powershell/m.ps1'),
            (Join-Path $Base 'powershell/mx.ps1')
        )
        foreach ($f in $files) {
            if (-not (Test-Path $f) -or (Get-Item $f).Length -eq 0) {
                Write-DevInstallError "verify failed: missing completion $f"
            }
        }
    }
    $mCmd = Get-Command m -ErrorAction SilentlyContinue
    if (-not $mCmd) {
        Write-DevInstallStage verify 'warning: m not on current shell PATH (restart terminal)'
    }
}

function Remove-DevInstallOwnedWindowsFiles {
    param(
        [string]$InstallDir,
        [string]$Base
    )
    $ext = Get-DevInstallBinExt
    Remove-Item -Force -ErrorAction SilentlyContinue `
        (Join-Path $InstallDir "m$ext"),
        (Join-Path $InstallDir "mx$ext"),
        (Join-Path $InstallDir 'm.cmd'),
        (Join-Path $InstallDir 'mx.cmd'),
        (Join-Path $InstallDir 'mew.cmd'),
        (Join-Path $InstallDir 'mewx.cmd')
    $psDir = Join-Path $Base 'powershell'
    Remove-Item -Force -ErrorAction SilentlyContinue `
        (Join-Path $psDir 'm.ps1'),
        (Join-Path $psDir 'mx.ps1')
    if (Test-Path $psDir) {
        $items = Get-ChildItem $psDir -ErrorAction SilentlyContinue
        if (-not $items) { Remove-Item -Force -ErrorAction SilentlyContinue $psDir }
    }
    if (Test-Path $Base) {
        $items = Get-ChildItem $Base -ErrorAction SilentlyContinue
        if (-not $items) { Remove-Item -Force -ErrorAction SilentlyContinue $Base }
    }
    if (Test-Path $InstallDir) {
        $items = Get-ChildItem $InstallDir -ErrorAction SilentlyContinue
        if (-not $items) { Remove-Item -Force -ErrorAction SilentlyContinue $InstallDir }
    }
}

function Write-DevInstallSummary {
    $dirtyStr = if ($script:DevInstallDirty) { 'true' } else { 'false' }
    @"
MewJS development install summary
  repo:        $script:DevInstallRepoRoot
  source:      working tree
  target:      $script:DevInstallTargetGoOS/$script:DevInstallTargetGoArch
  version:     $script:DevInstallVersion
  commit:      $script:DevInstallCommit
  short:       $script:DevInstallShortCommit
  dirty:       $dirtyStr
  build date:  $script:DevInstallBuildDate
  install dir: $(if ($script:DevInstallInstallDir) { $script:DevInstallInstallDir } else { '<build-only>' })
  completion:  $(if ($script:DevInstallCompletionBase) { $script:DevInstallCompletionBase } else { '<skipped>' })
"@ | Write-Host
}

function Convert-DevInstallGoOS {
    param([string]$Raw)
    $n = ConvertTo-DevInstallOS $Raw
    if ($n -eq 'unsupported') { throw "unsupported GOOS: $Raw" }
    return $n
}

function Convert-DevInstallGoArch {
    param([string]$Raw)
    $n = ConvertTo-DevInstallArch $Raw
    if ($n -eq 'unsupported') { throw "unsupported GOARCH: $Raw" }
    return $n
}

function Get-DevInstallUniqueWindowsPaths {
    param([string[]]$Entries)
    $seen = @{}
    $out = @()
    foreach ($e in $Entries) {
        if (-not $e) { continue }
        $norm = ConvertTo-DevInstallWindowsPathEntry $e
        $key = $norm.ToLowerInvariant()
        if (-not $seen.ContainsKey($key)) {
            $seen[$key] = $true
            $out += $norm
        }
    }
    return $out
}

function Update-DevInstallManagedBlockContent {
    param(
        [string]$Content,
        [string]$StartMarker,
        [string]$EndMarker,
        [string]$Body
    )
    $block = "$StartMarker`n$Body`n$EndMarker"
    if ($Content.Contains($StartMarker)) {
        $before = $Content.Substring(0, $Content.IndexOf($StartMarker))
        $afterStart = $Content.IndexOf($EndMarker)
        if ($afterStart -ge 0) {
            $after = $Content.Substring($afterStart + $EndMarker.Length)
            if ($after.StartsWith("`n")) { $after = $after.Substring(1) }
            if ($after.StartsWith("`r`n")) { $after = $after.Substring(2) }
        }
        else { $after = '' }
        if ($after) { return "$before$block`n$after" }
        return "$before$block"
    }
    $suffix = if ($Content -and -not $Content.EndsWith("`n")) { "`n" } else { '' }
    return "$Content$suffix$block`n"
}

function Remove-DevInstallManagedBlockContent {
    param(
        [string]$Content,
        [string]$StartMarker,
        [string]$EndMarker
    )
    if (-not $Content.Contains($StartMarker)) { return $Content }
    $before = $Content.Substring(0, $Content.IndexOf($StartMarker))
    $afterStart = $Content.IndexOf($EndMarker)
    if ($afterStart -ge 0) {
        $after = $Content.Substring($afterStart + $EndMarker.Length)
        if ($after.StartsWith("`n")) { $after = $after.Substring(1) }
        if ($after.StartsWith("`r`n")) { $after = $after.Substring(2) }
    }
    else { $after = '' }
    if ($after) { return "$($before.TrimEnd())`n$after" }
    return $before.TrimEnd()
}

function Get-DevInstallShimContent {
    param([string]$TargetExe)
    $name = $TargetExe -replace '\.exe$', ''
    $lines = Get-DevInstallWindowsShimContent $name
    return ($lines -join "`r`n") + "`r`n"
}

function Get-DevInstallHostOS {
    Get-DevInstallHost
    return $script:DevInstallHostGoOS
}

function Get-DevInstallHostArch {
    Get-DevInstallHost
    return $script:DevInstallHostGoArch
}

function Invoke-DevInstallDetect {
    param(
        [string]$GoOS = '',
        [string]$GoArch = ''
    )
    Get-DevInstallHost
    Resolve-DevInstallTarget -FlagGoOS $GoOS -FlagGoArch $GoArch
    return [pscustomobject]@{
        HostOS = $script:DevInstallHostGoOS
        HostArch = $script:DevInstallHostGoArch
        TargetOS = $script:DevInstallTargetGoOS
        TargetArch = $script:DevInstallTargetGoArch
        Emulated = ($script:DevInstallHostGoArch -ne $script:DevInstallTargetGoArch)
        CanInstall = $script:DevInstallCanInstall
    }
}

function Get-DevInstallDefaultPaths {
    param([string]$InstallDir = '')
    if ($InstallDir) {
        $root = Split-Path -Parent $InstallDir
        return [pscustomobject]@{
            InstallDir = $InstallDir
            CompletionRoot = Join-Path $root 'completions'
        }
    }
    $base = Get-DevInstallDefaultInstallDirWindows
    $root = Split-Path -Parent $base
    return [pscustomobject]@{
        InstallDir = $base
        CompletionRoot = Join-Path $root 'completions'
    }
}

Export-ModuleMember -Function @(
    'Write-DevInstallStage',
    'Write-DevInstallError',
    'ConvertTo-DevInstallOS',
    'ConvertTo-DevInstallArch',
    'Test-DevInstallMatrix',
    'Get-DevInstallHost',
    'Resolve-DevInstallTarget',
    'Test-DevInstallGo',
    'Resolve-DevInstallMetadata',
    'Test-DevInstallRepo',
    'Get-DevInstallBinExt',
    'Invoke-DevInstallBuild',
    'Get-DevInstallDefaultInstallDirWindows',
    'Get-DevInstallProductRootFromInstallDir',
    'Get-DevInstallCompletionBaseFromInstallDir',
    'Get-DevInstallWindowsShimContent',
    'Get-DevInstallPowerShellCompletionBlock',
    'Write-DevInstallUtf8NoBom',
    'Set-DevInstallManagedBlock',
    'Remove-DevInstallManagedBlock',
    'Update-DevInstallManagedBlockContent',
    'Remove-DevInstallManagedBlockContent',
    'Convert-DevInstallGoOS',
    'Convert-DevInstallGoArch',
    'Get-DevInstallUniqueWindowsPaths',
    'Get-DevInstallShimContent',
    'Get-DevInstallHostOS',
    'Get-DevInstallHostArch',
    'Invoke-DevInstallDetect',
    'Get-DevInstallDefaultPaths',
    'Add-DevInstallWindowsPath',
    'Remove-DevInstallWindowsPath',
    'ConvertTo-DevInstallWindowsPathEntry',
    'Get-DevInstallUserPathEntries',
    'Set-DevInstallUserPathEntries',
    'Install-DevInstallWindowsFiles',
    'Set-DevInstallWindowsCompletionProfile',
    'Remove-DevInstallWindowsCompletionProfile',
    'Invoke-DevInstallGenerateCompletionsWindows',
    'Test-DevInstallVerifyWindows',
    'Remove-DevInstallOwnedWindowsFiles',
    'Write-DevInstallSummary'
)

Export-ModuleMember -Variable @(
    'DevInstallMarkerInstallerStart',
    'DevInstallMarkerInstallerEnd',
    'DevInstallMarkerCompletionStart',
    'DevInstallMarkerCompletionEnd',
    'DevInstallRepoRoot',
    'DevInstallHostGoOS',
    'DevInstallHostGoArch',
    'DevInstallTargetGoOS',
    'DevInstallTargetGoArch',
    'DevInstallVersion',
    'DevInstallCommit',
    'DevInstallShortCommit',
    'DevInstallDirty',
    'DevInstallBuildDate',
    'DevInstallInstallDir',
    'DevInstallCompletionBase',
    'DevInstallCanInstall',
    'DevInstallGoMinVersion'
)
