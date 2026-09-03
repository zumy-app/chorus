# REQUIREMENTS SLICE — Teacher Marketplace (S-T-01..06) — BA Trace

> **Authority:** `docs/WIREFRAME_TRACE.md:28` (Teacher marketplace 1/13 PASS, 12 GAP — `become_a_teacher` only), `docs/WIREFRAME_TRACE.md:13` `browse_tutors` GAP, `:91-92` `tutor_profile_sofia`/`confirm_trial_booking`/`:88` `trial_credit_dashboard`/`:83-84-65` `teacher_dashboard`/`teacher_earnings_overview`/`payout_settings_history`, `docs/CREWAI_GAP_CLOSURE_PLAN.md:71-85` Cluster T (S-T-01..06), `REQUIREMENTS_MASTER.md:5.1` `12.1-12.7` Teacher marketplace + `13.1` Credits & Access, `REQUIREMENTS.md:139` P1–P4 + `§5.3` Teacher vetting, `backend/cmd/server/main.go:638-668` teacher+payout routes, `backend/internal/services/teacher.go:132` `BrowseTutors` / `:105` `GetTutorProfile` / `:232` `GetTrialCredit` / `:249` `GetTrialCreditDashboard` / `:635` `GetDashboard`, `backend/internal/services/payout.go:107` `GetOverview`, `backend/internal/services/dev_seed.go:79` `seedTutorMarketplace` Sofia tutor (approved, 8yrs, 2500c, 4 slots, 2 reviews, trial credit), `backend/internal/handlers/teacher.go:57` `Browse` / `:96` `GetProfile` / `:286` `CreateBooking`, `backend/internal/handlers/payout.go:22` `GetOverview`, `packages/shared/src/api.ts:1107-1187` `teacher.*` + `:926-953` `payouts.*`, `frontend/src/App.tsx:247-253` marketplace routes, `mobile/src/components/MainTabs.tsx:180-202` `MarketplaceTab`
> **BA Owner:** analyst (`crew/roles.py:22` — owns requirements traceability matrix, maps every wireframe → requirement id → code, gap list is source of truth, QA cannot pass until trace green)
> **Common Contract:** `crew/roles.py:97` — mobile-first / web parity (NFR-22), Go+Gin + Postgres (source of truth) + Redis (cache/pubsub/registry only), Vite React web, Expo RN primary, wireframes ARE spec
> **Generated:** 2026-09-03 | **Status:** BA FINAL — backend already exists (Sofia seed), slices focus on **UI parity + tests** only. Do NOT yet implement screens — just spec. Impl blocked until QA writes failing tests first (`docs/CREWAI_GAP_CLOSURE_PLAN.md:39` Stage 2 red gate, `docs/TDD_RESCUE_SPEC.md:12`)

---

## 0. Premise — Backend Done, UI Is the Gap

**Backend is DONE and live on `main.go:638-668`:**

```
GET    /teachers/payouts/overview          payout.go:22
GET    /teachers/payouts/history           payout.go:82
POST   /teachers/payouts/withdraw          payout.go:94
GET    /teachers/payouts/methods           payout.go:35
POST   /teachers/payouts/methods           payout.go:44
DELETE /teachers/payouts/methods/:methodId payout.go:58
PUT    /teachers/payouts/methods/:methodId/default payout.go:70

POST   /teachers/apply                     teacher.go:27
GET    /teachers/me                        teacher.go:43
GET    /teachers/browse                    teacher.go:57  → services/teacher.go:132 BrowseTutors
GET    /teachers/dashboard                 teacher.go:233 → services/teacher.go:635 GetDashboard
GET    /teachers/trial-credits             teacher.go:114 → services/teacher.go:232 GetTrialCredit
GET    /teachers/trial-credits/dashboard   teacher.go:124 → services/teacher.go:249 GetTrialCreditDashboard
GET    /teachers/bookings                  teacher.go:310
GET    /teachers/bookings/:id              teacher.go:134
POST   /teachers/availability              teacher.go:257
DELETE /teachers/availability/:id          teacher.go:272
POST   /teachers/bookings/:id/cancel       teacher.go:324
POST   /teachers/bookings/:id/confirm      teacher.go:445
POST   /teachers/bookings/:id/complete     teacher.go:153
PUT    /teachers/bookings/:id/review-notes teacher.go:172
POST   /teachers/srs/push                  teacher.go:343
GET    /teachers/srs/pushes                teacher.go:370
GET    /teachers/srs/pushes/:id            teacher.go:395
GET    /teachers/srs/sandbox/:studentId    teacher.go:418
GET    /teachers/:id                       teacher.go:96  → services/teacher.go:105 GetTutorProfile
GET    /teachers/:id/reviews               teacher.go:196 → services/teacher.go:284 ListReviews
POST   /teachers/:id/reviews               teacher.go:209
GET    /teachers/:id/availability          teacher.go:247 → services/teacher.go:351 GetAvailability
POST   /teachers/:id/book                  teacher.go:286 → services/teacher.go:416 CreateBooking (isTrial)
```

Deterministic seed `backend/internal/services/dev_seed.go:31` `SeedDevData` + `:79` `seedTutorMarketplace` provisions:
- `sofia.tutor@chorus.test` / `sofia.tutor` / `Sofia Tutor` — `teacher_applications` approved, bio ES, `{es}`, expertise `Conversational Spanish, DELE A1-B1 prep` `:83`, `rate_cents=2500` ($25), verified `language_certificate` Instituto Cervantes 2018 `:91`, 4 availability slots next 4 days hourly `:99-108`, 2 reviews (Alice 5, Bob 4) `:110-126`, trial credit for Alice `:128`
- Learners `alice.dev@chorus.test` / `bob.dev@chorus.test` — password `ChorusDev123!` `:17`

**UI thin but present:** `frontend/src/App.tsx:247-253` already mounts `/tutors`, `/tutors/:id`, `/tutors/:id/confirm`, `/trial-credits`, `/teacher/dashboard`, `/teacher/payouts` + pages `BrowseTutors.tsx:1`, `TutorProfile.tsx:1`, `ConfirmBooking.tsx:1`, `TrialCredits.tsx:1`, `TeacherDashboard.tsx:1`, `Payouts.tsx:1`; `mobile/src/components/MainTabs.tsx:180-202` mounts `MarketplaceTab` with `BrowseTutors`/`TutorProfile`/`ConfirmBooking`/`TrialCredits`/`TeacherDashboard`/`Payouts`. `docs/WIREFRAME_TRACE.md:28` GAP was audit-time; current slices **harden** those screens to wireframe visual contract + add failing-tests-first coverage + navigation reachability. No new backend migration is required for S-T-01..06. Implementing screens without this spec is **out of scope**.

---

## 1. Slice Map — Wireframe → Requirement → Backend → Frontend/Mobile Route

### 1.1 Traceability matrix (every marketplace wireframe cited in `WIREFRAME_TRACE.md:28-40`)

| # | Wireframe folder (`wireframes/`) | `REQUIREMENTS_MASTER.md` | `REQUIREMENTS.md` | Backend route (`main.go:line` + service) | Frontend route (`App.tsx:line`) | Mobile route (`MainTabs.tsx:line`) | Status |
|---|---|---|---|---|---|---|---|
| 1 | `browse_tutors` `code.html:131-312` | `12.2` Browse tutors / find trial tutor | `P5 T01` Tutor Search | `GET /teachers/browse` `:648` → `teacher.go:132` `BrowseTutors` (filter `language`, `search`, `verified`, `minRating`, `maxRate`, `sort`, `limit/offset`) | `/tutors` `App.tsx:247` `BrowseTutors.tsx:1` | `MarketplaceTab/BrowseTutors` `MainTabs.tsx:183` `BrowseTutorsScreen` | HARDEN — filters + Featured/Available Now sections must match `code.html:156-287` |
| 2 | `find_a_trial_tutor` `code.html` (filtered browse variant) | `12.2` + `12.5` trial credit | `P5 T01` trial | `GET /teachers/browse` `:648` (filter `isTrial` via search) + `GET /teachers/trial-credits` `:650` | `/tutors?filter=trial` (same `BrowseTutors.tsx:18` `teacherAPI.browse`) | same `BrowseTutorsScreen` filtered | HARDEN — same screen, query-param filter |
| 3 | `tutor_profile_sofia` `code.html:171-396` | `12.3` Tutor profile | `P5 T01` profiles | `GET /teachers/:id` `:664` → `teacher.go:105` `GetTutorProfile` + `GET /teachers/:id/reviews` `:665` + `GET /teachers/:id/availability` `:667` + `GET /teachers/:id/book` via `CreateBooking` | `/tutors/:id` `App.tsx:248` `TutorProfile.tsx:1` | `MarketplaceTab/TutorProfile {userId}` `MainTabs.tsx:188` `TutorProfileScreen` | HARDEN — Sofia hero + stats + reviews + pricing + booking calendar per `code.html:175-393` |
| 4 | `confirm_trial_booking` `code.html` | `12.5` Session flows confirm booking | `P5 T01` booking | `POST /teachers/:id/book` `:668` → `teacher.go:416` `CreateBooking` `isTrial=true` (consumes `tutor_trial_credits`, validates overlap `teacher.go:438`, duration 15m-3h) + `GET /teachers/trial-credits/dashboard` `:651` for credit check | `/tutors/:id/confirm` `App.tsx:249` `ConfirmBooking.tsx:1` | `MarketplaceTab/ConfirmBooking {userId}` `MainTabs.tsx:192` `ConfirmBookingScreen` | HARDEN — Payment Summary $0.00 + cancellation policy per wireframe |
| 5 | `trial_credit_dashboard` `code.html` | `12.5` trial credit dashboard + `13.1` 1 credit/mo | `P4` Credits & Access | `GET /teachers/trial-credits/dashboard` `:651` → `teacher.go:249` `GetTrialCreditDashboard` (credits/nextGrant/history) + `GET /teachers/trial-credits` `:650` | `/trial-credits` `App.tsx:250` `TrialCredits.tsx:1` | `MarketplaceTab/TrialCredits` `MainTabs.tsx:198` `TrialCreditsScreen` | HARDEN — credits card + How Trials Work + Recommended + History per wireframe |
| 6 | `teacher_dashboard` `code.html` | `12.4` Teacher dashboard | `P5 T02` | `GET /teachers/dashboard` `:649` → `teacher.go:635` `GetDashboard` (application+checklist+pct+availability+earnings+upcoming+students) | `/teacher/dashboard` `App.tsx:251` `TeacherDashboard.tsx:1` | `MarketplaceTab/TeacherDashboard` `MainTabs.tsx:199` `TeacherDashboardScreen` | HARDEN — checklist 7 items, earnings, availability, students per wireframe |
| 7 | `teacher_earnings_overview` `code.html` | `12.7` Payouts + `12.4` earnings + `13.1` 10/15% fee | `P5 T02` Payments | `GET /teachers/payouts/overview` `:638` → `payout.go:107` `GetOverview` (available/pending/lifetimeGross/lifetimeNet/feePct/nextPayoutDate/hoursTaught/activeStudents/recentTx) | `/teacher/payouts` (earnings section) `App.tsx:252` `Payouts.tsx:1` | `MarketplaceTab/Payouts` `MainTabs.tsx:200` `PayoutsScreen` earnings card | HARDEN — Earnings Overview section |
| 8 | `payout_settings_history` `code.html` | `12.7` Payout settings/history | `P5 T02` Payments | `GET /teachers/payouts/methods` `:641` + `POST` `:642` + `DELETE` `:643` + `PUT default` `:644` + `POST withdraw` `:640` + `GET history` `:639` → `payout.go:107-409` | `/teacher/payouts` `App.tsx:252` `Payouts.tsx:46-105` methods+withdraw+history | same | HARDEN — methods + withdraw + history list |
| 9 | `become_a_teacher` `code.html` | `12.1` Sign up as teacher | `P5` Marketplace | `POST /teachers/apply` `:646` + `GET /teachers/me` `:647` → `teacher.go:54` `Apply` | `/become-teacher` `App.tsx:242` `BecomeTeacher.tsx:1` | `MarketplaceTab/BecomeTeacher` + `ProfileTab/BecomeTeacher` `MainTabs.tsx:193,196` | PASS — scope excluded (already green, not in S-T) |
| 10 | `student_management_progress_teacher` `code.html` | `12.4` students | `P5 T02` | `GET /teachers/dashboard` `:649` students array | `/teacher/dashboard` students section `TeacherDashboard.tsx:58-61` | same | Covered by S-T-05 |
| 11 | `lesson_review_notes_student` `code.html` | `12.5` review notes | `P5` | `PUT /teachers/bookings/:id/review-notes` `:659` → `teacher.go:593` + `GET /teachers/bookings/:id` `:653` | `/teacher/dashboard` → booking detail (future) | same | P1 polish — noted, not in S-T slices |
| 12 | `teacher_student_learning_chat`, `custom_activity_builder_pusher`, `teacher_pronunciation_review_dashboard`, `group_study_hub` etc. | `12.6` SRS push + `11.x` | `P5 T03/T04` | `POST /teachers/srs/push` `:660` | `/teacher/srs` (future) | sandbox via `teacher.go:418` | Out of S-T scope (deferred) |

**Remaining 6 GAP rows above that are NOT in S-T-01..06 are explicitly deferred to post-marketplace slices (see §6).**

---

## 2. Slices — S-T-01..06 (UI + tests only, backend already green)

### S-T-01 — Browse Tutors + Find a Trial Tutor

| Axis | Value |
|---|---|
| **Wireframe refs** | `wireframes/browse_tutors/code.html:131-312` — `header:120-130` TopAppBar (Chorus + wallet), `134-154` Search `Find a tutor or language...` + filters Language/Price/Rating `139-153`, `156-198` Featured Tutors horizontal carousel (Elena 4.9, David 5.0 verified), `200-287` Available Now list (Amelie $18 Premium Book Trial, Lukas $22 View Profile, Yui $25 Book Trial), `290-311` BottomNav Tutors active; `find_a_trial_tutor/code.html:1` same screen filtered by trial availability (Sofia triallable) |
| **REQUIREMENTS_MASTER.md** | `12.2` Browse tutors / find trial tutor (filters, rating, verified, trial credit). Reconciles `REQUIREMENTS.md:7b` Phase 5 T01. Narrative: `wireframes/chorus_missing_requirements_roadmap.md:1` Phase 1 gaps. |
| **Backend contract** | `GET /teachers/browse` `main.go:648` `teacher.go:57` → `services/teacher.go:132` `BrowseTutors(models.TutorBrowseFilter) ([]TutorProfile, int, error)`<br>Query: `?language=es&search=sofia&verified=true&minRating=4.5&maxRate=3000&sort=rating|price_asc|price_desc|newest&limit=20&offset=0`<br>Auth: `Authorization: Bearer <JWT>` (protected). Pagination `limit<=50` else 50. Returns `{tutors: TutorProfile[], total:int, hasMore:bool}`. No new endpoint. `BrowseTutors` orders by `created_at DESC` / rating / price; verified filter checks `EXISTS teacher_certificates verified=true`. Seed proves `sofia.tutor` browsable with `search=sofia&language=es`.<br>Shared client `packages/shared/src/api.ts:1116-1119` `teacher.browse(params)` + `frontend/src/services/api.ts:52` `teacherAPI`. |
| **Frontend route** | `frontend/src/pages/BrowseTutors.tsx:1` mounted `App.tsx:247` `/tutors` (auth guard → `/login`). State `q`, `tutors`, `loading`, `msg`; calls `teacherAPI.browse({search: q, limit:20})` `BrowseTutors.tsx:18`. |
| **Mobile route** | `mobile/src/screens/BrowseTutorsScreen.tsx:1` via `MainTabs.tsx:183` `MarketplaceTab/BrowseTutors` title `Tutors`. 4th tab `Tutors` `MainTabs.tsx:207` `label=Tutors glyph=🏫` `TabIconMarketplace`. |
| **Depends on** | Auth (`POST /auth/login` → `alice.dev@chorus.test` / `ChorusDev123!` `dev_seed.go:17`), `MarketplaceTab` exists. No prereq slice. |

**API contract (existing — must not drift):**

```
GET /api/v1/teachers/browse?search=sofia&language=es&verified=1&minRating=4&sort=rating&limit=20&offset=0
Authorization: Bearer <jwt>

200 {
  "tutors": [{
    "id": "uuid",
    "userId": "sofia.tutor uuid",
    "displayName": "Sofia Tutor",
    "bio": "Hola! I am Sofia...",
    "languages": ["es"],
    "expertise": "Conversational Spanish, DELE A1-B1 prep",
    "rateCents": 2500,
    "videoUrl": "",
    "status": "approved",
    "verified": true,
    "ratingAvg": 4.5,
    "ratingCount": 2,
    "avatarColor": "#6b38d4",
    "avatarUrl": null,
    "certificates": [...],
    "createdAt": "2026-09-03T...",
    "updatedAt": "2026-09-03T..."
  }],
  "total": 1,
  "hasMore": false
}
```

**Gherkin — S-T-01:**

```gherkin
@S-T-01 @marketplace @browse @wireframe-browse_tutors @wireframe-find_a_trial_tutor
Feature: Browse tutors + Find a trial tutor

  Background:
    Given dev seed ran (sofia.tutor approved, rateCents 2500, verified cert, 2 reviews, alice trial credit 1 — dev_seed.go:79)
    And I am authenticated as alice.dev@chorus.test (POST /auth/login with ChorusDev123!)

  Scenario: Web browse renders Featured + Available Now with filters
    When I open "/tutors" on web (App.tsx:247 BrowseTutors.tsx:31)
    Then I see h2 "Tutors" and link "Become a teacher" href "/become-teacher"
    And I see input placeholder "Find a tutor or language..." (data-testid tutor-search) and button "Search"
    And I see section "Featured Tutors" (or carousel) and "Available Now" list
    And Sofia card shows displayName "Sofia Tutor", languages "es", rating ~4.5, "$25/session", Verified badge (when verified)
    And Browse uses GET /teachers/browse with limit 20 (teacherAPI.browse)
    And filters Language/Price/Rating are visible (browse_tutors/code.html:139-153)

  Scenario: Search filters to Sofia
    When I type "sofia" and press Search (or Enter)
    Then GET /teachers/browse?search=sofia returns 1 tutor (sofia.tutor uuid)
    And list shows Sofia card; other tutors filtered out

  Scenario: Find a trial tutor (filtered variant)
    Given GET /teachers/trial-credits returns credits=1 (teacher.go:232)
    When I filter browse for trial-eligible (or open /tutors?filter=trial which maps to search with trial badge)
    Then Sofia is bookable via "Book Trial" and trial credit badge is visible

  Scenario: Mobile parity
    When I open MarketplaceTab → BrowseTutors on iOS/Android (MainTabs.tsx:183)
    Then same search input, Featured + Available Now, Sofia card, Verified badge, $25/session render with native StyleSheet
    And tapping card navigates to TutorProfile {userId: sofia uuid}

  Scenario: Empty state
    When I search "zzz_no_tutor_zzz"
    Then I see "No tutors yet" + "Try a different search." (BrowseTutors.tsx:41)
```

**QA testRefs (failing first, then green — `CREWAI_GAP_CLOSURE_PLAN.md:39`):**

| Suite | File | Locator (must fail until parity hardened) |
|---|---|---|
| `e2e` | `e2e/tests/tutor-browse.spec.ts:browse` | `await page.goto('/tutors')` → `expect(page.getByPlaceholder('Find a tutor or language')).toBeVisible()` + `expect(page.getByRole('heading', {name:'Tutors'}))` + `expect(page.getByText('Sofia Tutor'))` + `page.getByTestId('tutor-search').fill('sofia')` → `expect(page.getByText('$25')).toBeVisible()` + verify `GET /teachers/browse?search=sofia` 200 |
| `vitest` | `frontend/src/__tests__/BrowseTutors.test.tsx` | `render(<BrowseTutors/>)` mock `teacherAPI.browse` → resolve sofia → `getByPlaceholder` + `getByText('Sofia Tutor')` + `getByText('Verified')` + search triggers browse with correct params |
| `jest` | `mobile/__tests__/BrowseTutorsScreen.test.tsx` | `render(<BrowseTutorsScreen/>)` → `getByPlaceholder('Find a tutor or language')` + `getByText('Sofia Tutor')`; `fireEvent.press` card → `navigate('TutorProfile', {userId})` |
| `backend` | existing `teacher_test.go: browse` + `dev_seed.go:31` | `GET /teachers/browse` returns sofia when `seedDevData` run; `BrowseTutors` verified/language/rating filters unit tests — already green, no new backend test |

**DoD — S-T-01:**
- `WIREFRAME_TRACE.md:13` `browse_tutors` row flips GAP → PASS with note `S-T-01 BrowseTutors.tsx:31 + BrowseTutorsScreen + GET /teachers/browse:648 sofia seed dev_seed.go:79 date BA sig`
- `find_a_trial_tutor` gap closed as filtered variant of same screen (no separate route; document `?filter=trial` param or trial badge in PR)
- Device screenshot (web + AVD) side-by-side with `browse_tutors/code.html:131` search + Featured + Available Now matches copy/assets
- `grep -R "BrowseTutors" frontend/src/App.tsx mobile/src/components/MainTabs.tsx` shows both surfaces mounted
- `cd frontend && npm test` (vitest) + `cd mobile && npm test` (jest) green, new Browse tests pass; `e2e/tests/tutor-browse.spec.ts` green on dev stack seeded with `go run ./cmd/server --seed-dev`

---

### S-T-02 — Tutor Profile (Sofia)

| Axis | Value |
|---|---|
| **Wireframe refs** | `wireframes/tutor_profile_sofia/code.html:171-396` — `175-216` Hero (aspect 4/3 image Sofia, play_arrow Video Intro 1:20, `190` `Sofia R.` + `192` Verified Tutor, `196` Native Spanish Speaker Madrid, badges Conversational/Business/DELE `201-209`, `212` Message Sofia), `219-235` Stats (4.9 124 Reviews, 850+ Classes, 320 Students), `237-253` About Me (Hola! 5 years… communicative… Chorus AI Summary), `255-303` Student Reviews (David K. 5★, Emma T. 4.5★), `308-393` Pricing Options (Single $25 selected, Monthly $80 Best Value) + `335-393` Booking calendar (Oct 16-22 date scroller, Tue 17 selected, time slots 09:00/10:00 selected/02:00/04:30, Confirm & Book $25) |
| **REQUIREMENTS_MASTER.md** | `12.3` Tutor profile (video intro, specialties, reviews, pricing) + booking/scheduling. Narrative `REQUIREMENTS.md:5.3` Teacher vetting note + `P5 T01`. |
| **Backend contract** | `GET /teachers/:id` `main.go:664` `teacher.go:96` → `services/teacher.go:105` `GetTutorProfile(userID)` returns `TutorProfile` (joins `teacher_applications` + `users`, aggregates `AVG rating`, `COUNT reviews`, `verified` cert). 404 if not approved.<br>`GET /teachers/:id/reviews` `:665` `teacher.go:196` → `services/teacher.go:284` `ListReviews(teacherUserID, limit, offset)` → `{reviews, total, hasMore}`<br>`GET /teachers/:id/availability` `:667` `teacher.go:247` → `services/teacher.go:351` `GetAvailability` → `[TutorAvailability]` filtered `end_time > NOW()`<br>Shared `teacher.getProfile`, `getReviews`, `getAvailability` `api.ts:1120-1139`. |
| **Frontend route** | `/tutors/:id` `App.tsx:248` `TutorProfile.tsx:1` (`useParams id`, `teacherAPI.getProfile(id)`, `getReviews(id, limit:5)`, block/report via `moderationAPI` `TutorProfile.tsx:28`). |
| **Mobile route** | `MarketplaceTab/TutorProfile {userId}` `MainTabs.tsx:188` `TutorProfileScreen` title `Tutor`. |
| **Depends on** | S-T-01 (entry via Browse card tap) |

**API contracts:**

```
GET /api/v1/teachers/:id           → 200 {tutor: TutorProfile}
GET /api/v1/teachers/:id/reviews?limit=5&offset=0 → 200 {reviews: TutorReview[], total, hasMore}
GET /api/v1/teachers/:id/availability           → 200 {availability: TutorAvailability[]}
Models: TutorProfile {id,userId,displayName,bio,languages,expertise,rateCents,videoUrl,status,verified,ratingAvg,ratingCount,avatarColor,certificates} models.go:1054; TutorReview {id,teacherUserId,studentUserId,rating,comment,studentName} models.go:1092; TutorAvailability {id,teacherUserId,startTime,endTime}
```

**Gherkin — S-T-02:**

```gherkin
@S-T-02 @marketplace @profile @wireframe-tutor_profile_sofia @sofia
Feature: Tutor profile — Sofia (verified tutor canonical)

  Background:
    Given sofia.tutor approved seed (dev_seed.go:79 bio "Hola! I am Sofia, a certified..." rateCents 2500 verified cert, 2 reviews)
    And I am authenticated as alice.dev

  Scenario: Web profile renders Sofia hero
    When I open "/tutors/:id" for sofia uuid (App.tsx:248 TutorProfile.tsx:24)
    Then I see avatar with initial "S", displayName "Sofia Tutor" (or "Sofia R." per wireframe), Verified badge when verified
    And I see languages "es" (or "Native Spanish Speaker"), rate "$25/session", rating "4.5" (avg of 5+4) and "2 reviews"
    And I see About card with bio containing "Hola! I am Sofia" and expertise "Conversational Spanish, DELE A1-B1"
    And I see Reviews section with 2 reviews (5★ Alice, 4★ Bob) when reviews exist
    And I see actions: "Message Sofia" (or Browse), "Book Trial" (data-testid book-trial), Block/Report (TutorProfile.tsx:72-77)
    And tapping Book Trial navigates to "/tutors/:id/confirm"

  Scenario: Availability & pricing
    Given GET /teachers/:id/availability returns 4 slots (seed 4 days hourly dev_seed.go:99)
    When I scroll to booking section (or pricing card per wireframe 308-393)
    Then I see pricing "$25 Single (50 min)" and availability dates/times (when UI surfaces calendar)

  Scenario: Mobile parity
    When I open TutorProfile {userId: sofia uuid} on mobile (MainTabs.tsx:188)
    Then same hero, Verified, rating, bio, expertise, reviews, Book Trial CTA render natively
    And 404 when GET /teachers/:id returns non-approved (teacher.go:107-109) → "Tutor not found" (TutorProfile.tsx:37)

  Scenario: Sofia is the example TutorProfile (traceability)
    Given WIREFRAME_TRACE.md:91 says "Sofia is example TutorProfile"
    Then GET /teachers/browse with search=sofia returns the same userId rendered here
```

**QA testRefs:**

| Suite | File | Locator |
|---|---|---|
| `e2e` | `e2e/tests/tutor-profile.spec.ts:sofia` | `goto('/tutors/${sofiaId}')` → `expect(page.getByText('Sofia Tutor')).toBeVisible()` + `getByText('Verified')` + `getByText(/Hola! I am Sofia/)` + `getByTestId('book-trial')` → click → `expect(page).toHaveURL(/\/tutors\/.*\/confirm/)` + mock `GET /teachers/:id/reviews` → `getByText('Sofia made my first lesson')` |
| `vitest` | `frontend/src/__tests__/TutorProfile.test.tsx` | `render(<TutorProfile/>)` with `teacherAPI.getProfile` mocked sofia → same assertions + `getByTestId('block-tutor')` + `getByTestId('report-tutor')` + 404 branch |
| `jest` | `mobile/__tests__/TutorProfileScreen.test.tsx` | `render(<TutorProfileScreen route={{params:{userId:sofiaId}}}/>)` → same hero + `fireEvent.press(getByTestId('book-trial'))` → navigate ConfirmBooking |
| `backend` | existing | `GET /teachers/:id` 200 for approved sofia, 404 for pending/unapproved, reviews page 1 returns 2 rows |

**DoD — S-T-02:**
- Visual match: web + AVD screenshots show Sofia hero (image + Verified + languages + rating + bio + reviews + Book Trial) vs `tutor_profile_sofia/code.html:175-303`
- Navigation: Browse card → Profile → Confirm is reachable without crash (`go test` + `e2e` + manual AVD `MarketplaceTab` tap)
- `WIREFRAME_TRACE.md:91` `tutor_profile_sofia` flips GAP → PASS with note `S-T-02 TutorProfile.tsx:44 + TutorProfileScreen + GET /teachers/:id:664 dev_seed.go:79`

---

### S-T-03 — Confirm Trial Booking

| Axis | Value |
|---|---|
| **Wireframe refs** | `wireframes/confirm_trial_booking/code.html:1` — `48-52` header Confirm Booking back arrow + title, `54-58` Great choice! check, `59-66` Your Tutor card (Sofia, Spanish Native), `67-77` Date/Time cards (Date label, Time 10:00 AM - 10:30 AM local), `78-84` Payment Summary (Trial Session 1 Credit, Credits Applied -1, Total $0.00), `85-86` Cancellation Policy (free up to 24h), `91-92` sticky Confirm Booking button |
| **REQUIREMENTS_MASTER.md** | `12.5` Session flows: confirm booking, trial credit dashboard. `13.1` 1 trial credit/mo (trial is $0, consumes 1 credit). |
| **Backend contract** | `POST /teachers/:id/book` `main.go:668` `teacher.go:286` → `services/teacher.go:416` `CreateBooking(studentUserID, teacherUserID, CreateBookingRequest{startTime, endTime, isTrial, note})`<br>Validation: not self `teacher.go:417`, `end>start` `duration 15m-3h` `start future` `:424-428`, tutor approved `:430`, no overlap `status pending/confirmed` `:438`, trial credit check `GetTrialCredit` ≤0 → `no trial credits remaining` `:453`, tx inserts `tutor_bookings status pending` + decrements `tutor_trial_credits` `:469`.<br>Idempotent error mapping `teacher.go:296-303`: 404 tutor not found/approved, 400 validation (no trial, overlap, yourself, duration).<br>Shared `teacher.book(userId, {startTime,endTime,isTrial,note})` `api.ts:1147`. |
| **Frontend route** | `/tutors/:id/confirm` `App.tsx:249` `ConfirmBooking.tsx:1` (`teacherAPI.book(id, {startTime,endTime,isTrial:true})` `:35`), computes `tomorrow 10:00-10:30` `:22-34`, on success navigates `trial-credits` `:37`. |
| **Mobile route** | `MarketplaceTab/ConfirmBooking {userId}` `MainTabs.tsx:192` `ConfirmBookingScreen` title `Confirm Booking`. |
| **Depends on** | S-T-02 (needs tutor profile), `GET /teachers/trial-credits` for credit guard |

**API contract:**

```
POST /api/v1/teachers/:id/book
Authorization: Bearer <jwt>
{ "startTime": "2026-09-04T10:00:00Z", "endTime": "2026-09-04T10:30:00Z", "isTrial": true, "note": "" }

201 { "booking": { "id":"uuid","teacherUserId":"sofia","studentUserId":"alice","startTime":"...","endTime":"...","status":"pending","isTrial":true,"note":"","createdAt":"..." } }
400 { "error": "no trial credits remaining" } | "time slot already booked" | "duration must be 15m-3h"
404 { "error": "tutor not found" }
```

**Gherkin — S-T-03:**

```gherkin
@S-T-03 @marketplace @booking @wireframe-confirm_trial_booking @trial
Feature: Confirm trial booking (isTrial=true, $0.00)

  Background:
    Given alice has credits=1 (GET /teachers/trial-credits/dashboard:651)
    And sofia has availability tomorrow 10:00-11:00 (dev_seed.go:79 + GET /teachers/:id/availability)
    And I am alice viewing sofia profile then tapping Book Trial

  Scenario: Confirm screen renders wireframe contract
    When I open "/tutors/:id/confirm" (App.tsx:249 ConfirmBooking.tsx:53)
    Then I see header "Confirm Booking" with back arrow
    And I see "Great choice!" + "Review your trial session details"
    And tutor card shows "Sofia Tutor" + "Spanish · Native"
    And Date shows tomorrow (e.g., "Thursday Sep 04") and Time "10:00 AM - 10:30 AM (local)"
    And Payment Summary shows "Trial Session 1 Credit", "Credits Applied -1 Credit", Total "$0.00"
    And Cancellation Policy text "24 hours before your trial"
    And CTA "Confirm Booking" (data-testid confirm-booking) is sticky bottom (ConfirmBooking.tsx:91)

  Scenario: Confirm consumes trial credit and navigates to dashboard
    When I tap "Confirm Booking"
    Then POST /teachers/:id/book with isTrial=true is sent with start 10:00 end 10:30 (ConfirmBooking.tsx:32-35)
    And on 201 I see "Trial booked! Check your bookings." and after 1.2s navigate to "/trial-credits"
    And GET /teachers/trial-credits returns credits=0 after booking (GetTrialCredit:242 decrements)

  Scenario: No trial credit guard
    Given alice credits=0 (after previous booking)
    When I tap Confirm Booking again
    Then POST fails 400 "no trial credits remaining" and error is shown (ConfirmBooking.tsx:39)

  Scenario: Validation guards
    When I attempt booking with end==start or duration <15m or >3h or overlapping slot
    Then API returns 400 "end must be after start" / "duration must be 15m-3h" / "time slot already booked"

  Scenario: Mobile parity
    When I open ConfirmBooking {userId: sofia} on mobile (MainTabs.tsx:192)
    Then same header, tutor card, date/time, Payment Summary $0.00, cancellation policy, sticky Confirm CTA render
```

**QA testRefs:**

| Suite | File | Locator |
|---|---|---|
| `e2e` | `e2e/tests/tutor-booking.spec.ts:confirm` | `goto('/tutors/${sofiaId}/confirm')` → `expect(getByText('Confirm Booking')).toBeVisible()` + `getByText('Great choice')` + `getByText('Payment Summary')` + `getByText('$0.00')` + `getByTestId('confirm-booking').click()` → `waitForResponse('/teachers/.*/book')` 201 → `expect(page).toHaveURL('/trial-credits')` + second booking `expect(page.getByText('no trial credits')).toBeVisible()` |
| `vitest` | `frontend/src/__tests__/ConfirmBooking.test.tsx` | mock `teacherAPI.getProfile` sofia + `teacherAPI.book` resolve → same render + `fireEvent.click(getByTestId('confirm-booking'))` → book called with `isTrial:true` |
| `jest` | `mobile/__tests__/ConfirmBookingScreen.test.tsx` | same + navigation to TrialCredits on success |
| `backend` | existing `teacher_test.go: createBooking` | `POST /teachers/:id/book isTrial true` consumes credit, 400 when credits 0, overlap 400, approved check 404 |

**DoD — S-T-03:**
- Screenshot web + AVD of confirm screen vs `confirm_trial_booking/code.html:48-92` (Great choice, tutor card, Date/Time, Payment Summary $0.00, sticky CTA)
- `WIREFRAME_TRACE.md:39` `confirm_trial_booking` flips GAP → PASS with note `S-T-03 ConfirmBooking.tsx:53 + ConfirmBookingScreen + POST /teachers/:id/book:668 isTrial`
- `e2e/tutor-booking.spec.ts` green on seeded dev stack (alice books sofia trial once, second attempt 400)

---

### S-T-04 — Trial Credit Dashboard

| Axis | Value |
|---|---|
| **Wireframe refs** | `wireframes/trial_credit_dashboard/code.html:1` — credits card (star, `credits` large number, Available to use right now, Next credit date, CTA Find a Tutor search), How Trials Work (20 Minutes, Meet & Greet 2 cards), Recommended for Trials (2 tutors + Book Trial), History (list of trial bookings) |
| **REQUIREMENTS_MASTER.md** | `12.5` trial credit dashboard + `13.1` 1 trial credit/mo model (credits=1 initially, reset 30d after granted_at when 0 `teacher.go:242`). |
| **Backend contract** | `GET /teachers/trial-credits/dashboard` `main.go:651` `teacher.go:124` → `services/teacher.go:249` `GetTrialCreditDashboard(userID) → TrialCreditDashboard{credits, grantedAt, updatedAt, nextGrantAt, history, totalUsed, totalTrial}`<br>`GET /teachers/trial-credits` `:650` `teacher.go:114` → `services/teacher.go:232` `GetTrialCredit` (auto-init credits=1, auto-refill after 30d).<br>History query `SELECT tutor_bookings WHERE student_user_id=$1 AND is_trial=true ORDER BY created_at DESC LIMIT 50` `:259`. `nextGrantAt = grantedAt + 30d` when credits==0 `:256`.<br>Shared via raw fetch in `TrialCredits.tsx:17` `GET /api/v1/teachers/trial-credits/dashboard` + `teacherAPI.browse` for Recommended. |
| **Frontend route** | `/trial-credits` `App.tsx:250` `TrialCredits.tsx:1` (credits `data.credits`, nextGrant `data.nextGrantAt`, history `data.history`, Recommended `tutors` `:71-79`). |
| **Mobile route** | `MarketplaceTab/TrialCredits` `MainTabs.tsx:198` `TrialCreditsScreen` title `Trial Credits`. |
| **Depends on** | S-T-03 (history populated after booking) |

**API contract:**

```
GET /api/v1/teachers/trial-credits/dashboard
200 { "dashboard": { "credits": 1, "grantedAt":"2026-09-03T...", "updatedAt":"...", "nextGrantAt": null, "history": [{id,teacherUserId,studentUserId,startTime,endTime,status,isTrial,note,createdAt}], "totalUsed":0, "totalTrial":0 } }
GET /api/v1/teachers/trial-credits
200 { "trialCredits": { "userId":"alice", "credits":1, "updatedAt":"...", "grantedAt":"..." } }
Model: TrialCreditDashboard models.go:1109; TrialCredit models.go:1102; TutorBooking models.go:1131
```

**Gherkin — S-T-04:**

```gherkin
@S-T-04 @marketplace @trial-credits @wireframe-trial_credit_dashboard @credits
Feature: Trial credit dashboard (1 credit/mo)

  Background:
    Given alice seeded credits=1, 0 trial bookings (dev_seed.go:128)

  Scenario: Web dashboard renders credits card
    When I open "/trial-credits" (App.tsx:250 TrialCredits.tsx:36)
    Then I see card "Trial Credits" with large number "1" (or current credits), "Available to use right now"
    And "Next credit: <date>" only when credits==0 (nextGrantAt = grantedAt+30d)
    And CTA "Find a Tutor" navigates to "/tutors"
    And section "How Trials Work" shows "20 Minutes" and "Meet & Greet" cards
    And section "Recommended for Trials" shows up to 2 tutors (GET /teachers/browse limit 2) with Book Trial
    And section "History" shows "No trial bookings yet." when empty

  Scenario: History after booking
    Given alice booked sofia trial (S-T-03)
    When I re-open "/trial-credits"
    Then dashboard credits=0, nextGrantAt visible, History lists 1 trial booking with date·Trial·status

  Scenario: Credits refill after 30 days
    Given alice credits=0 and grantedAt = 31 days ago
    When I GET /teachers/trial-credits
    Then credits auto-refills to 1 (teacher.go:242-244)

  Scenario: Mobile parity
    When I open TrialCredits on mobile (MainTabs.tsx:198)
    Then same credits card, How Trials Work, Recommended, History render natively
```

**QA testRefs:**

| Suite | File | Locator |
|---|---|---|
| `e2e` | `e2e/tests/tutor-trial-credits.spec.ts:dashboard` | `goto('/trial-credits')` → `expect(page.getByText('Trial Credits')).toBeVisible()` + `getByText('Available to use right now')` + `getByRole('button',{name:'Find a Tutor'})` → click → `/tutors` + after booking `expect(page.getByText('History'))` contains date |
| `vitest` | `frontend/src/__tests__/TrialCredits.test.tsx` | mock dashboard `{credits:1,history:[]}` → `getByText('Available to use right now')` + `getByText('How Trials Work')` + `getByText('Recommended for Trials')`; second test `credits:0` → `getByText('Next credit')` |
| `jest` | `mobile/__tests__/TrialCreditsScreen.test.tsx` | same card + navigation |
| `backend` | existing `teacher_test.go: trialCredit` | `GetTrialCredit` init 1, dashboard history query, nextGrant calc |

**DoD — S-T-04:**
- Screenshot vs `trial_credit_dashboard/code.html` (credits star card + How Trials Work + Recommended + History)
- `WIREFRAME_TRACE.md:88` `trial_credit_dashboard` flips GAP → PASS with note `S-T-04 TrialCredits.tsx:36 + TrialCreditsScreen + GET /teachers/trial-credits/dashboard:651`
- `e2e/tutor-trial-credits.spec.ts` green

---

### S-T-05 — Teacher Dashboard (Availability + Students + Checklist)

| Axis | Value |
|---|---|
| **Wireframe refs** | `wireframes/teacher_dashboard/code.html:1` — Welcome back! header + Status Accepting New Students, Earnings Overview (3 cols Total Earned / Pending Payout / Platform Fee %), Premium Program card, Availability (3 slots + Edit Schedule, Upcoming Sessions 2), Recent Students (3 + View All), Profile Completion pct bar + checklist Basic Bio/Video/Certs |
| **REQUIREMENTS_MASTER.md** | `12.4` Teacher dashboard (earnings, availability, students, profile checklist). `REQUIREMENTS.md:7b` T02. |
| **Backend contract** | `GET /teachers/dashboard` `main.go:649` `teacher.go:233` → `services/teacher.go:635` `GetDashboard(teacherUserID) → TeacherDashboard{application, checklist, earnings, students, totalStudents, upcomingBookings, upcomingAvailability}`<br>Checklist `teacher.go:638-683`: hasBio (≥10 chars), hasLanguages, hasExpertise, hasRate>0, hasVideo, hasCert, hasVerifiedCert, isApproved; pct = done*100/7.<br>Earnings `teacher.go:688-731`: loops `tutor_bookings` non-trial, gross=rateCents*hours, net=gross*(100-feePct)/100 (fee 10 if verified else 15 `teacher.go:692-697`), counts pending/completed/cancelled.<br>Upcoming `teacher.go:733-749` start > NOW pending/confirmed LIMIT 20; students `teacher.go:751-775` group by studentUserID.<br>Auth: caller must be teacher (has application, else 404). Shared raw fetch `TeacherDashboard.tsx:12` `GET /teachers/dashboard`. |
| **Frontend route** | `/teacher/dashboard` `App.tsx:251` `TeacherDashboard.tsx:1` (pct `dash.checklist.completionPct`, earnings `dash.earnings`, availability `dash.upcomingAvailability`, students `dash.students`, upcoming `dash.upcoming`). |
| **Mobile route** | `MarketplaceTab/TeacherDashboard` `MainTabs.tsx:199` `TeacherDashboardScreen` title `Dashboard`. |
| **Depends on** | `become_a_teacher` approved status (sofia tutor approved seed); availability `POST /teachers/availability` could be added but not required for read |

**API contract:**

```
GET /api/v1/teachers/dashboard
200 { "dashboard": {
  "application": {id,userId,bio,languages,expertise,rateCents,videoUrl,status,certificates},
  "checklist": {hasBio,hasLanguages,hasExpertise,hasRate,hasVideo,hasCertificate,hasVerifiedCert,isApproved,complete,completionPct: 71},
  "earnings": {totalGrossCents,totalNetCents,pendingGrossCents,pendingNetCents,completedCount,pendingCount,cancelledCount,totalBookings,platformFeePct:10,ratingAvg,ratingCount},
  "students": [{studentUserId,displayName,avatarColor,bookingsCount,completedCount,lastBookingAt}],
  "totalStudents": 2,
  "upcomingBookings": [{id,teacherUserId,studentUserId,startTime,endTime,status,isTrial,note}],
  "upcomingAvailability": [{id,teacherUserId,startTime,endTime}]
}}
404 { "error": "Teacher application not found" }
Model: TeacherDashboard models.go:1243; TeacherChecklist 1166; TeacherEarnings 1179; TutorBooking 1131; TutorAvailability 1123
```

**Gherkin — S-T-05:**

```gherkin
@S-T-05 @marketplace @teacher-dashboard @wireframe-teacher_dashboard
Feature: Teacher dashboard (checklist + earnings + availability + students)

  Background:
    Given I am authenticated as sofia.tutor@chorus.test (approved, verified cert, rate 2500 — dev_seed.go:79)
    And teacher has 4 availability slots, 0-2 bookings, 2 students (alice,bob) from seed/bookings

  Scenario: Web dashboard renders wireframe sections
    When I open "/teacher/dashboard" (App.tsx:251 TeacherDashboard.tsx:30)
    Then I see h2 "Teacher Dashboard" and "Welcome back!" card with Status "Accepting New Students"
    And Earnings Overview shows Total Earned, Pending Payout, Platform Fee (10% when verified per payout.go:23) from GET /teachers/dashboard
    And Premium Program card "Manage Premium Settings" links to "/teacher/payouts"
    And Availability section shows up to 3 upcomingAvailability slots with date+time (or "No availability set.")
    And Recent Students shows up to 3 students (or "No students yet.")
    And Profile Completion shows "Profile Completion — {pct}%" bar width pct% and checklist Basic Bio/Video/Certs done states (TeacherDashboard.tsx:64-73)
    And footer links Payouts + Browse tutors visible

  Scenario: Not a teacher guard
    Given I am alice (no teacher application)
    When I GET /teachers/dashboard
    Then 404 "Teacher application not found" and web shows error with "Apply" link (TeacherDashboard.tsx:30)

  Scenario: Checklist math
    Given sofia has bio, languages, expertise, rate, verified cert, approved but no video (dev_seed.go:79 video empty)
    Then completionPct = 6/7 ≈85 (TeacherDashboard.tsx:23 checklist.completionPct) and HasVideo false

  Scenario: Mobile parity
    When I open TeacherDashboard on mobile (MainTabs.tsx:199)
    Then same Welcome, Earnings, Premium, Availability, Students, Profile Completion render natively
```

**QA testRefs:**

| Suite | File | Locator |
|---|---|---|
| `e2e` | `e2e/tests/tutor-dashboard.spec.ts:teacher` | `loginAs('sofia.tutor')` → `goto('/teacher/dashboard')` → `expect(page.getByText('Teacher Dashboard')).toBeVisible()` + `getByText('Welcome back')` + `getByText('Earnings Overview')` + `getByText('Availability')` + `getByText('Recent Students')` + `getByText('Profile Completion')` + change user to alice → `expect(page.getByText('Apply'))` |
| `vitest` | `frontend/src/__tests__/TeacherDashboard.test.tsx` | mock `GET /teachers/dashboard` → dashboard pct 85 → `getByText('Profile Completion — 85%')` + `getByText('$25')` / earnings cards |
| `jest` | `mobile/__tests__/TeacherDashboardScreen.test.tsx` | same sections + navigation to Payouts |
| `backend` | existing `teacher_test.go: dashboard` | `GetDashboard` checklist 7 → pct, earnings gross/net fee 10 when verified, availability filtered future |

**DoD — S-T-05:**
- Screenshots web + AVD of dashboard vs `teacher_dashboard/code.html` (Welcome + Earnings 3 cols + Premium + Availability + Students + Profile Completion bar)
- `WIREFRAME_TRACE.md:83` `teacher_dashboard` flips GAP → PASS, `80` `student_management_progress_teacher` closed via same dashboard students section
- `e2e/tutor-dashboard.spec.ts` green for both sofia (PASS) and alice (404 guard)

---

### S-T-06 — Payout Settings & History + Teacher Earnings Overview (Combined)

| Axis | Value |
|---|---|
| **Wireframe refs** | `wireframes/payout_settings_history/code.html:1` — Total Lifetime Earnings card (Available for payout, Withdraw Funds →), Payout Methods (paypal/bank list, Add Method), This Month's Breakdown (Gross, Platform Fee %, Net Income), Withdraw input, Performance Insight (12% more, Hours Taught 42h, Active Students N), Payout History list; `teacher_earnings_overview/code.html:1` — earnings breakdown alias (Recent Transactions, HoursTaught, ActiveStudents) — same `Payouts` screen `Payouts.tsx:1` merges both |
| **REQUIREMENTS_MASTER.md** | `12.7` Payments/payouts: Paypal payouts, platform fee 10/15%, payout history/settings. `12.4` earnings portion. `13.1` Reduced marketplace fee for premium (10% when verified or student premium `payout.go:84-86`). Narrative `REQUIREMENTS.md:139` P1 free/premium fee. |
| **Backend contract** | `GET /teachers/payouts/overview` `main.go:638` `payout.go:22` → `services/payout.go:107` `GetOverview(teacherID)` → `PayoutOverview{availableCents, pendingCents, pendingGrossCents, totalGrossCents, totalNetCents, lifetimeGross, lifetimeNet, totalPaidCents, platformFeePct, completedCount, pendingCount, cancelledCount, totalBookings, nextPayoutDate, hoursTaught, activeStudents, recentTransactions[5]}` (fee 10 if verified else 15 `payout.go:37`, paidCents already `teacher_payouts` pending/completed `payout.go:118`, nextPayout 15th `payout.go:130`)<br>`GET /teachers/payouts/methods` `:641` `payout.go:35` → `payout.go:199` `ListMethods`<br>`POST /teachers/payouts/methods` `:642` body `{type:paypal|bank, label, details, isDefault?}` → `payout.go:219` `AddMethod` (paypal must contain @ `:231`)<br>`DELETE /teachers/payouts/methods/:methodId` `:643` → `payout.go:259`<br>`PUT /teachers/payouts/methods/:methodId/default` `:644` → `payout.go:271`<br>`GET /teachers/payouts/history` `:639` `payout.go:82` → `payout.go:293` `ListPayouts` → `{payouts, total, hasMore}`<br>`POST /teachers/payouts/withdraw` `:640` `payout.go:94` → `payout.go:334` `RequestPayout({amountCents, methodId?})` validates `>0, >=100, <=availableCents` `:335-346`, resolves method (default or given) `:349-366`, creates `teacher_payouts` row status pending/processing, paypal batch when enabled `payout.go:379`.<br>Shared `payouts.overview/methods/addMethod/removeMethod/history/withdraw` `api.ts:926-953`. |
| **Frontend route** | `/teacher/payouts` `App.tsx:252` `Payouts.tsx:1` (overview `payoutsAPI.overview()`, methods `payoutsAPI.methods()`, history `payoutsAPI.history({limit:10})` `Payouts.tsx:20-22`; add method `Payouts.tsx:28`; withdraw `Payouts.tsx:31`). |
| **Mobile route** | `MarketplaceTab/Payouts` `MainTabs.tsx:200` `PayoutsScreen` title `Payouts`. |
| **Depends on** | S-T-05 (earnings card links to Payouts), S-T-06 can land parallel to S-T-05 but after S-T-01..04 |

**API contracts:**

```
GET /api/v1/teachers/payouts/overview
200 { "overview": { "availableCents": 0, "pendingCents": 0, "totalGrossCents": 0, "totalNetCents": 0, "lifetimeGross": 0, "lifetimeNet": 0, "totalPaidCents":0, "platformFeePct":10, "completedCount":0, "pendingCount":0, "cancelledCount":0, "totalBookings":0, "nextPayoutDate":"2026-09-15T00:00:00Z", "hoursTaught":0, "activeStudents":0, "recentTransactions":[] } }

GET /api/v1/teachers/payouts/methods
200 { "methods": [{id,teacherUserId,type, label, details, isDefault, createdAt}] }

POST /api/v1/teachers/payouts/methods
{ "type":"paypal", "label":"PayPal", "details":"sofia@chorus.test" }
201 { "method": {...} }
400 { "error": "paypal email invalid" } | "label required"

GET /api/v1/teachers/payouts/history?limit=10&offset=0
200 { "payouts": [{id,teacherUserId,amountCents,feeCents,grossCents,methodId,destination,status,reference,paypalBatchId,createdAt}], "total":0, "hasMore":false }

POST /api/v1/teachers/payouts/withdraw
{ "amountCents": 1000, "methodId": "optional-uuid" }
201 { "payout": {id, amountCents, feeCents, grossCents, destination, status:"pending", reference:"TRX-..."} }
400 { "error": "insufficient available balance: 0 available" } | "minimum payout is $1.00"
```

**Gherkin — S-T-06:**

```gherkin
@S-T-06 @marketplace @payouts @earnings @wireframe-payout_settings_history @wireframe-teacher_earnings_overview
Feature: Payout settings & history + Teacher earnings overview

  Background:
    Given I am sofia.tutor (approved, verified → fee 10%)

  Scenario: Web payouts renders wireframe sections
    When I open "/teacher/payouts" (App.tsx:252 Payouts.tsx:39)
    Then I see "Payout Settings & History" + "Manage your connected bank accounts..."
    And I see card "Total Lifetime Earnings" with available "${availableCents/100}" and "Withdraw Funds →" (Payouts.tsx:47-49)
    And I see section "Payout Methods" (or "No methods yet. Add PayPal or bank.") with list of methods type·label·Default and Remove
    And I see add-method row with select paypal/bank + label + details (Email/IBAN) + Add button (Payouts.tsx:59-63)
    And I see "This Month's Breakdown" with Gross, Platform Fee {feePct}%, Net Income (Payouts.tsx:67-71)
    And I see Withdraw section with amount input and Withdraw button + Available/Pending line (Payouts.tsx:75-81)
    And I see Performance Insight "12% more" + Hours Taught + Active Students (Payouts.tsx:83-87)
    And I see "Payout History" list (empty "No payouts yet." or rows ${amount}·{status}·{date} reference)

  Scenario: Add payout method (PayPal)
    When I select type "paypal", label "PayPal", details "sofia@chorus.test" and tap Add
    Then POST /teachers/payouts/methods succeeds 201 and methods list shows "paypal · PayPal · Default" (first method auto-default payout.go:236)

  Scenario: Add method validation
    When I add paypal with details missing "@"
    Then 400 "paypal email invalid" (payout.go:231) and error banner shows (Payouts.tsx:42)

  Scenario: Withdraw guard
    Given overview availableCents=0
    When I withdraw 1000 cents
    Then 400 "insufficient available balance" (payout.go:345) and error banner

  Scenario: Withdraw success (when earnings >0)
    Given completed bookings give availableCents >= 100 (GetOverview computes net - paidCents payout.go:119)
    And a default payout method exists
    When I withdraw 100 cents (minimum $1.00 payout.go:339)
    Then POST /teachers/payouts/withdraw 201 and history shows new payout status pending/processing

  Scenario: Earnings overview alias (teacher_earnings_overview)
    When I view This Month's Breakdown + RecentTransactions (Payouts overview recentTransactions 5 — payout.go:139)
    Then it satisfies wireframe/teacher_earnings_overview code.html (earnings = same overview card)

  Scenario: Mobile parity
    When I open Payouts on mobile (MainTabs.tsx:200)
    Then same Lifetime Earnings, Methods, Breakdown, Withdraw, Performance Insight, History render natively
```

**QA testRefs:**

| Suite | File | Locator |
|---|---|---|
| `e2e` | `e2e/tests/tutor-payouts.spec.ts:settings` | `loginAs('sofia')` → `goto('/teacher/payouts')` → `expect(page.getByText('Payout Settings & History')).toBeVisible()` + `getByText('Total Lifetime Earnings')` + `getByText('Payout Methods')` + method add: `selectOption('paypal')` + `fill(label PayPal)` + `fill(details sofia@chorus.test)` → `click(Add)` → `expect(page.getByText('paypal · PayPal'))` + invalid `@` → `expect(page.getByText('paypal email invalid'))` + withdraw 1000 when 0 → `expect(page.getByText('insufficient available'))` |
| `vitest` | `frontend/src/__tests__/Payouts.test.tsx` | mock `payoutsAPI.overview/methods/history` → same sections + `fireEvent.click(Add)` params check + error banner for invalid paypal |
| `jest` | `mobile/__tests__/PayoutsScreen.test.tsx` | same methods/withdraw/history render + remove method |
| `backend` | existing `payout_test.go` | `GetOverview` fee 10 verified vs 15, `AddMethod` @ validation, `RequestPayout` minimum 100, insufficient balance, nextPayout 15th |

**DoD — S-T-06:**
- Screenshots web + AVD of Payouts vs `payout_settings_history/code.html` + `teacher_earnings_overview/code.html` (Lifetime Earnings + Methods + Breakdown + Withdraw + History)
- `WIREFRAME_TRACE.md:65` `payout_settings_history` + `84` `teacher_earnings_overview` flip GAP → PASS with note `S-T-06 Payouts.tsx:39 + PayoutsScreen + GET /teachers/payouts/overview:638 dev_seed.go:79`
- `e2e/tutor-payouts.spec.ts` green (valid add + invalid @ + withdraw guard)

---

## 3. Cross-Cutting Contracts (applies to all S-T-*)

### Auth + route guard

All `main.go:638-668` marketplace routes are **protected** `r.Group("/api/v1")` with `middleware.AuthMiddleware` `main.go:495-496`. Unauthenticated → 401 → `onAuthFailure` redirect `/login` (`packages/shared/src/api.ts:122-151` refresh interceptor + `frontend/src/services/api.ts:33` + `mobile/src/services/api.ts` storage adapter). `App.tsx:247-253` wraps every tutor route `isAuthenticated ? <Page/> : <Navigate to="/login"/>`. Mobile `MainTabs` is inside authenticated stack too (outside auth stack Landing/Login).

### Navigation contract

| Surface | Entry | Navigation |
|---|---|---|
| Web | `BrowseTutors.tsx:32` `Become a teacher` → `/become-teacher` + footer links `Trial credits / Teacher dashboard / Payouts` `BrowseTutors.tsx:62-68` | Chats/Learn via `BottomNav` + direct links; Profile → Tutor via `/tutors/:id` |
| Mobile | `MainTabs.tsx:180-202` `MarketplaceTab` 4th tab `Tutors` label — always reachable from tab bar `TABS:207` | `BrowseTutors` → card `navigate('TutorProfile',{userId})` → `ConfirmBooking` → `TrialCredits` / `TeacherDashboard` / `Payouts` all inside same `MarketplaceStack` |

`verify-wireframe-parity.sh` parity gate must assert `App.tsx:247` `/tutors` + `MainTabs.tsx:183` `BrowseTutors` both exist and are reachable without auth-crash (redirect to login is correct when unauthenticated, but authenticated alice/sofia must land).

### Data contracts (traceable models)

| Model | File | Fields used in slices |
|---|---|---|
| `TutorProfile` | `models.go:1054` | id,userId,displayName,bio,languages,expertise,rateCents,videoUrl,status,verified,ratingAvg,ratingCount,avatarColor,certificates |
| `TutorReview` | `models.go:1092` | id,teacherUserId,studentUserId,rating,comment,studentName |
| `TutorAvailability` | `models.go:1123` | id,teacherUserId,startTime,endTime |
| `TutorBooking` | `models.go:1131` | id,teacherUserId,studentUserId,startTime,endTime,status,isTrial,note,reviewNotes,confirmedAt,completedAt |
| `TrialCredit` | `models.go:1102` | userId,credits,updatedAt,grantedAt |
| `TrialCreditDashboard` | `models.go:1109` | credits,grantedAt,updatedAt,nextGrantAt,history,totalUsed,totalTrial |
| `TeacherApplication` | `models.go:1031` | id,userId,bio,languages,expertise,rateCents,videoUrl,status,certificates |
| `TeacherChecklist` | `models.go:1166` | hasBio,hasLanguages,hasExpertise,hasRate,hasVideo,hasCertificate,hasVerifiedCert,isApproved,complete,completionPct |
| `TeacherEarnings` | `models.go:1179` | totalGrossCents,totalNetCents,pendingGrossCents,pendingNetCents,completedCount,pendingCount,cancelledCount,totalBookings,platformFeePct,ratingAvg,ratingCount |
| `TeacherDashboard` | `models.go:1243` | application,checklist,earnings,students,totalStudents,upcomingBookings,upcomingAvailability |
| `PayoutOverview` | `payout.go:53` models | availableCents,pendingCents,pendingGrossCents,totalGrossCents,totalNetCents,lifetimeGross,lifetimeNet,totalPaidCents,platformFeePct,completedCount,pendingCount,cancelledCount,totalBookings,nextPayoutDate,hoursTaught,activeStudents,recentTransactions |
| `PayoutMethod` | `payout.go:5` | id,teacherUserId,type,label,details,isDefault,createdAt |
| `PayoutRecord` | `payout.go:22` | id,teacherUserId,amountCents,feeCents,grossCents,methodId,destination,status,reference,paypalBatchId,createdAt |

### Seed & test accounts

`backend/internal/services/dev_seed.go:17` `DevPassword=ChorusDev123!`, `:18` `alice.dev@chorus.test`, `:19` `bob.dev@chorus.test`, `:20` `sofia.tutor@chorus.test` (also `TEST_ACCOUNTS.md`). Every e2e must `go run ./cmd/server --seed-dev` before run (or rely on `docs/TEST_SPEC.md` acceptance fixtures). Flaky seed → `utils/db_isolation` truncation.

### File refs index

| Artifact | File ref |
|---|---|
| Canonical wireframes | `wireframes/browse_tutors/code.html:1` (131 header → 312), `find_a_trial_tutor/code.html:1` (filtered variant), `tutor_profile_sofia/code.html:1` (Sofia hero `175`), `confirm_trial_booking/code.html:1` (header `48`), `trial_credit_dashboard/code.html:1`, `teacher_dashboard/code.html:1`, `payout_settings_history/code.html:1`, `teacher_earnings_overview/code.html:1` |
| Master backlog | `REQUIREMENTS_MASTER.md:144-152` `12.1-12.7` + `13.1` Credits & Access, `REQUIREMENTS.md:139` Phase 5 T01-T04 premium/marketplace |
| Backend routes | `backend/cmd/server/main.go:638-668` all marketplace + payouts, `handlers/teacher.go:57`/`96`/`286`/`124`/`233`, `handlers/payout.go:22-94` |
| Services truth | `services/teacher.go:132` BrowseTutors / `:105` GetTutorProfile / `:249` Dashboard credits / `:416` CreateBooking / `:635` GetDashboard / `:284` ListReviews / `:351` GetAvailability / `:232` GetTrialCredit, `services/payout.go:107` GetOverview / `:219` AddMethod / `:334` RequestPayout / `:293` ListPayouts |
| Seed | `services/dev_seed.go:79` Sofia tutor, `:31` SeedDevData, `:99` 4 availability slots, `:110` 2 reviews, `:128` trial credit |
| Shared client | `packages/shared/src/api.ts:1107-1187` teacher.* + `:926-953` payouts.*, `packages/shared/src/types.ts` TutorProfile/TrialCreditDashboard/PayoutOverview |
| Frontend routes | `frontend/src/App.tsx:247` `/tutors` / `:248` `/tutors/:id` / `:249` `/tutors/:id/confirm` / `:250` `/trial-credits` / `:251` `/teacher/dashboard` / `:252` `/teacher/payouts` + pages `BrowseTutors.tsx:1` `TutorProfile.tsx:1` `ConfirmBooking.tsx:1` `TrialCredits.tsx:1` `TeacherDashboard.tsx:1` `Payouts.tsx:1` |
| Mobile routes | `mobile/src/components/MainTabs.tsx:180-202` MarketplaceTab 6 screens, `MarketplaceStackParamList:62-70`, `TABS:204-209` 4th tab Tutors |
| Trace | `docs/WIREFRAME_TRACE.md:28` marketplace summary, `docs/CREWAI_GAP_CLOSURE_PLAN.md:71-85` Cluster T S-T-01..06, `docs/REQUIREMENTS_SLICE_HOME_V2.md:373` pattern for BA spec |
| Home precedent | `docs/REQUIREMENTS_SLICE_HOME_V2.md:1` (slice template, Gherkin + API contract + DoD) |

---

## 4. Verification Checklist (attach to PR — `docs/CREWAI_GAP_CLOSURE_PLAN.md:158`)

### BA sign-off criteria — EXPLICIT (slice NOT DONE until all checked, `crew/phase_status.json:58` stays PENDING otherwise)

- [ ] **Wireframe visual match (6 slices):** Device screenshot (Android AVD or iOS sim **and** web `npm run dev` snapshot) side-by-side with wireframe PNGs for: `browse_tutors/code.html:131` (search+filters+Featured+Available Now), `tutor_profile_sofia/code.html:175` (Sofia hero+stats+About+reviews+pricing+calendar), `confirm_trial_booking/code.html:48` (Great choice + Date/Time + Payment Summary $0.00 + sticky CTA), `trial_credit_dashboard/code.html` (credits star card + How Trials Work + Recommended + History), `teacher_dashboard/code.html` (Welcome+Earnings+Availability+Students+Profile Completion), `payout_settings_history/code.html` + `teacher_earnings_overview/code.html` (Lifetime Earnings + Methods + Breakdown + Withdraw + History). BA initials + date.
- [ ] **Navigation reachability:** On AVD, `MarketplaceTab` 4th tab `Tutors` (`MainTabs.tsx:207`) visible; Browse → tap Sofia card → Profile → Book Trial → Confirm → confirm 201 → TrialCredits reachable; TrialCredits → Find a Tutor loops back; Teacher Dashboard reachable from Profile/Browse footer and from Confirm success; Payouts reachable from Dashboard Premium/earnings card and from Browse footer. No dead tap. Unauthenticated Browse/Profile/Confirm redirect to `/login` (web) / auth stack (mobile) without crash.
- [ ] **Backend contract cited correctly:** Every Gherkin step that hits API cites `main.go:638-668` line + service line; `curl` proof: `GET /teachers/browse` 200, `GET /teachers/:id` 200 for sofia uuid, `POST /teachers/:id/book isTrial` 201 then credits 0, `GET /teachers/trial-credits/dashboard` 200, `GET /teachers/dashboard` 200 for sofia, `GET /teachers/payouts/overview` 200 — all on seeded dev stack.
- [ ] **Both surfaces built green:**
  - `cd backend && go vet ./...` exit 0
  - `cd backend && go test ./...` exit 0 (incl. `teacher_test.go` browse/profile/booking/trial-credit + `payout_test.go` overview/methods/withdraw)
  - `cd frontend && npm run build` (tsc && vite build) exit 0 + `grep -R "alice.dev" frontend/dist` 0 (NO_LEAK)
  - `cd frontend && npm test` (vitest) exit 0 (new `BrowseTutors.test.tsx`, `TutorProfile.test.tsx`, `ConfirmBooking.test.tsx`, `TrialCredits.test.tsx`, `TeacherDashboard.test.tsx`, `Payouts.test.tsx` green)
  - `cd mobile && npx tsc --noEmit` exit 0
  - `cd mobile && npm test` (jest) exit 0 (matching 6 mobile screen tests green)
- [ ] **Automation green (TDD red→green proven):**
  - `e2e/tests/tutor-browse.spec.ts` + `tutor-profile.spec.ts:sofia` + `tutor-booking.spec.ts:confirm` + `tutor-trial-credits.spec.ts:dashboard` + `tutor-dashboard.spec.ts:teacher` + `tutor-payouts.spec.ts:settings` pass on dev stack (`e2e/global-setup.ts:19` `BACKEND_HEALTH=http://localhost:8080/health` `go run --seed-dev` fixtures `alice/bob/sofia`)
  - Existing `frontend vitest` + `mobile jest` show exactly **+N** new pass (6 frontend + 6 mobile), 0 fail
  - `backend` tests show `dev_seed.go:31` SeedDevData idempotent (counts stable on second run)
- [ ] **Device-parity gate (not just `npm test`):**
  - `.\start-android.ps1` boots AVD `emulator-5554 device` + `adb shell getprop sys.boot_completed` == `1`
  - `curl -fsS http://localhost:8080/health | jq .commit` == `git rev-parse HEAD` (proves `backend/internal/observability/health.go:39` `CHORUS_BUILD_COMMIT` is fresh — `docs/TDD_RESCUE_SPEC.md:52` `S-SMOKE-02`)
  - On AVD, Marketplace flows render without crash; keyboard/tap on search triggers browse; Profile Book Trial navigates; Confirm sticky CTA tappable; TrialCredits Find a Tutor works; Teacher Dashboard premium card visible; Payouts methods add/remove works
  - `verify-wireframe-parity.sh` (or `docs/WIREFRAME_TRACE.md:60` audit) row for `browse_tutors`, `tutor_profile_sofia`, `confirm_trial_booking`, `trial_credit_dashboard`, `teacher_dashboard`, `teacher_earnings_overview`, `payout_settings_history` flipped `GAP → PASS`
- [ ] **Mobile parity asserted:** Each locator in S-T-01..06 has both a `page.getBy*` (Playwright) and a `getByText`/`getByTestId` (jest/RNTL) — none is web-only. PR description lists `frontend/src/pages/BrowseTutors.tsx:1` etc. + `mobile/src/screens/BrowseTutorsScreen.tsx:1` etc. lines touched.
- [ ] **`docs/WIREFRAME_TRACE.md:28` updated:** Marketplace 1/13 → 8/13 PASS (6 GAP closed, `become_a_teacher` was already PASS, `find_a_trial_tutor` as filtered variant note), remaining 5 still GAP (group/srs etc. per `docs/WIREFRAME_TRACE.md:40`) + date + BA sign.
- [ ] **No secrets/outline leaks:** `.env*` untouched, no `WAITLIST_ADMIN_EMAILS` or `SMTP_PASSWORD` printed, `grep -R "sk-" frontend/dist mobile/dist` 0, `grep -R "ChorusDev123!" frontend/dist` 0.
- [ ] **Gap sign-off Sheet:** BA signs `docs/GAP_SIGNOFF.md` (add rows `S-T-01 | browse_tutors/code.html:131 | BrowseTutors.tsx:31 BrowseTutorsScreen MainTabs.tsx:183 GET /teachers/browse:648 | BA initials | date | commit SHA` … `S-T-06 | payout_settings_history/code.html:1 teacher_earnings_overview | Payouts.tsx:1 PayoutsScreen MainTabs.tsx:200 GET /teachers/payouts/overview:638 | BA | date | commit`) — only then `crew/state.py:97` `phase_complete()` may flip slice DONE (mirrors `docs/CREWAI_GAP_CLOSURE_PLAN.md:50` BA gate).

**Rejection rule:** If any box unchecked (e.g., device does not boot, commit mismatch, stale GAP row, e2e not run on real dev stack with `dev_seed.go:79` Sofia, mobile not touched, fee 10% vs 15% mismatch for verified), BA marks `CHANGES-REQUIRED` and slice returns to QA (new failing test added) per loop `docs/CREWAI_GAP_CLOSURE_PLAN.md:39` outer cycle.

---

## 5. Sequencing & Dependencies

```
S-T-01 (Browse + Find trial tutor) → S-T-02 (Profile Sofia) → S-T-03 (Confirm booking) → S-T-04 (Trial credits dashboard)
                                                        ↘ S-T-05 (Teacher dashboard) → S-T-06 (Payouts/earnings)
```

- S-T-01 is first — proves marketplace navigation (`MarketplaceTab` 4th tab) and that Sofia seed is browseable; filters are hardened here.
- S-T-02 depends on 01 (entry via Browse card). Sofia is canonical per `WIREFRAME_TRACE.md:91`; no other tutor profile needed for BA sign-off.
- S-T-03 depends on 02 (Confirm is reached from Profile Book Trial). Validates `isTrial` trial-credit consumption.
- S-T-04 depends on 03 (History populated after booking, credits 0 state). Can land same PR as 03.
- S-T-05 requires approved tutor auth (sofia.tutor login) and is independent of 03/04 but after 01-02 for narrative; can run parallel to 04.
- S-T-06 depends on 05 (earnings card links to Payouts, `GetOverview` fee logic shared with `GetDashboard` earnings). Add-method validation is tested here.
- Recommended: **one PR for S-T-01..04** (student marketplace journey Browse→Profile→Confirm→Credits) + **second PR for S-T-05..06** (teacher dashboard+payouts). Both PRs must keep device green; splitting 6 singletons would still require `MarketplaceTab` present from first PR. Supervisor `crew/autonomous_flow.py:200` may schedule `slice_marketplace` as one BA spec → 2 QA red batches → 2 impl greens.

No backend ordering — all `main.go:638-668` routes already exist. No dependency on S-HOME-01..04 (Home v2) beyond `MarketplaceTab` being visible alongside Chats/Learn/Profile tabs.

---

## 6. Out of Scope (explicitly NOT in S-T-01..06)

- `become_a_teacher` flow (`teacher.go:54` `Apply` + `BecomeTeacher.tsx:1` / `BecomeTeacherScreen` `MainTabs.tsx:193`) — already PASS per `WIREFRAME_TRACE.md:12`, hardening is separate P1 polish if needed.
- Post-lesson SRS push `POST /teachers/srs/push :660` + sandbox `GET /teachers/srs/sandbox/:studentId :663` (wireframes `custom_activity_builder_pusher`, `teacher_student_learning_chat`, `live_lesson_shared_sandbox`) — teacher pushes drills to student SRS; service `teacher_srs.go:20` ready but UI deferred to follow-up slice.
- `lesson_review_notes_student` `PUT /teachers/bookings/:id/review-notes :659` student reader — backend ready, UI deferred (`student_insights_progress_dashboard` etc. `WIREFRAME_TRACE.md:79` P1).
- Group/Community: `group_study_hub`, `group_session_management`, `live_group_study_session`, `host_a_study_session`, `learning_community_feed`, `trending_community_hub` — no backend (`WIREFRAME_TRACE.md:40` + `REQUIREMENTS_MASTER.md:6` backlog deferred), no S-T coverage.
- Pronunciation review `teacher_pronunciation_review_dashboard` — no UI.
- `trust_safety_*` surfaces — Trust & Safety cluster `docs/WIREFRAME_TRACE.md:74-76` P2.
- New backend work — none. If a contract drifts (e.g., fee pct changes, trial credit interval), file an amendment to this spec and mirror service line citation; do not code without BA update.

---

## 7. Traceability Summary

| Slice | Wireframe `code.html:line` | Master FR `REQUIREMENTS_MASTER.md:5.1` | Narrative FR `REQUIREMENTS.md` | Backend route `main.go:line` → service `teacher.go/payout.go:line` | Frontend `App.tsx:line` → page `*.tsx:line` | Mobile `MainTabs.tsx:line` → screen |
|---|---|---|---|---|---|---|
| S-T-01 | `browse_tutors:131-155` filters + `156-287` Featured/Available Now; `find_a_trial_tutor:1` filtered variant | `12.2` Browse tutors / find trial tutor | P5 T01 Tutor Search | `GET /teachers/browse:648` → `BrowseTutors:132` | `App.tsx:247` `/tutors` → `BrowseTutors.tsx:18` `teacherAPI.browse` | `MainTabs.tsx:183` `BrowseTutors` 4th tab `Tutors:207` |
| S-T-02 | `tutor_profile_sofia:175-393` Sofia hero+stats+About+reviews+pricing+calendar | `12.3` Tutor profile | P5 T01 Profiles | `GET /teachers/:id:664` → `GetTutorProfile:105` + `:665` reviews `:284` + `:667` availability `:351` | `App.tsx:248` `/tutors/:id` → `TutorProfile.tsx:24` `teacherAPI.getProfile` | `MainTabs.tsx:188` `TutorProfile {userId}` |
| S-T-03 | `confirm_trial_booking:48-92` Great choice+Date/Time+$0.00+sticky CTA | `12.5` confirm booking | P5 T01 Booking | `POST /teachers/:id/book:668` → `CreateBooking:416` `isTrial=true` trial credit debit `:469` | `App.tsx:249` `/tutors/:id/confirm` → `ConfirmBooking.tsx:32` `teacherAPI.book isTrial` | `MainTabs.tsx:192` `ConfirmBooking {userId}` |
| S-T-04 | `trial_credit_dashboard:1` credits+How Trials+Recommended+History | `12.5` trial credit dashboard + `13.1` 1 credit/mo | P5 T01/P4 Credits | `GET /teachers/trial-credits/dashboard:651` → `GetTrialCreditDashboard:249` + `:650` `GetTrialCredit:232` | `App.tsx:250` `/trial-credits` → `TrialCredits.tsx:17` fetch dashboard | `MainTabs.tsx:198` `TrialCredits` |
| S-T-05 | `teacher_dashboard:1` Welcome+Earnings+Availability+Students+Checklist; `student_management_progress_teacher:1` students section | `12.4` Teacher dashboard | P5 T02 Dashboard | `GET /teachers/dashboard:649` → `GetDashboard:635` checklist `:638` earnings `:688` students `:751` | `App.tsx:251` `/teacher/dashboard` → `TeacherDashboard.tsx:12` fetch dashboard | `MainTabs.tsx:199` `TeacherDashboard` |
| S-T-06 | `payout_settings_history:1` Lifetime+Methods+Breakdown+Withdraw+History; `teacher_earnings_overview:1` earnings alias | `12.7` Payouts + `12.4` earnings + `13.1` 10/15% fee | P5 T02 Payments + P1 Premium fee | `GET /teachers/payouts/overview:638` → `GetOverview:107` + `GET/POST/DELETE/PUT methods:641-644` → `ListMethods:199`/`AddMethod:219`/`RemoveMethod:259`/`SetDefault:271` + `GET history:639` `:293` + `POST withdraw:640` `:334` | `App.tsx:252` `/teacher/payouts` → `Payouts.tsx:20` `payoutsAPI.*` | `MainTabs.tsx:200` `Payouts` |

*Seed anchor:* `backend/internal/services/dev_seed.go:79` Sofia tutor is the canonical fixture for all marketplace slices — browse returns her, profile renders her, confirm books her, credits reflect booking her, dashboard is hers, payouts are hers.

*BA signature line: ___________________________  Date: __________  Commit: __________  AVD screenshots attached: [ ] browse  [ ] Sofia profile  [ ] confirm $0  [ ] trial credits  [ ] teacher dashboard  [ ] payouts*

