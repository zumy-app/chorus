# Chorus — Iterative Release Plan

> **Document status:** Active. Supersedes all previous `IMPLEMENTATION_PLAN_V*.md` and
> `PHASE_*_IMPLEMENTATION_PLAN.md` files. Those documents are historical reference only.
> This file is the single source of truth for **what ships, when, and to whom**.
>
> Last revised: 2026-09-21

---

## 0. Philosophy — Ship the Smallest Useful App First

The previous plans described large sequential phases in waterfall fashion, gating production
on completion of every feature in a phase before anything ships. Even the earlier iterative
revision kept too much in the first release: full grammar drawers, word collectors, Phoenix
pipelines, iOS store submissions, and long P0 carry-overs all had to land together.

**The new model:**

| Old | New |
|-----|-----|
| 4 large waterfall phases | A tiny production release, then weekly/bi-weekly point releases |
| "All features done" before launch | Release 1 is intentionally minimal; everything else is flagged |
| One audience sees all features | Feature flags gate every non-core feature per user group |
| Risk concentrated at launch | Risk spread across many small, reversible releases |

**Core principle:** get a real app with real users into the market fast. Release 1 contains
only the features required by the phase-1 wireframes and no more. Every subsequent release
adds one tested increment. Untested/experimental features exist in the codebase behind flags —
admins see them, general users don't.

This means we never have to choose between "ship broken" and "delay forever." If a feature
isn't ready, we turn its flag off.

---

## 1. Audience Tiers & Feature Flag Model

### 1.1 User Groups

| Group | Who | Access |
|-------|-----|--------|
| `admin` | Internal team, investors, design reviewers | All features, all flags ON by default |
| `beta` | Trusted waitlist users (invite-only cohort) | Release 1 core + selected flags promoted to `beta` |
| `general` | Public users post-launch | Only flags marked `stable` |

Group membership is stored in `users.role` (`member` | `admin`) plus a new
`users.beta_access` boolean. `role='admin'` implicitly sees every flag. `beta_access=true`
sees flags whose `beta_access` tier is ON. General users see only flags whose `stable` tier
is ON.

> **Note:** the schema's existing `moderator` role (if present in any legacy data) is treated
> as `member` for flag resolution. Do not build `moderator`-specific flag logic.

### 1.2 Feature Flag Architecture

**Implementation approach: DB-backed flags with in-process cache (30s TTL) + server-side enforcement**

A new table `feature_flags` holds flag definitions. A sidecar service resolves flags per user
at request time. The mobile/web clients receive their resolved flag map on session start (via
`GET /users/me/flags`) and cache it for the session, refreshing on foreground resume.

> **Enforcement is server-side, not cosmetic.** Client-side visibility is derived from the
> same source of truth the API uses. Every gated route enforces its flag at the handler or
> middleware layer (403 when off) — clients must never be the only guard. Add a
> `RequireFlag(flagKey)` middleware and register it on every flagged endpoint listed in §9.
> DoD: *"gated endpoint rejects (403) when its flag is off for that user."*

```sql
-- New table: feature_flags
CREATE TABLE feature_flags (
    key            TEXT PRIMARY KEY,               -- e.g. "video_calls"
    description    TEXT NOT NULL,
    default_state  BOOLEAN NOT NULL DEFAULT FALSE, -- fallback if no override
    admin_only     BOOLEAN NOT NULL DEFAULT FALSE, -- true = only role='admin'
    beta_access    BOOLEAN NOT NULL DEFAULT FALSE, -- true = admin OR beta_access=true
    stable         BOOLEAN NOT NULL DEFAULT FALSE, -- true = all authenticated users
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Per-user overrides (for individual testers, dogfooding, emergency grants)
CREATE TABLE feature_flag_overrides (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    flag_key   TEXT NOT NULL REFERENCES feature_flags(key) ON DELETE CASCADE,
    enabled    BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, flag_key)
);
```

**Flag resolution logic (server-side, single source of truth):**
```
isAdmin   = user.role == 'admin'
isBeta    = user.beta_access == true OR isAdmin
flag.on   = override(user) ?? (
              flag.stable  ||
              (flag.beta_access && isBeta) ||
              (flag.admin_only && isAdmin) ||
              flag.default_state
            )
```
The result is computed once server-side, used for both `GET /users/me/flags` (client
visibility) and `RequireFlag` middleware (authorization). Never evaluate flags in the client
from raw DB rows — always consume the resolved map.

**Instant disable / kill-switch:** the 30s in-process cache bounds server propagation, but
client session caching can leave a flag "on" for a stale session. To kill a flag quickly:
- Include the resolved flag map in the **authenticated user payload** of every authenticated
  API response (or a lightweight `flags` object in the WS `hello` handshake), so clients
  converge within one request/round-trip, not one session.
- Do not rely on "wait for foreground resume" for security/abuse-sensitive flags
  (payouts, moderation). For those, also gate server-side (which this plan mandates) so the
  client cache is only a UX concern, never a control.

**Reconciliation with existing toggle systems:**
1. **`backend/internal/services/feature_flags.go`** — the current env-var/canary-percentage
   service is a *different* mechanism and is **superseded** by this DB model. Migrate its two
   flags (`learning_v3_engine`, `redis_session_cache`) to DB flag rows with matching
   `default_state` values, then delete the env service. The codebase must not carry two flag
   systems.
2. **FR-25 per-account toggles** (`user_settings.translation_enabled`, `grammar_auto`,
   `highlights_enabled`) are **user preferences, not rollout flags**. They stay in
   `user_settings` and are enforced in `handlers/message.go` and `services/grammar.go`.
   `grammar_auto` and `highlights_enabled` are currently stored but not enforced — R1.2 wires
   them into `GrammarService`.

**Client usage (React Native / Web):**
```typescript
// useFeatureFlag hook
const canSeeVideoCalls = useFeatureFlag('video_calls')  // false for general users
const canSeeFlashcards = useFeatureFlag('word_flashcards') // false until promoted
```

### 1.3 Initial Flag Definitions at Release 1

| Flag Key | Description | admin | beta | general | Stable when |
|----------|-------------|-------|------|---------|-------------|
| `grammar_insights` | Basic grammar analysis in chat | ✓ | ✓ | ✓ | **Release 1** |
| `word_collector` | Tap-to-save vocabulary from messages | ✓ | ✓ | ✓ | **Release 1** |
| `feature_voting` | Community feature voting in Hub | ✓ | ✓ | ✓ | **Release 1** |
| `referral_bump` | Waitlist referral position bump | ✓ | ✓ | ✓ | **Release 1** |
| `google_oauth` | "Continue with Google" login/register | ✓ | — | — | R1.1 |
| `push_notifications` | FCM token registration + delivery | ✓ | — | — | R1.1 |
| `word_flashcards` | Spaced-repetition flashcard practice | ✓ | — | — | R1.2 |
| `ai_writing_assistant` | "Help me write" composer feature | ✓ | — | — | R2 |
| `placement_test` | Full CEFR placement quiz | ✓ | — | — | R3 |
| `scenario_roleplay` | AI scenario role-play | ✓ | — | — | R4 |
| `voice_message` | In-chat voice recording | ✓ | — | — | R4 |
| `media_sharing` | Photo/video sharing in chat | ✓ | — | — | R4 |
| `document_sharing` | PDF/doc sharing in chat | ✓ | — | — | R4 |
| `real_talk` | RealTalk practice prompts (nudge + hub) | ✓ | — | — | R4 |
| `video_calls` | WebRTC audio/video calls | ✓ | — | — | R6 |
| `study_pods` | Group study pods | ✓ | — | — | R7 |
| `teacher_marketplace` | Teacher booking & payouts | ✓ | — | — | R8 |
| `payout_teacher` | Payout dashboard for teachers | ✓ | — | — | R8 |
| `xp_leaderboard` | League/ranking tables | ✓ | — | — | R4 |
| `social_feed` | Public community activity feed | ✓ | — | — | deferred |
| `group_chat` | Multi-person chat rooms | ✓ | — | — | R7 |
| `gdpr_export` | GDPR data export flow | ✓ | — | — | R3 |

### 1.4 Admin Feature Management UI

A dedicated admin panel screen (`/admin/features`) lets admins:
- Toggle any flag's `admin_only`, `beta_access`, `stable` tiers
- Add/remove a per-user override (promote a specific user to see a flag early)
- See real-time counts: how many users have each flag enabled
- Audit log of flag changes (who changed, when, old → new value)
- "Preview as user" — simulate the app experience with a given user's flag set

A second screen (`/admin/beta-users`) lets admins:
- List users with `beta_access = true`
- Bulk-promote from waitlist position range (e.g. "promote next 500")
- Revoke beta access
- Export beta user emails for comms

These screens are net-new. Existing admin handlers (`admin_waitlist.go`, `admin_users.go`,
`admin_quality.go`, `admin_translations.go`) remain but do not cover flags.

---

## 2. Release 1 — Minimal Production Mobile App

Release 1 is the first production release. It is intentionally small: home, auth, waitlist,
onboarding, dashboard, chats, chat room, minimal Hub/Learn, XP/streaks, and feature flags.
Everything else lives behind flags.

> **DoD carve-out for inert affordances.** The global DoD ("no stubs") conflicts with the
> wireframes showing a non-functional attachment/mic button (chat room). Resolution: an
> affordance may be inert **only when** (a) it is gated behind a flag that is OFF for that
> user tier, or (b) it shows a labeled "coming soon" toast and has no dead end (no silent
> no-op). Attachment button = "coming soon" toast (acceptable). Mic = `voice_message` flag
> OFF (acceptable). Anything else must be functional.

### 2.1 Release 1 Screens & Flows

**Screen 1: Home / Landing (unauthenticated)**
- Marketing landing page (`landing/` exists; polish to match `chorus_home_complete_showcase`)
- Value prop headline, feature highlights, social proof with live waitlist count
- Waitlist join CTA with email capture
- Log In CTA → login screen
- "WhatsApp, Apple & Facebook login coming soon" teaser chip

**Screen 2: Waitlist**
- Email capture + confirmation
- Position counter + progress bar (% to beta launch)
- Referral mechanic: unique referral code, copy link / share on Twitter — each referral moves
  user up 100 spots (`referral_bump` flag stable; no bump if OFF)
- Success state with "check your inbox" message

**Screen 3: Login / Register**
- Email + password login (already exists; verify end-to-end on mobile)
- Registration: full name + email + password
- Forgot password flow (email recovery link)
- **Google OAuth button hidden until `google_oauth` flag is ON** (ships R1.1)
- Language switcher pill (EN default)
- "WhatsApp, Apple & Facebook login coming soon" teaser chip

**Screen 4: Onboarding — Step 1: Profile Identity** *(shown once, first login only)*
- Avatar picker: initials + deterministic color palette, upload custom photo CTA (reserved,
  no upload infra yet)
- First name (required), Last name (optional)
- Display name / handle (required, @username, availability check)
- Mobile phone number (required, with country code picker) — for recovery; **not verified in R1**
- Bio (required, min 20 chars, 160 char max) — AI topic suggestion chip
- City / location (optional)
- "Continue to Languages" CTA

**Screen 5: Onboarding — Step 2: Language Setup** *(Step 2 of 3)*
- Native language selector (flag + language name, dropdown)
- Target language selector (flag + language name, quick-switch chips for top languages)
- CEFR level self-selection: Beginner (A1-A2), Intermediate (B1), Advanced (B2)
  - Note: "Don't worry if you're unsure — Chorus adapts as you chat"
- "Continue to Goals" CTA

**Screen 6: Onboarding — Step 3: Goals & Interests** *(Step 3 of 3)*
- Daily learning commitment: Casual (5 min/+10 XP), Regular (10 min/+25 XP),
  Intensive (20 min/+50 XP)
- Interest tag cloud (multi-select, pick ≥ 2): Food & Tapas, Travel, Music & Arts,
  Work & Careers, Sports, Movies & Shows, Daily Life & Culture, Books & Tech
- Daily practice notification toggle (on by default) — stored but delivery is R1.1
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
  - Tapping a word saves it (`word_collector` stable in R1)
  - Full flashcard practice gated behind `word_flashcards` (R1.2)
- Sneak Peek Showcase section (Video Calls, Study Pods, Teacher Marketplace)
  - "Notify Me (X interested)" buttons — drives `feature_interest` table, no navigation
- Bottom nav: Chats | Hub | Learn

**Screen 8: Chats / Partner Discovery**
- Search bar + filter pills (All, Unread, Native Speakers, Tutors)
- Active Conversations list
  - Per-card: avatar + online dot, name, CEFR level badge, language pair, last message +
    translation, timestamp, unread count badge
- Suggested Learning Partners section (scrollable horizontal cards)
  - Match % score, native/learning language, "Why we matched" AI insight
  - "Start Chat" CTA
  - Filter: Native Speakers | Online Now | Near Me | Filters

**Screen 9: Chat Room**
- Chat header: back arrow, partner avatar + online dot, name, CEFR badge, language pair pill,
  "Ask AI" button
- Daily streak + today's focus bar (grammar topic)
- Message stream:
  - Incoming message bubble: original text with vocabulary highlights (tappable words),
    auto-translated substrip (ES→EN), word highlight cards (level badge, tap to save)
  - Outgoing message bubble: sent text + translation toggle
  - Date dividers
- AI Grammar Drawer (accessible via "Ask AI" button or tapping grammar-highlighted word):
  - AI insight card with grammar explanation + Sparky tip bubble
  - Quick prompt chips: "Explain X vs Y", "How do I reply naturally?", "Practice 3 phrases"
- Input composer:
  - Text field ("Type in English or Spanish...")
  - Real-time translation preview indicator
  - Add attachment button (non-functional in R1, shows "coming soon" toast)
  - Voice mic button (non-functional, `voice_message` flag OFF)
  - AI hint spark button inside input field
  - Send button

**Screen 10: Hub Tab** *(minimal in R1)*
- Today's Conversation Starters: daily phrases with "Use in Chat" CTA
- Community Feature Voting section (`feature_voting` flag stable)
  - Upcoming features with status + "I'm Interested (X votes)" buttons
- "Invite a Friend" card with referral link
- Safe Learning Pledge badge explanation

**Screen 11: Learn Tab** *(minimal in R1)*
- CEFR progress bar (A1 → C2) with current level indicator
- Daily streak card
- "Today's Focus" topic card (driven by onboarding language level)
- Word Collector summary (X words saved, X due for review)
  - "Review Now" CTA gated behind `word_flashcards` (shows "Coming Soon" chip)
- Placement Test card — gated behind `placement_test` (shows "Coming Soon")
- Scenario Role-play card — gated behind `scenario_roleplay` (shows "Coming Soon")

**Screen 12: User Profile (own + others)**
- Profile hero: cover photo area, avatar, name, CEFR badge, location
- XP Balance + Level card (e.g. "1,420 XP — Level 4")
- Active Streak card
- CEFR League ranking card (gated behind `xp_leaderboard`)
- Language pair(s) section
- Interests tag cloud
- Bio
- For other users: "Send Message" CTA, "Invite to Study Pod" CTA (gated)
- Safe Learning Pledge badge toggle

### 2.2 Release 1 Feature Completeness Checklist

Every item below must be end-to-end functional before Release 1 goes to general users:

- [ ] Home marketing page + waitlist form (email capture, position counter, referral link)
- [ ] Login (email/password)
- [ ] Registration (name, email, password)
- [ ] Forgot password (email recovery)
- [ ] Onboarding 3-step flow: profile identity → language setup → goals & interests
- [ ] Dashboard with active conversations + AI tutor + word collector teaser
- [ ] Chats screen: list of conversations + suggested partners
- [ ] Chat room: send/receive messages, WebSocket delivery
- [ ] Automatic chat translation (source → target language, <500ms p95 cache)
- [ ] Translation toggle per message (show/hide)
- [ ] Grammar insights: AI-powered explanation drawer (inline, tap to open)
- [ ] Word highlights in messages (tappable, level badge)
- [ ] Word Collector: tap-to-save words to vocabulary deck
- [ ] AI Tutor bot conversation (Sparky — always available chat)
- [ ] Hub tab: conversation starters + feature voting + invite friends
- [ ] Learn tab: CEFR progress + daily streak + word collector summary
- [ ] User profiles (own + others)
- [ ] XP accumulation (minimal: onboarding + messages + daily goal)
- [ ] Streaks (daily activity)
- [ ] Feature flags system + admin panel
- [ ] Feature flag server-side enforcement (`RequireFlag` middleware; 403 when off)

### 2.3 What's Flagged in Release 1

These features exist in the codebase or are partially built but are **not** exposed to general
users in Release 1. Admins see them all. Beta users see only flags explicitly promoted to
`beta`.

| Feature | Flag | Why flagged |
|---------|------|-------------|
| Google OAuth | `google_oauth` | Net-new backend work; fast-follow in R1.1 |
| Push notifications | `push_notifications` | FCM delivery infra; R1.1 |
| Flashcard practice | `word_flashcards` | Full SRS practice; validate in R1.2 |
| AI writing assistant | `ai_writing_assistant` | Composer AI; R2 |
| Placement test | `placement_test` | Content quality; R3 |
| Scenario role-play | `scenario_roleplay` | Prompt quality; R4 |
| Voice messages | `voice_message` | Recording/upload; R4 |
| Media sharing | `media_sharing` | Upload infra; R4 |
| Document sharing | `document_sharing` | Upload infra; R4 |
| Video/audio calls | `video_calls` | WebRTC signaling; R6 |
| Study pods | `study_pods` | Group rooms; R7 |
| Teacher marketplace | `teacher_marketplace` | Payouts/vetting; R8 |
| XP leaderboard | `xp_leaderboard` | Economy balance; R4 |
| Group chat | `group_chat` | WebSocket complexity; R7 |
| GDPR export | `gdpr_export` | Compliance; R3 |

---

## 3. Fast-Follow Releases

After Release 1, each release promotes at most 1–2 flags and ships only what has been
dogfooded by admins and a small beta cohort. Releases are not blocked by unrelated work.

### R1.1 — Auth Polish & Push Tokens (1 week after R1)
- Promote `google_oauth` → stable
- Backend `GET /auth/google` + `GET /auth/google/callback`
- Mobile `expo-auth-session`; web redirect flow
- FCM token registration (`push_notifications` → beta)
- Store token in `device_tokens` table

### R1.2 — Word Collector Hardening & Flashcards (1 week after R1.1)
- Promote `word_flashcards` → beta
- Spaced-repetition (SM-2) practice sessions from saved vocabulary
- "Review Now" CTA enabled for beta users
- Read receipts visual (sent → delivered → read ticks)
- Phone OTP verification (Twilio/AWS SNS)

### R2 — AI Writing Assistant (1 week after R1.2)
- Promote `ai_writing_assistant` → stable
- "Help me write" composer feature
- Settings screen: translation on/off, grammar auto-analysis, highlights
- Wire server-side enforcement for `grammar_auto` + `highlights_enabled`

### R3 — Placement Test & GDPR Export (1–2 weeks)
- Promote `placement_test` → beta
- Promote `gdpr_export` → stable
- Placement test UI + scoring
- CEFR recalibration suggestion after 20+ messages

### R4 — Engagement & Social (2 weeks)
- Promote `scenario_roleplay` → stable
- Promote `voice_message` → beta
- Promote `media_sharing` → beta
- Promote `xp_leaderboard` → stable (opt-in)
- Weekly quest system

### R5 — Communication Parity (2 weeks)
- Promote `voice_message` → stable
- Promote `media_sharing` → stable
- Promote `document_sharing` → beta
- Message forward + pin, archive/mute UI, block/report UX

### R6 — Audio Calls (2–4 weeks)
- Promote `document_sharing` → stable
- Promote `video_calls` → beta (audio-first)
- WebRTC audio call with live transcription

### R7+ — Video Calls, Study Pods, Teacher Marketplace
- Promote `video_calls` → stable
- Promote `study_pods` → beta
- Promote `group_chat` → beta
- Teacher marketplace beta + payouts beta

---

## 4. XP System — Minimal Release 1 Version

The wireframes show XP at multiple touchpoints (dashboard header, profile, onboarding reward,
goals selector). This is load-bearing UI and must be truthful in Release 1. The full economy
(leagues, quests, referral milestones) comes later.

### 4.1 Release 1 XP Events

| Event | XP | Notes |
|-------|----|-------|
| Complete onboarding | +50 | One-time bonus shown in wireframe |
| Send a message | +2 | Capped at 40/day (20 messages) |
| Receive and read a message | +1 | Capped at 20/day |
| Tap a word to save to vocabulary | +3 | Max 10 saves/day |
| Open grammar explanation (first time per message) | +5 | Drives AI feature adoption |
| Daily goal achieved (casual/regular/intensive) | +10 / +25 / +50 | Based on `daily_commitment` |
| Streak maintained | +10 | Per day, awarded at first XP event of the day |
| Invite a friend who registers | +100 | Referral reward |
| Invite a friend who completes onboarding | +150 | Referral reward |

**Deferred to R4+:** flashcard sessions (+15), scenario completion (+25), weekly quests
(+50–200), league ranking, milestone badges.

### 4.2 XP Levels

| Level | XP Required | Badge |
|-------|------------|-------|
| 1 | 0 | Newcomer |
| 2 | 200 | Explorer |
| 3 | 500 | Conversationalist |
| 4 | 1,000 | Communicator |
| 5 | 2,000 | Linguist |
| 6 | 4,000 | Polyglot |
| 7 | 8,000 | Fluent |
| 8 | 15,000 | Master |

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
    event_type  TEXT NOT NULL,
    xp_awarded  INT  NOT NULL,
    reference_id TEXT,
    awarded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, event_type, reference_id)
);
CREATE INDEX idx_xp_events_user_date ON xp_events(user_id, awarded_at DESC);

-- Daily cap tracker
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
GET  /xp/leaderboard    -- gated: xp_leaderboard flag
```

### 4.5 Streak Logic

- A streak increments when the user has ≥1 XP event on consecutive calendar days (UTC)
- `streak_last_activity_date` is updated on any XP event
- On login, if `streak_last_activity_date` < yesterday → streak resets to 0
- The dashboard header flame badge (🔥 Xd) reads from `users.streak_days`

---

## 5. Data Model Gaps — What the Wireframes Require vs What Exists

### 5.1 User Profile — Missing Fields

The `users` table already has: `first_name`, `last_name`, `display_name`, `username`,
`avatar_url`, `phone`, `native_language`, `target_languages`, `plan`, `role`.

**Missing in `users` table:**
```sql
ALTER TABLE users ADD COLUMN bio             TEXT;
ALTER TABLE users ADD COLUMN city            TEXT;
ALTER TABLE users ADD COLUMN cover_photo_url TEXT;
ALTER TABLE users ADD COLUMN beta_access     BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN xp_total        INT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN xp_level        INT NOT NULL DEFAULT 1;
ALTER TABLE users ADD COLUMN streak_days     INT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN streak_last_activity_date DATE;
ALTER TABLE users ADD COLUMN onboarding_completed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN onboarding_step INT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN safe_learning_pledge BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN safe_learning_pledged_at TIMESTAMPTZ;
```

**Missing in `user_language_profiles`:**
```sql
ALTER TABLE user_language_profiles 
    ADD COLUMN daily_commitment TEXT NOT NULL DEFAULT 'regular',
    ADD COLUMN interests        TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN notifications_enabled BOOLEAN NOT NULL DEFAULT TRUE;
```

### 5.2 Onboarding State Tracking

```sql
CREATE TABLE onboarding_checkpoints (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    step       INT  NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    data       JSONB,
    PRIMARY KEY (user_id, step)
);
```

### 5.3 Partner Matching / Suggestions

```sql
CREATE TABLE partner_match_scores (
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    partner_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    match_score    SMALLINT NOT NULL,
    match_reasons  TEXT[] NOT NULL DEFAULT '{}',
    computed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, partner_id)
);
CREATE INDEX idx_partner_match_score ON partner_match_scores(user_id, match_score DESC);
```

**MVP heuristic:**
- Language pair complement: +50
- Shared interests: +10 each, max +30
- Similar CEFR level (±1 level): +10
- Same timezone region: +5
- Online now: +5

### 5.4 Waitlist Referral Mechanic

```sql
ALTER TABLE waitlist_entries
    ADD COLUMN referral_code    TEXT UNIQUE DEFAULT encode(gen_random_bytes(6), 'hex'),
    ADD COLUMN referred_by_code TEXT REFERENCES waitlist_entries(referral_code),
    ADD COLUMN referral_count   INT NOT NULL DEFAULT 0,
    ADD COLUMN position         INT;
```

Position = `base_queue_position - (referral_count * 100)`, floored at 1. Compute on read to
avoid race conditions, or under `SELECT ... FOR UPDATE` on the referrer row.

### 5.5 Feature Interest Counters

```sql
CREATE TABLE feature_interest (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    feature_key TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, feature_key)
);
CREATE INDEX idx_feature_interest_key ON feature_interest(feature_key);
```

### 5.6 Push Notification Device Tokens

```sql
CREATE TABLE device_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT NOT NULL,
    platform    TEXT NOT NULL CHECK (platform IN ('android', 'ios')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, token)
);
```

### 5.7 Notification Preferences

```sql
ALTER TABLE user_settings
    ADD COLUMN push_notifications_enabled  BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN daily_reminder_time         TIME,
    ADD COLUMN marketing_emails_enabled    BOOLEAN NOT NULL DEFAULT TRUE;
```

---

## 6. Wireframe → Implementation Gap Summary

| Wireframe folder (`wireframes/phase-1/`) | Plan screen | Status | Gaps / Work Required |
|---|---|---|---|
| `chorus_home_complete_showcase` | Screen 1 | PARTIAL | landing exists; wire live waitlist count, mobile polish |
| `join_waitlist` | Screen 2 | PARTIAL | add referral_code, position counter, share buttons |
| `chorus_login_sign_up_mobile` | Screen 3 | PARTIAL | email/password works; hide Google button until R1.1 |
| `onboarding_profile_identity` | Screen 4 | PARTIAL | add bio, city, phone, checkpoints, username check |
| `onboarding_language_level` | Screen 5 | PARTIAL | native+target pair UI, quick-switch chips |
| `onboarding_goals_interests` | Screen 6 | GAP | net-new screen: commitment, interests, nudge toggle, +50 XP |
| `chorus_dashboard_updated` | Screen 7 | PARTIAL | wire XP/streak badges, word-collector teaser, sneak-peek |
| `chorus_chats_partner_discovery` | Screen 8 | PARTIAL | chat list exists; add suggested-partners section, match % |
| `chorus_phase_1_chat_with_translation_ai_helper_word_collector` | Screen 9 | PARTIAL | core messaging+translation works; add highlights, grammar drawer |
| `chorus_hub_phase_1` | Screen 10 | GAP | net-new: conversation starters, feature voting, invite link |
| `chorus_learn_phase_1` | Screen 11 | PARTIAL | CEFR bar, streak card, word summary, gated cards |
| `user_profile_mateo_pods_xp_integrated` | Screen 12 | PARTIAL | XP/level/streak cards, interests, bio, pledge |
| `suggested_learning_partners` | Screen 8/10 | PARTIAL | data via `partner_match_scores`; card UI + "Why we matched" |
| `linguist_flow/DESIGN.md` | — | N/A | design tokens for all screens |

---

## 7. User Profile Data — What to Collect

### Mandatory at Onboarding
| Field | Why |
|-------|-----|
| First name | Personalisation |
| Display name / @handle | Unique identity, partner discovery |
| Email | Auth, recovery |
| Password | Auth |
| Mobile phone | Recovery (not verified until R1.2) |
| Native language | Translation direction, matching |
| Target language | Translation direction, matching |
| CEFR self-assessment | Translation verbosity, curriculum starting point |
| Daily commitment | Push frequency, XP target |
| Interest tags (≥2) | Matching, AI conversation starters |
| Bio (20–160 chars) | Required by wireframe; profile display + AI ice-breaker |

### Optional at Onboarding
| Field | Why |
|-------|-----|
| Last name | Display name composition |
| City / location | Timezone inference, proximity matching |
| Profile photo | Humanises profile |
| Cover photo | Profile aesthetics |

### NOT Collected
- Date of birth, gender, nationality, occupation (privacy risk, low value)

---

## 8. Go-to-Market Strategy — Fastest Path to Mobile Production

### 8.1 Pre-Launch (now → R1 ready)
1. Waitlist keeps running; referral mechanic ships with R1.
2. Internal dogfooding under `admin` role with all flags enabled.
3. **Android-first** — existing APK build path is proven. Ship Play Store internal track first;
   iOS TestFlight follows 2 weeks later.
4. **EAS Build + EAS Update** for OTA JS updates that bypass store review.

### 8.2 Beta Launch (R1 done)
- Invite first 500 waitlist users via email.
- `beta_access = true` for invited users.
- Beta users see R1 core only (no extra flags yet).
- Weekly cohort expansion: +500 users/week based on capacity.

### 8.3 General Availability (R1 core validated)
- Promote R1 flags to `stable`.
- Remove waitlist gate.
- Play Store & App Store public listing.
- XP referral mechanic: existing users invited to share.

### 8.4 Web vs Mobile Priority
Mobile-first; web is maintained for landing page, admin panel, and backup app surface.

---

## 9. Untested Features — Current Codebase Status & Flag Assignment

| Feature | Code Location | Flag | Risk | Path to Stable |
|---------|--------------|------|------|----------------|
| Video/audio calls | `handlers/call.go` | `video_calls` | High | R6 |
| Teacher marketplace | `handlers/teacher.go` | `teacher_marketplace` | High | R8 |
| Payout system | `handlers/payout.go` | `payout_teacher` | High | R8 |
| Placement test | `handlers/learning.go` | `placement_test` | Medium | R3 |
| Scenario role-play | `handlers/learning.go` | `scenario_roleplay` | Medium | R4 |
| AI writing assistant | `handlers/grammar.go` | `ai_writing_assistant` | Low | R2 |
| Full SRS flashcards | `handlers/vocabulary.go` | `word_flashcards` | Low | R1.2 |
| Group chat | `handlers/chat.go` (partial) | `group_chat` | Medium | R7 |
| Gallery / media | `handlers/gallery.go` | `media_sharing` | Low | R4 |
| Location sharing | `handlers/location.go` | `location_sharing` | Low | R5 |
| GDPR export | `handlers/gdpr.go` | `gdpr_export` | Low | R3 |
| Google OAuth | UI only | `google_oauth` | Medium | R1.1 |
| Push notifications | None | `push_notifications` | Medium | R1.1 |
| Moderation tools | `handlers/moderation.go` | internal admin | Low | Stable (admin-only) |

---

## 10. Admin Panel Requirements

### Existing (keep, verify working)
- User list + search + suspend/unsuspend
- Waitlist management
- Translation quality review

### New — Feature Flag Management
- Flag list, tier toggles, per-user overrides, audit log, "preview as user"

### New — Beta Group Management
- List beta users, bulk-promote from waitlist range, revoke, export emails

### New — XP & Engagement Dashboard
- DAU, average XP/user/day, streak distribution, top 20 users, XP event breakdown

---

## 11. Success Metrics

### 11.1 Pre-launch release gates (block beta launch)
| Gate | Target |
|------|--------|
| Backend green | `go build ./... && go test ./...` |
| Frontend green | `tsc && vite build && npm test` |
| Mobile green | jest + Android EAS build |
| e2e smoke | Playwright suite green |
| Translation p95 | < 500 ms cache-hit |
| Flag enforcement | every gated endpoint 403s when off |
| Dogfood crash-free | > 99% |

### 11.2 Post-launch success metrics (gate GA)
| Metric | Target |
|--------|--------|
| Beta users with ≥1 conversation in first 24h | > 60% |
| D7 retention | > 30% |
| Onboarding completion rate | > 70% |
| Grammar drawer opens per conversation | > 1.5 |
| Referrals sent per active user | > 0.3 |
| App Store rating | > 4.2 |

---

## Appendix A — Migration Checklist

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
ALTER TABLE waitlist_entries
    ADD COLUMN referral_code TEXT UNIQUE DEFAULT encode(gen_random_bytes(6), 'hex'),
    ADD COLUMN referred_by_code TEXT,
    ADD COLUMN referral_count INT NOT NULL DEFAULT 0,
    ADD COLUMN position INT;

-- 8. Feature interest + device tokens
CREATE TABLE feature_interest (...);
CREATE TABLE device_tokens (...);

-- 9. Settings extensions
ALTER TABLE user_settings
    ADD COLUMN push_notifications_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN daily_reminder_time TIME,
    ADD COLUMN marketing_emails_enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- 10. Seed feature flags
INSERT INTO feature_flags (key, description, admin_only, beta_access, stable) VALUES
    ('grammar_insights',     'Basic grammar analysis in chat',         FALSE, TRUE,  TRUE),
    ('word_collector',       'Tap-to-save vocabulary from messages',   FALSE, TRUE,  TRUE),
    ('feature_voting',       'Community feature voting in Hub',        FALSE, TRUE,  TRUE),
    ('referral_bump',        'Waitlist referral position bump',        FALSE, TRUE,  TRUE),
    ('google_oauth',         'Continue with Google login',             TRUE,  FALSE, FALSE),
    ('push_notifications',   'FCM token registration and delivery',    TRUE,  FALSE, FALSE),
    ('word_flashcards',      'Spaced-repetition flashcard practice',   TRUE,  FALSE, FALSE),
    ('ai_writing_assistant', 'Help me write composer feature',         TRUE,  FALSE, FALSE),
    ('placement_test',       'Full CEFR placement quiz',               TRUE,  FALSE, FALSE),
    ('scenario_roleplay',    'AI scenario role-play',                  TRUE,  FALSE, FALSE),
    ('voice_message',        'In-chat voice recording',                TRUE,  FALSE, FALSE),
    ('media_sharing',        'Photo/video sharing in chat',            TRUE,  FALSE, FALSE),
    ('document_sharing',     'PDF/doc sharing in chat',                TRUE,  FALSE, FALSE),
    ('location_sharing',     'Location sharing in chat',               TRUE,  FALSE, FALSE),
    ('real_talk',            'RealTalk practice prompts (nudge + hub)', TRUE,  FALSE, FALSE),
    ('video_calls',          'WebRTC audio/video calls',               TRUE,  FALSE, FALSE),
    ('study_pods',           'Group study pods',                       TRUE,  FALSE, FALSE),
    ('teacher_marketplace',  'Teacher booking & payouts',              TRUE,  FALSE, FALSE),
    ('payout_teacher',       'Payout dashboard for teachers',          TRUE,  FALSE, FALSE),
    ('xp_leaderboard',       'League/ranking tables',                  TRUE,  FALSE, FALSE),
    ('group_chat',           'Multi-person chat rooms',                TRUE,  FALSE, FALSE),
    ('gdpr_export',          'GDPR data export flow',                  TRUE,  FALSE, FALSE),
    ('social_feed',          'Public community activity feed',         TRUE,  FALSE, FALSE);
```

---

## Appendix B — Technical Implementation Notes

### B.1 Backend Feature Flag Service

Replace the existing env-based `services/feature_flags.go` with a DB-backed service:

```go
type FeatureFlagService struct {
    db    *sql.DB
    cache *lru.Cache // key -> *ResolvedFlag, TTL 30s via stamp
}

type ResolvedFlag struct {
    Key     string `json:"key"`
    Enabled bool   `json:"enabled"`
}

func (s *FeatureFlagService) ResolveForUser(ctx context.Context, userID uuid.UUID, isAdmin, isBeta bool) (map[string]bool, error)
func (s *FeatureFlagService) IsEnabled(ctx context.Context, userID uuid.UUID, key string, isAdmin, isBeta bool) (bool, error)
```

Resolution must:
1. Check per-user override in `feature_flag_overrides`.
2. Fall back to flag row: `stable || (beta_access && isBeta) || (admin_only && isAdmin) || default_state`.
3. Cache per-user result for 30s.
4. Invalidate cache on override write or flag tier change.

### B.2 Middleware

```go
func RequireFlag(svc *services.FeatureFlagService, key string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := UserFromContext(r.Context())
            enabled, err := svc.IsEnabled(r.Context(), user.ID, key, user.Role == "admin", user.BetaAccess)
            if err != nil || !enabled {
                http.Error(w, "feature not enabled", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

Register `RequireFlag` on every flagged route in `cmd/server/main.go`.

### B.3 Client Hook

Web:
```typescript
export function useFeatureFlag(key: string): boolean {
  const { flags } = useAuth(); // resolved map from /users/me/flags
  return flags[key] ?? false;
}
```

Mobile:
```typescript
export function useFeatureFlag(key: string): boolean {
  const flags = useStore(state => state.featureFlags); // fetched on login
  return flags[key] ?? false;
}
```

Refresh on login and foreground resume.

### B.4 Migration of Legacy Env Flags

Add rows for `learning_v3_engine` and `redis_session_cache` with `default_state` matching
current env behavior, then remove the old `FeatureFlagService` and update callers to use the
new DB-backed service.

### B.5 Gating Existing Untested Features

For each feature in §9, wrap the handler route with `RequireFlag` and the UI with
`useFeatureFlag`. Examples:
- `POST /calls` → `RequireFlag(svc, "video_calls")`
- `POST /teachers` → `RequireFlag(svc, "teacher_marketplace")`
- Marketplace tab in mobile → `useFeatureFlag('teacher_marketplace')`

---

*Revision history: 2026-09-21 — rewritten to a minimal Release 1 + fast-follow model.*
