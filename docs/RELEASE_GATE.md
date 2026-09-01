# Release Gate — #36 Test & Release-Gate Strategy

> Authority: `REQUIREMENTS_MASTER.md` 14.3, `PHASE_1_IMPLEMENTATION_PLAN.md` §6,
> NFR-26. Enforced mechanically by `deploy/ci/verify-release-gate.sh` and the
> `deploy-dev → gate-on-dev → promote-prod` pipeline in `.github/workflows/ci.yml`.

## 1. Promotion model

```
push main → test + security (PR + main) → images (:run_number + :dev)
        → deploy-dev → gate-on-dev (seed, e2e, phoenix-eval, load-smoke, release-gate, mail-isolation)
        → promote-prod (imagetools create :run_number → :prod, no rebuild) → prod /health
        → verify-promotion → notify-failure on red
```

Prod never rebuilds from source: promotion re-tags the exact digest dev passed.
`docker-compose.prod.yml` pins `image: ${IMAGE_NAME}/backend|frontend:${IMAGE_TAG:-prod}`.
Rollback: `PREV_TAG=<last-good> bash deploy/ci/rollback.sh`.

See `docs/CI_CD_NFR26.md` for the full NFR-26 handoff.

## 2. Gates

| Gate | When | Checks (all green or NO-GO) | Evidence | Signer |
|---|---|---|---|---|
| **P0** | Phase 1 P0 exit | P0 bugs + onboarding + mobile parity; no `Sorry...` generic error; `go test` + `frontend build+test` + `mobile jest` + `compose config` green; e2e smoke on dev | CI `test` job, `e2e.sh` | batchu + Kushagra |
| **Phase 4** | Pre-prod promotion | **Marketplace + payouts + auto-promotion** functional (14.1), **data retention/GDPR** tested (14.2), **release gate + Go/No-Go** signed (14.3), **support runbook + dashboards** live (14.4); marketplace gated on vetting | `gate-on-dev.sh`, `verify-release-gate.sh`, payout/teacher tests | All leads |
| **Prod promotion** | Every push to `main` | `gate-on-dev.sh` suite on **dev** host: `seed` → `e2e` → `phoenix-eval` (accuracy ≥ 80, p95 < 500ms, ≥10 evals) → `load-smoke` (≤2% errors) → `release-gate` → `mail-isolation` → `verify-drain` when soak | dev `gate-on-dev.sh` logs + `/metrics` | CI (automated); human GO/NO-GO for tagged releases |
| **Soak / durability** | Phase 3+tagged | 1k WS / 50 msg/s / 24h drain, `errors.rate==0`, Postgres inbox drain `ws_fast_dropped_total==0` | `docs/SOAK_TEST.md`, `verify-drain.sh` | SRE |

## 3. Automated thresholds (release-gate script)

`deploy/ci/verify-release-gate.sh` fails the pipeline when any threshold is red.
Defaults mirror `phoenix-eval.sh` + NFRs:

| Metric | Threshold | Source |
|---|---|---|
| Translation golden-set accuracy | `≥ 80` | `translation_evals` |
| Translation p95 latency (cache-hit) | `< 500 ms` | `translation_jobs.latency_ms` |
| Eval sample size | `≥ 10` | `translation_evals` |
| Load smoke error rate | `≤ 2%` | Artillery `errors.rate` |
| Soak (when run) | `errors.rate == 0`, `ws_fast_dropped_total == 0` | `verify-drain.sh` |
| Mail isolation | `PASS` | `deploy/mail/verify-isolation.sh` |
| Security | `govulncheck` clean, `npm audit ≥ high` clean | `security-scan.sh` |
| Docs presence | `docs/RELEASE_GATE.md`, `docs/GO_NO_GO.md`, `docs/DATA_RETENTION_GDPR.md`, `docs/CI_CD_NFR26.md`, `docs/TEACHER_VETTING.md`, `docs/SUPPORT_RUNBOOK.md` | file check |
| Compose + shell | `docker compose config --quiet` + `bash -n` | CI `test` |
| GoNoGo sign-off | all required owners `GO` or `GO with risks` and no `NO-GO` | `docs/GO_NO_GO.md` sign-off table |

Override via env: `ACCURACY_THRESHOLD`, `P95_MS_MAX`, `MIN_EVALS`, `SKIP_*=1` for offline mode.

## 4. Manual gates (Go/No-Go checklist #37)

Tracked in `docs/GO_NO_GO.md`. Each axis has an **owner**, **evidence**, and a **GO / NO-GO** vote.
A single `NO-GO` blocks release; `GO with risks` requires a mitigated risk entry.
See that doc for the 13-axis checklist (translation, grammar, mobile, premium,
observability, security, legal, support, marketplace, payouts, auto-promotion, soak, rollback).

## 5. Verification

```bash
# Local (no infra):
bash deploy/ci/verify-release-gate.sh --offline
bash deploy/mail/verify-isolation.sh
bash deploy/ci/verify-teacher-vetting.sh
go vet ./... && go test ./...

# On dev host (with DB):
bash deploy/ci/gate-on-dev.sh
# subset:
GATES="release-gate mail-isolation" bash deploy/ci/gate-on-dev.sh

# Compose + YAML sanity:
docker compose -f docker-compose.prod.yml config --quiet
bash -n deploy/ci/*.sh deploy/mail/*.sh
```

## 6. Branch & environment protection

- `main` branch protection: require `test` + `security` green, no direct pushes.
- Dev and prod hosts hold GHCR creds; workflow never ships registry secrets over SSH except as env.
- `ESCALATION.md` must be empty at gate time; any open blocker in `crew/phase_status.json` fails `verify-release-gate.sh`.

## 7. Artifacts & retention

- Per-run: `e2e/playwright-report` artifact, `/metrics` snapshot, `translation_evals` sample.
- Per-promotion: recorded digest (`IMAGE_NAME/*:$run_number → :prod`) + prod `/health` probe.
- Retention: evidence retained per `docs/DATA_RETENTION_GDPR.md` (translation jobs 90d, etc.).
