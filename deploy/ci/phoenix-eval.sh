#!/usr/bin/env bash
# Gate: translation/grammar quality (golden-set eval via Phoenix / FR-30 data).
#
# The FR-30 cross-model evaluator records per-translation accuracy in
# `translation_evals` and per-job latency in `translation_jobs`. This gate
# asserts the quality thresholds the release requires: golden-set accuracy
# above a floor and p95 latency below a budget on the evaluated set.
#
# Reads the metrics directly from the dev Postgres. The "cache-hit set" is
# preferred for the latency budget; if the dev env has not yet warmed the cache
# we fall back to all completed jobs with a warning so a cold set does not block
# promotion.
#
# Required env: DATABASE_URL (dev DB) unless PSQL provides a container exec path.
#   Optional: ACCURACY_THRESHOLD (80), P95_MS_MAX (500), MIN_EVALS (10),
#   P95_CACHE_HIT_ONLY (true), PSQL (override psql invocation, e.g.
#   "docker compose exec -T postgres-dev psql -U chorus_dev -d chorus_dev").
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

ACCURACY_THRESHOLD="${ACCURACY_THRESHOLD:-80}"
P95_MS_MAX="${P95_MS_MAX:-500}"
MIN_EVALS="${MIN_EVALS:-10}"
P95_CACHE_HIT_ONLY="${P95_CACHE_HIT_ONLY:-true}"

# Build the psql invocation. Default: psql against DATABASE_URL. If PSQL is set
# (e.g. to exec into the container on the dev host), use that.
if [ -n "${PSQL:-}" ]; then
  PFX=()
  read -r -a PFX <<< "$PSQL"
else
  req DATABASE_URL
  PFX=(psql "$DATABASE_URL")
fi
q() { "${PFX[@]}" -tA -v ON_ERROR_STOP=1 "$@"; }

avg_acc=$(q -c "SELECT COALESCE((SELECT ROUND(AVG(accuracy_score),1) FROM translation_evals WHERE status='done'),-1);")
eval_cnt=$(q -c "SELECT COUNT(*)::int FROM translation_evals WHERE status='done';")

# p95 latency over the cache-hit set when requested, else over all done jobs.
p95=""
if is_true "$P95_CACHE_HIT_ONLY"; then
  p95_cache=$(q -c "SELECT COALESCE((SELECT percentile_cont(0.95) WITHIN GROUP (ORDER BY latency_ms)
        FROM translation_jobs WHERE status='done' AND cache_hit AND latency_ms IS NOT NULL),-1);")
  if [ "${p95_cache:--1}" = "-1" ]; then
    warn "no cache-hit latency samples yet — using all completed jobs"
  else
    p95="$p95_cache"
  fi
fi
if [ -z "$p95" ]; then
  p95=$(q -c "SELECT COALESCE((SELECT percentile_cont(0.95) WITHIN GROUP (ORDER BY latency_ms)
        FROM translation_jobs WHERE status='done' AND latency_ms IS NOT NULL),-1);")
fi

log "Golden-set eval: accuracy=$avg_acc, eval_count=$eval_cnt, p95=${p95}ms"

[ "${eval_cnt:-0}" -ge "$MIN_EVALS" ] || fail "golden set too small: $eval_cnt evals < $MIN_EVALS"
float_ge "$avg_acc" "$ACCURACY_THRESHOLD" || fail "accuracy $avg_acc < threshold $ACCURACY_THRESHOLD"
float_lt "$p95" "$P95_MS_MAX" || fail "p95 latency ${p95}ms exceeds budget ${P95_MS_MAX}ms"

ok "accuracy=$avg_acc (>= $ACCURACY_THRESHOLD), p95=${p95}ms (< $P95_MS_MAX), evals=$eval_cnt"
