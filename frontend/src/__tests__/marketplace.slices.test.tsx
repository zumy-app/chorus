import { describe, it, expect } from 'vitest'
import fs from 'fs'
import path from 'path'

/**
 * S-T-01..06 — Marketplace slices TDD red gate (frontend vitest)
 * Authority: docs/REQUIREMENTS_SLICE_MARKETPLACE.md:1, docs/WIREFRAME_TRACE.md:28 12 GAP,
 * frontend/src/App.tsx:183 /tutors etc. — must FAIL until BrowseTutors, TutorProfile, ConfirmBooking,
 * TrialCredits, TeacherDashboard, Payouts hardened to wireframe visual contract.
 * This file is the REQUIRED failing placeholder per CREWAI_GAP_CLOSURE_PLAN.md:39 Stage 2 red gate.
 * Do NOT fix screens yet — report red as instructed.
 */

const frontendRoot = path.resolve(__dirname, '..')
const appPath = path.resolve(frontendRoot, 'App.tsx')
const mobileTabsPath = path.resolve(__dirname, '../../../mobile/src/components/MainTabs.tsx')

describe('Marketplace slices — frontend vitest placeholder (S-T-01..06) — TDD red gate', () => {
  it('navigation reachability — App.tsx exposes 6 marketplace routes', () => {
    const content = fs.readFileSync(appPath, 'utf-8')
    expect(content).toContain('/tutors')
    expect(content).toContain('/tutors/:id')
    expect(content).toContain('/tutors/:id/confirm')
    expect(content).toContain('/trial-credits')
    expect(content).toContain('/teacher/dashboard')
    expect(content).toContain('/teacher/payouts')
  })

  it('mobile navigation parity — MainTabs.tsx exposes MarketplaceTab 6 screens', () => {
    const content = fs.readFileSync(mobileTabsPath, 'utf-8')
    expect(content).toContain('BrowseTutors')
    expect(content).toContain('TutorProfile')
    expect(content).toContain('ConfirmBooking')
    expect(content).toContain('TrialCredits')
    expect(content).toContain('TeacherDashboard')
    expect(content).toContain('Payouts')
    expect(content).toContain('MarketplaceTab')
  })

  it('S-T-01 BrowseTutors — web must have Featured Tutors + Available Now + filters per browse_tutors/code.html:156-287', () => {
    const content = fs.readFileSync(path.resolve(frontendRoot, 'pages/BrowseTutors.tsx'), 'utf-8')
    expect(content).toContain('Featured Tutors')
    expect(content).toContain('Available Now')
    expect(content).toContain('Language')
  })

  it('S-T-02 TutorProfile — must have Sofia hero + Verified + Pricing Options + calendar', () => {
    const content = fs.readFileSync(path.resolve(frontendRoot, 'pages/TutorProfile.tsx'), 'utf-8')
    expect(content).toContain('Verified')
    expect(content).toContain('Book Trial')
    // Hardened requirement — currently GAP
    expect(content).toContain('Pricing Options')
  })

  it('S-T-03 ConfirmBooking — must have Great choice + Payment Summary $0.00 + sticky CTA', () => {
    const content = fs.readFileSync(path.resolve(frontendRoot, 'pages/ConfirmBooking.tsx'), 'utf-8')
    expect(content).toContain('Great choice')
    expect(content).toContain('Payment Summary')
    expect(content).toContain('$0.00')
    expect(content).toContain('Cancellation Policy')
  })

  it('S-T-04 TrialCredits — must have credits card + How Trials Work + Recommended + History', () => {
    const content = fs.readFileSync(path.resolve(frontendRoot, 'pages/TrialCredits.tsx'), 'utf-8')
    expect(content).toContain('Trial Credits')
    expect(content).toContain('How Trials Work')
    expect(content).toContain('Recommended for Trials')
    expect(content).toContain('History')
  })

  it('S-T-05 TeacherDashboard — must have Welcome + Earnings Overview + Availability + Students + Profile Completion', () => {
    const content = fs.readFileSync(path.resolve(frontendRoot, 'pages/TeacherDashboard.tsx'), 'utf-8')
    expect(content).toContain('Teacher Dashboard')
    expect(content).toContain('Welcome back')
    expect(content).toContain('Earnings Overview')
    expect(content).toContain('Profile Completion')
  })

  it('S-T-06 Payouts — must have Lifetime Earnings + Methods + Breakdown + Withdraw + History', () => {
    const content = fs.readFileSync(path.resolve(frontendRoot, 'pages/Payouts.tsx'), 'utf-8')
    expect(content).toContain('Payout Settings')
    expect(content).toContain('Total Lifetime Earnings')
    expect(content).toContain('Payout Methods')
    expect(content).toContain('Payout History')
  })

  it('S-T-01..06 hardened — wireframe parity green (TDD red gate removed)', () => {
    // TDD red gate removed after hardening per REQUIREMENTS_SLICE_MARKETPLACE.md:82-573
    // Assert all 6 slices contain their wireframe contract strings (real content, not placeholder)
    const browse = fs.readFileSync(path.resolve(frontendRoot, 'pages/BrowseTutors.tsx'), 'utf-8')
    const profile = fs.readFileSync(path.resolve(frontendRoot, 'pages/TutorProfile.tsx'), 'utf-8')
    const confirm = fs.readFileSync(path.resolve(frontendRoot, 'pages/ConfirmBooking.tsx'), 'utf-8')
    const trial = fs.readFileSync(path.resolve(frontendRoot, 'pages/TrialCredits.tsx'), 'utf-8')
    const dash = fs.readFileSync(path.resolve(frontendRoot, 'pages/TeacherDashboard.tsx'), 'utf-8')
    const payouts = fs.readFileSync(path.resolve(frontendRoot, 'pages/Payouts.tsx'), 'utf-8')
    expect(browse).toContain('Featured Tutors')
    expect(browse).toContain('Available Now')
    expect(browse).toContain('Price')
    expect(browse).toContain('Rating')
    expect(profile).toContain('Pricing Options')
    expect(profile).toContain('Booking calendar')
    expect(profile).toContain('Verified')
    expect(confirm).toContain('Great choice')
    expect(confirm).toContain('$0.00')
    expect(trial).toContain('How Trials Work')
    expect(dash).toContain('Welcome back')
    expect(dash).toContain('Earnings Overview')
    expect(payouts).toContain('Total Lifetime Earnings')
    expect(true).toBeTruthy()
  })
})
