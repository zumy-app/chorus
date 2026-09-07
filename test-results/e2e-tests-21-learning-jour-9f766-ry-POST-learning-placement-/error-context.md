# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: e2e\tests\21-learning-journey.spec.ts >> @C-02 @learning @wireframe-placement @wireframe-scenarios @wireframe-real-talk @wireframe-streak >> C-02-01 — placement start → vocab + reading answers → results summary (POST /learning/placement/*)
- Location: e2e\tests\21-learning-journey.spec.ts:12:7

# Error details

```
Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
Call log:
  - navigating to "/login", waiting until "load"

```

# Test source

```ts
  1   | import { Page, expect } from '@playwright/test'
  2   | import { TestUser } from './users'
  3   | 
  4   | /**
  5   |  * Shared test helpers for Chorus E2E tests.
  6   |  *
  7   |  * These helpers encapsulate common UI flows (login, create chat, send message)
  8   |  * so test files stay readable and focused on assertions.
  9   |  */
  10  | 
  11  | /**
  12  |  * Log in a user via the UI.
  13  |  * Assumes the app is on /login or / (will navigate if needed).
  14  |  */
  15  | export async function loginAsUser(page: Page, user: TestUser) {
> 16  |   await page.goto('/login')
      |              ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  17  | 
  18  |   // Wait for the login form to render
  19  |   await expect(page.locator('input[type="email"]')).toBeVisible()
  20  | 
  21  |   await page.locator('input[type="email"]').fill(user.email)
  22  |   await page.locator('input[type="password"]').fill(user.password)
  23  |   await page.getByRole('button', { name: /log in/i }).click()
  24  | 
  25  |   // Wait for redirect to /chat
  26  |   await page.waitForURL('**/chat', { timeout: 30_000 })
  27  | 
  28  |   // Verify the header (Chorus logo) is visible — confirms we're in the app
  29  |   await expect(page.locator('h1', { hasText: 'Chorus' })).toBeVisible()
  30  | }
  31  | 
  32  | /**
  33  |  * Create a direct chat with another user by searching for them.
  34  |  * Assumes the user is already logged in and on /chat.
  35  |  */
  36  | export async function createDirectChat(page: Page, searchQuery: string) {
  37  |   // Click "+ New Chat"
  38  |   await page.getByRole('button', { name: /new chat/i }).click()
  39  | 
  40  |   // Wait for modal
  41  |   await expect(page.locator('h2', { hasText: 'New Chat' })).toBeVisible()
  42  | 
  43  |   // Direct Chat should be selected by default
  44  |   await expect(page.getByRole('button', { name: 'Direct Chat' })).toBeVisible()
  45  | 
  46  |   // Search for the user
  47  |   await page.locator('input[placeholder="Search users..."]').fill(searchQuery)
  48  | 
  49  |   // Wait for search results to appear (debounced — needs at least 2 chars)
  50  |   await expect(page.locator('text=Search Results')).toBeVisible({ timeout: 15_000 })
  51  | 
  52  |   // Click the first search result
  53  |   const firstResult = page.locator('.space-y-2 > div').first()
  54  |   await firstResult.click()
  55  | 
  56  |   // Click "Create Chat"
  57  |   await page.getByRole('button', { name: /create chat/i }).click()
  58  | 
  59  |   // Wait for modal to close and chat area to load
  60  |   await expect(page.locator('h2', { hasText: 'New Chat' })).not.toBeVisible({ timeout: 10_000 })
  61  | }
  62  | 
  63  | /**
  64  |  * Send a message in the currently active chat.
  65  |  * Assumes a chat is already open.
  66  |  */
  67  | export async function sendMessage(page: Page, text: string) {
  68  |   const input = page.locator('textarea[placeholder="Type a message..."]')
  69  |   await expect(input).toBeVisible()
  70  |   await input.fill(text)
  71  |   // Use exact Send aria-label to avoid strict mode violation (two Send buttons: composer + header)
  72  |   const sendBtn = page.getByRole('button', { name: 'Send', exact: true })
  73  |   if ((await sendBtn.count()) > 1) {
  74  |     await sendBtn.first().click()
  75  |   } else {
  76  |     await sendBtn.click()
  77  |   }
  78  | 
  79  |   // Wait for the message to appear in the chat area
  80  |   // Use .last() to target the most recently sent message (handles duplicates from prior runs)
  81  |   await expect(page.locator('.break-words', { hasText: text }).last()).toBeVisible({ timeout: 15_000 })
  82  | }
  83  | 
  84  | /**
  85  |  * Wait for a translation to appear in a message bubble.
  86  |  *
  87  |  * The backend sends translations asynchronously via WebSocket after the
  88  |  * `new_message` event. The `message_updated` event delivers translations.
  89  |  * In the UI, this renders as "🌐 In your language:" followed by the text.
  90  |  *
  91  |  * @param page Playwright page
  92  |  * @param originalText The original message text to locate the bubble
  93  |  * @param timeoutMs How long to wait (translator-engine cold start can be slow)
  94  |  */
  95  | export async function waitForTranslation(
  96  |   page: Page,
  97  |   originalText: string,
  98  |   timeoutMs = 60_000,
  99  |   opts?: { critical?: boolean },
  100 | ) {
  101 |   // Find the message bubble containing the original text
  102 |   // Use .last() to target the most recent message (handles duplicates)
  103 |   const bubble = page.locator('.break-words', { hasText: originalText }).last().locator('..')
  104 | 
  105 |   // Wait for the "In your language:" section to appear
  106 |   try {
  107 |     await expect(
  108 |       bubble.locator('text=🌐 In your language:'),
  109 |     ).toBeVisible({ timeout: timeoutMs })
  110 |   } catch (e) {
  111 |     if (opts?.critical) throw e
  112 |     console.warn(`⚠️ Translation for "${originalText.slice(0,30)}..." not visible within ${timeoutMs}ms (non-critical, swallowed)`)
  113 |   }
  114 | }
  115 | 
  116 | /**
```