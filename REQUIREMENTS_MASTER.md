# Chorus — Single Source of Requirements

> **Authority:** This file is the consolidated, phase-ordered source of truth that the
> autonomous pipeline reads. It supersedes the scattered docs for *planning and gating*
> purposes. The narrative docs (`REQUIREMENTS.md`, `PHASE_1_IMPLEMENTATION_PLAN.md`,
> `PHASE_2_3_IMPLEMENTATION.md`, `wireframes/*.md`) remain the **design narrative**; this
> file is the **machine-readable backlog**. Section metadata is used by `crew/loop.py`.

---

## Phase Status Legend
- `status`: NOT_STARTED | IN_PROGRESS | QA_GATE | DONE | BLOCKED
- A phase is `DONE` only when: (a) all its features are implemented with no stubs/placeholders
  in the shipped surfaces, (b) build + unit + e2e + mobile tests pass (green gate), and
  (c) the QA + teacher reviewers sign off.

---

## 0. Global Definition of Done (applies to every phase)

- **Mobile (Expo RN, Android + iOS) and Web (Vite) both** expose the feature — the app is
  *mobile-first, web parity (NFR-22)*.
- No `TODO`/`stub`/`placeholder`/"Sorry, something went wrong" paths in shipped UX.
- `go build ./... && go test ./...` in `backend/` = green.
- `npm test` (vitest) in `frontend/` = green.
- `npm test` (jest) in `mobile/` = green.
- Playwright e2e (`e2e/`) smoke passes against a running dev stack.
- Backend achieves the durability rule: **persist before ack**, durable per-recipient delivery,
  Redis never the source of truth (PostgreSQL is).
- No secrets in repo (NFR-17); `.env*` are gitignored.
- Each phase's `ESCALATION.md` is empty (no unresolved blocker).

---

## 1. PHASE 0 — Foundation & Green Baseline

**Goal:** a known-green, runnable monorepo that all later phases build on. No new features.

- [ ] P0-1 Backend builds & `go test ./...` green (28 existing tests).
- [ ] P0-2 `frontend` builds (`tsc && vite build`) & `npm test` green.
- [ ] P0-3 `mobile` jest suite green & Android debug build succeeds.
- [ ] P0-4 `docker compose -f docker-compose.dev.yml config` valid; dev stack boots.
- [ ] P0-5 Create `docs/` working-set + phase status tracking (this file + `crew/phase_status.json`).
- [ ] P0-6 Remove/flag dead artifacts (`.env.prod`, `tmp_*`, stray logs) so secrets are not committed.
- [ ] P0-7 Document canonical run commands in `RUN_GUIDE.md` so the pipeline can invoke them.

---

## 2. PHASE 1 — Launch-Blocking Core (P0)  ·  status: NOT_STARTED

### 2.1 Bugs & UX quick wins (P0)
- [ ] 1.1 Home button link works from dashboard (#40).
- [ ] 1.2 Admin → dashboard back affordance on mobile + web (#41).
- [ ] 1.3 Premium copy 280 vs 28 — confirm server word cap, then fix copy or cap to match (#42).
- [ ] 1.4 Emoji picker in message input; emojis pass through translation unchanged (#43) **[MISSING]**.
- [ ] 1.5 Replace generic error "Sorry, something went wrong" with typed errors + toast + retry (#47).

### 2.2 Identity & onboarding (P0)
- [ ] 2.1 Onboarding first + last name; editable composed displayName (#45).
- [ ] 2.2 Avatar from initials/color, deterministic; upload path reserved (#46).
- [ ] 2.3 Onboarding level self-selection (Beginner/Intermediate/Advanced → CEFR seed) (#45/18).
- [ ] 2.4 Contacts & Invites epic: permission-gated scan, on-platform detect (hashed), invite via
          SMS/WhatsApp/email single-use links + status tracking (#44) **[PARTIAL=>SI]**.

### 2.3 Translation & grammar hardening
- [ ] 3.1 Feature toggles FR-25: `translation_enabled`, `grammar_auto`, `highlights_enabled` in
          `user_settings`; gate enqueue; Settings UI + API.
- [ ] 3.2 Learned-word optimization FR-26: `FilterKnownWords`, per-word cache, skip known words.
- [ ] 3.3 Simplify Chat Language Settings FR-35 (remove "other person's language" dropdown).
- [ ] 3.4 Highlight new words FR-27/28 in `MessageBubble` + tap-to-save + practice CTA.
- [ ] 3.5 Quality pipeline FR-30: persist lineage + cross-model evaluator + KPIs; Arize Phoenix (NFR-25).
- [ ] 3.6 Improve in-chat `presence` + `typing` rendering on web (currently omitted) (FR-9).

### 2.4 Learning engine core
- [ ] 4.1 Your Learning Path FR-31: real metrics (words/sentences per month, XP/streak, due, CEFR).
- [ ] 4.2 AI writing assistant FR-33 "Help me write" **[MISSING]**.
- [ ] 4.3 Seed + personal + unified SRS queue FR-32 (#17/19/20).
- [ ] 4.4 Scenario role-play FR-34 polish + entry points (Learn + Chat).

### 2.5 Security, NFRs, infra
- [ ] 5.1 Mail server isolation NFR-22 (Mailu off app network, 587-only, SPF/DKIM/DMARC).
- [ ] 5.2 Rate limiting coverage NFR-24 (extend beyond waitlist/register: login, translation; WS).
- [ ] 5.3 Observability NFR-18/25: `/health` + `/metrics`, Prometheus/Grafana hooks, Phoenix traces.
- [ ] 5.4 Dev environment + CI/CD quality gates NFR-26 (GitHub Actions `ci.yml`).

**Phase 1 gate:** all P0 closed, e2e on dev passes, mobile builds, no generic errors, p95 basics met.

---

## 3. PHASE 2 — Production Messaging Parity + Audio Call Captions

### 3.1 Messaging parity (WhatsApp/Telegram class)
- [ ] 6.1 Read receipts & delivery status ticks (sent/delivered/read, per recipient) **[PARTIAL]**.
- [ ] 6.2 Message actions: reply+forward+delete+pin **[PARTIAL]**.
- [ ] 6.3 Message + media search (universal) **[PARTIAL]**.
- [ ] 6.4 Archive & mute conversations **[MISSING]**.
- [ ] 6.5 Media gallery (photos/videos/links per chat) **[MISSING]**.
- [ ] 6.6 Document sharing (PDF/doc/xlsx) **[MISSING]**.
- [ ] 6.7 Location sharing **[MISSING]**.

### 3.2 Privacy & security
- [ ] 7.1 Block & report (have it) + surfaced UX everywhere.
- [ ] 7.2 WhatsApp OTP phone verification / 2FA **[MISSING]** (FR-24).
- [ ] 7.3 Privacy settings: last seen, profile photo visibility, contacts visibility **[MISSING]**.

### 3.3 Audio calls with smart captions
- [ ] 8.1 WebRTC audio call (real signaling, not stub).
- [ ] 8.2 Live transcription + translated captions, scrollable, bookmark/save-to-SRS per phrase.
- [ ] 8.3 Client UI (web + mobile): call screen, controls, transcript panel.

**Phase 2 gate:** messaging-parity features end-to-end; audio call with live+translated
scrollable captions works on both surfaces; DURABLE delivery + `ws:registry` cross-server routing live.

---

## 4. PHASE 3 — Video Call + Advanced Learning + Scaled Architecture

### 4.1 Video calling
- [ ] 9.1 WebRTC video call (dual-view / PiP), screen sharing, immersive captions.

### 4.2 Scaled architecture (Level 4 LB + Redis registry)
- [ ] 10.1 **Layer 4 load balancer** (leastconn) fronts chat servers; no sticky sessions.
- [ ] 10.2 **Redis connection registry** `ws:registry:{userId}` → `{serverID, connID}`; written on
          connect, cleared on disconnect, TTL heartbeat.
- [ ] 10.3 **Cross-server routing**: recipient on server S2 ⇒ S1 persists, looks up registry,
          publishes to `server:{S2}` channel; S2 delivers + marks delivered. (Round-trip tested.)
- [ ] 10.4 Chat servers horizontally scalable (stateless; state in Postgres/Redis only).
- [ ] 10.5 Durable per-recipient delivery fully wired (inbox service active, not discarded).
- [ ] 10.6 Load/soak test (Artillery): 1k WS, 50 msg/s, 24h drain, message loss = 0.

### 4.3 Advanced learning
- [ ] 11.1 Word mining pipeline: track new words from messages + auto-mine + route to curriculum.
- [ ] 11.2 Depth-of-processing practice ladder (recognition→cued→free→production→spontaneous).
- [ ] 11.3 Scenario/real-talk hub + group-study sandbox hooks.
- [ ] 11.4 Teacher vetting process note (doc-only) + assessment harness.

**Phase 3 gate:** video call works; L4 LB + Redis registry + cross-server delivery proven;
word-mining feeds the SRS; soak test passes with zero loss.

---

## 5. PHASE 4 — Teacher Marketplace + Monetization + Release

### 5.1 Teacher marketplace
- [ ] 12.1 Sign up as a teacher (become_a_teacher: basic info, expertise/certs, video + rate).
- [ ] 12.2 Browse tutors / find trial tutor (filters, rating, verified, trial credit).
- [ ] 12.3 Tutor profile (video intro, specialties, reviews, pricing) + booking/scheduling.
- [ ] 12.4 Teacher dashboard (earnings, availability, students, profile checklist).
- [ ] 12.5 Session flows: confirm booking, trial credit dashboard, lesson review notes.
- [ ] 12.6 Post-lesson SRS push: teacher pushes drills/cards to student queue (shared sandbox).
- [ ] 12.7 Payments/payouts: Paypal payouts, platform fee (10/15%), payout history/settings.

### 5.2 Monetization reconciliation
- [ ] 13.1 Credits & Access model: Free=280-char, Premium=1000-char; 1 trial credit/mo; 10% fee for PM users.

### 5.3 Release & operations
- [ ] 14.1 CI/CD automatic dev→prod promotion with quality gates (NFR-26) handed off.
- [ ] 14.2 Data retention policy + GDPR notes (NFR-23) minimal enforcement (#26).
- [ ] 14.3 Release gate doc (#36) + Go/No-Go checklist (#37) satisfied.
- [ ] 14.4 Support runbook; observability dashboards live.

**Phase 4 gate:** marketplace + payouts functional; dev→prod auto-promotion works; Go/No-Go signed.

---

## 6. Backlog Items NOT in scope (explicitly deferred / flagged)

- Full-text call transcript search across all users (post-mvp).
- Public social discovery / dating-solicitation platform features (the wireframe trust/safety +
  dating modes are *moderation* surfaces, kept minimal, not a dating product).
- HelloTalk-style public community feed as a peer-to-peer social app (community feed is a
  Phase 2+ nice-to-have, low priority).
- Group study hub / live group sessions (social) — Phase 3 nice-to-have; not launch-blocking.

> These are listed so the pipeline does NOT burn effort building them before core messaging is ready.

---

*Revision: 2026-08-31 — reorganized for autonomous multi-phase execution. Maintained by the supervisor loop; `crew/phase_status.json` mirrors it.*
