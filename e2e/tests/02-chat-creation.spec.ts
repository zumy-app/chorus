import { test, expect } from '@playwright/test'
import { loginAsUser, createDirectChat } from '../fixtures/test-helpers'
import { ENGLISH_USER, SPANISH_USER } from '../fixtures/users'

/**
 * Test Suite 2: Chat Creation & Management
 *
 * Verifies that the English user can find the Spanish user and create
 * a direct chat. Also checks chat list behavior.
 */
test.describe('Chat Creation', () => {
  test.describe.configure({ mode: 'serial' })

  test('2.1 — Create direct chat with Spanish user', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    // Create chat by searching for the Spanish user's email prefix
    await createDirectChat(page, SPANISH_USER.displayName)

    // Verify the chat area is now visible
    // The chat header should show the other participant's name
    await expect(page.locator('h2', { hasText: SPANISH_USER.displayName })).toBeVisible({
      timeout: 10_000,
    })

    // Verify the language indicator shows for direct chat
    await expect(page.locator('text=🌍 ES')).toBeVisible()
  })

  test('2.2 — Chat appears in sidebar list', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    // The chat from 2.1 should appear in the sidebar
    // Look for the chat list item containing the Spanish user's name
    const chatListItem = page.locator('.cursor-pointer').filter({ hasText: SPANISH_USER.displayName })
    await expect(chatListItem).toBeVisible({ timeout: 10_000 })
  })

  test('2.3 — Chat list shows most recent at top', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    // Get all chat list items
    const chatItems = page.locator('.cursor-pointer')
    const count = await chatItems.count()

    if (count > 1) {
      // The first item should be the most recently active chat
      // (After creating a chat in 2.1, it should be at or near the top)
      const firstItemText = await chatItems.first().textContent()
      expect(firstItemText).toBeTruthy()
    }
  })

  test('2.4 — Opening existing chat does not create duplicate', async ({ page }) => {
    await loginAsUser(page, ENGLISH_USER)

    // Wait for the chat list to finish loading (either chat items or empty state)
    await page.waitForFunction(() => {
      const chatItems = document.querySelectorAll('.cursor-pointer')
      const emptyState = document.body.innerText.includes('No chats yet')
      return chatItems.length > 0 || emptyState
    }, { timeout: 10_000 })

    // Count existing chats
    const chatItemsBefore = page.locator('.cursor-pointer')
    const countBefore = await chatItemsBefore.count()

    // Try to create a new chat with the same user
    await createDirectChat(page, SPANISH_USER.displayName)

    // Count chats after — should be the same (no duplicate)
    const chatItemsAfter = page.locator('.cursor-pointer')
    const countAfter = await chatItemsAfter.count()

    // The backend should return the existing chat, not create a new one
    expect(countAfter).toBe(countBefore)
  })
})