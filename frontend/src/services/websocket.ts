import { createWebSocketService, resolveApiConfig } from '@chorus/shared'
import { Capacitor } from '@capacitor/core'

// Derived from the same resolver as the HTTP base URL so the two never drift.
const platform: 'web' | 'ios' | 'android' = Capacitor.isNativePlatform()
  ? (Capacitor.getPlatform() as 'ios' | 'android')
  : 'web'

const { wsUrl } = resolveApiConfig({
  platform,
  dev: import.meta.env.DEV,
  origin: import.meta.env.VITE_API_URL,
  version: import.meta.env.VITE_API_VERSION,
})

// Same-host ('/ws') resolves against the current window origin; native builds
// get an absolute ws:// URL from the resolver (e.g. ws://<lan-ip>:8080/ws).
const absoluteWsUrl = wsUrl.startsWith('/')
  ? `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}${wsUrl}`
  : wsUrl

// WebSocket connection logic is shared across web/mobile (packages/shared).
// This file adapts it to the browser: tokens come from localStorage and the
// URL is derived from the current window origin.
export const wsService = createWebSocketService({
  getToken: async () => localStorage.getItem('accessToken'),
  createUrl: token => `${absoluteWsUrl}?token=${encodeURIComponent(token)}`,
})