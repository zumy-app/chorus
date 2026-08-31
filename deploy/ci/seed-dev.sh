#!/usr/bin/env bash
# Gate: seed the dev DB with the canonical demo/ci users (NFR-26).
#
# Users are created via the public /api/v1/auth/register endpoint so passwords
# are bcrypt-hashed by the backend (never by raw SQL). Idempotent: if a user
# already exists, registration returns a non-2xx and we fall back to a login
# check. Run against the deployed dev LB.
#
# Required env: E2E_BASE_URL (default https://dev.chorus.talk).
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

BASE_URL="${E2E_BASE_URL:-https://dev.chorus.talk}"

ensure_user() {
  local email="$1" password="$2" display_name="$3" native="$4" targets="$5"
  local body code
  body=$(printf '{"email":"%s","password":"%s","displayName":"%s","nativeLanguage":"%s","targetLanguages":["%s"]}' \
    "$email" "$password" "$display_name" "$native" "$targets")

  code=$(curl -s -o /dev/null -w '%{http_code}' -X POST \
    -H 'Content-Type: application/json' -d "$body" \
    "$BASE_URL/api/v1/auth/register")

  if [ "$code" = "200" ] || [ "$code" = "201" ]; then
    ok "seeded user $email"
  elif [ "$code" = "400" ] || [ "$code" = "409" ]; then
    warn "$email already registered (HTTP $code) — verifying login"
    login_ok "$email" "$password"
  else
    fail "register $email returned HTTP $code"
  fi
}

login_ok() {
  local email="$1" password="$2" code
  code=$(curl -s -o /dev/null -w '%{http_code}' -X POST \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"$email\",\"password\":\"$password\"}" \
    "$BASE_URL/api/v1/auth/login")
  if [ "$code" != "200" ]; then
    fail "login check for $email failed (HTTP $code)"
  fi
  ok "login verified for $email"
}

log "Seeding dev DB via $BASE_URL"
ensure_user "uhsarp@gmail.com"      "Demor@cer1" "Prashanth"    "en" "es"
ensure_user "avcxafefwer@gmail.com" "Demor@cer1" "avcxafefwer"  "es" "en"
ok "dev DB seeded"
