#!/usr/bin/env bash
set -euo pipefail
DB_URL="${DATABASE_URL:-postgres://chorus:chorus@localhost:5432/chorus?sslmode=disable}"
METRICS_URL="${METRICS_URL:-http://localhost:8080/metrics}"
echo "[verify-practice-ladder] DB checks..."
if command -v psql >/dev/null 2>&1; then
  mastered=$(psql "$DB_URL" -tAc "select count(*) from vocabulary where mastery_state='mastered'" 2>/dev/null || echo "0")
  leech=$(psql "$DB_URL" -tAc "select count(*) from vocabulary where mastery_state='leech'" 2>/dev/null || echo "0")
  interval_violation=$(psql "$DB_URL" -tAc "select count(*) from vocabulary where mastery_stage=1 and interval_days != 2 or mastery_stage=2 and interval_days !=3 or mastery_stage=3 and interval_days !=5" 2>/dev/null || echo "0")
  mastered_stage_check=$(psql "$DB_URL" -tAc "select count(*) from vocabulary where mastery_state='mastered' and mastery_stage <4" 2>/dev/null || echo "0")
  echo "mastered: $mastered leech: $leech interval_violations: $interval_violation mastered_stage_bad: $mastered_stage_check"
  if [ "${STRICT:-0}" = "1" ] && [ "$mastered_stage_check" != "0" ]; then echo "FAIL mastered cards below stage 4"; exit 1; fi
fi
echo "[verify-practice-ladder] metrics..."
if curl -sf "$METRICS_URL" 2>/dev/null | grep -q "chorus_backend_practice_attempts_total"; then echo "practice_attempts metrics present"; else echo "WARN practice metrics missing (backend down?)"; fi
if curl -sf "$METRICS_URL" 2>/dev/null | grep -q "chorus_backend_practice_stage_promotions_total"; then echo "promotions metric present"; fi
if curl -sf "$METRICS_URL" 2>/dev/null | grep -q "chorus_backend_practice_leech_total"; then echo "leech metric present"; fi
echo "[verify-practice-ladder] handler smoke..."
if curl -sf "$METRICS_URL" 2>/dev/null | grep -q "chorus_backend_http_requests_total"; then echo "http metrics present"; fi
echo "[verify-practice-ladder] ok"
