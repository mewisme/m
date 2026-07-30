# Backward-compatible wrapper — canonical entrypoint: verify_plan_generation.py
$ErrorActionPreference = 'Stop'
$script = Join-Path $PSScriptRoot 'verify_plan_generation.py'
python $script @args
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
