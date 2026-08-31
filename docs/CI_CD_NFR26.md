# CI/CD Quality Gates + Dev Environment (NFR-26, id 5.4)

Chorus ships to `prod` only after the built image passes a gate suite **on the
isolated `dev` environment**. Push to `main` → build images → deploy to `dev` →
run gates → promote the **same image digest** to `prod`. Prod is never rebuilt
from source by the pipeline; it consumes the image `prod` was promoted from.

Owner: **Raju (gosangiraju)**. See `PHASE_1_IMPLEMENTATION_PLAN.md` §5.5 and
`REQUIREMENTS.md` NFR-26.

## Flow

```
push to main / workflow_dispatch
        │
        ▼
  [test]  build + go test/vet + npm test (frontend+mobile) + compose validate
  [security]  govulncheck + npm audit
        │ (both green; also required on every PR)
        ▼
  [images]  build backend + frontend → push ghcr.io/zumy-app/chorus/*:dev + :$run_number
        │
        ▼
  [deploy-dev]  ssh → chorus-dev stack pulls :$run_number → up -d → wait /health/ready
        │
        ▼
  [gates on dev]  seed-user → Playwright e2e → Phoenix golden-set eval → Artillery WS smoke
        │  all green?
        ├── yes ──► [promote-prod]  buildx imagetools create :$run_number → :prod
        │                        └─► ssh prod host → pull :prod → up -d
        └── no  ──► job fails → [notify-failure] Slack alert
```

The web client uses a **relative** `/api/{version}` origin
(`packages/shared/src/config.ts`), so the **same** frontend image works in dev
and prod behind their own LBs — this is why promotion is a re-tag, not a rebuild,
and why `VITE_API_URL` is not baked in.

## Dev environment (`deploy/dev/`)

- `docker-compose.yml` — isolated `chorus-dev` project: own Postgres/Redis,
  image-based (no `build:`), LB = Caddy, Phoenix trace backend.
- `Caddyfile` — dev LB. Proxies `/api/*`, `/ws`, `/health*` → backend; static →
  frontend. Only the LB publishes host ports. Swap `dev.chorus.talk` for your host.
- `seed/00-init.sql` — idempotent first-boot extensions.
- `seed/01-demo-data.sql` — documents the canonical dev users (created via API).
- `README.md` — bring-up + verification.

Demo/CI users (seeded by `deploy/ci/seed-dev.sh` through `/api/v1/auth/register`):

| Email | Password | Langs |
|-------|----------|-------|
| `uhsarp@gmail.com` | `Demor@cer1` | en → es |
| `avcxafefwer@gmail.com` | `Demor@cer1` | es → en |

## Gate scripts (`deploy/ci/`)

| Script | Gate | Env |
|--------|------|-----|
| `security-scan.sh` | `govulncheck` + `npm audit` (frontend/mobile/e2e) | `NPM_AUDIT_LEVEL` |
| `seed-dev.sh` | create/verify demo users on dev | `E2E_BASE_URL` |
| `e2e.sh` | Playwright suite against dev LB | `E2E_BASE_URL` |
| `phoenix-eval.sh` | golden-set accuracy + p95 latency | `DATABASE_URL`/`PSQL`, `ACCURACY_THRESHOLD`, `P95_MS_MAX`, `MIN_EVALS` |
| `load-smoke.sh` | Artillery WS (100 concurrent / 10 msg/s / 5m) | `E2E_BASE_URL`, `WS_TOKEN` |
| `gate-on-dev.sh` | orchestrate the above on the dev host | `E2E_BASE_URL`, `DOCKER_COMPOSE_FILE`, `GATES` |
| `promote.sh` | re-tag `:dev` snapshot → `:prod` (no rebuild) | registry creds |

Throttle defaults: accuracy `>= 80`, p95 latency `< 500 ms` (cache-hit set, falls
back to all jobs if the cache is cold), `>= 10` eval samples, max `2%` WS error.

## Repo hooks / settings required

- `.github/workflows/ci.yml` (the only workflow; includes gates + promotion +
  failure notify). `test` + `security` are required on every PR.
- **Branch protection on `main`**: require `test` and `security` to pass,
  no direct pushes, signed commits optional. (Configured in the GitHub repo
  settings > Branches; the workflow file itself cannot enforce it.)

## Repository secrets (never commit)

`CR_USERNAME`/`CR_PASSWORD` (or `GITHUB_TOKEN`), `DEV_SSH_HOST`/`DEV_SSH_USER`/
`DEV_SSH_KEY`, `PROD_SSH_HOST`/`PROD_SSH_USER`/`PROD_SSH_KEY`,
`PROVIDER_OPENROUTER_KEY`, `PROVIDER_NVIDIA_KEY`, `DEV_POSTGRES_PASSWORD`,
`DEV_JWT_SECRET`, `DEV_WAITLIST_ADMIN_EMAILS`, `CADDY_EMAIL`, optional
`SLACK_WEBHOOK_URL`.

The dev/prod hosts must hold the GHCR registry credential (Dokploy registry
account) so `docker compose pull` is authenticated — the workflow does not push
credentials over SSH.

## Verify locally (no infra)

```bash
# compose + config validity (client-side, no daemon needed)
docker compose -f deploy/dev/docker-compose.yml config --quiet
docker compose -f docker-compose.yml config --quiet
docker compose -f docker-compose.dev.yml config --quiet

# shell syntax
bash -n deploy/ci/*.sh

# YAML validity
python -c "import yaml; [yaml.safe_load(open(p)) for p in ('.github/workflows/ci.yml','deploy/ci/artillery.load.yml','deploy/dev/docker-compose.yml')]"
```

## Promotion semantics

`promote.sh` uses `docker buildx imagetools create` to copy the manifest for the
exact `:$run_number` digest to `:prod` — the production host then `pull`s `:prod`.
No source rebuild, no fork between environments. If any dev gate fails, the
`promote-prod` job never runs and `prod` stays on the last green digest; the
failure is notified.
