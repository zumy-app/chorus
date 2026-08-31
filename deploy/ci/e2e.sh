#!/usr/bin/env bash
# Gate: e2e (Playwright) against the deployed dev stack (NFR-26).
#
# Runs the full e2e suite in `e2e/` against the dev LB. We set E2E_SKIP_STARTUP
# so the Playwright global-setup does NOT try to spawn a local Docker stack, and
# point the health/frontend probes at the dev LB (which the Caddyfile proxies).
#
# Required env: E2E_BASE_URL (dev LB, default https://dev.chorus.talk).
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

BASE_URL="${E2E_BASE_URL:-https://dev.chorus.talk}"
E2E_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../e2e" && pwd)"

log "Running Playwright e2e against $BASE_URL"
pushd "$E2E_DIR" >/dev/null
npm ci --no-audit --no-fund
npx playwright install --with-deps chromium
E2E_SKIP_STARTUP=true \
E2E_BASE_URL="$BASE_URL" \
E2E_BACKEND_HEALTH="$BASE_URL/health" \
E2E_FRONTEND_URL="$BASE_URL" \
npx playwright test "$@"
popd >/dev/null
ok "e2e passed"
