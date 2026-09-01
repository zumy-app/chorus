# Soak test — zero loss (10.6)

**Profile:** 1k concurrent WS, 50 msg/s, 24h drain, message loss = 0.

Artifacts: `deploy/ci/artillery.soak.yml` (Artillery), `deploy/ci/soak-processor.js`, `deploy/ci/load-soak.sh` (runner + DB drain), `deploy/ci/verify-drain.sh` (post-soak invariant), `backend/internal/services/soak_test.go` (unit soak).

## Invariant

Postgres is source of truth. `persist before ack` — `message_receipts` row with `received_at IS NULL` is the durable inbox. Redis `ws:registry` is never truth. Loss = 0 means every `messages` row has a receipt and every receipt eventually reaches `received_at` or is replayable via `GET /inbox/pending` → `POST /inbox/ack`.

## Run

```bash
# short (CI, ~60s, 1k WS / 50 msg/s clamp):
SOAK_DURATION=60 SOAK_LOOPS=5 SOAK_HTTP_MSGS=20 bash deploy/ci/load-soak.sh

# full 24h (on dev host, needs SOAK_ALLOW_LONG=1):
SOAK_ALLOW_LONG=1 SOAK_DURATION=86400 bash deploy/ci/load-soak.sh

# drain only:
bash deploy/ci/verify-drain.sh

# via gate orchestrator:
GATES="load-soak verify-drain" bash deploy/ci/gate-on-dev.sh
```

## Gate

Artillery `ensure.thresholds`: `errors.rate == 0`. Runner also checks `psql` pending counts and `ws_fast_dropped_total == 0` via `/metrics`. Any `>0` fails the gate.

## Prometheus

Alert `SoakMessageLoss` fires on `increase(ws_fast_dropped_total[5m]) > 0`. See `deploy/monitoring/alerts-soak.yml`.
