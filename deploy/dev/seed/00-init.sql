-- Chorus dev database — first-boot init (runs once on an empty volume).
-- Kept idempotent: it only creates extensions the backend relies on, and it is
-- safe to re-run. Schema + demo users are NOT created here — the backend runs
-- migrations on start (backend/internal/database/postgres.go), and the `seed`
-- CI gate registers the dev demo users through the API (deploy/ci/seed-dev.sh).

CREATE EXTENSION IF NOT EXISTS pgcrypto;
