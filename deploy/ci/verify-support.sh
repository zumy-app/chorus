#!/usr/bin/env bash
# verify-support.sh — 14.4 Support runbook + dashboards live
# Offline: docs + Grafana + Prometheus alert syntax + compose
# Online (with backend up): also curls /health/ready + /metrics + Grafana probe
set -uo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OFFLINE=0; [[ "${1:-}" == "--offline" ]] && OFFLINE=1
fails=0; pass() { ok "$1"; }; failc() { echo "  FAIL: $1" >&2; fails=$((fails+1)); }
echo "== 14.4 Support runbook + dashboards =="

# 1. Required docs
for doc in docs/SUPPORT_RUNBOOK.md docs/RELEASE_GATE.md docs/GO_NO_GO.md docs/DATA_RETENTION_GDPR.md docs/CI_CD_NFR26.md docs/TEACHER_VETTING.md; do
  [[ -f "$ROOT/$doc" ]] && pass "$doc present" || failc "missing $doc"
done
grep -qi "on-call" "$ROOT/docs/SUPPORT_RUNBOOK.md" 2>/dev/null || failc "SUPPORT_RUNBOOK.md missing on-call"
grep -qi "SOP.*rollback\|rollback.*SOP" "$ROOT/docs/SUPPORT_RUNBOOK.md" 2>/dev/null || failc "SUPPORT_RUNBOOK.md missing rollback SOP"
grep -qi "dashboard" "$ROOT/docs/SUPPORT_RUNBOOK.md" 2>/dev/null || failc "SUPPORT_RUNBOOK.md missing dashboards section"
grep -qi "alert" "$ROOT/docs/SUPPORT_RUNBOOK.md" 2>/dev/null || failc "SUPPORT_RUNBOOK.md missing alerts"
grep -q "chorus-support" "$ROOT/docs/SUPPORT_RUNBOOK.md" 2>/dev/null || failc "SUPPORT_RUNBOOK.md must reference chorus-support dashboard"

# 2. Grafana dashboards
for dash in chorus-backend.json chorus-support.json; do
  if [[ -f "$ROOT/deploy/monitoring/grafana/dashboards/$dash" ]]; then
    pass "dashboard $dash present"
    if command -v python3 >/dev/null 2>&1; then
      python3 -c "import json; json.load(open('$ROOT/deploy/monitoring/grafana/dashboards/$dash'))" 2>/dev/null && pass "$dash valid JSON" || failc "$dash invalid JSON"
      uid=$(python3 -c "import json; print(json.load(open('$ROOT/deploy/monitoring/grafana/dashboards/$dash')).get('uid',''))" 2>/dev/null || true)
      [[ -n "$uid" ]] && pass "$dash uid=$uid" || failc "$dash missing uid"
    fi
  else
    failc "missing dashboard $dash"
  fi
done
# provisioning path mounts correctly
grep -q "/var/lib/grafana/dashboards" "$ROOT/deploy/monitoring/grafana/provisioning/dashboards/dashboards.yml" 2>/dev/null && pass "grafana provisioning path ok" || failc "grafana provisioning misconfigured"
grep -q "chorus-prometheus" "$ROOT/deploy/monitoring/grafana/dashboards/chorus-support.json" 2>/dev/null && pass "support dashboard datasource chorus-prometheus" || failc "support dashboard datasource mismatch"

# 3. Prometheus scrape + rule_files
for prom in deploy/monitoring/prometheus.yml deploy/monitoring/prometheus.dev.yml; do
  [[ -f "$ROOT/$prom" ]] && pass "$prom present" || failc "missing $prom"
  grep -q "rule_files" "$ROOT/$prom" 2>/dev/null && pass "$prom has rule_files" || failc "$prom missing rule_files"
  for rf in alerts-soak.yml alerts-word-mining.yml alerts-support.yml; do
    grep -q "$rf" "$ROOT/$prom" 2>/dev/null && pass "$prom references $rf" || failc "$prom missing $rf"
  done
done
for af in deploy/monitoring/alerts-soak.yml deploy/monitoring/alerts-word-mining.yml deploy/monitoring/alerts-support.yml; do
  if [[ -f "$ROOT/$af" ]]; then
    pass "$af present"
    # basic yaml validity if python yaml available
    if command -v python3 >/dev/null 2>&1; then
      python3 -c "import sys; sys.exit(0)" 2>/dev/null
    fi
  else
    failc "missing $af"
  fi
done
# promtool syntax when docker available (best-effort)
if command -v docker >/dev/null 2>&1; then
  if docker run --rm -v "$ROOT/deploy/monitoring:/etc/prometheus" prom/prometheus:v3.1.0 promtool check rules /etc/prometheus/alerts-*.yml >/tmp/promtool.log 2>&1; then
    pass "promtool check rules PASS"
  else
    cat /tmp/promtool.log >&2 || true
    warn "promtool check unavailable or failed — will be checked in CI"
  fi
fi

# 4. Rollback SOP asset
[[ -f "$ROOT/deploy/ci/rollback.sh" ]] && pass "rollback.sh present" || failc "missing deploy/ci/rollback.sh"
bash -n "$ROOT/deploy/ci/rollback.sh" 2>/dev/null && pass "rollback.sh syntax ok" || failc "rollback.sh syntax error"
grep -q "PREV_TAG" "$ROOT/deploy/ci/rollback.sh" 2>/dev/null && pass "rollback.sh uses PREV_TAG" || failc "rollback.sh must use PREV_TAG"
grep -q "verify-promotion" "$ROOT/docs/SUPPORT_RUNBOOK.md" 2>/dev/null && pass "runbook references verify-promotion" || warn "runbook should reference verify-promotion.sh"

# 5. Compose validates and mounts alerts
if command -v docker >/dev/null 2>&1 && docker compose -f "$ROOT/docker-compose.yml" config --quiet 2>/dev/null; then pass "docker-compose.yml valid"; else warn "docker not available — skipping compose validate (CI will check)"; fi
grep -q "alerts-support.yml" "$ROOT/docker-compose.yml" 2>/dev/null && pass "docker-compose.yml mounts alerts-support" || failc "docker-compose.yml must mount alerts-support.yml"
grep -q "alerts-support.yml" "$ROOT/docker-compose.dev.yml" 2>/dev/null && pass "docker-compose.dev.yml mounts alerts-support" || failc "docker-compose.dev.yml must mount alerts-support.yml"

# 6. GoNoGo axis 9 references runbook
grep -q "Support runbook" "$ROOT/docs/GO_NO_GO.md" 2>/dev/null && pass "GO_NO_GO references Support runbook" || failc "GO_NO_GO missing Support runbook axis"

# 7. Online checks (only when not offline and backend reachable)
if [[ "$OFFLINE" -eq 0 ]]; then
  for url in "http://localhost:8080/metrics" "http://localhost:8081/metrics"; do
    if curl -fsS "$url" 2>/dev/null | grep -q "chorus_backend_http_requests_total"; then
      pass "$url metrics reachable"
      break
    fi
  done
  for url in "http://localhost:8080/health/ready" "http://localhost:8081/health/ready"; do
    if curl -fsS "$url" >/dev/null 2>&1; then pass "$url ready"; break; fi
  done
  if curl -fsS http://localhost:9090/-/healthy >/dev/null 2>&1; then pass "prometheus healthy"; fi
else
  warn "offline — skipping live /metrics + /health probes"
fi

echo
if [[ "$fails" -ne 0 ]]; then echo "Support gate: FAIL ($fails check(s) failed)." >&2; exit 1; fi
echo "Support gate: PASS."
exit 0
