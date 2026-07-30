# Enrich all plans/00xx-*.md from enrichment-*.json catalogs,
# then generate CHECKLIST.md and manifest.json

$ErrorActionPreference = 'Stop'
$PlansRoot = Resolve-Path (Join-Path $PSScriptRoot '..')
$RepoRoot = Resolve-Path (Join-Path $PlansRoot '..')

. (Join-Path $PSScriptRoot 'Read-Status.ps1')

function Get-Catalog {
  $merged = [ordered]@{}
  Get-ChildItem (Join-Path $PSScriptRoot 'enrichment-*.json') | ForEach-Object {
    $obj = Get-Content $_.FullName -Raw -Encoding utf8 | ConvertFrom-Json
    foreach ($p in $obj.PSObject.Properties) {
      $merged[$p.Name] = $p.Value
    }
  }
  return $merged
}

function Get-CheckedItemTexts {
  param([string]$Raw)
  $checked = @{}
  foreach ($m in [regex]::Matches($Raw, '(?m)^- \[x\] (.+)$')) {
    $checked[$m.Groups[1].Value] = $true
  }
  return $checked
}

function Format-ChecklistItem {
  param([string]$Text, $Checked)
  $mark = if ($Checked.ContainsKey($Text)) { 'x' } else { ' ' }
  return "- [$mark] $Text"
}

function ConvertTo-ChecklistMarkdown {
  param($Entry, $Checked)
  if (-not $Checked) { $Checked = @{} }
  $sb = New-Object System.Text.StringBuilder
  $groups = [ordered]@{
    'Contracts & types'    = @()
    'Core logic'           = @()
    'CLI / UX'             = @()
    'Tests & fixtures'     = @()
    'Docs & observability' = @()
  }
  $i = 0
  $items = @($Entry.scopeItems)
  foreach ($item in $items) {
    $keys = @($groups.Keys)
    $bucket = $keys[$i % 5]
    $groups[$bucket] += $item
    $i++
  }
  [void]$sb.AppendLine('## Detailed Implementation Checklist')
  [void]$sb.AppendLine()
  foreach ($g in $groups.Keys) {
    if ($groups[$g].Count -eq 0) { continue }
    [void]$sb.AppendLine("### $g")
    [void]$sb.AppendLine()
    foreach ($it in $groups[$g]) {
      [void]$sb.AppendLine((Format-ChecklistItem -Text $it -Checked $Checked))
    }
    [void]$sb.AppendLine()
  }
  return $sb.ToString().TrimEnd() + "`n"
}

function ConvertTo-EnrichmentBlock {
  param([string]$Id, $Entry, [string]$Title)

  $feat = (@($Entry.featureRows) | ForEach-Object { $_ }) -join "`n"
  if (-not $feat) { $feat = '| (see 0002 inventory) | - | - | ' + $Id + ' |' }

  $pkgLines = (@($Entry.packages) | ForEach-Object { "- ``$_``" }) -join "`n"
  $forbid = if (@($Entry.forbiddenImports).Count -gt 0) {
    (@($Entry.forbiddenImports) | ForEach-Object { "- $_" }) -join "`n"
  }
  else { '- None beyond architecture rules in 0003 / AGENTS.md.' }

  $fixLines = (@($Entry.fixtures) | ForEach-Object { "- ``$_``" }) -join "`n"
  $acc = ''
  $n = 1
  foreach ($a in @($Entry.acceptance)) { $acc += "$n. $a`n"; $n++ }
  $confLines = (@($Entry.conformance) | ForEach-Object { "- $_" }) -join "`n"
  $open = (@($Entry.openDecisions) | ForEach-Object { "- $_" }) -join "`n"

  $df = @"
``````mermaid
flowchart LR
  $($Entry.dataFlowNodes)
``````
"@

  return @"
<!-- ENRICHMENT:BEGIN -->

## Feature Inventory Links

Rows this MVP owns or primarily advances (from ``0002`` inventory themes):

| Feature | Nub baseline | Mew target | Primary MVP |
|---|---|---|---|
$feat

## Go Package Map

**Packages / paths:**

$pkgLines

**Forbidden import edges:**

$forbid

## Data Flow

$df

## Commands and Flags

$($Entry.commandsFlags)

## Persistent Artifacts

$($Entry.artifacts)

## Concrete Test Fixtures

$fixLines

## Acceptance Scenarios

$acc
## Nub Conformance Targets

$confLines

## Open Decisions

$open

<!-- ENRICHMENT:END -->
"@
}

function Update-PlanFile {
  param([string]$Path, [string]$Id, $Entry)

  $raw = Get-Content $Path -Raw -Encoding utf8
  $checked = Get-CheckedItemTexts -Raw $raw

  # Strip prior enrichment block only (keep surrounding paragraph breaks).
  $raw = [regex]::Replace($raw, '(?s)<!-- ENRICHMENT:BEGIN -->.*?<!-- ENRICHMENT:END -->\r?\n*', '')
  $raw = [regex]::Replace($raw, '(?s)<!-- ENRICHMENT-TESTS -->.*?(?=\r?\nRequired test layers:|\r?\n## )', '')

  $title = ([regex]::Match($raw, '^#\s+(.+)$', 'Multiline')).Groups[1].Value
  $enrich = ConvertTo-EnrichmentBlock -Id $Id -Entry $Entry -Title $title
  $newChecklist = ConvertTo-ChecklistMarkdown -Entry $Entry -Checked $checked

  # Replace Detailed Implementation Checklist section through Test Plan header
  $pattern = '(?s)## Detailed Implementation Checklist\r?\n.*?(?=## Test Plan)'
  if ($raw -notmatch $pattern) {
    throw "No Detailed Implementation Checklist in $Path"
  }
  $raw = [regex]::Replace($raw, $pattern, ($newChecklist + "`n"))

  # Insert enrichment before AI-Agent Handoff Contract (or at end if missing)
  $marker = '## AI-Agent Handoff Contract'
  if ($raw.Contains($marker)) {
    $raw = [regex]::Replace($raw, "(\r?\n){3,}(?=$([regex]::Escape($marker)))", "`n`n")
    $raw = $raw.Replace($marker, ("`n`n" + $enrich.TrimEnd() + "`n`n" + $marker))
  }
  else {
    $raw = $raw.TrimEnd() + "`n`n" + $enrich
  }

  # Expand Test Plan with MVP-specific bullets from acceptance (once)
  if ($raw -notmatch '<!-- ENRICHMENT-TESTS -->') {
    $extraTests = @('<!-- ENRICHMENT-TESTS -->')
    foreach ($a in @($Entry.acceptance)) {
      $extraTests += (Format-ChecklistItem -Text "Acceptance: $a" -Checked $checked)
    }
    foreach ($f in @($Entry.fixtures)) {
      $extraTests += (Format-ChecklistItem -Text "Fixture ready: ``$f``" -Checked $checked)
    }
    $extraTests += ''
    $block = ($extraTests -join "`n") + "`n"
    $raw = [regex]::Replace($raw, '(## Test Plan\r?\n\r?\n)', ('$1' + $block))
  }

  $toWrite = $raw.TrimEnd() + "`n"
  [System.IO.File]::WriteAllText($Path, $toWrite, [System.Text.UTF8Encoding]::new($false))
}

function Get-PlanMeta {
  param([string]$Path)
  $raw = Get-Content $Path -Raw -Encoding utf8
  $idPart = ''
  $namePart = ''
  $first = (($raw -split "`r?`n") | Where-Object { $_ -match '^#' } | Select-Object -First 1)
  if ($first -match '^#\s+(\d{4})\s+\S+\s+(.+)$') {
    $idPart = $Matches[1]
    $namePart = $Matches[2].Trim()
  }
  elseif ($Path -match '(\d{4})-([^.]+)') {
    $idPart = $Matches[1]
    $namePart = ($Matches[2] -replace '-', ' ')
  }
  $obj = ([regex]::Match($raw, '(?m)^\|\s*Primary objective\s*\|\s*(.+?)\s*\|')).Groups[1].Value
  $pred = ([regex]::Match($raw, '(?m)^\|\s*Required predecessors\s*\|\s*(.+?)\s*\|')).Groups[1].Value
  $phase = ([regex]::Match($raw, '(?m)^\|\s*Phase\s*\|\s*(.+?)\s*\|')).Groups[1].Value
  $exit = [regex]::Matches($raw, '(?m)^- \[ \] (.+)$') | ForEach-Object { $_.Groups[1].Value }
  $exitSection = [regex]::Match($raw, '(?s)## Exit Criteria\r?\n\r?\n(.*?)(?=\r?\n## )')
  $exitItems = @()
  if ($exitSection.Success) {
    $exitItems = [regex]::Matches($exitSection.Groups[1].Value, '(?m)^- \[ \] (.+)$') | ForEach-Object { $_.Groups[1].Value }
  }
  $nonGoals = [regex]::Match($raw, '(?s)## Explicit Non-Goals\r?\n\r?\n(.*?)(?=\r?\n## )')
  $ng = @()
  if ($nonGoals.Success) {
    $ng = [regex]::Matches($nonGoals.Groups[1].Value, '(?m)^- (.+)$') | ForEach-Object { $_.Groups[1].Value }
  }
  return [pscustomobject]@{
    Id            = $idPart
    Name          = $namePart
    Objective     = $obj
    Predecessors  = $pred
    Phase         = $phase
    ExitItems     = @($exitItems)
    NonGoals      = @($ng)
    AllCheckboxes = @($exit)
  }
}

function Get-PreservedNarrative {
  param([string]$ChecklistPath)
  if (-not (Test-Path $ChecklistPath)) { return '' }
  $raw = Get-Content $ChecklistPath -Raw -Encoding utf8
  $m = [regex]::Match($raw, '(?s)<!-- CHECKLIST:NARRATIVE:BEGIN -->\r?\n(.*?)\r?\n<!-- CHECKLIST:NARRATIVE:END -->')
  if ($m.Success) {
    return $m.Groups[1].Value.TrimEnd()
  }
  # One-time migration: capture stabilization notes after **Next:** line.
  $doNow = [regex]::Match($raw, '(?s)## Do now\r?\n\r?\n\*\*Next:\*\*[^\n]*\r?\n\r?\n(.*?)(?=\r?\n## MVP completion)')
  if ($doNow.Success) {
    return $doNow.Groups[1].Value.TrimEnd()
  }
  return ''
}

function Write-Checklist {
  param($Catalog, $PlanFiles, $Status)

  $today = $Status.LastUpdated
  if (-not $today) { $today = Get-Date -Format 'yyyy-MM-dd' }
  $checklistPath = Join-Path $PlansRoot 'CHECKLIST.md'
  $narrative = Get-PreservedNarrative -ChecklistPath $checklistPath

  $currentMeta = $null
  $currentSlug = $null
  foreach ($f in $PlanFiles) {
    $id = ($f.Name -split '-')[0]
    if ($id -eq $Status.CurrentMvp) {
      $currentMeta = Get-PlanMeta -Path $f.FullName
      $currentSlug = $f.BaseName
      break
    }
  }
  if (-not $currentMeta) {
    throw "no plan file for currentMvp $($Status.CurrentMvp)"
  }

  $sb = New-Object System.Text.StringBuilder
  [void]$sb.AppendLine('# Mew Implementation Master Checklist')
  [void]$sb.AppendLine()
  [void]$sb.AppendLine('## Program status')
  [void]$sb.AppendLine()
  [void]$sb.AppendLine("- Current MVP: **$($Status.CurrentMvp)** — $($currentMeta.Name)")
  [void]$sb.AppendLine("- Last updated: $today")
  [void]$sb.AppendLine('- Source of truth: per-MVP files under `plans/00xx-*.md`')
  [void]$sb.AppendLine('- Regenerate: `.\plans\scripts\enrich-and-generate.ps1`')
  if ($Status.LastCertifiedCoreCommit) {
    [void]$sb.AppendLine("- Last certified core commit: ``$($Status.LastCertifiedCoreCommit)``")
  }
  [void]$sb.AppendLine()
  [void]$sb.AppendLine('## Do now')
  [void]$sb.AppendLine()
  [void]$sb.AppendLine("**Next:** [$($Status.CurrentMvp) - $($currentMeta.Name)]($currentSlug.md)")
  [void]$sb.AppendLine()
  if ($narrative) {
    [void]$sb.AppendLine('<!-- CHECKLIST:NARRATIVE:BEGIN -->')
    [void]$sb.AppendLine($narrative)
    [void]$sb.AppendLine('<!-- CHECKLIST:NARRATIVE:END -->')
    [void]$sb.AppendLine()
  }
  [void]$sb.AppendLine('## MVP completion (65)')
  [void]$sb.AppendLine()
  [void]$sb.AppendLine('| ID | MVP | Phase | Predecessors | Status | Plan |')
  [void]$sb.AppendLine('|----|-----|-------|--------------|--------|------|')

  $agg = New-Object System.Text.StringBuilder
  [void]$agg.AppendLine('## Aggregated tasks by MVP')
  [void]$agg.AppendLine()

  foreach ($f in ($PlanFiles | Sort-Object Name)) {
    if ($f.Name -notmatch '^(\d{4})-') { continue }
    $id = $Matches[1]
    if ($id -eq '0000') { continue }
    $meta = Get-PlanMeta -Path $f.FullName
    $slug = $f.BaseName
    $phase = $meta.Phase
    if (-not $phase) { $phase = $Catalog[$id].phase }
    $pred = $meta.Predecessors
    if (-not $pred) { $pred = '-' }
    $short = $meta.Name
    if ($short.Length -gt 60) { $short = $short.Substring(0, 57) + '...' }
    $done = Test-MvpCompleted -Status $Status -Id $id
    $mark = if ($done) { '[x]' } else { '[ ]' }
    [void]$sb.AppendLine("| $id | $short | $phase | $pred | $mark | [$id]($slug.md) |")

    $rollup = Get-MvpRollupStatus -Status $Status -Id $id
    [void]$agg.AppendLine("### $id - $($meta.Name)")
    [void]$agg.AppendLine()
    [void]$agg.AppendLine("- status: $rollup")
    [void]$agg.AppendLine("- plan: [$slug.md]($slug.md)")
    [void]$agg.AppendLine()
    $entry = $Catalog[$id]
    if ($entry) {
      foreach ($it in @($entry.scopeItems)) {
        $itemMark = if ($done) { 'x' } else { ' ' }
        [void]$agg.AppendLine("- [$itemMark] $it")
      }
      foreach ($a in @($entry.acceptance)) {
        $itemMark = if ($done) { 'x' } else { ' ' }
        [void]$agg.AppendLine("- [$itemMark] Acceptance: $a")
      }
      foreach ($e in @($meta.ExitItems)) {
        $itemMark = if ($done) { 'x' } else { ' ' }
        [void]$agg.AppendLine("- [$itemMark] Exit: $e")
      }
    }
    else {
      [void]$agg.AppendLine('- [ ] (enrichment catalog missing for this id)')
    }
    [void]$agg.AppendLine()
  }

  [void]$sb.AppendLine()
  [void]$sb.AppendLine($agg.ToString())
  $out = Join-Path $PlansRoot 'CHECKLIST.md'
  [System.IO.File]::WriteAllText($out, ($sb.ToString().TrimEnd() + "`n"), [System.Text.UTF8Encoding]::new($false))
  return $out
}

function Get-ProductIdentity {
  $path = Join-Path $RepoRoot 'product\identity.json'
  if (-not (Test-Path $path)) {
    throw "missing product identity: $path"
  }
  return Get-Content $path -Raw -Encoding utf8 | ConvertFrom-Json
}

function Update-Manifest {
  $identity = Get-ProductIdentity
  $files = Get-ChildItem $PlansRoot -Recurse -File | Where-Object {
    $_.FullName -notmatch '\\\.git\\' -and $_.Name -ne 'manifest.json'
  } | Sort-Object FullName

  $entries = @()
  foreach ($f in $files) {
    $rel = $f.FullName.Substring($PlansRoot.Path.Length).TrimStart('\', '/') -replace '\\', '/'
    $hash = (Get-FileHash -Path $f.FullName -Algorithm SHA256).Hash.ToLower()
    $entries += [ordered]@{
      path   = $rel
      bytes  = $f.Length
      sha256 = $hash
    }
  }
  $planCount = @(Get-ChildItem $PlansRoot -Filter '0*.md' | Where-Object { $_.Name -match '^\d{4}-' }).Count
  $mdCount = @(Get-ChildItem $PlansRoot -Recurse -Filter '*.md').Count
  $manifest = [ordered]@{
    name                = 'Mew Implementation Plan'
    product             = [ordered]@{
      full_name        = $identity.full_name
      short_name       = $identity.short_name
      binary           = $identity.primary_binary
      primary_alias    = $identity.primary_alias
      executor_binary  = $identity.executor_binary
      executor_alias   = $identity.executor_alias
      native_lockfile  = $identity.native_lockfile
    }
    language            = 'English'
    reference           = [ordered]@{
      repository = 'nubjs/nub'
      commit     = '08a804359ef301ef8b9307f1258cc185b3270698'
    }
    plan_file_count     = $planCount
    markdown_file_count = $mdCount
    validation_errors   = @()
    files               = $entries
  }
  $json = $manifest | ConvertTo-Json -Depth 8
  $outPath = Join-Path $PlansRoot 'manifest.json'
  [System.IO.File]::WriteAllText($outPath, ($json + "`n"), [System.Text.UTF8Encoding]::new($false))
}

# ---- main ----
Write-Host 'Loading status...'
$Status = Get-PlanStatus -PlansRoot $PlansRoot

Write-Host 'Loading catalogs...'
$Catalog = Get-Catalog
Write-Host ("Catalog entries: {0}" -f $Catalog.Count)

$planFiles = Get-ChildItem $PlansRoot -Filter '0*.md' | Where-Object { $_.Name -match '^(\d{4})-' -and $_.Name -notlike '0000-*' } | Sort-Object Name
$missing = @()
foreach ($f in $planFiles) {
  $id = ($f.Name -split '-')[0]
  if (-not $Catalog.Contains($id)) { $missing += $id }
}
if ($missing.Count -gt 0) {
  throw ("Missing catalog for: {0}" -f ($missing -join ', '))
}

Write-Host 'Enriching plan files...'
foreach ($f in $planFiles) {
  $id = ($f.Name -split '-')[0]
  Write-Host "  enrich $id"
  Update-PlanFile -Path $f.FullName -Id $id -Entry $Catalog[$id]
}

Write-Host 'Writing CHECKLIST.md...'
Write-Checklist -Catalog $Catalog -PlanFiles $planFiles -Status $Status | Out-Null

Write-Host 'Updating manifest.json...'
Update-Manifest

Write-Host 'Done.'
