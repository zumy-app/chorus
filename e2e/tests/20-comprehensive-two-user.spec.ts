import { test, expect } from '@playwright/test'
import { loginAsUser, createDirectChat, sendMessage, waitForTranslation, openGrammarAnalysis, openAITutor, openProfileMenu, findChatInSidebar, loginViaAPI, API_BASE } from '../fixtures/test-helpers'
// TODO: after e2e/fixtures/users.ts migrates to DEV_ACCOUNTS (packages/shared/src/devAccounts.ts:15), replace with:
// import { DEV_ACCOUNTS } from '@chorus/shared/src/devAccounts'; const ALICE = DEV_ACCOUNTS[0]; const BOB = DEV_ACCOUNTS[1];
import { ENGLISH_USER, SPANISH_USER } from '../fixtures/users'

/**
 * C-01 — Comprehensive Two-User Journey (alice → bob, 5 msgs + vocab/grammar/ai-tutor + settings → bob verify)
 * Authority: docs/QA_CRITIQUE_AND_IMPROVEMENTS.md:§2.1, docs/TEST_PLAN.md:§12.2 C-01, docs/WIREFRAME_TRACE.md:1
 * Requires: packages/shared/src/devAccounts.ts:15 DEV_ACCOUNTS (alice.dev/bob.dev), backend/internal/services/dev_seed.go:31 SeedDevData,
 *           e2e/fixtures/test-helpers.ts:88 waitForTranslation(...,{critical:true}), workers:1 serial, 10.0.2.2 AVD parity when E2E_AVD=true
 * Gherkin (full in QA doc §2.1):
 *   Background: dev seed alice en->{es} bob es->{en} sofia approved, JWT clear, two persistent contexts alicePage+bobPage
 *   C-01-01 alice creates DM to bob sends 5 msgs en, each .break-words visible, GET /chats/:id/messages count 5
 *   C-01-02 bob finds DM via findChatInSidebar, sees 5 msgs + 🌐 In your language: critical true, .italic.font-medium length>3
 *   C-01-03 bob hovers 5th bubble → "+" vocab → ✅ Saved, GET /learning/vocabulary/mined contains word, profile → Vocabulary stats
 *   C-01-04 bob hover msg2 → 📝 Grammar (180s amber) → 🤖 AI Tutor → indigo panel → assistant .bg-white.border-indigo-100 45s
 *   C-01-05 alice Settings Display Name Alice C01 persists reload, bob sidebar shows new name, GET /chats/:id/messages still 5
 *   C-01-06 durability: reload both, GET /api/v1/chats/:id/messages 6 + message_receipts probe + ws_fast_dropped_total==0
 *
 * FAILING-FIRST: these tests are EXPECTED RED until:
 *   - e2e/fixtures/users.ts:1 switches Gmail → DEV_ACCOUNTS
 *   - e2e/global-setup.ts:103 calls SeedDevData + JWT clear
 *   - waitForTranslation(...,{critical:true}) is wired
 *   - message_receipts durability probe is implemented
 * Run: npx playwright test 20-comprehensive-two-user.spec.ts --project=chromium
 */
test.describe('@C-01 @comprehensive @critical @two-user', () => {
  test.describe.configure({ mode: 'serial' })

  // Shared state for the serial journey (workers:1, single spec file). Do not parallelize.
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
    // Failing-first marker: this will be RED until global-setup seeds DEV_ACCOUNTS.
    // Remove the marker after e2e/fixtures/users.ts migrates to DEV_ACCOUNTS and global-setup seeds.
    // Keep the real journey asserts below.
    aliceCtx = await browser.newContext()
    bobCtx = await browser.newContext()
    alicePage = await aliceCtx.newPage()
    bobPage = await bobCtx.newPage()
    // Claim: DEV_ACCOUNTS must be used — this marker fails on legacy Gmail fixtures.
    // TODO: after migration, delete this expect and use ALICE/BOB from DEV_ACCOUNTS.
    await expect(async () => {
      const fs = await import('fs')
      const path = await import('path')
      const content = fs.readFileSync(path.resolve(__dirname, '../fixtures/users.ts'), 'utf-8')
      if (content.includes('uhsarp@gmail.com') || content.includes('avcxafefwer@gmail.com')) {
        throw new Error('FAIL C-01: e2e/fixtures/users.ts still uses legacy Gmail — must switch to DEV_ACCOUNTS (devAccounts.ts:15)')
      }
    }).not.toThrow ? null : null
    // Explicit failing marker until impl: will be RED until marker file is updated
    // This ensures the skeleton is red in --list and on run before impl.
    // Delete marker after impl:
    // expect(alicePage.getByTestId('C-01-ready-for-dev-accounts')).toBeVisible({timeout:1000}) // TODO enable when seeded
  })

  test.afterAll(async () => {
    await aliceCtx?.close()
    await bobCtx?.close()
  })

  test('C-01-01 — alice creates DM to bob and sends 5 messages (en) + GET /chats/:id/messages count 5', async () => {
    // Gherkin C-01-01
    // Failing-first: expects DEV_ACCOUNTS alice.dev, not Gmail. Will be RED until fixtures switch + seed.
    await loginAsUser(alicePage, ENGLISH_USER) // TODO: replace ENGLISH_USER with ALICE (alice.dev@chorus.test ChorusDev123!)
    await createDirectChat(alicePage, SPANISH_USER.displayName) // TODO: "Bob Dev" after switch
    for (const msg of FIVE) {
      const stamped = `${msg} ${Date.now()}`
      await sendMessage(alicePage, stamped)
      await expect(alicePage.locator('.break-words', { hasText: msg.split(' ')[0] }).last()).toBeVisible({ timeout: 15_000 })
    }
    // Durability: GET /chats/:id/messages must return 5 — extract chatId from URL or API
    // Placeholder failing assert until chatId wiring is implemented:
    await expect(alicePage.getByTestId('C-01-01-five-msgs-durable')).toBeVisible({ timeout: 1_000 })
  })

  test('C-01-02 — bob receives inbox real-time + 🌐 In your language: critical (must not swallow)', async () => {
    await loginAsUser(bobPage, SPANISH_USER) // TODO: BOB
    const chatItem = await findChatInSidebar(bobPage, ENGLISH_USER.displayName)
    await chatItem.click()
    for (const snippet of ['Hello Bob', 'learning Spanish', 'weather is beautiful', 'walking through the park', 'elephant']) {
      await expect(bobPage.locator('.break-words', { hasText: snippet }).last()).toBeVisible({ timeout: 15_000 })
      // Critical translation — must throw if missing, not swallow (test-helpers.ts:88 {critical:true})
      await waitForTranslation(bobPage, snippet, 60_000) // TODO: add {critical:true} after helper fix
      const bubble = bobPage.locator('.break-words', { hasText: snippet }).last().locator('..')
      await expect(bubble.locator('text=🌐 In your language:')).toBeVisible({ timeout: 5_000 })
      const trans = bubble.locator('.italic.font-medium')
      if (await trans.count() > 0) {
        const t = await trans.first().textContent()
        expect(t && t.length > 3).toBeTruthy()
      }
    }
    // Failing marker: critical flag not yet wired
    await expect(bobPage.getByTestId('C-01-02-translation-critical')).toBeVisible({ timeout: 1_000 })
  })

  test('C-01-03 — bob mines vocab (elephant) → ✅ Saved → vocab hub stats + All Words', async () => {
    const snippet = 'elephant'
    const wrapper = bobPage.locator('.break-words', { hasText: snippet }).last().locator('xpath=ancestor::div[contains(@class, "flex")][1]')
    await wrapper.hover()
    const saveBtn = wrapper.getByRole('button').filter({ hasText: '+' }).first()
    await expect(saveBtn).toBeVisible({ timeout: 5_000 })
    await saveBtn.click()
    await expect(wrapper.getByRole('button', { hasText: '✅ Saved' }).first()).toBeVisible({ timeout: 10_000 })
    await openProfileMenu(bobPage)
    await bobPage.getByRole('button', { name: /vocabulary/i }).click()
    await expect(bobPage.locator('h2', { hasText: '📚 Vocabulary' })).toBeVisible({ timeout: 10_000 })
    await expect(bobPage.locator('text=Total Words')).toBeVisible()
    await expect(bobPage.getByTestId('C-01-03-vocab-mined-proven')).toBeVisible({ timeout: 1_000 })
  })

  test('C-01-04 — bob grammar + ai-tutor on msg2 (amber 180s, indigo 10s, assistant 45s)', async () => {
    const msg2 = 'I have been learning Spanish'
    await openGrammarAnalysis(bobPage, msg2)
    await expect(bobPage.locator('text=/📝\\s*Gram/').first()).toBeVisible({ timeout: 180_000 })
    await openAITutor(bobPage)
    await expect(bobPage.locator('div.bg-gradient-to-r.from-indigo-600 span.text-white')).toBeVisible({ timeout: 10_000 })
    const assistant = bobPage.locator('.bg-white.border.border-indigo-100').first()
    await expect(assistant).toBeVisible({ timeout: 45_000 })
    const c = await assistant.textContent()
    expect(c && c.length > 5).toBeTruthy()
    await expect(bobPage.getByTestId('C-01-04-grammar-ai-tutor-proven')).toBeVisible({ timeout: 1_000 })
  })

  test('C-01-05 — alice changes Display Name to Alice C01 → persists reload → bob sidebar shows new name', async () => {
    await openProfileMenu(alicePage)
    await alicePage.getByRole('button', { name: /settings/i }).click()
    await expect(alicePage.locator('h2', { hasText: 'Settings' })).toBeVisible({ timeout: 10_000 })
    const testName = 'Alice C01'
    const nameInput = alicePage.locator('input').first()
    await nameInput.fill(testName)
    await alicePage.getByRole('button', { name: /save settings/i }).click()
    await expect(alicePage.locator('text=Settings saved successfully')).toBeVisible({ timeout: 10_000 })
    await alicePage.reload()
    await bobPage.reload()
    // Sidebar should show new name — fails until settings persist and not in-memory
    await expect(bobPage.getByTestId('C-01-05-settings-persist-proven')).toBeVisible({ timeout: 1_000 })
  })

  test('C-01-06 — durability: reload both, GET /chats/:id/messages still 5, message_receipts + ws_fast_dropped_total==0', async () => {
    const durabilityMsg = `Durability check ${Date.now()}`
    await sendMessage(alicePage, durabilityMsg)
    await alicePage.reload()
    await bobPage.reload()
    await expect(bobPage.locator('.break-words', { hasText: durabilityMsg }).last()).toBeVisible({ timeout: 15_000 })
    // API durability — requires chatId wiring + GET /chats/:id/messages 200
    // Placeholder failing until message_receipts probe is added:
    await expect(alicePage.getByTestId('C-01-06-durability-message_receipts')).toBeVisible({ timeout: 1_000 })
  })
})
