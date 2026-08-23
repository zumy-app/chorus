#!/usr/bin/env bash
# Creates the 14 Phase 1 issues — bash version for Linux/macOS.
# Dry-run by default; use --execute to POST. Requires GITHUB_CHORUS_ISSUES_PAT.
set -euo pipefail
REPO="zumy-app/chorus"
API="https://api.github.com/repos/$REPO/issues"
EXECUTE=false
if [[ "${1:-}" == "--execute" ]]; then EXECUTE=true; fi

TOKEN="${GITHUB_CHORUS_ISSUES_PAT:-}"
if [[ -z "$TOKEN" && -f "$(dirname "$0")/../.env" ]]; then
  TOKEN="$(grep -E '^\s*GITHUB_CHORUS_ISSUES_PAT\s*=' "$(dirname "$0")/../.env" | head -n1 | cut -d= -f2- | tr -d '[:space:]')"
fi
if [[ "$EXECUTE" == true && -z "$TOKEN" ]]; then echo "GITHUB_CHORUS_ISSUES_PAT not set"; exit 1; fi

# Fetch Phase 1 milestone number
MILESTONE=""
if [[ "$EXECUTE" == true ]]; then
  MILESTONE="$(curl -s -H "Authorization: Bearer $TOKEN" -H "Accept: application/vnd.github+json" "https://api.github.com/repos/$REPO/milestones?state=all&per_page=100" | python3 -c "import sys,json; data=json.load(sys.stdin); m=[x for x in data if x['title']=='Phase 1 Release']; print(m[0]['number'] if m else '')" 2>/dev/null || true)"
  echo "Phase 1 milestone: ${MILESTONE:-<none>}"
fi

post_issue() {
  local title="$1" labels="$2" body="$3" assignee="$4"
  local json
  if [[ -n "$MILESTONE" ]]; then
    json=$(python3 -c "import json,sys; print(json.dumps({'title':sys.argv[1],'body':sys.argv[2],'labels':sys.argv[3].split(','),'milestone':int(sys.argv[4]),'assignees':[sys.argv[5]] if sys.argv[5] else []}))" "$title" "$body" "$labels" "$MILESTONE" "$assignee")
  else
    json=$(python3 -c "import json,sys; print(json.dumps({'title':sys.argv[1],'body':sys.argv[2],'labels':sys.argv[3].split(',')}))" "$title" "$body" "$labels")
  fi
  if [[ "$EXECUTE" == false ]]; then
    echo "--- DRY-RUN: $title [$labels] assignee:$assignee ---"
    echo "$json" | python3 -m json.tool | head -n 40
    return
  fi
  curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Accept: application/vnd.github+json" -H "Content-Type: application/json" -d "$json" "$API" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f\"Created #{d.get('number')} {d.get('html_url')}\")"
  sleep 0.4
}

# Bodies are short here — full markdown in BACKLOG_REFINEMENT §6 and .ps1 payloads.
post_issue "Phase 1: Feature controls — translation/grammar/highlights toggles" "enhancement,phase-1,priority-high" "FR-25. See BACKLOG_REFINEMENT_2026-08-23.md §6.1 for full body." "batchu"
post_issue "Phase 1: Word-bank–aware translation (skip known words, cache word-level)" "enhancement,phase-1,priority-high,ai / llm,performance" "FR-26. §6.2" "gayatriagarwal19"
post_issue "Phase 1: Your Learning Path — real metrics (words/sentences per month)" "enhancement,phase-1,priority-high,mobile" "FR-31. §6.3" "Kushagra1122"
post_issue "Phase 1: Highlight new words in translations + quick practice" "enhancement,phase-1,priority-high,mobile" "FR-28. §6.4" "gayatriagarwal19"
post_issue "Phase 1: Translation/grammar quality pipeline + Arize Phoenix (offline + realtime)" "ai / llm,phase-1,priority-high,infrastructure,performance" "FR-30/NFR-25. §6.5 https://arize.com/phoenix" "batchu"
post_issue "Phase 1: AI writing assistant (draft in target language)" "enhancement,phase-1,priority-high,ai / llm" "FR-33. §6.6" "gayatriagarwal19"
post_issue "Phase 1: AI scenario role-play (restaurant, date, etc.)" "enhancement,phase-1,priority-medium,ai / llm" "FR-34. §6.7" "gayatriagarwal19"
post_issue "Phase 1: Simplify Chat Language Settings — own language only" "enhancement,phase-1,priority-high,mobile" "FR-35. ChatLanguageModal.tsx:14-77 §6.8" "Kushagra1122"
post_issue "Phase 1: Harden mail server — isolate from app prod (security)" "security,phase-1,priority-high,infrastructure" "NFR-22. §6.9" "batchu"
post_issue "Phase 1: Dev environment + CI/CD quality gates (Raju) — dev→prod auto-promotion" "infrastructure,phase-1,priority-high,testing" "NFR-26. §6.10 assignee Raju (gosangiraju)" "gosangiraju"
post_issue "Phase 2: Teacher vetting process — assessment → recording → expert video review" "documentation,phase-2,priority-low,product" "NFR-27 doc. §6.11 Daniella for Spanish" "batchu"
post_issue "Tracking: Break Contacts & Invites epic (#44) into 3 PRs" "epic,phase-1,priority-high,mobile" "Subtask of #44 §6.12" "Kushagra1122"
post_issue "Housekeeping: Close Phase 0 milestone + relabel phase-0 → phase-1" "product,phase-1,priority-high" "Move 10 issues, close milestone. §6.13" "batchu"
post_issue "Docs: Deprecate phase-0 label + document Phase 1 P0 convention" "documentation,phase-1,priority-low" "§6.14" "batchu"

if [[ "$EXECUTE" == false ]]; then echo "Dry-run done. Re-run with --execute once PAT is valid."; fi
