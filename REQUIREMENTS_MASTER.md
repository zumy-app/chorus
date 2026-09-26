# Chorus — Single Source of Requirements

> **Authority:** This file is the machine-readable backlog. It is kept in sync with
> `MVP_RELEASE_PLAN.md` (narrative + design rationale) and `crew/phase_status.json`
> (runtime status). In any conflict, this file wins for implementation scope;
> `MVP_RELEASE_PLAN.md` wins for design rationale and wireframe mapping.
>
> **Philosophy change (2026-09-21):** The previous waterfall "Phase 0 → 4" gate model is
> retired. Delivery is now **iterative**: a minimal production release (Release 1) ships
> first, followed by small point releases (R1.1, R1.2, R2, ...). Work items below are
> grouped by release and flagged with their audience (`[all]` = all users, `[beta]` = beta +
> admin, `[admin]` = admin only). Untested/experimental features live behind feature flags —
> general users never see them until a flag is promoted to `stable`.
>
> Last revised: 2026-09-21

---

## Global Definition of Done (applies to every work item)

- Feature is end-to-end functional — no `TODO`, `stub`, `placeholder`, or
  "Sorry, something went wrong" paths in any shipped surface.
- **Carve-out:** an inert affordance is permitted only when it is (a) gated behind a flag OFF
  for that user tier, or (b) shows a labeled "coming soon" toast (never a silent no-op).
- **Mobile (Expo RN, Android first, then iOS)** is the primary surface. Web parity follows
  within the same release unless the item is mobile-only by nature.
- `go build ./... && go test ./...` in `backend/` = green.
- `npm test` (vitest) in `frontend/` = green.
- `npm test` (jest) in `mobile/` = green.
- Playwright e2e smoke passes against a running dev stack.
- Durability rule: **persist before ack**; Redis is never the source of truth.
- No secrets in repo; `.env*` are gitignored.
- Feature-flagged items: flag resolves correctly for the target audience tier
  (`admin` / `beta` / `stable`) **and the gated endpoint rejects (403) when the flag is off**
  — client-side visibility is never the only guard.

---

## Release 0 — Foundation & Green Baseline  ·  status: IN_PROGRESS

**Goal:** a known-green, runnable monorepo that all later releases build on. No new features.

- [ ] F0-1  Backend builds & `go test ./...` green.
- [ ] F0-2  `frontend` builds (`tsc && vite build`) & `npm test` green.
- [ ] F0-3  `mobile` jest suite green & Android debug build succeeds on device/emulator.
- [ ] F0-4  `docker compose -f docker-compose.dev.yml config` valid; dev stack boots.
- [ ] F0-5  Dev environment documented in `RUN_GUIDE.md`; canonical run commands work.
- [ ] F0-6  Dead/secret artifacts removed (`tmp_*`, stray `.env` history, stray logs).
- [ ] F0-7  EAS Build configured (`.github/workflows/eas.yml`); Android debug APK produced
             in CI; OTA update channel set up.
- [ ] F0-8  GitHub Actions `ci.yml`: build → test → push image → deploy to `chorus-dev`
             Dokploy project; quality gates block promotion on failure.

**Release 0 exit:** CI green end-to-end; dev stack boots; Android build succeeds in CI.

---

## Release 1 — Minimal Production Mobile App  ·  status: NOT_STARTED

This release produces the first production release. It is intentionally minimal.
Items marked `[admin]` are visible only to the internal team; `[all]` means all authenticated
users.

### FF-0: Feature Flag Infrastructure  [all]
- [ ] FF-0.1  Create `feature_flags` table and `feature_flag_overrides` table (schema in
               `MVP_RELEASE_PLAN.md Appendix A`). Resolution order: override → stable →
               beta_access → admin_only → `default_state`.
- [ ] FF-0.2  `GET /users/me/flags` — returns resolved flag map for the authenticated user.
- [ ] FF-0.3  30-second in-process LRU cache on flag resolution; invalidated on override write.
- [ ] FF-0.4  Mobile client: `useFeatureFlag(key)` hook; flag map fetched on session start,
               refreshed on foreground resume.
- [ ] FF-0.5  Web client: equivalent `useFeatureFlag` hook for admin panel and web app.
- [ ] FF-0.6  Seed all initial flags from `MVP_RELEASE_PLAN.md Appendix A` via migration.
- [ ] FF-0.7  Admin panel: `/admin/features` — list flags, toggle tier, per-user override,
               audit log of changes (who/when/old→new).
- [ ] FF-0.8  Admin panel: `/admin/beta-users` — list beta users, bulk-promote from waitlist
               position range, revoke beta access.
- [ ] FF-0.9  **Server-side enforcement**: `RequireFlag(flagKey)` middleware returning 403
               when the flag is off; registered on every flagged route in `MVP_RELEASE_PLAN.md
               §9`. Client `useFeatureFlag` is visibility-only, never the control.
- [ ] FF-0.10 Reconcile/deprecate legacy `backend/internal/services/feature_flags.go`
               (env/canary service). Migrate `learning_v3_engine` + `redis_session_cache` to
               DB flag rows with matching `default_state`; mark the env service deprecated.
- [ ] FF-0.11 Kill-switch: resolved flag map included in the authenticated user payload of
               every authenticated API response (or WS hello), so a flag flip propagates
               within one request, not one client session.

### DB-1: Data Model Migrations  [infrastructure]
All migrations from `MVP_RELEASE_PLAN.md Appendix A` in order:
- [ ] DB-1.1  `users` table: add `bio`, `city`, `cover_photo_url`, `beta_access`, `xp_total`,
               `xp_level`, `streak_days`, `streak_last_activity_date`, `onboarding_completed`,
               `onboarding_step`, `safe_learning_pledge`, `safe_learning_pledged_at`.
- [ ] DB-1.2  `user_language_profiles`: add `daily_commitment`, `interests[]`,
               `notifications_enabled`.
- [ ] DB-1.3  Create `onboarding_checkpoints` table.
- [ ] DB-1.4  Create `xp_events` and `xp_daily_caps` tables.
- [ ] DB-1.5  Create `partner_match_scores` table.
- [ ] DB-1.6  `waitlist_entries` table: add `referral_code`, `referred_by_code`,
               `referral_count`, `position`; define position semantics per
               `MVP_RELEASE_PLAN.md §5.4`.
- [ ] DB-1.7  `user_settings` table: add `push_notifications_enabled`, `daily_reminder_time`,
               `marketing_emails_enabled`.
- [ ] DB-1.8  Create `feature_interest` table — Sneak Peek "Notify Me" counters.
- [ ] DB-1.9  Create `device_tokens` table — FCM token registration.
- [ ] DB-1.10 `xp_events`: add `UNIQUE (user_id, event_type, reference_id)` for idempotency;
               adopt existing `xp_awarded` columns on `learning_sessions` / `scenario_runs`.

### R1-1: Home / Landing Page  [unauthenticated]
- [ ] R1-1.1  Marketing landing page: value prop, feature highlights, social proof with live
               waitlist count from DB.
- [ ] R1-1.2  Waitlist join CTA with email capture; position counter + progress bar.
- [ ] R1-1.3  Referral mechanic: unique `referral_code` per waitlist entry, copy-link +
               Twitter share buttons; each confirmed referral moves position up 100 spots.
- [ ] R1-1.4  Log In CTA → login screen; "Join Waitlist" CTA → waitlist screen.
- [ ] R1-1.5  Mobile-responsive layout; matches `chorus_home_complete_showcase` wireframe.

### R1-2: Auth — Login & Registration  [unauthenticated]
- [ ] R1-2.1  Email + password login (already exists; verify working end-to-end on mobile).
- [ ] R1-2.2  Registration: full name + email + password; `first_name` + `last_name` captured.
- [ ] R1-2.3  Forgot password: send recovery email link; password reset screen.
- [ ] R1-2.4  Language switcher pill in top bar (EN default; persisted to `user_settings`).
- [ ] R1-2.5  "WhatsApp, Apple & Facebook login coming soon" teaser chip (non-functional).
- [ ] R1-2.6  Google OAuth button **hidden** until `google_oauth` flag is ON (ships R1.1).
- [ ] R1-2.7  Typed error responses for login/register (invalid credentials, duplicate email,
               rate-limited) — replace any generic "Something went wrong" messages.
- [ ] R1-2.8  Rate limiting on `/auth/login` and `/auth/register` (extend existing rate limiter).

### R1-3: Onboarding — 3-Step Flow  [new users]
Shown once only, on first login. Completion sets `users.onboarding_completed = true`.
Each step saves to `onboarding_checkpoints` so partial completion can be resumed.

**Step 1 — Profile Identity:**
- [ ] R1-3.1  Screen: avatar picker (initials + deterministic color palette; upload CTA reserved).
- [ ] R1-3.2  Fields: first name (required), last name (optional), display name/handle (required,
               availability check via `GET /users/check-username`), mobile phone number with
               country code picker (required, stored in `users.phone`), bio (required, 20–160
               chars, AI topic suggestion chip), city/location (optional).
- [ ] R1-3.3  "Continue to Languages" CTA; saves checkpoint step=1.

**Step 2 — Language Setup:**
- [ ] R1-3.4  Screen: native language selector (flag + name, full dropdown), target language
               selector (flag + name, quick-switch chips for top 8 languages + "More...").
- [ ] R1-3.5  CEFR self-selection: Beginner (A1–A2), Intermediate (B1), Advanced (B2).
               Maps to `user_language_profiles.current_cefr_level` and sets
               `placement_status = 'self_selected'`.
- [ ] R1-3.6  AI insight callout: "Don't worry — Chorus adapts as you chat."
- [ ] R1-3.7  "Continue to Goals" CTA; saves checkpoint step=2.

**Step 3 — Goals & Interests:**
- [ ] R1-3.8  Daily commitment selector: Casual (+10 XP/day target), Regular (+25 XP/day,
               pre-selected, "Popular" badge), Intensive (+50 XP/day).
               Saves to `user_language_profiles.daily_commitment`.
- [ ] R1-3.9  Interest tag cloud (multi-select, ≥2 required): Food & Tapas, Travel & Adventures,
               Music & Arts, Work & Careers, Sports & Fitness, Movies & Shows, Daily Life &
               Culture, Books & Tech. Saves to `user_language_profiles.interests[]`.
- [ ] R1-3.10 Daily practice notification toggle (default ON).
               Saves to `user_settings.push_notifications_enabled`.
- [ ] R1-3.11 "+50 XP Profile Bonus unlocked!" reward pill shown at bottom; awards 50 XP via
               XP service on step 3 completion.
- [ ] R1-3.12 "Complete Profile & Start Chatting" CTA → Dashboard; sets
               `users.onboarding_completed = true`, saves checkpoint step=3.

### R1-4: XP & Streak System  [all]
- [ ] R1-4.1  `XPService`: `AwardXP(userID, eventType, referenceID)` — validates daily cap,
               inserts `xp_events` row, updates `users.xp_total` + `users.xp_level`.
- [ ] R1-4.2  Level thresholds: 1=0, 2=200, 3=500, 4=1000, 5=2000, 6=4000, 7=8000, 8=15000.
- [ ] R1-4.3  Streak service: on any XP award, check `streak_last_activity_date`; if yesterday
               → increment `streak_days`; if today → no-op; if older → reset to 1.
- [ ] R1-4.4  `GET /users/me/xp` → `{ total, level, levelLabel, streakDays, todayXP, weeklyXP }`.
- [ ] R1-4.5  `GET /users/:id/xp` → `{ total, level }` (public, viewer-safe).
- [ ] R1-4.6  XP events wired for Release 1: `profile_complete` (+50), `message_sent` (+2, cap
               40/day), `message_received_read` (+1, cap 20/day), `word_saved` (+3, cap 10/day),
               `grammar_insight_opened` (+5, first per message), `daily_goal_casual` (+10),
               `daily_goal_regular` (+25), `daily_goal_intensive` (+50), `streak_maintained`
               (+10), `referral_registered` (+100), `referral_onboarded` (+150).
- [ ] R1-4.7  Dashboard header: streak badge (🔥 Xd) and XP badge (⚡ XXX XP) wired to real
               `GET /users/me/xp` data.

### R1-5: Dashboard  [all]
- [ ] R1-5.1  Top bar: logo, streak badge, XP badge, user avatar + language badge.
- [ ] R1-5.2  Welcome banner: personalised greeting; on first visit show referral attribution
               ("Invited by X") if `referred_by_code` is set.
- [ ] R1-5.3  Active Conversations section: list of conversations ordered by last message time,
               each card showing partner avatar, name, online dot, language pair, last message
               preview with translation, timestamp, unread count.
- [ ] R1-5.4  AI Tutor card (Sparky/Chorus AI): always present, shows quick prompt chips,
               taps into the existing AI bot conversation flow.
- [ ] R1-5.5  Word Collector teaser card: shows 3 recent vocabulary words from today's chats
               (or stub "Start chatting to build your deck" if no words yet). Tapping a word
               saves it to the user's vocabulary. Full flashcard practice gated behind
               `word_flashcards` flag.
- [ ] R1-5.6  Sneak Peek Showcase section: Video Calls, Study Pods, Teacher Marketplace cards
               with "Notify Me (X interested)" buttons. Clicking "Notify Me" increments an
               interest counter in DB and confirms with a toast — no navigation.
- [ ] R1-5.7  Bottom navigation: Chats | Hub | Learn (3 tabs, matching wireframe).

### R1-6: Chats & Partner Discovery  [all]
- [ ] R1-6.1  Search bar + filter pills: All, Unread (with count badge), Native Speakers, Tutors.
- [ ] R1-6.2  Active conversations list: per-card avatar, name, CEFR badge, language pair,
               last message + translation snippet, timestamp, unread badge.
- [ ] R1-6.3  Suggested Learning Partners section: horizontal scrollable cards, each showing
               match % (from `partner_match_scores`), native/learning language, "Why we matched"
               AI insight chip, "Start Chat" CTA.
- [ ] R1-6.4  Partner match score computation: nightly background job that scores
               complementary language pair (+50), shared interests (+10 each, max +30),
               similar CEFR (±1 level, +10), same timezone (+5), online now (+5).
               Stores results in `partner_match_scores`. Triggered also on profile update.
- [ ] R1-6.5  Filter pills on partner suggestions: Native Speakers, Online Now, Near Me, Filters.
- [ ] R1-6.6  New conversation flow: tap a suggested partner → confirm → open chat room.

### R1-7: Chat Room — Core Messaging  [all]
- [ ] R1-7.1  Chat header: back arrow, partner avatar + online dot, name, CEFR badge,
               language pair pill (ES ⇆ EN), "Ask AI" button.
- [ ] R1-7.2  Streak + today's focus bar: "Daily Streak: X days · Today's Focus: [grammar topic]".
- [ ] R1-7.3  Message stream: incoming bubbles (original text + translation substrip + word
               highlights), outgoing bubbles (sent text + collapsible translation), date dividers.
- [ ] R1-7.4  Automatic translation on receive: source language auto-detected; translated to
               user's native language. Translation shown as substrip with "Auto-translated (ES→EN)"
               label and purple left border. Toggle show/hide.
- [ ] R1-7.5  Vocabulary word highlights: new words in incoming messages highlighted (primary-fixed
               background for vocabulary, secondary underline for grammar points). Tapping a word
               opens a word detail card (definition + level badge + "Save to library" button).
- [ ] R1-7.6  Grammar drawer ("Ask AI" / tap grammar-highlighted word): AI insight card
               (explains grammar rule), Sparky tip bubble, 3 contextual quick-prompt chips.
               `grammar_insights` flag: ON for all users in Release 1.
- [ ] R1-7.7  Word Collector chips row: detected vocabulary words shown as tappable chips below
               incoming messages. "+ Add to Library" button saves all at once. `word_collector`
               flag: ON for all users in Release 1.
- [ ] R1-7.8  Input composer: text field, add attachment button (non-functional, shows "coming
               soon" toast), voice mic button (non-functional, `voice_message` flag OFF),
               AI hint spark button (opens grammar drawer), send button.
- [ ] R1-7.9  Real-time translation preview indicator in input bar.
- [ ] R1-7.10 WebSocket message delivery (already implemented; verify parity on mobile).
- [ ] R1-7.11 Read receipts: sent → delivered → read ticks (visual in Release 1; full
               delivery tracking in R1.2).
- [ ] R1-7.12 XP award on message send (+2, daily cap enforced) and word save (+3, cap 10/day).

### R1-8: Hub Tab  [all]
- [ ] R1-8.1  Today's Conversation Starters: daily phrases with "Use in Chat" CTA.
- [ ] R1-8.2  Community Feature Voting section (`feature_voting` flag stable): upcoming features
               with status + "I'm Interested (X votes)" buttons. Stores in `feature_interest`.
- [ ] R1-8.3  "Invite a Friend" card: user's referral link, copy-link + share buttons.
               Referral XP reward displayed: "+150 XP when friend completes onboarding".
- [ ] R1-8.4  Safe Learning Pledge badge: brief explanation of community trust guidelines.

### R1-9: Learn Tab  [all]
- [ ] R1-9.1  CEFR progress bar: A1 → C2 with current level dot; level label (e.g. "B1 —
               Intermediate").
- [ ] R1-9.2  Daily streak card: 🔥 X day streak + "at risk" indicator if no activity today.
- [ ] R1-9.3  "Today's Focus" card: driven by current CEFR level and language pair (from seeded
               curriculum data); shows grammar topic for the day.
- [ ] R1-9.4  Word Collector summary: total words saved, X due for review.
               "Review Now" CTA gated behind `word_flashcards` flag.
- [ ] R1-9.5  Placement Test card: gated behind `placement_test` flag (shows "Coming Soon").
- [ ] R1-9.6  Scenario Role-play card: gated behind `scenario_roleplay` flag (shows "Coming Soon").

### R1-10: User Profiles  [all]
- [ ] R1-10.1 Own profile: cover photo area, avatar (photo or initials+color), name, CEFR badge
               (e.g. "A2 Explorer"), location.
- [ ] R1-10.2 XP + Level card: "X XP — Level N", level label.
- [ ] R1-10.3 Active Streak card: "X day streak".
- [ ] R1-10.4 Language pair(s) section: native + learning with flags.
- [ ] R1-10.5 Interests tag cloud (from onboarding data).
- [ ] R1-10.6 Bio text.
- [ ] R1-10.7 Edit profile flow: inline edit for bio, city, interests, display name, avatar.
- [ ] R1-10.8 Other user profile: same layout + match % chip, "Send Message" CTA,
               "Invite to Study Pod" CTA (gated: `study_pods` flag).
- [ ] R1-10.9 Safe Learning Pledge badge on profile when `safe_learning_pledge = true`.
               Toggle available in profile settings.
- [ ] R1-10.10 CEFR League ranking card: gated behind `xp_leaderboard` flag.

### R1-11: Infrastructure  [internal]
- [ ] R1-11.1 Rate limiting: extend to `/auth/login`, `/translation/quick`, WebSocket
               connections. Existing Redis token-bucket applies.
- [ ] R1-11.2 Android Play Store internal track: build AAB via EAS, submit.
- [ ] R1-11.3 iOS TestFlight: build IPA via EAS, submit.

**Release 1 exit — two gates (from `MVP_RELEASE_PLAN.md §11`):**

**Pre-launch release gates** (testable before beta, block beta launch):
- Backend `go build/test` green; frontend `tsc && vite build && npm test` green;
  mobile jest + Android EAS build green
- Playwright e2e smoke green on dev stack
- Translation p95 < 500ms on cache-hit set
- Every gated endpoint returns 403 when flag off (FF-0.9)
- Load smoke (100 WS / 10 msg/s / 5 min) no message loss
- Dogfood crash-free sessions > 99%

**Post-launch success metrics** (measured after beta cohorts, gate GA):
- Beta users with ≥1 conversation in first 24h > 60%
- D7 retention > 30%
- Onboarding completion rate > 70%
- Grammar drawer opens per conversation > 1.5
- Referrals sent per active user > 0.3
- App Store rating (beta) > 4.2

---

## Release 1.1 — Google OAuth & Push Tokens  ·  status: NOT_STARTED

Flags promoted to `stable`: `google_oauth`, `push_notifications` → `beta`.

- [ ] R1.1-1  Promote `google_oauth` → **stable**.
- [ ] R1.1-2  Backend `GET /auth/google` + `GET /auth/google/callback`: code exchange,
              create-or-link account, JWT issue.
- [ ] R1.1-3  Env config: `GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET`.
- [ ] R1.1-4  Per-platform redirect URIs: Expo dev (`exp://...`) + prod (`https://auth.chorus.talk/...`).
- [ ] R1.1-5  OAuth `state` nonce with session binding to prevent CSRF.
- [ ] R1.1-6  Scopes `openid profile email`; email-verified check.
- [ ] R1.1-7  Account-linking rule: new user → register with Google identity;
              existing email → link + login.
- [ ] R1.1-8  Mobile `expo-auth-session`; web redirect flow.
- [ ] R1.1-9  Show Google button in login/register when flag is ON.
- [ ] R1.1-10 Promote `push_notifications` → **beta**.
- [ ] R1.1-11 FCM token registration on mobile login; store in `device_tokens` table.
- [ ] R1.1-12 Daily streak reminder at user's `daily_reminder_time` (beta only).

---

## Release 1.2 — Flashcards & Chat Depth  ·  status: NOT_STARTED

Flag promoted to `stable`: `word_flashcards` → **beta**.

- [ ] R1.2-1  Promote `word_flashcards` → **beta** after flashcard session works end-to-end.
- [ ] R1.2-2  Spaced-repetition (SM-2) flashcard practice: session of ≥5 cards from saved
              vocabulary; correct/incorrect scoring; next-review date updated. XP +15 per session.
- [ ] R1.2-3  "Review Now" CTA enabled for beta users in Learn tab.
- [ ] R1.2-4  Read receipts full implementation: `message_deliveries` table; sent → delivered →
              read ticks persisted and rendered.
- [ ] R1.2-5  Message reply (inline quote) and message delete (soft delete, "Message deleted" tombstone).
- [ ] R1.2-6  Phone OTP verification: Twilio/AWS SNS integration; 6-digit code; verify sets
              `users.phone_verified = true`.
- [ ] R1.2-7  Settings screen: translation on/off, grammar auto-analysis on/off, highlights on/off,
               notification preferences, privacy (last seen, profile photo visibility).
- [ ] R1.2-8  Wire server-side enforcement for `grammar_auto` + `highlights_enabled`
               (translation gating already exists in `handlers/message.go`).

---

## Release 2 — AI Writing Assistant  ·  status: NOT_STARTED

Flag promoted to `stable`: `ai_writing_assistant`.

- [ ] R2-1  Promote `ai_writing_assistant` → **stable**.
- [ ] R2-2  "Help me write" in message composer: tap spark icon → AI suggests a sentence in the
            target language given conversation context; user edits before sending.
- [ ] R2-3  Your Learning Path (Learn tab): real metrics wired — words/month, sentences
            understood, XP/streak history. Uses existing `GET /learning/dashboard`.
- [ ] R2-4  CEFR level recalibration: after 20+ messages sent, recalculate estimate from grammar
            analysis accuracy; surface "You're improving! Want to update your level?" prompt.

---

## Release 3 — Placement Test & GDPR Export  ·  status: NOT_STARTED

Flags promoted: `placement_test` → **beta**, `gdpr_export` → **stable**.

- [ ] R3-1  Promote `placement_test` → **beta**.
- [ ] R3-2  Placement test UI: series of listening/reading/matching questions at A1–B2 levels;
            submits result to `POST /learning/placement`; updates CEFR level.
- [ ] R3-3  Promote `gdpr_export` → **stable**.
- [ ] R3-4  `GET /gdpr/export` → zip of user data. Wires existing `handlers/gdpr.go`.

---

## Release 4 — Engagement & Social  ·  status: NOT_STARTED

Flags promoted: `scenario_roleplay` → **stable**, `voice_message` → **beta**,
`media_sharing` → **beta**, `xp_leaderboard` → **stable**.

- [ ] R4-1  Promote `scenario_roleplay` → **stable**.
- [ ] R4-2  Promote `xp_leaderboard` → **stable** (opt-in display).
- [ ] R4-3  AI scenario role-play: Learn tab entry + in-chat entry. Scenarios: restaurant,
            travel, daily shopping, job interview (seeded). XP +25 per completed scenario.
- [ ] R4-4  XP leaderboard / league: weekly reset; Bronze/Silver/Gold. Displayed on profile card.
- [ ] R4-5  Weekly quest system: 3 rotating quests per week.
- [ ] R4-6  Partner matching refinement: weight interests, timezone overlap, CEFR match,
            activity recency. Re-run nightly.
- [ ] R4-7  Referral milestone rewards: at 1, 5, 10 successful referrals, award bonus XP and
            a profile badge. Badges stored in a new `user_badges` table.
- [ ] R4-8  Promote `voice_message` → **beta** and `media_sharing` → **beta**.

---

## Release 5 — Communication Parity  ·  status: NOT_STARTED

- [ ] R5-1  Promote `voice_message` → **stable**.
- [ ] R5-2  Promote `media_sharing` → **stable**.
- [ ] R5-3  Promote `document_sharing` → **beta**.
- [ ] R5-4  Message forward and pin.
- [ ] R5-5  Universal message + contact search.
- [ ] R5-6  Archive & mute conversations (backend exists; wire to mobile UI).
- [ ] R5-7  Block & report UX: surfaced on every profile card and message long-press.
- [ ] R5-8  Privacy settings full UI: last seen, profile photo visibility, contacts visibility.

---

## Release 6 — Audio Calls  ·  status: NOT_STARTED

- [ ] R6-1  Promote `document_sharing` → **stable**.
- [ ] R6-2  Promote `video_calls` → **beta** (audio-only first).
- [ ] R6-3  WebRTC audio call: real signaling server (not stub). Caller/receiver screens.
- [ ] R6-4  Live transcription + translated captions: scrollable, bookmark phrase to SRS.
- [ ] R6-5  Call screen UI (mobile + web).

---

## Release 7 — Video Calls & Study Pods  ·  status: NOT_STARTED

- [ ] R7-1  Promote `video_calls` → **stable**.
- [ ] R7-2  Promote `study_pods` → **beta**.
- [ ] R7-3  Promote `group_chat` → **beta**.
- [ ] R7-4  WebRTC video call: dual-view / PiP, screen sharing.
- [ ] R7-5  Immersive bilingual caption overlay on video.
- [ ] R7-6  Study Pods: group chat rooms (3–8 members), daily topic prompt, bilingual host.

---

## Release 8 — Teacher Marketplace  ·  status: NOT_STARTED

- [ ] R8-1  Promote `teacher_marketplace` → **beta**.
- [ ] R8-2  Promote `payout_teacher` → **beta**.
- [ ] R8-3  Teacher sign-up: become_a_teacher form (basic info, expertise, video intro, rate).
- [ ] R8-4  Browse tutors: filters (language, rating, price, availability), trial credit flow.
- [ ] R8-5  Tutor profile: video intro, specialties, reviews, pricing, booking widget.
- [ ] R8-6  Teacher dashboard: earnings, availability calendar, student list, profile checklist.
- [ ] R8-7  Booking flow: confirm → lesson → post-lesson review notes → SRS push.
- [ ] R8-8  Payments/payouts: PayPal payout integration, platform fee (10/15%), payout history.
- [ ] R8-9  Monetization: Free=280-char, Premium=1000-char; 1 trial credit/month.
- [ ] R8-10 Promote `teacher_marketplace` → **stable** after QA + vetting.
- [ ] R8-11 Promote `payout_teacher` → **stable**.

---

## Always-ON (Admin-only, not release-gated)

- Admin quality review panel (`admin_quality.go`)
- Admin translation review (`admin_translations.go`)
- Admin user management (`admin_users.go`)
- Admin waitlist management (`admin_waitlist.go`)
- Feature flag management (`feature_flags.go`) — new
- Beta user management — new
- Moderation tools (`moderation.go`)

---

## Explicitly Deferred (not in any release)

- Full-text call transcript search across all users (post-marketplace).
- Public social discovery / dating-solicitation features.
- Native app video filters / AR camera effects.
- Multi-language learning simultaneously (single pair in Release 1).

---

## Feature Flag Quick Reference

| Flag Key | Release 1 Tier | Stable Release | Code location |
|----------|---------------|----------------|---------------|
| `grammar_insights` | stable | Release 1 | `services/grammar.go` |
| `word_collector` | stable | Release 1 | `handlers/vocabulary.go` |
| `feature_voting` | stable | Release 1 | new |
| `referral_bump` | stable | Release 1 | `services/waitlist.go` |
| `google_oauth` | admin | R1.1 | new |
| `push_notifications` | admin | R1.1 | new |
| `word_flashcards` | admin | R1.2 | `handlers/vocabulary.go`, `models/vocabulary.go` |
| `ai_writing_assistant` | admin | R2 | `handlers/grammar.go` |
| `placement_test` | admin | R3 | `handlers/learning.go` |
| `scenario_roleplay` | admin | R4 | `handlers/learning.go` |
| `voice_message` | admin | R4 | new |
| `media_sharing` | admin | R4 | `handlers/gallery.go` |
| `document_sharing` | admin | R5 | `handlers/attachment.go` |
| `video_calls` | admin | R6 | `handlers/call.go` |
| `study_pods` | admin | R7 | new |
| `group_chat` | admin | R7 | `handlers/chat.go` (partial) |
| `teacher_marketplace` | admin | R8 | `handlers/teacher.go` |
| `payout_teacher` | admin | R8 | `handlers/payout.go` |
| `xp_leaderboard` | admin | R4 | new |
| `gdpr_export` | admin | R3 | `handlers/gdpr.go` |
| `social_feed` | admin | deferred | — |

---

*Revision: 2026-09-21 — converted to minimal Release 1 + fast-follow model.*
