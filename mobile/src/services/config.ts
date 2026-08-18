const ANDROID_EMULATOR_API_URL = 'http://10.0.2.2:8080/api/v1'

export function getApiUrl(configuredUrl: string | undefined): string {
  return (configuredUrl || ANDROID_EMULATOR_API_URL).replace(/\/+$/, '')
}

export function getWebSocketUrl(apiUrl: string): string {
  const url = new URL(apiUrl)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  url.pathname = '/ws'
  url.search = ''
  return url.toString().replace(/\/$/, '')
}

export const API_URL = getApiUrl(process.env.EXPO_PUBLIC_API_URL)
export const WEBSOCKET_URL = getWebSocketUrl(API_URL)
