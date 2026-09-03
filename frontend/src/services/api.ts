import { createApiClient, resolveApiConfig, type StorageAdapter } from '@chorus/shared'
import { Capacitor } from '@capacitor/core'

// Same-host /api/{version} by default; override with VITE_API_URL /
// VITE_API_VERSION when the backend lives on another host/port.
const platform: 'web' | 'ios' | 'android' = Capacitor.isNativePlatform()
  ? (Capacitor.getPlatform() as 'ios' | 'android')
  : 'web'

const { baseURL } = resolveApiConfig({
  platform,
  dev: import.meta.env.DEV,
  origin: import.meta.env.VITE_API_URL,
  version: import.meta.env.VITE_API_VERSION,
})

// The API client logic is shared across web/mobile (packages/shared). This file
// adapts it to the browser: localStorage-backed storage and a redirect on
// failed token refresh.
const storage: StorageAdapter = {
  getItem: async key => localStorage.getItem(key),
  setItem: async (key, value) => {
    localStorage.setItem(key, value)
  },
  removeItem: async key => {
    localStorage.removeItem(key)
  },
}

const client = createApiClient({
  baseURL,
  storage,
  onAuthFailure: () => {
    window.location.href = '/login'
  },
})

export const api = client.api
export const authAPI = client.auth
export const waitlistAPI = client.waitlist
export const adminAPI = client.admin
export const moderationAPI = client.moderation
export const chatAPI = client.chat
export const messageAPI = client.message
export const vocabularyAPI = client.vocabulary
export const billingAPI = client.billing
export const grammarAPI = client.grammar
export const translationAPI = client.translation
export const learningAPI = client.learning
export const presenceAPI = client.presence
export const settingsAPI = client.settings
export const teacherAPI = client.teacher
export const payoutsAPI = client.payouts
export const searchAPI = client.search
export const otpAPI = client.otp

export default api