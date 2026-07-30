# Backward-compatible wrapper — canonical entrypoint: check_regression.py
param(
  [ValidateSet('cold', 'warm')]
  [string]$Mode = 'warm'
)

$ErrorActionPreference = 'Stop'
$script = Join-Path $PSScriptRoot 'check_regression.py'
python $script --mode $Mode
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
