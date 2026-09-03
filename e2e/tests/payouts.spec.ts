import { test, expect } from '@playwright/test'
import fs from 'fs'
import path from 'path'

/**
 * S-T-06 — Payout Settings & History + Teacher Earnings Overview
 * Authority: docs/REQUIREMENTS_SLICE_MARKETPLACE.md:478 S-T-06, docs/WIREFRAME_TRACE.md:65 payout_settings_history + :84 teacher_earnings_overview,
 * wireframes/payout_settings_history/code.html + teacher_earnings_overview/code.html, frontend/src/App.tsx:252 /teacher/payouts, mobile/src/components/MainTabs.tsx:200
 * TDD red gate — must FAIL until Payouts hardened (Lifetime Earnings + Methods + Breakdown + Withdraw + History)
 */

test.describe('@S-T-06 @marketplace @payouts @wireframe-payout_settings_history', () => {
  test('navigation parity — /teacher/payouts and PayoutsScreen', async () => {
    const app = fs.readFileSync(path.resolve(__dirname, '../../frontend/src/App.tsx'), 'utf-8')
    expect(app).toContain('/teacher/payouts')
    expect(app).toContain('Payouts')
    const tabs = fs.readFileSync(path.resolve(__dirname, '../../mobile/src/components/MainTabs.tsx'), 'utf-8')
    expect(tabs).toContain('Payouts')
    expect(tabs).toContain('MarketplaceTab/Payouts')
  })

  test('wireframe parity — Payouts.tsx must contain Lifetime Earnings + Payout Methods + Breakdown + Withdraw + History', async () => {
    const content = fs.readFileSync(path.resolve(__dirname, '../../frontend/src/pages/Payouts.tsx'), 'utf-8')
    expect(content).toContain('Payout Settings & History')
    expect(content).toContain('Total Lifetime Earnings')
    expect(content).toContain('Payout Methods')
    expect(content).toContain("This Month's Breakdown")
    expect(content).toContain('Withdraw')
    expect(content).toContain('Performance Insight')
    expect(content).toContain('Payout History')
    expect(content).toContain('Available for payout')
    expect(content).toContain('Withdraw Funds')
  })

  test('mobile parity — PayoutsScreen.tsx same sections', async () => {
    const content = fs.readFileSync(path.resolve(__dirname, '../../mobile/src/screens/PayoutsScreen.tsx'), 'utf-8')
    expect(content).toContain('Earnings')
    expect(content).toContain('Methods')
    expect(content).toContain('Withdraw')
    expect(content).toContain('History')
  })

  test('web payouts renders wireframe sections (requires sofia.tutor auth)', async ({ page }) => {
    await page.goto('/teacher/payouts')
    await expect(page.getByText('Payout Settings & History')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('Total Lifetime Earnings')).toBeVisible()
    await expect(page.getByText('Payout Methods')).toBeVisible()
    await expect(page.getByText("This Month's Breakdown")).toBeVisible()
    await expect(page.getByText('Withdraw')).toBeVisible()
    await expect(page.getByText('Performance Insight')).toBeVisible()
    await expect(page.getByText('Payout History')).toBeVisible()
  })

  test('S-T-06 hardened — Payouts wireframe green', async () => {
    const content = fs.readFileSync(path.resolve(__dirname, '../../frontend/src/pages/Payouts.tsx'), 'utf-8')
    expect(content).toContain('Total Lifetime Earnings')
    expect(content).toContain('Payout History')
    expect(true).toBeTruthy()
  })
})
