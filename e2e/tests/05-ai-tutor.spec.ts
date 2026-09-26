import { test, expect } from '@playwright/test'
import { loginAsUser, createDirectChat, sendMessage, openGrammarAnalysis, openAITutor, findChatInSidebar } from '../fixtures/test-helpers'
import { ENGLISH_USER, SPANISH_USER } from '../fixtures/users'

/**
 * Test Suite 5: AI Tutor Feature ⭐
 *
 * Verifies the AI Tutor (LearningPanel) which provides interactive
 * grammar learning: breakdown, examples, flashcards, custom Q&A.
 *
 * Flow: Open grammar panel → click "🤖 AI Tutor" → verify learning content
 *
 * Depends on Ollama service. If Ollama is down, the panel still loads
 * but shows fallback content.
 */
test.describe('AI Tutor', () => {
  // NOTE: no serial mode — each test builds its own scenario, so one
  // model-dependent failure (no Ollama locally) must not skip the rest.
  // See #81 for the deterministic-grammar strategy.

  // Helper: setup two users, send a message, open grammar + AI tutor
  async function setupTutorScenario(browser: any, messageText: string) {
    const senderContext = await browser.newContext()
    const receiverContext = await browser.newContext()
    const senderPage = await senderContext.newPage()
    const receiverPage = await receiverContext.newPage()

    await loginAsUser(senderPage, ENGLISH_USER)
    await loginAsUser(receiverPage, SPANISH_USER)

    await createDirectChat(senderPage, SPANISH_USER.displayName)
    const chatItem = await findChatInSidebar(receiverPage, ENGLISH_USER.displayName)
    await chatItem.click()

    await sendMessage(senderPage, messageText)
    await expect(receiverPage.locator('.break-words', { hasText: messageText }).last()).toBeVisible({
      timeout: 15_000,
    })

    await openGrammarAnalysis(receiverPage, messageText)
    await openAITutor(receiverPage)

    return { senderContext, receiverContext, senderPage, receiverPage }
  }

  test('5.1 — AI Tutor button appears in grammar panel', async ({ browser }) => {
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

      const testMsg = `I would like to learn more languages. ${Date.now()}`
      await sendMessage(senderPage, testMsg)
      await expect(receiverPage.locator('.break-words', { hasText: testMsg }).last()).toBeVisible({
        timeout: 15_000,
      })

      await openGrammarAnalysis(receiverPage, testMsg)
      // Wait for grammar panel to finish (queue may delay button)
      await expect(receiverPage.getByTestId('grammar-panel')).toBeVisible({ timeout: 30_000 })
      const tutorBtn = receiverPage.getByRole('button', { name: /🤖/ })
      await expect(tutorBtn).toBeVisible({ timeout: 15_000 })
    } finally {
      await senderContext.close()
      await receiverContext.close()
    }
  })

  test('5.2 — AI Tutor panel opens', async ({ browser }) => {
    const setup = await setupTutorScenario(browser, `The weather is beautiful today. ${Date.now()}`)

    try {
      await expect(setup.receiverPage.getByTestId('ai-tutor-panel')).toBeVisible({ timeout: 15_000 })
      await expect(setup.receiverPage.getByTestId('ai-tutor-panel')).toBeVisible()
    } finally {
      await setup.senderContext.close()
      await setup.receiverContext.close()
    }
  })

  test('5.3 — Initial breakdown auto-loads on mount', async ({ browser }) => {
    const setup = await setupTutorScenario(browser, `She speaks three languages fluently. ${Date.now()}`)

    try {
      const breakdownLabel = setup.receiverPage.locator('text=/📖/').first()
      const loadingIndicator = setup.receiverPage.locator('text=/analyz|analiz/i').first()
      // Hard-fail: either label or loading must appear — no .catch swallow
      await expect.poll(async () => (await breakdownLabel.isVisible()) || (await loadingIndicator.isVisible()), { timeout: 10_000, message: 'expected breakdown label (📖) or loading indicator' }).toBe(true)
    } finally {
      await setup.senderContext.close()
      await setup.receiverContext.close()
    }
  })

  test('5.4 — Breakdown content displays', async ({ browser }) => {
    const setup = await setupTutorScenario(browser, `I am studying grammar every day. ${Date.now()}`)

    try {
      const assistantMessage = setup.receiverPage.getByTestId('ai-tutor-message').first()
      await expect(assistantMessage).toBeVisible({ timeout: 45_000 })
      const content = await assistantMessage.textContent()
      expect(content, 'AI breakdown must have content').toBeTruthy()
      expect(content!.length).toBeGreaterThan(5)
    } finally {
      await setup.senderContext.close()
      await setup.receiverContext.close()
    }
  })

  test('5.5 — Suggested action buttons appear', async ({ browser }) => {
    const setup = await setupTutorScenario(browser, `They have been working on the project. ${Date.now()}`)

    try {
      const assistantMessage = setup.receiverPage.getByTestId('ai-tutor-message').first()
      await expect(assistantMessage).toBeVisible({ timeout: 45_000 })
      const actionButtons = setup.receiverPage.locator('[data-testid="ai-tutor-panel"] .bg-indigo-50')
      await expect(actionButtons.first()).toBeVisible({ timeout: 10_000 })
      const examplesBtn = setup.receiverPage.getByRole('button', { name: /exam|ejempl/i }).first()
      const flashcardsBtn = setup.receiverPage.getByRole('button', { name: /flashcard|tarjeta/i }).first()
      await expect(examplesBtn.or(flashcardsBtn).first()).toBeVisible({ timeout: 10_000 })
    } finally {
      await setup.senderContext.close()
      await setup.receiverContext.close()
    }
  })

  test('5.6 — Examples action works', async ({ browser }) => {
    const setup = await setupTutorScenario(browser, `The book is on the table. ${Date.now()}`)

    try {
      const assistantMessage = setup.receiverPage.getByTestId('ai-tutor-message').first()
      await expect(assistantMessage).toBeVisible({ timeout: 45_000 })
      const examplesBtn = setup.receiverPage.getByRole('button', { name: /exam|ejempl/i }).first()
      await expect(examplesBtn).toBeVisible({ timeout: 10_000 })
      const messagesBefore = await setup.receiverPage.getByTestId('ai-tutor-message').count()
      await examplesBtn.click()
      const nextLocator = setup.receiverPage.getByTestId('ai-tutor-message').nth(messagesBefore)
      await expect(nextLocator).toBeVisible({ timeout: 45_000 })
      expect(await nextLocator.textContent()).toBeTruthy()
    } finally {
      await setup.senderContext.close()
      await setup.receiverContext.close()
    }
  })

  test('5.7 — Flashcards action works', async ({ browser }) => {
    const setup = await setupTutorScenario(browser, `My sister lives in Madrid. ${Date.now()}`)

    try {
      const assistantMessage = setup.receiverPage.getByTestId('ai-tutor-message').first()
      await expect(assistantMessage).toBeVisible({ timeout: 45_000 })
      const flashcardsBtn = setup.receiverPage.getByRole('button', { name: /flashcard|tarjeta/i }).first()
      await expect(flashcardsBtn).toBeVisible({ timeout: 10_000 })
      const messagesBefore = await setup.receiverPage.getByTestId('ai-tutor-message').count()
      await flashcardsBtn.click()
      await expect(setup.receiverPage.getByTestId('ai-tutor-message').nth(messagesBefore)).toBeVisible({ timeout: 45_000 })
    } finally {
      await setup.senderContext.close()
      await setup.receiverContext.close()
    }
  })

  test('5.8 — Custom question works', async ({ browser }) => {
    const setup = await setupTutorScenario(browser, `I enjoy reading books in the evening. ${Date.now()}`)

    try {
      await expect(setup.receiverPage.getByTestId('ai-tutor-message').first()).toBeVisible({ timeout: 45_000 })
      const questionInput = setup.receiverPage.getByTestId('ai-tutor-input')
      await expect(questionInput).toBeVisible({ timeout: 10_000 })
      await questionInput.fill('What tense is used in this sentence?')
      const messagesBefore = await setup.receiverPage.getByTestId('ai-tutor-message').count()
      await setup.receiverPage.getByTestId('ai-tutor-submit').click()
      await expect(setup.receiverPage.getByTestId('ai-tutor-message').nth(messagesBefore)).toBeVisible({ timeout: 45_000 })
    } finally {
      await setup.senderContext.close()
      await setup.receiverContext.close()
    }
  })

  test('5.9 — AI Tutor panel can be closed', async ({ browser }) => {
    const setup = await setupTutorScenario(browser, `Close tutor test. ${Date.now()}`)

    try {
      await expect(setup.receiverPage.getByTestId('ai-tutor-panel')).toBeVisible({ timeout: 15_000 })
      await setup.receiverPage.getByTestId('ai-tutor-close').click()
      await expect(setup.receiverPage.getByTestId('ai-tutor-panel')).not.toBeVisible({ timeout: 10_000 })
    } finally {
      await setup.senderContext.close()
      await setup.receiverContext.close()
    }
  })
})