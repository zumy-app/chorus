import { test, expect } from '@playwright/test'

/**
 * S-HOME-01..04 — Home v2 canonical wireframe: wireframes/chorus_home_desktop_v2/code.html:134
 * Authority: docs/REQUIREMENTS_SLICE_HOME_V2.md:1
 * Surfaces: frontend/src/pages/Landing.tsx:47 (stale STRINGS to purge) and mobile/src/screens/LandingScreen.tsx:96
 *
 * Fail-first contract: these must be RED until both pages are rebuilt to v2.
 * See docs/CREWAI_GAP_CLOSURE_PLAN.md:39 Stage 2 red gate and docs/TDD_RESCUE_SPEC.md:12
 */

test.describe('@S-HOME-01..04 Home v2 — chorus_home_desktop_v2', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
  })

  test('@S-HOME-01 heroV2 — Communication is Learning + brain + dual CTAs', async ({ page }) => {
    // heading Communication is Learning. as h1
    await expect(page.getByRole('heading', { name: /Communication is Learning/ })).toBeVisible()
    // subline Redefining how we acquire language. as primary span
    await expect(page.getByText('Redefining how we acquire language')).toBeVisible()
    // paragraph Bridging the gap...
    await expect(page.getByText(/Bridging the gap between messaging apps and learning platforms/)).toBeVisible()
    await expect(page.getByText(/making communication and learning the exact same function/)).toBeVisible()
    // brain image
    await expect(page.getByRole('img', { name: /Brain Neural Pathways/ })).toBeVisible()
    // CTAs
    await expect(page.getByRole('button', { name: /Start Your Journey/ })).toBeVisible()
    await expect(page.getByRole('button', { name: /Watch Demo/ })).toBeVisible()
    // icon play_circle inside Watch Demo
    await expect(page.getByText('play_circle')).toBeVisible()
    // stale purged
    await expect(page.getByText('Break Language Barriers')).toHaveCount(0)
    await expect(page.getByText('Connect Globally')).toHaveCount(0)
  })

  test('@S-HOME-02 bridging + ecosystem — 4 cards + mockup', async ({ page }) => {
    // Bridging section
    await expect(page.getByRole('heading', { name: /Bridging Messaging and Learning/ })).toBeVisible()
    await expect(page.getByText('The Best of Both Worlds.')).toBeVisible()
    await expect(page.getByText(/Why choose between a messenger like WhatsApp and a learning tool like Duolingo/)).toBeVisible()

    // Ecosystem header
    await expect(page.getByRole('heading', { name: 'A Complete Language Ecosystem' })).toBeVisible()
    await expect(page.getByText('Everything you need to go from basic phrases to true fluency.')).toBeVisible()
    // mockup
    await expect(page.getByRole('img', { name: /Chorus App Mockup/ })).toBeVisible()

    // 4 cards in order — titles as headings
    const cardTitles = ['AI Deep Dive', 'Real Talk', 'Teacher Marketplace', 'Phase 2 Ready']
    for (const title of cardTitles) {
      await expect(page.getByRole('heading', { name: title })).toBeVisible()
    }
    // verify order: AI Deep Dive before Real Talk etc by checking DOM order via locator
    const headings = page.getByRole('heading', { name: /AI Deep Dive|Real Talk|Teacher Marketplace|Phase 2 Ready/ })
    await expect(headings).toHaveCount(4)
    await expect(headings.nth(0)).toHaveText(/AI Deep Dive/)
    await expect(headings.nth(1)).toHaveText(/Real Talk/)
    await expect(headings.nth(2)).toHaveText(/Teacher Marketplace/)
    await expect(headings.nth(3)).toHaveText(/Phase 2 Ready/)

    // card descriptions
    await expect(page.getByText(/Instant grammar analysis and CEFR-aligned drills/)).toBeVisible()
    await expect(page.getByText(/AI-guided roleplays for real-world scenarios/)).toBeVisible()
    await expect(page.getByText(/Book 1:1 sessions with professional tutors/)).toBeVisible()
    await expect(page.getByText(/High-fidelity voice & video calls with live translated captions/)).toBeVisible()

    // icons
    await expect(page.getByText('analytics').first()).toBeVisible()
    await expect(page.getByText('forum').first()).toBeVisible()
    await expect(page.getByText('school').first()).toBeVisible()
    await expect(page.getByText('video_call').first()).toBeVisible()

    // Coming Soon badge only on Phase 2 Ready
    await expect(page.getByText('Coming Soon')).toBeVisible()
    // badge positioned top-right is styling — we just assert exactly one Coming Soon
    await expect(page.getByText('Coming Soon')).toHaveCount(1)

    // stale 6 cards must not exist
    await expect(page.getByText('Instant Translation')).toHaveCount(0)
    await expect(page.getByText('Vocabulary Builder')).toHaveCount(0)
    await expect(page.getByText('Group Chats')).toHaveCount(0)
    await expect(page.getByText('Privacy First')).toHaveCount(0)
    await expect(page.getByRole('heading', { name: /How Chorus Works/ })).toHaveCount(0)
  })

  test('@S-HOME-03 mission + final CTA + footer 7 links', async ({ page }) => {
    // Mission
    await expect(page.getByRole('heading', { name: 'Our Mission' })).toBeVisible()
    await expect(page.getByText(/We believe language shouldn't be a barrier, but a bridge\. Chorus was built by a team of linguists and engineers/)).toBeVisible()

    // Final CTA
    await expect(page.getByRole('heading', { name: 'Ready to reach fluency?' })).toBeVisible()
    await expect(page.getByText(/Join thousands of learners who have transformed their daily chats into a masterclass/)).toBeVisible()
    await expect(page.getByRole('button', { name: 'Get Started Now' })).toBeVisible()

    // Stale CTA must not exist
    await expect(page.getByRole('heading', { name: /Ready to Break Language Barriers/ })).toHaveCount(0)

    // Footer brand + copyright
    await expect(page.getByText('Chorus').first()).toBeVisible()
    await expect(page.getByText('© 2024 Chorus AI. Language learning reimagined.')).toBeVisible()

    // Footer columns headers
    await expect(page.getByText('Product')).toBeVisible()
    await expect(page.getByText('Company')).toBeVisible()
    await expect(page.getByText('Support')).toBeVisible()

    // 7 links with correct hrefs
    const featuresLink = page.getByRole('link', { name: 'Features' })
    await expect(featuresLink).toBeVisible()
    await expect(featuresLink).toHaveAttribute('href', '#features')

    const pricingLink = page.getByRole('link', { name: 'Pricing' })
    await expect(pricingLink).toBeVisible()
    await expect(pricingLink).toHaveAttribute('href', '#pricing')

    await expect(page.getByRole('link', { name: 'About Us' })).toHaveAttribute('href', '#about')
    await expect(page.getByRole('link', { name: 'Privacy Policy' })).toHaveAttribute('href', '#')
    await expect(page.getByRole('link', { name: 'Terms of Service' })).toHaveAttribute('href', '#')
    await expect(page.getByRole('link', { name: 'Help Center' })).toHaveAttribute('href', '#')

    // Exactly 7 footer links: count inside footer
    const footer = page.locator('footer')
    await expect(footer.getByRole('link')).toHaveCount(7)

    // Stale footer links must not exist in footer
    await expect(footer.getByRole('link', { name: 'Web App' })).toHaveCount(0)
    await expect(footer.getByRole('link', { name: 'How It Works' })).toHaveCount(0)
    await expect(footer.getByRole('link', { name: 'Languages' })).toHaveCount(0)
    await expect(footer.getByText('http://localhost:8080/health')).toHaveCount(0)
  })

  test('@S-HOME-04 pricing — Free 280-char + Premium $7.99 1000-char', async ({ page }) => {
    const pricing = page.locator('#pricing')
    await expect(pricing.getByRole('heading', { name: 'Simple, Transparent Pricing' })).toBeVisible()
    await expect(pricing.getByText("Start for free, upgrade when you're ready to accelerate.")).toBeVisible()

    // Free card
    await expect(pricing.getByRole('heading', { name: 'Free' })).toBeVisible()
    await expect(pricing.getByText('$0')).toBeVisible()
    await expect(pricing.getByText('/month')).toBeVisible()
    await expect(pricing.getByText('280-character messages')).toBeVisible()
    await expect(pricing.getByText('Basic AI translations')).toBeVisible()
    await expect(pricing.getByText('Limited daily AI insights')).toBeVisible()
    await expect(pricing.getByRole('button', { name: 'Get Started Free' })).toBeVisible()

    // Premium card
    await expect(pricing.getByText('Most Popular')).toBeVisible()
    await expect(pricing.getByRole('heading', { name: 'Premium' })).toBeVisible()
    await expect(pricing.getByText('$7.99')).toBeVisible()
    await expect(pricing.getByText('1000-character messages')).toBeVisible()
    await expect(pricing.getByText('Unlimited AI Deep Dives')).toBeVisible()
    await expect(pricing.getByText('Monthly trial credits for live tutors')).toBeVisible()
    await expect(pricing.getByText('Reduced marketplace fees')).toBeVisible()
    await expect(pricing.getByRole('button', { name: 'Upgrade to Premium' })).toBeVisible()

    // Only 2 pricing cards — locate via pricing section buttons/cards count or headings Free/Premium
    await expect(pricing.getByRole('heading', { name: /Free|Premium/ })).toHaveCount(2)

    // Stale pricing must not appear inside #pricing
    await expect(pricing.getByText('$79.90')).toHaveCount(0)
    await expect(pricing.getByText('200 characters')).toHaveCount(0)
    await expect(pricing.getByText('Unlimited chats & groups')).toHaveCount(0)
    await expect(pricing.getByText('Enterprise')).toHaveCount(0)
  })

  test('@S-HOME TopNav — Features/Pricing/About Us + Get Started sticky', async ({ page }) => {
    const header = page.locator('header')
    await expect(header.getByRole('link', { name: 'Features' })).toHaveAttribute('href', '#features')
    await expect(header.getByRole('link', { name: 'Pricing' })).toHaveAttribute('href', '#pricing')
    await expect(header.getByRole('link', { name: 'About Us' })).toHaveAttribute('href', '#about')
    await expect(header.getByRole('button', { name: 'Get Started' })).toBeVisible()
    // sticky backdrop-blur is class check
    await expect(header).toHaveClass(/sticky/)
    await expect(header).toHaveClass(/backdrop-blur/)
  })
})
