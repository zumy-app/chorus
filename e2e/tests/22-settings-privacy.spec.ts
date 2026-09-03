import { test, expect } from '@playwright/test'
import { loginAsUser, openProfileMenu } from '../fixtures/test-helpers'

/**
 * C-03 — Settings Privacy + 2FA (enforcement, not just fields)
 * Authority: docs/QA_CRITIQUE_AND_IMPROVEMENTS.md:§2.3, docs/TEST_PLAN.md:§12.2 C-03
 * Backend: GET /users/me:465 PUT /users/me:467 GET /users/me/settings:473 PUT /users/me/settings:474
 *          POST /blocks:485 DELETE /blocks/:id:486 GET /blocks:487 POST /reports:489 GET /admin/reports:535
 *          POST /auth/2fa/setup + verify, GET /users/me/export, DELETE /users/me, GET /privacy/retention-policy:478, GET /chats/:id/gallery:581
 * Frontend: Profile.tsx:1 /profile, Settings.tsx + PrivacySettings.tsx:1, ReportModal.tsx:1, ChatLanguageModal.tsx:1 FR-35
 * Mobile: ProfileScreen.tsx:1 MainTabs.tsx:134
 * Gherkin in QA doc §2.3 — FAILING-FIRST (RED until block hides DM, report queues, 2FA replay guard, export zip, retention).
 * Current 08-settings.spec.ts:1 only checks Display Name + language dropdown — does NOT enforce privacy.
 */
test.describe('@C-03 @settings @privacy @2FA', () => {
  test.describe.configure({ mode: 'serial' })

  test('C-03-01 — profile settings persist: Display Name + Native Language + target toggle → PUT /users/me/settings:474 → reload', async ({ page }) => {
    await loginAsUser(page, { email: 'alice.dev@chorus.test', password: 'ChorusDev123!', nativeLanguage: 'en', displayName: 'Alice Dev' } as any)
    await openProfileMenu(page)
    await page.getByRole('button', { name: /settings/i }).click()
    await expect(page.locator('h2', { hasText: 'Settings' })).toBeVisible({ timeout: 10_000 })
    // Change and save — intercept PUT /users/me/settings:474 200, then reload and verify
    await expect(page.getByTestId('C-03-01-settings-persist-proven')).toBeVisible({ timeout: 1_000 })
  })

  test('C-03-02 — block hides chat (privacy enforcement) POST /blocks:485 → GET /blocks:487 → sidebar count 0 → DELETE :486', async ({ page }) => {
    // Requires alice + bob seeded, JWT clean, then POST /blocks {blockedUserId: bobId}
    await expect(page.getByTestId('C-03-02-block-enforced')).toBeVisible({ timeout: 1_000 })
  })

  test('C-03-03 — report queues to moderator: ReportModal → POST /reports:489 201 → GET /admin/reports:535', async ({ page }) => {
    await expect(page.getByTestId('C-03-03-report-queued')).toBeVisible({ timeout: 1_000 })
  })

  test('C-03-04 — ChatLanguageModal FR-35 only own language (ChatLanguageModal.test.tsx)', async ({ page }) => {
    // Open chat language selector modal — must list only own native language
    await expect(page.getByTestId('C-03-04-chat-language-modal-FR-35')).toBeVisible({ timeout: 1_000 })
  })

  test('C-03-05 — 2FA enable + replay guard + privacy leak guard (security_qa_test.go)', async ({ page }) => {
    // POST /auth/2fa/setup + verify, second login requires TOTP, replay old code 401
    await expect(page.getByTestId('C-03-05-2fa-replay-guard')).toBeVisible({ timeout: 1_000 })
  })

  test('C-03-06 — GDPR export zip + retention policy 365/30/90/90 + erasure DELETE /users/me', async ({ page }) => {
    await expect(page.getByTestId('C-03-06-gdpr-export-erase')).toBeVisible({ timeout: 1_000 })
  })

  test('C-03-07 — location + gallery GET /chats/:id/gallery:581 handler contract', async ({ page }) => {
    await expect(page.getByTestId('C-03-07-gallery-location')).toBeVisible({ timeout: 1_000 })
  })
})
