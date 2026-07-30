# Backward-compatible wrapper — canonical entrypoint: fuzz_smoke.py
$ErrorActionPreference = 'Stop'
$script = Join-Path $PSScriptRoot 'fuzz_smoke.py'
python $script @args
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
