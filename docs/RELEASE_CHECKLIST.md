# Release Checklist — Play Store MVP + Daily Releases

> Use for: (A) first Play Store submission, (B) every daily release.
> Strategy background: `docs/TEST_STRATEGY.md`.

## A. First Play Store Submission (one-time)

### A1. Pre-submission gates (all must be green on `main`)
- [ ] CI `main` run green: `test`, `e2e-mocked`, `mobile-blackbox`,
      `android-build`, `mobile-native` (emulator Maestro PASS), `security`.
- [ ] `deploy-dev` green on the release commit; dev smoke passed.
- [ ] Live flags probe green (general user: stable ON, admin-only OFF).
- [ ] `docs/GO_NO_GO.md` sign-off: ≥4 GO, zero NO-GO, no pending placeholders.
- [ ] `bash deploy/ci/verify-release-gate.sh` PASS (not just `--offline`).

### A2. Play Store artifacts
- [ ] Version bumped (`mobile/app.json` `expo.version`, `versionCode` incremented).
- [ ] Release AAB built from the **exact release commit** (no rebuild drift):
      `cd mobile/android && ./gradlew bundleRelease`.
- [ ] AAB installed on a physical device + emulator; cold start, login, chat,
      send message, flags fetch verified manually once.
- [ ] Crash reporting + backend `/health/ready` dashboard linked in
      `docs/SUPPORT_RUNBOOK.md`.
- [ ] Rollback plan written down: previous prod image digest + DB note
      (migrations are additive/idempotent — rollback is image re-tag only).

### A3. Store listing
- [ ] Title, short + full description, screenshots (phone + 7" tablet),
      feature graphic, privacy policy URL, content rating questionnaire.
- [ ] Closed-testing track first (internal testers ≥ 3 devices), then
      promote to production track.

## B. Daily Release (every `main` push)

### B1. Automated (CI does this — human verifies)
1. [ ] PR merged only with ALL PR gates green (branch protection on `main`):
      `test` (incl. contract + maestro-offline), `e2e-mocked`,
      `mobile-blackbox` (incl. live double-migrate + live flags probe),
      `android-build`, `security`.
2. [ ] `main` run: `mobile-native` Maestro emulator PASS.
3. [ ] `deploy-dev` PASS + dev smoke (open dev app, login, send one message).
4. [ ] `promote-prod`: image digest promoted (no rebuild), prod health green.
5. [ ] Post-prod smoke (≤5 min): prod `/health`, login, `GET /users/me/flags`
      returns stable flags, send one message in a test chat.

### B2. Flag-gated feature rollout (the daily-release mechanism)
1. [ ] Feature merged with flag default OFF for general users
      (`AdminOnly: true` seed, or `Stable: false`).
2. [ ] Verified on dev as admin (flag visible), verified invisible as general
      user (`PreviewUserFlags` or a general test account).
3. [ ] Rollout = `PUT /admin/features/:key/tiers` (beta → stable), one tier
      per day. **Never bundle a flag promotion with a code deploy.**
4. [ ] Watch: error rate, `/health/ready`, login success, message send p95
      for 30 min after each promotion.
5. [ ] Rollback = set tiers back (instant, no deploy, no migration).

### B3. If anything is red — STOP rules
- **PR gate red → do not merge.** Fix forward; no `main` pushes while red.
- **`mobile-native` red → do not promote to prod.** Native crash/regression
      ships to the store otherwise. File a P0, fix, re-run.
- **Prod health red after promote → rollback immediately:**
      `PREV_TAG=<last-good> bash deploy/ci/rollback.sh`, then investigate.
      DB needs no rollback (migrations are additive + idempotent).

## C. Emergency Hotfix
1. Branch from the prod tag, minimal fix, flag OFF if behavioral.
2. Fast-track PR gates (same gates, no skipping).
3. `main` → dev → 15-min soak → promote → post-prod smoke.
4. Post-mortem note in `docs/SUPPORT_RUNBOOK.md` within 24h.
