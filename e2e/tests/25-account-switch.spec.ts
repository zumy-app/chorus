import { test, expect } from '@playwright/test'
import {
  loginAsUser,
  createDirectChat,
  sendMessage,
  findChatInSidebar,
  openProfileMenu,
} from '../fixtures/test-helpers'
import { DEV_ALICE, DEV_BOB } from '../fixtures/users'

/**
 * Test Suite 25: Account switching on the SAME session.
 *
 * Regression suite for the mobile quick-switch outage (typed
 * `apiService.switchUser` fix): alice sends a message, the SAME page
 * switches identity to bob, and bob must see alice's message. Two isolated
 * browser contexts would never catch a same-session identity bug, so every
 * assert here is HARD — no try/catch, no console.warn fallbacks.
 */
test.describe('Account switching (same session)', () => {
  test.describe.configure({ mode: 'serial' })

  test('25.1 — alice sends, logout, login as bob on the same page, bob sees the message', async ({
    page,
  }) => {
    const probe = `switch-probe-${Date.now()}`
    await loginAsUser(page, DEV_ALICE)

    // Ensure the DM exists (backend dedupes direct chats — safe on reruns)
    await createDirectChat(page, DEV_BOB.displayName)
    await sendMessage(page, probe)

    // Log out on the SAME page/context (mirrors the reported repro)
    await openProfileMenu(page)
    await page.getByRole('button', { name: /sign out/i }).click()
    await expect(page).toHaveURL(/\/login/, { timeout: 10_000 })
    expect(await page.evaluate(() => localStorage.getItem('accessToken'))).toBeNull()

    // Log in as bob — same context, fresh identity
    await loginAsUser(page, DEV_BOB)

    // Prove the session really belongs to bob
    await openProfileMenu(page)
    await expect(page.locator('text=' + DEV_BOB.email)).toBeVisible()
    await page.keyboard.press('Escape')

    // Bob must see alice's message in the DM (sidebar + opened thread)
    const chat = await findChatInSidebar(page, DEV_ALICE.displayName)
    await chat.click()
    await expect(page.locator('.break-words', { hasText: probe })).toBeVisible({
      timeout: 15_000,
    })
  })

  test('25.2 — dev quick-switch button switches identity (skipped when DEV UI absent)', async ({
    page,
  }) => {
    await loginAsUser(page, DEV_ALICE)
    await page.goto('/profile')

    // The quick-switch panel only renders in dev builds (import.meta.env.DEV).
    // Production preview builds honestly skip instead of soft-passing.
    const switcher = page.locator('text=Quick switch test account')
    if ((await switcher.count()) === 0) {
      test.skip(true, 'DEV ONLY switcher not rendered in this build — covered by 25.1 logout/login path')
      return
    }

    const bobRow = page.locator('button', { hasText: DEV_BOB.email })
    await expect(bobRow).toBeVisible()
    await bobRow.click()

    // handleDevSwitch writes tokens+user then location.href = /chat
    await page.waitForURL('**/chat', { timeout: 30_000 })
    await expect(page.locator('h1', { hasText: 'Chorus' })).toBeVisible()

    // Prove the session really belongs to bob
    await openProfileMenu(page)
    await expect(page.locator('text=' + DEV_BOB.email)).toBeVisible()
  })
})
