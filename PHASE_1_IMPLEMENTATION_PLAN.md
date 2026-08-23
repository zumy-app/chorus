# Phase 1 Implementation Plan — Consolidated (former Phase 0 + Phase 1)

> **Status 2026-08-23:** Former Phase 0 is merged into Phase 1. The only active milestone is **"Phase 1 Release"**. Everything below is P0 (launch-blocking, former Phase 0), P1 (Phase 1 must-have), or P2 (Phase 1 nice-to-have). P0 must ship first; P1 in the same release; P2 if time allows or as fast-follow within Phase 1.

## Objective

Ship a **production-ready, mobile-first** Chorus that delivers: automatic translation (<500 ms p95 on cache), CEFR grammar feedback, vocabulary + learning-path tracking, and a horizontally scalable stateless backend — with basic NFRs, security, and CI/CD quality gates. All features are end-to-end functional, no stubs.

## 0. Consolidation — What Changed

- GitHub milestone **"Phase 0 Release"** → **close** after moving its 10 issues (#40–#49) to **"Phase 1 Release"**. Relabel `phase-0` → `phase-1` + `priority-high` (P0).
- `REQUIREMENTS.md` §0 is now archived; `§4/§5` carry the merged functional + NFR requirements (FR-25…FR-34, NFR-22/NFR-25/NFR-26).
- `PHASE_2_3_IMPLEMENTATION.md` stays as Phase 2/3 tracking; nothing from it moves forward except the teacher-vetting process note (doc-only in Phase 1).

## 1. Phasing Within Phase 1

| Band | Meaning | Items | Owner hint |
|------|---------|-------|------------|
| **P0 — Launch blocking** | Former Phase 0 bugs + onboarding + mobile parity. **Must pass QA before any public release.** | Home link (#40), Admin back (#41), Premium copy (#42), Emoji picker (#43), Contacts & Invites epic (#44), First/last name (#45), Initials avatar (#46), Generic error handling (#47), Mobile build working (7, 58) | Kushagra1122 (mobile), gosangiraju (bugs) |
| **P1 — Phase 1 must-have** | Translation hardening, grammar, vocab, learning path, controls, highlighting, quality eval infra, NFRs, security, dev env. | §2–§5 below (13 new issues + existing #1, #2, #3, #6, #17, #19, #20, #22, #24, #25, #26, #27, #34) | batchu, gayatri, Raju (infra) |
| **P2 — Nice-to-have in Phase 1** | WhatsApp OTP (#50), stickers (#52), deep polish; may slip to 1.5. | #50, #52, AI tutor (#51) partial | gosangiraju |

## 2. P0 — Close the former Phase 0 gap (1–2 weeks)

### 2.1 Bugs & UX quick fixes
- **#40 Home button link from dashboard** — fix router; add regression test.
- **#41 Admin → dashboard back affordance** — add back button; verify on mobile + web.
- **#42 Premium plans copy 280 vs 28** — **first confirm** `translationWordLimit` / plan word cap from `entitlements` service; then fix copy *or* cap to match. Do not change copy blindly (open item).
- **#43 Emoji picker in message input** — emoji list + insert + translation passthrough test. Already partially done; finish + e2e.
- **#47 Generic error handling** — replace "Sorry, something went wrong" with typed errors (validation, auth, rate-limit, translation failure) + toast + retry; cover with unit tests.

### 2.2 Growth & identity essentials
- **#45 Onboarding: first + last name** — form fields, composed `displayName`, editable in Profile.
- **#46 Profile avatar (initials/color)** — deterministic color; placeholder until upload infra.
- **#44 Epic: Contacts & Invites** — permission-gated contact scan, on-platform detection (hashed), invite via SMS/WhatsApp/email with single-use token + status tracking. Biggest P0; break into 3 PRs.

### 2.3 Mobile parity
- **#7 + #58 Mobile strategy (Expo)** — Expo-managed RN is decided. P0 exit criteria: `mobile/` builds on Android (EAS) + runs on device/emulator, WebSocket + translation flow parity with web, CI check passes. Tie to `mobile/MOBILE_TEST_PLAN.md`.

> **P0 Exit gate:** All P0 issues pass functional + e2e on `dev` and `prod` parity; release-gate doc #36 + Go/No-Go #37 are satisfied for P0 subset. See §6 for gates.

## 3. P1 — Working Translation & Grammar (core loop)

### 3.1 Translation hardening (NFR-1, NFR-11)
Current backend (`backend/internal/services/translation.go`, `pkg/translation`) already: detection cache (24h), translation cache (v2, 24h), ChainProvider fallback (OpenRouter → DeepSeek), `translation_jobs` durability. Still needed for Phase 1:

1. **Feature toggles (FR-25)** — new `user_settings` columns `translation_enabled bool`, `grammar_auto bool`, `highlights_enabled bool`. Gate `translation_jobs` enqueue and `TranslateQuick` on the flag. Settings UI + API + per-chat override (future). Issue: **`Phase 1: Feature controls — translation/grammar/highlights toggles`**.
2. **Learned-word optimization (FR-26)** — expose `vocabulary` learned set to `TranslationService`. New helper `FilterKnownWords(text, userID)` + per-word translation cache `translation:word:{word}:{lang}`. Skip LLM calls for known words; when sentence contains mix, keep sentence translation but surface word-level cache hits. Prevents re-translating "hola"/"hi" for advanced users. Issue: **`Phase 1: Word-bank–aware translation (skip known words)`**.
3. **Chat Language Settings simplification (FR-35)** — `frontend/src/components/ChatLanguageModal.tsx:14-77` currently has `myLanguage` + `theirLanguage`. Remove the second dropdown and its preview. Keep only `myLanguage` (bound to `user.nativeLanguage` / target). One PR. Issue: **`Phase 1: Simplify Chat Language Settings — own language only`**.

### 3.2 Grammar & Vocabulary
- Existing: `GrammarService` (`backend/internal/services/grammar.go:142`), regex + AI analysis, `CachedAIAnalysis`, `GenerateAIAnalysis`, `vocabulary.go` spaced repetition SM-2, `GrammarPanel` + `MessageBubble`.
- Still needed to call "working":
  - **Word-bank add + highlight** — FR-27/FR-28: highlight new words in `MessageBubble` (compare vs. vocabulary known set), tap → save + animate, practice CTA to `PracticeScreen`. Issue: **`Phase 1: Highlight new words in translations + quick practice`**.
  - **Your Learning Path dashboard** — FR-31: wire `Learn.tsx` (currently mock 1.2k/342) to real `vocabulary.GetLearningProgress` + new aggregates `sentences_understood`, time-bucketed `words_learned_by_month`. Issue: **`Phase 1: Your Learning Path — real metrics (words/sentences per month)`**.
  - **Writing assistant** — FR-33: composer extension "Help me write" → AI gap-fill/feedback without auto-sending. Issue: **`Phase 1: AI writing assistant (draft in target language)`**.
  - **Scenario role-play** — FR-34: Learn tab + Chat entry that starts an AI scenario ("restaurant", "date") with role-play prompt, turn-taking, corrections. Issue: **`Phase 1: AI scenario role-play`**.

### 3.3 Seed / Personal / Unified queue
Already spec'd as #17, #19, #20. Consolidate acceptance: seed sequences per `(language_pair, level)` cached; personal items mined from messages+GrammarService; unified queue endpoint interleaves with SM-2.

## 4. P1 — Evaluation, Metrics & Efficiency

### 4.1 Persist & evaluate (FR-30, NFR-25)
Today `messages.translations` + `grammar_jobs.result` are stored but not systematically evaluated.

- **Store all translations/grammar with lineage** — ensure `translation_jobs` + new `translation_evals` / `grammar_evals` tables capture `(source, target, provider, latency, tokens, cache_hit, result)`; add `prompt_version` column.
- **Cross-model critique** — after a translation/grammar write, enqueue an **evaluator job** that calls a *different* model (e.g., if primary was OpenRouter, evaluator is DeepSeek or local) to score accuracy/fluency/CEFR and produce a critique + `accuracy_score`. Nightly batch also re-scores a sample.
- **Arize Phoenix locally** — deploy `arizephoenix/phoenix` as a compose service (`phoenix:`), send traces from `TranslationService` + `GrammarService` via OTLP. Use for **offline evaluation** (golden set) + **realtime tracing**. Expose KPIs: accuracy, p95 latency, cost/1k tokens, cache hit rate. Dashboard links in admin console. Issue: **`Phase 1: Translation/grammar quality pipeline + Arize Phoenix (offline + realtime)`** (covers FR-30 + NFR-25; label `ai / llm`, `infrastructure`).

### 4.2 Prompt & cost refinement loop
Monthly (or on KPI regression): review Phoenix evals → refine prompts/model tier → bump `cacheVersion` (`translation.go:106` / `grammar.go:692`) to invalidate cache → measure lift.

## 5. P1 — NFRs, Security, Infra

### 5.1 Mobile as primary surface
- **NFR-22** — Capacitor → Expo transition is in progress (#58). Remaining: EAS build workflows (`.github/workflows/eas.yml`), parity testing, push notifications (FCM) stub, App Store metadata. Keep `frontend/` tracking mobile UX but not diverging. Owner: **Kushagra1122**.

### 5.2 Stateless scaling & resilience (already largely built)
- #1 L4 LB + multi-server + Redis registry (`ws:registry:{userId}`) — verify via #31 concurrency tests (1k WS / 50 msg/s).
- #34 durable per-recipient delivery via `message_deliveries` + startup replay — complete.
- #23 DeepSeek fallback + circuit breaker, #24 rate limiting (Redis token-bucket) + LLM spend tracking — complete.

### 5.3 Observability
- #25 structured logs/metrics/health — wire Prometheus/Grafana; add Phoenix as trace backend. Must expose `GET /health` + `/metrics` per server; LB health checks (# NFR-19).

### 5.4 Security — mail server isolation (NFR-22)
Current `docker-compose.prod.yml` / env runs Mailu on the same host as the app ("same space"). Harden:
1. Move Mailu to **separate host or isolated Docker network** with only `587` (submission) open from app servers; block `25` from public; enforce SPF/DKIM/DMARC per #3.
2. Rotate `SMTP_PASSWORD` (exposed in `.env` history), move to secret store (Dokploy secrets / env file not in repo), scope to app network only.
3. Add `Caddy`/`Traefik` TLS for mail submission if not already.
4. Document isolation in `ARCHITECTURE.md` + `DOKPLOY_DEPLOY.md`. Issue: **`Phase 1: Harden mail server — isolate from app prod (security)`**.

### 5.5 Dev environment + promotion gates (NFR-26)
Assign **Raju (gosangiraju)** — requested in notes.

- Create **`dev` environment**: `docker-compose.dev.yml` + `deploy/dev/` overlay, separate Dokploy project `chorus-dev`, seeded DB, `VITE_API_URL` pointing to dev LB.
- **CI/CD**: GitHub Actions `ci.yml` → build images → push → deploy to `dev`; run **quality gates** on dev before promotion:
  - unit + integration (`go test ./...`, `npm test`)
  - e2e Playwright (`e2e/`)
  - translation/grammar golden-set eval via Phoenix (accuracy ≥ threshold, p95 latency < 500 ms on cache-hit set)
  - load smoke (Artillery: 100 concurrent WS, 10 msg/s, 5 min)
  - security scan (govulncheck + npm audit)
- **Auto-promotion** only if gates pass; otherwise notify. Use image promotion (not rebuild) to prod. Branch protection on `main`. Issue: **`Phase 1: Dev environment + CI/CD quality gates (Raju) — auto-promote dev→prod`**.

### 5.6 Data, compliance, safety
- #26 data retention policy (define window, minimal enforcement).
- #27 report/block (already in schema `blocked_users`, `reports` — wire moderation-inbox email).
- #28 read receipts, #30 email notifications — keep as P1/P2 per current labels.

## 6. Release Gates & Timeline

### Gates (from #36 Product: test & release-gate strategy + #37 Go/No-Go)

| Gate | Checks | Who signs |
|------|--------|-----------|
| **P0 gate** | All P0 issues closed, e2e on dev + one prod smoke, mobile builds on EAS, no "Sorry..." generic error surfaces | batchu + Kushagra |
| **Phase 1 gate** | P1 issues closed, NFR-1/2/3 p95 met on staging, Phoenix eval accuracy ≥ baseline, LB failover drill, mail isolation verified, dev→prod promotion demo | batchu + Raju + Daniella (learning review) |
| **Go/No-Go** | Checklist in #37: translation, grammar, mobile, premium, observability, security, legal (privacy/retention), support runbook | All leads |

### Suggested timeline (adjust to capacity)

- **Week 1–2:** P0 (2.1–2.3). Close milestones for 10 moved issues.
- **Week 3–4:** §3.1 + §3.2 highlight/path (§4.1 Phoenix deploy).
- **Week 5–6:** §3.2 writing/scenario + §5.4 mail hardening + §5.5 dev env.
- **Week 7:** Hardening, load tests (#31), polish, release gates.

## 7. Redis Pub/Sub vs Kafka (unchanged)

- Redis Pub/Sub for transient real-time; Kafka deferred. For Phase 1, Redis is correct.

## 8. Backlog: What to Close / Move / Create

Full per-issue mapping is in `BACKLOG_REFINEMENT_2026-08-23.md`. Summary:

- **Move** (#40–#49) Phase 0 → Phase 1 P0.
- **Keep** existing Phase 1 issues (#1, #2, #3, #6, #7, #17–#26, etc.) — re-triage priorities where noted.
- **Create 14 new issues** from session notes (see §3–§5 + backlog doc). If the PAT lacked scope, create via the provided script (`scripts/create_phase1_issues.sh`) or manually.

## 9. Verify Plan

```bash
# After editing REQUIREMENTS.md + this file:
npm run lint --prefix frontend
go vet ./...  # in backend/
docker compose -f docker-compose.yml -f docker-compose.dev.yml config --quiet
```

## 10. Teacher Onboarding — Note for Phase 2 (doc-only in Phase 1)

Do not build marketplace infra in Phase 1. Record the vetted process: basic assessment → live recording (language demo) → certificate upload → manual video-call review by language expert (Daniella for Spanish; recruit one expert per additional language). Rubric is documented; future work trains an AI evaluator on top of it. Track under #53 epic.

---
*Last updated: 2026-08-23. Owner: batchu + leads. Next review: after P0 gate.*
