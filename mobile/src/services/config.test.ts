import { describe, expect, it } from 'vitest'

import { getApiUrl, getWebSocketUrl } from './config'

describe('getApiUrl', () => {
  it('uses the Android emulator host when no API URL is configured', () => {
    expect(getApiUrl(undefined)).toBe('http://10.0.2.2:8080/api/v1')
  })

  it('normalizes a configured API URL without a trailing slash', () => {
    expect(getApiUrl('https://api.chorus.talk/api/v1/')).toBe('https://api.chorus.talk/api/v1')
  })
})

describe('getWebSocketUrl', () => {
  it('derives a secure WebSocket endpoint from the API URL', () => {
    expect(getWebSocketUrl('https://api.chorus.talk/api/v1')).toBe('wss://api.chorus.talk/ws')
  })
})
