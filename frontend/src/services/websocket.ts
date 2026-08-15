import { createWebSocketService } from '@chorus/shared'

// WebSocket connection logic is shared across web/mobile (packages/shared).
// This file adapts it to the browser: tokens come from localStorage and the
// URL is derived from the current window origin.
export const wsService = createWebSocketService({
  getToken: async () => localStorage.getItem('accessToken'),
  createUrl: token => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${protocol}//${window.location.host}/ws?token=${encodeURIComponent(token)}`
  },
})