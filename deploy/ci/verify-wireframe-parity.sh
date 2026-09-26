#!/usr/bin/env bash
# verify-wireframe-parity.sh — MKT-QA: every wireframe has reachable screen+route + learn dashboard flows
# Offline: checks routing table against wireframes/ inventory. No DB needed.
set -uo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
fails=0
pass() { ok "$1"; }
failc() { echo "  FAIL: $1" >&2; fails=$((fails+1)); }

echo "== MKT-QA Wireframe parity gate =="

# 1. Required wireframe → route map (P0 required for marketplace + learn dashboard)
# Format: wireframe_folder|frontend_route|mobile_screen
REQUIRED=(
  "browse_tutors|/tutors|BrowseTutors"
  "tutor_profile_sofia|/tutors/:id|TutorProfile"
  "trial_credit_dashboard|/trial-credits|TrialCredits"
  "teacher_dashboard|/teacher/dashboard|TeacherDashboard"
  "payout_settings_history|/teacher/payouts|Payouts"
  "become_a_teacher|/become-teacher|BecomeTeacher"
  "streak_recovery_challenge|/learn/streak-recovery|StreakRecovery"
  "chats|/chat|ChatList"
  "chat_with_sofia_1|/chat/:slug|Chat"
  "video_call_learning_deep_dive|/learn/scenarios|Scenarios"
  "ai_scenario_roleplay_ordering_coffee|/learn/scenarios/:scenarioId|ScenarioRoleplay"
  "daily_practice_session|/learn/session|LessonSession"
  "my_vocabulary_hub|/learn/vocabulary|VocabularyReview"
  "placement_test_welcome|/learn/placement|Placement"
  "learning_roadmap_streak_status|/learn/roadmap|LearningRoadmap"
  "real_talk_hub|/learn/real-talk|RealTalkHub"
  "settings|/profile|Profile"
  "user_profile_mateo|/profile|Profile"
)

FRONTEND_APP="$ROOT/frontend/src/App.tsx"
MOBILE_TABS="$ROOT/mobile/src/components/MainTabs.tsx"
MOBILE_SCREENS="$ROOT/mobile/src/screens"
FRONTEND_PAGES="$ROOT/frontend/src/pages"

for entry in "${REQUIRED[@]}"; do
  IFS='|' read -r wf froute mscreen <<< "$entry"
  if [[ ! -d "$ROOT/wireframes/$wf" ]]; then failc "wireframe folder missing: $wf"; continue; fi
  # frontend route check (substring search)
  if grep -q "$froute" "$FRONTEND_APP" 2>/dev/null || grep -q "${froute#*\/}" "$FRONTEND_APP" 2>/dev/null; then
    pass "$wf -> frontend $froute"
  else
    # fuzzy: check for route file import
    if grep -q "$mscreen" "$FRONTEND_APP" 2>/dev/null; then pass "$wf -> frontend $mscreen (import)"; else failc "$wf missing frontend route $froute (not in App.tsx)"; fi
  fi
  # mobile check
  if grep -q "$mscreen" "$MOBILE_TABS" 2>/dev/null || [[ -f "$MOBILE_SCREENS/${mscreen}Screen.tsx" ]] || [[ -f "$MOBILE_SCREENS/${mscreen}.tsx" ]]; then
    pass "$wf -> mobile $mscreen"
  else
    failc "$wf missing mobile screen $mscreen"
  fi
done

# 2. Learn dashboard flows: API + navigation
FRONTEND_LEARN="$ROOT/frontend/src/pages/Learn.tsx"
MOBILE_LEARN="$ROOT/mobile/src/screens/LearnScreen.tsx"
SHARED_API="$ROOT/packages/shared/src/api.ts"
for pat in "getDashboard|getLearningDashboard" "MonthlyActivity|monthlyActivity" "dailyGoal" "streak"; do
  if grep -Eq "$pat" "$FRONTEND_LEARN" 2>/dev/null && grep -Eq "$pat" "$MOBILE_LEARN" 2>/dev/null; then pass "learn dashboard flows: $pat present on both"; else failc "learn dashboard flows: $pat missing on web or mobile"; fi
done
# getPath is roadmap flow, verify presence somewhere in codebase
if grep -RqE "getPath|getLearningPath" "$ROOT/frontend/src" 2>/dev/null && grep -RqE "getPath|getLearningPath" "$ROOT/mobile/src" 2>/dev/null; then pass "roadmap getPath present on both"; else failc "roadmap getPath missing"; fi

# Streak recovery flow
if grep -q "streak-recovery" "$FRONTEND_APP" && grep -q "StreakRecovery" "$MOBILE_TABS"; then pass "streak recovery route present both platforms"; else failc "streak recovery route missing"; fi
if grep -q "recoverStreak" "$SHARED_API" 2>/dev/null; then pass "shared api recoverStreak present"; else failc "shared api recoverStreak missing"; fi
if grep -q "StreakRecovery" "$FRONTEND_PAGES/StreakRecovery.tsx" 2>/dev/null && grep -q "StreakRecovery" "$MOBILE_SCREENS/StreakRecoveryScreen.tsx" 2>/dev/null; then pass "streak recovery screens exist"; else failc "streak recovery screens missing"; fi

# Monthly activity card test coverage
if grep -qi "monthly" "$ROOT/frontend/src/pages/__tests__/learningPages.test.tsx" 2>/dev/null && grep -qi "monthly" "$ROOT/mobile/src/screens/__tests__/learning.test.tsx" 2>/dev/null; then pass "monthly activity tests present"; else failc "monthly activity tests missing"; fi

# Activity hub: embedded in Learn is acceptable, document it
if grep -q "activity_hub" "$ROOT/docs/WIREFRAME_TRACE.md" 2>/dev/null; then pass "WIREFRAME_TRACE.md documents activity_hub"; else failc "WIREFRAME_TRACE.md missing activity_hub entry"; fi

# Marketplace tab wiring
if grep -q "MarketplaceTab" "$MOBILE_TABS" && grep -q "/tutors" "$FRONTEND_APP"; then pass "marketplace tab + /tutors route wiring present"; else failc "marketplace wiring missing"; fi

# 3. Learn → Tutors bridge
if grep -qi "tutor" "$FRONTEND_LEARN" && grep -qi "tutor" "$MOBILE_LEARN"; then pass "Learn -> Tutors bridge present"; else failc "Learn -> Tutors bridge missing"; fi

# Shell syntax
bash -n "$ROOT/deploy/ci/verify-wireframe-parity.sh" 2>/dev/null && pass "verify-wireframe-parity.sh syntax ok" || failc "verify-wireframe-parity.sh syntax error"

echo
if [[ "$fails" -ne 0 ]]; then
  echo "Wireframe parity: FAIL ($fails check(s) failed)" >&2
  exit 1
fi
echo "Wireframe parity: PASS."
exit 0
