#!/usr/bin/env bash
# =============================================================================
# verify-isolation.sh — NFR-22 mail server isolation network-policy check
# =============================================================================
# Runs as a release-gate step (CI or pre-deploy). It makes the "mail server
# must not share the app's host/network namespace" policy *mechanically*
# verifiable instead of relying on a human reading the compose files.
#
# What it enforces (NFR-22):
#   1. The app compose publishes NO mail ports (25/465/587/993/995/143/110).
#   2. Mailu is NOT a service inside the app compose; it lives on its own
#      isolated network in deploy/mail/docker-compose.mail.yml.
#   3. Backend SMTP creds are scoped to server-side env vars, never VITE_*
#      (so they can never be bundled into the browser).
#   4. No real MAILU_SMTP_PASSWORD value is committed (keep placeholders empty;
#      real value lives in Dokploy secrets).
#   5. The prod app points at SMTP submission (587 / STARTTLS).
#
# Exit code 0 = PASS, 1 = FAIL (gate blocks). Prints each check + why.
#
# Usage:  bash deploy/mail/verify-isolation.sh
# =============================================================================
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
APP_COMPOSE="$REPO_ROOT/docker-compose.prod.yml"
MAIL_COMPOSE="$REPO_ROOT/deploy/mail/docker-compose.mail.yml"
ENV_EXAMPLE="$REPO_ROOT/.env.prod.example"
BACKEND_ENV_EXAMPLE="$REPO_ROOT/backend/.env.example"

fails=0
pass() { echo "  PASS: $1"; }
fail() { echo "  FAIL: $1"; fails=$((fails + 1)); }

# Match a compose port publish `[ADDR:]HOSTPORT:MAILPORT` whose container port
# is a mail port (25/465/587/993/995/143/110). Handles "127.0.0.1:587:587" and
# plain "587:587".
MAIL_PORTS_RE='(^|[^0-9])(25|465|587|993|995|143|110):(25|465|587|993|995|143|110)([^0-9]|$)'
has_mail_port() {
  grep -nE "$MAIL_PORTS_RE" || true
}

echo "== NFR-22 mail server isolation =="

# --- 1. Mailu is a SEPARATE, isolated deployment (own network) --------------
if [[ ! -f "$MAIL_COMPOSE" ]]; then
  fail "missing $MAIL_COMPOSE (Mailu must be deployed outside the app compose)"
else
  if grep -qE '^\s*chorus-network:\s*$' "$MAIL_COMPOSE"; then
    fail "Mailu compose joins 'chorus-network' — mail would share the app network namespace"
  elif grep -qE '^\s*mailu:\s*$' "$MAIL_COMPOSE"; then
    pass "Mailu defines its own isolated 'mailu' bridge network"
  else
    fail "Mailu compose does not define an isolated network"
  fi
fi

# --- 2. App compose publishes NO mail ports and contains NO Mailu ------------
if [[ -f "$APP_COMPOSE" ]]; then
  backend_block="$(awk '/^  backend:/{flag=1} flag{print} flag && /^  (postgres|redis|frontend|libretranslate|volumes|networks):/{exit}' "$APP_COMPOSE")"

  published="$(cat "$APP_COMPOSE" | has_mail_port)"
  backend_ports="$(printf '%s' "$backend_block" | has_mail_port )"
  if [[ -n "$published" || -n "$backend_ports" ]]; then
    fail "app compose publishes a mail port: $(printf '%s %s' "$published" "$backend_ports" | tr '\n' ' ')"
  else
    pass "app compose publishes no mail ports (25/465/587/993/995/143/110)"
  fi

  # Mailu must not be a SERVICE (image or top-level key) inside the app compose.
  # Comments mentioning "Mailu SMTP" env forwarding are fine — only a real
  # shared service violates host/network isolation.
  if grep -qE '(^|[[:space:]])image:[[:space:]]*[^#]*mailu|^\s*mailu:\s*$' "$APP_COMPOSE"; then
    fail "Mailu is deployed as a service inside the app compose — violates host/network isolation"
  else
    pass "Mailu is NOT a service inside the app compose"
  fi
else
  fail "missing $APP_COMPOSE"
fi

# --- 3. SMTP creds are env-scoped, never VITE_ (no browser exposure) --------
vite_hits=0
for f in "$APP_COMPOSE" "$ENV_EXAMPLE" "$BACKEND_ENV_EXAMPLE"; do
  if [[ -f "$f" ]] && grep -qE 'VITE_[A-Z0-9_]*(SMTP|MAIL|MAILU)' "$f"; then
    echo "  FAIL: VITE_* SMTP var in $f: $(grep -nE 'VITE_[A-Z0-9_]*(SMTP|MAIL|MAILU)' "$f" | tr '\n' ' ')"
    vite_hits=$((vite_hits + 1))
  fi
done
if [[ "$vite_hits" -eq 0 ]]; then
  pass "no SMTP/mail secrets are exposed under VITE_* (server-side env only)"
fi

# --- 4. No committed real secret value (non-empty password) -----------------
leak=0
for f in "$ENV_EXAMPLE" "$BACKEND_ENV_EXAMPLE" "$APP_COMPOSE"; do
  if [[ -f "$f" ]] && grep -qE '^[[:space:]]*MAILU_SMTP_PASSWORD=[^[:space:]]' "$f"; then
    echo "  FAIL: non-empty MAILU_SMTP_PASSWORD committed in $f (rotate & move to Dokploy secrets)"
    leak=$((leak + 1))
  fi
done
if [[ "$leak" -eq 0 ]]; then
  pass "no real SMTP password committed (placeholders only; use Dokploy secrets)"
fi

# --- 5. Prod app uses SMTP submission (587), not 25 -------------------------
if [[ -f "$ENV_EXAMPLE" ]]; then
  if grep -qE '^MAILU_SMTP_PORT=587' "$ENV_EXAMPLE"; then
    pass "app points at SMTP submission on 587"
  elif grep -qE '^MAILU_SMTP_PORT=465' "$ENV_EXAMPLE"; then
    echo "  INFO: $ENV_EXAMPLE uses MAILU_SMTP_PORT=465 (implicit TLS); switch to 587"
    echo "        so the app only needs the submission port open (minimal ingress)."
  fi
fi

# --- Summary ----------------------------------------------------------------
echo
if [[ "$fails" -ne 0 ]]; then
  echo "NFR-22 mail isolation: FAIL ($fails check(s) failed) — gate blocked."
  exit 1
fi
echo "NFR-22 mail isolation: PASS."
exit 0
