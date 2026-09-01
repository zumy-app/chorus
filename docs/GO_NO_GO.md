# Go / No-Go Checklist — #37

> Release 14.3. Decision meeting is `GO` / `GO with risks` / `NO-GO`.
> One `NO-GO` blocks promotion to prod. `GO with risks` requires a mitigated risk entry
> and a rollback owner. Evidence must be linked; `verify-release-gate.sh` checks the
> mechanical subset automatically, the rest is human sign-off in §3.

## 1. Entry criteria (must be true before the meeting)

- `crew/phase_status.json` Phase 4 tasks 12.1–14.4 = `DONE`; 14.4 runbook + Support SLO dashboards live.
- `gate-on-dev.sh` suite green on `dev` for the candidate digest (`:run_number`).
- No secrets in repo (`verify-isolation.sh` pass), `ESCALATION.md` empty.
- Candidate image is the same digest dev passed (no rebuild).

## 2. Checklist — 13 axes

| # | Axis | Check | Required evidence | Owner | Status |
|---|---|---|---|---|---|
| 1 | Translation | Cache hit p95 < 500ms, chain fallback works, DeepSeek fallback exercised | `phoenix-eval.sh` log (p95, accuracy), `translation_jobs` sample | batchu | _GO / NO-GO_ |
| 2 | Grammar | CEFR feedback + evaluator (accuracy ≥ 80, ≥10 evals) | `translation_evals`/`grammar_evals`, Phoenix trace | batchu | _GO / NO-GO_ |
| 3 | Mobile | Expo RN builds on EAS, WS + translation + call parity with web | `mobile jest` green, EAS build link, manual smoke on device | Kushagra1122 | _GO / NO-GO_ |
| 4 | Premium / Credits & Access | Free 280 / Premium 1000 char limits, plan gating, trial credit 1/mo, fee 10/15% per payout | `entitlement.go` + e2e `translation_blocked` test, billing tests | batchu | _GO / NO-GO_ |
| 5 | Observability | `/health`, `/health/ready`, `/metrics`, Phoenix traces, Grafana dashboards, soak alerts | `curl /health/ready`, `deploy/monitoring/grafana/dashboards/*.json`, `alerts-*.yml` | Raju | _GO / NO-GO_ |
| 6 | Security — mail isolation | Mailu isolated network, only 587 from app, no `VITE_*` SMTP, no committed password | `deploy/mail/verify-isolation.sh` PASS | batchu | _GO / NO-GO_ |
| 7 | Security — app | Rate limits (login/translation/WS), govulncheck clean, npm audit high clean, JWT + 2FA | `security-scan.sh` log, `ratelimit.go` tests | Raju | _GO / NO-GO_ |
| 8 | Privacy & Legal | Retention windows (msg 365d/ inbox 30d / translation 90d / transcript 90d), export + erasure | `docs/DATA_RETENTION_GDPR.md`, `GET /users/me/export`, `DELETE /users/me` | batchu | _GO / NO-GO_ |
| 9 | Support runbook | On-call, runbook, dashboards, SOP for restore/rollback | 14.4 runbook doc + Grafana + `rollback.sh` | SRE (Raju) | _GO / NO-GO_ |
| 10 | Teacher Marketplace | Sign-up, browse/find, profile+booking, dashboard, sessions, review notes, SRS push — all gated on vetting | `12.1–12.6` e2e + `verify-teacher-vetting.sh` | batchu | _GO / NO-GO_ |
| 11 | Payouts | PayPal payouts, fee 10% verified / 15% standard, `TRX-` ref, payout history/settings | `payout.go` tests, sandbox payout log | batchu | _GO / NO-GO_ |
| 12 | Auto-promotion | `dev → prod` image promotion (digest-preserving), prod pull + health, branch protection | `deploy/ci/verify-promotion.sh` + `ci.yml` + prod `/health` | Raju | _GO / NO-GO_ |
| 13 | Load / Soak & durability | 1k WS / 50 msg/s smoke green; soak zero-loss (when run); durable inbox (Postgres source of truth) | `load-smoke.sh` / `load-soak.sh` + `verify-drain.sh` + `ws_fast_dropped_total==0` | SRE | _GO / NO-GO_ |

`verify-release-gate.sh` checks axes 1,2,5,6,7,8,10,12,13 mechanically; axes 3,4,9,11
need human sign-off (device test, payout sandbox, dashboard walkthrough).

## 3. Sign-off

Complete before `promote-prod`. Use `GO` / `GO with risks` / `NO-GO`.

| Owner / Role | Axis | Vote | Risks / notes | Date | Signature |
|---|---|---|---|---|---|
| batchu — product/tech lead | 1,2,4,8,10,11 | GO | | 2026-08-31 | batchu |
| Raju (gosangiraju) — infra/SRE | 5,7,12,13 | GO | | 2026-08-31 | Raju |
| Kushagra1122 — mobile | 3 | GO | | 2026-08-31 | Kushagra |
| Daniella — learning/teacher vetting | 10 (vetting) | GO | vetting doc + harness green | 2026-08-31 | Daniella |
| All leads | 6,9 | GO | mail isolation PASS, runbook live | 2026-08-31 | all |

> Template for next release: copy this table, reset `Status` to `_GO / NO-GO_`, fill Votes.
> `verify-release-gate.sh` parses this table: any `NO-GO` fails the gate; missing
> `GO` entries are reported as `PENDING`.

## 4. Decision matrix

| Condition | Decision | Action |
|---|---|---|
| All 13 axes `GO` | **GO** | Promote `:$run_number → :prod`, verify prod `/health`, announce |
| Any axis `GO with risks` and no `NO-GO` | **GO with risks** | Record mitigation + owner, set alert, promote |
| Any axis `NO-GO` | **NO-GO** | Do not promote; file issue, fix on `dev`, re-run gates |

Rollback owner is the SRE on call. Rollback: `PREV_TAG=<last-good> bash deploy/ci/rollback.sh` then `verify-promotion.sh`.

## 5. Evidence bundle (attach to release tag)

- `gate-on-dev.sh` full log (seed + e2e + phoenix-eval + load-smoke + release-gate).
- `verify-isolation.sh` + `verify-teacher-vetting.sh` logs.
- `/metrics` snapshot + Grafana screenshots.
- `docker buildx imagetools inspect ghcr.io/zumy-app/chorus/backend:prod` + `frontend:prod`.
- This doc with signed §3.

## 6. How to verify locally

```bash
bash deploy/ci/verify-release-gate.sh --offline   # no DB, checks docs + sign-off + shell + compose
bash deploy/ci/verify-release-gate.sh             # with DB: also checks phoenix-eval sample size + accuracy/p95 when PSQL/DATABASE_URL is set
```
