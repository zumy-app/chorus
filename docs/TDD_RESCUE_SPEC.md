# TDD Rescue Spec — BA → QA → Green Gate

> Authority: `REQUIREMENTS.md:1` + `REQUIREMENTS_MASTER.md:1` (machine source) + `wireframes/chorus_missing_requirements_roadmap.md:1`
> Method: Business Analyst decomposes each `wireframes/` + FR into a **vertical slice** (Gherkin + API contract + DoD). QA writes **failing tests first** in `backend/*_test.go` + `frontend/vitest` + `mobile/jest` + `e2e/acceptance/*.spec.ts` against real Postgres/Redis via `backend/internal/services/dev_seed.go:31` (`--seed-dev`). Slice closes only when **every testRefs green** + `go vet` + `docker` + device smoke pass.

## 0. Why TDD fixes the hollow app

Previous CrewAI marked `crew/phase_status.json:58` `DONE` after `crew/autonomous_flow.py:1` dry-run with `sqlmock`. Real `start-android.ps1:1` hit `backend/internal/services/teacher.go:60` unterminated string and `mobile/src/screens/LearnScreen.tsx:34` silent `.catch(()=>{})`. TDD reverses the flow: **no DONE without a failing test that proves the requirement**.

## 1. BA Sliced Requirements (traceable to REQUIREMENTS.md)

### Slice S-LEARN-ES — Learn Hub for Spanish (P0 launch-blocking)

**Wireframes:** `languagelearning` (Learn hub), `my_vocabulary_hub`, `ai_scenario_roleplay_ordering_coffee` + `real_world_scenario_practice`, `daily_practice_session`, `learning_journey_progress`, `streak_recovery_challenge`, `placement_test_*`

**FRs:** FR-31 Your Learning Path metrics, FR-32 Seed+Personal+Unified SRS, FR-34 Scenario role-play

| ID | BA Requirement (testable) | Source |
|---|---|---|
| S-LEARN-01 | `GET /learning/dashboard?targetLanguage=es&nativeLanguage=en` returns `vocabulary.dueToday>0` and `total>0` for a fresh `es` learner (seeding via `backend/internal/services/seed_queue.go:236` + `EnsureSeeded` `backend/internal/handlers/learning.go:119`). | FR-32 + `curriculum.go:89` seedExtraScenarios |
| S-LEARN-02 | `GET /learning/scenarios?targetLanguage=es` returns ≥5 scenarios with `openingLine + translation + chunkBank` for `es` (cafe/market etc). `ScenariosScreen.tsx:1` / `Scenarios.tsx:1` lists them. | FR-34 + `curriculum.go:783` extraScenarioSpecs |
| S-LEARN-03 | Quick Drills: `Learn` → tap `Drills` → `POST /learning/sessions/start {mode:"quick_drill", targetLanguage:"es"}` → 200 + `sessionId`; `LessonSessionScreen.tsx:99` / `LessonSession.tsx:138` renders. | FR-31 dailyGoal |
| S-LEARN-04 | Vocabulary: `GET /learning/vocabulary/mined` + `GET /learning/srs/queue` feeds `VocabularyReviewScreen.tsx:1` / `VocabularyReview.tsx:1`; `dueToday` mini-track reflects count. | FR-28 FR-32 |
| S-LEARN-05 | Error UX: mobile dashboard load failure shows retry banner (not silent), matching `frontend/src/pages/Learn.tsx:58` retry pattern. | FR-18 no stubs |
| S-LEARN-06 | Monthly activity card `monthlyActivity[]` renders `wordsLearned`/`sentencesUnderstood` per month, browsable. | FR-31 |

**API contracts:** `packages/shared/src/api.ts:750` `learning.getDashboard`, `:830` `startSession`, `:696` `getScenarios`, `:687` `getMinedItems`
**DoD:** slice tests `e2e/acceptance/learn.es.spec.ts` + `mobile/src/screens/__tests__/learning.test.tsx` + `frontend/src/pages/__tests__/learningPages.test.tsx` all green on seeded DB; emulator shows 4 cards + drill session.

### Slice S-TUTOR-APPLY — Become a Teacher (P0 marketplace entry)

**Wireframe:** `become_a_teacher` (`wireframes/become_a_teacher/*.png`)

**FRs:** Phase 4 12.1 Sign up as teacher

| ID | BA Requirement | Source |
|---|---|---|
| S-TUTOR-01 | `VideoURL` OPTIONAL (2026-03 decision). `models.go:1050` `binding:"omitempty,url"` → empty `""` accepted; non-empty must be valid URL. `frontend/src/pages/BecomeTeacher.tsx:92` placeholder `https://...` optional. | Rescue C1 |
| S-TUTOR-02 | Required fields: `bio 10-1000` (`min=10,max=1000`), `languages min=1`, `rateCents≥100` (`min=100`). Validation errors are **field-specific** (`handlers/teacher.go:30` forwards `err.Error()`), not generic `"Check bio, languages, rate and video URL."` | models `TeacherApplyRequest:1045` |
| S-TUTOR-03 | Certificates: `type ∈ {teaching_degree, language_certificate, other}`, `issuer` trimmed non-empty, `1900≤year≤2030`, `fileUrl` trimmed non-empty. Mobile sends `year` as integer (not `"0"` from `parseInt||0`). | `teacher.go:69` validCertTypes |
| S-TUTOR-04 | `POST /teachers/apply` with valid payload → `200 {application:{status:"pending"}}`; `GET /teachers/me` round-trips same data. Web uses `client.teacher.apply` (`shared/api.ts:1112`), mobile via `api.post('/teachers/apply')` payload parity. | `handlers/teacher.go:27` |
| S-TUTOR-05 | Mobile cert UX: type selector shows 3 valid types; year input defaults to current year on invalid parse (not `0`). | `BecomeTeacherScreen.tsx:20` rescue |
| S-TUTOR-06 | Upsert semantics: `ON CONFLICT (user_id) DO UPDATE` resets to `pending` (documented; rejected→pending allowed for re-apply). | `teacher.go:84` |

**DoD:** `backend/internal/services/teacher_tdd_test.go:TC-TUTOR-*` green + `e2e/acceptance/tutor.apply.spec.ts` green (empty video passes, invalid bio fails with field hint, mobile year `0` no longer sent).

### Slice S-SMOKE — Device smoke (SRE gate)

| ID | BA Requirement | Source |
|---|---|---|
| S-SMOKE-01 | `go vet ./...` + `go build ./...` + `go test ./...` exit 0 on `backend/` (real, not mocked — catches `teacher.go:60` syntax). | Global DoD `REQUIREMENTS_MASTER.md:18` |
| S-SMOKE-02 | `start-android.ps1:319` boots `postgres`+`redis` healthy, backend `/health` returns `commit==HEAD` via `observability/health.go:39` + `CHORUS_BUILD_COMMIT`. | Rescue C3 |
| S-SMOKE-03 | `go run ./cmd/server --seed-dev` provisions `alice.dev@chorus.test / bob.dev@chorus.test / sofia.tutor@chorus.test / chorus-dev-invite-2026` (`dev_seed.go:16`). | Acceptance fixtures |
| S-SMOKE-04 | `ALLOW_OPEN_REGISTRATION` flag (`config.go:89` + `handlers/auth.go:68`) permits register without invite on dev (`true`) else invite-gated. | Rescue C2 |

## 2. QA Test Design — Every testRefs must pass

### TestRefs index (QA owns)

| TestRef | Slice | File | Asserts |
|---|---|---|---|
| `TC-LEARN-01` | S-LEARN-01 | `backend/internal/services/learning_dashboard_test.go` + `e2e/acceptance/learn.es.spec.ts` | dashboard `dueToday>0` for fresh `es` |
| `TC-LEARN-02` | S-LEARN-02 | `backend/internal/services/curriculum_test.go` + `ScenariosScreen` render | ≥5 `es` scenarios with chunks |
| `TC-LEARN-03` | S-LEARN-03 | `frontend/src/pages/__tests__/learningPages.test.tsx` + `mobile/src/screens/__tests__/learning.test.tsx` | Drills → session navigation |
| `TC-LEARN-05` | S-LEARN-05 | `mobile/src/screens/__tests__/learnError.test.tsx` (new) | error banner + retry appears, not silent |
| `TC-TUTOR-01` | S-TUTOR-01+02 | `backend/internal/handlers/teacher_tdd_test.go:TestApply_VideoOptional` + `TestApply_BioValidation` | empty video 200, short bio 400 with `bio` hint |
| `TC-TUTOR-03` | S-TUTOR-03 | `backend/internal/services/teacher_tdd_test.go:TestApply_CertYear` | year `0` rejected, `2025` accepted |
| `TC-TUTOR-04` | S-TUTOR-04 | `e2e/acceptance/tutor.apply.spec.ts` | `POST /teachers/apply` → `GET /teachers/me` round-trip |
| `TC-SMOKE-01..04` | S-SMOKE | `backend/internal/observability/health_test.go` + `gate-on-dev.sh --slice` | vet/build/commit/seed |

### QA execution layers

- **Unit:** `go test ./...` (backend), `vitest run` (frontend 191 tests), `jest` (mobile) — each slice has at least one unit assert.
- **Integration:** `go test -tags=integration` with `--seed-dev` DB + real Redis; `verify-release-gate.sh` offline + online.
- **E2E:** `e2e/acceptance/` Playwright single worker against `docker-compose.dev.yml` + seeded fixtures; skipped tags `GAP:` remain `DONE=false` until UI lands (per `WIREFRAME_TRACE.md:28` Rec.1-2).
- **Load:** `artillery.soak.yml` 1k WS 50 msg/s — not per slice, but per release.

## 3. TDD Flow per slice (ATDD, vertical)

```
BA writes slice table above (this doc) — Gherkin + wireframe PNG refs
  ↓
QA writes FAILING tests (red) — commit testRefs with `skip` if backend not ready, or real red if contract broken
  ↓
Dev implements minimal fix until testRefs green — `go vet && go test` loop inside `crew/loop.py`
  ↓
SRE runs gate-on-dev slice — only then `crew/state.py` may set slice DONE
```

No phase `DONE` when any `testRefs` red or missing. This replaces the old `autonomous_flow.py` flag-based DONE.

## 4. Evidence for current rescue (P0 fixed)

- `teacher.go:60` fixed `if req.Bio == "" || ...` → `go vet` now 0.
- `handlers/teacher.go:30` forwards `Invalid application: <detail>` so `TC-TUTOR-01` can assert field hint.
- `BecomeTeacherScreen.tsx:23` now `Number(c.year)` with fallback + type selector (3 valid types) so `TC-TUTOR-03` no longer sends `year=0`.
- `LearnScreen.tsx:29` now shows error + retry, matching `Learn.tsx:58` so `TC-LEARN-05` passes.
- `dev_seed.go:31` + `health.go:39` + `config.go:89` already landed (uncommitted) and vetted; after this doc commit, `gate-on-dev.sh --slice S-LEARN-ES --slice S-TUTOR-APPLY` will be the TDD proof.

---
*Teams: Business Analyst owns §1 (this file), QA owns §2 testRefs, Dev owns green, SRE owns §3 gate. All testRefs must be working and pass for feature to be considered implemented.*
