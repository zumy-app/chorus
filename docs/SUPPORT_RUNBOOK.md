# Support Runbook + Dashboards — 14.4 (SRE)

> Authority: `REQUIREMENTS_MASTER.md` 14.4, `docs/GO_NO_GO.md` axis 9.
> Owner: **SRE / Raju (gosangiraju)**. Backup: batchu (product/tech).
> Last updated: 2026-08-31.

## 1. On-call

| Field | Value |
|---|---|
| Primary on-call | SRE Raju — Slack `#chorus-incidents`, PagerDuty `chorus-backend` |
| Secondary | batchu — escalate after 15m no-ack or P1 >30m |
| Rotation | Weekly Mon 09:00 UTC. Handover: open `GO_NO_GO.md` risks + Grafana screenshots |
| Escalation | P1 → page secondary + product lead. P2 → Slack #chorus-incidents. P3 → GitHub issue |

`ESCALATION.md` must be empty at release gate. Any open `crew/phase_status.json` blocker fails `verify-release-gate.sh`.

## 2. Service overview

```
Client (web/mobile) → LB (HAProxy :8080, Caddy dev) → backend :8080 (stateless, N replicas)
                              ↓
              Postgres (source of truth)  Redis (registry + pub/sub, never truth)  Phoenix (traces)
              Mailu (isolated network, submission 587 only)
```

Key invariants: **persist before ack** (Postgres `message_receipts`/`inbox` is truth), Redis `ws:registry:{userId}` is ephemeral (TTL 45s, heartbeat 15s), translation jobs are durable (`translation_jobs` + sweeper).

## 3. Dashboards

### 3.1 Grafana

- URL: prod `https://grafana.chorus.talk` (or `http://localhost:3001` local / `3006` dev). Login `admin/$GRAFANA_ADMIN_PASSWORD` (default `admin`).
- Datasource: Prometheus `chorus-prometheus` (15s scrape).
- Provisioning: `deploy/monitoring/grafana/provisioning/` → loads every `*.json` in `deploy/monitoring/grafana/dashboards/` (no manual import).

| Dashboard | File | Panels |
|---|---|---|
| **Chorus Backend** | `chorus-backend.json` (UID `chorus-backend`, default home) | 16 panels: req rate/latency p95, WS active/connections, messages, translation p95/tokens, Go runtime, heap, 5xx/4xx, word-mining jobs/duration/items/failure |
| **Chorus Support SLO** | `chorus-support.json` (UID `chorus-support`) | 8 panels: SLO error budget burn (5xx), availability (health/ready), inbox pending drain, retention purge lag, translation failure ratio, payout failure, DB/Redis exporter up, Phoenix export lag |

Both dashboards are provisioned read-only; `allowUiUpdates:true` lets on-call annotate.

### 3.2 Prometheus

- UI: `http://localhost:9090` (or `9092` dev). `http://backend:8080/metrics` / `backend-dev:8080/metrics`.
- Key metrics: `chorus_backend_http_requests_total`, `chorus_backend_http_request_duration_seconds`, `chorus_backend_ws_active_*`, `chorus_backend_messages_sent_total`, `chorus_backend_translation_*`, `chorus_backend_word_mining_*`, `go_goroutines`, `go_memstats_heap_inuse_bytes`.
- Verification:
```bash
curl -fsS http://localhost:8080/metrics | grep chorus_backend
curl -fsS http://localhost:8080/health          # 200 always (liveness)
curl -fsS http://localhost:8080/health/ready    # 200 only when pg+redis+translation ready
curl -fsS http://localhost:9090/-/healthy
```

### 3.3 Phoenix (tracing)

- UI: `http://localhost:6006` (dev `6007`). OTLP gRPC `4317` / `4319` dev.
- Use for translation/grammar lineage + FR-30 evals (`translation_evals`, `grammar_evals`, KPIs via `/admin/quality/kpis`).

## 4. Alerts

Configured in `deploy/monitoring/*.yml` and loaded via `prometheus.yml:rule_files`.

| Group | File | Alert | Expr | Sev | Action |
|---|---|---|---|---|---|
| soak-zero-loss | `alerts-soak.yml` | SoakMessageLoss | `increase(ws_fast_dropped_total[5m])>0` | critical | check hub buffer, scale backend, `verify-drain.sh` |
| soak-zero-loss | `alerts-soak.yml` | SoakWsErrors / InboxDrainStalled | active<900 / `pg_inbox_pending>0` 10m | warning | check LB/replicas, `SELECT count(*) FROM message_receipts WHERE received_at IS NULL` |
| word-mining | `alerts-word-mining.yml` | HighFailureRate >5% / StuckPending >100 / NoProgress / DurationHigh p95>10s | see file | warning | `verify-word-mining.sh`, `psql word_mining_jobs`, check provider keys |
| support-slo | `alerts-support.yml` | High5xxRate | `sum(rate(http_requests_total{status=~"5.."}[5m]))/sum(rate(http_requests_total[5m]))>0.01` 2m | critical | rollback candidate, tail `docker logs backend`, `go vet` |
| support-slo | `alerts-support.yml` | High4xxRate >5% 5m | warning | check rate limiter, auth, WAF |
| support-slo | `alerts-support.yml` | HealthNotReady / BackendDown | `health==0` / `up==0` | critical | `curl /health/ready`, `docker compose ps`, `pg_isready`, `redis-cli ping` |
| support-slo | `alerts-support.yml` | TranslationHighFailure >10% 5m | warning | `translation_jobs` status, `TRANSLATION_FALLBACK_ORDER`, provider keys |
| support-slo | `alerts-support.yml` | InboxPendingHigh >500 10m | warning | `verify-drain.sh`, sweeper logs |
| support-slo | `alerts-support.yml` | PayoutFailures | `increase(payout_failed[1h])>0` when wired | warning | check PayPal creds, `teacher_payouts` |
| support-slo | `alerts-support.yml` | DBDown / RedisDown | `pg_up==0` / `redis_up==0` | critical | `docker compose restart postgres/redis`, verify PVC |
| support-slo | `alerts-support.yml` | DiskSpaceLow <15% | via node exporter if present | warning | `df -h`, prune `uploads_data`/logs |

Prometheus evaluates every 15s. Wire Alertmanager or Grafana contact points to Slack `#chorus-incidents` (optional `SLACK_WEBHOOK_URL` in CI).

Verify alert syntax:
```bash
docker run --rm -v "$PWD/deploy/monitoring:/etc/prometheus" prom/prometheus:v3.1.0 promtool check rules /etc/prometheus/alerts-*.yml
bash deploy/ci/verify-support.sh --offline
```

## 5. SLOs

| SLO | Target | Measure |
|---|---|---|
| Availability | 99.9% / 30d | `sum(rate(http_requests_total{status!~"5.."}[5m]))/sum(rate(http_requests_total[5m]))` |
| Latency p95 (HTTP) | <500 ms | `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))` |
| Translation cache-hit p95 | <500 ms | phoenix-eval `P95_MS_MAX=500` |
| Translation accuracy (golden) | ≥80% | `translation_evals` via `phoenix-eval.sh` (`ACCURACY_THRESHOLD=80`, `MIN_EVALS=10`) |
| Message loss (soak) | 0 | `ws_fast_dropped_total==0` + inbox drain |
| Load smoke error | ≤2% | Artillery `ensure.errors.rate<=0.02` |

Burn rate: breach of High5xxRate 2m → page. Review weekly in `GO_NO_GO.md` evidence bundle (`/metrics` snapshot + Grafana screenshots).

## 6. Incident response

### 6.1 Severity

| Sev | Meaning | Response | Example |
|---|---|---|---|
| P1 | Data loss or full outage | Page primary+secondary, mitigate <15m, rollback if needed | Postgres down, all `/health/ready` 503, `ws_fast_dropped>0` |
| P2 | Degraded, single feature | Slack #chorus-incidents, mitigate <1h | Translation chain degraded, word-mining stuck, high 5xx burst |
| P3 | Warning, no user impact | Issue + next deploy | Duration p95 drift, pending <500, disk 80% |

### 6.2 Generic flow

1. **Ack** alert in Slack/PagerDuty, open incident thread.
2. **Triage** — check `curl /health/ready`, `curl /metrics`, Grafana Support+Backend dashboards, Prometheus Alerts, `docker compose ps && docker logs --tail 200 backend`.
3. **Mitigate** — use SOP below (rollback, restart, scale). Prefer mitigations that preserve durability (never truncate `message_receipts`/`translation_jobs`).
4. **Verify** — `curl /health/ready` 200, `verify-support.sh`, `verify-drain.sh` if WS/inbox, Grafana recovery.
5. **Post-mortem** — file issue, attach evidence bundle (gate logs, `/metrics` snapshot, Grafana screenshots, `docker buildx imagetools inspect :prod`).

## 7. SOPs

### 7.1 Health / readiness 503

```bash
curl -i http://localhost:8080/health/ready   # inspect {"checks":{"postgres":...,"redis":...,"translation":...}}
docker compose ps
docker compose logs --tail 200 backend postgres redis
pg_isready -h postgres -U messenger           # or: docker compose exec postgres pg_isready -U messenger
redis-cli -h redis ping                      # or: docker compose exec redis redis-cli ping
# Fix env/secrets, then:
docker compose up -d postgres redis backend && curl -fsS http://localhost:8080/health/ready
```

### 7.2 High 5xx / latency

```bash
curl -s http://localhost:8080/metrics | grep chorus_backend_http_requests_total
# Top failing route:
#   sum by (path) (rate(chorus_backend_http_requests_total{status=~"5.."}[5m]))
# p95:
#   histogram_quantile(0.95, sum by (le) (rate(chorus_backend_http_request_duration_seconds_bucket[5m])))
docker compose logs --tail 500 backend | grep -i "error\|panic"
go vet ./...   # must be green before redeploy
```

If cause is bad deploy → **rollback** (§7.6). If DB/Redis overload → scale backend replicas behind LB (stateless; `docker compose up -d --scale backend=3` where compose allows, or add LB backend entries).

### 7.3 WebSocket drops / ws_fast_dropped >0 (message loss risk)

```bash
curl -s http://localhost:8080/metrics | grep ws_fast_dropped
bash deploy/ci/verify-drain.sh
psql "$DATABASE_URL" -c "select count(*) from message_receipts where received_at is null"
# Durability invariant: Postgres is truth — pending rows are replayable via GET /inbox/pending → POST /inbox/ack
# Mitigate: scale backends, increase ws write buffer (code), check LB leastconn config
```

### 7.4 Translation chain failure

```bash
curl -s http://localhost:8080/metrics | grep translation_requests_total
psql "$DATABASE_URL" -c "select status,count(*) from translation_jobs group by status"
# Check provider keys:
grep TRANSLATION_FALLBACK_ORDER backend/.env
curl -fsS http://localhost:8080/admin/translations/health -H "Authorization: Bearer <admin-token>"  # moderator
bash deploy/ci/phoenix-eval.sh  # accuracy/p95 gate
```

Fallback chain auto-skips misconfigured providers. Last resort is local LibreTranslate/Ollama — ensure `ollama`/`libretranslate` containers healthy.

### 7.5 Word-mining stuck

```bash
bash deploy/ci/verify-word-mining.sh
psql "$DATABASE_URL" -c "select status,count(*) from word_mining_jobs group by status"
curl -s http://localhost:8080/metrics | grep word_mining
# Sweeper requeues stale: 15s→5m backoff, 24h at max attempts. Check logs:
docker compose logs --tail 300 backend | grep -i "word_mining"
```

### 7.6 Rollback (image promotion reversal)

Prev image is the last green `:prod` is a digest-preserving re-tag (no rebuild). Rollback re-promotes the prior tag.

```bash
# Discover last green tag (GitHub run number or docker inspect):
docker buildx imagetools inspect ghcr.io/zumy-app/chorus/backend:prod --raw | sha256sum
# Rollback (local or CI):
CR_REGISTRY=ghcr.io IMAGE_NAME=ghcr.io/zumy-app/chorus CR_USERNAME=... CR_PASSWORD=... PREV_TAG=<last-good-run-number> bash deploy/ci/rollback.sh
# If PROD_HOST/PROD_USER set, rollback.sh also pulls + redeploys prod:
PREV_TAG=123 PROD_HOST=prod.example.com PROD_USER=deploy bash deploy/ci/rollback.sh
# Verify:
curl -fsS https://chorus.talk/health && curl -fsS https://chorus.talk/health/ready
bash deploy/ci/verify-promotion.sh   # checks :prod digest + prod /health
bash deploy/ci/verify-release-gate.sh --offline
```

On Dokploy: alternatively use dashboard → service → Redeploy with prior image tag.

### 7.7 DB restore (point-in-time)

```bash
# Backup (cron on host):
pg_dump -Fc -h postgres -U messenger messenger_dev > /backups/chorus-$(date +%F).dump
# Restore (downtime window):
docker compose stop backend
pg_restore -h postgres -U messenger -d messenger_dev --clean --if-exists /backups/chorus-YYYY-MM-DD.dump
docker compose up -d backend && curl -fsS http://localhost:8080/health/ready
# Verify invariants after restore:
psql "$DATABASE_URL" -c "select count(*) from messages" -c "select count(*) from message_receipts where received_at is null"
bash deploy/ci/verify-drain.sh
```

Retention windows (inbox 30d, translation 90d, messages per `user_settings.message_retention_days`) are enforced by `RetentionService` (`POST /admin/retention/purge`). Do not manually delete `word_mining_jobs`/`translation_jobs` — they are the durable outbox.

### 7.8 GDPR operations (on-call may be asked to assist)

```bash
# Export (user right to access):
curl -fsS -H "Authorization: Bearer <user-token>" https://chorus.talk/api/v1/users/me/export | jq .
# Erasure (anonymize + soft-delete):
curl -X DELETE -H "Authorization: Bearer <user-token>" https://chorus.talk/api/v1/users/me
# Retention policy (public):
curl -fsS https://chorus.talk/api/v1/privacy/retention-policy | jq .
```

### 7.9 Mail isolation check

```bash
bash deploy/mail/verify-isolation.sh   # FAIL if Mailu shares app network or VITE_* SMTP leaks
# Isolated compose: deploy/mail/docker-compose.mail.yml on its own bridge; only 587 from app subnet
```

## 8. Runbooks index (quick links)

- Verify everything offline: `bash deploy/ci/verify-support.sh --offline` (docs + dashboards + prometheus syntax + compose)
- Full gate suite: `bash deploy/ci/gate-on-dev.sh` (seed→e2e→phoenix-eval→load-smoke→release-gate→mail-isolation) — on dev host.
- Release gate: `bash deploy/ci/verify-release-gate.sh --offline` (docs + GO_NO_GO sign-off + phase_status 12.1-14.2 DONE + compose/branch)
- Word mining: `bash deploy/ci/verify-word-mining.sh` (+ `STRICT=1` to fail on >20 pending)
- Drain: `bash deploy/ci/verify-drain.sh` (ws_fast_dropped==0 + inbox empty)
- Teacher vetting: `bash deploy/ci/verify-teacher-vetting.sh`
- Mail isolation: `bash deploy/mail/verify-isolation.sh`

## 9. Contacts & escalation

- SRE Raju (gosangiraju) — infra/alerts/LB/Redis registry/soak — axes 5,7,12,13
- batchu — product/tech, translation/grammar/marketplace/payouts — axes 1,2,4,8,10,11
- Kushagra1122 — mobile — axis 3
- Daniella — learning/teacher vetting — axis 10 vetting
- All leads sign `docs/GO_NO_GO.md` §3 before any `prod` promotion. One `NO-GO` blocks promotion.

## 10. Evidence bundle (attach to release tag)

Per `docs/GO_NO_GO.md` §5 + `docs/RELEASE_GATE.md` §7: `gate-on-dev.sh` log, `verify-isolation.sh` + `verify-teacher-vetting.sh` logs, `/metrics` snapshot, Grafana screenshots (Backend + Support), `docker buildx imagetools inspect ...:prod`, signed `GO_NO_GO.md` §3.

## 11. Change log

- 2026-08-31 — 14.4 initial runbook + Support SLO dashboard + `alerts-support.yml`, wired into Prometheus `rule_files` and `verify-support.sh`.
