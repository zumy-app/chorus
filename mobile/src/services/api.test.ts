import { describe, expect, it, vi } from 'vitest'

vi.mock('./session', () => ({
  clearSession: vi.fn(),
  getAccessToken: vi.fn(),
  getRefreshToken: vi.fn(),
  saveSession: vi.fn(),
}))
import { getApiErrorMessage } from './api'

describe('getApiErrorMessage', () => {
  it('maps an unauthorized API response to a session-expired message', () => {
    expect(getApiErrorMessage({ response: { status: 401 } })).toBe('Your session has expired. Please sign in again.')
  })

  it('does not expose unknown server errors to the user', () => {
    expect(getApiErrorMessage({ response: { status: 500, data: { message: 'database failure' } } })).toBe(
      'Something went wrong. Please try again.',
    )
  })
})
