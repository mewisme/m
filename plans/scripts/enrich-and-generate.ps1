# Enrich all plans/00xx-*.md from enrichment-*.json catalogs,
# then generate CHECKLIST.md, plans/cursor/*.plan.md, and ~/.cursor/plans/mew-*.plan.md

$ErrorActionPreference = 'Stop'
$PlansRoot = Resolve-Path (Join-Path $PSScriptRoot '..')
$RepoRoot = Resolve-Path (Join-Path $PlansRoot '..')
$CursorOut = Join-Path $PlansRoot 'cursor'
$UserPlans = Join-Path $env:USERPROFILE '.cursor\plans'

New-Item -ItemType Directory -Force -Path $CursorOut | Out-Null
New-Item -ItemType Directory -Force -Path $UserPlans | Out-Null

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

function ConvertTo-ChecklistMarkdown {
    param($Entry)
    $sb = New-Object System.Text.StringBuilder
    $groups = [ordered]@{
        'Contracts & types'   = @()
        'Core logic'          = @()
        'CLI / UX'            = @()
        'Tests & fixtures'    = @()
        'Docs & observability'= @()
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
            [void]$sb.AppendLine("- [ ] $it")
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
    } else { '- None beyond architecture rules in 0003 / AGENTS.md.' }

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
    # Strip prior enrichment
    $raw = [regex]::Replace($raw, '(?s)\r?\n<!-- ENRICHMENT:BEGIN -->.*?<!-- ENRICHMENT:END -->\r?\n?', "`n")
    $raw = [regex]::Replace($raw, '(?s)<!-- ENRICHMENT-TESTS -->.*?(?=\r?\nRequired test layers:|\r?\n## )', '')

    $title = ([regex]::Match($raw, '^#\s+(.+)$', 'Multiline')).Groups[1].Value
    $enrich = ConvertTo-EnrichmentBlock -Id $Id -Entry $Entry -Title $title
    $newChecklist = ConvertTo-ChecklistMarkdown -Entry $Entry

    # Replace Detailed Implementation Checklist section through Test Plan header
    $pattern = '(?s)## Detailed Implementation Checklist\r?\n.*?(?=## Test Plan)'
    if ($raw -notmatch $pattern) {
        throw "No Detailed Implementation Checklist in $Path"
    }
    $raw = [regex]::Replace($raw, $pattern, ($newChecklist + "`n"))

    # Insert enrichment before AI-Agent Handoff Contract (or at end if missing)
    $marker = '## AI-Agent Handoff Contract'
    if ($raw.Contains($marker)) {
        $raw = $raw.Replace($marker, ($enrich.TrimEnd() + "`n`n" + $marker))
    } else {
        $raw = $raw.TrimEnd() + "`n`n" + $enrich
    }

    # Expand Test Plan with MVP-specific bullets from acceptance (once)
    if ($raw -notmatch '<!-- ENRICHMENT-TESTS -->') {
        $extraTests = @('<!-- ENRICHMENT-TESTS -->')
        foreach ($a in @($Entry.acceptance)) {
            $extraTests += "- [ ] Acceptance: $a"
        }
        foreach ($f in @($Entry.fixtures)) {
            $extraTests += "- [ ] Fixture ready: ``$f``"
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
    } elseif ($Path -match '(\d{4})-([^.]+)') {
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
        Id = $idPart
        Name = $namePart
        Objective = $obj
        Predecessors = $pred
        Phase = $phase
        ExitItems = @($exitItems)
        NonGoals = @($ng)
        AllCheckboxes = @($exit)
    }
}

function Write-CursorPlan {
    param([string]$Path, $Meta, $Entry)

    $slug = [IO.Path]::GetFileNameWithoutExtension($Path)
    # Path is plan md like 0010-cli-foundation.md
    $base = Split-Path $Path -Leaf
    $slug = $base -replace '\.md$',''
    $todos = @()
    $ti = 0
    foreach ($t in @($Entry.todoSummaries)) {
        $ti++
        if ($t -match '^\s*([^:]+):\s*(.+)$') {
            $tid = ($Matches[1] -replace '[^a-zA-Z0-9\-]', '-').ToLower()
            $content = $Matches[2]
        } else {
            $tid = "t$ti"
            $content = $t
        }
        $todos += "  - id: $tid"
        $todos += "    content: `"$($content -replace '"','''')`""
        $todos += "    status: pending"
    }
    if ($todos.Count -eq 0) {
        $todos = @(
            '  - id: implement'
            '    content: "Implement MVP per source contract"'
            '    status: pending'
        )
    }

    $done = ($Meta.ExitItems | ForEach-Object { "- $_" }) -join "`n"
    if (-not $done) { $done = '- Meet exit criteria in source contract.' }
    $out = ($Meta.NonGoals | ForEach-Object { "- $_" }) -join "`n"
    if (-not $out) { $out = '- Do not implement later MVPs.' }

    $acc = ''; $n = 1
    foreach ($a in @($Entry.acceptance)) { $acc += "$n. $a`n"; $n++ }

    $pkg = ($Entry.packages | Select-Object -First 8) -join ', '
    $steps = @"
1. Confirm predecessors satisfied: $($Meta.Predecessors)
2. Read source contract ``plans/$slug.md`` and ``0003``/``0007``/``0008`` as required.
3. Implement packages: $pkg
4. Add fixtures and tests listed in the source contract.
5. Run focused ``go test`` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).
"@

    $body = @"
---
name: "$($Meta.Id) $($Meta.Name)"
overview: "$($Meta.Objective -replace '"','''')"
todos:
$($todos -join "`n")
isProject: false
---

# $($Meta.Id) - $($Meta.Name)

## Source contract

Canonical scope lives in [``plans/$slug.md``](../$slug.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

## Done when

$done

## Out of scope

$out

## Predecessors

$($Meta.Predecessors)

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

$steps

## Verification

$acc
Suggested commands (once code exists):

``````powershell
go test ./$($Entry.packages[0] -replace '\\','/' -replace '^internal','internal')/... -count=1
``````

Adjust package paths to those actually created. Always include a clean-home fixture test for install-family work.

## Handoff

Before submitting work provide:

1. Behavior summary and compatibility target
2. Files and public interfaces changed
3. Test/benchmark/static-analysis commands
4. Known gaps and platform limits
5. Determinism evidence for generated files
6. Rollback note for persistent-format changes
"@

    $outFile = Join-Path $CursorOut "$slug.plan.md"
    [System.IO.File]::WriteAllText($outFile, ($body.TrimEnd() + "`n"), [System.Text.UTF8Encoding]::new($false))
    $userFile = Join-Path $UserPlans "mew-$slug.plan.md"
    Copy-Item -Path $outFile -Destination $userFile -Force
    return $outFile
}

function Write-Checklist {
    param($Catalog, $PlanFiles)

    $today = Get-Date -Format 'yyyy-MM-dd'
    $sb = New-Object System.Text.StringBuilder
    [void]$sb.AppendLine('# Mew Implementation Master Checklist')
    [void]$sb.AppendLine()
    [void]$sb.AppendLine('## Program status')
    [void]$sb.AppendLine()
    [void]$sb.AppendLine('- Current MVP: (none - planning enrichment complete)')
    [void]$sb.AppendLine("- Last updated: $today")
    [void]$sb.AppendLine('- Source of truth: per-MVP files under `plans/00xx-*.md`')
    [void]$sb.AppendLine('- Regenerate: `.\plans\scripts\enrich-and-generate.ps1`')
    [void]$sb.AppendLine()
    [void]$sb.AppendLine('## Do now')
    [void]$sb.AppendLine()
    [void]$sb.AppendLine('Predecessors satisfied for:')
    [void]$sb.AppendLine()
    [void]$sb.AppendLine('1. [0001 - Program charter](0001-program-charter.md) - no predecessors')
    [void]$sb.AppendLine()
    [void]$sb.AppendLine('## MVP completion (65)')
    [void]$sb.AppendLine()
    [void]$sb.AppendLine('| ID | MVP | Phase | Predecessors | Status | Plan | Cursor plan |')
    [void]$sb.AppendLine('|----|-----|-------|--------------|--------|------|-------------|')

    $agg = New-Object System.Text.StringBuilder
    [void]$agg.AppendLine('## Aggregated tasks by MVP')
    [void]$agg.AppendLine()

    foreach ($f in ($PlanFiles | Sort-Object Name)) {
        if ($f.Name -notmatch '^(\d{4})-') { continue }
        $id = $Matches[1]
        if ($id -eq '0000') { continue }
        $meta = Get-PlanMeta -Path $f.FullName
        $slug = $f.BaseName
        $cursor = "cursor/$slug.plan.md"
        $phase = $meta.Phase
        if (-not $phase) { $phase = $Catalog[$id].phase }
        $pred = $meta.Predecessors
        if (-not $pred) { $pred = '-' }
        $short = $meta.Name
        if ($short.Length -gt 60) { $short = $short.Substring(0, 57) + '...' }
        [void]$sb.AppendLine("| $id | $short | $phase | $pred | [ ] | [$id]($slug.md) | [$slug]($cursor) |")

        [void]$agg.AppendLine("### $id - $($meta.Name)")
        [void]$agg.AppendLine()
        [void]$agg.AppendLine("- status: planned")
        [void]$agg.AppendLine("- plan: [$slug.md]($slug.md)")
        [void]$agg.AppendLine("- cursor: [$cursor]($cursor)")
        [void]$agg.AppendLine()
        $entry = $Catalog[$id]
        if ($entry) {
            foreach ($it in @($entry.scopeItems)) {
                [void]$agg.AppendLine("- [ ] $it")
            }
            foreach ($a in @($entry.acceptance)) {
                [void]$agg.AppendLine("- [ ] Acceptance: $a")
            }
            foreach ($e in @($meta.ExitItems)) {
                [void]$agg.AppendLine("- [ ] Exit: $e")
            }
        } else {
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

function Update-Manifest {
    $files = Get-ChildItem $PlansRoot -Recurse -File | Where-Object {
        $_.FullName -notmatch '\\\.git\\' -and $_.Name -ne 'manifest.json'
    } | Sort-Object FullName

    $entries = @()
    foreach ($f in $files) {
        $rel = $f.FullName.Substring($PlansRoot.Path.Length).TrimStart('\','/') -replace '\\','/'
        $hash = (Get-FileHash -Path $f.FullName -Algorithm SHA256).Hash.ToLower()
        $entries += [ordered]@{
            path = $rel
            bytes = $f.Length
            sha256 = $hash
        }
    }
    $planCount = @(Get-ChildItem $PlansRoot -Filter '0*.md' | Where-Object { $_.Name -match '^\d{4}-' }).Count
    $mdCount = @(Get-ChildItem $PlansRoot -Recurse -Filter '*.md').Count
    $manifest = [ordered]@{
        name = 'Mew Implementation Plan'
        product = [ordered]@{
            full_name = 'Mew'
            binary = 'm'
            executor_full_name = 'Mewx'
            executor_binary = 'mx'
            native_lockfile = 'm.lock'
        }
        language = 'English'
        reference = [ordered]@{
            repository = 'nubjs/nub'
            commit = '08a804359ef301ef8b9307f1258cc185b3270698'
        }
        plan_file_count = $planCount
        markdown_file_count = $mdCount
        validation_errors = @()
        files = $entries
    }
    $json = $manifest | ConvertTo-Json -Depth 8
    Set-Content -Path (Join-Path $PlansRoot 'manifest.json') -Value $json -Encoding utf8
}

# ---- main ----
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

Write-Host 'Generating cursor plans...'
foreach ($f in $planFiles) {
    $id = ($f.Name -split '-')[0]
    $meta = Get-PlanMeta -Path $f.FullName
    Write-CursorPlan -Path $f.FullName -Meta $meta -Entry $Catalog[$id] | Out-Null
}

Write-Host 'Writing CHECKLIST.md...'
Write-Checklist -Catalog $Catalog -PlanFiles $planFiles | Out-Null

Write-Host 'Updating 0000-README and INDEX hints...'
# README / INDEX updated by separate small patch below if needed

Write-Host 'Updating manifest.json...'
Update-Manifest

Write-Host 'Done.'
Write-Host ("Cursor plans: {0}" -f @(Get-ChildItem $CursorOut -Filter '*.plan.md').Count)
Write-Host ("User plans mew-*: {0}" -f @(Get-ChildItem $UserPlans -Filter 'mew-*.plan.md').Count)
