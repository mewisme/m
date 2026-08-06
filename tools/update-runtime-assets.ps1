# Thin wrapper — canonical implementation: update-runtime-assets.py
$ErrorActionPreference = 'Stop'
$script = Join-Path $PSScriptRoot 'update-runtime-assets.py'
python $script @args
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
