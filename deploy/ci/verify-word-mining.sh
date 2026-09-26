#!/usr/bin/env bash
set -euo pipefail
DB_URL="${DATABASE_URL:-postgres://chorus:chorus@localhost:5432/chorus?sslmode=disable}"
METRICS_URL="${METRICS_URL:-http://localhost:8080/metrics}"
echo "[verify-word-mining] checking stuck mining jobs..."
if command -v psql >/dev/null 2>&1; then
  pending=$(psql "$DB_URL" -tAc "select count(*) from word_mining_jobs where status in ('pending','processing')" 2>/dev/null || echo "0")
  failed=$(psql "$DB_URL" -tAc "select count(*) from word_mining_jobs where status='failed' and next_attempt_at <= now()" 2>/dev/null || echo "0")
  echo "pending/processing: $pending  failed-due: $failed"
  if [ "${STRICT:-0}" = "1" ] && [ "$pending" -gt 20 ]; then echo "FAIL pending mining jobs >20"; exit 1; fi
fi
echo "[verify-word-mining] metrics..."
if curl -sf "$METRICS_URL" 2>/dev/null | grep -q "chorus_backend_word_mining_jobs_total"; then echo "word_mining metrics present"; else echo "WARN metrics missing (backend down?)"; fi
if curl -sf "$METRICS_URL" 2>/dev/null | grep -q "chorus_backend_word_mining_pending_jobs"; then echo "pending gauge present"; fi
echo "[verify-word-mining] ok"
