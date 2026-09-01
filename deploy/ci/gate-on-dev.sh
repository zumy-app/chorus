#!/usr/bin/env bash
# Orchestrator: run ALL NFR-26 dev quality gates on the dev host.
# Invoked by `.github/workflows/ci.yml` in the `deploy-dev` job AFTER the dev
# stack is up. Runs against the dev LB and reaches the dev Postgres through the
# compose network (psql exec), so it works from the runner that deployed the
# stack or directly on the dev host. Fails fast on the first red gate.
#
# Required env: E2E_BASE_URL (dev LB, default https://dev.chorus.talk).
#   Optional: DOCKER_COMPOSE_FILE (deploy/dev/docker-compose.yml),
#   GATES (space-separated subset, default "seed e2e phoenix-eval load-smoke").
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

BASE_URL="${E2E_BASE_URL:-https://dev.chorus.talk}"
COMPOSE_FILE="${DOCKER_COMPOSE_FILE:-deploy/dev/docker-compose.yml}"
CI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATES="${GATES:-seed e2e phoenix-eval load-smoke}"

export E2E_BASE_URL="$BASE_URL"

# Route phoenix-eval's psql into the postgres-dev container so the DB is never
# published to the host. Unquoted so read -a in phoenix-eval.sh splits cleanly.
if [[ " $GATES " == *" phoenix-eval "* ]]; then
  export PSQL="docker compose -f $COMPOSE_FILE exec -T postgres-dev psql -U chorus_dev -d chorus_dev"
fi

for gate in $GATES; do
  case "$gate" in
    seed)         script="seed-dev.sh" ;;
    e2e)          script="e2e.sh" ;;
    phoenix-eval) script="phoenix-eval.sh" ;;
    load-smoke)   script="load-smoke.sh" ;;
    load-soak)    script="load-soak.sh" ;;
    verify-drain) script="verify-drain.sh" ;;
    *)            warn "unknown gate '$gate' skipped" ; continue ;;
  esac
  log "===== GATE: $gate ====="
  "$CI_DIR/$script" || fail "gate '$gate' failed"
  ok "gate '$gate' passed"
done

ok "all dev gates passed ($GATES) on $BASE_URL"
