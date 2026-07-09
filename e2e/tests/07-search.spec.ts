import { test, expect } from '@playwright/test'
import { loginAsUser, createDirectChat, sendMessage, findChatInSidebar } from '../fixtures/test-helpers'
import { ENGLISH_USER, SPANISH_USER } from '../fixtures/users'

/**
 * Test Suite 7: Search Feature
 *
 * Verifies the message search functionality — searching across
 * all chats or within a specific chat.
 */
test.describe('Search', () => {
  test.describe.configure({ mode: 'serial' })

  test('7.1 — Search modal opens', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    // Click "Search Messages" button
    await page.getByRole('button', { name: /search messages/i }).click()

    // Verify the search modal is visible
    await expect(page.locator('input[placeholder="Search messages..."]')).toBeVisible({ timeout: 10_000 })
  })

  test('7.2 — Search returns results', async ({ browser }) => {
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

      // Send a message with a unique searchable keyword
      const uniqueKeyword = `searchable_${Date.now()}`
      const testMsg = `This is a ${uniqueKeyword} message for testing`
      await sendMessage(senderPage, testMsg)

      await expect(receiverPage.locator('.break-words', { hasText: testMsg }).last()).toBeVisible({
        timeout: 15_000,
      })

      // Open search on receiver page
      await receiverPage.getByRole('button', { name: /search messages/i }).click()
      await expect(receiverPage.locator('input[placeholder="Search messages..."]')).toBeVisible({
        timeout: 10_000,
      })

      // Search for the unique keyword
      await receiverPage.locator('input[placeholder="Search messages..."]').fill(uniqueKeyword)
      await receiverPage.getByRole('button', { name: 'Search', exact: true }).click()

      // Verify results appear - wait for the search results section or message content
      await expect(receiverPage.locator('.line-clamp-2', { hasText: uniqueKeyword }).first()).toBeVisible({
        timeout: 15_000,
      })
    } finally {
      await senderContext.close()
      await receiverContext.close()
    }
  })

  test('7.3 — Search with no results shows empty state', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await page.getByRole('button', { name: /search messages/i }).click()
    await expect(page.locator('input[placeholder="Search messages..."]')).toBeVisible({ timeout: 10_000 })

    // Search for something that definitely doesn't exist
    const nonexistentQuery = `zzz_nonexistent_${Date.now()}_zzz`
    await page.locator('input[placeholder="Search messages..."]').fill(nonexistentQuery)
    await page.getByRole('button', { name: 'Search', exact: true }).click()

    // Wait for either the no-results state or the initial search prompt to update
    await page.waitForFunction((query) => {
      const body = document.body.innerText
      return body.includes('No messages found') || body.includes('📭')
    }, nonexistentQuery, { timeout: 15_000 })
  })

  test('7.4 — Search modal can be closed', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    await page.getByRole('button', { name: /search messages/i }).click()
    await expect(page.locator('input[placeholder="Search messages..."]')).toBeVisible({ timeout: 10_000 })

    // Click the × button
    await page.locator('input[placeholder="Search messages..."]').locator('..').locator('..').getByText('×').click()

    // Verify modal is closed
    await expect(page.locator('input[placeholder="Search messages..."]')).not.toBeVisible({ timeout: 5_000 })
  })
})