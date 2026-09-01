# Word-mining pipeline (11.1) — SRE

Pipeline: `SendMessage` → `WordMiningQueue.EnqueueForMessage` (DB outbox `word_mining_jobs`) → Redis `wordmining:jobs` pub/sub → workers (`WordMiningService.ProcessJob` extract/classify/route/dedupe → `mined_items` + `vocabulary`) → WS `word_mining_completed`.

Durability: `word_mining_jobs` is source of truth (persist before trigger). Redis never truth. Sweeper requeues stale/failed with exp backoff (15s→5m, 24h at max attempts). `recover()` on boot re-queues pending/failed. Deduped by `UNIQUE(user_id,message_id)`.

Metrics (per `/metrics`): `chorus_backend_word_mining_jobs_total{status}`, `chorus_backend_word_mining_duration_seconds{status}`, `chorus_backend_word_mining_items_total{status,route}`, `chorus_backend_word_mining_pending_jobs`.

Dash: Grafana `Chorus Backend` panels 13-16 (jobs, duration p95, items by route, failure ratio).

Alerts: `deploy/monitoring/alerts-word-mining.yml` (HighFailureRate >5%, StuckPending >100/5m, NoProgress, DurationHigh p95>10s). Wire via `rule_files` in Prometheus.

Verify: `psql -c "select status,count(*) from word_mining_jobs group by status"` should drain to `pending=0` after sweeper. Zero loss: every chat message matching learner's `target_language` yields a job (unless 750-char limit). Check `curl -s localhost:8080/metrics | grep word_mining` and alertmanager.
