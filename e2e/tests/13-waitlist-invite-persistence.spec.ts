import { test, expect } from '@playwright/test'
import { execSync } from 'child_process'
import { createHash, randomBytes } from 'crypto'
import { resolve } from 'path'
import { API_BASE, loginViaAPI } from '../fixtures/test-helpers'

/**
 * Test Suite 13: Waitlist → Invitation → Registration persistence
 *
 * Verifies the fix in backend/internal/services/auth.go (PR #61): when a user
 * registers via an invitation, the target languages and target language level
 * they picked on the waitlist are carried over to their new account. The
 * register page intentionally submits empty targetLanguages/level, so the
 * backend must fall back to the waitlist entry's preferences.
 *
 * Each test creates a fresh user via the real invitation flow, asserts the
 * persisted profile, then deletes the user and cleans up the waitlist entry,
 * invitation, and outbox rows it created.
 */
const inviteEmail = (kind: string) =>
  `invite-e2e-${kind}-${Date.now().toString(36)}${randomBytes(3).toString('hex')}@example.com`

test.describe('Waitlist → Invitation → Registration persistence', () => {
  test.describe.configure({ mode: 'serial' })

  test('13.1 — Registering via an invite carries over waitlist target languages and level', async ({ page, request }) => {
    const email = inviteEmail('carry')
    const password = 'InviteTest#123'
    const waitlistPrefs = { targetLanguages: ['es', 'fr'], targetLanguageLevel: 'B2' }

    // 1. Join the waitlist with language preferences via the public API.
    const joinRes = await request.post(`${API_BASE}/waitlist`, {
      data: {
        email,
        spokenLanguages: ['en'],
        targetLanguages: waitlistPrefs.targetLanguages,
        targetLanguageLevel: waitlistPrefs.targetLanguageLevel,
        reasons: ['For travel'],
      },
    })
    expect(joinRes.ok()).toBeTruthy()
    const joinBody = await joinRes.json()
    const entryId: string = joinBody.entry.id
    expect(entryId).toBeTruthy()

    // 2. Issue an invitation for that waitlist entry with a known token
    //    (mirrors InvitationService.Create, which hashes the raw token).
    const token = `e2e-${randomBytes(16).toString('hex')}`
    insertInvitation(entryId, email, token)

    // 3. Register via the UI invite flow. The register page prefills the
    //    invited email and submits empty targetLanguages/level.
    await page.goto(`/register?invite=${token}`)
    await expect(page.locator('input[type="email"]')).toHaveValue(email, { timeout: 15_000 })
    await page.locator('input[type="password"]').fill(password)
    await page.getByRole('button', { name: /create account/i }).click()
    await page.waitForURL('**/chat', { timeout: 30_000 })
    await expect(page.locator('h1', { hasText: 'Chorus' })).toBeVisible()

    // 4. Verify the new account persisted the waitlist preferences.
    const userToken = await loginViaAPI({
      email,
      password,
      nativeLanguage: 'en',
      displayName: email.split('@')[0],
    })
    const meRes = await request.get(`${API_BASE}/users/me`, {
      headers: { Authorization: `Bearer ${userToken}` },
    })
    expect(meRes.ok()).toBeTruthy()
    const me = await meRes.json()
    expect(me.targetLanguages).toEqual(waitlistPrefs.targetLanguages)
    expect(me.targetLanguageLevel).toBe(waitlistPrefs.targetLanguageLevel)

    // 5. Clean up: delete the created user and its rows.
    await deleteUserAndRows(email)
  })

  test('13.2 — Explicit register preferences override the waitlist values', async ({ request }) => {
    const email = inviteEmail('override')
    const password = 'Override#123'
    const waitlistPrefs = { targetLanguages: ['de'], targetLanguageLevel: 'A1' }
    const overridePrefs = { targetLanguages: ['pt', 'it'], targetLanguageLevel: 'C1' }

    // 1. Join the waitlist with one set of preferences.
    const joinRes = await request.post(`${API_BASE}/waitlist`, {
      data: {
        email,
        spokenLanguages: ['en'],
        targetLanguages: waitlistPrefs.targetLanguages,
        targetLanguageLevel: waitlistPrefs.targetLanguageLevel,
        reasons: ['For work'],
      },
    })
    expect(joinRes.ok()).toBeTruthy()
    const joinBody = await joinRes.json()
    const entryId: string = joinBody.entry.id
    expect(entryId).toBeTruthy()

    // 2. Issue an invitation with a known token.
    const token = `e2e-${randomBytes(16).toString('hex')}`
    insertInvitation(entryId, email, token)

    // 3. Register directly with explicit (non-empty) preferences — the request
    //    values must win over the waitlist entry.
    const regRes = await request.post(`${API_BASE}/auth/register`, {
      data: {
        email,
        username: email,
        password,
        displayName: email.split('@')[0],
        nativeLanguage: 'en',
        targetLanguages: overridePrefs.targetLanguages,
        targetLanguageLevel: overridePrefs.targetLanguageLevel,
        inviteToken: token,
      },
    })
    expect(regRes.ok()).toBeTruthy()

    // 4. Verify the request preferences were persisted, not the waitlist ones.
    const userToken = await loginViaAPI({
      email,
      password,
      nativeLanguage: 'en',
      displayName: email.split('@')[0],
    })
    const meRes = await request.get(`${API_BASE}/users/me`, {
      headers: { Authorization: `Bearer ${userToken}` },
    })
    expect(meRes.ok()).toBeTruthy()
    const me = await meRes.json()
    expect(me.targetLanguages).toEqual(overridePrefs.targetLanguages)
    expect(me.targetLanguageLevel).toBe(overridePrefs.targetLanguageLevel)

    // 5. Clean up: delete the created user and its rows.
    await deleteUserAndRows(email)
  })
})

/**
 * Insert an invitation row directly so the test can control the raw token.
 * This mirrors InvitationService.Create (backend/internal/services/invitation.go),
 * which stores only the sha256 hex of the token.
 */
function insertInvitation(entryId: string, email: string, token: string) {
  const tokenHash = createHash('sha256').update(token).digest('hex')
  psql(
    `INSERT INTO invitations (waitlist_entry_id, email, token_hash, expires_at) ` +
      `VALUES ('${entryId}', '${email}', '${tokenHash}', CURRENT_TIMESTAMP + INTERVAL '168 hours')`
  )
}

/**
 * Best-effort cleanup: soft-delete the user and remove the waitlist entry,
 * invitation, and any outbox rows. Mirrors the admin delete semantics.
 */
async function deleteUserAndRows(email: string) {
  try {
    psql(`UPDATE users SET deleted_at = CURRENT_TIMESTAMP WHERE email = '${email}' AND deleted_at IS NULL`)
    psql(`DELETE FROM invitations WHERE email = '${email}'`)
    psql(`DELETE FROM waitlist_entries WHERE email = '${email}'`)
    psql(`DELETE FROM email_outbox WHERE recipient = '${email}'`)
  } catch (err) {
    console.log(`⚠️ Cleanup for ${email} failed (best-effort):`, err)
  }
}

/**
 * Run a psql command against the Dockerized Postgres. Tries the production
 * compose stack first (used by the E2E suite), then the dev stack as a
 * fallback, and returns the first command that succeeds.
 */
function psql(sql: string): string {
  const root = resolve(__dirname, '..', '..')
  const candidates = [
    { cwd: root, cmd: `docker compose -f docker-compose.yml exec -T postgres psql -U messenger -d messenger_dev -t -A -c "${sql}"` },
    { cwd: root, cmd: `docker compose -f docker-compose.yml exec -T postgres psql -U messenger -d messenger_prod -t -A -c "${sql}"` },
    { cwd: root, cmd: `docker compose -f docker-compose.dev.yml exec -T postgres-dev psql -U chorus_dev -d chorus_dev -t -A -c "${sql}"` },
  ]
  let lastErr: unknown
  for (const { cwd, cmd } of candidates) {
    try {
      return execSync(cmd, { cwd, stdio: ['pipe', 'pipe', 'ignore'], timeout: 30_000 }).toString().trim()
    } catch (err) {
      lastErr = err
    }
  }
  throw lastErr instanceof Error ? lastErr : new Error(`psql failed for: ${sql}`)
}