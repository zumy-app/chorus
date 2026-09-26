#!/usr/bin/env bash
# Rollback prod to a previous image tag (NFR-26).
# Re-promotes PREV_TAG to :prod and redeploys prod if PROD_HOST is reachable.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

req CR_REGISTRY IMAGE_NAME CR_USERNAME CR_PASSWORD
PREV_TAG="${PREV_TAG:-}"
PROD_TAG="${PROD_TAG:-prod}"
[ -n "$PREV_TAG" ] || fail "PREV_TAG is required (e.g. PREV_TAG=123 bash deploy/ci/rollback.sh)"

echo "$CR_PASSWORD" | docker login "$CR_REGISTRY" -u "$CR_USERNAME" --password-stdin

for svc in backend frontend; do
  src="$IMAGE_NAME/$svc:$PREV_TAG"
  dst="$IMAGE_NAME/$svc:$PROD_TAG"
  log "rolling back $dst -> $src"
  docker buildx imagetools create --tag "$dst" "$src"
  ok "rolled back $dst to $PREV_TAG"
done

if [ -n "${PROD_HOST:-}" ] && [ -n "${PROD_USER:-}" ]; then
  log "redeploying prod at $PROD_HOST"
  ssh "$PROD_USER@$PROD_HOST" "cd /opt/chorus && IMAGE_TAG=$PROD_TAG IMAGE_NAME=$IMAGE_NAME docker compose -f docker-compose.prod.yml pull && IMAGE_TAG=$PROD_TAG IMAGE_NAME=$IMAGE_NAME docker compose -f docker-compose.prod.yml up -d"
  ok "prod redeployed to $PREV_TAG"
fi
