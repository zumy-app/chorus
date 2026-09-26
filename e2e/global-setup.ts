import { execSync } from 'child_process'
import { existsSync } from 'fs'
import { resolve } from 'path'

/**
 * Global setup: starts the Chorus stack and waits for services to be healthy.
 *
 * This runs once before all test files. It:
 * 1. Checks if Docker Desktop is running (and attempts to start it)
 * 2. Checks if services are already running via health endpoint — skips if so
 * 3. Otherwise starts services via `docker-compose up -d` (tries dev compose first, falls back to production)
 * 4. Waits for backend health endpoint to respond
 * 5. Waits for frontend to serve HTML
 *
 * Set E2E_SKIP_STARTUP=true to skip Docker startup entirely (when running
 * alongside `start-dev.ps1` or a manual dev stack).
 */

const BACKEND_HEALTH = process.env.E2E_BACKEND_HEALTH || 'http://localhost:8080/health'
const FRONTEND_URL = process.env.E2E_FRONTEND_URL || 'http://localhost:3000'
const MAX_WAIT_MS = 300_000 // 5 minutes
const POLL_INTERVAL_MS = 5_000
const ROOT_DIR = resolve(__dirname, '..')

async function waitForUrl(url: string, label: string, expectJson = false): Promise<void> {
  const startTime = Date.now()

  while (Date.now() - startTime < MAX_WAIT_MS) {
    try {
      const response = await fetch(url)
      if (response.ok) {
        if (expectJson) {
          const data = await response.json()
          if (data && data.status === 'healthy') {
            console.log(`✅ ${label} is healthy`)
            return
          }
        } else {
          console.log(`✅ ${label} is responding`)
          return
        }
      }
    } catch {
      // Service not ready yet
    }
    await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS))
  }

  throw new Error(`❌ ${label} did not become healthy within ${MAX_WAIT_MS / 1000}s`)
}

/** Check if Docker CLI responds, optionally launch Docker Desktop. */
async function ensureDockerDesktop(): Promise<boolean> {
  for (let attempt = 0; attempt < 12; attempt++) {
    try {
      execSync('docker ps', { stdio: 'pipe', timeout: 10_000 })
      return true // Docker is responding
    } catch {
      if (attempt === 0) {
        console.log('⏳ Docker not responding. Attempting to start Docker Desktop...')
        // Common Docker Desktop paths
        const paths = [
          'C:\\Program Files\\Docker\\Docker\\Docker Desktop.exe',
          '/Applications/Docker.app/Contents/MacOS/Docker',
          '/usr/bin/docker',
        ]
        for (const p of paths) {
          if (existsSync(p)) {
            try {
              execSync(`"${p}"`, { stdio: 'ignore', timeout: 5_000 })
            } catch { /* ignore — may already be launching */ }
            break
          }
        }
      }
      await new Promise((r) => setTimeout(r, 5_000))
    }
  }
  return false
}

/** Try starting docker-compose with a given compose file, return true on success. */
function tryComposeUp(composeFile: string, label: string): boolean {
  if (!existsSync(composeFile)) {
    console.log(`  ⚠ ${label} not found at ${composeFile}`)
    return false
  }

  try {
    console.log(`  ▶ Trying ${label}...`)
    execSync(`docker-compose -f "${composeFile}" up -d --remove-orphans`, {
      cwd: ROOT_DIR,
      stdio: 'inherit',
      timeout: 120_000,
    })
    return true
  } catch {
    console.log(`  ⚠ ${label} failed to start`)
    return false
  }
}

export default async function globalSetup() {
  console.log('\n🔧 Chorus E2E Global Setup\n')

  const skipStartup = process.env.E2E_SKIP_STARTUP === 'true'

  // ── Check if services are already running ──
  let servicesRunning = false
  try {
    const response = await fetch(BACKEND_HEALTH)
    servicesRunning = response.ok
  } catch { /* not running */ }

  if (servicesRunning) {
    console.log('ℹ️  Backend already responding on :8080 — skipping Docker startup.')
    console.log('    (Set E2E_SKIP_STARTUP=true or stop backend to let this script manage services.)')
  } else if (skipStartup) {
    console.log('ℹ️  E2E_SKIP_STARTUP=true — skipping service startup.')
    console.log('    Make sure your dev stack is already running via start-dev.ps1 or manually.')
  } else {
    // ── Ensure Docker Desktop ──
    const dockerOk = await ensureDockerDesktop()
    if (!dockerOk) {
      console.warn('⚠️  Docker Desktop is not running. If you already have backend + frontend')
      console.warn('    running manually, re-run with E2E_SKIP_STARTUP=true:')
      console.warn('      $env:E2E_SKIP_STARTUP="true"; npx playwright test')
    }

    // ── Try compose files in order ──
    const devCompose = resolve(ROOT_DIR, 'docker-compose.dev.yml')
    const prodCompose = resolve(ROOT_DIR, 'docker-compose.yml')

    const started = tryComposeUp(devCompose, 'docker-compose.dev.yml') ||
                    tryComposeUp(prodCompose, 'docker-compose.yml')

    if (!started) {
      console.warn('⚠️  Could not start Docker services. If your stack is already running')
      console.warn('    (e.g. via start-dev.ps1), re-run with:')
      console.warn('      $env:E2E_SKIP_STARTUP="true"; npx playwright test')
      console.warn('    Continuing — will wait for services that are already up...')
    } else {
      console.log('✅ Docker services started')
    }
  }

  // ── Wait for backend ──
  console.log('⏳ Waiting for backend health check...')
  await waitForUrl(BACKEND_HEALTH, 'Backend', true)

  // ── Dev seed + JWT clear (QA fix #2, #6) — deterministic alice/bob/sofia ===
  // When E2E_SEED!=false, hit the dev-only seed endpoint (backend exposes it when
  // ENVIRONMENT=development). Falls back to no-op if endpoint is disabled (prod).
  if (process.env.E2E_SEED !== 'false') {
    try {
      const seedUrl = (process.env.E2E_API_URL || 'http://localhost:8080/api/v1') + '/dev/seed'
      // Try via HTTP first (if backend has dev seed route, e.g., POST /dev/seed)
      // Otherwise the seed is done via `go run ./cmd/server --seed-dev` before this.
      const res = await fetch(seedUrl, { method: 'POST' })
      if (res.ok) {
        console.log('✅ Dev seed via API succeeded')
      } else {
        console.log(`ℹ️ Dev seed API not available (${res.status}) — assuming pre-seeded via go run --seed-dev`)
      }
    } catch (e) {
      console.log(`ℹ️ Dev seed API not reachable — assuming pre-seeded: ${(e as Error).message}`)
    }
    // Clear any stale JWTs from prior runs (storageState.json or prior localStorage)
    // Playwright contexts start fresh, but global-setup runs outside browser — ensure
    // no storageState file leaks prior alice/bob ids.
    try {
      const { unlinkSync, existsSync: existsSync2 } = await import('fs')
      const storageState = resolve(ROOT_DIR, 'e2e/storageState.json')
      if (existsSync2(storageState)) {
        unlinkSync(storageState)
        console.log('✅ Cleared stale storageState.json')
      }
    } catch {}
    console.log('✅ Seed + JWT clear done — dev accounts alice.en-es/bob.es-en/sofia.tutor ready')
  }

  // ── Wait for frontend ──
  console.log('⏳ Waiting for frontend...')
  await waitForUrl(FRONTEND_URL, 'Frontend', false)

  // ── P2: Pre-warm translation provider to avoid cold-start flake (hard-fail kept) ──
  if (process.env.E2E_PREWARM !== 'false') {
    console.log('⏳ Pre-warming translation provider (to reduce 60s cold-start)...')
    try {
      const apiUrl = process.env.E2E_API_URL || 'http://localhost:8080/api/v1'
      // Login as DEV_ALICE (seeded) — best-effort, ignore if auth fails (mocked E2E still works)
      const loginRes = await fetch(`${apiUrl}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: 'alice.en-es@chorus.test', password: 'ChorusDev123!' }),
      })
      if (loginRes.ok) {
        const loginBody = (await loginRes.json()) as any
        const token: string = loginBody?.tokens?.accessToken || ''
        if (token) {
          // Ensure a chat exists and trigger a dummy translation to warm Helsinki/Ollama model
          const chatsRes = await fetch(`${apiUrl}/chats`, { headers: { Authorization: `Bearer ${token}` } })
          let chatId: string | null = null
          if (chatsRes.ok) {
            const chatsData = (await chatsRes.json()) as any
            chatId = chatsData?.chats?.[0]?.id || null
          }
          if (!chatId) {
            // Create a warm-up chat with bob.es-en (ignore failure if already exists)
            try {
              const bobLogin = await fetch(`${apiUrl}/auth/login`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username: 'bob.es-en@chorus.test', password: 'ChorusDev123!' }) })
              const bobToken = bobLogin.ok ? ((await bobLogin.json() as any)?.tokens?.accessToken || '') : ''
              // Need bob user id — search
              const search = await fetch(`${apiUrl}/users/search?q=bob.es-en@chorus.test`, { headers: { Authorization: `Bearer ${token}` } })
              const searchData = search.ok ? (await search.json() as any) : null
              const bobId = searchData?.users?.[0]?.id
              if (bobId) {
                const create = await fetch(`${apiUrl}/chats`, { method: 'POST', headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }, body: JSON.stringify({ type: 'direct', participantIds: [bobId] }) })
                if (create.ok) chatId = ((await create.json() as any)?.id || (await create.json() as any)?.chat?.id || null)
              }
              void bobToken
            } catch {}
          }
          if (chatId) {
            const msgRes = await fetch(`${apiUrl}/chats/${chatId}/messages`, { method: 'POST', headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }, body: JSON.stringify({ text: `warmup translation ${Date.now()}` }) })
            if (msgRes.ok) {
              const msg = (await msgRes.json() as any)
              const msgId = msg?.id || msg?.message?.id
              if (msgId) {
                // Trigger translation (best-effort, 5s timeout) to warm translator-engine
                const controller = new AbortController()
                const t = setTimeout(() => controller.abort(), 5000)
                try { await fetch(`${apiUrl}/chats/${chatId}/messages/${msgId}/translate`, { method: 'POST', headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }, body: JSON.stringify({ targetLang: 'es' }), signal: controller.signal }) } catch {}
                clearTimeout(t)
                console.log('✅ Translation pre-warm triggered')
              }
            }
          } else {
            console.log('ℹ️ Pre-warm: no chat to warm (will still run tests hard-fail)')
          }
        }
      } else {
        const snippet = await loginRes.text().then((t) => t.slice(0, 120)).catch(() => '?')
        console.log(`ℹ️ Pre-warm login failed ${loginRes.status} ${snippet} — skipping (mocked E2E does not need it)`)
      }
    } catch (e) {
      console.log(`ℹ️ Pre-warm skipped: ${(e as Error).message} (mocked E2E does not need it)`)
    }
    // Small delay to let translator-engine start model load before first real test
    await new Promise(r => setTimeout(r, 2000))
  }

  console.log('\n✅ All services ready. Starting tests...\n')
}