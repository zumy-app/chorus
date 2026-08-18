import * as SecureStore from 'expo-secure-store'

import type { AuthTokens } from '@/types'

const ACCESS_TOKEN_KEY = 'chorus.access-token'
const REFRESH_TOKEN_KEY = 'chorus.refresh-token'

export async function saveSession(tokens: AuthTokens): Promise<void> {
  await Promise.all([
    SecureStore.setItemAsync(ACCESS_TOKEN_KEY, tokens.accessToken),
    SecureStore.setItemAsync(REFRESH_TOKEN_KEY, tokens.refreshToken),
  ])
}

export async function clearSession(): Promise<void> {
  await Promise.all([
    SecureStore.deleteItemAsync(ACCESS_TOKEN_KEY),
    SecureStore.deleteItemAsync(REFRESH_TOKEN_KEY),
  ])
}

export async function hasSession(): Promise<boolean> {
  return Boolean(await SecureStore.getItemAsync(ACCESS_TOKEN_KEY))
}

export async function getAccessToken(): Promise<string | null> {
  return SecureStore.getItemAsync(ACCESS_TOKEN_KEY)
}

export async function getRefreshToken(): Promise<string | null> {
  return SecureStore.getItemAsync(REFRESH_TOKEN_KEY)
}
