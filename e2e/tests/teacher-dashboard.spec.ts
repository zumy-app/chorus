import { test, expect } from '@playwright/test'
import fs from 'fs'
import path from 'path'
import { loginAsUser } from '../fixtures/test-helpers'
import { DEV_SOFIA } from '../fixtures/users'

/**
 * S-T-05 — Teacher Dashboard
 * Authority: docs/REQUIREMENTS_SLICE_MARKETPLACE.md:400 S-T-05, docs/WIREFRAME_TRACE.md:83 teacher_dashboard,
 * wireframes/teacher_dashboard/code.html, frontend/src/App.tsx:251 /teacher/dashboard, mobile/src/components/MainTabs.tsx:199
 * TDD red gate — must FAIL until TeacherDashboard hardened (Welcome + Earnings + Availability + Students + Profile Completion)
 */

test.describe('@S-T-05 @marketplace @teacher-dashboard @wireframe-teacher_dashboard', () => {
  test('navigation parity — /teacher/dashboard and TeacherDashboardScreen', async () => {
    const app = fs.readFileSync(path.resolve(__dirname, '../../frontend/src/App.tsx'), 'utf-8')
    expect(app).toContain('/teacher/dashboard')
    expect(app).toContain('TeacherDashboard')
    const tabs = fs.readFileSync(path.resolve(__dirname, '../../mobile/src/components/MainTabs.tsx'), 'utf-8')
    expect(tabs).toContain('TeacherDashboard')
    expect(tabs).toContain('MarketplaceTab')
  })

  test('wireframe parity — TeacherDashboard.tsx must contain Welcome + Earnings Overview + Availability + Students + Profile Completion', async () => {
    const content = fs.readFileSync(path.resolve(__dirname, '../../frontend/src/pages/TeacherDashboard.tsx'), 'utf-8')
    expect(content).toContain('Teacher Dashboard')
    expect(content).toContain('Welcome back')
    expect(content).toContain('Earnings Overview')
    expect(content).toContain('Availability')
    expect(content).toContain('Recent Students')
    expect(content).toContain('Profile Completion')
    expect(content).toContain('Premium Program')
    expect(content).toContain('Manage Premium Settings')
  })

  test('mobile parity — TeacherDashboardScreen.tsx same Welcome + Earnings + Profile Completion', async () => {
    const content = fs.readFileSync(path.resolve(__dirname, '../../mobile/src/screens/TeacherDashboardScreen.tsx'), 'utf-8')
    expect(content).toContain('Teacher Dashboard')
    expect(content).toContain('Welcome back')
    expect(content).toContain('Earnings Overview')
    expect(content).toContain('Profile Completion')
  })

  test('web dashboard renders wireframe sections (requires sofia.tutor auth)', async ({ page }) => {
    await loginAsUser(page, DEV_SOFIA)
    await page.goto('/teacher/dashboard')
    await expect(page.getByText('Teacher Dashboard').first()).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('Welcome back').first()).toBeVisible()
    await expect(page.getByText('Earnings Overview').first()).toBeVisible()
    await expect(page.getByText('Availability').first()).toBeVisible()
    await expect(page.getByText('Recent Students').first()).toBeVisible()
    await expect(page.getByText(/Profile Completion/).first()).toBeVisible()
  })

  test('S-T-05 hardened — TeacherDashboard wireframe green', async () => {
    const content = fs.readFileSync(path.resolve(__dirname, '../../frontend/src/pages/TeacherDashboard.tsx'), 'utf-8')
    expect(content).toContain('Earnings Overview')
    expect(content).toContain('Profile Completion')
    expect(true).toBeTruthy()
  })
})
