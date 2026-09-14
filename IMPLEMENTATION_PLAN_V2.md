# Implementation Plan V2: Language Learning Engine Overhaul

> Revised from V1 incorporating architectural critique. Mobile-first. TDD enforced. Content stored in SQL, not Go code. Domain-agnostic for future language pairs. Horizontally scalable. All estimates split into engineering vs. content work.

---

## Guiding Principles (Changes from V1)

| V1 | V2 |
|----|-----|
| Curriculum data in Go struct literals | All content in SQL seed files, never in Go code |
| Boot-time seeding of everything | Separate `cmd/seed` binary, checksum-gated |
| Grammar cloze items in JSONB column | Proper `grammar_items` table (queryable, trackable) |
| `any` for payloads throughout | Typed discriminated-union structs at all API boundaries |
| Teacher-authored lessons in `curriculum_lessons` | Separate `teacher_lessons` table, seeder never touches it |
| AI grading in goroutines | Durable `grading_jobs` queue following existing outbox pattern |
| Mobile UI as a final Phase 9 batch | Mobile shipped with every backend phase, not after |
| Phase ordering: content before working features | Correctness order: foundation → core loop → teachers |
| `learning.go` absorbs all new types | Domain-split model files |
| Performance budgets untracked | p95 assertions in Go tests, AVD frame-budget tests in Detox |

---

## Architecture Decisions

### A. Content Storage: SQL Seed Files

All curriculum content (vocabulary, grammar points + items, scenarios, placement items, reading passages, real-talk prompts) lives in the database. It is populated from versioned SQL seed files:

```
backend/
  internal/
    database/
      migrations/         ← schema DDL (existing inline Go strings, kept as-is)
      seeds/              ← reference data, NEW
        seed_checksums.sql   ← tracks what has run
        en_es/               ← Spanish for English speakers
          01_course.sql
          02_units.sql
          03_lexical_items.sql
          04_grammar_points.sql
          05_grammar_items.sql
          06_scenarios.sql
          07_placement_items.sql
          08_real_talk_prompts.sql
        es_en/               ← English for Spanish speakers (Phase 7)
          ...
        en_fr/               ← future
```

Each file is plain SQL using `INSERT ... ON CONFLICT (...) DO UPDATE SET ...`. The `SeedRunner` service reads files from the embedded `seeds/` directory (Go `embed.FS`), checksums each file, and skips files whose checksum hasn't changed — so unchanged content costs one DB round-trip per file on startup.

```go
//go:embed seeds/**/*.sql
var seedFiles embed.FS

type SeedRunner struct{ db *sql.DB }

func (r *SeedRunner) Run(ctx context.Context, pair string) error {
    entries, _ := fs.Glob(seedFiles, fmt.Sprintf("seeds/%s/*.sql", pair))
    for _, path := range entries {
        content, _ := seedFiles.ReadFile(path)
        sum := fmt.Sprintf("%x", sha256.Sum256(content))
        var stored string
        r.db.QueryRowContext(ctx, `SELECT checksum FROM seed_checksums WHERE file=$1`, path).Scan(&stored)
        if stored == sum { continue }
        if _, err := r.db.ExecContext(ctx, string(content)); err != nil {
            return fmt.Errorf("seed %s: %w", path, err)
        }
        r.db.ExecContext(ctx, `INSERT INTO seed_checksums(file,checksum,run_at)
            VALUES($1,$2,NOW()) ON CONFLICT(file) DO UPDATE SET checksum=$2,run_at=NOW()`, path, sum)
    }
    return nil
}
```

**Adding content**: write SQL, commit. No Go code changes, no deploy needed (seed can be run against live DB by ops).

### B. `cmd/seed` Binary (separate from `cmd/server`)

```
backend/cmd/seed/main.go
```

Commands:
```
go run ./cmd/seed check  --pair en-es       # dry run: show files that would change
go run ./cmd/seed apply  --pair en-es       # apply seeds for one pair
go run ./cmd/seed apply  --all              # apply all pairs
go run ./cmd/seed status                    # show last-applied checksum per file
```

The server's `main.go` keeps calling `SeedRunner.Run("en-es")` at boot — but it's now a 30-line read-checksum-skip loop, not 1000 upserts.

### C. `grammar_items` Table (not JSONB)

Grammar drill items are first-class rows:

```sql
CREATE TABLE grammar_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    grammar_point_id UUID NOT NULL REFERENCES grammar_points(id) ON DELETE CASCADE,
    ordinal INT NOT NULL,
    item_type TEXT NOT NULL CHECK (item_type IN ('cloze','mcq','reconstruction')),
    prompt TEXT NOT NULL,
    sentence_with_blank TEXT,
    choices TEXT[] NOT NULL DEFAULT '{}',
    correct TEXT NOT NULL,
    accept_variants TEXT[] NOT NULL DEFAULT '{}',
    note TEXT NOT NULL DEFAULT '',
    is_active BOOL NOT NULL DEFAULT true,
    UNIQUE (grammar_point_id, ordinal)
);
```

Per-item mastery tracking:
```sql
CREATE TABLE user_grammar_item_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    grammar_item_id UUID NOT NULL REFERENCES grammar_items(id) ON DELETE CASCADE,
    grammar_point_id UUID NOT NULL,
    correct BOOL NOT NULL,
    quality INT NOT NULL DEFAULT 0,
    latency_ms INT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX ON user_grammar_item_attempts(user_id, grammar_point_id, created_at DESC);
```

### D. Typed Content Structs (no `any` on API surface)

```go
// AssignmentContent is a typed discriminated union at the API boundary.
type AssignmentContent struct {
    Type     AssignmentType              `json:"type"`
    Vocab    *VocabAssignmentContent     `json:"vocab,omitempty"`
    Writing  *WritingAssignmentContent   `json:"writing,omitempty"`
    Reading  *ReadingAssignmentContent   `json:"reading,omitempty"`
    Scenario *ScenarioAssignmentContent  `json:"scenario,omitempty"`
}

// Internal DB storage uses json.RawMessage — never interface{}/any in models.
type TeacherAssignment struct {
    ID      string          `json:"id"      db:"id"`
    Type    AssignmentType  `json:"type"    db:"type"`
    Content json.RawMessage `json:"-"       db:"content"`
    // Typed payload populated after scan:
    Parsed  *AssignmentContent `json:"content"`
}
```

The handler scans the raw JSON column and unmarshals into the typed struct. No `any` escapes into service logic.

### E. Durable Grading Queue (not goroutines)

Follows the existing `grammar_jobs` / `word_mining_jobs` durable outbox pattern exactly:

```sql
CREATE TABLE grading_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    submission_id UUID REFERENCES assignment_submissions(id) ON DELETE CASCADE,
    vocabulary_id UUID REFERENCES vocabulary(id) ON DELETE CASCADE,
    job_type TEXT NOT NULL CHECK (job_type IN ('production','writing','placement_open')),
    payload JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','done','failed')),
    result JSONB,
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    next_attempt_at TIMESTAMPTZ DEFAULT NOW(),
    processing_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);
CREATE INDEX ON grading_jobs(status, next_attempt_at);
```

`GradingQueueService` follows the same worker+sweeper pattern as `GrammarQueueService`. Results pushed over WebSocket via existing `wsHub.SendToUser`.

### F. `teacher_lessons` Table (separate from `curriculum_lessons`)

```sql
CREATE TABLE teacher_lessons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_language VARCHAR(10) NOT NULL,
    native_language VARCHAR(10) NOT NULL DEFAULT 'en',
    title VARCHAR(255) NOT NULL,
    objective TEXT NOT NULL DEFAULT '',
    estimated_minutes INT NOT NULL DEFAULT 5,
    cefr_level VARCHAR(2),
    is_published BOOL NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE teacher_lesson_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID NOT NULL REFERENCES teacher_lessons(id) ON DELETE CASCADE,
    ordinal INT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('intro','mcq','cloze','free_recall',
                                        'production','reading_passage','writing','gap_fill')),
    prompt JSONB NOT NULL DEFAULT '{}',
    answer_key JSONB NOT NULL DEFAULT '{}',
    UNIQUE (lesson_id, ordinal)
);
```

The `LessonService` queries both tables and unifies behind a `LessonSource` enum. Seeder never reads `teacher_lessons`.

### G. Domain-Split Model Files

Split `learning.go` (currently 700+ lines) before adding anything new:

```
backend/internal/models/
  learning_profile.go     — UserLanguageProfile, LearningPairCapability, DailyGoal*
  learning_curriculum.go  — CurriculumCourse, Unit, Lesson, Step, LexicalItem, GrammarPoint
  learning_grammar.go     — GrammarItem, GrammarMastery, GrammarItemAttempt
  learning_session.go     — LearningSession, SessionItem, SRSQueue*, SessionQuestion
  learning_placement.go   — PlacementQuestion, PlacementResult, PlacementItem
  learning_scenario.go    — ScenarioScript, Phase, Run, Turn, ScenarioAIReply
  learning_teacher.go     — TeacherAssignment, AssignmentContent, AssignmentSubmission, StudentProgress*
  vocabulary.go           — VocabularyCard, MinedItem (keep as-is)
  models.go               — non-learning domain (Chat, Message, User, etc.)
```

### H. Performance Budgets (enforced in tests)

| Operation | Budget | Test location |
|-----------|--------|---------------|
| SRS queue load (12 items) | p95 < 80ms | `perf_benchmark_test.go` |
| Scenario AI reply (provider chain) | p95 < 3000ms | `perf_benchmark_test.go` (mock provider) |
| Placement test answer | p95 < 50ms | `perf_benchmark_test.go` |
| Grammar drill item fetch | p95 < 30ms | `perf_benchmark_test.go` |
| Daily session start (15 items) | p95 < 100ms | `perf_benchmark_test.go` |
| Mobile: scenario screen render | < 16ms/frame | Detox `waitForExpect` + `getByTestId` |
| Mobile: session start → first item visible | < 1500ms | Detox E2E |
| DB: `grammar_items` per-user due fetch | p95 < 20ms | `go test -bench` |

### I. Scalability Notes

The existing architecture is already horizontally scalable for messaging (Redis-backed `ConnectionRegistry`, HAProxy LB, stateless Go servers). For the learning engine:

- **Learning sessions**: stateless — all state in DB, no in-memory session. Any server can handle any request.
- **AI grading queue**: uses the durable outbox pattern. Multiple workers can process independently (row-level locking via `FOR UPDATE SKIP LOCKED`).
- **SRS queue**: pure DB reads + SM-2 writes. Scales with connection pool. At high user counts, `user_grammar_item_attempts` and `vocabulary_practice_attempts` will be the hottest tables — partition by `created_at` (monthly) when they exceed 50M rows.
- **DB sharding readiness**: `vocabulary`, `vocabulary_practice_attempts`, `learning_sessions`, `learning_session_items` are all keyed by `user_id`. If sharding becomes necessary, these are the shard key. Language pair is a good tenant key for a multi-region deployment.
- **Connection pool**: raise `SetMaxOpenConns` from 25 to 100 in `database.Connect` — the learning engine adds ~5 additional concurrent queries per active user session vs. the messaging baseline.
- **Redis for SRS queue caching**: cache the `GetSRSQueue` result per user with a 30-second TTL using the existing Redis client. Invalidate on any SRS write for that user.

---

## Testing Strategy

### Test Pyramid

```
           ┌────────────────────────┐
           │   Detox E2E on AVD     │  ← real app, real backend, real DB
           │   (per phase, ~5 TCs)  │
           ├────────────────────────┤
           │  Mobile Blackbox       │  ← ts-node axios against live API
           │  (mobile/tests/*.ts)   │
           ├────────────────────────┤
           │  Playwright Acceptance │  ← web browser / API HTTP
           ├────────────────────────┤
           │  Go Integration Tests  │  ← real Postgres/Redis via Docker
           │  (new: *_integration_  │
           │   test.go with build   │
           │   tag `integration`)   │
           ├────────────────────────┤
           │  Go Unit Tests         │  ← sqlmock + miniredis (existing)
           ├────────────────────────┤
           │  Mobile Component      │  ← RTL + jest.mock (existing)
           └────────────────────────┘
                   wide base
```

### Test Layer Specifications

#### 1. Go Unit Tests (white-box, existing pattern, extend for all new services)
- **Pattern**: sqlmock for DB, miniredis for Redis, exact same as `auth_test.go`
- **New services requiring unit tests**: `GrammarItemService`, `PlacementService` (updated), `GradingQueueService`, `TeacherAssignmentService`, `SeedRunner`
- **Convention**: one `_test.go` per `_service.go` or `_handler.go` in the same package
- **Run**: `go test ./...` (in CI job `test`)
- **Coverage target**: ≥ 80% statement coverage on all new `services/` files

#### 2. Go Integration Tests (new — talks to real Postgres)
Build tag `//go:build integration` keeps them out of `go test ./...` (unit CI). Run in a dedicated CI job with real Postgres/Redis service containers.

```go
//go:build integration

package services_test

import (
    "database/sql"
    "os"
    "testing"
)

func integrationDB(t *testing.T) *sql.DB {
    t.Helper()
    url := os.Getenv("TEST_DATABASE_URL")
    if url == "" {
        t.Skip("TEST_DATABASE_URL not set — skipping integration test")
    }
    db, err := database.Connect(url)
    if err != nil { t.Fatalf("connect: %v", err) }
    t.Cleanup(func() { db.Close() })
    return db
}
```

**What these verify** (examples):
- `TestIntegration_PlacementItemBank_SpanishHas20PerBand` — seeds the EN→ES course, queries `placement_items`, asserts ≥20 rows per CEFR band
- `TestIntegration_GrammarItems_AllPointsHaveDrillItems` — verifies every `grammar_points` row has ≥3 linked `grammar_items`
- `TestIntegration_SeedRunner_ChecksumSkipsUnchanged` — runs seed twice, asserts DB execution count (pg_stat_user_tables) doesn't increase on second run
- `TestIntegration_GradingQueue_WrittenToDurable` — submits a writing assignment, triggers AI grade job, verifies `grading_jobs` row appears with correct payload
- `TestIntegration_TeacherAssignment_CreateAndSubmit` — full lifecycle: teacher creates, student submits, AI pre-grade fires, teacher reviews
- `TestIntegration_ScenarioRun_IntentsDetectedCorrectly` — sends known phrases for each scenario phase, verifies `covered_intents` in DB matches expected
- `TestIntegration_SRSQueue_GrammarItemsInterleaved` — starts daily session, verifies at most 4 grammar items and none are consecutive

**CI job**: `integration-tests` — runs after `test`, requires Postgres + Redis service containers. Uses seeded EN→ES course via `go run ./cmd/seed apply --pair en-es`.

#### 3. Mobile Blackbox Tests (ts-node axios, existing `mobile/tests/*.ts`)

Extend `learning-functional-tests.ts` to cover all new endpoints:

```
TestLearning_Placement_20Items          — start, answer all 20, assert CEFR assigned
TestLearning_Placement_GapFill          — answer a gap_fill_type item with typed text
TestLearning_GrammarItems_FetchDue      — GET /learning/grammar/points, assert items[] non-empty
TestLearning_GrammarDrill_RecordAttempt — POST drill attempt, verify mastery updated in DB
TestLearning_Scenario_FullCoffeeFlow    — start ordering-coffee, send all 5 phases, complete
TestLearning_Scenario_IntentsCovered    — after each phase, GET run and assert covered_intents
TestLearning_Scenario_HintReturnsChunks — POST hint, assert chunk_bank non-empty
TestLearning_Writing_SubmitAndGrade     — POST writing answer, poll grading_jobs until done
TestLearning_TeacherAssign_Create       — sofia creates writing assignment for alice
TestLearning_TeacherAssign_Submit       — alice submits, verify submission in DB
TestLearning_TeacherAssign_Review       — sofia reviews, verify teacher_feedback stored
TestLearning_StudentProgress_VocabStages— GET /teachers/students/:id/progress, assert byStage keys
TestLearning_SeedRunner_PairExists      — GET /learning/capabilities?pair=es-en, assert full_course (Phase 7+)
```

Each test that touches the DB cross-verifies via a direct `TEST_DATABASE_URL` query. For example, after completing a scenario:

```ts
async function verifyScenarioCompletedInDB(runId: string) {
    const db = new pg.Client(process.env.TEST_DATABASE_URL);
    await db.connect();
    const res = await db.query(`SELECT status, xp_awarded FROM scenario_runs WHERE id=$1`, [runId]);
    assert(res.rows[0].status === 'completed', 'scenario_run not marked completed in DB');
    assert(res.rows[0].xp_awarded > 0, 'xp not awarded');
    await db.end();
}
```

Run in CI `mobile-blackbox` job — already has Postgres + Redis + live backend.

#### 4. Detox E2E on Android AVD (functional + E2E on real app)

**Setup**: `.detoxrc.js` is already configured for `Medium_Phone_API_36.1`. Detox runs the actual built APK against the real backend.

**Add to `mobile/e2e/` directory**:

```
mobile/e2e/
  helpers/
    auth.helper.ts         — login(user), switchUser(userA, userB)
    learning.helper.ts     — navigateToLearn(), startPlacement(), etc.
    db.helper.ts           — direct DB queries for cross-verification
  placement.e2e.ts
  scenario-coffee.e2e.ts
  scenario-switch-users.e2e.ts   ← two-user test: alice starts, bob resumes
  grammar-drill.e2e.ts
  lesson-session.e2e.ts
  teacher-assignment.e2e.ts      ← sofia assigns, alice submits (user switch)
  student-progress.e2e.ts
```

**Example: `scenario-coffee.e2e.ts`**

```ts
describe('Scenario: Ordering Coffee — A1', () => {
  beforeAll(async () => {
    await device.launchApp({ newInstance: true });
    await auth.login('alice.en-es@chorus.test', 'ChorusDev2026!');
  });

  it('navigates to Scenarios screen', async () => {
    await learning.navigateToLearn();
    await element(by.text('Scenarios')).tap();
    await expect(element(by.text('Ordering Coffee'))).toBeVisible();
  });

  it('starts the scenario and sees AI opening line', async () => {
    await element(by.text('Ordering Coffee')).tap();
    await element(by.text('Start')).tap();
    // AI opening line in target language (Spanish for EN→ES)
    await waitFor(element(by.id('ai-message-0'))).toBeVisible().withTimeout(5000);
    const msg = await element(by.id('ai-message-0')).getAttributes();
    expect(msg.text).toBeTruthy();
  });

  it('phase 1: greet intent covers, phase advances', async () => {
    await element(by.id('scenario-input')).typeText('Hola, quiero un café por favor.');
    await element(by.id('scenario-send')).tap();
    await waitFor(element(by.id('phase-indicator'))).toHaveText('2 / 5').withTimeout(4000);
    // Cross-verify in DB
    const run = await db.queryOne(`SELECT current_phase_ordinal, covered_intents
        FROM scenario_runs WHERE user_id=(SELECT id FROM users WHERE email='alice.en-es@chorus.test')
        ORDER BY started_at DESC LIMIT 1`);
    expect(run.current_phase_ordinal).toBe(2);
    expect(run.covered_intents).toContain('greet');
  });

  it('completes all 5 phases and shows summary', async () => {
    // phases 2–5 via helper
    await learning.completeScenarioPhases(['Un café con leche', 'Grande', '¿Cuánto es?', 'Gracias, adiós']);
    await waitFor(element(by.id('scenario-summary'))).toBeVisible().withTimeout(6000);
    await expect(element(by.id('xp-awarded'))).toBeVisible();
    // Verify completion in DB
    const run = await db.queryOne(`SELECT status, xp_awarded FROM scenario_runs
        WHERE user_id=(SELECT id FROM users WHERE email='alice.en-es@chorus.test')
        ORDER BY started_at DESC LIMIT 1`);
    expect(run.status).toBe('completed');
    expect(run.xp_awarded).toBeGreaterThan(0);
  });
});
```

**Example: `teacher-assignment.e2e.ts`** (user switch pattern)

```ts
describe('Teacher creates writing assignment, student submits', () => {
  it('sofia creates writing assignment for alice', async () => {
    await auth.login('sofia.tutor@chorus.test', 'ChorusDev2026!');
    await learning.navigateToTeacherDashboard();
    await element(by.text('Assignments')).tap();
    await element(by.id('new-assignment-fab')).tap();
    // Fill form...
    await element(by.id('assignment-type-writing')).tap();
    await element(by.id('assignment-prompt')).typeText('Escribe un email a un amigo...');
    await element(by.id('submit-assignment')).tap();
    await expect(element(by.text('Assignment created'))).toBeVisible();
  });

  it('alice sees the assignment and submits', async () => {
    await auth.switchUser('alice.en-es@chorus.test', 'ChorusDev2026!');
    await learning.navigateToAssignments();
    await expect(element(by.text('Escribe un email...'))).toBeVisible();
    await element(by.text('Escribe un email...')).tap();
    await element(by.id('writing-input')).typeText('Hola María, este fin de semana voy a...');
    await element(by.id('submit-btn')).tap();
    await waitFor(element(by.text('Submitted'))).toBeVisible().withTimeout(3000);
    // DB cross-verify
    const sub = await db.queryOne(`SELECT content, submitted_at FROM assignment_submissions
        WHERE student_id=(SELECT id FROM users WHERE email='alice.en-es@chorus.test')
        ORDER BY submitted_at DESC LIMIT 1`);
    expect(sub.content).toBeDefined();
  });
});
```

**`auth.helper.ts`** `switchUser` implementation:
```ts
export async function switchUser(email: string, password: string) {
  await device.clearKeychain(); // clear stored tokens
  await device.launchApp({ newInstance: true });
  await element(by.id('login-email')).typeText(email);
  await element(by.id('login-password')).typeText(password);
  await element(by.id('login-submit')).tap();
  await waitFor(element(by.id('main-tabs'))).toBeVisible().withTimeout(5000);
}
```

**CI integration for Detox**: Add a `detox-e2e` CI job that runs on a self-hosted runner (AVD requires KVM/hardware acceleration, not available on GitHub-hosted runners). On-premise or use a cloud Android farm (e.g. Firebase Test Lab, BrowserStack App Automate). The CI job is gated — only runs on `main` push, after `mobile-blackbox` passes.

```yaml
detox-e2e:
  needs: [mobile-blackbox]
  if: github.ref == 'refs/heads/main'
  runs-on: self-hosted  # needs AVD/KVM
  steps:
    - uses: actions/checkout@v4
    - name: Build debug APK
      working-directory: mobile
      run: cd android && ./gradlew assembleDebug assembleAndroidTest -DtestBuildType=debug
    - name: Start AVD
      run: $ANDROID_SDK_ROOT/emulator/emulator -avd Medium_Phone_API_36.1 -no-audio -no-window &
    - name: Run Detox learning suite
      working-directory: mobile
      env:
        CHORUS_API_BASE_URL: http://10.0.2.2:8080
        TEST_DATABASE_URL: ${{ secrets.TEST_DATABASE_URL }}
      run: npx detox test --configuration android.emu.debug --testPathPattern="e2e/(placement|scenario|grammar|lesson|teacher)"
```

#### 5. Performance Tests (extend existing `perf_benchmark_test.go`)

Add to the existing performance test file:

```go
// NFR-Learning-1: SRS queue assembly ≤ 80ms p95
func TestPerf_Learning_SRSQueueAssembly(t *testing.T) { ... }

// NFR-Learning-2: Grammar item fetch (due items for user) ≤ 30ms p95
func TestPerf_Learning_GrammarItemFetch(t *testing.T) { ... }

// NFR-Learning-3: Placement answer + ability update ≤ 50ms p95
func TestPerf_Learning_PlacementAnswerUpdate(t *testing.T) { ... }

// NFR-Learning-4: Session start (15-item interleaved) ≤ 100ms p95
func TestPerf_Learning_SessionStart15Items(t *testing.T) { ... }

// NFR-Learning-5: Teacher progress query (vocab by stage + grammar) ≤ 150ms p95
func TestPerf_Learning_TeacherProgressQuery(t *testing.T) { ... }
```

All use sqlmock with realistic row counts (e.g. 200 vocabulary rows for an active learner) to keep the p95 assertions honest.

---

## Phase Overview

| Phase | Backend | Mobile | DB | Tests shipped with phase |
|-------|---------|--------|----|--------------------------|
| **0** | Schema + SeedRunner + model split | — | 8 new/altered tables | Integration: seed checksums, grammar_items shape |
| **1** | Redesigned placement | PlacementScreen (3 modules) | placement_items | Detox: placement flow; Unit: IRT formula; Blackbox: 20-item run |
| **2** | GrammarItemService, GrammarPointService | GrammarDrillScreen, GrammarPointScreen | grammar_items, rule_text, ordinal | Detox: drill session; Unit: SM-2 on grammar items; Integration: all points have items |
| **3** | Scenario catalogue (12), improved AI prompt, intent classification | ScenarioRoleplayScreen (improved), ScenariosScreen | scaffold_hints, ai_follow_up, success_examples on phases | Detox: coffee E2E + DB verify; Unit: intent detection A1 vs B1; Blackbox: phases advance correctly |
| **4** | Authored lesson steps, production grading, GradingQueueService | LessonSessionScreen (production step type), WritingTaskScreen | grading_jobs, reading_passages, lesson_steps authored | Unit: grading queue durable; Integration: writing grade stored in DB; Detox: lesson session flow |
| **5** | TeacherAssignmentService, StudentProgressService | AssignmentsScreen, AssignmentDetailScreen, CreateAssignmentScreen, StudentProgressScreen | teacher_assignments, assignment_submissions, teacher_lessons, teacher_lesson_steps | Detox: teacher→student two-user flow; Integration: full lifecycle; Blackbox: create/submit/review |
| **6** | GradingQueueService worker wired, WebSocket push of grade result | WritingTaskScreen shows AI feedback + teacher feedback | — | Detox: submit writing, wait for AI grade bubble; Unit: queue worker retry logic |
| **7** | Real-talk prompt service (seed bank + Redis daily cache) | RealTalkHubScreen (improved) | real_talk_prompts | Unit: cache key scoped to CEFR+pair; Blackbox: prompts return for A2 level |
| **8** | Daily session loop fixes (grammar interleaving, dynamic goal) | LearnScreen daily goal bar, StreakRecoveryScreen | — | Unit: interleaving invariant (no consecutive same type); Detox: daily session 15 items, completes goal |
| **9** | ES→EN second language pair (cmd/seed, new seed file) | Language pair selection screen | seeds/es_en/*.sql | Integration: capabilities returns full_course for es→en; Blackbox: bob (es→en) can complete placement |

---

## Phase 0: Foundation

### 0.1 Schema Migrations (append to `postgres.go`)

```go
// Learning engine expansion — grammar_items, teacher tables, grading queue,
// content tables, seed checksum tracker.

`CREATE TABLE IF NOT EXISTS seed_checksums (
    file TEXT PRIMARY KEY,
    checksum TEXT NOT NULL,
    run_at TIMESTAMPTZ DEFAULT NOW()
)`,

// Grammar items: first-class rows replacing the items JSONB column
`CREATE TABLE IF NOT EXISTS grammar_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    grammar_point_id UUID NOT NULL REFERENCES grammar_points(id) ON DELETE CASCADE,
    ordinal INT NOT NULL,
    item_type TEXT NOT NULL CHECK (item_type IN ('cloze','mcq','reconstruction')),
    prompt TEXT NOT NULL,
    sentence_with_blank TEXT,
    choices TEXT[] NOT NULL DEFAULT '{}',
    correct TEXT NOT NULL,
    accept_variants TEXT[] NOT NULL DEFAULT '{}',
    note TEXT NOT NULL DEFAULT '',
    is_active BOOL NOT NULL DEFAULT true,
    UNIQUE (grammar_point_id, ordinal)
)`,
`CREATE INDEX IF NOT EXISTS idx_grammar_items_point ON grammar_items(grammar_point_id, ordinal)`,

`CREATE TABLE IF NOT EXISTS user_grammar_item_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    grammar_item_id UUID NOT NULL REFERENCES grammar_items(id) ON DELETE CASCADE,
    grammar_point_id UUID NOT NULL REFERENCES grammar_points(id) ON DELETE CASCADE,
    correct BOOL NOT NULL,
    quality INT NOT NULL DEFAULT 0,
    latency_ms INT,
    created_at TIMESTAMPTZ DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_grammar_item_attempts_user_point
    ON user_grammar_item_attempts(user_id, grammar_point_id, created_at DESC)`,

// Enrich grammar_points with display fields (items column is now in grammar_items table)
`ALTER TABLE grammar_points ADD COLUMN IF NOT EXISTS rule_text TEXT NOT NULL DEFAULT ''`,
`ALTER TABLE grammar_points ADD COLUMN IF NOT EXISTS common_trap TEXT NOT NULL DEFAULT ''`,
`ALTER TABLE grammar_points ADD COLUMN IF NOT EXISTS ordinal INT NOT NULL DEFAULT 0`,
`ALTER TABLE grammar_points ADD COLUMN IF NOT EXISTS is_active BOOL NOT NULL DEFAULT true`,

// Placement item bank
`CREATE TABLE IF NOT EXISTS placement_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES curriculum_courses(id) ON DELETE CASCADE,
    cefr_level TEXT NOT NULL CHECK (cefr_level IN ('A1','A2','B1','B2')),
    module TEXT NOT NULL CHECK (module IN ('receptive_vocab','grammar_production','discourse_reading')),
    item_type TEXT NOT NULL,
    prompt JSONB NOT NULL DEFAULT '{}',
    choices TEXT[] NOT NULL DEFAULT '{}',
    correct TEXT NOT NULL,
    accept_variants TEXT[] NOT NULL DEFAULT '{}',
    difficulty_value INT NOT NULL DEFAULT 500,
    is_active BOOL NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_placement_items_course_level
    ON placement_items(course_id, cefr_level, module)`,

// Reading passages
`CREATE TABLE IF NOT EXISTS reading_passages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES curriculum_courses(id) ON DELETE CASCADE,
    unit_id UUID REFERENCES curriculum_units(id) ON DELETE SET NULL,
    cefr_level TEXT NOT NULL CHECK (cefr_level IN ('A1','A2','B1','B2')),
    title TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL,
    word_count INT NOT NULL DEFAULT 0,
    questions JSONB NOT NULL DEFAULT '[]',
    vocabulary_ids UUID[] NOT NULL DEFAULT '{}',
    is_active BOOL NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_reading_passages_course_level
    ON reading_passages(course_id, cefr_level)`,

// Real-talk prompt bank
`CREATE TABLE IF NOT EXISTS real_talk_prompts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES curriculum_courses(id) ON DELETE CASCADE,
    cefr_level TEXT NOT NULL CHECK (cefr_level IN ('A1','A2','B1','B2')),
    category TEXT NOT NULL CHECK (category IN ('conversation_starter','topic_injector','opinion_phrase')),
    prompt_for_learner TEXT NOT NULL,
    target_phrase TEXT NOT NULL,
    why_useful TEXT NOT NULL DEFAULT '',
    follow_up_chunks JSONB NOT NULL DEFAULT '[]',
    is_active BOOL NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_real_talk_prompts_course_level
    ON real_talk_prompts(course_id, cefr_level, category)`,

// Teacher-authored lessons (never touched by seeder)
`CREATE TABLE IF NOT EXISTS teacher_lessons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_language TEXT NOT NULL,
    native_language TEXT NOT NULL DEFAULT 'en',
    title TEXT NOT NULL,
    objective TEXT NOT NULL DEFAULT '',
    estimated_minutes INT NOT NULL DEFAULT 5,
    cefr_level TEXT CHECK (cefr_level IN ('A1','A2','B1','B2')),
    is_published BOOL NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_teacher_lessons_teacher ON teacher_lessons(teacher_id)`,

`CREATE TABLE IF NOT EXISTS teacher_lesson_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID NOT NULL REFERENCES teacher_lessons(id) ON DELETE CASCADE,
    ordinal INT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('intro','mcq','cloze','free_recall',
                                        'production','reading_passage','writing','gap_fill')),
    prompt JSONB NOT NULL DEFAULT '{}',
    answer_key JSONB NOT NULL DEFAULT '{}',
    UNIQUE (lesson_id, ordinal)
)`,

// Teacher assignments
`CREATE TABLE IF NOT EXISTS teacher_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    booking_id UUID REFERENCES tutor_bookings(id) ON DELETE SET NULL,
    type TEXT NOT NULL CHECK (type IN ('vocabulary_push','reading','writing','scenario','lesson','mixed')),
    title TEXT NOT NULL,
    instructions TEXT NOT NULL DEFAULT '',
    content JSONB NOT NULL DEFAULT '{}',
    due_date TIMESTAMPTZ,
    target_language TEXT NOT NULL,
    native_language TEXT NOT NULL DEFAULT 'en',
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','in_progress','submitted','reviewed')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_teacher_assignments_teacher
    ON teacher_assignments(teacher_id, created_at DESC)`,
`CREATE INDEX IF NOT EXISTS idx_teacher_assignments_student
    ON teacher_assignments(student_id, status)`,

`CREATE TABLE IF NOT EXISTS assignment_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID NOT NULL REFERENCES teacher_assignments(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content JSONB NOT NULL DEFAULT '{}',
    ai_feedback JSONB,
    teacher_feedback JSONB,
    score INT CHECK (score >= 0 AND score <= 10),
    submitted_at TIMESTAMPTZ DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    UNIQUE(assignment_id, student_id)
)`,

// Durable grading queue (mirrors grammar_jobs / word_mining_jobs pattern)
`CREATE TABLE IF NOT EXISTS grading_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    submission_id UUID REFERENCES assignment_submissions(id) ON DELETE CASCADE,
    vocabulary_id UUID REFERENCES vocabulary(id) ON DELETE CASCADE,
    job_type TEXT NOT NULL CHECK (job_type IN ('production','writing','placement_open')),
    payload JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','processing','done','failed')),
    result JSONB,
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    next_attempt_at TIMESTAMPTZ DEFAULT NOW(),
    processing_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
)`,
`CREATE INDEX IF NOT EXISTS idx_grading_jobs_status ON grading_jobs(status, next_attempt_at)`,
`CREATE INDEX IF NOT EXISTS idx_grading_jobs_submission ON grading_jobs(submission_id)`,

// Scenario phase enrichment
`ALTER TABLE scenario_phases ADD COLUMN IF NOT EXISTS scaffold_hints JSONB NOT NULL DEFAULT '[]'`,
`ALTER TABLE scenario_phases ADD COLUMN IF NOT EXISTS ai_follow_up TEXT NOT NULL DEFAULT ''`,
`ALTER TABLE scenario_phases ADD COLUMN IF NOT EXISTS success_examples JSONB NOT NULL DEFAULT '[]'`,
`ALTER TABLE scenario_scripts ADD COLUMN IF NOT EXISTS scaffold_initial TEXT NOT NULL DEFAULT 'guided'
    CHECK (scaffold_initial IN ('guided','supported','independent'))`,

// Add source/teacher columns to curriculum_lessons to distinguish origin
`ALTER TABLE curriculum_lessons ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'curriculum'
    CHECK (source IN ('curriculum','teacher'))`,
`ALTER TABLE curriculum_lessons ADD COLUMN IF NOT EXISTS is_active BOOL NOT NULL DEFAULT true`,
`ALTER TABLE scenario_runs ADD COLUMN IF NOT EXISTS assignment_id
    UUID REFERENCES teacher_assignments(id) ON DELETE SET NULL`,

// Widen lesson_step type constraint for new types
`ALTER TABLE curriculum_lesson_steps DROP CONSTRAINT IF EXISTS curriculum_lesson_steps_type_check`,
`ALTER TABLE curriculum_lesson_steps ADD CONSTRAINT curriculum_lesson_steps_type_check
    CHECK (type IN ('intro','mcq','cloze','free_recall','translation','listening',
                    'speaking','production','chat_prompt','explanation',
                    'reading_passage','gap_fill','writing'))`,

// Performance: connection pool bump
// (handled in database.Connect(), not a migration)
```

### 0.2 Model Files Split

Create the 7 new model files listed in Architecture Decision G. Move types from `learning.go` into their domain files. `learning.go` is then deleted. This is a pure refactor — no behaviour change. All imports update automatically since they're in the same `models` package.

**Test**: `go build ./...` passes after the split.

### 0.3 SeedRunner Service

Create `backend/internal/services/seed_runner.go`:
- `embed.FS` for `seeds/**/*.sql`
- Checksum-gated execution
- Returns list of applied files for logging

Create `backend/cmd/seed/main.go` with `check`, `apply --pair`, `apply --all`, `status` subcommands using stdlib `flag`.

### 0.4 Tests for Phase 0

**Unit** (`seed_runner_test.go`):
- `TestSeedRunner_SkipsUnchangedFile` — run twice, assert second run does zero DB writes (mock)
- `TestSeedRunner_AppliesNewFile` — first run, assert SQL executed
- `TestSeedRunner_HandlesParseError` — bad SQL file returns descriptive error

**Integration** (`seed_runner_integration_test.go`, build tag `integration`):
- `TestIntegration_SeedRunner_ChecksumPersisted` — runs against real DB, queries `seed_checksums`
- `TestIntegration_ModelSplit_AllTypesExist` — compiles after split (caught at `go build`)

---

## Phase 1: Redesigned Placement Assessment

### Backend

**`placement.go` changes:**
1. `buildItemBank` queries `placement_items` table first; calls `buildPlacementFallback()` only if result is empty
2. Improved IRT logistic formula (replace linear delta with `80*(1-P(θ,b))`)
3. 20-item structure: 8 receptive_vocab → 8 grammar_production → 4 discourse_reading
4. `PlacementQuestion` gains `Module` and `ItemType` fields
5. `normalizeAnswer` strips accents for A1/A2, requires them for B1+
6. Accept typed answers: `gap_fill_type` → free-text; `sentence_reconstruction` → ordered word list

**New SQL seed file: `backend/internal/database/seeds/en_es/07_placement_items.sql`**
Contains 80+ items (≥20 per band × 3 modules). Structured as:
```sql
INSERT INTO placement_items (course_id, cefr_level, module, item_type, prompt, choices, correct, accept_variants, difficulty_value)
SELECT
    (SELECT id FROM curriculum_courses WHERE native_language='en' AND target_language='es' AND version='v1'),
    'A1', 'receptive_vocab', 'L2_to_L1_context',
    '{"sentence":"El niño tiene cinco años.","target_word":"cinco","context_translation":"The child is ___ years old."}'::jsonb,
    ARRAY['five','fifteen','yesterday','many'],
    'five', ARRAY[]::TEXT[], 125
WHERE NOT EXISTS (SELECT 1 FROM placement_items WHERE correct='five' AND cefr_level='A1' AND module='receptive_vocab' LIMIT 1);
-- ... 79 more rows
```

### Mobile: `PlacementScreen.tsx` updates

Add 3 input modes driven by `question.itemType`:
- `L2_to_L1_context` → existing MCQ with context sentence highlight
- `gap_fill_type` → `<TextInput>` below sentence-with-blank, keyboard auto-shown, submit on return
- `sentence_reconstruction` → draggable word chips (use `react-native` `FlatList` with reorder, or tappable "add to answer" chips for simpler implementation)
- `passage_mcq` → `<ScrollView>` with passage text (word-tappable for gloss) + MCQ below

Add `testID` attributes to all interactive elements for Detox:
- `placement-option-{index}`, `placement-text-input`, `placement-submit-btn`, `phase-indicator`, `placement-result-cefr`

### Tests for Phase 1

**Unit (`placement_test.go`):**
- `TestPlacement_IRTFormula_CorrectMovesUp`
- `TestPlacement_IRTFormula_WrongMovesDown`
- `TestPlacement_IRTFormula_StaysInBounds` (0-1000)
- `TestPlacement_BuildItemBank_QueriesDBFirst`
- `TestPlacement_BuildItemBank_FallsBackWhenEmpty`
- `TestPlacement_NormalizeAnswer_StripsAccentsForA1`
- `TestPlacement_NormalizeAnswer_PreservesAccentsForB1`
- `TestPlacement_20ItemStructure_ModuleOrder`

**Integration (`placement_integration_test.go`):**
- `TestIntegration_PlacementItems_SpanishHas80Items`
- `TestIntegration_PlacementItems_AllBandsRepresented`
- `TestIntegration_Placement_FullRunAssignsCEFR`

**Blackbox (add to `learning-functional-tests.ts`):**
- `TestPlacement_20Items_FullRun`
- `TestPlacement_GapFillAnswer_TypedTextAccepted`
- `TestPlacement_SkipAssignsA1`

**Detox (`mobile/e2e/placement.e2e.ts`):**
- Navigates to placement, sees 20 items, answers all, lands on result screen
- Gap-fill item shows TextInput, typed answer accepted
- CEFR result stored in DB (DB cross-verify)

---

## Phase 2: Grammar System

### Backend

**New `grammar_point_service.go`:**
```go
type GrammarPointService struct {
    db *sql.DB
}

// GetDueItems returns grammar items due for review for this user,
// across all grammar points they have been exposed to.
func (s *GrammarPointService) GetDueItems(ctx context.Context, userID, targetLang string, limit int) ([]GrammarDrillItem, error)

// GetMicroLesson returns explanation card shown before first drill.
func (s *GrammarPointService) GetMicroLesson(ctx context.Context, pointID string) (*GrammarMicroLesson, error)

// GetNextItem returns the next undrilled item for a specific grammar point.
func (s *GrammarPointService) GetNextItem(ctx context.Context, userID, pointID string) (*GrammarItem, error)

// RecordItemAttempt updates user_grammar_item_attempts and user_grammar_mastery.
func (s *GrammarPointService) RecordItemAttempt(ctx context.Context, userID, itemID string, correct bool, quality int, latencyMs int) error
```

**Session composer fix**: replace the grammar slot stub with a real call to `GrammarPointService.GetDueItems`. Each grammar slot uses one `grammar_items` row as its prompt, not just a reference to the parent `grammar_points` row.

**New endpoints (add to `learning.go` handler):**
```
GET  /learning/grammar/points             — list points with mastery stats
GET  /learning/grammar/points/:id         — get one point + micro-lesson card
GET  /learning/grammar/points/:id/next    — get next due item for this point
POST /learning/grammar/items/:id/attempt  — record attempt result
```

**New seed file: `backend/internal/database/seeds/en_es/04_grammar_points.sql`**
60 grammar points for Spanish A1→B2.

**New seed file: `backend/internal/database/seeds/en_es/05_grammar_items.sql`**
≥4 items per point = ≥240 rows. Example (cloze):
```sql
INSERT INTO grammar_items (grammar_point_id, ordinal, item_type, prompt, sentence_with_blank, correct, note)
SELECT gp.id, 1, 'cloze',
    'Present Tense -ar: Complete the sentence.',
    'Pedro _____ de Madrid.',
    'es', ''
FROM grammar_points gp
JOIN curriculum_courses cc ON gp.course_id = cc.id
WHERE cc.native_language='en' AND cc.target_language='es'
  AND gp.slug = 'ser-personal-pronouns'
ON CONFLICT (grammar_point_id, ordinal) DO NOTHING;
```

### Mobile: `GrammarDrillScreen.tsx` (new), `GrammarPointScreen.tsx` (new)

`GrammarPointScreen`: Shows the micro-lesson card (rule, 3 examples, common trap). "Start Drilling" button navigates to `GrammarDrillScreen`.

`GrammarDrillScreen`: Renders items based on `item_type`:
- `cloze` → sentence with `___`, TextInput
- `mcq` → 4-choice buttons
- `reconstruction` → word chips
Progress bar, feedback overlay (correct/wrong + `note`).
All interactive elements have `testID` for Detox.

Add both to `LearnStack` in `MainTabs.tsx`:
```ts
GrammarDrill: { grammarPointId?: string }
GrammarPoint: { grammarPointId: string }
```

### Tests for Phase 2

**Unit:**
- `TestGrammarPoint_GetDueItems_ReturnsSortedByConfidence`
- `TestGrammarPoint_RecordAttempt_UpdatesMastery`
- `TestGrammarPoint_RecordAttempt_SM2EaseDecreaseOnWrong`
- `TestGrammarPoint_GetNextItem_SkipsCompletedItems`

**Integration:**
- `TestIntegration_GrammarItems_AllPointsHaveAtLeast4Items`
- `TestIntegration_GrammarItems_ClozeItemsHaveBlank`
- `TestIntegration_GrammarDrill_AttemptStoredInDB`

**Blackbox:**
- `TestGrammar_ListPoints_ReturnsWithMastery`
- `TestGrammar_DrillItem_RecordAttempt_DBUpdated`

**Detox (`grammar-drill.e2e.ts`):**
- Navigate Learn → Grammar → tap a point → see micro-lesson → Start Drilling
- Answer first item, see feedback
- After 2 correct answers, mastery stage updates (DB verify)

---

## Phase 3: Full Scenario Catalogue + AI Partner

### Backend

**New seed file: `backend/internal/database/seeds/en_es/06_scenarios.sql`**
All 12 scenarios with full phase data including `scaffold_hints`, `ai_follow_up`, `success_examples`. Each phase has ≥3 scaffold hints, ≥2 success examples.

**`scenario.go` changes:**
1. Load `scaffold_hints`, `ai_follow_up`, `success_examples` in `getPhases()`
2. Replace sparse `genAIReply` payload with full structured prompt from `LEARNING_DESIGN.md §9.1`
3. Add `IntentClassifier` call for CEFR B1+ (with keyword fallback, 1s timeout)
4. Scaffold level progression: on completion without hints used, upgrade level stored in `user_language_profiles.metadata`
5. Fix phase advancement: only advance when ALL required intents are covered, not ANY

**`learning_ai.go` additions:**
- `ClassifyIntents(ctx, message, requiredIntents, intentDefs, cefrLevel)` — AI intent classification for B1+
- Updated `GenerateScenarioReply` system prompt (full §9.1 text parameterized)
- `GradeProduction(ctx, term, sentence, targetLang, nativeLang, cefrLevel)` — stage 4 grader
- `GradeWritingTask(ctx, taskDesc, text, cefr, focusGrammar, focusVocab)` — writing grader

### Mobile: `ScenarioRoleplayScreen.tsx` improvements

- `ChunkBankPanel` component: collapsible, auto-expanded for `guided`, tap-to-show for `supported`, hidden for `independent`
- `GrammarCorrectionBubble` component: underlined error span + correction + 1-line rule, tappable to navigate to `GrammarPoint` screen
- Phase progress indicator (`testID="phase-indicator"` showing `N / total`)
- Hint button (`testID="hint-btn"`) that calls `POST /scenario-runs/:id/hint`
- Scenario summary screen shown on completion with XP, vocab count

### Tests for Phase 3

**Unit:**
- `TestScenario_IntentDetection_KeywordMatchA1`
- `TestScenario_IntentDetection_FallsBackOnAIError`
- `TestScenario_PhaseAdvance_RequiresAllIntents`
- `TestScenario_ScaffoldLevel_UpgradesOnCleanRun`
- `TestScenario_AIPrompt_ContainsCEFRLevel`

**Integration:**
- `TestIntegration_Scenario_12ScenariosSeeded`
- `TestIntegration_Scenario_AllPhasesHaveScaffoldHints`
- `TestIntegration_Scenario_PhaseAdvancePersisted`

**Blackbox:**
- `TestScenario_FullCoffeeFlow_5Phases`
- `TestScenario_IntentsCovered_StoredInDB`
- `TestScenario_HintReturnsChunkBank`

**Detox (`scenario-coffee.e2e.ts`):**
Full coffee scenario E2E with DB cross-verification (shown above in Testing Strategy section).

---

## Phase 4: Lesson System + Grading Queue

### Backend

**New seed files:**
- `backend/internal/database/seeds/en_es/08_reading_passages.sql` — 2 passages per unit (A1–B2 = 32 passages total)
- Extend `06_scenarios.sql` to be idempotent with authored lesson steps

**Authored lesson step seeding**: Add `seedLessonSteps(ctx, lessonID, type, unitData)` to `CurriculumService` that creates actual `curriculum_lesson_steps` rows from SQL data (not synthesized on the fly). Called by the `SeedRunner` via the seed SQL files.

**`GradingQueueService`** (`grading_queue_service.go`):
- Follows `GrammarQueueService` pattern exactly
- Worker loop: `FOR UPDATE SKIP LOCKED` on `grading_jobs WHERE status='pending'`
- Max 3 attempts, 30s backoff
- On completion: `wsHub.SendToUser(userID, "grade_result", result)`
- Registered in `main.go` after `learningAIService`

**`LessonService.gradeStep` update**: production steps enqueue a `grading_jobs` row; return immediate `status: "grading"` response. Client polls or waits for WebSocket push.

### Mobile: `WritingTaskScreen.tsx` (new), `LessonSessionScreen.tsx` updates

`WritingTaskScreen`: word count display, submit button, loading spinner while grading, inline error corrections when result arrives via WebSocket or polling.

`LessonSessionScreen` handles new step types:
- `reading_passage` → renders `ReadingPassageComponent` (tappable words, passage text, MCQ)
- `production` → shows term + context, TextInput, submits to grading queue, shows "Checking..." then result
- `writing` → navigates to `WritingTaskScreen`

### Tests for Phase 4

**Unit:**
- `TestGradingQueue_EnqueuesOnSubmission`
- `TestGradingQueue_RetriesOnFailure`
- `TestGradingQueue_MaxAttemptsExhausted`
- `TestGradingQueue_WorkerMarksProcessing` (FOR UPDATE SKIP LOCKED semantics via sqlmock)
- `TestLesson_ProductionStep_ReturnsGradingStatus`

**Integration:**
- `TestIntegration_GradingJob_CreatedOnWritingSubmit`
- `TestIntegration_GradingJob_ResultStoredAfterWorker`
- `TestIntegration_ReadingPassages_AllUnitsHave2`

**Detox (`lesson-session.e2e.ts`):**
- Start daily session, navigate through vocabulary + grammar + reading_passage items
- Production item shows "Checking…" then feedback
- Session completion screen shows XP

---

## Phase 5: Teacher Assignments

### Backend

**`TeacherAssignmentService`** (`teacher_assignment_service.go`):
- `CreateAssignment`: validates booking relationship, typed content validation per type, inserts row
- `GetForTeacher(teacherID, studentID)`: returns all assignments with submission status
- `GetForStudent(studentID)`: returns pending/in-progress assignments
- `SubmitAssignment(studentID, id, content)`: inserts `assignment_submissions`, enqueues `grading_jobs` for writing type
- `ReviewAssignment(teacherID, id, feedback)`: updates `teacher_feedback`, sets `reviewed_at`, updates assignment `status`
- `GetStudentProgress(teacherID, studentID, targetLang)`: returns `StudentProgressSummary`

**New handler: `teacher_assignment_handler.go`**

New routes (add to `main.go` protected group):
```
POST /teachers/assignments              — create
GET  /teachers/assignments              — list (query param: ?studentId=)
GET  /teachers/assignments/:id          — get one
POST /teachers/assignments/:id/review   — submit teacher feedback
GET  /teachers/students/:studentId/progress — student progress
GET  /students/assignments              — student: list
GET  /students/assignments/:id          — student: get one
POST /students/assignments/:id/submit   — student: submit
```

### Mobile: New Screens

**`AssignmentsScreen.tsx`**: List with type icon, title, due countdown, status chip. FAB for students opens a `WritingTaskScreen` or `ScenarioRoleplayScreen` depending on type. Teachers see all student assignments here.

**`AssignmentDetailScreen.tsx`**: Full assignment content, submit button, shows AI feedback and teacher feedback after review.

**`CreateAssignmentScreen.tsx`**: Type picker (writing / reading / scenario / vocabulary), form per type. Assign to a specific student (picker showing students with bookings).

**`StudentProgressScreen.tsx`**: Vocabulary progress (stages 1–5 bar chart), grammar weakness list, weekly XP bar chart, assignment list with statuses.

Add all to navigation stacks in `MainTabs.tsx`:
```ts
// LearnStack additions:
Assignments: undefined
AssignmentDetail: { assignmentId: string }

// MarketplaceStack additions:
StudentProgress: { studentId: string; studentName: string }
CreateAssignment: { studentId: string; studentName: string; bookingId?: string }
```

Add `testID` to all interactive elements.

### Tests for Phase 5

**Unit:**
- `TestTeacherAssignment_Create_ValidatesBookingExists`
- `TestTeacherAssignment_Create_RequiresApprovedTeacher`
- `TestTeacherAssignment_Create_WritingContentRequired`
- `TestTeacherAssignment_Submit_EnqueuesGradingForWriting`
- `TestTeacherAssignment_Review_UpdatesFeedback`
- `TestStudentProgress_VocabByStage_CorrectCount`
- `TestStudentProgress_GrammarWeakestFirst`

**Integration:**
- `TestIntegration_TeacherAssignment_FullLifecycle`
- `TestIntegration_TeacherAssignment_NoBooking_Rejected`
- `TestIntegration_StudentProgress_Returns5GrammarPoints`

**Blackbox:**
- `TestTeacher_CreateAssignment_Writing`
- `TestStudent_SubmitAssignment`
- `TestTeacher_ReviewAssignment_FeedbackStored`
- `TestTeacher_StudentProgress_VocabStagesPresent`

**Detox (`teacher-assignment.e2e.ts`):**
Two-user test: sofia creates, alice submits (user switch via `auth.switchUser`). Full flow with DB cross-verify.

---

## Phase 6: Grading Queue Worker + AI Feedback Push

This phase wires the `GradingQueueService` worker into `main.go` and ensures the WebSocket push of graded results reaches the mobile client.

**`main.go` change**: add `gradingQueueService.Start()` / `defer gradingQueueService.Stop()` after `wordMiningQueue`.

**`WritingTaskScreen.tsx` update**: subscribe to WebSocket event `grade_result`, update UI when AI feedback arrives. Use same pattern as existing grammar analysis WebSocket subscription.

**Tests for Phase 6:**

**Unit:**
- `TestGradingQueue_WebSocketPushOnComplete` — mock wsHub.SendToUser, verify called
- `TestGradingQueue_SweepStaleProcessingJobs` — items stuck in `processing` > 60s get reset to `pending`

**Detox:**
- Submit a writing assignment, wait for WebSocket push, verify AI feedback bubble appears on screen

---

## Phase 7: Real-Talk Prompts

**`RealTalkService`** (`real_talk_service.go`):
```go
func (s *RealTalkService) GetPromptsForUser(ctx context.Context, userID, targetLang, nativeLang string) ([]RealTalkPrompt, error)
// Cache key: "realtalk:{courseID}:{cefrLevel}:{date}" (Redis, 24h TTL)
// Falls back to seeded bank if AI unavailable or < 3 prompts generated
```

**New seed file: `backend/internal/database/seeds/en_es/09_real_talk_prompts.sql`**
50 prompts per CEFR level × 3 categories = 600 rows for EN→ES.

Replace the hardcoded 3-prompt list in the current `RealTalkPrompts` handler.

**`RealTalkHubScreen.tsx` improvements**: group prompts by category tabs, show `why_useful` tooltip, show `follow_up_chunks` as copyable chips.

**Tests for Phase 7:**

**Unit:**
- `TestRealTalk_CacheKeyIncludesPairAndDate`
- `TestRealTalk_FallsBackToSeededBankOnCacheMiss`
- `TestRealTalk_Returns3PromptsCapped`

**Blackbox:**
- `TestRealTalk_PromptsReturnForA2Level`
- `TestRealTalk_CacheHitOnSecondCall`

---

## Phase 8: Daily Loop + Session Refinements

### Backend

Fix `SessionComposerService`:
1. Grammar slots use `GrammarPointService.GetNextItem` to get actual `grammar_items` rows (not just point metadata)
2. Daily goal from `user_language_profiles.daily_goal_items` (5/10/15/20/25) controls session length
3. Enforce interleaving: no two consecutive items of same type (existing logic, verify passes tests)
4. New `mode: "quick_drill"` creates a 5-item session for learners with limited time

**`LearnScreen.tsx` updates:**
- Daily goal progress bar (N / target items completed today)
- "Quick drill" button (5 items, new session mode)
- Streak at-risk indicator

**Tests for Phase 8:**

**Unit:**
- `TestSession_Interleaving_NoConsecutiveSameType`
- `TestSession_GoalFromProfile_OverridesDefault`
- `TestSession_QuickDrill_5Items`
- `TestSession_GrammarSlot_UsesActualItemRow`

**Detox (`lesson-session.e2e.ts` extension):**
- Complete a full daily session (all N items), verify daily goal bar hits 100%

---

## Phase 9: ES→EN Second Language Pair

### Backend

Create `backend/internal/database/seeds/es_en/` directory with all 9 seed files for English-for-Spanish-speakers.

Key differences vs EN→ES:
- `01_course.sql`: `native_language='es'`, `target_language='en'`
- `03_lexical_items.sql`: `translations = '{"es": "..."}'::jsonb` (Spanish glosses)
- `04_grammar_points.sql`: English grammar pain points for Spanish speakers (articles, progressive vs. simple, perfect tenses, phrasal verbs, conditional mood)
- `06_scenarios.sql`: AI opening lines in English, AI partner speaks English
- `07_placement_items.sql`: prompts in Spanish, target words in English

Update `SeedRunner` to apply `es_en` seeds and insert `learning_pair_capabilities` row for `(es, en)` → `full_course`.

Update `languageName()` in `learning_ai.go` to cover all planned languages: `es, en, fr, de, it, pt, hi, zh, ar, ru, ja, ko`.

Audit all service files for hardcoded language strings — grep for string literals `"es"` and `"en"` outside of `languageName()` and normalize them.

### Mobile

Language pair selection screen in onboarding (if user's `target_languages` has multiple values, let them pick which pair to practice in the Learn tab).

### Tests for Phase 9

**Integration:**
- `TestIntegration_EsEn_CourseSeeded`
- `TestIntegration_EsEn_CapabilitiesReturnFullCourse`
- `TestIntegration_EsEn_PlacementItemsBankPopulated`

**Blackbox:**
- `TestLanguagePair_EsEn_PlacementFullRun` (using bob `es-en@chorus.test`)
- `TestLanguagePair_EsEn_ScenarioStartsInEnglish`

**Detox:**
- Bob (ES→EN user) logs in, completes placement in Spanish, gets English scenario

---

## CI Pipeline Changes

Add two new jobs to `.github/workflows/ci.yml`:

### New Job: `integration-tests`
```yaml
integration-tests:
  needs: [test]
  runs-on: ubuntu-latest
  services:
    postgres:
      image: postgres:15-alpine
      env:
        POSTGRES_DB: messenger_test
        POSTGRES_USER: messenger
        POSTGRES_PASSWORD: password
      ports: ["5432:5432"]
      options: >-
        --health-cmd "pg_isready -U messenger -d messenger_test"
        --health-interval 5s --health-timeout 3s --health-retries 10
    redis:
      image: redis:7-alpine
      ports: ["6379:6379"]
  env:
    TEST_DATABASE_URL: postgres://messenger:password@localhost:5432/messenger_test?sslmode=disable
    TEST_REDIS_URL: localhost:6379
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: ${{ env.GO_VERSION }}
        cache-dependency-path: backend/go.sum
    - name: Migrate + seed
      working-directory: backend
      run: |
        go run ./cmd/seed apply --all
    - name: Run integration tests
      working-directory: backend
      run: go test -tags=integration ./... -v -timeout 120s
    - name: Run performance benchmarks
      working-directory: backend
      run: go test -bench=. -benchtime=1s ./internal/services/ | tee /tmp/bench.txt
    - name: Assert NFR budgets
      run: go test -tags=integration -run TestPerf_ ./internal/services/ -v
```

### New Job: `detox-e2e` (self-hosted, gated)
As described in Testing Strategy section above. Runs on `main` push only, after `mobile-blackbox` passes.

### Update `mobile-blackbox` job
Extend with new learning blackbox tests:
```yaml
- name: Run extended learning blackbox
  working-directory: mobile
  run: npx ts-node --project tsconfig.tests.json tests/learning-functional-tests-v2.ts
```

Where `learning-functional-tests-v2.ts` covers all new endpoints added in Phases 1–9.

---

## Content Work (separate from engineering estimates)

The following is content creation work, not engineering. It should be tracked separately from sprints and done in parallel with engineering by a language expert or AI-assisted batch generation with expert review:

| Content | Owner | Output |
|---------|-------|--------|
| 80 EN→ES placement items (SQL) | Language expert + AI draft | `seeds/en_es/07_placement_items.sql` |
| 60 grammar points + 240+ items (SQL) | Language expert | `seeds/en_es/04_grammar_points.sql` + `05_grammar_items.sql` |
| 12 scenario scripts with full phase data (SQL) | Language expert | `seeds/en_es/06_scenarios.sql` |
| 32 reading passages A1–B2 (SQL) | Language expert | `seeds/en_es/08_reading_passages.sql` |
| 600 real-talk prompts (SQL) | Language expert + AI | `seeds/en_es/09_real_talk_prompts.sql` |
| 80 ES→EN placement items (SQL) | Language expert | `seeds/es_en/07_placement_items.sql` |
| ES→EN grammar points — English for Spanish speakers | Language expert | `seeds/es_en/04+05` |
| ES→EN scenario scripts (English AI partner) | Language expert | `seeds/es_en/06_scenarios.sql` |

All AI-generated content goes into a review table (`content_review_queue`) before being promoted to `is_active = true`. A simple admin endpoint `POST /admin/content/:table/:id/approve` flips the flag.

---

## Engineering Estimates (code only, not content)

| Phase | Engineering Days |
|-------|-----------------|
| 0 — Foundation | 3 |
| 1 — Placement | 3 |
| 2 — Grammar | 3 |
| 3 — Scenarios | 4 |
| 4 — Lessons + Grading Queue | 4 |
| 5 — Teacher Assignments | 4 |
| 6 — Grading Worker + Push | 2 |
| 7 — Real-Talk | 2 |
| 8 — Daily Loop | 2 |
| 9 — ES→EN Pair | 3 |
| **Total** | **~30 engineering days** |

Content creation is on a separate track. Engineering should not be blocked by content — seed files are committed as stubs with enough rows to pass integration tests, and content is filled in later with `go run ./cmd/seed apply`.

---

## What Does NOT Change

- Auth, messaging, WebSocket hub, calls, billing — untouched
- Teacher marketplace (booking, availability, reviews, payouts) — untouched
- `SRSQueueService` / `PracticeService` / SM-2 implementation — untouched (grammar items use the same SM-2 logic via `user_grammar_mastery`)
- `WordMiningService` — untouched
- `FluencyScoreService` — untouched
- HAProxy config, Redis deployment, Prometheus/Grafana — untouched
- Detox `.detoxrc.js` device config — kept as-is, new test files added under `mobile/e2e/`

---

*Document version: 2.0 — September 2026*
*Supersedes IMPLEMENTATION_PLAN.md*
