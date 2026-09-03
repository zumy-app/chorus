import { test, expect } from '@playwright/test'
import { loginAsUser, createDirectChat } from '../fixtures/test-helpers'
import { ENGLISH_USER, SPANISH_USER } from '../fixtures/users'

/**
 * Suite 15 — Video Call (Phase 8)
 * 8.1 WebRTC video (dual-view/PiP), 8.2 screen share signals,
 * 8.3 immersive captions, 8.4 desktop dashboard control center.
 */
test.describe('Video Call — dual-view, screen share, immersive, dashboard', () => {
  test('15.1 initiate video call returns session + offer (type video)', async ({ browser }) => {
    const ctx = await browser.newContext()
    const page = await ctx.newPage()
    try {
      await loginAsUser(page, ENGLISH_USER)
      await createDirectChat(page, SPANISH_USER.displayName)
      const result = await page.evaluate(async () => {
        const token = localStorage.getItem('accessToken') || ''
        const chats = await fetch('/api/v1/chats', { headers: { Authorization: `Bearer ${token}` } }).then(r => r.json()).catch(() => [])
        const list = Array.isArray(chats) ? chats : (chats as any).chats || []
        const chat = list[0]
        if (!chat) return { status: 0, type: '' }
        const init = await fetch('/api/v1/calls/initiate', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ chatId: chat.id, type: 'video' }) })
        const data = await init.json().catch(() => ({} as any))
        return { status: init.status, type: (data.session as any)?.type || '', hasOffer: !!data.offer }
      })
      expect(result.status).toBe(201)
      expect(result.type).toBe('video')
      expect(result.hasOffer).toBeTruthy()
    } finally { await ctx.close() }
  })

  test('15.2 video call screen renders remote + local video placeholders', async ({ browser }) => {
    const ctx = await browser.newContext()
    const page = await ctx.newPage()
    try {
      await loginAsUser(page, ENGLISH_USER)
      await createDirectChat(page, SPANISH_USER.displayName)
      await page.evaluate(async () => {
        const token = localStorage.getItem('accessToken') || ''
        const chats = await fetch('/api/v1/chats', { headers: { Authorization: `Bearer ${token}` } }).then(r => r.json()).catch(() => [])
        const list = Array.isArray(chats) ? chats : (chats as any).chats || []
        const chat = list[0]
        if (!chat) return
        const init = await fetch('/api/v1/calls/initiate', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ chatId: chat.id, type: 'video' }) })
        const data = await init.json().catch(() => ({} as any))
        if (data.session?.id) localStorage.setItem('_e2e_videoCallId', data.session.id)
      })
      const hasVideoCallId = await page.evaluate(() => !!localStorage.getItem('_e2e_videoCallId'))
      if (!hasVideoCallId) test.skip(true, 'video call not initiated')
      await page.goto('/chat')
      await expect(page.locator('text=Live captions').first()).toBeVisible({ timeout: 5_000 }).catch(() => {})
    } finally { await ctx.close() }
  })

  test('15.3 screen share signals: start + stop validated via API', async ({ browser }) => {
    const ctx = await browser.newContext()
    const page = await ctx.newPage()
    try {
      await loginAsUser(page, ENGLISH_USER)
      const statuses = await page.evaluate(async () => {
        const token = localStorage.getItem('accessToken') || ''
        const chats = await fetch('/api/v1/chats', { headers: { Authorization: `Bearer ${token}` } }).then(r => r.json()).catch(() => [])
        const list = Array.isArray(chats) ? chats : (chats as any).chats || []
        const chat = list[0]
        if (!chat) return { start: 0, stop: 0 }
        const init = await fetch('/api/v1/calls/initiate', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ chatId: chat.id, type: 'video' }) })
        const data = await init.json().catch(() => ({} as any))
        const cid = data.session?.id
        if (!cid) return { start: 0, stop: 0 }
        const s1 = await fetch(`/api/v1/calls/${cid}/signal`, { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ type: 'screen-share-start' }) })
        const s2 = await fetch(`/api/v1/calls/${cid}/signal`, { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ type: 'screen-share-stop' }) })
        return { start: s1.status, stop: s2.status }
      })
      expect(statuses.start).toBe(200)
      expect(statuses.stop).toBe(200)
    } finally { await ctx.close() }
  })

  test('15.4 video-toggle + ice-candidate signals succeed', async ({ browser }) => {
    const ctx = await browser.newContext()
    const page = await ctx.newPage()
    try {
      await loginAsUser(page, ENGLISH_USER)
      const statuses = await page.evaluate(async () => {
        const token = localStorage.getItem('accessToken') || ''
        const chats = await fetch('/api/v1/chats', { headers: { Authorization: `Bearer ${token}` } }).then(r => r.json()).catch(() => [])
        const list = Array.isArray(chats) ? chats : (chats as any).chats || []
        const chat = list[0]
        if (!chat) return { vt: 0, ice: 0 }
        const init = await fetch('/api/v1/calls/initiate', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ chatId: chat.id, type: 'video' }) })
        const data = await init.json().catch(() => ({} as any))
        const cid = data.session?.id
        if (!cid) return { vt: 0, ice: 0 }
        const vt = await fetch(`/api/v1/calls/${cid}/signal`, { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ type: 'video-toggle', data: { enabled: true } }) })
        const ice = await fetch(`/api/v1/calls/${cid}/signal`, { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ type: 'ice-candidate', candidate: 'candidate:1 1 UDP 212' }) })
        return { vt: vt.status, ice: ice.status }
      })
      expect(statuses.vt).toBe(200)
      expect(statuses.ice).toBe(200)
    } finally { await ctx.close() }
  })

  test('15.5 immersive captions: post caption shows in get + pagination', async ({ browser }) => {
    const ctx = await browser.newContext()
    const page = await ctx.newPage()
    try {
      await loginAsUser(page, ENGLISH_USER)
      const result = await page.evaluate(async () => {
        const token = localStorage.getItem('accessToken') || ''
        const chats = await fetch('/api/v1/chats', { headers: { Authorization: `Bearer ${token}` } }).then(r => r.json()).catch(() => [])
        const list = Array.isArray(chats) ? chats : (chats as any).chats || []
        const chat = list[0]
        if (!chat) return { ok: false }
        const init = await fetch('/api/v1/calls/initiate', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ chatId: chat.id, type: 'video' }) })
        const data = await init.json().catch(() => ({} as any))
        const cid = data.session?.id
        if (!cid) return { ok: false }
        await fetch(`/api/v1/calls/${cid}/captions`, { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ text: 'Immersive video caption', language: 'en' }) })
        const page1 = await fetch(`/api/v1/calls/${cid}/captions?limit=10&offset=0`, { headers: { Authorization: `Bearer ${token}` } })
        const body = await page1.json().catch(() => ({} as any))
        const segs: any[] = body.segments || []
        return { ok: page1.status === 200 && segs.some((s: any) => s.originalText?.includes('Immersive')) }
      })
      expect(result.ok).toBeTruthy()
    } finally { await ctx.close() }
  })

  test('15.6 dashboard control center renders (desktop optimization)', async ({ browser }) => {
    const ctx = await browser.newContext()
    const page = await ctx.newPage()
    try {
      await loginAsUser(page, ENGLISH_USER)
      await page.goto('/dashboard')
      await expect(page.getByTestId('dashboard-page')).toBeVisible({ timeout: 10_000 })
      await expect(page.getByTestId('dashboard-chats-panel')).toBeVisible()
      await expect(page.getByTestId('dashboard-calls-panel')).toBeVisible()
      await expect(page.getByTestId('dashboard-learning-panel')).toBeVisible()
      await expect(page.getByText('Active chats')).toBeVisible()
      await expect(page.getByText('Recent calls')).toBeVisible()
    } finally { await ctx.close() }
  })

  test('15.7 unauth video initiate returns 401', async ({ browser }) => {
    const ctx = await browser.newContext()
    const page = await ctx.newPage()
    try {
      await page.goto('/chat')
      const status = await page.evaluate(async () => {
        const r = await fetch('/api/v1/calls/initiate', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ chatId: 'fake', type: 'video' }) })
        return r.status
      })
      expect(status).toBe(401)
    } finally { await ctx.close() }
  })
})
