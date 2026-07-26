# Fuzz smoke: run short fuzz on packages that declare Fuzz* tests.
# Exits 0 when no fuzz targets exist.
$ErrorActionPreference = 'Stop'
$root = Resolve-Path (Join-Path $PSScriptRoot '..')
Set-Location $root

$pkgs = go list ./...
$found = $false
foreach ($pkg in $pkgs) {
  $files = go list -f '{{range .TestGoFiles}}{{.}} {{end}}{{range .XTestGoFiles}}{{.}} {{end}}' $pkg 2>$null
  if (-not $files) { continue }
  $dir = go list -f '{{.Dir}}' $pkg
  $hasFuzz = Get-ChildItem -Path $dir -Filter '*_test.go' -ErrorAction SilentlyContinue |
    Select-String -Pattern '^func Fuzz' -SimpleMatch:$false |
    Select-Object -First 1
  if (-not $hasFuzz) { continue }
  $found = $true
  Write-Host "fuzz-smoke: $pkg"
  go test $pkg -fuzz=. -fuzztime=1s -count=1
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
if (-not $found) {
  Write-Host 'fuzz-smoke: no Fuzz* targets; ok'
}
