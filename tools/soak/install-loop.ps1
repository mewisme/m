# Backward-compatible wrapper — canonical entrypoint: install_loop.py
param(
  [int]$Count = 100,
  [ValidateSet('cold', 'warm')]
  [string]$Mode = 'cold',
  [string]$Project = ''
)

$ErrorActionPreference = 'Stop'
$script = Join-Path $PSScriptRoot 'install_loop.py'
$args = @('--count', $Count, '--mode', $Mode)
if ($Project) { $args += @('--project', $Project) }
python $script @args
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
