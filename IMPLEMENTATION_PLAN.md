# Implementation Plan: Language Learning Engine Overhaul

> **Scope**: Implements the full design from `LEARNING_DESIGN.md`. Covers backend (Go), mobile (React Native), and data. All changes are incremental — each phase ships working, independently testable code that doesn't break existing behaviour.
>
> **Multi-language design principle**: Every new table, service, and seed function is language-pair-agnostic. The Spanish course (`en→es`) remains the launch pair, but the same code path serves `es→en`, `en→fr`, `en→de`, etc. by adding a new `SeedXxxCourse` call and a `learning_pair_capabilities` row. Nothing is hardcoded to Spanish.

---

## Architecture Decisions Before We Start

### A. Migration Strategy
Migrations are Go strings in `database.Migrate()`. We keep this pattern — no external tool is introduced. Every new statement is appended to the bottom of the slice and is idempotent (`IF NOT EXISTS`, `ON CONFLICT DO NOTHING`, `DROP CONSTRAINT IF EXISTS` / `ADD CONSTRAINT`).

### B. Curriculum Seeding Strategy
`CurriculumService.SeedDefaultCourses` runs at boot and is idempotent. We refactor it so each language pair has its own `SeedXxxCourse(ctx, db)` function that:
1. Upserts the course row
2. Upserts all units (data defined in a Go file per pair, e.g. `curriculum_es.go`, `curriculum_fr.go`)
3. Upserts all lexical items, grammar points, and scenarios from the same file
4. Upserts the `learning_pair_capabilities` row

`SeedDefaultCourses` becomes a dispatcher that calls each pair's seed function. Adding a new language = adding one file and one call.

### C. Language-Pair-Agnostic Code Contract
Every service that touches curriculum data must accept `(nativeLang, targetLang string)` parameters and never reference `"es"` or `"en"` as string literals in business logic. Prompts use `languageName()` from `learning_ai.go` to convert ISO codes to human labels. All placement item banks, scenario scripts, and grammar points are scoped to a `course_id` which already carries the language pair.

### D. Teacher Assignment Architecture
New tables `teacher_assignments` and `assignment_submissions` are added. New service `TeacherAssignmentService` wraps them. New routes added to `/api/v1/teachers/*`. No changes to existing teacher routes.

### E. Placement Items — Separating Data from Code
The hardcoded `buildPlacementFallback()` function in `placement.go` is retained as a last-resort safety net but is never reached for pairs that have a seeded item bank. A new table `placement_items` stores items per language pair. `buildItemBank()` queries this table first, falls back to the hardcoded Spanish array only if the table has no rows for the pair.

---

## Phase Overview

| Phase | What Ships | Duration Estimate |
|-------|------------|-------------------|
| **0** | Database migrations, language-pair-agnostic seed architecture | 2–3 days |
| **1** | Redesigned placement test (20 items, 3 modules) | 3–4 days |
| **2** | Grammar points — full content + drill delivery | 3–4 days |
| **3** | Full scenario catalogue (12 scenarios, improved AI partner) | 4–5 days |
| **4** | Lesson system overhaul (authored steps, production grading) | 4–5 days |
| **5** | Teacher assignments (create, assign, track, grade) | 4–5 days |
| **6** | Student progress view for teachers | 2–3 days |
| **7** | Multi-language pair support (2nd pair: es→en) | 2–3 days |
| **8** | Daily practice loop refinements + real-talk prompts | 2 days |
| **9** | Mobile UI: new screens and flows | 5–7 days |

---

## Phase 0: Infrastructure and Architecture Refactoring

### 0.1 New Database Tables (append to `postgres.go`)

Add to the end of the `migrations` slice:

```go
// Phase: Learning engine expansion — placement item bank, teacher assignments,
// assignment submissions, reading passages, real-talk prompts, grammar cloze items.

// Placement item bank (replaces hardcoded fallback in placement.go)
`CREATE TABLE IF NOT EXISTS placement_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES curriculum_courses(id) ON DELETE CASCADE,
    target_language VARCHAR(10) NOT NULL,
    native_language VARCHAR(10) NOT NULL,
    cefr_level VARCHAR(2) NOT NULL CHECK (cefr_level IN ('A1','A2','B1','B2')),
    module VARCHAR(30) NOT NULL CHECK (module IN ('receptive_vocab','grammar_production','discourse_reading')),
    item_type VARCHAR(40) NOT NULL,
    prompt JSONB NOT NULL DEFAULT '{}',
    choices TEXT[] NOT NULL DEFAULT '{}',
    correct TEXT NOT NULL,
    accept_variants TEXT[] NOT NULL DEFAULT '{}',
    difficulty_value INT NOT NULL DEFAULT 500,
    is_active BOOL NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_placement_items_pair_level
    ON placement_items(course_id, cefr_level, module)`,

// Grammar cloze items (one row per drill item, linked to a grammar_point)
`ALTER TABLE grammar_points ADD COLUMN IF NOT EXISTS items JSONB NOT NULL DEFAULT '[]'`,
`ALTER TABLE grammar_points ADD COLUMN IF NOT EXISTS rule_text TEXT NOT NULL DEFAULT ''`,
`ALTER TABLE grammar_points ADD COLUMN IF NOT EXISTS common_trap TEXT NOT NULL DEFAULT ''`,
`ALTER TABLE grammar_points ADD COLUMN IF NOT EXISTS ordinal INT NOT NULL DEFAULT 0`,
`ALTER TABLE grammar_points ADD COLUMN IF NOT EXISTS is_active BOOL NOT NULL DEFAULT true`,

// Reading passages (curriculum content, linked to lesson steps)
`CREATE TABLE IF NOT EXISTS reading_passages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES curriculum_courses(id) ON DELETE CASCADE,
    unit_id UUID REFERENCES curriculum_units(id) ON DELETE SET NULL,
    cefr_level VARCHAR(2) NOT NULL CHECK (cefr_level IN ('A1','A2','B1','B2')),
    title VARCHAR(255) NOT NULL DEFAULT '',
    body TEXT NOT NULL,
    word_count INT NOT NULL DEFAULT 0,
    questions JSONB NOT NULL DEFAULT '[]',
    vocabulary_ids UUID[] NOT NULL DEFAULT '{}',
    is_active BOOL NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_reading_passages_course_level
    ON reading_passages(course_id, cefr_level)`,

// Teacher assignments (homework from teacher to student)
`CREATE TABLE IF NOT EXISTS teacher_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    booking_id UUID REFERENCES tutor_bookings(id) ON DELETE SET NULL,
    type TEXT NOT NULL CHECK (type IN ('vocabulary_push','reading','writing','scenario','mixed')),
    title TEXT NOT NULL,
    instructions TEXT NOT NULL DEFAULT '',
    content JSONB NOT NULL DEFAULT '{}',
    due_date TIMESTAMPTZ,
    target_language VARCHAR(10) NOT NULL,
    native_language VARCHAR(10) NOT NULL DEFAULT 'en',
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','in_progress','submitted','reviewed')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_teacher_assignments_teacher
    ON teacher_assignments(teacher_id, created_at DESC)`,
`CREATE INDEX IF NOT EXISTS idx_teacher_assignments_student
    ON teacher_assignments(student_id, status)`,

// Student submissions for teacher assignments
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

// Real-talk prompt bank
`CREATE TABLE IF NOT EXISTS real_talk_prompts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES curriculum_courses(id) ON DELETE CASCADE,
    target_language VARCHAR(10) NOT NULL,
    cefr_level VARCHAR(2) NOT NULL CHECK (cefr_level IN ('A1','A2','B1','B2')),
    category TEXT NOT NULL CHECK (category IN ('conversation_starter','topic_injector','opinion_phrase')),
    prompt_for_learner TEXT NOT NULL,
    target_phrase TEXT NOT NULL,
    why_useful TEXT NOT NULL DEFAULT '',
    follow_up_chunks JSONB NOT NULL DEFAULT '[]',
    is_active BOOL NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
)`,
`CREATE INDEX IF NOT EXISTS idx_real_talk_prompts_pair_level
    ON real_talk_prompts(course_id, cefr_level, category)`,

// Link scenarios to assignments
`ALTER TABLE scenario_runs ADD COLUMN IF NOT EXISTS assignment_id
    UUID REFERENCES teacher_assignments(id) ON DELETE SET NULL`,

// Source column on curriculum_lessons to distinguish teacher-authored from curriculum
`ALTER TABLE curriculum_lessons ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'curriculum'
    CHECK (source IN ('curriculum','teacher'))`,
`ALTER TABLE curriculum_lessons ADD COLUMN IF NOT EXISTS teacher_id
    UUID REFERENCES users(id) ON DELETE SET NULL`,
`ALTER TABLE curriculum_lessons ADD COLUMN IF NOT EXISTS is_active
    BOOL NOT NULL DEFAULT true`,

// Scaffold hints per scenario phase (ordered hints shown when learner is stuck)
`ALTER TABLE scenario_phases ADD COLUMN IF NOT EXISTS scaffold_hints JSONB NOT NULL DEFAULT '[]'`,
`ALTER TABLE scenario_phases ADD COLUMN IF NOT EXISTS ai_follow_up TEXT NOT NULL DEFAULT ''`,
`ALTER TABLE scenario_phases ADD COLUMN IF NOT EXISTS success_examples JSONB NOT NULL DEFAULT '[]'`,

// Improved scaffold tracking on scenario_scripts
`ALTER TABLE scenario_scripts ADD COLUMN IF NOT EXISTS scaffold_initial TEXT NOT NULL DEFAULT 'guided'
    CHECK (scaffold_initial IN ('guided','supported','independent'))`,
```

### 0.2 Curriculum Seed Refactoring

**Create `backend/internal/services/curriculum_seed.go`** — base interfaces and the dispatcher:

```go
// CurriculumCourseSeed is implemented by each language-pair seed file.
// Add a new pair = implement this interface in a new file.
type CurriculumCourseSeed interface {
    NativeLanguage() string
    TargetLanguage() string
    CourseTitle() string
    Version() string
    SupportTier() string
    Units() []curriculumUnitSeed
    LexicalItems() []lexicalSeed
    GrammarPoints() []grammarPointSeed // NEW — richer than current
    ScenarioSeeds() []scenarioSeed     // NEW — typed seed
    PlacementItems() []placementItemSeed // NEW
    RealTalkPrompts() []realTalkPromptSeed // NEW
}
```

**Create `backend/internal/services/curriculum_es_en.go`** — the EN→ES seed (extracted from `curriculum.go`'s current inline slices). This is a pure data file — no logic, just Go struct literals.

**Create `backend/internal/services/curriculum_es_en_reverse.go`** — the ES→EN seed (new pair, Phase 7).

**Update `SeedDefaultCourses`** to:
```go
func (s *CurriculumService) SeedDefaultCourses(ctx context.Context) error {
    seeds := []CurriculumCourseSeed{
        &EnEsCourseSeed{}, // Spanish for English speakers
        // &EsEnCourseSeed{}, // Phase 7: English for Spanish speakers
        // &EnFrCourseSeed{}, // future
    }
    for _, seed := range seeds {
        if err := s.seedCourse(ctx, seed); err != nil {
            return fmt.Errorf("seed %s→%s: %w", seed.NativeLanguage(), seed.TargetLanguage(), err)
        }
    }
    return nil
}
```

The `seedCourse` method does the upserts for all entities (course, units, lessons, grammar points, lexical items, scenarios, placement items, real-talk prompts, capabilities row).

### 0.3 New Model Types (add to `learning.go`)

```go
// PlacementItem is a single question in the placement item bank.
type PlacementItem struct {
    ID              string   `json:"id" db:"id"`
    CourseID        string   `json:"courseId" db:"course_id"`
    CEFRLevel       string   `json:"cefrLevel" db:"cefr_level"`
    Module          string   `json:"module" db:"module"`
    ItemType        string   `json:"itemType" db:"item_type"`
    Prompt          any      `json:"prompt" db:"prompt"`
    Choices         []string `json:"choices,omitempty" db:"choices"`
    Correct         string   `json:"-" db:"correct"`
    AcceptVariants  []string `json:"-" db:"accept_variants"`
    DifficultyValue int      `json:"difficultyValue" db:"difficulty_value"`
}

// GrammarPointSeed is used during curriculum seeding only.
type GrammarPointSeed struct {
    Slug        string
    Level       string
    Title       string
    RuleText    string
    CommonTrap  string
    Ordinal     int
    Examples    []struct{ Sentence, Translation string }
    Items       []GrammarClozeItem
}

type GrammarClozeItem struct {
    Type    string   // "cloze", "multiple_choice", "sentence_reconstruction"
    Prompt  string
    Choices []string // for MCQ only
    Correct string
    Note    string   // extra hint shown after answering wrong
}

// TeacherAssignment is a homework item from teacher to student.
type TeacherAssignment struct {
    ID             string     `json:"id" db:"id"`
    TeacherID      string     `json:"teacherId" db:"teacher_id"`
    StudentID      string     `json:"studentId" db:"student_id"`
    BookingID      string     `json:"bookingId,omitempty" db:"booking_id"`
    Type           string     `json:"type" db:"type"`
    Title          string     `json:"title" db:"title"`
    Instructions   string     `json:"instructions" db:"instructions"`
    Content        any        `json:"content" db:"content"`
    DueDate        *time.Time `json:"dueDate,omitempty" db:"due_date"`
    TargetLanguage string     `json:"targetLanguage" db:"target_language"`
    NativeLanguage string     `json:"nativeLanguage" db:"native_language"`
    Status         string     `json:"status" db:"status"`
    CreatedAt      time.Time  `json:"createdAt" db:"created_at"`
    UpdatedAt      time.Time  `json:"updatedAt" db:"updated_at"`
    // Populated on read
    Submission     *AssignmentSubmission `json:"submission,omitempty"`
}

type AssignmentSubmission struct {
    ID             string     `json:"id" db:"id"`
    AssignmentID   string     `json:"assignmentId" db:"assignment_id"`
    StudentID      string     `json:"studentId" db:"student_id"`
    Content        any        `json:"content" db:"content"`
    AIFeedback     any        `json:"aiFeedback,omitempty" db:"ai_feedback"`
    TeacherFeedback any       `json:"teacherFeedback,omitempty" db:"teacher_feedback"`
    Score          *int       `json:"score,omitempty" db:"score"`
    SubmittedAt    time.Time  `json:"submittedAt" db:"submitted_at"`
    ReviewedAt     *time.Time `json:"reviewedAt,omitempty" db:"reviewed_at"`
}

// StudentProgressSummary is returned by the teacher's progress view.
type StudentProgressSummary struct {
    Student     UserProfile                `json:"student"`
    Profile     UserLanguageProfile        `json:"profile"`
    Vocabulary  StudentVocabSummary        `json:"vocabulary"`
    Grammar     StudentGrammarSummary      `json:"grammar"`
    Activity    StudentActivitySummary     `json:"activity"`
    Assignments []TeacherAssignment        `json:"assignments"`
}

type StudentVocabSummary struct {
    Total       int            `json:"total"`
    Mastered    int            `json:"mastered"`
    DueToday    int            `json:"dueToday"`
    ByStage     map[string]int `json:"byStage"`
    WeakCards   []VocabularyCard `json:"weakCards"`
}

type StudentGrammarSummary struct {
    WeakestPoints []GrammarMasteryRow `json:"weakestPoints"`
}

type GrammarMasteryRow struct {
    ID            string  `json:"id"`
    Title         string  `json:"title"`
    ConfidencePct int     `json:"confidencePct"`
}

type StudentActivitySummary struct {
    StreakDays       int                `json:"streakDays"`
    SessionsThisWeek int                `json:"sessionsThisWeek"`
    XPThisWeek       int                `json:"xpThisWeek"`
    ScenariosCompleted int              `json:"scenariosCompleted"`
    WeeklyChart      []DailyActivityPoint `json:"weeklyChart"`
}
```

---

## Phase 1: Redesigned Placement Assessment

### Files to Change
- `backend/internal/services/placement.go` — replace `buildItemBank`, improve IRT
- `backend/internal/services/curriculum_es_en.go` — add placement item seed data
- `backend/internal/database/postgres.go` — Phase 0 migrations already add `placement_items`
- `backend/internal/models/learning.go` — extend `PlacementQuestion` for multi-module

### 1.1 Placement Service Changes

**`buildItemBank` rewrite** — query `placement_items` first, fall back to hardcoded only if empty:

```go
func (s *PlacementService) buildItemBank(ctx context.Context, targetLang, nativeLang string) ([]placementItem, error) {
    cap, err := s.capabilitiesFor(ctx, nativeLang, targetLang)
    if err != nil || cap.ActiveCourseID == "" {
        return buildPlacementFallback(), nil
    }

    rows, err := s.db.QueryContext(ctx, `
        SELECT id::text, cefr_level, module, item_type, prompt, choices, correct, accept_variants, difficulty_value
        FROM placement_items
        WHERE course_id = $1 AND is_active = true
        ORDER BY cefr_level, module, random()
    `, cap.ActiveCourseID)
    // ... scan into []placementItem, return if len > 0
    // if empty: return buildPlacementFallback()
}
```

**Improved IRT formula** (replace `updatePlacementAbility`):
```go
func updatePlacementAbility(ability, itemDifficulty float64, correct bool) float64 {
    // Logistic probability P(correct)
    p := 1.0 / (1.0 + math.Exp(-1.7*(ability-itemDifficulty)/200.0))
    delta := 80.0
    if correct {
        ability += delta * (1 - p)
    } else {
        ability -= delta * p
    }
    return math.Max(0, math.Min(1000, ability))
}
```

**20-item structure** — modify `StartPlacement` to select items in module order:
```
Module A: 8 items (receptive_vocab), adaptive by band
Module B: 8 items (grammar_production), mixed types  
Module C: 4 items (discourse_reading), 2 passages × 2 questions
```
Items within each module are sorted by CEFR then shuffled within band. The `PlacementStartResponse.TotalQuestions` becomes 20.

**Module display in `PlacementQuestion`** — add `Module` and `ItemType` fields so the mobile UI can render the right widget:
```go
type PlacementQuestion struct {
    ID        string   `json:"id"`
    Ref       string   `json:"ref"`
    Module    string   `json:"module"`    // receptive_vocab | grammar_production | discourse_reading
    ItemType  string   `json:"itemType"`  // mcq | gap_fill_type | passage_mcq | sentence_reconstruction
    CEFRLevel string   `json:"cefrLevel"`
    Prompt    any      `json:"prompt"`
    Choices   []string `json:"choices,omitempty"`
    // For passage_mcq: passage text is inside Prompt.passage
}
```

**Gap-fill answer grading** — normalize typed answers before comparison:
```go
func normalizeAnswer(s string) string {
    s = strings.ToLower(strings.TrimSpace(s))
    // For A1-A2: strip accents for forgiving matching
    // For B1+: keep accents (already done for acceptVariants)
    return s
}
```

### 1.2 Placement Item Seed Data (in `curriculum_es_en.go`)

Add a `PlacementItems() []placementItemSeed` method that returns 20+ items per CEFR band per module. Structure:

```go
func (s *EnEsCourseSeed) PlacementItems() []placementItemSeed {
    return []placementItemSeed{
        // Module A: Receptive Vocabulary
        {
            CEFRLevel:       "A1",
            Module:          "receptive_vocab",
            ItemType:        "L2_to_L1_context",
            Prompt:          map[string]any{
                "sentence": "El niño tiene cinco años.",
                "target_word": "cinco",
                "context_translation": "The child is ___ years old.",
            },
            Choices:         []string{"five", "fifteen", "yesterday", "many"},
            Correct:         "five",
            DifficultyValue: 125,
        },
        // ... 7 more A1 receptive_vocab items
        // ... 8 A2, 8 B1, 8 B2 items

        // Module B: Grammar Production
        {
            CEFRLevel:       "A1",
            Module:          "grammar_production",
            ItemType:        "gap_fill_type",
            Prompt:          map[string]any{
                "sentence_with_blank": "Yo _____ español.",
                "hint":                "to speak",
            },
            Correct:         "hablo",
            AcceptVariants:  []string{"Hablo"},
            DifficultyValue: 125,
        },
        // ... sentence_reconstruction, odd_one_out items for A1-B2

        // Module C: Discourse Reading
        {
            CEFRLevel:       "B1",
            Module:          "discourse_reading",
            ItemType:        "passage_mcq",
            Prompt:          map[string]any{
                "passage": "La semana pasada, mi jefa me pidió que presentara...",
                "question": "What did the boss ask?",
            },
            Choices:         []string{"To present a report","To go on a trip","To hire someone","To cancel a meeting"},
            Correct:         "To present a report",
            DifficultyValue: 650,
        },
    }
}
```

### 1.3 Mobile Placement Screen Changes

**`mobile/src/screens/PlacementScreen.tsx`** — handle the new module types:

- `receptive_vocab` → existing MCQ layout (4 choices, word highlighted in context sentence)
- `gap_fill_type` → new: sentence displayed with `___`, text input below
- `sentence_reconstruction` → new: word chips that can be reordered (tap to add/remove from answer row)
- `passage_mcq` → new: scrollable passage + 1–2 MCQ below (word tap shows gloss)

Add a progress bar showing module number (A / B / C) and item count within module.

---

## Phase 2: Grammar Points — Full Content and Delivery

### Files to Create/Change
- `backend/internal/services/grammar_points_service.go` — new service
- `backend/internal/services/curriculum_es_en.go` — add `GrammarPoints()` data
- `backend/internal/handlers/learning.go` — add grammar point endpoints
- `backend/internal/services/session_composer.go` — fix grammar injection

### 2.1 Grammar Point Data in `curriculum_es_en.go`

The `GrammarPoints()` method returns 60 points for Spanish covering A1–B2. Each point has:
- `Slug`, `Level`, `Title`, `RuleText`, `CommonTrap`, `Ordinal`
- `Examples` (3 sentences: affirmative, negative, interrogative)
- `Items` (≥ 4 cloze/MCQ/reconstruction drill items)

Sample from Phase 0 data structures:
```go
{
    Slug:    "ser-personal-pronouns",
    Level:   "A1",
    Ordinal: 1,
    Title:   "Ser — Personal Pronouns",
    RuleText: "Use SER for identity: name, origin, occupation, nationality.\nyo soy | tú eres | él/ella es | nosotros somos | vosotros sois | ellos son",
    CommonTrap: "Don't use SER for age or location. 'Yo soy 25 años' ✗ → 'Yo tengo 25 años' ✓",
    Examples: []struct{ Sentence, Translation string }{
        {"Yo soy estudiante.", "I am a student."},
        {"Ella no es médica.", "She is not a doctor."},
        {"¿Tú eres de Madrid?", "Are you from Madrid?"},
    },
    Items: []GrammarClozeItem{
        {Type:"cloze", Prompt:"Pedro _____ de Madrid.", Correct:"es"},
        {Type:"cloze", Prompt:"Nosotros _____ estudiantes.", Correct:"somos"},
        {Type:"multiple_choice", Prompt:"Which is correct for 'they are'?",
         Choices:[]string{"son","sos","somos","sois"}, Correct:"son"},
        {Type:"cloze", Prompt:"Yo _____ 22 años.", Correct:"tengo",
         Note:"Age uses TENER, not SER."},
    },
},
```

The full grammar point list covers (60 points across A1–B2):
- A1 (12): ser, estar, tener for age, gender agreement, question formation, tú vs. usted, present -ar/-er/-ir verbs, gustar, hay, articles (el/la/los/las), numbers 1-100, negation
- A2 (15): preterite regular, preterite ir/ser/tener, imperfect, ser vs estar extended, direct object pronouns, indirect object pronouns, reflexive verbs, future with ir+a, adjective position, comparison, demonstratives, possessives, por vs para (intro), time expressions
- B1 (18): present subjunctive (wish/doubt/emotion), simple conditional, relative clauses (que/quien/donde/cuyo), past subjunctive intro, present perfect, pluperfect, progressive aspect, passive voice, reported speech, connectors (sin embargo, aunque, a pesar de), nominalization, diminutives/augmentatives, si-clauses type 1-2, gerund vs infinitive, por vs para (advanced), impersonal se
- B2 (15): imperfect subjunctive, conditional perfect, hypothetical si-clauses type 3, past perfect subjunctive, formal written connectors, academic register, subjunctive in relative clauses, concessive clauses, causative constructions, passive alternatives (se pasiva), complex noun phrases, discourse markers, hedging, subjunctive of doubt/probability, C1-preview: absolute constructions

### 2.2 Grammar Point Service (`grammar_points_service.go`)

```go
type GrammarPointService struct { db *sql.DB }

// GetForUser returns the user's grammar mastery rows + point metadata for
// a given language pair, ordered weakest-first.
func (s *GrammarPointService) GetDuePoints(ctx context.Context, userID, targetLang string, limit int) ([]GrammarDrillItem, error)

// GetMicroLesson returns the explanation card for a grammar point (shown before
// the first drill session for that point).
func (s *GrammarPointService) GetMicroLesson(ctx context.Context, pointID string) (*GrammarMicroLesson, error)

// RecordAttempt records one cloze/MCQ attempt and updates user_grammar_mastery.
func (s *GrammarPointService) RecordAttempt(ctx context.Context, userID, pointID string, correct bool, quality int) error
```

`GrammarDrillItem` wraps a `grammar_points` row with the user's current mastery state and the next cloze item to show.

### 2.3 Session Composer Fix

The current grammar injection in `SessionComposerService` queries `grammar_points` but doesn't generate cloze items from the `items` JSONB column. Fix this so grammar session items are actual cloze prompts with answer keys, not just metadata stubs.

### 2.4 New Endpoints in `learning.go` handler

```
GET  /learning/grammar/points             — list grammar points with mastery
GET  /learning/grammar/points/:id         — get one point + micro-lesson card
POST /learning/grammar/points/:id/drill   — get next drill item for this point
```

These are served by `GrammarPointService` and registered in `main.go`.

---

## Phase 3: Full Scenario Catalogue + AI Partner Improvements

### Files to Change
- `backend/internal/services/curriculum_es_en.go` — add all 12 scenarios
- `backend/internal/services/scenario.go` — improve AI partner prompt, intent detection, scaffold levels
- `backend/internal/services/learning_ai.go` — improve `GenerateScenarioReply` prompt
- `backend/internal/models/learning.go` — extend `ScenarioPhase` with new fields

### 3.1 Scenario Data Seed

Add a `ScenarioSeeds()` method to `EnEsCourseSeed` returning all 12 scenarios from the design doc. Each scenario uses a typed `scenarioSeed` struct:

```go
type scenarioSeed struct {
    Slug             string
    Title            string
    Domain           string
    CEFRLevel        string
    CanDoStatement   string
    AIRoleName       string
    AIRoleDescription string
    AIPersonality    string   // NEW: cheerful/professional/formal/etc.
    OpeningLine      string
    MaxTurns         int
    EstimatedMinutes int
    ScaffoldInitial  string   // guided|supported|independent
    UnitSlug         string
    Phases           []scenarioPhaseSeed
}

type scenarioPhaseSeed struct {
    Ordinal         int
    Title           string
    LearnerGoal     string
    RequiredIntents []string
    ChunkBank       []map[string]string  // {text, translation}
    ScaffoldHints   []string             // NEW
    AIFollowUp      string               // NEW: AI question if stuck
    SuccessExamples []string             // NEW
}
```

The `CurriculumService.seedCourse` function iterates `ScenarioSeeds()` and upserts each scenario + phases, including the new columns from Phase 0 migrations.

### 3.2 AI Partner Prompt Improvement

Replace the current sparse payload in `scenario.go`'s `genAIReply` with the full structured prompt from the design doc:

```go
func (s *ScenarioService) genAIReply(...) models.ScenarioAIReply {
    payload := map[string]any{
        "system_role":     "scenario_partner",
        "ai_role_name":    sc.AIRoleName,
        "ai_role_description": sc.AIRoleDescription,
        "target_language": targetLang,
        "native_language": nativeLang,
        "cefr_level":      userCEFR,  // fetch from user profile
        "current_phase": map[string]any{
            "ordinal":          phase.Ordinal,
            "title":            phase.Title,
            "learner_goal":     phase.LearnerGoal,
            "required_intents": phase.RequiredIntents,
            "covered_intents":  covered,
        },
        "total_phases":    totalPhases,
        "learner_message": message,
        "scaffold_level":  scaffold,
        // Instruct the model to follow the language rules in LEARNING_DESIGN.md §9.1
    }
    // ...
}
```

Update `GenerateScenarioReply` in `learning_ai.go` to pass the improved system prompt from design doc §9.1 verbatim (parameterized with ai_role_name, ai_role_description, cefr_level, target_language, native_language).

### 3.3 Intent Detection for B1+

Add an `IntentClassifier` to `LearningAIService`:

```go
func (s *LearningAIService) ClassifyIntents(
    ctx context.Context,
    message string,
    requiredIntents []string,
    intentDefinitions map[string]string,
    cefrLevel string,
) ([]string, error)
```

In `ScenarioService.SendMessage`, choose the classifier based on CEFR:
```go
if cefrLevel >= "B1" && s.ai.HasProviders() {
    intents, err = s.ai.ClassifyIntents(ctx, message, phase.RequiredIntents, intentDefs, cefrLevel)
    if err != nil || len(intents) == 0 {
        intents = detectIntents(message, phase.RequiredIntents) // fallback
    }
} else {
    intents = detectIntents(message, phase.RequiredIntents) // keyword matching
}
```

### 3.4 Scaffold Level Progression

Add a method to `ScenarioService` that upgrades a user's scaffold level when they complete a scenario without using any hints:

```go
func (s *ScenarioService) maybeAdvanceScaffold(ctx context.Context, userID, scenarioID, cefrLevel string, hintsUsed int)
```

Store per-user scaffold preference in `user_language_profiles.metadata` (JSONB) keyed by `"scenario_scaffold_<cefr_level>"`.

---

## Phase 4: Lesson System Overhaul

### 4.1 Authored Lesson Steps

The current `LessonService.ensureSteps` lazily synthesizes steps from lexical items — it generates MCQs on the fly from whatever vocabulary is in the unit. This is fine as a fallback but we need properly authored steps per lesson.

**Approach**: Add a `seedLessonSteps(ctx, lessonID, lesson type, unitData)` function to the curriculum seeder that creates actual step rows for each lesson type:

- `vocabulary` lesson → 10 MCQ steps (one per vocabulary item, stage 1 format)
- `grammar` lesson → intro step (explanation card) + 8 cloze steps
- `reading` lesson → 1–2 reading passage steps + 3–5 MCQ comprehension steps
- `production` lesson → 5 free_recall steps + 2 production steps (use-in-sentence)
- `scenario_intro` lesson → 1 intro step that links to the unit's scenario
- `checkpoint` lesson → mixed 20-item assessment

The `curriculum_lesson_steps` table already exists with the right shape. The `type` column values to add: `production` (use-in-sentence), `reading_passage` (passage + questions).

### 4.2 Step Type Extensions

Add to the `curriculum_lesson_steps.type` check constraint:

```sql
ALTER TABLE curriculum_lesson_steps DROP CONSTRAINT IF EXISTS curriculum_lesson_steps_type_check;
ALTER TABLE curriculum_lesson_steps ADD CONSTRAINT curriculum_lesson_steps_type_check
    CHECK (type IN ('intro','mcq','cloze','free_recall','translation','listening',
                    'speaking','production','chat_prompt','explanation',
                    'reading_passage','gap_fill'));
```

### 4.3 Stage 4 Production Grading in `LessonService`

When answering a `production` step, route to AI grading:

```go
// In LessonService.AnswerStep:
if step.Type == "production" {
    result, err := s.gradeProduction(ctx, userID, step, answer, targetLang, nativeLang, cefrLevel)
    // result.correct, result.feedback, result.correctedSentence, result.errorSpans
}
```

```go
func (s *LessonService) gradeProduction(ctx context.Context, ...) (*ProductionGradeResult, error) {
    if s.ai == nil || !s.ai.HasProviders() {
        // Deterministic fallback: mark as correct if the target term appears in the answer
        term := extractTermFromStep(step)
        correct := strings.Contains(strings.ToLower(answer), strings.ToLower(term))
        return &ProductionGradeResult{Correct: correct, Feedback: "Answer recorded."}, nil
    }
    return s.ai.GradeProduction(ctx, step, answer, targetLang, nativeLang, cefrLevel)
}
```

Add `GradeProduction` to `LearningAIService` using the prompt from design doc §9.3.

### 4.4 Writing Task Grading

Add a `writing` step type. When the step type is `writing`:
- The prompt JSON contains `task_description`, `word_count_min`, `word_count_max`, `focus_grammar`, `focus_vocabulary`
- On answer: route to `LearningAIService.GradeWritingTask` (prompt from §9.4 of design doc)
- Response includes: `score`, `corrected_text`, `error_spans`, `improvement_tip`
- If AI unavailable: return `"Your answer was recorded. A teacher can provide feedback."` with `score: null`

### 4.5 New `LessonService` Method: `GradeStep`

Extract the grading logic from `AnswerStep` into a `gradeStep(ctx, step, answer, userID, targetLang, nativeLang, cefr) (*StepResult, error)` method that handles all step types with the correct grading strategy.

---

## Phase 5: Teacher Assignments

### New Files
- `backend/internal/services/teacher_assignment.go`
- `backend/internal/handlers/teacher_assignment.go`

### 5.1 `TeacherAssignmentService`

```go
type TeacherAssignmentService struct {
    db  *sql.DB
    ai  *LearningAIService
}

// CreateAssignment validates teacher-student relationship exists (via bookings),
// validates content shape by type, inserts assignment row.
func (s *TeacherAssignmentService) CreateAssignment(ctx context.Context, teacherID string, req CreateAssignmentRequest) (*TeacherAssignment, error)

// GetAssignmentsForTeacher returns all assignments the teacher created, with submission status.
func (s *TeacherAssignmentService) GetAssignmentsForTeacher(ctx context.Context, teacherID, studentID string) ([]TeacherAssignment, error)

// GetAssignmentsForStudent returns all pending/in-progress assignments for the student.
func (s *TeacherAssignmentService) GetAssignmentsForStudent(ctx context.Context, studentID string) ([]TeacherAssignment, error)

// SubmitAssignment records the student's submission, triggers AI pre-grading asynchronously.
func (s *TeacherAssignmentService) SubmitAssignment(ctx context.Context, studentID, assignmentID string, content any) (*AssignmentSubmission, error)

// ReviewAssignment lets the teacher add grades/feedback to a submission.
func (s *TeacherAssignmentService) ReviewAssignment(ctx context.Context, teacherID, assignmentID string, feedback TeacherFeedback) (*AssignmentSubmission, error)

// GetStudentProgress returns the StudentProgressSummary for a given student (teacher view).
func (s *TeacherAssignmentService) GetStudentProgress(ctx context.Context, teacherID, studentID, targetLang string) (*StudentProgressSummary, error)
```

Validation rules in `CreateAssignment`:
- Teacher must have status `approved` in `teacher_applications`
- Teacher-student relationship must exist (booking exists with status not `cancelled`)
- Content shape validated by type (vocabulary: 1–20 cards; writing: prompt required; scenario: scenario_id must exist; reading: passage + ≥1 question required)
- Due date must be in the future if provided

### 5.2 AI Pre-Grading (async)

When a writing submission arrives, enqueue an AI grade job (similar to grammar job queue pattern):

```go
// In SubmitAssignment, after inserting submission:
if assignment.Type == "writing" {
    go func() {
        result, err := s.ai.GradeWritingTask(...)
        if err == nil {
            s.updateSubmissionAIFeedback(submissionID, result)
        }
    }()
}
```

For a production app, this would be a proper queue row. For now, a goroutine is acceptable since failure just means no AI feedback (teacher can still grade manually).

### 5.3 New Handler `teacher_assignment.go`

```go
type TeacherAssignmentHandler struct {
    assignments *services.TeacherAssignmentService
}

// Teacher endpoints:
POST /teachers/assignments                    — create assignment
GET  /teachers/assignments                    — list teacher's assignments (filter by studentId)
GET  /teachers/assignments/:id                — get one assignment + submission
POST /teachers/assignments/:id/review         — submit teacher feedback + grade
GET  /teachers/students/:studentId/progress   — student progress summary

// Student endpoints:
GET  /students/assignments                    — list student's assignments
GET  /students/assignments/:id                — get one assignment
POST /students/assignments/:id/submit         — submit response
```

Register in `main.go` under the `protected` group.

---

## Phase 6: Student Progress View (Teacher Dashboard)

This phase wires up the `GetStudentProgress` service method (created in Phase 5) and builds the mobile teacher-side screen.

### 6.1 Backend

`GetStudentProgress` queries:
```sql
-- Profile
SELECT * FROM user_language_profiles WHERE user_id=$1 AND target_language=$2

-- Vocabulary summary
SELECT
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE mastery_state = 'mastered') as mastered,
    COUNT(*) FILTER (WHERE next_review <= NOW()) as due_today,
    mastery_stage, COUNT(*) as stage_count
FROM vocabulary WHERE user_id=$1 AND language=$2
GROUP BY mastery_stage

-- Grammar weaknesses
SELECT gp.id, gp.title, ugm.confidence
FROM user_grammar_mastery ugm
JOIN grammar_points gp ON gp.id = ugm.grammar_point_id
WHERE ugm.user_id=$1 AND ugm.target_language=$2
ORDER BY ugm.confidence ASC LIMIT 5

-- Activity
SELECT SUM(xp), COUNT(DISTINCT activity_date) as active_days
FROM daily_learning_stats
WHERE user_id=$1 AND target_language=$2 AND activity_date >= NOW()-INTERVAL '7 days'
```

### 6.2 Mobile: `StudentProgressScreen.tsx` (new)

Add to `MarketplaceStack` in `MainTabs.tsx`:
```
MarketplaceStack.Screen "StudentProgress" → StudentProgressScreen
  props: { studentId: string, studentName: string }
```

Screen layout:
- Header: student name + CEFR badge
- Cards: Vocabulary progress (pie by stage), Grammar weak points (bar chart), Activity (streak + weekly XP bars)
- List: Assignments with status chips (pending / submitted / reviewed) and scores
- FAB: "+ New Assignment"

Navigation: `TeacherDashboardScreen` gets a "My Students" section that lists students with recent booking history and taps through to `StudentProgressScreen`.

---

## Phase 7: Multi-Language Pair Support (ES→EN)

This phase validates the language-agnostic architecture by adding the second pair.

### 7.1 Create `backend/internal/services/curriculum_es_en_reverse.go`

Implements `CurriculumCourseSeed` for `native=es, target=en`:

```go
type EsEnCourseSeed struct{}

func (s *EsEnCourseSeed) NativeLanguage() string { return "es" }
func (s *EsEnCourseSeed) TargetLanguage() string { return "en" }
func (s *EsEnCourseSeed) CourseTitle() string { return "English for Spanish Speakers" }
// ... all data methods
```

Key differences vs EN→ES:
- All translations are `{"es": "..."}` instead of `{"en": "..."}` in lexical items
- Placement item prompts are in Spanish, target words are English
- Scenario opening lines are in English (the target language)
- Grammar points cover English grammar pain points for Spanish speakers (articles, progressive aspect, perfect tenses, conditionals, phrasal verbs)
- AI partner speaks English in scenarios

### 7.2 Update `SeedDefaultCourses`

```go
seeds := []CurriculumCourseSeed{
    &EnEsCourseSeed{},
    &EsEnCourseSeed{}, // NEW
}
```

### 7.3 Verify Language-Agnostic Code Paths

Audit every service file for hardcoded `"es"` or `"en"` string literals:
- `languageName()` in `learning_ai.go` — add all new language names
- `buildPlacementFallback()` — add a comment marking it as Spanish-only emergency fallback
- AI prompts — all use `languageName(targetLang)` already

---

## Phase 8: Daily Practice Loop + Real-Talk Prompts

### 8.1 Real-Talk Prompt System

Create `backend/internal/services/real_talk_service.go`:

```go
type RealTalkService struct {
    db  *sql.DB
    ai  *LearningAIService
}

// GetPromptsForUser returns 3 daily prompts for the user's CEFR level.
// Caches by (course_id, cefr_level, date) in Redis for 24h.
// Falls back to the seeded prompt bank if AI is unavailable.
func (s *RealTalkService) GetPromptsForUser(ctx context.Context, userID, targetLang, nativeLang string) ([]RealTalkPrompt, error)
```

The current `RealTalkPrompts` handler in `learning.go` is replaced with a call to this service.

Seed 50 prompts per CEFR level into `real_talk_prompts` via `CurriculumCourseSeed.RealTalkPrompts()`.

### 8.2 Session Composer Refinements

Fix the 3 most important issues in `SessionComposerService`:
1. Grammar items currently use placeholder prompts — fix to use the actual cloze item from `grammar_points.items` JSONB
2. Ensure the 15-item sequence matches the interleaving spec from design doc §4.8
3. Add the `DailyGoalItems` from `user_language_profiles` to control session length (not hardcoded to 15)

---

## Phase 9: Mobile UI

### New Screens

| Screen | Route | Purpose |
|--------|-------|---------|
| `PlacementScreen.tsx` | `Placement` | Handles 3-module placement (update existing) |
| `GrammarDrillScreen.tsx` | `GrammarDrill` | Grammar cloze/MCQ session |
| `GrammarPointScreen.tsx` | `GrammarPoint` | Micro-lesson card (explanation + examples) |
| `ReadingPassageScreen.tsx` | `ReadingPassage` | Passage with tappable words + MCQ |
| `WritingTaskScreen.tsx` | `WritingTask` | Writing prompt + submission + feedback view |
| `AssignmentsScreen.tsx` | `Assignments` | Student: list homework assignments |
| `AssignmentDetailScreen.tsx` | `AssignmentDetail` | Student: view + submit an assignment |
| `StudentProgressScreen.tsx` | `StudentProgress` | Teacher: view one student's progress |
| `CreateAssignmentScreen.tsx` | `CreateAssignment` | Teacher: create a new assignment |

### Navigation Changes (`MainTabs.tsx`)

Add to `LearnStackParamList`:
```ts
GrammarDrill: { grammarPointId?: string }
GrammarPoint: { grammarPointId: string }
ReadingPassage: { passageId: string; lessonAttemptId?: string }
WritingTask: { assignmentId?: string; prompt?: string; cefrLevel: string }
Assignments: undefined
AssignmentDetail: { assignmentId: string }
```

Add to `MarketplaceStackParamList`:
```ts
StudentProgress: { studentId: string; studentName: string }
CreateAssignment: { studentId: string; studentName: string; bookingId?: string }
```

### Key UI Components

**`WordGlossOverlay`** — shown when a word is tapped in a reading passage. Displays: word, translation, POS badge, CEFR badge, "Add to deck" button. Implemented as a React Native `Modal` with a bottom-sheet feel.

**`GrammarCorrectionBubble`** — shown in scenario chat when AI detects an error. Underlined span + corrected text + 1-line rule. Tappable to see full grammar point.

**`ChunkBankPanel`** — collapsible panel in scenario roleplay that shows the chunk bank. Default open for `guided`, behind a button tap for `supported`, hidden for `independent`.

**`AssignmentCard`** — used in both the student assignments list and the teacher's student progress view. Shows type icon, title, due date countdown, status chip.

---

## Implementation Order Within Each Phase

For every phase, the implementation order is:
1. **DB migrations** (append to `postgres.go`)
2. **Model types** (add to `learning.go` or `models.go`)
3. **Seed data** (add to `curriculum_es_en.go`)
4. **Service layer** (create/update service files)
5. **Handler layer** (add/update handler)
6. **Route registration** (update `main.go`)
7. **Mobile screen** (create `.tsx` file)
8. **Navigation wiring** (update `MainTabs.tsx`)
9. **API client** (update `mobile/src/services/api.ts`)
10. **Manual smoke test** against the dev seed account

---

## What Does NOT Need to Change

- The `LearningCapabilityService` — already language-agnostic, just needs a row per pair in `learning_pair_capabilities`
- The `WordMiningService` — already language-agnostic, uses `languageName()` and pair from profile
- The `SRSQueueService` / `PracticeService` / SM-2 implementation — already language-agnostic
- The `FluencyScoreService` — already language-agnostic
- Teacher marketplace (booking, availability, reviews, payouts) — no changes
- Auth, messaging, calls, billing — no changes

---

## Key Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| AI provider unavailable during scenario | Scripted fallback already in place; ensure every scenario has scaffold_hints for all phases so fallback is coherent |
| Placement item bank empty for a new language | `buildPlacementFallback()` returns a Spanish-specific fallback — for non-Spanish pairs with no items, return a generic "we're still building your course" assessment result |
| Stage 4 production grading latency | Show "Grading your answer..." spinner; 5s timeout; on timeout fall back to "Answer recorded, graded as correct" to not block the learner |
| Teacher assignment abuse (pushing to random students) | `CreateAssignment` validates `tutor_bookings` relationship; same constraint as SRS push |
| ES→EN grammar point quality | Phase 7 grammar points should be reviewed by a native Spanish speaker before enabling `full_course` tier for the pair |
| `curriculum_es_en.go` file size | If the file exceeds ~3000 lines, split into `curriculum_es_en_units.go`, `curriculum_es_en_grammar.go`, `curriculum_es_en_scenarios.go`. All implement methods on `EnEsCourseSeed`. |

---

## Testing Checkpoints

After each phase, verify these with the dev seed accounts (alice `en→es`, bob `es→en` in Phase 7):

- **Phase 0**: `go run ./cmd/server --seed-dev` succeeds; `GET /api/v1/learning/capabilities?nativeLanguage=en&targetLanguage=es` returns `full_course`
- **Phase 1**: Placement test shows 20 items across 3 modules; gap-fill items accept typed answers; result assigns correct CEFR level
- **Phase 2**: `GET /learning/grammar/points` returns 60 points; grammar session items have real cloze prompts
- **Phase 3**: All 12 scenarios listed; "Job Interview" scenario has 4 phases; AI partner replies in Spanish for `en→es`
- **Phase 4**: Lesson steps include `production` and `reading_passage` types; stage 4 grading returns feedback
- **Phase 5**: Teacher (sofia) can create writing assignment for alice; alice can submit; AI pre-grade fires; sofia can add grade
- **Phase 6**: `GET /teachers/students/:id/progress` returns vocabulary by stage, grammar weaknesses, weekly XP
- **Phase 7**: `GET /api/v1/learning/capabilities?nativeLanguage=es&targetLanguage=en` returns `full_course`; bob can take placement test in Spanish, get English scenarios
- **Phase 8**: `GET /learning/real-talk/prompts` returns 3 prompts at user's CEFR level; daily session is 15 items properly interleaved

---

*Document version: 1.0 — September 2026*  
*Implementation order can be adjusted — each phase is self-contained and independently deployable.*
