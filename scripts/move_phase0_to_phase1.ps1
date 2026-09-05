# Moves the 10 Phase 0 issues to Phase 1 milestone + swaps phase-0 → phase-1.
param([switch]$Execute)
$ErrorActionPreference="Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
$EnvFile = Join-Path $RepoRoot ".env"
$Token = $env:GITHUB_CHORUS_ISSUES_PAT
if (-not $Token -and (Test-Path $EnvFile)) {
  $line = Get-Content $EnvFile | Where-Object { $_ -match "^\s*GITHUB_CHORUS_ISSUES_PAT\s*=" } | Select-Object -First 1
  if ($line) { $Token = ($line -split "=",2)[1].Trim() }
}
$Owner="zumy-app"; $Repo="chorus"; $ApiBase="https://api.github.com/repos/$Owner/$Repo"
$Phase0Issues = @(40,41,42,43,44,45,46,47,48,49)

function Get-Milestones {
  $h=@{ Authorization="Bearer $Token"; Accept="application/vnd.github+json"; "X-GitHub-Api-Version"="2022-11-28"; "User-Agent"="chorus-script" }
  Invoke-RestMethod -Uri "$ApiBase/milestones?state=all&per_page=100" -Headers $h
}

if (-not $Token) { Write-Host "No PAT — dry-run only"; $Execute=$false }

if ($Execute) {
  $ms=Get-Milestones
  $p1=($ms | Where-Object title -eq "Phase 1 Release")[0]
  $p0=($ms | Where-Object title -eq "Phase 0 Release")[0]
  if (-not $p1) { throw "Phase 1 milestone not found" }
  Write-Host "Phase 1 #$($p1.number), Phase 0 #$($p0.number)"
  foreach ($n in $Phase0Issues) {
    $issue = Invoke-RestMethod -Uri "$ApiBase/issues/$n" -Headers @{Authorization="Bearer $Token"; Accept="application/vnd.github+json"; "User-Agent"="chorus-script"}
    $labels = @($issue.labels | ForEach-Object name | Where-Object { $_ -ne "phase-0" }) + "phase-1"
    $labels = $labels | Select-Object -Unique
    if ("priority-high" -notin $labels) { $labels += "priority-high" }
    $body = @{ milestone=$p1.number; labels=$labels } | ConvertTo-Json
    if ($n -eq 48) { Write-Host "Skipping move for #48 (will be closed)" ; continue }
    Write-Host "Moving #$n → milestone $($p1.number), labels $($labels -join ',')"
    Invoke-RestMethod -Uri "$ApiBase/issues/$n" -Method Patch -Headers @{Authorization="Bearer $Token"; Accept="application/vnd.github+json"; "User-Agent"="chorus-script"} -Body $body -ContentType "application/json" | Out-Null
    Start-Sleep -Milliseconds 400
  }
  # Close housekeeping issues after move
  foreach ($n in @(48,12)) {
    Write-Host "Closing #$n as superseded"
    $b=@{ state="closed"; state_reason="completed" } | ConvertTo-Json
    try { Invoke-RestMethod -Uri "$ApiBase/issues/$n" -Method Patch -Headers @{Authorization="Bearer $Token"; Accept="application/vnd.github+json"; "User-Agent"="chorus-script"} -Body $b -ContentType "application/json" | Out-Null } catch { Write-Warning $_ }
  }
  Write-Host "Done. Close Phase 0 milestone manually after verifying: PATCH /milestones/$($p0.number) {state:closed}"
} else {
  Write-Host "Dry-run — would move issues $($Phase0Issues -join ',') to Phase 1 Release and swap phase-0 → phase-1"
  Write-Host "Re-run with -Execute after fixing PAT."
}
