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

verify_digest() {
  local src="$1" dst="$2"
  if command -v docker >/dev/null 2>&1; then
    src_dgst=$(docker buildx imagetools inspect "$src" --raw 2>/dev/null | sha256sum 2>/dev/null | awk '{print $1}' || true)
    dst_dgst=$(docker buildx imagetools inspect "$dst" --raw 2>/dev/null | sha256sum 2>/dev/null | awk '{print $1}' || true)
    if [ -n "$src_dgst" ] && [ -n "$dst_dgst" ]; then
      if [ "$src_dgst" != "$dst_dgst" ]; then
        warn "digest mismatch after promotion: $src ($src_dgst) != $dst ($dst_dgst)"
      else
        log "digest verified: $dst == $src ($src_dgst)"
      fi
    fi
  fi
}

for svc in backend frontend; do
  src="$IMAGE_NAME/$svc:$DEV_TAG"
  dst="$IMAGE_NAME/$svc:$PROD_TAG"
  log "promoting $src -> $dst"
  docker buildx imagetools create --tag "$dst" "$src"
  ok "promoted $dst"
  verify_digest "$src" "$dst"
done

log "Released: $IMAGE_NAME/*:$PROD_TAG from $DEV_TAG"
