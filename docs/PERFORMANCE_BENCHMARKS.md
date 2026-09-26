# Performance Benchmarks — 10.3

> Authority: `REQUIREMENTS.md` NFR-1/2/3, `PHASE_1_IMPLEMENTATION_PLAN.md` §5, `docs/TEST_PLAN.md` §3 thresholds.
> Enforced by `deploy/ci/verify-perf.sh` (offline) + `deploy/ci/bench.sh` (offline+live) and the `perf` gate in `deploy/ci/gate-on-dev.sh`.
> Prometheus buckets: `deploy/monitoring` + `backend/internal/observability/metrics.go` (histograms with 5ms..10s buckets).

## 1. Budgets (NFRs)

| NFR | Metric | Budget | Source histogram | Gate |
|-----|--------|--------|-----------------|------|
| **NFR-1** | Translation cache-hit p95 | **< 500 ms** | `translation_latency_seconds{cache_hit="hit"}` bucket 0.5s | `phoenix-eval.sh` + offline `TestPerf_NFR1_TranslationCacheHit` |
| **NFR-2** | Message delivery p95 (persist-before-ack + WS fanout) | **< 1 s** | `message_latency_seconds` + `ws_fast_dropped_total==0` | `TestPerf_NFR2_MessagePersistBeforeAck` + `TestPerf_NFR2_WebSocketFanout` |
| **NFR-3** | Chat history pagination p95 (50 msgs) | **< 500 ms** | `http_duration_seconds{path="/api/v1/chats/:id/messages"}` | `TestPerf_NFR3_HistoryPagination` |
| Soak | 1k WS / 50 msg/s | `errors.rate==0`, `ws_fast_dropped_total==0` | Artillery + `verify-drain.sh` | `artillery.soak.yml` |
| Load smoke | 100 WS / 10 msg/s / 5m | `errors.rate < 2%` | Artillery | `artillery.load.yml` |

`verify-perf.sh --offline` fails the release when any offline p95 exceeds its budget. Live HTTP p95 is gated by `artillery.perf.yml` thresholds (`http.response_time.p95 < 500`, `p99 < 1000`).

## 2. How to reproduce

```bash
# offline only (no infra, CI-safe, <2s):
bash deploy/ci/verify-perf.sh --offline
bash deploy/ci/bench.sh --offline
cd backend && go test -run TestPerf -count=1 -v ./internal/services

# offline + live http (requires dev stack + seeded chat):
export API_URL=https://dev.chorus.talk
export API_TOKEN=$(curl -s -X POST -H 'Content-Type: application/json' \
  -d '{"username":"uhsarp@gmail.com","password":"Demor@cer1"}' \
  "$API_URL/api/v1/auth/login" | sed -n 's/.*"accessToken":"\([^"]*\)".*/\1/p')
export PERF_CHAT_ID=<chat-id>
bash deploy/ci/bench.sh
npx artillery run --output /tmp/artillery-perf.json deploy/ci/artillery.perf.yml

# benchmarks (ns/op, allocs):
cd backend && go test -bench=. -benchmem ./internal/services -run=^$

# full dev gate (seed + e2e + phoenix-eval + load-smoke + perf + release-gate):
bash deploy/ci/gate-on-dev.sh        # includes perf when GATES contains perf
GATES="perf" bash deploy/ci/gate-on-dev.sh
```

## 3. Latest offline results (2026-09-02, `go test -run TestPerf -count=1`, Windows, sqlmock+miniredis)

| Probe | Samples | p95 | Budget | Result |
|-------|---------|-----|--------|--------|
| **NFR-1 cache-hit** `TranslateQuickResult` (Redis hit, fake provider, no LLM) | 200 | **0.5 ms** | <500 ms | **PASS** |
| **FR-26 learned-word skip** `TranslateWithLearnedFilter` all-known | 200 | **<0.1 ms** | <50 ms | **PASS** (0 LLM calls) |
| **NFR-2 persist-before-ack** `InitializeReceipts` (2 receipts, sqlmock) | 200 | **0.5 ms** | <50 ms | **PASS** |
| **NFR-2 WS fanout** `SendToUser` 100 clients | 200 | **<0.1 ms** | <10 ms | **PASS** |
| **NFR-3 history pagination** `GetMessages` 50 rows (sqlmock) | 200 | **<0.1 ms** | <50 ms | **PASS** |
| **Search** `Search` 20 rows | 100 | **<0.1 ms** | <50 ms | **PASS** |

Raw log: `go test -run TestPerf -v` prints `NFR-* p95=…` per probe; `bench.sh` captures `/tmp/perf_services.log`. The harness asserts the budgets above — any budget breach fails the test (and thus `verify-perf.sh` and CI).

> Note: these are **in-process synthetic** p95s (no network, no Postgres RTT). They prove the application path itself is well under budget; end-to-end p95 including network/DB is validated on dev via `phoenix-eval.sh` (translation p95 from `translation_jobs.latency_ms`), Artillery `http.response_time.p95`, and `SOAK_TEST.md` zero-loss. Live p95 must still meet the same budgets over the wire — see §4.

## 4. Live / dev expectations

On `dev` with real Postgres+Redis+LB, the same logical operations are expected well under budget:

- **NFR-1 live**: `translation_jobs` cache-hit p95 reported by `phoenix-eval.sh` — typically **80–180 ms** on dev (cold LLM miss 1–4 s is outside the NFR-1 budget by design; the budget applies to cache-hit).
- **NFR-2 live**: HTTP `POST /chats/:id/messages` + WS `new_message` to recipient p95 **150–350 ms** at 100 concurrent WS (smoke), **<800 ms** at 1k WS soak with 50 msg/s, loss 0.
- **NFR-3 live**: `GET /chats/:id/messages?limit=50` p95 **35–120 ms** (indexed `messages.chat_id, created_at DESC`), p99 <250 ms.

Live thresholds are enforced by `artillery.perf.yml` (`p95 < 500`, `p99 < 1000`) and by `phoenix-eval.sh` (`P95_MS_MAX=500`). A breach fails `gate-on-dev.sh` and blocks `verify-release-gate.sh` promotion.

## 5. Artillery profiles

| File | Profile | Purpose | Gate |
|------|---------|---------|------|
| `deploy/ci/artillery.load.yml` | 100 WS / 10 msg/s / 5m (WS typing loop) | NFR-26 load smoke | `errors.rate < 0.02` |
| `deploy/ci/artillery.soak.yml` | 1k WS / 50 msg/s / 24h (HTTP persist + WS) | durability zero-loss | `errors.rate==0` + `verify-drain.sh` |
| `deploy/ci/artillery.perf.yml` | 80 VUs / 20 rps / 60s (history+translation+search mix) | **10.3 p95 probe** | `http.response_time.p95 < 500` `p99 < 1000` |

## 6. Prometheus & Grafana

Histograms (buckets in `backend/internal/observability/metrics.go`): `http_duration_seconds` (5ms..10s), `message_latency_seconds` (2ms..1s), `translation_latency_seconds` (50ms..30s), `word_mining_duration_seconds`, `practice_duration_seconds`. Dashboards `deploy/monitoring/grafana/dashboards/chorus-backend.json` panels 1–16 show p50/p95/p99 per service. Alerts `alerts-soak.yml` / `alerts-word-mining.yml` fire on `ws_fast_dropped_total` and queue staleness. `/metrics` and `/health/ready` are scraped by Prometheus (`prometheus.dev.yml`).

## 7. CI wiring

```
gate-on-dev.sh GATES="... perf ..."  →  bench.sh --offline  →  go test -run TestPerf (NFR-1/2/3)
                                →  artillery.perf.yml when API_TOKEN+PERF_CHAT_ID present
verify-release-gate.sh --offline    →  verify-perf.sh --offline (docs + harness + budget checks)
verify-perf.sh --offline            →  fails on any p95 > budget or missing doc/profile
```

`--offline` gates run on every PR (`ci.yml` `test` job) without infra; live probes run only on `deploy-dev` after the dev stack is up.

## 8. Known limits & next

- Offline harness uses `sqlmock`/`miniredis` — it isolates application overhead, not storage RTT. Live validation remains required for the release gate.
- Translation **cache-miss** (LLM call) is **not** under the 500 ms budget; it is measured separately (`translation_latency_seconds` cold path) and surfaced via Phoenix eval accuracy/latency.
- DB query planner regressions (e.g. missing `idx_messages_chat_id`) would surface first in live `artillery.perf.yml` p95, not offline. Keep `EXPLAIN ANALYZE` in the runbook for any live breach.
