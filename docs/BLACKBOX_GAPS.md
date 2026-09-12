# Blackbox Coverage Gaps — Chorus (dev/test only)

> Source of truth for "every core feature must be blackbox-tested". Each row
> links the GitHub issue that closes it. Policy: **hard asserts by default** —
> a `try/catch → console.warn` in a spec needs a linked issue justifying why
> (e.g. model-dependent), otherwise it is a bug in the test.

## 0. How we got here (2026-09 rollout)

A user-visible crash — mobile Profile quick-switch:
`Switch failed: Cannot read properties of undefined (reading 'data')` —
shipped green through `tsc`, eslint, 99 mobile jest tests, 30 Playwright
specs and the acceptance suite. Root cause was `(apiService as any).api`
on an object with no `.api` member (deterministic, 100% repro on device).
Every layer missed it for a structural reason (see §3), so the rollout below
fixed the bug **and** the structures that hid it.

## 1. Bugs found by the rollout (fixed + proven)

| # | Bug | Root cause | Fix | Proof |
|---|-----|-----------|-----|-------|
| 1 | Mobile quick-switch crash (`raw.data` on `undefined`) | `(apiService as any).api?.post` — default export has no `.api`; `as any` hid it from tsc; no `else` branch | Typed `apiService.switchUser()` + rewritten handlers | New jest suites fail 6/7 on pre-fix code, pass post-fix |
| 2 | API registration 100% broken (all 400s) | `first_name`/`last_name` NULL scanned into non-nullable Go `string` in both Register paths (Login already used NullString) | NullString intermediaries in `Register` + `RegisterWithInvitation` | TC-REG-02 goes 400 → 201 |
| 3 | `SeedDevData` not rerunnable (23503) once fixtures own chats | `chats.created_by` + second-order refs (`vocabulary` → chats/messages) lack `ON DELETE CASCADE`; seed assumed cascade | Catalog-driven ordered cleanup (`deleteFixtureDependents`) | Seed run 3× consecutively green |
| 4 | Contacts-invite endpoint 100% broken (500) | `$5` used as varchar column value AND inside `CASE` → Postgres 42P08 with untyped params | Compute `sent_at` in Go; removed SQL CASE | TC-APPLY-03/04, GDPR suites green |
| 5 | Same 42P08 class in scenario completion | `$5::text` dual-use on varchar `status` | Boolean flag param | Unit mock updated; path live |
| 6 | Session start 500 on unknown mode | No mode validation before INSERT hit the DB CHECK | `ErrInvalidSessionMode` → HTTP 400 | TC-LEARN-04 asserts 400 |
| 7 | Typing indicator never renders for the joiner | Sidebar chats carry no `participants`; `typingNames` needs `participant.user` | `setActiveChat` merges `GET /chats/:id` detail | 9.2 hard assert green |
| 8 | Acceptance runner was a no-op (`run.ts` ended at `SPLIT_MARKER_MAIN`, exit 0) | Main entrypoint never written; nothing in CI invoked it | Implemented `main()` (green/red/auto + ratchet) + `known-failures.txt` | 31/31 green, exit 0 |
| 9 | `sendMessage` helper false-passed on live preview | Assert ran before submit; `.break-words` preview (`ChatArea.tsx:632`) matched unsent text | Assert input-cleared FIRST, then bubble | Caught a real phantom-send during rollout |
| 10 | `createDirectChat` never proved the DM opened | No post-condition; later steps typed into wrong thread | Assert `h2` peer header after modal closes | Same |
| 11 | `loginAsUser` unusable on authed pages | `/login` bounces authed sessions to `/chat` (no form) → restores/switches timed out | Storage reset + reload when form absent | C-04-05/06 restores green |
| 12 | Stale test identities (Gmail everywhere, wrong field names, wrong id precedence) | Fixtures drifted from product (`avgRating` vs `ratingAvg`, app-id vs user-id, `{email:}` vs `{username:}`) | Aligned to canonical `alice.en-es`/`bob.es-en` + real contracts | Full suites green |
| 13 | `mobile/tests/*.ts` dead (403s, exit-0-always, Gmail creds, self-DM) | Invite gate + no exit codes + stale fixtures | Invite pipeline, peer resolution, env URLs, real exit codes | 8/8 + 17/17 green |
| 14 | Frontend Docker build broken two ways | (a) `vite@5`+`vitest@4` skew broke `npm ci`; (b) `./frontend` context hides `packages/shared` (`@chorus/shared` alias) | vitest→v3 (aligned), root-context Dockerfile + compose/CI contexts | `docker compose build frontend` green |
| 15 | Error-envelope objects crash React (blank unmount) | Screens stashed `err.response.data.error` (`{kind,message}` object) in state and rendered it — e.g. BecomeTeacher submit | `apiErrorMessage()` in `@chorus/shared` + render-path sweep (20+ sites, web+mobile) | C-05-02 asserts readable banner, never blank |
| 16 | `loginAsUser` session handling (see #11) | Same as #11 | Same as #11 | C-04-05/06 restores green |
| 17 | Stale translation-text selector (`.italic.font-medium`) | UI moved to `.font-translation-text`; old selector silently matched nothing (C-01-02 conditional skipped, 03-3.3 raced) | Live-DOM inspection + selector fix | 03-3.3 green with real Spanish text |
| 18 | C-05 specs asserted a retired single-form UI | BecomeTeacher is a 3-step wizard; specs also depended on bob's mutable application state | Wizard-driven specs + per-run throwaway applicant | C-05-01..04 green |
| 19 | Substring role matchers hit wrong elements | `name: 'ES'` matched tracker "3 Profile & Rates" (nth=0) and jumped the wizard to Step 3 | `exact: true` for short labels | C-05-03 green |
| 20 | Rate-limit budgets starve test suites (login 10/15m, register 10/hr, WS 100/15m) | Full suite + reruns tripped 429s (WS connects logged 429; suites flaked) | Env knobs (`RATE_LIMIT_*_MAX`, defaults unchanged) + generous dev-only values in `docker-compose.override.yml` | Full suite stable across reruns |
| 21 | C-01 API probes matched the group chat instead of the DM | Substring chat matching hit any chat containing the peer | Require `type === 'direct'` in matchers | C-01-01/06 counts correct |

## 2. Coverage matrix (after rollout)

HARD = failing assert vs real backend/state. SOFT = warns by design (linked issue).

| Feature | Mobile jest | Playwright (web) | Acceptance (API) | Notes |
|---|---|---|---|---|
| Login form + wiring | HARD (login.test.tsx) | HARD (01-auth, 25.1) | HARD (TC-AUTH) | |
| Account quick-switch | HARD (profile-switch.test.tsx) | HARD (25.1 logout/login; 25.2 skips non-dev builds honestly) | n/a (UI-only) | #1 |
| Logout → other user, sees data | — | HARD (25.1) | — | The exact reported scenario |
| Registration open/invite/invalid | — | — | HARD (TC-REG-01/02/03) | #2; invite-gated default pinned |
| Token refresh (valid/garbage) | — | — | HARD (TC-SESS-01/05) | No server logout exists (#87) |
| Forgot/reset (contracts) | — | — | HARD (TC-SESS-02/03) | Full round-trip needs email (#83) |
| 2FA negatives | HARD (login.test.tsx 2FA branch) | SOFT (C-03-05 text+setup probe) | HARD (TC-SESS-04) | Positive path needs phone (#82) |
| Profile/settings persist + restore | — | HARD (C-01-05, C-03-01) | — | Rename always restored |
| Blocks lifecycle | — | HARD API (C-03-02) | HARD (TC-SAFE-02, incl. 403 enforcement) | Sidebar hiding TBD (#84) |
| Reports | — | SOFT (C-03-03 modal+201 warn) | HARD (TC-SAFE-01, 201+id) | |
| DM create + dedupe | — | HARD (02, C-01-01 count) | HARD (TC-MSG-02) | |
| Group chat | — | — | HARD (TC-MSG-01, both members read) | No UI group spec yet (#88) |
| Messaging durability + receipts | — | HARD (C-01-06, ws drops 0) | HARD (TC-MSG-02/03) | |
| Translation display | — | HARD (03-3.3, C-01-02 critical) | — | Needs provider stack (libretranslate in compose) |
| Typing indicator (receiver) | — | HARD (9.2) | — | #7 |
| Vocab save + read-back | — | HARD (C-01-03 API) | — | Hub is mined-candidates, not cards |
| SRS review | mocked only | mocked only | — | Real SRS blackbox missing (#89) |
| AI tutor / grammar panels | mocked (mobile) | SOFT (04, 05, C-01-04: Ollama-dependent) | API tolerant (11.x) | #81 |
| Placement/lessons/sessions | mocked (mobile) | mocked UI (21) | HARD (TC-LEARN-01..04) | |
| Marketplace browse/profile/book/credits | mocked (mobile) | HARD (C-04-01..04) | HARD (TC-MKT-01..03) | Rerun-safe credit logic |
| Teacher apply/dashboard/payouts guards | — | HARD (C-04-06 API guards, C-04-07 contract; C-05-01..04 wizard UI via throwaway) | HARD (TC-APPLY-01..04) | Dashboard sections soft (copy) |
| Search | — | HARD text (07-7.2, 10-10.7) | — | Media/gallery soft |
| Calls (API contract) | mocked (mobile) | HARD contract (14/15), media soft | — | No real WebRTC assert (#90) |
| Presence | — | SOFT (existence) | — | Batch endpoint 404s (see #91) |
| GDPR export/erasure | — | SOFT (C-03-06 notes) | HARD (TC-GDPR-01/02, throwaway lifecycle) | Export is JSON, not zip |
| Waitlist | — | HARD (12.x) | — | |
| Error envelope `{error:{kind,…}}` | covered (helper passthrough) | covered (helper) | HARD (TC-ERR-01/02) | #15 |

## 3. Why each layer missed the switch bug (prevention rules)

1. `as any` on the API client defeats `tsc` → **RULE: `src/services/*.ts` (mobile) is `no-explicit-any: error`; screens use typed helpers, never envelopes.**
2. Jest suites mocked the world and mounted nothing under test → **RULE: auth/session/profile/chat-discovery screens get mount-and-drive wiring tests.**
3. E2E never exercised the switch button and used isolated contexts → **RULE: same-session identity switches are first-class scenarios (25.x).**
4. Soft asserts made red green → **RULE: hard by default; soft needs a linked issue.**
5. Acceptance runner never executed → **RULE: `mobile-blackbox` CI job + `npm run acceptance:green` gate; runners must fail loudly (exit codes).**
6. Env-gated behavior (rate limits, invite gate) untested assumptions → **RULE: dev overrides live in `docker-compose.override.yml` (gitignored); prod keeps strict defaults; tests assert real contracts.**
7. Lockfile skew broke builds silently → **RULE: `npm ci` must pass everywhere; align majors (vite/vitest).**
8. Docker context hid workspace deps → **RULE: frontend image builds from repo root.**
9. Short role/name matchers hit wrong elements → **RULE: `exact: true` for short labels; never `nth()` without a same-page uniqueness proof.**
10. Wireframe specs bit-rot (retired copy, placeholder UUIDs, no-login navigation) → **RULE: parity specs use real seeded ids + login; stale copy updated, never deleted silently.**
11. Error envelopes rendered raw crash React → **RULE: all display-bound errors go through `apiErrorMessage()`; axios-shaped failures use curated fallbacks.**
12. Full-suite load flakes (WS starvation, translation queue) → **RULE: generous dev rate budgets; rerun-failures get isolated reruns before product blame; model-dependent specs stay issue-linked soft.**

## 4. Remaining gaps (open GitHub issues)

| Issue | Severity | Gap |
|---|---|---|
| #80 | P1 | vite-dev-only send anomaly (optimistic bubble, POST never fired) — needs repro under vite HMR vs prod parity |
| #81 | P1 | Grammar/AI-tutor UI asserts still soft (Ollama-dependent) — deterministic strategy |
| #82 | P1 | 2FA positive path not blackbox-testable (needs phone/SMS) |
| #83 | P1 | Password-reset full round-trip not blackbox-testable (needs email inbox) |
| #84 | P2 | Blocked-chat sidebar-hiding semantics undefined (backend enforces on send only) |
| #85 | P2 | `quick_drill` sessions legitimately empty for fresh users (no cards) |
| #86 | P2 | sqlmock unit tests cannot catch SQL-dialect bugs (two 42P08s) — DB-backed integration job |
| #87 | P2 | No server-side logout/session revocation (client-only logout; refresh survives) |
| #88 | P2 | No UI spec for group-chat creation |
| #89 | P2 | No real SRS review blackbox (due → answer → mastered vs backend) |
| #90 | P2 | No real WebRTC media assertions (API contract only) |
| #91 | P2 | `GET /presence/batch` 404s (frontend calls it; store tolerates) — route missing or moved |
| #92 | P2 | `docker-compose.dev.yml` stale (`target: development` stage absent) |
