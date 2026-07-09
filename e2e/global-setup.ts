import { execSync } from 'child_process'
import { existsSync } from 'fs'
import { resolve } from 'path'

/**
 * Global setup: starts the Chorus Docker stack and waits for services to be healthy.
 *
 * This runs once before all test files. It:
 * 1. Checks if Docker is available
 * 2. Starts services via `docker-compose up -d` (if not already running)
 * 3. Waits for backend health endpoint to respond
 * 4. Waits for frontend to serve HTML
 *
 * If services are already running, it skips startup (useful for dev iteration).
 */

const BACKEND_HEALTH = 'http://localhost:8080/health'
const FRONTEND_URL = 'http://localhost:3000'
const MAX_WAIT_MS = 300_000 // 5 minutes — ALMA-7B GGUF model download on first run
const POLL_INTERVAL_MS = 5_000

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

export default async function globalSetup() {
  console.log('\n🔧 Chorus E2E Global Setup\n')

  const composeFile = resolve(__dirname, '..', 'docker-compose.yml')
  const skipStartup = process.env.E2E_SKIP_STARTUP === 'true'

  if (!skipStartup) {
    // Check if Docker is available
    try {
      execSync('docker --version', { stdio: 'pipe' })
    } catch {
      console.warn('⚠️  Docker not found. Assuming services are running externally.')
    }

    // Check if services are already running
    let servicesRunning = false
    try {
      const response = await fetch(BACKEND_HEALTH)
      servicesRunning = response.ok
    } catch {
      // Not running
    }

    if (!servicesRunning) {
      if (!existsSync(composeFile)) {
        throw new Error(`docker-compose.yml not found at ${composeFile}`)
      }

      console.log('📦 Starting Chorus services via docker-compose...')
      try {
        execSync('docker-compose up -d', {
          cwd: resolve(__dirname, '..'),
          stdio: 'inherit',
          timeout: 120_000,
        })
      } catch (err) {
        console.error('❌ Failed to start docker-compose:', err)
        throw err
      }
    } else {
      console.log('ℹ️  Services already running, skipping startup.')
    }
  } else {
    console.log('ℹ️  E2E_SKIP_STARTUP=true — skipping service startup.')
  }

  // Wait for backend
  console.log('⏳ Waiting for backend health check...')
  await waitForUrl(BACKEND_HEALTH, 'Backend', true)

  // Wait for frontend
  console.log('⏳ Waiting for frontend...')
  await waitForUrl(FRONTEND_URL, 'Frontend', false)

  console.log('\n✅ All services ready. Starting tests...\n')
}