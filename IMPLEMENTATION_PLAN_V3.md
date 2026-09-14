# Implementation Plan V3: Language Learning Engine Overhaul

> Revised from V2 to incorporate specialized micro-caching structures, non-blocking asynchronous threads, zero-day deployment infrastructure loops, and strict shift-left automated backend validation.

---

## Guiding Principles (Changes from V2)

| V2 | V3 (Architectural Correction) |
|----|-------------------------------|
| "No deploy needed" for embedded seeds | **Deploy-synced configuration seeds** via `embed.FS` for strict CI validation |
| Pure DB-bound SRS session composition | **Redis-decoupled transient session state** to protect Postgres |
| Dense per-click database telemetry tables | **State-tracking mastery ledger** + local open-source object storage archiving |
| Synchronous AI blocking loops | **Non-blocking async job queues** + UI polling/WebSockets |
| Heavy emulator-driven Detox E2E dominance | **Shift-Left Integration Tests** + production Feature Flags / Canaries |

---

## Architecture Decisions

### A. Content Storage: Deploy-Synced Configurations via `embed.FS`
Core curriculum data is treated exactly like application source code. It lives in versioned SQL seed files compiled directly into the application binary using Go's native embedding system (`embed.FS`). 

**Deployment Tradeoff:** Content modifications require a standard rolling binary deployment. This ensures that curriculum data is subjected to code reviews (PRs), syntax checks, and automated integration testing in CI before hitting production.

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
```

The `SeedRunner` service reads files from the embedded `seeds/` directory at startup, calculates a SHA-256 checksum for each file, and matches it against the database. Unchanged files cost exactly one database lookup, bypassing heavy startup overhead.

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

---

### B. Session State: Decoupled Redis Caching Layer
To protect Postgres from high-frequency connection starvation and disk write amplification during active user practice loops, the transient session state is decoupled from the transactional layer.

```
[Mobile App UI Client]
         │ 
         ▼ (High-Frequency State Requests / Every 5 Seconds)
[Go API Service Layer] ◀───(Real-Time Read/Write)───▶ [Redis Distributed Cache]
         │
         ▼ (Asynchronous Session Flush Upon Lesson Completion)
[Postgres Database Persistent Core Engine]
```

1. **Session Activation:** When a user initializes an interaction loop, the server compiles a 15-item payload using standard relational lookups, transforms the structural content into an internal object format, caches it inside **Redis** as a JSON string under a 1-hour expiration limit, and returns the payload.
2. **In-Flight Synchronization:** Mid-session status checks, answer scoring adjustments, and item positioning changes update Redis memory exclusively. 
3. **State Recovery Fallback:** If a Redis cluster partition drops or errors out mid-session, the backend service handles failures gracefully by reading active user histories from Postgres and regenerating an operational transient mirror on the fly.

---

### C. Analytics Capture: State Master Ledgers & Object Archiving
The persistence core avoids infinite transaction entry growth by separating operational application states from granular clickstream event logs.

#### 1. Operational Mastery Blueprint
User learning states use dedicated rows modified via predictive upsert logic paths inside the transactional cluster:
```sql
CREATE TABLE user_grammar_mastery (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    grammar_point_id UUID NOT NULL REFERENCES grammar_points(id) ON DELETE CASCADE,
    mastery_stage INT NOT NULL DEFAULT 1, -- Active tier categorization (1-5)
    ease_factor NUMERIC(4,2) NOT NULL DEFAULT 2.5, -- SM-2 execution variable
    repetitions INT NOT NULL DEFAULT 0,
    interval_days INT NOT NULL DEFAULT 0,
    next_review_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_reviewed_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, grammar_point_id)
);
CREATE INDEX idx_grammar_mastery_due ON user_grammar_mastery(user_id, next_review_at ASC);
```

#### 2. Open-Source Local Storage Telemetry Archiving (MinIO)
High-frequency client events (such as exact text input tracking, latency microsecond scores, and interface touch locations) bypass relational databases completely. The application streams raw analytics payloads via low-priority channel groups directly into an on-premise instance of **MinIO** (an open-source, S3-compatible cloud object store API engine).

---

### D. AI Resilience: Asynchronous Non-Blocking Workers
API transaction paths run decoupled from synchronous external LLM operations. Conversational cycles and text processing actions evaluate using an asynchronous message pattern.

```sql
-- Client posts text -> Worker saves state immediately and breaks network connection dependency
INSERT INTO grading_jobs (user_id, job_type, payload, status)
VALUES ($1, 'production', $2, 'pending')
RETURNING id;
```

1. **Immediate Ack:** The user's active API request thread receives an immediate `202 Accepted` response containing a `job_id`. The server thread is instantly freed back to the connection pool.
2. **Background Processing:** A dedicated Go worker pool processes the `grading_jobs` table asynchronously using `FOR UPDATE SKIP LOCKED`.
3. **UI Delivery:** The mobile app displays an inline shimmering skeleton state ("AI partner is thinking...") while listening to a dedicated **WebSocket** channel. When the background job completes, the outcome is pushed over the socket to render the corrections. If the socket fails, the client cleanly falls back to manual polling against `GET /jobs/:id` every 2 seconds.

---

## Testing & Canary Deployment Strategy

To achieve zero-day delivery velocities securely, business logic verification shifts away from slow, brittle emulators and replaces it with hermetic isolation layers, feature toggles, and targeted production rollouts.

```
       ┌────────────────────────┐
       │   Canary Rollouts      │ 🚀 Managed release paths (1% -> 10% -> 100%)
       ├────────────────────────┘
       │  Detox Smoke Testing   │ 📱 Restricted to 3-5 core happy paths
       ├────────────────────────┤
       │  Go Integration Suites │ 🧪 Automated Docker DB validation layers
       └────────────────────────┘
```

### 1. Minimal Smoke-Test Detox Footprint
End-to-end device testing is strictly capped at **3-5 core user paths** to confirm general interface cohesion. Brittle UI validation layers are never used to catch language logic or content typos.
* **Sequence 1:** Profile authentication setup, login, and application tab switching.
* **Sequence 2:** Baseline placement evaluation entry, wizard progression, and completion.
* **Sequence 3:** Main curriculum lesson initialization, standard item interaction, and score screen processing.

### 2. Shift-Left Content Validation
Because curriculum structures compile directly into application definitions, content sets pass verification checks inside the local automation pipeline before hitting staging systems:
* **Syntax Enforcement:** CI runners stand up short-lived isolated Postgres containers via Docker, build the application binary, and automatically run the `SeedRunner`. 
* **Validation Assertions:** The automated test runner verifies that every grammar entity has an adequate volume of items, all prompt payloads are well-formed JSON strings, and no invalid relational links exist.

### 3. Dynamic Feature Flag Toggles & Canaries
All structural upgrades and interface modules deploy embedded inside dormant code enclosures managed through internal server toggle wrappers.
* **Progressive Exposure:** Features open initially to a **1% internal Canary audience** (e.g., developers, QA testers, content managers).
* **Automatic Rollback:** If monitoring dashboards reveal anomalies in system performance, connection volumes, or error logs, the deployment system flips the toggle off instantly without rebuilding the binary or taking down the cluster. Once validated, feature exposure safely scales to 10%, 25%, and 100% of live production traffic.

---

## Phase Overview

| Phase | Backend Engineering Layers | Mobile Client Layout Modules | Verification Controls & Gateways |
|:---|:---|:---|:---|
| **0** | Base migrations; domain file splitting; `MinIO` initialization hooks | Implementation of structural Feature Flag toggle components | **Integration:** Automated validation checks on embedded data parsing rules |
| **1** | Redesigned placement calculation engine running updated IRT models | `PlacementScreen` featuring structural free-text input variations | **Blackbox:** Comprehensive simulation runs across complete placement datasets |
| **2** | `GrammarPointService` logic tied to active Redis caching clusters | `GrammarPointScreen` paired with micro-lessons and drill views | **Unit:** Verification of proper interval logic outputs |
| **3** | Async conversational workers and background grading queue managers | `ScenarioRoleplayScreen` using non-blocking async visual skeletons | **Integration:** Validation testing against diverse input intent sets |
| **4** | Real-time step processing and asynchronous `grading_jobs` management | `LessonSessionScreen` showing live WebSocket connection states | **Unit:** Structural isolation checking across asynchronous queue routines |
| **5** | Dynamic teacher configuration assignment services | Custom dashboards tracking real-time user progress | **Integration:** Functional validation across full assignment task loops |
| **6** | Custom cleaning tasks for stalled background tasks | Active support for manual polling fallback loops | **Detox:** Device-side confirmation of successful async text evaluation runs |
| **7** | Real-talk content retrieval pipelines with localized daily caching | Interface tabs sorting diverse topic vectors | **Blackbox:** Output accuracy checking for cached daily datasets |
| **8** | Dynamic lesson size orchestration based on user profile settings | Adaptive indicator tracks reflecting user milestones | **Detox:** Clean execution of a core happy-path lesson sequence |
| **9** | Multi-pair expansion routing capabilities (ES → EN) | Layout toggles for changing target profiles during onboarding | **Integration:** End-to-end evaluation loops on secondary language sets |

---

## Phase 0: Foundation

### 0.1 Schema Migrations (append to `postgres.go`)

```sql
CREATE TABLE IF NOT EXISTS seed_checksums (
    file TEXT PRIMARY KEY,
    checksum TEXT NOT NULL,
    run_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS grammar_items (
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
CREATE INDEX IF NOT EXISTS idx_grammar_items_point ON grammar_items(grammar_point_id, ordinal);

CREATE TABLE IF NOT EXISTS user_grammar_mastery (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    grammar_point_id UUID NOT NULL REFERENCES grammar_points(id) ON DELETE CASCADE,
    mastery_stage INT NOT NULL DEFAULT 1,
    ease_factor NUMERIC(4,2) NOT NULL DEFAULT 2.5,
    repetitions INT NOT NULL DEFAULT 0,
    interval_days INT NOT NULL DEFAULT 0,
    next_review_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_reviewed_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, grammar_point_id)
);
CREATE INDEX IF NOT EXISTS idx_grammar_mastery_due ON user_grammar_mastery(user_id, next_review_at ASC);

CREATE TABLE IF NOT EXISTS grading_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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
CREATE INDEX IF NOT EXISTS idx_grading_jobs_status ON grading_jobs(status, next_attempt_at);
```

### 0.2 Asynchronous Background Task Polling Architecture

Distributed instances securely process work records using non-overlapping selection limits to ensure zero lock contention between operational nodes.

```go
package services

import (
	"context"
	"database/sql"
)

type GradingWorker struct {
	db *sql.DB
}

func (w *GradingWorker) PollAndExecute(ctx context.Context) error {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var jobID string
	var payload string
    
	// Target the oldest single pending task entry while bypassing items locked by adjacent nodes
	query := `
		SELECT id, payload FROM grading_jobs
		WHERE status = 'pending' AND next_attempt_at <= NOW()
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`

	err = tx.QueryRowContext(ctx, query).Scan(&jobID, &payload)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return err
	}

	_, _ = tx.ExecContext(ctx, `UPDATE grading_jobs SET status = 'processing', processing_at = NOW() WHERE id = $1`, jobID)
	if err := tx.Commit(); err != nil {
		return err
	}

	// Hand off to processing modules, record analytics objects to MinIO, and push downstream signals via WebSockets
	return nil
}
```

---

## Phase 1: Redesigned Placement Assessment

### Backend

**`placement.go` changes:**
1. `buildItemBank` queries `placement_items` table first; calls `buildPlacementFallback()` only if result is empty.
2. Improved IRT logistic formula (replace linear delta with `80*(1-P(θ,b))`).
3. 20-item structure: 8 receptive_vocab → 8 grammar_production → 4 discourse_reading.
4. `PlacementQuestion` gains `Module` and `ItemType` fields.
5. `normalizeAnswer` strips accents for A1/A2, requires them for B1+.
6. Accept typed answers: `gap_fill_type` → free-text; `sentence_reconstruction` → ordered word list.

### Mobile: `PlacementScreen.tsx` updates

Add 3 input modes driven by `question.itemType`:
- `L2_to_L1_context` → existing MCQ with context sentence highlight.
- `gap_fill_type` → `<TextInput>` below sentence-with-blank, keyboard auto-shown, submit on return.
- `sentence_reconstruction` → draggable word chips (use tappable "add to answer" chips for simpler layout structure).
- `passage_mcq` → `<ScrollView>` with passage text (word-tappable for gloss) + MCQ below.

---

## Phase 2: Grammar System

### Backend

**New `grammar_point_service.go`:**
```go
type GrammarPointService struct {
    db    *sql.DB
    rdb   *redis.Client
}

func (s *GrammarPointService) GetDueItems(ctx context.Context, userID, targetLang string, limit int) ([]GrammarDrillItem, error)
func (s *GrammarPointService) GetMicroLesson(ctx context.Context, pointID string) (*GrammarMicroLesson, error)
func (s *GrammarPointService) RecordItemAttempt(ctx context.Context, userID, itemID string, correct bool, quality int) error
```

**Session composer mapping:** replace the grammar slot stub with a real call to `GrammarPointService.GetDueItems`. Active 15-item runs are pushed directly to Redis memory caches. Mid-session interactions avoid hitting Postgres entirely.

---

## Phase 3: Full Scenario Catalogue + AI Partner

### Backend

**`scenario.go` changes:**
1. Load `scaffold_hints`, `ai_follow_up`, `success_examples` dynamically from metadata models.
2. Replace sparse `genAIReply` payload with the full structured prompt tree.
3. Add an asynchronous timeout threshold context window wrapper (3000ms max). If third-party AI links time out, the transaction cleanly disconnects and passes down an elegant UI signal error envelope rather than letting open network handlers hold application threads hostage.

---

## Phase 4: Lesson System + Grading Queue

### Backend

**`GradingQueueService`** (`grading_queue_service.go`):
* Background daemon loop scales via `FOR UPDATE SKIP LOCKED`.
* On engine calculation execution: `wsHub.SendToUser(userID, "grade_result", result)`.
* Mobile layout transitions elegantly out of shimmering states when the socket thread passes payload packets.

---

## Phase 5: Teacher Assignments

### Backend

**`TeacherAssignmentService`** (`teacher_assignment_service.go`):
* `CreateAssignment`: validates booking relationships, validates type parameters, and registers the baseline record entry.
* `SubmitAssignment`: inserts metadata states and registers an un-evaluated entry wrapper down into the background grading queue pipeline.

---

## Phase 6: Grading Queue Worker + AI Feedback Push

Wires the `GradingQueueService` tracking pool worker layers inside `main.go`.

* **Cleanup Routines:** Includes automated garbage metrics processors. Processing jobs left hanging inside state tables for over 60 seconds due to network link resets automatically clear out and re-evaluate cleanly inside upstream assignment workers.

---

## Phase 7: Real-Talk Prompts

**`RealTalkService`** (`real_talk_service.go`):
* Leverages Redis daily string key hashing maps structured around specific configurations: `"realtalk:{courseID}:{cefrLevel}:{date}"` with 24-hour expiration limits.

---

## Phase 8: Daily Loop + Session Refinements

Fixes target items inside `SessionComposerService`. The length parameters dynamically alter according to specific values registered directly inside `user_language_profiles.daily_goal_items`. Session completion events flush progress tracking arrays down into `user_grammar_mastery` via a single database transactional record update.

---

## Phase 9: ES→EN Second Language Pair

Appends secondary parsing files tracking alternatives inside the base file directory system (`seeds/es_en/`). The infrastructure checks and maps values across dynamic multi-pair routing routines safely using configuration variables throughout core internal systems.

---

## What Does NOT Change

* Auth, messaging, WebSocket hub core architecture, calls, billing — untouched.
* Teacher marketplace (booking, availability, reviews, payouts) — untouched.
* `WordMiningService` — untouched.
* `FluencyScoreService` — untouched.
* HAProxy configs, baseline Prometheus system alert setups — untouched.

---

*Document version: 3.0 — September 2026*
*Supersedes IMPLEMENTATION_PLAN_V2.md*