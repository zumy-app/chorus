# WORKING_SET — Scope Fence for the Autonomous Pipeline

> This file bounds what opencode agents may touch. It prevents the loop from refactoring
> unrelated subsystems, deleting secrets, or drifting outside the phase. The supervisor reads
> it before every delegation. Only paths under the current phase's `ALLOWED` list may be
> modified; the rest is READ-ONLY (consult but do not alter).

## Rules (read by every agent, appended to each task)
1. **READ-ONLY zones:** files that are "edit" targets must be under an `ALLOWED` path for the
   active phase. Anything else is read-only context.
2. **Never** modify: `.git/`, `.env`, `.env.*`, `node_modules/` (except package lock via installers),
   `venv*/`, `.venv-agents/`, `agent_jobs/`, `crew/`, `tools/`, `data/`, any `*.log`, `*.csv`,
   `token*`, `tmp_*`, `*.apk`, secrets.
3. **Everything you do** must be verifiable: after editing, run the relevant build/test command
   and report exit code.
4. **No data migration** runs against `prod` unless the phase explicitly says so.
5. **Preserve existing functionality** — do not break green tests to add a feature.
6. Report files changed + commands run + exit code at the end of every task.

## Phase → ALLOWED paths

### PHASE 0 (Green baseline)
- ALLOWED: `backend/**/*_test.go`, `backend/go.mod`, `frontend/src/**`, `mobile/src/**`,
  `mobile/jest.config.js`, `mobile/jest.setup.js`, `docs/**`, `RUN_GUIDE.md`,
  `docker-compose.dev.yml`, `scripts/**`
- READ-ONLY: everything else.

### PHASE 1 (Launch-blocking core)
- ALLOWED: `backend/internal/handlers/**`, `backend/internal/services/**`,
  `backend/internal/middleware/**`, `backend/internal/database/postgres.go`,
  `backend/internal/models/**`, `backend/cmd/server/main.go`, `frontend/src/**`,
  `mobile/src/**`, `packages/shared/src/**`, `docker-compose.prod.yml`, `deploy/**`,
  `.github/workflows/ci.yml`, `docs/**`

### PHASE 2 (Messaging parity + audio captions)
- ALLOWED: `backend/internal/handlers/**`, `backend/internal/services/**`,
  `backend/internal/database/postgres.go`, `backend/internal/models/**`,
  `backend/cmd/server/main.go`, `frontend/src/**`, `mobile/src/**`, `packages/shared/src/**`,
  `e2e/**`, `docs/**`, `docker-compose.yml`, `docker-compose.prod.yml`

### PHASE 3 (Video + scaled arch)
- ALLOWED: `backend/internal/handlers/**`, `backend/internal/services/**`,
  `backend/internal/database/postgres.go`, `backend/internal/models/**`,
  `backend/cmd/server/main.go`, `deploy/**`, `docker-compose.yml`, `docker-compose.prod.yml`,
  `nginx.conf`, `frontend/src/**`, `mobile/src/**`, `packages/shared/src/**`,
  `e2e/load/**` (new), `docs/**`

### PHASE 4 (Marketplace + monetization + release)
- ALLOWED: `backend/internal/handlers/**`, `backend/internal/services/**`,
  `backend/internal/database/postgres.go`, `backend/internal/models/**`, `frontend/src/**`,
  `mobile/src/**`, `packages/shared/src/**`, `deploy/**`, `.github/workflows/**`,
  `docker-compose.prod.yml`, `docs/**`
