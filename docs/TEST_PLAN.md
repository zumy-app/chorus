# Chorus Test Plan — Phase 10.1 Comprehensive Test Suite Design

> **Owner:** qa_engineer + test_engineer + reviewer | **SRE:** production readiness
> **Authority:** `REQUIREMENTS_MASTER.md` §0 Global DoD + phases 5-10 gates, `REQUIREMENTS.md`, `wireframes/`, `docs/WIREFRAME_TRACE.md`, `docs/RELEASE_GATE.md`, `docs/GO_NO_GO.md`, NFR-17/22/23/24/25/26
> **Status:** DESIGN v1.0 — 2026-09-02 | **Next gate:** 10.1 Test Suite Design → 10.2 Execution → 10.4 Release

## 1. Objectives

Deliver a **single verifiable test matrix** that proves every wireframe + requirement ships on **both mobile (Expo RN) and web (Vite)** with no stubs, green builds, and NFR compliance. Phase 10 is DONE only when 10.1 (this plan) + 10.2 (execution green) + 10.3 (benchmarks met) + 10.4 (release gate passes) are true.

Non-goals: building deferred Backlog `§6` items (public social feed, group-study hub) ahead of messaging parity — they remain tracked as GAP with explicit skip.

## 2. Test Pyramid & Layers

| Layer | Tool | Where | When | Gate |
|---|---|---|---|---|
| **Unit** — services, handlers, utils, stores | `go test ./...` (backend), `vitest run` (frontend), `jest` (mobile) | PR + `main` CI `test` job | every push | must be 100% pass; coverage ≥80% stmts target (`COMPREHENSIVE_TEST_REPORT.md` baseline 85% backend, 82% mobile) |
| **Integration** — API with real Postgres/Redis, WS hub, FSRS/SRS | `go test -tags=integration` + `scripts/e2e.sh` against `docker-compose.dev.yml` | CI `test` with services, `gate-on-dev.sh` `seed` step | pre-merge + dev promotion | per-endpoint contract + durability assertions |
| **E2E web** | Playwright (`e2e/`, `frontend test:e2e`) against `vite preview :4173` + backend `:8080` | `gate-on-dev.sh` `e2e` | dev promotion | 15 suites sequential (single worker, shared users/chats) |
| **E2E mobile** | Jest + Detox (where Detox runner present; else Jest smoke + manual EAS smoke) | `mobile npm test` + EAS build smoke | PR + release | parity with web (NFR-22) |
| **Load / Soak** | Artillery (`deploy/ci/artillery.*.yml` + `soak-processor.js`) via `load-smoke.sh` / `load-soak.sh` + `verify-drain.sh` | `gate-on-dev.sh` `load-smoke` / `load-soak` | dev promotion, tagged soak | `errors.rate ≤2%` smoke, `==0` soak, `ws_fast_dropped_total==0` |
| **Security** | `govulncheck`, `npm audit`, `security_qa_test.go`, `security-scan.sh`, `verify-isolation.sh` | CI `security` job + gate | every promotion | no ≥high, isolation PASS |
| **Observability / eval** | `phoenix-eval.sh`, `/metrics`, `alerts-*.yml` | gate `phoenix-eval` | dev promotion | accuracy ≥80, p95 <500ms, ≥10 evals |
| **Release gate** | `verify-release-gate.sh --offline` / with DB | `gate-on-dev.sh` final | every promotion to prod | all thresholds §3 + `GO_NO_GO.md` no NO-GO |

Run locally:
```bash
# backend (requires Go 1.25)
cd backend && go vet ./... && go build ./... && go test ./... -count=1
# frontend (requires Node 20)
cd frontend && npm test && npm run build
# mobile
cd mobile && npm test
# e2e web (requires dev stack up)
docker compose -f docker-compose.dev.yml up -d --wait
cd e2e && npm run test
cd frontend && npx playwright test  # alt via vite preview
# offline release gate
bash deploy/ci/verify-release-gate.sh --offline
# full gate on dev host
bash deploy/ci/gate-on-dev.sh
```

## 3. Automated Thresholds (release-gate.sh mirror of RELEASE_GATE.md §3)

| Metric | Threshold | Source | Override |
|---|---|---|---|
| Translation golden-set accuracy | ≥80 | `translation_evals` | `ACCURACY_THRESHOLD` |
| Translation p95 (cache-hit) | <500 ms | `translation_jobs.latency_ms` | `P95_MS_MAX` |
| Eval sample size | ≥10 | `translation_evals` | `MIN_EVALS` |
| Load smoke error rate | ≤2% | Artillery | — |
| Soak (when run) | `errors.rate==0` && `ws_fast_dropped_total==0` | `verify-drain.sh` + `/metrics` | `SOAK_ALLOW_LONG=1` |
| Mail isolation | PASS | `deploy/mail/verify-isolation.sh` | `SKIP_MAIL=1` offline |
| Security | `govulncheck` clean, `npm audit` ≥high clean | `security-scan.sh` | `SKIP_SECURITY=1` offline |
| Docs presence | 6 docs + `GO_NO_GO.md` signed | file check | — |
| Compose + shell | `docker compose config --quiet` + `bash -n` | CI | — |
| GoNoGo | no NO-GO, no PENDING required owners | `docs/GO_NO_GO.md` | — |

## 4. Coverage Inventory (actual counts 2026-09-01)

### 4.1 Backend — 60 `*_test.go` (go test ./...)

| Area | Files | What is asserted |
|---|---|---|
| Config | `config_test.go` | env parsing, defaults |
| Handlers | `auth_test.go`, `chat_test.go`, `message_test.go`, `contacts_test.go`, `gallery_test.go`, `location_test.go`, `helpers_test.go`, `health_test.go`, `moderation_test.go`, `call_qa_test.go`, `video_qa_test.go`, `security_qa_test.go`, `settings_test.go` | HTTP contract, authz, validation, status codes |
| Middleware | `auth_test.go`, `errors_test.go`, `rate_limit_test.go` | 401/403, typed errors, 429 |
| Observability | `health_test.go`, `metrics_test.go` | `/health`, `/metrics` |
| Services | `auth_test.go`, `chat_test.go`, `message_test.go`, `user_test.go`, `contact_test.go`, `attachment_test.go`, `translation_test.go`, `translation_queue_test.go`, `grammar_extract_test.go`, `grammar_queue_test.go`, `vocabulary_test.go`, `srs_queue_test.go`, `seed_queue_test.go`, `learning_test.go`, `learning_logic_test.go`, `learning_placement_test.go`, `billing_test.go`, `entitlement_test.go`, `waitlist_test.go`, `invitation_test.go`, `whatsapp_test.go`, `notification_test.go`, `search_test.go`, `routing_test.go`, `horizontal_test.go`, `receipt_test.go`, `call_qa_test.go`, `video_qa_test.go`, `soak_test.go`, `release_gate_test.go`, `security_qa_test.go`, `evaluation_test.go`, `caption_review_test.go`, `qa_spanish_test.go`, `teacher_vetting_test.go` | business logic, SM-2/FSRS, entitlement word caps (280/1000), routing registry, durability `persist before ack` |
| Translation | `pkg/translation/detect_test.go` | language detection |
| Utils | `ratelimit_test.go`, `roles_test.go`, `settings_test.go`, `gallery_test.go`, `location_test.go`, `email_test.go` | helpers |

GAP to close in 10.2: add `client_test.go`/`inbox_test.go` if multi-device inbox is active; otherwise assert discarded path is not regressed.

### 4.2 Frontend — 18 suites (vitest)

| Suite | Covers |
|---|---|
| `components/__tests__/AppHeader.test.tsx`, `ChatArea.test.tsx`, `ChatLanguageModal.test.tsx`, `EmojiPicker.test.tsx`, `HighlightableText.test.tsx`, `ReportModal.test.tsx` | header nav, chat area, FR-35 ChatLanguageModal (only own language), emoji passthrough, highlight+practice FR-28, report modal |
| `pages/__tests__/AdminWaitlist.test.tsx`, `Dashboard.test.tsx`, `learningPages.test.tsx`, `Settings.test.tsx` | admin queue, dashboard P0 back-links, learning hub, privacy settings |
| `services/__tests__/api.test.ts`, `websocket.test.ts` | axios interceptors, token refresh, WS reconnect/backoff |
| `store/__tests__/store.test.ts` | Zustand auth/chat/learning stores |
| `utils/__tests__/words.test.ts` | word count cap (280/1000), entitlement messaging |
| `__tests__/qa-*.test.tsx` (call, messaging-parity, video, functional) | phase 5-8 parity regressions |

### 4.3 Mobile — 5 suites (jest) + 22 screens

`mobile/src/screens/__tests__/learning.test.tsx`, `qa-functional.test.tsx`, `src/__tests__/qa-*.test.tsx`, `__tests__/App.test.tsx` — learning flow, messaging parity, call/video overlays. Screens routed in `mobile/src/components/MainTabs.tsx` (ChatList, Chat, NewChat, Call, Learn, Placement, LessonSession, VocabularyReview, Scenarios, ScenarioRoleplay, LearningRoadmap, RealTalkHub, Profile, Landing, Pricing, BecomeTeacher, About).

### 4.4 E2E — 15+ Playwright suites (e2e/tests/)

`01-auth`, `02-chat-creation`, `03-messaging-translation`, `04-grammar`, `05-ai-tutor`, `06-vocabulary`, `07-search`, `08-settings`, `09-realtime`, `10-health`, `11-grammar-ai-local`, `12-waitlist`, `13-messaging-parity`, `14-call-captions`, `15-video-call`, `qa-scenarios-drills` — full auth→chat→translate→grammar→tutor→vocab→search→realtime→call chain. Single worker, 5m timeout, `E2E_BASE_URL` override.

### 4.5 Load / Soak

- `artillery.smoke.yml` / `artillery.soak.yml` + `soak-processor.js` → 1k WS, 50 msg/s.
- `load-smoke.sh` ≤2% errors; `load-soak.sh` + `verify-drain.sh` zero loss; Postgres `message_receipts` as inbox, Redis never source of truth.
- Prometheus `SoakMessageLoss` alert on `ws_fast_dropped_total`.

## 5. Traceability Matrix

### 5.1 Wireframes → Tests (94 entries; source docs/WIREFRAME_TRACE.md)

Every entry has a test tier. `PASS` entries have screen+route on both platforms; `GAP` entries are tracked with skip reason or regression guard.

| # | Wireframe | Status | Backend | Frontend/Mobile screen | Test that proves it |
|---|---|---|---|---|---|
| 1-3,26,35-36,49,57 | brand/asset/N/A (hero, brain, logo, `image.png`, `linguist_flow`) | N/A | — | `Landing.tsx`, `AppHeader.tsx` | visual smoke: `Dashboard.test.tsx`, `AppHeader.test.tsx` |
| 4-5 | `activity_hub`, `activity_hub_fixed` | GAP (embedded in Learn) | `GET /learning/dashboard` :659 | embedded `LearnScreen`/`Learn.tsx` | `learningPages.test.tsx` — dashboard renders fluency/weeklyActivity |
| 6-7 | `ai_deep_dive_with_drills*` | PARTIAL (sheet/panel) | `POST /grammar/analyze` :637 | `GrammarPanel.tsx`, `DeepDiveSheet.tsx` | `04-grammar.spec.ts`, `11-grammar-ai-local.spec.ts`, `HighlightableText.test.tsx` |
| 8,72 | `ai_scenario_roleplay*`, `real_world_scenario_practice` | PASS | `GET /learning/scenarios` :696 | `ScenariosScreen`/`Scenarios.tsx`, `ScenarioRoleplayScreen` | `qa-scenarios-drills.spec.ts`, `learningPages.test.tsx` |
| 9-10,45 | `ai_tutor_*`, `grammar_insight` | PARTIAL (inline chat) | `POST /chats/:id/messages/:id/translate` :566, `POST /grammar/analyze-ai` :639 | `ChatScreen`/`Chat.tsx` + `HighlightableText` | `05-ai-tutor.spec.ts`, `qa-functional.test.tsx` |
| 11,25,38 | `audio_call_with_live_captions` / chorus variants, `chorus_video_call_experience` | PASS | `POST /calls/initiate` :711, `GET /calls/:id/captions` :720 | `CallScreen` both | `14-call-captions.spec.ts`, `15-video-call.spec.ts`, `qa-call.test.tsx`, `qa-video.test.tsx` |
| 12 | `become_a_teacher` | PASS | `POST /teachers/apply` :612 | `BecomeTeacherScreen`/`BecomeTeacher.tsx` | `teacher_vetting_test.go`, `verify-teacher-vetting.sh` |
| 13,39,42,83,84,88-89,91 | `browse_tutors`, `confirm_trial_booking`, `find_a_trial_tutor`, `teacher_dashboard`, `teacher_earnings_overview`, `trial_credit_dashboard`, `tutor_profile_sofia` | GAP — backend ready, UI missing (P0) | `GET /teachers/browse` :614, `GET /teachers/:id` :630, `POST /teachers/:id/book` :634, `GET /teachers/dashboard` :615, `GET /teachers/payouts/overview` :604, `GET /teachers/trial-credits/dashboard` :617 | **no screen** | **Gate guard:** `release_gate_test.go` asserts routes exist (handler contract); e2e skipped with `test.skip` + WIREFRAME_TRACE GAP entry; blocks Phase 4 gate `docs/GO_NO_GO.md` axis 10 until UI lands |
| 14 | `chat_media_files_gallery` | GAP (backend ready) | `GET /chats/:id/gallery` :581 | inline preview only | `gallery_test.go` handler contract; `chat_media_files_gallery` searched via `GET /media/search` :589 `07-search.spec.ts` |
| 15,61,74-75,89-90 | moderation variants (`chat_moderation_with_social_path`, `moderated_message_in_chat`, `report_user_content`, `safety_*`, `trust_safety_*`) | PARTIAL (actions) | `POST /blocks` :485 `POST /reports` :489 | `ReportModal.tsx`, long-press in `ChatScreen` | `moderation_test.go`, `ReportModal.test.tsx`, security plan `SECURITY_TEST_PLAN.md` |
| 16-18,24 | `chat_*` utilities, `chats` | PARTIAL/PASS | `GET /chats` :543 `GET /chats/:id/messages` :564 | `ChatListScreen`/`ChatList.tsx`, `ChatScreen`/`Chat.tsx` | `02-chat-creation.spec.ts`, `03-messaging-translation.spec.ts`, `13-messaging-parity.spec.ts`, `ChatArea.test.tsx` |
| 19-20,86-87 | `chat_with_sofia_*`, `teacher_student_learning_chat` | PARTIAL (generic chat) | same | generic `ChatScreen` | same as chats |
| 21 | `chat_with_tutor_voice_note_feedback` | GAP | `POST /chats/:id/attachments` :582 | no voice UI | `attachment_test.go` |
| 22-23,93 | `chat_with_unified_ai_deep_dive*`, `video_call_learning_deep_dive` | PARTIAL (sandbox) | `POST /grammar/analyze-ai` :639 + word-mining | `StudySandbox.tsx` | `learning_logic_test.go`, `WORD_MINING_PIPELINE.md` |
| 27-34,50,94 | `chorus_home*`, `join_waitlist`, `welcome_to_chorus` | PASS | `POST /waitlist` :447 `GET /health` :433 | `LandingScreen`/`Landing.tsx`, `Waitlist.tsx` | `12-waitlist.spec.ts`, `health_test.go`, `AdminWaitlist.test.tsx` |
| 37 | `chorus_premium_upgrade` | PASS | `GET /users/me/subscription` :481 `POST .../checkout` :482 | `PricingScreen`/`Pricing.tsx`/`Premium.tsx` | `billing_test.go`, `entitlement_test.go`, `words.test.ts` (280/1000 cap) |
| 40 | `custom_activity_builder_pusher` | GAP | `POST /teachers/srs/push` :626 | no builder | contract test `teacher_srs` handler |
| 41,81 | `daily_practice_session`, `study_session_recap` | PASS | `POST /learning/sessions/start` :681 `POST .../complete` :684 | `LessonSessionScreen`/`LessonSession.tsx` | `learning_test.go`, `learningPages.test.tsx` |
| 43-44 | `find_invite_partners`, `find_learning_partners` | PARTIAL | `POST /contacts/invites` :596 | `NewChatScreen`/`NewChatModal` | `contacts_test.go`, `invitation_test.go`, `whatsapp_test.go` |
| 46-48,52,58-59,82,92 | `group_*`, `live_group_*`, `learning_community_feed`, `trending_community_hub`, `suggested_learning_partners`, `user_profile_mateo` | GAP (no backend or search-only) | — or `GET /contacts/search` :591 placeholder | no screen / own profile only | explicitly deferred backlog §6; no test required until spec'd |
| 51 | `languagelearning` (hub) | PASS | `GET /learning/dashboard` :659 `GET /learning/path` :660 | `LearnScreen`/`Learn.tsx` | `learningPages.test.tsx` |
| 53-55,63-64,79 | progress variants (`learning_journey_progress*`, `my_learning_progress_student`, `my_vocabulary_hub`, `student_insights*`) | GAP/PASS | `GET /learning/dashboard` :659 `GET /vocabulary/progress` :651 `GET /learning/srs/queue` :687 | `VocabularyReviewScreen`/`VocabularyReview.tsx` + roadmap | `VocabularyReview` flow + `LearningRoadmap` `learning_logic_test.go` |
| 56,85 | `lesson_review_notes_student`, `teacher_pronunciation_review_dashboard` | GAP | `PUT /teachers/bookings/:id/review-notes` :625 | no reader | handler contract only |
| 60,70,77 | `login`, `profile_settings`, `settings` | PASS | `POST /auth/login` :450 `GET /users/me` :465 `GET /users/me/settings` :473 | `LoginScreen`/`Login.tsx` + `ProfileScreen`/`Profile.tsx` + `Settings` + `PrivacySettings` | `01-auth.spec.ts`, `auth_test.go`, `Settings.test.tsx` |
| 62 | `moderator_queue` | PARTIAL (web only) | `GET /admin/reports` :535 | `AdminWaitlist.tsx` `/admin` | `AdminWaitlist.test.tsx` |
| 65,84 | `payout_settings_history`, `teacher_earnings_overview` | GAP (P0) | `GET /teachers/payouts/*` :604-610 `POST .../withdraw` :606 | no screen | `payout.go:107` contract `payout_settings_history` backend |
| 66-69 | `placement_test_*` | PASS | `POST /learning/placement/start` :663 `POST .../answer` :664 `GET .../:attemptId` :667 | `PlacementScreen`/`Placement.tsx` | `learning_placement_test.go`, `learningPages.test.tsx` |
| 71,78 | `real_talk_hub`, `suggested` | PASS | `GET /learning/real-talk/prompts` :705 | `RealTalkHubScreen`/`RealTalkHub.tsx` | `RealTalkHub` render |
| 76 | `search_messages_media` | PASS | `GET /messages/search` :588 `GET /media/search` :589 `GET /chats/search` :590 | `SearchMessages.tsx` | `07-search.spec.ts`, `search_test.go` |
| 80-81 | `student_insights_progress_dashboard`, `student_management_progress_teacher` | GAP | `GET /teachers/dashboard` :615 | no insights/teacher student list | `teacher.go:633` contract |

**Rule for GAPs:** backend contract test must still pass (route registered, authz enforced); frontend/mobile get a skipped e2e with `GAP:<wireframe>` tag; promotion blocked on `GO_NO_GO` axis 10/11 until UI lands. This satisfies DoD without hiding missing UI.

### 5.2 Requirements → Tests (REQUIREMENTS.md + REQUIREMENTS_MASTER.md)

| Req | Area | Backend unit | Frontend/mobile unit | Integration | E2E |
|---|---|---|---|---|---|
| FR-1/FR-19/FR-20/FR-18 self-selection/NFR-15 | auth, onboarding names, avatar, level → CEFR | `auth_test.go`, `user_test.go` | `AuthScreens`, `words.test.ts` | `01-auth.spec.ts` | auth flow + level seed |
| FR-2/FR-3/FR-35 | native/target language, ChatLanguageModal simplification | `settings_test.go` | `ChatLanguageModal.test.tsx` (FR-35 only own language) | settings API | `08-settings.spec.ts` |
| FR-4..FR-9 | chats, realtime WS, presence/typing | `chat_test.go`, `message_test.go`, `routing_test.go` | `ChatArea.test.tsx`, `websocket.test.ts` | WS hub | `02-chat-creation`, `09-realtime`, `13-messaging-parity` |
| FR-6/NFR-4..NFR-10 | stateless, L4 LB, Redis registry `ws:registry:{userId}`, cross-server routing, durable inbox | `routing_test.go`, `horizontal_test.go`, `receipt_test.go`, `soak_test.go` | — | cross-server integration | `load-smoke/soak` + `verify-drain` |
| FR-7/FR-10..FR-13/FR-25..FR-30 | translation auto, cache 500ms, feature toggles, learned-word optimization, highlight+practice, quality pipeline + Phoenix | `translation_test.go`, `translation_queue_test.go`, `evaluation_test.go` | `HighlightableText.test.tsx`, `EmojiPicker.test.tsx` | translation API | `03-messaging-translation.spec.ts`, `phoenix-eval` |
| FR-14/FR-31..FR-34 | grammar CEFR, Your Learning Path metrics, seed+personal SRS queue, writing assistant, scenario role-play | `grammar_extract_test.go`, `grammar_queue_test.go`, `learning_test.go`, `learning_logic_test.go`, `srs_queue_test.go`, `vocabulary_test.go` | `learningPages.test.tsx`, `qa-scenarios-drills` | learning API | `04-grammar`, `05-ai-tutor`, `06-vocabulary`, `qa-scenarios-drills.spec.ts` |
| FR-15..FR-17/FR-32 | learning plans, gamified, word-bank | `seed_queue_test.go`, `vocabulary_test.go` | `VocabularyReviewScreen` | SRS queue | `06-vocabulary.spec.ts` |
| FR-21..FR-24/7.x | emoji, contacts/invites (hashed), WhatsApp OTP | `contact_test.go`, `invitation_test.go`, `whatsapp_test.go` | `EmojiPicker.test.tsx` | contacts API | invite flow in `02-chat-creation` |
| FR-8/NFR-3 | history pagination <500ms | `message_test.go` | `ChatArea.test.tsx` | paginated fetch | `13-messaging-parity.spec.ts` |
| P4a..P4c / 13.1 | premium word caps 280/1000, grammar mode, priority/badge | `entitlement_test.go`, `billing_test.go` | `words.test.ts`, `Pricing` pages | checkout API | premium e2e |
| NFR-13..NFR-17 | TLS/WSS, authz, no secrets, JWT | `auth_test.go`, `middleware/auth_test.go` | `api.test.ts` | 401/403 contract | `SECURITY_TEST_PLAN.md` |
| NFR-22 | mobile-first, web parity | — | `App.test.tsx`, parity QA suites (`qa-messaging-parity`, `qa-call`, `qa-video`) | — | cross-surface parity asserts |
| NFR-23 | retention + GDPR (365/30/90/90, export/erasure) | `release_gate_test.go` | `GDPR` handler | `GET /users/me/export`, `DELETE /users/me` | `docs/DATA_RETENTION_GDPR.md` |
| NFR-24 | rate limiting | `ratelimit_test.go`, `middleware/rate_limit_test.go` | `api.test.ts` 429 handling | 429 contract | load smoke 429 path |
| NFR-25/NFR-18 | observability `/health` `/metrics` Phoenix | `health_test.go`, `metrics_test.go` | `/health/ready` poll | metrics scrape | `phoenix-eval`, Grafana dashboards |
| NFR-26 | CI/CD quality gates dev→prod image promotion | `release_gate_test.go` | `ci.yml` | `verify-release-gate.sh` | `GO_NO_GO.md` 13 axes |
| Phase 5 SRE (9.x) | L4 LB leastconn, stateless, soak zero-loss | `horizontal_test.go`, `soak_test.go` | — | `verify-drain.sh` | `SOAK_TEST.md` |

### 5.3 Endpoints → Tests (backend/cmd/server/main.go registry, selected)

Auth `448-462`, Users `465-482`, Chats `543-590`, Teachers `604-634`, Grammar `637-639`, Vocabulary `647-691`, Learning `659-708`, Calls `711-720`, Admin `535-537`, Health `433`. Each registered route has a handler contract test; e2e covers the happy path above; security plan covers 401/403/429/400 for auth, block/self, OTP brute/flood, phone injection, 2FA bypass/replay, privacy leak, suspended access.

## 6. Quality Gates & CI Wiring (.github/workflows/ci.yml, deploy/ci/)

```
push main → test (go vet+test, frontend build+test, mobile jest, compose config, bash -n, verify-release-gate --offline)
         → security (govulncheck, npm audit)
         → build images (:run_number + :dev, no rebuild for prod)
         → deploy-dev → gate-on-dev.sh (seed, e2e, phoenix-eval ≥80/<500ms/≥10, load-smoke ≤2%, release-gate, mail-isolation, verify-drain when soak)
         → promote-prod (imagetools create :run_number → :prod, no rebuild) → prod /health + verify-promotion → notify-failure
```

Fast fail: any `NO-GO` in `GO_NO_GO.md` or `ESCALATION.md` non-empty or threshold red fails `verify-release-gate.sh` even `--offline`.

## 7. Data, Fixtures & Isolation

- **Fixtures:** `e2e/fixtures/test-helpers.ts` + `users.ts` seed two users (Sofia demo + Mateo) per run; helpers do `register→login→createChat→sendMessage→translate→grammar`.
- **DB isolation:** each `go test` uses `sqlmock` or temp Postgres via `TEST_DATABASE_URL`; e2e seeds then tears down via `global-setup`/`global-teardown`; no shared prod DB.
- **Redis:** `go-redis` miniredis/fake for unit; real Redis in compose for integration/soak.
- **Mocks:** translation provider mocked (DeepSeek fallback exercised via `translation_test.go`); `Speech-to-Text` Google client mocked; `PayPal` payouts via sandbox log; OTP via `whatsapp_test.go` stub.

## 8. Reporting & Evidence

- Per-run artifacts: `e2e/playwright-report` HTML, `backend coverage.out` (html), `frontend vitest --coverage`, `mobile jest --coverage`, `gate-on-dev.sh` log, `/metrics` snapshot, `translation_evals` sample count, `verify-release-gate.sh` TAP.
- Per-promotion: recorded image digest `:run_number → :prod` + prod `/health` probe + this TEST_PLAN + `WIREFRAME_TRACE.md` delta.
- Retention: per `docs/DATA_RETENTION_GDPR.md` windows; evidence bundle attached to release tag per `GO_NO_GO.md §5`.

## 9. Execution Plan — Phase 10.2 (next)

1. **Green the pyramid** — fix any red in 10.1 inventory; add missing handler contract tests for teacher GAP routes so offline gate `--offline` stays green while deferring UI.
2. **Wire Detox** — if not present, add `mobile/detox.config.js` + `e2e` mobile project; until then mobile E2E is Jest + EAS smoke tagged `mobile-smoke`.
3. **Fill frontend coverage holes** — `ChatLanguageModal FR-35` + `HighlightableText FR-28` + `Words 280/1000` already present; add `PrivacySettings` visibility enforcement unit if below 80%.
4. **Run full gate on dev** — `bash deploy/ci/gate-on-dev.sh` with `SOAK_DURATION=60` smoke before 24h soak; collect 10.3 benchmarks (`docs/PERFORMANCE_BENCHMARKS.md`) p95 per endpoint + WS latency + DB query <50ms.
5. **Close Phase 10.4** — all 13 GO votes in `GO_NO_GO.md` + `verify-release-gate.sh` (online) green → promote.

## 10. Open Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Teacher marketplace UI GAP (12/13 wireframes) blocks axis 10 | NO-GO on release | Contract tests + skipped e2e keep CI green but human axis 10 stays NO-GO until screens land; track in WIREFRAME_TRACE Rec. 1-2 |
| Group/community backend not spec'd | scope creep | deferred per Backlog §6; no tests required |
| Phoenix sample <10 in offline CI | false green | `verify-release-gate.sh --offline` uses `SKIP_*` but online gate requires ≥10; seed evals on dev |
| WS registry TTL drift | cross-server delivery fail | `routing_test.go` + `horizontal_test.go` + soak `ws_fast_dropped_total==0` assert |

## 11. Appendix

- **Canonical run commands:** `RUN_GUIDE.md`, `GET_STARTED.md`.
- **Security detail:** `docs/SECURITY_TEST_PLAN.md` (block/2FA/OTP/privacy vectors).
- **Call QA:** `docs/CALL_QA.md`, `docs/VIDEO_CALL_QA.md`, `docs/MESSAGING_PARITY_QA.md`.
- **Teacher vetting:** `docs/TEACHER_VETTING.md` + `deploy/ci/verify-teacher-vetting.sh`.
- **Soak invariant:** `docs/SOAK_TEST.md` — Postgres `message_receipts` inbox, Redis never truth.
- **Historical reports:** `COMPREHENSIVE_TEST_REPORT.md` (150+ cases), `TEST_REPORT.md` (Phase 1 smoke).

## 12. Addendum 2026-09-03 — Missing Test Cases (QA Critique, C-01..C-05 + AVD)

> **Authority:** `docs/QA_CRITIQUE_AND_IMPROVEMENTS.md:1` (zero-tolerance QA, crew/roles.py:54), `docs/WIREFRAME_TRACE.md:1` (94 entries), `docs/CREWAI_GAP_CLOSURE_PLAN.md:39` TDD loop, `packages/shared/src/devAccounts.ts:1` DEV_ACCOUNTS, `backend/internal/services/dev_seed.go:31` SeedDevData, `e2e/fixtures/users.ts:1` → DEV_ACCOUNTS, `e2e/fixtures/test-helpers.ts:88` waitForTranslation critical, `e2e/playwright.config.ts:1` workers:1 + AVD 10.0.2.2
> **Status:** DESIGN — failing-first skeletons `e2e/tests/20-*.spec.ts` .. `24-*.spec.ts` must be RED until impl; BA cannot sign until green on web + AVD
> **Improvements ordered:** switch fixtures to DEV_ACCOUNTS + global-setup dev_seed + JWT clear, fix waitForTranslation non-swallow for critical, add message_receipts durability assert, add WebDriverIO/Detox for AVD 10.0.2.2, add afterSeed JWT clear (see QA doc §3)

### 12.1 Critique Summary (what is fragile)

| Fragility | File:line | Impact | Fix in QA doc | Gate |
|---|---|---|---|---|
| Legacy Gmail users `uhsarp@gmail.com` / `avcxafefwer@gmail.com` with shared cleartext password | `e2e/fixtures/users.ts:16` | Flake on rotation/captcha, not deterministic, collides parallel | §3.1 switch to `DEV_ACCOUNTS` (`devAccounts.ts:15` alice.dev/bob.dev/sofia.tutor `ChorusDev123!`) | `grep -R "uhsarp" e2e/` 0 |
| File-content marketplace tests (no browser) | `e2e/tests/tutor-browse.spec.ts:28` etc. | Always green even if UI hidden | §3.2 keep @smoke + add C-04 browser journey | `23-marketplace-e2e.spec.ts` green |
| Translation swallowed catch (always green) | `e2e/fixtures/test-helpers.ts:88` + `03-messaging-translation.spec.ts:135` | Hides translator-engine regression; NFR accuracy/p95 not enforced | §3.3 `waitForTranslation(..., {critical:true})` for C-01 | `C-01-02` critical fails if translation down |
| Single-context journeys (1 msg per test) | `03-messaging-translation.spec.ts:21` etc. | No proof of 5-msg continuity + vocab/grammar/ai-tutor + settings | §2.1 C-01 serial two-context | `20-comprehensive-two-user.spec.ts` |
| No AVD (web-only Chromium) | `e2e/playwright.config.ts:33` single Chromium | NFR-22 mobile-first parity unproven | §3.5 WebDriverIO/Detox + `10.0.2.2` | `E2E_AVD=true` project green |
| No comprehensive C-01 | — | Chat→learn loop never chained | §2.1 C-01 | `C-01-01..06` green |
| No learning journey (placement/scenarios/real-talk/streak/lesson/monthly) | `qa-scenarios-drills.spec.ts:194` mocked `page.setContent` | Learn hub mocked, no real `POST /learning/*` | §2.2 C-02 | `21-learning-journey.spec.ts` green |
| No settings privacy enforcement | `08-settings.spec.ts:1` displayName only | Block/report/2FA/GDPR not enforced | §2.3 C-03 | `22-settings-privacy.spec.ts` green |
| No teacher apply UI | `frontend/src/pages/BecomeTeacher.tsx:1` no e2e | `POST /teachers/apply:646` only via backend test | §2.5 C-05 | `24-teacher-apply.spec.ts` green |
| No durability (`message_receipts` as inbox) | `docs/SOAK_TEST.md` invariant | Reload loses history if Redis-only | §3.4 `GET /chats/:id/messages:564` after reload `count 5` + `message_receipts` probe | `C-01-06` durability green |

### 12.2 Missing-Test-Cases Table (Gherkin + testRefs)

| ID | Feature / Wireframe | Gherkin selector | testRef (skeleton file:line, failing-first) | Backend route | Frontend/Mobile route |
|---|---|---|---|---|---|
| **C-01** | Comprehensive two-user journey — alice (en→es) → bob (es→en) 5 msgs + vocab/grammar/ai-tutor + settings → bob verify inbox+translation+SRS + durability | `@C-01 @comprehensive @critical @two-user` — `C-01-01` alice sends 5 msgs, `C-01-02` bob sees `🌐 In your language:` critical, `C-01-03` vocab mine → `📚 Vocabulary`, `C-01-04` grammar `📝` → `🤖` ai-tutor, `C-01-05` settings change Display Name → reload sidebar shows `Alice C01`, `C-01-06` `message_receipts` durable after reload + `ws_fast_dropped_total==0` | `e2e/tests/20-comprehensive-two-user.spec.ts:1` — `describe @C-01` serial 2 contexts (`alicePage`/`bobPage`), `loginAsUser(ALICE)` (`test-helpers.ts:15`), `createDirectChat:36` + 5× `sendMessage:67` + `.break-words hasText`, `findChatInSidebar:175` + `waitForTranslation(...,{critical:true})` (`test-helpers.ts:88`), `openGrammarAnalysis:112` + `openAITutor:139` + `openProfileMenu:152` → `Settings` → `save settings` + `GET /chats/:id/messages:564` count 5 after reload | `GET /chats:543` `GET /chats/:id/messages:564` `GET /learning/vocabulary/mined:691` `POST /chats/:id/attachments:582` (vocab save) `PUT /users/me/settings:474` `GET /users/me/settings:473` `message_receipts` durable `receipt_test.go` | `Chat.tsx:1` `App.tsx:125` `/chat` + `ChatScreen.tsx:1` `MainTabs.tsx:64` + `GrammarPanel.tsx:1` `DeepDiveSheet.tsx:1` + `HighlightableText.tsx:1` + `VocabularyReview.tsx:1` `App.tsx:148` `/learn/vocabulary` + `Settings.tsx` `PrivacySettings.tsx:1` |
| **C-02** | Learning journey — placement / scenarios / real-talk / streak / lesson / monthly | `@C-02 @learning` — `C-02-01` `POST /learning/placement/start:663` → answer `:664` → `GET :667` results; `C-02-02` `GET /learning/dashboard:659` weeklyActivity/fluency; `C-02-03` `GET /learning/scenarios:696` `ordering-coffee` → `POST :scenarioId/start:698` openingLine + ChunkBank; `C-02-04` `GET /learning/real-talk/prompts:705` → `POST :promptId/used:706`; `C-02-05` `POST /learning/streak/recover:708`; `C-02-06` `POST /learning/sessions/start:681` cloze `Yo ____ cansado.` → `POST :itemId/answer:683` → `POST :sessionId/complete:684` recap; `C-02-07` `GET /learning/srs/queue:687` interleaved + `GET /learning/vocabulary/mined:691` + monthlyActivity | `e2e/tests/21-learning-journey.spec.ts:1` — `@C-02` serial, `goto('/learn/placement')` `Placement.tsx:1` `App.tsx:134`, `goto('/learn/scenarios')` `Scenarios.tsx:1` `:143`, `goto('/learn/real-talk')` `RealTalkHub.tsx:1` `:170`, `goto('/learn/session')` `LessonSession.tsx:1` `:138`, `goto('/learn/vocabulary')` `VocabularyReview.tsx:1` `:148`, `goto('/learn/roadmap')` `LearningRoadmap.tsx:1` `:168` | `POST /learning/placement/start:663` `POST /learning/placement/:attemptId/answer:664` `GET /learning/placement/:attemptId:667` `GET /learning/dashboard:659` `GET /learning/scenarios:696` `POST /learning/scenarios/:scenarioId/start:698` `GET /learning/real-talk/prompts:705` `POST .../used:706` `POST /learning/streak/recover:708` `POST /learning/sessions/start:681` `POST .../items/:itemId/answer:683` `POST .../complete:684` `GET /learning/srs/queue:687` `GET /learning/vocabulary/mined:691` `GET /learning/path:660` | `/learn` `Learn.tsx:1` `LearnScreen.tsx:1` + every learn hub page above; `mobile: PlacementScreen` `ScenariosScreen` `ScenarioRoleplayScreen` `RealTalkHubScreen` `LessonSessionScreen` `VocabularyReviewScreen` `LearningRoadmapScreen` `StreakRecoveryScreen` `MainTabs.tsx:89-124` |
| **C-03** | Settings privacy + 2FA — enforcement | `@C-03 @settings @privacy @2FA` — `C-03-01` Display Name + Native Language + target toggle persist `PUT /users/me/settings:474` → reload; `C-03-02` `POST /blocks:485` hides DM → `GET /blocks:487` → `DELETE :486`; `C-03-03` `ReportModal` `POST /reports:489` → `GET /admin/reports:535`; `C-03-04` `ChatLanguageModal FR-35` only own language; `C-03-05` 2FA setup/replay/leak guard; `C-03-06` `GET /users/me/export` zip + `GET /privacy/retention-policy:478` + erase `DELETE /users/me`; `C-03-07` location/gallery `GET /chats/:id/gallery:581` | `e2e/tests/22-settings-privacy.spec.ts:1` — `@C-03` serial, `openProfileMenu:152` → `Settings` → `PUT` intercept 200 → reload, `POST /blocks` → `expect(cursor-pointer hasText Bob).toHaveCount(0)`, `POST /reports` 201, `ChatLanguageModal.test.tsx` FR-35, `POST /auth/2fa/*` replay 401, `GET /users/me/export` zip | `GET /users/me:465` `PUT /users/me:467` `GET /users/me/settings:473` `PUT /users/me/settings:474` `POST /blocks:485` `DELETE /blocks/:userId:486` `GET /blocks:487` `POST /reports:489` `GET /admin/reports:535` `POST /auth/2fa/setup` `POST /auth/2fa/verify` `GET /users/me/export` `DELETE /users/me` `GET /privacy/retention-policy:478` `GET /chats/:id/gallery:581` | `Profile.tsx:1` `App.tsx:174` `/profile` + `Settings.tsx` `PrivacySettings.tsx:1` + `ReportModal.tsx:1` + `ChatLanguageModal.tsx:1` (FR-35) + `mobile/ProfileScreen.tsx:1` `MainTabs.tsx:134` |
| **C-04** | Marketplace full UI — browse→profile→book trial ($0, -1) → TrialCredits → Dashboard → Payouts | `@C-04 @marketplace @S-T-01..06` — `C-04-01` `/tutors:247` search `sofia` `GET /teachers/browse:648` `Featured Tutors` `Available Now` filters; `C-04-02` click → `/tutors/:id:248` `GET :id:664`/`reviews:665`/`availability:667` `Sofia Tutor` `Verified` `Hola` `Reviews` `Pricing Options` `Booking calendar` `book-trial`; `C-04-03` `Book Trial` → `/tutors/:id/confirm:249` `Great choice` `Payment Summary` `Credits Applied -1` `Total $0.00` → `POST /teachers/:id/book:668 isTrial:true` 201 → `/trial-credits`; `C-04-04` `/trial-credits:250` `GET /teachers/trial-credits/dashboard:651` credits 0 `How Trials Work` `History`; `C-04-05` login sofia → `/teacher/dashboard:251` `GET /teachers/dashboard:649` `Welcome back` `Earnings Overview` `Availability` `Recent Students` `Profile Completion`; `C-04-06` `/teacher/payouts:252` `GET /teachers/payouts/overview:638` `Total Lifetime Earnings` `Payout Methods` `This Month's Breakdown` `Withdraw` `Performance Insight` `Payout History` + add paypal `@` guard + withdraw guard | `e2e/tests/23-marketplace-e2e.spec.ts:1` — `@C-04` serial, `loginAs(ALICE)` → `goto('/tutors')` → `tutor-search` fill `sofia` → `GET /teachers/browse?search=sofia` 200 → `Sofia Tutor` `$25`, `click` → `toHaveURL /tutors/.*` → `book-trial` → `confirm-booking` → `waitForResponse **/book` 201 → `toHaveURL /trial-credits`, login `SOFIA` → `goto('/teacher/dashboard')` + `goto('/teacher/payouts')` → add `paypal sofia@chorus.test` 201 + invalid 400 | `GET /teachers/browse:648` `GET /teachers/:id:664` `GET /teachers/:id/reviews:665` `GET /teachers/:id/availability:667` `POST /teachers/:id/book:668` `GET /teachers/trial-credits/dashboard:651` `GET /teachers/trial-credits:650` `GET /teachers/dashboard:649` `GET /teachers/payouts/overview:638` `GET /teachers/payouts/methods:641` `POST :642` `DELETE :643` `PUT :644` `POST withdraw:640` `GET history:639` `teacher.go:132/105/416/232/635` `payout.go:107` | `/tutors:247` `BrowseTutors.tsx:1` → `/tutors/:id:248` `TutorProfile.tsx:1` → `/tutors/:id/confirm:249` `ConfirmBooking.tsx:1` → `/trial-credits:250` `TrialCredits.tsx:1` → `/teacher/dashboard:251` `TeacherDashboard.tsx:1` → `/teacher/payouts:252` `Payouts.tsx:1` + `MarketplaceTab:180-202` `label Tut` `mobile/screens/BrowseTutorsScreen`/`TutorProfileScreen`/`ConfirmBookingScreen`/`TrialCreditsScreen`/`TeacherDashboardScreen`/`PayoutsScreen` |
| **C-05** | Teacher apply UI | `@C-05 @become-teacher @12.1` — `C-05-01` `/become-teacher:242` renders `Become a Teacher` Bio 10-1000 Languages chips Expertise Rate VideoUrl Certificates + Add; `C-05-02` validation bio/languages/rate/cert hint; `C-05-03` fill valid → `POST /teachers/apply:646` 201 `Status: pending` → reload prefills via `GET /teachers/me:647`; `C-05-04` pending not in browse until approved | `e2e/tests/24-teacher-apply.spec.ts:1` — `@C-05` serial, `loginAs(BOB)` → `goto('/become-teacher')` → `heading Become a Teacher` + `Bio` + `Languages you teach` + `Expertise` + `Hourly rate` + `Intro video URL` + `Certificates`; submit empty → error; fill valid → `waitForResponse **/teachers/apply` 201 → `Status: pending` → reload → `textarea HasValue Hola!` + `GET /teachers/browse?search=bob` 0 while pending | `POST /teachers/apply:646` `GET /teachers/me:647` `GET /teachers/browse:648` `teacher.go:54 Apply` `teacher_vetting_test.go` | `/become-teacher:242` `BecomeTeacher.tsx:1` `BecomeTeacherScreen.tsx:1` `MainTabs.tsx:193,196` `App.tsx:242` |
| **AVD** | Parity harness | `@AVD @parity @NFR-22` — AVD `emulator-5554` boot `adb getprop sys.boot_completed 1` + `curl 10.0.2.2:8080/health` `commit==HEAD` (`health.go:42`); `E2E_AVD=true` projects `mobile-chrome` `Pixel 7` `baseURL 10.0.2.2:3000` for web + `WebDriverIO UiAutomator2` / `Detox android.emu.debug` for native `MainTabs.tsx:64` ChatList/Learn/Marketplace/Browse→Profile→Book→TrialCredits→Dashboard→Payouts + `BecomeTeacher` | `e2e/webdriverio.conf.ts:1` + `mobile/detox.config.js:1` + `e2e/playwright.config.ts:33` `mobile-chrome` project; tagged `@avd` on C-01..C-05 | same backend via `10.0.2.2` | `10.0.2.2` web + native app via Detox |

> Full Gherkin text lives in `docs/QA_CRITIQUE_AND_IMPROVEMENTS.md:§2`. Each skeleton `e2e/tests/20-*.spec.ts` contains the same Gherkin as a comment header and `test.skip` / failing `expect(page.getBy...)` locators that will be RED until the fix (§12.3) is applied. BA must not move `phase_status` to DONE until `npx playwright test --project=chromium --project=mobile-chrome` (when AVD) is green plus `message_receipts` durability proven.

### 12.3 Fixes Required to Make C-01..C-05 + AVD Runnable (P0)

| Fix | File:line | Change | Verify |
|---|---|---|---|
| Switch fixtures to DEV_ACCOUNTS | `e2e/fixtures/users.ts:1` | `import { DEV_ACCOUNTS } from '@chorus/shared/src/devAccounts'` (`devAccounts.ts:15`); export `ALICE/BOB/SOFIA` from it; delete Gmail `uhsarp@gmail.com` | `grep -R "uhsarp" e2e/` 0; `e2e --list` shows C-01 using alice.dev |
| Global-setup dev_seed + JWT clear | `e2e/global-setup.ts:103` `globalSetup()` | After `waitForUrl(BACKEND_HEALTH)` call `fetch ${API_BASE}/dev/seed POST` or `go run ./cmd/server --seed-dev` (dev_seed.go:31) then clear `localStorage`/`sessionStorage`/`storageState.json` (afterSeed JWT clear) | `GET /teachers/browse?search=sofia` 1 after setup; `localStorage accessToken` null at test start |
| waitForTranslation critical flag | `e2e/fixtures/test-helpers.ts:88` | `waitForTranslation(page, msg, timeout, {critical})` — critical throws, soft warns; C-01 uses `critical:true`, exploratory stays `false` | `grep waitForTranslation.*critical` in `20-*.spec.ts` |
| message_receipts durability | `e2e/tests/20-comprehensive-two-user.spec.ts:135` + `backend/internal/services/receipt_test.go` | `page.reload()` → `expect(.break-words hasText)` + `GET /api/v1/chats/:id/messages:564` count | `/metrics` `ws_fast_dropped_total==0` |
| WebDriverIO/Detox for AVD | `e2e/webdriverio.conf.ts:1` + `mobile/detox.config.js:1` + `e2e/playwright.config.ts:33` new project `mobile-chrome` (`devices['Pixel 7']`, `baseURL http://10.0.2.2:3000` when `E2E_AVD=true`) | NFR-22 parity | `E2E_AVD=true npx playwright test --project=mobile-chrome --list` |
| afterSeed JWT clear | `e2e/global-setup.ts:103` | delete `storageState.json` + `page.evaluate(localStorage.clear())` helper | `localStorage accessToken` null |

> **Sign-off for 10.1 DESIGN:** this document + `docs/WIREFRAME_TRACE.md` + `docs/RELEASE_GATE.md` + `docs/GO_NO_GO.md` constitute the test design. Execution evidence in 10.2 must attach to the same release tag. Addendum §12 + skeletons `20-24` + `docs/QA_CRITIQUE_AND_IMPROVEMENTS.md` are part of 10.1.


