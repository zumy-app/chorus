#!/usr/bin/env bash
# Image promotion (NFR-26): copy a built `:dev` snapshot to `:prod` WITHOUT a
# rebuild. Uses `docker buildx imagetools create`, which copies the manifest
# (and exact digest) between registry refs without re-downloading/building layer
# blobs — true image promotion, not a rebuild.
#
# Required env: CR_REGISTRY, IMAGE_NAME (org/repo), DEV_TAG (source snapshot,
#   e.g. the GitHub run number), CR_USERNAME, CR_PASSWORD (token with pull/push
#   on the registry). Optional: PROD_TAG (prod).
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

req CR_REGISTRY IMAGE_NAME DEV_TAG CR_USERNAME CR_PASSWORD
PROD_TAG="${PROD_TAG:-prod}"

echo "$CR_PASSWORD" | docker login "$CR_REGISTRY" -u "$CR_USERNAME" --password-stdin

for svc in backend frontend; do
  src="$IMAGE_NAME/$svc:$DEV_TAG"
  dst="$IMAGE_NAME/$svc:$PROD_TAG"
  log "promoting $src -> $dst"
  docker buildx imagetools create --tag "$dst" "$src"
  ok "promoted $dst"
done

log "Released: $IMAGE_NAME/*:$PROD_TAG from $DEV_TAG"
