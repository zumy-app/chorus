# Chorus Test Strategy — MVP + Daily Releases

> **Goal:** ship the minimal production mobile app (Release 1) to the Play Store
> with confidence, then release small, flag-gated increments **every day**.
> **Authority:** `MVP_RELEASE_PLAN.md`, `REQUIREMENTS_MASTER.md`.
> Companion runbook: `docs/RELEASE_CHECKLIST.md`.
> Legacy phase-gate plan: `docs/TEST_PLAN.md` (still valid for NFR/load gates).

## 1. Principles

1. **Fast feedback first.** PR gates must finish in <10 min. Anything slow or
   flaky moves to `main`-only jobs.
2. **Fail fast, in order:** static (lint/type/build) → unit → contract →
   blackbox → native/device → deploy gates.
3. **Deterministic PRs.** No PR gate may depend on wall-clock timing, external
   providers, or a live Redis. Those run on `main` with real services.
4. **Contract over mocks.** Mobile/web unit tests mock the API; the
   `verify-api-contract.sh` gate + blackbox jobs prove the mocks match reality.
5. **Flags are the release mechanism.** Every non-Release-1 feature ships
   behind a flag, OFF for general users. The flags probe proves visibility.
6. **Migrations must be re-runnable.** `Migrate()` runs on every boot; the
   migration-safety test blocks non-idempotent SQL at PR time.

## 2. Test Pyramid (MVP)

| Level | What | Tool / Command | Runs on | Must be green for |
|---|---|---|---|---|
| L0 static | go vet/build, tsc, eslint, compose config, shell syntax | CI `test` job | every PR | merge |
| L1 unit | backend `go test -short ./...`, frontend `vitest`, mobile `jest` | CI `test` job | every PR | merge |
| L2 contract | `verify-api-contract.sh` (routes ↔ client ↔ wiring) | CI `test` job | every PR | merge |
| L2b migration-static | `TestMigrations_IdempotentPatterns` (in `go test -short`) | CI `test` job | every PR | merge |
| L2c maestro-offline | `verify-maestro.sh` (flows valid + selectors live) | CI `test` job | every PR | merge |
| L3 blackbox | real backend + Postgres + Redis: functional, learning, acceptance, live double-migrate, live flags probe | CI `mobile-blackbox` | every PR | merge |
| L4 web e2e | Playwright mocked (no Docker) | CI `e2e-mocked` | every PR | merge |
| L5 native build | `assembleDebug` APK | CI `android-build` | every PR | merge |
| L6 native e2e | Maestro on emulator (login → chat → send → flags) | CI `mobile-native` | `main` only | daily release |
| L7 deploy gates | dev deploy + `gate-on-dev.sh`, prod promote + health | CI `deploy-dev` / `promote-prod` | `main` only | prod release |
| L8 security | govulncheck + npm audit | CI `security` | every PR | merge |

Full suites (`go test ./...` without `-short`) run in `mobile-blackbox`
(which has real Postgres/Redis) and locally before release.

## 3. What Changed and Why (gap closure)

| Gap | Before | After |
|---|---|---|
| Flaky PRs (routing/soak/perf timing) | `go test ./...` on PRs; occasional `TestHandleServerMessageMarksDelivered` flakes | `-short` skips in `routing_test.go`, `soak_test.go`, `perf_benchmark_test.go`; full suite on `main` |
| Feature flags untested | service tests only; no handler/middleware coverage | `middleware/feature_flags_test.go` (5 tests: admin allow, general deny, stable allow, unauth, defaultOff) + `handlers/feature_flags_test.go` (9 tests: all endpoints incl. preview-as-general) |
| Client/server drift | nothing checked route ↔ client parity | `deploy/ci/verify-api-contract.sh`: 15 backend routes, 11 client methods, 9 flag keys both sides, app wiring, call gating |
| Migration risk on daily deploys | no guard; `Migrate()` runs every boot | `migration_safety_test.go`: static idempotency test (PR) + live double-run (CI with Postgres) |
| Native e2e never ran | Detox configured but `mobile/e2e/` missing | Maestro flows (`login`, `chat-send-message`, `flags-gate`) + offline selector-drift gate + emulator job on `main` |
| Native build untested in CI | no Android build job | `android-build` job (assembleDebug + APK artifact) on every PR |
| Release-1 visibility unproven | no live check | blackbox flags probe: alice (general) sees stable ON, admin-only OFF |

## 4. Daily Release Flow

```
PR (fast, <10 min):  L0 → L1(-short) → L2 contract → L2b migration-static
                     → L2c maestro-offline → L3 blackbox → L4 mocked e2e
                     → L5 android APK → L8 security
                     = merge when ALL green.

main (full, ~30 min): everything above (full, no -short) + L6 Maestro emulator
                     + images + dev deploy + dev gates
                     = daily release candidate.

prod: promote exact dev image digest → health verify → smoke probe.
      Rollback = re-tag previous digest (see RELEASE_CHECKLIST.md).
```

## 5. Feature-Flag Testing Contract

- **Unit:** resolution rules (stable/beta/admin/override/defaultOff) in
  `services/feature_flags_test.go`.
- **Middleware:** `RequireFlag` allows/denies per tier; unknown flags deny when
  `defaultOff` (production posture).
- **Handlers:** `GetMyFlags` per role; `PreviewUserFlags` always resolves as
  general (admin sees what the user sees); tier/override CRUD.
- **Contract:** flag keys identical in `DefaultFlags()`, `RolloutFlagKey`,
  client defaults.
- **Live:** blackbox probe asserts general-user visibility after every merge.
- **Native:** `flags-gate.yaml` asserts 📞/📹 hidden for general users.

Adding a new flag requires: seed row + `RolloutFlagKey` entry + client default
+ `RequireFlag` on routes + UI gating. The contract gate fails the PR if any
piece is missing.

## 6. Flaky-Test Policy

A test that fails intermittently is a **P0 bug in the test**, not "CI noise":

1. Quarantine with `testing.Short()` skip + a tracking note (as done for
   routing/soak/perf).
2. Full suite still runs it on `main` (mobile-blackbox has real services).
3. Fix root cause (deterministic waits, miniredis isolation) before removing
   the skip.
4. Never add `time.Sleep`-based assertions to new tests; use channels with
   timeouts or poll with deadlines.

## 7. Local Commands

```bash
# PR-equivalent, backend (fast)
cd backend && go build ./... && go vet ./... && go test -short ./... -count=1

# Full backend (needs Postgres/Redis for live tests, else they skip)
cd backend && go test ./... -count=1

# Offline gates (no Docker, <10s each)
bash deploy/ci/verify-api-contract.sh
bash deploy/ci/verify-maestro.sh

# Frontend / mobile
cd frontend && npx tsc --noEmit && npm test
cd mobile && npx tsc --noEmit && npm test

# Maestro locally (emulator running, backend reachable)
cd mobile && npm run maestro:test
```
