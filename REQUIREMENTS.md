# Chorus — Phase 1 Requirements

## 0. Phase 0 (First Release)

Phase 0 is the first deployable milestone, shipping ahead of the full Phase 1 scope: a **mobile-first** surface with launch-blocking fixes and growth essentials.

### Phase 0 Scope

**Bugs / Quick Fixes**
- Home button link works from the dashboard.
- Admin screen has a back affordance to the dashboard.
- Premium plans page copy matches the enforced message cap (open item: copy shows "280 words", a fix requests "28 words per message" — confirm the server-enforced value, see P4a note).
- Emoji picker in the message input box.

**Features**
- Contacts & Invites: scan/import device contacts, detect which are already on Chorus, and run a robust invite flow (SMS / WhatsApp / email) so friends and family can join.
- Onboarding captures first and last name (display name composed from them, still editable).
- Profile avatars — Phase 0 uses generated avatars (initials/color); image upload follows when attachment infrastructure exists.
- Meaningful error handling replaces the generic "Sorry, something went wrong" surface.

**Platform**
- Mobile (Capacitor Android/iOS) is the primary working surface; web tracks it.
- Phase 0 deployment timeline and release gates are tracked under the GitHub "Phase 0 Release" milestone.

### Phase 0 Out of Scope
WhatsApp OTP, AI tutor, premium cosmetics, tutoring marketplace, teacher ecosystems, and competitive research land in later phases (see §1.5 and §7b).

## 1. Overview

Chorus ("chorus.talk") is a multilingual real-time messenger that lets users **learn a language while they communicate**. Every message is automatically translated into the recipient's target language, and every conversation becomes a learning opportunity. Phase 1 delivers a fully functional chat product with automatic translation, grammar feedback, personalized learning plans, and a horizontally scalable server architecture.

## 2. Goals

- Users can send and receive real-time chat messages via chorus.talk.
- Translation of messages is **automatic** and delivered **within 500 ms**.
- Grammar analysis is performed on messages to surface learning opportunities.
- Users can create and manage **custom learning plans and tutorials** (positioned as more adaptive than Duolingo).
- Language learning is woven into messaging in a **fun, gamified** way.
- Architecture is **scalable**: stateless chat servers behind a Layer 4 load balancer with a Redis-backed routing registry.
- All features exposed in Phase 1 must be fully functional (no placeholders or stubs).

## 3. Scope

**In scope:** registration/auth, profiles and language preferences, direct chats, real-time messaging, automatic translation, grammar analysis, custom learning plans/tutorials, message history, presence, and typing indicators.

**Out of scope (Phase 1):**
- File/image attachments
- Voice and video calls (including call transcription)
- Full-text message search across chats (Phase 2+)
- Public social discovery / friend suggestions / forums
- Community moderation tooling beyond basic safety rails

## 4. Functional Requirements

### Profile & Authentication

- **FR-1** Users must be able to register, log in, and log out (password + JWT access/refresh tokens).
- **FR-2** Each user profile must declare a native (source) language and one or more target languages.
- **FR-3** Users must be able to update their target languages at any time from settings.
- **FR-19** Registration/onboarding must capture first and last name; a display name is composed from them and remains editable.
- **FR-20** Every user has a profile avatar. Phase 0: generated from initials/color. Image upload follows when file/attachment infrastructure exists.

### Messaging (chorus.talk)

- **FR-4** Users must be able to create direct chats, send messages, and receive messages in real time over WebSockets.
- **FR-5** Messages must be delivered to all online participants of the chat.
- **FR-6** A chat server must be able to route a message to a target user connected via WebSocket to a *different* chat server (see NFR-7).
- **FR-7** Each message must present both the original text and its automatic translation into the recipient's native language.
- **FR-8** Chat history must be persisted and retrievable with pagination.
- **FR-9** Presence (online/offline) and typing indicators must be shown to chat participants.
- **FR-21** The message input must support inserting emojis; emojis pass through the translation pipeline unchanged.

### Translation

- **FR-10** Incoming messages must be translated automatically — no manual "translate" action required.
- **FR-11** The sender always sees the original message; the recipient sees the original plus the translation in their language.
- **FR-12** Message language must be auto-detected before translation so the correct target is used.
- **FR-13** Translation results must be cached (per source text + target language) so repeated content hits the 500 ms target (NFR-1).

### Learning & Grammar

- **FR-14** Grammar analysis must run on messages, returning corrections, patterns, and CEFR-framed feedback to the learner.
- **FR-15** Users must be able to create custom learning plans/tutorials (e.g., a sequence of lessons or grammar drills) adapted to their target language and level.
- **FR-16** The learning experience must be engaging and fun (gamification such as streaks or XP) — but any gamified feature that ships must be fully functional.
- **FR-17** Saving/championing learned words and phrases from a conversation must work as part of the learning loop.

### Contacts & Invites (Phase 0)

- **FR-22** Users can scan/import device contacts (with permission; raw contact data is never uploaded — matched on-device or via hashed values) to identify which contacts are already on Chorus.
- **FR-23** Users can invite off-platform contacts to join Chorus via SMS / WhatsApp / email using single-use invite links or tokens, and can track invite status (sent / accepted / pending).

### WhatsApp OTP (Phase 1)

- **FR-24** Users can verify their phone number via a WhatsApp OTP. (Phase 1; requires WhatsApp Business API.)

### Functional Completeness

- **FR-18** Every feature exposed in Phase 1 must work end-to-end: no dead buttons, placeholders, or stubs.

## 5. Non-Functional Requirements

### Performance

- **NFR-1** Automatic translation must complete within **500 ms** (p95) from message receive to translation being available to the recipient.
- **NFR-2** End-to-end message delivery to an online recipient must feel instantaneous (< 1 s p95).
- **NFR-3** Chat history pagination must respond within 500 ms (p95) under normal load.

### Scalability & Architecture

- **NFR-4** Chat servers must be **stateless**: all runtime state lives in shared Redis/PostgreSQL, never in server memory.
- **NFR-5** Scaling must be horizontal — adding chat servers must increase capacity without downtime.
- **NFR-6** A **Layer 4 load balancer** must front multiple chat servers and distribute both WebSocket and HTTP traffic.
- **NFR-7** Chat servers must maintain a **Redis-backed registry mapping user → server/WebSocket connection** so a message entering any server can be routed to a user connected on any other server.
- **NFR-8** **Redis is never the source of truth.** Redis carries pub/sub events and the routing registry; PostgreSQL is the durable store.
- **NFR-9** All chat servers must share one logical backend through the load balancer; WebSocket connections must survive the loss of an individual chat server via reconnect.

### Reliability & Durability

- **NFR-10** Messages must be persisted to PostgreSQL before acknowledgment; a Redis failure must not lose a message.
- **NFR-11** If the translation provider fails or times out, chat must keep working — the original message is delivered and translation is retried/queued.
- **NFR-12** WebSocket clients must auto-reconnect with exponential backoff, and reconnect must preserve chat continuity (messages are re-synchronized from history).

### Security

- **NFR-13** TLS must encrypt all traffic in transit (WSS+HTTPS at the load balancer).
- **NFR-14** Only authenticated users may access chats they belong to; all chat/message reads are authorization-checked.
- **NFR-15** JWT access tokens must be short-lived; refresh tokens must be revocable server-side.
- **NFR-16** WebSocket connections must be authenticated; no anonymous sessions may send messages.
- **NFR-17** Secrets and API keys must not be committed to the repository.

### Observability & Operations

- **NFR-18** Logs, metrics (connection, message, and translation latency), and health endpoints must be available for every chat server.
- **NFR-19** The load balancer must health-check chat servers and stop routing to unhealthy ones.
- **NFR-20** Deployments must be reproducible (Docker images, environment-driven configuration).
- **NFR-21** Public endpoints (registration, login, translation) must be rate-limited to resist abuse and cost exhaustion.

### Compatibility & Compliance

- **NFR-22** Mobile (Capacitor Android/iOS) is the **primary working surface**; the web remains supported with consistent functionality but tracks the mobile-first UX.
- **NFR-23** A data retention policy must be defined for stored messages and translation content (GDPR-oriented), even if enforced minimally in Phase 1.

## 6. Architectural Assumptions

- Redis Pub/Sub is the Phase 1 event-delivery mechanism (chosen over Kafka for low-latency, transient real-time events).
- The routing registry (user → server/connection) lives in Redis, so a **Layer 4** LB suffices (no server-affinity/sticky sessions required).
- Translation uses a provider chain (primary + fallback, e.g., OpenRouter/paid models) with caching to hit the 500 ms target.

## 7. Monetization & Premium (Phase 1.5)

### Plans & Pricing

- **P1** Two paid plans: **Premium Monthly — $7.99/mo** and **Premium Yearly — $79.90/yr** ("2 months free"; list price $95.88 struck through). No one-time/lifetime tier; subscriptions are recurring only.
- **P2** Every account starts on the **Free** plan. Users can upgrade to Premium from the app; purchases are processed by **PayPal Billing (Subscriptions)**.
- **P3** There are **no fixed per-day usage quotas** (e.g., "N messages/day") for Free or Premium. The only plan limitations are the ones listed under P4 — with a single exception for the AI tutor's daily allowance (P16), which is a separate product decision.

### Plan Limitations (P4 — the ONLY plan differentiators)

- **P4a. Message size** — Free messages are capped at **280 words** (translation/grammar limit); Premium messages at **1,000 words**. Larger messages are blocked client-side and server-side with a clear message. (Phase 1 shipped a character-based cap; the word-based cap supersedes it.)
  > **Open item (dev sync Aug 16):** the premium plans page shows "Live translation up to 280 words"; a fix requests the copy "28 words per message". Confirm which value is server-enforced before changing either the page or this requirement (tracked in GitHub #42).
- **P4b. Grammar mode** — Free users get **manual/lazy grammar analysis** (analyze on request, e.g., a per-message action); Premium users get **automatic grammar analysis** on every message. Grammar *results* remain visible to everyone; only automation differs.
- **P4c. Response priority / experience** — Premium receives faster translation pipeline priority and an ad-free experience; Free may see ads. Premium-only cosmetics (badge) may ship.

### Subscription Lifecycle (PayPal)

- **P5** Checkout must be a **server-side subscription creation** via the PayPal Billing API with the user's id in `custom_id`; the client then redirects to PayPal's approval URL.
- **P6** Webhooks (with signature verification) drive the lifecycle:
  - `BILLING.SUBSCRIPTION.ACTIVATED` → grant Premium, set `premium_since`, store `subscription_id`.
  - `PAYMENT.SALE.COMPLETED` (and the equivalent billing-plan payment event) → confirm/extend, refresh `next_billing_date`.
  - `BILLING.SUBSCRIPTION.CANCELLED` / `EXPIRED` → start the **grace period** (until the end of the paid period) then downgrade to Free.
  - `BILLING.SUBSCRIPTION.SUSPENDED` / failed payment → grace period; `PAYMENT.SALE.REVERSED` / `REFUNDED` → immediate downgrade.
- **P7** Grace uses the existing `plan_grace_until` machinery: during grace the user keeps Premium entitlements; afterwards `Resolve` downgrades to Free.
- **P8** Webhook events must be idempotently recorded (`subscription_events`) and audit trail of plan changes kept (`plan_changes`: from, to, actor, reason, timestamp).

### Self-Service (User)

- **P9** The plan badge shown after login must be **clickable** and lead to a `/premium` page.
- **P10** `/premium` must reflect current state: Free users see upgrade CTA + monthly/annual pricing (with "2 months free" on annual); Premium users see their active plan, next billing date, and a **manage link** (PayPal subscription dashboard) plus cancel/change via PayPal.
- **P11** Entitlements/plan state must refresh after purchase (e.g., refetch on returning to the app).

### Admin Console (Premium Management)

- **P12** Admin console must expose Premium analytics: total premium users, stored vs. in-grace counts, monthly/yearly mix, projected MRR, new/churned this month, and top users by usage.
- **P13** Admins can **grant Premium** to a user temporarily (N days), until a date, or indefinitely (with a reason), **extend**, and **revoke** (immediately or with a grace window). All changes are recorded with the acting admin.
- **P14** Admins can view a user's plan history (grant/revoke/webhook changes with reasons).

### Emails & Notifications

- **P15** Notify users on Premium activation, entering grace, and after downgrade. (Phase 1.5 may record/queue; delivery wiring is follow-up.)

### Future Premium (Phase 1.5 backlog)

- **P16** AI tutor chatbot — included for Premium users; Free users get a limited daily allowance (this is the single P3 exception). Open decision: premium-only vs. free daily allowance.
- **P17** Premium stickers/emojis/themes as paid cosmetics (extension of P4c).
- **P18** Hall of Fame / Patreon-style contributions for users who want to support Chorus financially.

## 7b. Phase 2+ (Future Scope)

Items from product brainstorming (dev sync Aug 16) that are explicitly **out of scope** for Phase 0/1/1.5. Tracked as GitHub epics/issues.

- **Live tutoring marketplace** — a separate tab with real tutors: profiles, certifications, ratings, and class sign-up (per hour, monthly, bundles). Platform keeps 10–20%. Video + text sessions. Focus on UX and teacher livelihoods.
- **Teacher group chats** — teachers share custom learning materials with students.
- **Public language-pair group chats** (e.g., EN→ES) — open membership; teachers answer questions; unique teacher avatars/tags; discovery funnel into 1-on-1 tutoring.
- **Group chat size limit** — research the ideal maximum (proposed 100 users).
- **Competitive research** — harvest HelloTalk (and similar) reviews; run AI analysis against the Chorus.talk mission/vision to produce prioritized improvement requirements.
- **Accelerated learning** — research scientific methods for accelerated language acquisition and incorporate them into the learning strategy.

## 8. Open Questions / Possible Gaps

1. **Onboarding** — Where do Phase 1 users find their first chat partner (waitlist invite, public room, friend-by-invite)? *Partially resolved by Phase 0 Contacts & Invites (FR-22/FR-23): friends/family invite flow. Public room and teacher discovery remain Phase 2+.*
2. **Report/block** — Minimal report and block capability is recommended even for Phase 1; confirm scope.
3. **Delivery semantics** — Define sent/delivered/read states (if "read receipts" ship) or confirm they're out of scope.
4. **Translation cost capping** — *Resolved by P4:* no per-day quotas; cost is bounded by the 280-word message cap and manual (lazy) grammar mode on Free. Revisit only if a separate hard monthly cap is desired.
5. **Message retention default** — how long to keep messages and translated content by default.
6. **PayPal webhook re-verification** — `PAYPAL_WEBHOOK_ID` must be configured at deploy time; see `backend/internal/config/config.go`.