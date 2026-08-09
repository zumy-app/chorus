# Chorus — Phase 1 Requirements

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

### Messaging (chorus.talk)

- **FR-4** Users must be able to create direct chats, send messages, and receive messages in real time over WebSockets.
- **FR-5** Messages must be delivered to all online participants of the chat.
- **FR-6** A chat server must be able to route a message to a target user connected via WebSocket to a *different* chat server (see NFR-7).
- **FR-7** Each message must present both the original text and its automatic translation into the recipient's native language.
- **FR-8** Chat history must be persisted and retrievable with pagination.
- **FR-9** Presence (online/offline) and typing indicators must be shown to chat participants.

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

- **NFR-22** chorus.talk must work on Web (desktop/mob) and mobile (Capacitor Android/iOS) with consistent functionality.
- **NFR-23** A data retention policy must be defined for stored messages and translation content (GDPR-oriented), even if enforced minimally in Phase 1.

## 6. Architectural Assumptions

- Redis Pub/Sub is the Phase 1 event-delivery mechanism (chosen over Kafka for low-latency, transient real-time events).
- The routing registry (user → server/connection) lives in Redis, so a **Layer 4** LB suffices (no server-affinity/sticky sessions required).
- Translation uses a provider chain (primary + fallback, e.g., OpenRouter/paid models) with caching to hit the 500 ms target.

## 7. Open Questions / Possible Gaps

1. **Onboarding** — Where do Phase 1 users find their first chat partner (waitlist invite, public room, friend-by-invite)? Not yet specified.
2. **Report/block** — Minimal report and block capability is recommended even for Phase 1; confirm scope.
3. **Delivery semantics** — Define sent/delivered/read states (if "read receipts" ship) or confirm they're out of scope.
4. **Rate limit / quota on translations** — Cost capping per user per day to protect the translation budget.
5. **Message retention default** — how long to keep messages and translated content by default.