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

const BACKEND_HEALTH = 'http://localhost:8080/health'
const FRONTEND_URL = 'http://localhost:3000'
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

  // ── Wait for frontend ──
  console.log('⏳ Waiting for frontend...')
  await waitForUrl(FRONTEND_URL, 'Frontend', false)

  console.log('\n✅ All services ready. Starting tests...\n')
}