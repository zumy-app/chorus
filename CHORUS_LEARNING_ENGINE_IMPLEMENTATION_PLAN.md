# Chorus Learning Engine Technical Implementation Plan

This is a GPT handoff plan for implementing the Chorus learning system described
in `chorus_lesson_design_and_vocabulary_engine.md` and guided by the attached
Google Stitch screens.

Primary goal: when implementation is complete, a user can open Chorus on web or
mobile, see a learning dashboard with real metrics, have new words from chat
messages mined into a personal vocabulary system, practice those words through
structured learning activities, progress through an A1 -> A2 -> B1 -> B2
roadmap, receive a language score, and complete real-world AI scenarios such as
ordering coffee at a cafe.

Primary implementation strategy: build the learning engine generically around
`native_language + target_language` pairs, but ship the first fully curated
structured course for English speakers learning Spanish (`en -> es`). Other
language pairs can still use model-powered translation, grammar analysis,
chat-mined vocabulary, and SRS where quality is acceptable, but should not show
a full A1-B2 roadmap until a course is seeded and marked active for that pair.

## 0. Copy-Paste Brief For The Implementing GPT

You are implementing the Chorus learning engine in the existing `C:\dev\chorus`
repo.

Current stack:

- Backend: Go, Gin, Postgres, Redis.
- Backend migrations currently live inline in
  `backend/internal/database/postgres.go`.
- Existing domain models live in `backend/internal/models/models.go`.
- Existing services include vocabulary, grammar, message, chat, translation,
  translation queue, and grammar queue under `backend/internal/services`.
- Existing handlers live in `backend/internal/handlers`.
- Web frontend: React, Vite, Tailwind in `frontend`.
- Mobile frontend: React Native in `mobile`.
- Shared TypeScript API/types package: `packages/shared`.
- Existing web Learn page is `frontend/src/pages/Learn.tsx`.
- Existing mobile Learn screen is `mobile/src/screens/LearnScreen.tsx`.
- Current vocabulary feature exists but is shallow:
  `backend/internal/services/vocabulary.go`,
  `backend/internal/handlers/vocabulary.go`, and the `vocabulary` table.
- Current AI grammar analysis already has a durable queue pattern using
  `grammar_jobs`; follow that pattern for word mining jobs.

Implement in phases:

1. Add pair-generic learning schema and models, then seed the first full
   English-to-Spanish A1-B2 curriculum.
2. Add learning dashboard/path endpoints and wire the current Learn screens to
   real data.
3. Add message mining jobs that extract useful words and chunks from target
   language chat messages, classify them against the curriculum, and route them
   into SRS or upcoming units.
4. Upgrade vocabulary practice from a single correct/incorrect counter to a
   depth-of-processing SRS ladder: recognition, cued recall, free recall,
   production, spontaneous use.
5. Add daily session composition and lesson progression.
6. Add placement test and readiness score.
7. Add AI scenario roleplay with script phases, hints, scoring, completion, and
   vocabulary reinjection.
8. Add chat nudges and AI tutor correction hooks that connect mistakes to
   lessons, grammar mastery, and daily practice.
9. Add web/mobile screens matching the Stitch intent: placement test, learning
   hub, roadmap, scenarios, scenario roleplay, real-talk starters, AI tutor
   correction sheet, streak recovery.
10. Add tests, analytics events, and privacy controls.

Definition of done:

- `/learn` dashboard shows real daily goal, streak, due vocabulary, current
  unit, roadmap progress, scenarios, grammar progress, and fluency readiness.
- User can take or skip a placement test. Placement assigns current CEFR level
  and starting unit.
- Incoming target-language chat messages produce mined item candidates.
- User can save a word from a chat bubble; saved words are classified, deduped,
  and scheduled for practice.
- Daily practice sessions combine due SRS, current unit work, recent chat-mined
  words, and recent mistakes.
- User can complete lessons and unlock subsequent lessons/units.
- User can run the "Ordering Coffee at a Cafe" scenario and receive AI roleplay
  responses, hints, feedback, and progress credit.
- Correct scenario or chat usage can mark vocabulary as spontaneous use.
- The language score updates from curriculum progress, vocabulary depth,
  grammar mastery, scenario completion, and consistency.
- Web and mobile share types and API methods through `packages/shared`.
- Full structured roadmap support is explicit per language pair. For Phase 1,
  `en -> es` is `full_course`; unseeded pairs use `vocab_only` or
  `beta_ai_assisted` surfaces and must not display fake unit completion.

## 1. Product Behavior And User Journey

### 1.1 Onboarding And Placement

Entry points:

- New learner sees a placement test screen inspired by the Stitch placement
  mockups.
- User can choose "Start Test" or "Start From Scratch".
- Placement should take about 5 minutes and adapt across vocabulary, grammar,
  reading, and scenario comprehension.
- Result assigns:
  - `current_cefr_level`: A1, A2, B1, or B2.
  - `active_unit_id`.
  - `readiness_score`: 0-1000 progress toward the next level.
  - `skipped_units`: units marked `skipped_by_placement`.

Important distinction:

- Store `current_cefr_level` as the last validated level band.
- Store `readiness_score` as progress toward the next level.
- The Stitch example "740 / 1000, Level: A2 (Approaching B1)" should mean the
  learner is A2 with 740/1000 readiness toward B1.

### 1.2 Learning Dashboard

The Learn tab becomes the user's main learning hub. It should show:

- Current streak and streak risk state.
- Daily goal progress, for example 8/10 completed.
- Quick Drills based on recent mistakes.
- Vocabulary SRS queue, due count, and mastery count.
- Scenarios card, with the next recommended real-world scenario.
- Grammar Deep Dive card, with weakest current grammar point and confidence.
- Weekly XP/activity chart.
- Roadmap preview with current unit, completed units, locked future units.
- Fluency readiness score and current CEFR label.

Data must come from a single dashboard endpoint so web/mobile screens stay
consistent.

### 1.3 Chat Mining And Personal Corpus

When a user receives or sends a target-language message:

1. The message is stored normally.
2. Translation and grammar jobs proceed as they do today.
3. A word-mining job is enqueued if learning mining is enabled.
4. The mining job extracts high-yield words and chunks from the original
   target-language text.
5. Items are classified against the CEFR curriculum.
6. Items are deduped against the user's vocabulary.
7. Items become either:
   - accepted vocabulary cards,
   - candidate cards awaiting confirmation,
   - upcoming-unit credit,
   - completed-unit reinforcement,
   - bonus cards that do not block curriculum progress,
   - ignored low-value items.

Manual tap-to-save from chat should bypass the candidate state and create or
update a vocabulary card immediately.

### 1.4 Daily Practice Session

The user taps Quick Drills, Vocabulary, Continue Learning, or Start.

The backend creates a session that mixes:

- Due SRS items. This is the retention floor.
- One micro lesson from the active unit or one scenario task.
- Top recent mined items by teachability score.
- Recent grammar mistakes that map to current or completed grammar points.

The session should fit a 5-10 minute window.

### 1.5 Lesson Progression

Lessons are short and composable. A unit has multiple lessons:

- Vocabulary intro/review.
- Grammar micro lesson.
- Listening/reading comprehension.
- Production exercise.
- Scenario or real-talk task.
- Unit checkpoint.

Unlock rules:

- Lesson unlocks when previous lesson in the unit is complete.
- Unit unlocks when previous unit checkpoint passes or placement skips it.
- User may continue practicing older units anytime.

### 1.6 Real-World Scenario Practice

Scenario screen:

- Lists scenario cards by category and CEFR level.
- Shows progress, estimated time, difficulty, and whether new chat-mined words
  will appear.
- User can start roleplay.

Roleplay screen:

- AI plays a role such as barista, cashier, hotel clerk, or friend.
- Scenario has phases such as greeting, request, customization, payment,
  closing.
- First pass is scaffolded with 2-3 suggested chunks.
- Second pass hides suggestions by default and offers hints.
- User can type or speak.
- AI responds in character.
- System evaluates each user turn for intent, grammar, target chunk use,
  fluency, and repair strategy.

Completion requires phase coverage, not just number of turns.

### 1.7 Real Talk Starters And Chat Nudges

The "Real Talk Starters" hub suggests prompts that can be inserted into a chat.

Examples:

- "What is the first thing you usually do when you wake up?"
- "Ask Sofia about her morning routine on weekends."
- "Describe your favorite weekend activity and why it helps you relax."

Prompts should be tied to:

- active unit,
- current CEFR level,
- recent vocabulary,
- recent grammar target,
- the chat participant context where possible.

Chat nudges should be opt-in and lightweight:

- Show one nudge card at a time.
- Offer "Send to Input".
- Allow dismiss.
- Penalize repeated dismissal by reducing nudge frequency.

### 1.8 AI Tutor Corrections

When the user makes a production error in chat, SRS, or scenario:

- Highlight the error span.
- Ask for a self-correction attempt when the surface permits it.
- If skipped or incorrect, show the corrected form and a short explanation.
- Link the correction to a `grammar_point_id`.
- Increase review priority for that grammar point.
- Offer "Practice Now", which starts a targeted quick drill session.

### 1.9 Language Pair Support Strategy

Translation and grammar analysis are broad model capabilities. The app can
translate and analyze almost any language pair because the configured models are
multilingual.

Structured learning is different. A serious A1 -> A2 -> B1 -> B2 path needs
pair-aware curriculum content:

- native-language explanations,
- target-language grammar sequencing,
- CEFR-calibrated vocabulary,
- culturally natural scenarios,
- accepted-answer rubrics,
- morphology/lemmatization rules,
- pronunciation and writing-system considerations,
- placement-test items,
- level-up checkpoints.

Build the platform as a generic learning engine keyed by:

```text
native_language + target_language + course_version
```

But expose support tiers honestly:

| Support tier | Meaning | Product behavior |
|---|---|---|
| `full_course` | Curated A1-B2 course, placement, lessons, scenarios, checkpoints, roadmap | Show complete Learn dashboard, roadmap, placement, lessons, scenario progression |
| `beta_ai_assisted` | AI-generated or lightly reviewed lesson variants exist, but not enough for validated progression | Show dashboard, SRS, scenarios marked beta, no hard fluency claims |
| `vocab_only` | No seeded course, but chat mining, translation, grammar analysis, and SRS are available | Show vocabulary/grammar dashboard and "course coming soon"; hide roadmap/checkpoints |
| `disabled` | Pair is unavailable or unsuitable for current learning features | Show translation/chat only |

Phase 1 should ship `en -> es` as `full_course`. The engineering foundation
should support any pair, but product quality should only promise structured
fluency progression for pairs with seeded curriculum.

This avoids two bad outcomes:

- hardcoding Spanish in a way that blocks future language pairs,
- pretending every pair has a real course just because the model can translate.

## 2. Existing System Baseline

Use the current implementation instead of replacing it.

Existing backend:

- `messages` table already stores chat messages, language, translations, and
  timestamps.
- `vocabulary` table already stores term, language, translation, definition,
  source context, review count, next review, interval.
- `VocabularyService` currently saves a word and records simple SRS.
- `GrammarService` already provides regex and AI-based analysis with provider
  fallback.
- `GrammarQueueService` already demonstrates a durable Postgres job plus Redis
  pub/sub worker pattern.
- `TranslationQueueService` already demonstrates async queue completion and
  WebSocket fanout.

Existing web:

- `frontend/src/pages/Learn.tsx` currently renders placeholder dashboard
  metrics.
- `frontend/src/components/MessageBubble.tsx` already has manual word-save,
  translation, grammar analysis, and deep dive actions.
- `frontend/src/store/index.ts` already handles chat state, translation blocks,
  grammar jobs, and WebSocket updates.

Existing mobile:

- `mobile/src/screens/LearnScreen.tsx` currently renders placeholder dashboard
  metrics.
- `mobile/src/screens/ChatScreen.tsx` supports messages, translation, and a
  simple Sparky bottom sheet.
- `mobile/src/components/MainTabs.tsx` has Learn as one of the primary tabs.

Existing shared API/types:

- `packages/shared/src/types.ts`.
- `packages/shared/src/api.ts`.
- Web and mobile should consume new learning methods from here.

## 3. Core Architecture

Add these backend services:

- `CurriculumService`: read/seed CEFR courses by native/target language pair,
  units, lessons, grammar points, lexical items, scenario scripts.
- `LearningCapabilityService`: resolve whether a pair is `full_course`,
  `beta_ai_assisted`, `vocab_only`, or `disabled`.
- `LearningProfileService`: user language profile, placement state, active
  unit, preferences.
- `LearningDashboardService`: aggregate dashboard/path metrics.
- `WordMiningQueueService`: durable extraction/classification jobs for
  messages and scenarios.
- `WordMiningService`: extract, normalize, classify, route, dedupe mined items.
- `PracticeService`: create SRS drills, grade answers, update item mastery.
- `LessonService`: manage lesson attempts and unit progression.
- `SessionComposerService`: build daily sessions from SRS, lessons, mistakes,
  and mined words.
- `PlacementService`: adaptive placement test and starting point assignment.
- `ScenarioService`: start roleplay runs, advance phases, grade turns, complete
  scenarios.
- `FluencyScoreService`: compute readiness score and snapshots.
- `StreakService`: daily goal, streak, streak recovery.
- `RealTalkService`: generate and rank chat prompts/nudges.

Suggested file layout:

```text
backend/internal/services/curriculum.go
backend/internal/services/curriculum_seed.go
backend/internal/services/learning_capability.go
backend/internal/services/learning_profile.go
backend/internal/services/learning_dashboard.go
backend/internal/services/word_mining.go
backend/internal/services/word_mining_queue.go
backend/internal/services/practice.go
backend/internal/services/lesson.go
backend/internal/services/session_composer.go
backend/internal/services/placement.go
backend/internal/services/scenario.go
backend/internal/services/fluency_score.go
backend/internal/services/streak.go
backend/internal/services/real_talk.go
backend/internal/handlers/learning.go
backend/internal/handlers/scenario.go
```

Add shared client groups:

```text
packages/shared/src/types.ts
packages/shared/src/api.ts
```

Add web surfaces:

```text
frontend/src/pages/Learn.tsx
frontend/src/pages/Placement.tsx
frontend/src/pages/LessonSession.tsx
frontend/src/pages/VocabularyReview.tsx
frontend/src/pages/Scenarios.tsx
frontend/src/pages/ScenarioRoleplay.tsx
frontend/src/pages/LearningRoadmap.tsx
frontend/src/components/learning/*
frontend/src/components/chat/RealTalkNudge.tsx
```

Add mobile surfaces:

```text
mobile/src/screens/LearnScreen.tsx
mobile/src/screens/PlacementScreen.tsx
mobile/src/screens/LessonSessionScreen.tsx
mobile/src/screens/VocabularyReviewScreen.tsx
mobile/src/screens/ScenariosScreen.tsx
mobile/src/screens/ScenarioRoleplayScreen.tsx
mobile/src/screens/LearningRoadmapScreen.tsx
mobile/src/components/learning/*
```

### 3.1 Language-Specific Learning Adapters

Do not hardcode Spanish throughout the product, but do add a Spanish adapter for
the first full course.

Introduce a small internal interface:

```go
type LearningLanguageAdapter interface {
	TargetLanguage() string
	NormalizeTerm(term string) string
	Lemmatize(term string, context string) (lemma string, confidence float64)
	Stopwords() map[string]bool
	AcceptedEquivalentAnswers(promptType string, answer string) []string
	ScoreTypedAnswer(expected []string, answer string) PracticeScore
	DetectLikelyGrammarTags(text string) []string
	ScriptConstraints(level string) ScenarioLanguageConstraints
}
```

Implement:

- `SpanishLearningAdapter` for `es`.
- `GenericLearningAdapter` fallback for all other target languages.

The generic adapter is enough for vocabulary capture, translation-backed
definitions, simple SRS, and AI grammar explanations. It is not enough for
validated CEFR progression. Full roadmap, placement, checkpoints, and fluency
claims require a seeded `curriculum_courses` row plus adapter/rubric quality
for that target language and native-language explanation pair.

For native-language differences, keep explanations and prompts in course data
where possible:

```text
English speaker learning Spanish:
  ser vs estar explanation compares both to English "to be".

Spanish speaker learning French:
  etre/avoir, gender, pronouns, and false friends need Spanish-language
  explanations and different examples.
```

## 4. Data Model

All database changes should be additive and idempotent in
`backend/internal/database/postgres.go`, matching the current migration style.
Use `CREATE TABLE IF NOT EXISTS`, `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`,
and named indexes.

### 4.0 Learning Pair Capabilities

Purpose: declare what level of learning support exists for each
native-language/target-language pair. This is separate from translation and
grammar support, which are model-powered and broadly available.

For Phase 1, seed:

```text
native_language = en
target_language = es
support_tier = full_course
```

For unseeded pairs, the backend can return a computed fallback of `vocab_only`
when translation, grammar analysis, and word mining are available.

```sql
CREATE TABLE IF NOT EXISTS learning_pair_capabilities (
  native_language VARCHAR(10) NOT NULL,
  target_language VARCHAR(10) NOT NULL,
  support_tier VARCHAR(30) NOT NULL DEFAULT 'vocab_only'
    CHECK (support_tier IN ('full_course','beta_ai_assisted','vocab_only','disabled')),
  active_course_id UUID,
  placement_enabled BOOLEAN NOT NULL DEFAULT false,
  roadmap_enabled BOOLEAN NOT NULL DEFAULT false,
  scenarios_enabled BOOLEAN NOT NULL DEFAULT false,
  srs_enabled BOOLEAN NOT NULL DEFAULT true,
  mining_enabled BOOLEAN NOT NULL DEFAULT true,
  grammar_feedback_enabled BOOLEAN NOT NULL DEFAULT true,
  quality_notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (native_language, target_language)
);

CREATE INDEX IF NOT EXISTS idx_learning_pair_capabilities_tier
  ON learning_pair_capabilities(support_tier);
```

If a migration order allows it, add a foreign key from `active_course_id` to
`curriculum_courses(id)` after `curriculum_courses` exists. Do not block Phase 1
on this; the service can validate the referenced course in code.

### 4.1 User Learning Profile

Purpose: one row per user and target language.

```sql
CREATE TABLE IF NOT EXISTS user_language_profiles (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_language VARCHAR(10) NOT NULL,
  native_language VARCHAR(10) NOT NULL DEFAULT 'en',
  current_cefr_level VARCHAR(2) NOT NULL DEFAULT 'A1'
    CHECK (current_cefr_level IN ('A1','A2','B1','B2')),
  readiness_score INTEGER NOT NULL DEFAULT 0
    CHECK (readiness_score >= 0 AND readiness_score <= 1000),
  active_course_id UUID,
  active_unit_id UUID,
  placement_status VARCHAR(20) NOT NULL DEFAULT 'not_started'
    CHECK (placement_status IN ('not_started','in_progress','completed','skipped')),
  primary_goal VARCHAR(30) NOT NULL DEFAULT 'conversational_fluency'
    CHECK (primary_goal IN ('conversational_fluency','structured_study','travel','work','exam_prep')),
  daily_goal_items INTEGER NOT NULL DEFAULT 10,
  mining_enabled BOOLEAN NOT NULL DEFAULT true,
  nudges_enabled BOOLEAN NOT NULL DEFAULT true,
  scenario_hints_enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, target_language)
);

CREATE INDEX IF NOT EXISTS idx_user_language_profiles_active_unit
  ON user_language_profiles(active_unit_id);
```

Add foreign keys to `active_course_id` and `active_unit_id` after curriculum
tables are created, or keep them unenforced in the first migration pass to avoid
ordering issues in inline migrations.

### 4.2 Curriculum Course

Purpose: versioned language-pair course container. A course is not just
"Spanish"; it is "Spanish for English speakers", "French for Spanish speakers",
and so on. This matters because explanations, translation choices, contrastive
grammar notes, and common interference errors are native-language-specific.

```sql
CREATE TABLE IF NOT EXISTS curriculum_courses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  target_language VARCHAR(10) NOT NULL,
  native_language VARCHAR(10) NOT NULL DEFAULT 'en',
  title VARCHAR(255) NOT NULL,
  version VARCHAR(50) NOT NULL DEFAULT 'v1',
  is_active BOOLEAN NOT NULL DEFAULT true,
  support_tier VARCHAR(30) NOT NULL DEFAULT 'full_course'
    CHECK (support_tier IN ('full_course','beta_ai_assisted')),
  metadata JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(target_language, native_language, version)
);

CREATE INDEX IF NOT EXISTS idx_curriculum_courses_active
  ON curriculum_courses(target_language, native_language, is_active);
```

### 4.3 Curriculum Units

Purpose: roadmap nodes from A1 through B2.

```sql
CREATE TABLE IF NOT EXISTS curriculum_units (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  course_id UUID NOT NULL REFERENCES curriculum_courses(id) ON DELETE CASCADE,
  cefr_level VARCHAR(2) NOT NULL CHECK (cefr_level IN ('A1','A2','B1','B2')),
  ordinal INTEGER NOT NULL,
  slug VARCHAR(120) NOT NULL,
  title VARCHAR(255) NOT NULL,
  can_do_statement TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  estimated_minutes INTEGER NOT NULL DEFAULT 30,
  checkpoint_required BOOLEAN NOT NULL DEFAULT true,
  metadata JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(course_id, ordinal),
  UNIQUE(course_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_curriculum_units_course_level
  ON curriculum_units(course_id, cefr_level, ordinal);
```

### 4.4 Grammar Points

Purpose: canonical grammar inventory tied to CEFR.

```sql
CREATE TABLE IF NOT EXISTS grammar_points (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  course_id UUID NOT NULL REFERENCES curriculum_courses(id) ON DELETE CASCADE,
  unit_id UUID REFERENCES curriculum_units(id) ON DELETE SET NULL,
  slug VARCHAR(120) NOT NULL,
  cefr_level VARCHAR(2) NOT NULL CHECK (cefr_level IN ('A1','A2','B1','B2')),
  title VARCHAR(255) NOT NULL,
  short_explanation TEXT NOT NULL DEFAULT '',
  examples JSONB NOT NULL DEFAULT '[]',
  prerequisites TEXT[] NOT NULL DEFAULT '{}',
  metadata JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(course_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_grammar_points_unit
  ON grammar_points(unit_id);
```

### 4.5 Lexical Items

Purpose: canonical vocabulary and phrase inventory for course coverage.

```sql
CREATE TABLE IF NOT EXISTS lexical_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  course_id UUID NOT NULL REFERENCES curriculum_courses(id) ON DELETE CASCADE,
  unit_id UUID REFERENCES curriculum_units(id) ON DELETE SET NULL,
  language VARCHAR(10) NOT NULL,
  lemma VARCHAR(255) NOT NULL,
  display_text VARCHAR(255) NOT NULL,
  part_of_speech VARCHAR(40) NOT NULL DEFAULT 'unknown',
  cefr_level VARCHAR(2) NOT NULL CHECK (cefr_level IN ('A1','A2','B1','B2')),
  translations JSONB NOT NULL DEFAULT '{}',
  forms JSONB NOT NULL DEFAULT '{}',
  tags TEXT[] NOT NULL DEFAULT '{}',
  frequency_rank INTEGER,
  is_chunk BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(course_id, language, lemma, part_of_speech)
);

CREATE INDEX IF NOT EXISTS idx_lexical_items_unit
  ON lexical_items(unit_id);
CREATE INDEX IF NOT EXISTS idx_lexical_items_lookup
  ON lexical_items(course_id, language, lemma);
```

### 4.6 Lessons And Lesson Steps

Purpose: deterministic lesson structure with JSON content.

```sql
CREATE TABLE IF NOT EXISTS curriculum_lessons (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  unit_id UUID NOT NULL REFERENCES curriculum_units(id) ON DELETE CASCADE,
  ordinal INTEGER NOT NULL,
  slug VARCHAR(120) NOT NULL,
  type VARCHAR(40) NOT NULL
    CHECK (type IN ('vocabulary','grammar','reading','listening','production','scenario_intro','checkpoint')),
  title VARCHAR(255) NOT NULL,
  objective TEXT NOT NULL DEFAULT '',
  estimated_minutes INTEGER NOT NULL DEFAULT 5,
  metadata JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(unit_id, ordinal),
  UNIQUE(unit_id, slug)
);

CREATE TABLE IF NOT EXISTS curriculum_lesson_steps (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lesson_id UUID NOT NULL REFERENCES curriculum_lessons(id) ON DELETE CASCADE,
  ordinal INTEGER NOT NULL,
  type VARCHAR(40) NOT NULL
    CHECK (type IN (
      'intro','mcq','cloze','free_recall','translation','listening',
      'speaking','production','chat_prompt','explanation'
    )),
  prompt JSONB NOT NULL DEFAULT '{}',
  answer_key JSONB NOT NULL DEFAULT '{}',
  content_refs JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(lesson_id, ordinal)
);

CREATE INDEX IF NOT EXISTS idx_curriculum_lessons_unit
  ON curriculum_lessons(unit_id, ordinal);
```

Step JSON examples:

```json
{
  "type": "mcq",
  "prompt": {
    "text": "Translate this phrase",
    "source": "The weekend",
    "choices": ["La semana", "El fin de semana", "El mes", "El año"]
  },
  "answer_key": {
    "correct": "El fin de semana"
  },
  "content_refs": {
    "lexical_item_ids": ["..."]
  }
}
```

```json
{
  "type": "cloze",
  "prompt": {
    "sentence": "Yo ____ muy cansado hoy.",
    "hint": "Temporary condition"
  },
  "answer_key": {
    "accepted": ["estoy"],
    "explanation": "Use estar for temporary physical states."
  },
  "content_refs": {
    "grammar_point_ids": ["estar-vs-ser-temporary-state"]
  }
}
```

### 4.7 Scenario Scripts

Purpose: real-world roleplay scripts made of phases.

```sql
CREATE TABLE IF NOT EXISTS scenario_scripts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  course_id UUID NOT NULL REFERENCES curriculum_courses(id) ON DELETE CASCADE,
  unit_id UUID REFERENCES curriculum_units(id) ON DELETE SET NULL,
  slug VARCHAR(120) NOT NULL,
  title VARCHAR(255) NOT NULL,
  domain VARCHAR(80) NOT NULL,
  cefr_level VARCHAR(2) NOT NULL CHECK (cefr_level IN ('A1','A2','B1','B2')),
  can_do_statement TEXT NOT NULL,
  ai_role_name VARCHAR(120) NOT NULL,
  ai_role_description TEXT NOT NULL,
  opening_line TEXT NOT NULL,
  max_turns INTEGER NOT NULL DEFAULT 10,
  estimated_minutes INTEGER NOT NULL DEFAULT 5,
  completion_criteria JSONB NOT NULL DEFAULT '{}',
  metadata JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(course_id, slug)
);

CREATE TABLE IF NOT EXISTS scenario_phases (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  scenario_id UUID NOT NULL REFERENCES scenario_scripts(id) ON DELETE CASCADE,
  ordinal INTEGER NOT NULL,
  title VARCHAR(255) NOT NULL,
  learner_goal TEXT NOT NULL,
  required_intents TEXT[] NOT NULL DEFAULT '{}',
  chunk_bank JSONB NOT NULL DEFAULT '[]',
  new_lexical_item_ids UUID[] NOT NULL DEFAULT '{}',
  grammar_point_ids UUID[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(scenario_id, ordinal)
);

CREATE INDEX IF NOT EXISTS idx_scenario_scripts_course_level
  ON scenario_scripts(course_id, cefr_level);
```

### 4.8 User Progress

```sql
CREATE TABLE IF NOT EXISTS user_unit_progress (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  unit_id UUID NOT NULL REFERENCES curriculum_units(id) ON DELETE CASCADE,
  target_language VARCHAR(10) NOT NULL,
  status VARCHAR(30) NOT NULL DEFAULT 'locked'
    CHECK (status IN ('locked','available','in_progress','completed','skipped_by_placement')),
  progress_pct INTEGER NOT NULL DEFAULT 0 CHECK (progress_pct >= 0 AND progress_pct <= 100),
  competency_score INTEGER NOT NULL DEFAULT 0 CHECK (competency_score >= 0 AND competency_score <= 1000),
  lessons_completed INTEGER NOT NULL DEFAULT 0,
  checkpoint_score INTEGER,
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, unit_id)
);

CREATE INDEX IF NOT EXISTS idx_user_unit_progress_user_status
  ON user_unit_progress(user_id, target_language, status);

CREATE TABLE IF NOT EXISTS user_lesson_attempts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  lesson_id UUID NOT NULL REFERENCES curriculum_lessons(id) ON DELETE CASCADE,
  target_language VARCHAR(10) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'in_progress'
    CHECK (status IN ('in_progress','completed','abandoned')),
  score INTEGER NOT NULL DEFAULT 0 CHECK (score >= 0 AND score <= 1000),
  correct_count INTEGER NOT NULL DEFAULT 0,
  total_count INTEGER NOT NULL DEFAULT 0,
  started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMP,
  metadata JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_user_lesson_attempts_user
  ON user_lesson_attempts(user_id, target_language, started_at DESC);

CREATE TABLE IF NOT EXISTS lesson_step_results (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  attempt_id UUID NOT NULL REFERENCES user_lesson_attempts(id) ON DELETE CASCADE,
  step_id UUID REFERENCES curriculum_lesson_steps(id) ON DELETE SET NULL,
  user_answer JSONB NOT NULL DEFAULT '{}',
  correct BOOLEAN NOT NULL DEFAULT false,
  score INTEGER NOT NULL DEFAULT 0 CHECK (score >= 0 AND score <= 1000),
  feedback JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 4.9 Vocabulary Extensions

Keep the existing `vocabulary` table for compatibility. Add fields that turn it
into a real card model.

```sql
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS lemma VARCHAR(255);
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS normalized_term VARCHAR(255);
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS part_of_speech VARCHAR(40) DEFAULT 'unknown';
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS is_chunk BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS source_type VARCHAR(30) NOT NULL DEFAULT 'chat'
  CHECK (source_type IN ('chat','manual','scenario','lesson','import'));
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS source_message_id UUID REFERENCES messages(id) ON DELETE SET NULL;
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS source_scenario_run_id UUID;
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS cefr_level VARCHAR(2)
  CHECK (cefr_level IN ('A1','A2','B1','B2'));
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS curriculum_lexical_item_id UUID REFERENCES lexical_items(id) ON DELETE SET NULL;
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS curriculum_unit_id UUID REFERENCES curriculum_units(id) ON DELETE SET NULL;
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS route_status VARCHAR(30) NOT NULL DEFAULT 'bonus'
  CHECK (route_status IN ('upcoming_unit','completed_unit','current_unit','bonus','ignored'));
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS mastery_stage INTEGER NOT NULL DEFAULT 1
  CHECK (mastery_stage >= 1 AND mastery_stage <= 5);
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS mastery_state VARCHAR(30) NOT NULL DEFAULT 'new'
  CHECK (mastery_state IN ('new','learning','reviewing','mastered','leech','ignored'));
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS ease_factor NUMERIC(4,2) NOT NULL DEFAULT 2.50;
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS lapses INTEGER NOT NULL DEFAULT 0;
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS stage_success_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS production_success_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS spontaneous_use_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS teachability_score NUMERIC(5,2) NOT NULL DEFAULT 0;
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS confidence NUMERIC(4,3) NOT NULL DEFAULT 0.5;
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE vocabulary ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_vocabulary_user_stage_due
  ON vocabulary(user_id, language, mastery_stage, next_review);
CREATE INDEX IF NOT EXISTS idx_vocabulary_curriculum_unit
  ON vocabulary(user_id, curriculum_unit_id);
CREATE INDEX IF NOT EXISTS idx_vocabulary_normalized
  ON vocabulary(user_id, language, normalized_term);
```

Add practice attempt history:

```sql
CREATE TABLE IF NOT EXISTS vocabulary_practice_attempts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vocabulary_id UUID NOT NULL REFERENCES vocabulary(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_language VARCHAR(10) NOT NULL,
  stage INTEGER NOT NULL CHECK (stage >= 1 AND stage <= 5),
  activity_type VARCHAR(40) NOT NULL,
  prompt JSONB NOT NULL DEFAULT '{}',
  answer JSONB NOT NULL DEFAULT '{}',
  correct BOOLEAN NOT NULL DEFAULT false,
  quality INTEGER NOT NULL DEFAULT 0 CHECK (quality >= 0 AND quality <= 5),
  latency_ms INTEGER,
  source_session_id UUID,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_vocab_attempts_vocab_created
  ON vocabulary_practice_attempts(vocabulary_id, created_at DESC);
```

Add source aggregation:

```sql
CREATE TABLE IF NOT EXISTS vocabulary_sources (
  vocabulary_id UUID NOT NULL REFERENCES vocabulary(id) ON DELETE CASCADE,
  source_type VARCHAR(30) NOT NULL,
  source_id UUID,
  sentence TEXT NOT NULL DEFAULT '',
  seen_count INTEGER NOT NULL DEFAULT 1,
  first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (vocabulary_id, source_type, source_id)
);
```

### 4.10 Mined Items And Jobs

```sql
CREATE TABLE IF NOT EXISTS word_mining_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
  message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
  source_type VARCHAR(30) NOT NULL DEFAULT 'chat'
    CHECK (source_type IN ('chat','scenario','lesson')),
  source_text TEXT NOT NULL,
  source_language VARCHAR(10) NOT NULL,
  native_language VARCHAR(10) NOT NULL DEFAULT 'en',
  status VARCHAR(20) NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending','processing','done','failed','ignored')),
  result JSONB,
  attempts INTEGER NOT NULL DEFAULT 0,
  last_error TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  next_attempt_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  processing_at TIMESTAMP,
  completed_at TIMESTAMP,
  UNIQUE(user_id, message_id)
);

CREATE INDEX IF NOT EXISTS idx_word_mining_jobs_status
  ON word_mining_jobs(status, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_word_mining_jobs_user
  ON word_mining_jobs(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS mined_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  job_id UUID REFERENCES word_mining_jobs(id) ON DELETE SET NULL,
  chat_id UUID REFERENCES chats(id) ON DELETE SET NULL,
  message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
  source_type VARCHAR(30) NOT NULL DEFAULT 'chat',
  surface_text VARCHAR(255) NOT NULL,
  lemma VARCHAR(255) NOT NULL,
  normalized_text VARCHAR(255) NOT NULL,
  language VARCHAR(10) NOT NULL,
  part_of_speech VARCHAR(40) NOT NULL DEFAULT 'unknown',
  translation VARCHAR(500) NOT NULL DEFAULT '',
  definition TEXT NOT NULL DEFAULT '',
  context_sentence TEXT NOT NULL DEFAULT '',
  text_span JSONB NOT NULL DEFAULT '{}',
  cefr_level VARCHAR(2) CHECK (cefr_level IN ('A1','A2','B1','B2')),
  confidence NUMERIC(4,3) NOT NULL DEFAULT 0.5,
  teachability_score NUMERIC(5,2) NOT NULL DEFAULT 0,
  is_chunk BOOLEAN NOT NULL DEFAULT false,
  is_proper_noun BOOLEAN NOT NULL DEFAULT false,
  grammar_tags TEXT[] NOT NULL DEFAULT '{}',
  curriculum_lexical_item_id UUID REFERENCES lexical_items(id) ON DELETE SET NULL,
  curriculum_unit_id UUID REFERENCES curriculum_units(id) ON DELETE SET NULL,
  route_status VARCHAR(30) NOT NULL DEFAULT 'bonus'
    CHECK (route_status IN ('upcoming_unit','completed_unit','current_unit','bonus','ignored')),
  status VARCHAR(30) NOT NULL DEFAULT 'candidate'
    CHECK (status IN ('candidate','auto_added','accepted','ignored','merged')),
  route_reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_mined_items_user_status
  ON mined_items(user_id, language, status, teachability_score DESC);
CREATE INDEX IF NOT EXISTS idx_mined_items_message
  ON mined_items(message_id);
CREATE INDEX IF NOT EXISTS idx_mined_items_lookup
  ON mined_items(user_id, language, normalized_text);
```

### 4.11 Grammar Mastery

```sql
CREATE TABLE IF NOT EXISTS user_grammar_mastery (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  grammar_point_id UUID NOT NULL REFERENCES grammar_points(id) ON DELETE CASCADE,
  target_language VARCHAR(10) NOT NULL,
  confidence NUMERIC(4,3) NOT NULL DEFAULT 0.3,
  seen_count INTEGER NOT NULL DEFAULT 0,
  correct_count INTEGER NOT NULL DEFAULT 0,
  error_count INTEGER NOT NULL DEFAULT 0,
  production_success_count INTEGER NOT NULL DEFAULT 0,
  next_review_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_error_text TEXT,
  last_seen_at TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, grammar_point_id)
);

CREATE INDEX IF NOT EXISTS idx_user_grammar_mastery_due
  ON user_grammar_mastery(user_id, target_language, next_review_at);
```

### 4.12 Daily Sessions

```sql
CREATE TABLE IF NOT EXISTS learning_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_language VARCHAR(10) NOT NULL,
  mode VARCHAR(40) NOT NULL DEFAULT 'daily'
    CHECK (mode IN ('daily','quick_drill','vocabulary','lesson','scenario','grammar','streak_recovery')),
  status VARCHAR(20) NOT NULL DEFAULT 'in_progress'
    CHECK (status IN ('in_progress','completed','abandoned')),
  source_unit_id UUID REFERENCES curriculum_units(id) ON DELETE SET NULL,
  source_lesson_id UUID REFERENCES curriculum_lessons(id) ON DELETE SET NULL,
  planned_item_count INTEGER NOT NULL DEFAULT 0,
  completed_item_count INTEGER NOT NULL DEFAULT 0,
  score INTEGER NOT NULL DEFAULT 0,
  xp_awarded INTEGER NOT NULL DEFAULT 0,
  started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMP,
  metadata JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS learning_session_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES learning_sessions(id) ON DELETE CASCADE,
  ordinal INTEGER NOT NULL,
  item_type VARCHAR(40) NOT NULL
    CHECK (item_type IN ('vocabulary','grammar','lesson_step','scenario_prompt','reflection')),
  vocabulary_id UUID REFERENCES vocabulary(id) ON DELETE SET NULL,
  grammar_point_id UUID REFERENCES grammar_points(id) ON DELETE SET NULL,
  lesson_step_id UUID REFERENCES curriculum_lesson_steps(id) ON DELETE SET NULL,
  payload JSONB NOT NULL DEFAULT '{}',
  result JSONB,
  status VARCHAR(20) NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending','answered','skipped')),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(session_id, ordinal)
);

CREATE INDEX IF NOT EXISTS idx_learning_sessions_user_created
  ON learning_sessions(user_id, target_language, started_at DESC);
```

### 4.13 Scenario Runs

```sql
CREATE TABLE IF NOT EXISTS scenario_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  scenario_id UUID NOT NULL REFERENCES scenario_scripts(id) ON DELETE CASCADE,
  target_language VARCHAR(10) NOT NULL,
  native_language VARCHAR(10) NOT NULL DEFAULT 'en',
  status VARCHAR(20) NOT NULL DEFAULT 'in_progress'
    CHECK (status IN ('in_progress','completed','abandoned','failed')),
  scaffold_level VARCHAR(20) NOT NULL DEFAULT 'guided'
    CHECK (scaffold_level IN ('guided','hinted','unscaffolded')),
  current_phase_ordinal INTEGER NOT NULL DEFAULT 1,
  phase_scores JSONB NOT NULL DEFAULT '{}',
  covered_intents TEXT[] NOT NULL DEFAULT '{}',
  score INTEGER NOT NULL DEFAULT 0 CHECK (score >= 0 AND score <= 1000),
  xp_awarded INTEGER NOT NULL DEFAULT 0,
  started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMP,
  metadata JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS scenario_turns (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id UUID NOT NULL REFERENCES scenario_runs(id) ON DELETE CASCADE,
  ordinal INTEGER NOT NULL,
  speaker VARCHAR(20) NOT NULL CHECK (speaker IN ('user','ai','system')),
  text TEXT NOT NULL,
  translation TEXT NOT NULL DEFAULT '',
  phase_ordinal INTEGER NOT NULL,
  evaluation JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(run_id, ordinal)
);

CREATE INDEX IF NOT EXISTS idx_scenario_runs_user
  ON scenario_runs(user_id, target_language, started_at DESC);
```

### 4.14 Placement

```sql
CREATE TABLE IF NOT EXISTS placement_attempts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_language VARCHAR(10) NOT NULL,
  native_language VARCHAR(10) NOT NULL DEFAULT 'en',
  status VARCHAR(20) NOT NULL DEFAULT 'in_progress'
    CHECK (status IN ('in_progress','completed','abandoned')),
  estimated_cefr VARCHAR(2) CHECK (estimated_cefr IN ('A1','A2','B1','B2')),
  readiness_score INTEGER NOT NULL DEFAULT 0,
  ability_estimate NUMERIC(5,2) NOT NULL DEFAULT 0,
  started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMP,
  metadata JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS placement_responses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  attempt_id UUID NOT NULL REFERENCES placement_attempts(id) ON DELETE CASCADE,
  item_ref VARCHAR(120) NOT NULL,
  item_type VARCHAR(40) NOT NULL,
  cefr_level VARCHAR(2) NOT NULL CHECK (cefr_level IN ('A1','A2','B1','B2')),
  prompt JSONB NOT NULL DEFAULT '{}',
  user_answer JSONB NOT NULL DEFAULT '{}',
  correct BOOLEAN NOT NULL DEFAULT false,
  score INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 4.15 Activity, Streaks, Score Snapshots

```sql
CREATE TABLE IF NOT EXISTS user_activity_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_language VARCHAR(10) NOT NULL,
  event_type VARCHAR(60) NOT NULL,
  source_type VARCHAR(40) NOT NULL DEFAULT 'learning',
  source_id UUID,
  xp INTEGER NOT NULL DEFAULT 0,
  payload JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_activity_events_user_date
  ON user_activity_events(user_id, target_language, created_at DESC);

CREATE TABLE IF NOT EXISTS daily_learning_stats (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_language VARCHAR(10) NOT NULL,
  activity_date DATE NOT NULL DEFAULT CURRENT_DATE,
  xp INTEGER NOT NULL DEFAULT 0,
  items_completed INTEGER NOT NULL DEFAULT 0,
  reviews_completed INTEGER NOT NULL DEFAULT 0,
  lessons_completed INTEGER NOT NULL DEFAULT 0,
  scenarios_completed INTEGER NOT NULL DEFAULT 0,
  corrections_completed INTEGER NOT NULL DEFAULT 0,
  minutes_active INTEGER NOT NULL DEFAULT 0,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, target_language, activity_date)
);

CREATE TABLE IF NOT EXISTS fluency_score_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_language VARCHAR(10) NOT NULL,
  current_cefr_level VARCHAR(2) NOT NULL CHECK (current_cefr_level IN ('A1','A2','B1','B2')),
  readiness_score INTEGER NOT NULL CHECK (readiness_score >= 0 AND readiness_score <= 1000),
  component_scores JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 5. Backend Models

Add Go structs in `backend/internal/models/models.go` or split into
`learning_models.go` under the same package.

Core structs:

```go
type CEFRLevel string

const (
	CEFR_A1 CEFRLevel = "A1"
	CEFR_A2 CEFRLevel = "A2"
	CEFR_B1 CEFRLevel = "B1"
	CEFR_B2 CEFRLevel = "B2"
)

type LearningSupportTier string

const (
	SupportFullCourse     LearningSupportTier = "full_course"
	SupportBetaAIAssisted LearningSupportTier = "beta_ai_assisted"
	SupportVocabOnly      LearningSupportTier = "vocab_only"
	SupportDisabled       LearningSupportTier = "disabled"
)

type LearningPairCapability struct {
	NativeLanguage         string    `json:"nativeLanguage" db:"native_language"`
	TargetLanguage         string    `json:"targetLanguage" db:"target_language"`
	SupportTier            string    `json:"supportTier" db:"support_tier"`
	ActiveCourseID         string    `json:"activeCourseId,omitempty" db:"active_course_id"`
	PlacementEnabled       bool      `json:"placementEnabled" db:"placement_enabled"`
	RoadmapEnabled         bool      `json:"roadmapEnabled" db:"roadmap_enabled"`
	ScenariosEnabled       bool      `json:"scenariosEnabled" db:"scenarios_enabled"`
	SRSEnabled             bool      `json:"srsEnabled" db:"srs_enabled"`
	MiningEnabled          bool      `json:"miningEnabled" db:"mining_enabled"`
	GrammarFeedbackEnabled bool      `json:"grammarFeedbackEnabled" db:"grammar_feedback_enabled"`
	QualityNotes           string    `json:"qualityNotes" db:"quality_notes"`
	CreatedAt              time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt              time.Time `json:"updatedAt" db:"updated_at"`
}

type UserLanguageProfile struct {
	UserID               string    `json:"userId" db:"user_id"`
	TargetLanguage       string    `json:"targetLanguage" db:"target_language"`
	NativeLanguage       string    `json:"nativeLanguage" db:"native_language"`
	CurrentCEFRLevel     string    `json:"currentCefrLevel" db:"current_cefr_level"`
	ReadinessScore       int       `json:"readinessScore" db:"readiness_score"`
	ActiveCourseID       string    `json:"activeCourseId,omitempty" db:"active_course_id"`
	ActiveUnitID         string    `json:"activeUnitId,omitempty" db:"active_unit_id"`
	PlacementStatus      string    `json:"placementStatus" db:"placement_status"`
	PrimaryGoal          string    `json:"primaryGoal" db:"primary_goal"`
	DailyGoalItems       int       `json:"dailyGoalItems" db:"daily_goal_items"`
	MiningEnabled        bool      `json:"miningEnabled" db:"mining_enabled"`
	NudgesEnabled        bool      `json:"nudgesEnabled" db:"nudges_enabled"`
	ScenarioHintsEnabled bool      `json:"scenarioHintsEnabled" db:"scenario_hints_enabled"`
	CreatedAt            time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt            time.Time `json:"updatedAt" db:"updated_at"`
}

type LearningDashboard struct {
	Profile             UserLanguageProfile  `json:"profile"`
	Capability          LearningPairCapability `json:"capability"`
	DailyGoal           DailyGoalSummary     `json:"dailyGoal"`
	Streak              StreakSummary        `json:"streak"`
	Fluency             FluencySummary       `json:"fluency"`
	CurrentUnit         UnitProgressSummary  `json:"currentUnit"`
	NextLesson          *LessonSummary       `json:"nextLesson,omitempty"`
	Vocabulary          VocabularySummary    `json:"vocabulary"`
	Grammar             GrammarSummary       `json:"grammar"`
	Scenario            ScenarioSummary      `json:"scenario"`
	RecommendedActivity []RecommendedActivity `json:"recommendedActivities"`
	WeeklyActivity      []DailyActivityPoint `json:"weeklyActivity"`
}
```

Do not remove existing `VocabularyEntry`. Extend it carefully or introduce a
new `VocabularyCard` and keep conversion helpers for the old endpoints.

## 6. Curriculum Roadmap

Seed an MVP Spanish course with 24 units: 6 per level. Each unit should have
short lessons and at least one scenario/real-talk connection.

### 6.1 A1 Units

| Unit | Title | Can-do | Grammar | Vocabulary | Scenario |
|---|---|---|---|---|---|
| A1.1 | Introductions | I can greet someone and introduce myself. | subject pronouns, ser basics, llamarse | greetings, names, countries | Meeting someone |
| A1.2 | Basics II | I can ask simple identity and language questions. | yes/no questions, question words | languages, nationalities, numbers 0-20 | Joining a language chat |
| A1.3 | Daily Routine | I can talk about simple daily habits. | present tense regular verbs, reflexive intro | time, routine verbs | Morning routine chat |
| A1.4 | Ordering Food | I can order food or drink and ask prices. | querer, quisiera chunks, gender/articles | cafe items, prices, preferences | Ordering coffee |
| A1.5 | Around Town | I can ask where something is and follow simple directions. | estar for location, hay, prepositions | places, left/right/near/far | Asking directions |
| A1.6 | A1 Checkpoint | I can complete basic social and service interactions. | A1 review | A1 review | Cafe plus directions |

### 6.2 A2 Units

| Unit | Title | Can-do | Grammar | Vocabulary | Scenario |
|---|---|---|---|---|---|
| A2.1 | Past Weekend | I can describe simple completed events. | preterite regular, ir/ser preterite | weekend, leisure, yesterday | Weekend recap |
| A2.2 | Shopping | I can ask for items, sizes, and totals. | demonstratives, direct objects intro | clothing, groceries, quantities | Grocery checkout |
| A2.3 | Plans | I can talk about near-future plans. | ir a + infinitive, tener que | dates, invitations, errands | Making plans |
| A2.4 | Health And Feelings | I can describe how I feel and ask for help. | estar vs ser, doler chunks | body, symptoms, emotions | Pharmacy visit |
| A2.5 | Travel Basics | I can book simple travel and lodging. | polite requests, poder, necesitar | hotels, transport, documents | Booking a room |
| A2.6 | A2 Checkpoint | I can handle predictable everyday tasks. | A2 review | A2 review | Travel day simulation |

### 6.3 B1 Units

| Unit | Title | Can-do | Grammar | Vocabulary | Scenario |
|---|---|---|---|---|---|
| B1.1 | Stories | I can narrate past experiences with sequence. | preterite vs imperfect | connectors, events | Telling a story |
| B1.2 | Opinions | I can give opinions and reasons. | porque, aunque, comparative structures | preferences, media, hobbies | Discussing a movie |
| B1.3 | Problems | I can explain a problem and request a solution. | object pronouns, formal requests | repairs, customer service | Returning an item |
| B1.4 | Social Plans | I can negotiate plans and preferences. | conditional intro, suggestions | schedules, constraints | Planning dinner |
| B1.5 | Work And Study | I can describe responsibilities and goals. | present perfect intro, obligation | work, study, skills | Job chat |
| B1.6 | B1 Checkpoint | I can sustain conversations on familiar topics. | B1 review | B1 review | Multi-step service issue |

### 6.4 B2 Units

| Unit | Title | Can-do | Grammar | Vocabulary | Scenario |
|---|---|---|---|---|---|
| B2.1 | Nuanced Opinions | I can defend a viewpoint with nuance. | subjunctive triggers intro | society, technology, tradeoffs | Debate a recommendation |
| B2.2 | Hypotheticals | I can discuss imagined outcomes. | si clauses, conditional | risks, consequences, priorities | Travel disruption |
| B2.3 | Professional Communication | I can handle formal work conversations. | register, formal commands | meetings, deadlines, feedback | Rescheduling a meeting |
| B2.4 | Media And Culture | I can summarize and react to articles. | reported speech, connectors | news, culture, abstract nouns | Discussing an article |
| B2.5 | Conflict And Repair | I can clarify misunderstandings and negotiate. | concession, hedging, repair phrases | disagreement, apologies, compromise | Resolving a complaint |
| B2.6 | B2 Checkpoint | I can interact with fluency in varied contexts. | B2 review | B2 review | Open-ended interview |

### 6.5 Lesson Template Per Unit

Each non-checkpoint unit should seed 5 lessons:

1. `vocabulary`: introduce 8-12 target words or chunks.
2. `grammar`: one grammar point with examples.
3. `reading`: short passage using target vocabulary.
4. `production`: typed or spoken prompts.
5. `scenario_intro`: guided scenario preparation.

Each checkpoint unit should seed:

1. Mixed vocabulary review.
2. Mixed grammar review.
3. Reading/listening check.
4. Scenario checkpoint.
5. Unit completion summary.

## 7. Ordering Coffee Scenario Seed

Add this scenario first because it is shown in the Stitch samples.

Scenario:

- slug: `ordering-coffee`
- title: `Ordering Coffee at a Cafe`
- level: A1
- unit: A1.4 Ordering Food
- domain: `food_drink`
- can-do: `I can order a drink, ask about prices, and respond to simple service questions.`
- AI role name: `Sparky`
- AI role description: `Friendly cafe barista. Speaks clear A1 Spanish.`
- opening line: `Hola. ¿Qué te gustaría pedir hoy?`
- max turns: 10
- estimated minutes: 3-5

Phases:

1. Greeting
   - learner goal: greet the barista.
   - required intents: `greet`.
   - chunks:
     - `Hola, buenos días.` -> `Hello, good morning.`
     - `Buenas tardes.` -> `Good afternoon.`
2. Order
   - learner goal: order one drink.
   - required intents: `order_drink`.
   - chunks:
     - `Quisiera un café con leche, por favor.` -> `I would like a coffee with milk, please.`
     - `¿Me puede dar un café, por favor?` -> `Can you give me a coffee, please?`
3. Customization
   - learner goal: answer or request a simple option.
   - required intents: `customize`.
   - chunks:
     - `Para llevar, por favor.` -> `To go, please.`
     - `Sin azúcar, por favor.` -> `Without sugar, please.`
     - `¿Tiene leche de avena?` -> `Do you have oat milk?`
4. Payment
   - learner goal: ask or understand the price.
   - required intents: `pay`.
   - chunks:
     - `¿Cuánto cuesta?` -> `How much does it cost?`
     - `¿Aceptan tarjeta?` -> `Do you accept card?`
5. Closing
   - learner goal: close politely.
   - required intents: `close`.
   - chunks:
     - `Gracias.` -> `Thank you.`
     - `Que tenga un buen día.` -> `Have a good day.`

Completion criteria:

```json
{
  "required_phase_count": 4,
  "required_intents": ["greet", "order_drink", "pay", "close"],
  "min_user_turns": 4,
  "min_score": 700,
  "allowed_native_language_turns": 1
}
```

## 8. Word Mining Pipeline

### 8.1 Enqueue Rules

Add mining enqueue after message persistence in `MessageHandler.SendMessage` or
`MessageService.CreateMessage`, depending on where dependencies are cleanest.

Only enqueue when:

- user has a `user_language_profiles` row for the message language, or the
  message language matches one of `users.target_languages`.
- `mining_enabled = true`.
- chat settings do not disable learning mining.
- message text is not empty and not too long. Start with max 750 characters.
- message language is not the user's native language unless the user is
  explicitly practicing that language.

For received messages:

- Mine for each participant whose target language matches the message language.

For sent messages:

- Mine if the sender is using the target language.
- Sent-message mining is important for spontaneous-use detection.

### 8.2 Extraction Contract

Create a `LearningAIService` that can call the same provider family used by
grammar. For MVP, it can duplicate the small provider-fallback pattern from
`GrammarService`; later it can be refactored into a shared AI client.

Prompt must return strict JSON:

```json
{
  "items": [
    {
      "surface_text": "café con leche",
      "lemma": "café con leche",
      "part_of_speech": "noun_phrase",
      "is_chunk": true,
      "translation": "coffee with milk",
      "definition": "a coffee drink served with milk",
      "cefr_level": "A1",
      "grammar_tags": ["articles", "food_ordering"],
      "is_proper_noun": false,
      "confidence": 0.93,
      "reason": "Useful high-frequency cafe phrase"
    }
  ]
}
```

Extraction rules:

- Prefer useful words and chunks over every token.
- Include multi-word chunks when they are more teachable than individual words.
- Exclude names, URLs, emoji-only items, filler, and extremely rare slang unless
  personally relevant and understandable.
- Return 0-8 items per message.
- Tag CEFR conservatively.
- For language-specific morphology, return lemma when possible.

Fallback when AI unavailable:

- Tokenize with a simple Unicode letter regex.
- Remove language stopwords.
- Keep words longer than 3 characters.
- Translate through existing `TranslationService`.
- Mark confidence 0.35 and status `candidate`.

### 8.3 Normalization

Implement:

```go
func NormalizeLearningTerm(s string, lang string) string {
	// lower, trim punctuation, collapse whitespace, remove inverted punctuation
	// for Spanish, keep accents for display but optionally store accent-folded
	// lookup in metadata if later needed.
}
```

Use `normalized_term` on `vocabulary` and `normalized_text` on `mined_items`.

### 8.4 Classification And Routing

For each mined item:

1. Exact match against `lexical_items` by `(course_id, language, lemma)`.
2. If no exact match, check `forms` JSON aliases.
3. If phrase/chunk, match `display_text` and `tags`.
4. If no lexical match, match grammar tags to `grammar_points`.
5. Determine user's progress relative to matched unit:
   - upcoming unit -> `upcoming_unit`.
   - current unit -> `current_unit`.
   - completed unit -> `completed_unit`.
   - no unit -> `bonus`.
6. Proper nouns or low confidence -> `ignored` or `candidate`.

Teachability score formula:

```text
teachability =
  25 * personal_context_score
  + 20 * curriculum_relevance
  + 15 * frequency_score
  + 15 * recency_score
  + 10 * not_already_mastered
  + 10 * confidence
  + 5  * chunk_bonus
  - 20 * proper_noun_penalty
  - 15 * too_advanced_penalty
```

Score ranges 0-100. Auto-add threshold starts at 70 for premium/opt-in users
and 85 for conservative default. Manual tap-to-save always adds.

### 8.5 Dedupe

If existing vocabulary exists for `(user_id, language, normalized_term)`:

- Update `last_seen_at`.
- Insert/update `vocabulary_sources`.
- Increment route-specific metadata.
- If seen in outgoing user message without prompt, increment
  `spontaneous_use_count`.
- Do not create duplicate cards.

If no existing vocabulary:

- Insert `mined_items`.
- Auto-add to `vocabulary` only if score and settings allow it.

## 9. Depth-Of-Processing SRS

Stages:

1. Recognition: see target term, pick meaning.
2. Cued recall: fill blank in original message sentence.
3. Free recall: see meaning/context, type target term.
4. Production: use target term in a new sentence.
5. Spontaneous use: unprompted correct use in chat or scenario.

Mastery rule:

- A vocabulary item is `mastered` only after:
  - stage 4 is passed in at least two different sessions, or
  - stage 4 is passed once and stage 5 occurs once later.
- Stage 1 alone never counts as mastery.

Scheduling:

- Reuse existing `next_review`, `interval_days`, and `ease_factor`.
- Add stage-aware updates.
- Incorrect stage 1-2 answer resets interval to 1 day.
- Incorrect stage 3-4 answer keeps or demotes stage depending on severity.
- Stage advancement requires correct answer quality >= 4.

Quality mapping:

```text
0 = blank or completely wrong
1 = wrong, recognized after reveal
2 = partially correct
3 = correct with major typo/help
4 = correct
5 = correct, fast, and confident
```

Simplified stage-aware SM-2:

```go
func UpdateVocabAfterAttempt(card, stage, quality) {
	card.ReviewCount++
	if quality < 3 {
		card.Lapses++
		card.StageSuccessCount = 0
		card.IntervalDays = 1
		card.EaseFactor = max(1.30, card.EaseFactor-0.20)
		card.NextReview = now + 1 day
		if stage >= 3 {
			card.MasteryStage = max(1, card.MasteryStage-1)
		}
		return
	}

	card.CorrectCount++
	card.StageSuccessCount++
	if stage == 4 {
		card.ProductionSuccessCount++
	}

	if quality >= 4 && card.StageSuccessCount >= RequiredSuccesses(stage) {
		card.MasteryStage = min(4, card.MasteryStage+1)
		card.StageSuccessCount = 0
	}

	card.EaseFactor = clamp(1.30, 3.00, card.EaseFactor + EaseDelta(quality))
	card.IntervalDays = NextInterval(card.IntervalDays, card.EaseFactor, stage)
	card.NextReview = now + interval
}
```

Initial intervals:

- new item: 1 day
- after stage 1 pass: 2 days
- after stage 2 pass: 3 days
- after stage 3 pass: 5 days
- after stage 4 first pass: 7 days
- mastered: 14+ days, then grows

## 10. Daily Session Composer

Endpoint:

```http
POST /api/v1/learning/sessions/start
```

Request:

```json
{
  "targetLanguage": "es",
  "mode": "daily",
  "source": "learn_home"
}
```

Response:

```json
{
  "session": {
    "id": "...",
    "mode": "daily",
    "plannedItemCount": 10
  },
  "items": [
    {
      "id": "...",
      "itemType": "vocabulary",
      "activityType": "cued_recall",
      "payload": {}
    }
  ]
}
```

Composition algorithm:

```text
target_count = user.daily_goal_items, default 10

1. Add due SRS items first, capped at 60 percent of target_count.
2. Add 1 current unit lesson step block, capped at 30 percent.
3. Add top 2-3 mined candidates or recent accepted words.
4. Add 1 recent grammar weakness when available.
5. If user chose conversational fluency, swap one lesson block for a scenario
   micro task every other day.
6. Interleave item types so the user does not do all vocabulary then all
   grammar.
7. If no due content exists, return the next lesson plus 2 bonus review items.
```

Interleaving:

```text
vocabulary -> grammar -> lesson -> vocabulary -> scenario_prompt -> vocabulary
```

Answer endpoint:

```http
POST /api/v1/learning/sessions/:sessionId/items/:itemId/answer
```

Request:

```json
{
  "answer": {
    "text": "estoy"
  },
  "latencyMs": 4200
}
```

Response:

```json
{
  "correct": true,
  "quality": 4,
  "feedback": {
    "message": "Nice. Use estar for a temporary feeling.",
    "correctAnswer": "estoy",
    "grammarPointId": "..."
  },
  "nextItem": {}
}
```

Complete endpoint:

```http
POST /api/v1/learning/sessions/:sessionId/complete
```

Updates:

- `learning_sessions`.
- `daily_learning_stats`.
- `user_activity_events`.
- `user_unit_progress` if lesson content was completed.
- `user_language_profiles.readiness_score`.
- `fluency_score_snapshots` optionally once per day.

## 11. Lesson Progression Rules

Lesson attempt flow:

```http
POST /api/v1/learning/lessons/:lessonId/start
POST /api/v1/learning/lesson-attempts/:attemptId/steps/:stepId/answer
POST /api/v1/learning/lesson-attempts/:attemptId/complete
```

Completion:

- A non-checkpoint lesson completes when at least 80 percent of required steps
  are answered and score >= 650.
- A checkpoint completes when score >= 750.
- If checkpoint score is 650-749, keep unit in progress and recommend targeted
  drills.
- If checkpoint score < 650, unlock review session and do not advance.

Unit progress:

```text
progress_pct =
  40 percent lesson completion
  25 percent vocabulary target coverage
  20 percent grammar confidence
  15 percent scenario task completion
```

Unit completion:

- all required lessons complete,
- unit checkpoint passed,
- at least 60 percent of target lexical items are stage 2+,
- required grammar points confidence >= 0.65,
- required scenario completed when unit has a scenario.

## 12. Placement Test

### 12.1 Item Bank

Seed placement items in code or JSON first; later store in DB if needed.

Item types:

- Vocabulary MCQ.
- Grammar cloze.
- Reading comprehension.
- Short production prompt.
- Scenario comprehension.

Need about 8-12 items per level for MVP.

### 12.2 Adaptive Algorithm

Use an IRT-lite ladder:

```text
start difficulty = A1/A2 boundary
correct high confidence -> move up one level
incorrect -> move down one level
after 8-12 questions or confidence >= threshold -> finish
```

Numeric ability:

```text
A1 item = 100
A2 item = 350
B1 item = 650
B2 item = 850

ability_estimate starts at 250.
For each answer:
  if correct: ability += 0.35 * (item_value - ability + 150)
  if wrong:   ability -= 0.35 * (ability - item_value + 150)
clamp 0..1000
```

Level assignment:

```text
0-249   -> A1, start Unit A1.1
250-549 -> A2, start first incomplete A2 unit or late A1 checkpoint if weak
550-799 -> B1, start first incomplete B1 unit
800-1000 -> B2, start first incomplete B2 unit
```

Readiness score:

- If assigned A2 with ability 490, readiness to B1 might be 740/1000.
- Calculate as progress within the assigned level band toward next level.

### 12.3 Placement Endpoints

```http
POST /api/v1/learning/placement/start
POST /api/v1/learning/placement/:attemptId/answer
POST /api/v1/learning/placement/:attemptId/skip
GET  /api/v1/learning/placement/:attemptId
```

Skip behavior:

- `placement_status = skipped`.
- `current_cefr_level = A1`.
- `active_unit_id = A1.1`.
- `readiness_score = 0`.

## 13. Fluency Readiness Score

Expose:

```json
{
  "currentCefrLevel": "A2",
  "readinessScore": 740,
  "readinessPercent": 74,
  "label": "Approaching B1",
  "componentScores": {
    "curriculum": 780,
    "vocabulary": 720,
    "grammar": 690,
    "scenarios": 810,
    "consistency": 730
  }
}
```

Weights:

- Curriculum completion: 30 percent.
- Vocabulary depth: 25 percent.
- Grammar mastery: 20 percent.
- Scenario/task completion: 15 percent.
- Consistency/streak: 10 percent.

Component formulas:

```text
curriculum = weighted completion of current level units
vocabulary = percent of level target lexical items at stage 3+, plus production bonus
grammar = average user_grammar_mastery confidence for level grammar points
scenarios = required scenario completion and score average
consistency = recent active days and daily goal completion
```

Level-up:

- When readiness_score >= 900 and level checkpoint passed, move to next CEFR
  level.
- Keep readiness at 1000 if user is B2 and all B2 checkpoints pass.

## 14. API Contract

Add one `LearningHandler` and optionally one `ScenarioHandler`.

Routes under protected `/api/v1`:

```text
GET    /learning/capabilities?nativeLanguage=en&targetLanguage=es
GET    /learning/profile?targetLanguage=es
PUT    /learning/profile
GET    /learning/dashboard?targetLanguage=es
GET    /learning/path?targetLanguage=es

POST   /learning/placement/start
POST   /learning/placement/:attemptId/answer
POST   /learning/placement/:attemptId/skip
GET    /learning/placement/:attemptId

GET    /learning/units/:unitId
POST   /learning/lessons/:lessonId/start
POST   /learning/lesson-attempts/:attemptId/steps/:stepId/answer
POST   /learning/lesson-attempts/:attemptId/complete

POST   /learning/sessions/start
GET    /learning/sessions/:sessionId
POST   /learning/sessions/:sessionId/items/:itemId/answer
POST   /learning/sessions/:sessionId/complete

GET    /learning/vocabulary/mined?targetLanguage=es&status=candidate
POST   /learning/vocabulary/mined/:id/accept
POST   /learning/vocabulary/mined/:id/ignore
POST   /learning/vocabulary/:id/review

GET    /learning/scenarios?targetLanguage=es
GET    /learning/scenarios/:scenarioId
POST   /learning/scenarios/:scenarioId/start
GET    /learning/scenario-runs/:runId
POST   /learning/scenario-runs/:runId/message
POST   /learning/scenario-runs/:runId/hint
POST   /learning/scenario-runs/:runId/complete

GET    /learning/real-talk/prompts?targetLanguage=es&chatId=...
POST   /learning/real-talk/prompts/:promptId/used
POST   /learning/nudges/:nudgeId/dismiss

POST   /learning/streak/recover
```

Dashboard response shape:

```json
{
  "capability": {
    "nativeLanguage": "en",
    "targetLanguage": "es",
    "supportTier": "full_course",
    "placementEnabled": true,
    "roadmapEnabled": true,
    "scenariosEnabled": true,
    "srsEnabled": true,
    "miningEnabled": true,
    "grammarFeedbackEnabled": true
  },
  "profile": {
    "targetLanguage": "es",
    "nativeLanguage": "en",
    "currentCefrLevel": "A2",
    "readinessScore": 740,
    "activeUnitId": "..."
  },
  "dailyGoal": {
    "targetItems": 10,
    "completedItems": 8,
    "percent": 80
  },
  "streak": {
    "days": 14,
    "atRisk": false,
    "canRecover": false
  },
  "fluency": {
    "readinessScore": 740,
    "readinessPercent": 74,
    "label": "Approaching B1"
  },
  "currentUnit": {
    "id": "...",
    "title": "Daily Routine",
    "cefrLevel": "A2",
    "progressPct": 62,
    "canDoStatement": "I can order food"
  },
  "vocabulary": {
    "total": 342,
    "dueToday": 45,
    "mastered": 120,
    "newFromChats": 10
  },
  "grammar": {
    "weakestPointTitle": "Past Tense",
    "confidencePct": 65,
    "dueToday": 3
  },
  "scenario": {
    "nextScenarioId": "...",
    "title": "Ordering Coffee at a Cafe",
    "progressPct": 60,
    "hasNewWords": true
  },
  "recommendedActivities": [],
  "weeklyActivity": []
}
```

Dashboard behavior by `supportTier`:

- `full_course`: include `currentUnit`, `nextLesson`, roadmap nodes, placement,
  scenarios, and score components.
- `beta_ai_assisted`: include dashboard, SRS, mined words, beta scenarios, and
  soft progress, but do not claim validated CEFR advancement.
- `vocab_only`: include vocabulary, mined words, grammar feedback, daily goal,
  and streak. Return empty/null roadmap and lesson fields with a clear
  capability flag so clients show "structured course coming soon".
- `disabled`: learning endpoints should return 404 or a typed capability error,
  while translation/chat remains unaffected.

## 15. LLM Prompt Contracts

### 15.1 Word Extraction Prompt

System:

```text
You are a language-learning vocabulary miner. Extract only useful vocabulary
and reusable chunks from a learner's real message context. Return strict JSON.
Do not include markdown or prose.
```

User:

```text
Target language: Spanish
Native language: English
Learner CEFR: A2
Source text:
"Hola Sofía. Me siento bien pero es un poco difícil recordar los verbos reflexivos."

Return JSON:
{
  "items": [
    {
      "surface_text": string,
      "lemma": string,
      "part_of_speech": string,
      "is_chunk": boolean,
      "translation": string,
      "definition": string,
      "cefr_level": "A1" | "A2" | "B1" | "B2",
      "grammar_tags": string[],
      "is_proper_noun": boolean,
      "confidence": number,
      "reason": string
    }
  ]
}
```

### 15.2 Scenario Response Prompt

System:

```text
You are the AI scene partner in a language-learning roleplay. Stay in character.
Use clear language at the learner's CEFR level. Keep replies short. Do not
complete the learner's task for them. Move the scenario forward one phase at a
time. Return strict JSON.
```

User payload:

```json
{
  "scenario": "Ordering Coffee at a Cafe",
  "ai_role": "Friendly cafe barista",
  "target_language": "es",
  "native_language": "en",
  "cefr_level": "A1",
  "current_phase": {
    "title": "Order",
    "learner_goal": "Order one drink",
    "required_intents": ["order_drink"]
  },
  "history": [],
  "learner_message": "Quisiera un café con leche, por favor.",
  "allowed_new_items": ["para llevar", "cuanto cuesta"],
  "return_schema": {
    "ai_message": "string",
    "translation": "string",
    "phase_complete": "boolean",
    "next_phase_ordinal": "number",
    "nudge": {
      "show": "boolean",
      "text": "string",
      "suggested_chunks": []
    }
  }
}
```

### 15.3 Scenario Turn Evaluation Prompt

Return:

```json
{
  "intent_score": 0.0,
  "grammar_score": 0.0,
  "vocabulary_score": 0.0,
  "fluency_score": 0.0,
  "covered_intents": ["order_drink"],
  "used_chunks": ["quisiera"],
  "errors": [
    {
      "span": "soy",
      "correction": "estoy",
      "grammar_tag": "estar-vs-ser",
      "explanation": "Use estar for temporary states."
    }
  ],
  "overall_quality": 4,
  "should_prompt_self_correction": false
}
```

### 15.4 Lesson Step Generation

Most lesson content should be seeded deterministically. Use AI generation only
for variants, feedback, and practice generation when the seed bank runs out.

Return strict JSON with:

- prompt,
- choices,
- accepted answers,
- explanation,
- CEFR,
- lexical item ids,
- grammar point ids.

## 16. Frontend Implementation

### 16.1 Shared Types And API

Add TypeScript interfaces to `packages/shared/src/types.ts`:

- `LearningPairCapability`.
- `UserLanguageProfile`.
- `LearningDashboard`.
- `DailyGoalSummary`.
- `StreakSummary`.
- `FluencySummary`.
- `UnitProgressSummary`.
- `LessonSummary`.
- `RecommendedActivity`.
- `LearningSession`.
- `LearningSessionItem`.
- `MinedItem`.
- `ScenarioScript`.
- `ScenarioRun`.
- `ScenarioTurn`.
- `PlacementAttempt`.

Add `learning` API group to `packages/shared/src/api.ts`:

```ts
const learning = {
  getCapabilities,
  getProfile,
  updateProfile,
  getDashboard,
  getPath,
  startPlacement,
  answerPlacement,
  skipPlacement,
  startSession,
  getSession,
  answerSessionItem,
  completeSession,
  getMinedItems,
  acceptMinedItem,
  ignoreMinedItem,
  getScenarios,
  getScenario,
  startScenario,
  getScenarioRun,
  sendScenarioMessage,
  requestScenarioHint,
  completeScenario,
  getRealTalkPrompts,
  markRealTalkPromptUsed,
}
```

Export it from web `frontend/src/services/api.ts` and mobile
`mobile/src/services/api.ts`.

### 16.2 Web Routes

Add routes in `frontend/src/App.tsx`:

```tsx
<Route path="/learn/placement" element={<Placement />} />
<Route path="/learn/session/:sessionId" element={<LessonSession />} />
<Route path="/learn/vocabulary" element={<VocabularyReview />} />
<Route path="/learn/scenarios" element={<Scenarios />} />
<Route path="/learn/scenarios/:scenarioId" element={<ScenarioRoleplay />} />
<Route path="/learn/roadmap" element={<LearningRoadmap />} />
```

### 16.3 Web Learn Dashboard

Replace placeholder values in `frontend/src/pages/Learn.tsx` with
`learningAPI.getDashboard`.

Before rendering full roadmap/course widgets, check
`dashboard.capability.supportTier`:

- `full_course`: render the full Stitch-inspired dashboard.
- `beta_ai_assisted`: render dashboard and practice, but label generated
  scenarios/lessons as beta and hide validated level-up claims.
- `vocab_only`: render SRS, mined words, grammar insights, streak, and a
  structured-course coming-soon state.
- `disabled`: render chat/translation learning unavailable state.

Actions:

- Quick Drills Start -> `learningAPI.startSession({ mode: "quick_drill" })`.
- Vocabulary card -> `/learn/vocabulary`.
- Scenario card -> `/learn/scenarios/:id`.
- Grammar Deep Dive -> `learningAPI.startSession({ mode: "grammar" })`.
- Continue Learning -> `learningAPI.startSession({ mode: "daily" })`.
- Roadmap card -> `/learn/roadmap`.

Use Stitch visual intent:

- Header: "Your Learning Path" or "Your Fluency Journey".
- Compact metric cards.
- Progress bars and rings.
- Blue for primary actions.
- Purple for AI/scenario surfaces.
- Emerald for completed progress.
- Red/rose only for at-risk streak or errors.

Do not import Stitch HTML directly. Recreate it as React components using the
existing Tailwind/theme tokens.

### 16.4 Web Lesson Session

Component responsibilities:

- Render one item at a time.
- Support MCQ, cloze, typed answer, production prompt, explanation.
- Post answers to backend.
- Show immediate feedback.
- Advance to next item.
- Complete session and navigate back to Learn with updated metrics.

### 16.5 Web Scenario Roleplay

Screen layout:

- Header with close/back, scenario title, streak protection badge.
- AI avatar/name and role.
- Chat transcript.
- Translation toggle or "View translation" under AI messages.
- Nudge panel with suggested chunks.
- Input with microphone button, text field, hint button, send button.

Flow:

1. `startScenario`.
2. Render opening AI turn.
3. User sends target-language response.
4. Backend returns AI response, evaluation, next phase, optional nudge.
5. Store turns.
6. On completion, show summary and XP.

### 16.6 Chat Integration

Extend `MessageBubble.tsx`:

- Replace naive word extraction buttons with backend mined items if available.
- On hover/tap show:
  - Save word/chunk.
  - Practice this.
  - Grammar.
  - Deep Dive.
- If message has high-value mined candidates, show subtle underlines or chips.
- Manual save calls current `vocabularyAPI.save` initially, then upgraded
  `/learning/vocabulary/mined/:id/accept` when a mined item exists.

Add nudge component:

```text
frontend/src/components/chat/RealTalkNudge.tsx
```

It should display active unit goal, suggested phrase, "Send to Input", and
dismiss.

### 16.7 Mobile Navigation

Update `mobile/src/components/MainTabs.tsx`:

```ts
export type LearnStackParamList = {
  LearnHome: undefined
  Placement: undefined
  LessonSession: { sessionId: string }
  VocabularyReview: undefined
  Scenarios: undefined
  ScenarioRoleplay: { scenarioId: string }
  LearningRoadmap: undefined
}
```

Wire new screens to the shared API.

### 16.8 Registration, Settings, And Pair Selection

Current gap:

- Web registration currently sets `targetLanguages: []`.
- Mobile registration currently collects only native language.
- Web Settings/Profile and mobile Profile let users choose target languages, but
  they do not explain whether a selected pair has a full course or only
  vocabulary/grammar support.

Required behavior:

- During registration or first Learn-tab visit, ask for:
  - native language,
  - target language(s),
  - primary learning goal.
- After target language selection, call:

```http
GET /api/v1/learning/capabilities?nativeLanguage=en&targetLanguage=es
```

- For `full_course`, show "Structured lessons available" and route to placement
  or the learning dashboard.
- For `vocab_only`, show "Translation, grammar help, and vocabulary review are
  available. Structured lessons for this language pair are coming soon."
- For `beta_ai_assisted`, show a beta disclosure before lessons/scenarios.
- For `disabled`, hide learning setup and keep chat/translation setup only.

UI copy should consistently say "I speak" for native/base language and "I want
to learn" for target language. Avoid using "preferred language" when the user is
really selecting the language they want messages translated into.

## 17. Public Site, Marketing, And i18n Strategy

### 17.1 Core Messaging

Marketing must separate three claims:

1. Messaging translation: multilingual and model-powered.
2. Grammar analysis: multilingual and model-powered, with variable quality by
   language.
3. Structured learning courses: curated per native/target language pair.

Recommended public wording for launch:

```text
Chat across languages with instant translation and AI grammar help.
For English speakers learning Spanish, Chorus also includes a guided A1-B2
learning path with lessons, vocabulary review, and real-world roleplay.
More structured courses are coming.
```

Do not say "learn any language fluently" unless that pair has
`support_tier = full_course`.

### 17.2 Public Surfaces To Update

Update all public/product-facing pages:

```text
landing/index.html
landing/download.html
landing/about.html
frontend/src/pages/Landing.tsx
frontend/src/pages/About.tsx
frontend/src/pages/Pricing.tsx
frontend/src/pages/Premium.tsx
frontend/src/pages/Waitlist.tsx
frontend/src/pages/Register.tsx
frontend/src/pages/Settings.tsx
frontend/src/pages/Profile.tsx
mobile/src/screens/LandingScreen.tsx
mobile/src/screens/AboutScreen.tsx
mobile/src/screens/PricingScreen.tsx
mobile/src/screens/RegisterScreen.tsx
mobile/src/screens/ProfileScreen.tsx
mobile/src/screens/LearnScreen.tsx
```

Current observed gaps:

- `landing/index.html` says 9 languages, while the React landing page says 10.
- Public copy emphasizes translation and generic learning, but not pair-specific
  structured course availability.
- Pricing pages mention grammar/vocabulary, but should clarify whether guided
  courses are included only for supported pairs.
- Mobile public/auth screens use hardcoded English strings and do not yet share
  the web i18n system.
- Waitlist collects spoken and target languages, but should also help measure
  demand for unsupported full courses.
- Registration does not yet collect target learning languages at account
  creation.
- Privacy copy should align with actual retention behavior. If messages are
  retained for 365 days by default, do not claim they are not stored
  permanently without explaining retention controls.

### 17.3 Language Availability Component

Create one reusable component for web and one equivalent for mobile:

```text
frontend/src/components/learning/LanguagePairAvailability.tsx
mobile/src/components/learning/LanguagePairAvailability.tsx
```

Inputs:

```ts
{
  nativeLanguage: string
  targetLanguage: string
  capability: LearningPairCapability
  compact?: boolean
}
```

States:

- `full_course`: "Full A1-B2 course available", Start placement / Start lessons.
- `beta_ai_assisted`: "Beta learning path", Start beta practice.
- `vocab_only`: "Course waitlist", Join course waitlist / Use vocabulary review.
- `disabled`: "Learning unavailable for this pair", Use translation only.

Use this component in:

- Landing language/course section.
- Waitlist confirmation.
- Registration completion.
- Learn dashboard empty/unsupported state.
- Settings/Profile language picker.

### 17.4 Public Language Section

Replace a single "Supported Languages" grid with two clear groups:

```text
Translation and grammar support
English, Spanish, French, German, Italian, Portuguese, Hindi, Chinese, Arabic,
Bengali, Russian, Urdu, and other configured model-supported languages.

Structured learning courses
Available at launch: English speakers learning Spanish.
Coming next: based on waitlist demand.
```

If the product wants to keep a concise hero stat, use:

```text
10+ translation languages
1 guided course at launch
More courses coming
```

### 17.5 Waitlist And Demand Capture

Extend waitlist submission data or metadata so the business can prioritize new
courses:

```json
{
  "spokenLanguages": ["en"],
  "targetLanguages": ["es", "fr"],
  "requestedStructuredCourses": [
    { "nativeLanguage": "en", "targetLanguage": "fr" }
  ],
  "reasons": ["Learn a new language"],
  "comments": ""
}
```

Product behavior:

- If user selects `en -> es`, tell them the first guided course is available or
  invite them to request access.
- If user selects unsupported pair, say they can still use translation, grammar,
  and vocabulary tools, and that the structured course request has been logged.
- Admin waitlist should show requested pair counts so course roadmap decisions
  are data-driven.

Backend additions:

- Add `requested_structured_courses JSONB DEFAULT '[]'` to `waitlist_entries`,
  or derive pair demand from `spoken_languages` and `target_languages`.
- Add admin analytics endpoint:

```http
GET /api/v1/admin/learning/course-demand
```

Response:

```json
{
  "pairs": [
    {
      "nativeLanguage": "en",
      "targetLanguage": "fr",
      "requests": 128,
      "supportTier": "vocab_only"
    }
  ]
}
```

### 17.6 i18n Requirements

There are two separate internationalization needs:

- App interface language: language used for buttons, menus, onboarding, pricing,
  and dashboard copy.
- Learning pair: native language and target language for curriculum.

Do not conflate them. A user might use the app UI in English while learning
Spanish from Hindi, or use the UI in Spanish while learning French.

Web:

- Move landing-page inline `STRINGS` toward the existing
  `frontend/src/i18n/locales/*.ts` structure, or ensure every landing string has
  a matching i18n key.
- Add learning availability keys to every locale file:

```ts
learningAvailability: {
  translationLanguagesTitle: string
  translationLanguagesSubtitle: string
  structuredCoursesTitle: string
  fullCourseAvailable: string
  fullCourseCta: string
  betaCourseAvailable: string
  vocabOnlyTitle: string
  vocabOnlyBody: string
  joinCourseWaitlist: string
  useVocabularyTools: string
  unavailableTitle: string
  unavailableBody: string
}
```

Mobile:

- Add an i18n layer to `mobile` instead of hardcoding English strings in
  screens.
- Prefer sharing locale keys with web where practical.
- At minimum, localize:
  - Landing,
  - Login/Register,
  - Profile/Settings,
  - Learn dashboard,
  - Placement,
  - Scenario roleplay,
  - unsupported-course messages,
  - pricing.

RTL and non-Latin scripts:

- Arabic and Urdu require RTL layout support.
- Chinese, Hindi, Bengali, Urdu, Arabic, Japanese, and Korean require font and
  line-height checks on mobile and web.
- Do visual QA for long translated strings. Buttons must not overflow.

### 17.7 Learn Tab Behavior For Unsupported Pairs

If the user chooses a target language whose pair is not `full_course`:

- Do not show the placement test.
- Do not show A1-B2 roadmap nodes.
- Do not show "fluency score" as if it is validated.
- Show:
  - chat-mined vocabulary,
  - due SRS,
  - grammar insights,
  - saved words,
  - course waitlist CTA,
  - explanation that structured lessons are not yet available for that pair.

For multiple target languages:

- Let the user switch active target language in Learn.
- Resolve capability separately for each pair.
- If at least one pair is `full_course`, default Learn to that pair.
- If none are `full_course`, default to the most recently active target language
  and show the vocab-only dashboard.

### 17.8 Pricing And Entitlement Copy

Pricing must avoid implying paid users get full structured courses for every
language pair.

Recommended wording:

```text
Premium expands AI translation, grammar, vocabulary, and practice limits.
Guided A1-B2 courses are available for supported language pairs, starting with
English speakers learning Spanish.
```

Entitlements should gate capacity/features, not course existence:

- Premium can increase mining volume, automatic grammar, scenario usage, and
  daily AI limits.
- Premium cannot turn an unseeded `vocab_only` pair into a validated
  `full_course`.

### 17.9 SEO And App Store Positioning

Public positioning:

- Primary category: multilingual messenger with AI translation.
- Secondary launch wedge: English-to-Spanish learning path from real chats.

Suggested SEO phrases:

- `language learning messenger`,
- `learn Spanish from real conversations`,
- `AI translation chat app`,
- `Spanish lessons with real-world roleplay`,
- `chat-based vocabulary practice`.

Avoid SEO pages for unsupported full courses until the product has at least
`beta_ai_assisted` content for that pair.

## 18. Design Mapping From Stitch Screens

Use these screen concepts:

- Activity Hub: primary Learn dashboard with Quick Drills, Vocabulary,
  Scenarios, Grammar Deep Dive, weekly goal, bottom nav.
- Placement Test Welcome: onboarding route before dashboard if placement is
  not completed or skipped.
- Placement Vocabulary Question: reusable MCQ component with progress bar,
  skip, check button disabled until selection.
- Placement Reading: passage component with progress and answers.
- Placement Results: current CEFR, skipped units, starting unit, Start Learning.
- Real Talk Hub: prompt list by current goal and prompt type tabs.
- Chat with Real Talk Prompt: chat nudge inserted above input.
- AI Tutor Learning Moment: correction bottom sheet/card with tabs:
  Overview, Word-by-Word, Grammar.
- Scenario List: categories and cards with progress, level, time.
- Scenario Roleplay: AI role chat, translation, suggested chunks, input.
- Roadmap/Streak: fluency score card, streak card, current unit, completed and
  locked nodes.
- Streak Recovery: modal with two quick tasks and reset option.

## 19. Privacy And Safety

Learning from real messages needs explicit controls.

Add settings:

- `mining_enabled` per user/target language.
- Optional per-chat `learningMiningEnabled` in `chats.settings`.
- Ability to delete a vocabulary card and source contexts.
- Ability to ignore a mined item.
- Do not mine messages from blocked users.
- Do not mine messages reported for moderation until resolved.
- Avoid saving full long messages as context. Store the sentence containing the
  term, not the whole chat transcript.

For human-to-human chats:

- Mine only for the learner who can already see the message.
- Do not expose another participant's private text in tutor/marketplace
  insights without explicit opt-in.

## 20. Backend Wiring

In `backend/cmd/server/main.go`:

1. Initialize curriculum service after DB and Redis.
2. Seed active Spanish course after migrations.
3. Initialize learning services.
4. Initialize word mining queue with:
   - DB
   - Redis
   - WordMiningService
   - Message lookup
   - WebSocket notifier
5. Start queue and defer stop.
6. Register learning routes under protected group.
7. Pass mining queue into `MessageHandler` or call enqueue from a message event
   hook.

Preferred event hook:

- Add optional callback to `MessageHandler` after successful message creation:
  `OnMessageCreated(ctx, message)`.
- This avoids making `MessageService` depend on learning services.

WebSocket events:

```text
learning_dashboard_updated
word_mining_completed
mined_item_created
scenario_turn_created
scenario_completed
streak_updated
```

These are convenience events. The client should still be able to refetch.

## 21. Testing Plan

Backend unit tests:

- `CurriculumService` seeds idempotently.
- `LearningCapabilityService` returns `full_course` for `en -> es` and
  `vocab_only` for unseeded but model-supported pairs.
- `LearningProfileService` creates default profile.
- `WordMiningService.NormalizeLearningTerm`.
- `WordMiningService.ClassifyAndRoute` for upcoming, current, completed, bonus.
- `WordMiningService.DedupeVocabulary`.
- `PracticeService.UpdateVocabAfterAttempt`.
- `SessionComposerService` includes due SRS first and interleaves items.
- `PlacementService` assigns expected levels.
- `FluencyScoreService` computes component scores.
- `ScenarioService` completes ordering coffee only after required intents.

Backend handler tests:

- Dashboard requires auth.
- Placement start/answer/skip.
- Session start/answer/complete.
- Mined item accept/ignore.
- Scenario start/message/complete.

Frontend tests:

- Learn dashboard renders API data.
- Placement MCQ disables check until selected.
- Session answer posts and advances.
- Scenario roleplay posts message and renders AI turn.
- Chat nudge sends text to input and records use.

E2E tests:

- Register/login user.
- Skip placement and land on A1.1 dashboard.
- Send Spanish message in chat.
- Mining creates candidate item.
- Accept mined item.
- Complete vocabulary review.
- Complete ordering coffee scenario.
- Dashboard score/streak/progress updates.

Commands to verify:

```powershell
cd C:\dev\chorus\backend
go test ./...

cd C:\dev\chorus\packages\shared
npm test -- --run

cd C:\dev\chorus\frontend
npm test -- --run
npm run build

cd C:\dev\chorus\mobile
npm test -- --runInBand
```

Use the repo's actual scripts if package.json names differ.

## 22. Implementation Phases And Acceptance Criteria

### Phase 1: Schema, Seed Curriculum, Profile

Tasks:

- Add schema from sections 4.0-4.15.
- Add Go models.
- Implement `learning_pair_capabilities`.
- Seed `en -> es` as `full_course`.
- Seed English-to-Spanish course A1-B2, units, grammar points, lexical items,
  lessons, and ordering coffee scenario.
- Return computed `vocab_only` capability for unseeded pairs where model-backed
  translation/grammar/mining can run.
- Add profile endpoints.

Acceptance:

- Backend boots and migrations are idempotent.
- New English-native user can fetch default profile for Spanish (`en -> es`).
- Curriculum path endpoint returns A1-B2 units for `en -> es`.
- A user selecting an unseeded pair, such as `es -> fr`, receives a
  `vocab_only` capability and no fake roadmap.

### Phase 2: Real Dashboard

Tasks:

- Add dashboard aggregation.
- Replace placeholder web/mobile Learn metrics.
- Add shared API/types.

Acceptance:

- `/learn` shows real data after login.
- Empty-state user sees A1 start, zero XP, no due words, streak 0.
- User with seeded activity sees correct dashboard metrics.

### Phase 3: Word Mining

Tasks:

- Add mining queue and service.
- Enqueue mining after messages.
- Add candidate list and accept/ignore endpoints.
- Upgrade manual save to populate new fields.

Acceptance:

- Sending/receiving Spanish text creates mining job.
- Mining results dedupe and classify.
- Manual saved word appears in dashboard due count.

### Phase 4: SRS And Sessions

Tasks:

- Add practice attempt history.
- Implement stage-aware SRS.
- Add session composer.
- Add session screens.

Acceptance:

- Due cards appear in a session.
- Correct/incorrect answers update stage, interval, and next review.
- Mastered count only includes production-level mastery.

### Phase 5: Lessons And Roadmap

Tasks:

- Implement lesson attempts.
- Implement unit progression.
- Implement roadmap screen.

Acceptance:

- User can start and complete a lesson.
- Completing lessons updates unit progress.
- Checkpoint gates next unit.

### Phase 6: Placement

Tasks:

- Add placement item bank.
- Implement adaptive placement endpoints.
- Add placement screens.
- Assign starting unit and skipped units.

Acceptance:

- User can skip and start at A1.1.
- User can complete placement and start at A2/B1/B2.
- Dashboard reflects assigned level and starting unit.

### Phase 7: Scenarios

Tasks:

- Implement scenario run lifecycle.
- Add AI response/evaluation prompts.
- Add scenario list and roleplay UI.
- Reinjection of scenario vocabulary into vocabulary pipeline.

Acceptance:

- User can complete ordering coffee.
- Roleplay advances phases.
- Correct target chunks update vocabulary/spontaneous use.
- Scenario completion updates roadmap and readiness score.

### Phase 8: Real Talk, Nudges, Corrections

Tasks:

- Generate prompts from active unit, weak grammar, and recent vocabulary.
- Add chat nudge UI.
- Connect grammar correction to `user_grammar_mastery`.
- Add "Practice Now" targeted quick drill.

Acceptance:

- Real Talk hub shows prompts by Icebreakers, Deep Dives, Task-Based.
- Nudge can send text to chat input.
- Production mistakes update grammar confidence and recommendations.

### Phase 9: Polish, Analytics, Privacy

Tasks:

- Settings controls.
- Delete/ignore flows.
- Activity events.
- Streak recovery.
- Loading, empty, error states.
- Public site and in-app marketing copy reflect language-pair support tiers.
- Mobile replaces hardcoded public/auth/learning strings with shared i18n keys
  or a mobile i18n layer.
- Waitlist/admin analytics capture demand for unsupported structured courses.

Acceptance:

- User can disable learning mining.
- Streak risk/recovery works.
- Analytics events support retention and learning-quality metrics.
- Public landing/about/pricing pages distinguish translation languages from
  structured learning courses.
- Unsupported pairs show course waitlist or vocab-only behavior, not placement
  or fake A1-B2 roadmap progress.
- RTL and long-string QA passes for public pages and learning availability
  states.

## 23. Metrics To Track

Product metrics:

- daily active learners,
- daily session completion,
- streak retention,
- placement completion vs skip,
- lesson completion,
- scenario completion,
- nudge acceptance/dismissal,
- chat-mined word acceptance.

Learning metrics:

- depth-of-processing progression rate,
- spontaneous-use rate,
- scenario-to-vocabulary reinjection rate,
- grammar confidence improvement,
- SRS overdue backlog,
- leech cards,
- readiness score movement per week.

Implementation detail:

- Store raw events in `user_activity_events`.
- Store daily aggregates in `daily_learning_stats`.
- Compute dashboard from aggregates plus live due counts.

## 24. Risks And Guardrails

Risk: mining too many useless words.

- Guardrail: teachability score, candidate state, proper noun filtering,
  per-message cap.

Risk: dashboard progress feels fake.

- Guardrail: mastery requires production, checkpoints gate level-up, score
  components are visible and explainable.

Risk: roleplay AI drifts above learner level.

- Guardrail: strict scenario prompt, max sentence length, allowed grammar list,
  phase-based script.

Risk: privacy concerns from mining personal chat.

- Guardrail: opt-out, per-chat setting, source sentence only, deletion support.

Risk: implementation tries to build every language at once.

- Guardrail: build the engine pair-generic, seed `en -> es` first, and expose
  support tiers so unseeded pairs use honest `vocab_only` behavior.

Risk: inline migrations get too large.

- Guardrail: keep changes idempotent and grouped with comments. If migration
  growth becomes painful, introduce a migration folder later, but do not block
  this feature on migration-framework refactoring.

## 25. First Concrete Implementation Slice

Start with this narrow vertical slice:

1. Add curriculum/profile/progress schema.
2. Add learning-pair capability support.
3. Seed English-to-Spanish A1.1-A1.4 only plus ordering coffee scenario.
4. Add `GET /learning/dashboard` and `GET /learning/path`.
5. Update web Learn dashboard to use the real endpoint and branch on
   `supportTier`.
6. Add manual saved word classification into A1.4 if it matches coffee terms.
7. Add `POST /learning/sessions/start` for vocabulary-only SRS.
8. Add a simple web vocabulary review screen.
9. Update landing/about/waitlist/register/settings copy so `en -> es` can start
   lessons and unseeded pairs see a clear waitlist/vocab-only state.
10. Add tests.

This proves the architecture before adding adaptive placement and full AI
scenario roleplay.

## 26. Final UX Acceptance Walkthrough

After full implementation:

1. User logs in on web or mobile.
2. User sees placement prompt if no placement status exists.
3. User skips or completes placement.
4. Learn dashboard shows current CEFR, readiness score, streak, daily goal,
   current unit, due vocabulary, grammar target, and scenario recommendation.
5. User receives a Spanish chat message.
6. The backend mines high-value words/chunks from that message.
7. Learn dashboard shows new words available for review.
8. User starts Quick Drills.
9. Session includes due SRS plus the new chat word in contextual cloze form.
10. User completes the active lesson and unlocks the next lesson.
11. User opens Scenarios and starts Ordering Coffee.
12. AI barista roleplay guides the user through greeting, order,
    customization, payment, and closing.
13. Scenario completion grants XP, updates roadmap, and reinjects used chunks
    into vocabulary.
14. User returns to Learn and sees updated readiness score and progress.
