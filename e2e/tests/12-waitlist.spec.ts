import { test, expect } from '@playwright/test'
import { API_BASE } from '../fixtures/test-helpers'

/**
 * Test Suite 12: Waitlist
 *
 * Verifies the public waitlist flow:
 *   - client-side email validation
 *   - joining with language + reason selections
 *   - idempotent re-submission (already joined, preferences updated, same queue spot)
 *   - the confirmation copy and Discord link
 *   - the backend /waitlist API reshape & idempotency
 */
const waitEmail = (kind: string) =>
  `waitlist-e2e-${kind}-${Date.now().toString(36)}@example.com`

test.describe('Waitlist', () => {
  test.describe.configure({ mode: 'serial' })

  const speakPicker = (page: import('@playwright/test').Page) =>
    page
      .locator('span.block.font-semibold.text-gray-700', { hasText: 'I speak' })
      .locator('..')
  const learnPicker = (page: import('@playwright/test').Page) =>
    page
      .locator('span.block.font-semibold.text-gray-700', { hasText: 'I want to learn' })
      .locator('..')

  test('12.1 — Waitlist page renders the signup form', async ({ page }) => {
    await page.goto('/waitlist')
    await expect(page.getByRole('heading', { name: /join the waitlist/i })).toBeVisible()
    await expect(page.locator('input[type="email"]')).toBeVisible()
    await expect(page.getByText('I speak')).toBeVisible()
    await expect(page.getByText('I want to learn')).toBeVisible()
    await expect(page.getByText('Reason for using Chorus.talk')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Join the waitlist' })).toBeVisible()
  })

  test('12.2 — Invalid email is rejected client-side', async ({ page }) => {
    await page.goto('/waitlist')
    await page.locator('input[type="email"]').fill('not-an-email')
    await page.getByRole('button', { name: 'Join the waitlist' }).click()
    await expect(page.locator('text=Please enter a valid email address.')).toBeVisible()
    // Still on the form (no navigation away / no success card)
    await expect(page.getByRole('heading', { name: /join the waitlist/i })).toBeVisible()
  })

  test('12.3 — User joins the waitlist and sees their place + confirmation copy', async ({ page }) => {
    const email = waitEmail('ui')
    await page.goto('/waitlist')

    await page.locator('input[type="email"]').fill(email)
    await speakPicker(page).getByRole('button', { name: /English/ }).click()
    await learnPicker(page).getByRole('button', { name: /Spanish/ }).click()
    await learnPicker(page).getByRole('button', { name: /French/ }).click()
    await page.locator('label', { hasText: 'For travel' }).click()

    await page.getByRole('button', { name: 'Join the waitlist' }).click()

    // Confirmation card
    await expect(page.getByRole('heading', { name: /on the list/i })).toBeVisible({ timeout: 15_000 })
    const body = await page.locator('main').innerText()
    expect(body).toMatch(/waitlist number is \d+/i)
    expect(body).toContain('info@chorus.talk')
    expect(body).toMatch(/spam|junk/i)
    expect(page.locator('a[href="https://discord.gg/7DVwM6jsS"]')).toBeVisible()
  })

  test('12.4 — Re-submitting the same email updates prefs and keeps the slot', async ({ page }) => {
    const email = waitEmail('rejoin')
    // First join
    await page.goto('/waitlist')
    await page.locator('input[type="email"]').fill(email)
    await speakPicker(page).getByRole('button', { name: /English/ }).click()
    await learnPicker(page).getByRole('button', { name: /Spanish/ }).click()
    await page.locator('label', { hasText: 'For travel' }).click()
    await page.getByRole('button', { name: 'Join the waitlist' }).click()
    await expect(page.getByRole('heading', { name: /on the list/i })).toBeVisible({ timeout: 15_000 })
    const firstBody = await page.locator('main').innerText()
    const firstNumber = (firstBody.match(/waitlist number is (\d+)/i) || [])[1]
    expect(firstNumber).toBeTruthy()

    // Second join — same email, tweaked preferences
    await page.goto('/waitlist')
    await page.locator('input[type="email"]').fill(email)
    await speakPicker(page).getByRole('button', { name: /English/ }).click()
    await learnPicker(page).getByRole('button', { name: /French/ }).click()
    await page.locator('label', { hasText: 'For work' }).click()
    await page.getByRole('button', { name: 'Join the waitlist' }).click()

    await expect(page.getByRole('heading', { name: /welcome back/i })).toBeVisible({ timeout: 15_000 })
    const secondBody = await page.locator('main').innerText()
    expect(secondBody).toContain(`#${firstNumber}`)
    expect(secondBody).toMatch(/updated your preferences/i)
  })

  test('12.5 — Backend join is idempotent and returns alreadyJoined', async ({ request }) => {
    const email = waitEmail('api')
    const payload = {
      email,
      spokenLanguages: ['en', 'fr'],
      targetLanguages: ['es', 'zh'],
      reasons: ['Connect with friends or family', 'For travel'],
      comments: 'e2e test',
    }

    const first = await request.post(`${API_BASE}/waitlist`, { data: payload })
    expect(first.ok()).toBeTruthy()
    const firstBody = await first.json()
    expect(firstBody.alreadyJoined).toBe(false)
    const firstPos = firstBody.entry.queuePosition

    const second = await request.post(`${API_BASE}/waitlist`, {
      data: { ...payload, targetLanguages: ['de', 'pt'] },
    })
    const secondBody = await second.json()
    expect(secondBody.alreadyJoined).toBe(true)
    expect(secondBody.entry.queuePosition).toBe(firstPos)
    expect(secondBody.entry.targetLanguages).toEqual(['de', 'pt'])

    // Clean up the E2E entry so the dev waitlist stays tidy
    await deleteWaitlistEntry(email)
  })
})

/** Best-effort cleanup: remove a waitlist entry directly from the DB. */
async function deleteWaitlistEntry(email: string) {
  try {
    const { execSync } = await import('child_process')
    execSync(
      `docker compose -f docker-compose.yml exec -T postgres psql -U messenger -d messenger_prod -c "DELETE FROM waitlist_entries WHERE email = '${email}'"`,
      { stdio: 'pipe' }
    )
  } catch {
    // Cleanup is best-effort — ignore failures (varies by stack).
  }
}