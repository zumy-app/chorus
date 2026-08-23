#Requires -Version 7.0
<#
.SYNOPSIS
  Creates the 14 Phase 1 consolidated issues from BACKLOG_REFINEMENT_2026-08-23.md §6.
.DESCRIPTION
  Dry-run by default (prints curl-equivalent). Use -Execute to actually POST.
  Reads PAT from $env:GITHUB_CHORUS_ISSUES_PAT or .env in repo root.
.PARAMETER Execute
  When present, POST to GitHub. Without it, just prints what would be sent.
.PARAMETER MilestoneNumber
  Number of "Phase 1 Release" milestone. Auto-fetched if omitted and Execute.
.EXAMPLE
  ./scripts/create_phase1_issues.ps1
  ./scripts/create_phase1_issues.ps1 -Execute
  ./scripts/create_phase1_issues.ps1 -Execute -MilestoneNumber 1
#>
param(
  [switch]$Execute,
  [int]$MilestoneNumber = 0
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
$EnvFile = Join-Path $RepoRoot ".env"

# Load PAT from env or .env
$Token = $env:GITHUB_CHORUS_ISSUES_PAT
if (-not $Token -and (Test-Path $EnvFile)) {
  $line = Get-Content $EnvFile | Where-Object { $_ -match "^\s*GITHUB_CHORUS_ISSUES_PAT\s*=" } | Select-Object -First 1
  if ($line) { $Token = ($line -split "=",2)[1].Trim() }
}
if (-not $Token) { Write-Warning "GITHUB_CHORUS_ISSUES_PAT not found in env or .env — dry-run will still print payloads." }

$Owner = "zumy-app"
$Repo  = "chorus"
$ApiBase = "https://api.github.com/repos/$Owner/$Repo"

# Auto-fetch Phase 1 milestone number if Execute and not supplied
if ($Execute -and $MilestoneNumber -eq 0 -and $Token) {
  try {
    $ms = Invoke-RestMethod -Uri "$ApiBase/milestones?state=all&per_page=100" -Headers @{ Authorization="Bearer $Token"; Accept="application/vnd.github+json"; "X-GitHub-Api-Version"="2022-11-28"; "User-Agent"="chorus-script" }
    $phase1 = $ms | Where-Object { $_.title -eq "Phase 1 Release" } | Select-Object -First 1
    if ($phase1) { $MilestoneNumber = $phase1.number; Write-Host "Phase 1 Release milestone number: $MilestoneNumber" }
    else { Write-Warning "Phase 1 Release milestone not found — creating issues without milestone." }
  } catch { Write-Warning "Milestone fetch failed: $($_.Exception.Message)" }
}

# 14 payloads mirroring BACKLOG_REFINEMENT §6
$Issues = @(
  @{
    title = "Phase 1: Feature controls — translation/grammar/highlights toggles"
    labels = @("enhancement","phase-1","priority-high")
    assignee = "batchu"
    body = @'
## Summary
Per-user (later per-chat) toggles for auto-translation, auto grammar, and learning highlights. When off, the server does not enqueue translation/grammar jobs.

## Scope
- DB: `user_settings` add `translation_enabled bool default true`, `grammar_auto bool`, `highlights_enabled bool` (or jsonb flag).
- API: `GET/PUT /api/v1/settings/features`
- Services: gate `translation_jobs` enqueue + GrammarQueueService on flags.
- UI: Settings → Features section with three switches.

## Acceptance
- Toggle off → no jobs created; on → resumes. No dead toggles (FR-18).
Related: FR-25, #6, #25.
'@
  }
  @{
    title = "Phase 1: Word-bank–aware translation (skip known words, cache word-level)"
    labels = @("enhancement","phase-1","priority-high","ai / llm","performance")
    assignee = "gayatriagarwal19"
    body = @'
## Summary
Avoid re-translating words the user already knows (e.g., "hola", "hi"). Consult per-user word bank and serve from word-level cache.

## Scope
- Known threshold: `interval_days >= 21` OR explicit known flag (decide in #35).
- Helper `FilterKnownWords(text, userID)` + Redis `translation:word:{word}:{targetLang}` (24h).
- Metrics: cost/1k tokens, cache hit rate before/after.

## Acceptance
- User with "hola" known → "hola amigo" skips LLM for "hola".
Related: FR-26, #19, #24, #32.
'@
  }
  @{
    title = "Phase 1: Your Learning Path — real metrics (words/sentences per month)"
    labels = @("enhancement","phase-1","priority-high","mobile")
    assignee = "Kushagra1122"
    body = @'
## Summary
Replace mock Learn.tsx (1.2k / 342) with real metrics: words learned per month, sentences understood per month, due reviews, streak, CEFR progress on "Your Learning Path".

## Scope
- Backend: extend `GetLearningProgress` + new `GET /api/v1/learning/path` (monthly buckets).
- Frontend: wire `frontend/src/pages/Learn.tsx` to real data; month picker; empty state.

## Acceptance
- Learn page shows real counts; month switch updates; e2e covers rollover + empty user.
Related: FR-31, #20, #21.
'@
  }
  @{
    title = "Phase 1: Highlight new words in translations + quick practice"
    labels = @("enhancement","phase-1","priority-high","mobile")
    assignee = "gayatriagarwal19"
    body = @'
## Summary
Highlight unlearned words in MessageBubble translations; tap → Save to word bank + Practice CTA.

## Scope
- Frontend: `MessageBubble.tsx` tokenize + diff vs known set, highlight span, tap → `vocabularyAPI.save` + PracticeScreen.
- Backend: reuse vocab endpoints; optional `GET /api/v1/vocabulary/known`.

## Acceptance
- Unknown words highlighted; tap saves; practice navigates.
Related: FR-28, #19, #20.
'@
  }
  @{
    title = "Phase 1: Translation/grammar quality pipeline + Arize Phoenix (offline + realtime)"
    labels = @("ai / llm","phase-1","priority-high","infrastructure","performance")
    assignee = "batchu"
    body = @'
## Summary
Store every translation/grammar with lineage, run cross-model evaluation (different model from producer), critique/refine prompts, expose KPIs. Deploy Arize Phoenix locally for offline + realtime eval.

## Scope
- DB: `translation_evals` + `grammar_evals` tables.
- Services: enqueue evaluator job after write; nightly batch sample.
- Infra: `arizephoenix/phoenix` compose service (6006), OTLP traces.
- Admin KPI cards: accuracy, p95 latency, cost/1k, cache hit.

## Acceptance
- Every row has eval within 5 min; Phoenix traces visible; accuracy graphed.
  See https://arize.com/phoenix
Related: FR-30, NFR-25, #25, #32.
'@
  }
  @{
    title = "Phase 1: AI writing assistant (draft in target language)"
    labels = @("enhancement","phase-1","priority-high","ai / llm")
    assignee = "gayatriagarwal19"
    body = @'
## Summary
"Help me write" in composer: user drafts in target language, AI fills gaps/corrects/explains before sending. Part of writing goals.

## Scope
- API: `POST /api/v1/ai/writing-assist {draft, targetLang, nativeLang}` → `{suggestion, corrections[]}`.
- UI: Composer sheet with diff + Insert vs Send.

## Acceptance
- Draft "Yo quiero ... restaurante" → AI completes/corrects, user can insert/edit before send.
Related: FR-33, #22.
'@
  }
  @{
    title = "Phase 1: AI scenario role-play (restaurant, date, etc.)"
    labels = @("enhancement","phase-1","priority-medium","ai / llm")
    assignee = "gayatriagarwal19"
    body = @'
## Summary
AI generates scenarios and role-plays ("restaurant order", "date"). Lives in Learn tab + Chat.

## Scope
- API: `POST /api/v1/ai/scenario/start` + turn endpoint.
- UI: scenario cards, turn bubbles, inline corrections, end recap → add words to review queue.

## Acceptance
- Restaurant scenario → 5-turn role-play with corrections; recap shows words to review.
Related: FR-34, #17.
'@
  }
  @{
    title = "Phase 1: Simplify Chat Language Settings — own language only"
    labels = @("enhancement","phase-1","priority-high","mobile")
    assignee = "Kushagra1122"
    body = @'
## Summary
Remove the "other person''s language" dropdown from Chat Language Settings. Only the user''s own language.

## Scope
- File: `frontend/src/components/ChatLanguageModal.tsx:14-77` — delete `theirLanguage` state + second <select> + preview. Keep `myLanguage`.
- i18n: keep or remove `chatLanguageModal.contactLanguage*` keys.
- Tests: update Settings test + visual regression.

## Acceptance
- Modal shows one dropdown (own language) + preview; no contact language reference.
Related: FR-35.
'@
  }
  @{
    title = "Phase 1: Harden mail server — isolate from app prod (security)"
    labels = @("security","phase-1","priority-high","infrastructure")
    assignee = "batchu"
    body = @'
## Summary
Prod mail (Mailu) runs "same space" as app — isolate it. Rotate SMTP_PASSWORD (was in repo history).

## Scope
- Move Mailu to separate host or isolated Docker network; only 587 from app; block 25 from public.
- Rotate `SMTP_PASSWORD`, move to Dokploy secrets; enforce SPF/DKIM/DMARC per #3.
- Document in ARCHITECTURE.md + DOKPLOY_DEPLOY.md.

## Acceptance
- App can only reach mail on 587 from app subnet; mail-tester passes; creds rotated.
Related: NFR-22, #3, #2.
'@
  }
  @{
    title = "Phase 1: Dev environment + CI/CD quality gates (Raju) — dev→prod auto-promotion"
    labels = @("infrastructure","phase-1","priority-high","testing")
    assignee = "gosangiraju"
    body = @'
## Summary
Create `dev` environment; Raju owns CI/CD. Quality gates run on dev (functional, e2e, Phoenix eval, load smoke) before auto-promotion to prod.

## Scope
- Infra: `docker-compose.dev.yml` overlay + Dokploy project `chorus-dev`.
- CI: `.github/workflows/ci.yml` — build → push → deploy to dev → gates → promote image to prod if green.
- Gates: go test, npm test, Playwright e2e, Phoenix golden-set eval, Artillery smoke (100 WS / 10 msg/s / 5m), govulncheck + npm audit.
- Branch protection on main; image promotion (not rebuild).

## Acceptance
- Push to main deploys to dev, gates run, prod only updated if green; demo recorded.
Assigned: Raju (gosangiraju).
Related: NFR-26, #6, #36, #37.
'@
  }
  @{
    title = "Phase 2: Teacher vetting process — assessment → recording → expert video review"
    labels = @("documentation","phase-2","priority-low","product")
    assignee = "batchu"
    body = @'
## Summary
Document (Phase 1) and later execute (Phase 2) teacher onboarding: assessment → live recording → certificates → manual video-call review by language experts (Daniella for Spanish; one expert per language). Later train AI on rubric.

## Scope
- Phase 1 (doc-only): `docs/TEACHER_VETTING.md` — rubric, recording prompt, cert checklist, video-call flow, expert roster.
- Phase 2: profiles/ratings/class sign-up gated on vetting (see #53).

## Acceptance (Phase 1)
- `docs/TEACHER_VETTING.md` merged; Daniella confirmed in writing.
Related: #53 epic.
'@
  }
  @{
    title = "Tracking: Break Contacts & Invites epic (#44) into 3 PRs"
    labels = @("epic","phase-1","priority-high","mobile")
    assignee = "Kushagra1122"
    body = @'
Tracking issue for epic #44 — break into: (1) permission + hashed matching, (2) invite token + email/SMS, (3) status UI. Link PRs here.
'@
  }
  @{
    title = "Housekeeping: Close Phase 0 milestone + relabel phase-0 → phase-1"
    labels = @("product","phase-1","priority-high")
    assignee = "batchu"
    body = @'
Execute BACKLOG_REFINEMENT_2026-08-23.md §4.1: move 10 issues from "Phase 0 Release" to "Phase 1 Release", replace `phase-0` with `phase-1`, close milestone "Phase 0 Release", close superseded #12 and #48.

Steps in backlog doc §7 — or run:
  ./scripts/move_phase0_to_phase1.ps1 -Execute
'@
  }
  @{
    title = "Docs: Deprecate phase-0 label + document Phase 1 P0 convention"
    labels = @("documentation","phase-1","priority-low")
    assignee = "batchu"
    body = @'
Deprecate `phase-0` label (keep or delete) and document that former Phase 0 is now Phase 1 P0 (`phase-1` + `priority-high`). Update `.github/labels.yml` if present and CONTRIBUTING.md.
'@
  }
)

function New-GitHubIssue {
  param($Payload)
  $json = $Payload | ConvertTo-Json -Depth 4
  if (-not $Execute) {
    Write-Host "`n--- DRY-RUN: would POST issue ---" -ForegroundColor Yellow
    Write-Host ("Title: " + $Payload.title)
    Write-Host ("Labels: " + ($Payload.labels -join ", ") + "  milestone: " + $Payload.milestone + "  assignee: " + $Payload.assignees)
    Write-Host $json
    return
  }
  $headers = @{ Authorization="Bearer $Token"; Accept="application/vnd.github+json"; "X-GitHub-Api-Version"="2022-11-28"; "User-Agent"="chorus-script" }
  # Keep milestones: if we have a number, include it; otherwise omit (API requires int or null)
  $body = @{ title=$Payload.title; body=$Payload.body; labels=$Payload.labels }
  if ($MilestoneNumber -ne 0) { $body.milestone = $MilestoneNumber }
  if ($Payload.assignees) { $body.assignees = $Payload.assignees }
  elseif ($Payload.assignee) { $body.assignees = @($Payload.assignee) }
  $json2 = $body | ConvertTo-Json -Depth 4
  $resp = Invoke-RestMethod -Uri "$ApiBase/issues" -Method POST -Headers $headers -Body $json2 -ContentType "application/json"
  Write-Host "Created #$($resp.number): $($resp.html_url)" -ForegroundColor Green
}

foreach ($it in $Issues) {
  $payload = @{
    title = $it.title
    body  = $it.body
    labels = $it.labels
    assignee = $it.assignee
  }
  if ($MilestoneNumber -ne 0) { $payload.milestone = $MilestoneNumber }
  New-GitHubIssue -Payload $payload
  Start-Sleep -Milliseconds 400
}

if (-not $Execute) {
  Write-Host "`nDry-run complete. Re-run with -Execute once PAT is valid." -ForegroundColor Cyan
  Write-Host "Example: `$env:GITHUB_CHORUS_ISSUES_PAT='github_pat_...' ; ./scripts/create_phase1_issues.ps1 -Execute" -ForegroundColor Cyan
}
