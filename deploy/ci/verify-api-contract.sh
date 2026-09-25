#!/usr/bin/env bash
# =============================================================================
# verify-api-contract.sh — MVP API contract gate for daily releases
# =============================================================================
# Ensures the backend routes and the shared API client stay in sync, so a
# backend rename/removal cannot silently break the mobile or web app.
#
# Checks (all offline, grep-based, <5s):
#   1. Backend (cmd/server/main.go) exposes every MVP-critical route.
#   2. packages/shared exposes the matching client groups/methods.
#   3. Frontend + mobile actually wire the feature-flag client (Release 1 gate).
#
# Usage: bash deploy/ci/verify-api-contract.sh
# Exit 1 on any mismatch.
# =============================================================================
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
fails=0
pass() { echo "  PASS: $1"; }
failc() { echo "  FAIL: $1" >&2; fails=$((fails+1)); }

echo "== API contract gate (backend <-> shared client <-> apps) =="

MAIN="$ROOT/backend/cmd/server/main.go"
SHARED="$ROOT/packages/shared/src/api.ts"
TYPES="$ROOT/packages/shared/src/types.ts"
WEB_API="$ROOT/frontend/src/services/api.ts"
MOBILE_API="$ROOT/mobile/src/services/api.ts"

for f in "$MAIN" "$SHARED" "$TYPES" "$WEB_API" "$MOBILE_API"; do
  if [[ ! -f "$f" ]]; then failc "missing file $f"; fi
done
[[ "$fails" -ne 0 ]] && { echo "API contract: FAIL (missing files)" >&2; exit 1; }

# --- 1. Backend MVP-critical routes ------------------------------------------
# Each entry: "description|grep-pattern-in-main.go"
backend_routes=(
  "auth register|/auth/register"
  "auth login|/auth/login"
  "auth me|/users/me"
  "feature flags (client)|/users/me/flags"
  "feature flags admin list|/features"
  "chats list|/chats"
  "send message|/chats/:chatId/messages"
  "translation|/messages/:messageId/translate"
  "grammar|/grammar"
  "waitlist|/waitlist"
  "learning dashboard|/learning/dashboard"
  "onboarding|/users/me/onboard"
  "user settings|/users/me/settings"
  "flag gating middleware|RequireFlag"
  "sparky ask (async)|/sparky/ask"
  "sparky job resync|/sparky/jobs/:jobId"
)
backend_routes_ere=(
  "flag seed on boot|SeedFlags|DefaultFlags"
)
for entry in "${backend_routes[@]}"; do
  desc="${entry%%|*}"; pat="${entry#*|}"
  if grep -q "$pat" "$MAIN"; then pass "backend route: $desc"; else failc "backend route missing: $desc (pattern $pat)"; fi
done
for entry in "${backend_routes_ere[@]}"; do
  desc="${entry%%|*}"; pat="${entry#*|}"
  if grep -Eq "$pat" "$MAIN"; then pass "backend route: $desc"; else failc "backend route missing: $desc (pattern $pat)"; fi
done

# Flag-gated routes must actually be gated (not just present).
for gated in 'video_calls' 'teacher_marketplace' 'payout_teacher' 'gdpr_export' 'document_sharing' 'location_sharing'; do
  if grep -q "RequireFlag(featureFlagService, \"$gated\")" "$MAIN"; then
    pass "route gate wired: $gated"
  else
    failc "route gate NOT wired: RequireFlag(\"$gated\") missing in main.go"
  fi
done

# --- 2. Shared client parity ---------------------------------------------------
# Each entry: "description|pattern-in-packages/shared/src/api.ts"
client_methods=(
  "auth.register|register:"
  "auth.login|login:"
  "auth.getMe|getMe:"
  "flags.getMyFlags|getMyFlags"
  "chat.getChats|getChats"
  "message.sendMessage|sendMessage"
  "translation.translateMessage|translateMessage"
  "grammar|grammar"
  "learning.getDashboard|getDashboard"
  "settings|settings"
  "flags group|const flags"
  "sparky.ask|sparky"
  "admin.listFlags|listFlags"
  "admin.updateFlagTiers|updateFlagTiers"
  "admin.setFlagOverride|setFlagOverride"
  "admin.previewUserFlags|previewUserFlags"
)
for entry in "${client_methods[@]}"; do
  desc="${entry%%|*}"; pat="${entry#*|}"
  if grep -q "$pat" "$SHARED"; then pass "shared client: $desc"; else failc "shared client missing: $desc (pattern $pat)"; fi
done

# RolloutFlagKey must cover the backend seed set (prevents client/server key drift).
for key in grammar_insights word_collector feature_voting referral_bump video_calls teacher_marketplace payout_teacher gdpr_export group_chat voice_message media_sharing document_sharing location_sharing google_oauth placement_test scenario_roleplay word_flashcards; do
  if grep -q "'$key'" "$TYPES"; then pass "flag key in shared types: $key";
  else failc "flag key missing in shared types: $key"; fi
done
for key in grammar_insights word_collector feature_voting referral_bump video_calls teacher_marketplace payout_teacher gdpr_export voice_message media_sharing document_sharing location_sharing google_oauth; do
  if grep -q "\"$key\"" "$ROOT/backend/internal/services/feature_flags.go"; then pass "flag key in backend seeds: $key";
  else failc "flag key missing in backend seeds: $key"; fi
done

# --- 3. App wiring --------------------------------------------------------------
if grep -q "flagsAPI" "$WEB_API" && grep -q "refreshRolloutFlags\|useFeatureFlag" "$ROOT/frontend/src/App.tsx" "$ROOT/frontend/src/store/index.ts" 2>/dev/null; then
  pass "web wires flags client"
else
  failc "web does not wire flags client (flagsAPI + refreshRolloutFlags/useFeatureFlag)"
fi

if grep -q "featureFlags\|getMyFlags" "$MOBILE_API"; then
  pass "mobile wires flags client"
else
  failc "mobile does not wire flags client (featureFlags/getMyFlags in services/api.ts)"
fi

if grep -q "sparkyAsk\|sparkyJob" "$MOBILE_API"; then
  pass "mobile wires sparky async client"
else
  failc "mobile does not wire sparky async client (sparkyAsk/sparkyJob in services/api.ts)"
fi

# Call UI must be flag-gated on both apps (Release 1: calls are admin-only).
if grep -q "video_calls" "$ROOT/frontend/src/components/ChatArea.tsx"; then
  pass "web ChatArea gates calls on video_calls"
else
  failc "web ChatArea does not reference video_calls flag"
fi
if grep -q "video_calls" "$ROOT/mobile/src/screens/ChatScreen.tsx"; then
  pass "mobile ChatScreen gates calls on video_calls"
else
  failc "mobile ChatScreen does not reference video_calls flag"
fi

# Composer attachments must be flag-gated (hide-until-enabled).
for flag in media_sharing document_sharing location_sharing voice_message; do
  if grep -q "$flag" "$ROOT/mobile/src/screens/ChatScreen.tsx"; then
    pass "mobile ChatScreen gates composer on $flag"
  else
    failc "mobile ChatScreen does not reference $flag"
  fi
done
for flag in media_sharing document_sharing location_sharing voice_message google_oauth; do
  if grep -q "$flag" "$ROOT/frontend/src/components/ChatArea.tsx" "$ROOT/frontend/src/pages/Login.tsx" 2>/dev/null; then
    pass "web gates $flag"
  else
    failc "web does not reference $flag (ChatArea.tsx/Login.tsx)"
  fi
done
if grep -q "teacher_marketplace" "$ROOT/mobile/src/components/MainTabs.tsx" "$ROOT/mobile/src/screens/LearnScreen.tsx" 2>/dev/null; then
  pass "mobile gates marketplace on teacher_marketplace"
else
  failc "mobile does not gate marketplace (MainTabs.tsx/LearnScreen.tsx)"
fi
if grep -q "teacher_marketplace" "$ROOT/frontend/src/components/BottomNav.tsx"; then
  pass "web BottomNav gates marketplace on teacher_marketplace"
else
  failc "web BottomNav does not reference teacher_marketplace"
fi

# Admin Flags console must exist and be admin-routed (internal tool, English-only).
if [[ -f "$ROOT/frontend/src/pages/AdminFlags.tsx" ]] && grep -q "adminAPI.listFlags" "$ROOT/frontend/src/pages/AdminFlags.tsx"; then
  pass "web AdminFlags console present"
else
  failc "web AdminFlags console missing (pages/AdminFlags.tsx + listFlags)"
fi
if grep -q "admin/flags" "$ROOT/frontend/src/App.tsx"; then
  pass "web routes /admin/flags"
else
  failc "web does not route /admin/flags (App.tsx)"
fi

# --- Summary -------------------------------------------------------------------
echo
if [[ "$fails" -ne 0 ]]; then
  echo "API contract: FAIL ($fails check(s) failed) — client/server drift detected." >&2
  exit 1
fi
echo "API contract: PASS."
exit 0
