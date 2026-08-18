import { beforeEach, describe, expect, it, vi } from 'vitest'

const storage = new Map<string, string>()

vi.mock('expo-secure-store', () => ({
  deleteItemAsync: vi.fn(async (key: string) => storage.delete(key)),
  getItemAsync: vi.fn(async (key: string) => storage.get(key) ?? null),
  setItemAsync: vi.fn(async (key: string, value: string) => storage.set(key, value)),
}))

import { clearSession, hasSession, saveSession } from './session'

describe('session storage', () => {
  beforeEach(() => storage.clear())

  it('persists a complete token pair as an active session', async () => {
    await saveSession({ accessToken: 'access', refreshToken: 'refresh', expiresIn: 3600 })

    await expect(hasSession()).resolves.toBe(true)
  })

  it('clears both tokens on logout', async () => {
    await saveSession({ accessToken: 'access', refreshToken: 'refresh', expiresIn: 3600 })

    await clearSession()

    await expect(hasSession()).resolves.toBe(false)
  })
})
