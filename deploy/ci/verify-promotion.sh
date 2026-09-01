#!/usr/bin/env bash
# Gate: verify dev→prod image promotion succeeded and prod is healthy (NFR-26).
# Checks the registry digests are reachable and the prod LB answers /health.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

PROD_URL="${PROD_URL:-https://chorus.talk}"
CR_REGISTRY="${CR_REGISTRY:-ghcr.io}"
IMAGE_NAME="${IMAGE_NAME:-ghcr.io/zumy-app/chorus}"
DEV_TAG="${DEV_TAG:-}"
PROD_TAG="${PROD_TAG:-prod}"

if [ -n "$DEV_TAG" ] && command -v docker >/dev/null 2>&1; then
  for svc in backend frontend; do
    src="$IMAGE_NAME/$svc:$DEV_TAG"
    dst="$IMAGE_NAME/$svc:$PROD_TAG"
    log "inspecting $dst"
    if docker buildx imagetools inspect "$dst" >/dev/null 2>&1; then
      ok "registry has $dst"
    else
      fail "registry missing $dst (promotion failed)"
    fi
  done
fi

log "probing prod health at $PROD_URL/health"
for i in $(seq 1 30); do
  if curl -fsS "$PROD_URL/health" >/dev/null 2>&1; then
    ok "prod healthy at $PROD_URL/health"
    exit 0
  fi
  if curl -fsS "$PROD_URL/health/ready" >/dev/null 2>&1; then
    ok "prod ready at $PROD_URL/health/ready"
    exit 0
  fi
  sleep 5
done
fail "prod health check failed at $PROD_URL/health after 150s"
