import { createApiClient, type StorageAdapter } from '@chorus/shared'
import { Capacitor } from '@capacitor/core'

// Get API URL based on environment
const getAPIUrl = () => {
  const platform = Capacitor.getPlatform()
  const isNative = Capacitor.isNativePlatform()

  if (isNative && platform === 'android') {
    return 'http://10.0.2.2:8080/api/v1'
  }

  if (isNative && platform === 'ios') {
    return 'http://localhost:8080/api/v1'
  }

  return '/api/v1'
}

const API_URL = getAPIUrl()

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
  baseURL: API_URL,
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

export default api