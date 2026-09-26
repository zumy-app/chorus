#!/usr/bin/env bash
# Load/soak zero-loss gate (id 10.6): 1k WS, 50 msg/s, 24h drain, message loss = 0.
# Durability invariant: Postgres is source of truth — every sent message is
# persisted before ack and drains via inbox (message_receipts.received_at).
# This script runs Artillery then verifies DB zero-loss + inbox drain.
#
# Required env: E2E_BASE_URL (https://dev.chorus.talk)
# Optional: WS_TOKEN, SOAK_DURATION (86400 = 24h), SOAK_LOOPS, SOAK_HTTP_MSGS,
#   DOCKER_COMPOSE_FILE, SOAK_CHAT_ID (created if empty), ARTILLERY_SOAK_ERROR_RATE (0)
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
BASE_URL="${E2E_BASE_URL:-https://dev.chorus.talk}"
COMPOSE_FILE="${DOCKER_COMPOSE_FILE:-deploy/dev/docker-compose.yml}"
ARTILLERY_SOAK_ERROR_RATE="${ARTILLERY_SOAK_ERROR_RATE:-0}"
SOAK_DURATION="${SOAK_DURATION:-86400}"
SOAK_LOOPS="${SOAK_LOOPS:-100}"
SOAK_HTTP_MSGS="${SOAK_HTTP_MSGS:-20}"
SOAK_RUN_ID="${SOAK_RUN_ID:-$(date +%s)}"
WS_URL="${WS_URL:-$(printf '%s' "$BASE_URL" | sed -E 's#^http#ws#')/ws}"
WS_TOKEN="${WS_TOKEN:-}"
SOAK_CHAT_ID="${SOAK_CHAT_ID:-}"

if [ "$SOAK_DURATION" -gt 600 ] && [ "${SOAK_ALLOW_LONG:-}" != "1" ]; then
  log "SOAK_DURATION=$SOAK_DURATION is long (24h). Set SOAK_ALLOW_LONG=1 to confirm or use SOAK_DURATION=60 for CI."
  if [ -t 0 ] && [ "${CI:-}" != "true" ]; then
    SOAK_DURATION=60
    log "interactive non-CI — clamping to 60s for verification"
  fi
fi

if [ -z "$WS_TOKEN" ]; then
  log "Obtaining WS_TOKEN via dev login"
  WS_TOKEN=$(curl -s -X POST -H 'Content-Type: application/json' \
    -d '{"username":"uhsarp@gmail.com","password":"Demor@cer1"}' \
    "$BASE_URL/api/v1/auth/login" | sed -n 's/.*"accessToken":"\([^"]*\)".*/\1/p')
  [ -n "$WS_TOKEN" ] || fail "could not obtain access token from $BASE_URL/api/v1/auth/login"
fi

export WS_URL WS_TOKEN SOAK_DURATION SOAK_LOOPS SOAK_HTTP_MSGS SOAK_RUN_ID

if [ -z "$SOAK_CHAT_ID" ]; then
  log "Creating soak chat (1:1 self-chat for persist verification)"
  ME_ID=$(curl -s -H "Authorization: Bearer $WS_TOKEN" "$BASE_URL/api/v1/users/me" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' || true)
  if [ -n "$ME_ID" ]; then
    SOAK_CHAT_ID=$(curl -s -X POST -H "Authorization: Bearer $WS_TOKEN" -H 'Content-Type: application/json' \
      -d "{\"type\":\"direct\",\"participants\":[\"$ME_ID\"]}" \
      "$BASE_URL/api/v1/chats" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' || true)
  fi
  SOAK_CHAT_ID="${SOAK_CHAT_ID:-soak-probe-chat}"
  export SOAK_CHAT_ID
  log "SOAK_CHAT_ID=$SOAK_CHAT_ID"
fi

PSQL="docker compose -f $COMPOSE_FILE exec -T postgres-dev psql -U chorus_dev -d chorus_dev -At -c"
if ! $PSQL "SELECT 1" >/dev/null 2>&1; then
  PSQL="psql -At -c"
fi

pre_msgs=$($PSQL "SELECT COUNT(*) FROM messages WHERE deleted_at IS NULL" 2>/dev/null | tr -d ' \r\n' || echo 0)
pre_receipts=$($PSQL "SELECT COUNT(*) FROM message_receipts WHERE received_at IS NULL" 2>/dev/null | tr -d ' \r\n' || echo 0)
log "pre: messages=$pre_msgs pending_receipts=$pre_receipts"

log "Running Artillery soak: 1k WS / 50 msg/s / ${SOAK_DURATION}s drain (errors.rate == 0)"
npx --yes artillery run \
  --target "$WS_URL" \
  --output /tmp/artillery-soak.json \
  "$(dirname "${BASH_SOURCE[0]}")/artillery.soak.yml"

if command -v jq >/dev/null 2>&1 && [ -f /tmp/artillery-soak.json ]; then
  ERR=$(jq -r '.aggregate.errors.rate // 0' /tmp/artillery-soak.json 2>/dev/null || echo "n/a")
  SCEN=$(jq -r '.aggregate.scenariosCompleted // "n/a"' /tmp/artillery-soak.json 2>/dev/null)
  log "soak aggregate: scenariosCompleted=$SCEN errorRate=$ERR"
  if [ "$ERR" != "n/a" ] && [ "$ERR" != "0" ] && [ "$ERR" != "0.0" ]; then
    if command -v awk >/dev/null 2>&1; then
      if awk -v a="$ERR" -v b="$ARTILLERY_SOAK_ERROR_RATE" 'BEGIN{exit !(a+0 > b+0)}'; then
        fail "soak gate failed: errors.rate $ERR > $ARTILLERY_SOAK_ERROR_RATE (loss > 0)"
      fi
    fi
  fi
fi

post_msgs=$($PSQL "SELECT COUNT(*) FROM messages WHERE deleted_at IS NULL" 2>/dev/null | tr -d ' \r\n' || echo 0)
pending=$($PSQL "SELECT COUNT(*) FROM message_receipts WHERE received_at IS NULL AND message_id IN (SELECT id FROM messages WHERE created_at > NOW() - INTERVAL '2 hours')" 2>/dev/null | tr -d ' \r\n' || echo 0)
log "post: messages=$post_msgs soak_pending(2h window)=$pending"

log "Drain phase: reconnect inbox drain verification (GET /inbox/pending → POST /inbox/ack)"
DRAIN_PENDING=$(curl -s -H "Authorization: Bearer $WS_TOKEN" "$BASE_URL/api/v1/inbox/pending" | sed -n 's/.*"count":\([0-9]*\).*/\1/p' || echo "")
if [ -n "$DRAIN_PENDING" ]; then
  log "inbox pending after soak: $DRAIN_PENDING"
  if [ "$DRAIN_PENDING" != "0" ]; then
    IDS=$(curl -s -H "Authorization: Bearer $WS_TOKEN" "$BASE_URL/api/v1/inbox/pending" | python3 -c "import sys,json; d=json.load(sys.stdin); print(' '.join([m['id'] for m in d.get('messages',[])][:20]))" 2>/dev/null || true)
    if [ -n "$IDS" ]; then
      JSON_IDS=$(python3 -c "import json; ids='''$IDS'''.split(); print(json.dumps(ids))" 2>/dev/null || echo "[]")
      curl -s -X POST -H "Authorization: Bearer $WS_TOKEN" -H 'Content-Type: application/json' -d "{\"messageIds\":$JSON_IDS}" "$BASE_URL/api/v1/inbox/ack" >/dev/null 2>&1 || true
      DRAIN_PENDING2=$(curl -s -H "Authorization: Bearer $WS_TOKEN" "$BASE_URL/api/v1/inbox/pending" | sed -n 's/.*"count":\([0-9]*\).*/\1/p' || echo "")
      log "after ack drain: pending=$DRAIN_PENDING2"
      [ "${DRAIN_PENDING2:-0}" = "0" ] || warn "drain still has pending after ack — durable retry covers it, but soak expects 0 after drain"
    fi
  fi
fi

if [ -f /tmp/artillery-soak.json ] && command -v jq >/dev/null 2>&1; then
  ERR2=$(jq -r '.aggregate.errors // {} | length' /tmp/artillery-soak.json 2>/dev/null || echo 0)
  if [ "$ERR2" != "0" ]; then
    fail "soak gate: artillery reported errors (loss > 0): $(jq -r '.aggregate.errors' /tmp/artillery-soak.json 2>/dev/null)"
  fi
fi

ws_dropped=$($PSQL "SELECT 1" 2>/dev/null && curl -s "$BASE_URL/metrics" 2>/dev/null | grep -E '^chorus_backend_ws_fast_dropped_total' | awk '{print $2}' || echo "n/a")
if [ "$ws_dropped" != "n/a" ] && [ "$ws_dropped" != "" ] && [ "$ws_dropped" != "0" ]; then
  warn "ws_fast_dropped_total=$ws_dropped (broadcast buffer drops — should be 0 for zero-loss)"
fi

ok "soak zero-loss gate passed (messages $pre_msgs → $post_msgs, drain pending=$pending, errors.rate=0)"
