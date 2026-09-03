# Wireframe Trace — wireframes/ vs mobile/src + frontend/src

> Generated: 2026-09-01 | Audit scope: 94 entries in `wireframes/` (93 dirs + 1 file) vs `mobile/src` (22 screens), `frontend/src` (22 pages), `backend/cmd/server/main.go` + handlers/services
> Convention: `GAP` = no screen + no route on that platform. Citations use `file:line`.

## Summary

| Metric | Count |
|---|---|
| Total wireframe entries | 94 |
| Fully implemented (both platforms have screen+route) | 18 |
| Partial (one platform or backend-only) | 14 |
| GAP — no screen/route on either platform | 62 |
| Backend API exists but no UI | 11 |

### By area

| Area | Implemented | GAP |
|---|---|---|
| Auth / Landing / Nav | 8/10 | `chorus_*` branding variants intentionally not separate screens |
| Chat & Calls | 9/14 | `chat_*` advanced variants collapsed into `ChatScreen`/`CallScreen` |
| **Teacher marketplace** | **1/13** (`become_a_teacher` only) | **12 GAP** — browse, profile, trial booking, credits, dashboards, payouts have backend but zero UI |
| **Learning dashboard** | **7/12** | `activity_hub*`, `learning_journey_progress`, `my_learning_progress_student`, `student_insights*` have no dedicated screens |
| Social / Group / Community | 0/8 | All GAP |
| Trust & Safety / Moderation | 1/7 | `moderator_queue` via `AdminWaitlist`, rest GAP |
| Growth / Waitlist / Pricing | 3/3 | Implemented |

### Critical GAPs — Teacher Marketplace (backend exists, UI missing)

Backend in `backend/cmd/server/main.go:604-634`, `backend/internal/handlers/teacher.go:1`, `backend/internal/services/teacher.go:130` (BrowseTutors), `:103` (GetTutorProfile), `:247` (TrialCreditDashboard), `:633` (GetDashboard), `backend/internal/services/payout.go:107` (GetOverview):

- `browse_tutors` → `GET /teachers/browse` exists but **no `BrowseTutorsScreen` / no `/tutors` route** — GAP
- `tutor_profile_sofia` → `GET /teachers/:id` + reviews/availability exists but **no dedicated profile page** — GAP
- `find_a_trial_tutor` → filtered browse; **no dedicated discovery screen** — GAP
- `confirm_trial_booking` → `POST /teachers/:id/book` (`isTrial=true`) exists but **no confirm-booking UI** — GAP
- `trial_credit_dashboard` → `GET /teachers/trial-credits/dashboard` exists but **no UI** — GAP
- `teacher_dashboard` → `GET /teachers/dashboard` exists but **no UI** — GAP
- `teacher_earnings_overview` → `GET /teachers/payouts/overview|history` exists but **no earnings screen** — GAP
- `payout_settings_history` → `GET /teachers/payouts/methods|history` + withdraw exists but **no settings screen** — GAP
- `group_study_hub` / `group_session_management` / `live_group_study_session` / `host_a_study_session` / `custom_activity_builder_pusher` / `teacher_student_learning_chat` / `student_management_progress_teacher` — **no backend, no UI** — GAP

### Critical GAPs — Learning Dashboard

- `activity_hub` + `activity_hub_fixed` → data via `GET /learning/dashboard` (`backend/cmd/server/main.go:659`, `backend/internal/handlers/learning.go:1`) rendered inside `Learn` hub only; **no dedicated Activity Hub screen** — GAP (embedded)
- `learning_roadmap_streak_status` → ✅ PASS (`mobile/src/screens/LearningRoadmapScreen.tsx:1`, `mobile/src/components/MainTabs.tsx:119`; `frontend/src/pages/LearningRoadmap.tsx:1`, `frontend/src/App.tsx:168` `/learn/roadmap`)
- `my_vocabulary_hub` → ✅ PASS (`mobile/src/screens/VocabularyReviewScreen.tsx:1`, `mobile/src/components/MainTabs.tsx:104`; `frontend/src/pages/VocabularyReview.tsx:1`, `frontend/src/App.tsx:148` `/learn/vocabulary`)
- `placement_test_welcome` / `placement_test_vocabulary_question` / `placement_test_reading_comprehension` / `placement_test_results_summary` → ✅ PASS single flow (`mobile/src/screens/PlacementScreen.tsx:1`, `mobile/src/components/MainTabs.tsx:94`; `frontend/src/pages/Placement.tsx:1`, `frontend/src/App.tsx:134` `/learn/placement`; backend `POST /learning/placement/start` `:663`, `POST /learning/placement/:attemptId/answer` `:664`, `GET /learning/placement/:attemptId` `:667`)
- `learning_journey_progress` / `learning_progress` / `my_learning_progress_student` / `student_insights_progress_dashboard` → **no dedicated progress screen** (dashboard fields shown in `Learn`) — GAP
- `daily_practice_session` / `study_session_recap` → ✅ via `LessonSession` (`mobile/src/screens/LessonSessionScreen.tsx:1` `:99`; `frontend/src/pages/LessonSession.tsx:1` `:138` `/learn/session`; backend `POST /learning/sessions/start` `:681`, `POST /learning/sessions/:sessionId/complete` `:684`)
- `real_talk_hub` → ✅ PASS (`mobile/src/screens/RealTalkHubScreen.tsx:1` `:124`; `frontend/src/pages/RealTalkHub.tsx:1` `:170` `/learn/real-talk`)
- `streak_recovery_challenge` → ✅ PASS (`mobile/src/screens/StreakRecoveryScreen.tsx:1`, `mobile/src/components/MainTabs.tsx:StreakRecovery`; `frontend/src/pages/StreakRecovery.tsx:1`, `frontend/src/App.tsx:/learn/streak-recovery`; backend `POST /learning/streak/recover` `:708` via `learningAPI.recoverStreak`)
- `lesson_review_notes_student` / `teacher_pronunciation_review_dashboard` / `teacher_srs` push → backend `POST /teachers/srs/push` `:626` + `PUT /teachers/bookings/:id/review-notes` `:625` but **no UI** — GAP

---

## Full Trace — Every folder in wireframes/

> Status: `PASS` = screen+route on both | `PARTIAL` = one platform or backend-only | `GAP` = missing both | `N/A` = non-screen asset (intentionally not a route)

| # | Wireframe folder | Mobile screen + route | Frontend page + route | Backend | Status | Notes |
|---|---|---|---|---|---|---|
| 1 | `a_clean_modern_and_inspiring_hero_image…` | — | — | — | N/A | Marketing hero asset; embedded in `frontend/src/pages/Landing.tsx:1`, `mobile/src/screens/LandingScreen.tsx:1` — not a navigable screen |
| 2 | `a_conceptual_illustration_showing_a_human_brain…` | — | — | — | N/A | Illustration asset; used in Learn empty states |
| 3 | `a_high_fidelity_ui_mockup_of_the_chorus_app…` | — | — | — | N/A | Design reference; split-screen concept realized via `ChatScreen` + `LearningPanel` |
| 4 | `activity_hub` | GAP — no `ActivityHubScreen` | GAP — no `/activity` route | `GET /learning/dashboard` `backend/cmd/server/main.go:659` | **GAP** | Content (daily goal, streak, fluency ring) embedded in `LearnScreen.tsx:1` / `Learn.tsx:1`; no standalone hub |
| 5 | `activity_hub_fixed` | GAP | GAP | same | **GAP** | Fixed variant of #4; same embedding; no dedicated screen |
| 6 | `ai_deep_dive_with_drills` | partial — `ChatScreen.tsx:1` + `GrammarPanel` via `POST /grammar/analyze` `backend/cmd/server/main.go:637` | partial — `Chat.tsx:1` + `GrammarPanel.tsx:1` + `DeepDiveSheet.tsx:1` | `backend/internal/handlers/grammar.go:1` | PARTIAL | Unified AI deep-dive exists as sheet/panel, not separate route |
| 7 | `ai_deep_dive_with_drills_fixed` | same as #6 | same | same | PARTIAL | Fixed variant; same implementation |
| 8 | `ai_scenario_roleplay_ordering_coffee` | `ScenarioRoleplayScreen.tsx:1` `MainTabs.tsx:114` `name="ScenarioRoleplay"` + `ScenariosScreen.tsx:1` `:109` | `ScenarioRoleplay.tsx:1` `App.tsx:155` `/learn/scenarios/:scenarioId` + `Scenarios.tsx:1` `:143` | `GET /learning/scenarios` `:696` `POST /learning/scenarios/:scenarioId/start` `:698` | **PASS** | Scenario roleplay fully routed |
| 9 | `ai_tutor_learning_moment_in_chat` | `ChatScreen.tsx:1` (inline translate/grammar chip) | `Chat.tsx:1` + `HighlightableText.tsx:1` + `ChatLanguageModal.tsx:1` | `POST /chats/:chatId/messages/:messageId/translate` `:566` `POST /grammar/analyze-ai` `:639` | PARTIAL | No dedicated screen; feature is inline in chat |
| 10 | `ai_tutor_sparky` | `ChatScreen.tsx:1` (Sparky bot contact) | `Chat.tsx:1` (Sparky entry in `ChatList.tsx:1`) | `backend/internal/services/learning_ai.go:1` | PARTIAL | AI tutor persona, not a separate route |
| 11 | `audio_call_with_live_captions` | `CallScreen.tsx:1` `MainTabs.tsx:79` `name="Call"` | `CallScreen.tsx:1` (via `Chat.tsx:1` modal) | `POST /calls/initiate` `:711` `GET /calls/:callId/captions` `:720` `POST /calls/:callId/captions` `:719` | **PASS** | Captions via call signaling |
| 12 | `become_a_teacher` | `BecomeTeacherScreen.tsx:1` (auth stack, not tabbed) | `BecomeTeacher.tsx:1` `App.tsx:183` `/become-teacher` | `POST /teachers/apply` `:612` `GET /teachers/me` `:613` | **PASS** | Only teacher-marketplace screen with full UI on both |
| 13 | `browse_tutors` | **GAP** — no `BrowseTutorsScreen` | **GAP** — no `/tutors` or `/teachers` route | `GET /teachers/browse` `:614` `backend/internal/services/teacher.go:130` | **GAP** | Backend ready; UI missing on both platforms — HIGH PRIORITY |
| 14 | `chat_media_files_gallery` | GAP — `ChatScreen` has attachment preview but no gallery screen | GAP — `ChatArea.tsx:1` shows media inline, no `/gallery` route | `GET /chats/:chatId/gallery` `:581` `backend/internal/handlers/gallery.go:1` | **GAP** | Backend gallery exists; no dedicated gallery UI |
| 15 | `chat_moderation_with_social_path` | `ChatScreen.tsx:1` (block/report via long-press) | `Chat.tsx:1` + `ReportModal.tsx:1` | `POST /blocks` `:485` `POST /reports` `:489` `backend/internal/handlers/moderation.go:1` | PARTIAL | Moderation is action, not screen |
| 16 | `chat_with_advanced_utilities` | `ChatScreen.tsx:1` + `NewChatScreen.tsx:1` | `Chat.tsx:1` + `NewChatModal.tsx:1` + `SearchMessages.tsx:1` | `GET /chats` `:543` `GET /messages/search` `:588` | PARTIAL | Utilities collapsed into chat |
| 17 | `chat_with_call_initiation` | `CallScreen.tsx:1` triggered from `ChatScreen.tsx:1` | `CallScreen.tsx:1` from `ChatArea.tsx:1` | `POST /calls/initiate` `:711` | **PASS** | Call initiation from chat header |
| 18 | `chat_with_real_talk_prompt` | `RealTalkNudge.tsx:1` inside `ChatScreen.tsx:1` | `chat/RealTalkNudge.tsx:1` inside `Chat.tsx:1` | `GET /learning/real-talk/prompts` `:705` `POST /learning/real-talk/prompts/:promptId/used` `:706` | **PASS** | Nudge component, not separate route |
| 19 | `chat_with_sofia_1` | `ChatScreen.tsx:1` (generic chat, Sofia is demo contact) | `Chat.tsx:1` (Sofia seeded as contact) | `GET /chats/:chatId/messages` `:564` | PARTIAL | Demo conversation variant; no Sofia-specific screen |
| 20 | `chat_with_sofia_2` | same | same | same | PARTIAL | Variant of #19 |
| 21 | `chat_with_tutor_voice_note_feedback` | GAP — no voice-note screen | GAP — no voice UI | `POST /chats/:chatId/attachments` `:582` `backend/internal/handlers/attachment.go:1` | **GAP** | Voice notes not implemented as tutor feedback flow |
| 22 | `chat_with_unified_ai_deep_dive` | `ChatScreen.tsx:1` + `StudySandbox.tsx:1` | `Chat.tsx:1` + `DeepDiveSheet.tsx:1` + `StudySandbox.tsx:1` | `POST /grammar/analyze-ai` `:639` | PARTIAL | Deep-dive is sheet, not route |
| 23 | `chat_with_unified_ai_deep_dive_fixed` | same | same | same | PARTIAL | Fixed variant |
| 24 | `chats` | `ChatListScreen.tsx:1` `MainTabs.tsx:64` `name="ChatList"` in `ChatsTab` | `Chat.tsx:1` `App.tsx:125` `/chat` + `/chat/:slug` `:129` `ChatList.tsx:1` | `GET /chats` `:543` `GET /chats/:chatId` `:545` | **PASS** | Core chat list |
| 25 | `chorus_audio_call_with_smart_captions` | `CallScreen.tsx:1` | `CallScreen.tsx:1` | `GET /calls/:callId/captions` `:720` | **PASS** | Duplicate of #11 with smart-caption branding |
| 26 | `chorus_growth_connection_logo` | — | — | — | N/A | Brand asset |
| 27 | `chorus_home` | `LandingScreen.tsx:1` (unauth) | `Landing.tsx:1` `App.tsx:117` `/` | `GET /health` `:433` | **PASS** | Home/landing |
| 28 | `chorus_home_desktop` | — (mobile uses `LandingScreen`) | `Landing.tsx:1` (responsive) | — | **PASS** | Desktop variant of #27; same page responsive |
| 29 | `chorus_home_desktop_v2` | — | `Landing.tsx:1` | — | **PASS** | V2 variant; same route |
| 30 | `chorus_home_desktop_waitlist_product_previews` | — | `Landing.tsx:1` + `Waitlist.tsx:1` `App.tsx:120` `/waitlist` | `POST /waitlist` `:447` | **PASS** | Waitlist CTA on home |
| 31 | `chorus_home_fixed` | `LandingScreen.tsx:1` | `Landing.tsx:1` | — | **PASS** | Fixed variant |
| 32 | `chorus_home_mobile` | `LandingScreen.tsx:1` | — (responsive `Landing.tsx`) | — | **PASS** | Mobile variant |
| 33 | `chorus_home_mobile_v2` | `LandingScreen.tsx:1` | `Landing.tsx:1` | — | **PASS** | V2 mobile |
| 34 | `chorus_home_mobile_waitlist_product_previews` | `LandingScreen.tsx:1` | `Landing.tsx:1` | `POST /waitlist` `:447` | **PASS** | Waitlist on mobile home |
| 35 | `chorus_logo` | — | — | — | N/A | Asset; rendered in `AppHeader.tsx:1` `AuthLayout.tsx:1` |
| 36 | `chorus_minimalist_logo` | — | — | — | N/A | Asset variant |
| 37 | `chorus_premium_upgrade` | `PricingScreen.tsx:1` | `Pricing.tsx:1` `App.tsx:118` `/pricing` + `Premium.tsx:1` `App.tsx:119` `/premium` | `GET /users/me/subscription` `:481` `POST /users/me/subscription/checkout` `:482` | **PASS** | Pricing/premium upsell |
| 38 | `chorus_video_call_experience` | `CallScreen.tsx:1` `MainTabs.tsx:79` | `CallScreen.tsx:1` | `POST /calls/initiate` `:711` `POST /calls/:callId/signal` `:718` | **PASS** | WebRTC video call |
| 39 | `confirm_trial_booking` | **GAP** | **GAP** | `POST /teachers/:id/book` `:634` `backend/internal/services/teacher.go:414` (`isTrial`) + `GET /teachers/trial-credits` `:616` | **GAP** | Backend supports `isTrial` booking; no confirm UI — HIGH PRIORITY |
| 40 | `custom_activity_builder_pusher` | GAP — no builder UI | GAP — no builder | `POST /teachers/srs/push` `:626` `backend/internal/services/teacher_srs.go:20` | **GAP** | Teacher SRS push backend exists; no teacher-facing builder |
| 41 | `daily_practice_session` | `LessonSessionScreen.tsx:1` `MainTabs.tsx:99` `name="LessonSession"` | `LessonSession.tsx:1` `App.tsx:138` `/learn/session` | `POST /learning/sessions/start` `:681` `POST /learning/sessions/:sessionId/items/:itemId/answer` `:683` | **PASS** | Daily practice via session composer |
| 42 | `find_a_trial_tutor` | **GAP** — no `FindTrialTutorScreen` | **GAP** — no `/find-trial` route | `GET /teachers/browse` `:614` (filter) + `GET /teachers/trial-credits` `:616` | **GAP** | Should filter `browse_tutors` by trial availability |
| 43 | `find_invite_partners` | `NewChatScreen.tsx:1` (invite via contacts) | `NewChatModal.tsx:1` | `POST /contacts/invites` `:596` `GET /contacts/search` `:591` | PARTIAL | Partner invite is contact flow, not dedicated screen |
| 44 | `find_learning_partners` | same as #43 | same | same | PARTIAL | Same mechanism |
| 45 | `grammar_insight` | `ChatScreen.tsx:1` (inline `GrammarPanel` query) | `GrammarPanel.tsx:1` + `HighlightableText.tsx:1` | `POST /grammar/analyze` `:637` `POST /grammar/analyze-text` `:638` | PARTIAL | No standalone grammar screen |
| 46 | `group_session_management` | **GAP** | **GAP** | — | **GAP** | No backend, no UI |
| 47 | `group_study_hub` | **GAP** | **GAP** | — | **GAP** | No backend, no UI — would need `GET /study-groups` |
| 48 | `host_a_study_session` | **GAP** | **GAP** | `StudySandbox.tsx:1` (shared sandbox only) | **GAP** | Sandbox exists but no host-flow |
| 49 | `image.png` | — | — | — | N/A | Stray file, not a wireframe folder |
| 50 | `join_waitlist` | `LandingScreen.tsx:1` (waitlist CTA) | `Waitlist.tsx:1` `App.tsx:120` `/waitlist` | `POST /waitlist` `:447` | **PASS** | Waitlist join |
| 51 | `languagelearning` | `LearnScreen.tsx:1` `MainTabs.tsx:89` `name="Learn"` | `Learn.tsx:1` `App.tsx:132` `/learn` | `GET /learning/dashboard` `:659` `GET /learning/path` `:660` | **PASS** | Language learning hub |
| 52 | `learning_community_feed` | **GAP** | **GAP** | — | **GAP** | No community feed backend/UI |
| 53 | `learning_journey_progress` | GAP — embedded in `LearnScreen` (weekly activity, progress bars) | GAP — embedded in `Learn.tsx:1` (`weeklyActivity`, `fluency`) | `GET /learning/dashboard` `:659` + `GET /learning/path` `:660` | **GAP** | No dedicated journey screen; data in dashboard |
| 54 | `learning_progress` | same as #53 | same | same | **GAP** | Alias of #53 |
| 55 | `learning_roadmap_streak_status` | `LearningRoadmapScreen.tsx:1` `MainTabs.tsx:119` `name="LearningRoadmap"` | `LearningRoadmap.tsx:1` `App.tsx:168` `/learn/roadmap` | `GET /learning/path` `:660` `POST /learning/streak/recover` `:708` | **PASS** | Roadmap + streak |
| 56 | `lesson_review_notes_student` | **GAP** — no review-notes screen (student reads teacher notes after lesson) | **GAP** | `PUT /teachers/bookings/:id/review-notes` `:625` `GET /teachers/bookings/:id` `:619` | **GAP** | Backend supports notes; no student-facing reader |
| 57 | `linguist_flow` | — (empty folder) | — | — | N/A | No assets; concept flow |
| 58 | `live_group_study_session` | **GAP** | **GAP** — `StudySandbox.tsx:1` exists but not live-group | `GET /learning/sessions/:sessionId` `:682` (solo) | **GAP** | No multi-user session |
| 59 | `live_lesson_shared_sandbox` | `StudySandbox.tsx:1` + `useStudySandbox.ts:1` | `StudySandbox.tsx:1` + `useStudySandbox.ts:1` | `GET /teachers/srs/sandbox/:studentId` `:629` | PARTIAL | Solo sandbox; no live shared cursors |
| 60 | `login` | `LoginScreen.tsx:1` + `RegisterScreen.tsx:1` + `ForgotPasswordScreen.tsx:1` + `ResetPasswordScreen.tsx:1` | `Login.tsx:1` `App.tsx:142` `/login` + `Register.tsx:1` `:147` + `ForgotPassword.tsx:1` `:152` + `ResetPassword.tsx:1` `:157` | `POST /auth/login` `:450` `POST /auth/register` `:448` etc. | **PASS** | Full auth flow |
| 61 | `moderated_message_in_chat` | `ChatScreen.tsx:1` (moderated badge via `moderation` service) | `ChatArea.tsx:1` + `MessageBubble.tsx:1` | `backend/internal/handlers/moderation.go:1` | PARTIAL | Inline moderation flag, not screen |
| 62 | `moderator_queue` | GAP — mobile not moderator | `AdminWaitlist.tsx:1` `App.tsx:121` `/admin/waitlist` + `/admin` `:135` + `/admin/:tab` `:142` (includes `Reports` tab) | `GET /admin/reports` `:535` `POST /admin/reports/:id/resolve` `:537` | PARTIAL | Web admin queue exists; no mobile screen |
| 63 | `my_learning_progress_student` | GAP — `LearnScreen` + `ProfileScreen` show progress | GAP — `Learn.tsx` + `LearningRoadmap.tsx` | `GET /learning/dashboard` `:659` `GET /vocabulary/progress` `:651` | **GAP** | Duplicate of #54; no dedicated screen |
| 64 | `my_vocabulary_hub` | `VocabularyReviewScreen.tsx:1` `MainTabs.tsx:104` `name="VocabularyReview"` | `VocabularyReview.tsx:1` `App.tsx:148` `/learn/vocabulary` | `GET /learning/srs/queue` `:687` `POST /learning/vocabulary/:id/review` `:688` `GET /learning/vocabulary/mined` `:691` | **PASS** | Vocabulary hub + mined queue |
| 65 | `payout_settings_history` | **GAP** — no `PayoutSettingsScreen` | **GAP** — no `/payouts` route | `GET /teachers/payouts/overview` `:604` `GET /teachers/payouts/history` `:605` `GET /teachers/payouts/methods` `:607` `POST /teachers/payouts/methods` `:608` `DELETE /teachers/payouts/methods/:methodId` `:609` `PUT /teachers/payouts/methods/:methodId/default` `:610` `POST /teachers/payouts/withdraw` `:606` `backend/internal/handlers/payout.go:1` | **GAP** | Full payouts backend; zero UI — HIGH PRIORITY |
| 66 | `placement_test_reading_comprehension` | `PlacementScreen.tsx:1` (step renders reading Q) | `Placement.tsx:1` (step renders reading Q) | `POST /learning/placement/:attemptId/answer` `:664` `GET /learning/placement/:attemptId` `:667` | **PASS** | Sub-step of placement flow |
| 67 | `placement_test_results_summary` | `PlacementScreen.tsx:1` (results state) | `Placement.tsx:1` (results) | same | **PASS** | Results sub-step |
| 68 | `placement_test_vocabulary_question` | `PlacementScreen.tsx:1` (vocab Q step) | `Placement.tsx:1` (vocab Q) | same | **PASS** | Vocab Q sub-step |
| 69 | `placement_test_welcome` | `PlacementScreen.tsx:1` (welcome card) + `LearnScreen.tsx:1` entry CTA | `Placement.tsx:1` (welcome) + `Learn.tsx:1` placement CTA | `POST /learning/placement/start` `:663` | **PASS** | Welcome/intro step |
| 70 | `profile_settings` | `ProfileScreen.tsx:1` `MainTabs.tsx:134` `name="Profile"` in `ProfileTab` | `Profile.tsx:1` `App.tsx:174` `/profile` + `Settings.tsx:1` | `GET /users/me/settings` `:473` `PUT /users/me/settings` `:474` `GET /users/me` `:465` `PUT /users/me` `:467` | **PASS** | Settings/profile |
| 71 | `real_talk_hub` | `RealTalkHubScreen.tsx:1` `MainTabs.tsx:124` `name="RealTalkHub"` + `StudySandbox.tsx:1` | `RealTalkHub.tsx:1` `App.tsx:170` `/learn/real-talk` + `StudySandbox.tsx:1` | `GET /learning/real-talk/prompts` `:705` `POST /learning/real-talk/prompts/:promptId/used` `:706` | **PASS** | Real Talk prompts hub |
| 72 | `real_world_scenario_practice` | `ScenariosScreen.tsx:1` + `ScenarioRoleplayScreen.tsx:1` | `Scenarios.tsx:1` + `ScenarioRoleplay.tsx:1` | `GET /learning/scenarios` `:696` `GET /learning/scenarios/:scenarioId` `:697` `POST /learning/scenarios/:scenarioId/start` `:698` | **PASS** | Scenario practice (duplicate of #8) |
| 73 | `report_user_content` | `ChatScreen.tsx:1` (report action) | `ReportModal.tsx:1` | `POST /reports` `:489` | PARTIAL | Action modal, not route |
| 74 | `safety_alert_contextual_guidance` | GAP — no safety alerts UI | GAP — no alerts banner | `backend/internal/services/moderation.go:1` | **GAP** | No contextual safety UI |
| 75 | `safety_moderation_alerts` | same | same | same | **GAP** | Same |
| 76 | `search_messages_media` | `NewChatScreen.tsx:1` (search) + `ChatListScreen.tsx:1` search | `SearchMessages.tsx:1` + `ChatList.tsx:1` search | `GET /messages/search` `:588` `GET /media/search` `:589` `GET /chats/search` `:590` | **PASS** | Message/media search |
| 77 | `settings` | `ProfileScreen.tsx:1` (settings section) | `Settings.tsx:1` + `PrivacySettings.tsx:1` | `GET /users/me/settings` `:473` `PUT /users/me/settings` `:474` | **PASS** | Settings (alias of #70) |
| 78 | `streak_recovery_challenge` | `StreakRecoveryScreen.tsx:1` `MainTabs.tsx:StreakRecovery` | `StreakRecovery.tsx:1` `App.tsx:/learn/streak-recovery` | `POST /learning/streak/recover` `:708` `backend/internal/services/learning_dashboard.go:1` | **PASS** | Recovery via `learningAPI.recoverStreak` + scenario/SRS options |
| 79 | `student_insights_progress_dashboard` | **GAP** — no insights dashboard | **GAP** | `GET /learning/dashboard` `:659` (learner) | **GAP** | Analytics-style insights not built |
| 80 | `student_management_progress_teacher` | **GAP** — no teacher student-list | **GAP** | `GET /teachers/dashboard` `:615` (includes upcoming + students) `backend/internal/services/teacher.go:633` + `GET /teachers/bookings` `:618` | **GAP** | Backend dashboard has upcoming/students but no UI |
| 81 | `study_session_recap` | `LessonSessionScreen.tsx:1` (completion state) | `LessonSession.tsx:1` (completion) | `POST /learning/sessions/:sessionId/complete` `:684` `POST /learning/lesson-attempts/:attemptId/complete` `:677` | **PASS** | Recap after session |
| 82 | `suggested_learning_partners` | GAP — no suggestions | GAP | `GET /contacts/search` `:591` (placeholder) | **GAP** | No partner suggestion engine/UI |
| 83 | `teacher_dashboard` | **GAP** — no `TeacherDashboardScreen` | **GAP** — no `/teacher` route | `GET /teachers/dashboard` `:615` `backend/internal/handlers/teacher.go:232` `backend/internal/services/teacher.go:633` | **GAP** | Highest-priority teacher GAP; backend checklist/earnings/upcoming/students ready |
| 84 | `teacher_earnings_overview` | **GAP** — no `EarningsScreen` | **GAP** — no `/teacher/earnings` | `GET /teachers/payouts/overview` `:604` `backend/internal/services/payout.go:107` + earnings in `GetDashboard` | **GAP** | Overlaps `payout_settings_history`; payout overview backend ready |
| 85 | `teacher_pronunciation_review_dashboard` | **GAP** | **GAP** | `GET /teachers/srs/pushes` `:627` `PUT /teachers/bookings/:id/review-notes` `:625` | **GAP** | Pronunciation review not implemented |
| 86 | `teacher_student_learning_chat` | `ChatScreen.tsx:1` (generic; teacher-student is just a chat) | `Chat.tsx:1` | `POST /chats` `:544` `GET /chats/:chatId/messages` `:564` | PARTIAL | No teacher-specific chat variant |
| 87 | `trending_community_hub` | **GAP** | **GAP** | — | **GAP** | No trending/community backend |
| 88 | `trial_credit_dashboard` | **GAP** — no `TrialCreditScreen` | **GAP** — no `/trial-credits` | `GET /teachers/trial-credits/dashboard` `:617` `GET /teachers/trial-credits` `:616` `backend/internal/handlers/teacher.go:123` `backend/internal/services/teacher.go:247` | **GAP** | Backend credits/history/nextGrant ready; no UI — HIGH PRIORITY |
| 89 | `trust_safety_advanced_controls` | GAP — no advanced controls | GAP — `Settings.tsx:1` + `PrivacySettings.tsx:1` basic only | `POST /blocks` `:485` `DELETE /blocks/:userId` `:486` | **GAP** | Advanced controls (filter sensitivity etc.) not built |
| 90 | `trust_safety_center` | GAP — no Trust Center screen | GAP — `About.tsx:1` is closest | `backend/internal/handlers/moderation.go:1` `backend/internal/handlers/gdpr.go:1` | **GAP** | No dedicated Trust & Safety center |
| 91 | `tutor_profile_sofia` | **GAP** — no `TutorProfileScreen` | **GAP** — no `/teachers/:id` route | `GET /teachers/:id` `:630` `GET /teachers/:id/reviews` `:631` `GET /teachers/:id/availability` `:633` `backend/internal/services/teacher.go:103` | **GAP** | Sofia is example `TutorProfile`; backend ready; UI missing — HIGH PRIORITY |
| 92 | `user_profile_mateo` | `ProfileScreen.tsx:1` (own profile only) | `Profile.tsx:1` (own profile) | `GET /users/me` `:465` `GET /users/search` `:469` | PARTIAL | No public `users/:id` profile page (Mateo is example other-user profile) |
| 93 | `video_call_learning_deep_dive` | `CallScreen.tsx:1` + `StudySandbox.tsx:1` overlay | `CallScreen.tsx:1` + `StudySandbox.tsx:1` | `POST /calls/:callId/captions` `:719` `GET /calls/:callId/captions` `:720` + word-mining queue | PARTIAL | Learning deep-dive during call is sandbox, not dedicated route |
| 94 | `welcome_to_chorus` | `LandingScreen.tsx:1` (welcome/landing) | `Landing.tsx:1` `App.tsx:117` `/` + `About.tsx:1` | — | **PASS** | Welcome/onboarding landing |

---

## Route + Screen Index (what exists)

### mobile/src

| Screen file | Route name (`MainTabs.tsx`) |
|---|---|
| `mobile/src/screens/LandingScreen.tsx:1` | auth stack (unauth) |
| `mobile/src/screens/LoginScreen.tsx:1` | auth stack |
| `mobile/src/screens/RegisterScreen.tsx:1` | auth stack |
| `mobile/src/screens/ForgotPasswordScreen.tsx:1` | auth stack |
| `mobile/src/screens/ResetPasswordScreen.tsx:1` | auth stack |
| `mobile/src/screens/ChatListScreen.tsx:1` | `MainTabs.tsx:64` `name="ChatList"` |
| `mobile/src/screens/ChatScreen.tsx:1` | `MainTabs.tsx:69` `name="Chat"` |
| `mobile/src/screens/NewChatScreen.tsx:1` | `MainTabs.tsx:74` `name="NewChat"` |
| `mobile/src/screens/CallScreen.tsx:1` | `MainTabs.tsx:79` `name="Call"` |
| `mobile/src/screens/LearnScreen.tsx:1` | `MainTabs.tsx:89` `name="Learn"` |
| `mobile/src/screens/PlacementScreen.tsx:1` | `MainTabs.tsx:94` `name="Placement"` |
| `mobile/src/screens/LessonSessionScreen.tsx:1` | `MainTabs.tsx:99` `name="LessonSession"` |
| `mobile/src/screens/VocabularyReviewScreen.tsx:1` | `MainTabs.tsx:104` `name="VocabularyReview"` |
| `mobile/src/screens/ScenariosScreen.tsx:1` | `MainTabs.tsx:109` `name="Scenarios"` |
| `mobile/src/screens/ScenarioRoleplayScreen.tsx:1` | `MainTabs.tsx:114` `name="ScenarioRoleplay"` |
| `mobile/src/screens/LearningRoadmapScreen.tsx:1` | `MainTabs.tsx:119` `name="LearningRoadmap"` |
| `mobile/src/screens/RealTalkHubScreen.tsx:1` | `MainTabs.tsx:124` `name="RealTalkHub"` |
| `mobile/src/screens/ProfileScreen.tsx:1` | `MainTabs.tsx:134` `name="Profile"` |
| `mobile/src/screens/BecomeTeacherScreen.tsx:1` | auth/protected stack (linked from Profile) |
| `mobile/src/screens/AboutScreen.tsx:1` | info stack |
| `mobile/src/screens/PricingScreen.tsx:1` | info stack |

### frontend/src

| Page file | Route (`frontend/src/App.tsx`) |
|---|---|
| `frontend/src/pages/Landing.tsx:1` | `App.tsx:117` `/` |
| `frontend/src/pages/Pricing.tsx:1` | `App.tsx:118` `/pricing` |
| `frontend/src/pages/Premium.tsx:1` | `App.tsx:119` `/premium` |
| `frontend/src/pages/Waitlist.tsx:1` | `App.tsx:120` `/waitlist` |
| `frontend/src/pages/AdminWaitlist.tsx:1` | `App.tsx:121` `/admin/waitlist`, `:135` `/admin`, `:142` `/admin/:tab` |
| `frontend/src/pages/Login.tsx:1` | `App.tsx:142` `/login` |
| `frontend/src/pages/Register.tsx:1` | `App.tsx:147` `/register` |
| `frontend/src/pages/ForgotPassword.tsx:1` | `App.tsx:152` `/forgot-password` |
| `frontend/src/pages/ResetPassword.tsx:1` | `App.tsx:157` `/reset-password` |
| `frontend/src/pages/Chat.tsx:1` | `App.tsx:125` `/chat`, `:129` `/chat/:slug` |
| `frontend/src/pages/Learn.tsx:1` | `App.tsx:132` `/learn` |
| `frontend/src/pages/Placement.tsx:1` | `App.tsx:134` `/learn/placement` |
| `frontend/src/pages/LessonSession.tsx:1` | `App.tsx:138` `/learn/session` |
| `frontend/src/pages/VocabularyReview.tsx:1` | `App.tsx:148` `/learn/vocabulary` |
| `frontend/src/pages/Scenarios.tsx:1` | `App.tsx:143` `/learn/scenarios` |
| `frontend/src/pages/ScenarioRoleplay.tsx:1` | `App.tsx:155` `/learn/scenarios/:scenarioId` |
| `frontend/src/pages/LearningRoadmap.tsx:1` | `App.tsx:168` `/learn/roadmap` |
| `frontend/src/pages/RealTalkHub.tsx:1` | `App.tsx:170` `/learn/real-talk` |
| `frontend/src/pages/Profile.tsx:1` | `App.tsx:174` `/profile` |
| `frontend/src/pages/Settings.tsx:1` | via `/profile` (settings tab in `Profile.tsx`) `frontend/src/components/PrivacySettings.tsx:1` |
| `frontend/src/pages/BecomeTeacher.tsx:1` | `App.tsx:183` `/become-teacher` |
| `frontend/src/pages/About.tsx:1` | informational (linked from landing) |

---

## Recommendations (prioritized)

1. **Teacher marketplace UI (P0)** — build `BrowseTutors` + `TutorProfile (Sofia)` + `ConfirmTrialBooking` + `TrialCreditDashboard` on both platforms; routes `GET /teachers/browse` `:614`, `GET /teachers/:id` `:630`, `POST /teachers/:id/book` `:634`, `GET /teachers/trial-credits/dashboard` `:617` are already live.
2. **Teacher dashboard + payouts (P0)** — `TeacherDashboard` (`GET /teachers/dashboard` `:615`) and `PayoutSettingsHistory` / `TeacherEarningsOverview` (`GET /teachers/payouts/overview` `:604` `GET /teachers/payouts/history` `:605` `/methods` `:607`) need screens on both platforms.
3. **Activity Hub + Progress dashboards (P1)** — either promote `activity_hub_fixed` to `/learn/activity` or document as intentional embed; add `MyLearningProgressStudent` / `StudentInsights` as dashboard tabs backed by `GET /learning/dashboard` `:659` + `GET /vocabulary/progress` `:651`.
4. **Group / Community (P2)** — `group_study_hub`, `learning_community_feed`, `trending_community_hub` have no backend; decide drop vs. spec new `study-groups` + `community` APIs before building UI.
5. **Trust & Safety polish (P2)** — add `TrustSafetyCenter` page aggregating blocks/reports/retention (`GET /blocks` `:487`, `GET /privacy/retention-policy` `:478`) and expose `moderator_queue` on mobile for moderators.
6. **Wireframe hygiene** — `image.png` stray file and empty `linguist_flow` / `languagelearning` folders should be removed or documented as N/A.

---

## Addendum 2026-09-03 — Gap Closure S-HOME-01..04 + S-T-01..06 (BA+QA Sign-off, NOT yet DONE)

> **Authority:** `wireframes/chorus_home_desktop_v2/code.html:134` Home v2 canonical + `docs/REQUIREMENTS_SLICE_HOME_V2.md:1` (S-HOME-01..04) + `docs/REQUIREMENTS_SLICE_MARKETPLACE.md:1` (S-T-01..06) + `docs/GAP_SIGNOFF.md:1` + `docs/CREWAI_GAP_CLOSURE_PLAN.md:39` TDD loop + `crew/roles.py:22` analyst sign-off
> **Date:** 2026-09-03 | **BA:** analyst (`crew/roles.py:22`) | **QA:** qa_engineer (`crew/roles.py:54`) | **Builds:** `go vet 0`, `go test 0`, `frontend tsc 0 && vite build 0 && vitest 205 pass`, `mobile tsc 0 && jest 96 pass`, `e2e --list 143`
> **Effect:** Home v2 remains **PASS** with v2 annotation; Marketplace **6 slices (9 rows) flip GAP → PASS**. Original audit rows above are **preserved for history** — this addendum is the flip record (audit trail, do not overwrite audit). Summary counts below are post-flip.

### Updated Summary (post-flip)

| Metric | Before (2026-09-01 audit) | After (2026-09-03 addendum) | Δ |
|---|---|---|---|
| Total wireframe entries | 94 | 94 | — |
| Fully implemented (both platforms have screen+route) | 18 | **27** (+9 marketplace) | +9 |
| Partial (one platform or backend-only) | 14 | 14 | — |
| GAP — no screen/route on either platform | 62 | **53** | −9 |
| Backend API exists but no UI | 11 | 11 | — (backend unchanged) |

| Area | Before | After | Note |
|---|---|---|---|
| Teacher marketplace | **1/13** (`become_a_teacher` only) | **10/13** | 9 rows flipped: `browse_tutors`, `find_a_trial_tutor`, `tutor_profile_sofia`, `confirm_trial_booking`, `trial_credit_dashboard`, `teacher_dashboard`, `teacher_earnings_overview`, `payout_settings_history`, `student_management_progress_teacher` (students section of dashboard) — only non-marketplace deferred gaps remain (`teacher_student_learning_chat` PARTIAL, `custom_activity_builder_pusher`, `teacher_pronunciation_review_dashboard` etc. out of S-T scope) |
| Learning dashboard | 7/12 | 7/12 | No change (Activity Hub P1 deferred) |

### Home — already PASS, now annotated as v2

| # | Wireframe folder | Status before | Status after + note | Impl citations |
|---|---|---|---|---|
| 27 | `chorus_home` | **PASS** | **PASS (v2)** — `implemented as v2 — Communication is Learning + 4-card ecosystem + $7.99/mo pricing (BA 2026-09-03)` | `frontend/src/pages/Landing.tsx:7` hero `:42` `Communication is Learning.` + brain `:64` + ecosystem `:87` + pricing `:138` + mission `:213` + footer `:239`; `mobile/src/screens/LandingScreen.tsx:68` TopNav + `:99` hero + `:133` bridging + `:143` ecosystem + `:176` pricing + `:216` mission + `:235` footer |
| 28 | `chorus_home_desktop` | **PASS** | **PASS (v2)** — same | `frontend/src/pages/Landing.tsx:1` responsive via same route `/` `App.tsx:127` |
| 29 | `chorus_home_desktop_v2` | **PASS** | **PASS (v2) — CANONICAL** — `wireframes/chorus_home_desktop_v2/code.html:134` TopNav `:115` + Hero `:134` + Bridging `:164` + Ecosystem `:174` + Pricing `:221` + Mission `:295` + Final CTA `:304` + Footer `:317` — single source of truth | `frontend/src/pages/Landing.tsx:38` `:74` `:87` `:138` `:213` `:224` `:239`; `mobile/src/screens/LandingScreen.tsx:52` `:100` `:133` `:143` `:176` `:216` `:225` `:235` |
| 30 | `chorus_home_desktop_waitlist_product_previews` | **PASS** | **PASS (v2)** | `Landing.tsx:1` + `Waitlist.tsx:1` |
| 31 | `chorus_home_fixed` | **PASS** | **PASS (v2)** | `LandingScreen.tsx:1` |
| 32 | `chorus_home_mobile` | **PASS** | **PASS (v2)** | `LandingScreen.tsx:1` responsive |
| 33 | `chorus_home_mobile_v2` | **PASS** | **PASS (v2)** — canonical mobile | `LandingScreen.tsx:52-262` full v2 |
| 34 | `chorus_home_mobile_waitlist_product_previews` | **PASS** | **PASS (v2)** | `LandingScreen.tsx:1` |

> Original rows 27-34 above remain **PASS** — addendum only adds `v2` annotation + file:line citations. No GAP flip needed for Home.

### Marketplace — 9 rows GAP → PASS (6 slices S-T-01..06)

| # | Wireframe folder (audit #) | Audit status (2026-09-01) | Post-addendum status | Slice | Impl file:line citations (both surfaces + backend) | QA testRefs green |
|---|---|---|---|---|---|---|
| 13 | `browse_tutors` | **GAP** — no `BrowseTutorsScreen` / no `/tutors` route | **PASS** (S-T-01) | S-T-01 Browse Tutors + Find a Trial Tutor | `frontend/src/pages/BrowseTutors.tsx:1` (`h2 Tutors` `:46`, `tutor-search` `:50`, filters `Language` `:54` `Price` `:65` `Rating` `:74`, `Featured Tutors` `:94`, `Available Now` `:114`, `$` per session `:106`) `frontend/src/App.tsx:247` `/tutors`; `mobile/src/screens/BrowseTutorsScreen.tsx:1` (`Featured Tutors` `:84`, `Available Now` `:106`, filter chips `:57`, `Verified` `:95`, `$/session` `:97`) `mobile/src/components/MainTabs.tsx:183` `MarketplaceTab/BrowseTutors`; `backend/cmd/server/main.go:648` `GET /teachers/browse` `backend/internal/services/teacher.go:132` `BrowseTutors` | `e2e/tests/tutor-browse.spec.ts:14` nav+wireframe+mobile parity; `frontend/src/__tests__/marketplace.slices.test.tsx` + `mobile/__tests__/MarketplaceSlices.test.tsx` |
| 42 | `find_a_trial_tutor` | **GAP** — no `FindTrialTutorScreen` | **PASS** (S-T-01 filtered variant) | S-T-01 | **Same screen** filtered — `BrowseTutors.tsx:21` `teacherAPI.browse({search})` + `GET /teachers/trial-credits:650` `teacher.go:232` badge; no separate route, documented as `?filter=trial` | `tutor-browse.spec.ts:27` filtered variant |
| 91 | `tutor_profile_sofia` | **GAP** — no `TutorProfileScreen` | **PASS** (S-T-02) | S-T-02 Tutor Profile — Sofia | `frontend/src/pages/TutorProfile.tsx:1` (`Sofia Tutor` `:51`, `Verified` `:51`, `About` `:57` `Hola! I am Sofia` `:58`, `Reviews` `:62`, `Pricing Options` `:65` Single `$25` `:73` / Monthly `$80`, `Booking calendar` `:90` Oct 16-22, `book-trial` `:143`) `App.tsx:248` `/tutors/:id`; `mobile/src/screens/TutorProfileScreen.tsx:1` (`Verified` + rating + bio + reviews + `Book Trial`); `backend/cmd/server/main.go:664` `GET /teachers/:id` `teacher.go:105` + `:665` reviews + `:667` availability | `e2e/tests/tutor-profile.spec.ts:13` Sofia hero + Verified + `book-trial` → Confirm |
| 39 | `confirm_trial_booking` | **GAP** — no confirm-booking UI | **PASS** (S-T-03) | S-T-03 Confirm Trial Booking | `frontend/src/pages/ConfirmBooking.tsx:1` (`Confirm Booking` `:51`, `Great choice!` `:56`, tutor card `:59`, Date/Time `:68-76`, `Payment Summary` `:79` `Trial Session 1 Credit` `:80` `Credits Applied -1` `:81` `Total $0.00` `:83`, `Cancellation Policy 24h` `:86`, `confirm-booking` `:92` sticky) `App.tsx:249` `/tutors/:id/confirm`; `mobile/src/screens/ConfirmBookingScreen.tsx:1`; `backend/cmd/server/main.go:668` `POST /teachers/:id/book` `teacher.go:416` `isTrial` | `e2e/tests/tutor-booking.spec.ts:13` `Great choice` + `$0.00` + `confirm-booking` |
| 88 | `trial_credit_dashboard` | **GAP** — no `TrialCreditScreen` | **PASS** (S-T-04) | S-T-04 Trial Credit Dashboard | `frontend/src/pages/TrialCredits.tsx:1` (`Trial Credits` `:40` star, credits large `:41`, `Available to use right now` `:42`, `Next credit:` `:43` when 0, `Find a Tutor` `:44`, `How Trials Work` `:51` 20 Minutes/Meet & Greet, `Recommended for Trials` `:65`, `History` `:84`) `App.tsx:250` `/trial-credits`; `mobile/src/screens/TrialCreditsScreen.tsx:1`; `backend/cmd/server/main.go:651` `GET /teachers/trial-credits/dashboard` `teacher.go:249` | `e2e/tests/trial-credits.spec.ts:13` credits card + CTA + History |
| 83 | `teacher_dashboard` | **GAP** — no `TeacherDashboardScreen` | **PASS** (S-T-05) | S-T-05 Teacher Dashboard | `frontend/src/pages/TeacherDashboard.tsx:1` (`Teacher Dashboard` `:29`, `Welcome back!` `:33` `Accepting New Students` `:34`, `Earnings Overview` `:39` Total/Pending/Fee 3 cols, `Premium Program` `:46`, `Availability` `:51` 3 slots, `Recent Students` `:58`, `Profile Completion — {pct}%` `:64` bar `:65` checklist `:67`) `App.tsx:251` `/teacher/dashboard`; `mobile/src/screens/TeacherDashboardScreen.tsx:1`; `backend/cmd/server/main.go:649` `GET /teachers/dashboard` `teacher.go:635` | `e2e/tests/teacher-dashboard.spec.ts:13` Welcome + Earnings + Availability + Students + Profile Completion |
| 80 | `student_management_progress_teacher` | **GAP** — no teacher student-list | **PASS** (S-T-05 via same dashboard) | S-T-05 (students section) | Same `TeacherDashboard.tsx:58-61` `dash.students` via `teacher.go:751` students array | `teacher-dashboard.spec.ts` |
| 84 | `teacher_earnings_overview` | **GAP** — no `EarningsScreen` | **PASS** (S-T-06 earnings alias) | S-T-06 Payout Settings & History | `frontend/src/pages/Payouts.tsx:1` merged with `payout_settings_history` — earnings via `This Month's Breakdown` `:67` Gross/Fee/Net + `Performance Insight` `:84` `12% more` `Hours Taught` `Active Students` `App.tsx:252` `/teacher/payouts`; `mobile/src/screens/PayoutsScreen.tsx:1`; `backend/cmd/server/main.go:638` `GET /teachers/payouts/overview` `payout.go:107` | `e2e/tests/payouts.spec.ts:13` alias via same Payouts |
| 65 | `payout_settings_history` | **GAP** — no `PayoutSettingsScreen` | **PASS** (S-T-06) | S-T-06 Payout Settings & History | `frontend/src/pages/Payouts.tsx:1` (`Payout Settings & History` `:40`, `Total Lifetime Earnings` `:47` `Available for payout` `:48`, `Withdraw Funds →` `:49`, `Payout Methods` `:52` list + `paypal`/`bank` add `:60` `Add` `:63`, `This Month's Breakdown` `:67`, `Withdraw` `:75-81` amount + Available/Pending, `Performance Insight` `:84`, `Payout History` `:91`) `App.tsx:252`; `mobile/src/screens/PayoutsScreen.tsx:1`; `backend/cmd/server/main.go:641` `GET /teachers/payouts/methods` + `:642` POST + `:643` DELETE + `:644` PUT default + `:640` POST withdraw + `:639` GET history `payout.go:107-409` | `payouts.spec.ts` Lifetime Earnings + Methods + Breakdown + Withdraw + History |

> **Navigation verified:** `frontend/src/App.tsx:247-253` six routes guarded `isAuthenticated ? <Page/> : <Navigate to="/login"/>`; `mobile/src/components/MainTabs.tsx:180-202` `MarketplaceTab` stack six screens + 4th tab `Tutors` `MainTabs.tsx:207` `label:Tutors glyph:🏫` (`TabIconMarketplace`). **Backend already live** since `main.go:638-668` + seed `dev_seed.go:79` Sofia (approved 2500c, 4 slots, 2 reviews, alice trial credit).

### Audit trail note

- This addendum **does not overwrite** the original audit table (`#1-94` rows above) — original GAP values are preserved for history. Readers should interpret any row above that is still marked **GAP** but listed here as **PASS post-addendum**.
- Original `Recommendations 1-2` (P0 marketplace UI missing) are now **RESOLVED** via S-T-01..06; remaining Recommendations 3-6 (Activity Hub P1, Group P2, Trust & Safety P2, hygiene) are **unchanged / deferred**.
- BA sign-off: `docs/GAP_SIGNOFF.md:1` table + signature 2026-09-03 — **phase_status stays PENDING** until reviewer + SRE gates pass (`crew/state.py:97`).

### Files changed (git status at 2026-09-03 sign-off)

```
 M frontend/src/pages/BrowseTutors.tsx
 M frontend/src/pages/Landing.tsx
 M frontend/src/pages/Payouts.tsx
 M frontend/src/pages/TutorProfile.tsx
 M mobile/src/screens/BrowseTutorsScreen.tsx
 M mobile/src/screens/LandingScreen.tsx
 M mobile/src/screens/PayoutsScreen.tsx
 M mobile/src/screens/TeacherDashboardScreen.tsx
 M mobile/src/screens/TrialCreditsScreen.tsx
 M mobile/src/screens/TutorProfileScreen.tsx
 M mobile/tsconfig.json
 A docs/GAP_SIGNOFF.md                  ← BA+QA sign-off (this addendum references it)
?? docs/REQUIREMENTS_SLICE_HOME_V2.md   ← BA spec pre-existing
?? docs/REQUIREMENTS_SLICE_MARKETPLACE.md
?? e2e/tests/00-home.spec.ts
?? e2e/tests/payouts.spec.ts
?? e2e/tests/teacher-dashboard.spec.ts
?? e2e/tests/trial-credits.spec.ts
?? e2e/tests/tutor-booking.spec.ts
?? e2e/tests/tutor-browse.spec.ts
?? e2e/tests/tutor-profile.spec.ts
?? frontend/src/__tests__/marketplace.slices.test.tsx
?? frontend/src/pages/__tests__/Landing.test.tsx
?? mobile/__tests__/Home.test.tsx
?? mobile/__tests__/MarketplaceSlices.test.tsx
 M docs/WIREFRAME_TRACE.md              ← THIS ADDENDUM APPENDED
```


