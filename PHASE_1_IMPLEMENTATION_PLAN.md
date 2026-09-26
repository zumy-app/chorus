# Sprint 0 & Sprint 1 — Implementation Guide

> **⚠️ This file has been restructured (2026-09-21).**
> The waterfall "Phase 0 → Phase 1" model is retired. This document now covers the two
> foundation sprints: **Sprint 0** (green baseline) and **Sprint 1** (MVP production release).
>
> - For the full iterative roadmap across all sprints, see **`MVP_RELEASE_PLAN.md`**.
> - For the machine-readable backlog with all sprint items and feature flag assignments,
>   see **`REQUIREMENTS_MASTER.md`**.
> - For design rationale, wireframe gap analysis, XP system design, and go-to-market
>   strategy, see **`MVP_RELEASE_PLAN.md`**.
>
> **Nothing has been deleted.** All prior P0/P1/P2 work items from the original plan have
> been mapped to Sprint 0 or Sprint 1 items in `REQUIREMENTS_MASTER.md`. The mapping table
> is in §8 of this document.

---

## Why We Changed

The previous plan asked us to complete translation hardening, grammar, vocabulary, a full
learning path, scenarios, role-play, security hardening, CI/CD, and scaling — all before
a single user could touch the app. That is a waterfall release with a 2–3 month delay to
market and concentrated risk at launch.

The new approach:
1. Ship the smallest genuinely useful version to real users (Sprint 1 = MVP).
2. Every untested or partially-built feature lives behind a **feature flag** so it is
   invisible to general users until validated.
3. Each subsequent 2-week sprint promotes one or two flags to `stable`, adds one clear
   improvement, and ships.

This means the team can work on advanced features (scenarios, placement tests, calls) in
parallel with the live app — users never see half-finished work.

---

## Sprint 0 — Green Baseline

**Goal:** a known-green, runnable monorepo. No new features. All later sprints build on this.

**Exit criteria:** CI green end-to-end; dev stack boots cleanly; Android debug APK produced
in CI; no secrets in repo.

| Item | Description | Owner hint |
|------|-------------|------------|
| F0-1 | Backend `go test ./...` green | batchu |
| F0-2 | `frontend` builds + `npm test` green | gosangiraju |
| F0-3 | `mobile` jest suite green; Android debug build on device/emulator | Kushagra1122 |
| F0-4 | `docker compose -f docker-compose.dev.yml config` valid; dev stack boots | Raju |
| F0-5 | `RUN_GUIDE.md` canonical run commands documented | Raju |
| F0-6 | Dead artifacts removed (`tmp_*`, stray `.env` history, stray logs) | batchu |
| F0-7 | EAS Build configured (`eas.yml` in CI); Android debug AAB produced; OTA update channel set up | Kushagra1122 |
| F0-8 | GitHub Actions `ci.yml`: build → test → push image → deploy to `chorus-dev`; quality gates block on failure | Raju |

**Verify:**
```bash
# Backend
cd backend && go build ./... && go test ./...

# Frontend
cd frontend && npm test

# Mobile
cd mobile && npm test

# Docker
docker compose -f docker-compose.dev.yml config --quiet
```

---

## Sprint 1 — MVP Production Release

**Goal:** ship the first production mobile app with complete, polished flows for: home page,
waitlist, login/register, 3-step onboarding, dashboard, chats + partner discovery, chat room
(send/receive/translate/grammar), Hub tab, Learn tab (minimal), user profiles, and XP/streaks.

All existing untested features are hidden behind feature flags. General users cannot see them.
Admins and beta users see them for internal validation.

**Exit criteria (from `MVP_RELEASE_PLAN.md §12`):**
- Beta D7 retention > 30%
- Onboarding completion rate > 70%
- Translation p95 < 500ms
- Crash-free sessions > 99%
- App Store rating (beta feedback) > 4.2

### Key Architectural Work in Sprint 1

#### 1. Feature Flag System (FF-0)
The single most important Sprint 1 infrastructure item. Everything else in the sprint depends on it.

- New tables: `feature_flags`, `feature_flag_overrides` (schema in `MVP_RELEASE_PLAN.md Appendix A`)
- `GET /users/me/flags` endpoint: resolves flag map for the authenticated user
- Resolution precedence: per-user override → stable → beta_access tier → admin_only tier
- 30-second in-process LRU cache; invalidated on override write
- `useFeatureFlag(key)` hook in both mobile (RN) and web clients
- Admin panel: `/admin/features` — list/toggle flags, per-user overrides, audit log
- Admin panel: `/admin/beta-users` — manage beta cohort, bulk-promote from waitlist

Initial flags seeded at stable (visible to all):
- `grammar_deep_dive`
- `word_collector`

Initial flags seeded at beta (visible to admin + beta users only):
- `word_flashcards`, `ai_writing_assistant`, `placement_test`, `scenario_roleplay`, `xp_leaderboard`

Initial flags seeded at admin-only:
- `video_calls`, `study_pods`, `teacher_marketplace`, `payout_teacher`, `voice_message`,
  `media_sharing`, `document_sharing`, `group_chat`, `location_sharing`, `social_feed`

#### 2. Data Model Migrations (DB-1)
Run in order before any Sprint 1 feature work. Full SQL in `MVP_RELEASE_PLAN.md Appendix A`.

New columns on `users`:
```
bio, city, cover_photo_url, beta_access, xp_total, xp_level,
streak_days, streak_last_activity_date, onboarding_completed,
onboarding_step, safe_learning_pledge, safe_learning_pledged_at
```

New columns on `user_language_profiles`:
```
daily_commitment, interests[], notifications_enabled
```

New tables:
```
onboarding_checkpoints   — multi-step onboarding state
xp_events                — XP ledger (audit + analytics)
xp_daily_caps            — per-event daily cap enforcement
partner_match_scores     — algorithmic partner suggestions
```

Additions to existing tables:
```
waitlist:      referral_code, referred_by_code, referral_count, position
user_settings: push_notifications_enabled, daily_reminder_time, marketing_emails_enabled
```

#### 3. XP & Streak System (S1-4)
XP appears in 7 of 14 wireframe screens. It is load-bearing UI, not cosmetic. Must ship in Sprint 1.

- `XPService.AwardXP(userID, eventType, referenceID)` — validates daily cap, inserts ledger row,
  updates `users.xp_total` + `users.xp_level`
- Streak: on any XP award, compare `streak_last_activity_date` to UTC today; increment or reset
- XP events wired for MVP: `profile_complete` (+50), `message_sent` (+2, cap 40/day),
  `message_received_read` (+1, cap 20/day), `referral_registered` (+100),
  `referral_onboarded` (+150), daily goal events (+10/+25/+50), `streak_maintained` (+10)
- Dashboard header badges wired to real `GET /users/me/xp` data

#### 4. Onboarding — Step 3 (Goals & Interests) — NEW SCREEN
This screen does not exist yet. It must be built from scratch matching the
`onboarding_goals_interests` wireframe.

- Daily commitment radio cards (Casual/Regular/Intensive) with XP/day label
- Interest tag cloud (multi-select chips, ≥2 required)
- Notifications toggle
- "+50 XP Profile Bonus unlocked!" pill at bottom
- "Complete Profile & Start Chatting" CTA

#### 5. Hub Tab — NEW SCREEN
The Hub tab does not exist yet. Minimal MVP implementation:
- Full-page partner suggestions list (same data as Chats suggested section)
- "Invite a Friend" card with referral link + copy/share
- Coming Soon cards for Study Pods and Teacher Marketplace
- Safe Learning Pledge explanation

#### 6. Partner Matching
No algorithmic matching exists. Required for the "Suggested for You" section in Chats and Hub:
- Nightly background job scoring users against each other (language complement, interests,
  CEFR proximity, timezone, online status)
- Stores top-N matches per user in `partner_match_scores`
- `GET /users/me/suggested-partners?limit=10` endpoint

### Sprint 1 Items Carried from Original Plan

The following items from the original P0/P1 plan map directly into Sprint 1:

| Original item | Sprint 1 item | Notes |
|---------------|---------------|-------|
| #40 Home button link | S1-5.x dashboard routing | Fix router |
| #41 Admin back affordance | S1-5.x + admin panel | Add back button |
| #42 Premium copy 280 vs 28 | S1-2.x settings | Verify entitlements cap first |
| #43 Emoji picker | Already done ✓ | Verify on mobile |
| #47 Generic error handling | S1-2.7 typed errors | All surfaces |
| #45 First + last name onboarding | S1-3.1–S1-3.3 | Extended with bio, phone, city |
| #46 Avatar initials/color | S1-3.1 | Part of onboarding step 1 |
| #44 Contacts & Invites | S1-8.2 referral mechanic | Simplified to referral link first; full contact scan in Sprint 2 |
| #7/#58 Expo mobile build | F0-3, F0-7, S1-11.4 | EAS in Sprint 0; Play Store in Sprint 1 |
| FR-25 feature toggles | FF-0 (entire feature flag system) | Expanded to full flag system |
| FR-26 learned-word optimization | Already done ✓ | Verify active |
| FR-35 chat language simplification | S2-8 | Moved to Sprint 2 (low risk) |
| FR-27/28 word highlights + save | S1-7.5–S1-7.7 (word collector) | `word_collector` flag = stable at MVP |
| FR-30 quality pipeline (Phoenix) | Always-on infra | Already implemented; keep running |
| FR-31 learning path real metrics | S3-4 | Sprint 3 (needs sufficient data first) |
| FR-33 AI writing assistant | S3-3 | Sprint 3, behind `ai_writing_assistant` flag |
| FR-34 scenario role-play | S4-3 | Sprint 4, behind `scenario_roleplay` flag |
| Mail server isolation | S1-11.1 | Sprint 1 infra |
| Rate limiting extension | S1-11.2 | Sprint 1 infra |
| Dev env + CI/CD gates | F0-8 | Sprint 0 |
| L4 LB + Redis registry | S8-1–S8-4 | Sprint 8 — not needed for single-server MVP |
| WhatsApp OTP #50 | S2-6 | Sprint 2 (Twilio first) |
| Stickers #52 | Deferred | Low priority |
| AI tutor #51 | S1-5.4 (Sparky card) | Basic at MVP; full in Sprint 3 |

### What Is NOT in Sprint 1

Explicitly excluded from Sprint 1 and hidden behind flags:

- Placement test (Sprint 3, `placement_test` flag)
- Scenario role-play (Sprint 4, `scenario_roleplay` flag)
- Flashcard practice sessions (Sprint 2, `word_flashcards` flag)
- AI writing assistant (Sprint 3, `ai_writing_assistant` flag)
- XP leaderboard / league (Sprint 4, `xp_leaderboard` flag)
- Audio/video calls (Sprint 6/7, `video_calls` flag)
- Study pods (Sprint 8, `study_pods` flag)
- Teacher marketplace (Sprint 9, `teacher_marketplace` flag)
- Payouts (Sprint 9, `payout_teacher` flag)
- Voice messages (Sprint 5/6, `voice_message` flag)
- Document sharing (Sprint 5, `document_sharing` flag)
- Media sharing / gallery (Sprint 5, `media_sharing` flag)
- Location sharing (Sprint 5, `location_sharing` flag)
- Full contact scan/sync (Sprint 2; referral link ships in Sprint 1)
- L4 load balancer / Redis registry (Sprint 8 — single server is fine for MVP scale)

---

## Sprint 1 Owner Assignments (suggested)

| Area | Owner |
|------|-------|
| Feature flag system (FF-0) | batchu |
| DB migrations (DB-1) | batchu / Raju |
| XP & Streak service (S1-4) | batchu |
| Onboarding Step 3 — new screen (S1-3.8–S1-3.12) | Kushagra1122 |
| Hub tab — new screen (S1-8) | Kushagra1122 |
| Partner matching background job (S1-6.4) | gosangiraju |
| Chat room grammar drawer + word chips (S1-7.5–S1-7.7) | Kushagra1122 |
| Dashboard XP/streak wiring (S1-5.1–S1-5.7) | Kushagra1122 |
| Google OAuth (S1-2.2) | gosangiraju |
| Waitlist referral mechanic (S1-1.3) | gosangiraju |
| Admin panel: feature flags + beta users (FF-0.7–FF-0.8) | batchu |
| Infra: mail hardening, rate limiting, EAS (S1-11) | Raju |

---

## Sprint 1 Release Process

1. **Internal dogfood** (admin role): team uses the app with all flags ON for 1 week.
   File and fix all crashes, broken flows, and UX regressions.

2. **Beta cohort launch** (beta_access = true): invite first 500 waitlist users via email.
   Monitor D1/D7 retention, onboarding completion, and crash-free sessions for 1 week.
   Use the admin panel to adjust flags based on feedback (e.g. if grammar_deep_dive causes
   performance issues, flip it back to beta-only while fixing).

3. **General availability**: once beta metrics hit targets (see MVP exit criteria above),
   remove waitlist gate, open Play Store + App Store listings publicly.

4. **OTA updates**: JS-only bug fixes and small polish changes ship as EAS OTA updates —
   no store review delay. Native changes (new permissions, native modules) require a new
   store build.

---

## Teacher Onboarding — Process Note (doc-only, Sprint 9)

Do not build marketplace infrastructure until Sprint 9. The vetting process is documented
for future reference:
- Basic info + expertise/certifications submission
- Live language demo recording upload
- Manual video-call review by a language expert (Daniella for Spanish; recruit per language)
- Rubric documented; future work trains AI evaluator on top of it
- Tracked under epic #53

---

## Quality Pipeline (Always On)

The Arize Phoenix quality pipeline introduced in the original plan is already implemented
and runs continuously:
- Translation jobs lineage captured in `translation_jobs` + `translation_evals`
- Cross-model critique: after each translation/grammar write, enqueue evaluator job
  (different model scores accuracy/fluency/CEFR)
- Phoenix traces via OTLP from `TranslationService` + `GrammarService`
- KPIs: accuracy, p95 latency, cost/1k tokens, cache hit rate
- Dashboard links in admin console

This is infrastructure — not user-facing — and stays running at all times regardless of
sprint. Review Phoenix eval results monthly; if accuracy drops below baseline, refine
prompts and bump `cacheVersion` to invalidate the cache.

---

## Mapping: Original FR/NFR References → Sprint Items

| Original ref | Description | Sprint item |
|-------------|-------------|-------------|
| FR-25 | Feature toggles (translation/grammar/highlights) | FF-0 |
| FR-26 | Learned-word optimisation | Done ✓ |
| FR-27/28 | Word highlight + save | S1-7.5–S1-7.7 |
| FR-31 | Learning path real metrics | S3-4 |
| FR-33 | AI writing assistant | S3-3 |
| FR-34 | Scenario role-play | S4-3 |
| FR-35 | Simplify chat language settings | S2-8 |
| NFR-17 | No secrets in repo | F0-6 |
| NFR-18/25 | Observability: health, metrics, Phoenix | Done ✓ |
| NFR-22 | Mobile-first (Expo RN, Android + iOS) | F0-3, F0-7, S1-11.4/5 |
| NFR-23 | GDPR data retention | S10-2 |
| NFR-24 | Rate limiting extension | S1-11.2 |
| NFR-26 | Dev env + CI/CD quality gates | F0-8 |

---

*Last updated: 2026-09-21. Restructured from waterfall phase doc to Sprint 0 + Sprint 1
implementation guide. Original P0/P1/P2 items fully mapped to sprint items in
`REQUIREMENTS_MASTER.md`. Owner: batchu + leads.*
