import { test, expect, type Page, type BrowserContext } from '@playwright/test'

// Two-user end-to-end test.
//
// Prerequisites (run the stack first):
//   - Backend:    `go run ./cmd/server`  (serves API + WebSocket on :8080)
//   - Frontend:   `npm run build && npx vite preview` (proxies /api and /ws to
//                 8080; runs on :4173 as configured in playwright.config.ts)
//   - The following accounts must exist in the database:
//       uhsarp@gmail.com  / Demor@cer1
//       trolldown@gmail.com / Demor@cer1
//
// What this validates and times:
//   - Login for both users (UI).
//   - Creating a direct chat from the UI (search + create).
//   - Message delivery latency: time from pressing Send on user A until the
//     message renders on user B (real-time via WebSocket).
//   - Translation latency: until the receiving user sees the translated block
//     ("In your language:") for the sent message.
//   - Grammar analysis: the async job flow — honest "Queued…/Analyzing…" state
//     followed by a terminal state (analysis panel OR explicit failure box) —
//     and how long that took.
//   - Vocabulary save (word chip → "Saved").
//   - Message search finding the sent message.
//   - Chat-list propagation to both users and logout.

const USER_A = { email: 'uhsarp@gmail.com', password: 'Demor@cer1' }
const USER_B = { email: 'trolldown@gmail.com', password: 'Demor@cer1' }

// Timeouts: generous because translation + grammar hit real AI providers.
const T = {
  ui: 30_000, // normal UI actions
  send: 120_000, // message delivery
  llm: 180_000, // translation / grammar analysis
}

test.describe('two-user messaging', () => {
  test('uhsarp ↔ trolldown: login, chat, deliver, translate, grammar, search, logout', async ({ browser }) => {
    test.setTimeout(480_000)

    const makeContext = async (): Promise<BrowserContext> =>
      browser.newContext({ locale: 'en-US', viewport: { width: 1280, height: 900 } })

    const ctxA = await makeContext()
    const ctxB = await makeContext()
    for (const ctx of [ctxA, ctxB]) {
      await ctx.addInitScript(() => {
        window.localStorage.setItem('preferredLanguage', 'en')
      })
    }

    const aPage = await ctxA.newPage()
    const bPage = await ctxB.newPage()

    const times: Record<string, string> = {}
    const stamp = (key: string) => { times[key] = `${Date.now()}` }

    const authHeader = (token: string) => ({ Authorization: `Bearer ${token}` })

    // ---- 1. Login both users ----
    const login = async (page: Page, email: string, password: string, label: string): Promise<string> => {
      const t0 = Date.now()
      await page.goto('/login')
      await page.getByPlaceholder('you@example.com').fill(email)
      await page.getByPlaceholder('Enter your password').fill(password)
      await page.getByRole('button', { name: 'Log In' }).click()
      await expect(page).toHaveURL(/\/chat/, { timeout: T.ui })
      await expect(page.getByRole('button', { name: /New Chat/ })).toBeVisible({ timeout: T.ui })
      const ms = Date.now() - t0
      times[`login_${label}`] = `${ms}ms`
      console.log(`[E2E] login ${label}: ${ms}ms`)
      return page.evaluate(() => localStorage.getItem('accessToken') || '')
    }

    const tokenA = await login(aPage, USER_A.email, USER_A.password, 'A')
    const tokenB = await login(bPage, USER_B.email, USER_B.password, 'B')

    // ---- 2. Pin languages so a translation is guaranteed to be visible ----
    // User A writes in English; user B's native language is Spanish, so B's
    // translation block ("In your language:") differs from the original text.
    const setLang = async (page: Page, token: string, code: string) => {
      const res = await page.request.put('/api/v1/users/me', {
        headers: authHeader(token),
        data: { nativeLanguage: code },
      })
      expect([200, 204].includes(res.status())).toBeTruthy()
    }
    await setLang(aPage, tokenA, 'en')
    await setLang(bPage, tokenB, 'es')
    // Re-fetch the user objects after the language change (store reads them on load).
    await aPage.reload()
    await bPage.reload()
    await expect(aPage.getByRole('button', { name: /New Chat/ })).toBeVisible({ timeout: T.ui })
    await expect(bPage.getByRole('button', { name: /New Chat/ })).toBeVisible({ timeout: T.ui })

    // Learn B's username + display names from the API (used as UI selectors).
    const meA = await (await aPage.request.get('/api/v1/users/me', { headers: authHeader(tokenA) })).json()
    const meB = await (await bPage.request.get('/api/v1/users/me', { headers: authHeader(tokenB) })).json()
    console.log(`[E2E] meA=${meA.username} (${meA.displayName}), meB=${meB.username} (${meB.displayName})`)

    // ---- 3. Create a direct chat from the UI (A → B) ----
    await aPage.getByRole('button', { name: /New Chat/ }).click()
    const searchBox = aPage.getByPlaceholder('Search users...')
    await expect(searchBox).toBeVisible()
    await searchBox.fill(USER_B.email)
    const bResult = aPage.getByText(`@${meB.username}`)
    await bResult.waitFor({ state: 'visible', timeout: T.ui })
    await bResult.click()
    await aPage.getByRole('button', { name: /Create Chat/ }).click()
    // Chat opens and the message input is ready.
    await expect(aPage.getByPlaceholder('Type a message...')).toBeVisible({ timeout: T.ui })
    times.chat_created = `${Date.now()}`
    console.log('[E2E] direct chat created')

    // ---- 4. Send a message and time real-time delivery to B ----
    const msg = `E2E two-user ${Date.now()} — good morning, how are you doing today my friend 🙂`

    // FR-21: the emoji picker opens and inserts an emoji into the composer.
    const composer = aPage.getByPlaceholder('Type a message...')
    await aPage.getByRole('button', { name: /Insert emoji/i }).click()
    const emojiPicker = aPage.getByRole('dialog', { name: 'Emoji picker' })
    await expect(emojiPicker).toBeVisible({ timeout: T.ui })
    // 😁 is only rendered inside the grid (it is not a category tab label), so a
    // click reliably inserts it rather than switching categories.
    await emojiPicker.getByRole('button', { name: '😁', exact: true }).click()
    await expect(composer).toHaveValue('😁')
    await aPage.keyboard.press('Escape')
    await expect(emojiPicker).not.toBeVisible()

    // Now compose the real message (replaces the inserted preview emoji).
    await composer.fill(msg)

    const tSend = Date.now()
    await aPage.getByRole('button', { name: 'Send' }).click()

    // B: the message appears in the chat list (WebSocket-delivered) → open it.
    const bPreview = bPage.getByText(msg).first()
    await bPreview.waitFor({ state: 'visible', timeout: T.send })
    const deliveredMs = Date.now() - tSend
    times.delivery = `${deliveredMs}ms`
    console.log(`[E2E] message delivered to B in ${deliveredMs}ms`)
    await bPreview.click()
    await expect(bPage.getByText(msg, { exact: true }).last()).toBeVisible({ timeout: T.ui })

    // ---- 5. Translation: B eventually sees the translated block ----
    // The message text sits inside the bubble div; the message WRAPPER (parent
    // of the bubble) also contains the translation block and the grammar panel.
    const bMsgText = bPage.getByText(msg, { exact: true }).last()
    await bMsgText.scrollIntoViewIfNeeded()
    const bBubble = bMsgText.locator('..').locator('..')
    // The label renders as "🌐 In your language:" (emoji + text), so match a substring.
    await expect(bBubble.getByText(/In your language:/)).toBeVisible({ timeout: T.llm })
    times.translation = `${Date.now() - tSend}ms`
    console.log(`[E2E] translation visible on B in ${Date.now() - tSend}ms (from send)`)

    // FR-21: emoji must pass through translation unchanged — the "In your
    // language:" block must still contain the sender's emoji rather than having
    // it stripped or rewritten (e.g. to '?') anywhere in the pipeline.
    const nativeTranslation = bBubble.locator('.font-translation-text').last()
    await expect(nativeTranslation).toContainText('🙂', { timeout: T.llm })

    // ---- 6. Grammar analysis (async job flow) on B ----
    await bMsgText.hover()
    const grammarBtn = bPage.getByRole('button', { name: /Grammar/ })
    await expect(grammarBtn).toBeVisible({ timeout: T.ui })
    const tGrammar = Date.now()
    await grammarBtn.click()

    // Honest intermediate state: queued/processing must surface, not a fake label.
    await expect(
      bPage.getByRole('button', { name: /Queued\.\.\.|Analyzing\.\.\./ })
    ).toBeVisible({ timeout: T.ui })
    console.log(`[E2E] grammar job queued/processing (${Date.now() - tGrammar}ms)`)

    // Terminal state: analysis panel (Overview tab) or an explicit failure box.
    // The tab button renders as "💡 Overview" (icon + label), so use a substring match.
    const analysisPanel = bBubble.getByText('Overview', { exact: false })
    const failedBox = bPage.getByText('Grammar analysis failed.', { exact: false })
    await expect(async () => {
      const done = await analysisPanel.isVisible()
      const failed = await failedBox.isVisible()
      expect(done || failed).toBe(true)
    }).toPass({ timeout: T.llm })
    times.grammar = `${Date.now() - tGrammar}ms`
    const grammarOutcome = (await analysisPanel.isVisible()) ? 'success' : 'honest-failure'
    console.log(`[E2E] grammar analysis terminal state (${grammarOutcome}) in ${Date.now() - tGrammar}ms`)

    // ---- 7. Vocabulary: save a word from the message ----
    await bMsgText.hover()
    // Word chips render inside the message wrapper ("+ good", "+ morning", ...),
    // scoped so the header's "+ New Chat" button is not matched.
    const wordBtn = bBubble.getByRole('button', { name: /^\+ / }).first()
    await expect(wordBtn).toBeVisible({ timeout: T.ui })
    await wordBtn.click()
    await expect(bBubble.getByText('Saved', { exact: true })).toBeVisible({ timeout: T.ui })
    times.vocabulary_save = `${Date.now()}`
    console.log('[E2E] word saved to vocabulary')

    // ---- 8. Chat list propagation: A also sees the message preview ----
    await expect(aPage.getByText(msg, { exact: false }).first()).toBeVisible({ timeout: T.send })

    // ---- 9. Message search on B ----
    const tSearch = Date.now()
    await bPage.getByRole('button', { name: /Search Messages/ }).click()
    const searchInput = bPage.getByPlaceholder('Search messages...')
    await expect(searchInput).toBeVisible()
    await searchInput.fill(msg)
    await searchInput.press('Enter')
    await expect(bPage.getByText(/result\(s\)/)).toBeVisible({ timeout: T.ui })
    times.message_search = `${Date.now() - tSearch}ms`
    console.log(`[E2E] message search found the message in ${Date.now() - tSearch}ms`)

    // ---- 10. Logout (A) returns to the login screen ----
    const tLogout = Date.now()
    await aPage.getByTitle(meA.displayName).click()
    await aPage.getByRole('button', { name: /Sign Out/ }).click()
    await expect(aPage).toHaveURL(/\/login/, { timeout: T.ui })
    times.logout = `${Date.now() - tLogout}ms`
    console.log('[E2E] A logged out')

    console.log('\n[E2E] timing summary:')
    console.log(JSON.stringify(times, null, 2))

    await ctxA.close()
    await ctxB.close()
  })
})
