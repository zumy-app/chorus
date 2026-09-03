# Depth-of-processing practice ladder (11.2) — SRE

Ladder: `1 recognition → 2 cued recall → 3 free recall → 4 production → 5 spontaneous`.
Mastery gated on production: `mastered` only when `mastery_stage >=4` and (`production_success_count >=2` or `production>=1 && spontaneous>=1`). Stage 5 via `TouchSpontaneousUse` when chat/scenario reuse detected. Leech when `lapses>=3 && stage<=2`. SRS intervals fixed: new 1d, s1 2d, s2 3d, s3 5d, s4 7d→ease*, s5 14d→ease*.

Service: `backend/internal/services/practice.go` (`PracticeService.UpdateVocabAfterAttempt`, `TouchSpontaneousUse`, `ReviewCard`). DB: `vocabulary.mastery_stage/state`, `stage_success_count`, `production_success_count`, `spontaneous_use_count`, `vocabulary_practice_attempts`.

Metrics (`/metrics`): `chorus_backend_practice_attempts_total{stage,outcome}`, `chorus_backend_practice_duration_seconds{stage}`, `chorus_backend_practice_stage_promotions_total{from_stage,to_stage}`, `chorus_backend_practice_leech_total`, `chorus_backend_practice_spontaneous_total`, `chorus_backend_practice_mastered_total`, `chorus_backend_practice_due_cards`.

Dash: Grafana `Chorus Backend` panels 17-20 (attempts by stage, promotions, leech/mastered/spontaneous, p95+due).

Alerts: `deploy/monitoring/alerts-practice-ladder.yml` (LeechRate, NoProgress, LowCorrect, HighError, DurationHigh). Rule_files in `prometheus.yml`/`prometheus.dev.yml`.

Verify: `deploy/ci/verify-practice-ladder.sh` (DB mastered/leech/interval checks + metrics presence) and `curl localhost:8080/metrics | grep practice`.
