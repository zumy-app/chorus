#!/usr/bin/env bash
# =============================================================================
# verify-release-gate.sh — 14.3 Release gate + Go/No-Go enforcement (#36/#37)
# =============================================================================
# Mechanical check for docs/RELEASE_GATE.md + docs/GO_NO_GO.md.
# Fails (exit 1) when any automatable gate is red so CI blocks promotion.
#
# Modes:
#   --offline  no DB — checks docs, sign-off table, compose, shell syntax only
#   (default)  also checks phoenix-eval thresholds when PSQL or DATABASE_URL is set
#
# Env overrides: ACCURACY_THRESHOLD(80) P95_MS_MAX(500) MIN_EVALS(10) SKIP_PHX=1
# =============================================================================
set -uo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OFFLINE=0
if [[ "${1:-}" == "--offline" ]]; then OFFLINE=1; fi

ACCURACY_THRESHOLD="${ACCURACY_THRESHOLD:-80}"
P95_MS_MAX="${P95_MS_MAX:-500}"
MIN_EVALS="${MIN_EVALS:-10}"
SKIP_PHX="${SKIP_PHX:-0}"

fails=0
pass() { ok "$1"; }
failc() { echo "  FAIL: $1" >&2; fails=$((fails+1)); }

echo "== 14.3 Release gate + Go/No-Go =="

# --- 1. Required docs present ------------------------------------------------
for doc in docs/RELEASE_GATE.md docs/GO_NO_GO.md docs/CI_CD_NFR26.md docs/DATA_RETENTION_GDPR.md docs/TEACHER_VETTING.md docs/SUPPORT_RUNBOOK.md; do
  if [[ -f "$ROOT/$doc" ]]; then
    pass "$doc present"
  else
    failc "missing $doc"
  fi
done

# --- 2. Docs reference expected headings ------------------------------------
grep -qi "release gate" "$ROOT/docs/RELEASE_GATE.md" 2>/dev/null || failc "docs/RELEASE_GATE.md missing heading"
grep -q "Go / No-Go" "$ROOT/docs/GO_NO_GO.md" 2>/dev/null || failc "docs/GO_NO_GO.md missing heading"
grep -q "Sign-off" "$ROOT/docs/GO_NO_GO.md" 2>/dev/null || failc "docs/GO_NO_GO.md missing Sign-off table"

# --- 3. Go/No-Go sign-off table must be GO (no NO-GO, no pending) -----------
GONO="$ROOT/docs/GO_NO_GO.md"
if [[ -f "$GONO" ]]; then
  if grep -qi "NO-GO" "$GONO" | grep -qv "_GO / NO-GO_" 2>/dev/null; then :; fi
  # Count real NO-GO votes outside the template placeholder "_GO / NO-GO_"
  no_go=$(grep -i "NO-GO" "$GONO" | grep -v "_GO / NO-GO_" | grep -v "NO-GO & product" | grep -v "One.*NO-GO" | grep -v "Any axis.*NO-GO" | wc -l | tr -d ' ')
  # More robust: look for "| NO-GO" in sign-off
  no_go2=$(grep -c "| NO-GO" "$GONO" 2>/dev/null || true)
  if [[ "${no_go2:-0}" -gt 0 ]]; then
    failc "Go/No-Go sign-off contains NO-GO ($no_go2) — release blocked"
  else
    pass "Go/No-Go sign-off has no NO-GO"
  fi
  # Require at least 4 GO votes in sign-off table
  go_votes=$(grep -c "| GO" "$GONO" 2>/dev/null || true)
  if [[ "${go_votes:-0}" -lt 4 ]]; then
    failc "Go/No-Go sign-off has only $go_votes GO votes — need ≥4"
  else
    pass "Go/No-Go sign-off GO votes=$go_votes"
  fi
  pending=$(awk '/^## 3\. Sign-off/{flag=1} flag && /^\|.*_GO \/ NO-GO_/{c++} END{print c+0}' "$GONO")
  if [[ "${pending:-0}" -gt 0 ]]; then
    failc "Go/No-Go sign-off still has $pending pending placeholder(s)"
  else
    pass "Go/No-Go sign-off has no pending placeholders"
  fi
fi

# --- 4. Phase gate: marketplace + payouts + auto-promotion tasks are DONE ----
STATUS_JSON="$ROOT/crew/phase_status.json"
if [[ -f "$STATUS_JSON" ]]; then
  if command -v python3 >/dev/null 2>&1; then
    py_out=$(python3 - "$STATUS_JSON" <<'PY' 2>&1 || true
import json, sys
p=json.load(open(sys.argv[1]))
phases={x["id"]:x for x in p["phases"]}
ph4=phases.get(4)
if not ph4:
  print("missing phase 4"); sys.exit(1)
need=["12.1","12.2","12.3","12.4","12.5","12.6","12.7","13.1","14.1","14.2"]
bad=[]
for tid in need:
  t=next((t for t in ph4["tasks"] if t["id"]==tid), None)
  if not t or t.get("status")!="DONE":
    bad.append(tid+":"+ (t.get("status") if t else "MISSING"))
if bad:
  print("NOT_DONE "+",".join(bad)); sys.exit(1)
print("OK")
PY
)
    if echo "$py_out" | grep -q "^OK"; then
      pass "phase_status 12.1-14.2 are DONE"
    else
      failc "phase_status gate: $py_out"
    fi
  else
    warn "python3 not found — skipping phase_status check"
  fi
fi

# --- 5. Shell + compose validity --------------------------------------------
if bash -n "$ROOT/deploy/ci/verify-release-gate.sh" 2>/dev/null; then pass "verify-release-gate.sh syntax ok"; else failc "verify-release-gate.sh syntax error"; fi
for f in "$ROOT/deploy/ci/"*.sh "$ROOT/deploy/mail/"*.sh; do
  bash -n "$f" 2>/dev/null || failc "bash -n failed: $f"
done
if [[ -f "$ROOT/docker-compose.prod.yml" ]]; then
  if command -v docker >/dev/null 2>&1 && docker compose -f "$ROOT/docker-compose.prod.yml" config --quiet 2>/dev/null; then
    pass "docker-compose.prod.yml valid"
  else
    warn "docker not available — skipping compose validate (CI will check)"
  fi
fi
# promotion image is digest-pinned, not rebuilt
if grep -q 'IMAGE_TAG:-prod' "$ROOT/docker-compose.prod.yml" 2>/dev/null; then pass "prod compose uses IMAGE_TAG=prod promotion"; else failc "prod compose must use IMAGE_TAG=prod"; fi

# --- 6. Mail isolation (NFR-22) ----------------------------------------------
if [[ -x "$ROOT/deploy/mail/verify-isolation.sh" ]] || [[ -f "$ROOT/deploy/mail/verify-isolation.sh" ]]; then
  if bash "$ROOT/deploy/mail/verify-isolation.sh" >/tmp/vmi.log 2>&1; then
    pass "mail isolation PASS"
  else
    cat /tmp/vmi.log >&2 || true
    failc "mail isolation FAIL — see log above"
  fi
fi

# --- 7. Support runbook + dashboards (14.4) ------------------------------------
if [[ -f "$ROOT/deploy/ci/verify-support.sh" ]]; then
  if bash "$ROOT/deploy/ci/verify-support.sh" --offline >/tmp/vsup.log 2>&1; then
    pass "support runbook + dashboards PASS (verify-support.sh --offline)"
  else
    cat /tmp/vsup.log >&2 || true
    failc "support gate FAIL — see verify-support.sh log above"
  fi
else
  warn "verify-support.sh not found — skipping support gate"
fi

# --- 8. Phoenix eval thresholds (when DB reachable, else skip in offline) ---
if [[ "$OFFLINE" -eq 1 ]] || [[ "$SKIP_PHX" == "1" ]]; then
  warn "offline — skipping phoenix-eval DB check"
else
  if [[ -n "${PSQL:-}" ]] || [[ -n "${DATABASE_URL:-}" ]]; then
    if [[ -f "$ROOT/deploy/ci/phoenix-eval.sh" ]]; then
      if PSQL="${PSQL:-}" DATABASE_URL="${DATABASE_URL:-}" ACCURACY_THRESHOLD="$ACCURACY_THRESHOLD" P95_MS_MAX="$P95_MS_MAX" MIN_EVALS="$MIN_EVALS" bash "$ROOT/deploy/ci/phoenix-eval.sh" 2>&1 | tee /tmp/phx.log; then
        pass "phoenix-eval thresholds met"
      else
        cat /tmp/phx.log >&2 || true
        failc "phoenix-eval thresholds not met"
      fi
    fi
  else
    warn "no PSQL/DATABASE_URL — skipping phoenix-eval DB check (set PSQL to gate on DB)"
  fi
fi

# --- 9. Wireframe parity (MKT-QA) -----------------------------------------------
if [[ -f "$ROOT/deploy/ci/verify-wireframe-parity.sh" ]]; then
  if bash "$ROOT/deploy/ci/verify-wireframe-parity.sh" >/tmp/vwp.log 2>&1; then
    pass "wireframe parity PASS (verify-wireframe-parity.sh)"
  else
    cat /tmp/vwp.log >&2 || true
    failc "wireframe parity FAIL — see verify-wireframe-parity.sh log above"
  fi
else
  warn "verify-wireframe-parity.sh not found — skipping wireframe parity gate"
fi

# --- 10. Go vet (fast, no DB) -------------------------------------------------
if command -v go >/dev/null 2>&1; then
  if (cd "$ROOT/backend" && go vet ./... 2>&1 | tee /tmp/govet.log); then
    pass "go vet ./... pass"
  else
    cat /tmp/govet.log >&2 || true
    failc "go vet failed"
  fi
fi

# --- Summary -----------------------------------------------------------------
echo
if [[ "$fails" -ne 0 ]]; then
  echo "Release gate: FAIL ($fails check(s) failed) — Go/No-Go blocked." >&2
  exit 1
fi
echo "Release gate: PASS."
exit 0
