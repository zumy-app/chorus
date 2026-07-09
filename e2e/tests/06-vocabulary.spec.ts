import { test, expect } from '@playwright/test'
import { loginAsUser, createDirectChat, sendMessage, openProfileMenu, findChatInSidebar } from '../fixtures/test-helpers'
import { ENGLISH_USER, SPANISH_USER } from '../fixtures/users'

/**
 * Test Suite 6: Vocabulary Feature
 *
 * Verifies saving words from messages, viewing the vocabulary list,
 * and the spaced repetition practice flow.
 */
test.describe('Vocabulary', () => {
  test.describe.configure({ mode: 'serial' })

  test('6.1 — Save word from message', async ({ browser }) => {
    const senderContext = await browser.newContext()
    const receiverContext = await browser.newContext()
    const senderPage = await senderContext.newPage()
    const receiverPage = await receiverContext.newPage()

    try {
      await loginAsUser(senderPage, ENGLISH_USER)
      await loginAsUser(receiverPage, SPANISH_USER)

      await createDirectChat(senderPage, SPANISH_USER.displayName)
      const chatItem = await findChatInSidebar(receiverPage, ENGLISH_USER.displayName)
      await chatItem.click()

      // Send a message with words longer than 3 chars (filter requirement)
      const testMsg = `The elephant walked carefully through the jungle. ${Date.now()}`
      await sendMessage(senderPage, testMsg)

      await expect(receiverPage.locator('.break-words', { hasText: testMsg }).last()).toBeVisible({
        timeout: 15_000,
      })

      // Hover over the message to reveal action buttons
      // Action buttons are in the grandparent container (not the immediate parent)
      const messageWrapper = receiverPage.locator('.break-words', { hasText: testMsg }).last()
        .locator('xpath=ancestor::div[contains(@class, "flex")][1]')
      await messageWrapper.hover()

      // Click the first word save button (e.g., "+ elephant")
      const saveWordBtn = messageWrapper.getByRole('button').filter({ hasText: '+' }).first()
      await expect(saveWordBtn).toBeVisible({ timeout: 5_000 })
      await saveWordBtn.click()

      // Verify "✅ Saved" appears
      await expect(messageWrapper.getByRole('button', { hasText: '✅ Saved' }).first()).toBeVisible({ timeout: 10_000 })
    } finally {
      await senderContext.close()
      await receiverContext.close()
    }
  })

  test('6.2 — Vocabulary modal opens', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await openProfileMenu(page)
    await page.getByRole('button', { name: /vocabulary/i }).click()

    // Verify the vocabulary modal is visible
    await expect(page.locator('h2', { hasText: '📚 Vocabulary' })).toBeVisible({ timeout: 10_000 })
  })

  test('6.3 — Vocabulary stats display', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await openProfileMenu(page)
    await page.getByRole('button', { name: /vocabulary/i }).click()

    await expect(page.locator('h2', { hasText: '📚 Vocabulary' })).toBeVisible({ timeout: 10_000 })

    // Verify stats grid appears (Total Words, Mastered, Due Today, Accuracy)
    await expect(page.locator('text=Total Words')).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('text=Mastered')).toBeVisible()
    await expect(page.locator('text=Due Today')).toBeVisible()
    await expect(page.locator('text=Accuracy')).toBeVisible()
  })

  test('6.4 — Vocabulary list loads', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await openProfileMenu(page)
    await page.getByRole('button', { name: /vocabulary/i }).click()

    await expect(page.locator('h2', { hasText: '📚 Vocabulary' })).toBeVisible({ timeout: 10_000 })

    // Click "All Words" tab
    await page.getByRole('button', { name: /all words/i }).click()

    // Wait for loading to complete
    // Either entries appear or the empty state shows
    await expect(
      page.locator('text=No vocabulary saved yet').or(page.locator('.border.border-gray-200.rounded-lg.p-4').first()),
    ).toBeVisible({ timeout: 15_000 })
  })

  test('6.5 — Vocabulary modal can be closed', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await openProfileMenu(page)
    await page.getByRole('button', { name: /vocabulary/i }).click()

    await expect(page.locator('h2', { hasText: '📚 Vocabulary' })).toBeVisible({ timeout: 10_000 })

    // Click the × button
    await page.locator('h2', { hasText: '📚 Vocabulary' }).locator('..').getByRole('button').click()

    // Verify modal is closed
    await expect(page.locator('h2', { hasText: '📚 Vocabulary' })).not.toBeVisible({ timeout: 5_000 })
  })
})