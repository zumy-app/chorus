# Chorus — MVP Release Plan & Iterative Roadmap

> **Document status:** Active. Supersedes the waterfall-style phase gating in all previous
> `IMPLEMENTATION_PLAN_V*.md` and `PHASE_*_IMPLEMENTATION_PLAN.md` files. Those documents
> are kept as historical reference only. This file is the single source of truth for
> **what ships, when, and to whom**.
>
> Last revised: 2026-09-21

---

## 0. Philosophy — Iterative, Risk-Reduced Releases

The previous plans described large sequential phases ("Phase 0 → 4") in waterfall fashion,
gating production on completion of every feature in a phase before anything ships. That model
maximises risk: months of work before real user feedback, big-bang deployments, and no ability
to discover what users actually care about.

**The new model:**

| Old | New |
|-----|-----|
| 4 large waterfall phases | Continuous thin vertical slices, each independently deployable |
| "All features done" before launch | Ship the smallest useful version, iterate weekly/bi-weekly |
| One audience sees all features | Feature flags gate features per user group (admin / beta / general) |
| Risk concentrated at launch | Risk spread across many small releases |

**Core principle:** get a real app with real users into the market fast. Every subsequent
release adds one tested increment. Untested/experimental features exist in the codebase behind
flags — admins see them, general users don't. This means we never have to choose between
"ship broken" and "delay forever."

---

## 1. Audience Tiers & Feature Flag Model

### 1.1 User Groups

| Group | Who | Access |
|-------|-----|--------|
| `admin` | Internal team, investors, design reviewers | All features, all flags ON |
| `beta` | Trusted waitlist users (invite-only cohort) | MVP features + selected experimental flags |
| `general` | Public users post-launch | Only flags marked `stable` |

Group membership is stored in `users.role` (already exists: `member`, `moderator`, `admin`)
and extended with a new `users.beta_access` boolean. Role + beta_access together determine
which flags resolve as ON.

### 1.2 Feature Flag Architecture

**Implementation approach: DB-backed flags with in-process cache (30s TTL)**

A new table `feature_flags` holds flag definitions. A sidecar service resolves flags per
user at request time. The mobile/web clients receive their resolved flag map on session
start (via `GET /users/me/flags`) and cache it for the session, refreshing on foreground resume.

```sql
-- New table: feature_flags
CREATE TABLE feature_flags (
    key            TEXT PRIMARY KEY,               -- e.g. "video_calls", "study_pods"
    description    TEXT NOT NULL,
    default_state  BOOLEAN NOT NULL DEFAULT FALSE, -- fallback if no override
    admin_only     BOOLEAN NOT NULL DEFAULT FALSE, -- true = only role='admin'
    beta_access    BOOLEAN NOT NULL DEFAULT FALSE, -- true = role='admin' OR beta_access=true
    stable         BOOLEAN NOT NULL DEFAULT FALSE, -- true = all authenticated users
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Per-user overrides (for individual testers, dogfooding etc.)
CREATE TABLE feature_flag_overrides (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    flag_key   TEXT NOT NULL REFERENCES feature_flags(key) ON DELETE CASCADE,
    enabled    BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, flag_key)
);
```

**Flag resolution logic (server-side):**
```
isAdmin   = user.role == 'admin'
isBeta    = user.beta_access == true OR isAdmin
flag.on   = override(user) ?? (
              flag.stable  ||
              (flag.beta_access && isBeta) ||
              (flag.admin_only && isAdmin)
            )
```

**Client usage (React Native / Web):**
```typescript
// useFeatureFlag hook
const canSeeVideoCalls = useFeatureFlag('video_calls')  // false for general users
const canSeeStudyPods  = useFeatureFlag('study_pods')   // false for general users
```

### 1.3 Initial Flag Definitions at MVP Launch

| Flag Key | Description | admin | beta | general | Stable when |
|----------|-------------|-------|------|---------|-------------|
| `video_calls` | WebRTC audio/video calls | ✓ | — | — | Phase 3 |
| `study_pods` | Group study pods | ✓ | — | — | Phase 3 |
| `teacher_marketplace` | Teacher booking & payouts | ✓ | — | — | Phase 4 |
| `placement_test` | Full CEFR placement quiz | ✓ | ✓ | — | Sprint 3 |
| `scenario_roleplay` | AI scenario role-play | ✓ | ✓ | — | Sprint 4 |
| `word_flashcards` | Spaced-repetition flashcard practice | ✓ | ✓ | — | Sprint 3 |
| `ai_writing_assistant` | "Help me write" composer feature | ✓ | ✓ | — | Sprint 3 |
| `grammar_deep_dive` | Full grammar breakdown drawer | ✓ | ✓ | ✓ | Sprint 2 (MVP+1) |
| `word_collector` | Tap-to-save vocabulary from messages | ✓ | ✓ | ✓ | Sprint 2 (MVP+1) |
| `voice_message` | In-chat voice recording | ✓ | — | — | Phase 2 |
| `document_sharing` | PDF/doc sharing in chat | ✓ | — | — | Phase 2 |
| `group_chat` | Multi-person chat rooms | ✓ | — | — | Phase 2 |
| `xp_leaderboard` | League/ranking tables | ✓ | ✓ | — | Sprint 4 |
| `social_feed` | Public community activity feed | ✓ | — | — | Phase 3 |
| `payout_teacher` | Payout dashboard for teachers | ✓ | — | — | Phase 4 |

### 1.4 Admin Feature Management UI

A dedicated admin panel screen (`/admin/features`) lets admins:
- Toggle any flag's `admin_only`, `beta_access`, `stable` tiers
- Add a per-user override (promote a specific user to see a flag early)
- See real-time counts: how many users have each flag enabled
- Audit log of flag changes (who changed, when, old → new value)

This panel already partially exists in `backend/internal/handlers/admin_*.go`. The feature
flag CRUD endpoints slot in alongside the existing admin routes.

---

## 2. MVP Release — "Chorus 1.0" (Target: Sprint 1)

This is the first production mobile app release. Every item listed here must ship. Nothing
else ships to general users until it does. The goal is a complete, polished, usable app for
the stated features — not a demo.

### 2.1 MVP Screens & Flows

**Screen 1: Home / Landing (unauthenticated)**
- Marketing landing page (already partially in `landing/`)
- Value prop headline, feature highlights, social proof (waitlist count)
- Waitlist join CTA with email capture
- Log In CTA → Login screen
- "WhatsApp, Apple & Facebook login coming soon" teaser chip

**Screen 2: Waitlist**
- Email capture + confirmation
- Position counter + progress bar (% to beta launch)
- Referral mechanic: copy link / share on Twitter — each referral moves user up 100 spots
- Success state with "check your inbox" message

**Screen 3: Login / Register**
- Email + password login
- Google OAuth ("Continue with Google")
- Register tab: full name + email + password
- Forgot password flow (email recovery link)
- Language switcher pill (EN default)

**Screen 4: Onboarding — Step 1: Profile Identity** *(shown once, first login only)*
- Avatar picker: initials + color palette, upload custom photo CTA (reserved, no upload infra yet)
- First name (required), Last name (optional)
- Display name / handle (required, @username, availability check)
- Mobile phone number (required, with country code picker) — for OTP/recovery
- Bio (required, min 20 chars, 160 char max) — AI topic suggestion chip
- City / location (optional)
- "Continue to Languages" CTA
- Allow Partner Discovery (required default is yes. Means that others can discover you as learning partners)

**Screen 5: Onboarding — Step 2: Language Setup** *(Step 2 of 3)*
- Native language selector (flag + language name, dropdown)
- Target language selector (flag + language name, quick-switch chips for top languages)
- CEFR level self-selection: Beginner (A1-A2), Intermediate (B1), Advanced (B2)
  - Note: "Don't worry if you're unsure — Chorus adapts as you chat"
- "Continue to Goals" CTA

**Screen 6: Onboarding — Step 3: Goals & Interests** *(Step 3 of 3)*
- Daily learning commitment: Casual (5 min/+10 XP), Regular (10 min/+25 XP), Intensive (20 min/+50 XP)
- Interest tag cloud (multi-select, pick ≥ 2): Food & Tapas, Travel, Music & Arts, Work & Careers,
  Sports, Movies & Shows, Daily Life & Culture, Books & Tech
- Daily practice notification toggle (on by default)
- "+50 XP Profile Bonus unlocked!" reward pill
- "Complete Profile & Start Chatting" CTA → Dashboard

**Screen 7: Dashboard (Home tab)** *(post-login, recurring)*
- Top bar: Chorus logo, streak badge (🔥 Xd), XP badge (⚡ XXX XP), user avatar + language badge
- Welcome banner (first visit: referral attribution e.g. "Invited by Sofia Martínez")
- Active Conversations section with conversation preview cards:
  - Partner avatar, name, online status, language pair indicator
  - Last message preview with auto-translation
  - "Reply" CTA
- AI Tutor Chorus card (always present, bot conversation)
  - Quick prompt chips: "Explain por vs para", "Practice natural phrasing"
- Word Collector & Library teaser card (3 words from today's chats, "Collect & Save" CTA)
  - *gated behind `word_collector` flag — shows stub "Start chatting to build your deck" for general users until Sprint 2*
- Sneak Peek Showcase section (upcoming features: Video Calls, Study Pods, Teacher Marketplace)
  - "Notify Me (X interested)" buttons — drives interest list, no navigation behind them
- Bottom nav: Chats | Hub | Learn

**Screen 8: Chats / Partner Discovery**
- Search bar + filter pills (All, Unread, Native Speakers, Tutors)
- Active Conversations list
  - Per-card: avatar + online dot, name, CEFR level badge, language pair, last message + translation, timestamp, unread count badge
- Suggested Learning Partners section (scrollable horizontal cards)
  - Match % score, native/learning language, "Why we matched" AI insight
  - "Start Chat" CTA
  - Filter: Native Speakers | Online Now | Near Me | Filters

**Screen 9: Chat Room**
- Chat header: back arrow, partner avatar + online dot, name, CEFR badge, language pair pill, "Ask AI" button
- Daily streak + today's focus bar (grammar topic)
- Message stream:
  - Incoming message bubble: original text with vocabulary highlights (tappable words), auto-translated substrip (ES→EN), word highlight cards (level badge, tap to save)
  - Outgoing message bubble: sent text + translation toggle
  - Date dividers
- AI Grammar Drawer (accessible via "Ask AI" button or tapping grammar-highlighted word):
  - AI insight card with grammar explanation + Sparky tip bubble
  - Quick prompt chips: "Explain X vs Y", "How do I reply naturally?", "Practice 3 phrases"
- Input composer:
  - Text field ("Type in English or Spanish...")
  - Real-time translation preview indicator
  - Add attachment button, voice mic button (non-functional in MVP, shows "coming soon"), send button
  - AI hint spark button inside input field

**Screen 10: Hub Tab** *(minimal in MVP)*
- Suggested Learning Partners list (same as Screen 8 suggested section, full page view)
- "Invite a Friend" card with referral link
- Upcoming features teaser: Study Pods "Coming Soon", Teacher Marketplace "Coming Soon"
- Safety pledge badge explanation

**Screen 11: Learn Tab** *(minimal in MVP)*
- CEFR progress bar (A1 → C2) with current level indicator
- Daily streak card
- "Today's Focus" topic card (driven by onboarding language level)
- Word Collector summary (X words saved, X due for review)
  - *word flashcard practice gated behind `word_flashcards` flag*
- "Placement Test" card — gated behind `placement_test` flag (shows "Coming Soon" for general users). users can take this anytime to update their CEFR level
- Upcoming: Scenario Role-play "Coming Soon" card — gated behind `scenario_roleplay` flag

**Screen 12: User Profile (own + others)**
- Profile hero: cover photo area, avatar, name, CEFR badge, location, timezone/active hours
- XP Balance + Level card (e.g. "1,420 XP — Level 4")
- Active Streak card
- CEFR League ranking card (gated behind `xp_leaderboard` flag for display)
- Language pair(s) section
- Interests tag cloud
- Bio
- Mutual connections / common contacts
- For other users: "Send Message" CTA, "Invite to Study Pod" CTA (gated)
- Verified Can-Do Milestones section (gated behind placement completion)

### 2.2 MVP Feature Completeness Checklist

Every item below must be end-to-end functional before MVP goes to general release:

- [ ] Home marketing page + waitlist form (email capture, position counter, referral link)
- [ ] Login (email/password + Google OAuth)
- [ ] Registration (name, email, password)
- [ ] Forgot password (email recovery)
- [ ] Onboarding 3-step flow: profile identity → language setup → goals & interests
- [ ] Dashboard with active conversations + AI tutor + word collector teaser
- [ ] Chats screen: list of conversations + suggested partners
- [ ] Chat room: send/receive messages, WebSocket delivery, read receipts
- [ ] Automatic chat translation (source → target language, <500ms p95 cache)
- [ ] Translation toggle per message (show/hide original)
- [ ] Grammar analysis: AI-powered explanation drawer (inline, tap to open)
- [ ] Word highlights in messages (tappable, level badge)
- [ ] Word Collector: tap-to-save words to vocabulary deck (ships at Sprint 2, see §3)
- [ ] AI Tutor bot conversation (Sparky — 24/7, always available chat)
- [ ] Hub tab: partner suggestions + invite friends
- [ ] Learn tab: CEFR progress + daily streak (minimal)
- [ ] User profiles (own + others)
- [ ] XP accumulation (see §4)
- [ ] Feature flags system + admin panel
- [ ] Push notifications stub (FCM token registration; actual delivery at Sprint 2)

---

## 3. Post-MVP Iterative Sprints

Each sprint is 2 weeks. Features ship as soon as their sprint's flag is promoted to `stable`.
No waiting for other sprints to complete.

### Sprint 2 — Chat Depth & Word Collecting (2 weeks after MVP)
- Promote `grammar_deep_dive` → stable (all users)
- Promote `word_collector` → stable (all users)
- Word flashcard preview in Learn tab (mark words: known / still learning)
- Push notification delivery (daily streak reminders, message notifications)
- Read receipts visual (sent → delivered → read ticks)
- Message reply + delete
- Real phone OTP verification (Twilio/AWS SNS)

### Sprint 3 — Learning Engine Core (2 weeks)
- Promote `word_flashcards` → stable
- Promote `ai_writing_assistant` → stable
- Promote `placement_test` → beta
- Spaced-repetition (SM-2) flashcard practice sessions
- AI writing assistant "Help me write" in composer
- Your Learning Path: real metrics (words/month, sentences understood, XP/streak chart)
- CEFR level recalibration based on chat history

### Sprint 4 — Engagement & Social (2 weeks)
- Promote `scenario_roleplay` → stable
- Promote `xp_leaderboard` → stable (opt-in)
- AI scenario role-play (restaurant, travel, daily life scenarios)
- XP leaderboard / Bronze-Silver-Gold league (weekly reset)
- Weekly quest system (e.g. "Send 20 messages", "Learn 5 new words")
- Partner matching refinement (interest + level + timezone weighting)
- Referral milestone rewards (XP bonuses for successful referrals)

### Sprint 5 — Communication Parity (2 weeks)
- Message forward + pin
- Media sharing (photos)
- Message + contact search
- Archive & mute conversations
- Promote `voice_message` → beta

### Sprint 6 — Audio Calls (4 weeks — longer sprint)
- Promote `voice_message` → stable
- WebRTC audio call (real signaling, not stub)
- Live transcription + translated captions
- Promote `document_sharing` → beta

### Sprint 7+ — Video Calls, Study Pods, Teacher Marketplace
- See original PHASE_2_3_IMPLEMENTATION.md for detailed specs
- These are now explicit sprints, not phases — each ships independently when ready

---

## 4. XP System — Design & Implementation

The wireframes prominently show XP at multiple touchpoints (dashboard header, profile, onboarding
commitment selector, goals completion bonus). This is **not yet implemented** in the backend.
XP is a first-class MVP feature — it drives retention and the referral loop.

### 4.1 XP Events & Values

| Event | XP | Notes |
|-------|----|-------|
| Complete onboarding (profile + language + goals) | +50 | One-time bonus shown in wireframe |
| Send a message | +2 | Capped at 40/day (20 messages) |
| Receive and read a message | +1 | Capped at 20/day |
| Tap a word to save to vocabulary | +3 | Max 10 saves/day |
| Open grammar explanation (first time per message) | +5 | Drives AI feature adoption |
| Complete a flashcard session (≥5 cards) | +15 | Unlocked Sprint 2 |
| Complete AI scenario | +25 | Unlocked Sprint 4 |
| Daily streak maintained | +10 | Per day, awarded at midnight UTC |
| Casual daily goal achieved | +10 | |
| Regular daily goal achieved | +25 | |
| Intensive daily goal achieved | +50 | |
| Invite a friend who registers | +100 | Referral reward |
| Invite a friend who completes onboarding | +150 | |
| Weekly quest completion | +50–200 | Varies by quest |
| Profile completion (all required fields) | +50 | One-time |

### 4.2 XP Levels

| Level | XP Required | Badge | |
|-------|------------|-------|--|
| 1 | 0 | Newcomer | |
| 2 | 200 | Explorer | |
| 3 | 500 | Conversationalist | |
| 4 | 1,000 | Communicator | |
| 5 | 2,000 | Linguist | |
| 6 | 4,000 | Polyglot | |
| 7 | 8,000 | Fluent | |
| 8 | 15,000 | Master | |

Level badge shown on profile (e.g. "A2 Explorer") as seen in the user profile wireframe.

### 4.3 Data Model — New Tables Required

```sql
-- User XP balance (updated via events, never decremented)
ALTER TABLE users ADD COLUMN xp_total INT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN xp_level  INT NOT NULL DEFAULT 1;
ALTER TABLE users ADD COLUMN streak_days INT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN streak_last_activity_date DATE;

-- XP event ledger (audit trail, analytics)
CREATE TABLE xp_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type  TEXT NOT NULL,   -- e.g. "message_sent", "word_saved", "daily_goal"
    xp_awarded  INT  NOT NULL,
    reference_id TEXT,           -- optional: message_id, word_id, etc.
    awarded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_xp_events_user_date ON xp_events(user_id, awarded_at DESC);

-- Daily cap tracker (prevent gaming)
CREATE TABLE xp_daily_caps (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date       DATE NOT NULL,
    event_type TEXT NOT NULL,
    count      INT  NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, date, event_type)
);
```

### 4.4 XP API

```
POST /xp/award          -- internal only, called by services
GET  /users/me/xp       -- returns { total, level, levelLabel, streakDays, todayXP, weeklyXP }
GET  /users/:id/xp      -- public view (total + level only)
GET  /xp/leaderboard    -- weekly leaderboard (gated: xp_leaderboard flag)
```

### 4.5 Streak Logic

- A streak increments when the user has ≥1 XP event on consecutive calendar days (UTC)
- `streak_last_activity_date` is updated on any XP event
- On login, if `streak_last_activity_date` < yesterday → streak resets to 0 (unless "streak
  recovery" mechanic exists, which is a Sprint 4 premium feature)
- The dashboard header flame badge (🔥 Xd) reads from `users.streak_days`

---

## 5. Data Model Gaps — What the Wireframes Require vs What Exists

After a thorough analysis of all 14 wireframe screens against the current data model, the
following gaps were identified:

### 5.1 User Profile — Missing Fields

The `users` table already has: `first_name`, `last_name`, `display_name`, `username`,
`avatar_url`, `phone`, `native_language`, `target_languages`, `plan`, `role`.

**Missing in `users` table:**
```sql
ALTER TABLE users ADD COLUMN bio             TEXT;           -- onboarding bio
ALTER TABLE users ADD COLUMN city            TEXT;           -- city/location (optional)
ALTER TABLE users ADD COLUMN cover_photo_url TEXT;           -- profile cover photo
ALTER TABLE users ADD COLUMN beta_access     BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN xp_total        INT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN xp_level        INT NOT NULL DEFAULT 1;
ALTER TABLE users ADD COLUMN streak_days     INT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN streak_last_activity_date DATE;
ALTER TABLE users ADD COLUMN onboarding_completed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN onboarding_step INT NOT NULL DEFAULT 0; -- 0=not started, 1,2,3
```

**Missing in `user_language_profiles` (extend `UserLanguageProfile`):**
- `daily_commitment` TEXT — 'casual'|'regular'|'intensive' (onboarding step 3)
- `interests` TEXT[] — array of selected interest tags (onboarding step 3)
- `notifications_enabled` BOOL — daily practice nudge toggle

```sql
ALTER TABLE user_language_profiles 
    ADD COLUMN daily_commitment TEXT NOT NULL DEFAULT 'regular',
    ADD COLUMN interests        TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN notifications_enabled BOOLEAN NOT NULL DEFAULT TRUE;
```

### 5.2 Onboarding State Tracking

No table currently tracks multi-step onboarding completion. Add to `users` table (above) and
create a lightweight checkpoint table:

```sql
CREATE TABLE onboarding_checkpoints (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    step       INT  NOT NULL,   -- 1, 2, 3
    completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    data       JSONB,           -- snapshot of data captured at that step
    PRIMARY KEY (user_id, step)
);
```

### 5.3 Partner Matching / Suggestions

The wireframes show "Suggested for You" with a **match % score** and a "Why we matched" AI
insight. No matching logic exists currently. Contacts exist but not algorithmic suggestion.

**New table:**
```sql
CREATE TABLE partner_match_scores (
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    partner_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    match_score    SMALLINT NOT NULL,         -- 0-100
    match_reasons  TEXT[] NOT NULL DEFAULT '{}', -- e.g. {"language_pair", "interests_3"}
    computed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, partner_id)
);
CREATE INDEX idx_partner_match_score ON partner_match_scores(user_id, match_score DESC);
```

**Match scoring algorithm (MVP, simple heuristic):**
- Language pair complement (they speak what you're learning): +50 points
- Shared interests (per overlap): +10 points each, max +30
- Similar CEFR level (±1 level): +10 points
- Same timezone region: +5 points
- Online now: +5 points
- Computed nightly or on profile update; refreshed on first login of day

### 5.4 Waitlist Referral Mechanic

The waitlist wireframe shows "1 invite = 100 spots" position movement. The existing
`waitlist` table needs a referral column:

```sql
ALTER TABLE waitlist 
    ADD COLUMN referral_code    TEXT UNIQUE DEFAULT encode(gen_random_bytes(6), 'hex'),
    ADD COLUMN referred_by_code TEXT REFERENCES waitlist(referral_code),
    ADD COLUMN referral_count   INT NOT NULL DEFAULT 0,
    ADD COLUMN position         INT;  -- recalculated: base_position - (referral_count * 100)
```

### 5.5 Feature Flags (new — see §1.2 above)

Tables `feature_flags` and `feature_flag_overrides` are net-new.

### 5.6 XP & Streaks (new — see §4.3 above)

Tables `xp_events` and `xp_daily_caps` are net-new; columns on `users` table are net-new.

### 5.7 Notification Preferences

The onboarding toggle "Daily Practice Nudge" has no persistence target. Add to the existing
`user_settings` table:

```sql
ALTER TABLE user_settings
    ADD COLUMN push_notifications_enabled  BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN daily_reminder_time         TIME,       -- e.g. '19:00'
    ADD COLUMN marketing_emails_enabled    BOOLEAN NOT NULL DEFAULT TRUE;
```

### 5.8 "Safe Learning Pledge" (user_profile wireframe)

The Mateo profile wireframe shows a "Safe Learning Pledge" verified badge. This is a voluntary
commitment checkbox during onboarding or profile setup. Track it:

```sql
ALTER TABLE users 
    ADD COLUMN safe_learning_pledge      BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN safe_learning_pledged_at  TIMESTAMPTZ;
```

---

## 6. Wireframe → Implementation Gap Summary

| Wireframe Screen | Status | Gaps / Work Required |
|-----------------|--------|----------------------|
| Home landing page | Exists (partial in `landing/`) | Waitlist count counter (live from DB), referral mechanic, mobile-responsive polish |
| Join Waitlist | Exists | Add referral code, position counter, share buttons |
| Login / Register | Exists | Google OAuth social login not wired; language switcher pill |
| Onboarding Step 1 — Profile Identity | Partial (name+avatar exist) | Bio, city, phone number collection, interests, `onboarding_checkpoints` table |
| Onboarding Step 2 — Language Setup | Exists (level selection exists) | Language pair UI (native + target selectors), CEFR UI polish, quick-switch chips |
| Onboarding Step 3 — Goals & Interests | **Missing entirely** | New screen: daily commitment selector, interest tag cloud, notification toggle, XP bonus reveal |
| Dashboard | Exists (partial) | XP badge wired to real data, streak badge wired, word collector teaser card, sneak peek section |
| Chats / Partner Discovery | Exists (chat list exists) | Suggested partners section, match %, "Why we matched" insight, filter pills |
| Chat Room | Exists (core messaging works) | Word highlight chips (tappable), auto-translation substrip, grammar drawer, AI tutor quick prompts |
| Hub Tab | **Missing entirely** | Partner suggestions full-page, invite friends card, coming-soon teasers |
| Learn Tab | Exists (partial, mock data) | Real CEFR progress bar, streak card, word collector summary, placeholders for gated features |
| User Profile (own) | Partial | XP/level/streak cards, interests display, bio, can-do milestones (gated) |
| User Profile (others) | Partial | Match score display, "Why we matched", mutual connections |
| AI Grammar Drawer | Exists (partial) | Word-level tap interaction, Sparky tip bubble, quick prompt chips for contextual suggestions |

---

## 7. Questions Answered: Should We Build XP Now?

**Yes.** Here is the reasoning:

1. **XP appears in 7 of 14 wireframes** — header badge, profile cards, onboarding reward, goals
   selector, dashboard. It is not an optional decorative feature; it is load-bearing UI.
2. **Retention mechanism from day one.** Without XP+streaks, daily active user numbers will be
   weak. The streak badge in the header is the single most effective daily re-engagement trigger
   in apps like Duolingo and BeReal.
3. **Low implementation cost.** The core XP system is 2 DB tables + a simple service + a badge
   component. The leaderboard/league is the complex part — that is gated behind `xp_leaderboard`
   flag for Sprint 4.
4. **Referral loop needs XP rewards.** The waitlist mechanic already shows XP incentives for
   referrals. If we ship referrals without XP, the reward loop is broken.

**Conclusion:** Build the XP ledger (events + daily caps + user columns) in MVP Sprint 1.
Build the league/leaderboard in Sprint 4 behind a flag.

---

## 8. Questions Answered: What User Profile Data Should We Collect?

Based on wireframe analysis and matching/personalization needs:

### Mandatory at Onboarding
| Field | Why |
|-------|-----|
| First name | Personalisation ("Welcome, Alex!") |
| Display name / @handle | Unique identity, partner discovery |
| Email | Auth, recovery |
| Password | Auth |
| Mobile phone | OTP recovery, 2FA (Sprint 2) |
| Native language | Translation direction, matching |
| Target language | Translation direction, matching, curriculum |
| CEFR self-assessment | Translation verbosity, curriculum starting point |
| Daily commitment (casual/regular/intensive) | Push notification frequency, XP target |
| Interest tags (≥2) | Partner matching, AI conversation starter topics |

### Optional at Onboarding (can complete later)
| Field | Why |
|-------|-----|
| Last name | Display name composition |
| Bio (20-160 chars) | Profile display, conversation ice-breaker |
| City / location | Timezone inference, partner proximity matching |
| Profile photo | Humanises the profile card |
| Cover photo | Profile aesthetics |

### NOT Collected (privacy risk, not needed)
- Date of birth (only needed if age-gating is required — not planned)
- Gender (not needed for matching; avoid discrimination risk)
- Nationality (city/country captures the relevant signal)
- Occupation (potential PII, low value for matching)

---

## 9. Go-to-Market Strategy — Fastest Path to Mobile Production

### 9.1 Pre-Launch (now → MVP ready)
1. **Waitlist continues running** — keep accumulating emails. Referral mechanic goes live
   immediately to accelerate list growth before app is ready.
2. **Internal dogfooding** — team uses the app under `admin` role with all flags enabled.
   Catch real bugs before any user sees them.
3. **Android-first** — The existing APK build path (`+apk.txt`, `start-android.ps1`) is
   already proven. Ship Android to Play Store beta track first; iOS follows 2 weeks later
   (avoids App Review delays blocking both simultaneously).
4. **EAS (Expo Application Services)** — Already decided in `PHASE_1_IMPLEMENTATION_PLAN.md`.
   Set up EAS Build + EAS Update for OTA JS updates that don't require store review. This is
   the key velocity enabler: most post-MVP sprints ship as OTA updates, not store submissions.

### 9.2 Beta Launch (Sprint 1 done)
- Invite first 500 waitlist users via email with a personal invite link
- `beta_access = true` set for each invited user
- Beta users see MVP features + grammar_deep_dive + word_collector flags
- Collect structured feedback: in-app rating prompt after first 5 conversations
- Weekly beta cohort expansion: +500 users/week based on capacity

### 9.3 General Availability (MVP features validated by beta)
- Promote all MVP flags to `stable`
- Remove waitlist gate — anyone can register
- Play Store & App Store public listing
- Landing page prominently linked from social channels
- XP referral mechanic: existing users invited to share ("Invite friends, earn XP")

### 9.4 Viral Loop Design
```
User signs up → completes onboarding (+50 XP) → receives chat from partner → 
  shares a grammar insight or learns a word → feels progress → 
    invites friend to join (+150 XP) → friend completes onboarding → 
      both earn XP → feedback loop repeats
```

The referral XP reward (150 XP for a friend completing onboarding) is deliberately higher
than most single-day activity — it creates a meaningful incentive to invite while not
inflating the XP economy (a user can only bring in a finite number of friends).

### 9.5 Web vs Mobile Priority

Mobile-first means the React Native Expo app is the **primary client**. The web frontend is
maintained for:
- Landing page (marketing) 
- Admin panel (`/admin/*` routes)
- Web app as backup/secondary surface

Web gets the same features as mobile but mobile bugs and UX polish are fixed first. If a
feature requires mobile-specific implementation (push notifications, voice recording), ship
mobile first; web follows or uses a degraded experience.

---

## 10. Untested Features — Current Codebase Status & Flag Assignment

The following features exist in the codebase but are untested or partially implemented.
They are assigned flags and hidden from general users until validated:

| Feature | Code Location | Flag | Risk | Path to Stable |
|---------|--------------|------|------|----------------|
| Video/audio calls | `handlers/call.go`, `handlers/video_qa_test.go` | `video_calls` | High — WebRTC needs real signaling | Sprint 6 |
| Teacher marketplace | `handlers/teacher.go`, `models/learning_teacher.go` | `teacher_marketplace` | High — payouts, vetting | Sprint 7 |
| Payout system | `handlers/payout.go`, `models/payout.go` | `payout_teacher` | High — real money | Sprint 7 |
| Placement test | `handlers/learning.go`, `models/learning_placement.go` | `placement_test` | Medium — content quality | Sprint 3 |
| Scenario role-play | `handlers/learning.go`, `models/learning_scenario.go` | `scenario_roleplay` | Medium — AI prompt quality | Sprint 4 |
| AI writing assistant | `handlers/grammar.go` | `ai_writing_assistant` | Low — isolated composer feature | Sprint 3 |
| Full SRS (flashcards) | `models/vocabulary.go`, `handlers/vocabulary.go` | `word_flashcards` | Low — SM-2 logic exists | Sprint 2 |
| Group chat | `handlers/chat.go` (partial) | `group_chat` | Medium — websocket complexity | Sprint 5+ |
| Gallery / media | `handlers/gallery.go` | `media_sharing` | Low — upload infra needed | Sprint 5 |
| Location sharing | `handlers/location.go` | `location_sharing` | Low — sensitive data | Sprint 5 |
| GDPR export | `handlers/gdpr.go` | `gdpr_export` | Low — compliance feature | Sprint 3 |
| Moderation tools | `handlers/moderation.go` | internal admin only | Low — already admin-only | Stable now |

---

## 11. Admin Panel Requirements

The admin panel (`/admin/*` in the web frontend) must expose:

### Existing (keep, verify working)
- User list + search + suspend/unsuspend
- Waitlist management
- Translation quality review (`admin_quality.go`)

### New — Feature Flag Management
- Flag list: all flags, their tier state (admin_only/beta/stable), and per-user override count
- Toggle tier per flag (single click, with confirmation dialog + audit log entry)
- Add/remove per-user override
- "Preview as user" — lets admin simulate the app experience with a given user's flag set

### New — Beta Group Management
- List of users with `beta_access = true`
- Bulk-add from waitlist position range (e.g. "promote next 500 waitlist users to beta")
- Remove beta access
- Export beta user emails for comms

### New — XP & Engagement Dashboard
- Daily active users
- Average XP per user per day
- Streak distribution histogram
- Top 20 users by XP (for sanity-checking the economy)
- XP event breakdown by type (messages vs invites vs goals)

---

## 12. Success Metrics — MVP Exit Criteria

Before declaring MVP "done" and promoting to general availability:

| Metric | Target |
|--------|--------|
| Beta users with ≥1 conversation in first 24h | > 60% |
| D7 retention (users active on day 7) | > 30% |
| Onboarding completion rate | > 70% |
| Translation p95 latency | < 500ms |
| Grammar drawer opens per conversation | > 1.5 |
| Crash-free sessions | > 99% |
| App Store rating (beta feedback) | > 4.2 |
| Referrals sent per active user | > 0.3 |

These are the instrumentation signals that tell us the core loop works. Anything below target
triggers immediate investigation before broader rollout.

---

## Appendix A — Migration Checklist

Before deploying MVP to production, run these migrations in order:

```sql
-- 1. Feature flags
CREATE TABLE feature_flags (...);
CREATE TABLE feature_flag_overrides (...);

-- 2. User profile extensions
ALTER TABLE users ADD COLUMN bio TEXT;
ALTER TABLE users ADD COLUMN city TEXT;
ALTER TABLE users ADD COLUMN cover_photo_url TEXT;
ALTER TABLE users ADD COLUMN beta_access BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN xp_total INT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN xp_level INT NOT NULL DEFAULT 1;
ALTER TABLE users ADD COLUMN streak_days INT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN streak_last_activity_date DATE;
ALTER TABLE users ADD COLUMN onboarding_completed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN onboarding_step INT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN safe_learning_pledge BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN safe_learning_pledged_at TIMESTAMPTZ;

-- 3. Language profile extensions
ALTER TABLE user_language_profiles 
    ADD COLUMN daily_commitment TEXT NOT NULL DEFAULT 'regular',
    ADD COLUMN interests TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN notifications_enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- 4. Onboarding checkpoints
CREATE TABLE onboarding_checkpoints (...);

-- 5. XP system
CREATE TABLE xp_events (...);
CREATE TABLE xp_daily_caps (...);

-- 6. Partner matching
CREATE TABLE partner_match_scores (...);

-- 7. Waitlist referral
ALTER TABLE waitlist
    ADD COLUMN referral_code TEXT UNIQUE DEFAULT encode(gen_random_bytes(6), 'hex'),
    ADD COLUMN referred_by_code TEXT,
    ADD COLUMN referral_count INT NOT NULL DEFAULT 0,
    ADD COLUMN position INT;

-- 8. Settings extensions
ALTER TABLE user_settings
    ADD COLUMN push_notifications_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN daily_reminder_time TIME,
    ADD COLUMN marketing_emails_enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- 9. Seed initial feature flags
INSERT INTO feature_flags (key, description, admin_only, beta_access, stable) VALUES
    ('video_calls',          'WebRTC audio/video calls',              TRUE,  FALSE, FALSE),
    ('study_pods',           'Group study pods',                       TRUE,  FALSE, FALSE),
    ('teacher_marketplace',  'Teacher booking & payouts',              TRUE,  FALSE, FALSE),
    ('placement_test',       'Full CEFR placement quiz',               FALSE, TRUE,  FALSE),
    ('scenario_roleplay',    'AI scenario role-play',                  FALSE, TRUE,  FALSE),
    ('word_flashcards',      'Spaced-repetition flashcard practice',   FALSE, TRUE,  FALSE),
    ('ai_writing_assistant', '"Help me write" composer feature',       FALSE, TRUE,  FALSE),
    ('grammar_deep_dive',    'Full grammar breakdown drawer',          FALSE, FALSE, TRUE),
    ('word_collector',       'Tap-to-save vocabulary from messages',   FALSE, FALSE, TRUE),
    ('voice_message',        'In-chat voice recording',                TRUE,  FALSE, FALSE),
    ('document_sharing',     'PDF/doc sharing in chat',                TRUE,  FALSE, FALSE),
    ('group_chat',           'Multi-person chat rooms',                TRUE,  FALSE, FALSE),
    ('xp_leaderboard',       'League/ranking tables',                  FALSE, TRUE,  FALSE),
    ('social_feed',          'Public community activity feed',         TRUE,  FALSE, FALSE),
    ('payout_teacher',       'Payout dashboard for teachers',          TRUE,  FALSE, FALSE),
    ('media_sharing',        'Photo/video sharing in chat',            TRUE,  FALSE, FALSE),
    ('location_sharing',     'In-chat location sharing',               TRUE,  FALSE, FALSE),
    ('gdpr_export',          'GDPR data export flow',                  FALSE, FALSE, FALSE);
```

---

*Revision history: Created 2026-09-21. Authored as the single iterative release plan,
replacing waterfall phase documents.*
