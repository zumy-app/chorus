# Chorus — Phase 1 Requirements (Consolidated)

> **Consolidation 2026-08-23:** Former **Phase 0** and **Phase 1** are merged. There is now a single **Phase 1** milestone. Everything that was `phase-0` is now `phase-1` with **P0 (launch-blocking)** priority. The GitHub milestone **"Phase 0 Release"** should be closed; all its issues move to **"Phase 1 Release"**. `PHASE_1_IMPLEMENTATION_PLAN.md` is the single execution plan.

## 1. Overview

Chorus ("chorus.talk") is a multilingual real-time messenger that lets users **learn a language while they communicate**. Every message is automatically translated into the recipient's target language, and every conversation becomes a learning opportunity. Phase 1 delivers a fully functional chat product with automatic translation, grammar feedback, personalized learning plans, horizontally scalable architecture, and a production-ready mobile surface.

**Goal for Phase 1 (from dev sync Aug 23):** working language translation, grammar analysis, mobile app, basic important NFRs, security, structured language-learning activities + tracking.

## 2. Goals

- Users can send and receive real-time chat messages via chorus.talk.
- Translation is **automatic** and delivered **within 500 ms** (p95) where cache hits; otherwise queued durably and delivered as soon as ready.
- Grammar analysis is performed on messages (async AI pipeline) with CEFR-framed feedback.
- Users can create and manage **custom learning plans/tutorials** (more adaptive than Duolingo) and track progress on **Your Learning Path**.
- Language learning is woven into messaging: word-bank, highlighting, practice, writing assistance, and scenario role-play.
- Architecture is **scalable**: stateless chat servers behind a Layer 4 load balancer with a Redis-backed routing registry.
- **Mobile (Expo React Native, Android + iOS)** is the **primary working surface**; web tracks it.
- All features exposed in Phase 1 are fully functional (no placeholders).
- Basic NFRs, security, observability, and a proper **dev → prod promotion** pipeline are in place.

## 3. Scope

**In scope (Phase 1 Consolidated):**
registration/auth, profiles & language preferences, direct chats, real-time messaging, automatic translation with feature controls & learned-word optimization, grammar analysis, vocabulary/word-bank, **Your Learning Path** dashboard & metrics, highlight + practice, AI writing assistant & scenario role-play, message history, presence & typing, contacts/invites, onboarding (first/last name, avatar, level), premium plans (PayPal), admin console, rate limiting, observability, translation/grammar quality evaluation, security hardening (incl. mail server isolation), dev environment + CI/CD quality gates, mobile app (Expo).

**Out of scope (Phase 2+):**
- File/image attachments beyond avatar upload
- Voice/video calls and call transcription (call infrastructure deferred)
- Full-text message search across chats (Phase 2+)
- Public social discovery / forums (except design spike for language-pair group chats)
- Tutoring marketplace, teacher group chats, public language-pair group chats (Phase 2 epics)
- Competitive research & accelerated-learning research (Phase 2 research spikes)

## 4. Functional Requirements

### Profile & Authentication
- **FR-1** Register / login / logout (password + JWT access/refresh).
- **FR-2** Each profile declares a native (source) language and one or more target languages.
- **FR-3** Target languages editable any time from Settings (and from chat if FR-25 is enabled).
- **FR-19** Onboarding captures **first and last name**; display name composed from them, remains editable. **[P0]**
- **FR-20** Profile avatar: generated from initials/color in Phase 1; image upload when attachment infra exists. **[P0]**
- **FR-18 self-selection** Onboarding level self-selection (Beginner / Intermediate / Advanced) — maps to CEFR initial + seed path. (Issue #18)

### Messaging (chorus.talk)
- **FR-4** Create direct chats, send/receive in real time over WebSockets.
- **FR-5** Deliver to all online participants of a chat.
- **FR-6** Route a message to a target user connected to a *different* chat server (via Redis registry).
- **FR-7** Each message presents original text + automatic translation into the recipient's native language.
- **FR-8** Persisted, paginated chat history (500 ms p95).
- **FR-9** Presence (online/offline) + typing indicators.
- **FR-21** Emoji picker in message input; emojis pass through translation unchanged. **[P0]**
- **FR-22/23** Contacts & Invites: scan/import device contacts (permission-gated, never upload raw contacts — match on-device or via hashed values), detect on-platform users, invite off-platform via SMS/WhatsApp/email with single-use links. **[P0]**
- **FR-24** WhatsApp OTP phone verification (Phase 1, requires WhatsApp Business API). (Issue #50)

### Translation — Core + Enhancements from Aug 23 notes
- **FR-10** Incoming messages translated **automatically** — no manual action.
- **FR-11** Sender sees original; recipient sees original + translation.
- **FR-12** Auto-detect source language before translation.
- **FR-13** Cache per (source text + target language) for 500 ms target.
- **FR-25 Feature controls** Users can **toggle features per-account** (and later per-chat): auto-translation on/off, auto grammar on/off, learning highlights on/off. Toggles live in Settings and are respected server-side (translation jobs not enqueued when off). Must not break message delivery. *Source: session note "Provide controls for turning on/off features like language translation."*
- **FR-26 Learned-word / word-bank optimization** Maintain a **per-user learned-word bank** (vocabulary with `interval_days >= threshold` or explicit "known"). Translation pipeline consults it to **avoid re-translating / re-highlighting known words** (e.g., "hola", "hi") and to reduce LLM cost. UI copy: words already known are dimmed, not re-translated. *Source: "no need to waste resources for words already translated... keep track of learned words and avoid translating them."*
- **FR-27 Add to word bank from context** From any translated message the user can tap an unknown word → **Add to word bank** (one tap). Already covered by `vocabulary.save` but now requires UX polish + dedup.
- **FR-28 Highlight + practice** **Highlight new/unlearned words** inside translated messages; tapping opens a quick **practice** affordance (flashcard / cloze / pronunciation stub). *Source: "highlight new words in the translated messages and allow users to practice"*
- **FR-30 Quality pipeline (store & evaluate)** **Persist every translation & grammar analysis** with metadata (source/target, provider, latency, cache hit). Run a **cross-model evaluator** (different model from the one that produced the output) to score quality. Surface **KPIs (accuracy, latency, cost)** and a **prompt-critique loop**. *Source: "store all translations/grammar analysis, do analysis of the quality using a different model... critique and refine prompts/improve efficiency. KPIs, accuracy number."*

### Learning & Grammar
- **FR-14** Grammar analysis returns corrections, patterns, CEFR feedback.
- **FR-15** Users can create custom learning plans/tutorials adapted to level.
- **FR-16** Gamified but fully functional (streaks/XP if shipped).
- **FR-17** Save/champion learned words/phrases from conversation.
- **FR-31 Structured activities & Your Learning Path metrics** Dashboard **"Your Learning Path"** shows: **words learned per month, sentences understood per month, XP/streak, due reviews, CEFR progress, translation consumption**. Metrics are time-bucketed (month/week) and feed recommendations. *Source: "user dashboard - learned x number of words by month, understand x number of sentences per month etc.. what are the metrics? These metrics should be shown on Your Learning Path"*
- **FR-32 Seed & personalized learning paths** Seed path: core vocab/grammar sequences per (language pair × level), cached; Personalized items mined from messages & GrammarService (vocab-recall / grammar-cloze); Unified queue interleaves seed + personal with spaced repetition. (Issues #17, #19, #20)
- **FR-33 Writing goals — AI co-writer** Users can **draft a message in their target language**; AI provides **feedback, corrections, and gap-fill completions** before sending. Lives in the composer as "Help me write". *Source: "Learning goals - allow users to draft messages in the language they need to learn. AI should provide feedback/fill in the gaps and work with the users to write messages (part of the writing goals)"*
- **FR-34 Scenario role-play** **AI generates scenarios** (restaurant order, date conversation, etc.) and role-plays with the user in the target language, with gentle corrections. Accessible from Learn tab + Chat. *Source: "Communicate with AI to learn. AI can generate scenarios. you are at a restaurant, place an order. or you are on a date, talking to a date."*
- **FR-35 Chat Language Settings simplification** Settings → Chat Language Settings shows **only the current user's language** (remove "other person's language" dropdown). Target vs native selector only. *Source: "In Chat Language Settings.. remove other persons language setting dropdown. Just focus on your own."* — tracked against `frontend/src/components/ChatLanguageModal.tsx:14-77`.

### Bugs / Launch blockers (former Phase 0 P0)
- **[P0] Home button link works from the dashboard.** (Issue #40)
- **[P0] Admin screen has a back affordance to the dashboard.** (Issue #41)
- **[P0] Premium plans page copy** — confirm server-enforced word cap before editing copy (open item: copy shows "280 words", fix requests "28" — tracked in #42 + P4a note).
- **[P0] Meaningful error handling** replaces generic "Sorry, something went wrong". (Issue #47)
- **[P0] Robust invite flow** already under FR-22/23.

### Functional Completeness
- **FR-18** Every feature exposed in Phase 1 works end-to-end: no dead buttons or stubs.

## 5. Non-Functional Requirements

### Performance
- **NFR-1** Translation available within **500 ms p95** from receive (cache hits; cold misses queued).
- **NFR-2** Online message delivery < 1 s p95.
- **NFR-3** Chat history pagination < 500 ms p95.

### Scalability & Architecture
- **NFR-4** Chat servers **stateless**: state in Redis/PostgreSQL only.
- **NFR-5** Horizontal scaling without downtime.
- **NFR-6** **Layer 4 load balancer** fronts chat servers for HTTP + WebSocket.
- **NFR-7** **Redis-backed registry** `ws:registry:{userId}` for cross-server routing.
- **NFR-8** Redis never source of truth; PostgreSQL is durable.
- **NFR-9** WebSocket reconnect survives loss of an individual server.

### Reliability & Durability
- **NFR-10** Persist before ack; Redis failure must not lose a message.
- **NFR-11** Translation provider failure → deliver original, retry/queue.
- **NFR-12** Auto-reconnect with exponential backoff + history re-sync.

### Security
- **NFR-13** TLS everywhere (WSS+HTTPS at LB).
- **NFR-14** AuthZ on every chat/message read.
- **NFR-15** Short-lived access tokens; revocable refresh.
- **NFR-16** WebSocket authentication; no anonymous sends.
- **NFR-17** No secrets in repo.
- **NFR-22 Security — mail server isolation** Prod mail (Mailu) must **not share the same host/network namespace** as the app servers. Isolate via separate host, VLAN, or container network with minimal ingress (587/25 only from app), add SPF/DKIM/DMARC per Issue #3, rotate `SMTP_PASSWORD`, and scope SMTP creds to env (no `VITE_` prefix). *Source: "Security - Mail server running on prod, same space"*
- **NFR-24** Rate limiting on public endpoints (registration, login, translation) to resist abuse/cost exhaustion.

### Observability, Evaluation & Operations
- **NFR-18** Logs, metrics (connection, message, translation latency), health endpoints per server.
- **NFR-19** LB health-checks; stops routing to unhealthy servers.
- **NFR-20** Reproducible deploys (Docker images, env-driven config).
- **NFR-25 Observability — Phoenix** Deploy **Arize Phoenix** locally (or self-hosted) for **offline + realtime evaluation** of translation/grammar. Traces, datasets, and prompt versions are stored; cross-model critique runs as batch or streaming. Metrics exported to the same Grafana/Prometheus stack. *Source: "KPIs, accuracy number https://arize.com/phoenix/ Deploy it locally, do offline evaluation or realtime evaluation.. realtime analysis"*
- **NFR-26 Quality gates on promotion** `dev` environment gates promotion to `prod`: functional tests, e2e (Playwright), translation/grammar golden-set eval (Phoenix), and load smoke must pass. Assigned to **Raju** (CI/CD). *Source: "create a dev environment, assign to raju who will work on ci/cd, create dev environment, quality gates, run functional, e2e tests on dev before auto promotion to pro"*

### Compatibility & Compliance
- **NFR-22** Mobile (Expo Android/iOS) is **primary**; web is consistent but follows mobile UX.
- **NFR-23** Data retention policy defined (GDPR-oriented), minimally enforced.
- **NFR-27 Teacher vetting (Phase 2 prep in Phase 1)** Process note kept in Phase 1 docs but execution is Phase 2: live recording, certificates, manual video-call review by language experts (Daniella for Spanish), curated expert panel per language; later train AI on rubric. *Source: teacher onboarding notes.*

## 6. Architectural Assumptions

- Redis Pub/Sub for transient real-time events; **durable** triggers live in `translation_jobs` / `grammar_jobs` tables (source of truth), Redis is only the near-real-time notifier.
- Routing registry in Redis, so a **Layer 4** LB suffices (no stickiness).
- Translation via provider chain (OpenRouter primary + DeepSeek fallback) with caching to hit 500 ms.
- Phoenix is deployed as a sidecar/compose service, not in the request hot path.
- Dev and prod are **separate environments** (separate compose/Dokploy projects), promotion is via **container image promotion**, not re-build.

## 7. Monetization & Premium (Phase 1.5 — still separate milestone if desired, but tracked under Phase 1 epic)

### Plans & Pricing
- **P1** Premium Monthly $7.99/mo, Premium Yearly $79.90/yr ("2 months free"). Recurring only.
- **P2** Every account starts Free; upgrade via **PayPal Billing (Subscriptions)**.
- **P3** No per-day quotas except AI tutor daily allowance (P16) exception.

### Plan Limitations (P4 — only differentiators)
- **P4a. Message size** — Free 280 words, Premium 1,000 words. Block client+server. Open item: copy shows "28 words per message" vs "280 words" — confirm server-enforced value (Issue #42).
- **P4b. Grammar mode** — Free manual/lazy; Premium automatic.
- **P4c. Response priority / experience** — Premium faster pipeline + ad-free + badge.

### Subscription Lifecycle / Self-Service / Admin / Emails / Future Premium — unchanged
(see previous REQUIREMENTS.md §7, P5–P18; no material change in this consolidation).

## 7b. Phase 2+ (Future Scope — unchanged)

- Live tutoring marketplace (10–20% take rate, video+text)
- Teacher group chats (custom materials)
- Public language-pair group chats (EN→ES open)
- Group chat size limit research (proposed 100)
- Competitive research (HelloTalk harvest + AI analysis)
- Accelerated learning research
- Agentic AI / database scaling deferred as separate epics.

## 8. Open Questions / Gaps (updated)

1. **Onboarding first partner** — partially resolved by Contacts & Invites; public room + teacher discovery remain Phase 2.
2. **Report/block** minimal scope — confirm in Phase 1 (Issue #27).
3. **Delivery semantics** — sent/delivered/read — confirm if read receipts ship (Issue #28).
4. **Translation cost capping** — resolved by P4 (word cap + lazy grammar); revisit only if hard monthly cap needed.
5. **Message retention default** — define window (Issue #26).
6. **PayPal webhook re-verification** — `PAYPAL_WEBHOOK_ID` at deploy.
7. **(New) Phoenix retention** — how long to keep traces/evals vs. PII redaction.
8. **(New) Learned-word threshold** — `interval_days >= 21` vs. explicit "marked known" vs. both.

---
*History: This file merges the former §0 "Phase 0 (First Release)" into Phase 1. The standalone Phase 0 milestone and `phase-0` labels are deprecated; see `BACKLOG_REFINEMENT_2026-08-23.md` for the per-issue migration.*
