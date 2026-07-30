# Backward-compatible wrapper — canonical entrypoint: run_core_cert.py
param(
  [Parameter(Mandatory = $true)]
  [ValidateSet('core-cert-fast', 'core-cert', 'core-cert-security', 'core-cert-crash', 'core-cert-performance')]
  [string]$Target
)

$ErrorActionPreference = 'Stop'
$script = Join-Path $PSScriptRoot 'run_core_cert.py'
python $script $Target
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
