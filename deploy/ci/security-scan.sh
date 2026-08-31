#!/usr/bin/env bash
# Gate: security scan (NFR-26) — govulncheck (Go) + npm audit (JS deps).
# Fails on Go vulnerabilities and on npm audit issues at or above the level.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
NPM_AUDIT_LEVEL="${NPM_AUDIT_LEVEL:-high}"
GOVULNCHECK_VERSION="${GOVULNCHECK_VERSION:-latest}"

log "Go vuln scan (govulncheck)"
pushd "$ROOT/backend" >/dev/null
go run "golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}" ./...
popd >/dev/null
ok "govulncheck passed"

log "npm audit (frontend)"
pushd "$ROOT/frontend" >/dev/null
npm ci --no-audit --no-fund >/dev/null
npm audit --audit-level="$NPM_AUDIT_LEVEL"
popd >/dev/null
ok "frontend npm audit passed"

log "npm audit (mobile)"
pushd "$ROOT/mobile" >/dev/null
npm ci --no-audit --no-fund >/dev/null
npm audit --audit-level="$NPM_AUDIT_LEVEL"
popd >/dev/null
ok "mobile npm audit passed"

log "npm audit (e2e)"
if [ -f "$ROOT/e2e/package-lock.json" ]; then
  pushd "$ROOT/e2e" >/dev/null
  npm ci --no-audit --no-fund >/dev/null
  npm audit --audit-level="$NPM_AUDIT_LEVEL"
  popd >/dev/null
  ok "e2e npm audit passed"
else
  warn "no e2e/package-lock.json — skipping e2e npm audit"
fi

ok "all security scans passed"
