# Chorus Language Learning — Comprehensive Design Plan

> **Purpose**: A science-backed, concrete specification for every learning activity in Chorus. This document covers the research foundations, the redesigned placement assessment, detailed activity designs, lesson plan templates, teacher-integration capabilities, and the technical approach (pre-authored vs. AI-realtime vs. hybrid) for each activity type.

---

## Table of Contents

1. [Scientific Foundations](#1-scientific-foundations)
2. [CEFR Level Framework](#2-cefr-level-framework)
3. [Redesigned Placement Assessment](#3-redesigned-placement-assessment)
4. [Core Learning Activities — Detailed Designs](#4-core-learning-activities)
   - 4.1 Vocabulary Acquisition (SRS + Depth-of-Processing)
   - 4.2 Grammar Instruction (Explicit + Implicit)
   - 4.3 Roleplay / Scenario Practice (CI + Output Hypothesis)
   - 4.4 Reading Comprehension (Input Hypothesis)
   - 4.5 Listening Comprehension
   - 4.6 Writing Tasks (Output + Feedback)
   - 4.7 Chat-Mined Vocabulary Review
   - 4.8 Daily Practice Loop
   - 4.9 Real-Talk Conversation Prompts
5. [Curriculum / Lesson Plan Templates](#5-curriculum--lesson-plan-templates)
   - 5.1 Unit Structure
   - 5.2 Lesson Type Catalogue
   - 5.3 Concrete Lesson Plans (A1–B2)
6. [Teacher Integration](#6-teacher-integration)
   - 6.1 Material Authoring
   - 6.2 Assigning Work to Students
   - 6.3 Progress Monitoring
   - 6.4 Feedback Loop
7. [Technical Implementation Strategy](#7-technical-implementation-strategy)
   - 7.1 Pre-Authored Content (seeded)
   - 7.2 AI-Generated On Demand
   - 7.3 Hybrid (Pre-authored frame + AI fill)
   - 7.4 Which Activities Use Which Approach
8. [Data Model Changes Required](#8-data-model-changes-required)
9. [AI Prompt Specifications](#9-ai-prompt-specifications)
10. [Quality & Correctness Guardrails](#10-quality--correctness-guardrails)

---

## 1. Scientific Foundations

Every activity design below traces to one or more peer-reviewed second-language acquisition (SLA) theories. Here is the theory map:

### 1.1 Krashen's Input Hypothesis (Comprehensible Input, i+1)
- Acquisition happens when learners receive input slightly above their current level ("i+1").
- **Application**: all reading/listening content and AI roleplay replies are calibrated to the learner's CEFR level ± 0.5 bands.
- **Implication for roleplay**: the AI partner must speak in short, clear sentences at A1–B1 and richer but still comprehensible sentences at B1–B2.

### 1.2 Swain's Output Hypothesis
- Producing language (speaking, writing) forces learners to notice gaps they don't encounter during comprehension.
- **Application**: every scenario phase requires the learner to produce a target-language turn before the AI advances. Free-recall and production stages in SRS also require production.
- **Implication for placement**: the placement test must include a production item (type-the-word), not only recognition (MCQ).

### 1.3 Long's Interaction Hypothesis
- Negotiation of meaning — where communication breaks down and repairs — is the most powerful catalyst for acquisition.
- **Application**: the AI partner deliberately creates communication breakdowns ("¿Cómo? No entiendo, ¿puedes repetir más despacio?") when the learner's input is ambiguous or incorrect.
- **Implication for scenario design**: phases should have multiple sub-intents so there is genuine back-and-forth, not a single keyword trigger.

### 1.4 Depth-of-Processing (Craik & Lockhart, 1972)
- Memory retention scales with how deeply an item is processed: recognition → recall → production → use-in-context.
- **Application**: the 5-stage mastery ladder (recognition → cued_recall → free_recall → production → spontaneous). Items must pass through each stage before being considered mastered.
- **Implication for SRS**: skipping stages (e.g., going straight to production at stage 1) is prohibited.

### 1.5 Spaced Repetition + SM-2 (Ebbinghaus Forgetting Curve)
- Items reviewed at expanding intervals just before forgetting are retained with minimal review cost.
- **Application**: SM-2 scheduling on all vocabulary cards and grammar points.
- **Key parameters**: initial interval 1 day (new), ease factor 2.5, ± 0.1/0.15 per correct/incorrect answer, minimum ease 1.3, minimum interval 1 day.

### 1.6 Interleaving (Kornell & Bjork, 2008)
- Mixed practice (A, B, C, A, C, B) produces better long-term retention than blocked practice (A, A, A, B, B, B).
- **Application**: daily sessions always interleave vocabulary items, grammar cloze, and lesson-step items. No two consecutive items of the same type.

### 1.7 Retrieval Practice (Roediger & Karpicke, 2006)
- Retrieving information from memory strengthens it more than re-reading.
- **Application**: every SRS review is a retrieval attempt (never show the answer first). Immediate corrective feedback is shown only after the learner commits an answer.

### 1.8 Noticing Hypothesis (Schmidt, 1990)
- Learners must consciously notice a form in input for it to be acquired.
- **Application**: grammar correction in roleplay uses an error span + correction + brief one-line explanation (not a wall of grammar theory). Mining candidates show the word in its original message context ("You saw this in: 'Voy al mercado'").

### 1.9 Task-Based Language Teaching (Willis 1996, Ellis 2003)
- Real communicative tasks (with information gap, choice, feedback) outperform drill-only approaches.
- **Application**: scenarios are designed as genuine tasks with a goal (order coffee, buy a bus ticket), not as vocabulary drills in disguise.

### 1.10 Lexical Approach (Lewis, 1993)
- Language is best acquired as chunks (multi-word units, collocations, idioms), not word-by-word.
- **Application**: the chunk bank in scenario phases explicitly teaches multi-word expressions. Word mining extracts chunks (`is_chunk: true`). SRS cards can be chunks.

---

## 2. CEFR Level Framework

| Level | Label          | Can-Do Summary                                                      | Readiness Score |
|-------|----------------|---------------------------------------------------------------------|-----------------|
| A1    | True Beginner  | Greet, introduce self, order food, handle numbers 1–20              | 0–249           |
| A2    | Elementary     | Shopping, simple past events, simple directions, family/daily life  | 250–549         |
| B1    | Intermediate   | Travel, work, opinions, plans, simple arguments                     | 550–799         |
| B2    | Upper-Intermediate | Abstract topics, current affairs, extended arguments, formal writing | 800–1000  |

**Level boundaries in the placement score (0–1000)**:
- A1: 0–249 (ability < 250)
- A2: 250–549 (ability 250–549)
- B1: 550–799 (ability 550–799)
- B2: 800–1000 (ability ≥ 800)

**Sub-levels for unit design** (each level is divided into 4 units, numbered 1–4):
- A1.1–A1.4, A2.1–A2.4, B1.1–B1.4, B2.1–B2.4 → 16 units per language pair

---

## 3. Redesigned Placement Assessment

### 3.1 Problems With the Current Test

The current test has these specific deficiencies:
1. **Only recognition items** (MCQ "which Spanish word means X?") — violates Output Hypothesis; a student can score B2 by guessing.
2. **Fixed hardcoded A1/A2/B1/B2 items** in `buildPlacementFallback()` are only Spanish grammar, not vocabulary-based placement.
3. **No discourse or production sample** — the test can't distinguish an A2 learner from a B1 learner on real output.
4. **3 items per level** is too few for statistical reliability; IRT requires ≥ 5 items per band for reasonable SEM.
5. **Prompt text mixes task type and grammar explanation** ("Present Tense: Complete…") — scaffolding leaks the answer.
6. **No listening/visual input** — placement is purely text-based even for learners who are stronger aurally.

### 3.2 Redesigned Placement Protocol

**Total: 20 items across 3 modules** — takes ≈ 8–12 minutes.

#### Module A: Receptive Vocabulary (8 items, MCQ, adaptive)
Goal: establishes initial ability band.
Format: a target-language word is shown with a sentence context. The learner picks the correct native-language translation from 4 options (1 correct, 3 distractors from same POS, different frequency band).

Adaptive rule: start at A1. If ≥ 2/3 correct in a band, move up. If < 2/3, stay.

Item structure:
```json
{
  "module": "receptive_vocab",
  "item_type": "L2_to_L1_context",
  "prompt": {
    "sentence": "El niño tiene ____ años.",
    "target_word": "cinco",
    "context_translation": "The child is ___ years old."
  },
  "choices": ["five", "fifteen", "yesterday", "many"],
  "correct": "five",
  "cefr": "A1"
}
```

#### Module B: Grammatical Judgment + Production (8 items, mixed)
Goal: measures grammatical accuracy without telegraphing the rule.
Formats:
- **Odd-one-out** (4 sentences, 1 grammatically wrong): "Which sentence has an error?"
- **Gap-fill** (type a word, not select): "Cuando era niño, siempre _____ (vivir) en Madrid." → learner types `vivía`
- **Sentence reconstruction**: 5 scrambled words, learner puts them in order (drag or type)

Scoring: typed/reconstructed answers are normalized (whitespace, accents ignored for A1–A2; accents required for B1+).

Item structure (gap-fill example):
```json
{
  "module": "grammar_production",
  "item_type": "gap_fill_type",
  "prompt": {
    "sentence_with_blank": "Ayer nosotros _____ (ir) a la playa.",
    "hint": null,
    "context": "Talking about yesterday."
  },
  "correct": "fuimos",
  "accept_variants": ["Fuimos"],
  "cefr": "A2"
}
```

#### Module C: Discourse Comprehension (4 items, reading)
Goal: discriminates B1+ from A2 with a short paragraph.
Format: a 3–5 sentence paragraph in the target language at B1/B2 level. 2 MCQ comprehension questions. Learner can tap any word for a gloss (tapping a word does NOT penalise the score — it's a usage signal for calibration).

```json
{
  "module": "discourse_reading",
  "item_type": "passage_mcq",
  "passage": "La semana pasada, mi jefa me pidió que presentara...",
  "questions": [
    {
      "q": "What did the boss ask?",
      "choices": ["To present a report", "To go on a trip", "To hire someone", "To cancel a meeting"],
      "correct": "To present a report"
    }
  ],
  "cefr": "B1"
}
```

### 3.3 IRT-Lite Scoring (improved)

```
ability_start = 250 (A2 lower bound — safer prior for a language-learning app)
per_item_update:
  correct:   ability += 80 * (1 - P(correct | ability, item_difficulty))
  incorrect: ability -= 80 * P(correct | ability, item_difficulty)

P(correct | θ, b) = 1 / (1 + e^(-1.7 * (θ - b) / 200))

where θ = ability (0-1000), b = item difficulty midpoint per CEFR:
  A1 = 125, A2 = 375, B1 = 650, B2 = 875
```

### 3.4 Self-Selection Path (keep, but improve)

The current "beginner / intermediate / advanced" self-selection is fine as a fallback but should be replaced with a **5-question quick screen** ("How well can you do X?") that maps to a CEFR band:

```
Q1: Can you introduce yourself in the target language? (Yes / A bit / No)
Q2: Can you talk about past events? (Yes / A bit / No)
Q3: Can you express opinions on abstract topics? (Yes / A bit / No)
Q4: Have you formally studied the language? (Never / 1-2 years / 3+ years)
Q5: Do you use the language at work or in daily life? (Yes / Sometimes / No)
```

This takes 30 seconds and gives a more reliable band than a 3-button pick.

### 3.5 Implementation Approach
- **Pre-authored**: item banks are seeded into `placement_attempts` table items. The Spanish item bank should have ≥ 20 items per band (80 total) so the adaptive engine can sample without repeating. New language pairs seed their own items.
- **No AI at runtime for placement questions** — scoring must be deterministic and auditable. AI is used only to generate additional placement item candidates offline (batch job) for admin review before seeding.

---

## 4. Core Learning Activities

### 4.1 Vocabulary Acquisition (SRS + Depth-of-Processing)

#### Theory basis: DoP ladder (Craik & Lockhart), SM-2, retrieval practice

**The 5-Stage Mastery Ladder** (maps to existing `mastery_stage` field):

| Stage | Name           | Activity Type          | Prompt Example                              | Correct → Next Stage | Wrong → Stay |
|-------|----------------|------------------------|---------------------------------------------|----------------------|--------------|
| 1     | Recognition    | `multiple_choice`      | "What does *hablar* mean?" (4 choices)      | 2 consecutive correct | Reset ease   |
| 2     | Cued Recall    | `fill_in_blank`        | "Yo ____ español. (to speak)"               | 2 consecutive correct | Stay stage 1 |
| 3     | Free Recall    | `type_translation`     | "Type the Spanish word for: to speak"       | 2 consecutive correct | Back to 2    |
| 4     | Production     | `use_in_sentence`      | "Write a sentence using *hablar*"           | 1 AI-graded correct   | Stay stage 3 |
| 5     | Spontaneous    | (auto-detected)        | Counted when word appears in chat/scenario  | Permanent             | —            |

**Stage 1 — Multiple Choice (Recognition)**

Item structure:
```json
{
  "activity_type": "multiple_choice",
  "prompt": {
    "text": "What does this word mean?",
    "term": "hablar",
    "context_sentence": "Me gusta hablar con mis amigos.",
    "context_translation": "I like to ___ with my friends."
  },
  "choices": ["to speak", "to listen", "to write", "to walk"],
  "correct": "to speak"
}
```

Distractors: always from the same POS, same CEFR band, different semantic field (not synonyms).

**Stage 2 — Cued Recall (Fill-in-Blank)**

```json
{
  "activity_type": "fill_in_blank",
  "prompt": {
    "text": "Complete the sentence:",
    "sentence": "Yo _____ español con mi familia.",
    "hint_translation": "(I ___ Spanish with my family.)"
  },
  "correct": "hablo",
  "accept_variants": ["Hablo"]
}
```

**Stage 3 — Free Recall (Type Translation)**

```json
{
  "activity_type": "type_translation",
  "prompt": {
    "text": "How do you say this in Spanish?",
    "translation": "to speak"
  },
  "correct": "hablar",
  "accept_variants": ["Hablar"]
}
```

**Stage 4 — Production (Use in Sentence)**

```json
{
  "activity_type": "use_in_sentence",
  "prompt": {
    "text": "Write a sentence using the word:",
    "term": "hablar",
    "example_context": "Maybe talk about who you speak Spanish with."
  },
  "grading": "ai_grammar_check"
}
```

Grading: sent to the AI grammar service. The AI must return:
- `correct: bool` (did the learner use the word correctly in context?)
- `feedback: string` (1 sentence, in the native language)
- `error_spans: [{span, correction, explanation}]` (optional)

**Stage 5 — Spontaneous** is automatically credited when `touchSpontaneousUse` detects the word in a chat message or scenario turn.

**SM-2 Scheduling parameters** (currently implemented but parameters need tuning):
- Initial interval: 1 day (stage 1), 3 days (stage 2–3), 7 days (stage 4)
- Ease factor range: 1.3–2.5, default 2.5
- Quality 0: interval reset to 1, ease -= 0.2
- Quality 1: interval unchanged, ease -= 0.15
- Quality 2: interval * 1.2, ease unchanged
- Quality 3: interval * ease, ease += 0.1
- Minimum interval: 1 day

---

### 4.2 Grammar Instruction

#### Theory basis: Noticing Hypothesis (Schmidt), explicit + focus-on-form

**Grammar in Chorus has three delivery modes**:

#### Mode A: Incidental Grammar Notes (triggered by errors)
When the AI grammar service flags an error in a learner's message or scenario turn:
1. Show the error span underlined.
2. Show the correction in green.
3. Show a one-sentence rule note ("Spanish verbs agree with their subject: *yo hablo*, *ella habla*").
4. Add a grammar point card to the learner's SRS queue if they haven't mastered this point yet.

This is already partially implemented via `ScenarioError`. The missing piece is the grammar point database.

#### Mode B: Grammar Cloze Sessions (explicit drill)
Grammar points are standalone SRS items scheduled with SM-2 (like vocabulary).

A grammar point has:
```json
{
  "id": "es-present-tense-ar-verbs",
  "cefr_level": "A1",
  "title": "Present Tense: -ar Verbs",
  "rule": "Remove -ar, add: -o, -as, -a, -amos, -áis, -an",
  "example_table": {
    "yo": "hablo", "tú": "hablas", "él/ella": "habla",
    "nosotros": "hablamos", "vosotros": "habláis", "ellos": "hablan"
  },
  "items": [
    {
      "type": "cloze",
      "prompt": "Ella _____ (hablar) tres idiomas.",
      "correct": "habla",
      "distractor_note": "Don't use -o; that's for yo."
    },
    {
      "type": "cloze",
      "prompt": "Nosotros _____ (caminar) al parque.",
      "correct": "caminamos"
    },
    {
      "type": "multiple_choice",
      "prompt": "Which is correct for 'they speak'?",
      "choices": ["hablas", "hablan", "hablamos", "hablo"],
      "correct": "hablan"
    }
  ]
}
```

Grammar cloze items are injected into daily sessions (max 4 per session, never back-to-back per interleaving rule).

#### Mode C: Grammar Explanation Cards (inductive presentation)
Before drilling a grammar point, learners see a **micro-lesson card** (not a quiz — just reading):
- 3 example sentences using the target form (positive, negative, question)
- One-line rule
- One "trap" (common error)

This satisfies the Noticing Hypothesis: learners must notice the form before they are drilled on it.

**Implementation**: grammar points are pre-authored in the DB (`grammar_points` table, one row per CEFR-banded point). The Spanish course needs ≈ 60 grammar points (A1: 12, A2: 15, B1: 18, B2: 15). See Section 5.3 for the complete list.

---

### 4.3 Roleplay / Scenario Practice

#### Theory basis: TBLT (Ellis), Interaction Hypothesis (Long), Output Hypothesis (Swain)

This is the most complex activity. Here is the complete redesigned specification.

#### Scenario Anatomy

```
ScenarioScript
  ├── metadata: title, domain, cefr_level, can_do_statement, estimated_minutes
  ├── ai_role: name, description, personality, language_register
  ├── opening_line: first AI utterance (in target language)
  ├── phases[]: ordered sequence of communicative goals
  │     ├── title: what happens in this phase
  │     ├── learner_goal: what the learner must communicate (shown as hint)
  │     ├── required_intents[]: string tags that must be detected to complete phase
  │     ├── chunk_bank[]: {text, translation} multi-word expressions for this phase
  │     ├── scaffold_hints[]: ordered hints if learner is stuck (shown on request)
  │     └── success_examples[]: example sentences that cover all intents (for AI grader)
  └── completion_criteria: how many phases must be completed, minimum score
```

#### Phases vs. Turns

A phase maps to a **communicative goal** (e.g., "Greet the barista and say how many coffees you want"). It is NOT one turn. The AI should loop within a phase — prompting, re-asking, repairing — until the learner's output covers all required intents. Only then does the scenario advance.

This is a key fix from the current implementation which advances on ANY intent match.

#### Intent Detection

Current implementation uses keyword matching. This is inadequate for B1+. The redesigned approach:

- **A1–A2 levels**: keyword matching (deterministic, no latency) — sufficient because vocabulary is limited.
- **B1–B2 levels**: AI intent classification (async, with 500ms timeout + keyword fallback).

The AI intent classifier prompt:
```
Given the learner's message and the required intents for this phase, 
return a JSON array of covered intent tags.

Required intents: ["greet_barista", "state_coffee_count", "specify_size"]
Learner message: "Hola! Quiero dos cafés grandes, por favor."

Return: ["greet_barista", "state_coffee_count", "specify_size"]
```

#### Scaffold Levels

Three scaffold levels, assigned based on CEFR and learner performance:

1. **Guided**: chunk_bank is always visible. Scaffold hints shown after 30 seconds of no input.
2. **Supported**: chunk_bank revealed on first tap of "Hint" button. Hints shown after 60 seconds.
3. **Independent**: no chunk bank shown. Hints available behind 2 taps.

Learners start at **guided** for their first scenario at a given CEFR level, and advance to supported/independent as they complete scenarios successfully.

#### AI Partner Behavior Rules

The AI partner is instructed (via system prompt) to:
1. Stay in the target language at all times (except for translations shown below the message).
2. Use only vocabulary and structures at the learner's CEFR level ± 0.5 bands.
3. Keep replies to 1–2 sentences.
4. If the learner makes a grammatical error: complete the exchange naturally, then add a gentle correction in parentheses: *(Nota: "soy cansado" → "estoy cansado" — feelings use "estar")*
5. If the learner's message doesn't cover the required intent: ask a clarifying question rather than advancing.
6. If the learner writes in the native language: respond in the target language and add: *(¡Intenta responder en español!)*
7. Never complete the learner's task for them (don't order the coffee on their behalf).

#### Complete Scenario Catalogue (Initial 12 Scenarios, A1–B2)

**A1 Scenarios (4)**

| ID | Title | Domain | AI Role | Can-Do Statement | Phases |
|----|-------|--------|---------|------------------|--------|
| `order-coffee` | Ordering Coffee | Café | Barista | Can order a coffee, specify size and type, pay | Greet → Order → Customise → Price → Farewell |
| `introduce-yourself` | Self Introduction | Social | New acquaintance | Can state name, origin, age, occupation | Greet → Name → Origin → Occupation → Wrap |
| `buy-ticket` | Bus Ticket Purchase | Transport | Ticket agent | Can ask for a ticket to a destination, ask price | Greet → Destination → Quantity → Price → Confirm |
| `at-the-market` | At the Market | Shopping | Market vendor | Can ask for produce, state quantity, understand price | Greet → Ask item → Quantity → Price → Pay |

**A2 Scenarios (4)**

| ID | Title | Domain | AI Role | Can-Do Statement | Phases |
|----|-------|--------|---------|------------------|--------|
| `restaurant-order` | Restaurant Dinner | Restaurant | Waiter | Can read a menu, order for 2, ask about dishes, pay | Greet → Menu questions → Order → Special requests → Bill |
| `hotel-checkin` | Hotel Check-In | Accommodation | Receptionist | Can check in, state needs, report a problem | Arrive → Identity → Room → Request → Problem |
| `doctor-visit` | Doctor's Visit | Health | Doctor | Can describe symptoms, understand advice, make appointment | Greet → Symptoms → Questions → Advice → Appointment |
| `phone-call-friend` | Calling a Friend | Social | Friend | Can initiate a call, make plans, suggest alternatives | Open → Plans → Agree/Disagree → Arrange details → Close |

**B1 Scenarios (2)**

| ID | Title | Domain | AI Role | Can-Do Statement | Phases |
|----|-------|--------|---------|------------------|--------|
| `job-interview` | Job Interview | Work | Interviewer | Can talk about experience, skills, salary expectations | Open → Background → Skills → Scenario Q → Close |
| `apartment-hunt` | Flat Viewing | Housing | Landlord | Can ask about flat features, negotiate terms, express preferences | Arrive → Questions → Negotiate → Commit/Decline |

**B2 Scenarios (2)**

| ID | Title | Domain | AI Role | Can-Do Statement | Phases |
|----|-------|--------|---------|------------------|--------|
| `debate-topic` | Discuss a News Story | Current affairs | Educated peer | Can express and defend a position with supporting arguments | Introduce topic → State position → Counter → Concede → Conclude |
| `formal-complaint` | Making a Formal Complaint | Customer service | Manager | Can describe a problem formally, insist politely, reach resolution | Identify → Describe → Escalate → Negotiate → Resolve |

#### Detailed Phase Design — "Ordering Coffee" (A1) 

```json
{
  "id": "order-coffee",
  "title": "Ordering Coffee",
  "domain": "café",
  "cefr_level": "A1",
  "can_do_statement": "Can order a coffee, specify type and size, and handle payment.",
  "estimated_minutes": 5,
  "ai_role_name": "Marco",
  "ai_role_description": "A friendly barista at a Madrid café. Speaks at A1 level. Cheerful. Occasionally mishears orders to create negotiation opportunities.",
  "opening_line": "¡Buenos días! ¿Qué le pongo?",
  "phases": [
    {
      "ordinal": 1,
      "title": "Greeting",
      "learner_goal": "Greet Marco and say you want a coffee.",
      "required_intents": ["greet", "want_coffee"],
      "chunk_bank": [
        {"text": "Buenos días", "translation": "Good morning"},
        {"text": "Hola", "translation": "Hello"},
        {"text": "Quiero un café", "translation": "I want a coffee"},
        {"text": "Me pone un café", "translation": "A coffee, please (formal)"},
        {"text": "Un café, por favor", "translation": "One coffee, please"}
      ],
      "scaffold_hints": [
        "Try saying 'Hola' first, then 'Quiero un café'.",
        "You can say: 'Hola, quiero un café, por favor.'"
      ],
      "success_examples": [
        "Hola, quiero un café.",
        "Buenos días, me pone un café por favor.",
        "Un café, por favor."
      ]
    },
    {
      "ordinal": 2,
      "title": "Coffee Type",
      "learner_goal": "Tell Marco what type of coffee you want (espresso, con leche, cortado…).",
      "required_intents": ["specify_coffee_type"],
      "chunk_bank": [
        {"text": "un café solo", "translation": "an espresso"},
        {"text": "un café con leche", "translation": "a white coffee"},
        {"text": "un cortado", "translation": "a cortado (espresso with a little milk)"},
        {"text": "un americano", "translation": "an Americano"},
        {"text": "un cappuccino", "translation": "a cappuccino"}
      ],
      "scaffold_hints": [
        "Marco is asking what type. Say 'Un café con leche, por favor.'",
        "Some options: solo (espresso), con leche (white), cortado (with a little milk)."
      ],
      "ai_follow_up_if_stuck": "¿Solo, con leche, o cortado?"
    },
    {
      "ordinal": 3,
      "title": "Size",
      "learner_goal": "Tell Marco the size you want.",
      "required_intents": ["specify_size"],
      "chunk_bank": [
        {"text": "pequeño", "translation": "small"},
        {"text": "mediano", "translation": "medium"},
        {"text": "grande", "translation": "large"}
      ],
      "ai_follow_up_if_stuck": "¿Grande, mediano o pequeño?"
    },
    {
      "ordinal": 4,
      "title": "Payment",
      "learner_goal": "Ask how much it costs and say you will pay.",
      "required_intents": ["ask_price", "agree_to_pay"],
      "chunk_bank": [
        {"text": "¿Cuánto es?", "translation": "How much is it?"},
        {"text": "¿Cuánto cuesta?", "translation": "How much does it cost?"},
        {"text": "Aquí tiene", "translation": "Here you go"},
        {"text": "Con tarjeta", "translation": "By card"},
        {"text": "En efectivo", "translation": "In cash"}
      ]
    },
    {
      "ordinal": 5,
      "title": "Farewell",
      "learner_goal": "Say goodbye to Marco.",
      "required_intents": ["farewell"],
      "chunk_bank": [
        {"text": "Gracias", "translation": "Thank you"},
        {"text": "Hasta luego", "translation": "See you later"},
        {"text": "Adiós", "translation": "Goodbye"},
        {"text": "Que tenga un buen día", "translation": "Have a good day"}
      ]
    }
  ],
  "completion_criteria": {
    "min_phases_completed": 4,
    "min_score": 300
  }
}
```

#### Detailed Phase Design — "Job Interview" (B1)

```json
{
  "id": "job-interview",
  "title": "Job Interview",
  "domain": "work",
  "cefr_level": "B1",
  "ai_role_name": "Dra. Sánchez",
  "ai_role_description": "HR manager at a tech company. Professional, friendly. Asks follow-up questions. Speaks in clear B1 Spanish.",
  "opening_line": "Buenas tardes, gracias por venir. Cuénteme un poco sobre usted y su experiencia profesional.",
  "phases": [
    {
      "ordinal": 1,
      "title": "Background",
      "learner_goal": "Introduce yourself professionally: name, background, and why you are interested in this job.",
      "required_intents": ["state_name", "describe_background", "express_interest"],
      "chunk_bank": [
        {"text": "Me llamo… y tengo experiencia en…", "translation": "My name is… and I have experience in…"},
        {"text": "Estoy interesado/a en este puesto porque…", "translation": "I'm interested in this position because…"},
        {"text": "Llevo X años trabajando en…", "translation": "I've been working in… for X years"}
      ]
    },
    {
      "ordinal": 2,
      "title": "Skills",
      "learner_goal": "Describe your top 2 skills and give a concrete example of each.",
      "required_intents": ["describe_skill_1", "describe_skill_2", "give_example"],
      "chunk_bank": [
        {"text": "Una de mis fortalezas es…", "translation": "One of my strengths is…"},
        {"text": "Por ejemplo, en mi trabajo anterior…", "translation": "For example, in my previous job…"},
        {"text": "Soy bueno/a trabajando en equipo", "translation": "I'm good at working in a team"}
      ]
    },
    {
      "ordinal": 3,
      "title": "Situational Question",
      "learner_goal": "Answer a hypothetical: 'If you had a conflict with a colleague, what would you do?'",
      "required_intents": ["describe_strategy", "express_willingness_to_collaborate"],
      "chunk_bank": [
        {"text": "En ese caso, hablaría directamente con…", "translation": "In that case, I would speak directly with…"},
        {"text": "Intentaría encontrar una solución…", "translation": "I would try to find a solution…"},
        {"text": "Es importante comunicarse con respeto", "translation": "It's important to communicate respectfully"}
      ]
    },
    {
      "ordinal": 4,
      "title": "Salary & Close",
      "learner_goal": "Ask about salary expectations and close the interview professionally.",
      "required_intents": ["ask_about_salary_or_conditions", "close_professionally"],
      "chunk_bank": [
        {"text": "¿Podría hablarme sobre el salario?", "translation": "Could you tell me about the salary?"},
        {"text": "¿Hay posibilidades de crecimiento?", "translation": "Are there growth opportunities?"},
        {"text": "Ha sido un placer. Espero sus noticias.", "translation": "It's been a pleasure. I look forward to hearing from you."}
      ]
    }
  ]
}
```

---

### 4.4 Reading Comprehension

#### Theory basis: Input Hypothesis (Krashen), Extensive Reading (Nation & Wang)

Reading activities are delivered as **lesson steps** (type `reading_passage`).

**Format**:
- A 50–150 word passage in the target language, at the learner's CEFR level.
- Every content word is tappable → shows gloss (translation + part of speech) without leaving the passage.
- 2–4 comprehension questions (MCQ at A1–A2, short-answer at B1–B2).
- After questions: a "Vocabulary from this passage" panel showing 3–5 mined words with Add to SRS button.

**CEFR calibration guidelines**:
- A1: max 50 words, present tense only, high-frequency vocabulary (top 500 words), simple sentences.
- A2: max 80 words, past/future, vocabulary top 1000.
- B1: max 120 words, complex sentences, subordinate clauses, vocabulary top 3000.
- B2: max 150 words, idiomatic language, low-frequency vocabulary, abstract topics.

**Implementation**: passages are pre-authored and seeded into `lesson_steps` with type `reading_passage`. The learner's CEFR level selects from passages at that level. New passages can be generated by AI offline and reviewed by an editor (or teacher for teacher-assigned material).

---

### 4.5 Listening Comprehension

#### Theory basis: Input Hypothesis, authentic input (Rost, 2011)

**Planned design** (requires TTS integration):

- A short audio clip (10–30 seconds) of the AI partner reading a text at the learner's CEFR level.
- Transcript is hidden initially; learner answers 1–2 MCQ.
- "Show transcript" button reveals the full text after answering.
- Vocabulary panel shown after.

**TTS approach**: Use the existing AI provider chain to generate audio via OpenAI TTS or equivalent. The synthesised audio is cached per script text, so the same lesson audio isn't regenerated on every request.

**This is a future activity type**. The current architecture must add:
- `lesson_step` type `listening_passage`
- A TTS service wrapper
- Audio file storage (CDN or S3)

---

### 4.6 Writing Tasks

#### Theory basis: Output Hypothesis (Swain), written corrective feedback (Ferris, 2010)

**Short Writing Tasks** (integrated into lessons and scenarios):

- Prompt: a context situation + a task ("Write a text message to your friend explaining you will be 10 minutes late.")
- Learner types a 2–5 sentence response.
- AI grades the response for: vocabulary range, grammatical accuracy, task completion.
- Returns: a corrected version of the text with inline error marks, a score 0–10, and 1–2 improvement tips.

**AI Grading Prompt** (for a B1 writing task):
```
You are a Spanish language teacher grading a B1-level short writing task.

Task: {{task_description}}
Learner's response: {{learner_text}}

Return strict JSON:
{
  "score": 0-10,
  "task_completed": true/false,
  "corrected_text": "...",
  "error_spans": [{"original": "...", "correction": "...", "rule": "..."}],
  "vocabulary_comment": "...",
  "grammar_comment": "...",
  "improvement_tip": "..."
}

Grading rubric:
- Task completion: 3 points
- Grammatical accuracy: 4 points  
- Vocabulary range and appropriacy: 2 points
- Coherence: 1 point
```

**Implementation approach**: AI-realtime for grading (POST to AI endpoint on submit). The task prompt itself is pre-authored.

---

### 4.7 Chat-Mined Vocabulary Review

#### Theory basis: Incidental vocabulary learning (Nagy, 1985), context advantage (Sternberg, 1987)

The word mining system already works. The missing design is:

**Mining Review UI**:
1. After a chat session where ≥ 3 words were mined, a card appears: "We noticed 3 new words in your conversation. Want to add them?"
2. Each candidate shows:
   - The word / chunk in target language
   - Its translation
   - The original message context (highlighted in the chat bubble)
   - CEFR level badge
   - [Add to my deck] [Skip]
3. Added words enter the SRS at stage 1 (recognition), with `source_type = 'chat_mined'`.

**Key design decision**: mining should also extract from **messages received** (what the native speaker said to the learner), not just messages sent. This is richer vocabulary at the learner's i+1 level.

**Mining from received messages**:
- Already partially supported (`source_type` can be `chat_received`).
- The mining job should be triggered on inbound message delivery too, not only on outbound send.
- Mined items from received messages are labelled "From [contact name]" in the review UI to provide social motivation.

---

### 4.8 Daily Practice Loop

#### Theory basis: SM-2, interleaving (Kornell & Bjork), habit loop (Duhigg)

The daily session is the core learning habit. It must be:
- Completable in 5–15 minutes
- Predictable in structure (same flow every day) to build habit
- Varied in content (never the same items twice in a row)

**Daily Session Structure** (15 items, interleaved):

```
1.  Vocabulary review — due SRS item (stage N)
2.  Grammar cloze — weakest grammar point
3.  Vocabulary review — due SRS item
4.  Vocabulary — new item from current unit (stage 1)
5.  Grammar cloze — same point, different item
6.  Vocabulary review — due SRS item
7.  Lesson step — next step from current lesson
8.  Vocabulary review — due SRS item
9.  Grammar review — second weakest point
10. Vocabulary — new item from current unit (stage 1)
11. Vocabulary review — due SRS item
12. Lesson step — next step
13. Grammar cloze
14. Vocabulary review
15. Streak/summary
```

Rules enforced by `SessionComposerService`:
- No two items of the same type consecutively (already enforced).
- Min 5 vocabulary items per session.
- If fewer than 5 due SRS items, fill with new items from current unit.
- If no grammar points seeded yet, skip grammar slots.
- Grammar cloze items capped at 4 per session (already enforced).

**Daily Goal Calculation**:
- Default: 15 items/day.
- User can set 5 / 10 / 15 / 20 / 25.
- Progress bar on dashboard shows N/target items.

---

### 4.9 Real-Talk Conversation Prompts

#### Theory basis: Conversation practice, willingness to communicate (WTC, MacIntyre et al.)

Real-Talk prompts encourage learners to use the target language in their real chats (with language partners or teachers). The current implementation has 3 hardcoded prompts.

**Redesigned Real-Talk prompt system**:

Prompts are generated daily by AI, calibrated to CEFR level and daily topic. Structure:

```json
{
  "id": "uuid",
  "cefr_level": "A2",
  "topic": "weekend_plans",
  "prompt_for_learner": "Next time you message a friend, try asking: '¿Qué planes tienes para el fin de semana?'",
  "target_phrase": "¿Qué planes tienes para el fin de semana?",
  "why_useful": "This is one of the most common conversation openers in Spanish.",
  "follow_up_chunks": [
    {"text": "Yo voy a…", "translation": "I'm going to…"},
    {"text": "¿Y tú?", "translation": "And you?"}
  ]
}
```

Three categories:
1. **Conversation starters** (A1–A2): phrases to begin a conversation
2. **Topic injectors** (A2–B1): phrases to pivot a conversation to a new topic
3. **Opinion phrases** (B1–B2): how to agree, disagree, qualify, emphasize

Implementation: A daily cron job generates 3–5 prompts per CEFR level per language pair using AI, stores them in a `real_talk_prompts` table. If AI is unavailable, fall back to a pre-seeded bank of ≥ 50 prompts per level.

---

## 5. Curriculum / Lesson Plan Templates

### 5.1 Unit Structure

Each CEFR unit has:
- 1 can-do statement (what the learner can do upon completion)
- 4–6 lessons
- 1 unit checkpoint (end-of-unit assessment)
- 1 scenario (from the scenario catalogue)
- ≈ 30 vocabulary items
- ≈ 5 grammar points

**Unit template**:
```
Unit: [CEFR].[Unit#] — [Title]
Can-Do: "By the end of this unit, I can [action]."
Vocabulary set: 30 items (sourced from lexical_items table)
Grammar points: 5 items (sourced from grammar_points table)

Lesson 1: Introduction (new vocabulary, recognition)
  - Lesson type: vocabulary_introduction
  - Steps: 10 vocabulary cards (stage 1 MCQ), 2 reading passages

Lesson 2: Grammar Focus
  - Lesson type: grammar_focus
  - Steps: grammar card (rule explanation), 8 cloze items, 2 sentence reconstructions

Lesson 3: Reading + Vocabulary in Context
  - Lesson type: reading_practice
  - Steps: 2 reading passages (longer), 5 vocabulary fill-in-blank

Lesson 4: Production Practice
  - Lesson type: production
  - Steps: 5 free-recall vocabulary, 2 short writing tasks

Lesson 5: Scenario Roleplay
  - Lesson type: scenario
  - Steps: links to scenario in catalogue (scenario_id)

Lesson 6 (optional): Review & Checkpoint
  - Lesson type: review
  - Steps: 15-item mixed session (vocab + grammar + lesson steps)

Unit Checkpoint (required for CheckpointRequired units):
  - 20-item mixed assessment: 8 vocab, 6 grammar, 4 reading MCQ, 2 writing
  - Passing score: 70%
  - Can be retried after 24h
```

### 5.2 Lesson Type Catalogue

| Type | Description | Typical Steps | Session Mode |
|------|-------------|---------------|--------------|
| `vocabulary_introduction` | Introduce 10 new items at recognition stage | MCQ + reading passage | `lesson` |
| `grammar_focus` | Explicit grammar point + drill | Rule card + cloze | `lesson` |
| `reading_practice` | Extended reading with questions | Passage + MCQ + vocab mine | `lesson` |
| `production` | Output-focused writing and free recall | Writing task + free recall | `lesson` |
| `scenario` | Real-world roleplay | Links to scenario | `scenario` |
| `review` | Spaced retrieval of prior items | Mixed SRS | `daily` |
| `checkpoint` | End-of-unit assessment | Mixed test | `lesson` |

### 5.3 Concrete Lesson Plans (A1 Unit 1 — Introductions)

**Unit: A1.1 — Greetings and Introductions**  
Can-Do: "I can greet people, introduce myself, and ask basic personal questions."

**Vocabulary Set (30 items)**:

| Term | Translation | POS | CEFR |
|------|-------------|-----|------|
| hola | hello | interjection | A1 |
| buenos días | good morning | phrase | A1 |
| buenas tardes | good afternoon | phrase | A1 |
| buenas noches | good evening | phrase | A1 |
| adiós | goodbye | interjection | A1 |
| hasta luego | see you later | phrase | A1 |
| ¿Cómo te llamas? | What's your name? | phrase | A1 |
| Me llamo… | My name is… | phrase | A1 |
| ¿Cómo estás? | How are you? | phrase | A1 |
| Estoy bien | I'm well | phrase | A1 |
| Estoy mal | I'm not well | phrase | A1 |
| más o menos | so-so | phrase | A1 |
| ¿De dónde eres? | Where are you from? | phrase | A1 |
| Soy de… | I'm from… | phrase | A1 |
| ¿Cuántos años tienes? | How old are you? | phrase | A1 |
| Tengo… años | I am… years old | phrase | A1 |
| ¿Qué haces? | What do you do? | phrase | A1 |
| estudiante | student | noun | A1 |
| profesor/profesora | teacher | noun | A1 |
| médico/médica | doctor | noun | A1 |
| mucho gusto | nice to meet you | phrase | A1 |
| encantado/encantada | pleased to meet you | phrase | A1 |
| gracias | thank you | interjection | A1 |
| de nada | you're welcome | phrase | A1 |
| por favor | please | interjection | A1 |
| sí | yes | interjection | A1 |
| no | no | interjection | A1 |
| perdón | excuse me / sorry | interjection | A1 |
| ¿Hablas inglés? | Do you speak English? | phrase | A1 |
| un poco | a little | adverb | A1 |

**Grammar Points (5)**:

1. **Ser — Personal Pronouns** (A1): yo soy, tú eres, él/ella es, nosotros somos, ellos son
2. **Question Formation** (A1): ¿Cómo…?, ¿Cuántos…?, ¿De dónde…? — intonation + inversion
3. **Tener for Age** (A1): tengo X años (NOT soy X años)
4. **Gender Agreement** (A1): estudiante, estudiante (same) vs. profesor/profesora, médico/médica
5. **Polite Address (Tú vs. Usted)** (A1): informal contexts use tú, formal use usted

---

**Lesson 1: Meet and Greet** (vocabulary_introduction, ≈ 15 min)

Step 1 — Warm-up reading passage:
```
Dialogue:
Ana: ¡Hola! Me llamo Ana. ¿Cómo te llamas?
Pedro: Hola, Ana. Me llamo Pedro. Mucho gusto.
Ana: ¡Encantada! ¿De dónde eres?
Pedro: Soy de Madrid. ¿Y tú?
Ana: Soy de Buenos Aires.
```
Task: 2 MCQ comprehension questions.

Steps 2–11: 10 vocabulary MCQ cards (items from vocabulary set above, stage 1):
- "What does 'buenos días' mean?" → Good morning / Good evening / Goodbye / Good night
- "What does 'encantado' mean?" → Pleased to meet you / I'm well / Thank you / Yes
- (etc. for remaining 8)

Step 12 — Mining panel: "3 words you can add to your deck from this lesson."

---

**Lesson 2: Grammar — Ser and Pronouns** (grammar_focus, ≈ 12 min)

Step 1 — Grammar card (explanation, not quiz):
```
Rule: "SER" — to be (permanent characteristics)
yo soy | tú eres | él/ella es | nosotros somos | vosotros sois | ellos/ellas son

Examples:
  • Yo soy estudiante. (I am a student.)
  • Ella es médica. (She is a doctor.)
  • Nosotros somos de España. (We are from Spain.)

Common trap: Don't say "Yo soy 25 años." Use TENER for age: "Yo tengo 25 años."
```

Steps 2–9 — 8 cloze items:
- "Pedro _____ de Madrid." → es
- "Nosotros _____ estudiantes." → somos
- "Yo _____ profesora." → soy
- "¿Tú _____ de aquí?" → eres
- "Ella _____ médica." → es
- "Ellos _____ de México." → son
- "¿Vosotros _____ hermanos?" → sois
- "Yo _____ 22 años." → tengo (trap — tests ser vs. tener)

Steps 10–11 — 2 sentence reconstructions:
- Scramble: [años / treinta / tengo / Yo] → Yo tengo treinta años.
- Scramble: [profesora / es / Ciudad / de / Ella / México] → Ella es profesora de Ciudad de México.

---

**Lesson 3: Reading — Who Am I?** (reading_practice, ≈ 15 min)

Step 1 — Reading passage (A1, 50 words):
```
Me llamo Carlos. Tengo veintiocho años. Soy de Barcelona, España.
Soy médico y trabajo en un hospital grande. Hablo español e inglés.
Me gusta mucho el fútbol y la música. Estoy muy bien, gracias.
¿Y tú? ¿Cómo te llamas?
```
Tappable words: all content words show gloss on tap.
Questions:
- "Where is Carlos from?" → Barcelona / Madrid / Buenos Aires / Sevilla
- "What does Carlos do?" → He's a doctor / He's a student / He's a teacher / He's a chef

Step 2 — Reading passage 2 (A1, 60 words, different person):
```
Hola, soy María. Tengo treinta y cinco años y soy de México.
Trabajo como profesora en una universidad. Tengo dos hijos: Ana y Luis.
En mi tiempo libre, me gusta leer libros y cocinar.
Hablo español y un poco de inglés. ¡Mucho gusto!
```
Questions:
- "What is María's job?" → Teacher / Doctor / Engineer / Chef
- "How many children does María have?" → Two / Three / One / None

Steps 3–7 — 5 fill-in-blank vocabulary items from the reading:
- "María _____ profesora." (soy / es / son / eres) → es
- "Tengo dos _____." (hijo / hijos / hija / hijas) → hijos
- "Hablo español y un poco de _____." (inglés / francés / alemán / chino) → inglés

---

**Lesson 4: Production Practice** (production, ≈ 15 min)

Steps 1–5 — 5 free-recall items:
- "How do you say 'good morning' in Spanish?" → buenos días
- "How do you say 'nice to meet you'?" → mucho gusto
- "How do you say 'I am from…'?" → Soy de…
- "How do you say 'how old are you?'?" → ¿Cuántos años tienes?
- "How do you say 'I have 30 years' (I'm 30)?" → Tengo treinta años

Steps 6–7 — 2 short writing tasks:
- Task 1: "Write 3 sentences introducing yourself in Spanish. Include your name, where you're from, and what you do."
  - AI grades for: used ser correctly, used tener for age, vocabulary appropriate to task.
- Task 2: "Write a short reply to Carlos's question: '¿De dónde eres y qué haces?' Answer in 2–3 sentences."
  - AI grades for: task completion, no English words, grammar accuracy.

---

**Lesson 5: Scenario** (scenario, ≈ 8 min)

Links to `introduce-yourself` scenario.
Pre-lesson prep shown:
- "In this roleplay, you'll meet a new acquaintance at a language exchange event. Introduce yourself and find out about them."
- Chunk bank preview from scenario phases.

---

**Lesson 6: Review & Checkpoint** (review, ≈ 10 min)

15-item mixed session:
- 8 vocabulary items (due SRS + new from unit)
- 4 grammar cloze
- 2 reading MCQ (short passage)
- 1 writing task (3 sentences)

Passing score: 70% → unlocks A1.2.

---

**Unit: A2.2 — Travel and Transport**

Can-Do: "I can buy tickets, ask for directions, and describe travel plans."

*(abbreviated — same template structure)*

Grammar points:
1. Preterite of ir and ser (A2): fui, fuiste, fue, fuimos, fueron
2. Direct object pronouns (A2): me, te, lo/la, nos, los/las
3. A + infinitive for plans (A2): voy a comprar, vas a viajar
4. Ordinal numbers (A2): primero, segundo, tercero…
5. Prepositions of location (A2): en, a, de, desde, hasta, por

Vocabulary: transport vocabulary (tren, avión, autobús, billete, andén, retraso, maleta, pasaporte, reserva, tarifa, etc.)

Scenario: `buy-ticket`

---

**Unit: B1.1 — Work and Career**

Can-Do: "I can talk about my job, describe my work experience, and discuss workplace situations."

Grammar points:
1. Present Subjunctive — want/prefer/recommend (B1): quiero que vayas, recomiendo que uses
2. Simple Conditional (B1): hablaría, vendría, haría
3. Relative Clauses with que/quien/donde (B1)
4. Reflexive Verbs for Daily Routines (B1): me levanto, me ducho
5. Estar + gerund for ongoing actions (B1): estoy trabajando

Scenario: `job-interview`

---

## 6. Teacher Integration

### 6.1 What Teachers Can Do Now vs. What They Should Be Able to Do

| Capability | Current State | Required State |
|-----------|--------------|----------------|
| Push SRS vocabulary cards to students | ✅ Implemented | Keep, extend |
| Create lesson materials | ❌ Not possible | Build |
| Assign lessons to students | ❌ Not possible | Build |
| Assign reading passages | ❌ Not possible | Build |
| Assign writing tasks | ❌ Not possible | Build |
| Assign a scenario for practice | ❌ Not possible | Build |
| View student's vocabulary deck | ❌ Not possible | Build |
| View student's session history | ❌ Not possible | Build |
| View student's grammar weak points | ❌ Not possible | Build |
| Grade student writing tasks | ❌ Not possible | Build |
| Leave feedback on writing tasks | ❌ Not possible | Build |
| Set a homework deadline | ❌ Not possible | Build |

### 6.2 Teacher Material Authoring

#### Teacher Assignment Object

```json
{
  "id": "uuid",
  "teacher_id": "uuid",
  "student_id": "uuid",
  "booking_id": "uuid (optional)",
  "type": "vocabulary_push | lesson | reading | writing | scenario | mixed",
  "title": "Homework: Vocabulary from today's class",
  "due_date": "2026-09-21T23:59:00Z",
  "instructions": "Practice the restaurant vocabulary we covered. Focus on ordering and paying.",
  "content": { /* type-specific payload, see below */ },
  "status": "pending | in_progress | submitted | reviewed",
  "created_at": "...",
  "student_submission": null,
  "teacher_feedback": null
}
```

**Vocabulary push** (already exists, extend with due_date + instructions):
```json
{
  "type": "vocabulary_push",
  "content": {
    "cards": [/* existing TeacherSrsCard array */],
    "language": "es",
    "note": "These are all words from today's lesson about the restaurant."
  }
}
```

**Reading assignment** (new):
```json
{
  "type": "reading",
  "content": {
    "passage": "El Restaurante\n\nAyer fui a un restaurante con mi familia...",
    "cefr_target": "A2",
    "questions": [
      {
        "q": "¿Adónde fue el autor ayer?",
        "type": "short_answer",
        "model_answer": "Fue a un restaurante"
      }
    ]
  }
}
```

**Writing assignment** (new):
```json
{
  "type": "writing",
  "content": {
    "prompt": "Escribe un email a un amigo describiendo tus planes para el próximo fin de semana. (50–80 palabras)",
    "cefr_target": "A2",
    "word_count_min": 40,
    "word_count_max": 100,
    "focus_grammar": ["ir a + infinitive", "present tense"],
    "focus_vocabulary": ["fin de semana", "planes", "ir a"]
  }
}
```

**Scenario assignment** (new):
```json
{
  "type": "scenario",
  "content": {
    "scenario_id": "restaurant-order",
    "instructions": "Complete the full restaurant roleplay. Try to get through all 5 phases.",
    "min_phases_required": 4
  }
}
```

### 6.3 Student Progress View for Teachers

New endpoint: `GET /teacher/students/:studentId/progress`

Returns:
```json
{
  "student": { "id": "...", "name": "..." },
  "profile": {
    "target_language": "es",
    "current_cefr_level": "A2",
    "readiness_score": 340,
    "placement_status": "completed",
    "daily_goal_items": 15
  },
  "vocabulary": {
    "total_cards": 87,
    "mastered": 23,
    "due_today": 12,
    "by_stage": { "1": 30, "2": 20, "3": 14, "4": 15, "5": 8 },
    "recent_additions": [/* last 5 cards */],
    "weak_cards": [/* 5 most lapsed */]
  },
  "grammar": {
    "weakest_points": [
      {"id": "...", "title": "Preterite vs. Imperfect", "confidence_pct": 32},
      {"id": "...", "title": "Object Pronouns", "confidence_pct": 41}
    ]
  },
  "activity": {
    "streak_days": 4,
    "sessions_this_week": 3,
    "xp_this_week": 450,
    "scenarios_completed": 2,
    "weekly_chart": [/* 7-day XP bars */]
  },
  "assignments": [
    {
      "id": "...",
      "title": "Restaurant vocabulary",
      "type": "vocabulary_push",
      "due_date": "...",
      "status": "submitted",
      "score": 8
    }
  ]
}
```

### 6.4 Teacher Feedback on Writing

Teachers can view submitted writing tasks and leave:
- **Inline corrections**: select a span of the student's text, add a correction + comment
- **Overall grade** (0–10 or rubric-based)
- **Encouraging note**

The writing submission model:
```json
{
  "assignment_id": "...",
  "student_text": "Hola Maria! Este fin de semana voy a ir al cine con mi amigos...",
  "word_count": 52,
  "submitted_at": "...",
  "ai_pre_feedback": {
    "score": 6,
    "corrections": [
      {"span": "con mi amigos", "correction": "con mis amigos", "rule": "Gender/number agreement"}
    ]
  },
  "teacher_feedback": {
    "grade": 7,
    "inline_corrections": [
      {"span": "Este fin de semana", "note": "Good use of this phrase!"},
      {"span": "mi amigos", "correction": "mis amigos", "note": "Remember: mis + plural noun"}
    ],
    "overall_comment": "Great effort! Your plans are clear. Watch out for possessive adjective agreement.",
    "reviewed_at": "..."
  }
}
```

### 6.5 Teacher-Created Lessons (Advanced)

For teachers who want to build a full custom lesson (not just push cards or assign scenarios):

A **lesson builder** UI that lets teachers:
1. Add a title and learning objective
2. Add steps from a step palette:
   - Vocabulary card (term + translation + context sentence)
   - Reading passage (paste text + add questions)
   - Grammar cloze item (sentence with blank + answer)
   - Writing task (prompt + rubric)
   - Listening (paste a script, teacher records audio OR use TTS)
3. Preview the lesson as a student
4. Assign to one or multiple students with a due date

This maps to the existing `curriculum_lessons` + `lesson_steps` tables — teacher-created lessons just have a `source = 'teacher'` field and a `teacher_id` owner.

---

## 7. Technical Implementation Strategy

### 7.1 Pre-Authored Content (seeded in DB)

These are created offline (by curriculum authors, teachers, or AI batch jobs reviewed by editors) and stored as static rows:

| Content | Table | Notes |
|---------|-------|-------|
| Curriculum units (A1.1–B2.4) | `curriculum_units` | 16 units, seeded |
| Lessons per unit | `curriculum_lessons` | 5–6 per unit |
| Lesson steps (MCQ, cloze, passages) | `lesson_steps` | ≈ 10 per lesson |
| Vocabulary items per unit | `lexical_items` | 30 per unit |
| Grammar points | `grammar_points` | 60 for Spanish |
| Placement item banks | `placement_items` (new) | ≥ 20 per band |
| Scenario scripts + phases | `scenario_scripts`, `scenario_phases` | 12 initial |
| Real-talk prompt bank | `real_talk_prompts` | ≥ 50 per level |

**Generation strategy**: Use AI (offline batch) with a review queue. A CLI tool generates items from a template, an admin can approve/reject each item before it's promoted to `is_active = true`.

### 7.2 AI-Generated On Demand (realtime)

These require realtime AI because they are personalised to the learner:

| Feature | AI Task | Latency Budget |
|---------|---------|----------------|
| Scenario AI partner reply | Generate in-character reply | < 3 seconds |
| Scenario intent detection (B1+) | Classify intents from learner message | < 1 second |
| Grammar error detection in messages | Flag + explain error | < 2 seconds |
| Stage 4 writing task grading | Score + correct + feedback | < 5 seconds |
| Word mining from chat messages | Extract vocabulary candidates | < 3 seconds (async) |
| Writing assignment AI pre-grading | Score + flag errors | < 5 seconds (async) |

All realtime AI calls use the existing `LearningAIService` provider chain (openrouter → opencode → nvidia → ollama fallback).

### 7.3 Hybrid (Pre-Authored Frame + AI Fill)

These use a static structure with AI-generated content on first use, then cached:

| Feature | Pre-Authored | AI Generated | Cached |
|---------|-------------|--------------|--------|
| Scenario scripted fallback | Phases, chunk banks | Varied in-character replies per turn | Per (phase, turn index) |
| Grammar point micro-lesson cards | Rule, examples template | Additional examples, analogies | Per grammar point |
| Real-talk prompts | Topic + CEFR tier | Today's specific prompt text | Daily, per level |
| Unit reading passages | Topics list | Actual passage text | Per lesson step |
| Daily writing prompts | Context template | Specific scenario | Daily, per CEFR |

### 7.4 Activity → Approach Matrix

| Activity | Approach | Rationale |
|----------|---------|-----------|
| Placement test items | Pre-authored | Must be deterministic, auditable |
| Vocabulary MCQ (stage 1) | Pre-authored | Deterministic grading |
| Grammar cloze items | Pre-authored | Deterministic |
| Reading passages | Hybrid (pre-topics, AI text, cached) | Scalable, varied |
| Scenario phases/structure | Pre-authored | Predictable flow needed |
| Scenario AI partner replies | AI realtime | Must respond to learner's unique input |
| Intent detection (A1–A2) | Keyword matching | Fast, accurate enough at limited vocab |
| Intent detection (B1–B2) | AI realtime | Required for free-form language |
| Stage 4 production grading | AI realtime | Can't grade free sentences deterministically |
| Writing task grading | AI realtime | Same reason |
| Teacher writing feedback | Human (teacher) + AI pre-grade | Combines AI speed + teacher expertise |
| Word mining from chats | AI realtime (async) | Personalised to the actual message |
| Real-talk prompts | Hybrid (cached daily) | Personal feel without per-user cost |

---

## 8. Data Model Changes Required

### New Tables

```sql
-- Stores placement item banks per language pair (replaces hardcoded fallback)
CREATE TABLE placement_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_language TEXT NOT NULL,
    native_language TEXT NOT NULL,
    cefr_level TEXT NOT NULL CHECK (cefr_level IN ('A1','A2','B1','B2','C1','C2')),
    item_type TEXT NOT NULL, -- receptive_vocab, gap_fill_type, passage_mcq, etc.
    prompt JSONB NOT NULL,
    choices TEXT[],
    correct TEXT NOT NULL,
    accept_variants TEXT[],
    difficulty_value INT NOT NULL, -- IRT b parameter (0-1000)
    is_active BOOL NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Grammar points (one row per teachable rule, per CEFR level)
CREATE TABLE grammar_points (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID REFERENCES curriculum_courses(id),
    cefr_level TEXT NOT NULL,
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    rule_text TEXT NOT NULL,
    example_sentences JSONB NOT NULL, -- array of {sentence, translation}
    common_trap TEXT,
    items JSONB NOT NULL, -- array of cloze/MCQ items
    ordinal INT NOT NULL DEFAULT 0,
    is_active BOOL NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (course_id, slug)
);

-- Teacher assignments (homework, reading, writing, scenario tasks)
CREATE TABLE teacher_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id UUID NOT NULL REFERENCES users(id),
    student_id UUID NOT NULL REFERENCES users(id),
    booking_id UUID REFERENCES tutor_bookings(id),
    type TEXT NOT NULL CHECK (type IN ('vocabulary_push','lesson','reading','writing','scenario','mixed')),
    title TEXT NOT NULL,
    instructions TEXT,
    content JSONB NOT NULL,
    due_date TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','in_progress','submitted','reviewed')),
    target_language TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Student submissions for teacher assignments
CREATE TABLE assignment_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID NOT NULL REFERENCES teacher_assignments(id),
    student_id UUID NOT NULL REFERENCES users(id),
    content JSONB NOT NULL, -- { text, word_count, answers, scenario_run_id, etc. }
    ai_feedback JSONB, -- pre-graded by AI
    teacher_feedback JSONB, -- teacher's inline corrections + comment + grade
    score INT,
    submitted_at TIMESTAMPTZ DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ
);

-- Real-talk prompt bank
CREATE TABLE real_talk_prompts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_language TEXT NOT NULL,
    cefr_level TEXT NOT NULL,
    category TEXT NOT NULL CHECK (category IN ('conversation_starter','topic_injector','opinion_phrase')),
    prompt_for_learner TEXT NOT NULL,
    target_phrase TEXT NOT NULL,
    why_useful TEXT,
    follow_up_chunks JSONB, -- [{text, translation}]
    is_active BOOL NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Existing Table Changes

```sql
-- Add source + teacher ownership to curriculum_lessons
ALTER TABLE curriculum_lessons
    ADD COLUMN source TEXT NOT NULL DEFAULT 'curriculum' CHECK (source IN ('curriculum','teacher')),
    ADD COLUMN teacher_id UUID REFERENCES users(id),
    ADD COLUMN is_active BOOL NOT NULL DEFAULT true;

-- Add stage_initial to scenario_scripts for assigning scaffold level
ALTER TABLE scenario_scripts
    ADD COLUMN scaffold_initial TEXT NOT NULL DEFAULT 'guided' CHECK (scaffold_initial IN ('guided','supported','independent'));

-- Add assignment linkage to scenario_runs
ALTER TABLE scenario_runs
    ADD COLUMN assignment_id UUID REFERENCES teacher_assignments(id);

-- Widen grammar_points items to include the full grammar_cloze structure
-- (if table doesn't exist yet, create it as above)
```

---

## 9. AI Prompt Specifications

### 9.1 Scenario Partner System Prompt (improved)

```
You are {{ai_role_name}}, {{ai_role_description}}.

LANGUAGE RULES:
- Speak exclusively in {{target_language}}.
- Use vocabulary and grammar at CEFR level {{cefr_level}} ± 0.5 bands.
- Keep each reply to 1–2 sentences maximum.
- After each reply, add a translation in parentheses on the next line: *({{native_language}}: ...)*

ROLEPLAY RULES:
- You are playing a real scene. Do not break character.
- The current phase goal: {{phase.learner_goal}}
- Required intents to complete this phase: {{phase.required_intents}}
- Do NOT complete the learner's task for them. Your job is to prompt, not to answer for them.
- If the learner hasn't covered all required intents yet, ask a clarifying sub-question.
- If the learner makes a grammatical error, complete the exchange naturally, then add on a new line: *(Nota: "{{error_span}}" → "{{correction}}" — {{one_line_rule}})*
- If the learner writes in {{native_language}}, respond in {{target_language}} and add: *(¡Intenta responder en {{target_language}}!)*

PHASE STATE:
Phase {{current_phase_ordinal}} of {{total_phases}}: {{phase.title}}
Intents covered so far: {{covered_intents}}
Learner's last message: {{learner_message}}

Return strict JSON:
{
  "ai_message": "string (in target language)",
  "translation": "string (in native language, without the parentheses)",
  "phase_complete": boolean,
  "grammar_correction": {"error_span": "...", "correction": "...", "rule": "..."} | null,
  "nudge": {"show": boolean, "text": "...", "suggested_chunks": ["..."]}
}
```

### 9.2 Intent Detection Prompt (B1+ levels)

```
Classify which of the required intents are covered by the learner's message.

Required intents: {{required_intents_json}}
Intent definitions: {{intent_definitions_json}}
Learner's message: "{{learner_message}}"

Rules:
- Be generous: partial or implied coverage counts.
- Do not require exact keywords; semantic meaning is sufficient.
- Return only intents from the required list.

Return strict JSON array: ["intent_tag_1", "intent_tag_2"]
```

### 9.3 Stage 4 Production Grader

```
You are grading a language learner's sentence at CEFR level {{cefr_level}}.

Target language: {{target_language}}
Native language: {{native_language}}
Word the learner was asked to use: "{{term}}"
Learner's sentence: "{{learner_sentence}}"

Evaluate:
1. Did the learner use the word correctly in context? (boolean)
2. Is the sentence grammatically correct for CEFR {{cefr_level}}? (boolean)
3. Does the sentence make semantic sense? (boolean)

Return strict JSON:
{
  "correct": boolean,
  "used_correctly": boolean,
  "grammar_ok": boolean,
  "feedback": "One sentence in {{native_language}} explaining the result.",
  "corrected_sentence": "The corrected version, or null if already correct.",
  "error_spans": [{"original": "...", "correction": "...", "rule": "..."}]
}
```

### 9.4 Writing Task Grader

```
You are a {{target_language}} language teacher grading a {{cefr_level}}-level writing task.

Task: {{task_description}}
Focus grammar: {{focus_grammar}}
Focus vocabulary: {{focus_vocabulary}}
Learner's response: "{{learner_text}}"

Rubric (10 points total):
- Task completion (3 pts): Did they address the prompt fully?
- Grammar accuracy (4 pts): Are sentences grammatically correct for {{cefr_level}}?
- Vocabulary range (2 pts): Did they use varied, appropriate vocabulary?
- Coherence (1 pt): Does the text flow logically?

Return strict JSON:
{
  "score": 0-10,
  "task_completed": boolean,
  "corrected_text": "Full corrected version",
  "error_spans": [{"original": "...", "correction": "...", "rule": "..."}],
  "vocabulary_comment": "1 sentence",
  "grammar_comment": "1 sentence",
  "improvement_tip": "1 actionable tip for next time"
}
```

---

## 10. Quality & Correctness Guardrails

### 10.1 AI Output Validation

All AI responses must pass validation before being shown to learners:

- **Scenario replies**: must be in target language (language detection check). If native language detected, retry with stronger instruction.
- **Grammar corrections**: must not introduce new errors. Run the correction through a deterministic grammar check before surfacing.
- **Placement items**: AI-generated items are never shown directly to learners. They go into a review queue (admin/editor approval required).
- **Writing feedback**: score 0–10 range enforced. Error spans must reference actual spans in the student text.

### 10.2 Fallback Chain

For each AI-dependent activity, there is a defined fallback:

| Activity | AI Failure Fallback |
|----------|---------------------|
| Scenario reply | Use scripted phase reply from `ScenarioPhase.scaffold_hints` |
| Intent detection | Use keyword matching |
| Stage 4 grading | Mark as "pending review", student can re-attempt in 5 min |
| Writing grader | Return "Could not grade automatically. Your teacher will review this." |
| Word mining | Use regex tokenizer + frequency-list classifier |

### 10.3 CEFR Calibration Review

A monthly quality review should:
1. Sample 50 random scenario turns per CEFR level and check the AI partner's language was within range.
2. Sample 20 random writing grades and have a language teacher spot-check them.
3. Track % of stage 4 production answers graded "pending review" (target: < 5%).

### 10.4 Accessibility

- All audio activities must have text transcripts.
- All images used in lesson steps must have alt text.
- Tappable word glosses must be keyboard-accessible on web.
- Color-coded feedback (green = correct, red = wrong) must also use icons (✓ / ✗) for color-blind users.

---

*Document version: 1.0 — September 2026*  
*Authors: Generated from codebase analysis + SLA research synthesis*  
*Next review: after initial A1.1 unit data is seeded and tested*
