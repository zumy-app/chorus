/**
 * Acceptance suite runner (rescue plan phase B2).
 *
 * Usage:
 *   npm run acceptance            # auto-detect mode from known-failures.txt
 *   npm run acceptance:green      # require 100% pass (release gate mode)
 *   npm run acceptance:red        # verify failures match the red baseline
 *
 * The ratchet (known-failures.txt) makes red→green mechanical:
 *   - green mode: every TC must pass; file must be empty.
 *   - red mode:   TCs listed as known-failing MUST fail, all others MUST pass.
 *                 A listed TC that starts passing is a freeze-violation signal
 *                 (fix landed but baseline not updated) and fails the run.
 * Env:
 *   CHORUS_API                backend base URL (default http://localhost:8080)
 *   CHORUS_EXPECTED_COMMIT    git short hash the served /health commit must match
 *   CHORUS_ACCEPTANCE_SEED    "0" to skip auto-seeding (default: seeds before run)
 */
import { spawnSync } from 'node:child_process'
import { readFileSync, writeFileSync, existsSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { buildCtx, API_BASE, type Ctx, type TestCase } from './harness.js'
import { foundationTests } from './tests/p0-foundation.js'
import { featureTests } from './tests/p0-features.js'

const here = dirname(fileURLToPath(import.meta.url))
const KNOWN_FAILURES_FILE = join(here, 'known-failures.txt')

const allTests: TestCase[] = [...foundationTests, ...featureTests]

function readKnownFailures(): Set<string> {
  if (!existsSync(KNOWN_FAILURES_FILE)) return new Set()
  return new Set(
    readFileSync(KNOWN_FAILURES_FILE, 'utf8')
      .split(/\r?\n/)
      .map((l) => l.trim())
      .filter((l) => l && !l.startsWith('#')),
  )
}

function seedFixtures(): void {
  if (process.env.CHORUS_ACCEPTANCE_SEED === '0') return
  const backendDir = join(here, '..', '..', 'backend')
  console.log('[runner] seeding deterministic dev fixtures (go run ./cmd/server --seed-dev)...')
  const r = spawnSync('go', ['run', './cmd/server', '--seed-dev'], {
    cwd: backendDir,
    stdio: ['ignore', 'inherit', 'inherit'],
    shell: true,
  })
  if (r.status !== 0) {
    console.error('[runner] FATAL: fixture seeding failed — cannot run acceptance suite')
    process.exit(2)
  }
}

interface Result {
  id: string
  name: string
  reqs: string[]
  ok: boolean
  error?: string
  ms: number
}

async function runSuite(ctx: Ctx): Promise<Result[]> {
  const results: Result[] = []
  for (const tc of allTests) {
    const t0 = Date.now()
    try {
      await tc.fn(ctx)
      results.push({ id: tc.id, name: tc.name, reqs: tc.reqs, ok: true, ms: Date.now() - t0 })
      console.log(`  PASS  ${tc.id}  (${Date.now() - t0}ms)`)
    } catch (e: any) {
      results.push({ id: tc.id, name: tc.name, reqs: tc.reqs, ok: false, error: String(e?.message ?? e), ms: Date.now() - t0 })
      console.log(`  FAIL  ${tc.id}  (${Date.now() - t0}ms)\n        ${String(e?.message ?? e).split('\n')[0]}`)
    }
  }
  return results
}

// SPLIT_MARKER_MAIN
