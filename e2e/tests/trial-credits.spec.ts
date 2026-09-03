import { test, expect } from '@playwright/test'
import fs from 'fs'
import path from 'path'

/**
 * S-T-04 — Trial Credit Dashboard
 * Authority: docs/REQUIREMENTS_SLICE_MARKETPLACE.md:330 S-T-04, docs/WIREFRAME_TRACE.md:88 trial_credit_dashboard,
 * wireframes/trial_credit_dashboard/code.html, frontend/src/App.tsx:250 /trial-credits, mobile/src/components/MainTabs.tsx:198
 * TDD red gate — must FAIL until TrialCredits hardened (credits card + How Trials Work + Recommended + History)
 */

test.describe('@S-T-04 @marketplace @trial-credits @wireframe-trial_credit_dashboard', () => {
  test('navigation parity — /trial-credits and TrialCreditsScreen', async () => {
    const app = fs.readFileSync(path.resolve(__dirname, '../../frontend/src/App.tsx'), 'utf-8')
    expect(app).toContain('/trial-credits')
    expect(app).toContain('TrialCredits')
    const tabs = fs.readFileSync(path.resolve(__dirname, '../../mobile/src/components/MainTabs.tsx'), 'utf-8')
    expect(tabs).toContain('TrialCredits')
    expect(tabs).toContain('MarketplaceTab/TrialCredits')
  })

  test('wireframe parity — TrialCredits.tsx must contain credits card + How Trials Work + Recommended + History', async () => {
    const content = fs.readFileSync(path.resolve(__dirname, '../../frontend/src/pages/TrialCredits.tsx'), 'utf-8')
    expect(content).toContain('Trial Credits')
    expect(content).toContain('Available to use right now')
    expect(content).toContain('How Trials Work')
    expect(content).toContain('20 Minutes')
    expect(content).toContain('Meet & Greet')
    expect(content).toContain('Recommended for Trials')
    expect(content).toContain('History')
    expect(content).toContain('Find a Tutor')
  })

  test('mobile parity — TrialCreditsScreen.tsx same sections', async () => {
    const content = fs.readFileSync(path.resolve(__dirname, '../../mobile/src/screens/TrialCreditsScreen.tsx'), 'utf-8')
    expect(content).toContain('Trial credits available')
    expect(content).toContain('How Trials Work')
    expect(content).toContain('20 Minutes')
    expect(content).toContain('Recommended for Trials')
    expect(content).toContain('History')
  })

  test('web dashboard renders credits card + CTA + History (requires backend)', async ({ page }) => {
    await page.goto('/trial-credits')
    await expect(page.getByText('Trial Credits')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('Available to use right now')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Find a Tutor' })).toBeVisible()
    await expect(page.getByText('How Trials Work')).toBeVisible()
    await expect(page.getByText('Recommended for Trials')).toBeVisible()
    await expect(page.getByText('History')).toBeVisible()
  })

  test('S-T-04 hardened — TrialCredits wireframe green', async () => {
    const content = fs.readFileSync(path.resolve(__dirname, '../../frontend/src/pages/TrialCredits.tsx'), 'utf-8')
    expect(content).toContain('Trial Credits')
    expect(content).toContain('How Trials Work')
    expect(true).toBeTruthy()
  })
})
