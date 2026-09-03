# QA Critique & Improvements — Chorus E2E (Zero-Tolerance, crew/roles.py:54)

> **Authority:** `docs/TEST_PLAN.md:1` (Phase 10.1 DESIGN v1.0), `docs/WIREFRAME_TRACE.md:1` (94 entries), `docs/GAP_SIGNOFF.md:1` (S-HOME + S-T PASS), `docs/CREWAI_GAP_CLOSURE_PLAN.md:1` (TDD loop), `docs/REQUIREMENTS_SLICE_HOME_V2.md:1`, `docs/REQUIREMENTS_SLICE_MARKETPLACE.md:1`, `crew/roles.py:54` qa_engineer (device-level DoD), `e2e/playwright.config.ts:1`, `e2e/fixtures/users.ts:1`, `e2e/fixtures/test-helpers.ts:1`, `packages/shared/src/devAccounts.ts:1`, `backend/internal/services/dev_seed.go:31`
> **Date:** 2026-09-03 | **Author:** qa_engineer (zero-tolerance) | **Status:** DESIGN — failing-first skeletons in `e2e/tests/20-*.spec.ts` must be RED until impl green, BA cannot sign C-01..C-05 until device + automation green

## 0. Executive Verdict

**Current TEST_PLAN + e2e is necessary but not sufficient for release.** It proves Home v2 + marketplace S-T-01..06 via file-content and single-context unit journeys, but it does **not prove** the product works for two real humans on browser + AVD talking to each other, learning, changing privacy, handling money, and surviving a restart. **NO-GO if shipped as-is** — gaps below are P0.

**What is green:** Home v2 S-HOME-01..04 (`00-home.spec.ts:17`), 6 marketplace screens (`tutor-*.spec.ts` + `payouts.spec.ts` + `trial-credits.spec.ts` + `teacher-dashboard.spec.ts`), chat/messaging/grammar/ai-tutor/vocab/settings/realtime/call suites (01-15), load/soak, security, observability.

**What is hollow:** 9 critical fragilities (§1) + 5 missing journey families (C-01..C-05, §2) + AVD parity harness (§2.6). This doc designs them as Gherkin + testRefs and lists the 6 code fixes that make them runnable.

---

## 1. Critique — What Is Fragile Today

### 1.1 Legacy Gmail users — `e2e/fixtures/users.ts:16` `uhsarp@gmail.com` / `avcxafefwer@gmail.com`

- **Finding:** Tests depend on two externally-owned Gmail accounts with a shared cleartext password `Demor@cer1` (`users.ts:18,24`). Any password rotation, 2FA, captcha, or Gmail deletion makes the entire 01/03/04/05/06 suite flake. Seed is not deterministic — `dev_seed.go:31` provisions `alice.dev@chorus.test` / `bob.dev@chorus.test` / `sofia.tutor@chorus.test` (`DevPassword ChorusDev123!` `dev_seed.go:17`) but no test uses them. Parallel runs collide on the same Gmail inboxes/chats.
- **Severity:** P0 — violates `TEST_PLAN.md:180` isolation ("seed two users per run") and `packages/shared/src/devAccounts.ts:1` canonical.
- **Evidence:** `users.ts:16-28` hard-codes Gmail; `devAccounts.ts:15-40` lists the correct accounts; `e2e/tests/01-auth.spec.ts:15` logs in as Gmail; no `global-setup.ts:1` calls `SeedDevData`.
- **Fix:** Switch `e2e/fixtures/users.ts:1` to re-export `DEV_ACCOUNTS` (`devAccounts.ts:15`), alias `ALICE = DEV_ACCOUNTS[0]`, `BOB = DEV_ACCOUNTS[1]`, `SOFIA = DEV_ACCOUNTS[2]`; delete Gmail literals; add `global-setup.ts:103` `await seedDevViaAPI()` (POST `/auth/register` or `go run ./cmd/server --seed-dev` equivalent via `fetch ${API_BASE}/dev/seed` when `E2E_SEED=true`). After seed, **clear JWTs** from storage so auth state starts clean (see §3.5).

### 1.2 File-content marketplace tests — not browser E2E

- **Finding:** `tutor-browse.spec.ts:28` / `tutor-profile.spec.ts:23` / `tutor-booking.spec.ts:23` / `trial-credits.spec.ts:23` / `teacher-dashboard.spec.ts:22` / `payouts.spec.ts:22` assert `fs.readFileSync(...BrowseTutors.tsx).toContain('Featured Tutors')` — these prove the file contains a string, not that the UI renders, navigates, or talks to `GET /teachers/browse:648`. A typo in CSS that hides the section would still pass.
- **Severity:** P0 — `TEST_PLAN.md:92` claims marketplace E2E but `qa-scenarios-drills.spec.ts:194` is the only browser flow and it is mocked (`page.setContent` + `route.fulfill`).
- **Evidence:** `tutor-browse.spec.ts:27-35` file-content; `tutor-profile.spec.ts:22-31` file-content; only `tutor-browse.spec.ts:47` does real `page.goto('/tutors')` but still no alice login nor seed lookup nor price assertion.
- **Fix:** Keep one file-content parity test as smoke (fast, no backend) but add `C-04` browser journey (`20-24` below) that does `loginAs(ALICE)` → `GET /teachers/browse?search=sofia 200` → `click Sofia card` → `book-trial` → `POST /teachers/:id/book isTrial:true 201` → `GET /teachers/trial-credits/dashboard credits 0` → `GET /teachers/dashboard` → `GET /teachers/payouts/overview`. Mark file-content tests `@smoke` and browser journey `@critical`.

### 1.3 Translation swallowed catch — `e2e/fixtures/test-helpers.ts:88` + `03-messaging-translation.spec.ts:135`

- **Finding:** `waitForTranslation` (`test-helpers.ts:89-102`) correctly waits for `🌐 In your language:` but every caller swallows its failure:
  ```ts
  try { await waitForTranslation(page, msg, 60_000) } catch { console.warn('⚠️ …'); /* don't fail */ }
  ```
  (`03-messaging-translation.spec.ts:123-140`, `3.4:189-196`, `3.5:241-252`). That makes `3.3` and `3.5` **always green** whether translation works or not. NFR translation accuracy ≥80 / p95 <500 ms (`TEST_PLAN.md:46`) is never enforced. Same for `04-grammar.spec.ts:91-94` and `05-ai-tutor.spec.ts` catch branches.
- **Severity:** P0 — hides regression in `translator-engine` / queue. Soak alert `ws_fast_dropped_total==0` is not enough.
- **Fix:** Add a `critical` flag: `waitForTranslation(page, msg, 60_000, {critical:true})` throws; `critical:false` keeps soft. C-01 must be `critical:true` (bob must see translation for alice's messages). Other exploratory tests can stay soft but must `test.skip` when `TRANSLATION_DISABLED=1` rather than swallow. Also add a cache-hit p95 assertion via `translation_jobs.latency_ms` histogram scrape in `global-setup` (see §3.3).

### 1.4 Single-context journeys — no comprehensive cross-role flow

- **Finding:** Each spec is one page / one context: `03-messaging-translation.spec.ts:21` opens a new context per test, sends 1 message, closes. No test proves alice sends **5** messages while bob is online, uses vocab/grammar/ai-tutor, changes settings, and bob sees inbox + translation + SRS. `08-settings.spec.ts:45` changes displayName then immediately restores inside the same test — no cross-user verification that settings propagate (privacy `Settings.tsx` blocks not enforced).
- **Severity:** P0 — `CREWAI_GAP_CLOSURE_PLAN.md:11` ("never booted AVD nor curl /health") is fixed for home/marketplace but not for messaging-learning-settings continuity.
- **Fix:** Add `C-01` comprehensive two-user journey (`20-comprehensive-two-user.spec.ts:1`) — single `serial` describe, two persistent contexts (`alicePage` + `bobPage`), 5 messages en→es, vocab mine → SRS, grammar → ai-tutor, settings change, reload, bob verifies. Must be `workers:1` already (`playwright.config.ts:14`) — keep it.

### 1.5 No AVD — web-only, `playwright.config.ts:33` single Chromium

- **Finding:** `TEST_PLAN.md:20` lists "mobile: Jest + Detox (where present; else Jest smoke + manual)" — Detox is not wired, no `mobile/detox.config.js`, no AVD project in `playwright.config.ts:33`. `e2e/` never touches `10.0.2.2` (emulator host). `GAP_SIGNOFF.md:35` says "AVD MarketplaceTab flow" but evidence is checkbox screenshots, not automation.
- **Severity:** P0 for NFR-22 mobile-first parity (`crew/roles.py:97`).
- **Fix:** Add AVD parity harness (§2.6): `e2e/webdriverio.conf.ts` (or `detox.config.js`) targeting `http://10.0.2.2:8080` + `http://10.0.2.2:3000` (or Expo `exp://10.0.2.2:19000`), plus a Playwright project `mobile-chrome` with `devices['Pixel 7']` as interim. Add `E2E_AVD=true` flag to `global-setup.ts`.

### 1.6 No comprehensive C-01 — the "5 messages + vocab/grammar/ai-tutor + settings" journey

- **Gap:** No test chains the core value prop: "chat to learn." `03` tests translation, `04` tests grammar, `05` tests ai-tutor, `06` tests vocab — never together on the same conversation.
- **Fix:** See `C-01` Gherkin below.

### 1.7 No learning journey — placement/scenarios/real-talk/streak/lesson/monthly

- **Gap:** Learning is mocked only in `qa-scenarios-drills.spec.ts:194` (`page.setContent` + `route.fulfill`). No real `POST /learning/placement/start:663` → answer → `GET /learning/placement/:attemptId` → `GET /learning/dashboard:659` → `POST /learning/sessions/start:681` → answer → `POST /learning/sessions/:sessionId/complete:684` → `GET /learning/srs/queue:687` → `GET /learning/real-talk/prompts:705` → `POST /learning/streak/recover:708` flow.
- **Fix:** `C-02` below.

### 1.8 No settings privacy enforcement — `08-settings.spec.ts:1` is displayName only

- **Gap:** `Settings.tsx` / `PrivacySettings.tsx:1` has block/report, data retention, 2FA, GDPR export/erasure, avatar, language — none asserted. `08-settings.spec.ts:24-127` only checks displayName + language dropdown + target toggle. No test proves `POST /blocks:485` hides chat, `POST /reports:489` queues, `GET /users/me/settings:473` persists, `POST /auth/2fa/*` enforces, `DELETE /users/me` erases after 30d, `GET /users/me/export` returns zip.
- **Fix:** `C-03` below.

### 1.9 No teacher apply UI journey

- **Gap:** `BecomeTeacher.tsx:1` has a full form (bio, languages, expertise, rate, videoUrl, certs, submit via `client.teacher.apply:50`) but no e2e exercises it. `TEST_PLAN.md:114` says `teacher_vetting_test.go` covers it — that's backend only. No test proves alice can apply, see `Status: pending`, then (after admin approve seed or API) appear in `GET /teachers/browse`.
- **Fix:** `C-05` below — must assert client-side validation (bio 10-1000, rate, cert required hint) and 201 → `Status: pending` → re-fetch `getMyApplication`.

### 1.10 No durability — `message_receipts` never asserted

- **Gap:** `TEST_PLAN.md:97` says `Postgres message_receipts as inbox, Redis never source of truth` and `SOAK_TEST.md` invariant, but no e2e asserts it. If WS delivers but DB does not persist, `page.reload()` would lose history — no test reloads after send and checks `GET /chats/:id/messages:564` returns the 5 messages.
- **Fix:** In `C-01` add `await page.reload(); await expect(...hasText(msg)).toBeVisible()` + `GET /api/v1/chats/:id/messages` `count 5` + optional `SELECT count(*) FROM message_receipts` via `API_BASE + /admin/...` or direct DB probe (see §3.4).

---

## 2. Missing Test Cases — Gherkin + testRefs

All Gherkin below assume **DEV_ACCOUNTS** (`packages/shared/src/devAccounts.ts:15`):
`alice.dev@chorus.test` / `bob.dev@chorus.test` / `sofia.tutor@chorus.test` password `ChorusDev123!` (`dev_seed.go:17`), seeded via `global-setup.ts:103` `SeedDevData` + JWT clear.

Common Background for headed browser: `workers:1` (`playwright.config.ts:14`), `timeout 300_000` (`playwright.config.ts:19`), `baseURL http://localhost:3000 || E2E_BASE_URL`.

---

### 2.1 C-01 — Comprehensive Two-User Journey (alice → bob, 5 msgs + vocab/grammar/ai-tutor + settings → bob verify inbox+translation+SRS)

**Goal:** Prove the flagship loop: **chat to learn** is durable and translated both ways.

```gherkin
@C-01 @comprehensive @critical @two-user @wireframe-chats @FR-4..FR-13 @FR-14..FR-17
Feature: C-01 Comprehensive two-user journey — alice (en→es) to bob (es→en), 5 messages + learning + settings

  Background:
    Given dev seed ran — alice.dev en->{es}, bob.dev es->{en}, sofia.tutor approved (dev_seed.go:31 seedTutorMarketplace)
    And global-setup cleared localStorage JWTs (storage.removeItem accessToken/refreshToken)
    And I have two persistent browser contexts alicePage and bobPage (workers:1, serial)

  Scenario: C-01-01 alice creates DM to bob and sends 5 messages (en)
    When alicePage logs in as alice.dev@chorus.test via UI (test-helpers.ts:15 loginAsUser) and creates DM to "Bob Dev" (createDirectChat:36)
    And alicePage sends 5 messages via textarea[placeholder="Type a message..."] (sendMessage:67):
      | # | text |
      | 1 | Hello Bob, how are you doing today? |
      | 2 | I have been learning Spanish for three years. |
      | 3 | The weather is beautiful today, shall we practice? |
      | 4 | She was walking through the park when it started raining. |
      | 5 | Vocabulary test: The elephant walked carefully through the jungle. |
    Then each message appears as .break-words hasText(msg).last() (sendMessage:75)
    And chat is reachable via GET /chats/:id/messages?limit=20 returns 5 (main.go:564)

  Scenario: C-01-02 bob receives inbox in real-time + sees translations (critical)
    When bobPage logs in as bob.dev@chorus.test and finds DM with "Alice Dev" in sidebar (findChatInSidebar:175)
    And bobPage clicks the chat
    Then bobPage sees all 5 alice messages as .break-words (hasText) within 15s
    And for each msg, bobPage sees "🌐 In your language:" via waitForTranslation(msg, 60_000, {critical:true}) (test-helpers.ts:88)
    And translation .italic.font-medium has length >3
    And "🌐 Translating..." appears at least once (or is skipped if cached — do not swallow critical)

  Scenario: C-01-03 bob mines vocab + SRS (alice message 5) + verifies vocab hub
    When bobPage hovers the 5th bubble (flex ancestor) and clicks vocab "+" button (06-vocabulary.spec.ts:39)
    Then button becomes "✅ Saved" (expect:48)
    And GET /learning/vocabulary/mined contains "elephant" (or via vocabularyAPI)
    When bobPage opens profile menu → Vocabulary (openProfileMenu:152, 06:59)
    Then h2 "📚 Vocabulary" visible, stats Total Words/Mastered/Due Today/Accuracy visible (06:74-77)
    And All Words tab lists saved word or shows seeded state (06:89-95)

  Scenario: C-01-04 bob opens grammar + ai-tutor on alice message 2 (critical path)
    When bobPage hovers message 2 and clicks "📝 Grammar" (openGrammarAnalysis:112 getByRole /📝\s*Gram/i)
    Then amber panel text=/📝\s*Gram/ visible within 180s (openGrammarAnalysis:130)
    When bobPage clicks "🤖" tutor button (openAITutor:139 getByRole /🤖/)
    Then div.bg-gradient-to-r.from-indigo-600 span.text-white visible within 10s (openAITutor:145)
    And assistant .bg-white.border.border-indigo-100 visible within 45s with length >5 (05:130-136)

  Scenario: C-01-05 bob changes settings (privacy) and alice sees effect after reload
    When alicePage opens profile → Settings (openProfileMenu:152, 08:17) and changes Display Name to "Alice C01" (08:45-62)
    Then text "Settings saved successfully" visible and GET /users/me (settingsAPI) reflects name
    When alicePage reloads and bobPage reloads
    Then bobPage sidebar shows "Alice C01" (or chat header) — proves persistence, not in-memory
    When bobPage verifies GET /chats/:id/messages still returns 5 messages (durability — message_receipts)
    Then all 5 still render after reload

  Scenario: C-01-06 durability — message_receipts is source of truth (Redis never is)
    When alicePage sends one more message "Durability check __TIMESTAMP__" and both pages reload
    Then GET /api/v1/chats/:id/messages contains that message (or DB probe SELECT count(*) FROM message_receipts WHERE chat_id=:id = 6)
    And /metrics ws_fast_dropped_total == 0 (if soak) or no dropped counter increment
```

**testRefs — C-01:**

| Suite | File | Locator / assertion (must be RED until impl) |
|---|---|---|
| `e2e` | `e2e/tests/20-comprehensive-two-user.spec.ts:20` `C-01-01` | `loginAsUser(alicePage, ALICE)` + `createDirectChat("Bob Dev")` + 5× `sendMessage` + `await expect(page.locator('.break-words', {hasText: msg}).last()).toBeVisible()` + `GET /api/v1/chats/:id/messages` `count 5` |
| `e2e` | `20-comprehensive-two-user.spec.ts:55` `C-01-02` | `findChatInSidebar(bobPage, "Alice C01"||"Alice Dev")` + `waitForTranslation(bobPage, msg, 60_000, {critical:true})` + `expect(bubble.locator('text=🌐 In your language:')).toBeVisible()` — fails if translation pipeline down |
| `e2e` | `20-comprehensive-two-user.spec.ts:80` `C-01-03` | `messageWrapper.hover()` + `getByRole('button', {hasText:'+'}) .click()` → `✅ Saved` + `GET /learning/vocabulary/mined` contains word |
| `e2e` | `20-comprehensive-two-user.spec.ts:95` `C-01-04` | `openGrammarAnalysis(bobPage, msg2)` → `expect(page.locator('text=/📝\\s*Gram/').first()).toBeVisible({timeout:180_000})` + `openAITutor` + `expect(.bg-white.border-indigo-100).toBeVisible({timeout:45_000})` |
| `e2e` | `20-comprehensive-two-user.spec.ts:115` `C-01-05` | `openProfileMenu` + `getByRole('button', {name:/settings/i}).click()` + `fill DisplayName` + `getByRole('button', {name:/save settings/i}).click()` → `Settings saved` + reload → sidebar shows new name |
| `e2e` | `20-comprehensive-two-user.spec.ts:135` `C-01-06` | `page.reload()` + `expect(.break-words hasText durabilityMsg).toBeVisible()` + `fetch(${API_BASE}/chats/${chatId}/messages)` length 6 + optional `message_receipts` probe |
| `backend` | `backend/internal/services/receipt_test.go` (existing) + new `soak_test.go` | `message_receipts` insert before ack durable |
| `unit` | `frontend/src/__tests__/HighlightableText.test.tsx` | vocab mine + highlight FR-28 |

---

### 2.2 C-02 — Learning Journey (placement / scenarios / real-talk / streak / lesson / monthly)

**Goal:** Prove learn hub is not mocked — real placement, scenarios, real-talk, streak recovery, lesson, SRS tie to chat mining.

```gherkin
@C-02 @learning @wireframe-placement @wireframe-scenarios @wireframe-real-talk @wireframe-streak @wireframe-lesson
Feature: C-02 Learning journey — placement → dashboard → scenarios → real-talk → streak → lesson → monthly

  Background:
    Given I am alice.dev (en->{es}, placement not started) and dev seed clean

  Scenario: C-02-01 placement start → answer vocab + reading → results
    When I open /learn/placement (Placement.tsx:1 App.tsx:134 /learn/placement) and tap Start (POST /learning/placement/start:663)
    Then I see vocab question (placement_test_vocabulary_question) and answer via POST /learning/placement/:attemptId/answer:664
    When I answer reading comprehension and GET /learning/placement/:attemptId:667
    Then results summary shows CEFR level and CTA to Learn hub (PlacementScreen.tsx:1)

  Scenario: C-02-02 dashboard shows weeklyActivity + fluency after placement
    When I GET /learning/dashboard:659
    Then dashboard displays weeklyActivity, fluency ring, dailyGoal (Learn.tsx:1 LearnScreen.tsx:1)

  Scenario: C-02-03 scenarios list + roleplay ordering coffee (Spanish)
    When I open /learn/scenarios (Scenarios.tsx:1 App.tsx:143 GET /learning/scenarios:696)
    Then I see "Pedir café en una cafetería" (curriculum.go ordering-coffee) and tap it
    When I start scenario POST /learning/scenarios/:scenarioId/start:698
    Then ScenarioRoleplay shows openingLine "Hola. ¿Qué te gustaría pedir hoy?" + SuggestedChunks + Translation (scenario.go OpeningLine/ChunkBank)
    When I send "Quisiera un café con leche, por favor." and request hint
    Then AI reply + translation arrives and phaseComplete visible (qa-scenarios-drills.spec.ts:194)

  Scenario: C-02-04 real-talk hub prompts + mark used
    When I open /learn/real-talk (RealTalkHub.tsx:1 App.tsx:170 GET /learning/real-talk/prompts:705)
    Then I see at least 1 prompt and tap it → POST /learning/real-talk/prompts/:promptId/used:706 200
    And RealTalkNudge in ChatScreen shows the same prompt

  Scenario: C-02-05 streak + recovery
    When I GET /learning/dashboard streak days 7 atRisk true
    Then /learn/roadmap shows LearningRoadmapScreen streak + StreakRecoveryScreen CTA
    When I POST /learning/streak/recover:708 (learningAPI.recoverStreak)
    Then recovered true and dashboard streak resets

  Scenario: C-02-06 lesson session (daily practice) start → answer → complete → recap
    When I POST /learning/sessions/start:681 via LessonSessionScreen (MainTabs.tsx:99 /learn/session)
    Then I see cloze "Yo ____ cansado." with choices estoy/soy (session_composer.go StartSession)
    When I answer via POST /learning/sessions/:sessionId/items/:itemId/answer:683
    Then feedback "¡Excelente!" and complete via POST /learning/sessions/:sessionId/complete:684 shows recap (study_session_recap)

  Scenario: C-02-07 SRS queue interleaves chat-mined word + monthly activity
    When I GET /learning/srs/queue:687 (VocabularyReviewScreen.tsx:1 GET /learning/srs/queue:687 + GET /learning/vocabulary/mined:691)
    Then queue contains interleaved vocab+grammar (srs_queue.go interleaveQueue) and dueToday >0 when words mined from C-01
    And dashboard monthlyActivity shows wordsLearned increment after session
```

**testRefs — C-02:**

| Suite | File | Locator |
|---|---|---|
| `e2e` | `e2e/tests/21-learning-journey.spec.ts:01` | `goto('/learn/placement')` → `expect(getByText('Placement Test')).toBeVisible()` → `POST /learning/placement/start` 201 → vocab Q |
| `e2e` | `21-learning-journey.spec.ts:25` | `GET /learning/scenarios` → `getByText('Pedir café')` → `POST /scenarios/:id/start` → `expect(getByText('Hola. ¿Qué te gustaría')).toBeVisible()` |
| `e2e` | `21-learning-journey.spec.ts:50` | `goto('/learn/real-talk')` → `getByText('Real Talk')` + `expect(route /prompts).toBeVisible()` |
| `e2e` | `21-learning-journey.spec.ts:70` | `POST /learning/streak/recover` → `expect(getByText('Streak recovered')).toBeVisible()` |
| `e2e` | `21-learning-journey.spec.ts:90` | `goto('/learn/session')` → `expect(getByText('Yo ____ cansado')).toBeVisible()` → choose `estoy` → `expect(getByText('¡Excelente!')).toBeVisible()` |
| `e2e` | `21-learning-journey.spec.ts:110` | `GET /learning/srs/queue` → `expect(queue.length >0)` + `GET /learning/dashboard` `monthlyActivity` |
| `backend` | `learning_placement_test.go` | placement start/answer/results |
| `mobile` | `mobile/src/screens/__tests__/learning.test.tsx` | Scenarios / LessonSession render |

---

### 2.3 C-03 — Settings Privacy + 2FA (enforcement, not just fields)

**Goal:** Prove privacy is enforced, not just rendered.

```gherkin
@C-03 @settings @privacy @2FA @wireframe-profile_settings @wireframe-settings @NFR-13 @FR-2 @FR-35
Feature: C-03 Settings — privacy enforcement + 2FA + avatar + language

  Background:
    Given alice.dev and bob.dev seeded, JWT clean

  Scenario: C-03-01 profile settings persist
    When alicePage logs in and opens Settings (openProfileMenu:152 → Settings) and changes Display Name + Native Language (select first option) + toggles targetLanguage es (08:94)
    Then PUT /users/me/settings:474 200 and GET /users/me/settings:473 reflects change
    When alicePage reloads
    Then Display Name and Native Language still show new values

  Scenario: C-03-02 block hides chat (privacy enforcement)
    When alicePage does POST /blocks {blockedUserId: bobId}:485
    Then GET /blocks:487 lists bob, and bob's DM disappears from alice sidebar (or GET /chats does not return it)
    When alice does DELETE /blocks/:userId:486
    Then DM reappears on reload

  Scenario: C-03-03 report queues to moderator
    When alicePage long-presses a message → Report (ReportModal.tsx:1) and POST /reports {messageId, reason}:489 201
    Then GET /admin/reports:535 (when admin) lists report (AdminWaitlist.test.tsx)

  Scenario: C-03-04 ChatLanguageModal FR-35 only own language
    When alicePage opens ChatLanguageModal (ChatHeader language selector) — FR-35
    Then modal lists only own native language (ChatLanguageModal.test.tsx), not all 80

  Scenario: C-03-05 2FA enable + replay guard + privacy leak guard
    When alicePage enables 2FA via POST /auth/2fa/setup and verifies TOTP
    Then second login requires TOTP; replay of old TOTP fails 401 (security_qa_test.go 2FA bypass/replay)
    And GET /users/me/settings does not leak tokens or secrets in response

  Scenario: C-03-06 GDPR export/erasure + retention policy link
    When alicePage does GET /users/me/export: (NFR-23) zip 200 and GET /privacy/retention-policy:478 shows 365/30/90/90
    And (if test account) DELETE /users/me erasure then login fails 401 after 30d window
    Then TrustSafetyCenter (if built) links blocks/reports/retention

  Scenario: C-03-07 location + gallery (if enabled)
    When alice shares location via POST /chats/:id/location or checks GET /chats/:id/gallery:581
    Then gallery_test.go contract: handler returns 200 or empty, and UI shows inline preview or gallery screen
```

**testRefs — C-03:**

| Suite | File | Locator |
|---|---|---|
| `e2e` | `e2e/tests/22-settings-privacy.spec.ts:01` | `openProfileMenu` → `getByRole('button', {name:/settings/i})` → `PUT /users/me/settings` intercept 200 → reload assert |
| `e2e` | `22-settings-privacy.spec.ts:30` | `POST /blocks` via `moderationAPI` or fetch → `expect(page.locator('.cursor-pointer').filter({hasText:'Bob Dev'})).toHaveCount(0)` |
| `e2e` | `22-settings-privacy.spec.ts:50` | `POST /reports` 201 → `GET /admin/reports` 200 |
| `e2e` | `22-settings-privacy.spec.ts:70` | `getByRole('button', {name:'Language'})` → modal → `expect(modal.getByText('English')).toBeVisible()` + `expect(modal.getByText('Español')).toHaveCount(0)` when own is en |
| `e2e` | `22-settings-privacy.spec.ts:90` | `POST /auth/2fa/setup` → `POST /auth/2fa/verify` → logout→login requires TOTP → replay old code `expect(401)` |
| `e2e` | `22-settings-privacy.spec.ts:110` | `GET /users/me/export` → `expect(contentType zip)` |
| `unit` | `frontend/src/pages/__tests__/Settings.test.tsx` | privacy enforcement unit |
| `backend` | `security_qa_test.go` | 2FA bypass/replay, privacy leak, suspended access |

---

### 2.4 C-04 — Marketplace Full UI (browse→profile→book→trialCredits→dashboard→payouts)

**Goal:** Replace file-content with real browser that proves money/credits move.

```gherkin
@C-04 @marketplace @S-T-01..06 @wireframe-browse_tutors @wireframe-tutor_profile_sofia @wireframe-confirm_trial_booking @wireframe-trial_credit_dashboard @wireframe-teacher_dashboard @wireframe-payout_settings_history
Feature: C-04 Marketplace full UI — Browse → Profile → Book trial ($0, -1 credit) → TrialCredits → Dashboard → Payouts

  Background:
    Given dev seed sofia.tutor approved 2500c verified + 4 slots + 2 reviews + alice credits 1 (dev_seed.go:79)
    And I am alice.dev authenticated

  Scenario: C-04-01 browse tutors + search filters
    When I open /tutors (App.tsx:247 BrowseTutors.tsx:1) and type "sofia" + Search (GET /teachers/browse:648 ?search=sofia)
    Then I see h2 Tutors + Become a teacher link + input tutor-search + Featured Tutors + Available Now + filters Language/Price/Rating (browse_tutors/code.html:131-312)
    And Sofia card shows "Sofia Tutor" es 4.5 $25/session Verified badge

  Scenario: C-04-02 tutor profile Sofia — hero + reviews + calendar + Book Trial
    When I click Sofia card → /tutors/:id (App.tsx:248 GET /teachers/:id:664 + reviews:665 + availability:667)
    Then I see Sofia hero Verified + Hola bio + Reviews + Pricing Options + Booking calendar + data-testid book-trial (tutor_profile_sofia/code.html:171-396)
    And block-tutor / report-tutor buttons exist (TutorProfile.tsx:72)

  Scenario: C-04-03 confirm trial booking — $0.00 Payment Summary
    When I tap Book Trial → /tutors/:id/confirm (App.tsx:249 ConfirmBooking.tsx:1)
    Then I see Great choice! + Your Tutor card + Date/Time + Payment Summary Trial Session 1 Credit / Credits Applied -1 / Total $0.00 + Cancellation Policy 24h + sticky confirm-booking (confirm_trial_booking/code.html:48-92)
    When I tap Confirm Booking
    Then POST /teachers/:id/book {isTrial:true, startTime tomorrow 10:00, endTime 10:30}:668 201 and navigate to /trial-credits within 1.2s

  Scenario: C-04-04 trial credit dashboard — credits 0 + nextGrantAt + history
    When I open /trial-credits (App.tsx:250 TrialCredits.tsx:1 GET /teachers/trial-credits/dashboard:651)
    Then I see Trial Credits 0 Available/Next credit + Find a Tutor → /tutors + How Trials Work 20 Minutes/Meet & Greet + Recommended for Trials (limit 2) + History 1 row

  Scenario: C-04-05 teacher dashboard as sofia — Welcome + Earnings + Availability + Students + Checklist
    When I login as sofia.tutor@chorus.test and open /teacher/dashboard (App.tsx:251 GET /teachers/dashboard:649)
    Then I see Teacher Dashboard Welcome back! + Earnings Overview 3 cols + Premium Program + Availability 3 slots + Recent Students alice/bob + Profile Completion pct bar + checklist (teacher_dashboard/code.html)

  Scenario: C-04-06 payouts — overview + methods + withdraw + history
    When I open /teacher/payouts (App.tsx:252 GET /teachers/payouts/overview:638)
    Then I see Payout Settings & History + Total Lifetime Earnings Available for payout + Withdraw Funds → + Payout Methods + This Month's Breakdown + Withdraw input + Performance Insight 12% more + Payout History (payout_settings_history/code.html + teacher_earnings_overview/code.html)
    When I add payout method paypal sofia@chorus.test (POST /teachers/payouts/methods:642) then invalid paypal "sofiacorus" → 400 paypal email invalid (payout.go:231)
    And withdraw 1000 when available 0 → 400 insufficient available balance

  Scenario: C-04-07 marketplace client NB: shared contract unchanged
    Then packages/shared/src/api.ts:1107 teacher.browse/getProfile/book/getTrialCredit/getDashboard and payouts overview/methods/withdraw still match teacher.go:132/105/416/232/635 and payout.go:107
```

**testRefs — C-04:**

| Suite | File | Locator |
|---|---|---|
| `e2e` | `e2e/tests/23-marketplace-e2e.spec.ts:01` | `loginAs(ALICE)` → `goto('/tutors')` → `fill tutor-search 'sofia'` → `expect(GET /teachers/browse?search=sofia 200)` → `getByText('Sofia Tutor')` + `$25` |
| `e2e` | `23-marketplace-e2e.spec.ts:30` | `click Sofia card` → `expect(page).toHaveURL(/\/tutors\/.*/)` → `getByText('Sofia Tutor')` + `getByTestId('book-trial')` |
| `e2e` | `23-marketplace-e2e.spec.ts:50` | `getByTestId('book-trial').click()` → `expect(getByText('Great choice'))` + `getByText('$0.00')` + `getByTestId('confirm-booking').click()` → `waitForResponse('**/teachers/*/book')` 201 → `expect(page).toHaveURL('/trial-credits')` |
| `e2e` | `23-marketplace-e2e.spec.ts:75` | `expect(getByText('Trial Credits')).toBeVisible()` + `getByText('How Trials Work')` + `getByText('History')` |
| `e2e` | `23-marketplace-e2e.spec.ts:95` | `loginAs(SOFIA)` → `goto('/teacher/dashboard')` → `getByText('Earnings Overview')` + `getByText('Profile Completion')` |
| `e2e` | `23-marketplace-e2e.spec.ts:115` | `goto('/teacher/payouts')` → `getByText('Total Lifetime Earnings')` → `addMethod paypal` → `expect(getByText('paypal · PayPal'))` + invalid `@` → `400` |
| `backend` | `teacher_test.go` + `payout_test.go` | browse verified filter, GetTutorProfile 200/404, CreateBooking isTrial, GetOverview fee 10/15 |

---

### 2.5 C-05 — Teacher Apply UI

**Goal:** Prove an unapproved learner can become a teacher via UI, not just `POST /teachers/apply:646` curl.

```gherkin
@C-05 @marketplace @become-teacher @wireframe-become_a_teacher @12.1 @S-T-apply
Feature: C-05 Teacher apply — UI form → pending → browse visibility after approval

  Background:
    Given I am bob.dev (no application) seeded, JWT clean

  Scenario: C-05-01 form renders with wireframe contract
    When I open /become-teacher (App.tsx:242 BecomeTeacher.tsx:1)
    Then I see h2 Become a Teacher + Bio 10-1000 + Languages you teach (en/es/fr/… chips) + Expertise + Hourly rate USD + Intro video URL + Certificates + Add + Submit application

  Scenario: C-05-02 client validation before submit
    When I leave bio empty and tap Submit
    Then API returns 400 "bio required" (or client hint) and no 201; error banner visible
    When I add Languages=[] and tap Submit
    Then 400 "languages required"
    When I set rate 0 or empty Certificates hint visible "Add at least one certificate for approval." (BecomeTeacher.tsx:115)

  Scenario: C-05-03 submit → Status pending + prefill on reload
    When I fill bio "Hola! I teach Spanish with 5 years…" (≥10 chars) + toggle es + expertise "Conversational Spanish" + rate 20 + Add cert {type language_certificate, issuer Instituto Cervantes, year 2020, fileUrl https://example.com/c.pdf} and tap Submit
    Then POST /teachers/apply:646 201 {status: "pending"} and UI shows "Status: pending" + "Application submitted: pending" (BecomeTeacher.tsx:51,60)
    When I reload /become-teacher
    Then form prefills via GET /teachers/me:647 (teacherAPI.getMyApplication:29) with same bio/languages

  Scenario: C-05-04 pending not yet in browse; after approval visible (or admin mock)
    When I GET /teachers/browse?search=bob.dev → total 0 while pending
    Then after approval (dev_seed style admin approve or test hook POST /admin/teachers/:id/approve) pending→approved, browse shows bob
    And mobile BecomeTeacherScreen (MainTabs.tsx:193,196) same flow via Status pending
```

**testRefs — C-05:**

| Suite | File | Locator |
|---|---|---|
| `e2e` | `e2e/tests/24-teacher-apply.spec.ts:01` | `loginAs(BOB)` → `goto('/become-teacher')` → `getByRole('heading', {name:'Become a Teacher'})` + `getByPlaceholder('Tell students about yourself')` + `getByText('Languages you teach')` |
| `e2e` | `24-teacher-apply.spec.ts:25` | `getByRole('button', {name:'Submit application'}).click()` → `expect(getByText('Application submitted: pending')).not.toBeVisible()` when empty → error |
| `e2e` | `24-teacher-apply.spec.ts:45` | `fill bio + toggle es` → `fill expertise/rate/cert` → `click Submit` → `waitForResponse('**/teachers/apply')` 201 → `expect(getByText('Status: pending')).toBeVisible()` → reload → `expect(textarea).toHaveValue(/Hola!/)` |
| `e2e` | `24-teacher-apply.spec.ts:70` | `GET /teachers/browse?search=bob` → `expect(tutors.length 0)` while pending |
| `unit` | `frontend/src/pages/__tests__/BecomeTeacher.test.tsx` (new) | mock `teacherAPI.apply` → form validation + pending display |
| `backend` | `teacher_vetting_test.go` | apply validation, state machine pending→approved→rejected |

---

### 2.6 AVD Parity Harness — `10.0.2.2` host for emulator

**Goal:** Prove NFR-22 — every journey above runs on browser (web) **and** AVD (mobile).

**Infra:**

| Artifact | Path | Contract |
|---|---|---|
| WebDriverIO config | `e2e/wdio.conf.ts:1` | host `10.0.2.2`, `capabilities: [{platformName: 'Android', deviceName: 'emulator-5554', automationName: 'UiAutomator2', app: 'mobile/android/app/build/outputs/apk/debug/app-debug.apk' }]` or Chrome via `browserName: chrome` |
| Detox config | `mobile/detox.config.js:1` | `apps: {android:{type:'android.apk', binaryPath:'android/app/build/outputs/apk/debug/app.apk'}}`, `devices: {emulator:{type:'android.emulator', device:{avDeviceName:'Pixel_7_API_34'}}}` |
| Playwright AVD project | `e2e/playwright.config.ts:33` new project `mobile-chrome` | `use: {...devices['Pixel 7'], baseURL: 'http://10.0.2.2:3000'}` when `E2E_AVD=true` |
| Emulator boot | `deploy/ci/start-android.ps1:19` + `global-setup.ts:53` `ensureAndroidEmulator()` | `emulator -avd Pixel_7_API_34 -no-snapshot` → `adb wait-for-device` → `adb shell getprop sys.boot_completed` 1 → `curl http://10.0.2.2:8080/health | jq .commit == git rev-parse HEAD` |

```gherkin
@AVD @parity @wireframe-every-folder @NFR-22 @crew/roles.py:97
Feature: AVD parity — every C-01..C-05 journey reachable on emulator 10.0.2.2

  Scenario: AVD boot + health
    Given emulator-5554 device booted via start-android.ps1
    Then adb shell getprop sys.boot_completed is 1 and curl http://10.0.2.2:8080/health 200 {status:healthy, commit:HEAD}

  Scenario: AVD web parity — C-01..C-04 on 10.0.2.2 Chrome
    When I open http://10.0.2.2:3000 on AVD Chrome (Playwright mobile-chrome project)
    Then alice→bob 5 msgs + vocab + grammar + marketplace browse→profile→book→payouts all pass (same locators but baseURL 10.0.2.2)

  Scenario: AVD native parity — Detox/WebDriverIO for mobile app
    When I launch mobile app on AVD via Detox and login as alice.dev / bob.dev (AsyncStorage adapter packages/shared/src/api.ts mobile)
    Then MainTabs.tsx:64 ChatList, LearnTab/LearnScreen, MarketplaceTab/BrowseTutors→TutorProfile→Confirm→TrialCredits→TeacherDashboard→Payouts, Profile/BecomeTeacher all navigate without crash
    And every LEARNING_DASHBOARD card navigates and loads data from backend (learning_dashboard.go)
```

**testRefs — AVD:**

| Suite | File | Locator |
|---|---|---|
| `e2e` | `e2e/tests/avd-parity.spec.ts:1` (optional) or tag `C-01 @avd` | `expect(page).toHaveURL(/10\.0\.2\.2/)` + same C-01 locators |
| `mobile` | `mobile/e2e/firstTest.e2e.js` (Detox) + `wdio.conf.ts` | `await element(by.text('Tutors')).tap()` → `await expect(element(by.text('Sofia Tutor'))).toBeVisible()` |
| `ci` | `deploy/ci/gate-on-dev.sh:avd` | `E2E_AVD=true npx playwright test --project=mobile-chrome` + `npx detox test -c android.emu.debug` |

---

## 3. Improvements — Code Changes (P0, in priority order)

| # | Fix | File:line | What to change | Why | Verify |
|---|---|---|---|---|---|
| 1 | Switch fixtures to DEV_ACCOUNTS | `e2e/fixtures/users.ts:1` | Replace Gmail literals with `import { DEV_ACCOUNTS } from '@chorus/shared/src/devAccounts'` (`devAccounts.ts:15`); export `ALICE = {...DEV_ACCOUNTS[0], displayName:'Alice Dev', nativeLanguage:'en'}`, `BOB = DEV_ACCOUNTS[1]`, `SOFIA = DEV_ACCOUNTS[2]`; delete `uhsarp@gmail.com` / `avcxafefwer@gmail.com`; keep `SAMPLE_ENGLISH_MESSAGE` for legacy if needed but not used | Deterministic, no external dependency, matches `dev_seed.go:17` `ChorusDev123!` | `grep -R "uhsarp\|avcxafefwer" e2e/` 0, `grep -R "DEV_ACCOUNTS" e2e/fixtures/users.ts` 1, `npx tsc --noEmit` 0 |
| 2 | Global-setup dev_seed + JWT clear | `e2e/global-setup.ts:103` `globalSetup()` | After `waitForUrl(BACKEND_HEALTH)` add: `await fetch(${API_BASE}/auth/login)` probe; if `E2E_SEED!=false`, call `await fetch('http://localhost:8080/api/v1/dev/seed', {method:'POST'})` or `execSync('go run ./cmd/server --seed-dev')` when backend exposes seed; then `await page.addInitScript(() => {localStorage.clear(); sessionStorage.clear();})` pattern or inject `storageState: undefined` + `clear` in new contexts (afterSeed JWT clear per mission). Log `Seeded alice/bob/sofia` | Isolation per run, cascade delete prior chats/bookings, credits reset to 1 | `e2e --list` shows 143+new, `GET /teachers/browse?search=sofia` returns 1 after setup |
| 3 | waitForTranslation non-swallow for critical | `e2e/fixtures/test-helpers.ts:88` `waitForTranslation` | Change signature to `waitForTranslation(page, msg, timeoutMs=60_000, opts?: {critical?: boolean})`; inside, if `opts?.critical` throw on timeout, else warn soft. Update `03-messaging-translation.spec.ts:123` `try` to pass `critical:false` for exploratory, and `20-comprehensive-two-user.spec.ts:55` to `critical:true` for C-01. Remove swallow in `04-grammar` non-critical paths or tag `@soft` | Prevents always-green translation regression; deep evaluation `translation_evals` accuracy ≥80 via `translation_test.go` still needs UI gate | `grep -R "waitForTranslation.*critical" e2e/tests` shows C-01 critical, others soft |
| 4 | message_receipts durability assert | `e2e/tests/20-comprehensive-two-user.spec.ts:135` `C-01-06` + `backend/internal/services/receipt_test.go` | After 5 msgs, do `await alicePage.reload(); await expect(.break-words hasText).toBeVisible()` + `const res = await fetch(${API_BASE}/chats/${chatId}/messages?limit=20, {headers:{Authorization:Bearer aliceToken}}); expect(res.length 5)` + optional `SELECT count(*) FROM message_receipts WHERE chat_id=:id` via new `GET /api/v1/debug/message_receipts?chatId=` debug endpoint (dev only) or direct DB `sql.DB` probe in `global-setup` | Proves Postgres is truth, Redis never is (SOAK_TEST.md) | `verify-drain.sh` + `/metrics` ws_fast_dropped_total 0, reload still shows 5 |
| 5 | WebDriverIO/Detox for AVD 10.0.2.2 | `e2e/webdriverio.conf.ts:1` + `mobile/detox.config.js:1` + `e2e/playwright.config.ts:33` new project | Add `wdio.conf.ts` with `services:['appium']`, `hostname:'127.0.0.1', port:4723, path:'/wd/hub'`, `capabilities: [{platformName:'Android', 'appium:deviceName':'emulator-5554', 'appium:automationName':'UiAutomator2', 'appium:app':'mobile/android/app/build/outputs/apk/debug/app-debug.apk'}]`; add `detox.config.js` per Detox 20 docs; add Playwright project `mobile-chrome` with `baseURL:'http://10.0.2.2:3000'` when `E2E_AVD=true` | NFR-22 mobile-first parity on real AVD, not just Jest | `npx detox build -c android.emu.debug` 0, `E2E_AVD=true npx playwright test --project=mobile-chrome --list` shows C-01..C-05 |
| 6 | afterSeed JWT clear + storageState reset | `e2e/global-setup.ts:103` + `e2e/fixtures/test-helpers.ts:198` `loginViaAPI` | After seed, clear `localStorage`/`sessionStorage`/`indexedDB` in all contexts by not reusing `storageState` file; if using `storageState.json`, delete it before tests; add `test.beforeEach(async ({page}) => { await page.evaluate(() => localStorage.clear()) })` helper `clearAuth` and call in `C-01 Background` | Prevents cross-test leakage where alice from prior run stays logged in and seed appears empty | `page.evaluate(()=>localStorage.getItem('accessToken'))` null at start |

---

## 4. Traceability Matrix — How New Tests Close Gaps

| Gap (WIREFRAME_TRACE / TEST_PLAN) | Slice | Gherkin | testRef | Backend route | Frontend / Mobile route | Status after |
|---|---|---|---|---|---|---|
| 62 GAP, 11 backend-only, no two-user comprehensive | C-01 | §2.1 C-01-01..06 | `20-comprehensive-two-user.spec.ts` | `GET /chats:543`, `/chats/:id/messages:564`, `/learning/vocabulary/mined:691`, `/learning/srs/queue:687`, `/users/me/settings:473` | `Chat.tsx:1` / `ChatScreen.tsx:1` + `HighlightableText` / `VocabularyReview` / `GrammarPanel` | GAP→PASS (new) |
| Learning hub 7/12 GAP (activity_hub, placement, scenarios, real-talk, streak, lesson, monthly) | C-02 | §2.2 C-02-01..07 | `21-learning-journey.spec.ts` | `POST /learning/placement/start:663` / answer `664` / `GET :667` / `GET /learning/dashboard:659` / `GET /learning/scenarios:696` / `POST start:698` / `GET prompts:705` / `POST used:706` / `POST streak/recover:708` / `POST sessions/start:681` / answer `683` / complete `684` / `GET srs/queue:687` | `/learn/placement` `App.tsx:134` / `/learn/scenarios` `:143` / `/learn/real-talk` `:170` / `/learn/session` `:138` / `/learn/vocabulary` `:148` / `/learn/roadmap` `:168` | GAP→PASS for placement/scenarios/real-talk/streak/lesson (activity_hub stays P1) |
| Settings privacy not enforced | C-03 | §2.3 C-03-01..07 | `22-settings-privacy.spec.ts` | `POST /blocks:485` / `DELETE /blocks/:id:486` / `GET /blocks:487` / `POST /reports:489` / `GET /admin/reports:535` / `POST /auth/2fa/*` / `GET /users/me/export` / `DELETE /users/me` / `GET /privacy/retention-policy:478` | `Profile.tsx:1` `App.tsx:174` `/profile` + `Settings.tsx` / `PrivacySettings.tsx:1` + `ReportModal.tsx` | PARTIAL→PASS (enforced) |
| Marketplace file-content only | C-04 | §2.4 C-04-01..07 | `23-marketplace-e2e.spec.ts` | `GET /teachers/browse:648` / `GET /teachers/:id:664` / reviews `665` / avail `667` / `POST /teachers/:id/book:668` isTrial / `GET /teachers/trial-credits/dashboard:651` / `GET /teachers/dashboard:649` / `GET /teachers/payouts/overview:638` / methods `641-644` / withdraw `640` / history `639` | `/tutors:247` / `/tutors/:id:248` / `/tutors/:id/confirm:249` / `/trial-credits:250` / `/teacher/dashboard:251` / `/teacher/payouts:252` + `MarketplaceTab:180-202` | file-content stays @smoke, browser journey PASS |
| Teacher apply no UI | C-05 | §2.5 C-05-01..04 | `24-teacher-apply.spec.ts` | `POST /teachers/apply:646` / `GET /teachers/me:647` / `GET /teachers/browse:648` | `/become-teacher:242` `BecomeTeacher.tsx:1` + `Mobile BecomeTeacherScreen:193` | GAP→PASS (apply) |
| No AVD | AVD harness | §2.6 | `avd-parity` project | `10.0.2.2:8080/health` `health.go:42` + `10.0.2.2:3000` | `10.0.2.2` mobile-chrome + Detox UiAutomator2 | NFR-22 parity proven |

---

## 5. Execution Plan — When to Run What

```
global-setup.ts: dev_seed (alice/bob/sofia) + JWT clear + waitFor /health + / @health 10.0.2.2 if E2E_AVD
  → workers:1 serial C-01 (2 contexts, 5 msgs, critical translation)
  → C-02 learning journey (placement→scenarios→real-talk→streak→lesson→SRS) — can reuse alice context, sequential
  → C-03 settings privacy+2FA — serial, uses alice/bob, cleans blocks after
  → C-04 marketplace full UI — serial, books trial (consumes alice credit), so reset seed or use fresh alice before this suite if ordered after C-01
  → C-05 teacher apply — uses bob (no app), creates pending, then cleanup
  → existing 00-15 suites — unchanged, but flip users.ts to DEV_ACCOUNTS first
global-teardown.ts: leave services unless E2E_STOP_SERVICES=true
```

**Order caveat:** C-01 and C-04 both mutate trial credits (C-01 does not consume, C-04 does). Run `C-04` before `C-01` OR re-seed between them (`POST /dev/seed` reset). The skeletons use `test.describe.serial` and document this.

**CI wiring:** `deploy/ci/gate-on-dev.sh:seed` must call `seedDevData` (or `curl -X POST :8080/api/v1/dev/seed` dev-only endpoint) before `e2e`. `verify-release-gate.sh` online gate must require `ws_fast_dropped_total==0` from `/metrics`.

---

## 6. Files Changed (this PR)

```
 M e2e/fixtures/users.ts                — switch to DEV_ACCOUNTS (devAccounts.ts:15), delete Gmail, re-export ALICE/BOB/SOFIA
 M e2e/fixtures/test-helpers.ts         — waitForTranslation(page, msg, timeout, {critical}) + critical flag
 M e2e/global-setup.ts                  — dev_seed + JWT clear (afterSeed)
 M e2e/playwright.config.ts             — add mobile-chrome project (Pixel 7, baseURL 10.0.2.2 when E2E_AVD)
 A e2e/webdriverio.conf.ts              — UiAutomator2 10.0.2.2 harness
 A mobile/detox.config.js               — Detox android.emu.debug
 A docs/QA_CRITIQUE_AND_IMPROVEMENTS.md — THIS FILE
 M docs/TEST_PLAN.md                    — addendum §12 Missing-Test-Cases (C-01..05 + AVD)
 A e2e/tests/20-comprehensive-two-user.spec.ts — failing-first C-01 (alice→bob 5 msgs + vocab/grammar/ai-tutor + settings → bob verify)
 A e2e/tests/21-learning-journey.spec.ts       — failing-first C-02 (placement/scenarios/real-talk/streak/lesson/monthly)
 A e2e/tests/22-settings-privacy.spec.ts       — failing-first C-03 (privacy+2FA enforcement)
 A e2e/tests/23-marketplace-e2e.spec.ts        — failing-first C-04 (browse→profile→book→trialCredits→dashboard→payouts)
 A e2e/tests/24-teacher-apply.spec.ts          — failing-first C-05 (become teacher UI)
```

**Builds must stay green:** `cd backend && go vet ./... && go test ./...` 0, `cd frontend && npx tsc --noEmit && npm run build && npm test` 205+new, `cd mobile && npx tsc --noEmit && npm test` 96+new, `cd e2e && npx playwright test --list` 143+~30 new (failing-first so list grows but run is red until impl).

---

## 7. Sign-off

| QA Engineer (zero-tolerance, crew/roles.py:54) | Date | Commit SHA | Device screenshots |
|---|---|---|---|
| ___________________________ | 2026-09-03 | `git rev-parse HEAD` | [ ] web C-01..C-05 [ ] AVD C-01..C-05 (10.0.2.2) [ ] ws_fast_dropped_total 0 [ ] message_receipts durable |

Rejected until: `e2e/fixtures/users.ts` Gmail purged + `global-setup` seeds + `waitForTranslation critical` enforced + `C-01..C-05` green on web + AVD project runs + `message_receipts` durability proven.

