# Chorus — Science-Based Lesson & Vocabulary Engine
**Companion to the original implementation plan.** Focus: how captured words become structured practice, how CEFR curriculum and real conversation merge, and how to design the real-world scenario lessons (ordering coffee, grocery checkout, etc.) so they actually accelerate acquisition rather than just feeling like content.

---

## 1. The core design principle: personal corpus > generic curriculum

The single biggest lever Chorus has that Duolingo/Babbel don't is that your vocabulary isn't manufactured — it comes from the user's own conversations. This isn't just a nice feature, it's a real retention advantage, for three converging reasons from memory research:

- **Contextual/episodic encoding** (Craik & Lockhart's levels-of-processing): information encoded with rich context and personal meaning is retained far better than information encoded as an isolated fact. A word tied to "what Sofia said to me yesterday" has a memory hook a generic flashcard never gets.
- **The self-reference effect**: material connected to the self (one's own life, relationships, choices) is recalled better than equivalent material with no personal connection.
- **The generation effect**: information you produce or actively retrieve is retained better than information you passively read — which is why tap-to-save should always trigger at least one active retrieval step before the word is filed away, not just a definition popup.

**Implication for architecture:** the CEFR curriculum should function as a *scaffold*, not the primary content source. Its job is to guarantee coverage (grammar/vocabulary the user hasn't encountered naturally yet) and sequencing (nothing taught before its prerequisites). The user's own chat-mined vocabulary should be the *preferred* material whenever it overlaps with what the curriculum would teach anyway — always prefer "teach the word from their real conversation" over "teach the word from a stock example sentence."

---

## 2. The word-capture → library → structured-practice pipeline

Extending the MinedItem/VocabCard model from the original plan, here's the specific lifecycle a word goes through:

```
1. CAPTURE
   Tap-to-save (manual) or auto-mining (background, from Section 5 of the original plan)
   → stores lemma + exact source sentence + speaker + timestamp + CEFR tag

2. CLASSIFY
   → match lemma/construction against curriculum's lexical/grammar inventory
   → does this word/point belong to a specific Unit the user hasn't reached yet,
     a Unit already completed, or is it outside the core inventory entirely
     (proper noun, slang, low-frequency)?

3. ROUTE  (this is the key design decision)
   Case A — matches an upcoming Unit's target vocab:
     → credit partial progress toward that unit; when the user reaches it,
       that item is already "known" and gets a lighter first exposure
       (skip pure presentation, go straight to practice) — avoids boring
       repetition of something they already half-know.
   Case B — matches an already-completed Unit:
     → reinforcement card; enters SRS queue with a small priority boost
       (recency + personal relevance = good candidate for review)
   Case C — outside the core inventory (idiom, slang, proper-noun-adjacent):
     → still enters SRS as a "bonus" card, but doesn't block curriculum
       progression or checkpoint requirements

4. PRACTICE (depth-of-processing ladder — see Section 3)
   → item cycles through recognition → cued recall → free recall → production
     across multiple spaced sessions, not within one session

5. RE-ENCOUNTER
   → the word is deliberately reinjected into later content: a future
     scenario roleplay, a future Real Talk suggestion, a future SRS review —
     varied contexts of re-exposure measurably improve retention flexibility
     versus reviewing the identical sentence every time (this is "contextual
     variability," a well-established spacing-effect refinement)
```

This is what makes "structured training activities around those words" (your phrase) actually work: the word isn't just added to a list, it's routed into the exact place in the curriculum where it's useful, and it keeps resurfacing in new contexts rather than being drilled to death in one form.

---

## 3. Depth-of-processing ladder (what "structured training activities" should actually contain)

Don't treat all review items as interchangeable. Sequence item types by depth of processing, and require a learner to pass a shallower stage before a deeper one is introduced for that specific item:

| Stage | Item type | What it tests | Research basis |
|---|---|---|---|
| 1. Recognition | Multiple choice: see word, pick meaning | Passive recall | Baseline encoding |
| 2. Cued recall | Cloze-in-original-sentence (fill the blank in the real message it came from) | Contextual retrieval | Encoding specificity (Tulving) — retrieval works best when cue matches encoding context |
| 3. Free recall | See meaning/context, type the target-language word unaided | Active retrieval, no cue | Testing effect (retrieval practice > re-reading) |
| 4. Production | Use the word correctly in a *new* self-generated sentence | Generative use | Output hypothesis (Swain) — production forces grammatical processing that comprehension doesn't |
| 5. Spontaneous use | Word appears unprompted in a real chat message or scenario roleploy | Functional acquisition | The actual end goal — track this as a completion signal, not just a drill type |

A card shouldn't be marked "Mastered" (as shown in your Vocab Hub filter tabs) until it's cleared stage 4 across at least two spaced sessions — recognizing a word once is not the same as having acquired it, and over-crediting mastery is a common failure mode that makes progress stats feel good but doesn't reflect real ability.

---

## 4. Real-world scenario lessons (ordering coffee, grocery checkout, etc.)

This is a distinct content type from vocabulary drilling — it belongs to **Task-Based Language Teaching (TBLT)**, which is well-supported as one of the fastest paths to functional fluency because it organizes learning around completing real communicative goals rather than around grammar topics in isolation.

### 4.1 Structure each scenario as a "script"
Cognitive science (Schank & Abelson's script theory) shows people store real-world routines — ordering coffee, checking out at a store — as structured event sequences with predictable phases and roles. Build scenario content around this same structure instead of a flat dialogue:

```
Scenario: Ordering coffee
CEFR can-do: "I can order food and drink and ask about prices" (A1)
Phases:
  1. Greeting          — chunk bank: "Hola, buenos días" / "¿Qué te gustaría pedir?"
  2. Ordering           — chunk bank: "Quisiera un café con leche" / "Para llevar / Para tomar aquí"
  3. Customization       — chunk bank: "¿Tiene leche de avena?" / "Sin azúcar, por favor"
  4. Payment             — chunk bank: "¿Cuánto es?" / "¿Aceptan tarjeta?"
  5. Closing             — chunk bank: "Gracias, que tenga un buen día"
```

Teach each phase's **formulaic chunks as units first**, not as decomposed grammar. Research on vocabulary/chunk acquisition (Nation; the lexical approach more broadly) shows that multi-word chunks ("¿Me puede dar...?", "Para llevar") are stored and retrieved as single units by fluent speakers and are acquired faster as wholes than assembled compositionally from individual grammar rules — grammatical analysis of *why* the chunk works can come after the learner can already use it, not before (this is a deliberate inversion of the "grammar first" approach most textbooks use, and it's why phrasebook-style learning often produces usable speech faster than grammar-first courses, even though it doesn't produce deep grammatical understanding on its own — you want both, sequenced chunk-first).

### 4.2 AI roleplay design: scaffold, then remove the scaffold
Mirror what Screen 3's "Sparky's Nudge" already does, but formalize it as a two-pass structure:

- **Pass 1 (scaffolded):** the AI plays the barista/cashier; the learner is given 2-3 suggested chunk options per turn (with base-language gloss, exactly like the current nudge design) and can tap one or type freely.
- **Pass 2 (unscaffolded):** same scenario, same AI role, but suggestions are hidden by default (available on a "hint" tap, at a small XP cost — this preserves autonomy while still preventing total dead-ends). This mirrors the classic scaffolding-removal principle from Vygotsky's zone of proximal development: support should fade as competence grows, not stay constant.

Difficulty should follow **i+1** (Krashen): each scenario should contain mostly vocabulary/structures the learner already knows, plus a small number (1-3) of new chunks — never a scenario built entirely from unfamiliar language.

### 4.3 Close the loop back into the vocabulary engine
Every new chunk or word used correctly in a scenario roleplay should auto-generate a candidate VocabCard (same pipeline as chat-mining, Section 2) — the scenario becomes another *input source*, not a dead-end activity. This is what turns "10 scenario packs" into a genuinely expanding personal corpus instead of static content.

### 4.4 Scenario library structure
Organize scenarios by CEFR level and tie each one to its Unit's can-do statement, same as Screen 3 already models ("Unit 4 — Daily Routine" → Real Talk starters). Suggested A1/A2 launch set for both language directions: ordering at a café, grocery checkout, asking for directions, introducing yourself, making small talk about weather/weekend plans, booking a table at a restaurant, asking for help in a store, simple phone call (confirming an appointment). Each should take 3-5 minutes to complete — short enough to fit inside the daily session budget from the original plan.

---

## 5. Composing the daily session (the algorithm)

Extending Section 4 of the original plan with the actual composition logic:

```
Daily session (5-10 min target) =
  1. All SRS-due items (hard floor — never skipped, this is the retention engine)
  2. If a Unit is in progress: 1 micro-lesson OR 1 scenario roleplay
     (alternate between these across sessions — don't do both same day,
     keep session length bounded)
  3. If chat-mined items are pending review/confirmation: surface top 2-3
     ranked by teachability score (from the original plan's mining pipeline)
  4. Interleave item types within the session — never block all vocabulary
     review together and all grammar review together; alternate types
     turn to turn (desirable-difficulty / interleaving effect — Bjork)
```

**Personalization axis:** let users declare a primary goal at onboarding — "Conversational fluency" (weights scenarios + chat-mining higher) vs. "Structured/exam prep" (weights unit progression + grammar checkpoints higher). This is a low-cost way to respect learner autonomy (a core driver of sustained motivation per self-determination theory) without building two separate products.

---

## 6. Feedback and correction design

When a learner makes a production error (in SRS free-recall, in a scenario roleplay, or in real chat with grammar-analysis flagging something):

1. **Don't just show the correct answer.** First highlight the error span and prompt a self-correction attempt ("something's off here — want to try again?"). This triggers the noticing hypothesis (Schmidt) — learners must consciously notice a gap before input becomes intake.
2. If the self-correction fails or they skip, reveal the correct form **and** a short AI Tutor explanation anchored to the relevant GrammarPoint from the CEFR inventory — not just "wrong, here's the answer," which research on corrective feedback consistently shows is weaker than explanation-based feedback for durable learning.
3. Log every correction as a signal for `UserGrammarMastery` confidence scoring (from the original data model) — repeated errors on the same grammar point should increase that point's review priority independent of the specific vocabulary item involved.

---

## 7. Marketplace differentiation: give tutors the data

Short addendum to the original plan's Phase 4. The tutor marketplace screens you've designed are solid on booking/payment mechanics, but that's now commodity UX (Preply and iTalki both do it well already). The genuine differentiator is connecting the structured-learning data to the human tutor:

- Add a **Student Insights** panel to the Teacher Dashboard (opt-in by the student): current CEFR unit, weakest 3-5 grammar points by confidence score, recently-struggled vocabulary, and scenario-completion history.
- This lets a tutor spend session time on what spaced repetition and AI roleplay genuinely can't fix — live pronunciation correction, natural conversational rhythm, cultural nuance, and confidence-building — instead of re-diagnosing the student's level from scratch every session, which is the single biggest efficiency loss in typical marketplace tutoring today.
- This also gives you a pitch to tutors themselves (supply-side acquisition): "spend your paid hour teaching, not diagnosing."

---

## 8. Metrics that tell you the science is actually working

Beyond the retention/conversion metrics in the original plan, track these specifically to validate this system:

- **Depth-of-processing progression rate**: what fraction of "Mastered" words have actually cleared the production stage (4), not just recognition (1)? If mastery is concentrated at stage 1, the SRS is over-crediting shallow recognition.
- **Spontaneous-use rate**: how often do chat-mined-and-drilled words reappear unprompted in the user's own outgoing messages afterward? This is the closest proxy you have to real acquisition, and it's a metric only Chorus (with access to real conversation) can measure at all.
- **Scenario → vocabulary reinjection rate**: are scenario roleplays actually generating new VocabCards, or are they a content dead-end?
- **Nudge acceptance vs. dismissal rate**, split by AI-contact threads vs. real-human-contact threads — this will tell you quickly whether the "Sparky's Nudge" pattern needs the opt-in gating discussed above.
