# CrewAI Gap Closure Plan — Home + 62 Wireframe GAPs → TDD Loop with Sign-off

> **Authority:** `wireframes/chorus_home*/code.html:1` + `wireframes/chorus_home_desktop_v2/code.html:134` (newest home), `docs/WIREFRAME_TRACE.md:1` (94 entries, 62 GAP, 14 PARTIAL, 11 backend-only), `REQUIREMENTS_MASTER.md:1`, `REQUIREMENTS.md:1`, `docs/TEST_PLAN.md:1`, `docs/TDD_RESCUE_SPEC.md:1`
> **Goal:** Close every launch-blocking GAP via a strict **BA → QA (failing tests) → Impl (backend+frontend+mobile) → QA verify (device + automation) → BA sign-off** loop. No `phase_status.json:58` DONE without all gates green.

---

## 1. Why CrewAI Reported DONE but Home & Marketplace Were Hollow

| Failure | Symptom | Fix in this plan |
|---|---|---|
| `crew/autonomous_flow.py:233` marked DONE after `go vet`/`npm test` on mocks, never booted AVD nor `curl /health` | `frontend/src/pages/Landing.tsx:47` still v1 copy (`Break Language Barriers`) while `chorus_home_desktop_v2/code.html:138` expects `Communication is Learning` + brain hero | BA owns requirement, QA writes wireframe-vis test that fails until v2 hero renders; BA signs off only after device screenshot match |
| `roles.py:54` `qa_engineer` checked route strings, not `MainTabs.tsx:89` reachability | `browse_tutors:74` `GET /teachers/browse:614` had backend, zero UI | QA parity gate now asserts screen+route on **both** surfaces per `WIREFRAME_TRACE.md:60` row |
| No TDD — tests written after impl, so gaps had no failing signal | 62 GAP had no `testRefs` | Every slice gets **failing tests first** (`docs/TDD_RESCUE_SPEC.md:12` pattern) |
| No BA sign-off | `chorus_home_desktop_v2` vs `chorus_home` ambiguity never resolved | BA decides freeze v1 vs rebuild to v2, records in `REQUIREMENTS_MASTER.md` and signs `Gap sign-off` column |

---

## 2. Crew Roles (maps to `crew/agents.json:1` + `crew/roles.py:16`)

| Crew role | Agent suffix | Owns | Output |
|---|---|---|---|
| **analyst / Business Analyst** | `analyst` | Traceability `wireframes/* → REQUIREMENTS_MASTER.md § + code:line`; decides v1 vs v2 for Home; writes `docs/REQUIREMENTS_SLICE_<id>.md` per gap cluster | `GapAnalysis` + signed slice spec |
| **qa_engineer + test_engineer** | `qa` / `test` | Writes **failing** unit/integration/e2e **first**: `backend/*_test.go`, `frontend/vitest`, `mobile/jest`, `e2e/tests/*.spec.ts` + `e2e/acceptance/tests/*.spec.ts` vs real Postgres/Redis (`dev_seed.go:31` fixtures `alice/bob/sofia`) | `TestRefs` + red run |
| **backend_engineer** | `backend` | `handlers`, `services`, `migrations`, `api.ts` contract | `go build && go vet && go test ./...` green |
| **frontend_engineer** | `frontend` | `frontend/src/pages/*`, `components/*`, `App.tsx` routes | `npm run build` + `vitest` green |
| **mobile_engineer** | `mobile` | `mobile/src/screens/*`, `components/MainTabs.tsx`, `App.tsx` | `npx tsc --noEmit` + `jest` green |
| **teacher** | `teacher` | Learning-content review for Home copy + marketplace copy | Sign-off on CEFR/linguistics |
| **sre** | `infra` | `docker-compose.yml`, `start-android.ps1:19` flags, `L4 LB`, `/health:433` `commit` | `verify-release-gate.sh` + device boot |
| **reviewer** | `review` | Diff review + `release_gate:13` (wireframe + build + security + BA sign-off) | `PASS/CHANGES-REQUIRED` |

Common contract `roles.py:97` — mobile-first, web parity, Postgres durable, Redis never source of truth, wireframes ARE spec.

---

## 3. Loop — One Slice = 5 Stages, No Skip

```
BA: slice spec (Gherkin + wireframe PNG refs + API contract)  →  QA: write FAILING tests (red)  →  Impl: backend + frontend + mobile until green  →  QA: verify (automation + AVD device)  →  BA: sign-off
 ^                                                                                                                                       |
 └──────────────────────────────── if BA rejects → new failing test, back to QA ────────────────────────────────────────────────────────┘
```

**Gate per stage:**

1. **BA gate:** `docs/REQUIREMENTS_SLICE_<id>.md` exists, references `wireframes/<folder>/code.html:line` + `REQUIREMENTS_MASTER.md` FR + `backend/cmd/server/main.go:line` contract. No spec → QA cannot start.
2. **QA gate (red):** `testRefs` listed in `crew/phase_status.json` + `TDD_RESCUE_SPEC.md:12` style run is **red** on purpose (proves gap). `npm test` shows 1 fail.
3. **Impl gate (green):** `go vet 0`, `go test ./... 0`, `frontend tsc && vite build 0` + `NO_LEAK` (`grep alice.dev dist → 0`), `mobile tsc 0`, `jest` green.
4. **QA verify gate (device):** `.\start-android.ps1` boots, `curl /health` `commit` == HEAD (`health.go:39`), AVD `MainTabs` navigation reaches new screen, `e2e/tests` + `acceptance/tests` green, `verify-wireframe-parity.sh` row flips `GAP → PASS`.
5. **BA sign-off gate:** BA checks device screenshot vs wireframe PNG (e.g., home brain hero) + copy; marks `docs/WIREFRAME_TRACE.md:60` row `PASS` and signs `Gap sign-off Sheet` (`docs/GAP_SIGNOFF.md`). Only then `crew/state.py` may set slice `DONE`.

**If any gate red → slice stays `PENDING`, `phase_complete()` false, supervisor loops.**

---

## 4. Gap Clusters → Slices (prioritized, maps to `WIREFRAME_TRACE.md:216` Rec.1-5)

### Cluster H — Home (P0, blocks first impression)

**Decision BA must make first:** Freeze `chorus_home/code.html:200` (current `Break Language Barriers` + waitlist cafe) vs rebuild to `chorus_home_desktop_v2/code.html:134` (`Communication is Learning` + brain + 4-card ecosystem). Recommendation: **rebuild to v2** — it is the newest Figma, adds `Teacher Marketplace` tease needed for monetization. BA records choice in slice spec `S-HOME-01`.

| Slice | Wireframe(s) | BA Requirement (Gherkin) | QA TestRefs (failing first) | Impl Owner |
|---|---|---|---|---|
| **S-HOME-01** Hero v2 | `chorus_home_desktop_v2/code.html:138` + `chorus_home_mobile_v2` | `Given home unauthenticated, When I open / (web) or LandingScreen (mobile), Then I see h1 "Communication is Learning. Redefining how we acquire language." + brain neural image + CTAs "Start Your Journey" / "Watch Demo"` | `e2e/tests/00-home.spec.ts:heroV2` — `expect(page.getByRole('heading', {name:/Communication is Learning/}))` + mobile `App.test.tsx:hero` screenshot | `frontend_engineer: Landing.tsx:48` replace hero + `mobile: LandingScreen.tsx:96` + `app.json` asset |
| **S-HOME-02** Problem/Solution + Ecosystem | `chorus_home_desktop_v2/code.html:164` `Bridging Messaging and Learning…` + `code.html:184` 4 cards `AI Deep Dive / Real Talk / Teacher Marketplace / Phase 2 Ready` + mockup | `Then I see 4 ecosystem cards in order, Phase 2 badge Coming Soon, and app mockup image` | `e2e/tests/00-home.spec.ts:ecosystem` — 4 cards + `Coming Soon` locator; `vitest: Landing.test.tsx` | frontend/mobile |
| **S-HOME-03** Mission + Final CTA + Footer | `code.html:298` `Our Mission… linguists…` + `code.html:305` `Ready to reach fluency?` + footer `Product/Company/Support` with `Privacy/Terms/Help Center` | `Then I see mission section and final CTA and footer has 3 columns with 7 links` | `e2e/tests/00-home.spec.ts:footer` | front/mobile |
| **S-HOME-04** Pricing alignment | `code.html:228` `Free $0 280-char` vs `Premium $7.99 1000-char / Unlimited Deep Dives` | Align with `REQUIREMENTS_MASTER.md:145` 280/1000 + `entitlement.go` 280/1000 (`P0` premium copy 280 vs 28) | `vitest: words.test.ts` word cap 280/1000 + `pricing.spec.ts` | frontend/mobile |

**DoD home:** `WIREFRAME_TRACE.md:27` `chorus_home*` rows remain PASS but note “implemented as v2”.

### Cluster T — Teacher Marketplace (P0, 12 GAP, `WIREFRAME_TRACE.md:28`)

Backend already complete `teacher.go:130` `BrowseTutors`, `:103` `GetTutorProfile`, `:247` `TrialCreditDashboard`, `:633` `GetDashboard`, `payout.go:107` + routes `main.go:604-634`. UI is the gap.

| Slice | Wireframe | QA TestRefs |
|---|---|---|
| **S-T-01** Browse tutors | `browse_tutors:74` | `e2e/tests/tutor-browse.spec.ts: GET /teachers/browse → render Sofia card, filter by language` |
| **S-T-02** Tutor profile (Sofia) | `tutor_profile_sofia:91` | `TutorProfileScreen: GET /teachers/:id + reviews + availability` |
| **S-T-03** Find trial tutor | `find_a_trial_tutor:42` | Filtered browse with `trial-credit` badge |
| **S-T-04** Confirm trial booking | `confirm_trial_booking:39` | `POST /teachers/:id/book isTrial=true` → `TrialCredits` dashboard |
| **S-T-05** Trial credit dashboard | `trial_credit_dashboard:88` | `GET /teachers/trial-credits/dashboard` → `trial_credit_dashboard` UI |
| **S-T-06** Teacher dashboard + Earnings + Payouts | `teacher_dashboard:83`, `teacher_earnings_overview:84`, `payout_settings_history:65` | `GET /teachers/dashboard`, `GET /teachers/payouts/overview/history/methods` → 3 screens |

Impl: `frontend: BrowseTutors.tsx, TutorProfile.tsx, ConfirmBooking.tsx, TrialCredits.tsx, TeacherDashboard.tsx, Payouts.tsx` + `mobile: matching Screens` + `App.tsx:183` routes `/tutors`, `/tutors/:id`, `/tutors/:id/confirm` + `components/MainTabs.tsx:64` `TutorsTab` (4th tab Chats/Learn/Tutors/Profile). QA parity gate `wireframe_parity_audit:3` enumerates each.

### Cluster L — Learning Dashboards (P1, 7 GAP, `WIREFRAME_TRACE.md:42`)

| Slice | Wireframe | Action |
|---|---|---|
| **S-L-01** Activity Hub | `activity_hub:4` | Either promote to `frontend/src/pages/ActivityHub.tsx` + `mobile ActivityHubScreen` (`/learn/activity`) or BA documents as embedded in `Learn` and closes GAP as `PARTIAL→PASS` with note |
| **S-L-02** Journey/Progress/Insights | `learning_journey_progress:53`, `my_learning_progress_student:63`, `student_insights_progress_dashboard:79` | Tabs in `Learn` hub backed by `GET /learning/dashboard:659` |
| **S-L-03** Review-notes reader | `lesson_review_notes_student:56` | Student reads `PUT /teachers/bookings/:id/review-notes:625` |

### Cluster S — Social/Group/Community (P2, 8 GAP, deferred per `REQUIREMENTS.md:31`)

BA decision: **defer** — no backend. QA creates skipped e2e with `GAP: group_study_hub` tag, blocks `GO_NO_GO` axis only if P2 promoted.

### Cluster TS — Trust & Safety (P2)

`trust_safety_center:90`, `safety_alert_contextual_guidance:74`, `trust_safety_advanced_controls:89` → `TrustSafetyCenter` page aggregating `GET /blocks:487`, `GET /privacy/retention-policy:478`, `moderator_queue` on mobile for moderators.

---

## 5. New Crew Tasks (add to `crew/tasks.json:1` + `crew/autonomous_plan.md:7`)

```json
{
  "slice_home_v2": {"description":"BA writes S-HOME-01..04 specs vs chorus_home_desktop_v2, QA writes failing e2e, impl rebuilds Landing.tsx + LandingScreen.tsx","agent":"analyst"},
  "slice_marketplace": {"description":"BA specs S-T-01..06, QA writes failing tutor browse/profile/booking/credits/dashboard/payouts tests, impl builds 6 screens both surfaces","agent":"analyst"},
  "slice_learning_hub": {"description":"BA decides activity_hub standalone vs embedded, QA writes parity tests, impl adds ActivityHub or closes GAP","agent":"analyst"},
  "verify_home_device": {"description":"QA boots AVD, asserts hero v2, ecosystem 4 cards, mission, footer 7 links, pricing 280/1000 on both surfaces","agent":"qa_engineer"},
  "verify_marketplace_device": {"description":"QA verifies Browse→Profile→Confirm→TrialCredits→Dashboard→Payouts reachable on mobile + web AVD","agent":"qa_engineer"},
  "ba_signoff_home": {"description":"BA screenshots vs wireframe PNGs, signs GAP_SIGNOFF for home","agent":"analyst"},
  "ba_signoff_marketplace": {"description":"BA signs marketplace GAPs, updates WIREFRAME_TRACE.md rows to PASS","agent":"analyst"}
}
```

Loop adds `doc/REQUIREMENTS_SLICE_<id>.md` per slice (BA), `backend/*_test.go` + `e2e/tests/00-home.spec.ts` (QA red), `frontend/src/pages/Landing.tsx:48` + `mobile/src/screens/LandingScreen.tsx:96` (Impl), `qa_parity_gate:3` + `verify-release-gate.sh` (QA verify), `GAP_SIGNOFF.md` (BA sign).

---

## 6. How QA Writes Tests First (TDD, `docs/TDD_RESCUE_SPEC.md:12` pattern)

Example `S-HOME-01`:

```ts
// e2e/tests/00-home.spec.ts (fails until Landing rebuilt)
test('home hero is v2', async ({page})=>{
  await page.goto('/')
  await expect(page.getByRole('heading', {name: /Communication is Learning/})).toBeVisible()
  await expect(page.getByText('Redefining how we acquire language')).toBeVisible()
  await expect(page.getByRole('button', {name: 'Start Your Journey'})).toBeVisible()
  await expect(page.getByRole('img', {name: /Brain Neural/})).toBeVisible()
})
// mobile: __tests__/Home.test.tsx — same locators + snapshot
// backend: none (home is static + GET /health:433)
// after BA spec committed, `npm test` shows 1 fail → impl makes it green
```

All `testRefs` listed in `crew/phase_status.json` `testRefs` field; `state.py:97` `phase_complete()` requires all green **and** device boot.

---

## 7. Timeline (supervisor loop `crew/autonomous_flow.py:200`)

| Week | Crew cycles | Deliverable | Gate |
|---|---|---|---|
| **W1** | BA: S-HOME-01..04 + S-T-01..02 | Slice specs committed | `requirements_trace:7` PASS |
| **W1-2** | QA: failing tests for home + marketplace (2 PRs) | Red runs | `qa_parity_gate:11` FAIL (expected) |
| **W2** | frontend+mobile: Home v2 rebuild `Landing.tsx:48` (both) | `npm run build` + `mobile tsc` green | `verify_home_device` PASS |
| **W2-3** | Impl: S-T-01..06 (6 screens) backend already done, web+mobile parallel | `go test` + `vitest` + `jest` green | `verify_marketplace_device` PASS |
| **W3** | BA: screenshot sign-off `GAP_SIGNOFF.md` + `WIREFRAME_TRACE.md:27` update 18→31 PASS | BA sig | `release_gate:13` + `reviewer` PASS |

S is P2 — only after T green.

---

## 8. Verification Checklist (must attach to PR)

- [ ] `go vet ./... 0` + `go test ./... 0` (`backend:355` `phase 0` still green) + `frontend tsc && vite build` + `NO_LEAK` + `mobile tsc` + `jest` (existing 84 pass)
- [ ] New `e2e/tests/00-home.spec.ts` + `tutor-*.spec.ts` green on `e2e/acceptance` seeded DB (`dev_seed.go:31` `alice/bob/sofia`)
- [ ] AVD boot `emulator-5554 device` + `adb shell getprop sys.boot_completed 1` + `/health` `commit` == `git rev-parse HEAD`
- [ ] `wireframe_parity_audit:3` row `GAP→PASS` for home v2 + each marketplace screen on **both** surfaces (`MainTabs.tsx` + `App.tsx:117` routes)
- [ ] BA sign-off comment: “Home v2 matches `chorus_home_desktop_v2/code.html:138` PNG, Marketplace 6/6 reachable, Pricing 280/1000 correct” + updated `docs/WIREFRAME_TRACE.md:22` count 62→49 GAP

---

## 9. Risks

Teacher marketplace backend is done — risk is UI scope creep (12 screens). Mitigate by ordering S-T-01..03 first (browsing), S-T-04..06 behind. Home v2 vs v1 ambiguity mitigated by BA decision **now** (this plan proposes v2). `clean_modern_and_inspiring_hero_image` etc. N/A assets stay N/A per `WIREFRAME_TRACE.md:61`.

---

*Next step: BA commits `docs/REQUIREMENTS_SLICE_HOME_V2.md` + `REQUIREMENTS_SLICE_MARKETPLACE.md`; QA commits failing `00-home.spec.ts`; supervisor `crew/autonomous_flow.py:310` runs with `role=analyst → qa → frontend/mobile → qa → analyst` sequence above.*
