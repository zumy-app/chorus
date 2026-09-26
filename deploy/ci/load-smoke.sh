#!/usr/bin/env bash
# Gate: WebSocket load smoke (NFR-26) — 100 concurrent connections, ~10 msg/s,
# 5 minutes. Drives the deployed dev stack with Artillery and fails on an
# error rate above the threshold (message loss / aborted sockets).
#
# Required env: E2E_BASE_URL (dev LB, default https://dev.chorus.talk).
#   Optional: WS_TOKEN (pre-fetched access token; if empty we log in ourselves),
#   ARTILLERY_MAX_ERROR_RATE (0.02 = 2%).
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

BASE_URL="${E2E_BASE_URL:-https://dev.chorus.talk}"
ARTILLERY_MAX_ERROR_RATE="${ARTILLERY_MAX_ERROR_RATE:-0.02}"
WS_TOKEN="${WS_TOKEN:-}"

# Derive the WS endpoint from the dev LB origin (http→ws, https→wss).
WS_URL="${WS_URL:-$(printf '%s' "$BASE_URL" | sed -E 's#^http#ws#')/ws}"

if [ -z "$WS_TOKEN" ]; then
  log "No WS_TOKEN provided — logging in with the seeded dev user"
  WS_TOKEN=$(curl -s -X POST -H 'Content-Type: application/json' \
    -d '{"username":"uhsarp@gmail.com","password":"Demor@cer1"}' \
    "$BASE_URL/api/v1/auth/login" | sed -n 's/.*"accessToken":"\([^"]*\)".*/\1/p')
  [ -n "$WS_TOKEN" ] || fail "could not obtain access token from $BASE_URL/api/v1/auth/login"
fi

log "Running Artillery WS smoke: 100 concurrent / 10 msg/s / 5m"
# The YAML `ensure.thresholds` (errors.rate < max) is the primary gate; a
# non-zero exit here fails the gate (set -e). We keep a JSON report for review.
npx --yes artillery run \
  --target "$WS_URL" \
  --output /tmp/artillery-report.json \
  "$(dirname "${BASH_SOURCE[0]}")/artillery.load.yml"

# Best-effort summary (not the gate — artillery already enforced the threshold).
if command -v jq >/dev/null 2>&1 && [ -f /tmp/artillery-report.json ]; then
  ERR=$(jq -r '.aggregate.errors.rate // 0' /tmp/artillery-report.json 2>/dev/null || echo "n/a")
  SCEN=$(jq -r '.aggregate.scenariosCompleted // "n/a"' /tmp/artillery-report.json 2>/dev/null)
  log "scenariosCompleted=$SCEN errorRate=$ERR"
fi

ok "load smoke passed (WS_URL=$WS_URL)"
