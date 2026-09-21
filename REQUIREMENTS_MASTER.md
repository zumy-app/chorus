# Chorus — Single Source of Requirements

> **Authority:** This file is the machine-readable backlog. It is kept in sync with
> `MVP_RELEASE_PLAN.md` (narrative + design rationale) and `crew/phase_status.json`
> (runtime status). In any conflict, this file wins for implementation scope;
> `MVP_RELEASE_PLAN.md` wins for design rationale and wireframe mapping.
>
> **Philosophy change (2026-09-21):** The previous waterfall "Phase 0 → 4" gate model is
> retired. Delivery is now **iterative**: thin vertical slices ship continuously. The Sprint
> model is defined in `MVP_RELEASE_PLAN.md §3`. Work items below are grouped by Sprint and
> flagged with their audience (`[all]` = all users, `[beta]` = beta + admin, `[admin]` =
> admin only). Untested/experimental features live behind feature flags — general users never
> see them until a flag is promoted to `stable`.
>
> Last revised: 2026-09-21

---

## Global Definition of Done (applies to every work item)

- Feature is end-to-end functional — no `TODO`, `stub`, `placeholder`, or
  "Sorry, something went wrong" paths in any shipped surface.
- **Mobile (Expo RN, Android first, then iOS)** is the primary surface. Web parity follows
  within the same sprint unless the item is mobile-only by nature.
- `go build ./... && go test ./...` in `backend/` = green.
- `npm test` (vitest) in `frontend/` = green.
- `npm test` (jest) in `mobile/` = green.
- Playwright e2e smoke passes against a running dev stack.
- Durability rule: **persist before ack**; Redis is never the source of truth.
- No secrets in repo; `.env*` are gitignored.
- Feature-flagged items: flag resolves correctly for the target audience tier
  (`admin` / `beta` / `stable`) before the item is marked done.

---

## Sprint 0 — Foundation & Green Baseline  ·  status: IN_PROGRESS

**Goal:** a known-green, runnable monorepo that all later sprints build on. No new features.

- [ ] F0-1  Backend builds & `go test ./...` green.
- [ ] F0-2  `frontend` builds (`tsc && vite build`) & `npm test` green.
- [ ] F0-3  `mobile` jest suite green & Android debug build succeeds on device/emulator.
- [ ] F0-4  `docker compose -f docker-compose.dev.yml config` valid; dev stack boots.
- [ ] F0-5  Dev environment documented in `RUN_GUIDE.md`; canonical run commands work.
- [ ] F0-6  Dead/secret artifacts removed (`tmp_*`, stray `.env` history, stray logs).
- [ ] F0-7  EAS Build configured (`.github/workflows/eas.yml`); Android debug APK produced
             in CI; OTA update channel set up so JS-only changes bypass store review.
- [ ] F0-8  GitHub Actions `ci.yml`: build → test → push image → deploy to `chorus-dev`
             Dokploy project; quality gates block promotion on failure.

**Sprint 0 exit:** CI green end-to-end; dev stack boots; Android build succeeds in CI.

---

## Sprint 1 — MVP: Production Mobile App  ·  status: NOT_STARTED

This sprint produces the first production release. Every item must ship before general
availability. Items marked `[admin]` are visible only to the internal team; `[all]` means
all authenticated users.

### FF-0: Feature Flag Infrastructure  [all]
- [ ] FF-0.1  Create `feature_flags` table and `feature_flag_overrides` table (schema in
               `MVP_RELEASE_PLAN.md Appendix A`).
- [ ] FF-0.2  `GET /users/me/flags` — returns resolved flag map for the authenticated user.
               Resolution logic: override → stable → beta_access → admin_only (see §1.2 in plan).
- [ ] FF-0.3  30-second in-process LRU cache on flag resolution; invalidated on override write.
- [ ] FF-0.4  Mobile client: `useFeatureFlag(key)` hook; flag map fetched on session start,
               refreshed on foreground resume.
- [ ] FF-0.5  Web client: equivalent `useFeatureFlag` hook for admin panel and web app.
- [ ] FF-0.6  Seed all initial flags from `MVP_RELEASE_PLAN.md Appendix A` via migration.
- [ ] FF-0.7  Admin panel: `/admin/features` — list flags, toggle tier, per-user override,
               audit log of changes (who/when/old→new).
- [ ] FF-0.8  Admin panel: `/admin/beta-users` — list beta users, bulk-promote from waitlist
               position range, revoke beta access.

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
- [ ] DB-1.6  `waitlist` table: add `referral_code`, `referred_by_code`, `referral_count`,
               `position`.
- [ ] DB-1.7  `user_settings` table: add `push_notifications_enabled`, `daily_reminder_time`,
               `marketing_emails_enabled`.

### S1-1: Home / Landing Page  [unauthenticated]
- [ ] S1-1.1  Marketing landing page: value prop, feature highlights, social proof with live
               waitlist count from DB.
- [ ] S1-1.2  Waitlist join CTA with email capture; position counter + progress bar.
- [ ] S1-1.3  Referral mechanic: unique `referral_code` per waitlist entry, copy-link +
               Twitter share buttons; each confirmed referral moves position up 100 spots
               (`referral_count * 100` subtracted from base position).
- [ ] S1-1.4  Log In CTA → login screen; "Join Waitlist" CTA → waitlist screen.
- [ ] S1-1.5  Mobile-responsive layout; matches `chorus_home_complete_showcase` wireframe.

### S1-2: Auth — Login & Registration  [unauthenticated]
- [ ] S1-2.1  Email + password login (already exists; verify working end-to-end on mobile).
- [ ] S1-2.2  Google OAuth "Continue with Google" — wire OAuth flow on mobile + web.
- [ ] S1-2.3  Registration: full name + email + password; `first_name` + `last_name` captured.
- [ ] S1-2.4  Forgot password: send recovery email link; password reset screen.
- [ ] S1-2.5  Language switcher pill in top bar (EN default; persisted to `user_settings`).
- [ ] S1-2.6  "WhatsApp, Apple & Facebook login coming soon" teaser chip (non-functional).
- [ ] S1-2.7  Typed error responses for login/register (invalid credentials, duplicate email,
               rate-limited) — replace any generic "Something went wrong" messages.
- [ ] S1-2.8  Rate limiting on `/auth/login` and `/auth/register` (extend existing rate limiter).

### S1-3: Onboarding — 3-Step Flow  [new users]
Shown once only, on first login. Completion sets `users.onboarding_completed = true`.
Each step saves to `onboarding_checkpoints` so partial completion can be resumed.

**Step 1 — Profile Identity:**
- [ ] S1-3.1  Screen: avatar picker (initials + color palette; upload CTA reserved/non-functional).
- [ ] S1-3.2  Fields: first name (required), last name (optional), display name/handle (required,
               availability check via `GET /users/check-username`), mobile phone number with
               country code picker (required, stored in `users.phone`), bio (required, 20–160
               chars, AI topic suggestion chip), city/location (optional).
- [ ] S1-3.3  "Continue to Languages" CTA; saves checkpoint step=1.

**Step 2 — Language Setup:**
- [ ] S1-3.4  Screen: native language selector (flag + name, full dropdown), target language
               selector (flag + name, quick-switch chips for top 8 languages + "More...").
- [ ] S1-3.5  CEFR self-selection: Beginner (A1–A2), Intermediate (B1), Advanced (B2).
               Maps to `user_language_profiles.current_cefr_level` and sets
               `placement_status = 'self_selected'`.
- [ ] S1-3.6  AI insight callout: "Don't worry — Chorus adapts as you chat."
- [ ] S1-3.7  "Continue to Goals" CTA; saves checkpoint step=2.

**Step 3 — Goals & Interests:**
- [ ] S1-3.8  Daily commitment selector: Casual (+10 XP/day target), Regular (+25 XP/day,
               pre-selected, "Popular" badge), Intensive (+50 XP/day).
               Saves to `user_language_profiles.daily_commitment`.
- [ ] S1-3.9  Interest tag cloud (multi-select, ≥2 required): Food & Tapas, Travel & Adventures,
               Music & Arts, Work & Careers, Sports & Fitness, Movies & Shows, Daily Life &
               Culture, Books & Tech. Saves to `user_language_profiles.interests[]`.
- [ ] S1-3.10 Daily practice notification toggle (default ON).
               Saves to `user_settings.push_notifications_enabled`.
- [ ] S1-3.11 "+50 XP Profile Bonus unlocked!" reward pill shown at bottom; awards 50 XP via
               XP service on step 3 completion.
- [ ] S1-3.12 "Complete Profile & Start Chatting" CTA → Dashboard; sets
               `users.onboarding_completed = true`, saves checkpoint step=3.

### S1-4: XP & Streak System  [all]
- [ ] S1-4.1  `XPService`: `AwardXP(userID, eventType, referenceID)` — validates daily cap,
               inserts `xp_events` row, updates `users.xp_total` + `users.xp_level`.
- [ ] S1-4.2  Level thresholds: 1=0, 2=200, 3=500, 4=1000, 5=2000, 6=4000, 7=8000, 8=15000.
               Level label computed from total XP (see `MVP_RELEASE_PLAN.md §4.2`).
- [ ] S1-4.3  Streak service: on any XP award, check `streak_last_activity_date`; if yesterday
               → increment `streak_days`; if today → no-op; if older → reset to 1.
- [ ] S1-4.4  `GET /users/me/xp` → `{ total, level, levelLabel, streakDays, todayXP, weeklyXP }`.
- [ ] S1-4.5  `GET /users/:id/xp` → `{ total, level }` (public, viewer-safe).
- [ ] S1-4.6  XP events wired for MVP: `profile_complete` (+50), `message_sent` (+2, cap 40/day),
               `message_received_read` (+1, cap 20/day), `referral_registered` (+100),
               `referral_onboarded` (+150), `daily_goal_casual` (+10), `daily_goal_regular` (+25),
               `daily_goal_intensive` (+50), `streak_maintained` (+10).
- [ ] S1-4.7  Dashboard header: streak badge (🔥 Xd) and XP badge (⚡ XXX XP) wired to real
               `GET /users/me/xp` data.

### S1-5: Dashboard  [all]
- [ ] S1-5.1  Top bar: logo, streak badge, XP badge, user avatar + language badge.
- [ ] S1-5.2  Welcome banner: personalised greeting; on first visit show referral attribution
               ("Invited by X") if `referred_by_code` is set.
- [ ] S1-5.3  Active Conversations section: list of conversations ordered by last message time,
               each card showing partner avatar, name, online dot, language pair, last message
               preview with translation, timestamp, unread count.
- [ ] S1-5.4  AI Tutor card (Sparky/Chorus AI): always present, shows quick prompt chips,
               taps into the existing AI bot conversation flow.
- [ ] S1-5.5  Word Collector teaser card: shows 3 recent vocabulary words from today's chats
               (or stub "Start chatting to build your deck" if no words yet). "Collect & Save"
               CTA navigates to word library. Card visible to all users; flashcard practice
               gated behind `word_flashcards` flag.
- [ ] S1-5.6  Sneak Peek Showcase section: Video Calls, Study Pods, Teacher Marketplace cards
               with "Notify Me (X interested)" buttons. Clicking "Notify Me" increments an
               interest counter in DB and confirms with a toast — no navigation.
- [ ] S1-5.7  Bottom navigation: Chats | Hub | Learn (3 tabs, matching wireframe).

### S1-6: Chats & Partner Discovery  [all]
- [ ] S1-6.1  Search bar + filter pills: All, Unread (with count badge), Native Speakers, Tutors.
- [ ] S1-6.2  Active conversations list: per-card avatar, name, CEFR badge, language pair,
               last message + translation snippet, timestamp, unread badge.
- [ ] S1-6.3  Suggested Learning Partners section: horizontal scrollable cards, each showing
               match % (from `partner_match_scores`), native/learning language, "Why we matched"
               AI insight chip, "Start Chat" CTA.
- [ ] S1-6.4  Partner match score computation: nightly background job that scores
               complementary language pair (+50), shared interests (+10 each, max +30),
               similar CEFR (±1 level, +10), same timezone (+5), online now (+5).
               Stores results in `partner_match_scores`. Triggered also on profile update.
- [ ] S1-6.5  Filter pills on partner suggestions: Native Speakers, Online Now, Near Me, Filters.
- [ ] S1-6.6  New conversation flow: tap a suggested partner → confirm → open chat room.

### S1-7: Chat Room — Core Messaging  [all]
- [ ] S1-7.1  Chat header: back arrow, partner avatar + online dot, name, CEFR badge,
               language pair pill (ES ⇆ EN), "Ask AI" button.
- [ ] S1-7.2  Streak + today's focus bar: "Daily Streak: X days · Today's Focus: [grammar topic]".
- [ ] S1-7.3  Message stream: incoming bubbles (original text + translation substrip + word
               highlights), outgoing bubbles (sent text + collapsible translation), date dividers.
- [ ] S1-7.4  Automatic translation on receive: source language auto-detected; translated to
               user's native language. Translation shown as substrip with "Auto-translated (ES→EN)"
               label and purple left border. Toggle show/hide.
- [ ] S1-7.5  Vocabulary word highlights: new words in incoming messages highlighted (primary-fixed
               background for vocabulary, secondary underline for grammar points). Tapping a word
               opens a word detail card (definition + level badge + "Save to library" button).
- [ ] S1-7.6  Grammar drawer ("Ask AI" / tap grammar-highlighted word): AI insight card
               (explains grammar rule), Sparky tip bubble, 3 contextual quick-prompt chips.
               `grammar_deep_dive` flag: ON for all users at MVP (flag pre-set to `stable`).
- [ ] S1-7.7  Word Collector chips row: detected vocabulary words shown as tappable chips below
               incoming messages. "+ Add to Library" button saves all at once. `word_collector`
               flag: ON for all users at MVP (flag pre-set to `stable`).
- [ ] S1-7.8  Input composer: text field, add attachment button (non-functional, shows "coming
               soon" toast), voice mic button (non-functional, `voice_message` flag gated),
               AI hint spark button (opens grammar drawer), send button.
- [ ] S1-7.9  Real-time translation preview indicator in input bar ("Typing in English will
               suggest Spanish").
- [ ] S1-7.10 WebSocket message delivery (already implemented; verify parity on mobile).
- [ ] S1-7.11 Read receipts: sent → delivered → read ticks (visual only in Sprint 1; full
               delivery tracking in Sprint 2).
- [ ] S1-7.12 XP award on message send (+2, daily cap enforced).

### S1-8: Hub Tab (minimal)  [all]
- [ ] S1-8.1  Partner suggestions full-page view (same data as S1-6.3, full scrollable list).
- [ ] S1-8.2  "Invite a Friend" card: user's referral link, copy-link + share buttons.
               Referral XP reward displayed: "+150 XP when friend completes onboarding".
- [ ] S1-8.3  Coming Soon cards: Study Pods, Teacher Marketplace — each with interest counter
               ("X interested", tap to register interest).
- [ ] S1-8.4  Safe Learning Pledge badge: brief explanation of community trust guidelines.

### S1-9: Learn Tab (minimal)  [all]
- [ ] S1-9.1  CEFR progress bar: A1 → C2 with current level dot; level label (e.g. "B1 —
               Intermediate").
- [ ] S1-9.2  Daily streak card: 🔥 X day streak + "at risk" indicator if no activity today.
- [ ] S1-9.3  "Today's Focus" card: driven by current CEFR level and language pair (from seeded
               curriculum data); shows grammar topic for the day.
- [ ] S1-9.4  Word Collector summary: total words saved, X due for review.
               "Review Now" CTA gated behind `word_flashcards` flag (shows "Coming Soon" chip
               for general users until Sprint 2).
- [ ] S1-9.5  Placement Test card: gated behind `placement_test` flag (shows "Coming Soon"
               chip for general users until Sprint 3).
- [ ] S1-9.6  Scenario Role-play card: gated behind `scenario_roleplay` flag (shows "Coming
               Soon" chip for general users until Sprint 4).

### S1-10: User Profiles  [all]
- [ ] S1-10.1 Own profile: cover photo area, avatar (photo or initials+color), name, CEFR badge
               (e.g. "A2 Explorer"), location, active timezone hours.
- [ ] S1-10.2 XP + Level card: "X XP — Level N", level label.
- [ ] S1-10.3 Active Streak card: "X day streak".
- [ ] S1-10.4 Language pair(s) section: native + learning with flags.
- [ ] S1-10.5 Interests tag cloud (from onboarding data).
- [ ] S1-10.6 Bio text.
- [ ] S1-10.7 Edit profile flow: inline edit for bio, city, interests, display name, avatar.
- [ ] S1-10.8 Other user profile: same layout + match % chip, "Send Message" CTA,
               "Invite to Study Pod" CTA (gated: `study_pods` flag, shows greyed-out for now).
- [ ] S1-10.9 Safe Learning Pledge badge on profile when `safe_learning_pledge = true`.
               Toggle available in profile settings.
- [ ] S1-10.10 CEFR League ranking card: gated behind `xp_leaderboard` flag.

### S1-11: Infrastructure & Infra Hardening  [internal]
- [ ] S1-11.1 Mail server isolation: Mailu on separate Docker network, only port 587 open from
               app servers; SPF/DKIM/DMARC enforced; `SMTP_PASSWORD` moved to Dokploy secret.
- [ ] S1-11.2 Rate limiting: extend to `/auth/login`, `/translation/quick`, WebSocket
               connections. Existing Redis token-bucket applies.
- [ ] S1-11.3 Push notification token registration: FCM token stored on mobile login
               (`device_tokens` table). Actual message delivery in Sprint 2; this wires the
               infrastructure.
- [ ] S1-11.4 Android Play Store beta track: build AAB via EAS, submit to internal test track.
- [ ] S1-11.5 iOS TestFlight: build IPA via EAS, submit for TestFlight distribution.

**Sprint 1 exit / MVP launch criteria (from `MVP_RELEASE_PLAN.md §12`):**
- Beta D7 retention > 30%
- Onboarding completion rate > 70%
- Translation p95 < 500ms
- Crash-free sessions > 99%
- App Store rating (beta) > 4.2

---

## Sprint 2 — Chat Depth & Word Collecting  ·  status: NOT_STARTED

Flags promoted to `stable` this sprint: `grammar_deep_dive`, `word_collector`.
(Both are pre-set stable at MVP; this sprint validates and hardens them.)

- [ ] S2-1   Promote `word_flashcards` flag → **stable** after flashcard session works end-to-end.
- [ ] S2-2   Spaced-repetition (SM-2) flashcard practice: session of ≥5 cards from saved
              vocabulary; correct/incorrect scoring; next-review date updated. XP +15 per session.
- [ ] S2-3   Push notification delivery: daily streak reminder at user's `daily_reminder_time`;
              new message notification; uses FCM tokens registered in Sprint 1.
- [ ] S2-4   Read receipts full implementation: `message_deliveries` table; sent → delivered →
              read ticks persisted and rendered. Existing partial implementation wired to UI.
- [ ] S2-5   Message reply (inline quote) and message delete (soft delete, "Message deleted" tombstone).
- [ ] S2-6   Phone OTP verification: Twilio/AWS SNS integration; 6-digit code; verify sets
              `users.phone_verified = true`. Used for account recovery.
- [ ] S2-7   Settings screen: translation on/off, grammar auto-analysis on/off, highlights on/off,
              notification preferences, privacy (last seen, profile photo visibility).
- [ ] S2-8   Simplify Chat Language Settings: remove "other person's language" dropdown; keep
              only own language setting (FR-35).

---

## Sprint 3 — Learning Engine Core  ·  status: NOT_STARTED

Flags promoted to `stable` this sprint: `word_flashcards` (if not done in S2), `ai_writing_assistant`.
Flags promoted to `beta` this sprint: `placement_test`.

- [ ] S3-1   Promote `ai_writing_assistant` flag → **stable**.
- [ ] S3-2   Promote `placement_test` flag → **beta**.
- [ ] S3-3   AI writing assistant "Help me write" in message composer: tap spark icon → AI
              suggests a sentence in the target language given current conversation context;
              user edits before sending. Does not auto-send.
- [ ] S3-4   Your Learning Path (Learn tab): real metrics wired — words/month chart,
              sentences understood, XP/streak history. Replaces mock 1.2k/342 data.
              (`GET /learning/dashboard` already exists in backend; wire to UI.)
- [ ] S3-5   CEFR level recalibration: after 20+ messages sent, recalculate level estimate
              from grammar analysis accuracy; surface "You're improving! Want to update your
              level?" prompt.
- [ ] S3-6   Placement test UI: series of listening/reading/matching questions at A1–B2 levels;
              submits result to `POST /learning/placement`; updates CEFR level.
              Gated behind `placement_test` flag (beta only until Sprint 4).
- [ ] S3-7   Seed + personal + unified SRS queue: interleaves seeded curriculum items with
              personally mined vocabulary; SM-2 scheduling. (Existing `#17/19/20` work.)
- [ ] S3-8   GDPR data export: `GET /gdpr/export` → zip of user data. Wires existing
              `handlers/gdpr.go`; gated behind `gdpr_export` flag → **stable** this sprint.

---

## Sprint 4 — Engagement & Social  ·  status: NOT_STARTED

Flags promoted to `stable` this sprint: `scenario_roleplay`, `xp_leaderboard` (opt-in).

- [ ] S4-1   Promote `scenario_roleplay` flag → **stable**.
- [ ] S4-2   Promote `xp_leaderboard` flag → **stable** (opt-in display).
- [ ] S4-3   AI scenario role-play: Learn tab entry + in-chat entry. Scenarios: restaurant,
              travel, daily shopping, job interview (seeded). Turn-taking with AI, corrections
              surfaced after each message. XP +25 per completed scenario.
- [ ] S4-4   XP leaderboard / league: weekly reset; Bronze (top 20% of week's XP), Silver, Gold.
              Displayed on profile card. Opt-in (user must turn on in settings to appear).
- [ ] S4-5   Weekly quest system: 3 rotating quests per week (e.g. "Send 20 messages +50 XP",
              "Learn 5 new words +75 XP", "Complete 1 scenario +100 XP"). Progress shown on
              dashboard.
- [ ] S4-6   Partner matching refinement: weight interests (top 3 matching topics), timezone
              overlap (prefer ±3hr), CEFR level match, activity recency. Re-run nightly.
- [ ] S4-7   Referral milestone rewards: at 1, 5, 10 successful referrals, award bonus XP and
              a profile badge. Badges stored in a new `user_badges` table.
- [ ] S4-8   Promote `placement_test` flag → **stable**.

---

## Sprint 5 — Communication Parity  ·  status: NOT_STARTED

- [ ] S5-1   Message forward and pin.
- [ ] S5-2   Media sharing: photos in chat (upload to object storage, thumbnail preview).
              Promote `media_sharing` flag → **beta** then **stable**.
- [ ] S5-3   Universal message + contact search.
- [ ] S5-4   Archive & mute conversations (backend exists; wire to mobile UI).
- [ ] S5-5   Promote `voice_message` flag → **beta**: in-chat voice recording + playback.
- [ ] S5-6   Document sharing PDF/DOCX. Promote `document_sharing` flag → **beta**.
- [ ] S5-7   Block & report UX: surfaced on every profile card and message long-press.
- [ ] S5-8   Privacy settings full UI: last seen, profile photo visibility, contacts visibility.

---

## Sprint 6 — Audio Calls  ·  status: NOT_STARTED

- [ ] S6-1   Promote `voice_message` flag → **stable**.
- [ ] S6-2   WebRTC audio call: real signaling server (not stub). Caller/receiver screens.
              `handlers/call.go` exists; wire to real WebRTC.
- [ ] S6-3   Live transcription + translated captions: scrollable, bookmark phrase to SRS.
- [ ] S6-4   Call screen UI (mobile + web): call controls, transcript panel, bilingual captions.
- [ ] S6-5   Promote `video_calls` flag → **beta** (audio-only first; video UI stubbed).

---

## Sprint 7 — Video Calls  ·  status: NOT_STARTED

- [ ] S7-1   Promote `video_calls` flag → **stable**.
- [ ] S7-2   WebRTC video call: dual-view / PiP, screen sharing.
- [ ] S7-3   Immersive bilingual caption overlay on video.
- [ ] S7-4   Study Pods: group chat rooms (3–8 members), daily topic prompt, bilingual host.
              Promote `study_pods` flag → **beta**.

---

## Sprint 8 — Scaled Architecture  ·  status: NOT_STARTED

- [ ] S8-1   Layer 4 load balancer (leastconn) fronts chat servers; no sticky sessions.
- [ ] S8-2   Redis connection registry `ws:registry:{userId}` → `{serverID, connID}`.
- [ ] S8-3   Cross-server routing: publisher on S1 → `server:{S2}` channel → S2 delivers.
- [ ] S8-4   Durable per-recipient delivery fully wired end-to-end.
- [ ] S8-5   Load/soak test (Artillery): 1k WS, 50 msg/s, 24h, zero message loss.
- [ ] S8-6   Promote `study_pods` flag → **stable**.
- [ ] S8-7   Word mining pipeline: auto-mine new vocabulary from chat messages; route to SRS.

---

## Sprint 9 — Teacher Marketplace  ·  status: NOT_STARTED

- [ ] S9-1   Promote `teacher_marketplace` flag → **beta**.
- [ ] S9-2   Teacher sign-up: become_a_teacher form (basic info, expertise, video intro, rate).
- [ ] S9-3   Browse tutors: filters (language, rating, price, availability), trial credit flow.
- [ ] S9-4   Tutor profile: video intro, specialties, reviews, pricing, booking widget.
- [ ] S9-5   Teacher dashboard: earnings, availability calendar, student list, profile checklist.
- [ ] S9-6   Booking flow: confirm → lesson → post-lesson review notes → SRS push.
- [ ] S9-7   Payments/payouts: PayPal payout integration, platform fee (10/15%),
              payout history. Promote `payout_teacher` flag → **beta**.
- [ ] S9-8   Monetization: Free=280-char, Premium=1000-char; 1 trial credit/month.
- [ ] S9-9   Promote `teacher_marketplace` flag → **stable** after QA + vetting process review.
- [ ] S9-10  Promote `payout_teacher` flag → **stable**.

---

## Sprint 10 — Operations & Monetisation Hardening  ·  status: NOT_STARTED

- [ ] S10-1  CI/CD automatic dev→prod promotion with quality gates (NFR-26).
- [ ] S10-2  GDPR data retention policy: automated deletion of messages older than
              `message_retention_days` per user settings (NFR-23).
- [ ] S10-3  Support runbook; observability dashboards (Prometheus + Grafana + Phoenix) live.
- [ ] S10-4  Release gate doc + Go/No-Go checklist for marketplace launch.
- [ ] S10-5  Advanced learning: depth-of-processing practice ladder
              (recognition → cued → free → production → spontaneous).
- [ ] S10-6  Teacher vetting process: doc-only + assessment harness.

---

## Always-ON (Admin-only, not sprint-gated)

These items are maintained continuously and are never exposed to general users unless
explicitly promoted via flag management:

- Admin quality review panel (`admin_quality.go`) — flag: internal
- Admin translation review (`admin_translations.go`) — flag: internal
- Admin user management (`admin_users.go`) — flag: internal
- Admin waitlist management (`admin_waitlist.go`) — flag: internal
- Moderation tools (`moderation.go`) — flag: internal
- Arize Phoenix quality pipeline: translation/grammar lineage + cross-model evaluation —
  always running in background, not user-visible

---

## Explicitly Deferred (not in any sprint)

- Full-text call transcript search across all users (post-marketplace).
- Public social discovery / dating-solicitation features — not a dating product.
- HelloTalk-style public community feed as peer-to-peer social — Phase 3+ low priority.
- Native app video filters / AR camera effects.
- Multi-language learning (learning 2 languages simultaneously) — single language pair
  only at MVP; multiple pairs unlocked in a future sprint.

---

## Feature Flag Quick Reference

| Flag Key | Current Tier | Stable Sprint | Code location |
|----------|-------------|---------------|---------------|
| `grammar_deep_dive` | stable | MVP | `handlers/grammar.go` |
| `word_collector` | stable | MVP | `handlers/vocabulary.go` |
| `word_flashcards` | beta | Sprint 2 | `handlers/vocabulary.go`, `models/vocabulary.go` |
| `ai_writing_assistant` | beta | Sprint 3 | `handlers/grammar.go` |
| `placement_test` | beta | Sprint 4 | `handlers/learning.go`, `models/learning_placement.go` |
| `scenario_roleplay` | beta | Sprint 4 | `handlers/learning.go`, `models/learning_scenario.go` |
| `xp_leaderboard` | beta | Sprint 4 | new |
| `voice_message` | admin | Sprint 5/6 | new |
| `media_sharing` | admin | Sprint 5 | `handlers/gallery.go` |
| `document_sharing` | admin | Sprint 5 | `handlers/attachment.go` |
| `video_calls` | admin | Sprint 7 | `handlers/call.go` |
| `study_pods` | admin | Sprint 8 | new |
| `group_chat` | admin | Sprint 8+ | `handlers/chat.go` (partial) |
| `teacher_marketplace` | admin | Sprint 9 | `handlers/teacher.go` |
| `payout_teacher` | admin | Sprint 9 | `handlers/payout.go` |
| `location_sharing` | admin | Sprint 5+ | `handlers/location.go` |
| `gdpr_export` | admin | Sprint 3 | `handlers/gdpr.go` |
| `social_feed` | admin | deferred | — |

---

*Revision: 2026-09-21 — converted from waterfall phase model to iterative sprint model.
Feature flag system introduced. XP system and onboarding gaps added.
Maintained by the supervisor loop; `crew/phase_status.json` mirrors sprint status.*
