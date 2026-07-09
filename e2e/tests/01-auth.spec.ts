import { test, expect } from '@playwright/test'
import { loginAsUser, openProfileMenu } from '../fixtures/test-helpers'
import { ENGLISH_USER, SPANISH_USER } from '../fixtures/users'

/**
 * Test Suite 1: Authentication & Session Management
 *
 * Verifies login flows for both test users, session persistence,
 * and logout. These tests must pass before the messaging suites
 * can run, as they establish the users are valid.
 */
test.describe('Authentication', () => {
  test.describe.configure({ mode: 'serial' })

  test('1.1 — Login as English user (uhsarp@gmail.com)', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    // Verify we're on the chat page
    await expect(page).toHaveURL(/\/chat/)

    // Verify the user is logged in by checking the header
    await expect(page.locator('h1', { hasText: 'Chorus' })).toBeVisible()

    // Open profile menu to verify user identity
    await openProfileMenu(page)
    await expect(page.locator('text=' + ENGLISH_USER.email)).toBeVisible()
  })

  test('1.2 — Login as Spanish user (avcxafefwer@gmail.com)', async ({ browser }) => {
    // Use a fresh browser context to simulate a separate device
    const context = await browser.newContext()
    const page = await context.newPage()

    await loginAsUser(page, SPANISH_USER)

    // Verify we're on the chat page
    await expect(page).toHaveURL(/\/chat/)

    // Verify the user is logged in
    await expect(page.locator('h1', { hasText: 'Chorus' })).toBeVisible()

    // Open profile menu to verify user identity
    await openProfileMenu(page)
    await expect(page.locator('text=' + SPANISH_USER.email)).toBeVisible()

    await context.close()
  })

  test('1.3 — Invalid credentials are rejected', async ({ page }) => {
    await page.goto('/login')

    await page.locator('input[type="email"]').fill(ENGLISH_USER.email)
    await page.locator('input[type="password"]').fill('WrongPassword123!')
    await page.getByRole('button', { name: /log in/i }).click()

    // Verify error message appears
    await expect(page.locator('.bg-red-100')).toBeVisible({ timeout: 10_000 })

    // Verify we're still on the login page
    await expect(page).toHaveURL(/\/login/)
  })

  test('1.4 — Session persists across page reload', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    // Reload the page
    await page.reload()

    // Should still be on /chat (token in localStorage keeps session)
    await expect(page).toHaveURL(/\/chat/)
    await expect(page.locator('h1', { hasText: 'Chorus' })).toBeVisible()
  })

  test('1.5 — Logout clears session and redirects to login', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    // Open profile menu and click Sign Out
    await openProfileMenu(page)
    await page.getByRole('button', { name: /sign out/i }).click()

    // Should redirect to /login
    await expect(page).toHaveURL(/\/login/, { timeout: 10_000 })

    // Verify localStorage tokens are cleared
    const accessToken = await page.evaluate(() => localStorage.getItem('accessToken'))
    expect(accessToken).toBeNull()
  })
})