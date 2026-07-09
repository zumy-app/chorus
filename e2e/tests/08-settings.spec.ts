import { test, expect } from '@playwright/test'
import { loginAsUser, openProfileMenu } from '../fixtures/test-helpers'
import { ENGLISH_USER } from '../fixtures/users'

/**
 * Test Suite 8: Settings & Profile
 *
 * Verifies the settings modal — display name, native language,
 * target languages, and the header language selector.
 */
test.describe('Settings & Profile', () => {
  test.describe.configure({ mode: 'serial' })

  test('8.1 — Settings modal opens', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await openProfileMenu(page)
    await page.getByRole('button', { name: /settings/i }).click()

    // Verify the settings modal is visible
    await expect(page.locator('h2', { hasText: 'Settings' })).toBeVisible({ timeout: 10_000 })
  })

  test('8.2 — Settings form fields are present', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await openProfileMenu(page)
    await page.getByRole('button', { name: /settings/i }).click()

    await expect(page.locator('h2', { hasText: 'Settings' })).toBeVisible({ timeout: 10_000 })

    // Verify Display Name field
    await expect(page.locator('label', { hasText: 'Display Name' })).toBeVisible()

    // Verify Email field (read-only)
    await expect(page.locator('label', { hasText: 'Email' })).toBeVisible()

    // Verify Native Language field
    await expect(page.locator('label', { hasText: 'Native Language' })).toBeVisible()

    // Verify Target Languages section
    await expect(page.locator('label', { hasText: /languages you want to learn/i })).toBeVisible()
  })

  test('8.3 — Update display name', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await openProfileMenu(page)
    await page.getByRole('button', { name: /settings/i }).click()

    await expect(page.locator('h2', { hasText: 'Settings' })).toBeVisible({ timeout: 10_000 })

    // Change display name to a fixed test value, then restore original
    const testName = `TestDisplayName`
    const nameInput = page.locator('input').first()
    await nameInput.fill(testName)

    // Save
    await page.getByRole('button', { name: /save settings/i }).click()

    // Verify success message
    await expect(page.locator('text=Settings saved successfully')).toBeVisible({ timeout: 10_000 })

    // Restore original display name so other tests can find the user
    await nameInput.fill(ENGLISH_USER.displayName)
    await page.getByRole('button', { name: /save settings/i }).click()
    await expect(page.locator('text=Settings saved successfully')).toBeVisible({ timeout: 10_000 })
  })

  test('8.4 — Native language dropdown works', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await openProfileMenu(page)
    await page.getByRole('button', { name: /settings/i }).click()

    await expect(page.locator('h2', { hasText: 'Settings' })).toBeVisible({ timeout: 10_000 })

    // Verify the native language select has options
    const langSelect = page.locator('select').first()
    await expect(langSelect).toBeVisible()

    // Verify it has multiple options (SUPPORTED_LANGUAGES)
    const optionCount = await langSelect.locator('option').count()
    expect(optionCount).toBeGreaterThan(10) // We support 80+ languages
  })

  test('8.5 — Target languages can be toggled', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await openProfileMenu(page)
    await page.getByRole('button', { name: /settings/i }).click()

    await expect(page.locator('h2', { hasText: 'Settings' })).toBeVisible({ timeout: 10_000 })

    // Find a target language checkbox (Spanish should be available since native is English)
    const spanishLabel = page.locator('label').filter({ hasText: /Español/i })
    await expect(spanishLabel).toBeVisible()

    // Get the checkbox and check its initial state
    const checkbox = spanishLabel.locator('input[type="checkbox"]')
    const wasChecked = await checkbox.isChecked()

    // Click to toggle it
    await spanishLabel.click()

    // Wait a moment for state to update
    await page.waitForTimeout(500)

    // Verify the checkbox state changed
    const isNowChecked = await checkbox.isChecked()
    expect(isNowChecked).not.toBe(wasChecked)
  })

  test('8.6 — Settings modal can be closed', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await openProfileMenu(page)
    await page.getByRole('button', { name: /settings/i }).click()

    await expect(page.locator('h2', { hasText: 'Settings' })).toBeVisible({ timeout: 10_000 })

    // Click Cancel button
    await page.getByRole('button', { name: /cancel/i }).click()

    // Verify modal is closed
    await expect(page.locator('h2', { hasText: 'Settings' })).not.toBeVisible({ timeout: 5_000 })
  })

  test('8.7 — Header language selector is visible', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    // The compact language selector should be in the header
    // It shows the current language code (e.g., "EN")
    const header = page.locator('header')
    await expect(header).toBeVisible()

    // Look for language-related elements in the header
    // The LanguageSelector component renders a button or dropdown
    const langSelector = header.locator('button, select').filter({ hasText: /en|english|🌐/i }).first()
    await expect(langSelector).toBeVisible({ timeout: 5_000 })
  })
})