# Install pinned golangci-lint and govulncheck into GOPATH/bin (or GOBIN).
$ErrorActionPreference = 'Stop'
$root = Resolve-Path (Join-Path $PSScriptRoot '..')
$envFile = Join-Path $root 'tools\versions.env'
Get-Content $envFile | ForEach-Object {
  if ($_ -match '^\s*#' -or $_ -match '^\s*$') { return }
  $k, $v = $_ -split '=', 2
  Set-Item -Path "Env:$k" -Value $v
}
Write-Host "Installing golangci-lint $($env:GOLANGCI_LINT_VERSION)"
go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$($env:GOLANGCI_LINT_VERSION)"
Write-Host "Installing govulncheck $($env:GOVULNCHECK_VERSION)"
go install "golang.org/x/vuln/cmd/govulncheck@$($env:GOVULNCHECK_VERSION)"
Write-Host 'ok: tools installed'
