# Chorus Messenger — Dokploy Deployment Guide

Deploy Chorus to production on Dokploy with the domain **chorus.talk**.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Step 1: DNS Configuration (Cloudflare)](#step-1-dns-configuration-cloudflare)
3. [Step 2: Prepare Environment Variables](#step-2-prepare-environment-variables)
4. [Step 3: Deploy via Dokploy Dashboard](#step-3-deploy-via-dokploy-dashboard)
5. [Step 4: Configure Domain in Dokploy](#step-4-configure-domain-in-dokploy)
6. [Step 5: Verify Deployment](#step-5-verify-deployment)
7. [Updating the Deployment](#updating-the-deployment)
8. [Troubleshooting](#troubleshooting)

---

## Prerequisites

- ✅ **Dokploy** installed on your VPS (the "Chorus" project is already created)
- ✅ **Domain** `chorus.talk` added to Cloudflare (you've done this)
- ✅ **This repo** cloned/available on the VPS (or Dokploy can pull from GitHub)

---

## Step 1: DNS Configuration (Cloudflare)

You need to add a DNS record in Cloudflare pointing `chorus.talk` to your VPS IP.

### 1. Log into Cloudflare
Go to https://dash.cloudflare.com and select your domain.

### 2. Add DNS Record

| Field | Value |
|-------|-------|
| **Type** | `A` |
| **Name** | `chorus` |
| **IPv4 address** | `<Your VPS IP Address>` |
| **Proxy status** | **DNS Only** (grey cloud) — _or_ **Proxied** (orange cloud) if you want Cloudflare in front |

> ⚠️ **Important**: If using **Proxied (orange cloud)**, Dokploy's SSL certificate may not work for direct connections. Use **DNS Only (grey cloud)** for simplicity, or configure Cloudflare's Origin Certificate. Grey cloud is recommended for initial setup.

### 3. Wait for propagation
```powershell
# Verify DNS resolves correctly
nslookup chorus.talk
# Should show your VPS IP
```

![DNS Record Example](https://i.imgur.com/placeholder.png)

---

## Step 2: Prepare Environment Variables

### 1. Copy the production env template

```bash
# On your local machine
cp .env.prod.example .env.prod
```

### 2. Generate a strong JWT secret

```bash
# Run this command (PowerShell or bash)
openssl rand -base64 64
# Output example: "xK8mZpL4qR9vN2wB5yE7hJ3fA1cG6iD0sT5uP8oM9nL2kV4rW7xZ1yC3vB6nM0..."
```

### 3. Fill in your `.env.prod`

```
# Note: DB_PASSWORD is no longer needed — the compose hardcodes the DB password
# and the postgres entrypoint re-syncs it on every start. JWT_SECRET is required.
JWT_SECRET=xK8mZpL4qR9vN2wB5yE7hJ3fA1cG6iD0sT5uP8oM9nL2kV4rW7xZ1yC3vB6nM0...
GOOGLE_TRANSLATE_API_KEY=   # Optional — leave blank for mock translations

# ─── Waitlist & admin (REQUIRED for invite-only registration) ───────────────
# Comma-separated allowlist of emails that can access /admin/waitlist and send
# invites. These are forwarded to the backend container by docker-compose.prod.yml.
WAITLIST_ADMIN_EMAILS=you@your-domain.com
INVITE_BASE_URL=https://chorus.talk/register
INVITE_TTL_HOURS=168

# ─── Mailu SMTP (REQUIRED to deliver waitlist confirmations + invites) ──────
# NFR-22 isolation: Mailu runs on a SEPARATE host/network (see
# deploy/mail/README.md). The app reaches it on SUBMISSION port 587 only.
MAILU_SMTP_HOST=mail.your-domain.com     # mail host private IP / hostname
MAILU_SMTP_PORT=587                       # submission (STARTTLS), NOT 465
MAILU_SMTP_USERNAME=your-mailbox@your-domain.com
MAILU_SMTP_PASSWORD=...                   # Dokploy SECRET, never VITE_*, never committed
MAILU_SMTP_FROM=your-mailbox@your-domain.com
```

> ⚠️ **Security**: Never commit `.env.prod` to git. It's already in `.gitignore`.
>
> ⚠️ **Note**: Env vars are only injected into the backend container if
> `docker-compose.prod.yml` references them (e.g. `- WAITLIST_ADMIN_EMAILS=${WAITLIST_ADMIN_EMAILS:-}`).
> If you add a new variable in Dokploy but the compose file doesn't list it under
> the backend service's `environment:`, the backend never sees it.
>
> ⚠️ **Secrets (NFR-22)**: `MAILU_SMTP_PASSWORD` must be set as a Dokploy
> **secret**, never as a plain env var or in `.env.*`, never prefixed `VITE_`,
> and never committed to the repo.

---

## Step 3: Deploy via Dokploy Dashboard

### 1. Open Dokploy Dashboard

Navigate to `https://<your-dokploy-instance>` and log in.

### 2. Select the "Chorus" Project

If you haven't created it yet, click **New Project** → Name: `Chorus`.

### 3. Add a New Service → "Docker Compose"

Dokploy supports deploying via Docker Compose. Use the `docker-compose.prod.yml` file:

**Option A: Deploy from GitHub (recommended)**

1. Connect your GitHub repository in Dokploy settings
2. Create a new service → **Docker Compose**
3. Set:
   - **Repository**: `your-org/chorus` (or wherever this repo is)
   - **Branch**: `main` (or your deployment branch)
   - **Compose file path**: `docker-compose.prod.yml`
4. Add environment variables from your `.env.prod` file in the Dokploy env vars section
5. Click **Deploy**

**Option B: Deploy via file upload**

1. Create a new service → **Docker Compose**
2. Upload or paste the contents of `docker-compose.prod.yml`
3. Add environment variables
4. Click **Deploy**

### 4. Wait for Build

Dokploy will:
1. Pull the source code (if from GitHub)
2. Build the Docker images (backend binary + frontend static files)
3. Start all 4 services (postgres, redis, backend, frontend)
4. Run health checks

You'll see logs streaming in real-time:

```
✅ chorus-postgres   Healthy
✅ chorus-redis      Healthy
✅ chorus-backend    Running
✅ chorus-frontend   Running
```

---

## Step 4: Configure Domain in Dokploy

### 1. Add Domain to Frontend Service

In the Dokploy dashboard, navigate to your **frontend** service:

1. Go to **Domains** tab
2. Click **Add Domain**
3. Enter: `chorus.talk`
4. Click **Save**

Dokploy (via Traefik) will:
- Automatically obtain an SSL certificate from Let's Encrypt
- Route `https://chorus.talk` → `frontend:3000` (internal)
- Handle all HTTP→HTTPS redirects

### 2. (Optional) Add API Subdomain

If you want the API accessible directly:
1. Add another domain: `api.chorus.talk`
2. Route it to the **backend** service (port `8080`)

---

## Step 5: Verify Deployment

### Check the website

```bash
# Should return the Chorus landing page HTML
curl https://chorus.talk

# Should redirect to /register or show login page
curl https://chorus.talk/login
```

### Check the backend health

```bash
# Via the frontend proxy (works if nginx forwards /api)
curl https://chorus.talk/health

# Expected: {"status":"healthy","version":"2.0.0"}
```

### Test registration

```bash
curl -X POST https://chorus.talk/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "TestPass123!"}'

# Expected: {"user": {...}, "tokens": {...}}
```

### Check Dokploy logs

```bash
# In Dokploy dashboard → Service → Logs
# Or via CLI if you have Dokploy CLI access
```

---

## Updating the Deployment

When you make changes to the code and push to GitHub:

1. In Dokploy dashboard → Service → **Redeploy**
2. Or set up **automatic deployments** via GitHub webhooks

Dokploy will:
1. Pull the latest code
2. Rebuild only changed images (Docker layer caching)
3. Restart services with zero downtime

```bash
# Manual trigger via Dokploy CLI (if configured)
dokploy service deploy chorus
```

---

## Architecture Overview (Production)

```
Internet
    │
    ▼
Cloudflare (DNS)
    │
    ▼
VPS (Dokploy)
    │
    ├─ Traefik (Reverse Proxy) — handles SSL, domain routing
    │   │
    │   ├─ https://chorus.talk → Frontend (Port 3000)
    │   │   │
    │   │   ├─ / (static files)          → nginx serves index.html
    │   │   ├─ /api/*                    → proxied to backend:8080
    │   │   └─ /ws (WebSocket)           → proxied to backend:8080
    │   │
    │   └─ (optional) https://api.chorus.talk → Backend (Port 8080)
    │
    ├─ chorus-frontend (nginx, port 3000)
    ├─ chorus-backend  (Go, port 8080)
    ├─ chorus-postgres (PostgreSQL, port 5432)
    └─ chorus-redis    (Redis, port 6379)

SEPARATE MAIL HOST (NFR-22 — NOT on the Dokploy VPS)
    │
    └─ Mailu (deploy/mail/docker-compose.mail.yml)
         └─ port 587 (submission) ← only reachable from the app subnet
```

### Service Network

All services are on the internal `chorus-network` bridge network.
- Frontend can reach backend via `http://backend:8080`
- Backend can reach postgres via `postgres:5432`
- Backend can reach redis via `redis:6379`

No external ports are exposed except through Dokploy's Traefik ingress.

---

## Troubleshooting

### Issue: "chorus.talk" doesn't load

```bash
# 1. Check DNS
nslookup chorus.talk
# Should resolve to your VPS IP

# 2. Check if Dokploy/Traefik is running
ssh user@your-vps
docker ps | grep traefik

# 3. Check service logs via Dokploy dashboard
# → Service → Logs tab
```

### Issue: SSL certificate not provisioning

If using Cloudflare **Proxied (orange cloud)**:
- Traefik may not get Let's Encrypt certs because Cloudflare proxy hides the real IP
- Solution: Use **DNS Only (grey cloud)** for initial setup
- Or: Use Cloudflare **Origin Certificate** + **Full (strict)** SSL mode

### Issue: Backend can't connect to PostgreSQL

```bash
# Check database logs in Dokploy
# Or via CLI:
docker exec -it chorus-postgres psql -U messenger -d messenger_prod -c "SELECT 1"
```

If the backend logs `pq: password authentication failed for user "messenger"`,
it usually means the `postgres_data` volume was initialized with an older
password. The official Postgres image only applies `POSTGRES_PASSWORD` when the
data volume is empty, so redeploying alone does NOT update the stored password —
the backend then connects with the new value while the DB still expects the old
one, and the postgres logs also show a `password authentication failed` every
time the backend retries.

This deploy sets a custom entrypoint (`deploy/postgres/entrypoint.sh`) that
re-syncs the `messenger` user's password to `POSTGRES_PASSWORD` on every
container start, so redeploys keep working automatically. Two caveats:

- The script must be LF-formatted: `dos2unix deploy/postgres/entrypoint.sh`
  (or commit it with LF line endings, which is required on Windows).
- The `postgres` container must be **recreated**, not just restarted, for the
  entrypoint to run with the new image/config. A Dokploy "Redeploy" does this
  automatically because the compose config changed.

If the postgres logs also repeat `FATAL: database "messenger" does not exist`,
that's just the container healthcheck probing without a database name —
`pg_isready -U messenger` defaults to a database named `messenger`. The current
compose uses `pg_isready -U messenger -d messenger_prod` to avoid the noise.

### Issue: WebSocket not working

The nginx config has a 24-hour read timeout for WebSocket connections. If issues occur:

1. Check browser console for WebSocket errors
2. Ensure the nginx config has the WebSocket upgrade headers
3. Verify the domain is reachable on port 443

### Issue: Messages not translating

Without a Google Translate API key:
- Backend uses mock translations (adds `[lang]` prefix to text)
- This is normal and doesn't affect functionality
- To enable real translations, set `GOOGLE_TRANSLATE_API_KEY` in env vars

---

## Rollback

If a deployment fails or breaks the site:

1. Go to Dokploy dashboard → Service → **Deployments** tab
2. Find the last working deployment
3. Click **Rollback**
4. Dokploy redeploys the previous version

---

## Mail Server Isolation (NFR-22)

Prod mail (Mailu) **must not share the same host/network namespace** as the app
servers. Deploy Mailu from `deploy/mail/docker-compose.mail.yml` on a **separate
host** (or an isolated Docker network), never inside `docker-compose.prod.yml`.

- **Ingress**: the app reaches Mailu on **587** (submission / STARTTLS) only;
  `25`, `465`, webmail, IMAP/POP3 are not open from the app subnet.
- **Secrets**: rotate `MAILU_SMTP_PASSWORD` (prior value is in repo history),
  store it as a Dokploy **secret**, no `VITE_` prefix. The backend consumes it
  from `MAILU_SMTP_*` env vars only.
- **Envelope auth**: SPF + DKIM + DMARC per Issue #3 — see
  `deploy/mail/README.md` for the exact DNS records.
- **Release gate**: run `bash deploy/mail/verify-isolation.sh` pre-deploy / in
  CI; it blocks the gate if Mailu is co-located, mail ports are published, a
  `VITE_*` SMTP secret exists, or a real password is committed.

```yaml
# .github/workflows/ci.yml (or the promotion gate)
- name: NFR-22 mail isolation check
  run: bash deploy/mail/verify-isolation.sh
```

---

## Security Checklist

- [ ] JWT_SECRET is a long (>64 chars) random string
- [ ] DB password in compose (POSTGRES_PASSWORD / DATABASE_URL) is not dashed default
- [ ] PostgreSQL port not exposed publicly (127.0.0.1 or Dokploy internal only)
- [ ] Redis port not exposed publicly (127.0.0.1 or Dokploy internal only)
- [ ] HTTPS is working (Let's Encrypt via Dokploy)
- [ ] CORS only allows known origins
- [ ] Production env vars not committed to git
- [ ] Regular backups configured for `postgres_data` volume
- [ ] **NFR-22: Mailu runs on a separate host/network** (`deploy/mail/`), never in the app compose
- [ ] **NFR-22: App reaches Mailu on 587 (submission) only**; `25` blocked from public
- [ ] **NFR-22: `MAILU_SMTP_PASSWORD` rotated + in Dokploy secrets** (no `VITE_*`, not committed)
- [ ] **NFR-22: SPF + DKIM + DMARC published and mail-tester passes (≥9/10)**
- [ ] **NFR-22: `bash deploy/mail/verify-isolation.sh` returns PASS**

---

**Last Updated**: June 28, 2026
**Domain**: https://chorus.talk
**Status**: 🚀 Ready for Deployment
