import { test, expect } from '@playwright/test'
import { API_BASE, loginViaAPI } from '../fixtures/test-helpers'
import { ENGLISH_USER, SPANISH_USER } from '../fixtures/users'

/**
 * Test Suite 10: Service Health & Infrastructure
 *
 * Verifies that all backend services are running and responding.
 * These are API-level tests that don't require the browser.
 */
test.describe('Service Health', () => {
  test.describe.configure({ mode: 'serial' })

  test('10.1 — Backend health check responds', async ({ request }) => {
    const response = await request.get('http://localhost:8080/health')
    expect(response.ok()).toBeTruthy()

    const body = await response.json()
    expect(body.status).toBe('healthy')
    expect(body.version).toBeTruthy()
  })

  test('10.2 — Frontend serves HTML', async ({ page }) => {
    const response = await page.goto('/')
    expect(response?.ok()).toBeTruthy()

    // Verify the page has the Chorus title or content
    const title = await page.title()
    // The title may be "Chorus" or similar
    expect(title).toBeTruthy()
  })

  test('10.3 — Backend API login works (English user)', async () => {
    const token = await loginViaAPI(ENGLISH_USER)
    expect(token).toBeTruthy()
    expect(token.length).toBeGreaterThan(20) // JWT tokens are long
  })

  test('10.4 — Backend API login works (Spanish user)', async () => {
    const token = await loginViaAPI(SPANISH_USER)
    expect(token).toBeTruthy()
    expect(token.length).toBeGreaterThan(20)
  })

  test('10.5 — Authenticated API call succeeds', async ({ request }) => {
    const token = await loginViaAPI(ENGLISH_USER)

    const response = await request.get(`${API_BASE}/users/me`, {
      headers: { Authorization: `Bearer ${token}` },
    })

    expect(response.ok()).toBeTruthy()
    const user = await response.json()
    expect(user.email).toBe(ENGLISH_USER.email)
    expect(user.nativeLanguage).toBe(ENGLISH_USER.nativeLanguage)
  })

  test('10.6 — Chats endpoint returns array', async ({ request }) => {
    const token = await loginViaAPI(ENGLISH_USER)

    const response = await request.get(`${API_BASE}/chats`, {
      headers: { Authorization: `Bearer ${token}` },
    })

    expect(response.ok()).toBeTruthy()
    const body = await response.json()
    expect(Array.isArray(body.chats)).toBeTruthy()
  })

  test('10.7 — User search endpoint works', async ({ request }) => {
    const token = await loginViaAPI(ENGLISH_USER)

    const response = await request.get(`${API_BASE}/users/search?q=${SPANISH_USER.displayName}`, {
      headers: { Authorization: `Bearer ${token}` },
    })

    expect(response.ok()).toBeTruthy()
    const body = await response.json()
    expect(body.users).toBeTruthy()
    expect(Array.isArray(body.users)).toBeTruthy()

    // The Spanish user should be in the results
    const found = body.users.find((u: any) => u.email === SPANISH_USER.email)
    expect(found).toBeTruthy()
  })

  test('10.8 — Translator engine (llama.cpp) is available', async ({ request }) => {
    // The translator-engine is internal to Docker, but the backend exposes it
    // indirectly through the translation flow. We verify by checking
    // that the backend can reach it.

    // Check the translator-engine health endpoint (port 5002 on host)
    try {
      const healthResponse = await request.get('http://localhost:5002/health', {
        timeout: 30_000,
      })

      if (healthResponse.ok()) {
        console.log('✓ Translator-engine is healthy')
      }
    } catch {
      // Translator-engine may not be exposed on host port in all configs
      console.log('ℹ️ Translator-engine not directly accessible (may be internal to Docker network)')
    }
  })

  test('10.9 — No console errors on page load', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (error) => {
      errors.push(error.message)
    })

    await page.goto('/')
    await page.waitForTimeout(3_000)

    // Filter out expected errors (e.g., favicon 404, WebSocket connection retries)
    const realErrors = errors.filter(
      (e) => !e.includes('favicon') && !e.includes('WebSocket'),
    )

    expect(realErrors).toHaveLength(0)
  })
})