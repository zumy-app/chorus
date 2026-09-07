# Chorus: Technical Requirements & Functional Specifications

## 1. Project Overview
Chorus is an AI-native language learning messenger designed to break language barriers by integrating structured pedagogy (CEFR-aligned) directly into real human-to-human and human-to-AI communication. Unlike traditional apps that use manufactured content, Chorus treats the user's personal conversation history as the primary learning corpus.

## 2. Core Functional Requirements

### A. AI-Powered Messaging & Translation
- **Real-time Translation:** Bi-directional translation within chat bubbles. Users can toggle visibility of original vs. translated text.
- **Grammar Analysis (The "Deep Dive"):** Automatic identification of grammar constructions within messages. 
- **Learning Guard:** An AI filter that can hide "off-topic" (e.g., dating/harassment) messages based on user-defined strictness levels.
- **Smart Captions:** Real-time transcription and translation for audio and video calls.

### B. Structured Learning Engine
- **CEFR Curriculum:** Support for levels A1 through B2. Each level is divided into Units based on "Can-Do" statements (e.g., "I can order coffee").
- **Spaced Repetition System (SRS):** A vocabulary management system using a modern algorithm (FSRS recommended) to schedule reviews of "mined" words.
- **Daily Practice Loop:** A singular entry point for daily study that interleaves SRS reviews, new micro-lessons, and "mined-from-chat" exercises.
- **Placement Testing:** An adaptive entry test to determine a user's starting CEFR level and known vocabulary.

### C. The Marketplace (Teachers & Tutors)
- **Tutor Profiles:** Detailed profiles featuring credentials, student reviews, and reputation scores.
- **Booking & Payments:** Support for private trial sessions and monthly subscription tiers.
- **Teacher Dashboard:** Tools for managing student progress, reviewing pronunciation recordings, and pushing custom activities to students' chat sidebars.

### D. Social & Community
- **Group Study Rooms:** Live collaborative spaces with a shared "Activity Sandbox" for group drills.
- **Trending Hub:** A community feed for sharing linguistic insights, cultural questions, and "Real Talk" prompts.
- **Matching Engine:** Suggests learning partners based on complementary native/target languages and shared interests.

---

## 3. Detailed Feature Specifications

### F01: The Unified "AI Deep Dive" Panel
- **Trigger:** Tapping a message or a dedicated "AI" icon.
- **Features:** 
    - Word-by-word breakdown (Morphology).
    - Grammar point explanation with "Base Language" contrast (e.g., explaining Spanish Subjunctive to an English speaker).
    - Contextual Drills: One-tap access to drills (cloze, reordering) using the specific grammar point from that message.

### F02: Chat-to-SRS Mining Pipeline
- **Mechanism:** Background process scans messages for high-frequency/leveled lemmas the user hasn't mastered.
- **Interaction:** "Tap-to-save" on any word in a chat creates a `VocabCard`.
- **Metadata:** Each card stores the *original source sentence* as the example, preserving the memory hook of the conversation.

### F03: Real Talk Starters
- **Mechanism:** AI suggests conversation openers based on the current Unit's theme (e.g., if studying "Travel," suggest "Where is the last place you visited?").
- **Goal:** Drive output production (Swain's Output Hypothesis).

### F04: Trust & Safety "Mutual Consent Mode"
- **Mechanism:** Hides non-educational/personal messages by default.
- **Unmasking:** Requires both parties to explicitly opt-in to "Social Mode" to reveal personal content, preventing unwanted solicitation while allowing genuine connections.

---

## 4. Technical Architecture Recommendations (for Implementation)

### Data Model Entities
- `User`: Profile, CEFR level, target/base language, subscription status.
- `Message`: Content, sender, timestamp, translation, identified grammar tags.
- `VocabCard`: Lemma, definition, example sentence, SRS metadata (due date, difficulty, logs).
- `GrammarPoint`: CEFR tag, language, explanation template.
- `StudySession`: Participants, activity_log, shared_sandbox_state.

### Key API Services
- **Translation Service:** LLM-based translation with context awareness.
- **Pedagogy Service:** Matches text to CEFR inventories and manages SRS scheduling.
- **Moderation Service:** Real-time L7 filtering of chat content against "Learning Guard" profiles.
- **RTC Service:** Handles signaling and media for Voice/Video with transcription hooks.

---

## 5. Visual Baseline (Reference Wireframes)
- **Onboarding:** {{DATA:SCREEN:SCREEN_23}}, {{DATA:SCREEN:SCREEN_90}}
- **Messaging:** {{DATA:SCREEN:SCREEN_4}}, {{DATA:SCREEN:SCREEN_104}}, {{DATA:SCREEN:SCREEN_18}}
- **Learning Hub:** {{DATA:SCREEN:SCREEN_91}}, {{DATA:SCREEN:SCREEN_64}}, {{DATA:SCREEN:SCREEN_92}}
- **Teacher Tools:** {{DATA:SCREEN:SCREEN_79}}, {{DATA:SCREEN:SCREEN_72}}, {{DATA:SCREEN:SCREEN_56}}
- **Marketplace:** {{DATA:SCREEN:SCREEN_83}}, {{DATA:SCREEN:SCREEN_82}}
- **Safety:** {{DATA:SCREEN:SCREEN_19}}, {{DATA:SCREEN:SCREEN_7}}
