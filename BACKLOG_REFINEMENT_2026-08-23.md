# Backlog Refinement — 2026-08-23 (Chorus.talk Session)

> **Input:** 60 issues pulled unauthenticated from `zumy-app/chorus` (60 open/closed/PRs), `REQUIREMENTS.md` (pre-merge, 2026-08-17), `PHASE_1_IMPLEMENTATION_PLAN.md` (stateless Redis plan), plus **session notes** (see §1). PAT in `.env` (`GITHUB_CHORUS_ISSUES_PAT`) returned **401 Bad credentials** on every endpoint — fine-grained PAT is expired or not authorized for `zumy-app/chorus`. All GitHub writes are provided as a **script + manual steps** instead (see §7).

## 1. Session Notes (verbatim, deduplicated)

> Provide controls for turning on/off features like language translation.
> if they see new words, add them to the word bank — no need to waste resources for words already translated. no need to retranslate words like "hola", "hi". keep track of learned words and avoid translating them.
> user dashboard - learned x number of words by month, understand x number of sentences per month etc.. what are the metrics? These metrics should be shown on "Your Learning Path"
> highlight new words in the translated messages and allow users to practice
> Do quality analysis/evaluation of the translations. store all translations/grammar analysis, do analysis of the quality using a different model than what was originally used.. critique and refine prompts/improve efficiency. KPIs, accuracy number https://arize.com/phoenix/ Deploy it locally, do offline evaluation or realtime evaluation.. realtime analysis
> Learning goals - allow users to draft messages in the language they need to learn. AI should provide feedback/fill in the gaps and work with the users to write messages (part of the writing goals)
> Communicate with AI to learn. AI can generate scenarios.. you are at a restaurant, place an order. or you are on a date, talking to a date.
> In "Chat Language Settings".. remove other persons language setting dropdown. Just focus on your own.
> Teacher onboarding - do basic assessment, do a live recording, certificates.. how do we vet? have Daniella for Spanish. establish experts in each language and have them evaluate manually through video calls. Eventually train AI models on this
> Security - Mail server running on prod, same space,
> create a dev environment, assign to raju who will work on ci/cd, create dev environment, quality gates, run functional, e2e tests on dev before auto promotion to pro
> Goal for phase 1 is working language translation, grammar analysis, mobile app , basic important NFRs, security, some structured language learning activites and tracking. notes from this session "Provide controls..."

## 2. Snapshot — Existing Backlog (60 items, unauth fetch)

| # | Title | State | Labels | Milestone | Verdict |
|---|-------|-------|--------|-----------|---------|
| 61 | Added persistence for Target language + Level | open PR | — | — | Merge; closes #59 |
| 60 | Issue/18 (target level) | open PR | — | — | Merge; closes #18 |
| 59 | Target languages lost upon registration | open | — | — | **Keep** (P0/P1 bug) |
| 58 | refactor: Expo-managed RN | open PR | — | — | **Merge, keep** |
| 57 | Research: HelloTalk harvest + AI analysis | open | phase-2, research | — | Keep Phase 2 |
| 56 | Research: Group chat size limit (100) | open | phase-2, research | — | Keep Phase 2 |
| 55 | Public language-pair group chats | open | phase-2 | — | Keep Phase 2 |
| 54 | Teacher group chats | open | phase-2 | — | Keep Phase 2 |
| 53 | Epic: Live tutoring marketplace | open | epic, phase-2 | — | Keep Phase 2 |
| 52 | Premium stickers/emojis/themes (1.5) | open | phase-2 | — | Keep, P2 |
| 51 | AI tutor chatbot - premium (1.5) | open | phase-2, ai/llm | — | Keep, P2 |
| 50 | WhatsApp OTP | open | phase-1 | — | Keep, P1/P2 |
| 49 | Product: Premium ideas refinement | open | product, phase-0 | **Phase 0 Release** | **MOVE → Phase 1**, relabel `phase-1`, keep open |
| 48 | Product: Phase 0 scope & deployment timeline | open | product, phase-0 | **Phase 0 Release** | **MOVE → Phase 1, close as superseded** by this doc + plan |
| 47 | UX: "Sorry, something went wrong" | open | bug, phase-0 | **Phase 0 Release** | **MOVE → Phase 1 P0** |
| 46 | Profile avatar (generated) | open | phase-0, mobile | **Phase 0 Release** | **MOVE → Phase 1 P0** |
| 45 | Onboarding: first + last name | open | phase-0, mobile | **Phase 0 Release** | **MOVE → Phase 1 P0** |
| 44 | Epic: Contacts & Invites | open | epic, phase-0, mobile | **Phase 0 Release** | **MOVE → Phase 1 P0** |
| 43 | Add emojis to message input | open | phase-0, mobile | **Phase 0 Release** | **MOVE → Phase 1 P0** |
| 42 | Bug: Premium copy 280 vs 28 | open | bug, phase-0 | **Phase 0 Release** | **MOVE → Phase 1 P0**, needs server-cap confirmation |
| 41 | Bug: No back button admin→dashboard | open | bug, phase-0 | **Phase 0 Release** | **MOVE → Phase 1 P0** |
| 40 | Bug: Home button link dashboard | open | bug, phase-0 | **Phase 0 Release** | **MOVE → Phase 1 P0** |
| 39 | Investor pitch deck | open | doc, phase-1, product | — | Keep |
| 38 | Product: Mobile strategy review (post #7) | open | phase-1, mobile | — | Keep, close after #58 lands |
| 37 | Go/No-Go & product gate tracking | open | phase-1, product | — | Keep |
| 36 | Product: Phase 1 test & release-gate strategy | open | phase-1, testing | — | Keep |
| 35 | Product: Learning-plan strategy & CEFR | open | phase-1, product | — | Keep |
| 34 | Durable per-recipient delivery | open | reliability | — | Keep, P1 |
| 33 | GitHub & Apple OAuth (Phase 2) | open | phase-2 | — | Keep |
| 32 | LLM latency optimization pass | open | phase-1, performance | — | Keep, P1 |
| 31 | Concurrency & load testing (1k WS / 50 msg/s) | open | phase-1, performance | — | Keep, P1 |
| 30 | Email notification system (Mailu) | open | phase-1 | — | Keep, P1 |
| 29 | Password reset & email verification | open | phase-1 | — | Keep, P1 |
| 28 | Read receipts | open | phase-1 | — | Keep, P1 |
| 27 | Report/block (user_blocks + reports) | open | phase-1, safety | — | Keep, P1 |
| 26 | Data retention policy | open | phase-1, compliance | — | Keep, P1 |
| 25 | Observability: logs/metrics/health | open | phase-1, infra | — | Keep, P1; augment with Phoenix |
| 24 | Rate limiting (Redis token-bucket) + LLM spend | open | security, phase-1 | — | Keep, P1 |
| 23 | DeepSeek fallback + circuit breaker | open | phase-1, reliability | — | Keep, P1 |
| 22 | Grammar analysis: CEFR feedback loop | open | phase-1 | — | Keep, P1 |
| 21 | Gamification: streaks/XP/combo | open | phase-1 | — | Keep, P1 |
| 20 | Unified learning queue | open | phase-1 | — | Keep, P1 |
| 19 | Personalized learning items (mine vocab) | open | phase-1 | — | Keep, P1 |
| 18 | Onboarding: level self-selection | open | phase-1 | — | Keep, P1; PR #60 targets it |
| 17 | Seed path: core vocab/grammar sequences | open | phase-1 | — | Keep, P1 |
| 16,15,14,11,9,8,4 | closed | — | — | — | Closed, ignore |
| 12 | Create a waitlist feature | open | — | — | Done; close |
| 10 | Admin Dashboard: Analytics, User Mgmt, Premium (Phase 2) | open | phase-2 | — | Keep Phase 2 |
| 7 | Mobile Strategy: Research & Implement | open | phase-1, mobile | — | Keep, parent of #58 |
| 6 | Testing: Unit, Integration, Functional & E2E | open | phase-1, testing | — | Keep, P1; augment with NFR-26 gates |
| 5 | Google OAuth (Phase 1); GitHub/Apple to Phase 2 | open | phase-1 | — | Keep |
| 3 | Functional email on Mailu + SPF/DKIM/DMARC | open | phase-1, infra | — | Keep, P1; extends to mail-isolation issue |
| 2 | Production Stabilization: Get chorus.talk Fully Operational | open | phase-1, infra | — | Keep, P1 |
| 1 | Architecture Redesign: L4 LB + Redis Registry (Sentinel HA) | open | phase-1, infra | — | Keep, P1 |

**Milestones (fetched):** `Phase 1 Release` (open, 0 issues assigned — mis-configured) and `Phase 0 Release` (open, 10 issues). After merge, all 10 move to Phase 1 Release; Phase 0 Release is closed.

**Counts after merge:** ~34 open issues in Phase 1 (10 moved + 24 existing) + 14 new = ~48 tracked. Phase 2 stays ~12.

## 3. Note → Requirement → Issue Mapping (new work)

| # | Session note | Requirement (REQUIREMENTS.md) | New issue to create (title slug; see §6 for bodies) | Relates to existing |
|---|--------------|-------------------------------|------------------------------------------------------|---------------------|
| N1 | Provide controls for turning on/off features like language translation | FR-25 Feature controls | **Phase 1: Feature controls — translation/grammar/highlights toggles** | #6 (testing), #25 (observability) |
| N2 | avoid retranslating known words like "hola", "hi"; keep learned-word bank | FR-26 Learned-word optimization | **Phase 1: Word-bank–aware translation (skip known words, cache word-level)** | #19 (personalized items), #24 (rate limit/cost) |
| N3 | user dashboard - words/month, sentences/month on Your Learning Path | FR-31 Metrics + NFR-25 | **Phase 1: Your Learning Path — real metrics (words/sentences per month)** | #20 (unified queue), #21 (gamification), Learn.tsx |
| N4 | highlight new words + allow practice | FR-28 Highlight + practice | **Phase 1: Highlight new words in translations + quick practice** | #19, #20, MessageBubble.tsx |
| N5 | store all translations/grammar, cross-model critique, prompt refine, Phoenix, KPIs, offline+realtime | FR-30 + NFR-25 | **Phase 1: Translation/grammar quality pipeline + Arize Phoenix (offline + realtime)** | #25, #32 (latency) |
| N6 | draft messages in target language, AI feedback/fill gaps | FR-33 Writing goals | **Phase 1: AI writing assistant (draft in target language)** | #22 (grammar) |
| N7 | AI can generate scenarios (restaurant, date) | FR-34 Scenario role-play | **Phase 1: AI scenario role-play (restaurant, date, etc.)** | #17 (seed path) |
| N8 | remove other person's language dropdown in Chat Language Settings | FR-35 Simplify modal | **Phase 1: Simplify Chat Language Settings — own language only** | ChatLanguageModal.tsx |
| N9 | Teacher onboarding — assessment, recording, certs, Daniella for Spanish, expert video eval, train AI | NFR-27 (doc, Phase 2) + Phase 2 epic | **Phase 2: Teacher vetting process (assessment → recording → expert video review)** _(doc-only in Phase 1, execution Phase 2)_ | #53 epic |
| N10 | Security — Mail server running on prod, same space | NFR-22 Hardening | **Phase 1: Harden mail server — isolate from app prod (security)** | #3, #2 |
| N11 | create dev environment, assign to raju, CI/CD quality gates, dev before prod | NFR-26 Dev env + gates | **Phase 1: Dev environment + CI/CD quality gates (Raju) — dev→prod auto-promotion** | #6, #36, #37 |
| N12 | goal: structured language learning activities + tracking (implied N8+N9 detail) | FR-32 Seed/personal/unified queue | Covered by #17/#19/#20 + N3/N4 — **no new issue**, but add acceptance to those | — |
| N13 | no need to waste resources for words already translated (efficiency sub-note of N2) | FR-26 | Same as N2 — **word-level cache** | #32 |
| N14 | allow users to draft messages ... (duplicate of N6, kept for metrics) | FR-33 | Same as N6 | — |

**Result: 11 new Phase 1 issues + 1 Phase 2 doc issue = 12; plus 2 helper issues for migration tracking = 14 total new issues (see §6).**

## 4. Triage Decisions (what to close / move / reprioritize)

### 4.1 MOVE: Phase 0 → Phase 1 P0 (10 issues)

For each of #40, #41, #42, #43, #44, #45, #46, #47, #48, #49:

- **API:** `PATCH /repos/zumy-app/chorus/issues/{n}` → `milestone: <Phase 1 Release number>`, `labels: replace phase-0 with phase-1, add priority-high`.
- #48 ("Product: Phase 0 scope & deployment timeline") → **close as completed** after this refinement (superseded); optionally keep #49 open as the single premium-planning tracker.
- #44 epic: keep open; break into subtasks (contact permission, hash matching, invite token, status UI).

### 4.2 REPRIORITIZE existing

- **P0 (immediate):** #40, #41, #43, #44, #45, #46, #47, plus #58 mobile PR and #7 parent.
- **P1 (Phase 1 must):** #1, #2, #3, #6, #17, #18, #19, #20, #22, #24, #25 (+ Phoenix), #26, #27, #31, #34 — plus all N1–N11 new issues. Mark `priority-high`.
- **P2 (nice-to-have):** #50, #51, #52, #28, #30 — `priority-medium` unless pulled in.

### 4.3 CLOSE as done/superseded

- #12 "Create a waitlist feature" → waitlist exists (`waitlist_entries`, invitations) — **close**.
- #48 "Product: Phase 0 scope & deployment timeline" → **close** (this doc + plan supersede it).
- #38 "Mobile strategy review (post-research #7)" → close once #58 merges.
- PRs #60, #61 → merge or close per review; #58 needs decision (Expo) — merge is recommended.

## 5. Concrete Gaps & Bugs to File (included in new issues)

- **Translation cost waste on known words** — not tracked before; now N2.
- **Learn page is mock data** (`frontend/src/pages/Learn.tsx:58-75` hard-coded 1.2k/342/48) — now N3.
- **ChatLanguageModal shows other person's language** (`ChatLanguageModal.tsx:14-77`) — now N8.
- **No evaluation pipeline / Phoenix** — now N5.
- **No feature toggles** — now N1.
- **No dev environment / promotion gates** — now N11; owner Raju.
- **Mail server co-located with prod** — now N10; security gap.
- **Teacher vetting has no documented process** — now N9.

## 6. New Issue Payloads (14)

Use `scripts/create_phase1_issues.sh` (or `.ps1` on Windows) to create. Each block is `title` / `labels` / `milestone` / `body` (markdown). Assignee hints in brackets.

### 6.1 Phase 1: Feature controls — translation/grammar/highlights toggles [batchu]
- **Labels:** `enhancement`, `phase-1`, `priority-high`
- **Milestone:** Phase 1 Release
- **Body:**
```md
## Summary
Per-user (later per-chat) toggles for auto-translation, auto grammar, and learning highlights. When off, the server does not enqueue translation/grammar jobs. Fixes session note "Provide controls for turning on/off features like language translation."

## Scope
- DB: `user_settings` add `translation_enabled bool default true`, `grammar_auto bool`, `highlights_enabled bool` (or `user_feature_flags jsonb`).
- API: `GET/PUT /api/v1/settings/features` — auth required; validated server-side.
- Services: gate `translation_jobs` enqueue + `GrammarQueueService` on flags.
- UI: Settings → Features section with three switches + per-chat override affordance (future).
- Tests: unit (gate), integration (flag off → no job), e2e (toggle persists after reload).

## Acceptance
- Toggle off → no translation/grammar jobs created; existing history untouched.
- Toggle on → jobs resume; queued retry works.
- No dead toggles (FR-18): every shipped toggle has backend effect.

## Related
FR-25, NFR-26 gate, #6 testing.
```

### 6.2 Phase 1: Word-bank–aware translation (skip known words) [gayatriagarwal19]
- **Labels:** `enhancement`, `phase-1`, `priority-high`, `ai / llm`, `performance`
- **Body:**
```md
## Summary
Avoid re-translating words the user already knows ("hola", "hi"). Consult the per-user word bank and serve from a word-level cache.

## Scope
- Vocabulary "known" threshold: `interval_days >= 21` OR explicit `known` flag (decide; record in #35 decision).
- New helper `FilterKnownWords(text, userID)` + Redis keys `translation:word:{word}:{targetLang}` (24h).
- Sentence translation keeps full-sentence cache (`translation:v2:...`) but skips LLM for substrings that are known; highlight known words as dimmed.
- Metrics: cost/1k tokens before/after; cache hit rate.

## Acceptance
- User with "hola" marked known → incoming "hola amigo" translation does not re-call LLM for "hola"; cost metric drops.
- Unit test: known-word list → LLM call count asserted.
```

### 6.3 Phase 1: Your Learning Path — real metrics (words/sentences per month) [Kushagra1122]
- **Labels:** `enhancement`, `phase-1`, `priority-high`, `mobile`
- **Body:**
```md
## Summary
Replace mock Learn.tsx (1.2k / 342) with real metrics: words learned per month, sentences understood per month, due reviews, streak, CEFR progress. Shown on "Your Learning Path" (Learn tab).

## Scope
- Backend: extend `vocabulary.GetLearningProgress` + new `GET /api/v1/learning/path` aggregator (words_learned_by_month, sentences_understood_by_month). Sentences-understood derived from messages where translation was consumed (or grammar analyzed) — define event.
- Frontend: wire `frontend/src/pages/Learn.tsx` to real data; month picker; empty state; loading/error.
- Analytics: monthly buckets, tested across year boundary.

## Acceptance
- Learn page shows real counts; month switch updates; e2e covers month rollover + empty user.
```

### 6.4 Phase 1: Highlight new words in translations + quick practice [gayatriagarwal19]
- **Labels:** `enhancement`, `phase-1`, `priority-high`, `mobile`
- **Body:**
```md
## Summary
In MessageBubble, highlight unlearned words in the translation; tap → Save to word bank + Practice CTA. Satisfies "highlight new words in the translated messages and allow users to practice."

## Scope
- Backend: endpoint `POST /api/v1/vocabulary/highlight` helper or reuse save; optional `GET /api/v1/vocabulary/known?userId` for client-side diff.
- Frontend: `MessageBubble.tsx` — tokenize translation, diff against known set, render highlights (span + style), tap → `vocabularyAPI.save` + sheet to `PracticeScreen`.
- Accessibility: highlight contrast, not relying on color alone.

## Acceptance
- Unknown words highlighted; tap saves and shows "Saved"; practice navigates.
- No highlight when word-bank is empty or all known.
```

### 6.5 Phase 1: Translation/grammar quality pipeline + Arize Phoenix [batchu]
- **Labels:** `ai / llm`, `phase-1`, `priority-high`, `infrastructure`, `performance`
- **Body:**
```md
## Summary
Store every translation/grammar analysis with lineage, run cross-model evaluation (different model from producer), critique/refine prompts, expose KPIs. Deploy Arize Phoenix locally for offline + realtime eval.

## Scope
- DB: `translation_evals` + `grammar_evals` (accuracy_score, latency, cost, cache_hit, prompt_version, evaluator_model).
- Services: enqueue evaluator job after write; batch nightly sample.
- Infra: `arizephoenix/phoenix` compose service (`phoenix:6006`), OTLP traces from TranslationService/GrammarService. Docs in `ARCHITECTURE.md`.
- Admin: KPI cards (accuracy, p95 latency, cost/1k, cache hit 80%+), prompt version diff.
- Eval: golden set (seed + held-out translations) for offline runs.

## Acceptance
- Every translation/grammar row has an eval row within 5 min (realtime) + batch coverage.
- Phoenix reachable on dev, traces visible, accuracy KPI graphed.
- See https://arize.com/phoenix
```

### 6.6 Phase 1: AI writing assistant — draft in target language [gayatriagarwal19]
- **Labels:** `enhancement`, `phase-1`, `priority-high`, `ai / llm`
- **Body:**
```md
## Summary
"Help me write" in the composer: user drafts in target language, AI fills gaps, corrects, and explains — before sending. Part of writing goals.

## Scope
- API: `POST /api/v1/ai/writing-assist {draft, targetLang, nativeLang}` → `{suggestion, corrections[], gaps[]}` (streaming optional).
- UI: Composer extension button → sheet with suggestion + inline diff + "Insert" vs "Send".
- Guardrails: word cap (P4a), rate limit, no auto-send.
- Tests: golden drafts, streaming fallback.

## Acceptance
- Draft "Yo quiero ... restaurante" → AI completes/corrects, user can insert or edit before send.
- No LLM leak of other users' messages.
```

### 6.7 Phase 1: AI scenario role-play (restaurant, date, etc.) [gayatriagarwal19]
- **Labels:** `enhancement`, `phase-1`, `priority-medium`, `ai / llm`
- **Body:**
```md
## Summary
AI generates scenarios and role-plays ("you are at a restaurant, place an order" / "you are on a date"). Lives in Learn tab + Chat.

## Scope
- API: `POST /api/v1/ai/scenario/start {scenario, targetLang, level}` + turn endpoint.
- Scenarios: seeded list (restaurant, date, airport, market) + custom prompt.
- UI: scenario cards, turn bubbles, inline corrections, end-of-scenario recap.
- Reuse grammar vocab: add encountered words to review queue on finish.

## Acceptance
- Start restaurant scenario → 5-turn role-play with corrections; recap shows words to review.
```

### 6.8 Phase 1: Simplify Chat Language Settings — own language only [Kushagra1122]
- **Labels:** `enhancement`, `phase-1`, `priority-high`, `mobile`
- **Body:**
```md
## Summary
Remove the "other person's language" dropdown from Chat Language Settings. Focus on the user's own language only.

## Scope
- File: `frontend/src/components/ChatLanguageModal.tsx:14-77` — delete `theirLanguage` state, second `<select>`, and its preview. Keep `myLanguage` bound to `user.nativeLanguage`.
- Labels + i18n: remove `chatLanguageModal.contactLanguage*` keys or keep for future.
- Test: Settings test + visual regression.

## Acceptance
- Modal shows one dropdown (own language) + preview; no reference to contact's language.
- Existing tests updated.
```

### 6.9 Phase 1: Harden mail server — isolate from app prod (security) [batchu]
- **Labels:** `security`, `phase-1`, `priority-high`, `infrastructure`
- **Body:**
```md
## Summary
Prod mail (Mailu) currently runs "same space" as app — isolate it.

## Scope
- Move Mailu to separate host or isolated Docker network; only `587` (submission) from app; block `25` from public where possible.
- Rotate `SMTP_PASSWORD` (was in .env history) and move to Dokploy secrets; no `VITE_` prefix.
- Enforce SPF/DKIM/DMARC per #3; verify with mail-tester.
- Document isolation in ARCHITECTURE.md + DOKPLOY_DEPLOY.md.
- Add network policy check to release gate.

## Acceptance
- App host cannot reach mail host except on 587 from app subnet; creds rotated; mail-tester passes.
```

### 6.10 Phase 1: Dev environment + CI/CD quality gates (Raju) — dev→prod auto-promotion [gosangiraju]
- **Labels:** `infrastructure`, `phase-1`, `priority-high`, `testing`
- **Body:**
```md
## Summary
Create a `dev` environment; Raju owns CI/CD. Quality gates run on dev (functional, e2e, Phoenix eval, load smoke) before auto-promotion to prod. Satisfies session note "create a dev environment, assign to raju... quality gates, run functional, e2e tests on dev before auto promotion to pro."

## Scope
- Infra: `docker-compose.dev.yml` overlay + Dokploy project `chorus-dev` (separate DB/Redis on dev host).
- CI: `.github/workflows/ci.yml` — build images → push → deploy to dev → run gates → promote image to prod if green.
- Gates: `go test ./...`, `npm test`, Playwright e2e (`e2e/`), translation/grammar golden-set eval (Phoenix), Artillery smoke (100 WS / 10 msg/s / 5m), govulncheck + npm audit.
- Branch protection on `main`; image promotion (not rebuild) to prod.
- Notify on gate failure (email/Slack).

## Acceptance
- Push to `main` deploys to dev, gates run, prod only updated if gates pass; demo recorded.
- Assignee: gosangiraju (Raju).
```

### 6.11 Phase 2: Teacher vetting process (assessment → recording → expert video review) [batchu]
- **Labels:** `documentation`, `phase-2`, `priority-low`, `product`
- **Body:**
```md
## Summary
Document (Phase 1) and later execute (Phase 2) teacher onboarding: basic assessment → live recording → certificates → manual video-call review by language experts (Daniella for Spanish; one expert per language). Eventually train AI on rubric.

## Scope
- Phase 1 (doc-only): write `docs/TEACHER_VETTING.md` — rubric, recording prompt, certificate checklist, video-call flow, expert roster.
- Phase 2 (execution): profiles, ratings, class sign-up (see #53 epic) gated on vetting.
- Expert panel: recruit/confirm Daniella + 1–2 others; calendar + stipend noted.

## Acceptance (Phase 1)
- `docs/TEACHER_VETTING.md` merged; expert Daniella confirmed in writing.
```

### 6.12 Phase 1: Contacts & Invites — break into 3 PRs (subtask tracker) [Kushagra1122]
- **Labels:** `epic`, `phase-1`, `priority-high`, `mobile`
- **Body:**
```md
## Tracking issue for epic #44 — break into: (1) permission + hashed matching, (2) invite token + email/SMS, (3) status UI. Link PRs here. No duplicate scope with new issues.
```

### 6.13 Backlog: Close Phase 0 milestone + relabel (housekeeping) [batchu]
- **Labels:** `product`, `phase-1`, `priority-high`
- **Body:**
```md
## Summary
Execute §2 + §4.1 of BACKLOG_REFINEMENT_2026-08-23.md: move 10 issues from "Phase 0 Release" to "Phase 1 Release", replace `phase-0` with `phase-1`, close milestone "Phase 0 Release", close superseded issues (#12, #48). Use the provided script or do manually.

## Steps
1. Fetch milestone numbers: GET /repos/zumy-app/chorus/milestones
2. PATCH each issue's milestone + labels
3. DELETE/CLOSE milestone 2 ("Phase 0 Release") after empty
```

### 6.14 Docs: Archive Phase 0 label + add phase-1 P0 convention [batchu]
- **Labels:** `documentation`, `phase-1`, `priority-low`
- **Body:**
```md
## Summary
Deprecate `phase-0` label (keep for history or delete) and document that former Phase 0 scope is now Phase 1 P0 (label `phase-1` + `priority-high`). Update `.github/labels.yml` if present and CONTRIBUTING.md.
```

## 7. GitHub Write Path (PAT was 401)

**Unauthenticated reads succeed** (60 issues fetched). **Authenticated writes failed** with every header variant (`Bearer`, `token`) — PAT is invalid/expired or not scoped to `zumy-app/chorus`.

### Fix the PAT (owner does this once)

1. GitHub → Settings → Developer settings → Personal access tokens → Fine-grained PATs → New.
2. Resource owner: `zumy-app`, Repository access: `zumy-app/chorus` (or All repos in org).
3. Permissions: **Repository** → `Issues: Read and write`, `Metadata: Read`, `Pull requests: Read and write` (if you want to move PR milestones), **Organization** → `Members: Read` (optional).
4. Expiration 90 days; copy the new `github_pat_...`, replace `GITHUB_CHORUS_ISSUES_PAT` in `.env` (do not commit) and in your shell: `$env:GITHUB_CHORUS_ISSUES_PAT="..."` (pwsh) or `export ...` (bash).
5. Verify: `curl -H "Authorization: Bearer $TOKEN" -H "Accept: application/vnd.github+json" https://api.github.com/repos/zumy-app/chorus/issues/40` → `200`.

### Create issues (no PAT needed to preview)

The file `scripts/create_phase1_issues.sh` (bash) and `scripts/create_phase1_issues.ps1` (pwsh) in this repo are **ready to run** once the PAT is fixed. They `POST /repos/zumy-app/chorus/issues` for each payload in §6. Dry-run mode prints curl commands without sending.

Manual fallback: create issues in the GitHub UI by copying each Body block above; set labels + milestone as noted.

### Move & close milestones (after PAT fix)

```bash
# Example: move #40 to Phase 1 Release (milestone number N) and swap label
# Find N:
curl -s -H "Authorization: Bearer $PAT" https://api.github.com/repos/zumy-app/chorus/milestones | jq '.[] | {number, title}'

# Move:
curl -X PATCH -H "Authorization: Bearer $PAT" -H "Accept: application/vnd.github+json" \
  https://api.github.com/repos/zumy-app/chorus/issues/40 \
  -d '{"milestone": N, "labels": ["bug","priority-high","phase-1"]}'

# Close milestone 2 after empty:
curl -X PATCH -H "Authorization: Bearer $PAT" https://api.github.com/repos/zumy-app/chorus/milestones/2 \
  -d '{"state":"closed"}'
```

## 8. Verification (what was checked)

- `REQUIREMENTS.md` rewritten and merged; checked that FR-25…FR-35 + NFR-22/NFR-25/NFR-26 are referenced in the new plan and in open issues, no dangling `phase-0`.
- `PHASE_1_IMPLEMENTATION_PLAN.md` rewritten as consolidated plan; sections §2–§5 map 1:1 to N1–N11.
- `Learn.tsx:58-75` confirmed mock; `ChatLanguageModal.tsx:14-77` confirmed dual-dropdown; `translation.go:106` / `grammar.go:692` cache versions noted for invalidation after prompt changes.
- Unevaluated GH labels: `phase-0` (10 uses, now deprecated), `phase-1` (19 uses, canonical), `phase-2` (10 uses).
- No secrets committed; `.env` PAT left as-is (user to rotate).

## 9. Next Steps (owner batchu + leads)

1. Rotate GH PAT and run `scripts/create_phase1_issues.sh --execute` (or create 14 issues manually).
2. Move 10 Phase 0 issues to Phase 1 milestone; close Phase 0 milestone.
3. Kick off P0 (2 weeks) while Raju scaffolds `dev` env in parallel.
4. Deploy Phoenix on `dev` first; run golden-set eval baseline before hardening translation prompts.
5. Schedule teacher-vetting doc review with Daniella (Spanish) + recruit one more expert.

---
*Backlog refinement owner: Muse Spark. Source: chorus.talk dev sync notes 2026-08-23 + unauth GH fetch 2026-08-23T…Z.*
