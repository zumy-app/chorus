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

    await expect(page.getByTestId('settings-modal').or(page.locator('h2', { hasText: 'Settings' })).first()).toBeVisible({ timeout: 10_000 })
  })

  test('8.2 — Settings form fields are present', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await openProfileMenu(page)
    await page.getByRole('button', { name: /settings/i }).click()

    await expect(page.getByTestId('settings-modal').or(page.locator('h2', { hasText: 'Settings' })).first()).toBeVisible({ timeout: 10_000 })

    await expect(page.getByTestId('settings-modal').locator('label', { hasText: 'Display Name' }).or(page.locator('label', { hasText: 'Display Name' })).first()).toBeVisible()
    await expect(page.locator('label', { hasText: 'Email' }).first()).toBeVisible()
    await expect(page.locator('label', { hasText: 'Native Language' }).first()).toBeVisible()
    await expect(page.locator('label', { hasText: /languages you want to learn/i }).first()).toBeVisible()
  })

  test('8.3 — Update display name', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await openProfileMenu(page)
    await page.getByRole('button', { name: /settings/i }).click()

    await expect(page.getByTestId('settings-modal').or(page.locator('h2', { hasText: 'Settings' })).first()).toBeVisible({ timeout: 10_000 })

    const testName = `TestDisplayName`
    const nameInput = page.getByTestId('settings-modal').locator('input[type="text"]').first()
    await expect(nameInput).toBeVisible({ timeout: 5_000 })
    await nameInput.fill(testName)

    await page.getByRole('button', { name: /save settings/i }).click()
    await expect(page.locator('text=Settings saved successfully').or(page.locator('text=Saved')).first()).toBeVisible({ timeout: 10_000 })

    await nameInput.fill(ENGLISH_USER.displayName)
    await page.getByRole('button', { name: /save settings/i }).click()
    await expect(page.locator('text=Settings saved successfully').or(page.locator('text=Saved')).first()).toBeVisible({ timeout: 10_000 })
  })

  test('8.4 — Native language dropdown works', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await openProfileMenu(page)
    await page.getByRole('button', { name: /settings/i }).click()

    await expect(page.getByTestId('settings-modal').or(page.locator('h2', { hasText: 'Settings' })).first()).toBeVisible({ timeout: 10_000 })

    const langSelect = page.getByTestId('settings-modal').locator('select').first()
    await expect(langSelect).toBeVisible()
    const optionCount = await langSelect.locator('option').count()
    expect(optionCount).toBeGreaterThan(10)
  })

  test('8.5 — Target languages can be toggled', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await openProfileMenu(page)
    await page.getByRole('button', { name: /settings/i }).click()

    await expect(page.getByTestId('settings-modal').or(page.locator('h2', { hasText: 'Settings' })).first()).toBeVisible({ timeout: 10_000 })

    const spanishLabel = page.getByTestId('settings-modal').locator('label').filter({ hasText: /Español/i }).or(page.locator('label').filter({ hasText: /Español/i })).first()
    await expect(spanishLabel).toBeVisible()
    const checkbox = spanishLabel.locator('input[type="checkbox"]')
    const wasChecked = await checkbox.isChecked()
    await spanishLabel.click()
    await page.waitForTimeout(500)
    const isNowChecked = await checkbox.isChecked()
    expect(isNowChecked).not.toBe(wasChecked)
  })

  test('8.6 — Settings modal can be closed', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await openProfileMenu(page)
    await page.getByRole('button', { name: /settings/i }).click()

    await expect(page.getByTestId('settings-modal').or(page.locator('h2', { hasText: 'Settings' })).first()).toBeVisible({ timeout: 10_000 })

    const closeBtn = page.getByTestId('settings-close').or(page.getByRole('button', { name: /cancel/i }))
    await closeBtn.first().click()

    await expect(page.getByTestId('settings-modal').or(page.locator('h2', { hasText: 'Settings' })).first()).not.toBeVisible({ timeout: 5_000 })
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