# Backward-compatible wrapper — canonical entrypoint: check_correctness.py
param(
  [ValidateSet('cold', 'warm')]
  [string]$Mode = 'warm'
)

$ErrorActionPreference = 'Stop'
$script = Join-Path $PSScriptRoot 'check_correctness.py'
python $script --mode $Mode
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
