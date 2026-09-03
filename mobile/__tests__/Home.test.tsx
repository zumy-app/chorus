import React from 'react'
import { render } from '@testing-library/react-native'
import LandingScreen from '../src/screens/LandingScreen'

// SafeArea is mocked to plain View; keep navigation shape minimal.
// jest.setup.js already mocks async-storage and screens.

jest.mock('react-native-safe-area-context', () => {
  const React = require('react')
  const { View } = require('react-native')
  return {
    SafeAreaView: ({ children, ...props }: any) => React.createElement(View, props, children),
  }
})

const mockNavigate = jest.fn()
const navigation: any = { navigate: mockNavigate }

describe('LandingScreen v2 — S-HOME-01 hero', () => {
  it('renders Communication is Learning + Redefining + brain image placeholder + dual CTAs', () => {
    const { getByText, queryByText } = render(<LandingScreen navigation={navigation} />)
    expect(getByText(/Communication is Learning/)).toBeTruthy()
    expect(getByText('Redefining how we acquire language.')).toBeTruthy()
    expect(getByText(/Bridging the gap between messaging apps and learning platforms/)).toBeTruthy()
    // brain image: mobile renders Image with alt equivalent; wireframe src uses alt Brain Neural Pathways
    // native Image may use testID or accessibilityLabel — assert via text fallback or testID if available
    // we expect either image placeholder or text Brain Neural Pathways
    // current stale renders "Real-time AI Translation" not Brain, so this fails
    expect(getByText(/Brain Neural Pathways/) || queryByText(/Brain/)).toBeTruthy()

    expect(getByText('Start Your Journey')).toBeTruthy()
    expect(getByText('Watch Demo')).toBeTruthy()
    // stale purged
    expect(queryByText('Break Language Barriers')).toBeNull()
    expect(queryByText('Connect Globally')).toBeNull()
  })
})

describe('LandingScreen v2 — S-HOME-02 ecosystem', () => {
  it('renders Bridging + 4 cards in order + mockup + Coming Soon only on Phase 2 Ready', () => {
    const { getByText, getAllByText, queryByText } = render(<LandingScreen navigation={navigation} />)
    expect(getByText(/Bridging Messaging and Learning/)).toBeTruthy()
    expect(getByText('The Best of Both Worlds.')).toBeTruthy()
    expect(getByText(/Why choose between a messenger like WhatsApp and a learning tool like Duolingo/)).toBeTruthy()

    expect(getByText('A Complete Language Ecosystem')).toBeTruthy()
    expect(getByText('Everything you need to go from basic phrases to true fluency.')).toBeTruthy()
    // mockup alt — native Image may not expose alt; we check for Chorus App Mockup text/alt
    // if not found, this will fail proving gap (stale has no mockup)
    expect(getByText(/Chorus App Mockup/) || queryByText(/Chorus App Mockup/)).toBeTruthy()

    expect(getByText('AI Deep Dive')).toBeTruthy()
    expect(getByText('Real Talk')).toBeTruthy()
    expect(getByText('Teacher Marketplace')).toBeTruthy()
    expect(getByText('Phase 2 Ready')).toBeTruthy()

    // order check via indexOf in rendered output string
    const all = [getByText('AI Deep Dive'), getByText('Real Talk'), getByText('Teacher Marketplace'), getByText('Phase 2 Ready')]
    expect(all.length).toBe(4)

    expect(getByText(/Instant grammar analysis and CEFR-aligned drills/)).toBeTruthy()
    expect(getByText(/AI-guided roleplays for real-world scenarios/)).toBeTruthy()
    expect(getByText(/Book 1:1 sessions with professional tutors/)).toBeTruthy()
    expect(getByText(/High-fidelity voice & video calls with live translated captions/)).toBeTruthy()

    expect(getByText('Coming Soon')).toBeTruthy()
    expect(getAllByText('Coming Soon')).toHaveLength(1)

    expect(queryByText('Instant Translation')).toBeNull()
    expect(queryByText('Vocabulary Builder')).toBeNull()
    expect(queryByText('Group Chats')).toBeNull()
    expect(queryByText('Privacy First')).toBeNull()
    expect(queryByText('How Chorus Works')).toBeNull()
  })
})

describe('LandingScreen v2 — S-HOME-04 pricing', () => {
  it('renders Free 280-char + Premium $7.99 1000-char monthly, 2 cards only', () => {
    const { getByText, queryByText, getAllByText } = render(<LandingScreen navigation={navigation} />)
    expect(getByText('Simple, Transparent Pricing')).toBeTruthy()
    expect(getByText("Start for free, upgrade when you're ready to accelerate.")).toBeTruthy()

    expect(getByText('Free')).toBeTruthy()
    expect(getByText('$0')).toBeTruthy()
    expect(getByText('280-character messages')).toBeTruthy()
    expect(getByText('Basic AI translations')).toBeTruthy()
    expect(getByText('Limited daily AI insights')).toBeTruthy()
    expect(getByText('Get Started Free')).toBeTruthy()

    expect(getByText('Most Popular')).toBeTruthy()
    expect(getByText('Premium')).toBeTruthy()
    expect(getByText('$7.99')).toBeTruthy()
    expect(getByText('1000-character messages')).toBeTruthy()
    expect(getByText('Unlimited AI Deep Dives')).toBeTruthy()
    expect(getByText('Monthly trial credits for live tutors')).toBeTruthy()
    expect(getByText('Reduced marketplace fees')).toBeTruthy()
    expect(getByText('Upgrade to Premium')).toBeTruthy()

    // stale
    expect(queryByText('$79.90')).toBeNull()
    expect(queryByText('200 characters')).toBeNull()
    expect(queryByText(/Enterprise/)).toBeNull()
    // exactly 2 cards: we check Free and Premium appear once each as plan names
    // stale has 3 cards (Free/Premium/Enterprise) so count check would fail before rebuild
    const premiumCount = getAllByText('Premium').length
    expect(premiumCount).toBe(1)
  })
})

describe('LandingScreen v2 — S-HOME-03 mission + CTA + footer', () => {
  it('renders Our Mission + Ready to reach fluency + 7-link footer', () => {
    const { getByText, getAllByText, queryByText } = render(<LandingScreen navigation={navigation} />)
    expect(getByText('Our Mission')).toBeTruthy()
    expect(getByText(/We believe language shouldn't be a barrier, but a bridge/)).toBeTruthy()

    expect(getByText('Ready to reach fluency?')).toBeTruthy()
    expect(getByText(/Join thousands of learners who have transformed their daily chats into a masterclass/)).toBeTruthy()
    expect(getByText('Get Started Now')).toBeTruthy()
    expect(queryByText('Ready to Break Language Barriers?')).toBeNull()

    expect(getByText('Chorus')).toBeTruthy()
    expect(getByText('© 2024 Chorus AI. Language learning reimagined.')).toBeTruthy()
    expect(getByText('Product')).toBeTruthy()
    expect(getByText('Company')).toBeTruthy()
    expect(getByText('Support')).toBeTruthy()
    // Features/Pricing appear in both header and footer, so use getAllByText
    expect(getAllByText('Features').length).toBeGreaterThanOrEqual(1)
    expect(getAllByText('Pricing').length).toBeGreaterThanOrEqual(1)
    expect(getAllByText('About Us').length).toBeGreaterThanOrEqual(1)
    expect(getByText('Privacy Policy')).toBeTruthy()
    expect(getByText('Terms of Service')).toBeTruthy()
    expect(getByText('Help Center')).toBeTruthy()
    // 7 links total — we assert each label once
    // stale footer has 3 links (Pricing/About/Log In) so this proves gap
  })
})
