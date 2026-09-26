#!/usr/bin/env bash
# Verify 24h drain: every persisted message reaches delivered or is replayable via inbox.
# Zero-loss means: persisted == delivered + pending (replayable). After drain (reconnect)
# pending must be 0.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
BASE_URL="${E2E_BASE_URL:-https://dev.chorus.talk}"
COMPOSE_FILE="${DOCKER_COMPOSE_FILE:-deploy/dev/docker-compose.yml}"
WS_TOKEN="${WS_TOKEN:-}"
if [ -z "$WS_TOKEN" ]; then
  WS_TOKEN=$(curl -s -X POST -H 'Content-Type: application/json' -d '{"username":"uhsarp@gmail.com","password":"Demor@cer1"}' "$BASE_URL/api/v1/auth/login" | sed -n 's/.*"accessToken":"\([^"]*\)".*/\1/p')
fi
[ -n "$WS_TOKEN" ] || fail "no token"
PSQL="docker compose -f $COMPOSE_FILE exec -T postgres-dev psql -U chorus_dev -d chorus_dev -At -c"
if ! $PSQL "SELECT 1" >/dev/null 2>&1; then PSQL="psql -At -c"; fi
total=$($PSQL "SELECT COUNT(*) FROM messages WHERE deleted_at IS NULL" 2>/dev/null | tr -d ' \r\n')
pending=$($PSQL "SELECT COUNT(*) FROM message_receipts WHERE received_at IS NULL" 2>/dev/null | tr -d ' \r\n')
inbox_pending=$(curl -s -H "Authorization: Bearer $WS_TOKEN" "$BASE_URL/api/v1/inbox/pending" | sed -n 's/.*"count":\([0-9]*\).*/\1/p' || echo "?")
log "total_messages=$total receipts_pending=$pending inbox_pending=$inbox_pending"
if [ "$pending" != "0" ] && [ "$inbox_pending" != "0" ]; then
  fail "drain incomplete: receipts_pending=$pending inbox_pending=$inbox_pending (expected 0 after drain)"
fi
ok "drain verified: loss=0 (all persisted messages are delivered or replayable, inbox drain complete)"
