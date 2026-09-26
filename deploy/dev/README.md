# Chorus `dev` environment (NFR-26, id 5.4)

The **`dev` environment** is a production-shaped but isolated deployment that
runs **the same images** the CI pipeline builds — it is never rebuilt locally.
Promotion to `prod` is an **image promotion** (re-tag the digest), not a rebuild
(see [docs/CI_CD_NFR26.md](../../docs/CI_CD_NFR26.md)).

## Architecture

```
                 ┌─────────────────────────── chorus-dev (Dokploy project) ──────┐
 client ─────────► lb (Caddy :80/:443) ──► frontend-dev (:3000, static SPA)      │
                       │  /api/*  ─────────► backend-dev (:8080) ──► postgres-dev│
                       │  /ws     ─────────► backend-dev         ──► redis-dev   │
                       │                                     └─► libretranslate-dev
                       │                                     └─► phoenix-dev (trace)
```

- **Isolated data plane**: its own `postgres-dev` + `redis-dev` volumes, **never**
  shared with prod.
- **Same origin for web**: the client defaults to a relative `/api/{version}`
  (see `packages/shared/src/config.ts`), so the **same** frontend image is valid
  for dev and prod. The LB reverse-proxies `/api` + `/ws` to the backend. This is
  why image promotion never needs `VITE_API_URL` baked in. (Mobile/dev builds that
  run against an external origin still use `VITE_API_URL`, per `frontend/src/services/api.ts`.)
- **Dev LB** is Caddy (`deploy/dev/Caddyfile`), the dev analog of prod Traefik;
  only `lb` publishes host ports.

## Files

| Path | Purpose |
|------|---------|
| `deploy/dev/docker-compose.yml` | Compose project `chorus-dev` (image-based) |
| `deploy/dev/Caddyfile` | Dev LB routing (`/api`, `/ws`, static) |
| `deploy/dev/seed/00-init.sql` | Idempotent first-boot DB init (extensions) |
| `deploy/dev/seed/01-demo-data.sql` | Documented demo users (created via API gate) |

## Bring-up (dev host / Dokploy project `chorus-dev`)

```bash
# Secrets (secrets store or Dokploy env): DEV_POSTGRES_PASSWORD, DEV_JWT_SECRET,
# PROVIDER_OPENROUTER_KEY, PROVIDER_NVIDIA_KEY, DEV_WAITLIST_ADMIN_EMAILS,
# CADDY_EMAIL, DEV_INVITE_BASE_URL. None are committed.

docker compose -f deploy/dev/docker-compose.yml up -d
```

Verify reachable via the LB domain (`https://dev.chorus.talk` by default):

```bash
curl -fsS https://dev.chorus.talk/api/v1/health   # -> {"status":"healthy",...}
curl -fsS https://dev.chorus.talk/                # -> SPA
```

## Local run (compose config validation only, no daemon)

```bash
docker compose -f deploy/dev/docker-compose.yml config --quiet
```

## Seeding

The backend auto-migrates the schema on start (`backend/internal/database/postgres.go`).
Demo users are registered through the public auth API by the CI `seed` step
(`deploy/ci/seed-dev.sh`) so passwords are correctly bcrypt-hashed:

| Email | Password | Langs |
|-------|----------|-------|
| `uhsarp@gmail.com` | `Demor@cer1` | en → es |
| `avcxafefwer@gmail.com` | `Demor@cer1` | es → en |

## Quality gates (run against this stack before promotion)

See `deploy/ci/*.sh`. Gates: Go/unit+integration, frontend+mobile tests, e2e
Playwright, Phoenix golden-set eval (accuracy threshold + p95 < 500 ms on cache-hit
set), Artillery load smoke (100 WS / 10 msg/s / 5 min), and `govulncheck` + `npm audit`.
