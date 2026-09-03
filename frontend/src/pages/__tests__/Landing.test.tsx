import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

// --- mocks must be hoisted before importing Landing ---
vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (k: string, opts?: any) => opts?.defaultValue ?? k }),
}))

vi.mock('../../store', () => ({
  useStore: (selector: any) => selector({ user: null }),
}))

vi.mock('../../components/LanguageSelector', () => ({
  default: () => <div data-testid="language-selector" />,
}))

// Let services/language use real implementation (detectBrowserLanguage reads navigator.language)
// but polyfill localStorage already via src/test/setup.ts

import Landing from '../Landing'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
})

describe('Landing v2 — S-HOME-01 hero', () => {
  it('renders Communication is Learning headline + brain image + dual CTAs', () => {
    render(<MemoryRouter><Landing /></MemoryRouter>)
    expect(screen.getByRole('heading', { name: /Communication is Learning/ })).toBeInTheDocument()
    expect(screen.getByText('Redefining how we acquire language.')).toBeInTheDocument()
    expect(screen.getByText(/Bridging the gap between messaging apps and learning platforms/)).toBeInTheDocument()
    expect(screen.getByRole('img', { name: /Brain Neural Pathways/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Start Your Journey/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Watch Demo/ })).toBeInTheDocument()
    expect(screen.getByText('play_circle')).toBeInTheDocument()
    expect(screen.queryByText('Break Language Barriers')).not.toBeInTheDocument()
    expect(screen.queryByText('Connect Globally')).not.toBeInTheDocument()
  })
})

describe('Landing v2 — S-HOME-02 ecosystem', () => {
  it('renders Bridging + 4 cards in order + mockup + Coming Soon only on Phase 2 Ready', () => {
    render(<MemoryRouter><Landing /></MemoryRouter>)
    expect(screen.getByRole('heading', { name: /Bridging Messaging and Learning/ })).toBeInTheDocument()
    expect(screen.getByText('The Best of Both Worlds.')).toBeInTheDocument()
    expect(screen.getByText(/Why choose between a messenger like WhatsApp and a learning tool like Duolingo/)).toBeInTheDocument()

    expect(screen.getByRole('heading', { name: 'A Complete Language Ecosystem' })).toBeInTheDocument()
    expect(screen.getByText('Everything you need to go from basic phrases to true fluency.')).toBeInTheDocument()
    expect(screen.getByRole('img', { name: /Chorus App Mockup/ })).toBeInTheDocument()

    const headings = screen.getAllByRole('heading', { name: /AI Deep Dive|Real Talk|Teacher Marketplace|Phase 2 Ready/ })
    expect(headings).toHaveLength(4)
    expect(headings[0].textContent).toMatch(/AI Deep Dive/)
    expect(headings[1].textContent).toMatch(/Real Talk/)
    expect(headings[2].textContent).toMatch(/Teacher Marketplace/)
    expect(headings[3].textContent).toMatch(/Phase 2 Ready/)

    expect(screen.getByText(/Instant grammar analysis and CEFR-aligned drills/)).toBeInTheDocument()
    expect(screen.getByText(/AI-guided roleplays for real-world scenarios/)).toBeInTheDocument()
    expect(screen.getByText(/Book 1:1 sessions with professional tutors/)).toBeInTheDocument()
    expect(screen.getByText(/High-fidelity voice & video calls with live translated captions/)).toBeInTheDocument()

    // Coming Soon badge exactly once
    expect(screen.getAllByText('Coming Soon')).toHaveLength(1)

    // stale 6 cards deleted
    expect(screen.queryByText('Instant Translation')).not.toBeInTheDocument()
    expect(screen.queryByText('Vocabulary Builder')).not.toBeInTheDocument()
    expect(screen.queryByText('Group Chats')).not.toBeInTheDocument()
    expect(screen.queryByText('Privacy First')).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: /How Chorus Works/ })).not.toBeInTheDocument()
  })
})

describe('Landing v2 — S-HOME-04 pricing', () => {
  it('renders Free 280-char and Premium $7.99 1000-char monthly, 2 cards only, no $79.90 or Enterprise', () => {
    render(<MemoryRouter><Landing /></MemoryRouter>)
    const pricing = document.getElementById('pricing')
    expect(pricing).toBeTruthy()

    expect(screen.getByRole('heading', { name: 'Simple, Transparent Pricing' })).toBeInTheDocument()
    expect(screen.getByText("Start for free, upgrade when you're ready to accelerate.")).toBeInTheDocument()

    // Free
    expect(screen.getByRole('heading', { name: /^Free$/ })).toBeInTheDocument()
    expect(screen.getByText('$0')).toBeInTheDocument()
    expect(screen.getByText('280-character messages')).toBeInTheDocument()
    expect(screen.getByText('Basic AI translations')).toBeInTheDocument()
    expect(screen.getByText('Limited daily AI insights')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Get Started Free' })).toBeInTheDocument()

    // Premium
    expect(screen.getByText('Most Popular')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Premium' })).toBeInTheDocument()
    expect(screen.getByText('$7.99')).toBeInTheDocument()
    expect(screen.getByText('1000-character messages')).toBeInTheDocument()
    expect(screen.getByText('Unlimited AI Deep Dives')).toBeInTheDocument()
    expect(screen.getByText('Monthly trial credits for live tutors')).toBeInTheDocument()
    expect(screen.getByText('Reduced marketplace fees')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Upgrade to Premium' })).toBeInTheDocument()

    // exactly 2 pricing cards — Free + Premium only
    expect(screen.getAllByRole('heading', { name: /^(Free|Premium)$/ })).toHaveLength(2)
    expect(screen.queryByText('$79.90')).not.toBeInTheDocument()
    expect(screen.queryByText('200 characters')).not.toBeInTheDocument()
    expect(screen.queryByText('Enterprise')).not.toBeInTheDocument()
  })
})

describe('Landing v2 — S-HOME-03 mission + final CTA + footer', () => {
  it('renders mission, Ready to reach fluency CTA, and 7-link footer', () => {
    render(<MemoryRouter><Landing /></MemoryRouter>)
    expect(screen.getByRole('heading', { name: 'Our Mission' })).toBeInTheDocument()
    expect(screen.getByText(/We believe language shouldn't be a barrier, but a bridge\. Chorus was built by a team of linguists and engineers/)).toBeInTheDocument()

    expect(screen.getByRole('heading', { name: 'Ready to reach fluency?' })).toBeInTheDocument()
    expect(screen.getByText(/Join thousands of learners who have transformed their daily chats into a masterclass/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Get Started Now' })).toBeInTheDocument()
    expect(screen.queryByText('Ready to Break Language Barriers?')).not.toBeInTheDocument()

    // footer
    expect(screen.getByText('Chorus')).toBeInTheDocument()
    expect(screen.getByText('© 2024 Chorus AI. Language learning reimagined.')).toBeInTheDocument()
    expect(screen.getByText('Product')).toBeInTheDocument()
    expect(screen.getByText('Company')).toBeInTheDocument()
    expect(screen.getByText('Support')).toBeInTheDocument()

    const footer = document.querySelector('footer')!
    expect(footer).toBeTruthy()
    const links = footer.querySelectorAll('a')
    expect(links.length).toBe(7)

    expect(within(footer).getByRole('link', { name: 'Features' })).toHaveAttribute('href', '#features')
    expect(within(footer).getByRole('link', { name: 'Pricing' })).toHaveAttribute('href', '#pricing')
    expect(within(footer).getByRole('link', { name: 'About Us' })).toHaveAttribute('href', '#about')
    expect(within(footer).getByRole('link', { name: 'Privacy Policy' })).toHaveAttribute('href', '#')
    expect(within(footer).getByRole('link', { name: 'Terms of Service' })).toHaveAttribute('href', '#')
    expect(within(footer).getByRole('link', { name: 'Help Center' })).toHaveAttribute('href', '#')

    expect(screen.queryByText('Web App')).not.toBeInTheDocument()
    expect(screen.queryByText('How It Works')).not.toBeInTheDocument()
    expect(footer.textContent).not.toContain('http://localhost:8080/health')
  })

  it('topNav has Features/Pricing/About Us anchors + Get Started', () => {
    render(<MemoryRouter><Landing /></MemoryRouter>)
    const header = document.querySelector('header')!
    expect(header).toBeTruthy()
    expect(header.className).toMatch(/sticky/)
    expect(header.className).toMatch(/backdrop-blur/)
    expect(within(header).getByRole('link', { name: 'Features' })).toHaveAttribute('href', '#features')
    expect(within(header).getByRole('link', { name: 'Pricing' })).toHaveAttribute('href', '#pricing')
    expect(within(header).getByRole('link', { name: 'About Us' })).toHaveAttribute('href', '#about')
    expect(within(header).getByRole('button', { name: 'Get Started' })).toBeInTheDocument()
  })
})
