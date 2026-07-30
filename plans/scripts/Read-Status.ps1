# Read and validate plans/status.json. Dot-source from enrich-and-generate.ps1.

function Get-PlanStatus {
  param(
    [string]$PlansRoot,
    [string]$StatusPath
  )

  if (-not $StatusPath) {
    $StatusPath = Join-Path $PlansRoot 'status.json'
  }
  if (-not (Test-Path $StatusPath)) {
    throw "missing status file: $StatusPath"
  }

  $raw = Get-Content $StatusPath -Raw -Encoding utf8
  $status = $raw | ConvertFrom-Json

  if ($status.schemaVersion -ne 1) {
    throw "unsupported status schemaVersion: $($status.schemaVersion)"
  }
  if (-not $status.currentMvp) {
    throw 'status.currentMvp is empty'
  }
  if ($status.completedMvps -contains $status.currentMvp) {
    throw "currentMvp $($status.currentMvp) must not appear in completedMvps"
  }
  if ($status.plannedMvps -notcontains $status.currentMvp) {
    throw "currentMvp $($status.currentMvp) not in plannedMvps"
  }

  $completed = @($status.completedMvps | ForEach-Object { $_.ToString().PadLeft(4, '0') })
  $planned = @($status.plannedMvps | ForEach-Object { $_.ToString().PadLeft(4, '0') })
  $current = $status.currentMvp.ToString().PadLeft(4, '0')

  $seen = @{}
  foreach ($id in $completed) {
    if ($seen.ContainsKey($id)) { throw "duplicate MVP id in status.json: $id" }
    $seen[$id] = 'completed'
  }
  foreach ($id in $planned) {
    if ($seen.ContainsKey($id) -and $seen[$id] -ne 'planned') {
      throw "duplicate MVP id in status.json: $id"
    }
    $seen[$id] = 'planned'
  }
  if ($seen.ContainsKey($current) -and $seen[$current] -eq 'completed') {
    throw "currentMvp $current must not appear in completedMvps"
  }

  $all = @($seen.Keys)

  $planFiles = Get-ChildItem $PlansRoot -Filter '0*.md' |
    Where-Object { $_.Name -match '^(\d{4})-' -and $_.Name -notlike '0000-*' }
  $known = @{}
  foreach ($f in $planFiles) {
    $known[($f.Name -split '-')[0]] = $f.FullName
  }

  foreach ($id in $all) {
    if (-not $known.ContainsKey($id)) {
      throw "unknown MVP id in status.json (no plan file): $id"
    }
  }

  # Predecessor ordering: completed MVPs must have all predecessors also completed.
  $completedSet = @{}
  foreach ($id in $completed) { $completedSet[$id] = $true }

  foreach ($id in $completed) {
    $path = $known[$id]
    $meta = Get-PlanMeta -Path $path
    foreach ($pred in (Get-PredecessorIds $meta.Predecessors)) {
      if ($pred -eq 'None' -or $pred -eq '-') { continue }
      if (-not $completedSet.ContainsKey($pred)) {
        throw "MVP $id is completed but predecessor $pred is not in completedMvps"
      }
    }
  }

  return [pscustomobject]@{
    SchemaVersion           = $status.schemaVersion
    CurrentMvp              = $current
    CompletedMvps           = $completed
    PlannedMvps             = $planned
    LastCertifiedCoreCommit = $status.lastCertifiedCoreCommit
    LastUpdated             = $status.lastUpdated
  }
}

function Get-PredecessorIds {
  param([string]$Predecessors)
  if (-not $Predecessors -or $Predecessors -eq '-' -or $Predecessors -eq 'None') {
    return @()
  }
  return [regex]::Matches($Predecessors, '\b(\d{4})\b') |
    ForEach-Object { $_.Groups[1].Value }
}

function Test-MvpCompleted {
  param($Status, [string]$Id)
  return $Status.CompletedMvps -contains $Id
}

function Get-MvpRollupStatus {
  param($Status, [string]$Id)
  if (Test-MvpCompleted -Status $Status -Id $Id) { return 'done' }
  if ($Id -eq $Status.CurrentMvp) { return 'in-progress' }
  return 'planned'
}
