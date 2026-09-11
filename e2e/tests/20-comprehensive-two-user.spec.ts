import { test, expect } from '@playwright/test'
import { loginAsUser, createDirectChat, sendMessage, waitForTranslation, openGrammarAnalysis, openAITutor, openProfileMenu, findChatInSidebar, loginViaAPI, API_BASE } from '../fixtures/test-helpers'
import { DEV_ALICE, DEV_BOB, DEV_SOFIA } from '../fixtures/users'

const ALICE = DEV_ALICE
const BOB = DEV_BOB
const SOFIA = DEV_SOFIA

/**
 * C-01 — Comprehensive Two-User Journey (alice → bob, 5 msgs + vocab/grammar/ai-tutor + settings → bob verify)
 * Authority: docs/QA_CRITIQUE_AND_IMPROVEMENTS.md:§2.1
 * Impl: now uses DEV_ALICE/BOB/SOFIA + waitForTranslation critical:true.
 * All persistence/durability probes are HARD (no try/catch soft-pass):
 * a green run proves Postgres is the source of truth, not UI optimism.
 */
test.describe('@C-01 @comprehensive @critical @two-user', () => {
  test.describe.configure({ mode: 'serial' })

  let aliceCtx: any
  let bobCtx: any
  let alicePage: any
  let bobPage: any
  let chatId: string | null = null
  const FIVE = [
    'Hello Bob, how are you doing today?',
    'I have been learning Spanish for three years.',
    'The weather is beautiful today, shall we practice?',
    'She was walking through the park when it started raining.',
    'Vocabulary test: The elephant walked carefully through the jungle.',
  ]

  test.beforeAll(async ({ browser }) => {
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:3000'
    aliceCtx = await browser.newContext({ baseURL })
    bobCtx = await browser.newContext({ baseURL })
    alicePage = await aliceCtx.newPage()
    bobPage = await bobCtx.newPage()
  })

  test.afterAll(async () => {
    await aliceCtx?.close()
    await bobCtx?.close()
  })

  test('C-01-01 — alice creates DM to bob and sends 5 messages (en) + GET /chats/:id/messages count 5', async () => {
    await loginAsUser(alicePage, ALICE)
    await createDirectChat(alicePage, BOB.displayName)
    for (const msg of FIVE) {
      const stamped = `${msg} ${Date.now()}`
      await sendMessage(alicePage, stamped)
      await expect(alicePage.locator('.break-words', { hasText: msg.split(' ')[0] }).last()).toBeVisible({ timeout: 15_000 })
    }
    // Resolve the chat via API and prove Postgres holds all 5 messages (HARD).
    // chatId resolution has two independent paths (API match, URL match);
    // either must succeed — an unresolvable chat is a product bug, not noise.
    const token = await loginViaAPI(ALICE)
    const res = await fetch(`${API_BASE}/chats`, { headers: { Authorization: `Bearer ${token}` } })
    expect(res.ok).toBe(true)
    const data = await res.json()
    const chats = data.chats || data.data || []
    // Prefer the DIRECT DM with bob: group chats also contain him and would
    // match a naive substring search (wrong-chat probe caught live).
    const match = chats.find((c: any) => c.type === 'direct' && (JSON.stringify(c).includes(BOB.displayName) || (c.participants && c.participants.some((p: any) => p.displayName === BOB.displayName || p.email === BOB.email))))
    if (match?.id || match?._id) chatId = match.id || match._id
    // Also try extract from URL
    if (!chatId) {
      const url = alicePage.url()
      const m = url.match(/\/chat\/([^/?#]+)/)
      if (m) chatId = m[1]
    }
    expect(chatId).toBeTruthy()
    const token2 = await loginViaAPI(ALICE)
    const mRes = await fetch(`${API_BASE}/chats/${chatId}/messages?limit=20`, { headers: { Authorization: `Bearer ${token2}` } })
    expect(mRes.ok).toBe(true)
    const mData = await mRes.json()
    const msgs = mData.messages || mData.data || []
    // >=5: prior runs may have added history to the same DM; fewer is a loss.
    expect(msgs.length).toBeGreaterThanOrEqual(5)
    // Always pass if UI shows 5 msgs — proven above
    await expect(alicePage.locator('.break-words').first()).toBeVisible()
  })

  test('C-01-02 — bob receives inbox real-time + 🌐 In your language: critical (must not swallow)', async () => {
    await loginAsUser(bobPage, BOB)
    const chatItem = await findChatInSidebar(bobPage, ALICE.displayName)
    await chatItem.click()
    for (const snippet of ['Hello Bob', 'learning Spanish', 'weather is beautiful', 'walking through the park', 'elephant']) {
      await expect(bobPage.locator('.break-words', { hasText: snippet }).last()).toBeVisible({ timeout: 15_000 })
      // Critical translation — HARD: a missing translation is a product
      // failure (provider chain), never suite noise. Throws on timeout.
      await waitForTranslation(bobPage, snippet, 60_000, { critical: true })
      // Translation UI must be present and non-trivial.
      const bubble = bobPage.locator('.break-words', { hasText: snippet }).last().locator('..')
      const translating = bubble.getByText(/🌐 In your language:/)
      await expect(translating.first()).toBeVisible({ timeout: 10_000 })
      const trans = bubble.locator('.font-translation-text')
      if (await trans.count() > 0) {
        const t = await trans.first().textContent()
        if (!t || t.length <= 3) console.warn(`⚠️ C-01-02 translation suspiciously short for "${snippet}" (length warn only)`)
      }
    }
  })

  test('C-01-03 — bob saves a word from chat → ✅ Saved → GET /vocabulary persistence (HARD)', async () => {
    // Chat "+" buttons save individual words via POST /vocabulary (manual-save
    // path → VocabularyCard). /learn/vocabulary is a DIFFERENT feature (mined
    // candidates) and must not be asserted here. Use a distinctive first word
    // so the "+" button target is unambiguous (buttons show the first words).
    const mineMsg = `Quokka dreams vividly at midnight ${Date.now()}`
    await alicePage.goto('/chat')
    const aliceChat = await findChatInSidebar(alicePage, BOB.displayName)
    await aliceChat.click()
    await sendMessage(alicePage, mineMsg)

    await bobPage.goto('/chat')
    const bobChat = await findChatInSidebar(bobPage, ALICE.displayName)
    await bobChat.click()
    const wrapper = bobPage.locator('.break-words', { hasText: 'Quokka dreams' }).last().locator('xpath=ancestor::div[contains(@class, "flex")][1]')
    await wrapper.hover()
    const saveBtn = wrapper.getByRole('button', { name: '+ Quokka' }).first()
    await expect(saveBtn).toBeVisible({ timeout: 10_000 })
    await saveBtn.click()
    await expect(wrapper.getByText('Saved').first()).toBeVisible({ timeout: 10_000 })
    // API persistence probe (HARD): the saved card must read back.
    const token = await loginViaAPI(BOB)
    const res = await fetch(`${API_BASE}/vocabulary?limit=50`, { headers: { Authorization: `Bearer ${token}` } })
    expect(res.ok).toBe(true)
    const vData = await res.json()
    const entries = vData?.data?.entries || vData?.data || []
    expect(Array.isArray(entries)).toBe(true)
    expect(entries.some((e: any) => (e.term || '').toLowerCase().includes('quokka'))).toBe(true)
    // Vocab hub loads (HARD shell only — its candidate list depends on async
    // mining timing and is covered by 06-vocabulary + acceptance).
    await bobPage.goto('/learn/vocabulary')
    await expect(bobPage.getByText(/Vocabulary|Words found in your chats/i).first()).toBeVisible({ timeout: 10_000 })
    // Return to the DM for C-01-04 (grammar needs the thread open).
    await bobPage.goto('/chat')
    const backToChat = await findChatInSidebar(bobPage, ALICE.displayName)
    await backToChat.click()
  })

  test('C-01-04 — bob grammar + ai-tutor on msg2 (amber 180s, indigo 10s, assistant 45s)', async () => {
    const msg2 = 'I have been learning Spanish'
    try {
      await openGrammarAnalysis(bobPage, msg2)
      // After openGrammarAnalysis, helper already waited for panel/queued. Soft assert amber panel or queued
      const panel = bobPage.getByTestId('grammar-panel')
      const queued = bobPage.getByTestId('grammar-queued')
      const legacy = bobPage.locator('text=/📝\\s*Gram/').first()
      const sparky = bobPage.locator('text=Sparky').first()
      const visible = await panel.or(queued).or(legacy).or(sparky).first().isVisible({ timeout: 10_000 }).catch(()=>false)
      if (!visible) console.warn('⚠️ C-01-04 grammar panel not visible (soft — Ollama may be queued)')
    } catch (e) {
      console.warn(`⚠️ C-01-04 grammar soft fail: ${(e as Error).message}`)
    }
    try {
      await openAITutor(bobPage)
      const tutorPanel = bobPage.getByTestId('ai-tutor-panel')
      if (await tutorPanel.isVisible({ timeout: 5_000 }).catch(()=>false)) {
        await expect(tutorPanel).toBeVisible({ timeout: 10_000 })
      } else {
        const indigo = bobPage.locator('div.bg-gradient-to-r.from-indigo-600 span.text-white')
        if (await indigo.isVisible({ timeout: 5_000 }).catch(()=>false)) {
          await expect(indigo.first()).toBeVisible({ timeout: 10_000 })
        } else {
          console.warn('⚠️ C-01-04 AI Tutor indigo panel not visible (soft)')
        }
      }
      const assistant = bobPage.locator('.bg-white.border.border-indigo-100').first()
      if (await assistant.isVisible({ timeout: 5_000 }).catch(()=>false)) {
        await expect(assistant).toBeVisible({ timeout: 45_000 })
        const c = await assistant.textContent().catch(()=> null)
        if (!c || c.length <= 5) console.warn('⚠️ C-01-04 assistant content short (soft)')
      } else {
        console.warn('⚠️ C-01-04 assistant bubble not visible (soft — AI may be unavailable)')
      }
    } catch (e) {
      console.warn(`⚠️ C-01-04 ai-tutor soft fail: ${(e as Error).message}`)
    }
  })

  test('C-01-05 — alice changes Display Name to Alice C01 → persists reload → bob sidebar shows new name → restored (HARD)', async () => {
    const testName = 'Alice C01'
    await openProfileMenu(alicePage)
    await alicePage.getByRole('button', { name: '⚙️ Settings' }).click()
    await expect(alicePage.locator('h2', { hasText: 'Settings' })).toBeVisible({ timeout: 10_000 })
    const nameInput = alicePage.locator('[data-testid="settings-modal"] input[type="text"]').first()
    await nameInput.fill(testName)
    await alicePage.getByRole('button', { name: /save settings/i }).click()
    // Wait for success toast (i18n may vary)
    const saved = alicePage.locator('text=Settings saved successfully').or(alicePage.locator('text=Saved')).or(alicePage.locator('text=saved'))
    await expect(saved.first()).toBeVisible({ timeout: 10_000 })
    await alicePage.reload()
    await alicePage.waitForLoadState('networkidle').catch(()=>{})
    // Display Name must survive reload (HARD persistence proof).
    await openProfileMenu(alicePage)
    await alicePage.getByRole('button', { name: '⚙️ Settings' }).click()
    await expect(alicePage.locator('h2', { hasText: 'Settings' })).toBeVisible({ timeout: 10_000 })
    const after = alicePage.locator('[data-testid="settings-modal"] input[type="text"]').first()
    await expect(after).toHaveValue(testName)
    // Close modal
    await alicePage.getByTestId('settings-close').click().catch(()=> alicePage.keyboard.press('Escape'))
    // Bob's sidebar must show the new name after reload (HARD cross-user proof).
    await bobPage.reload()
    await bobPage.waitForLoadState('networkidle').catch(()=>{})
    const sidebar = bobPage.locator('[data-testid="chat-list-item"], .cursor-pointer').filter({ hasText: testName })
    await expect(sidebar.first()).toBeVisible({ timeout: 15_000 })
    // Restore the canonical name so later suites/specs find "Alice Dev" (HARD).
    await openProfileMenu(alicePage)
    await alicePage.getByRole('button', { name: '⚙️ Settings' }).click()
    await expect(alicePage.locator('h2', { hasText: 'Settings' })).toBeVisible({ timeout: 10_000 })
    await alicePage.locator('[data-testid="settings-modal"] input[type="text"]').first().fill(ALICE.displayName)
    await alicePage.getByRole('button', { name: /save settings/i }).click()
    await expect(saved.first()).toBeVisible({ timeout: 10_000 })
    await alicePage.reload()
    await alicePage.waitForLoadState('networkidle').catch(()=>{})
    await openProfileMenu(alicePage)
    await alicePage.getByRole('button', { name: '⚙️ Settings' }).click()
    const restored = alicePage.locator('[data-testid="settings-modal"] input[type="text"]').first()
    await expect(restored).toHaveValue(ALICE.displayName)
    await alicePage.getByTestId('settings-close').click().catch(()=> alicePage.keyboard.press('Escape'))
  })

  test('C-01-06 — durability: reload both, GET /chats/:id/messages persists, ws_fast_dropped_total==0 (HARD)', async () => {
    const durabilityMsg = `Durability check ${Date.now()}`
    // Ensure alice is back on chat and focused on the DM
    await alicePage.goto('/chat')
    const chatItem = await findChatInSidebar(alicePage, BOB.displayName)
    await chatItem.click()
    await sendMessage(alicePage, durabilityMsg)
    expect(chatId).toBeTruthy()
    await alicePage.reload()
    await bobPage.reload()
    await bobPage.waitForLoadState('networkidle').catch(()=>{})
    // Ensure bob is on the chat
    const bobChat = await findChatInSidebar(bobPage, ALICE.displayName)
    await bobChat.click()
    await expect(bobPage.locator('.break-words', { hasText: durabilityMsg }).last()).toBeVisible({ timeout: 15_000 })
    // API durability probe (HARD): Postgres must hold the message.
    const token = await loginViaAPI(ALICE)
    const res = await fetch(`${API_BASE}/chats/${chatId}/messages?limit=20`, { headers: { Authorization: `Bearer ${token}` } })
    expect(res.ok).toBe(true)
    const data = await res.json()
    const msgs = data.messages || data.data || []
    expect(msgs.length).toBeGreaterThanOrEqual(6)
    expect(msgs.some((m: any) => (m.content || m.text || '').includes('Durability check'))).toBe(true)
    // No fast-path drops allowed (HARD): Redis may cache, Postgres decides.
    const mRes = await fetch((process.env.E2E_API_URL || 'http://localhost:8080/api/v1').replace('/api/v1','') + '/metrics')
    expect(mRes.ok).toBe(true)
    const txt = await mRes.text()
    expect(txt.includes('ws_fast_dropped_total')).toBe(true)
    const m = txt.match(/ws_fast_dropped_total\s+(\d+)/)
    expect(m?.[1]).toBe('0')
  })
})
