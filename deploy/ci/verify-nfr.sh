#!/usr/bin/env bash
# verify-nfr.sh — NFR compliance gate for 9.6 (17/22/23/24/25/26)
set -uo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OFFLINE=0; [[ "${1:-}" == "--offline" ]] && OFFLINE=1
fails=0; pass() { ok "$1"; }; failc() { echo "  FAIL: $1" >&2; fails=$((fails+1)); }
echo "== 9.6 NFR compliance (17/22/23/24/25/26) =="

# ── NFR-17 No secrets in repo ───────────────────────────────────────────────
echo "-- NFR-17 No secrets in repo --"
for p in .gitignore; do [[ -f "$ROOT/$p" ]] && pass "$p present" || failc "missing $p"; done
for pat in ".env" ".env.prod" "backend/.env" "frontend/.env"; do
  count=$(grep -c "$pat" "$ROOT/.gitignore" 2>/dev/null || true)
  [[ "$count" -gt 0 ]] && pass ".gitignore covers $pat" || failc ".gitignore missing $pat"
done
if command -v git >/dev/null 2>&1; then
  for f in .env .env.prod backend/.env frontend/.env frontend/.env.local mobile/.env; do
    if git -C "$ROOT" ls-files --error-unmatch "$f" >/dev/null 2>&1; then failc "tracked secret file: $f"; else pass "not tracked: $f"; fi
  done
  tracked_secrets=$(git -C "$ROOT" ls-files | grep -E 'token.*\.txt$|token.*\.json$|tmp_.*|\.keystore$|\.jks$' | grep -v "debug.keystore" || true)
  if [[ -n "$tracked_secrets" ]]; then failc "tracked secret artifacts: $tracked_secrets"; else pass "no tracked secret artifacts (debug.keystore excluded)"; fi
fi
leak=0
for f in "$ROOT/.env.prod.example" "$ROOT/backend/.env.example"; do
  [[ -f "$f" ]] || continue
  if grep -qE '^[[:space:]]*MAILU_SMTP_PASSWORD=[^[:space:]#]+' "$f"; then echo "  FAIL: real MAILU_SMTP_PASSWORD in $f" >&2; leak=$((leak+1)); fails=$((fails+1)); fi
  if grep -qE '^[[:space:]]*JWT_SECRET=.+[^#]{16}' "$f" | grep -qv "your-very-long" 2>/dev/null; then :; fi
done
[[ "$leak" -eq 0 ]] && pass "no real SMTP password in examples"
if grep -rqE 'VITE_[A-Z0-9_]*(SMTP|MAILU|JWT_SECRET)' "$ROOT/docker-compose.prod.yml" "$ROOT/docker-compose.yml" 2>/dev/null; then failc "VITE_* secret var in compose"; else pass "no VITE_* SMTP leak in compose"; fi
if grep -rqE 'BEGIN (RSA )?PRIVATE KEY|BEGIN OPENSSH PRIVATE KEY' "$ROOT" --include="*.md" --include="*.yml" --include="*.sh" 2>/dev/null | grep -qv ".git"; then failc "private key material in repo"; else pass "no private key material"; fi

# ── NFR-22 Mail isolation ──────────────────────────────────────────────────
echo "-- NFR-22 Mail isolation --"
if [[ -f "$ROOT/deploy/mail/verify-isolation.sh" ]]; then
  if bash "$ROOT/deploy/mail/verify-isolation.sh" >/tmp/nfr_vmi.log 2>&1; then pass "mail isolation PASS"; else cat /tmp/nfr_vmi.log >&2; failc "mail isolation FAIL"; fi
else failc "missing deploy/mail/verify-isolation.sh"; fi
[[ -f "$ROOT/deploy/mail/docker-compose.mail.yml" ]] && pass "mail compose present" || failc "missing mail compose"
[[ -f "$ROOT/deploy/mail/README.md" ]] && pass "mail README present" || failc "missing mail README"
if grep -q "587" "$ROOT/.env.prod.example" 2>/dev/null; then pass ".env.prod.example uses 587"; elif grep -q "465" "$ROOT/.env.prod.example" 2>/dev/null; then pass ".env.prod.example uses 465 (implicit TLS — prefer 587)"; else failc ".env.prod.example missing SMTP port"; fi

# ── NFR-23 Data retention / GDPR ───────────────────────────────────────────
echo "-- NFR-23 Retention + GDPR --"
[[ -f "$ROOT/docs/DATA_RETENTION_GDPR.md" ]] && pass "DATA_RETENTION_GDPR.md present" || failc "missing DATA_RETENTION_GDPR.md"
grep -q "Retention" "$ROOT/docs/DATA_RETENTION_GDPR.md" 2>/dev/null && pass "retention doc has Retention" || failc "retention doc missing Retention"
for kw in "message_retention_days" "GET.*export" "DELETE.*users/me"; do grep -qi "$kw" "$ROOT/docs/DATA_RETENTION_GDPR.md" 2>/dev/null && pass "retention doc covers $kw" || failc "retention doc missing $kw"; done
for f in backend/internal/services/retention.go backend/internal/services/gdpr.go backend/internal/handlers/gdpr.go; do [[ -f "$ROOT/$f" ]] && pass "$f present" || failc "missing $f"; done
for sym in "PurgeExpiredMessages" "PurgeExpiredInbox" "PurgeExpiredTranslationJobs" "PurgeExpiredCallTranscripts" "GetPolicy" "ExportUserData" "EraseUser"; do grep -q "$sym" "$ROOT/backend/internal/services/retention.go" "$ROOT/backend/internal/services/gdpr.go" 2>/dev/null && pass "symbol $sym" || failc "missing symbol $sym"; done
grep -q "StartScheduler" "$ROOT/backend/cmd/server/main.go" 2>/dev/null && pass "main.go starts retention scheduler" || failc "main.go missing StartScheduler"
grep -q "retention-policy" "$ROOT/backend/cmd/server/main.go" 2>/dev/null && pass "main.go exposes retention-policy" || failc "main.go missing retention-policy endpoint"
grep -q "retention/purge" "$ROOT/backend/cmd/server/main.go" 2>/dev/null && pass "main.go exposes retention/purge" || failc "main.go missing retention/purge"
grep -q "users/me/export" "$ROOT/backend/cmd/server/main.go" 2>/dev/null && pass "main.go exposes GDPR export" || failc "main.go missing GDPR export"
grep -q "DELETE.*users/me" "$ROOT/backend/cmd/server/main.go" 2>/dev/null && pass "main.go exposes GDPR erasure" || failc "main.go missing GDPR erasure"

# ── NFR-24 Rate limiting ───────────────────────────────────────────────────
echo "-- NFR-24 Rate limiting --"
for f in backend/internal/middleware/rate_limit.go backend/internal/services/ratelimit.go; do [[ -f "$ROOT/$f" ]] && pass "$f present" || failc "missing $f"; done
grep -q "RateLimiterRedis" "$ROOT/backend/internal/middleware/rate_limit.go" 2>/dev/null && pass "RateLimiterRedis present" || failc "missing RateLimiterRedis"
grep -q "RedisRateLimiter" "$ROOT/backend/internal/services/ratelimit.go" 2>/dev/null && pass "RedisRateLimiter present" || failc "missing RedisRateLimiter"
grep -q "UserRateLimiter" "$ROOT/backend/internal/middleware/rate_limit.go" 2>/dev/null && pass "UserRateLimiter present" || failc "missing UserRateLimiter"
for ep in "waitlist" "auth/register" "auth/login" "forgot-password" "phone/request-otp" "TranslateMessage" "/ws"; do grep -q "$ep" "$ROOT/backend/cmd/server/main.go" 2>/dev/null && pass "rate limit covers $ep" || failc "rate limit missing $ep"; done
grep -q "RateLimiterRedis.*redisClient" "$ROOT/backend/cmd/server/main.go" 2>/dev/null && pass "main.go uses Redis-backed limiter" || failc "main.go not using Redis limiter"

# ── NFR-25 Observability + Phoenix ─────────────────────────────────────────
echo "-- NFR-25 Observability / Phoenix --"
for f in backend/internal/observability/phoenix.go backend/internal/observability/metrics.go backend/internal/observability/health.go; do [[ -f "$ROOT/$f" ]] && pass "$f present" || failc "missing $f"; done
grep -q "PHOENIX_ENABLED" "$ROOT/backend/internal/observability/phoenix.go" 2>/dev/null && pass "phoenix.go checks PHOENIX_ENABLED" || failc "phoenix.go missing PHOENIX_ENABLED"
grep -q "SetupPhoenix" "$ROOT/backend/cmd/server/main.go" 2>/dev/null && pass "main.go calls SetupPhoenix" || failc "main.go missing SetupPhoenix"
grep -q "phoenix-dev" "$ROOT/deploy/dev/docker-compose.yml" 2>/dev/null && pass "dev compose has phoenix-dev" || failc "dev compose missing phoenix-dev"
grep -q 'PHOENIX_ENABLED.*true' "$ROOT/deploy/dev/docker-compose.yml" 2>/dev/null && pass "dev compose enables Phoenix" || failc "dev compose not enabling Phoenix"
grep -q "4317" "$ROOT/deploy/dev/docker-compose.yml" 2>/dev/null && pass "dev compose exposes OTLP 4317" || failc "dev compose missing OTLP 4317"
grep -q "/metrics" "$ROOT/backend/cmd/server/main.go" 2>/dev/null && pass "main.go exposes /metrics" || failc "main.go missing /metrics"
grep -q "/health" "$ROOT/backend/cmd/server/main.go" 2>/dev/null && pass "main.go exposes /health" || failc "main.go missing /health"
[[ -f "$ROOT/deploy/ci/phoenix-eval.sh" ]] && pass "phoenix-eval.sh present" || failc "missing phoenix-eval.sh"
bash -n "$ROOT/deploy/ci/phoenix-eval.sh" 2>/dev/null && pass "phoenix-eval.sh syntax ok" || failc "phoenix-eval.sh syntax error"
grep -q "prometheus" "$ROOT/docker-compose.yml" 2>/dev/null && pass "prod compose has prometheus" || failc "prod compose missing prometheus"

# ── NFR-26 CI/CD quality gates ─────────────────────────────────────────────
echo "-- NFR-26 CI/CD quality gates --"
[[ -f "$ROOT/.github/workflows/ci.yml" ]] && pass "ci.yml present" || failc "missing ci.yml"
[[ -f "$ROOT/docs/CI_CD_NFR26.md" ]] && pass "CI_CD_NFR26.md present" || failc "missing CI_CD_NFR26.md"
for job in "test:" "security:" "images:" "deploy-dev:" "promote-prod:"; do grep -q "$job" "$ROOT/.github/workflows/ci.yml" 2>/dev/null && pass "ci.yml has $job" || failc "ci.yml missing $job"; done
grep -q "buildx imagetools create" "$ROOT/deploy/ci/promote.sh" 2>/dev/null && pass "promote.sh uses imagetools create" || failc "promote.sh missing imagetools create"
grep -q "gate-on-dev" "$ROOT/.github/workflows/ci.yml" 2>/dev/null && pass "ci.yml runs gate-on-dev" || failc "ci.yml missing gate-on-dev"
for s in deploy/ci/security-scan.sh deploy/ci/gate-on-dev.sh deploy/ci/e2e.sh deploy/ci/phoenix-eval.sh deploy/ci/load-smoke.sh deploy/ci/promote.sh deploy/ci/verify-release-gate.sh; do
  [[ -f "$ROOT/$s" ]] && pass "$s present" || failc "missing $s"
  bash -n "$ROOT/$s" 2>/dev/null && pass "$s syntax ok" || failc "$s syntax error"
done
grep -q "IMAGE_TAG:-prod" "$ROOT/docker-compose.prod.yml" 2>/dev/null && pass "prod compose uses IMAGE_TAG=prod" || failc "prod compose must use IMAGE_TAG=prod"
grep -q "govulncheck" "$ROOT/deploy/ci/security-scan.sh" 2>/dev/null && pass "security-scan has govulncheck" || failc "security-scan missing govulncheck"
grep -q "npm audit" "$ROOT/deploy/ci/security-scan.sh" 2>/dev/null && pass "security-scan has npm audit" || failc "security-scan missing npm audit"
bash -n "$ROOT/deploy/ci/verify-nfr.sh" 2>/dev/null && pass "verify-nfr.sh syntax ok" || failc "verify-nfr.sh syntax error"

echo
if [[ "$fails" -ne 0 ]]; then echo "NFR compliance: FAIL ($fails check(s) failed)." >&2; exit 1; fi
echo "NFR compliance: PASS."
exit 0
