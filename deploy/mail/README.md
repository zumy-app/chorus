# Chorus Mail — Isolated Mailu (NFR-22)

This directory holds the **isolated Mailu mail-server** deployment for Chorus.

NFR-22 (Security — mail server isolation) states: *Prod mail (Mailu) must not
share the same host/network namespace as the app servers.* This is the
verifiable, repeatable way to satisfy that requirement — a separate host, its
own Docker network, minimal ingress (submission `587` only from the app), and
envelope-auth (SPF/DKIM/DMARC) per Issue #3.

## Files

| File | Purpose |
|------|---------|
| `docker-compose.mail.yml` | Mailu stack, isolated on its own `mailu` bridge network. |
| `mailu.env.example` | Mailu configuration — **copy to `mailu.env`, fill in, never commit**. |
| `verify-isolation.sh` | Release-gate network-policy check (run in CI / pre-deploy). |

> **Never** deploy this compose on the same node that runs
> `docker-compose.prod.yml`.

---

## 1. Isolation topology

```
                       ┌───────────────────────────┐
  Internet ──┬─►        │   MAIL HOST (separate)    │
             │          │  deploying this compose   │
             │          │   front : 587  (submission)│
             │          │   front : 25   (ingest)    │
             │          │   mailu internal network   │
             │          └───────────────────────────┘
             │                        ▲
             │   only SMTP 587        │ private network / VPN /
             │   (authenticated)     │ restricted interface
             ▼                        │
  ┌───────────────────────────┐       │
  │  APP HOST (Dokploy)        │───────┘
  │   chorus-backend services  │
  │   docker-compose.prod.yml  │
  └───────────────────────────┘
```

**Recommended: separate host.** A separate VPS for Mailu gives a true
network-namespace/host boundary. The app reaches it over the mail host's
private interface (or a restricted public IP + source firewall) on **587 only**.

**Alternative (single node, isolated network):** if you must run both on one
node, create a dedicated network and do **not** attach `chorus-network` to the
Mailu stack. The app's `backend` still connects to
`MAILU_SMTP_HOST:<587>`; the bridge network + firewall between them is the
isolation boundary. Verify it with `verify-isolation.sh`.

---

## 2. Host firewall (minimal ingress / NFR-22)

On the **mail host** (using `ufw` as an example):

```bash
# Only the Chorus APP SUBNET may initiate SMTP submission (587).
ufw allow from <APP_SUBNET>/32 to any port 587 proto tcp

# Block inbound SMTP (25) from the public for a relay-only mailbox;
# or restrict receipt to major sender ASNs if you keep inbound mail.
ufw deny 25/tcp           # relay-only: no inbound mail is accepted

# Everything else for the Chorus app is NOT exposed to the app subnet.
ufw deny from <APP_SUBNET>/32 to any port 465 proto tcp
ufw deny from <APP_SUBNET>/32 to any port 80,443,993,995,143,110 proto tcp

# Webmail/admin UI (80/443) reaches the world only for YOUR admin IP.
ufw allow from <YOUR_ADMIN_IP>/32 to any port 80,443 proto tcp
```

Verify no app service port colon-maps were introduced (automated):

```bash
bash deploy/mail/verify-isolation.sh
```

---

## 3. Envelope authentication: SPF / DKIM / DMARC (Issue #3)

Mailu auto-generates the DKIM key on first boot
(`mail_data/dkim/{domain}.dkim.{selector}.key`). Add these DNS records at your
domain provider.

### SPF (TXT `@`)

```
v=spf1 a mx a:mail.your-domain.com -all
```

### DKIM (TXT `mail._domainkey`)

On the mail host, print the public key:

```bash
docker exec -it <resolver> ...   # or read mail_data/dkim/<domain>.dkim.<selector>.key.pub
```

Publish as a TXT record at `<selector>._domainkey.your-domain.com`:

```
v=DKIM1; k=rsa; p=<base64 public key from Mailu>
```

### DMARC (TXT `_dmarc`)

```
v=DMARC1; p=quarantine; rua=mailto:postmaster@your-domain.com; fo=1;
```

Start `p=none` → monitor reports → move to `quarantine` → then `reject` once
mail-tester is clean.

---

## 4. Secret rotation & scoping (no `VITE_` prefix)

NFR-22: *Rotate `SMTP_PASSWORD` (was in `.env` history), move to Dokploy
secrets, scope SMTP creds to env (no `VITE_` prefix).*

1. **Rotate**: change the Mailu mailbox password (admin UI or
   `mailu.env`). The prior value was in repo history, so treat it as
   compromised.
2. **Store in Dokploy secrets**: add `MAILU_SMTP_PASSWORD` as a **secret**
   (never as a plain env var) in the Chorus project. `docker-compose.prod.yml`
   already forwards `MAILU_SMTP_*` to the `backend` container; the real value is
   injected from Dokploy, not the repo.
3. **Scope**: these are server-side only. They must **never** be prefixed
   `VITE_` — `VITE_*` vars are bundled into the browser bundle and are
   public. Confirmed by `verify-isolation.sh` (no `VITE_*(SMTP|MAIL|MAILU)`).
4. **App config** (Dokploy env): set `MAILU_SMTP_PORT=587` (submission /
   STARTTLS) so the app only needs the submission port open.

---

## 5. Release-gate check

Add to CI / the promotion gate (before `dev`→`prod`):

```yaml
- name: NFR-22 mail isolation check
  run: bash deploy/mail/verify-isolation.sh
```

The script exits non-zero (blocks the gate) on any of:

- Mailu inside the app compose, or Mailu joining `chorus-network`.
- Any app service publishing a mail port (25/465/587/993/995/143/110).
- `VITE_*` SMTP/mail secret exposure.
- A committed, non-empty `MAILU_SMTP_PASSWORD` (real secret in repo).

---

## 6. Verify delivery (mail-tester)

1. Send a test from the Chorus mailbox (backend live test:
   `MAILU_SMTP_LIVE_TEST=1 go test ./internal/services -run Mailu -v`).
2. Open <https://www.mail-tester.com>, copy the address, send to it.
3. Score should be **≥ 9/10** — SPF + DKIM + DMARC all pass.

---

## 7. Notes

- `front` binds `587` to `0.0.0.0` and `25/465/443` to `127.0.0.1`; set
  `BIND_ADDRESS` in `mailu.env` / the compose to a private interface for a
  stricter footprint.
- Keep `mailu.env` and `mail_data/` off the app host and out of git.
