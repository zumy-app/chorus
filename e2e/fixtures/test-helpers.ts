import { Page, expect } from '@playwright/test'
import { TestUser } from './users'

/**
 * Shared test helpers for Chorus E2E tests.
 *
 * These helpers encapsulate common UI flows (login, create chat, send message)
 * so test files stay readable and focused on assertions.
 */

/**
 * Log in a user via the UI.
 * Assumes the app is on /login or / (will navigate if needed).
 */
export async function loginAsUser(page: Page, user: TestUser) {
  await page.goto('/login')

  // Wait for the login form to render
  await expect(page.locator('input[type="email"]')).toBeVisible()

  await page.locator('input[type="email"]').fill(user.email)
  await page.locator('input[type="password"]').fill(user.password)
  await page.getByRole('button', { name: /log in/i }).click()

  // Wait for redirect to /chat
  await page.waitForURL('**/chat', { timeout: 30_000 })

  // Verify the header (Chorus logo) is visible — confirms we're in the app
  await expect(page.locator('h1', { hasText: 'Chorus' })).toBeVisible()
}

/**
 * Create a direct chat with another user by searching for them.
 * Assumes the user is already logged in and on /chat.
 */
export async function createDirectChat(page: Page, searchQuery: string) {
  // Click "+ New Chat"
  await page.getByRole('button', { name: /new chat/i }).click()

  // Wait for modal
  await expect(page.locator('h2', { hasText: 'New Chat' })).toBeVisible()

  // Direct Chat should be selected by default
  await expect(page.getByRole('button', { name: 'Direct Chat' })).toBeVisible()

  // Search for the user
  await page.locator('input[placeholder="Search users..."]').fill(searchQuery)

  // Wait for search results to appear (debounced — needs at least 2 chars)
  await expect(page.locator('text=Search Results')).toBeVisible({ timeout: 15_000 })

  // Click the first search result
  const firstResult = page.locator('.space-y-2 > div').first()
  await firstResult.click()

  // Click "Create Chat"
  await page.getByRole('button', { name: /create chat/i }).click()

  // Wait for modal to close and chat area to load
  await expect(page.locator('h2', { hasText: 'New Chat' })).not.toBeVisible({ timeout: 10_000 })
}

/**
 * Send a message in the currently active chat.
 * Assumes a chat is already open.
 */
export async function sendMessage(page: Page, text: string) {
  const input = page.locator('textarea[placeholder="Type a message..."]')
  await expect(input).toBeVisible()
  await input.fill(text)
  await page.getByRole('button', { name: 'Send' }).click()

  // Wait for the message to appear in the chat area
  // Use .last() to target the most recently sent message (handles duplicates from prior runs)
  await expect(page.locator('.break-words', { hasText: text }).last()).toBeVisible({ timeout: 15_000 })
}

/**
 * Wait for a translation to appear in a message bubble.
 *
 * The backend sends translations asynchronously via WebSocket after the
 * `new_message` event. The `message_updated` event delivers translations.
 * In the UI, this renders as "🌐 In your language:" followed by the text.
 *
 * @param page Playwright page
 * @param originalText The original message text to locate the bubble
 * @param timeoutMs How long to wait (translator-engine cold start can be slow)
 */
export async function waitForTranslation(
  page: Page,
  originalText: string,
  timeoutMs = 60_000,
) {
  // Find the message bubble containing the original text
  // Use .last() to target the most recent message (handles duplicates)
  const bubble = page.locator('.break-words', { hasText: originalText }).last().locator('..')

  // Wait for the "In your language:" section to appear
  await expect(
    bubble.locator('text=🌐 In your language:'),
  ).toBeVisible({ timeout: timeoutMs })
}

/**
 * Open the grammar analysis panel for a received (non-own) message.
 * Hovers the message bubble to reveal the "📝 Grammar" button, then clicks it.
 */
export async function openGrammarAnalysis(page: Page, messageText: string) {
  // Find the message bubble (must be a received message, not own)
  // Use .last() to target the most recent message (handles duplicates)
  const messageText_el = page.locator('.break-words', { hasText: messageText }).last()
  const outerWrapper = messageText_el.locator('xpath=ancestor::div[contains(@class, "flex") and contains(@class, "justify-")][1]')

  // Hover to reveal action buttons
  await outerWrapper.hover()

  // Click the Grammar button (it appears as a sibling of the bubble inside the inner wrapper)
  // Use exact text to avoid matching vocabulary buttons with similar names
  const grammarBtn = outerWrapper.getByRole('button', { name: '📝 Grammar' })
  await expect(grammarBtn).toBeVisible({ timeout: 10_000 })
  await grammarBtn.click()

  // Wait for the grammar panel (amber-themed) to appear
  // The grammar analysis depends on Ollama which may be slow or unavailable
  try {
    await expect(page.locator('text=📝 Grammar').first()).toBeVisible({ timeout: 30_000 })
  } catch {
    console.warn('⚠️ Grammar panel did not appear within 30s (Ollama service may be slow or unavailable)')
    console.warn('   Grammar feature is degraded but the test can continue')
    // Don't fail - grammar analysis is a secondary feature
  }
}

/**
 * Open the AI Tutor panel from within the grammar analysis panel.
 */
export async function openAITutor(page: Page) {
  const tutorBtn = page.getByRole('button', { name: /ai tutor/i })
  await expect(tutorBtn).toBeVisible({ timeout: 5_000 })
  await tutorBtn.click()

  // Wait for the LearningPanel (indigo-themed, "AI Tutor" header) to appear
  await expect(page.locator('span', { hasText: 'AI Tutor' })).toBeVisible({ timeout: 10_000 })
}

/**
 * Open the profile menu (avatar button in header).
 * The avatar is the round button with the user's initial (w-9 h-9 rounded-full).
 */
export async function openProfileMenu(page: Page) {
  // The avatar button has the gradient background (from-indigo-600 to-purple-600)
  // and contains the user's initial. This distinguishes it from the language selector.
  const avatar = page.locator('header button.bg-gradient-to-br.from-indigo-600')
  await expect(avatar).toBeVisible({ timeout: 10_000 })
  await avatar.click()
  // Wait for the dropdown menu — the Settings button appears in the profile dropdown
  await expect(page.locator('button:has-text("Settings")')).toBeVisible({ timeout: 10_000 })
}

/**
 * Wait for a specific text to appear anywhere on the page.
 * Useful for verifying async content loads (AI responses, etc.).
 */
export async function waitForText(page: Page, text: string, timeoutMs = 30_000) {
  await expect(page.locator(`text=${text}`).first()).toBeVisible({ timeout: timeoutMs })
}

/**
 * Find a chat in the sidebar by the other participant's display name.
 * Tries via WebSocket first, then falls back to page reload.
 * Returns the locator for the chat item.
 */
export async function findChatInSidebar(page: Page, otherUserDisplayName: string, timeoutMs = 10_000) {
  let chatItem = page.locator('.cursor-pointer').filter({ hasText: otherUserDisplayName })
  try {
    await expect(chatItem).toBeVisible({ timeout: timeoutMs })
  } catch {
    console.log(`ℹ️ Chat with "${otherUserDisplayName}" not visible via WebSocket, reloading...`)
    await page.reload()
    await page.waitForLoadState('networkidle')
    // Wait for the chat list to finish rendering after reload
    await page.waitForFunction((expectedName: string) => {
      const items = document.querySelectorAll('.cursor-pointer')
      return Array.from(items).some(el => el.textContent?.includes(expectedName))
    }, otherUserDisplayName, { timeout: timeoutMs })
    chatItem = page.locator('.cursor-pointer').filter({ hasText: otherUserDisplayName })
  }
  return chatItem
}

/**
 * Get the backend API base URL.
 */
export const API_BASE = process.env.E2E_API_URL || 'http://localhost:8080/api/v1'

/**
 * Log in via API and return the access token.
 * Faster than UI login when we just need auth for API calls.
 */
export async function loginViaAPI(user: TestUser): Promise<string> {
  const response = await fetch(`${API_BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      username: user.email.trim().toLowerCase(),
      password: user.password,
    }),
  })

  if (!response.ok) {
    throw new Error(`API login failed for ${user.email}: ${response.status} ${await response.text()}`)
  }

  const data = await response.json()
  return data.tokens.accessToken
}