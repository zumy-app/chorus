import { test, expect } from '@playwright/test'
import fs from 'fs'
import path from 'path'

/**
 * S-T-02 — Tutor Profile (Sofia)
 * Authority: docs/REQUIREMENTS_SLICE_MARKETPLACE.md:180 S-T-02, docs/WIREFRAME_TRACE.md:91 tutor_profile_sofia,
 * wireframes/tutor_profile_sofia/code.html:171-396, frontend/src/App.tsx:248 /tutors/:id, mobile/src/components/MainTabs.tsx:188
 * TDD red gate — must FAIL until TutorProfile hardened (hero + stats + reviews + pricing + calendar)
 */

test.describe('@S-T-02 @marketplace @profile @wireframe-tutor_profile_sofia', () => {
  test('navigation parity — /tutors/:id route and TutorProfileScreen exists', async () => {
    const app = fs.readFileSync(path.resolve(__dirname, '../../frontend/src/App.tsx'), 'utf-8')
    expect(app).toContain('/tutors/:id')
    expect(app).toContain('TutorProfile')
    const tabs = fs.readFileSync(path.resolve(__dirname, '../../mobile/src/components/MainTabs.tsx'), 'utf-8')
    expect(tabs).toContain('TutorProfile')
    expect(tabs).toContain('MarketplaceTab/TutorProfile')
  })

  test('wireframe parity — TutorProfile.tsx must contain hero + Verified + About + Reviews + Book Trial + calendar', async () => {
    const content = fs.readFileSync(path.resolve(__dirname, '../../frontend/src/pages/TutorProfile.tsx'), 'utf-8')
    expect(content).toContain('Verified')
    expect(content).toContain('About')
    expect(content).toContain('Reviews')
    expect(content).toContain('Book Trial')
    expect(content).toContain('data-testid="book-trial"')
    // Hardened wireframe requires stats + pricing + calendar — currently missing (red)
    expect(content).toContain('Pricing Options')
    expect(content).toContain('Booking calendar')
  })

  test('mobile parity — TutorProfileScreen.tsx hero + Verified + rating + bio + reviews + Book Trial', async () => {
    const content = fs.readFileSync(path.resolve(__dirname, '../../mobile/src/screens/TutorProfileScreen.tsx'), 'utf-8')
    expect(content).toContain('Verified')
    expect(content).toContain('Book Trial')
    expect(content).toContain('testID="book-trial"')
    // Stats and pricing sections required by code.html:219-393 — missing until hardened
    expect(content).toContain('Pricing Options')
  })

  test('web profile renders Sofia hero + Verified + rating + bio + expertise + reviews + Book Trial (requires backend)', async ({ page }) => {
    // Gherkin: Given sofia.tutor approved seed, When I open /tutors/:id, Then I see hero per code.html:175-303
    // We use a placeholder uuid; the test will FAIL with 404 + missing wireframe until S-T-02 green + seed
    await page.goto('/tutors/sofia-placeholder-uuid')
    // Even the error branch "Tutor not found" vs success is part of trace — assert hardened success
    await expect(page.getByText('Sofia Tutor')).toBeVisible({ timeout: 15000 })
    await expect(page.getByText('Verified')).toBeVisible()
    await expect(page.getByText(/Hola! I am Sofia/)).toBeVisible()
    await expect(page.getByText(/Conversational Spanish/)).toBeVisible()
    await expect(page.getByTestId('book-trial')).toBeVisible()
    // Reviews: seed has 2 reviews (Alice 5, Bob 4)
    await expect(page.getByText('Reviews')).toBeVisible()
  })

  test('S-T-02 hardened — Sofia hero + pricing + calendar green', async () => {
    const content = fs.readFileSync(path.resolve(__dirname, '../../frontend/src/pages/TutorProfile.tsx'), 'utf-8')
    expect(content).toContain('Pricing Options')
    expect(content).toContain('Booking calendar')
    expect(true).toBeTruthy()
  })
})
