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
import { sessionTests } from './tests/p0-session.js'
import { safetyTests } from './tests/p0-safety.js'
import { messagingTests } from './tests/p0-messaging.js'

const here = dirname(fileURLToPath(import.meta.url))
const KNOWN_FAILURES_FILE = join(here, 'known-failures.txt')

const allTests: TestCase[] = [
  ...foundationTests,
  ...featureTests,
  ...sessionTests,
  ...safetyTests,
  ...messagingTests,
]

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

// ---------------------------------------------------------------------------
// Entry point: seed → buildCtx → runSuite → ratchet → exit code.
// Modes: green (release gate: all must pass), red (verify listed failures
// fail and nothing else does), auto (default: fail on unlisted failures and
// on listed-but-passing TCs so the baseline can only move deliberately).
// ---------------------------------------------------------------------------
async function main(): Promise<void> {
  const argv = process.argv.slice(2);
  const modeIdx = argv.indexOf('--mode');
  const modeValue = modeIdx >= 0 ? argv[modeIdx + 1] : undefined;
  const mode: 'green' | 'red' | 'auto' =
    modeValue === 'green' || argv.includes('green') || argv.includes('--green')
      ? 'green'
      : modeValue === 'red' || argv.includes('red') || argv.includes('--red')
        ? 'red'
        : 'auto';

  seedFixtures();

  const expectedCommit = process.env.CHORUS_EXPECTED_COMMIT;
  if (expectedCommit) {
    const health = await (async () => {
      try {
        const r = await fetch(`${API_BASE}/health`);
        return (await r.json()) as any;
      } catch {
        return null;
      }
    })();
    if (!health || health.commit !== expectedCommit) {
      console.error(
        `[runner] FATAL: served commit ${health?.commit ?? '<unreachable>'} != expected ${expectedCommit}`,
      );
      process.exit(2);
    }
  }

  console.log(`[runner] mode=${mode} suites=${allTests.length}`);
  const ctx = await buildCtx();
  const results = await runSuite(ctx);
  const known = readKnownFailures();

  const failed = results.filter((r) => !r.ok);
  const passedIds = new Set(results.filter((r) => r.ok).map((r) => r.id));
  let exit = 0;

  if (mode === 'green') {
    if (known.size > 0) {
      console.error(`[runner] green mode requires an empty ${KNOWN_FAILURES_FILE} (${known.size} listed)`);
      exit = 1;
    }
    if (failed.length > 0) {
      console.error(`[runner] green mode: ${failed.length} failing TC(s): ${failed.map((r) => r.id).join(', ')}`);
      exit = 1;
    }
  } else {
    // red + auto share the ratchet: listed TCs MUST fail; unlisted MUST pass.
    const unexpectedPasses = [...known].filter((id) => passedIds.has(id));
    const unlistedFailures = failed.filter((r) => !known.has(r.id));
    if (mode === 'red') {
      const listedIds = new Set(allTests.map((t) => t.id));
      for (const id of known) {
        if (!listedIds.has(id)) {
          console.error(`[runner] known-failure ${id} matches no TC — stale baseline`);
          exit = 1;
        }
      }
    }
    if (unexpectedPasses.length > 0) {
      console.error(
        `[runner] freeze violation: ${unexpectedPasses.length} listed TC(s) now PASS — update ${KNOWN_FAILURES_FILE}: ${unexpectedPasses.join(', ')}`,
      );
      exit = 1;
    }
    if (unlistedFailures.length > 0) {
      console.error(
        `[runner] ${unlistedFailures.length} unlisted failure(s): ${unlistedFailures.map((r) => r.id).join(', ')}`,
      );
      exit = 1;
    }
  }

  const passed = results.length - failed.length;
  console.log(`[runner] ${passed}/${results.length} passed (mode=${mode})`);
  if (exit === 0) console.log('[runner] ACCEPTANCE GREEN');
  process.exit(exit);
}

main().catch((e) => {
  console.error(`[runner] FATAL: ${e?.message ?? e}`);
  process.exit(2);
});
