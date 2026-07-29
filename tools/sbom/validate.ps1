# Validate CycloneDX SBOM golden structure (no external schema CLI required).
param(
    [string]$Path = (Join-Path $PSScriptRoot "..\..\fixtures\sbom\medium-graph-cyclonedx-golden.json")
)

$ErrorActionPreference = "Stop"
$raw = Get-Content -Raw -Path $Path
$doc = $raw | ConvertFrom-Json

if ($doc.bomFormat -ne "CycloneDX") { throw "bomFormat must be CycloneDX" }
if ($doc.specVersion -ne "1.5") { throw "specVersion must be 1.5" }
if (-not $doc.metadata.component.'bom-ref') { throw "metadata.component.bom-ref required" }
if (-not $doc.components -or $doc.components.Count -lt 1) { throw "components required" }
if (-not $doc.dependencies -or $doc.dependencies.Count -lt 1) { throw "dependencies required" }

$refs = @{}
foreach ($c in $doc.components) {
    if (-not $c.'bom-ref') { throw "component missing bom-ref: $($c.name)" }
    $refs[$c.'bom-ref'] = $true
}
$refs[$doc.metadata.component.'bom-ref'] = $true

foreach ($dep in $doc.dependencies) {
    if (-not $dep.ref) { throw "dependency missing ref" }
    if (-not $refs.ContainsKey($dep.ref)) { throw "unknown dependency ref: $($dep.ref)" }
    foreach ($to in $dep.dependsOn) {
        if (-not $refs.ContainsKey($to)) { throw "unknown dependsOn ref: $to" }
    }
}

Write-Host "OK: $Path"
